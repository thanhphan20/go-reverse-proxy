package proxy

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
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
}

var roundRobinIndices sync.Map // map[string]*atomic.Int64

func NewProxy(worker *events.EventWorker) *Proxy {
	return &Proxy{
		eventWorker: worker,
		cache:       make(map[string]CacheItem),
	}
}

func generateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func singleJoiningSlash(a, b string) string {
	if a == "" {
		return b
	}
	if b == "" {
		return a
	}
	if strings.HasSuffix(a, "/") || strings.HasPrefix(b, "/") {
		return a + b
	}
	return a + "/" + b
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	metrics.IncRequest()

	id := generateID()
	r.Header.Set("X-Request-ID", id)

	var ev events.Event
	ev.Path = r.URL.Path
	ev.RequestID = id
	ev.FromCache = false

	defer func() {
		ev.Latency = time.Since(start).Milliseconds()
		metrics.IncLatency(ev.Latency)
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
	var targets []string
	var ok bool
	routes := config.GetAll()
	var bestMatch string
	for path, targetUrls := range routes {
		if strings.HasPrefix(r.URL.Path, path) && len(path) > len(bestMatch) {
			bestMatch = path
			targets = targetUrls
			ok = true
		}
	}

	if !ok {
		ev.Status = http.StatusNotFound
		metrics.IncError()
		slog.Error("route not found", "path", r.URL.Path, "id", id)
		http.Error(w, "Route not found", http.StatusNotFound)
		return
	}

	// Round-robin selection
	var startIndex int
	if len(targets) > 1 {
		val, _ := roundRobinIndices.LoadOrStore(bestMatch, &atomic.Int64{})
		idx := val.(*atomic.Int64)
		startIndex = int(idx.Add(1) - 1) % len(targets)
	} else {
		startIndex = 0
	}

	// Proxy with retries
	res, err := p.proxyRequest(r, targets, startIndex, config.GetConfig().Proxy.Retries, bestMatch)
	if err != nil {
		ev.Status = http.StatusInternalServerError
		metrics.IncError()
		slog.Error("proxy failed", "error", err, "id", id)
		http.Error(w, "Proxy failed", http.StatusInternalServerError)
		return
	}

	body, _ := io.ReadAll(res.Body)

	// Cache GET responses
	if r.Method == http.MethodGet && res.StatusCode == http.StatusOK {
		p.cacheMu.Lock()
		p.cache[r.URL.String()] = CacheItem{
			Body:       body,
			Header:     res.Header,
			StatusCode: res.StatusCode,
			ExpiresAt:  time.Now().Add(config.GetConfig().Cache.TTL),
		}
		p.cacheMu.Unlock()
	}

	ev.Status = res.StatusCode
	if ev.Status >= 400 {
		metrics.IncError()
	}

	for k, v := range res.Header {
		w.Header()[k] = v
	}
	w.WriteHeader(res.StatusCode)
	w.Write(body)
}

func (p *Proxy) proxyRequest(r *http.Request, targets []string, startIndex int, retries int, prefix string) (*http.Response, error) {
	for attempt := 0; attempt <= retries; attempt++ {
		idx := (startIndex + attempt) % len(targets)
		target := targets[idx]
		targetURL, err := url.Parse(target)
		if err != nil {
			continue
		}
		proxy := httputil.NewSingleHostReverseProxy(targetURL)
		proxy.Transport = &http.Transport{
			Dial:                  (&net.Dialer{Timeout: config.GetConfig().Proxy.Timeout.Dial}).Dial,
			ResponseHeaderTimeout: config.GetConfig().Proxy.Timeout.ResponseHeader,
		}
		proxy.Director = func(req *http.Request) {
			trimmed := strings.TrimPrefix(req.URL.Path, prefix)
			req.URL.Path = singleJoiningSlash(targetURL.Path, trimmed)
			req.URL.Scheme = targetURL.Scheme
			req.URL.Host = targetURL.Host
			req.Host = targetURL.Host
			req.Header.Set("User-Agent", "Go-Reverse-Proxy/1.0")
		}
		rec := httptest.NewRecorder()
		proxy.ServeHTTP(rec, r)
		res := rec.Result()
		if res.StatusCode < 500 || attempt == retries {
			return res, nil
		}
		slog.Warn("retrying request", "attempt", attempt+1, "target", target, "status", res.StatusCode, "id", r.Header.Get("X-Request-ID"))
	}
	return nil, errors.New("max retries exceeded")
}

type customWriter struct {
	http.ResponseWriter
	status int
}

func (w *customWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
