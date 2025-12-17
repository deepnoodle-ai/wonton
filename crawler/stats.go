package crawler

import "sync/atomic"

// CrawlerStats tracks crawling statistics. All methods are thread-safe.
type CrawlerStats struct {
	processed int64
	succeeded int64
	failed    int64
}

// GetProcessed returns the number of URLs processed
func (s *CrawlerStats) GetProcessed() int64 {
	return atomic.LoadInt64(&s.processed)
}

// GetSucceeded returns the number of URLs successfully processed
func (s *CrawlerStats) GetSucceeded() int64 {
	return atomic.LoadInt64(&s.succeeded)
}

// GetFailed returns the number of URLs that failed to process
func (s *CrawlerStats) GetFailed() int64 {
	return atomic.LoadInt64(&s.failed)
}

// IncrementProcessed atomically increments the processed counter
func (s *CrawlerStats) IncrementProcessed() {
	atomic.AddInt64(&s.processed, 1)
}

// IncrementSucceeded atomically increments the succeeded counter
func (s *CrawlerStats) IncrementSucceeded() {
	atomic.AddInt64(&s.succeeded, 1)
}

// IncrementFailed atomically increments the failed counter
func (s *CrawlerStats) IncrementFailed() {
	atomic.AddInt64(&s.failed, 1)
}
