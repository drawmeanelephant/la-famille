package diff

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/tbuddy/la-famille/internal/jsonutil"
)

const (
	JSONFileName = "diff.json"
	TextFileName = "diff.txt"
)

// Ledger is the stable artifact wrapper. A first build establishes a baseline,
// rather than reporting all pre-existing problems as new regressions.
type Ledger struct {
	Version  int    `json:"version"`
	Baseline bool   `json:"baseline"`
	Changes  Report `json:"changes"`
}

// Write writes into a caller-owned output directory (the generator's staging
// tree or an explicit CLI destination).
func Write(dir string, ledger Ledger) error {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	if err := jsonutil.WriteJSON(filepath.Join(dir, JSONFileName), ledger); err != nil {
		return fmt.Errorf("write change ledger: %w", err)
	}
	text := ledger.Changes.Summary("previous build", "current build")
	if ledger.Baseline {
		text = "Baseline established; no previous build to compare.\n" + text
	}
	return os.WriteFile(filepath.Join(dir, TextFileName), []byte(text), 0600)
}
