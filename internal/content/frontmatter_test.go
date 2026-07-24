package content

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeContentFile drops a single markdown file into a fresh content dir and
// returns its parsed metadata.
func writeContentFile(t *testing.T, body string) *FileMeta {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "page.md"), []byte(body), 0600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	fileMap, err := GatherMetadata(dir)
	if err != nil {
		t.Fatalf("GatherMetadata failed: %v", err)
	}
	meta, ok := fileMap["page.md"]
	if !ok {
		t.Fatalf("page.md missing from file map")
	}
	return meta
}

func TestGatherMetadataTagShapes(t *testing.T) {
	cases := []struct {
		name     string
		frontier string
		wantTags []string
	}{
		// The defect: a scalar tags: value was dropped, while the same shape
		// works for category:.
		{"scalar string", "tags: golang", []string{"golang"}},
		{"scalar quoted string", `tags: "Golang"`, []string{"golang"}},

		// Every shape below already worked before the scalar support was
		// added and must keep working byte for byte: yaml coerces non string
		// scalars inside a sequence, and those tags are legal under
		// validTagRegex, so dropping them would delete published taxonomy URLs.
		{"flow sequence of strings", "tags: [go, rust]", []string{"go", "rust"}},
		{"flow sequence with int", "tags: [2024, golang]", []string{"2024", "golang"}},
		{"flow sequence all ints", "tags: [2024, 2025]", []string{"2024", "2025"}},
		{"flow sequence with float", "tags: [1.5]", []string{"15"}},
		{"flow sequence with bool", "tags: [go, true]", []string{"go", "true"}},
		{"block sequence with int", "tags:\n  - 2024\n  - go", []string{"2024", "go"}},

		// Shapes that carried no usable tag before and still carry none.
		{"scalar int", "tags: 42", nil},
		{"scalar bool", "tags: yes", nil},
		{"scalar float", "tags: 1.5", nil},
		{"mapping", "tags: {a: b}", nil},
		{"empty sequence", "tags: []", nil},
		{"null", "tags:", nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			meta := writeContentFile(t, "---\ntitle: T\n"+tc.frontier+"\n---\nBody.\n")
			if strings.Join(meta.Tags, ",") != strings.Join(tc.wantTags, ",") {
				t.Errorf("tags = %v, want %v", meta.Tags, tc.wantTags)
			}
			if meta.Title != "T" {
				t.Errorf("title = %q, want %q (sibling fields must survive a tags type error)", meta.Title, "T")
			}
		})
	}
}

// A tag longer than one path component aborted the whole build with
// ENAMETOOLONG and produced no output at all. It is dropped with a warning
// instead. The limit is the filesystem's, so a tag at the limit must still be
// published exactly as before.
func TestGatherMetadataOverLongTagIsDropped(t *testing.T) {
	atLimit := strings.Repeat("c", MaxTaxonomyValueLen)
	overLimit := strings.Repeat("c", MaxTaxonomyValueLen+1)

	meta := writeContentFile(t, fmt.Sprintf("---\ntitle: T\ntags: [%q]\n---\nBody.\n", atLimit))
	if len(meta.Tags) != 1 || meta.Tags[0] != atLimit {
		t.Errorf("tag of exactly %d bytes must be kept, got %v", MaxTaxonomyValueLen, meta.Tags)
	}

	meta = writeContentFile(t, fmt.Sprintf("---\ntitle: T\ntags: [%q]\n---\nBody.\n", overLimit))
	if len(meta.Tags) != 0 {
		t.Errorf("tag of %d bytes must be dropped, got %v", MaxTaxonomyValueLen+1, meta.Tags)
	}

	// Same guard through the scalar spelling, which only became reachable
	// once scalar tags were honoured.
	meta = writeContentFile(t, fmt.Sprintf("---\ntitle: T\ntags: %q\n---\nBody.\n", overLimit))
	if len(meta.Tags) != 0 {
		t.Errorf("over-long scalar tag must be dropped, got %v", meta.Tags)
	}
}

func TestNormalizeTaxonomyValue(t *testing.T) {
	cases := []struct {
		in       string
		wantNorm string
		wantOK   bool
	}{
		{"golang", "golang", true},
		{"Inv@lid_Tag", "invlidtag", true},
		{"  spaced  ", "spaced", true},
		{"", "", false},
		{"!!!", "", false},
		{strings.Repeat("c", MaxTaxonomyValueLen), strings.Repeat("c", MaxTaxonomyValueLen), true},
		{strings.Repeat("c", MaxTaxonomyValueLen+1), strings.Repeat("c", MaxTaxonomyValueLen+1), false},
	}
	for _, tc := range cases {
		gotNorm, gotOK := NormalizeTaxonomyValue(tc.in)
		if gotNorm != tc.wantNorm || gotOK != tc.wantOK {
			t.Errorf("NormalizeTaxonomyValue(%q) = (%q, %v), want (%q, %v)", tc.in, gotNorm, gotOK, tc.wantNorm, tc.wantOK)
		}
	}
}

// Case-variant spellings of one key used to be collapsed by ranging a Go map,
// so the surviving title, slug or render flag was picked at random on every
// build -- which moves a page's published URL between builds.
func TestNormalizeFrontmatterKeysIsDeterministic(t *testing.T) {
	raw := map[string]interface{}{
		"Title":  "alpha",
		"title":  "beta",
		"TITLE":  "gamma",
		"Slug":   "alpha-slug",
		"slug":   "beta-slug",
		"Render": false,
		"render": true,
		"author": "solo",
	}

	first, _ := NormalizeFrontmatterKeys(raw)
	for i := 0; i < 500; i++ {
		got, _ := NormalizeFrontmatterKeys(raw)
		for k, want := range first {
			if got[k] != want {
				t.Fatalf("run %d: key %q resolved to %v, first run gave %v (resolution is not deterministic)", i, k, got[k], want)
			}
		}
		if len(got) != len(first) {
			t.Fatalf("run %d: %d keys, first run had %d", i, len(got), len(first))
		}
	}

	// The exact lowercase spelling is the documented winner.
	if first["title"] != "beta" {
		t.Errorf("title = %v, want %q (exact lowercase spelling should win)", first["title"], "beta")
	}
	if first["slug"] != "beta-slug" {
		t.Errorf("slug = %v, want %q", first["slug"], "beta-slug")
	}
	if first["render"] != true {
		t.Errorf("render = %v, want true", first["render"])
	}
	if first["author"] != "solo" {
		t.Errorf("author = %v, want %q (a key with no variants must pass through)", first["author"], "solo")
	}
}

func TestNormalizeFrontmatterKeysWarnsOnCollapsedVariants(t *testing.T) {
	raw := map[string]interface{}{
		"Title":  "alpha",
		"title":  "beta",
		"author": "solo",
	}
	_, warnings := NormalizeFrontmatterKeys(raw)
	if len(warnings) != 1 {
		t.Fatalf("expected exactly 1 warning, got %d: %v", len(warnings), warnings)
	}
	for _, want := range []string{"Title", "title", "duplicate frontmatter key"} {
		if !strings.Contains(warnings[0], want) {
			t.Errorf("warning %q does not mention %q", warnings[0], want)
		}
	}

	_, warnings = NormalizeFrontmatterKeys(map[string]interface{}{"title": "beta"})
	if len(warnings) != 0 {
		t.Errorf("a key with no variants must not warn, got %v", warnings)
	}
}

// Type mismatches used to be discarded by `_ = yaml.Unmarshal(...)`, so a value
// vanished with no diagnostic at all -- unlike every other rejected frontmatter
// value, which logs a warning.
func TestDecodeFrontmatterWarnsOnUnusableValue(t *testing.T) {
	var out struct {
		Title  string `yaml:"title"`
		Render *bool  `yaml:"render"`
	}
	warnings := DecodeFrontmatter(map[string]interface{}{
		"title":  "kept",
		"render": "true",
	}, &out)

	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning for the unusable render value, got %d: %v", len(warnings), warnings)
	}
	// yaml.v2 reports the source line and the offending value but not the
	// destination field name, so that is the whole of what is guaranteed here.
	if !strings.Contains(warnings[0], "true") || !strings.Contains(warnings[0], "line 1") {
		t.Errorf("warning %q does not locate the offending value", warnings[0])
	}
	// Warnings are appended to a sorted list and persisted in the build cache,
	// so they must stay on one line.
	if strings.Contains(warnings[0], "\n") {
		t.Errorf("warning must be a single line, got %q", warnings[0])
	}
	if out.Title != "kept" {
		t.Errorf("title = %q, want %q (a sibling type error must not discard good values)", out.Title, "kept")
	}
	// Unchanged from before this package reported anything: yaml.v2 allocates
	// the pointer and leaves it at the zero value when the scalar will not
	// decode. The warning is the only new signal.
	if out.Render == nil || *out.Render {
		t.Errorf("render = %v, want a non-nil pointer to false", out.Render)
	}

	if got := DecodeFrontmatter(nil, &out); got != nil {
		t.Errorf("nil frontmatter must produce no warnings, got %v", got)
	}
	if got := DecodeFrontmatter(map[string]interface{}{"title": "fine"}, &out); len(got) != 0 {
		t.Errorf("clean frontmatter must produce no warnings, got %v", got)
	}
}

func TestGatherMetadataReportsFrontmatterWarnings(t *testing.T) {
	meta := writeContentFile(t, "---\nTitle: alpha\ntitle: beta\nrender: \"true\"\n---\nBody.\n")

	var sawDuplicate, sawUnusable bool
	for _, w := range meta.Warnings {
		if !strings.Contains(w, "page.md") {
			t.Errorf("warning %q does not name the file it came from", w)
		}
		if strings.Contains(w, "duplicate frontmatter key") {
			sawDuplicate = true
		}
		if strings.Contains(w, "frontmatter value ignored") {
			sawUnusable = true
		}
	}
	if !sawDuplicate {
		t.Errorf("expected a duplicate-key warning, got %v", meta.Warnings)
	}
	if !sawUnusable {
		t.Errorf("expected an ignored-value warning, got %v", meta.Warnings)
	}
}
