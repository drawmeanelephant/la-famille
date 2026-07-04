package main

import (
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tbuddy/la-famille/internal/config"
)

func getFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func TestTUIServeShutdownAndRestart(t *testing.T) {
	port, err := getFreePort()
	if err != nil {
		t.Fatalf("Failed to get free port: %v", err)
	}

	cfg := config.Config{
		ContentDir: "content",
		OutputDir:  "public",
		Template:   "templates/layout.html",
		AssetDir:   "assets",
		RagDir:     "rag-archive",
		Port:       port,
	}

	m := initialModel(cfg)

	var serveIdx int
	for i, choice := range m.choices {
		if choice.label == "Serve Site" {
			serveIdx = i
			break
		}
	}
	m.cursor = serveIdx

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	m = newModel.(model)

	if cmd != nil {
		cmd()
	}

	serverURL := fmt.Sprintf("http://127.0.0.1:%d/", port)

	up := false
	for i := 0; i < 20; i++ {
		resp, err := http.Get(serverURL)
		if err == nil {
			resp.Body.Close()
			up = true
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if !up {
		t.Fatalf("Server never started listening on port %d", port)
	}

	newModel, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = newModel.(model)

	if m.screen != screenMenu {
		t.Fatalf("Expected screenMenu after 'q', got %v", m.screen)
	}

	addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatalf("Failed to resolve addr: %v", err)
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		t.Fatalf("Failed to bind to port %d, server probably didn't shutdown cleanly: %v", port, err)
	}
	l.Close()
}
