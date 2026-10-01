package ask

import (
	"context"
	"testing"

	"github.com/tbuddy/la-famille/internal/llm"
)

type noCompletionProvider struct{ t *testing.T }

func (p noCompletionProvider) Name() string                    { return "no-completion-test" }
func (p noCompletionProvider) Available(context.Context) error { return nil }
func (p noCompletionProvider) Complete(context.Context, llm.Request) (llm.Response, error) {
	p.t.Fatal("completion must not run for insufficient lexical evidence")
	return llm.Response{}, nil
}

func TestServerNearMissAbstainsBeforeCompletionAcrossGraphToggles(t *testing.T) {
	s := graphServer(t)
	s.provider = noCompletionProvider{t}
	for _, enabled := range []bool{false, true, false} {
		for _, query := range []string{
			"salinity sensor warranty duration",
			"chloride assay insurance deductible",
		} {
			answer, err := s.Answer(context.Background(), AnswerRequest{
				Question: query, GraphExpansion: &enabled,
			})
			if err != nil {
				t.Fatal(err)
			}
			if answer.Status != "no_answer" || !answer.NoAnswer ||
				answer.NoAnswerMessage != "This site does not provide enough information to answer that question." ||
				answer.Diagnostics.ChunksRetrieved != 0 || len(answer.Sources) != 0 ||
				len(answer.Paths) != 0 || answer.Answer != "" {
				t.Fatalf("near-miss %q (graph=%t) leaked answer context: %+v", query, enabled, answer)
			}
			if answer.Diagnostics.GraphExpansion != enabled {
				t.Fatal("request's graph toggle was lost")
			}
		}
	}
}
