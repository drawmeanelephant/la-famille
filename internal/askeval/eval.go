// Package askeval runs a deterministic golden-question evaluation against
// La Famille's current retrieval ranker.
package askeval

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/tbuddy/la-famille/internal/ask"
	"github.com/tbuddy/la-famille/internal/ragfmt"
	"github.com/tbuddy/la-famille/internal/retrieval"
)

const noAnswerFallbackMessage = "This site does not provide enough information to answer that question."

// Dataset is the checked-in, provider-independent retrieval quality contract.
// Recall is averaged across answerable questions; unanswerable questions are
// checked separately against the ask command's no-answer fallback.
type Dataset struct {
	Version           int      `json:"version"`
	Name              string   `json:"name"`
	K                 int      `json:"k"`
	BaselineRecallAt5 *float64 `json:"baseline_recall_at_5,omitempty"`
	Sites             []Site   `json:"sites"`
}

// Site identifies a fixture tree relative to the project root.
type Site struct {
	ID         string     `json:"id"`
	Fixture    string     `json:"fixture"`
	ContentDir string     `json:"content_dir"`
	Questions  []Question `json:"questions"`
}

// Question records relevant source pages and the minimum acceptable recall.
// Unanswerable questions must have no acceptable pages and must produce the
// canonical no-answer response when the ranker has no matches.
type Question struct {
	ID               string   `json:"id"`
	Text             string   `json:"question"`
	AcceptablePages  []string `json:"acceptable_pages,omitempty"`
	MinimumRecallAtK float64  `json:"minimum_recall_at_k,omitempty"`
	Unanswerable     bool     `json:"unanswerable,omitempty"`
}

// Options configures a single eval run.
type Options struct {
	DatasetPath string
	ProjectRoot string
	Provider    string
}

// QuestionResult is the measured outcome for one golden question.
type QuestionResult struct {
	SiteID           string
	QuestionID       string
	Text             string
	AcceptablePages  []string
	RetrievedPages   []string
	RecallAtK        float64
	MinimumRecallAtK float64
	Unanswerable     bool
	NoAnswer         bool
	NoAnswerMessage  string
	Passed           bool
}

// Report contains the per-question outcomes and aggregate answerable recall.
type Report struct {
	DatasetName       string
	K                 int
	Provider          string
	BaselineRecallAt5 *float64
	RecallAtK         float64
	AnswerableCount   int
	PassedCount       int
	FailedCount       int
	Questions         []QuestionResult
	Passed            bool
}

// Run loads and validates the dataset, exports each fixture to a temporary
// RAG archive, and evaluates the existing retrieval ranker without modifying
// its scoring or index.
func Run(ctx context.Context, opts Options) (Report, error) {
	if strings.TrimSpace(opts.DatasetPath) == "" {
		return Report{}, errors.New("ask eval: dataset path is required")
	}
	if strings.TrimSpace(opts.ProjectRoot) == "" {
		opts.ProjectRoot = "."
	}
	if strings.TrimSpace(opts.Provider) == "" {
		opts.Provider = "fake"
	}

	data, err := readDataset(opts.DatasetPath)
	if err != nil {
		return Report{}, err
	}
	if err := data.Validate(); err != nil {
		return Report{}, err
	}

	projectRoot, err := filepath.Abs(opts.ProjectRoot)
	if err != nil {
		return Report{}, fmt.Errorf("ask eval: resolve project root: %w", err)
	}

	report := Report{
		DatasetName:       data.Name,
		K:                 data.K,
		Provider:          opts.Provider,
		BaselineRecallAt5: data.BaselineRecallAt5,
		Passed:            true,
	}
	recallTotal := 0.0

	for _, site := range data.Sites {
		fixtureRoot := filepath.Join(projectRoot, filepath.FromSlash(site.Fixture))
		if !isWithin(projectRoot, fixtureRoot) {
			return Report{}, fmt.Errorf("ask eval: site %q fixture escapes project root: %q", site.ID, site.Fixture)
		}
		info, err := os.Stat(fixtureRoot)
		if err != nil {
			return Report{}, fmt.Errorf("ask eval: site %q fixture: %w", site.ID, err)
		}
		if !info.IsDir() {
			return Report{}, fmt.Errorf("ask eval: site %q fixture is not a directory: %s", site.ID, fixtureRoot)
		}

		ragDir, err := os.MkdirTemp("", "la-famille-ask-eval-*")
		if err != nil {
			return Report{}, fmt.Errorf("ask eval: create temporary archive: %w", err)
		}
		err = evaluateSite(ctx, site, fixtureRoot, ragDir, opts.Provider, data.K, &report, &recallTotal)
		removeErr := os.RemoveAll(ragDir)
		if err != nil {
			return Report{}, err
		}
		if removeErr != nil {
			return Report{}, fmt.Errorf("ask eval: remove temporary archive: %w", removeErr)
		}
	}

	if report.AnswerableCount > 0 {
		report.RecallAtK = recallTotal / float64(report.AnswerableCount)
	}
	report.Passed = report.FailedCount == 0
	return report, nil
}

func evaluateSite(ctx context.Context, site Site, fixtureRoot, ragDir, provider string, k int, report *Report, recallTotal *float64) error {
	if err := writeContentArchive(fixtureRoot, site.ContentDir, ragDir); err != nil {
		return fmt.Errorf("ask eval: export site %q: %w", site.ID, err)
	}

	loaded, err := retrieval.Load(retrieval.LoadOptions{
		RagDir:     ragDir,
		ContentDir: site.ContentDir,
	})
	if err != nil {
		return fmt.Errorf("ask eval: load site %q corpus: %w", site.ID, err)
	}
	ranker := retrieval.NewRanker(loaded.Corpus)

	var server *ask.Server
	for _, question := range site.Questions {
		if question.Unanswerable {
			server, err = ask.NewServer(ask.Config{
				ProviderName: provider,
				RagDir:       ragDir,
				ContentDir:   site.ContentDir,
				OutputDir:    filepath.Join(ragDir, "no-generated-site"),
				DisableUI:    true,
			})
			if err != nil {
				return fmt.Errorf("ask eval: prepare no-answer check for site %q: %w", site.ID, err)
			}
			break
		}
	}

	for _, question := range site.Questions {
		scored := ranker.Rank(question.Text, k)
		pages := uniquePages(scored)
		result := QuestionResult{
			SiteID:           site.ID,
			QuestionID:       question.ID,
			Text:             question.Text,
			AcceptablePages:  append([]string(nil), question.AcceptablePages...),
			RetrievedPages:   pages,
			MinimumRecallAtK: question.MinimumRecallAtK,
			Unanswerable:     question.Unanswerable,
		}

		if question.Unanswerable {
			if len(scored) == 0 && server != nil {
				answer, err := server.Answer(ctx, ask.AnswerRequest{Question: question.Text})
				if err != nil {
					return fmt.Errorf("ask eval: no-answer check %s/%s: %w", site.ID, question.ID, err)
				}
				result.NoAnswer = answer.NoAnswer &&
					answer.Status == "no_answer" &&
					answer.Diagnostics.ChunksRetrieved == 0 &&
					answer.NoAnswerMessage == noAnswerFallbackMessage
				result.NoAnswerMessage = answer.NoAnswerMessage
			}
			result.Passed = len(scored) == 0 && result.NoAnswer
		} else {
			result.RecallAtK = recallAtK(pages, question.AcceptablePages)
			result.Passed = result.RecallAtK >= question.MinimumRecallAtK
			report.AnswerableCount++
			*recallTotal += result.RecallAtK
		}

		report.Questions = append(report.Questions, result)
		if result.Passed {
			report.PassedCount++
		} else {
			report.FailedCount++
		}
	}
	return nil
}

func writeContentArchive(projectRoot, contentDir, ragDir string) error {
	contentRoot := filepath.Join(projectRoot, filepath.FromSlash(contentDir))
	info, err := os.Stat(contentRoot)
	if err != nil {
		return fmt.Errorf("read content directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("content path is not a directory: %s", contentRoot)
	}

	julesDir := filepath.Join(contentRoot, "jules")
	var markdownFiles []string
	err = filepath.WalkDir(contentRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path == julesDir {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) == ".md" {
			markdownFiles = append(markdownFiles, path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("walk content directory: %w", err)
	}
	if len(markdownFiles) == 0 {
		return fmt.Errorf("content directory contains no Markdown pages: %s", contentRoot)
	}
	sort.Strings(markdownFiles)

	archive, err := os.Create(filepath.Join(ragDir, "rag-content.md"))
	if err != nil {
		return fmt.Errorf("create content archive: %w", err)
	}
	defer archive.Close()
	for _, path := range markdownFiles {
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		relPath, err := filepath.Rel(projectRoot, path)
		if err != nil {
			return fmt.Errorf("make archive path relative: %w", err)
		}
		if _, err := fmt.Fprintf(archive, "<file path=\"%s\">\n<content>\n%s\n</content>\n</file>\n\n",
			filepath.ToSlash(relPath), ragfmt.EscapeContent(string(content))); err != nil {
			return fmt.Errorf("write content archive: %w", err)
		}
	}
	return nil
}

func readDataset(path string) (Dataset, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Dataset{}, fmt.Errorf("ask eval: read dataset: %w", err)
	}
	var data Dataset
	if err := json.Unmarshal(raw, &data); err != nil {
		return Dataset{}, fmt.Errorf("ask eval: parse dataset: %w", err)
	}
	return data, nil
}

// Validate checks the dataset structure before any fixture is exported.
func (d Dataset) Validate() error {
	if d.Version != 1 {
		return fmt.Errorf("ask eval: unsupported dataset version %d", d.Version)
	}
	if strings.TrimSpace(d.Name) == "" {
		return errors.New("ask eval: dataset name is required")
	}
	if d.K < 1 {
		return errors.New("ask eval: k must be at least 1")
	}
	if d.BaselineRecallAt5 != nil && (*d.BaselineRecallAt5 < 0 || *d.BaselineRecallAt5 > 1) {
		return errors.New("ask eval: baseline_recall_at_5 must be between 0 and 1")
	}
	if d.BaselineRecallAt5 != nil && d.K != 5 {
		return errors.New("ask eval: baseline_recall_at_5 requires k=5")
	}
	if len(d.Sites) == 0 {
		return errors.New("ask eval: dataset must contain at least one site")
	}

	siteIDs := make(map[string]bool)
	questionCount := 0
	answerableCount := 0
	for _, site := range d.Sites {
		if strings.TrimSpace(site.ID) == "" {
			return errors.New("ask eval: site id is required")
		}
		if siteIDs[site.ID] {
			return fmt.Errorf("ask eval: duplicate site id %q", site.ID)
		}
		siteIDs[site.ID] = true
		if !filepath.IsLocal(site.Fixture) || filepath.Clean(site.Fixture) == "." {
			return fmt.Errorf("ask eval: site %q fixture must be a project-relative path", site.ID)
		}
		if !filepath.IsLocal(site.ContentDir) || strings.TrimSpace(site.ContentDir) == "" {
			return fmt.Errorf("ask eval: site %q content_dir must be a local path", site.ID)
		}
		if len(site.Questions) == 0 {
			return fmt.Errorf("ask eval: site %q must contain at least one question", site.ID)
		}

		questionIDs := make(map[string]bool)
		for _, question := range site.Questions {
			questionCount++
			if strings.TrimSpace(question.ID) == "" {
				return fmt.Errorf("ask eval: site %q question id is required", site.ID)
			}
			if questionIDs[question.ID] {
				return fmt.Errorf("ask eval: site %q has duplicate question id %q", site.ID, question.ID)
			}
			questionIDs[question.ID] = true
			if strings.TrimSpace(question.Text) == "" {
				return fmt.Errorf("ask eval: site %q question %q text is required", site.ID, question.ID)
			}
			if question.Unanswerable {
				if len(question.AcceptablePages) != 0 {
					return fmt.Errorf("ask eval: unanswerable question %s/%s cannot list acceptable pages", site.ID, question.ID)
				}
				continue
			}
			answerableCount++
			if len(question.AcceptablePages) == 0 {
				return fmt.Errorf("ask eval: answerable question %s/%s needs acceptable_pages", site.ID, question.ID)
			}
			if question.MinimumRecallAtK <= 0 || question.MinimumRecallAtK > 1 {
				return fmt.Errorf("ask eval: minimum_recall_at_k for %s/%s must be greater than 0 and at most 1", site.ID, question.ID)
			}
			pageIDs := make(map[string]bool)
			for _, page := range question.AcceptablePages {
				if strings.TrimSpace(page) == "" || !isLocalPageID(page) {
					return fmt.Errorf("ask eval: invalid acceptable page %q in %s/%s", page, site.ID, question.ID)
				}
				if pageIDs[page] {
					return fmt.Errorf("ask eval: duplicate acceptable page %q in %s/%s", page, site.ID, question.ID)
				}
				pageIDs[page] = true
			}
		}
	}
	if questionCount == 0 || answerableCount == 0 {
		return errors.New("ask eval: dataset must contain at least one answerable question")
	}
	return nil
}

func isLocalPageID(page string) bool {
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(page)))
	return clean == page && clean != "." && clean != ".." && !strings.HasPrefix(clean, "../") && !strings.HasPrefix(clean, "/")
}

func isWithin(root, target string) bool {
	rel, err := filepath.Rel(root, target)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

func uniquePages(scored []retrieval.Scored) []string {
	seen := make(map[string]bool, len(scored))
	pages := make([]string, 0, len(scored))
	for _, match := range scored {
		page := match.Chunk.PageID
		if page == "" || seen[page] {
			continue
		}
		seen[page] = true
		pages = append(pages, page)
	}
	return pages
}

func recallAtK(retrieved, acceptable []string) float64 {
	if len(acceptable) == 0 {
		return 0
	}
	retrievedSet := make(map[string]bool, len(retrieved))
	for _, page := range retrieved {
		retrievedSet[page] = true
	}
	hits := 0
	for _, page := range acceptable {
		if retrievedSet[page] {
			hits++
		}
	}
	return float64(hits) / float64(len(acceptable))
}

// WriteReport writes a compact, stable per-question table and overall
// recall@K. A non-zero result is left to the caller so the table can still be
// emitted before the CLI exits with a failure status.
func WriteReport(w io.Writer, report Report) error {
	if w == nil {
		return errors.New("ask eval: output writer is required")
	}
	if _, err := fmt.Fprintf(w, "Ask This Site evaluation: %s\nRanker: BM25-lite · K=%d · provider=%s\n\n",
		report.DatasetName, report.K, report.Provider); err != nil {
		return err
	}
	table := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	if _, err := fmt.Fprintln(table, "SITE\tQUESTION\tRECALL@K\tMINIMUM\tSTATUS\tRETRIEVED PAGES"); err != nil {
		return err
	}
	for _, result := range report.Questions {
		recall := fmt.Sprintf("%.2f", result.RecallAtK)
		minimum := fmt.Sprintf("%.2f", result.MinimumRecallAtK)
		if result.Unanswerable {
			recall = "n/a"
			minimum = "no-answer"
		}
		status := "FAIL"
		if result.Passed {
			status = "PASS"
		}
		if _, err := fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t%s\n",
			result.SiteID, result.QuestionID, recall, minimum, status, strings.Join(result.RetrievedPages, ", ")); err != nil {
			return err
		}
	}
	if err := table.Flush(); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "\nRecall@%d: %.2f (%d answerable questions); questions passed: %d/%d",
		report.K, report.RecallAtK, report.AnswerableCount, report.PassedCount, report.PassedCount+report.FailedCount); err != nil {
		return err
	}
	if report.BaselineRecallAt5 != nil {
		if _, err := fmt.Fprintf(w, " · recorded BM25-lite recall@5 baseline: %.2f", *report.BaselineRecallAt5); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(w)
	return err
}
