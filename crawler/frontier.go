package crawler

import (
	"container/heap"
	"context"
	"errors"
	"sync"
	"time"
)

// ErrFrontierClosed is returned when work is pushed to a closed Frontier.
var ErrFrontierClosed = errors.New("crawler frontier is closed")

// URLItem is one unit of pending crawl work.
type URLItem struct {
	// URL is the normalized URL to crawl.
	URL string

	// Depth is the number of links between this URL and its seed URL.
	Depth int

	// Referrer is the URL of the page where this URL was discovered. It is
	// empty for seed URLs.
	Referrer string

	// Score controls priority. Higher-scored items are returned first.
	Score float64

	// DiscoveredAt records when the URL was admitted to the frontier.
	DiscoveredAt time.Time
}

// Frontier stores pending URLs. Implementations must be safe for concurrent
// use and must report, rather than silently discard, work they cannot accept.
type Frontier interface {
	// Push adds items to the frontier. A bounded implementation may block until
	// capacity is available or the context is canceled.
	Push(ctx context.Context, items ...URLItem) error

	// Next blocks until an item is available, the context is canceled, or the
	// frontier has been closed and drained. ok is false only when no more work
	// will ever arrive.
	Next(ctx context.Context) (item URLItem, ok bool, err error)

	// Len reports the number of items waiting in the frontier.
	Len() int

	// Close prevents future pushes. Already queued items remain available.
	Close() error
}

// MemoryFrontier is an in-memory priority frontier. Items are ordered by
// descending score, then ascending depth, then insertion order.
//
// A positive maxPending applies backpressure to Push when that many items are
// waiting. A value less than one makes the frontier unbounded.
type MemoryFrontier struct {
	mu         sync.Mutex
	items      memoryFrontierHeap
	maxPending int
	nextSeq    uint64
	closed     bool
	changed    chan struct{}
}

var _ Frontier = (*MemoryFrontier)(nil)

// NewMemoryFrontier returns an empty in-memory frontier. See MemoryFrontier
// for ordering and capacity semantics.
func NewMemoryFrontier(maxPending int) *MemoryFrontier {
	f := &MemoryFrontier{
		maxPending: maxPending,
		changed:    make(chan struct{}),
	}
	heap.Init(&f.items)
	return f
}

// Push adds items to the frontier, waiting for capacity when it is bounded.
func (f *MemoryFrontier) Push(ctx context.Context, items ...URLItem) error {
	for _, item := range items {
		for {
			f.mu.Lock()
			if f.closed {
				f.mu.Unlock()
				return ErrFrontierClosed
			}
			if f.maxPending <= 0 || f.items.Len() < f.maxPending {
				f.nextSeq++
				heap.Push(&f.items, memoryFrontierItem{URLItem: item, seq: f.nextSeq})
				f.signalLocked()
				f.mu.Unlock()
				break
			}
			changed := f.changed
			f.mu.Unlock()

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-changed:
			}
		}
	}
	return nil
}

// Next returns the highest-priority pending item.
func (f *MemoryFrontier) Next(ctx context.Context) (URLItem, bool, error) {
	for {
		f.mu.Lock()
		if f.items.Len() > 0 {
			item := heap.Pop(&f.items).(memoryFrontierItem).URLItem
			f.signalLocked()
			f.mu.Unlock()
			return item, true, nil
		}
		if f.closed {
			f.mu.Unlock()
			return URLItem{}, false, nil
		}
		changed := f.changed
		f.mu.Unlock()

		select {
		case <-ctx.Done():
			return URLItem{}, false, ctx.Err()
		case <-changed:
		}
	}
}

// Len returns the number of pending items.
func (f *MemoryFrontier) Len() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.items.Len()
}

// Close prevents future pushes. It is safe to call more than once.
func (f *MemoryFrontier) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.closed {
		f.closed = true
		f.signalLocked()
	}
	return nil
}

// signalLocked wakes every waiter. Replacing the channel after closing it
// avoids missed notifications without requiring a polling loop.
func (f *MemoryFrontier) signalLocked() {
	close(f.changed)
	f.changed = make(chan struct{})
}

type memoryFrontierItem struct {
	URLItem
	seq uint64
}

type memoryFrontierHeap []memoryFrontierItem

func (h memoryFrontierHeap) Len() int { return len(h) }

func (h memoryFrontierHeap) Less(i, j int) bool {
	if h[i].Score != h[j].Score {
		return h[i].Score > h[j].Score
	}
	if h[i].Depth != h[j].Depth {
		return h[i].Depth < h[j].Depth
	}
	return h[i].seq < h[j].seq
}

func (h memoryFrontierHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h *memoryFrontierHeap) Push(value any) {
	*h = append(*h, value.(memoryFrontierItem))
}

func (h *memoryFrontierHeap) Pop() any {
	old := *h
	last := len(old) - 1
	item := old[last]
	old[last] = memoryFrontierItem{}
	*h = old[:last]
	return item
}
