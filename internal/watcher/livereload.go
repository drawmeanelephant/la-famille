package watcher

import (
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

var (
	clients   = make(map[chan struct{}]bool)
	clientsMu sync.Mutex
)

// LiveReloadHandler handles SSE connections from the browser.
func LiveReloadHandler(w http.ResponseWriter, r *http.Request) {
	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	// Allow CORS just in case
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create a channel for this client
	clientChan := make(chan struct{})

	clientsMu.Lock()
	clients[clientChan] = true
	clientsMu.Unlock()

	defer func() {
		clientsMu.Lock()
		delete(clients, clientChan)
		clientsMu.Unlock()
	}()

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	// The serve command sets a server-wide WriteTimeout, which covers this
	// whole response. Left in place it kills an idle stream, and any reload
	// broadcast during the browser's reconnect gap is lost.
	if err := http.NewResponseController(w).SetWriteDeadline(time.Time{}); err != nil {
		slog.Warn("Live reload stream keeps the server write deadline", "error", err)
	}

	// Commit the response head straight away: the browser only considers the
	// EventSource open once the headers arrive, and an idle stream would
	// otherwise send nothing at all.
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	// Wait for a message or client disconnect
	for {
		select {
		case <-clientChan:
			if _, err := fmt.Fprintf(w, "data: reload\n\n"); err != nil {
				// The stream is broken; drop the client so the browser can
				// reconnect instead of us swallowing further broadcasts.
				return
			}
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

// BroadcastReload sends a reload signal to all connected SSE clients.
func BroadcastReload() {
	clientsMu.Lock()
	defer clientsMu.Unlock()
	for clientChan := range clients {
		// Non-blocking send
		select {
		case clientChan <- struct{}{}:
		default:
		}
	}
}
