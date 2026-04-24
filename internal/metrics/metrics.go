package metrics

import (
	"sync/atomic"
)

type Metrics struct {
	TotalRequests int64 `json:"total_requests"`
	ErrorCount    int64 `json:"error_count"`
	CacheHits     int64 `json:"cache_hits"`
	CacheMisses   int64 `json:"cache_misses"`
}

var currentMetrics Metrics

func IncRequest() { atomic.AddInt64(&currentMetrics.TotalRequests, 1) }
func IncError()   { atomic.AddInt64(&currentMetrics.ErrorCount, 1) }
func IncHit()     { atomic.AddInt64(&currentMetrics.CacheHits, 1) }
func IncMiss()    { atomic.AddInt64(&currentMetrics.CacheMisses, 1) }

func Get() Metrics {
	return Metrics{
		TotalRequests: atomic.LoadInt64(&currentMetrics.TotalRequests),
		ErrorCount:    atomic.LoadInt64(&currentMetrics.ErrorCount),
		CacheHits:     atomic.LoadInt64(&currentMetrics.CacheHits),
		CacheMisses:   atomic.LoadInt64(&currentMetrics.CacheMisses),
	}
}
