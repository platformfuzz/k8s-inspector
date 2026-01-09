package server

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/platformfuzz/k8s-inspector/internal/dashboard"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for simplicity
		// In production, you should validate the origin
		return true
	},
}

// WebSocketHandler handles WebSocket connections for real-time updates
func WebSocketHandler(dash *dashboard.Dashboard, refreshInterval int) gin.HandlerFunc {
	return func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("WebSocket upgrade error: %v", err)
			return
		}
		defer func() {
			if closeErr := conn.Close(); closeErr != nil {
				log.Printf("Failed to close WebSocket connection: %v", closeErr)
			}
		}()

		// Send initial data
		data := dash.GetData()
		if err := conn.WriteJSON(data); err != nil {
			log.Printf("WebSocket write error: %v", err)
			return
		}

		// Set up ticker for periodic updates
		ticker := time.NewTicker(time.Duration(refreshInterval) * time.Second)
		defer ticker.Stop()

		// Set up ping/pong to keep connection alive
		if err := conn.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
			log.Printf("Failed to set read deadline: %v", err)
			return
		}
		conn.SetPongHandler(func(string) error {
			if err := conn.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
				log.Printf("Failed to set read deadline: %v", err)
			}
			return nil
		})

		// Start ping ticker
		pingTicker := time.NewTicker(54 * time.Second)
		defer pingTicker.Stop()

		// Channel for handling client messages
		done := make(chan struct{})

		// Read messages from client (for closing connection)
		go func() {
			defer close(done)
			for {
				_, _, err := conn.ReadMessage()
				if err != nil {
					if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
						log.Printf("WebSocket error: %v", err)
					}
					break
				}
			}
		}()

		for {
			select {
			case <-done:
				return

			case <-ticker.C:
				// Send updated data
				data := dash.GetData()
				if err := conn.WriteJSON(data); err != nil {
					log.Printf("WebSocket write error: %v", err)
					return
				}

			case <-pingTicker.C:
				// Send ping to keep connection alive
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			}
		}
	}
}
