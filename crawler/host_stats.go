package crawler

import (
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/deepnoodle-ai/wonton/fetch"
)

// HostStats summarizes the network impact of the current or most recent
// Crawl call for one host. Cache hits are not network requests and are not
// included.
type HostStats struct {
	Host        string
	Requests    int64
	Bytes       int64
	Errors      int64
	StatusCodes map[int]int64
	PeakRPS     float64
	P50         time.Duration
	P95         time.Duration
	FinalDelay  time.Duration
}

type hostStatsState struct {
	requests     int64
	bytes        int64
	errors       int64
	statusCodes  map[int]int64
	peakRPS      float64
	recentStarts []time.Time
	latencies    []time.Duration
	finalDelay   time.Duration
}

type hostStatsCollector struct {
	mu    sync.RWMutex
	hosts map[string]*hostStatsState
}

func newHostStatsCollector() *hostStatsCollector {
	return &hostStatsCollector{hosts: make(map[string]*hostStatsState)}
}

func (c *hostStatsCollector) reset() {
	c.mu.Lock()
	c.hosts = make(map[string]*hostStatsState)
	c.mu.Unlock()
}

func (c *hostStatsCollector) requestStarted(host string, started time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	state := c.state(host)
	state.requests++

	cutoff := started.Add(-time.Second)
	first := 0
	for first < len(state.recentStarts) && state.recentStarts[first].Before(cutoff) {
		first++
	}
	if first > 0 {
		state.recentStarts = append(state.recentStarts[:0], state.recentStarts[first:]...)
	}
	state.recentStarts = append(state.recentStarts, started)
	if rps := float64(len(state.recentStarts)); rps > state.peakRPS {
		state.peakRPS = rps
	}
}

func (c *hostStatsCollector) requestFinished(host string, latency time.Duration, response *fetch.Response, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	state := c.state(host)
	state.latencies = append(state.latencies, latency)
	if response != nil {
		if response.StatusCode != 0 {
			state.statusCodes[response.StatusCode]++
		}
		bodyBytes := int64(len(response.HTML))
		if response.RawHTML != "" {
			bodyBytes = int64(len(response.RawHTML))
		}
		if contentLength, parseErr := strconv.ParseInt(headerValue(response.Headers, "Content-Length"), 10, 64); parseErr == nil && contentLength >= 0 {
			bodyBytes = contentLength
		}
		state.bytes += bodyBytes
	}
	if err != nil || response != nil && response.StatusCode >= 400 {
		state.errors++
	}
}

func (c *hostStatsCollector) setDelay(host string, delay time.Duration) {
	c.mu.Lock()
	c.state(host).finalDelay = delay
	c.mu.Unlock()
}

func (c *hostStatsCollector) snapshot() []HostStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := make([]HostStats, 0, len(c.hosts))
	for host, state := range c.hosts {
		latencies := append([]time.Duration(nil), state.latencies...)
		sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
		statusCodes := make(map[int]int64, len(state.statusCodes))
		for status, count := range state.statusCodes {
			statusCodes[status] = count
		}
		stats = append(stats, HostStats{
			Host:        host,
			Requests:    state.requests,
			Bytes:       state.bytes,
			Errors:      state.errors,
			StatusCodes: statusCodes,
			PeakRPS:     state.peakRPS,
			P50:         percentile(latencies, 0.50),
			P95:         percentile(latencies, 0.95),
			FinalDelay:  state.finalDelay,
		})
	}
	sort.Slice(stats, func(i, j int) bool { return stats[i].Host < stats[j].Host })
	return stats
}

func (c *hostStatsCollector) state(host string) *hostStatsState {
	state := c.hosts[host]
	if state == nil {
		state = &hostStatsState{statusCodes: make(map[int]int64)}
		c.hosts[host] = state
	}
	return state
}

func percentile(sorted []time.Duration, fraction float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	index := int(float64(len(sorted)-1) * fraction)
	return sorted[index]
}
