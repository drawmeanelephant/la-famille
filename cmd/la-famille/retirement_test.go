package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tbuddy/la-famille/internal/config"
)

func TestRetirementCommandRegistration(t *testing.T) {
	root := setupRootCmd(config.DefaultConfig())
	for _, cmd := range root.Commands() {
		if cmd.Name() == "ask" {
			t.Fatal("retired command is still registered")
		}
	}
	for _, name := range []string{"build", "serve", "check", "rag", "pack", "tui", "themes"} {
		cmd, _, err := root.Find([]string{name})
		if err != nil || cmd.Name() != name {
			t.Fatalf("preserved command %q unavailable: %v", name, err)
		}
	}
	var help bytes.Buffer
	root.SetOut(&help)
	root.SetArgs([]string{"--help"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(help.String(), "ask") || strings.Contains(help.String(), "Ollama") {
		t.Fatalf("help advertises the retired assistant:\n%s", &help)
	}
}

func TestRetirementUnsupportedCommand(t *testing.T) {
	dir := t.TempDir()
	binary := sharedGateBinary()
	// Unknown-command errors must remain clear even with unusable site config.
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("port: [broken"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"ask"}, {"ask", "--help"}, {"ask", "--pack", "corpus.tar", "--provider", "fake"}} {
		cmd := exec.Command(binary, args...)
		cmd.Dir = dir
		output, err := cmd.CombinedOutput()
		message := strings.ReplaceAll(string(output), `\"`, `"`)
		if err == nil || !strings.Contains(message, `unknown command "ask"`) {
			t.Fatalf("%v: error = %v, output = %s", args, err, output)
		}
	}
}

func TestRetirementTUIMenu(t *testing.T) {
	m := initialModel(config.DefaultConfig())
	var labels []string
	for _, choice := range m.choices {
		labels = append(labels, choice.label)
	}
	want := []string{"Build Site", "Serve Site", "Toggle Watch Mode", "Stats", "Changes", "Diagnostics", "RAG Export", "Help", "Just Raoul"}
	if !reflect.DeepEqual(labels, want) {
		t.Fatalf("menu = %v, want %v", labels, want)
	}
	if strings.Contains(m.View(), "Ask") {
		t.Fatal("menu still advertises the assistant")
	}
	// Navigate through the former assistant position to Help, then back.
	for range 7 {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = updated.(model)
	}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)
	if m.screen != screenHelp {
		t.Fatalf("selection after RAG Export = %v, want Help", m.screen)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if updated.(model).screen != screenMenu {
		t.Fatal("Help did not return to the menu")
	}
}

func TestRetirementTUIRAGExport(t *testing.T) {
	cfg := setupValidTestConfig(t, 0)
	cfg.ProjectRoot = filepath.Dir(cfg.ContentDir)
	m := initialModel(cfg)
	for i, choice := range m.choices {
		if choice.label == "RAG Export" {
			m.cursor = i
		}
	}
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)
	if m.screen != screenWorking || cmd == nil {
		t.Fatal("RAG Export did not start")
	}
	var result *workResultMsg
	for _, work := range cmd().(tea.BatchMsg) {
		if msg, ok := work().(workResultMsg); ok {
			result = &msg
		}
	}
	if result == nil || result.err != nil {
		t.Fatalf("RAG Export result = %+v", result)
	}
	updated, _ = m.Update(*result)
	m = updated.(model)
	if !m.workDone() || m.workMsg != "RAG Export complete" {
		t.Fatalf("RAG Export did not complete: %+v", m)
	}
	for _, name := range []string{"rag-system.md", "rag-config.md", "rag-content.md"} {
		if _, err := os.Stat(filepath.Join(cfg.RagDir, name)); err != nil {
			t.Fatalf("missing archive %s: %v", name, err)
		}
	}
	content, err := os.ReadFile(filepath.Join(cfg.RagDir, "rag-content.md"))
	if err != nil || !strings.Contains(string(content), "<file path=\"content/index.md\">\n<content>\n") {
		t.Fatalf("RAG archive format changed: %s, %v", content, err)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if updated.(model).screen != screenMenu {
		t.Fatal("completed RAG Export did not return to the menu")
	}
}

func TestRetirementModelFreeWorkflowPreservesUserData(t *testing.T) {
	root, home, emptyPath := t.TempDir(), t.TempDir(), t.TempDir()
	binary := sharedGateBinary()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(binary, args...)
		cmd.Dir = root
		// No external executable, including a model runtime, is available.
		cmd.Env = append(os.Environ(), "PATH="+emptyPath, "HOME="+home, "XDG_CACHE_HOME="+filepath.Join(home, ".cache"))
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %v\n%s", args, err, output)
		}
	}
	run("init")
	legacy := []string{
		filepath.Join(root, ".la-famille-vectors.json"),
		filepath.Join(root, "existing-corpus.tar"),
		filepath.Join(home, ".cache", "la-famille", "ask-eval", "existing.json"),
		filepath.Join(home, ".ollama", "models", "existing-model"),
	}
	const sentinel = "existing user data, not a valid embedding cache"
	for _, name := range legacy {
		if err := os.MkdirAll(filepath.Dir(name), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(name, []byte(sentinel), 0600); err != nil {
			t.Fatal(err)
		}
	}
	run("build")
	run("check")
	run("rag")
	run("publish-check")
	run("pack", "build", "--output", "corpus.tar")
	run("pack", "verify", "corpus.tar")
	for _, name := range []string{"search.json", "meta.json", "graph.json", "assets/js/search.js", "assets/css/search.css"} {
		if _, err := os.Stat(filepath.Join(root, "public", name)); err != nil {
			t.Fatalf("missing preserved search artifact %s: %v", name, err)
		}
	}
	for _, name := range legacy {
		data, err := os.ReadFile(name)
		if err != nil || string(data) != sentinel {
			t.Fatalf("existing user data changed at %s: %q, %v", name, data, err)
		}
	}
}
