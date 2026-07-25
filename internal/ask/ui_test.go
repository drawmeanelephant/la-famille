package ask

import (
	"io/fs"
	"strings"
	"testing"
)

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
