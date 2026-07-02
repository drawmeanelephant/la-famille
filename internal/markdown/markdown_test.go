package markdown

import (
	"bytes"
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/transform"
)

func TestNewEngine(t *testing.T) {
	// Provide a dummy transformer
	transformer := &transform.LinkTransformer{}
	engine := NewEngine(transformer)

	if engine == nil {
		t.Fatal("expected engine to not be nil")
	}

	// Test a simple conversion to ensure it is configured properly
	source := []byte("# Hello World\n\nThis is a test.")
	var buf bytes.Buffer
	if err := engine.Convert(source, &buf); err != nil {
		t.Fatalf("failed to convert markdown: %v", err)
	}

	result := buf.String()
	if !strings.Contains(result, "<h1>Hello World</h1>") {
		t.Errorf("expected output to contain <h1>Hello World</h1>, got: %s", result)
	}
	if !strings.Contains(result, "<p>This is a test.</p>") {
		t.Errorf("expected output to contain <p>This is a test.</p>, got: %s", result)
	}
}
