package content

import "testing"

func TestPublishedFilesDefaultsToIncludedAndHonorsFlag(t *testing.T) {
	publish := true
	exclude := false
	all := map[string]*FileMeta{
		"default.md": {Title: "Default"},
		"include.md": {Title: "Included", Publish: &publish},
		"exclude.md": {Title: "Excluded", Publish: &exclude},
	}

	got := PublishedFiles(all)
	if len(got) != 2 {
		t.Fatalf("PublishedFiles() returned %d pages, want 2: %v", len(got), got)
	}
	if got["default.md"] == nil || got["include.md"] == nil {
		t.Errorf("published pages = %v, want default.md and include.md", got)
	}
	if _, exists := got["exclude.md"]; exists {
		t.Error("publish:false page is included")
	}
	if len(all) != 3 {
		t.Errorf("PublishedFiles() modified the source map: %v", all)
	}
}
