package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"

	"github.com/user/go-reverse-proxy/internal/blocklist"
	"github.com/user/go-reverse-proxy/internal/config"
	"github.com/user/go-reverse-proxy/internal/events"
	"github.com/user/go-reverse-proxy/internal/metrics"
	"github.com/user/go-reverse-proxy/internal/proxy"
)

// ---------------------------------------------------------------------------
// SSE broker — fans out dashboard state to all connected clients.
// ---------------------------------------------------------------------------

type DashboardState struct {
	Metrics metrics.Metrics   `json:"metrics"`
	Events  []events.Event    `json:"events"`
	Routes  map[string]string `json:"routes"`
}

type SSEBroker struct {
	mu      sync.RWMutex
	clients map[chan string]struct{}
}

func NewSSEBroker() *SSEBroker {
	return &SSEBroker{clients: make(map[chan string]struct{})}
}

func (b *SSEBroker) Subscribe() chan string {
	ch := make(chan string, 8)
	b.mu.Lock()
	b.clients[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

func (b *SSEBroker) Unsubscribe(ch chan string) {
	b.mu.Lock()
	delete(b.clients, ch)
	b.mu.Unlock()
	close(ch)
}

func (b *SSEBroker) Broadcast(payload string) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.clients {
		select {
		case ch <- payload:
		default: // drop if client is slow
		}
	}
}

// broadcaster pushes state to all SSE clients whenever metrics change.
func broadcaster(broker *SSEBroker, worker *events.EventWorker) {
	var lastJSON string
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		state := DashboardState{
			Metrics: metrics.Get(),
			Events:  worker.GetRecent(),
			Routes:  config.GetAll(),
		}
		data, _ := json.Marshal(state)
		js := string(data)
		if js == lastJSON {
			continue // skip broadcast — nothing changed
		}
		lastJSON = js
		broker.Broadcast(js)
	}
}

// ---------------------------------------------------------------------------
// CORS helper
// ---------------------------------------------------------------------------

func enableCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
		allowList := strings.Split(allowedOrigins, ",")

		allowed := false
		if origin != "" {
			for _, o := range allowList {
				if strings.TrimSpace(o) == origin {
					allowed = true
					break
				}
			}
		}

		if allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if r.Method == "OPTIONS" {
			if origin != "" && !allowed {
				http.Error(w, "CORS origin not allowed", http.StatusForbidden)
				return
			}
			w.WriteHeader(http.StatusOK)
			return
		}

		if origin != "" && !allowed {
			http.Error(w, "CORS origin not allowed", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}

// ---------------------------------------------------------------------------
// main
// ---------------------------------------------------------------------------

func main() {
	godotenv.Load()
	config.Load()
	worker := events.NewWorker(100)
	worker.Start()

	prx := proxy.NewProxy(worker, 10*time.Second)

	broker := NewSSEBroker()
	go broadcaster(broker, worker)

	apiMux := http.NewServeMux()

	// --- SSE stream: single persistent connection replaces all polling ---
	apiMux.HandleFunc("/api/stream", enableCORS(func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming not supported", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")

		ch := broker.Subscribe()
		defer broker.Unsubscribe(ch)

		// Send current state immediately on connect
		state := DashboardState{
			Metrics: metrics.Get(),
			Events:  worker.GetRecent(),
			Routes:  config.GetAll(),
		}
		if data, err := json.Marshal(state); err == nil {
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}

		for {
			select {
			case <-r.Context().Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				fmt.Fprintf(w, "data: %s\n\n", msg)
				flusher.Flush()
			}
		}
	}))

	// --- Keep REST endpoints (used for mutations from the UI) ---
	apiMux.HandleFunc("/api/metrics", enableCORS(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(metrics.Get())
	}))

	apiMux.HandleFunc("/api/events", enableCORS(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(worker.GetRecent())
	}))

	apiMux.HandleFunc("/api/routes", enableCORS(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			json.NewEncoder(w).Encode(config.GetAll())
			return
		}
		if r.Method == http.MethodPost {
			var input struct {
				Path   string `json:"path"`
				Target string `json:"target"`
			}
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			config.UpdateRoute(input.Path, input.Target)
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method == http.MethodDelete {
			path := r.URL.Query().Get("path")
			if path == "" {
				http.Error(w, "path is required", http.StatusBadRequest)
				return
			}
			config.DeleteRoute(path)
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}))

	// --- Proxy handler wrapped with bot blocklist ---
	apiMux.Handle("/", blocklist.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		prx.ServeHTTP(w, r)
	})))

	log.Println("Proxy running on :8080")
	if err := http.ListenAndServe(":8080", apiMux); err != nil {
		log.Fatal(err)
	}
}
