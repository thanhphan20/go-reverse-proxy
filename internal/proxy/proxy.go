package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/user/go-reverse-proxy/internal/config"
	"github.com/user/go-reverse-proxy/internal/events"
	"github.com/user/go-reverse-proxy/internal/metrics"
)

type CacheItem struct {
	Body       []byte
	Header     http.Header
	StatusCode int
	ExpiresAt  time.Time
}

type Proxy struct {
	eventWorker *events.EventWorker
	cache       map[string]CacheItem
	cacheMu     sync.RWMutex
	ttl         time.Duration
}

func NewProxy(worker *events.EventWorker, ttl time.Duration) *Proxy {
	return &Proxy{
		eventWorker: worker,
		cache:       make(map[string]CacheItem),
		ttl:         ttl,
	}
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	metrics.IncRequest()

	var ev events.Event
	ev.Path = r.URL.Path
	ev.FromCache = false

	defer func() {
		ev.Latency = time.Since(start).Milliseconds()
		p.eventWorker.Emit(ev)
	}()

	// GET caching
	if r.Method == http.MethodGet {
		p.cacheMu.RLock()
		item, found := p.cache[r.URL.String()]
		p.cacheMu.RUnlock()
		if found && time.Now().Before(item.ExpiresAt) {
			ev.FromCache = true
			ev.Status = item.StatusCode
			metrics.IncHit()
			for k, v := range item.Header {
				w.Header()[k] = v
			}
			w.WriteHeader(item.StatusCode)
			w.Write(item.Body)
			return
		}
		metrics.IncMiss()
	}

	// Routing config (longest prefix)
	var target string
	var ok bool
	routes := config.GetAll()
	var bestMatch string
	for path, targetUrl := range routes {
		if strings.HasPrefix(r.URL.Path, path) && len(path) > len(bestMatch) {
			bestMatch = path
			target = targetUrl
			ok = true
		}
	}

	if !ok {
		ev.Status = http.StatusNotFound
		metrics.IncError()
		http.Error(w, "Route not found", http.StatusNotFound)
		return
	}

	targetURL, err := url.Parse(target)
	if err != nil {
		ev.Status = http.StatusInternalServerError
		metrics.IncError()
		http.Error(w, "Invalid target", http.StatusInternalServerError)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = targetURL.Host
		req.Header.Set("User-Agent", "Go-Reverse-Proxy/1.0")
	}

	if r.Method == http.MethodGet {
		rec := httptest.NewRecorder()
		
		// strip matched prefix so upstream doesn't get it if desired
		// Actually let's just forward as-is for simplicity
		proxy.ServeHTTP(rec, r)
		
		result := rec.Result()
		body, _ := io.ReadAll(result.Body)
		
		if result.StatusCode == http.StatusOK {
			p.cacheMu.Lock()
			p.cache[r.URL.String()] = CacheItem{
				Body:       body,
				Header:     result.Header,
				StatusCode: result.StatusCode,
				ExpiresAt:  time.Now().Add(p.ttl),
			}
			p.cacheMu.Unlock()
		}

		ev.Status = result.StatusCode
		if ev.Status >= 400 {
			metrics.IncError()
		}

		for k, v := range result.Header {
			w.Header()[k] = v
		}
		w.WriteHeader(result.StatusCode)
		w.Write(body)
		
	} else {
		cw := &customWriter{ResponseWriter: w, status: http.StatusOK}
		proxy.ServeHTTP(cw, r)
		ev.Status = cw.status
		if ev.Status >= 400 {
			metrics.IncError()
		}
	}
}

type customWriter struct {
	http.ResponseWriter
	status int
}

func (w *customWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
