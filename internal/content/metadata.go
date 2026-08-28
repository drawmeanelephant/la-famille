package content

import (
	"bytes"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/adrg/frontmatter"
)

var validTagRegex = regexp.MustCompile(`^[a-z0-9-]+$`)

// MaxTaxonomyValueLen bounds the length of a normalized tag or category.
// A taxonomy value becomes a single path component in the output tree
// (<output>/tags/<tag>/index.html) and common filesystems reject a component
// longer than 255 bytes with ENAMETOOLONG, which aborts the entire build and
// leaves no output at all. Anything longer is dropped with a warning instead.
const MaxTaxonomyValueLen = 255

type FileMeta struct {
	Content         []byte
	Rest            []byte // The content after frontmatter
	Tags            []string
	Categories      []string
	RelPath         string
	Title           string
	Author          string
	Date            string
	VideoScript     string
	AnimationCues   string
	SoundtrackTheme string
	Layout          string
	ComplianceModal string
	Slug            string
	Description     string
	Image           string
	Render          *bool
	Warnings        []string
}

// GatherMetadata walks the content directory and parses the frontmatter for each markdown file.
func GatherMetadata(contentDir string) (map[string]*FileMeta, error) {
	fileMap := make(map[string]*FileMeta)

	err := filepath.WalkDir(contentDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("error accessing path %s: %w", path, err)
		}

		if d.Type()&os.ModeSymlink != 0 {
			slog.Warn("Skipping symlink in content", "path", path)
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".md" {
			return nil
		}

		relPath, err := filepath.Rel(contentDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path for %s: %w", path, err)
		}
		// Always use forward slashes for internal map keys to match web links
		relPath = filepath.ToSlash(relPath)

		contentBytes, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", path, err)
		}

		var warnings []string

		// Parse into a generic map to normalize casing first
		var rawMatter map[string]interface{}
		rest, err := frontmatter.Parse(bytes.NewReader(contentBytes), &rawMatter)
		if err != nil {
			// If frontmatter parsing fails, treat the whole file as content
			rest = contentBytes
			warnMsg := fmt.Sprintf("frontmatter parse warning in %s: %v, falling back to raw markdown", relPath, err)
			warnings = append(warnings, warnMsg)
			slog.Warn("Frontmatter parse failed", "file", relPath, "error", err)
		} else if len(rawMatter) == 0 && bytes.Equal(rest, contentBytes) && frontmatterOpener(string(contentBytes)) != "" {
			// An unterminated block (an opener with no matching closer) is a
			// silent failure mode: adrg/frontmatter reports it as "no
			// frontmatter" with no error, so the whole document, opener and
			// all, rendered as body text and the summary stayed at warnings=0
			// (#530). The parse error case above already warns; this catches
			// the one the library cannot distinguish from an ordinary
			// frontmatter-less file.
			warnMsg := fmt.Sprintf("unterminated frontmatter in %s: %s opens a frontmatter block that is never closed with the matching delimiter, so it renders as body text", relPath, frontmatterOpener(string(contentBytes)))
			warnings = append(warnings, warnMsg)
			slog.Warn("Unterminated frontmatter block", "file", relPath)
		}

		var matter struct {
			Category        interface{} `yaml:"category"`
			Categories      interface{} `yaml:"categories"`
			Render          *bool       `yaml:"render"`
			SoundtrackTheme string      `yaml:"soundtrack_theme"`
			VideoScript     string      `yaml:"video_script"`
			AnimationCues   string      `yaml:"animation_cues"`
			Title           string      `yaml:"title"`
			Layout          string      `yaml:"layout"`
			ComplianceModal string      `yaml:"compliance_modal"`
			Slug            string      `yaml:"slug"`
			Date            string      `yaml:"date"`
			Author          string      `yaml:"author"`
			Description     string      `yaml:"description"`
			Image           string      `yaml:"image"`
			Tags            StringList  `yaml:"tags"`
		}

		for _, detail := range DecodeFrontmatter(rawMatter, &matter) {
			warnMsg := fmt.Sprintf("frontmatter warning in %s: %s", relPath, detail)
			warnings = append(warnings, warnMsg)
			slog.Warn("Frontmatter warning", "file", relPath, "detail", detail)
		}

		// Date validation
		if matter.Date != "" {
			if _, err := time.Parse(time.DateOnly, matter.Date); err != nil {
				warnMsg := fmt.Sprintf("invalid date format in %s: %s", relPath, matter.Date)
				warnings = append(warnings, warnMsg)
				slog.Warn("Invalid date format", "file", relPath, "date", matter.Date)
				matter.Date = ""
			}
		}

		var rawCategories []string
		rawCategories = append(rawCategories, extractStringSlice(matter.Categories)...)
		rawCategories = append(rawCategories, extractStringSlice(matter.Category)...)

		normalizedTags := normalizeTaxonomyList(matter.Tags, relPath, "tag", &warnings)
		normalizedCategories := normalizeTaxonomyList(rawCategories, relPath, "category", &warnings)

		fileMap[relPath] = &FileMeta{
			RelPath:         relPath,
			Title:           matter.Title,
			Author:          matter.Author,
			Date:            matter.Date,
			Render:          matter.Render,
			VideoScript:     matter.VideoScript,
			AnimationCues:   matter.AnimationCues,
			SoundtrackTheme: matter.SoundtrackTheme,
			Layout:          matter.Layout,
			ComplianceModal: matter.ComplianceModal,
			Slug:            matter.Slug,
			Tags:            normalizedTags,
			Categories:      normalizedCategories,
			Content:         contentBytes,
			Rest:            rest,
			Description:     matter.Description,
			Image:           matter.Image,
			Warnings:        warnings,
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk content directory: %w", err)
	}

	return fileMap, nil
}

func extractStringSlice(val interface{}) []string {
	if val == nil {
		return nil
	}
	switch v := val.(type) {
	case string:
		if strings.TrimSpace(v) != "" {
			return []string{strings.TrimSpace(v)}
		}
	case []interface{}:
		var res []string
		for _, elem := range v {
			if s, ok := elem.(string); ok && strings.TrimSpace(s) != "" {
				res = append(res, strings.TrimSpace(s))
			}
		}
		return res
	case []string:
		var res []string
		for _, s := range v {
			if strings.TrimSpace(s) != "" {
				res = append(res, strings.TrimSpace(s))
			}
		}
		return res
	}
	return nil
}

// NormalizeTaxonomyValue reduces one raw tag or category to the [a-z0-9-] form
// used for output paths. The second return value reports whether the result is
// usable: an empty result, or one longer than MaxTaxonomyValueLen, is not.
//
// Exported so internal/checker can validate against the same rules the
// generator publishes with, instead of its own copy of them.
func NormalizeTaxonomyValue(item string) (string, bool) {
	item = strings.TrimSpace(item)
	if item == "" {
		return "", false
	}

	norm := item
	if !validTagRegex.MatchString(item) {
		lower := strings.ToLower(item)
		var sb strings.Builder
		for _, r := range lower {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
				sb.WriteRune(r)
			}
		}
		norm = sb.String()
	}

	if norm == "" || len(norm) > MaxTaxonomyValueLen {
		return norm, false
	}
	return norm, true
}

func normalizeTaxonomyList(items []string, relPath, kind string, warnings *[]string) []string {
	var normalizedList []string
	seen := make(map[string]bool)

	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		norm, usable := NormalizeTaxonomyValue(item)
		if !usable {
			// A tag that normalizes to an empty string used to vanish with only
			// a log line while the build summary stayed at warnings=0, so a
			// purely non-ASCII tag silently lost its taxonomy page (#532).
			// Dropping one is lossy, so it must be counted, naming the file.
			if len(norm) > MaxTaxonomyValueLen {
				msg := fmt.Sprintf("dropped over-long %s in %s: %q normalizes to %d bytes, over the %d-byte limit", kind, relPath, item, len(norm), MaxTaxonomyValueLen)
				*warnings = append(*warnings, msg)
				slog.Warn("Dropped over-long "+kind, "length", len(norm), "limit", MaxTaxonomyValueLen, "file", relPath)
			} else {
				msg := fmt.Sprintf("dropped %s in %s: %q normalizes to an empty value, which cannot be published as a path", kind, relPath, item)
				*warnings = append(*warnings, msg)
				slog.Warn("Dropped unusable "+kind, "original", item, "file", relPath)
			}
			continue
		}
		if norm != item {
			// The current normalization strips runs of non-ASCII characters
			// ("café ☕" → "caf"), so this line doubles as the signal that a
			// value was mangled. Counting it makes the lossy rewrite visible in
			// the build summary instead of living only in a log line (#532).
			warnMsg := fmt.Sprintf("normalized %s in %s: %q became %q", kind, relPath, item, norm)
			*warnings = append(*warnings, warnMsg)
			slog.Warn("Normalized "+kind, "original", item, "normalized", norm, "file", relPath)
		}
		if !seen[norm] {
			seen[norm] = true
			normalizedList = append(normalizedList, norm)
		}
	}
	return normalizedList
}

// frontmatterOpener returns the opening delimiter line if content begins with
// an adrg/frontmatter opener, and "" otherwise. Only the default YAML/TOML/JSON
// delimiters are considered, matching what GatherMetadata relies on.
func frontmatterOpener(content string) string {
	for _, opener := range []string{"---", "+++", ";;;"} {
		if strings.HasPrefix(content, opener+"\n") {
			return opener
		}
	}
	return ""
}
