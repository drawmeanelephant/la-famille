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
	Title      string   `yaml:"title"`
	Date       string   `yaml:"date"`
	Layout     string   `yaml:"layout,omitempty"`
	Tags       []string `yaml:"tags,omitempty"`
	Categories []string `yaml:"categories,omitempty"`
}

func setupNewCmd(cfg config.Config) *cobra.Command {
	var (
		newTitle      string
		newTags       []string
		newCategories []string
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
				contentDir = resolveProjectPath(cfg.ProjectRoot, newContentDir)
			}

			cleanInput := inputPath
			slashContent := filepath.ToSlash(contentDir)
			slashInput := filepath.ToSlash(inputPath)
			if strings.HasPrefix(slashInput, slashContent+"/") {
				cleanInput = slashInput[len(slashContent)+1:]
			}

			targetPath := filepath.Join(contentDir, cleanInput)
			if !pathutil.IsSafePath(contentDir, targetPath) {
				return fmt.Errorf("target path %q escapes content directory %q", inputPath, contentDir)
			}
			// A slug with spaces or uppercase ships broken URLs and malformed
			// sitemap entries, so it is normalized up front and reported
			// rather than created verbatim (#508).
			normalizedInput, normalized := normalizeSlugSegments(cleanInput)
			if normalized && !pathutil.IsSafePath(contentDir, filepath.Join(contentDir, normalizedInput)) {
				return fmt.Errorf("normalized path %q escapes content directory %q", normalizedInput, contentDir)
			}
			if normalized {
				targetPath = filepath.Join(contentDir, normalizedInput)
				cleanInput = normalizedInput
			}
			// IsSafePath is lexical: it compares path strings and does not
			// resolve symlinks. A link inside the content tree therefore passes
			// it while redirecting the write outside the tree entirely.
			if err := refuseSymlinkedComponents(contentDir, targetPath); err != nil {
				return err
			}

			if _, err := os.Lstat(targetPath); err == nil && !newForce {
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
				Title:      titleStr,
				Date:       dateStr,
				Tags:       newTags,
				Categories: newCategories,
				Layout:     newLayout,
			}

			yamlBytes, err := yaml.Marshal(fm)
			if err != nil {
				return fmt.Errorf("failed to generate frontmatter: %w", err)
			}

			content := fmt.Sprintf("---\n%s---\n\n# %s\n", string(yamlBytes), titleStr)

			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", filepath.Dir(targetPath), err)
			}

			// Re-check containment against the canonical paths now that the
			// parents exist, so nothing that appeared between the check above
			// and this write can redirect it.
			if err := refuseEscapeAfterResolution(contentDir, targetPath); err != nil {
				return err
			}

			if err := writeNewContentFile(targetPath, content, newForce); err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			if normalized {
				fmt.Fprintf(out, "note: normalized slug %q to %q\n", args[0], cleanInput)
			}
			fmt.Fprintf(out, "Created content file: %s\n\n", targetPath)
			fmt.Fprintln(out, "Next steps:")
			fmt.Fprintf(out, "  1. Edit the file: %s\n", displayPathFromCwd(targetPath))
			// Hints follow the documented workflow (#511): a --project-root
			// form when one is in play, CWD-relative paths everywhere.
			checkCmd := "la-famille check"
			serveCmd := "la-famille serve --watch"
			if rootHint := projectRootHint(cfg.ProjectRoot); rootHint != "" {
				checkCmd = fmt.Sprintf("la-famille --project-root %s check", rootHint)
				serveCmd = fmt.Sprintf("la-famille --project-root %s serve --watch", rootHint)
			} else if cmd.Flags().Changed("content") {
				checkCmd = fmt.Sprintf("la-famille check --content %s", displayPathFromCwd(contentDir))
			}
			fmt.Fprintf(out, "  2. Validate content: %s\n", checkCmd)
			fmt.Fprintf(out, "  3. Preview your site: %s\n", serveCmd)

			return nil
		},
	}

	newCmd.Flags().StringVarP(&newTitle, "title", "t", "", "Title of the content file")
	newCmd.Flags().StringSliceVar(&newTags, "tags", nil, "Tags for the content file (comma-separated or multiple flags)")
	newCmd.Flags().StringSliceVar(&newCategories, "categories", nil, "Categories for the content file (comma-separated or multiple flags)")
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

// normalizeSlugSegments rewrites each path segment of a content filename into
// a URL-safe form: lowercase, [a-z0-9-_.], runs of anything else collapsed to
// a single hyphen. It reports whether anything changed (#508).
func normalizeSlugSegments(input string) (string, bool) {
	slash := filepath.ToSlash(input)
	segments := strings.Split(slash, "/")
	changed := false
	for i, seg := range segments {
		if seg == "" {
			continue
		}
		ext := filepath.Ext(seg)
		isMarkdown := strings.EqualFold(ext, ".md")
		name := seg
		if isMarkdown {
			name = seg[:len(seg)-len(ext)]
		}
		norm := normalizeSlugName(name)
		if isMarkdown {
			norm += ".md"
		}
		if norm != seg {
			segments[i] = norm
			changed = true
		}
	}
	return strings.Join(segments, "/"), changed
}

// normalizeSlugName lowercases and maps every character outside the safe set
// to a single hyphen. It mirrors internal/checker's suggestion logic so `new`
// creates exactly what `check` would recommend renaming to.
func normalizeSlugName(name string) string {
	var sb strings.Builder
	lastHyphen := false
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			sb.WriteRune(r)
			lastHyphen = false
		case r == '-' || r == '_' || r == '.':
			sb.WriteRune(r)
			lastHyphen = r == '-'
		default:
			if !lastHyphen {
				sb.WriteRune('-')
				lastHyphen = true
			}
		}
	}
	normalized := strings.Trim(sb.String(), "-")
	if normalized == "" {
		return "untitled"
	}
	return normalized
}

// displayPathFromCwd renders an absolute path relative to the working
// directory when possible so the printed hint matches what the user can type.
func displayPathFromCwd(p string) string {
	cwd, err := os.Getwd()
	if err != nil {
		return p
	}
	rel, err := filepath.Rel(cwd, p)
	if err != nil || strings.HasPrefix(rel, "..") {
		return p
	}
	return rel
}

// projectRootHint renders --project-root value for hint commands; "" means the
// project root is the working directory and the flag would be noise.
func projectRootHint(root string) string {
	if root == "" || root == "." {
		return ""
	}
	cwd, err := os.Getwd()
	if err != nil {
		return root
	}
	rel, err := filepath.Rel(cwd, root)
	if err != nil || rel == "." {
		return ""
	}
	if strings.HasPrefix(rel, "..") {
		return root
	}
	return rel
}

// refuseSymlinkedComponents walks the destination one component at a time and
// rejects any existing component that is a symlink.
//
// pathutil.IsSafePath compares path strings only. A symlinked directory inside
// the content tree satisfies it while sending the write wherever the link
// points, so `new link/page` could create a file outside the content root
// entirely. Components that do not exist yet are fine: MkdirAll creates real
// directories, never links.
func refuseSymlinkedComponents(root, target string) error {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return fmt.Errorf("target path %q escapes content directory %q", target, root)
	}

	current := root
	for _, part := range strings.Split(filepath.ToSlash(rel), "/") {
		if part == "" || part == "." {
			continue
		}
		current = filepath.Join(current, part)

		info, statErr := os.Lstat(current)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				continue
			}
			return fmt.Errorf("failed to inspect %s: %w", current, statErr)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing to write through symlink %s: a link inside the content directory can redirect the write outside it", current)
		}
	}
	return nil
}

// refuseEscapeAfterResolution re-checks containment using canonical paths, so a
// destination is confirmed inside the content root as it exists on disk rather
// than as it was spelled on the command line.
func refuseEscapeAfterResolution(root, target string) error {
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return fmt.Errorf("failed to resolve content directory %s: %w", root, err)
	}
	resolvedParent, err := filepath.EvalSymlinks(filepath.Dir(target))
	if err != nil {
		return fmt.Errorf("failed to resolve destination directory for %s: %w", target, err)
	}

	resolvedTarget := filepath.Join(resolvedParent, filepath.Base(target))
	if !pathutil.IsSafePath(resolvedRoot, resolvedTarget) {
		return fmt.Errorf("target path %q resolves to %q, outside content directory %q", target, resolvedTarget, resolvedRoot)
	}
	return nil
}

// writeNewContentFile creates the file without following a symlink at the
// destination. Without O_EXCL a pre-existing link would be followed and the
// file it points at truncated, which is exactly what --force must not be
// allowed to mean.
func writeNewContentFile(target, content string, force bool) error {
	if force {
		if info, err := os.Lstat(target); err == nil && info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing to overwrite symlink %s even with --force: it points outside the file you named", target)
		}
		if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to replace %s: %w", target, err)
		}
	}

	file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return fmt.Errorf("failed to write content file %s: %w", target, err)
	}
	defer file.Close()

	if _, err := file.WriteString(content); err != nil {
		return fmt.Errorf("failed to write content file %s: %w", target, err)
	}
	return nil
}
