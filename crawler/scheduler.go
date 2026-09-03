package crawler

import (
	"context"
	"net/url"
	"strings"
	"sync"
	"time"
)

// HostPolicy controls how work is dispatched to each host.
type HostPolicy struct {
	// MaxConcurrent is the maximum number of in-flight pages per host.
	// Values less than one default to one.
	MaxConcurrent int

	// MinDelay is the minimum interval between starting requests to the same
	// host. Options.RequestDelay is used when it is larger.
	MinDelay time.Duration

	// MaxDelay is reserved as the ceiling for adaptive host backoff. Values
	// less than MinDelay are raised to MinDelay.
	MaxDelay time.Duration
}

type scheduledItem struct {
	item URLItem
	host string
}

type completedItem struct {
	host       string
	crawlDelay time.Duration
}

type frontierResult struct {
	item URLItem
	ok   bool
	err  error
}

type schedulerHostState struct {
	inFlight       int
	lastStarted    time.Time
	nextEligibleAt time.Time
	minDelay       time.Duration
}

// hostScheduler is the only owner of host scheduling state. A frontier pump
// supplies pending work, and workers receive only items whose host currently
// satisfies both its concurrency and delay limits.
type hostScheduler struct {
	frontier Frontier
	policy   HostPolicy

	ready     chan scheduledItem
	completed chan completedItem
	incoming  chan frontierResult
	err       error
	wg        sync.WaitGroup
}

func newHostScheduler(frontier Frontier, policy HostPolicy) *hostScheduler {
	if policy.MaxConcurrent <= 0 {
		policy.MaxConcurrent = 1
	}
	if policy.MaxDelay < policy.MinDelay {
		policy.MaxDelay = policy.MinDelay
	}
	return &hostScheduler{
		frontier:  frontier,
		policy:    policy,
		ready:     make(chan scheduledItem),
		completed: make(chan completedItem),
		incoming:  make(chan frontierResult),
	}
}

func (s *hostScheduler) start(ctx context.Context) {
	s.wg.Add(2)
	go s.run(ctx)
	go s.pump(ctx, s.incoming)
}

func (s *hostScheduler) wait() {
	s.wg.Wait()
}

func (s *hostScheduler) failure() error {
	return s.err
}

func (s *hostScheduler) next(ctx context.Context) (scheduledItem, bool) {
	select {
	case <-ctx.Done():
		return scheduledItem{}, false
	case item, ok := <-s.ready:
		return item, ok
	}
}

func (s *hostScheduler) complete(ctx context.Context, item scheduledItem, crawlDelay time.Duration) {
	select {
	case <-ctx.Done():
	case s.completed <- completedItem{host: item.host, crawlDelay: crawlDelay}:
	}
}

func (s *hostScheduler) run(ctx context.Context) {
	defer s.wg.Done()
	defer close(s.ready)

	incoming := s.incoming

	states := make(map[string]*schedulerHostState)
	var waiting []scheduledItem
	totalInFlight := 0
	frontierOpen := true

	for {
		now := time.Now()
		eligibleIndex, nextEligibleAt := nextEligibleItem(waiting, states, s.policy, now)

		var ready chan scheduledItem
		var next scheduledItem
		if eligibleIndex >= 0 {
			ready = s.ready
			next = waiting[eligibleIndex]
		}

		var timer *time.Timer
		var timerC <-chan time.Time
		if eligibleIndex < 0 && !nextEligibleAt.IsZero() {
			delay := time.Until(nextEligibleAt)
			if delay < 0 {
				delay = 0
			}
			timer = time.NewTimer(delay)
			timerC = timer.C
		}

		if !frontierOpen && len(waiting) == 0 && totalInFlight == 0 {
			if timer != nil {
				timer.Stop()
			}
			return
		}

		select {
		case <-ctx.Done():
			stopTimer(timer)
			return

		case result, ok := <-incoming:
			stopTimer(timer)
			if !ok {
				frontierOpen = false
				incoming = nil
				continue
			}
			if result.err != nil {
				if ctx.Err() != nil {
					return
				}
				s.err = result.err
				frontierOpen = false
				incoming = nil
				continue
			}
			if !result.ok {
				frontierOpen = false
				incoming = nil
				continue
			}
			waiting = append(waiting, scheduledItem{
				item: result.item,
				host: schedulerHost(result.item.URL),
			})

		case completion := <-s.completed:
			stopTimer(timer)
			state := stateForHost(states, completion.host, s.policy.MinDelay)
			if state.inFlight > 0 {
				state.inFlight--
				totalInFlight--
			}
			if completion.crawlDelay > state.minDelay {
				state.minDelay = completion.crawlDelay
				next := state.lastStarted.Add(state.minDelay)
				if next.After(state.nextEligibleAt) {
					state.nextEligibleAt = next
				}
			}

		case ready <- next:
			stopTimer(timer)
			waiting = append(waiting[:eligibleIndex], waiting[eligibleIndex+1:]...)
			state := stateForHost(states, next.host, s.policy.MinDelay)
			state.inFlight++
			totalInFlight++
			state.lastStarted = time.Now()
			state.nextEligibleAt = state.lastStarted.Add(state.minDelay)

		case <-timerC:
		}
	}
}

func (s *hostScheduler) pump(ctx context.Context, incoming chan<- frontierResult) {
	defer s.wg.Done()
	defer close(incoming)
	for {
		item, ok, err := s.frontier.Next(ctx)
		result := frontierResult{item: item, ok: ok, err: err}
		select {
		case <-ctx.Done():
			return
		case incoming <- result:
		}
		if err != nil || !ok {
			return
		}
	}
}

func nextEligibleItem(
	waiting []scheduledItem,
	states map[string]*schedulerHostState,
	policy HostPolicy,
	now time.Time,
) (int, time.Time) {
	var nextEligibleAt time.Time
	for i, item := range waiting {
		state := stateForHost(states, item.host, policy.MinDelay)
		if state.inFlight >= policy.MaxConcurrent {
			continue
		}
		if !now.Before(state.nextEligibleAt) {
			return i, time.Time{}
		}
		if nextEligibleAt.IsZero() || state.nextEligibleAt.Before(nextEligibleAt) {
			nextEligibleAt = state.nextEligibleAt
		}
	}
	return -1, nextEligibleAt
}

func stateForHost(states map[string]*schedulerHostState, host string, minDelay time.Duration) *schedulerHostState {
	state := states[host]
	if state == nil {
		state = &schedulerHostState{minDelay: minDelay}
		states[host] = state
	}
	return state
}

func schedulerHost(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return strings.ToLower(parsed.Host)
}

func stopTimer(timer *time.Timer) {
	if timer != nil {
		timer.Stop()
	}
}
