package crawler

import (
	"context"
	"math"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	defaultAdaptiveInitialDelay = time.Second
	defaultAdaptiveLatencyDelay = 100 * time.Millisecond
	defaultAdaptiveMaxDelay     = 30 * time.Second
	healthyResponsesBeforeDecay = 5
)

// HostPolicy controls how work is dispatched to each host.
type HostPolicy struct {
	// MaxConcurrent is the maximum number of in-flight pages per host.
	// Values less than one default to one.
	MaxConcurrent int

	// MinDelay is the minimum interval between starting requests to the same
	// host. Options.RequestDelay is used when it is larger.
	MinDelay time.Duration

	// MaxDelay is the ceiling for adaptive host backoff. When Adaptive is
	// enabled, zero defaults to 30 seconds. Values less than MinDelay are
	// raised to MinDelay.
	MaxDelay time.Duration
}

type scheduledItem struct {
	item          URLItem
	host          string
	attempt       int
	attemptErrors []error
	leased        bool
}

type completedItem struct {
	scheduled   scheduledItem
	crawlDelay  time.Duration
	observation requestObservation
	retry       bool
	retryDelay  time.Duration
}

type requestObservation struct {
	statusCode    int
	latency       time.Duration
	failed        bool
	retryAfter    time.Duration
	hasRetryAfter bool
}

type frontierResult struct {
	item   URLItem
	ok     bool
	leased bool
}

type schedulerHostState struct {
	inFlight       int
	lastStarted    time.Time
	nextEligibleAt time.Time
	minDelay       time.Duration
	currentDelay   time.Duration
	baseline       time.Duration
	latencyEWMA    time.Duration
	healthy        int
	latencyBackoff bool
}

type hostWaitingQueues struct {
	queues    map[string][]scheduledItem
	hosts     []string
	positions map[string]int
	total     int
}

func newHostWaitingQueues() *hostWaitingQueues {
	return &hostWaitingQueues{
		queues:    make(map[string][]scheduledItem),
		positions: make(map[string]int),
	}
}

func (q *hostWaitingQueues) add(item scheduledItem) {
	if len(q.queues[item.host]) == 0 {
		q.positions[item.host] = len(q.hosts)
		q.hosts = append(q.hosts, item.host)
	}
	q.queues[item.host] = append(q.queues[item.host], item)
	q.total++
}

func (q *hostWaitingQueues) peek(host string) (scheduledItem, bool) {
	items := q.queues[host]
	if len(items) == 0 {
		return scheduledItem{}, false
	}
	return items[0], true
}

func (q *hostWaitingQueues) pop(host string) (scheduledItem, bool) {
	items := q.queues[host]
	if len(items) == 0 {
		return scheduledItem{}, false
	}
	item := items[0]
	items[0] = scheduledItem{}
	if len(items) == 1 {
		delete(q.queues, host)
		q.removeHost(host)
	} else {
		q.queues[host] = items[1:]
	}
	q.total--
	return item, true
}

func (q *hostWaitingQueues) removeHost(host string) {
	position, ok := q.positions[host]
	if !ok {
		return
	}
	last := len(q.hosts) - 1
	lastHost := q.hosts[last]
	q.hosts[position] = lastHost
	q.positions[lastHost] = position
	q.hosts[last] = ""
	q.hosts = q.hosts[:last]
	delete(q.positions, host)
}

func (q *hostWaitingQueues) len() int {
	return q.total
}

// hostScheduler is the only owner of host scheduling state. A frontier pump
// supplies pending work, and workers receive only items whose host currently
// satisfies both its concurrency and delay limits.
type hostScheduler struct {
	frontier   Frontier
	policy     HostPolicy
	adaptive   bool
	stats      *hostStatsCollector
	maxPending int
	cancel     context.CancelFunc

	ready     chan scheduledItem
	completed chan completedItem
	incoming  chan frontierResult
	failed    chan error
	err       error
	wg        sync.WaitGroup
}

func newHostScheduler(frontier Frontier, policy HostPolicy, adaptive bool, stats *hostStatsCollector, maxPending int) *hostScheduler {
	if policy.MaxConcurrent <= 0 {
		policy.MaxConcurrent = 1
	}
	if adaptive && policy.MaxDelay <= 0 {
		policy.MaxDelay = defaultAdaptiveMaxDelay
	}
	if policy.MaxDelay < policy.MinDelay {
		policy.MaxDelay = policy.MinDelay
	}
	return &hostScheduler{
		frontier:   frontier,
		policy:     policy,
		adaptive:   adaptive,
		stats:      stats,
		maxPending: maxPending,
		ready:      make(chan scheduledItem),
		completed:  make(chan completedItem),
		incoming:   make(chan frontierResult),
		failed:     make(chan error),
	}
}

func (s *hostScheduler) start(ctx context.Context, cancel context.CancelFunc) {
	s.cancel = cancel
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

func (s *hostScheduler) complete(ctx context.Context, completion completedItem) {
	select {
	case <-ctx.Done():
	case s.completed <- completion:
	}
}

func (s *hostScheduler) run(ctx context.Context) {
	defer s.wg.Done()
	defer close(s.ready)

	incoming := s.incoming

	states := make(map[string]*schedulerHostState)
	waiting := newHostWaitingQueues()
	defer s.releaseWaiting(waiting)
	totalInFlight := 0
	leasedInFlight := 0
	defer func() {
		for leasedInFlight > 0 {
			s.releaseLease()
			leasedInFlight--
		}
	}()
	frontierOpen := true

	for {
		now := time.Now()
		candidate, eligible, nextEligibleAt := nextEligibleItem(waiting, states, s.policy, now)

		var ready chan scheduledItem
		var next scheduledItem
		if eligible {
			ready = s.ready
			next = candidate
		}

		var timer *time.Timer
		var timerC <-chan time.Time
		if !eligible && !nextEligibleAt.IsZero() {
			delay := time.Until(nextEligibleAt)
			if delay < 0 {
				delay = 0
			}
			timer = time.NewTimer(delay)
			timerC = timer.C
		}

		if !frontierOpen && waiting.len() == 0 && totalInFlight == 0 {
			if timer != nil {
				timer.Stop()
			}
			return
		}

		admitIncoming := incoming
		if s.maxPending > 0 && waiting.len()+totalInFlight >= s.maxPending {
			admitIncoming = nil
		}

		select {
		case <-ctx.Done():
			stopTimer(timer)
			return

		case err := <-s.failed:
			stopTimer(timer)
			if ctx.Err() != nil {
				return
			}
			s.err = err
			if s.cancel != nil {
				s.cancel()
			}
			return

		case result, ok := <-admitIncoming:
			stopTimer(timer)
			if !ok {
				frontierOpen = false
				incoming = nil
				continue
			}
			if !result.ok {
				frontierOpen = false
				incoming = nil
				continue
			}
			waiting.add(scheduledItem{
				item:    result.item,
				host:    schedulerHost(result.item.URL),
				attempt: 1,
				leased:  result.leased,
			})

		case completion := <-s.completed:
			stopTimer(timer)
			state := stateForHost(states, completion.scheduled.host, s.policy.MinDelay)
			if state.inFlight > 0 {
				state.inFlight--
				totalInFlight--
			}
			if completion.scheduled.leased && leasedInFlight > 0 {
				leasedInFlight--
			}
			if completion.crawlDelay > state.minDelay {
				state.minDelay = completion.crawlDelay
				next := state.lastStarted.Add(state.minDelay)
				if next.After(state.nextEligibleAt) {
					state.nextEligibleAt = next
				}
			}
			s.applyObservation(state, completion.observation)
			if completion.retry {
				completion.scheduled.attempt++
				if retryAt := time.Now().Add(completion.retryDelay); retryAt.After(state.nextEligibleAt) {
					state.nextEligibleAt = retryAt
				}
				waiting.add(completion.scheduled)
			} else if completion.scheduled.leased {
				s.releaseLease()
			}
			if s.stats != nil {
				s.stats.setDelay(completion.scheduled.host, effectiveHostDelay(state))
			}

		case ready <- next:
			stopTimer(timer)
			queued, _ := waiting.pop(candidate.host)
			state := stateForHost(states, next.host, s.policy.MinDelay)
			state.inFlight++
			totalInFlight++
			if queued.leased {
				leasedInFlight++
			}
			state.lastStarted = time.Now()
			state.nextEligibleAt = state.lastStarted.Add(effectiveHostDelay(state))

		case <-timerC:
		}
	}
}

func (s *hostScheduler) pump(ctx context.Context, incoming chan<- frontierResult) {
	defer s.wg.Done()
	defer close(incoming)
	for {
		var item URLItem
		var ok bool
		var err error
		leased := false
		if frontier, supportsLeases := s.frontier.(*MemoryFrontier); supportsLeases {
			item, ok, err = frontier.nextForScheduling(ctx)
			leased = ok && err == nil
		} else {
			item, ok, err = s.frontier.Next(ctx)
		}
		if err != nil {
			if leased {
				s.releaseLease()
			}
			select {
			case <-ctx.Done():
			case s.failed <- err:
			}
			return
		}
		result := frontierResult{item: item, ok: ok, leased: leased}
		select {
		case <-ctx.Done():
			if leased {
				s.releaseLease()
			}
			return
		case incoming <- result:
		}
		if !ok {
			return
		}
	}
}

func nextEligibleItem(
	waiting *hostWaitingQueues,
	states map[string]*schedulerHostState,
	policy HostPolicy,
	now time.Time,
) (scheduledItem, bool, time.Time) {
	var nextEligibleAt time.Time
	for _, host := range waiting.hosts {
		item, ok := waiting.peek(host)
		if !ok {
			continue
		}
		state := stateForHost(states, host, policy.MinDelay)
		if state.inFlight >= policy.MaxConcurrent {
			continue
		}
		if !now.Before(state.nextEligibleAt) {
			return item, true, time.Time{}
		}
		if nextEligibleAt.IsZero() || state.nextEligibleAt.Before(nextEligibleAt) {
			nextEligibleAt = state.nextEligibleAt
		}
	}
	return scheduledItem{}, false, nextEligibleAt
}

func (s *hostScheduler) releaseLease() {
	if frontier, ok := s.frontier.(*MemoryFrontier); ok {
		frontier.releaseScheduled()
	}
}

func (s *hostScheduler) releaseWaiting(waiting *hostWaitingQueues) {
	for len(waiting.hosts) > 0 {
		host := waiting.hosts[len(waiting.hosts)-1]
		for {
			item, ok := waiting.pop(host)
			if !ok {
				break
			}
			if item.leased {
				s.releaseLease()
			}
		}
	}
}

func stateForHost(states map[string]*schedulerHostState, host string, minDelay time.Duration) *schedulerHostState {
	state := states[host]
	if state == nil {
		state = &schedulerHostState{minDelay: minDelay, currentDelay: minDelay}
		states[host] = state
	}
	return state
}

func (s *hostScheduler) applyObservation(state *schedulerHostState, observation requestObservation) {
	if !s.adaptive {
		return
	}
	now := time.Now()
	if observation.statusCode == 429 || observation.statusCode == 503 {
		state.healthy = 0
		delay := observation.retryAfter
		if !observation.hasRetryAfter {
			delay = effectiveHostDelay(state) * 2
			if delay <= 0 {
				delay = defaultAdaptiveInitialDelay
			}
		}
		state.currentDelay = clampDelay(delay, state.minDelay, s.policy.MaxDelay)
		if next := now.Add(state.currentDelay); next.After(state.nextEligibleAt) {
			state.nextEligibleAt = next
		}
		return
	}
	if observation.failed || observation.statusCode >= 400 {
		state.healthy = 0
		return
	}
	// Some platforms have a coarse monotonic clock and can report a zero
	// duration for an in-memory fetch. Status-driven backoff above must still
	// apply; only latency adaptation needs a positive measurement.
	if observation.latency <= 0 {
		return
	}

	if state.baseline <= 0 || observation.latency < state.baseline {
		state.baseline = observation.latency
	}
	if state.latencyEWMA <= 0 {
		state.latencyEWMA = observation.latency
	} else {
		state.latencyEWMA = time.Duration(0.8*float64(state.latencyEWMA) + 0.2*float64(observation.latency))
	}

	if state.baseline > 0 && state.latencyEWMA > 2*state.baseline {
		state.healthy = 0
		if delay := effectiveHostDelay(state); !state.latencyBackoff {
			if delay <= 0 {
				delay = defaultAdaptiveLatencyDelay
			}
			state.currentDelay = clampDelay(time.Duration(math.Ceil(float64(delay)*1.25)), state.minDelay, s.policy.MaxDelay)
		}
		state.latencyBackoff = true
		return
	}
	state.latencyBackoff = false

	if observation.latency <= time.Duration(1.25*float64(state.baseline)) {
		state.healthy++
	} else {
		state.healthy = 0
	}
	if state.healthy >= healthyResponsesBeforeDecay && state.currentDelay > state.minDelay {
		state.currentDelay = clampDelay(time.Duration(float64(state.currentDelay)*0.8), state.minDelay, s.policy.MaxDelay)
		state.healthy = 0
	}
}

func effectiveHostDelay(state *schedulerHostState) time.Duration {
	if state.currentDelay > state.minDelay {
		return state.currentDelay
	}
	return state.minDelay
}

func clampDelay(delay, minimum, maximum time.Duration) time.Duration {
	if maximum > 0 && delay > maximum {
		delay = maximum
	}
	// A configured or robots.txt minimum is always authoritative, even when
	// it is larger than the adaptive ceiling.
	if delay < minimum {
		delay = minimum
	}
	return delay
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
