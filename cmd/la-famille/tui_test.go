package main

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
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
		resp, err := http.Get(serverURL) // #nosec G107 - Test URL is internal/local
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

	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
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

func TestRunServerReportsStartupError(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Reserve port: %v", err)
	}
	defer listener.Close()

	server := &http.Server{
		Addr:              listener.Addr().String(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	reported := make(chan tea.Msg, 1)
	runServer(server, func(msg tea.Msg) {
		reported <- msg
	})

	select {
	case msg := <-reported:
		serverErr, ok := msg.(serverErrorMsg)
		if !ok {
			t.Fatalf("Reported %T, want serverErrorMsg", msg)
		}
		if serverErr.err == nil {
			t.Fatal("Expected ListenAndServe error")
		}
	case <-time.After(time.Second):
		t.Fatal("ListenAndServe startup error was not reported")
	}
}

func TestTUIServerErrorReturnsToVisibleErrorState(t *testing.T) {
	serverCanceled := false
	watcherCanceled := false
	wantErr := errors.New("address already in use")
	m := initialModel(config.Config{})
	m.screen = screenServe
	m.server = &http.Server{ReadHeaderTimeout: 5 * time.Second}
	m.serverCancel = func() { serverCanceled = true }
	m.watcherCancel = func() { watcherCanceled = true }

	newModel, _ := m.Update(serverErrorMsg{err: wantErr})
	m = newModel.(model)

	if m.screen != screenWorking {
		t.Fatalf("Screen = %v, want %v", m.screen, screenWorking)
	}
	if !errors.Is(m.workErr, wantErr) {
		t.Fatalf("workErr = %v, want %v", m.workErr, wantErr)
	}
	if m.server != nil || m.serverCancel != nil || m.watcherCancel != nil {
		t.Fatal("Expected server lifecycle fields to be cleared")
	}
	if !serverCanceled || !watcherCanceled {
		t.Fatalf("Expected cancels to run, server=%t watcher=%t", serverCanceled, watcherCanceled)
	}
	if !strings.Contains(m.View(), wantErr.Error()) {
		t.Fatalf("Error view does not include server error: %q", m.View())
	}
}
