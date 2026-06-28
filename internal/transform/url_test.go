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
			expected: "home/index.html",
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
