package config

import (
	"encoding/json"
	"log"
	"os"
	"sync"
)

const configFile = "routes.json"

type RouteMap struct {
	mu     sync.RWMutex
	routes map[string]string // e.g. "/users" -> "https://jsonplaceholder.typicode.com"
}

var currentRoutes = &RouteMap{
	routes: make(map[string]string),
}

func GetRoute(path string) (string, bool) {
	currentRoutes.mu.RLock()
	defer currentRoutes.mu.RUnlock()
	target, ok := currentRoutes.routes[path]
	return target, ok
}

func UpdateRoute(path, target string) {
	currentRoutes.mu.Lock()
	currentRoutes.routes[path] = target
	currentRoutes.mu.Unlock()
	Save()
}

func GetAll() map[string]string {
	currentRoutes.mu.RLock()
	defer currentRoutes.mu.RUnlock()
	res := make(map[string]string)
	for k, v := range currentRoutes.routes {
		res[k] = v
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
	data, err := json.MarshalIndent(currentRoutes.routes, "", "  ")
	if err != nil {
		log.Printf("Error marshaling routes: %v", err)
		return
	}
	if err := os.WriteFile(configFile, data, 0644); err != nil {
		log.Printf("Error writing routes file: %v", err)
	}
}

func Load() {
	data, err := os.ReadFile(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		log.Printf("Error reading routes file: %v", err)
		return
	}
	currentRoutes.mu.Lock()
	defer currentRoutes.mu.Unlock()
	if err := json.Unmarshal(data, &currentRoutes.routes); err != nil {
		log.Printf("Error unmarshaling routes: %v", err)
	}

	// Add default routes if none exist
	if len(currentRoutes.routes) == 0 {
		currentRoutes.routes["/posts"] = "https://jsonplaceholder.typicode.com/posts"
		currentRoutes.routes["/users"] = "https://jsonplaceholder.typicode.com/users"
		currentRoutes.routes["/todos"] = "https://jsonplaceholder.typicode.com/todos"
		Save()
	}
}
