package config

import (
	"encoding/json"
	"log/slog"
	"os"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

const configFile = "config.yaml"
const oldConfigFile = "routes.json"

type Config struct {
	Routes map[string][]string `yaml:"routes"`
	Proxy  struct {
		Timeout struct {
			Dial            time.Duration `yaml:"dial"`
			ResponseHeader  time.Duration `yaml:"response_header"`
		} `yaml:"timeout"`
		Retries int `yaml:"retries"`
	} `yaml:"proxy"`
	Cache struct {
		TTL        time.Duration `yaml:"ttl"`
		SkipErrors bool          `yaml:"skip_errors"`
	} `yaml:"cache"`
	RateLimit struct {
		Requests int           `yaml:"requests"`
		Window   time.Duration `yaml:"window"`
		PerIP    bool          `yaml:"per_ip"`
	} `yaml:"rate_limit"`
	Logging struct {
		Level string `yaml:"level"`
	} `yaml:"logging"`
	Health struct {
		Interval time.Duration `yaml:"interval"`
		Timeout  time.Duration `yaml:"timeout"`
	} `yaml:"health"`
}

type RouteMap struct {
	mu     sync.RWMutex
	routes map[string][]string // e.g. "/users" -> ["https://api1.com", "https://api2.com"]
}

var currentRoutes = &RouteMap{
	routes: make(map[string][]string),
}
var currentConfig *Config

func GetConfig() *Config {
	return currentConfig
}

func GetRoute(path string) ([]string, bool) {
	currentRoutes.mu.RLock()
	defer currentRoutes.mu.RUnlock()
	targets, ok := currentRoutes.routes[path]
	return targets, ok
}

func UpdateRoute(path string, targets []string) {
	currentRoutes.mu.Lock()
	currentRoutes.routes[path] = targets
	currentRoutes.mu.Unlock()
	Save()
}

func GetAll() map[string][]string {
	currentRoutes.mu.RLock()
	defer currentRoutes.mu.RUnlock()
	res := make(map[string][]string)
	for k, v := range currentRoutes.routes {
		res[k] = append([]string(nil), v...)
	}
	return res
}

func DeleteRoute(path string) {
	currentRoutes.mu.Lock()
	delete(currentRoutes.routes, path)
	currentRoutes.mu.Unlock()
	Save()
}

func Save() {
	currentRoutes.mu.RLock()
	defer currentRoutes.mu.RUnlock()
	currentConfig.Routes = make(map[string][]string)
	for k, v := range currentRoutes.routes {
		currentConfig.Routes[k] = append([]string(nil), v...)
	}
	data, err := yaml.Marshal(currentConfig)
	if err != nil {
		slog.Error("Error marshaling config", "error", err)
		return
	}
	if err := os.WriteFile(configFile, data, 0644); err != nil {
		slog.Error("Error writing config file", "error", err)
	}
}

func Load() {
	data, err := os.ReadFile(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Try old routes.json
			loadOldConfig()
			return
		}
		slog.Error("Error reading config file", "error", err)
		return
	}
	currentConfig = &Config{}
	if err := yaml.Unmarshal(data, currentConfig); err != nil {
		slog.Error("Error unmarshaling config", "error", err)
		return
	}
	currentRoutes.mu.Lock()
	currentRoutes.routes = make(map[string][]string)
	for k, v := range currentConfig.Routes {
		currentRoutes.routes[k] = append([]string(nil), v...)
	}
	currentRoutes.mu.Unlock()

	// Set defaults if missing
	setDefaults()
}

func loadOldConfig() {
	data, err := os.ReadFile(oldConfigFile)
	if err != nil {
		slog.Error("Error reading old routes file", "error", err)
		return
	}
	oldRoutes := make(map[string]string)
	if err := json.Unmarshal(data, &oldRoutes); err != nil {
		slog.Error("Error unmarshaling old routes", "error", err)
		return
	}
	currentConfig = &Config{}
	currentRoutes.mu.Lock()
	currentRoutes.routes = make(map[string][]string)
	for k, v := range oldRoutes {
		currentRoutes.routes[k] = []string{v}
	}
	currentRoutes.mu.Unlock()
	setDefaults()
	Save() // Save as new config
}

func setDefaults() {
	if currentConfig.Proxy.Timeout.Dial == 0 {
		currentConfig.Proxy.Timeout.Dial = 30 * time.Second
	}
	if currentConfig.Proxy.Timeout.ResponseHeader == 0 {
		currentConfig.Proxy.Timeout.ResponseHeader = 30 * time.Second
	}
	if currentConfig.Proxy.Retries == 0 {
		currentConfig.Proxy.Retries = 3
	}
	if currentConfig.Cache.TTL == 0 {
		currentConfig.Cache.TTL = 10 * time.Second
	}
	if currentConfig.Cache.SkipErrors == false {
		currentConfig.Cache.SkipErrors = true
	}
	if currentConfig.RateLimit.Requests == 0 {
		currentConfig.RateLimit.Requests = 20
	}
	if currentConfig.RateLimit.Window == 0 {
		currentConfig.RateLimit.Window = 10 * time.Second
	}
	if currentConfig.Logging.Level == "" {
		currentConfig.Logging.Level = "info"
	}
	if currentConfig.Health.Interval == 0 {
		currentConfig.Health.Interval = 30 * time.Second
	}
	if currentConfig.Health.Timeout == 0 {
		currentConfig.Health.Timeout = 5 * time.Second
	}
}
