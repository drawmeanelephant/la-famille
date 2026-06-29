package watcher

import (
	"fmt"
	"net/http"
	"sync"
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

	// Wait for a message or client disconnect
	for {
		select {
		case <-clientChan:
			fmt.Fprintf(w, "data: reload\n\n")
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
