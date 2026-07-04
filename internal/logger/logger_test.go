package logger

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetupCLI(t *testing.T) {
	// Verify standard CLI behavior (writing to Stderr is harder to test directly without hijacking os.Stderr)
	// But we can verify it doesn't return a file.
	f, err := Setup("", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f != nil {
		t.Fatalf("expected nil file in CLI mode")
	}
}

func TestSetupTUI_Success(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	f, err := Setup(logFile, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f == nil {
		t.Fatalf("expected file to be returned")
	}
	defer f.Close()

	slog.Info("test message", "key", "value")

	// Note: We need to flush or just read because file is opened in append mode and writing directly via slog
	// Actually, slog writes synchronously, so we should be fine.

	// We should wait a tiny bit or just read it directly since it's local disk
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}
	if !strings.Contains(string(content), "test message") || !strings.Contains(string(content), "key=value") {
		t.Errorf("expected log message not found in file: %s", string(content))
	}
}

func TestSetupTUI_FallbackToDiscard(t *testing.T) {
	// Try to open a file in a non-existent directory
	f, err := Setup("/nonexistent/dir/test.log", true)
	if err == nil {
		t.Fatalf("expected error when opening invalid path")
	}
	if f != nil {
		t.Fatalf("expected nil file on error")
	}

	// Should not panic, should go to discard
	slog.Info("test message fallback")
}

func TestSetupTUI_EmptyFileFallback(t *testing.T) {
	f, err := Setup("", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f != nil {
		t.Fatalf("expected nil file when empty filename provided")
	}

	// Should not panic, should go to discard
	slog.Info("test message empty")
}

func TestSetupTUI_LogLevels(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "levels.log")

	f, err := Setup(logFile, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer f.Close()

	slog.Info("info msg")
	slog.Warn("warn msg")
	slog.Error("error msg")

	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	strContent := string(content)
	if !strings.Contains(strContent, "level=INFO") || !strings.Contains(strContent, "info msg") {
		t.Errorf("expected INFO log missing")
	}
	if !strings.Contains(strContent, "level=WARN") || !strings.Contains(strContent, "warn msg") {
		t.Errorf("expected WARN log missing")
	}
	if !strings.Contains(strContent, "level=ERROR") || !strings.Contains(strContent, "error msg") {
		t.Errorf("expected ERROR log missing")
	}
}
