package transform

import "testing"

func TestGetOutputURL(t *testing.T) {
	tests := []struct {
		name     string
		relPath  string
		slug     string
		expected string
	}{
		{
			name:     "standard md file",
			relPath:  "about.md",
			slug:     "",
			expected: "about/index.html",
		},
		{
			name:     "index md file",
			relPath:  "index.md",
			slug:     "",
			expected: "index.html",
		},
		{
			name:     "standard md file with slug",
			relPath:  "about.md",
			slug:     "bio",
			expected: "bio/index.html",
		},
		{
			name:     "index md file with slug",
			relPath:  "index.md",
			slug:     "home",
			expected: "index.html",
		},
		{
			name:     "nested standard md file",
			relPath:  "blog/post.md",
			slug:     "",
			expected: "blog/post/index.html",
		},
		{
			name:     "nested index md file",
			relPath:  "blog/index.md",
			slug:     "",
			expected: "blog/index.html",
		},
		{
			name:     "nested md file with slug",
			relPath:  "blog/post.md",
			slug:     "my-post",
			expected: "blog/my-post/index.html",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := GetOutputURL(tc.relPath, tc.slug)
			if actual != tc.expected {
				t.Errorf("GetOutputURL(%q, %q) = %q; expected %q", tc.relPath, tc.slug, actual, tc.expected)
			}
		})
	}
}

func TestGetOutputURLBoundaries(t *testing.T) {
	tests := []struct {
		name    string
		relPath string
		slug    string
		want    string
	}{
		{"root index", "index.md", "", "index.html"},
		{"nested index", "docs/index.md", "", "docs/index.html"},
		{"root page", "about.md", "", "about/index.html"},
		{"nested page", "docs/install.md", "", "docs/install/index.html"},
		{"slug replaces filename", "docs/install.md", "setup", "docs/setup/index.html"},
		{"index ignores slug by current contract", "index.md", "home", "index.html"},
		{"nested index ignores slug by current contract", "docs/index.md", "home", "docs/index.html"},
		{"dot-like source basename", "docs/v1.2.md", "", "docs/v1.2/index.html"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetOutputURL(tt.relPath, tt.slug); got != tt.want {
				t.Fatalf("GetOutputURL(%q, %q) = %q, want %q",
					tt.relPath, tt.slug, got, tt.want)
			}
		})
	}
}
