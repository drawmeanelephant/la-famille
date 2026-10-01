package ask

import (
	"io/fs"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestAskGraphUIController(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node is optional; graph UI controller check requires it")
	}
	script, err := filepath.Abs("ui_graph_test.js")
	if err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command(node, script).CombinedOutput(); err != nil {
		t.Fatalf("graph UI controller: %v\n%s", err, output)
	}
}

func TestAskUIAccessibilityMarkup(t *testing.T) {
	uiFS, err := fs.Sub(uiAssets, "ui")
	if err != nil {
		t.Fatalf("failed to sub uiAssets: %v", err)
	}

	htmlBytes, err := fs.ReadFile(uiFS, "index.html")
	if err != nil {
		t.Fatalf("failed to read index.html: %v", err)
	}
	html := string(htmlBytes)

	// Verify <span class="ask-badge"> does not use aria-label on non-interactive element
	if strings.Contains(html, `<span class="ask-badge" aria-label=`) {
		t.Errorf("ask-badge should not use aria-label on non-interactive span")
	}

	// Verify required form control accessibility attributes
	expectedStrings := []string{
		`id="question"`,
		`for="question"`,
		`aria-describedby="question-help status-label"`,
		`id="question-help"`,
		`id="diagnostics-toggle"`,
		`aria-expanded="false"`,
		`aria-controls="diagnostics-drawer"`,
		`id="status-bar"`,
		`aria-live="polite"`,
		`id="answer-region"`,
		`aria-live="polite"`,
		`id="copy-answer"`,
		`aria-label="Copy answer with citations"`,
	}

	for _, str := range expectedStrings {
		if !strings.Contains(html, str) {
			t.Errorf("index.html missing expected accessibility markup: %s", str)
		}
	}
}

func TestAskGraphUIWiring(t *testing.T) {
	html, err := uiAssets.ReadFile("ui/index.html")
	if err != nil {
		t.Fatal(err)
	}
	js, err := uiAssets.ReadFile("ui/app.js")
	if err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{`for="graph-expansion"`, `id="graph-expansion"`,
		`aria-describedby="graph-help"`, `id="diag-retrieval-mode"`} {
		if !strings.Contains(string(html), marker) {
			t.Fatalf("graph UI missing accessible control: %s", marker)
		}
	}
	for _, marker := range []string{"request.graph_expansion", "renderPaths(payload.paths",
		`document.createElement("ol")`, `label.textContent`, `How I got there`, "state.graphExplicit"} {
		if !strings.Contains(string(js), marker) {
			t.Fatalf("graph UI not wired: %s", marker)
		}
	}
}
