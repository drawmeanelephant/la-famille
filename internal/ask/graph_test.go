package ask

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/llm"
	"github.com/tbuddy/la-famille/internal/retrieval"
)

func graphServer(t *testing.T) *Server {
	t.Helper()
	dir := t.TempDir()
	bundle := ""
	for _, page := range []struct{ id, title, text string }{
		{"sensor", "Field Sensor", "Salinity anomaly recorded at dawn."},
		{"registry", "Dispatch Register", "Courier Mara delivered the sealed amber vial to bench four."},
		{"assay", "Bench Assay", "Chloride concentration was elevated."},
		{"decoy", "Sensor Manual", "Field sensor bench assay reference instructions."},
	} {
		bundle += "<file path=\"content/" + page.id + ".md\">\n<content>\n---\ntitle: " +
			page.title + "\n---\n" + page.text + "\n</content>\n</file>\n"
	}
	files := map[string]string{
		"rag-content.md": bundle,
		"graph.json":     `{"nodes":{"sensor":{"type":"page","render":true},"registry":{"type":"page","render":true},"assay":{"type":"page","render":true},"decoy":{"type":"page","render":true}},"edges":[["sensor","registry"],["registry","assay"]]}`,
		"backlinks.json": `{"registry":["sensor"],"assay":["registry"]}`,
		"meta.json":      `{"sensor":{"url":"/project/field/"},"registry":{"url":"/project/dispatch/"},"assay":{"url":"/project/lab/"}}`,
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	server, err := NewServer(Config{ProviderName: "fake", RagDir: dir, OutputDir: dir, LoopbackOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	return server
}

func TestGraphToggleHTTPInSameServer(t *testing.T) {
	s := graphServer(t)
	mux := http.NewServeMux()
	s.registerRoutes(mux)
	call := func(enabled bool) AnswerResponse {
		t.Helper()
		raw, err := json.Marshal(AnswerRequest{Question: "Field Sensor Bench Assay connection", MaxChunks: 3, GraphExpansion: &enabled})
		if err != nil {
			t.Fatal(err)
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/ask", bytes.NewReader(raw)))
		if w.Code != http.StatusOK {
			t.Fatalf("ask failed: %d %s", w.Code, w.Body)
		}
		var answer AnswerResponse
		if err := json.Unmarshal(w.Body.Bytes(), &answer); err != nil {
			t.Fatal(err)
		}
		return answer
	}
	before := call(false)
	on := call(true)
	after := call(false)
	if len(on.Paths) != 1 || len(on.Sources) != 3 || on.NoAnswer ||
		!on.Diagnostics.GraphExpansion || on.Diagnostics.RetrievalMode != "lexical+graph" {
		t.Fatalf("graph not grounded: %+v", on)
	}
	for _, node := range on.Paths[0].Nodes {
		if !strings.Contains(on.Answer, node.Title) || !strings.HasPrefix(node.URL, "/project/") ||
			!slices.ContainsFunc(on.Sources, func(src retrieval.SourceCard) bool {
				return src.ChunkID == node.ChunkID && src.Key == "["+node.Key+"]"
			}) {
			t.Fatalf("uncited node or lost slug/base path: %+v", node)
		}
	}
	if len(before.Paths) != 0 || len(after.Paths) != 0 || before.Answer != after.Answer ||
		before.Diagnostics.GraphExpansion || after.Diagnostics.RetrievalMode != "lexical" {
		t.Fatalf("off/on/off did not restore baseline: %+v %+v", before, after)
	}
	status := s.Snapshot(context.Background())
	if status.GraphExpansion || status.GraphEdges != 2 {
		t.Fatalf("request mutated startup defaults: %+v", status)
	}
}

func TestGraphModelMustCiteEveryPromptedRoutePage(t *testing.T) {
	s := graphServer(t)
	s.provider = citeOnlyFaker{}
	on := true
	answer, err := s.Answer(context.Background(), AnswerRequest{
		Question: "Field Sensor Bench Assay connection", GraphExpansion: &on,
	})
	if err != nil || !answer.NoAnswer || len(answer.Paths) != 0 {
		t.Fatalf("partially cited model answer advertised full path: %+v %v", answer, err)
	}
	s.provider = &llm.FakeProvider{EchoMode: "miss"}
	answer, err = s.Answer(context.Background(), AnswerRequest{
		Question: "Field Sensor Bench Assay connection", GraphExpansion: &on,
	})
	if err != nil || !answer.NoAnswer || len(answer.Sources) != 0 || len(answer.Paths) != 0 {
		t.Fatalf("invented citation grounded a graph path: %+v %v", answer, err)
	}
}

type graphEvidenceProvider struct{ t *testing.T }

func (p graphEvidenceProvider) Name() string                    { return "evidence-test" }
func (p graphEvidenceProvider) Available(context.Context) error { return nil }
func (p graphEvidenceProvider) Complete(_ context.Context, req llm.Request) (llm.Response, error) {
	if !strings.Contains(req.Context, "Courier Mara") || len(req.GroundingPaths) != 1 {
		p.t.Fatal("bridge evidence was not supplied to the completion provider")
	}
	var labels []string
	for _, hint := range req.GroundingPaths[0] {
		labels = append(labels, hint.Title+" ["+hint.Key+"]")
	}
	return llm.Response{Answer: strings.Join(labels, " → ") + ". Courier Mara carried the sealed amber vial."}, nil
}

func TestGraphAnswerContainsEndpointAndBridgeEvidence(t *testing.T) {
	s := graphServer(t)
	s.cfg.GraphExpansion = true
	s.provider = graphEvidenceProvider{t}
	answer, err := s.Answer(context.Background(), AnswerRequest{Question: "Field Sensor Bench Assay connection"})
	if err != nil || answer.NoAnswer || len(answer.Sources) != 3 ||
		len(answer.Paths) != 1 || !strings.Contains(answer.Answer, "Courier Mara") {
		t.Fatalf("missing cited multi-hop answer: %+v %v", answer, err)
	}
}

type neverCalledScorer struct{ t *testing.T }

func (n neverCalledScorer) Rank(string, int) []retrieval.Scored {
	n.t.Fatal("graph comparison touched the non-lexical scorer")
	return nil
}

func TestRequestGraphComparisonBypassesEmbeddingScorer(t *testing.T) {
	s := graphServer(t)
	s.ranker = neverCalledScorer{t}
	for _, enabled := range []bool{false, true, false} {
		if _, err := s.Answer(context.Background(), AnswerRequest{
			Question: "Field Sensor Bench Assay connection", GraphExpansion: &enabled,
		}); err != nil {
			t.Fatal(err)
		}
	}
	if err := (&Config{Embeddings: true, GraphExpansion: true}).Validate(); err == nil {
		t.Fatal("startup entangled graph expansion with embeddings")
	}
}

func TestGraphNoAnswerAndTightBudgetsCannotInventPaths(t *testing.T) {
	s := graphServer(t)
	enabled := true
	for _, req := range []AnswerRequest{
		{Question: "quuxivorzz"},
		{Question: "Field Sensor Bench Assay connection", MaxContextChars: 1},
		{Question: "Field Sensor Bench Assay connection", MaxChunks: 2},
	} {
		req.GraphExpansion = &enabled
		answer, err := s.Answer(context.Background(), req)
		if err != nil || len(answer.Paths) != 0 {
			t.Fatalf("partial/invented path: %+v %v", answer, err)
		}
		if req.MaxContextChars == 1 && !answer.NoAnswer {
			t.Fatal("a route that cannot fit the prompt budget was answered anyway")
		}
	}
	s.cfg.MaxContext = 1
	answer, err := s.Answer(context.Background(), AnswerRequest{
		Question: "Field Sensor Bench Assay connection", GraphExpansion: &enabled,
	})
	if err != nil || !answer.NoAnswer {
		t.Fatalf("server context budget was ignored: %+v %v", answer, err)
	}
}
