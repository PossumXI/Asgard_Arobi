// Package realtime provides real-time event broadcasting via WebSocket.
package realtime

import (
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"golang.org/x/net/websocket"
)

// HandleWebSocket handles WebSocket connections for real-time events.
func HandleWebSocket(w http.ResponseWriter, r *http.Request, broadcaster *Broadcaster) {
	websocket.Handler(func(ws *websocket.Conn) {
		client := &Client{
			id:         uuid.New().String(),
			send:       make(chan []byte, 256),
			broadcaster: broadcaster,
		}

		broadcaster.register <- client

		// Send goroutine
		go func() {
			defer ws.Close()
			for {
				select {
				case message, ok := <-client.send:
					if !ok {
						return
					}
					if err := websocket.Message.Send(ws, message); err != nil {
						log.Printf("WebSocket send error: %v", err)
						return
					}
				}
			}
		}()

		// Receive goroutine (for ping/pong)
		go func() {
			defer ws.Close()
			for {
				var msg string
				if err := websocket.Message.Receive(ws, &msg); err != nil {
					break
				}
				// Handle ping/pong or other messages
			}
		}()

		// Keep connection alive
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// Send ping
				if err := websocket.Message.Send(ws, `{"type":"ping"}`); err != nil {
					return
				}
			}
		}
	}).ServeHTTP(w, r)
}
