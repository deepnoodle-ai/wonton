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
// into the frontier. Workers hand off raw batches before returning to the
// scheduler, so bounded frontier backpressure cannot strand every worker in
// Push while eligible work is waiting to be dispatched.
type frontierAdmitter struct {
	crawler  *Crawler
	frontier Frontier
	cancel   context.CancelFunc
	batches  chan frontierBatch
	active   int64
	wg       sync.WaitGroup

	mu  sync.Mutex
	err error
}

func newFrontierAdmitter(crawler *Crawler, frontier Frontier, cancel context.CancelFunc) *frontierAdmitter {
	bufferSize := crawler.workers
	if bufferSize < 1 {
		bufferSize = 1
	}
	return &frontierAdmitter{
		crawler:  crawler,
		frontier: frontier,
		cancel:   cancel,
		// Each worker can hand off one completed page while another batch is
		// applying backpressure, leaving every worker free to drain ready work.
		batches: make(chan frontierBatch, bufferSize),
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
	close(a.batches)
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
	batch := frontierBatch{urls: urls}
	if parent != nil {
		batch.parent = *parent
		batch.hasParent = true
	}
	atomic.AddInt64(&a.active, 1)
	select {
	case a.batches <- batch:
		return nil
	case <-ctx.Done():
		a.finishBatch()
		return ctx.Err()
	}
}

func (a *frontierAdmitter) run(ctx context.Context) {
	defer a.wg.Done()
	for batch := range a.batches {
		if ctx.Err() != nil {
			a.finishBatch()
			continue
		}
		a.admitBatch(ctx, batch)
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
