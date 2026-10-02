package pack

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	sitediff "github.com/tbuddy/la-famille/internal/diff"
	"github.com/tbuddy/la-famille/internal/sitedata"
)

// Comparison reports member-level changes. Ledger is informational and only
// available when both packs carry supported, parseable site manifests.
type Comparison struct {
	BaseSHA256        string           `json:"base_sha256"`
	TargetContentRoot string           `json:"target_content_root"`
	Added             []string         `json:"added"`
	Removed           []string         `json:"removed"`
	Changed           []string         `json:"changed"`
	Ledger            *sitediff.Report `json:"ledger,omitempty"`
	LedgerWarning     string           `json:"ledger_warning,omitempty"`
}

func memberIndex(m Manifest) map[string]Member {
	index := make(map[string]Member, len(m.Members))
	for _, member := range m.Members {
		index[member.Path] = member
	}
	return index
}

func compareMembers(before, after Manifest) Comparison {
	report := Comparison{
		TargetContentRoot: after.ContentRoot,
		Added:             []string{}, Removed: []string{}, Changed: []string{},
	}
	old, next := memberIndex(before), memberIndex(after)
	for _, member := range after.Members {
		base, exists := old[member.Path]
		if !exists {
			report.Added = append(report.Added, member.Path)
		} else if base != member {
			report.Changed = append(report.Changed, member.Path)
		}
	}
	for _, member := range before.Members {
		if _, exists := next[member.Path]; !exists {
			report.Removed = append(report.Removed, member.Path)
		}
	}
	return report
}

func readSiteManifest(s *snapshot) (sitedata.Manifest, error) {
	member, ok := memberIndex(s.manifest)[sitedata.ManifestFileName]
	if !ok {
		return sitedata.Manifest{}, fmt.Errorf("site-manifest.json not present")
	}
	// Semantic reporting is optional. Bound Ledger state independently of the
	// opaque member, which can still be compared and applied without parsing.
	if member.Size > MaxManifestSize {
		return sitedata.Manifest{}, fmt.Errorf("site manifest exceeds Ledger report bound %d bytes", MaxManifestSize)
	}
	data, err := io.ReadAll(s.member(member))
	if err != nil {
		return sitedata.Manifest{}, err
	}
	if int64(len(data)) != member.Size || fmt.Sprintf("%x", sha256.Sum256(data)) != member.SHA256 {
		return sitedata.Manifest{}, fmt.Errorf("site manifest changed while reading")
	}
	var envelope struct {
		Pages []json.RawMessage `json:"pages"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return sitedata.Manifest{}, err
	}
	if len(envelope.Pages) > MaxMembers {
		return sitedata.Manifest{}, fmt.Errorf("site manifest exceeds Ledger report bound %d pages", MaxMembers)
	}
	return sitedata.ParseManifest(data)
}

func addLedger(report *Comparison, before, after *snapshot) {
	a, err := readSiteManifest(before)
	if err != nil {
		report.LedgerWarning = "Before pack: " + err.Error()
		return
	}
	b, err := readSiteManifest(after)
	if err != nil {
		report.LedgerWarning = "After pack: " + err.Error()
		return
	}
	ledger, err := sitediff.Compare(a, b)
	if err != nil {
		report.LedgerWarning = err.Error()
		return
	}
	report.Ledger = &ledger
}

func (r Comparison) Summary() string {
	var out strings.Builder
	fmt.Fprintf(&out, "Pack diff: %d added, %d removed, %d changed members\n",
		len(r.Added), len(r.Removed), len(r.Changed))
	for _, name := range r.Added {
		fmt.Fprintf(&out, "  + %s\n", name)
	}
	for _, name := range r.Removed {
		fmt.Fprintf(&out, "  - %s\n", name)
	}
	for _, name := range r.Changed {
		fmt.Fprintf(&out, "  ~ %s\n", name)
	}
	if len(r.Added)+len(r.Removed)+len(r.Changed) == 0 {
		out.WriteString("No payload member changes.\n")
	}
	fmt.Fprintf(&out, "Exact base SHA256: %s\nTarget content root: %s\n", r.BaseSHA256, r.TargetContentRoot)
	if r.Ledger != nil {
		out.WriteString(r.Ledger.Summary("before pack", "after pack"))
	} else {
		fmt.Fprintf(&out, "Ledger comparison unavailable: %s\n", r.LedgerWarning)
	}
	return out.String()
}
