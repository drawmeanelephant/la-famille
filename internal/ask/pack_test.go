package ask

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/generator"
	"github.com/tbuddy/la-famille/internal/pack"
	"github.com/tbuddy/la-famille/internal/ragexport"
	"github.com/tbuddy/la-famille/internal/retrieval"
)

func packAskFixture(t *testing.T) (config.Config, string) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "site")
	if err := os.CopyFS(root, os.DirFS("../../assets/testdata/pack-ask")); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(filepath.Join(root, "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	cfg, err = cfg.ResolvePaths(root)
	if err != nil {
		t.Fatal(err)
	}
	name := filepath.Join(t.TempDir(), "corpus.tar")
	buildAskFixturePack(t, cfg, name)
	return cfg, name
}

func buildAskFixturePack(t *testing.T, cfg config.Config, name string) {
	t.Helper()
	if _, err := generator.Build(cfg); err != nil {
		t.Fatal(err)
	}
	if err := ragexport.RunExport(cfg); err != nil {
		t.Fatal(err)
	}
	if _, err := pack.Build(pack.BuildOptions{
		OutputDir: cfg.OutputDir, RagDir: cfg.RagDir,
		Site:       pack.Site{Name: cfg.SiteName, URL: cfg.SiteURL},
		Provenance: pack.Provenance{Generator: "la-famille", Version: "dev"},
	}, name); err != nil {
		t.Fatal(err)
	}
}

func TestPackAskFixtureCorpusAndTwoNoteCitations(t *testing.T) {
	cfg, name := packAskFixture(t)
	if err := os.RemoveAll(cfg.ProjectRoot); err != nil {
		t.Fatal(err)
	}
	srv, err := NewServer(Config{PackFile: name, ProviderName: "fake", LoopbackOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if srv.corpus.DocumentCount != 2 || srv.corpus.ChunkCount != 2 {
		t.Fatalf("unexpected fixture corpus: %+v", srv.corpus)
	}
	birds, ok := srv.corpus.ChunkByID("birds#h0")
	if !ok || birds.SourcePath != "notes/research/birds.md" || !strings.Contains(birds.Text, "seven days") {
		t.Fatalf("fixture bird corpus = %+v", birds)
	}
	maps, ok := srv.corpus.ChunkByID("maps#h0")
	if !ok || maps.SourcePath != "notes/research/maps.md" || !strings.Contains(maps.Text, "elevation lines") {
		t.Fatalf("fixture map corpus = %+v", maps)
	}
	if srv.cfg.RagDir != "" || srv.cfg.OutputDir != "" || srv.cfg.ContentDir != "" {
		t.Fatalf("pack mode inherited local directories: %+v", srv.cfg)
	}
	srv.provider = allCitationsFaker{}
	answer, err := srv.Answer(context.Background(), AnswerRequest{
		Question: "What do the meadow bird counts and local contour maps support?",
	})
	if err != nil || answer.Status != "answered" {
		t.Fatalf("answer = %+v, %v", answer, err)
	}
	var got []string
	for _, source := range answer.Sources {
		got = append(got, source.Title+"|"+source.URL)
	}
	sort.Strings(got)
	want := "Bird Observation Notes|/field-guide/bird-observations/\nContour Map Notes|/field-guide/contour-maps/"
	if strings.Join(got, "\n") != want {
		t.Fatalf("source cards = %v, want %s", got, want)
	}
	if status := srv.Snapshot(context.Background()); status.SourceDir != name || !status.Ready {
		t.Fatalf("status = %+v", status)
	}
}

func TestPackAskSeesAppliedFactWithoutSourceDirectories(t *testing.T) {
	cfg, before := packAskFixture(t)
	note := filepath.Join(cfg.ContentDir, "birds.md")
	body, err := os.ReadFile(note)
	if err != nil {
		t.Fatal(err)
	}
	body = []byte(strings.ReplaceAll(string(body), "seven days", "three days"))
	if err := os.WriteFile(note, body, 0600); err != nil {
		t.Fatal(err)
	}
	after, delta, applied := filepath.Join(t.TempDir(), "after.tar"), filepath.Join(t.TempDir(), "delta.tar"), filepath.Join(t.TempDir(), "applied.tar")
	buildAskFixturePack(t, cfg, after)
	if _, err := pack.Diff(before, after, delta); err != nil {
		t.Fatal(err)
	}
	if _, err := pack.Apply(before, delta, applied); err != nil {
		t.Fatal(err)
	}
	// Only the applied pack remains. Diff/apply themselves are already covered
	// by #618; this test proves Ask consumes the updated retrieval evidence.
	for _, name := range []string{cfg.ProjectRoot, before, after, delta} {
		if err := os.RemoveAll(name); err != nil {
			t.Fatal(err)
		}
	}
	srv, err := NewServer(Config{PackFile: applied, ProviderName: "fake", LoopbackOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	answer, err := srv.Answer(context.Background(), AnswerRequest{Question: "What is the migration survey interval?"})
	if err != nil || answer.Status != "answered" || len(answer.Sources) != 1 {
		t.Fatalf("answer = %+v, %v", answer, err)
	}
	source := answer.Sources[0]
	if !strings.Contains(source.Excerpt, "three days") || strings.Contains(source.Excerpt, "seven days") ||
		source.Title != "Bird Observation Notes" || source.URL != "/field-guide/bird-observations/" {
		t.Fatalf("stale or incorrect evidence: %+v", source)
	}
}

func TestPackAskDoesNotServeUnrelatedPublicFiles(t *testing.T) {
	_, name := packAskFixture(t)
	cwd := t.TempDir()
	if err := os.Mkdir(filepath.Join(cwd, "public"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cwd, "public", "unrelated.txt"), []byte("private-local-sentinel"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(cwd)
	srv, err := NewServer(Config{PackFile: name, ProviderName: "fake", LoopbackOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	srv.handleIndex(w, httptest.NewRequest(http.MethodGet, "/unrelated.txt", nil))
	if strings.Contains(w.Body.String(), "private-local-sentinel") {
		t.Fatal("pack Ask served unrelated local file")
	}
}

func TestPackAskRejectsDirectoryInputs(t *testing.T) {
	for _, cfg := range []Config{
		{PackFile: "relative.tar"},
		{PackFile: "/pack.tar", RagDir: "rag"},
		{PackFile: "/pack.tar", OutputDir: "public"},
		{PackFile: "/pack.tar", ContentDir: "content"},
		{PackFile: "/pack.tar", Rebuild: true},
	} {
		if _, err := NewServer(cfg); err == nil || !strings.Contains(err.Error(), "--pack") {
			t.Fatalf("config %+v error = %v", cfg, err)
		}
	}
}

func TestPackAskEmbeddingCacheUsesCorpusNotLocalBuild(t *testing.T) {
	_, name := packAskFixture(t)
	cache := t.TempDir()
	if err := os.WriteFile(filepath.Join(cache, ".la-famille-cache.json"),
		[]byte(`{"version":3,"fingerprint":"unrelated-local-build"}`), 0600); err != nil {
		t.Fatal(err)
	}
	srv, err := NewServer(Config{
		PackFile: name, ProviderName: "fake", LoopbackOnly: true,
		Embeddings: true, CacheDir: cache, embedder: &askEmbedder{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := srv.ranker.(*retrieval.HybridRanker); !ok {
		t.Fatalf("pack embeddings not wired: %T", srv.ranker)
	}
	data, err := os.ReadFile(filepath.Join(cache, retrieval.VectorFileName))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "unrelated-local-build") || !strings.Contains(string(data), retrieval.CorpusDigest(srv.corpus)) {
		t.Fatalf("pack embeddings used unrelated build fingerprint: %s", data)
	}
}
