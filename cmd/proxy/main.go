package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"github.com/user/go-reverse-proxy/internal/config"
	"github.com/user/go-reverse-proxy/internal/events"
	"github.com/user/go-reverse-proxy/internal/metrics"
	"github.com/user/go-reverse-proxy/internal/proxy"
)

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

func main() {
	godotenv.Load()
	config.Load()
	worker := events.NewWorker(100)
	worker.Start()

	prx := proxy.NewProxy(worker, 10*time.Second)

	apiMux := http.NewServeMux()
	
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

	apiMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		prx.ServeHTTP(w, r)
	})

	// Routes are loaded from config.Load()

	log.Println("Proxy running on :8080")
	if err := http.ListenAndServe(":8080", apiMux); err != nil {
		log.Fatal(err)
	}
}
