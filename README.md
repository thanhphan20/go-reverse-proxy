# Go Reverse Proxy

High-performance Go reverse proxy server with integrated Next.js dashboard. Manage dynamic routing, monitor real-time traffic, and cache GET requests automatically.

## Quick Start

Start proxy backend (runs on port 8080):
```bash
go run cmd/proxy/main.go
```

Start UI dashboard (runs on port 3000):
```bash
cd ui
pnpm install
pnpm run dev
```

## Features

### 1. Dynamic Routing
- Proxy routes traffic based on URL prefix matching.
- Add, edit, or delete routes via UI dashboard. No hardcoding.
- Routes persist to `routes.json` file automatically.
- Fallback default routes created if `routes.json` empty.

### 2. TTL Request Caching
- Intercepts and caches HTTP GET requests in memory.
- Responses stored with headers and body.
- Subsequent identical requests served from RAM. Fast response, zero upstream load.
- Configurable Time-To-Live (TTL) expiry.

### 3. Real-Time Telemetry & Events
- **Metrics**: Track total requests, error rates, cache hits, and cache misses.
- **Event Worker**: Background goroutine buffers last 100 HTTP request logs.
- Data kept in memory. Reset on server restart.

### 4. Interactive UI Dashboard
- Built with Next.js and Tailwind CSS.
- Live-polling stat cards and scrolling event feed.
- **Proxy Terminal**: Built-in tool to run GET requests against proxy endpoints directly from browser.

## Configuration

### Backend Config
- `routes.json`: Auto-generated file in root directory. Stores routing map. Edit via UI or directly.

### Frontend Config
Create `ui/.env.local`:
```env
NEXT_PUBLIC_API_URL=http://localhost:8080
```
Change this if proxy runs on different host/port.

## API Reference

Base URL: `http://localhost:8080`

### GET `/api/metrics`
Return current server metrics.
```json
{
  "total_requests": 150,
  "error_count": 2,
  "cache_hits": 45,
  "cache_misses": 105
}
```

### GET `/api/events`
Return array of recent proxy events.
```json
[
  {
    "path": "/api/users",
    "status": 200,
    "latency": 45,
    "from_cache": true
  }
]
```

### GET `/api/routes`
Return current dynamic route mappings.
```json
{
  "/posts": "https://jsonplaceholder.typicode.com/posts"
}
```

### POST `/api/routes`
Create or update route.
```json
// Request Body
{
  "path": "/test",
  "target": "https://example.com"
}
```

### DELETE `/api/routes?path=/test`
Remove specified route from memory and disk.

## Architecture

- `cmd/proxy/main.go`: Entry point. Setup HTTP handlers, init workers, load config.
- `internal/config`: Handle `routes.json` load/save and mutex locking for map access.
- `internal/proxy`: Core reverse proxy logic. Override `Director` to rewrite `Host` header. Handle GET cache.
- `internal/metrics`: Thread-safe counters.
- `internal/events`: Background worker with circular buffer for request logs.

## License

MIT
