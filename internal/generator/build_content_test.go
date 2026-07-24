package generator

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
)

// Unrendered pages are copied verbatim, but they are still part of the link
// graph: a page linked only from one of them is not an orphan.
func TestBuild_UnrenderedPagesContributeLinks(t *testing.T) {
	cfg := setupCollisionSite(t, map[string]string{
		"hidden.md": "---\ntitle: Hidden\nrender: false\n---\nSee [target](target.md)\n",
		"target.md": "---\ntitle: Target\n---\nTARGET_BODY\n",
		"index.md":  "---\ntitle: Index\n---\nINDEX_BODY\n",
	})

	res, err := Build(cfg)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	var backlinks map[string][]string
	if err := json.Unmarshal([]byte(readOutput(t, cfg, "backlinks.json")), &backlinks); err != nil {
		t.Fatalf("invalid backlinks.json: %v", err)
	}
	if got := backlinks["target"]; !slices.Contains(got, "hidden.md") {
		t.Errorf("backlinks[\"target\"] = %v, want it to contain \"hidden.md\"", got)
	}

	var parsedGraph struct {
		Edges [][2]string `json:"edges"`
	}
	if err := json.Unmarshal([]byte(readOutput(t, cfg, "graph.json")), &parsedGraph); err != nil {
		t.Fatalf("invalid graph.json: %v", err)
	}
	if !slices.Contains(parsedGraph.Edges, [2]string{"hidden.md", "target"}) {
		t.Errorf("graph edges = %v, want an edge from hidden.md to target", parsedGraph.Edges)
	}

	if slices.Contains(res.Health.OrphanedPages, "target") {
		t.Errorf("OrphanedPages = %v, want target excluded: it is linked from hidden.md", res.Health.OrphanedPages)
	}

	// The unrendered source itself is still copied verbatim, links and all.
	if raw := readOutput(t, cfg, "hidden.md"); !strings.Contains(raw, "[target](target.md)") {
		t.Errorf("hidden.md = %q, want the original markdown copied verbatim", raw)
	}
}

// Scanning an unrendered page's links is an enrichment step. Its output is a
// byte copy that does not depend on the conversion, so a conversion failure
// must not turn a previously successful build into a failure or drop the file.
func TestBuild_UnrenderedPageSurvivesConversionFailure(t *testing.T) {
	cfg := setupCollisionSite(t, map[string]string{
		"hidden.md": "---\ntitle: Hidden\nrender: false\n---\nHIDDEN_RAW_BODY\n",
		"index.md":  "---\ntitle: Index\n---\nINDEX_BODY\n",
	})

	original := getConvertMarkdown()
	t.Cleanup(func() { setConvertMarkdown(original) })
	setConvertMarkdown(func(md goldmark.Markdown, source []byte, w *bytes.Buffer) error {
		if strings.Contains(string(source), "HIDDEN_RAW_BODY") {
			return errors.New("synthetic conversion failure")
		}
		return md.Convert(source, w)
	})

	if _, err := Build(cfg); err != nil {
		t.Fatalf("Build() error = %v, want success: the unrendered page is copied, not converted", err)
	}

	if raw := readOutput(t, cfg, "hidden.md"); !strings.Contains(raw, "HIDDEN_RAW_BODY") {
		t.Errorf("hidden.md = %q, want it still copied verbatim", raw)
	}
}

// GFM task lists carry their state in a disabled checkbox input; stripping it
// makes done and pending items render identically.
func TestBuild_TaskListStateSurvivesSanitizer(t *testing.T) {
	cfg := setupCollisionSite(t, map[string]string{
		"index.md": "---\ntitle: Tasks\n---\n- [x] finished item\n- [ ] pending item\n",
	})

	if _, err := Build(cfg); err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	html := readOutput(t, cfg, "index.html")
	if strings.Count(html, "type=\"checkbox\"") != 2 {
		t.Fatalf("index.html = %q, want two checkbox inputs", html)
	}

	var finished, pending string
	for _, item := range strings.Split(html, "<li>") {
		switch {
		case strings.Contains(item, "finished item"):
			finished = item
		case strings.Contains(item, "pending item"):
			pending = item
		}
	}
	if finished == "" || pending == "" {
		t.Fatalf("index.html = %q, want both task items rendered", html)
	}
	if !strings.Contains(finished, "checked") {
		t.Errorf("completed item = %q, want it checked", finished)
	}
	if strings.Contains(pending, "checked") {
		t.Errorf("pending item = %q, want it unchecked", pending)
	}
}

// The task-list allowance must not become a general <input> allowance: raw HTML
// in markdown still may not introduce scriptable or data-collecting fields.
func TestBuild_SanitizerRejectsScriptableInputs(t *testing.T) {
	cfg := setupCollisionSite(t, map[string]string{
		"index.md": "---\ntitle: Raw\n---\n<input type=\"text\" name=\"user\" onfocus=\"alert(1)\">\n<input type=\"password\">\n",
	})

	if _, err := Build(cfg); err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	html := readOutput(t, cfg, "index.html")
	for _, unwanted := range []string{"onfocus", "type=\"text\"", "type=\"password\"", "name=\"user\""} {
		if strings.Contains(html, unwanted) {
			t.Errorf("index.html = %q, want %q stripped", html, unwanted)
		}
	}
}

// A build that fails validation must not leave a half-published site or any
// intermediate directory behind.
func TestBuild_FailedValidationLeavesOutputUntouched(t *testing.T) {
	cfg := setupCollisionSite(t, map[string]string{
		"index.md": "---\ntitle: Index\n---\nFIRST_BUILD_BODY\n",
	})

	if _, err := Build(cfg); err != nil {
		t.Fatalf("first Build() error = %v", err)
	}
	before := readOutput(t, cfg, "index.html")

	collide := "---\ntitle: Clash\nslug: shared\n---\nBODY\n"
	for _, name := range []string{"a.md", "b.md"} {
		if err := os.WriteFile(filepath.Join(cfg.ContentDir, name), []byte(collide), 0600); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := Build(cfg); err == nil {
		t.Fatal("Build() error = nil, want a collision for the shared slug")
	}

	if after := readOutput(t, cfg, "index.html"); after != before {
		t.Errorf("index.html changed after a failed build: got %q, want %q", after, before)
	}
	entries, err := os.ReadDir(filepath.Dir(cfg.OutputDir))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".staging-") || strings.Contains(entry.Name(), ".previous-") {
			t.Errorf("failed build left an intermediate directory behind: %s", entry.Name())
		}
	}
}
