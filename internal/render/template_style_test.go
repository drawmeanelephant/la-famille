package render_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBundledLayoutsUseSharedStyleFoundations(t *testing.T) {
	root := filepath.Join("..", "..", "templates")
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".html") {
			continue
		}
		contents, err := os.ReadFile(filepath.Join(root, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(contents), "/assets/css/theme-foundations.css") {
			t.Errorf("%s does not include shared style foundations", entry.Name())
		}
		if !strings.Contains(string(contents), `name="viewport"`) {
			t.Errorf("%s is missing a responsive viewport", entry.Name())
		}
	}
}
