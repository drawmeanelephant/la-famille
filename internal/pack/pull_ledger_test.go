package pack

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/sitedata"
)

func TestPullDeltaLedgerUsesTargetMemberOverlay(t *testing.T) {
	for _, name := range []string{"no change", "other member changed", "site manifest changed", "site manifest removed"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			base := filepath.Join(t.TempDir(), "base.tar")
			site := sitedata.Manifest{Version: 2, OutputCaptured: true, Pages: []sitedata.ManifestPage{
				{Identity: "page", Title: "Before"},
			}}
			data, err := json.Marshal(site)
			if err != nil {
				t.Fatal(err)
			}
			entries := []testEntry{{"rag-content.md", []byte("corpus")}, {sitedata.ManifestFileName, data}}
			before := writeTestPack(t, base, entries...)
			switch name {
			case "other member changed":
				entries[0].data = []byte("new corpus")
			case "site manifest changed":
				site.Pages[0].Title = "After"
				entries[1].data, _ = json.Marshal(site)
			case "site manifest removed":
				entries = entries[:1]
			}
			target := writeTestPack(t, filepath.Join(dir, "target.tar"), entries...)
			if _, err := Diff(base, filepath.Join(dir, "target.tar"), filepath.Join(dir, "delta.tar")); err != nil {
				t.Fatal(err)
			}
			writeTestFeed(t, dir, Feed{
				SchemaVersion: FeedVersion,
				Full:          FeedPack{Path: "target.tar", SHA256: archiveHash(target)},
				Deltas:        []FeedDelta{{Path: "delta.tar", BaseSHA256: archiveHash(before)}},
			})
			if err := os.Remove(filepath.Join(dir, "target.tar")); err != nil {
				t.Fatal(err)
			}
			result, err := Pull(dir, base, filepath.Join(t.TempDir(), "result.tar"))
			if err != nil || result.Changes == nil {
				t.Fatalf("pull = %+v, %v", result, err)
			}
			if name == "site manifest removed" {
				if result.Changes.Ledger != nil ||
					!strings.Contains(result.Changes.LedgerWarning, "After pack: site-manifest.json not present") {
					t.Fatalf("removed site manifest = %+v", result.Changes)
				}
			} else if result.Changes.Ledger == nil || result.Changes.LedgerWarning != "" {
				t.Fatalf("available Ledger comparison lost: %+v", result.Changes)
			}
		})
	}
}
