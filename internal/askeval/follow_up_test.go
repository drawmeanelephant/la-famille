package askeval

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/graph"
	"github.com/tbuddy/la-famille/internal/retrieval"
)

func followUpDataset() Dataset {
	return Dataset{Version: 1, Name: "unit", K: 5, Sites: []Site{{
		ID: "site", Fixture: "site", ContentDir: "content", Questions: []Question{{
			ID: "q", Text: "shared fact", AcceptablePages: []string{"index"}, MinimumRecallAtK: 1,
		}},
	}}}
}

func testProjectRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func testDatasetPath(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join(testProjectRoot(t), "assets", "testdata", "ask-eval", name)
}

func writeFollowUpDataset(t *testing.T, root string, d Dataset) string {
	t.Helper()
	data, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "dataset.json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestFollowUpValidation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*Dataset)
	}{
		{"large k", func(d *Dataset) { d.K = 101 }},
		{"nan precision", func(d *Dataset) { d.MinimumPrecisionAtK = math.NaN() }},
		{"infinite recall", func(d *Dataset) { d.Sites[0].Questions[0].MinimumRecallAtK = math.Inf(1) }},
		{"zero needs measurement flag", func(d *Dataset) { d.Sites[0].Questions[0].MinimumRecallAtK = 0 }},
		{"path duplicate", func(d *Dataset) { d.Sites[0].Questions[0].PathPages = []string{"index", "index"} }},
		{"path traversal", func(d *Dataset) { d.Sites[0].Questions[0].PathPages = []string{"index", "../secret"} }},
		{"short path", func(d *Dataset) { d.Sites[0].Questions[0].PathPages = []string{"index"} }},
		{"source extension", func(d *Dataset) { d.Sites[0].Questions[0].AcceptablePages = []string{"index.md"} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := followUpDataset()
			tc.mutate(&d)
			if err := d.Validate(); err == nil {
				t.Fatal("invalid dataset accepted")
			}
		})
	}
	d := followUpDataset()
	d.Sites[0].Questions[0].MinimumRecallAtK = 0
	d.Sites[0].Questions[0].MeasurementOnly = true
	if err := d.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestReadDatasetStrictJSON(t *testing.T) {
	d := followUpDataset()
	d.Sites[0].Questions[0].MeasurementOnly = true
	base, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	for _, body := range []string{"", "null", string(base) + " {}",
		strings.Replace(string(base), `,"minimum_recall_at_k":1`, "", 1),
		strings.Replace(string(base), `"minimum_recall_at_k":1`, `"minimum_recall_at_k":null`, 1),
		strings.Replace(string(base), `"version":1`, `"version":1,"typo":true`, 1),
		string(base) + strings.Repeat(" ", 4<<20)} {
		path := filepath.Join(t.TempDir(), "bad.json")
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
		_, err := Run(context.Background(), Options{DatasetPath: path})
		if err == nil {
			t.Fatal("invalid JSON accepted")
		}
	}
}

func TestPrecisionAndChunkDeduplication(t *testing.T) {
	if got := precisionAtK([]string{"a", "decoy"}, []string{"a", "b"}); got != 0.5 {
		t.Fatalf("precision = %v", got)
	}
	if precisionAtK(nil, []string{"a"}) != 0 {
		t.Fatal("empty retrieval precision must be zero")
	}
	hits := []retrieval.Scored{{Chunk: retrieval.Chunk{PageID: "a"}}, {Chunk: retrieval.Chunk{PageID: "a"}}, {Chunk: retrieval.Chunk{PageID: "b"}}}
	if got := uniquePages(hits[:2]); !slices.Equal(got, []string{"a"}) {
		t.Fatalf("deduplication must not backfill beyond chunk K: %v", got)
	}
}

func TestGoldenFollowUpBaselines(t *testing.T) {
	for _, name := range []string{"golden-questions.json", "golden-questions-hard.json"} {
		t.Run(name, func(t *testing.T) {
			r, err := Run(context.Background(), Options{DatasetPath: testDatasetPath(t, name), ProjectRoot: testProjectRoot(t)})
			if err != nil {
				t.Fatal(err)
			}
			var output bytes.Buffer
			if err := WriteReport(&output, r); err != nil {
				t.Fatal(err)
			}
			t.Log("\n" + output.String())
			if r.BaselineRecallAt5 == nil || math.Abs(r.RecallAtK-*r.BaselineRecallAt5) > 1e-9 {
				t.Fatalf("lexical baseline changed: %+v", r)
			}
			if name == "golden-questions.json" {
				if !r.Passed {
					t.Fatal("merged regression contract failed")
				}
				if math.Abs(r.PrecisionAtK-0.4375) > 1e-9 {
					t.Fatalf("merged regression precision changed: %v", r.PrecisionAtK)
				}
				return
			}
			if !r.Passed || r.PassedCount != 8 || r.FailedCount != 0 || !r.BaselineOK || !r.PrecisionOK || math.Abs(r.PrecisionAtK-1.0/3) > 1e-9 {
				t.Fatalf("hard gates changed: %+v", r)
			}
			for _, q := range r.Questions {
				if q.QuestionID == "sensor-warranty-absent" {
					if !q.Passed || len(q.RetrievedPages) != 0 || !q.NoAnswer || q.NoAnswerMessage != noAnswerFallbackMessage {
						t.Fatalf("near-miss must abstain with canonical no-answer: %+v", q)
					}
				} else if !q.Passed {
					t.Errorf("unexpected failure: %+v", q)
				}
				if q.Class == "paraphrase" && q.RecallAtK != 0 {
					t.Errorf("paraphrase leaked lexical evidence: %+v", q)
				}
				if q.QuestionID == "sensor-to-assay" && (q.PathCoverage == nil || *q.PathCoverage != 2.0/3 || slices.Contains(q.RetrievedPages, "registry")) {
					t.Errorf("graph bridge baseline changed: %+v", q)
				}
			}
		})
	}
}

func TestCopyContentSymlinks(t *testing.T) {
	for _, kind := range []string{"external file", "internal file", "directory"} {
		t.Run(kind, func(t *testing.T) {
			source, outside := t.TempDir(), t.TempDir()
			target := filepath.Join(outside, "external.md")
			if kind == "internal file" {
				target = filepath.Join(source, "original.md")
			}
			if err := os.WriteFile(target, []byte("# Not fixture evidence"), 0600); err != nil {
				t.Fatal(err)
			}
			if kind == "directory" {
				target = outside
			}
			if err := os.Symlink(target, filepath.Join(source, "link")); err != nil {
				t.Skipf("symlinks unavailable: %v", err)
			}
			root, err := os.OpenRoot(source)
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()
			if err := copyContent(root.FS(), filepath.Join(t.TempDir(), "copy")); err == nil || !strings.Contains(err.Error(), "symlink") {
				t.Fatalf("symlink accepted: %v", err)
			}
		})
	}
}

func TestFixtureRootCannotEscapeViaSymlink(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	content := filepath.Join(outside, "content")
	if err := os.MkdirAll(content, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(content, "index.md"), []byte("# Shared fact"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "site")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := prepareSite(root, t.TempDir(), followUpDataset().Sites[0]); err == nil {
		t.Fatal("external fixture root accepted")
	}
}

func TestCustomContentIsolation(t *testing.T) {
	for _, contentDir := range []string{"pages", "templates", "public", "rag-archive", "nested/pages"} {
		t.Run(contentDir, func(t *testing.T) {
			root := t.TempDir()
			content := filepath.Join(root, "site", contentDir)
			if err := os.MkdirAll(content, 0700); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(content, "index.md")
			original := []byte("# Fact\n\nShared fact.\n")
			if err := os.WriteFile(path, original, 0600); err != nil {
				t.Fatal(err)
			}
			d := followUpDataset()
			d.Sites[0].ContentDir = contentDir
			r, err := Run(context.Background(), Options{DatasetPath: writeFollowUpDataset(t, root, d), ProjectRoot: root})
			if err != nil {
				t.Fatal(err)
			}
			if !r.Passed || !slices.Equal(r.Questions[0].RetrievedPages, []string{"index"}) {
				t.Fatalf("content identity changed: %+v", r)
			}
			after, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(after, original) {
				t.Fatal("source fixture mutated")
			}
			if _, err := os.Stat(filepath.Join(root, ".la-famille-cache.json")); !os.IsNotExist(err) {
				t.Fatalf("source project cache was written: %v", err)
			}
		})
	}
}

func TestGeneratedGraphAndVocabulary(t *testing.T) {
	d, err := readDataset(testDatasetPath(t, "golden-questions-hard.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, site := range d.Sites {
		cfg, err := prepareSite(testProjectRoot(t), t.TempDir(), site)
		if err != nil {
			t.Fatal(err)
		}
		loaded, err := retrieval.Load(retrieval.LoadOptions{RagDir: cfg.RagDir, OutputDir: cfg.OutputDir, ContentDir: "content"})
		if err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(filepath.Join(cfg.OutputDir, "graph.json"))
		if err != nil {
			t.Fatal(err)
		}
		var g graph.Graph
		if err := json.Unmarshal(data, &g); err != nil {
			t.Fatal(err)
		}
		pages := make(map[string][]retrieval.Chunk)
		for _, ch := range loaded.Corpus.Chunks {
			pages[ch.PageID] = append(pages[ch.PageID], ch)
			if node, ok := g.Nodes[ch.PageID]; !ok || !node.Render {
				t.Errorf("chunk %s has no published graph identity", ch.ID)
			}
		}
		if len(pages) != 12 {
			t.Fatalf("site %s has %d pages", site.ID, len(pages))
		}
		for _, q := range site.Questions {
			if err := validateLabels(q, loaded.Corpus, g); err != nil {
				t.Fatal(err)
			}
			var disjointPages []string
			if q.Class == "paraphrase" {
				disjointPages = q.AcceptablePages
			} else if len(q.PathPages) > 2 {
				disjointPages = q.PathPages[1 : len(q.PathPages)-1]
			}
			for _, page := range disjointPages {
				ranker := retrieval.NewRanker(retrieval.Corpus{Chunks: pages[page]})
				if len(ranker.Rank(q.Text, 5)) != 0 {
					t.Errorf("query %s leaks into evidence/bridge %s", q.ID, page)
				}
			}
		}
		if site.ID == "estuary" {
			if ch, ok := loaded.Corpus.ChunkByID("registry#h2-routing"); !ok || ch.URL != "/registry/" {
				t.Errorf("citation metadata missing: %+v", ch)
			}
			if err := validateLabels(Question{ID: "shortcut", PathPages: []string{"sensor", "assay"}}, loaded.Corpus, g); err == nil {
				t.Fatal("invented graph hop accepted")
			}
		}
		if err := validateLabels(Question{ID: "missing", AcceptablePages: []string{"missing"}}, loaded.Corpus, g); err == nil {
			t.Fatal("stale evidence accepted")
		}
	}
}

func TestFollowUpMetricGates(t *testing.T) {
	root := t.TempDir()
	content := filepath.Join(root, "site", "content")
	if err := os.MkdirAll(content, 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"index", "decoy"} {
		if err := os.WriteFile(filepath.Join(content, name+".md"), []byte("# "+name+"\n\nShared fact."), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		name   string
		mutate func(*Dataset)
		k      int
		passed bool
	}{
		{"precision floor", func(d *Dataset) { d.MinimumPrecisionAtK = 0.75 }, 5, false},
		{"precision override", func(d *Dataset) { v := 0.75; d.Sites[0].Questions[0].MinimumPrecisionAtK = &v }, 5, false},
		{"recall floor", func(d *Dataset) { d.Sites[0].Questions[0].AcceptablePages = []string{"index", "decoy"} }, 1, false},
		{"baseline regression", func(d *Dataset) {
			v := 1.0
			d.BaselineRecallAt5 = &v
			d.Sites[0].Questions[0].Text = "missingtoken"
			d.Sites[0].Questions[0].MinimumRecallAtK = 0
			d.Sites[0].Questions[0].MeasurementOnly = true
		}, 5, false},
		{"lift allowed", func(d *Dataset) { v := 0.5; d.BaselineRecallAt5 = &v }, 5, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := followUpDataset()
			tc.mutate(&d)
			r, err := Run(context.Background(), Options{DatasetPath: writeFollowUpDataset(t, root, d), ProjectRoot: root, K: tc.k})
			if err != nil {
				t.Fatal(err)
			}
			if r.Passed != tc.passed {
				t.Fatalf("gate status: %+v", r)
			}
		})
	}
}

func TestFollowUpOptionsAndReport(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	opts := Options{DatasetPath: testDatasetPath(t, "golden-questions.json"), ProjectRoot: testProjectRoot(t)}
	if _, err := Run(ctx, opts); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation: %v", err)
	}
	for _, k := range []int{-1, 101} {
		opts.K = k
		if _, err := Run(context.Background(), opts); err == nil {
			t.Fatal("invalid K accepted")
		}
	}
	opts.K, opts.Provider = 0, "remote"
	if _, err := Run(context.Background(), opts); err == nil {
		t.Fatal("remote provider accepted")
	}
	baseline := 1.0
	r := Report{K: 8, BaselineRecallAt5: &baseline, Classes: map[string]Metrics{"z": {Questions: 1}, "a": {Questions: 2}}}
	var a, b bytes.Buffer
	if err := WriteReport(&a, r); err != nil {
		t.Fatal(err)
	}
	if err := WriteReport(&b, r); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a.Bytes(), b.Bytes()) || strings.Index(a.String(), "Class a") > strings.Index(a.String(), "Class z") || !strings.Contains(a.String(), "comparison skipped") {
		t.Fatal("unstable report or incorrect baseline comparison")
	}
	if err := WriteReport(followUpErrorWriter{}, r); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("writer error lost: %v", err)
	}
}

type followUpErrorWriter struct{}

func (followUpErrorWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }
