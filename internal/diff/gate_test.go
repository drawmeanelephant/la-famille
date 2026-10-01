package diff

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/sitedata"
)

func TestGateRegressionClasses(t *testing.T) {
	before := sitedata.Manifest{
		Version: sitedata.ManifestVersion, OutputCaptured: true,
		Pages: []sitedata.ManifestPage{
			{Identity: "index", URL: "/", Rendered: true, ContentHash: "old", Links: []sitedata.ManifestLink{{Destination: "note.md", Target: "note.md", Resolved: true}}},
			{Identity: "note", URL: "/note/", Rendered: true, InboundLinkCount: 1, Tags: []string{"keep"}, Categories: []string{"guides"}},
			{Identity: "old-orphan", URL: "/old-orphan/", Rendered: true},
			{Identity: "raw.md", Tags: []string{"keep", "private"}},
		},
		Sitemap: []string{"/", "/note/", "/old-orphan/"},
		Files:   []sitedata.ManifestFile{{Path: "assets/image.png", Hash: "old"}},
	}
	for _, test := range []struct {
		name string
		edit func(*sitedata.Manifest)
		kind string
	}{
		{"prose", func(m *sitedata.Manifest) { m.Pages[0].ContentHash = "new" }, ""},
		{"asset", func(m *sitedata.Manifest) { m.Files[0].Hash = "new" }, ""},
		{"existing debt", func(_ *sitedata.Manifest) {}, ""},
		{"unpublished taxonomy", func(m *sitedata.Manifest) { m.Pages[3].Tags = nil }, ""},
		{"broken link", func(m *sitedata.Manifest) { m.Pages[0].Links[0].Resolved = false }, "broken_link"},
		{"new orphan", func(m *sitedata.Manifest) { m.Pages[1].InboundLinkCount = 0 }, "orphan"},
		{"removed linked page", func(m *sitedata.Manifest) { m.Pages = append(m.Pages[:1], m.Pages[2:]...) }, "removed_linked_page"},
		{"vanished tag", func(m *sitedata.Manifest) { m.Pages[1].Tags = nil }, "taxonomy"},
		{"vanished category", func(m *sitedata.Manifest) { m.Pages[1].Categories = nil }, "taxonomy"},
		{"lost sitemap entry", func(m *sitedata.Manifest) { m.Sitemap = m.Sitemap[:1] }, "sitemap"},
		{"renamed linked URL", func(m *sitedata.Manifest) { m.Pages[1].Identity = "new"; m.Pages[1].URL = "/new/" }, "removed_linked_page"},
	} {
		t.Run(test.name, func(t *testing.T) {
			data, _ := json.Marshal(before)
			var after sitedata.Manifest
			if err := json.Unmarshal(data, &after); err != nil {
				t.Fatal(err)
			}
			test.edit(&after)
			report, err := Compare(before, after)
			if err != nil {
				t.Fatal(err)
			}
			if test.kind == "" {
				if err := report.GateError(); err != nil {
					t.Fatal(err)
				}
			} else {
				if report.GateError() == nil {
					t.Fatal("regression passed the gate")
				}
				found := false
				for _, regression := range report.Regressions {
					found = found || regression.Kind == test.kind
				}
				if !found {
					t.Fatalf("missing %s: %+v", test.kind, report.Regressions)
				}
			}
			for _, regression := range report.Regressions {
				if regression.Page == "old-orphan" {
					t.Fatal("pre-existing orphan classified as regression")
				}
			}
		})
	}
}

func TestLegacyAndIncompleteGateFailClosed(t *testing.T) {
	for _, version := range []int{1, 2} {
		m := sitedata.Manifest{Version: version}
		report, err := Compare(m, m)
		if err != nil {
			t.Fatal(err)
		}
		if report.GateError() == nil || len(report.CoverageWarnings) == 0 {
			t.Fatal("incomplete snapshot passed gate")
		}
		if strings.Contains(report.Summary("a", "b"), "No changes.\n") {
			t.Fatal("incomplete comparison claimed no changes")
		}
	}
}

func TestBrokenLinkLineMovementDoesNotRegress(t *testing.T) {
	before := sitedata.Manifest{Version: 2, OutputCaptured: true, Pages: []sitedata.ManifestPage{
		{Identity: "index", Links: []sitedata.ManifestLink{{Destination: "missing.md", Target: "missing.md", Line: 1}}},
	}}
	after := sitedata.Manifest{Version: 2, OutputCaptured: true, Pages: []sitedata.ManifestPage{
		{Identity: "index", Links: []sitedata.ManifestLink{{Destination: "missing.md", Target: "missing.md", Line: 20}}},
	}}
	report, err := Compare(before, after)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Empty() || report.GateError() != nil {
		t.Fatalf("line movement caused noise: %+v", report)
	}
}

func TestTemplateOnlyOutputIsVisible(t *testing.T) {
	a := sitedata.Manifest{Version: 2, OutputCaptured: true, Pages: []sitedata.ManifestPage{{Identity: "index", OutputHash: "old"}}}
	b := sitedata.Manifest{Version: 2, OutputCaptured: true, Pages: []sitedata.ManifestPage{{Identity: "index", OutputHash: "new"}}}
	r, err := Compare(a, b)
	if err != nil {
		t.Fatal(err)
	}
	if r.Empty() || len(r.RenderedPages) != 1 || len(r.ChangedPages) != 0 || r.GateError() != nil {
		t.Fatalf("output drift hidden or mislabeled: %+v", r)
	}
}
