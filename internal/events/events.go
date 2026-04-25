package events

import (
	"sync"
)

type Event struct {
	Path      string `json:"path"`
	Status    int    `json:"status"`
	Latency   int64  `json:"latency"`
	FromCache bool   `json:"from_cache"`
	RequestID string `json:"request_id"`
}

type EventWorker struct {
	ch       chan Event
	buffer   []Event
	limit    int
	mu       sync.RWMutex
}

func NewWorker(limit int) *EventWorker {
	return &EventWorker{
		ch:     make(chan Event, 100),
		buffer: make([]Event, 0, limit),
		limit:  limit,
	}
}

func (w *EventWorker) Start() {
	go func() {
		for ev := range w.ch {
			w.mu.Lock()
			if len(w.buffer) >= w.limit {
				w.buffer = w.buffer[1:] // simple ring buffer behavior
			}
			w.buffer = append(w.buffer, ev)
			w.mu.Unlock()
		}
	}()
}

func (w *EventWorker) Emit(ev Event) {
	select {
	case w.ch <- ev:
	default:
		// drop event if channel is full
	}
}

func (w *EventWorker) GetRecent() []Event {
	w.mu.RLock()
	defer w.mu.RUnlock()
	res := make([]Event, len(w.buffer))
	copy(res, w.buffer)
	// reverse to show newest first
	for i, j := 0, len(res)-1; i < j; i, j = i+1, j-1 {
		res[i], res[j] = res[j], res[i]
	}
	return res
}
