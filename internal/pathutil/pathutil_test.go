package pathutil

import (
	"os"
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
		{
			name:     "equal paths are safe",
			base:     "/app/public",
			target:   "/app/public",
			expected: true,
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

func TestIsPathWithin(t *testing.T) {
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
			name:     "empty base",
			base:     "",
			target:   "/app/public/index.html",
			expected: false,
		},
		{
			name:     "empty target",
			base:     "/app/public",
			target:   "",
			expected: false,
		},
		{
			name:     "equal paths are not within",
			base:     "/app/public",
			target:   "/app/public",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := filepath.FromSlash(tt.base)
			target := filepath.FromSlash(tt.target)
			actual := IsPathWithin(base, target)
			if actual != tt.expected {
				t.Errorf("IsPathWithin(%q, %q) = %v; expected %v", base, target, actual, tt.expected)
			}
		})
	}
}

func TestIsPathWithinSymlink(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a real directory and a symlink to it
	realDir := filepath.Join(tmpDir, "real")
	linkDir := filepath.Join(tmpDir, "link")
	if err := os.MkdirAll(realDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realDir, linkDir); err != nil {
		t.Skip("cannot create symlink:", err)
	}

	// A file inside the real directory should be within the link
	// (since symlinks are resolved)
	insideFile := filepath.Join(realDir, "file.txt")
	if _, err := os.Create(insideFile); err != nil {
		t.Fatal(err)
	}

	// Both directions should work after symlink resolution
	if !IsPathWithin(linkDir, insideFile) {
		t.Error("file inside real dir should be within symlinked dir")
	}
}