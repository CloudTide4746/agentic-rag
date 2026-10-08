package mq

import (
	"fmt"
	"testing"

	"teamflow/internal/event"
)

func TestRegistryDispatchesRegisteredHandler(t *testing.T) {
	registry := NewRegistry()
	called := false
	handler := event.NewEventHandler(func(event.Envelope) error {
		called = true
		return nil
	})
	if err := registry.Register(event.TaskAssignedV1, handler); err != nil {
		t.Fatalf("register handler: %v", err)
	}
	received, err := registry.Get(event.TaskAssignedV1)
	if err != nil {
		t.Fatalf("get handler: %v", err)
	}
	if err := received.Handle(event.Envelope{EventType: event.TaskAssignedV1}); err != nil {
		t.Fatalf("handle event: %v", err)
	}
	if !called {
		t.Fatal("registered handler was not called")
	}
}

func TestIsPermanentErrorRecognizesWrappedError(t *testing.T) {
	if !IsPermanentError(fmt.Errorf("decode: %w", ErrPermanent)) {
		t.Fatal("permanent errors must be recognized across wrapping")
	}
}
