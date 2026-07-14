package logger

import (
	"io"
	"log/slog"
	"os"
)

// Setup configures the default slog logger.
// In CLI mode (tuiMode=false), it writes text logs to os.Stderr.
// In TUI mode (tuiMode=true), it attempts to write to logFile.
// If logFile creation fails or is empty, it falls back to io.Discard to prevent TUI corruption.
// It returns the *os.File so the caller can close it if necessary.
func Setup(logFile string, tuiMode bool) (*os.File, error) {
	var w io.Writer
	var f *os.File
	var err error

	if tuiMode {
		if logFile != "" {
			f, err = os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
			if err == nil {
				w = f
			} else {
				w = io.Discard
			}
		} else {
			w = io.Discard
		}
	} else {
		if logFile != "" {
			f, err = os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
			if err == nil {
				w = f
			} else {
				w = os.Stderr // fallback to stderr in CLI mode if file fails
			}
		} else {
			w = os.Stderr
		}
	}

	logger := slog.New(slog.NewTextHandler(w, nil))
	slog.SetDefault(logger)

	return f, err
}
