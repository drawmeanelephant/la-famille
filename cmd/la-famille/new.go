package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/pathutil"
)

type frontmatterData struct {
	Title  string   `yaml:"title"`
	Date   string   `yaml:"date"`
	Tags   []string `yaml:"tags,omitempty"`
	Layout string   `yaml:"layout,omitempty"`
}

func setupNewCmd(cfg config.Config) *cobra.Command {
	var (
		newTitle      string
		newTags       []string
		newLayout     string
		newDate       string
		newForce      bool
		newContentDir string
	)

	var newCmd = &cobra.Command{
		Use:   "new <slug-or-filename>",
		Short: "Scaffold a Markdown content file with standard frontmatter",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			inputPath := strings.TrimSpace(args[0])
			if inputPath == "" {
				return fmt.Errorf("filename or slug cannot be empty")
			}

			if !strings.HasSuffix(strings.ToLower(inputPath), ".md") {
				inputPath += ".md"
			}

			contentDir := cfg.ContentDir
			if newContentDir != "" {
				contentDir = newContentDir
			}

			targetPath := filepath.Join(contentDir, inputPath)
			if !pathutil.IsSafePath(contentDir, targetPath) {
				return fmt.Errorf("target path %q escapes content directory %q", inputPath, contentDir)
			}

			if _, err := os.Stat(targetPath); err == nil && !newForce {
				return fmt.Errorf("file already exists: %s (use --force to overwrite)", targetPath)
			}

			dateStr := newDate
			if dateStr == "" {
				dateStr = time.Now().Format("2006-01-02")
			} else {
				if _, err := time.Parse(time.DateOnly, dateStr); err != nil {
					return fmt.Errorf("invalid date format %q: must be YYYY-MM-DD", dateStr)
				}
			}

			titleStr := newTitle
			if titleStr == "" {
				titleStr = deriveTitle(inputPath)
			}

			fm := frontmatterData{
				Title:  titleStr,
				Date:   dateStr,
				Tags:   newTags,
				Layout: newLayout,
			}

			yamlBytes, err := yaml.Marshal(fm)
			if err != nil {
				return fmt.Errorf("failed to generate frontmatter: %w", err)
			}

			content := fmt.Sprintf("---\n%s---\n\n# %s\n", string(yamlBytes), titleStr)

			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", filepath.Dir(targetPath), err)
			}

			if err := os.WriteFile(targetPath, []byte(content), 0644); err != nil {
				return fmt.Errorf("failed to write content file %s: %w", targetPath, err)
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Created content file: %s\n\n", targetPath)
			fmt.Fprintln(out, "Next steps:")
			fmt.Fprintf(out, "  1. Edit the file: %s\n", targetPath)
			fmt.Fprintln(out, "  2. Validate content: la-famille check")
			fmt.Fprintln(out, "  3. Preview your site: la-famille serve --watch")

			return nil
		},
	}

	newCmd.Flags().StringVarP(&newTitle, "title", "t", "", "Title of the content file")
	newCmd.Flags().StringSliceVar(&newTags, "tags", nil, "Tags for the content file (comma-separated or multiple flags)")
	newCmd.Flags().StringVar(&newLayout, "layout", "", "Custom layout template for the content file")
	newCmd.Flags().StringVar(&newDate, "date", "", "Publication date in YYYY-MM-DD format (defaults to today)")
	newCmd.Flags().BoolVarP(&newForce, "force", "f", false, "Overwrite existing file if it already exists")
	newCmd.Flags().StringVarP(&newContentDir, "content", "c", cfg.ContentDir, "Directory containing markdown files")

	return newCmd
}

func deriveTitle(input string) string {
	base := filepath.Base(input)
	if ext := filepath.Ext(base); strings.EqualFold(ext, ".md") {
		base = base[:len(base)-len(ext)]
	}
	words := strings.Fields(strings.ReplaceAll(strings.ReplaceAll(base, "-", " "), "_", " "))
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}
