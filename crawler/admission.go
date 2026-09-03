package crawler

import (
	"context"
	"sync"
	"sync/atomic"
)

type frontierBatch struct {
	urls      []string
	parent    URLItem
	hasParent bool
}

// frontierAdmitter is the only goroutine that pushes crawler-discovered work
// into the frontier. Workers hand off raw discovery batches through a
// nonblocking broker before returning to the scheduler, so bounded frontier
// backpressure cannot strand every worker in Push while eligible work is
// waiting to be dispatched. QueueSize applies after URLs are normalized and
// admitted; MaxURLs separately bounds how many URLs are ultimately admitted.
type frontierAdmitter struct {
	crawler  *Crawler
	frontier Frontier
	cancel   context.CancelFunc
	wake     chan struct{}
	active   int64
	wg       sync.WaitGroup

	mu     sync.Mutex
	queue  []frontierBatch
	closed bool
	err    error
}

func newFrontierAdmitter(crawler *Crawler, frontier Frontier, cancel context.CancelFunc) *frontierAdmitter {
	return &frontierAdmitter{
		crawler:  crawler,
		frontier: frontier,
		cancel:   cancel,
		wake:     make(chan struct{}, 1),
	}
}

func (a *frontierAdmitter) start(ctx context.Context) {
	a.wg.Add(1)
	go a.run(ctx)
}

func (a *frontierAdmitter) wait() {
	a.wg.Wait()
}

func (a *frontierAdmitter) close() {
	a.mu.Lock()
	a.closed = true
	a.signalLocked()
	a.mu.Unlock()
}

func (a *frontierAdmitter) failure() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.err
}

func (a *frontierAdmitter) submit(ctx context.Context, urls []string, parent *URLItem) error {
	if len(urls) == 0 {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	batch := frontierBatch{urls: urls}
	if parent != nil {
		batch.parent = *parent
		batch.hasParent = true
	}
	atomic.AddInt64(&a.active, 1)
	a.mu.Lock()
	if err := ctx.Err(); err != nil {
		a.mu.Unlock()
		a.finishBatch()
		return err
	}
	if a.closed {
		a.mu.Unlock()
		a.finishBatch()
		return ErrFrontierClosed
	}
	a.queue = append(a.queue, batch)
	a.signalLocked()
	a.mu.Unlock()
	return nil
}

func (a *frontierAdmitter) run(ctx context.Context) {
	defer a.wg.Done()
	for {
		batch, ok, closed := a.takeBatch()
		if ok {
			if ctx.Err() != nil {
				a.finishBatch()
				continue
			}
			a.admitBatch(ctx, batch)
			continue
		}
		if closed {
			return
		}
		select {
		case <-a.wake:
		case <-ctx.Done():
			a.discardQueued()
			return
		}
	}
}

func (a *frontierAdmitter) takeBatch() (frontierBatch, bool, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.queue) == 0 {
		return frontierBatch{}, false, a.closed
	}
	batch := a.queue[0]
	a.queue[0] = frontierBatch{}
	a.queue = a.queue[1:]
	return batch, true, a.closed
}

func (a *frontierAdmitter) discardQueued() {
	a.mu.Lock()
	queued := a.queue
	a.queue = nil
	a.mu.Unlock()
	for range queued {
		a.finishBatch()
	}
}

func (a *frontierAdmitter) signalLocked() {
	select {
	case a.wake <- struct{}{}:
	default:
	}
}

func (a *frontierAdmitter) admitBatch(ctx context.Context, batch frontierBatch) {
	defer a.finishBatch()
	var parent *URLItem
	if batch.hasParent {
		parent = &batch.parent
	}
	for _, rawURL := range batch.urls {
		if ctx.Err() != nil {
			return
		}
		item, prepared, stop := a.crawler.prepareURL(rawURL, parent)
		if stop {
			break
		}
		if !prepared {
			continue
		}
		if err := a.push(ctx, item); err != nil {
			a.crawler.releaseURL(item.URL)
			if ctx.Err() == nil {
				a.mu.Lock()
				if a.err == nil {
					a.err = err
				}
				a.mu.Unlock()
				a.cancel()
			}
			return
		}
	}
}

func (a *frontierAdmitter) push(ctx context.Context, item URLItem) error {
	if frontier, ok := a.frontier.(*MemoryFrontier); ok {
		return frontier.pushForScheduling(ctx, item, func() {
			atomic.AddInt64(&a.crawler.pendingURLs, 1)
		})
	}

	// A custom Frontier cannot expose an atomic capacity/admission boundary.
	// Count before Push so a fast consumer cannot complete the item first.
	atomic.AddInt64(&a.crawler.pendingURLs, 1)
	if err := a.frontier.Push(ctx, item); err != nil {
		atomic.AddInt64(&a.crawler.pendingURLs, -1)
		return err
	}
	return nil
}

func (a *frontierAdmitter) finishBatch() {
	if atomic.AddInt64(&a.active, -1) == 0 {
		a.cancelIfIdle()
	}
}

func (a *frontierAdmitter) completeURL() {
	if atomic.AddInt64(&a.crawler.pendingURLs, -1) == 0 {
		a.cancelIfIdle()
	}
}

func (a *frontierAdmitter) cancelIfIdle() {
	if atomic.LoadInt64(&a.crawler.pendingURLs) == 0 && atomic.LoadInt64(&a.active) == 0 {
		a.cancel()
	}
}
