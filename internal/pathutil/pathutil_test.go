package pathutil

import (
	"path/filepath"
	"testing"
)

func TestIsSafePath(t *testing.T) {
	tests := []struct {
		name     string
		base     string
		target   string
		expected bool
	}{
		{
			name:     "safe file inside base",
			base:     "/app/public",
			target:   "/app/public/index.html",
			expected: true,
		},
		{
			name:     "safe nested directory",
			base:     "/app/public",
			target:   "/app/public/blog/posts/post.html",
			expected: true,
		},
		{
			name:     "unsafe parent traversal",
			base:     "/app/public",
			target:   "/app/public/../../etc/passwd",
			expected: false,
		},
		{
			name:     "unsafe same level folder breakout",
			base:     "/app/public",
			target:   "/app/private/keys.json",
			expected: false,
		},
		{
			name:     "unsafe relative escape",
			base:     "public",
			target:   "public/../private/secrets.txt",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := filepath.FromSlash(tt.base)
			target := filepath.FromSlash(tt.target)
			actual := IsSafePath(base, target)
			if actual != tt.expected {
				t.Errorf("IsSafePath(%q, %q) = %v; expected %v", base, target, actual, tt.expected)
			}
		})
	}
}
