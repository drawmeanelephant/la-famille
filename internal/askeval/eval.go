// Package askeval runs deterministic golden-question evaluations against
// La Famille's current retrieval ranker without changing production scoring.
package askeval

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/tbuddy/la-famille/internal/ask"
	"github.com/tbuddy/la-famille/internal/graph"
	"github.com/tbuddy/la-famille/internal/llm"
	"github.com/tbuddy/la-famille/internal/retrieval"
)

const noAnswerFallbackMessage = "This site does not provide enough information to answer that question."

// Dataset is the provider-independent retrieval contract. Macro averages
// exclude unanswerable questions. AcceptablePages names all required evidence,
// not interchangeable alternatives. The original version-1 schema is retained.
type Dataset struct {
	Version             int      `json:"version"`
	Name                string   `json:"name"`
	K                   int      `json:"k"`
	BaselineRecallAt5   *float64 `json:"baseline_recall_at_5,omitempty"`
	Sites               []Site   `json:"sites"`
	MinimumPrecisionAtK float64  `json:"minimum_precision_at_k,omitempty"`
}

// Site identifies read-only source content relative to the project root.
type Site struct {
	ID         string     `json:"id"`
	Fixture    string     `json:"fixture"`
	ContentDir string     `json:"content_dir"`
	Questions  []Question `json:"questions"`
}

// Question records evidence floors and optional graph-route labels. A zero
// recall floor requires MeasurementOnly so known misses are not success claims.
// Strict unanswerable controls require zero retrieval and canonical fallback.
type Question struct {
	RequireGroundedPath bool     `json:"require_grounded_path,omitempty"`
	ID                  string   `json:"id"`
	Text                string   `json:"question"`
	AcceptablePages     []string `json:"acceptable_pages,omitempty"`
	MinimumRecallAtK    float64  `json:"minimum_recall_at_k"`
	Unanswerable        bool     `json:"unanswerable,omitempty"`
	Class               string   `json:"class,omitempty"`
	PathPages           []string `json:"path_pages,omitempty"`
	MinimumPrecisionAtK *float64 `json:"minimum_precision_at_k,omitempty"`
	MeasurementOnly     bool     `json:"measurement_only,omitempty"`
}

// Options configures an eval run. A zero K uses the dataset's original depth.
type Options struct {
	GraphExpansion    bool
	CompareGraph      bool
	DatasetPath       string
	ProjectRoot       string
	Provider          string
	K                 int
	Embeddings        bool
	EmbeddingModel    string
	EmbeddingCacheDir string
	Embedder          retrieval.Embedder
	// Ranker overrides the production BM25-lite scorer so that alternative
	// retrieval arms (#581 embeddings, hybrid fusion, graph expansion) can be
	// measured through the same gates. Nil uses the production ranker, which
	// is the default and the only behaviour any release claim may rest on.
	Ranker func(Corpus) Scorer
}

// Corpus and Scorer are the minimal ranking surface the harness measures, so a
// candidate arm can be evaluated without importing internal/retrieval.
type Corpus = retrieval.Corpus

// Scorer ranks chunks for a query. retrieval.Ranker satisfies it.
type Scorer interface {
	Rank(query string, topK int) []retrieval.Scored
}

// QuestionResult keeps known misses visible independently of passing floors.
// PathCoverage measures retrieved route pages, not generated answer provenance.
type QuestionResult struct {
	GroundedPath     bool
	Answer           string
	Paths            []retrieval.GraphPath
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
	Class            string
	MissingPages     []string
	PrecisionAtK     float64
	PathCoverage     *float64
}

// Metrics are macro averages over answerable questions only.
type Metrics struct {
	RecallAtK    float64
	PrecisionAtK float64
	Questions    int
}

// Report retains the original recall fields and adds precision and diagnostics.
// Baseline comparison is meaningful only at K=5; improvements are permitted.
type Report struct {
	Measurements      []SiteMeasurement
	Comparison        *GraphComparison
	DatasetName       string
	K                 int
	Provider          string
	Ranker            string
	BaselineRecallAt5 *float64
	RecallAtK         float64
	AnswerableCount   int
	PassedCount       int
	FailedCount       int
	Questions         []QuestionResult
	Passed            bool
	PrecisionAtK      float64
	BaselineOK        bool
	PrecisionOK       bool
	Classes           map[string]Metrics
}

// Run builds and exports content-only fixtures in disposable projects, then
// measures the unchanged ranker. Nothing is written into source fixtures.
func Run(ctx context.Context, opts Options) (Report, error) {
	if opts.CompareGraph {
		return compareGraph(ctx, opts)
	}
	if opts.GraphExpansion && opts.Embeddings {
		return Report{}, errors.New("ask eval: graph expansion is lexical-only; disable embeddings")
	}
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
	if opts.K < 0 || opts.K > 100 {
		return Report{}, errors.New("ask eval: k override must be between 1 and 100 (or zero for dataset default)")
	}
	if opts.K > 0 {
		data.K = opts.K
	}
	if opts.Provider != "fake" && opts.Provider != "ollama" {
		return Report{}, fmt.Errorf("ask eval: unsupported provider %q", opts.Provider)
	}
	projectRoot, err := filepath.Abs(opts.ProjectRoot)
	if err != nil {
		return Report{}, fmt.Errorf("ask eval: resolve project root: %w", err)
	}
	report := Report{
		DatasetName: data.Name, K: data.K, Provider: opts.Provider,
		BaselineRecallAt5: data.BaselineRecallAt5, BaselineOK: true,
		Classes: make(map[string]Metrics),
		Ranker:  "BM25-lite",
	}
	if opts.GraphExpansion {
		report.Ranker = "BM25-lite + link graph"
	}
	if opts.Embeddings {
		report.Ranker = "BM25-lite + Ollama embeddings (RRF)"
		if opts.EmbeddingModel == "" {
			opts.EmbeddingModel = "nomic-embed-text"
		}
		if opts.Embedder == nil {
			opts.Embedder = llm.NewOllama(llm.OllamaConfig{})
		}
		if opts.EmbeddingCacheDir == "" {
			userCache, err := os.UserCacheDir()
			if err != nil {
				return Report{}, fmt.Errorf("ask eval: locate embedding cache: %w", err)
			}
			opts.EmbeddingCacheDir = filepath.Join(userCache, "la-famille", "ask-eval")
		}
	}
	for _, site := range data.Sites {
		if err := ctx.Err(); err != nil {
			return Report{}, err
		}
		tmp, err := os.MkdirTemp("", "la-famille-ask-eval-*")
		if err != nil {
			return Report{}, fmt.Errorf("ask eval: create temporary project: %w", err)
		}
		err = evaluateSite(ctx, site, projectRoot, tmp, opts, data.K, data.MinimumPrecisionAtK, &report)
		removeErr := os.RemoveAll(tmp)
		if err != nil {
			return Report{}, err
		}
		if removeErr != nil {
			return Report{}, fmt.Errorf("ask eval: remove temporary project: %w", removeErr)
		}
	}
	for _, site := range report.Measurements {
		for _, query := range site.Queries {
			if report.Ranker != query.Ranker {
				report.Ranker = "mixed (see per-question measurement modes)"
			}
		}
	}
	if report.AnswerableCount > 0 {
		report.RecallAtK /= float64(report.AnswerableCount)
		report.PrecisionAtK /= float64(report.AnswerableCount)
	}
	for class, metrics := range report.Classes {
		metrics.RecallAtK /= float64(metrics.Questions)
		metrics.PrecisionAtK /= float64(metrics.Questions)
		report.Classes[class] = metrics
	}
	if report.K == 5 && report.BaselineRecallAt5 != nil {
		report.BaselineOK = report.RecallAtK+1e-9 >= *report.BaselineRecallAt5
	}
	report.PrecisionOK = report.PrecisionAtK+1e-9 >= data.MinimumPrecisionAtK
	report.Passed = report.FailedCount == 0 && report.BaselineOK && report.PrecisionOK
	return report, nil
}

func evaluateSite(ctx context.Context, site Site, projectRoot, tmp string, opts Options, k int, precisionFloor float64, report *Report) error {
	cfg, err := prepareSite(projectRoot, tmp, site)
	if err != nil {
		return fmt.Errorf("ask eval: build fixture %q: %w", site.ID, err)
	}
	loaded, err := retrieval.Load(retrieval.LoadOptions{
		RagDir: cfg.RagDir, OutputDir: cfg.OutputDir, ContentDir: "content",
	})
	if err != nil {
		return fmt.Errorf("ask eval: load site %q corpus: %w", site.ID, err)
	}
	newRanker := opts.Ranker
	if newRanker == nil {
		newRanker = func(c Corpus) Scorer { return retrieval.NewRanker(c) }
	}
	ranker := newRanker(loaded.Corpus)
	mode := "BM25-lite"
	measurement := SiteMeasurement{
		SiteID: site.ID, CorpusDigest: retrieval.CorpusDigest(loaded.Corpus), Chunks: len(loaded.Corpus.Chunks),
	}
	var embedder *measuredEmbedder
	if opts.GraphExpansion {
		ranker = retrieval.NewGraphRanker(loaded.Corpus)
		mode = "BM25-lite + link graph"
	}
	if opts.Embeddings {
		key := sha256.Sum256([]byte(projectRoot + "\x00" + site.ID + "\x00" + site.Fixture + "\x00" + site.ContentDir))
		path := filepath.Join(opts.EmbeddingCacheDir, hex.EncodeToString(key[:]), retrieval.VectorFileName)
		measurement.EmbeddingModel, measurement.CachePath = opts.EmbeddingModel, path
		measurement.IndexSHA256Before, err = indexSHA256(path)
		if err != nil {
			return err
		}
		embedder = &measuredEmbedder{inner: opts.Embedder}
		// Disposable eval builds have a fresh project path and binary hash
		// each run. Their loaded corpus digest is the stable input fingerprint.
		start := time.Now()
		hybrid, err := retrieval.NewHybridRanker(ctx, loaded.Corpus, embedder, opts.EmbeddingModel,
			path, retrieval.CorpusDigest(loaded.Corpus))
		measurement.IndexDuration, measurement.ChunkEmbedding = time.Since(start), embedder.stats
		if err != nil {
			if !errors.Is(err, llm.ErrUnavailable) {
				return fmt.Errorf("ask eval: embed site %q: %w", site.ID, err)
			}
			mode = "BM25-lite + Ollama embeddings (RRF; lexical fallback: unavailable)"
			measurement.FallbackReason = err.Error()
		} else {
			ranker = hybrid
			mode = "BM25-lite + Ollama embeddings (RRF)"
		}
		report.Ranker = mode
		measurement.IndexSHA256After, err = indexSHA256(path)
		if err != nil {
			return err
		}
	}
	graphBytes, err := os.ReadFile(filepath.Join(cfg.OutputDir, "graph.json"))
	if err != nil {
		return err
	}
	var g graph.Graph
	if err := json.Unmarshal(graphBytes, &g); err != nil {
		return err
	}
	var server *ask.Server
	for _, question := range site.Questions {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := validateLabels(question, loaded.Corpus, g); err != nil {
			return fmt.Errorf("ask eval: site %s: %w", site.ID, err)
		}
		var scored []retrieval.Scored
		queryMeasurement := QueryMeasurement{
			QuestionID: question.ID, Ranker: mode, FallbackReason: measurement.FallbackReason,
		}
		if embedder != nil {
			embedder.stats = EmbeddingMeasurement{}
		}
		start := time.Now()
		if hybrid, ok := ranker.(*retrieval.HybridRanker); ok {
			scored, err = hybrid.RankContext(ctx, question.Text, k)
			if err != nil {
				if !errors.Is(err, llm.ErrUnavailable) {
					return fmt.Errorf("ask eval: embed question %s/%s: %w", site.ID, question.ID, err)
				}
				report.Ranker = "BM25-lite + Ollama embeddings (RRF; lexical fallback: unavailable)"
				queryMeasurement.Ranker, queryMeasurement.FallbackReason = report.Ranker, err.Error()
				scored = retrieval.NewRanker(loaded.Corpus).Rank(question.Text, k)
			}
		} else {
			scored = ranker.Rank(question.Text, k)
		}
		queryMeasurement.RankDuration = time.Since(start)
		if embedder != nil {
			queryMeasurement.Embedding = embedder.stats
			queryMeasurement.RankDuration -= embedder.stats.Duration
		}
		measurement.Queries = append(measurement.Queries, queryMeasurement)
		pages := uniquePages(scored)
		result := QuestionResult{
			SiteID: site.ID, QuestionID: question.ID, Text: question.Text,
			AcceptablePages: slices.Clone(question.AcceptablePages), RetrievedPages: pages,
			MinimumRecallAtK: question.MinimumRecallAtK, Unanswerable: question.Unanswerable, Class: question.Class,
		}
		if question.Unanswerable {
			// Nonempty retrieval already fails this strict gate. Do not invoke a
			// model whose synthetic/semantic fallback could mask that failure.
			if len(scored) == 0 {
				if server == nil {
					server, err = ask.NewServer(ask.Config{
						ProviderName: opts.Provider, RagDir: cfg.RagDir, ContentDir: "content",
						OutputDir: cfg.OutputDir, DisableUI: true, LoopbackOnly: true,
						GraphExpansion: opts.GraphExpansion,
					})
					if err != nil {
						return err
					}
				}
				answer, err := server.Answer(ctx, ask.AnswerRequest{Question: question.Text, MaxChunks: k})
				if err != nil {
					return fmt.Errorf("ask eval: no-answer check %s/%s: %w", site.ID, question.ID, err)
				}
				result.NoAnswer = answer.NoAnswer && answer.Status == "no_answer" &&
					answer.Diagnostics.ChunksRetrieved == 0 && answer.NoAnswerMessage == noAnswerFallbackMessage
				result.NoAnswerMessage = answer.NoAnswerMessage
			}
			result.Passed = len(scored) == 0 && result.NoAnswer
		} else {
			result.RecallAtK = recallAtK(pages, question.AcceptablePages)
			result.PrecisionAtK = precisionAtK(pages, question.AcceptablePages)
			floor := precisionFloor
			if question.MinimumPrecisionAtK != nil {
				floor = *question.MinimumPrecisionAtK
			}
			result.Passed = result.RecallAtK+1e-9 >= question.MinimumRecallAtK && result.PrecisionAtK+1e-9 >= floor
			for _, page := range question.AcceptablePages {
				if !slices.Contains(pages, page) {
					result.MissingPages = append(result.MissingPages, page)
				}
			}
			if len(question.PathPages) > 0 {
				coverage := recallAtK(pages, question.PathPages)
				result.PathCoverage = &coverage
			}
			if opts.GraphExpansion && question.RequireGroundedPath {
				if server == nil {
					server, err = ask.NewServer(ask.Config{
						ProviderName: opts.Provider, RagDir: cfg.RagDir, ContentDir: "content",
						OutputDir: cfg.OutputDir, DisableUI: true, LoopbackOnly: true,
						GraphExpansion: true,
					})
					if err != nil {
						return err
					}
				}
				answer, err := server.Answer(ctx, ask.AnswerRequest{Question: question.Text, MaxChunks: k})
				if err != nil {
					return fmt.Errorf("ask eval: grounded path %s/%s: %w", site.ID, question.ID, err)
				}
				result.Answer, result.Paths = answer.Answer, answer.Paths
				result.GroundedPath = groundedPathMatches(answer, question.PathPages)
			}
			if question.RequireGroundedPath {
				result.Passed = result.Passed && result.GroundedPath
			}
			class := firstClass(question.Class)
			metrics := report.Classes[class]
			metrics.RecallAtK += result.RecallAtK
			metrics.PrecisionAtK += result.PrecisionAtK
			metrics.Questions++
			report.Classes[class] = metrics
			report.RecallAtK += result.RecallAtK
			report.PrecisionAtK += result.PrecisionAtK
			report.AnswerableCount++
		}
		report.Questions = append(report.Questions, result)
		if result.Passed {
			report.PassedCount++
		} else {
			report.FailedCount++
		}
	}
	report.Measurements = append(report.Measurements, measurement)
	return nil
}

func readDataset(path string) (Dataset, error) {
	f, err := os.Open(path)
	if err != nil {
		return Dataset{}, fmt.Errorf("ask eval: read dataset: %w", err)
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, (4<<20)+1))
	if err != nil {
		return Dataset{}, err
	}
	if len(raw) > 4<<20 {
		return Dataset{}, errors.New("ask eval: dataset exceeds 4 MiB")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var data Dataset
	if err := decoder.Decode(&data); err != nil {
		return Dataset{}, fmt.Errorf("ask eval: parse dataset: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return Dataset{}, errors.New("ask eval: expected one JSON document")
	}
	// Keep the exported float64 field compatible with #587, but distinguish
	// explicit zero from an omitted/null JSON floor for measurement probes.
	var presence struct {
		Sites []struct {
			Questions []struct {
				MinimumRecallAtK *float64 `json:"minimum_recall_at_k"`
			} `json:"questions"`
		} `json:"sites"`
	}
	if err := json.Unmarshal(raw, &presence); err != nil {
		return Dataset{}, err
	}
	for i, site := range data.Sites {
		for j, q := range site.Questions {
			if !q.Unanswerable && presence.Sites[i].Questions[j].MinimumRecallAtK == nil {
				return Dataset{}, fmt.Errorf("ask eval: question %s/%s requires an explicit non-null recall threshold", site.ID, q.ID)
			}
		}
	}
	return data, nil
}

func fraction(f float64) bool {
	return !math.IsNaN(f) && !math.IsInf(f, 0) && f >= 0 && f <= 1
}

// Validate checks schema before reading fixtures. Run also validates evidence
// and graph-hop labels against the real generated corpus and graph.
func (d Dataset) Validate() error {
	if d.Version != 1 {
		return fmt.Errorf("ask eval: unsupported dataset version %d", d.Version)
	}
	if strings.TrimSpace(d.Name) == "" {
		return errors.New("ask eval: dataset name is required")
	}
	if d.K < 1 || d.K > 100 {
		return errors.New("ask eval: k must be between 1 and 100")
	}
	if !fraction(d.MinimumPrecisionAtK) || (d.BaselineRecallAt5 != nil && !fraction(*d.BaselineRecallAt5)) {
		return errors.New("ask eval: dataset metrics must be between 0 and 1")
	}
	if d.BaselineRecallAt5 != nil && d.K != 5 {
		return errors.New("ask eval: baseline_recall_at_5 requires k=5")
	}
	if len(d.Sites) == 0 {
		return errors.New("ask eval: dataset must contain at least one site")
	}
	siteIDs := make(map[string]bool)
	answerableCount := 0
	for _, site := range d.Sites {
		if strings.TrimSpace(site.ID) == "" || siteIDs[site.ID] {
			return fmt.Errorf("ask eval: invalid or duplicate site id %q", site.ID)
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
		for _, q := range site.Questions {
			if strings.TrimSpace(q.ID) == "" || questionIDs[q.ID] || strings.TrimSpace(q.Text) == "" {
				return fmt.Errorf("ask eval: site %q requires unique question ids and nonempty text", site.ID)
			}
			questionIDs[q.ID] = true
			if !fraction(q.MinimumRecallAtK) || (q.MinimumPrecisionAtK != nil && !fraction(*q.MinimumPrecisionAtK)) {
				return fmt.Errorf("ask eval: invalid metric in %s/%s", site.ID, q.ID)
			}
			if q.Unanswerable {
				if len(q.AcceptablePages) != 0 || len(q.PathPages) != 0 || q.MinimumRecallAtK != 0 || q.MeasurementOnly || q.RequireGroundedPath {
					return fmt.Errorf("ask eval: unanswerable question %s/%s cannot list evidence or measurement-only labels", site.ID, q.ID)
				}
				continue
			}
			answerableCount++
			if len(q.AcceptablePages) == 0 {
				return fmt.Errorf("ask eval: answerable question %s/%s needs acceptable_pages", site.ID, q.ID)
			}
			if q.MinimumRecallAtK == 0 && !q.MeasurementOnly {
				return fmt.Errorf("ask eval: minimum_recall_at_k for %s/%s must be positive unless measurement_only is set", site.ID, q.ID)
			}
			for _, pages := range [][]string{q.AcceptablePages, q.PathPages} {
				seen := make(map[string]bool)
				for _, page := range pages {
					if strings.TrimSpace(page) == "" || !isLocalPageID(page) || seen[page] {
						return fmt.Errorf("ask eval: invalid or duplicate page %q in %s/%s", page, site.ID, q.ID)
					}
					seen[page] = true
				}
			}
			if len(q.PathPages) == 1 {
				return fmt.Errorf("ask eval: path in %s/%s needs at least two pages", site.ID, q.ID)
			}
			if q.RequireGroundedPath && len(q.PathPages) < 3 {
				return fmt.Errorf("ask eval: grounded path in %s/%s needs at least three pages", site.ID, q.ID)
			}
		}
	}
	if answerableCount == 0 {
		return errors.New("ask eval: dataset must contain at least one answerable question")
	}
	return nil
}

func isLocalPageID(page string) bool {
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(page)))
	return clean == page && filepath.IsLocal(page) && clean != "." && filepath.Ext(page) != ".md" && !strings.Contains(page, "\\")
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
	hits := 0
	for _, page := range acceptable {
		if slices.Contains(retrieved, page) {
			hits++
		}
	}
	return float64(hits) / float64(len(acceptable))
}

func precisionAtK(retrieved, acceptable []string) float64 {
	if len(retrieved) == 0 {
		return 0
	}
	hits := 0
	for _, page := range retrieved {
		if slices.Contains(acceptable, page) {
			hits++
		}
	}
	return float64(hits) / float64(len(retrieved))
}

func firstClass(class string) string {
	if class == "" {
		return "standard"
	}
	return class
}

// WriteReport retains the original table/recall summary and adds stable
// diagnostics. The caller decides exit status after the report is written.
func WriteReport(w io.Writer, report Report) error {
	if w == nil {
		return errors.New("ask eval: output writer is required")
	}
	var b strings.Builder
	ranker := report.Ranker
	if ranker == "" {
		ranker = "BM25-lite"
	}
	fmt.Fprintf(&b, "Ask This Site evaluation: %s\nRanker: %s · K=%d · provider=%s\n\n", report.DatasetName, ranker, report.K, report.Provider)
	table := tabwriter.NewWriter(&b, 0, 4, 2, ' ', 0)
	fmt.Fprintln(table, "SITE\tQUESTION\tRECALL@K\tMINIMUM\tPRECISION\tPATH\tSTATUS\tRETRIEVED / MISSING PAGES")
	for _, result := range report.Questions {
		recall, minimum := fmt.Sprintf("%.2f", result.RecallAtK), fmt.Sprintf("%.2f", result.MinimumRecallAtK)
		precision, path := fmt.Sprintf("%.2f", result.PrecisionAtK), "-"
		if result.PathCoverage != nil {
			path = fmt.Sprintf("%.2f", *result.PathCoverage)
		}
		if result.Unanswerable {
			recall, minimum, precision = "n/a", "no-answer", "-"
		}
		status := "FAIL"
		if result.Passed {
			status = "PASS"
		}
		fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%v / %v\n", result.SiteID, result.QuestionID, recall, minimum, precision, path, status, result.RetrievedPages, result.MissingPages)
	}
	if err := table.Flush(); err != nil {
		return err
	}
	fmt.Fprintf(&b, "\nRecall@%d: %.4f (%d answerable questions); questions passed: %d/%d\nPrecision@%d: %.4f (unique pages within top %d chunks)\n", report.K, report.RecallAtK, report.AnswerableCount, report.PassedCount, report.PassedCount+report.FailedCount, report.K, report.PrecisionAtK, report.K)
	if report.BaselineRecallAt5 != nil {
		fmt.Fprintf(&b, "recorded BM25-lite recall@5 baseline: %.4f", *report.BaselineRecallAt5)
		if report.K == 5 {
			fmt.Fprintf(&b, "; delta: %+.4f; floor passed: %t\n", report.RecallAtK-*report.BaselineRecallAt5, report.BaselineOK)
		} else {
			fmt.Fprintln(&b, "; comparison skipped: K is not 5")
		}
	}
	classes := make([]string, 0, len(report.Classes))
	for class := range report.Classes {
		classes = append(classes, class)
	}
	sort.Strings(classes)
	for _, class := range classes {
		m := report.Classes[class]
		fmt.Fprintf(&b, "Class %s: recall=%.4f precision=%.4f n=%d\n", class, m.RecallAtK, m.PrecisionAtK, m.Questions)
	}
	fmt.Fprintf(&b, "Precision floor passed: %t; evaluation passed: %t\n", report.PrecisionOK, report.Passed)
	for _, result := range report.Questions {
		if len(result.Paths) > 0 {
			fmt.Fprintf(&b, "Grounded route %s/%s: %s; cited/named all pages: %t\n",
				result.SiteID, result.QuestionID, pathLabel(result.Paths[0]), result.GroundedPath)
			fmt.Fprintf(&b, "  Answer: %s\n", result.Answer)
		}
	}
	if report.Comparison != nil {
		writeGraphComparison(&b, report.Comparison)
	}
	if len(report.Measurements) > 0 {
		measurements, err := json.Marshal(report.Measurements)
		if err != nil {
			return err
		}
		fmt.Fprintf(&b, "\nMeasurement JSON (durations in ns; excludes completion): %s\n", measurements)
	}
	_, err := io.WriteString(w, b.String())
	return err
}
