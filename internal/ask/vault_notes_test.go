package ask

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/generator"
	"github.com/tbuddy/la-famille/internal/llm"
	"github.com/tbuddy/la-famille/internal/ragexport"
)

type allCitationsFaker struct{}

func (allCitationsFaker) Name() string                    { return "all-citations-test" }
func (allCitationsFaker) Available(context.Context) error { return nil }
func (allCitationsFaker) Complete(_ context.Context, req llm.Request) (llm.Response, error) {
	keys := make([]string, 0, len(req.Citations))
	for _, citation := range req.Citations {
		keys = append(keys, "["+citation.Key+"]")
	}
	answer := fmt.Sprintf("These notes cover the topic %s.", strings.Join(keys, " and "))
	return llm.Response{Answer: answer, Markdown: answer}, nil
}

func TestAskVaultNotesCiteEveryExpectedSource(t *testing.T) {
	root := t.TempDir()
	vaultDir := filepath.Join(root, "vault")
	templateDir := filepath.Join(root, "templates")
	for _, dir := range []string{vaultDir, templateDir, filepath.Join(root, "assets")} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	template := `<!doctype html><html><head><title>{{.Title}}</title></head><body>{{.Content}}</body></html>`
	if err := os.WriteFile(filepath.Join(templateDir, "layout.html"), []byte(template), 0600); err != nil {
		t.Fatal(err)
	}
	notes := map[string]string{
		"birds.md": `---
title: Bird Observation Notes
---
# Bird observations

Meadow bird counts use repeated field observations to estimate migration.
`,
		"maps.md": `---
title: Contour Map Notes
---
# Local contour maps

Local contour maps show elevation lines that guide meadow survey routes.
`,
	}
	for name, body := range notes {
		if err := os.WriteFile(filepath.Join(vaultDir, name), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# Fixture vault\n"), 0600); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	cfg.ProjectRoot = root
	cfg.ContentDir = vaultDir
	cfg.OutputDir = filepath.Join(root, "public")
	cfg.AssetDir = filepath.Join(root, "assets")
	cfg.Template = filepath.Join(templateDir, "layout.html")
	cfg.RagDir = filepath.Join(root, "rag-archive")
	cfg.SiteURL = "https://example.test/vault"

	if _, err := generator.Build(cfg); err != nil {
		t.Fatalf("build fixture vault: %v", err)
	}
	if err := ragexport.RunExport(cfg); err != nil {
		t.Fatalf("export fixture vault: %v", err)
	}

	srv, err := NewServer(Config{
		ProviderName: "fake",
		RagDir:       cfg.RagDir,
		OutputDir:    cfg.OutputDir,
		ContentDir:   "vault",
		LoopbackOnly: true,
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	srv.provider = allCitationsFaker{}
	answer, err := srv.Answer(context.Background(), AnswerRequest{
		Question: "What do the meadow bird counts and local contour maps support?",
	})
	if err != nil {
		t.Fatalf("Answer: %v", err)
	}
	if answer.Status != "answered" {
		t.Fatalf("Answer status = %q, want answered", answer.Status)
	}

	got := make([]string, 0, len(answer.Sources))
	for _, source := range answer.Sources {
		got = append(got, source.Title+"|"+source.URL)
	}
	sort.Strings(got)
	want := []string{
		"Bird Observation Notes|/vault/birds/",
		"Contour Map Notes|/vault/maps/",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("source cards = %v, want %v", got, want)
	}
}
