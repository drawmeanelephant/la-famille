package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tbuddy/la-famille/internal/runtimeassets"
)

func TestDemoContentFilesSwitchAwayFromDefaultTheme(t *testing.T) {
	curated := map[string]bool{}
	for _, name := range runtimeassets.CuratedLayoutNames {
		curated[name] = true
	}
	for _, theme := range []string{"", "layout", "layout-octoburger", "layout-terminal"} {
		demos := demoContentFiles(theme, time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC))
		for name, data := range demos {
			// A scaffold that fails its own first-run hygiene check (#517)
			// ships warnings to every fresh site: descriptions are required.
			if !strings.Contains(string(data), "description:") {
				t.Errorf("theme %q: demo %s is missing a description frontmatter field", theme, name)
			}
		}
		theming, ok := demos["theming.md"]
		if !ok {
			t.Fatalf("theme %q: demo packet missing theming.md", theme)
		}
		body := string(theming)
		defaultLayout := theme
		if defaultLayout == "" {
			defaultLayout = "layout"
		}
		pinned := ""
		for _, name := range runtimeassets.CuratedLayoutNames {
			if strings.Contains(body, "\nlayout: "+name+"\n") {
				pinned = name
				break
			}
		}
		if pinned == "" {
			t.Errorf("theme %q: theming.md does not pin a bundled layout via frontmatter:\n%s", theme, body)
			continue
		}
		if pinned == defaultLayout {
			t.Errorf("theme %q: theming.md pins the site default instead of switching away", theme)
		}
		if !curated[pinned] {
			t.Errorf("theme %q: theming.md pins non-bundled layout %q", theme, pinned)
		}
	}
}

// Issue #529: the scaffolded homepage carries a tag so a fresh init + build
// demonstrates the tag archive flow (and the nav link) immediately, instead of
// leaving a binary-only author to discover tags from build output.
func TestDemoContentIndexCarriesTag(t *testing.T) {
	demos := demoContentFiles("", time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC))
	index, ok := demos["index.md"]
	if !ok {
		t.Fatalf("demo packet missing index.md")
	}
	body := string(index)
	if !strings.Contains(body, "tags:") || !strings.Contains(body, "- welcome") {
		t.Errorf("scaffolded index.md should carry a tags: list, got:\n%s", body)
	}
}

func TestScaffoldDemoContentCreatesAndPreserves(t *testing.T) {
	dir := t.TempDir()
	demos := demoContentFiles("layout-octoburger", time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC))

	created, err := scaffoldDemoContent(dir, demos)
	if err != nil {
		t.Fatal(err)
	}
	if len(created) != len(demos) {
		t.Errorf("expected %d files created, got %v", len(demos), created)
	}
	for rel, want := range demos {
		got, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("expected scaffolded %s: %v", rel, err)
		}
		if string(got) != string(want) {
			t.Errorf("scaffolded %s content mismatch", rel)
		}
	}

	// An operator edit must survive a re-run of init.
	if err := os.WriteFile(filepath.Join(dir, "index.md"), []byte("my own words"), 0600); err != nil {
		t.Fatal(err)
	}
	created, err = scaffoldDemoContent(dir, demos)
	if err != nil {
		t.Fatal(err)
	}
	if len(created) != 0 {
		t.Errorf("expected no files created on re-run, got %v", created)
	}
	got, err := os.ReadFile(filepath.Join(dir, "index.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "my own words" {
		t.Errorf("re-run clobbered operator content, got %q", got)
	}
}

func TestFormatThemeChoicesListsEveryTheme(t *testing.T) {
	choices := formatThemeChoices()
	for _, name := range runtimeassets.CuratedLayoutNames {
		if !strings.Contains(choices, name) {
			t.Errorf("formatThemeChoices output missing %q:\n%s", name, choices)
		}
	}
	for _, theme := range runtimeassets.CuratedThemes() {
		if !strings.Contains(choices, theme.Description) {
			t.Errorf("formatThemeChoices output missing description for %q:\n%s", theme.Name, choices)
		}
	}
}
