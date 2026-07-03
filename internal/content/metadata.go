package content

import (
	"bytes"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/adrg/frontmatter"
)

type FileMeta struct {
	RelPath         string
	Title           string
	Author          string
	Date            string
	Render          *bool
	VideoScript     string
	AnimationCues   string
	SoundtrackTheme string
	Layout          string
	ComplianceModal string
	Slug            string
	Tags            []string
	Content         []byte
	Rest            []byte // The content after frontmatter
	Description     string
	Image           string
}

// GatherMetadata walks the content directory and parses the frontmatter for each markdown file.
func GatherMetadata(contentDir string) (map[string]*FileMeta, error) {
	fileMap := make(map[string]*FileMeta)

	err := filepath.WalkDir(contentDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("error accessing path %s: %w", path, err)
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

		// Parse into a generic map to normalize casing first
		var rawMatter map[string]interface{}
		rest, err := frontmatter.Parse(bytes.NewReader(contentBytes), &rawMatter)
		if err != nil {
			// If frontmatter parsing fails, treat the whole file as content
			rest = contentBytes
		}

		var matter struct {
			Title           string   `yaml:"title"`
			Author          string   `yaml:"author"`
			Date            string   `yaml:"date"`
			Render          *bool    `yaml:"render"`
			VideoScript     string   `yaml:"video_script"`
			AnimationCues   string   `yaml:"animation_cues"`
			SoundtrackTheme string   `yaml:"soundtrack_theme"`
			Layout          string   `yaml:"layout"`
			ComplianceModal string   `yaml:"compliance_modal"`
			Slug            string   `yaml:"slug"`
			Tags            []string `yaml:"tags"`
			Description     string   `yaml:"description"`
			Image           string   `yaml:"image"`
		}

		if rawMatter != nil {
			// Lowercase keys
			normalizedMatter := make(map[string]interface{})
			for k, v := range rawMatter {
				// Convert to lower case, but preserve underscores for things like video_script
				normalizedMatter[strings.ToLower(k)] = v
			}

			yamlBytes, err := yaml.Marshal(normalizedMatter)
			if err == nil {
				_ = yaml.Unmarshal(yamlBytes, &matter)
			}
		}

		// Date validation
		if matter.Date != "" {
			if _, err := time.Parse(time.DateOnly, matter.Date); err != nil {
				log.Printf("Warning: Invalid date format in %s: %s", relPath, matter.Date)
				matter.Date = ""
			}
		}

		// Tag validation and normalization
		var normalizedTags []string
		for _, tag := range matter.Tags {
			lower := strings.ToLower(tag)
			var sb strings.Builder
			for _, r := range lower {
				if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
					sb.WriteRune(r)
				}
			}
			normalized := sb.String()
			if normalized != tag {
				log.Printf("Warning: Normalized tag '%s' to '%s' in %s", tag, normalized, relPath)
			}
			if normalized != "" {
				normalizedTags = append(normalizedTags, normalized)
			}
		}

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
			Content:         contentBytes,
			Rest:            rest,
			Description:     matter.Description,
			Image:           matter.Image,
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk content directory: %w", err)
	}

	return fileMap, nil
}
