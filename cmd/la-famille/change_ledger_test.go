package main

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tbuddy/la-famille/internal/config"
	sitediff "github.com/tbuddy/la-famille/internal/diff"
	"github.com/tbuddy/la-famille/internal/generator"
	"github.com/tbuddy/la-famille/internal/sitedata"
)

// This is the strict CI self-test: prose passes, synthetic broken links fail,
// and existing debt is ignored, on the issue's actual fixture sites.
func TestChangeLedgerGateFixtures(t *testing.T) {
	for _, fixture := range []string{"anchor-links", "artisanal-ceramics"} {
		t.Run(fixture, func(t *testing.T) {
			root := t.TempDir()
			source := filepath.Join("..", "..", "assets", "testdata", "sites", fixture, "content")
			err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if entry.IsDir() {
					return nil
				}
				rel, err := filepath.Rel(source, path)
				if err != nil {
					return err
				}
				data, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				putChangeFile(t, filepath.Join(root, "content", rel), string(data))
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			cfg, err := config.DefaultConfig().ResolvePaths(root)
			if err != nil {
				t.Fatal(err)
			}
			cfg.Template = filepath.Join(root, "templates", "layout.html")
			cfg.GraphExplorer = false
			putChangeFile(t, cfg.Template, "{{.Content}}")
			putChangeFile(t, filepath.Join(cfg.ContentDir, "existing-orphan.md"), "# Existing orphan\n[Existing broken](old-missing.md)\n")
			if _, err := generator.Build(cfg); err != nil {
				t.Fatal(err)
			}
			baseline, err := os.ReadFile(filepath.Join(cfg.OutputDir, sitedata.ManifestFileName))
			if err != nil {
				t.Fatal(err)
			}
			putChangeFile(t, filepath.Join(root, "before.json"), string(baseline))
			path := filepath.Join(cfg.ContentDir, "index.md")
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			putChangeFile(t, path, string(data)+"\nReworded prose only.\n")
			if _, err := generator.Build(cfg); err != nil {
				t.Fatal(err)
			}
			runGate := func(wantFailure bool) sitediff.Report {
				t.Helper()
				var output bytes.Buffer
				cmd := setupRootCmd(cfg)
				cmd.SetOut(&output)
				cmd.SetArgs([]string{"diff", "before.json", "public", "--gate", "--json", "--report-dir", "reports"})
				err := cmd.Execute()
				if (err != nil) != wantFailure {
					t.Fatalf("gate error = %v, want failure %t\n%s", err, wantFailure, output.String())
				}
				var report struct {
					Changes sitediff.Report `json:"changes"`
				}
				if err := json.Unmarshal(output.Bytes(), &report); err != nil {
					t.Fatalf("invalid gate JSON: %v\n%s", err, output.String())
				}
				for _, name := range []string{sitediff.JSONFileName, sitediff.TextFileName} {
					if _, err := os.Stat(filepath.Join(root, "reports", name)); err != nil {
						t.Fatal(err)
					}
				}
				for _, regression := range report.Changes.Regressions {
					if regression.Page == "existing-orphan" {
						t.Fatal("pre-existing debt caused gate failure")
					}
				}
				return report.Changes
			}
			prose := runGate(false)
			if len(prose.ChangedPages) != 1 || len(prose.NewBrokenLinks) != 0 {
				t.Fatalf("prose diff = %+v", prose)
			}
			putChangeFile(t, path, string(data)+"\n[New broken](new-missing.md)\n")
			if _, err := generator.Build(cfg); err != nil {
				t.Fatal(err)
			}
			broken := runGate(true)
			if len(broken.NewBrokenLinks) != 1 || broken.NewBrokenLinks[0].Destination != "new-missing.md" {
				t.Fatalf("newly broken links = %+v", broken.NewBrokenLinks)
			}
		})
	}
}

func TestDiffSourceGitRefBuildsWithoutCheckoutChanges(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	git := func(args ...string) []byte {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		data, err := cmd.Output()
		if err != nil {
			t.Fatalf("git %v: %v", args, err)
		}
		return data
	}
	before := git("status", "--porcelain")
	cfg, err := config.DefaultConfig().ResolvePaths(root)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := loadDiffInput("ref:HEAD", cfg)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Version != sitedata.ManifestVersion || !manifest.OutputCaptured || len(manifest.Pages) == 0 {
		t.Fatalf("Git snapshot incomplete: %+v", manifest)
	}
	after := git("status", "--porcelain")
	if !bytes.Equal(before, after) {
		t.Fatalf("Git ref comparison changed checkout:\n%s\n%s", before, after)
	}
}

func TestChangesPaneNavigationFilterAndDiagnostics(t *testing.T) {
	root := t.TempDir()
	ledger := sitediff.Ledger{Version: 1, Changes: sitediff.Report{
		ChangedPages: []sitediff.PageChange{
			{After: sitediff.PageRef{Identity: "a"}},
			{After: sitediff.PageRef{Identity: "b"}},
			{After: sitediff.PageRef{Identity: "c"}},
		},
	}}
	if err := sitediff.Write(root, ledger); err != nil {
		t.Fatal(err)
	}
	m := initialModel(config.Config{OutputDir: root})
	m.screen, m.width, m.height = screenChanges, 80, 24
	if !strings.Contains(m.View(), "no regressions, 3 pages changed") {
		t.Fatal(m.View())
	}
	m.width = 70
	narrow := m.View()
	if !strings.Contains(narrow, "Menu") {
		t.Fatal("narrow pane clipped footer")
	}
	if !strings.Contains(narrow, "╯") {
		t.Fatal("narrow pane clipped border")
	}
	for _, line := range strings.Split(narrow, "\n") {
		if lipgloss.Width(line) > m.width {
			t.Fatal("narrow pane exceeds terminal width")
		}
	}
	key := func(value string) {
		t.Helper()
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(value)})
		m = updated.(model)
	}
	key("j")
	if m.changesCursor != 1 {
		t.Fatal("j did not advance")
	}
	key("k")
	if m.changesCursor != 0 {
		t.Fatal("k did not retreat")
	}
	key("r")
	if len(m.changeRows()) != 0 {
		t.Fatal("regression filter included prose")
	}
	key("d")
	if m.screen != screenDiagnostics {
		t.Fatal("diagnostics inaccessible")
	}
	key("d")
	if m.screen != screenChanges {
		t.Fatal("diagnostics did not return")
	}
	key("?")
	key("?")
	if m.screen != screenChanges {
		t.Fatal("help did not return")
	}
	broken := ledger
	broken.Changes.NewBrokenLinks = []sitediff.BrokenLink{{Page: "b", Destination: "missing.md"}}
	broken.Changes.Regressions = []sitediff.Regression{{Kind: "broken_link", Page: "b", Detail: "newly-broken link missing.md"}}
	updated, _ := m.Update(statsUpdateMsg{res: generator.BuildResult{Ledger: &broken}})
	m = updated.(model)
	if !strings.Contains(m.View(), "1 newly-broken link") || len(m.changeRows()) != 1 || m.changeRows()[0].page != "b" {
		t.Fatal(m.View())
	}
	key("q")
	if m.screen != screenMenu {
		t.Fatal("q did not return to menu")
	}
}

func TestChangesPanePreservesLedgerOnBuildError(t *testing.T) {
	m := initialModel(config.Config{OutputDir: t.TempDir()})
	original := &sitediff.Ledger{Version: 1}
	m.ledger = original
	updated, _ := m.Update(workResultMsg{err: os.ErrNotExist, res: &generator.BuildResult{Ledger: &sitediff.Ledger{Version: 1, Baseline: true}}})
	if updated.(model).ledger != original {
		t.Fatal("failed build replaced successful ledger")
	}
}

func putChangeFile(t *testing.T, path, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
}
