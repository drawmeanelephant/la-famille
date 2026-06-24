package content

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

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
	Content         []byte
	Rest            []byte // The content after frontmatter
}

// GatherMetadata walks the content directory and parses the frontmatter for each markdown file.
func GatherMetadata(contentDir string) (map[string]*FileMeta, error) {
	fileMap := make(map[string]*FileMeta)

	err := filepath.WalkDir(contentDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".md" {
			return nil
		}

		relPath, err := filepath.Rel(contentDir, path)
		if err != nil {
			return err
		}
		// Always use forward slashes for internal map keys to match web links
		relPath = filepath.ToSlash(relPath)

		contentBytes, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		// Parse into a generic map to normalize casing first
		var rawMatter map[string]interface{}
		rest, err := frontmatter.Parse(bytes.NewReader(contentBytes), &rawMatter)
		if err != nil {
			// If frontmatter parsing fails, treat the whole file as content
			rest = contentBytes
		}

		var matter struct {
			Title           string `yaml:"title"`
			Author          string `yaml:"author"`
			Date            string `yaml:"date"`
			Render          *bool  `yaml:"render"`
			VideoScript     string `yaml:"video_script"`
			AnimationCues   string `yaml:"animation_cues"`
			SoundtrackTheme string `yaml:"soundtrack_theme"`
			Layout          string `yaml:"layout"`
		}

		if rawMatter != nil {
			// Lowercase keys
			normalizedMatter := make(map[string]interface{})
			for k, v := range rawMatter {
				// Convert to lower case, but preserve underscores for things like video_script
				normalizedMatter[strings.ToLower(k)] = v
			}

			// Safe extraction
			if v, ok := normalizedMatter["title"].(string); ok {
				matter.Title = v
			}
			if v, ok := normalizedMatter["author"].(string); ok {
				matter.Author = v
			}
			if v, ok := normalizedMatter["date"].(string); ok {
				matter.Date = v
			}
			if v, ok := normalizedMatter["render"].(bool); ok {
				matter.Render = &v
			}
			if v, ok := normalizedMatter["video_script"].(string); ok {
				matter.VideoScript = v
			}
			if v, ok := normalizedMatter["animation_cues"].(string); ok {
				matter.AnimationCues = v
			}
			if v, ok := normalizedMatter["soundtrack_theme"].(string); ok {
				matter.SoundtrackTheme = v
			}
			if v, ok := normalizedMatter["layout"].(string); ok {
				matter.Layout = v
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
			Content:         contentBytes,
			Rest:            rest,
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk content directory: %w", err)
	}

	return fileMap, nil
}
