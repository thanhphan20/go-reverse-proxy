package metrics

import (
	"sync/atomic"
)

type Metrics struct {
	TotalRequests int64   `json:"total_requests"`
	ErrorCount    int64   `json:"error_count"`
	CacheHits     int64   `json:"cache_hits"`
	CacheMisses   int64   `json:"cache_misses"`
	TotalLatency  int64   `json:"total_latency"`
	LatencyCount  int64   `json:"latency_count"`
	AvgLatency    float64 `json:"avg_latency"`
	ErrorRate     float64 `json:"error_rate"`
}

var currentMetrics Metrics

func IncRequest() { atomic.AddInt64(&currentMetrics.TotalRequests, 1) }
func IncError()   { atomic.AddInt64(&currentMetrics.ErrorCount, 1) }
func IncHit()     { atomic.AddInt64(&currentMetrics.CacheHits, 1) }
func IncMiss()    { atomic.AddInt64(&currentMetrics.CacheMisses, 1) }
func IncLatency(lat int64) {
	atomic.AddInt64(&currentMetrics.TotalLatency, lat)
	atomic.AddInt64(&currentMetrics.LatencyCount, 1)
}

func Get() Metrics {
	totalReq := atomic.LoadInt64(&currentMetrics.TotalRequests)
	errors := atomic.LoadInt64(&currentMetrics.ErrorCount)
	totalLat := atomic.LoadInt64(&currentMetrics.TotalLatency)
	latCount := atomic.LoadInt64(&currentMetrics.LatencyCount)
	avgLat := float64(0)
	if latCount > 0 {
		avgLat = float64(totalLat) / float64(latCount)
	}
	errorRate := float64(0)
	if totalReq > 0 {
		errorRate = float64(errors) / float64(totalReq) * 100
	}
	return Metrics{
		TotalRequests: totalReq,
		ErrorCount:    errors,
		CacheHits:     atomic.LoadInt64(&currentMetrics.CacheHits),
		CacheMisses:   atomic.LoadInt64(&currentMetrics.CacheMisses),
		TotalLatency:  totalLat,
		LatencyCount:  latCount,
		AvgLatency:    avgLat,
		ErrorRate:     errorRate,
	}
}
