package events

import (
	"context"
	"log"
	"sync"
)

// EventHandler processes events.
type EventHandler func(context.Context, Event) error

// EventBus manages event distribution across the system.
type EventBus struct {
	mu        sync.RWMutex
	handlers  map[EventType][]EventHandler
	wildcard  []EventHandler // Handlers for all events
	eventChan chan Event
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

// NewEventBus creates a new event bus.
func NewEventBus() *EventBus {
	ctx, cancel := context.WithCancel(context.Background())

	return &EventBus{
		handlers:  make(map[EventType][]EventHandler),
		eventChan: make(chan Event, 10000),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Subscribe registers a handler for a specific event type.
func (eb *EventBus) Subscribe(eventType EventType, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.handlers[eventType] = append(eb.handlers[eventType], handler)
	log.Printf("[EventBus] Handler subscribed to %s", eventType)
}

// SubscribeAll registers a handler for all events.
func (eb *EventBus) SubscribeAll(handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.wildcard = append(eb.wildcard, handler)
}

// Publish sends an event to all subscribers.
func (eb *EventBus) Publish(event Event) error {
	select {
	case eb.eventChan <- event:
		return nil
	case <-eb.ctx.Done():
		return eb.ctx.Err()
	}
}

// Start begins processing events.
func (eb *EventBus) Start() {
	eb.wg.Add(1)
	go eb.processEvents()
	log.Println("[EventBus] Started")
}

// Stop gracefully shuts down the event bus.
func (eb *EventBus) Stop() {
	eb.cancel()
	eb.wg.Wait()
	close(eb.eventChan)
	log.Println("[EventBus] Stopped")
}

func (eb *EventBus) processEvents() {
	defer eb.wg.Done()

	for {
		select {
		case event := <-eb.eventChan:
			eb.dispatch(event)
		case <-eb.ctx.Done():
			return
		}
	}
}

func (eb *EventBus) dispatch(event Event) {
	eb.mu.RLock()
	handlers := eb.handlers[event.Type]
	wildcardHandlers := eb.wildcard
	eb.mu.RUnlock()

	// Dispatch to specific handlers
	for _, handler := range handlers {
		if err := handler(eb.ctx, event); err != nil {
			log.Printf("[EventBus] Handler error for event %s: %v", event.ID, err)
		}
	}

	// Dispatch to wildcard handlers
	for _, handler := range wildcardHandlers {
		if err := handler(eb.ctx, event); err != nil {
			log.Printf("[EventBus] Wildcard handler error: %v", err)
		}
	}
}
