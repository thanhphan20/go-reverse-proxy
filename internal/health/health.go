package health

import (
	"net/http"
	"sync"
	"time"

	"github.com/user/go-reverse-proxy/internal/config"
)

var healthStatus sync.Map // string -> bool

func StartHealthChecks() {
	ticker := time.NewTicker(config.GetConfig().Health.Interval)
	for range ticker.C {
		routes := config.GetAll()
		for _, targets := range routes {
			for _, target := range targets {
				go checkHealth(target)
			}
		}
	}
}

func checkHealth(target string) {
	client := &http.Client{Timeout: config.GetConfig().Health.Timeout}
	resp, err := client.Get(target)
	healthy := err == nil && resp != nil && resp.StatusCode < 400
	if resp != nil {
		resp.Body.Close()
	}
	healthStatus.Store(target, healthy)
}

func IsHealthy(target string) bool {
	if val, ok := healthStatus.Load(target); ok {
		return val.(bool)
	}
	return true // assume healthy if not checked
}

func GetHealth() map[string]bool {
	m := make(map[string]bool)
	healthStatus.Range(func(key, value interface{}) bool {
		m[key.(string)] = value.(bool)
		return true
	})
	return m
}
