package content

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

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

		rest, err := frontmatter.Parse(bytes.NewReader(contentBytes), &matter)
		if err != nil {
			// If frontmatter parsing fails, treat the whole file as content
			rest = contentBytes
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
