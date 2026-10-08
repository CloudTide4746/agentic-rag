package event

// Handler processes one decoded event. AMQP delivery and acknowledgement
// remain in the messaging adapter.
type Handler interface {
	Handle(Envelope) error
}

// EventHandler is the default no-op handler kept for wiring and tests.
type EventHandler struct {
	HandleFunc func(Envelope) error
}

func (h *EventHandler) Handle(envelope Envelope) error {
	if h == nil || h.HandleFunc == nil {
		return nil
	}
	return h.HandleFunc(envelope)
}

func NewEventHandler(handle func(Envelope) error) *EventHandler {
	return &EventHandler{HandleFunc: handle}
}
