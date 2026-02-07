package api

import (
	"log/slog"
	"net/http"
	"time"
)

// LogStreamer provides access to broadcast log messages.
type LogStreamer interface {
	Subscribe() chan string
	Unsubscribe(ch chan string)
}

// handleLogStream handles WebSocket connections for streaming logs.
func handleLogStream(streamer LogStreamer, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			logger.Error("WebSocket upgrade failed for log stream", "error", err)
			return
		}
		defer conn.Close()

		// Subscribe to log broadcasts
		logCh := streamer.Subscribe()
		defer streamer.Unsubscribe(logCh)

		// Send initial message
		conn.WriteJSON(map[string]string{"event": "connected"})

		// Context for cleanup
		done := make(chan struct{})

		// Read pump (handle ping/close from client)
		go func() {
			defer close(done)
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					return
				}
			}
		}()

		// Write pump (send logs to client)
		for {
			select {
			case <-done:
				return
			case log, ok := <-logCh:
				if !ok {
					return
				}
				conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := conn.WriteJSON(map[string]string{"event": "log", "data": log}); err != nil {
					return
				}
			}
		}
	}
}
