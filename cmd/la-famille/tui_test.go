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
	"github.com/tbuddy/la-famille/internal/generator"
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

func TestTUIStatsCacheStatus(t *testing.T) {
	m := initialModel(config.Config{})
	m.screen = screenStats

	// Test Cache Status: Hit
	m.stats = &generator.BuildResult{
		Duration:   10 * time.Millisecond,
		PageCount:  5,
		ErrorCount: 0,
		CacheHit:   true,
	}
	viewHit := m.View()
	if !strings.Contains(viewHit, "Cache Status: Hit") {
		t.Errorf("Expected view to contain 'Cache Status: Hit', got: %s", viewHit)
	}

	// Test Cache Status: Miss
	m.stats = &generator.BuildResult{
		Duration:   10 * time.Millisecond,
		PageCount:  5,
		ErrorCount: 0,
		CacheHit:   false,
	}
	viewMiss := m.View()
	if !strings.Contains(viewMiss, "Cache Status: Miss") {
		t.Errorf("Expected view to contain 'Cache Status: Miss', got: %s", viewMiss)
	}
}

func TestTUICommandMenuOpenNavigationAndEscape(t *testing.T) {
	m := initialModel(config.Config{})
	if !m.menuOpen {
		t.Fatal("command menu should be open initially")
	}
	start := m.cursor
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = newModel.(model)
	if m.cursor != start+1 {
		t.Fatalf("cursor = %d, want %d after down", m.cursor, start+1)
	}

	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = newModel.(model)
	if m.menuOpen {
		t.Fatal("escape should close the command menu")
	}
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	m = newModel.(model)
	if !m.menuOpen {
		t.Fatal("m should reopen the command menu")
	}
}

func TestTUICommandMenuSelection(t *testing.T) {
	m := initialModel(config.Config{})
	for i, choice := range m.choices {
		if choice.label == "Diagnostics" {
			m.cursor = i
			break
		}
	}
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(model)
	if m.screen != screenDiagnostics {
		t.Fatalf("screen = %v, want diagnostics after selecting command", m.screen)
	}

	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = newModel.(model)
	if m.screen != screenMenu || !m.menuOpen {
		t.Fatalf("escape should return to open menu, screen=%v open=%t", m.screen, m.menuOpen)
	}
}

func TestTUICommandMenuToggleWatch(t *testing.T) {
	m := initialModel(config.Config{})
	for i, choice := range m.choices {
		if choice.label == "Toggle Watch Mode" {
			m.cursor = i
			break
		}
	}
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(model)
	if !m.cfg.WatchMode {
		t.Fatal("watch mode should be enabled after first toggle")
	}
}

func TestTUIBuildProgressTransitions(t *testing.T) {
	m := initialModel(config.Config{})
	m.screen = screenWorking
	m.workTotal = 4

	for _, progress := range []workProgressMsg{
		{phase: "Preparing build", completed: 1, total: 4},
		{phase: "Rendering pages", completed: 2, total: 4},
		{phase: "Writing assets and indexes", completed: 3, total: 4},
	} {
		updated, _ := m.Update(progress)
		m = updated.(model)
		if m.workPhase != progress.phase || m.workCompleted != progress.completed || m.workTotal != progress.total {
			t.Fatalf("progress state = (%q, %d/%d), want (%q, %d/%d)", m.workPhase, m.workCompleted, m.workTotal, progress.phase, progress.completed, progress.total)
		}
	}
	view := m.View()
	if !strings.Contains(view, "Phase: Writing assets and indexes (3/4)") {
		t.Fatalf("progress view missing current phase: %q", view)
	}
}

func TestTUIBuildProgressCompletionAndWarning(t *testing.T) {
	m := initialModel(config.Config{})
	m.screen = screenWorking
	m.workTotal = 4
	updated, _ := m.Update(workResultMsg{
		msg: "Build complete (cache miss)",
		res: &generator.BuildResult{ErrorCount: 2},
	})
	m = updated.(model)
	if m.workPhase != "Complete" || m.workCompleted != 4 {
		t.Fatalf("completion state = (%q, %d/%d), want Complete 4/4", m.workPhase, m.workCompleted, m.workTotal)
	}
	view := m.View()
	if !strings.Contains(view, "Warning: 2 build errors reported") {
		t.Fatalf("completion view missing warning: %q", view)
	}
}

func TestTUIBuildProgressFailure(t *testing.T) {
	m := initialModel(config.Config{})
	m.screen = screenWorking
	m.workTotal = 4
	wantErr := errors.New("render failed")
	updated, _ := m.Update(workResultMsg{err: wantErr, msg: "Build failed"})
	m = updated.(model)
	if m.workPhase != "Build failed" || m.workCompleted != 4 {
		t.Fatalf("failure state = (%q, %d/%d), want Build failed 4/4", m.workPhase, m.workCompleted, m.workTotal)
	}
	if !strings.Contains(m.View(), "Error: render failed") {
		t.Fatalf("failure view missing error: %q", m.View())
	}
}
