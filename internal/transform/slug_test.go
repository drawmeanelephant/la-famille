package transform

import "testing"

func TestIsUsableSlug(t *testing.T) {
	cases := []struct {
		slug string
		want bool
	}{
		{"clean", true},
		{"with-dashes", true},
		{"with_underscores", true},
		{"v1.0", false},        // a dot would split the output directory name
		{"nested/path", false}, // a separator would move the page
		{"../escape", false},   // traversal
		{"/absolute", false},   // not local
		{"file.html", false},   // dot again
	}
	for _, c := range cases {
		if got := IsUsableSlug(c.slug); got != c.want {
			t.Errorf("IsUsableSlug(%q) = %v, want %v", c.slug, got, c.want)
		}
	}
}

// TestGetOutputURLIgnoresUnusableSlug is the contract that keeps every URL
// predictor honest. The renderer drops a slug it cannot use and falls back to
// the filename, so a consumer that trusted the raw frontmatter value emitted a
// link to a directory that was never written — a taxonomy listing pointing at
// /v1.0/ for a page published at /post/.
func TestGetOutputURLIgnoresUnusableSlug(t *testing.T) {
	cases := []struct {
		name    string
		relPath string
		slug    string
		want    string
	}{
		{"usable slug is honoured", "post.md", "custom", "custom/index.html"},
		{"dotted slug is ignored", "post.md", "v1.0", "post/index.html"},
		{"separator slug is ignored", "post.md", "a/b", "post/index.html"},
		{"traversal slug is ignored", "post.md", "../evil", "post/index.html"},
		{"empty slug keeps the filename", "post.md", "", "post/index.html"},
		{"nested page keeps its directory", "docs/post.md", "v2.0", "docs/post/index.html"},
		{"index is unaffected", "index.md", "v1.0", "index.html"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := GetOutputURL(c.relPath, c.slug, true); got != c.want {
				t.Errorf("GetOutputURL(%q, %q, true) = %q, want %q", c.relPath, c.slug, got, c.want)
			}
		})
	}
}
