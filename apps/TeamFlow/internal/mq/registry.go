package mq

import (
	"fmt"
	"sync"

	"teamflow/internal/event"
)

type Registry struct {
	mu       sync.RWMutex
	handlers map[string]event.Handler
}

var defaultRegistry = NewRegistry()

func NewRegistry() *Registry { return &Registry{handlers: make(map[string]event.Handler)} }

// Register/Get keep the original package-level API for simple consumers.
func Register(eventType string, handler event.Handler) error {
	return defaultRegistry.Register(eventType, handler)
}
func Get(eventType string) (event.Handler, error) { return defaultRegistry.Get(eventType) }

func (r *Registry) Register(eventType string, handler event.Handler) error {
	if r == nil || eventType == "" || handler == nil {
		return fmt.Errorf("event type and handler are required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.handlers[eventType]; exists {
		return fmt.Errorf("handler already registered for event type %q", eventType)
	}
	r.handlers[eventType] = handler
	return nil
}

func (r *Registry) Get(eventType string) (event.Handler, error) {
	if r == nil {
		return nil, fmt.Errorf("registry is nil")
	}
	r.mu.RLock()
	handler, ok := r.handlers[eventType]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("handler not found for event type %q", eventType)
	}
	return handler, nil
}
