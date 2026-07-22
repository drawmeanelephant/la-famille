package render

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

var (
	htmlLangRE   = regexp.MustCompile(`(?is)<html\b[^>]*\blang\s*=\s*"[^"]+"`)
	titleRE      = regexp.MustCompile(`(?is)<title\b[^>]*>\s*.+?\s*</title>`)
	mainRE       = regexp.MustCompile(`(?is)<main\b[^>]*\bid\s*=\s*"main-content"`)
	skipRE       = regexp.MustCompile(`(?is)<a\b[^>]*\bhref\s*=\s*"#main-content"`)
	headingRE    = regexp.MustCompile(`(?is)<h([1-6])\b`)
	imageRE      = regexp.MustCompile(`(?is)<img\b[^>]*>`)
	ariaHiddenRE = regexp.MustCompile(`(?i)\baria-hidden\s*=\s*"true"`)
	altRE        = regexp.MustCompile(`(?i)\balt\s*=\s*"([^"]*)"`)
)

func TestLayoutsMeetAccessibilityStructure(t *testing.T) {
	templatesDir := filepath.Join("..", "..", "templates")
	entries, err := os.ReadDir(templatesDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".html" {
			continue
		}
		t.Run(entry.Name(), func(t *testing.T) {
			path := filepath.Join(templatesDir, entry.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			source := string(data)
			for name, pattern := range map[string]*regexp.Regexp{
				"html lang":     htmlLangRE,
				"title":         titleRE,
				"main landmark": mainRE,
				"skip link":     skipRE,
			} {
				if !pattern.MatchString(source) {
					t.Errorf("missing accessible %s", name)
				}
			}

			headings := headingRE.FindAllStringSubmatch(source, -1)
			h1Count := 0
			previous := 0
			for _, heading := range headings {
				level, _ := strconv.Atoi(heading[1])
				if level == 1 {
					h1Count++
				}
				if previous > 0 && level > previous+1 {
					t.Errorf("heading level jumps from h%d to h%d", previous, level)
				}
				previous = level
			}
			if h1Count != 1 {
				t.Errorf("expected exactly one h1, found %d", h1Count)
			}

			for _, image := range imageRE.FindAllString(source, -1) {
				if ariaHiddenRE.MatchString(image) {
					alt := altRE.FindStringSubmatch(image)
					if len(alt) == 0 || strings.TrimSpace(alt[1]) != "" {
						t.Errorf("aria-hidden image must have an empty alt attribute: %s", image)
					}
				}
			}
		})
	}
}
