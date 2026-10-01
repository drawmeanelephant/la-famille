package diff

import (
	"fmt"
	"sort"

	"github.com/tbuddy/la-famille/internal/sitedata"
)

func addOutputChanges(report *Report, before, after sitedata.Manifest) {
	old, next := make(map[string]string), make(map[string]string)
	paths := make(map[string]bool)
	for _, file := range before.Files {
		old[file.Path], paths[file.Path] = file.Hash, true
	}
	for _, file := range after.Files {
		next[file.Path], paths[file.Path] = file.Hash, true
	}
	for path := range paths {
		if old[path] == next[path] {
			continue
		}
		action := "modified"
		if old[path] == "" {
			action = "added"
		} else if next[path] == "" {
			action = "removed"
		}
		report.FileChanges = append(report.FileChanges, FileChange{
			Path: path, Action: action, Before: old[path], After: next[path],
		})
	}
	sort.Slice(report.FileChanges, func(i, j int) bool { return report.FileChanges[i].Path < report.FileChanges[j].Path })
	oldURLs, newURLs := stringSet(before.Sitemap), stringSet(after.Sitemap)
	for url := range oldURLs {
		if !newURLs[url] {
			report.RemovedSitemap = append(report.RemovedSitemap, url)
		}
	}
	for url := range newURLs {
		if !oldURLs[url] {
			report.AddedSitemap = append(report.AddedSitemap, url)
		}
	}
	sort.Strings(report.AddedSitemap)
	sort.Strings(report.RemovedSitemap)
}

func addRegressions(report *Report, before, after sitedata.Manifest, beforePages, afterPages map[string]sitedata.ManifestPage) {
	for _, link := range report.NewBrokenLinks {
		report.Regressions = append(report.Regressions, Regression{
			Kind: "broken_link", Page: link.Page,
			Detail: fmt.Sprintf("newly-broken link %q (line %d)", link.Destination, link.Line),
		})
	}
	for _, change := range report.OrphanChanges {
		if change.State == "orphaned" {
			report.Regressions = append(report.Regressions, Regression{Kind: "orphan", Page: change.Page, Detail: "new orphan"})
		}
	}
	// Even a matched rename removes the old public page unless its URL remains
	// available. Title-based rename heuristics must not hide inbound-page loss.
	afterURLs := make(map[string]bool, len(after.Pages))
	for _, page := range after.Pages {
		if page.URL != "" {
			afterURLs[page.URL] = true
		}
	}
	for _, page := range before.Pages {
		if page.InboundLinkCount == 0 {
			continue
		}
		preserved := afterURLs[page.URL]
		if page.URL == "" {
			_, preserved = afterPages[page.Identity]
		}
		if !preserved {
			report.Regressions = append(report.Regressions, Regression{
				Kind: "removed_linked_page", Page: page.Identity, Detail: "removed page with inbound links: " + page.URL,
			})
		}
	}
	for _, kind := range []string{"tag", "category"} {
		oldTerms, newTerms := taxonomyTerms(beforePages, kind), taxonomyTerms(afterPages, kind)
		for term := range oldTerms {
			if !newTerms[term] {
				report.Regressions = append(report.Regressions, Regression{Kind: "taxonomy", Detail: kind + " vanished: " + term})
			}
		}
	}
	for _, url := range report.RemovedSitemap {
		report.Regressions = append(report.Regressions, Regression{Kind: "sitemap", Detail: "sitemap entry lost: " + url})
	}
}

func taxonomyTerms(pages map[string]sitedata.ManifestPage, kind string) map[string]bool {
	terms := make(map[string]bool)
	for _, page := range pages {
		if !page.Rendered {
			continue
		}
		values := page.Tags
		if kind == "category" {
			values = page.Categories
		}
		for _, term := range values {
			terms[term] = true
		}
	}
	return terms
}

// GateError is separate from output writing so callers can always save the
// report before returning failure. Legacy snapshots cannot prove gate safety.
func (r Report) GateError() error {
	if len(r.CoverageWarnings) > 0 {
		return fmt.Errorf("regression gate requires complete v2 manifests; rebuild both snapshots")
	}
	if len(r.Regressions) > 0 {
		return fmt.Errorf("regression gate failed: %d regression(s)", len(r.Regressions))
	}
	return nil
}
