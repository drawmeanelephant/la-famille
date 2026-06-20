package git

import (
	"testing"
)

func TestParseOwnerRepo(t *testing.T) {
	tests := []struct {
		url           string
		expectedOwner string
		expectedRepo  string
		expectError   bool
	}{
		{"https://github.com/tbuddy/la-famille.git", "tbuddy", "la-famille", false},
		{"https://github.com/tbuddy/la-famille", "tbuddy", "la-famille", false},
		{"http://github.com/owner/repo.git", "owner", "repo", false},
		{"git@github.com:tbuddy/la-famille.git", "tbuddy", "la-famille", false},
		{"git@github.com:owner/repo", "owner", "repo", false},
		{"https://gitlab.com/owner/repo", "", "", true},
		{"invalid-url", "", "", true},
	}

	for _, tt := range tests {
		owner, repo, err := ParseOwnerRepo(tt.url)
		if tt.expectError {
			if err == nil {
				t.Errorf("expected error for url %q, got none", tt.url)
			}
			continue
		}
		if err != nil {
			t.Errorf("unexpected error for url %q: %v", tt.url, err)
			continue
		}
		if owner != tt.expectedOwner {
			t.Errorf("for url %q expected owner %q, got %q", tt.url, tt.expectedOwner, owner)
		}
		if repo != tt.expectedRepo {
			t.Errorf("for url %q expected repo %q, got %q", tt.url, tt.expectedRepo, repo)
		}
	}
}
