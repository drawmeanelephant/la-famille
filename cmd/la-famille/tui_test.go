package main

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/generator"
)

func getFreePort() (int, error) {
	for i := 0; i < 20; i++ {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			continue
		}
		port := l.Addr().(*net.TCPAddr).Port
		_ = l.Close()
		time.Sleep(20 * time.Millisecond)

		l2, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err == nil {
			_ = l2.Close()
			return port, nil
		}
	}
	return 0, errors.New("failed to find free port")
}

func setupValidTestConfig(t *testing.T, port int) config.Config {
	t.Helper()
	tmpDir := t.TempDir()

	contentDir := filepath.Join(tmpDir, "content")
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatalf("Failed to create content dir: %v", err)
	}
	mdContent := []byte("---\ntitle: Test Page\n---\n# Hello World\n")
	if err := os.WriteFile(filepath.Join(contentDir, "index.md"), mdContent, 0600); err != nil {
		t.Fatalf("Failed to write index.md: %v", err)
	}

	templateDir := filepath.Join(tmpDir, "templates")
	if err := os.MkdirAll(templateDir, 0755); err != nil {
		t.Fatalf("Failed to create template dir: %v", err)
	}
	layoutPath := filepath.Join(templateDir, "layout.html")
	htmlContent := []byte("<!DOCTYPE html><html><body>{{.Content}}</body></html>")
	if err := os.WriteFile(layoutPath, htmlContent, 0600); err != nil {
		t.Fatalf("Failed to write layout.html: %v", err)
	}

	outputDir := filepath.Join(tmpDir, "public")
	assetDir := filepath.Join(tmpDir, "assets")
	ragDir := filepath.Join(tmpDir, "rag-archive")
	_ = os.MkdirAll(assetDir, 0755)

	cfg := config.DefaultConfig()
	cfg.ContentDir = contentDir
	cfg.OutputDir = outputDir
	cfg.Template = layoutPath
	cfg.AssetDir = assetDir
	cfg.RagDir = ragDir
	cfg.Port = port
	return cfg
}

func TestTUIServeShutdownAndRestart(t *testing.T) {
	port, err := getFreePort()
	if err != nil {
		t.Fatalf("Failed to get free port: %v", err)
	}

	cfg := setupValidTestConfig(t, port)

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

	if m.screen != screenServe {
		t.Fatalf("Expected screenServe, got %v (workErr=%v)", m.screen, m.workErr)
	}
	if m.server == nil {
		t.Fatalf("Expected m.server != nil")
	}

	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = newModel.(model)

	if m.screen != screenMenu {
		t.Fatalf("Expected screenMenu after 'q', got %v", m.screen)
	}
	if m.server != nil {
		t.Fatalf("Expected m.server == nil post-shutdown")
	}
}

func TestTUIServeInitialBuildFailure(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Template = "nonexistent/layout.html"

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
		t.Errorf("Expected nil tea.Cmd on initial build failure, got %v", cmd)
	}
	if m.screen != screenWorking {
		t.Errorf("Expected screenWorking on initial build failure, got %v", m.screen)
	}
	if m.server != nil {
		t.Errorf("Expected m.server == nil on initial build failure")
	}
	if m.watcherCancel != nil {
		t.Errorf("Expected m.watcherCancel == nil on initial build failure")
	}
	if m.workErr == nil {
		t.Errorf("Expected m.workErr != nil on initial build failure")
	}
}

func TestTUIServeWatchModeEnabled(t *testing.T) {
	port, err := getFreePort()
	if err != nil {
		t.Fatalf("Failed to get free port: %v", err)
	}

	cfg := setupValidTestConfig(t, port)
	cfg.WatchMode = true

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

	if m.screen != screenServe {
		t.Fatalf("Expected screenServe, got %v (workErr: %v)", m.screen, m.workErr)
	}
	if m.server == nil {
		t.Fatalf("Expected m.server != nil when watch mode enabled")
	}
	if m.watcherCancel == nil {
		t.Fatalf("Expected m.watcherCancel != nil when watch mode enabled")
	}

	m.stopServing()
	if m.server != nil || m.watcherCancel != nil || m.serverCancel != nil {
		t.Fatalf("Expected lifecycle fields cleared after stopServing")
	}
}

func TestTUIServeCancellationKeys(t *testing.T) {
	testCases := []struct {
		name       string
		keyMsg     tea.KeyMsg
		wantScreen screen
	}{
		{
			name:       "quit via q key",
			keyMsg:     tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}},
			wantScreen: screenMenu,
		},
		{
			name:       "exit via esc key",
			keyMsg:     tea.KeyMsg{Type: tea.KeyEscape},
			wantScreen: screenMenu,
		},
		{
			name:       "force quit via ctrl+c key",
			keyMsg:     tea.KeyMsg{Type: tea.KeyCtrlC},
			wantScreen: screenServe,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			port, err := getFreePort()
			if err != nil {
				t.Fatalf("Failed to get free port: %v", err)
			}

			cfg := setupValidTestConfig(t, port)
			cfg.WatchMode = true

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

			if m.server == nil || m.watcherCancel == nil {
				t.Fatalf("Failed to start server/watcher")
			}
			if m.screen != screenServe {
				t.Fatalf("Expected screenServe, got %v", m.screen)
			}

			done := make(chan struct{})
			go func() {
				updated, _ := m.Update(tc.keyMsg)
				m = updated.(model)
				close(done)
			}()

			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Fatalf("Key handling/stopServing timed out")
			}

			if tc.keyMsg.Type != tea.KeyCtrlC && m.screen != tc.wantScreen {
				t.Errorf("Screen = %v, want %v", m.screen, tc.wantScreen)
			}

			if m.server != nil {
				t.Errorf("m.server != nil post-shutdown")
			}
			if m.watcherCancel != nil {
				t.Errorf("m.watcherCancel != nil post-shutdown")
			}
		})
	}
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

func TestTUIStatsContentHealthRendering(t *testing.T) {
	m := initialModel(config.Config{})
	m.screen = screenStats

	m.stats = &generator.BuildResult{
		Duration:   15 * time.Millisecond,
		PageCount:  3,
		ErrorCount: 0,
		CacheHit:   true,
		Health: generator.ContentHealth{
			TotalWordCount:  450,
			AvgWordsPerPage: 150.0,
			TopTags: []generator.TagCount{
				{Tag: "go", Count: 3},
				{Tag: "tui", Count: 2},
			},
			OrphanedPages:       []string{"blog/orphan"},
			NodeCount:           5,
			EdgeCount:           4,
			MissingDescriptions: []string{"index", "blog/orphan"},
			MissingDates:        []string{"contact"},
		},
	}

	view := m.View()

	expectedSubstrings := []string{
		"Content Health & Observability",
		"Total Word Count: 450",
		"Average Words per Page: 150.0",
		"Top Tags: go (3), tui (2)",
		"Graph Nodes: 5 | Graph Edges: 4",
		"Orphaned Pages (1): blog/orphan",
		"Missing Descriptions (2): index, blog/orphan",
		"Missing Dates (1): contact",
	}

	for _, expected := range expectedSubstrings {
		if !strings.Contains(view, expected) {
			t.Errorf("Stats view missing expected content health string %q. Full view:\n%s", expected, view)
		}
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

func TestTUIDiagnosticsDrawerNavigationAndClear(t *testing.T) {
	m := initialModel(config.Config{})
	m.addDiagnostic("error", errors.New("content/about.md:12:3: bad link"))
	m.diagnostics = append(m.diagnostics, diagnostic{level: "warning", message: "watcher warning"})
	m.screen = screenStats
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = updated.(model)
	if m.screen != screenDiagnostics || !strings.Contains(m.View(), "content/about.md:12:3") {
		t.Fatalf("diagnostics view missing source: %s", m.View())
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(model)
	if m.diagnosticCursor != 1 {
		t.Fatalf("cursor = %d, want 1", m.diagnosticCursor)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	m = updated.(model)
	if len(m.diagnostics) != 0 || !strings.Contains(m.View(), "No diagnostics recorded") {
		t.Fatalf("clear/empty state failed: %s", m.View())
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	m = updated.(model)
	if m.screen != screenStats {
		t.Fatalf("screen after escape = %v, want stats", m.screen)
	}
}

func TestTUIWorkErrorPopulatesDiagnostics(t *testing.T) {
	m := initialModel(config.Config{})
	wantErr := errors.New("content/index.md:7: render failed")
	updated, _ := m.Update(workResultMsg{err: wantErr, msg: "Build failed"})
	m = updated.(model)
	if len(m.diagnostics) != 1 || m.diagnostics[0].source != "content/index.md:7" {
		t.Fatalf("diagnostics = %#v", m.diagnostics)
	}
}

func TestTUIDashboardLayoutWidths(t *testing.T) {
	m := initialModel(config.Config{
		WatchMode: true,
	})
	m.stats = &generator.BuildResult{
		Duration:   120 * time.Millisecond,
		PageCount:  12,
		ErrorCount: 0,
		CacheHit:   true,
	}

	// 1. Test WindowSizeMsg updates model
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 50, Height: 24})
	mNarrow := updated.(model)
	if mNarrow.width != 50 || mNarrow.height != 24 {
		t.Fatalf("WindowSizeMsg did not set dimensions: width=%d height=%d", mNarrow.width, mNarrow.height)
	}

	// 2. Test narrow view (50 cols)
	narrowView := mNarrow.View()
	if !strings.Contains(narrowView, "OCTOBURGER MENU") {
		t.Errorf("narrow view missing OCTOBURGER MENU header: %s", narrowView)
	}
	if !strings.Contains(narrowView, "DASHBOARD STATUS") {
		t.Errorf("narrow view missing DASHBOARD STATUS header: %s", narrowView)
	}
	if !strings.Contains(narrowView, "Watch Mode:") || !strings.Contains(narrowView, "ENABLED") {
		t.Errorf("narrow view missing watch mode status: %s", narrowView)
	}
	if !strings.Contains(narrowView, "Cache Status:") || !strings.Contains(narrowView, "HIT") {
		t.Errorf("narrow view missing cache status: %s", narrowView)
	}

	// 3. Test wide view (100 cols)
	updatedWide, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	mWide := updatedWide.(model)
	wideView := mWide.View()
	if !strings.Contains(wideView, "OCTOBURGER MENU") {
		t.Errorf("wide view missing OCTOBURGER MENU header: %s", wideView)
	}
	if !strings.Contains(wideView, "DASHBOARD STATUS") {
		t.Errorf("wide view missing DASHBOARD STATUS header: %s", wideView)
	}
	if !strings.Contains(wideView, "12 pages") {
		t.Errorf("wide view missing page count stats: %s", wideView)
	}
}

func TestTUIRecoveryGuidance(t *testing.T) {
	tests := []struct {
		err  error
		want string
	}{
		{errors.New("listen tcp 127.0.0.1:8080: bind: address already in use"), "Port conflict"},
		{errors.New("template: layout.html:12: unclosed tag"), "Template syntax error"},
		{errors.New("yaml: unmarshal error in frontmatter"), "Content syntax error"},
		{errors.New("open public/index.html: no such file or directory"), "Path missing"},
		{errors.New("unknown error"), "Check configuration in config.yaml"},
	}

	for _, tt := range tests {
		got := getRecoveryGuidance(tt.err)
		if !strings.Contains(got, tt.want) {
			t.Errorf("getRecoveryGuidance(%v) = %q, want substring %q", tt.err, got, tt.want)
		}
	}
}

func TestTUIServeScreenDetails(t *testing.T) {
	m := initialModel(config.Config{
		Port:      8088,
		WatchMode: true,
	})
	m.screen = screenServe

	view := m.View()
	if !strings.Contains(view, "http://127.0.0.1:8088") {
		t.Errorf("serve view missing server URL: %s", view)
	}
	if !strings.Contains(view, "Watch Mode: ENABLED (Live Reload active)") {
		t.Errorf("serve view missing watch mode enabled badge: %s", view)
	}
	if !strings.Contains(view, "Server Status: RUNNING") {
		t.Errorf("serve view missing server status badge: %s", view)
	}

	m.cfg.WatchMode = false
	viewDisabled := m.View()
	if !strings.Contains(viewDisabled, "Watch Mode: DISABLED") {
		t.Errorf("serve view missing watch mode disabled badge: %s", viewDisabled)
	}
}

func TestTUIWorkErrorRecoveryGuidanceRendering(t *testing.T) {
	m := initialModel(config.Config{})
	m.screen = screenWorking
	m.workErr = errors.New("address already in use")
	m.workMsg = "Unable to start server"

	view := m.View()
	if !strings.Contains(view, "Error: address already in use") {
		t.Errorf("working view missing error: %s", view)
	}
	if !strings.Contains(view, "Recovery Guidance: Port conflict") {
		t.Errorf("working view missing recovery guidance: %s", view)
	}
	if !strings.Contains(view, "Press Enter or Esc to return") {
		t.Errorf("working view missing return path guidance: %s", view)
	}
}

func TestTUIStatsNextStepGuidance(t *testing.T) {
	m := initialModel(config.Config{})
	m.screen = screenStats

	// 1. With health issues
	m.stats = &generator.BuildResult{
		Duration:   10 * time.Millisecond,
		PageCount:  2,
		ErrorCount: 1,
		Health: generator.ContentHealth{
			OrphanedPages:       []string{"draft"},
			MissingDescriptions: []string{"index"},
			MissingDates:        []string{"about"},
		},
	}

	viewWithIssues := m.View()
	if !strings.Contains(viewWithIssues, "Next-Step Guidance") {
		t.Errorf("stats view missing Next-Step Guidance header: %s", viewWithIssues)
	}
	if !strings.Contains(viewWithIssues, "Orphaned pages detected") {
		t.Errorf("stats view missing orphaned guidance: %s", viewWithIssues)
	}
	if !strings.Contains(viewWithIssues, "Missing descriptions") {
		t.Errorf("stats view missing description guidance: %s", viewWithIssues)
	}
	if !strings.Contains(viewWithIssues, "Missing dates") {
		t.Errorf("stats view missing date guidance: %s", viewWithIssues)
	}
	if !strings.Contains(viewWithIssues, "Build warnings/errors") {
		t.Errorf("stats view missing error guidance: %s", viewWithIssues)
	}

	// 2. Clean health
	m.stats = &generator.BuildResult{
		Duration:   10 * time.Millisecond,
		PageCount:  2,
		ErrorCount: 0,
		Health:     generator.ContentHealth{},
	}
	viewClean := m.View()
	if !strings.Contains(viewClean, "Content health is optimal") {
		t.Errorf("stats view missing optimal health badge: %s", viewClean)
	}
}

func TestTUIDiagnosticsRecoveryActionRendering(t *testing.T) {
	m := initialModel(config.Config{})
	m.screen = screenDiagnostics
	m.diagnostics = []diagnostic{
		{level: "error", message: "content/post.md:5: yaml error", source: "content/post.md:5"},
	}

	view := m.View()
	if !strings.Contains(view, "Diagnostics & Recovery Guidance") {
		t.Errorf("diagnostics view missing title: %s", view)
	}
	if !strings.Contains(view, "Source: content/post.md:5") {
		t.Errorf("diagnostics view missing source line: %s", view)
	}
	if !strings.Contains(view, "Action: Content syntax error") {
		t.Errorf("diagnostics view missing action guidance line: %s", view)
	}
}

func TestTUIHelpScreenFormatting(t *testing.T) {
	m := initialModel(config.Config{})
	m.screen = screenHelp

	view := m.View()
	if !strings.Contains(view, "La Famille Help & Keybindings") {
		t.Errorf("help view missing title: %s", view)
	}
	if !strings.Contains(view, "Navigation & Commands") {
		t.Errorf("help view missing Navigation section: %s", view)
	}
	if !strings.Contains(view, "Global Shortcuts") {
		t.Errorf("help view missing Global Shortcuts section: %s", view)
	}
	if !strings.Contains(view, "Workflow Hints") {
		t.Errorf("help view missing Workflow Hints section: %s", view)
	}
}

func TestTUIReturnPathFromFailedWork(t *testing.T) {
	m := initialModel(config.Config{})
	m.screen = screenWorking
	m.workErr = errors.New("build error")

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	mRes := updated.(model)
	if mRes.screen != screenMenu {
		t.Fatalf("q key on failed work screen = %v, want screenMenu", mRes.screen)
	}

	m.screen = screenWorking
	m.workErr = errors.New("build error")
	updatedEsc, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	mResEsc := updatedEsc.(model)
	if mResEsc.screen != screenMenu {
		t.Fatalf("esc key on failed work screen = %v, want screenMenu", mResEsc.screen)
	}
}

