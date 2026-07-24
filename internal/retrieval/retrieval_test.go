package retrieval

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeBundle writes a minimal rag-content.md bundle with one file.
func writeBundle(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	bundle := "<file path=\"" + name + "\">\n<content>\n" + content + "\n</content>\n</file>\n"
	if err := os.WriteFile(filepath.Join(dir, "rag-content.md"), []byte(bundle), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "rag-system.md"), []byte("# System\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestLoad_ReadsXMLBundles(t *testing.T) {
	dir := t.TempDir()
	writeBundle(t, dir, "content/docs/foo.md", "---\ntitle: Foo Page\n---\n# Foo Page\n\n## Intro\n\nWelcome to Foo.\n\n## Setup\n\nRun `make` to build.\n")

	res, err := Load(LoadOptions{RagDir: dir})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if res.MalformedArtifact != "" {
		t.Errorf("unexpected malformation: %s", res.MalformedArtifact)
	}
	if got := res.Corpus.DocumentCount; got < 1 {
		t.Errorf("DocumentCount=%d", got)
	}
	if got := res.Corpus.ChunkCount; got < 1 {
		t.Errorf("ChunkCount=%d", got)
	}
	if res.Corpus.Version != "v1" {
		t.Errorf("Version=%q", res.Corpus.Version)
	}
}

func TestLoad_MalformedArtifact(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Open <file> but never close it.
	bad := "<file path=\"x.md\">\n<content>\nnever closes"
	if err := os.WriteFile(filepath.Join(dir, "rag-content.md"), []byte(bad), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(LoadOptions{RagDir: dir}); err == nil {
		t.Fatalf("expected error on malformed bundle")
	}
}

func TestLoad_RequiresDirectory(t *testing.T) {
	if _, err := Load(LoadOptions{RagDir: ""}); err == nil {
		t.Fatalf("Load with empty RagDir should fail")
	}
}

func TestLoad_MissingArtifactsReportedNotFatal(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Only rag-content exists — the others are "missing".
	if err := os.WriteFile(filepath.Join(dir, "rag-content.md"), []byte("<file path=\"x.md\">\n<content>\n# Hi\n</content>\n</file>\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := Load(LoadOptions{RagDir: dir})
	if err != nil {
		t.Fatalf("missing+present bundles should still load: %v", err)
	}
	found := map[string]bool{}
	for _, m := range res.MissingArtifacts {
		found[m] = true
	}
	if !found["rag-system.md"] || !found["rag-config.md"] {
		t.Errorf("expected both system/config in MissingArtifacts: %v", res.MissingArtifacts)
	}
}

func TestChunkFileSplitsAtHeadings(t *testing.T) {
	body := "---\ntitle: T\n---\n# Top\n\nPre-body.\n\n## Section A\n\nA prose here.\n\n## Section B\n\nB prose here.\n\n### Sub\n\nSub prose here.\n"
	chunks := chunkFile(body, "content/docs/x.md")
	if len(chunks) < 3 {
		t.Fatalf("expected at least 3 chunks, got %d", len(chunks))
	}
	if chunks[0].Title != "T" {
		t.Errorf("first chunk title = %q, want T", chunks[0].Title)
	}
	stableIDs := map[string]bool{}
	for _, c := range chunks {
		if stableIDs[c.ID] {
			t.Errorf("duplicate chunk ID %q", c.ID)
		}
		stableIDs[c.ID] = true
	}
	if chunks[0].HeadingText != "" {
		t.Errorf("pre-body chunk should have empty HeadingText, got %q", chunks[0].HeadingText)
	}
	wants := []string{"section-a", "section-b", "sub"}
	for _, w := range wants {
		found := false
		for _, c := range chunks {
			if strings.Contains(c.ID, w) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected chunk ID containing %q in %+v", w, chunks)
		}
	}
}

func TestChunkFileSingleChunkWhenNoHeadings(t *testing.T) {
	body := "# No headings here\n\nJust plain prose.\n"
	chunks := chunkFile(body, "content/docs/y.md")
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].HeadingText != "" {
		t.Errorf("HeadingText should be empty, got %q", chunks[0].HeadingText)
	}
}

func TestRankerKeysAreScoped(t *testing.T) {
	body := "# La Famille FAQ\n\n## Installation\n\nRun go install to add the binary.\n\n## Configuration\n\nAdd the config.yaml file.\n"
	dir := t.TempDir()
	writeBundle(t, dir, "content/docs/faq.md", body)
	res, err := Load(LoadOptions{RagDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	r := NewRanker(res.Corpus)
	if got := len(r.Rank("how to install?", 3)); got == 0 {
		t.Fatal("expected at least one result for 'how to install?'")
	}
	if got := len(r.Rank("absolute nonsense query xyzqqq", 3)); got != 0 {
		t.Errorf("expected zero hits for nonsense, got %d", got)
	}
	if got := r.Rank("", 3); got != nil {
		t.Errorf("empty query should yield nil, got %v", got)
	}
}

func TestRankerPreferentiallyScoresReleventSection(t *testing.T) {
	dir := t.TempDir()
	body := "---\ntitle: T\n---\n\n## Introduction\n\nWelcome to the project. This is a friendly hello.\n\n## Configuration\n\nTo configure the project, edit the configuration file. The configuration controls every aspect.\n"
	writeBundle(t, dir, "content/docs/x.md", body)
	res, err := Load(LoadOptions{RagDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	r := NewRanker(res.Corpus)
	scored := r.Rank("configuration file", 3)
	if len(scored) == 0 {
		t.Fatal("expected hits")
	}
	if !strings.Contains(scored[0].Chunk.Text, "configure the project") {
		t.Errorf("top chunk should be Configuration section, got %q", scored[0].Chunk.Text[:minLen(scored[0].Chunk.Text, 60)])
	}
}

func minLen(s string, n int) int {
	if len(s) < n {
		return len(s)
	}
	return n
}

func TestCitationsKeysStableAcrossNew(t *testing.T) {
	dir := t.TempDir()
	body := "---\ntitle: T\n---\n## One\n\nfoo.\n\n## Two\n\nbar.\n## Three\n\nbaz.\n"
	writeBundle(t, dir, "content/docs/x.md", body)
	res, err := Load(LoadOptions{RagDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	chunks := res.Corpus.SortedChunks()
	c := NewCitations(chunks)
	keys := map[string]string{}
	for _, ch := range chunks {
		keys[ch.ID] = c.KeyFor(ch.ID)
	}
	// Two constructions must produce the same mapping.
	c2 := NewCitations(chunks)
	for id, k := range keys {
		if c2.KeyFor(id) != k {
			t.Errorf("non-deterministic key for %s: %s vs %s", id, k, c2.KeyFor(id))
		}
	}
}

func TestCitationsVerifyRejectsUnknown(t *testing.T) {
	dir := t.TempDir()
	body := "## One\n\nfoo.\n## Two\n\nbar.\n"
	writeBundle(t, dir, "content/docs/x.md", body)
	res, err := Load(LoadOptions{RagDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	c := NewCitations(res.Corpus.Chunks)
	out := c.Verify("I claim [1] is correct and [99] is invented.")
	if len(out.VerifiedKeys) != 1 || out.VerifiedKeys[0] != "1" {
		t.Errorf("VerifiedKeys=%v want [1]", out.VerifiedKeys)
	}
	if len(out.DroppedKeys) != 1 || out.DroppedKeys[0] != "99" {
		t.Errorf("DroppedKeys=%v want [99]", out.DroppedKeys)
	}
}

func TestCitationsResolveSourceCardsPreservesOrder(t *testing.T) {
	dir := t.TempDir()
	body := "## One\n\nfoo prose one.\n## Two\n\nfoo prose two.\n## Three\n\nfoo prose three.\n"
	writeBundle(t, dir, "content/docs/x.md", body)
	res, err := Load(LoadOptions{RagDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	c := NewCitations(res.Corpus.Chunks)
	cards := c.ResolveSourceCards([]string{"3", "1", "2"})
	if len(cards) != 3 || cards[0].Key != "[3]" || cards[1].Key != "[1]" || cards[2].Key != "[2]" {
		t.Errorf("ResolveSourceCards did not preserve input order: %+v", cards)
	}
}

func TestBuildAnswerPromptIncludesAllBounds(t *testing.T) {
	dir := t.TempDir()
	body := "## One\n\nFirst section content.\n## Two\n\nSecond section content.\n## Three\n\nThird section content.\n"
	writeBundle(t, dir, "content/docs/x.md", body)
	res, err := Load(LoadOptions{RagDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	// Use synthesized Scored so we don't depend on the ranker here.
	scored := make([]Scored, 0, len(res.Corpus.Chunks))
	for _, ch := range res.Corpus.SortedChunks() {
		scored = append(scored, Scored{Chunk: ch, Score: 1})
	}
	c := NewCitations(scoredChunksOf(scored))
	budget := PromptBudget{MaxChunks: 2, MaxContextChars: 800}
	prompt, hints := BuildAnswerPrompt("any question", scored, c, budget)
	if !strings.Contains(prompt, "First section content") {
		t.Errorf("prompt missing first chunk text")
	}
	if !strings.Contains(prompt, "any question") {
		t.Errorf("prompt missing question")
	}
	if len(hints) > 2 {
		t.Errorf("hints=%d > MaxChunks=2", len(hints))
	}
}

func scoredChunksOf(s []Scored) []Chunk {
	out := make([]Chunk, len(s))
	for i, v := range s {
		out[i] = v.Chunk
	}
	return out
}

func TestPromptBudgetZeroFillsDefaults(t *testing.T) {
	prompt, hints := BuildAnswerPrompt("anything", nil, NewCitations(nil), PromptBudget{})
	if !strings.Contains(prompt, systemPrompt()) {
		t.Errorf("prompt should contain system instructions")
	}
	if hints != nil && len(hints) != 0 {
		t.Errorf("hints=%v want empty", hints)
	}
}

func TestEndToEndQueryWithFixture(t *testing.T) {
	dir := t.TempDir()
	body := "---\ntitle: La Famille FAQ\n---\n# FAQ\n\n## How do I install La Famille?\n\nRun `go install`.\n\n## How do I configure the project?\n\nEdit your config.yaml file. La Famille reads it at build time.\n\n## Where is the RAG archive?\n\nIt lives in the rag-archive directory.\n"
	writeBundle(t, dir, "content/docs/faq.md", body)
	res, err := Load(LoadOptions{RagDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	r := NewRanker(res.Corpus)
	scored := r.Rank("where is the rag archive located", 3)
	if len(scored) == 0 {
		t.Fatalf("expected hits")
	}
	if !strings.Contains(scored[0].Chunk.Text, "rag-archive") {
		t.Errorf("top chunk should mention rag-archive, got %q", scored[0].Chunk.Text)
	}
}
