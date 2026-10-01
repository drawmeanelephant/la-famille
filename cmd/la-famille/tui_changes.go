package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	sitediff "github.com/tbuddy/la-famille/internal/diff"
	"github.com/tbuddy/la-famille/internal/generator"
)

func (m *model) loadLedger() {
	data, err := os.ReadFile(filepath.Join(m.cfg.OutputDir, sitediff.JSONFileName))
	if os.IsNotExist(err) {
		return
	}
	if err == nil {
		var ledger sitediff.Ledger
		err = json.Unmarshal(data, &ledger)
		if err == nil && ledger.Version != 1 {
			err = fmt.Errorf("unsupported change ledger version %d", ledger.Version)
		}
		if err == nil {
			m.ledger = &ledger
		}
	}
	if err != nil {
		m.addDiagnostic("warning", fmt.Errorf("load change ledger: %w", err))
	}
}

func (m *model) recordLedger(result generator.BuildResult) {
	if result.Ledger != nil {
		m.ledger = result.Ledger
		m.changesCursor = 0
	}
}

type changeRow struct {
	page    string
	details []string
}

func (m model) changeRows() []changeRow {
	if m.ledger == nil {
		return nil
	}
	report := m.ledger.Changes
	details := make(map[string][]string)
	for _, regression := range report.Regressions {
		page := regression.Page
		if page == "" {
			page = "(site)"
		}
		details[page] = append(details[page], "! "+regression.Detail)
	}
	if !m.regressionsOnly {
		for _, page := range report.AddedPages {
			details[page.Identity] = append(details[page.Identity], "Added: "+page.Title)
		}
		for _, page := range report.RemovedPages {
			details[page.Identity] = append(details[page.Identity], "Removed: "+page.Title)
		}
		for _, change := range report.ChangedPages {
			details[change.After.Identity] = append(details[change.After.Identity],
				fmt.Sprintf("%s: %s → %s", change.Kind, change.Before.URL, change.After.URL))
		}
		for _, page := range report.RenderedPages {
			details[page.Identity] = append(details[page.Identity], "Rendered output changed (template or dependencies)")
		}
		for _, field := range report.MetadataChanges {
			details[field.Page] = append(details[field.Page], fmt.Sprintf("%s: %v → %v", field.Field, field.Before, field.After))
		}
		for _, link := range report.AddedLinks {
			details[link.Page] = append(details[link.Page], "Link + "+link.Destination)
		}
		for _, link := range report.RemovedLinks {
			details[link.Page] = append(details[link.Page], "Link - "+link.Destination)
		}
		for _, file := range report.FileChanges {
			key := "file: " + file.Path
			details[key] = append(details[key], file.Action)
		}
	}
	keys := make([]string, 0, len(details))
	for page := range details {
		keys = append(keys, page)
	}
	sort.Strings(keys)
	rows := make([]changeRow, 0, len(keys))
	for _, page := range keys {
		rows = append(rows, changeRow{page: page, details: details[page]})
	}
	return rows
}

func (m model) changesView() string {
	var out strings.Builder
	out.WriteString(titleStyle.Render("Changes") + "\n\n")
	if m.ledger == nil {
		out.WriteString("No build ledger yet. Run Build Site to establish a baseline.\n")
	} else {
		report := m.ledger.Changes
		if m.ledger.Baseline {
			out.WriteString("Baseline established; no previous build to compare.\n")
		} else if len(report.CoverageWarnings) > 0 {
			out.WriteString(warningBadge.Render("Incomplete comparison coverage") + "\n")
		} else if len(report.Regressions) == 0 {
			out.WriteString(successBadge.Render(fmt.Sprintf("no regressions, %d pages changed", len(report.ChangedPages)+len(report.AddedPages)+len(report.RemovedPages))) + "\n")
		} else if len(report.NewBrokenLinks) > 0 {
			out.WriteString(errorBadge.Render(fmt.Sprintf("%d newly-broken %s", len(report.NewBrokenLinks), plural(len(report.NewBrokenLinks), "link", "links"))) + "\n")
		} else {
			out.WriteString(errorBadge.Render(fmt.Sprintf("%d regressions", len(report.Regressions))) + "\n")
		}
		filter := "all changes"
		if m.regressionsOnly {
			filter = "regressions only"
		}
		out.WriteString("Filter: " + filter + "\n\n")
		rows := m.changeRows()
		if len(rows) == 0 {
			out.WriteString("No changes matching this filter.\n")
		} else {
			cursor := min(m.changesCursor, len(rows)-1)
			count := 6
			if m.height > 0 {
				count = max(1, min(6, (m.height-14)/2))
			}
			start := max(0, cursor-min(2, count-1))
			for i := start; i < min(len(rows), start+count); i++ {
				marker := "  "
				if i == cursor {
					marker = "> "
				}
				out.WriteString(marker + rows[i].page + "\n")
			}
			fmt.Fprintf(&out, "\nSelected %d/%d: %s\n", cursor+1, len(rows), rows[cursor].page)
			for i, detail := range rows[cursor].details {
				if i >= 4 {
					out.WriteString("  More details in diff.txt.\n")
					break
				}
				out.WriteString("  " + detail + "\n")
			}
		}
		for _, warning := range report.CoverageWarnings {
			out.WriteString(warning + "\n")
		}
	}
	out.WriteString("\nj/k: Pages • r: Regressions only • d: Diagnostics • ?: Help • Esc/q: Menu")
	if m.width > 0 {
		return boxBorder.Width(max(1, m.width-boxBorder.GetHorizontalFrameSize())).Render(out.String())
	}
	return out.String()
}
