package github

import "testing"

func TestDefaultBranch(t *testing.T) {
	tests := []struct {
		name   string
		branch string
		want   string
	}{
		{name: "empty uses repository default", want: "master"},
		{name: "configured branch is preserved", branch: "release", want: "release"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := defaultBranch(tt.branch); got != tt.want {
				t.Errorf("defaultBranch(%q) = %q, want %q", tt.branch, got, tt.want)
			}
		})
	}
}
