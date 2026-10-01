// Package diff compares deterministic site manifests.
package diff

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/tbuddy/la-famille/internal/sitedata"
)

// Report contains semantic changes between two site manifests. All slices are
// initialized and sorted so the JSON representation is stable.
type Report struct {
	AddedPages          []PageRef        `json:"added_pages"`
	RemovedPages        []PageRef        `json:"removed_pages"`
	ChangedPages        []PageChange     `json:"changed_pages"`
	MetadataChanges     []MetadataChange `json:"metadata_changes"`
	TaxonomyChanges     []TaxonomyChange `json:"taxonomy_changes"`
	AddedLinks          []LinkChange     `json:"added_links"`
	RemovedLinks        []LinkChange     `json:"removed_links"`
	AddedEdges          []GraphEdge      `json:"added_edges"`
	RemovedEdges        []GraphEdge      `json:"removed_edges"`
	OrphanChanges       []OrphanChange   `json:"orphan_changes"`
	NewBrokenLinks      []BrokenLink     `json:"new_broken_links"`
	ResolvedBrokenLinks []BrokenLink     `json:"resolved_broken_links"`
	FileChanges         []FileChange     `json:"file_changes,omitempty"`
	AddedSitemap        []string         `json:"added_sitemap,omitempty"`
	RemovedSitemap      []string         `json:"removed_sitemap,omitempty"`
	Regressions         []Regression     `json:"regressions,omitempty"`
	CoverageWarnings    []string         `json:"coverage_warnings,omitempty"`
	RenderedPages       []PageRef        `json:"rendered_pages,omitempty"`
}

type FileChange struct {
	Path   string `json:"path"`
	Action string `json:"action"`
	Before string `json:"before"`
	After  string `json:"after"`
}

// Regression is one actionable gate finding. Kinds are stable machine keys.
type Regression struct {
	Kind   string `json:"kind"`
	Page   string `json:"page,omitempty"`
	Detail string `json:"detail"`
}

// PageRef is the stable, human-readable portion of a manifest page.
type PageRef struct {
	Identity   string `json:"identity"`
	SourcePath string `json:"source_path"`
	URL        string `json:"url"`
	Title      string `json:"title"`
}

// PageChange records a modification or rename of a page present in both
// manifests.
type PageChange struct {
	Kind   string  `json:"kind"`
	Before PageRef `json:"before"`
	After  PageRef `json:"after"`
}

// MetadataChange describes one changed page metadata field.
type MetadataChange struct {
	Page   string `json:"page"`
	Field  string `json:"field"`
	Before any    `json:"before"`
	After  any    `json:"after"`
}

// TaxonomyChange describes a tag or category membership change.
type TaxonomyChange struct {
	Page   string `json:"page"`
	Kind   string `json:"kind"`
	Term   string `json:"term"`
	Action string `json:"action"`
}

// LinkChange describes a Markdown link-set addition or removal.
type LinkChange struct {
	Page        string `json:"page"`
	Destination string `json:"destination"`
	Target      string `json:"target"`
}

// GraphEdge describes a directed site-graph edge.
type GraphEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// OrphanChange reports a transition into or out of the orphan state.
type OrphanChange struct {
	Page  string `json:"page"`
	State string `json:"state"`
}

// BrokenLink identifies an unresolved internal link.
type BrokenLink struct {
	Page        string `json:"page"`
	Destination string `json:"destination"`
	Target      string `json:"target"`
	Line        int    `json:"line"`
}

type pagePair struct {
	before sitedata.ManifestPage
	after  sitedata.ManifestPage
}

// Compare returns the semantic difference from before to after.
func Compare(before, after sitedata.Manifest) (Report, error) {
	report := Report{
		AddedPages:          []PageRef{},
		RemovedPages:        []PageRef{},
		ChangedPages:        []PageChange{},
		MetadataChanges:     []MetadataChange{},
		TaxonomyChanges:     []TaxonomyChange{},
		AddedLinks:          []LinkChange{},
		RemovedLinks:        []LinkChange{},
		AddedEdges:          []GraphEdge{},
		RemovedEdges:        []GraphEdge{},
		OrphanChanges:       []OrphanChange{},
		NewBrokenLinks:      []BrokenLink{},
		ResolvedBrokenLinks: []BrokenLink{},
	}

	beforePages, err := indexPages(before)
	if err != nil {
		return Report{}, fmt.Errorf("before manifest: %w", err)
	}
	afterPages, err := indexPages(after)
	if err != nil {
		return Report{}, fmt.Errorf("after manifest: %w", err)
	}

	pairs, beforeToAfter := matchPages(beforePages, afterPages)
	for _, pair := range pairs {
		addPageChanges(&report, pair)
		if !pageChanged(pair.before, pair.after) && pair.before.OutputHash != "" &&
			pair.after.OutputHash != "" && pair.before.OutputHash != pair.after.OutputHash {
			report.RenderedPages = append(report.RenderedPages, pageRef(pair.after))
		}
		addTaxonomyChanges(&report, pair)
		addMetadataChanges(&report, pair)
	}

	for id, page := range beforePages {
		if _, matched := beforeToAfter[id]; matched {
			continue
		}
		report.RemovedPages = append(report.RemovedPages, pageRef(page))
		appendPageTaxonomy(&report, id, page, "removed")
		for _, link := range page.Links {
			report.RemovedLinks = append(report.RemovedLinks, LinkChange{
				Page: id, Destination: link.Destination, Target: link.Target,
			})
			if !link.Resolved {
				report.ResolvedBrokenLinks = append(report.ResolvedBrokenLinks, brokenLink(id, link))
			}
		}
	}
	matchedAfter := make(map[string]bool, len(pairs))
	for _, pair := range pairs {
		matchedAfter[pair.after.Identity] = true
	}
	for id, page := range afterPages {
		if matchedAfter[id] {
			continue
		}
		report.AddedPages = append(report.AddedPages, pageRef(page))
		appendPageTaxonomy(&report, id, page, "added")
	}

	addLinkChanges(&report, beforePages, afterPages, beforeToAfter)
	addGraphChanges(&report, before, after)
	addOrphanChanges(&report, beforePages, afterPages, beforeToAfter)
	if before.Version == 1 || after.Version == 1 {
		report.CoverageWarnings = append(report.CoverageWarnings,
			"Legacy v1 manifest: prose, extra frontmatter, published bytes, and sitemap coverage is incomplete. Rebuild both snapshots.")
	} else if !before.OutputCaptured || !after.OutputCaptured {
		report.CoverageWarnings = append(report.CoverageWarnings,
			"Published output was not captured in both snapshots. Rebuild both snapshots for complete comparison coverage.")
	} else {
		addOutputChanges(&report, before, after)
	}
	addRegressions(&report, before, after, beforePages, afterPages)

	sortReport(&report)
	return report, nil
}

// Empty reports whether the comparison found any changes.
func (r Report) Empty() bool {
	return len(r.AddedPages) == 0 &&
		len(r.RemovedPages) == 0 &&
		len(r.ChangedPages) == 0 &&
		len(r.MetadataChanges) == 0 &&
		len(r.TaxonomyChanges) == 0 &&
		len(r.AddedLinks) == 0 &&
		len(r.RemovedLinks) == 0 &&
		len(r.AddedEdges) == 0 &&
		len(r.RemovedEdges) == 0 &&
		len(r.OrphanChanges) == 0 &&
		len(r.NewBrokenLinks) == 0 &&
		len(r.ResolvedBrokenLinks) == 0 &&
		len(r.FileChanges) == 0 &&
		len(r.RenderedPages) == 0 &&
		len(r.AddedSitemap) == 0 &&
		len(r.RemovedSitemap) == 0
}

// Summary formats a readable report for terminal output.
func (r Report) Summary(beforeLabel, afterLabel string) string {
	var out strings.Builder
	fmt.Fprintf(&out, "Site diff: %s → %s\n", beforeLabel, afterLabel)
	fmt.Fprintf(&out, "Pages: %d added, %d removed, %d changed\n",
		len(r.AddedPages), len(r.RemovedPages), len(r.ChangedPages))
	for _, page := range r.AddedPages {
		fmt.Fprintf(&out, "  + %s (%s)\n", page.Identity, page.Title)
	}
	for _, page := range r.RemovedPages {
		fmt.Fprintf(&out, "  - %s (%s)\n", page.Identity, page.Title)
	}
	for _, page := range r.ChangedPages {
		if page.Kind == "renamed" {
			fmt.Fprintf(&out, "  ~ %s → %s (%s)\n", page.Before.Identity, page.After.Identity, page.After.Title)
		} else {
			fmt.Fprintf(&out, "  ~ %s (%s)\n", page.After.Identity, page.After.Title)
		}
	}

	fmt.Fprintf(&out, "Metadata: %d field changes\n", len(r.MetadataChanges))
	for _, change := range r.MetadataChanges {
		fmt.Fprintf(&out, "  %s.%s: %v → %v\n", change.Page, change.Field, change.Before, change.After)
	}
	fmt.Fprintf(&out, "Taxonomy: %d membership changes\n", len(r.TaxonomyChanges))
	for _, change := range r.TaxonomyChanges {
		sign := "+"
		if change.Action == "removed" {
			sign = "-"
		}
		fmt.Fprintf(&out, "  %s %s %q on %s\n", sign, change.Kind, change.Term, change.Page)
	}
	fmt.Fprintf(&out, "Links: %d added, %d removed\n", len(r.AddedLinks), len(r.RemovedLinks))
	for _, link := range r.AddedLinks {
		fmt.Fprintf(&out, "  + %s → %s\n", link.Page, link.Destination)
	}
	for _, link := range r.RemovedLinks {
		fmt.Fprintf(&out, "  - %s → %s\n", link.Page, link.Destination)
	}
	fmt.Fprintf(&out, "Graph edges: %d added, %d removed\n", len(r.AddedEdges), len(r.RemovedEdges))
	for _, edge := range r.AddedEdges {
		fmt.Fprintf(&out, "  + %s → %s\n", edge.From, edge.To)
	}
	for _, edge := range r.RemovedEdges {
		fmt.Fprintf(&out, "  - %s → %s\n", edge.From, edge.To)
	}
	fmt.Fprintf(&out, "Orphans: %d newly orphaned, %d no longer orphaned\n",
		countOrphanState(r.OrphanChanges, "orphaned"), countOrphanState(r.OrphanChanges, "resolved"))
	for _, change := range r.OrphanChanges {
		if change.State == "orphaned" {
			fmt.Fprintf(&out, "  ! %s became an orphan\n", change.Page)
		} else {
			fmt.Fprintf(&out, "  ✓ %s is no longer an orphan\n", change.Page)
		}
	}
	fmt.Fprintf(&out, "Broken links: %d newly broken, %d resolved\n",
		len(r.NewBrokenLinks), len(r.ResolvedBrokenLinks))
	for _, link := range r.NewBrokenLinks {
		fmt.Fprintf(&out, "  ! %s:%d %q → %s\n", link.Page, link.Line, link.Destination, link.Target)
	}
	for _, link := range r.ResolvedBrokenLinks {
		fmt.Fprintf(&out, "  ✓ %s:%d %q → %s\n", link.Page, link.Line, link.Destination, link.Target)
	}
	for _, file := range r.FileChanges {
		fmt.Fprintf(&out, "Published file %s: %s\n", file.Action, file.Path)
	}
	for _, page := range r.RenderedPages {
		fmt.Fprintf(&out, "Rendered output changed: %s (%s)\n", page.Identity, page.Title)
	}
	for _, url := range r.AddedSitemap {
		fmt.Fprintf(&out, "Sitemap + %s\n", url)
	}
	for _, url := range r.RemovedSitemap {
		fmt.Fprintf(&out, "Sitemap - %s\n", url)
	}
	fmt.Fprintf(&out, "Regressions: %d\n", len(r.Regressions))
	for _, regression := range r.Regressions {
		fmt.Fprintf(&out, "  ! %s: %s %s\n", regression.Kind, regression.Page, regression.Detail)
	}
	for _, warning := range r.CoverageWarnings {
		fmt.Fprintf(&out, "Coverage warning: %s\n", warning)
	}
	if r.Empty() {
		if len(r.CoverageWarnings) == 0 {
			out.WriteString("No changes.\n")
		} else {
			out.WriteString("No changes in the covered fields.\n")
		}
	}
	return out.String()
}

func countOrphanState(changes []OrphanChange, state string) int {
	count := 0
	for _, change := range changes {
		if change.State == state {
			count++
		}
	}
	return count
}

func indexPages(manifest sitedata.Manifest) (map[string]sitedata.ManifestPage, error) {
	if manifest.Version != 1 && manifest.Version != sitedata.ManifestVersion {
		return nil, fmt.Errorf("unsupported site manifest version %d", manifest.Version)
	}
	pages := make(map[string]sitedata.ManifestPage, len(manifest.Pages))
	for _, page := range manifest.Pages {
		if page.Identity == "" {
			return nil, fmt.Errorf("page has an empty identity")
		}
		if _, exists := pages[page.Identity]; exists {
			return nil, fmt.Errorf("duplicate page identity %q", page.Identity)
		}
		pages[page.Identity] = page
	}
	return pages, nil
}

func matchPages(
	before, after map[string]sitedata.ManifestPage,
) ([]pagePair, map[string]string) {
	beforeIDs := sortedPageIDs(before)
	afterIDs := sortedPageIDs(after)
	pairs := make([]pagePair, 0)
	beforeToAfter := make(map[string]string, len(before))
	matchedAfter := make(map[string]bool, len(after))

	for _, id := range beforeIDs {
		if page, ok := after[id]; ok {
			pairs = append(pairs, pagePair{before: before[id], after: page})
			beforeToAfter[id] = id
			matchedAfter[id] = true
		}
	}

	// A unique unchanged title is the only rename signal carried by the v1
	// manifest. Requiring uniqueness avoids guessing between duplicate titles.
	beforeTitles := make(map[string][]string)
	afterTitles := make(map[string][]string)
	for _, id := range beforeIDs {
		if _, matched := beforeToAfter[id]; matched {
			continue
		}
		title := strings.TrimSpace(before[id].Title)
		if title != "" {
			beforeTitles[title] = append(beforeTitles[title], id)
		}
	}
	for _, id := range afterIDs {
		if matchedAfter[id] {
			continue
		}
		title := strings.TrimSpace(after[id].Title)
		if title != "" {
			afterTitles[title] = append(afterTitles[title], id)
		}
	}
	titles := make([]string, 0, len(beforeTitles))
	for title := range beforeTitles {
		if len(beforeTitles[title]) == 1 && len(afterTitles[title]) == 1 {
			titles = append(titles, title)
		}
	}
	sort.Strings(titles)
	for _, title := range titles {
		oldID := beforeTitles[title][0]
		newID := afterTitles[title][0]
		pairs = append(pairs, pagePair{before: before[oldID], after: after[newID]})
		beforeToAfter[oldID] = newID
		matchedAfter[newID] = true
	}

	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].after.Identity != pairs[j].after.Identity {
			return pairs[i].after.Identity < pairs[j].after.Identity
		}
		return pairs[i].before.Identity < pairs[j].before.Identity
	})
	return pairs, beforeToAfter
}

func sortedPageIDs(pages map[string]sitedata.ManifestPage) []string {
	ids := make([]string, 0, len(pages))
	for id := range pages {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func pageRef(page sitedata.ManifestPage) PageRef {
	return PageRef{
		Identity:   page.Identity,
		SourcePath: page.SourcePath,
		URL:        page.URL,
		Title:      page.Title,
	}
}

func addPageChanges(report *Report, pair pagePair) {
	kind := "modified"
	if pair.before.Identity != pair.after.Identity {
		kind = "renamed"
	}
	if !pageChanged(pair.before, pair.after) {
		return
	}
	report.ChangedPages = append(report.ChangedPages, PageChange{
		Kind:   kind,
		Before: pageRef(pair.before),
		After:  pageRef(pair.after),
	})
}

func pageChanged(before, after sitedata.ManifestPage) bool {
	if before.SourcePath != after.SourcePath ||
		before.URL != after.URL ||
		before.Title != after.Title ||
		before.Date != after.Date ||
		before.Rendered != after.Rendered ||
		(before.ContentHash != "" && after.ContentHash != "" && before.ContentHash != after.ContentHash) ||
		(before.Frontmatter != nil && after.Frontmatter != nil && !reflect.DeepEqual(before.Frontmatter, after.Frontmatter)) ||
		!sameStrings(before.Tags, after.Tags) ||
		!sameStrings(before.Categories, after.Categories) ||
		!sameStrings(before.OutboundLinks, after.OutboundLinks) ||
		!sameStrings(before.AssetReferences, after.AssetReferences) {
		return true
	}
	return !sameLinks(before.Links, after.Links)
}

func addMetadataChanges(report *Report, pair pagePair) {
	page := pair.after.Identity
	appendMetadataChange(report, page, "url", pair.before.URL, pair.after.URL)
	appendMetadataChange(report, page, "title", pair.before.Title, pair.after.Title)
	appendMetadataChange(report, page, "date", pair.before.Date, pair.after.Date)
	appendMetadataChange(report, page, "rendered", pair.before.Rendered, pair.after.Rendered)
	if pair.before.Frontmatter != nil && pair.after.Frontmatter != nil {
		fields := make(map[string]bool)
		for field := range pair.before.Frontmatter {
			fields[field] = true
		}
		for field := range pair.after.Frontmatter {
			fields[field] = true
		}
		for field := range fields {
			appendMetadataChange(report, page, field, pair.before.Frontmatter[field], pair.after.Frontmatter[field])
		}
	}
}

func appendMetadataChange(report *Report, page, field string, before, after any) {
	if before == after {
		return
	}
	report.MetadataChanges = append(report.MetadataChanges, MetadataChange{
		Page:   page,
		Field:  field,
		Before: before,
		After:  after,
	})
}

func addTaxonomyChanges(report *Report, pair pagePair) {
	page := pair.after.Identity
	appendTaxonomyChanges(report, page, "tag", pair.before.Tags, pair.after.Tags)
	appendTaxonomyChanges(report, page, "category", pair.before.Categories, pair.after.Categories)
}

func appendPageTaxonomy(report *Report, page string, metadata sitedata.ManifestPage, action string) {
	for _, term := range metadata.Tags {
		report.TaxonomyChanges = append(report.TaxonomyChanges, TaxonomyChange{
			Page: page, Kind: "tag", Term: term, Action: action,
		})
	}
	for _, term := range metadata.Categories {
		report.TaxonomyChanges = append(report.TaxonomyChanges, TaxonomyChange{
			Page: page, Kind: "category", Term: term, Action: action,
		})
	}
}

func appendTaxonomyChanges(report *Report, page, kind string, before, after []string) {
	beforeSet, afterSet := stringSet(before), stringSet(after)
	for term := range beforeSet {
		if !afterSet[term] {
			report.TaxonomyChanges = append(report.TaxonomyChanges, TaxonomyChange{
				Page: page, Kind: kind, Term: term, Action: "removed",
			})
		}
	}
	for term := range afterSet {
		if !beforeSet[term] {
			report.TaxonomyChanges = append(report.TaxonomyChanges, TaxonomyChange{
				Page: page, Kind: kind, Term: term, Action: "added",
			})
		}
	}
}

type linkKey struct {
	destination string
	target      string
}

func addLinkChanges(
	report *Report,
	beforePages, afterPages map[string]sitedata.ManifestPage,
	beforeToAfter map[string]string,
) {
	beforeIDs := sortedPageIDs(beforePages)
	handledAfter := make(map[string]bool, len(afterPages))
	for _, beforeID := range beforeIDs {
		beforePage := beforePages[beforeID]
		afterID, matched := beforeToAfter[beforeID]
		if !matched {
			continue
		}
		handledAfter[afterID] = true
		comparePageLinks(report, afterID, beforePage.Links, afterPages[afterID].Links)
	}

	for _, afterID := range sortedPageIDs(afterPages) {
		if handledAfter[afterID] {
			continue
		}
		comparePageLinks(report, afterID, nil, afterPages[afterID].Links)
	}
}

func comparePageLinks(
	report *Report,
	page string,
	before, after []sitedata.ManifestLink,
) {
	beforeLinks := indexLinks(before)
	afterLinks := indexLinks(after)
	for key, oldLink := range beforeLinks {
		newLink, exists := afterLinks[key]
		if !exists {
			report.RemovedLinks = append(report.RemovedLinks, LinkChange{
				Page: page, Destination: key.destination, Target: key.target,
			})
			if !oldLink.Resolved {
				report.ResolvedBrokenLinks = append(report.ResolvedBrokenLinks, brokenLink(page, oldLink))
			}
			continue
		}
		if !oldLink.Resolved && newLink.Resolved {
			report.ResolvedBrokenLinks = append(report.ResolvedBrokenLinks, brokenLink(page, oldLink))
		} else if oldLink.Resolved && !newLink.Resolved {
			report.NewBrokenLinks = append(report.NewBrokenLinks, brokenLink(page, newLink))
		}
	}
	for key, newLink := range afterLinks {
		if _, exists := beforeLinks[key]; exists {
			continue
		}
		report.AddedLinks = append(report.AddedLinks, LinkChange{
			Page: page, Destination: key.destination, Target: key.target,
		})
		if !newLink.Resolved {
			report.NewBrokenLinks = append(report.NewBrokenLinks, brokenLink(page, newLink))
		}
	}
}

func indexLinks(links []sitedata.ManifestLink) map[linkKey]sitedata.ManifestLink {
	indexed := make(map[linkKey]sitedata.ManifestLink, len(links))
	for _, link := range links {
		key := linkKey{destination: link.Destination, target: link.Target}
		if previous, exists := indexed[key]; !exists || (!previous.Resolved && link.Resolved) {
			indexed[key] = link
		}
	}
	return indexed
}

func brokenLink(page string, link sitedata.ManifestLink) BrokenLink {
	return BrokenLink{
		Page:        page,
		Destination: link.Destination,
		Target:      link.Target,
		Line:        link.Line,
	}
}

func addGraphChanges(report *Report, before, after sitedata.Manifest) {
	beforeEdges := manifestEdges(before)
	afterEdges := manifestEdges(after)
	for edge := range beforeEdges {
		if !afterEdges[edge] {
			report.RemovedEdges = append(report.RemovedEdges, GraphEdge{From: edge.from, To: edge.to})
		}
	}
	for edge := range afterEdges {
		if !beforeEdges[edge] {
			report.AddedEdges = append(report.AddedEdges, GraphEdge{From: edge.from, To: edge.to})
		}
	}
}

type edgeKey struct {
	from string
	to   string
}

func manifestEdges(manifest sitedata.Manifest) map[edgeKey]bool {
	edges := make(map[edgeKey]bool)
	for _, page := range manifest.Pages {
		for _, target := range page.OutboundLinks {
			edges[edgeKey{from: page.Identity, to: target}] = true
		}
	}
	return edges
}

func addOrphanChanges(
	report *Report,
	beforePages, afterPages map[string]sitedata.ManifestPage,
	beforeToAfter map[string]string,
) {
	beforeOrphans := make(map[string]bool)
	for id, page := range beforePages {
		if afterID, matched := beforeToAfter[id]; matched && isOrphan(page) {
			beforeOrphans[afterID] = true
		}
	}
	afterOrphans := make(map[string]bool)
	for id, page := range afterPages {
		if isOrphan(page) {
			afterOrphans[id] = true
		}
	}
	for id := range beforeOrphans {
		if !afterOrphans[id] {
			report.OrphanChanges = append(report.OrphanChanges, OrphanChange{Page: id, State: "resolved"})
		}
	}
	for id := range afterOrphans {
		if !beforeOrphans[id] {
			report.OrphanChanges = append(report.OrphanChanges, OrphanChange{Page: id, State: "orphaned"})
		}
	}
}

func isOrphan(page sitedata.ManifestPage) bool {
	return page.Rendered && page.InboundLinkCount == 0 && page.Identity != "index"
}

func sameStrings(left, right []string) bool {
	leftSet, rightSet := stringSet(left), stringSet(right)
	if len(leftSet) != len(rightSet) {
		return false
	}
	for value := range leftSet {
		if !rightSet[value] {
			return false
		}
	}
	return true
}

func stringSet(values []string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, value := range values {
		set[value] = true
	}
	return set
}

func sameLinks(left, right []sitedata.ManifestLink) bool {
	leftIndex, rightIndex := indexLinks(left), indexLinks(right)
	if len(leftIndex) != len(rightIndex) {
		return false
	}
	for key, leftLink := range leftIndex {
		rightLink, ok := rightIndex[key]
		if !ok || leftLink.Resolved != rightLink.Resolved {
			return false
		}
	}
	return true
}

func sortReport(report *Report) {
	sort.Slice(report.Regressions, func(i, j int) bool {
		a, b := report.Regressions[i], report.Regressions[j]
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		if a.Page != b.Page {
			return a.Page < b.Page
		}
		return a.Detail < b.Detail
	})
	sort.Slice(report.AddedPages, func(i, j int) bool {
		return report.AddedPages[i].Identity < report.AddedPages[j].Identity
	})
	sort.Slice(report.RemovedPages, func(i, j int) bool {
		return report.RemovedPages[i].Identity < report.RemovedPages[j].Identity
	})
	sort.Slice(report.ChangedPages, func(i, j int) bool {
		if report.ChangedPages[i].After.Identity != report.ChangedPages[j].After.Identity {
			return report.ChangedPages[i].After.Identity < report.ChangedPages[j].After.Identity
		}
		return report.ChangedPages[i].Before.Identity < report.ChangedPages[j].Before.Identity
	})
	sort.Slice(report.MetadataChanges, func(i, j int) bool {
		left, right := report.MetadataChanges[i], report.MetadataChanges[j]
		if left.Page != right.Page {
			return left.Page < right.Page
		}
		return left.Field < right.Field
	})
	sort.Slice(report.TaxonomyChanges, func(i, j int) bool {
		left, right := report.TaxonomyChanges[i], report.TaxonomyChanges[j]
		if left.Page != right.Page {
			return left.Page < right.Page
		}
		if left.Kind != right.Kind {
			return left.Kind < right.Kind
		}
		if left.Term != right.Term {
			return left.Term < right.Term
		}
		return left.Action < right.Action
	})
	sort.Slice(report.AddedLinks, func(i, j int) bool { return lessLink(report.AddedLinks[i], report.AddedLinks[j]) })
	sort.Slice(report.RemovedLinks, func(i, j int) bool { return lessLink(report.RemovedLinks[i], report.RemovedLinks[j]) })
	sort.Slice(report.AddedEdges, func(i, j int) bool { return lessEdge(report.AddedEdges[i], report.AddedEdges[j]) })
	sort.Slice(report.RemovedEdges, func(i, j int) bool { return lessEdge(report.RemovedEdges[i], report.RemovedEdges[j]) })
	sort.Slice(report.OrphanChanges, func(i, j int) bool {
		if report.OrphanChanges[i].Page != report.OrphanChanges[j].Page {
			return report.OrphanChanges[i].Page < report.OrphanChanges[j].Page
		}
		return report.OrphanChanges[i].State < report.OrphanChanges[j].State
	})
	sort.Slice(report.NewBrokenLinks, func(i, j int) bool { return lessBroken(report.NewBrokenLinks[i], report.NewBrokenLinks[j]) })
	sort.Slice(report.ResolvedBrokenLinks, func(i, j int) bool {
		return lessBroken(report.ResolvedBrokenLinks[i], report.ResolvedBrokenLinks[j])
	})
}

func lessLink(left, right LinkChange) bool {
	if left.Page != right.Page {
		return left.Page < right.Page
	}
	if left.Destination != right.Destination {
		return left.Destination < right.Destination
	}
	return left.Target < right.Target
}

func lessEdge(left, right GraphEdge) bool {
	if left.From != right.From {
		return left.From < right.From
	}
	return left.To < right.To
}

func lessBroken(left, right BrokenLink) bool {
	if left.Page != right.Page {
		return left.Page < right.Page
	}
	if left.Destination != right.Destination {
		return left.Destination < right.Destination
	}
	if left.Target != right.Target {
		return left.Target < right.Target
	}
	return left.Line < right.Line
}
