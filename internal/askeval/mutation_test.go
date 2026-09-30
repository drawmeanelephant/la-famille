package askeval

import (
	"context"
	"slices"
	"testing"

	"github.com/tbuddy/la-famille/internal/retrieval"
)

// reversedRanker returns the production ranking in reverse order. It is a
// realistic mistake (wrong sort comparator, inverted score) that still
// returns plausible pages, so it probes ranking quality rather than plumbing.
type reversedRanker struct{ inner Scorer }

func (r reversedRanker) Rank(query string, topK int) []retrieval.Scored {
	got := r.inner.Rank(query, topK)
	out := make([]retrieval.Scored, 0, len(got))
	for i := len(got) - 1; i >= 0; i-- {
		out = append(out, got[i])
	}
	return out
}

// firstChunkRanker always returns the corpus's first chunk regardless of the
// query. It stands in for a stub or placeholder retriever that would otherwise
// let a green suite hide a completely broken ranking path.
type firstChunkRanker struct{ corpus Corpus }

func (r firstChunkRanker) Rank(string, int) []retrieval.Scored {
	if len(r.corpus.Chunks) == 0 {
		return nil
	}
	return []retrieval.Scored{{Chunk: r.corpus.Chunks[0]}}
}

// A regression gate that cannot fail is not a gate. These tests assert the
// harness rejects broken ranking on the checked-in datasets, so a future
// "improvement" cannot pass by accident or by making the ranker weaker.
//
// The frozen #587 dataset is saturated at 1.0 recall over 2-4 page fixtures,
// so reversed ranking alone may still land every required page inside K. These
// tests therefore assert on whichever signal the dataset can actually observe,
// and TestSaturatedRegressionSetCannotDetectOrdering records that limit rather
// than hiding it.
func TestGateRejectsBrokenRanker(t *testing.T) {
	for _, name := range []string{"golden-questions.json", "golden-questions-hard.json"} {
		t.Run(name, func(t *testing.T) {
			t.Run("stub ranker", func(t *testing.T) {
				r, err := Run(context.Background(), Options{
					DatasetPath: testDatasetPath(t, name),
					ProjectRoot: testProjectRoot(t),
					Ranker:      func(c Corpus) Scorer { return firstChunkRanker{corpus: c} },
				})
				if err != nil {
					t.Fatal(err)
				}
				if r.Passed {
					t.Fatalf("gate passed a stub ranker that ignores the query: recall=%.4f", r.RecallAtK)
				}
				if r.RecallAtK >= *r.BaselineRecallAt5 {
					t.Fatalf("stub ranker should fall below the recorded recall floor: %.4f vs %.4f", r.RecallAtK, *r.BaselineRecallAt5)
				}
				if r.BaselineOK {
					t.Fatal("stub ranker satisfied the recorded recall floor")
				}
			})
			t.Run("reversed ranker", func(t *testing.T) {
				r, err := Run(context.Background(), Options{
					DatasetPath: testDatasetPath(t, name),
					ProjectRoot: testProjectRoot(t),
					Ranker: func(c Corpus) Scorer {
						return reversedRanker{inner: retrieval.NewRanker(c)}
					},
				})
				if err != nil {
					t.Fatal(err)
				}
				// Reversal must at minimum change measured retrieval. If the
				// dataset is too small to observe it, that is a documented
				// saturation limit, not a silent pass.
				production, err := Run(context.Background(), Options{
					DatasetPath: testDatasetPath(t, name),
					ProjectRoot: testProjectRoot(t),
				})
				if err != nil {
					t.Fatal(err)
				}
				if !orderingDiffers(r, production) {
					t.Fatalf("reversed ranking was indistinguishable from production: %+v", r)
				}
				if r.Passed && r.RecallAtK > production.RecallAtK {
					t.Fatalf("reversed ranking improved recall %.4f > %.4f; the gate is not measuring the ranker", r.RecallAtK, production.RecallAtK)
				}
			})
		})
	}
}

func orderingDiffers(a, b Report) bool {
	if len(a.Questions) != len(b.Questions) {
		return true
	}
	for i := range a.Questions {
		if !slices.Equal(a.Questions[i].RetrievedPages, b.Questions[i].RetrievedPages) {
			return true
		}
	}
	return false
}

// TestSaturatedRegressionSetCannotDetectOrdering documents a real limitation:
// the frozen dataset is too small for reversed ranking to change recall, so
// only the stub-ranker case above is a hard gate there. Should a future
// retriever regress ordering on small corpora, this test is the canary that
// tells us to grow the dataset rather than to trust a green run.
func TestSaturatedRegressionSetCannotDetectOrdering(t *testing.T) {
	production, err := Run(context.Background(), Options{
		DatasetPath: testDatasetPath(t, "golden-questions.json"),
		ProjectRoot: testProjectRoot(t),
	})
	if err != nil {
		t.Fatal(err)
	}
	reversedRun, err := Run(context.Background(), Options{
		DatasetPath: testDatasetPath(t, "golden-questions.json"),
		ProjectRoot: testProjectRoot(t),
		Ranker: func(c Corpus) Scorer {
			return reversedRanker{inner: retrieval.NewRanker(c)}
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !orderingDiffers(production, reversedRun) {
		t.Skip("dataset no longer distinguishes ordering; grow the fixtures")
	}
	if reversedRun.Passed {
		t.Log("KNOWN LIMITATION: the frozen dataset detects a stub ranker but " +
			"not reversed ordering, because 2-4 page fixtures at K=5 retrieve " +
			"every candidate page. The hard dataset and the stub-ranker case " +
			"carry the regression signal for ordering.")
	}
}

// TestRankerSeamDefaultsToProduction guards against the injectable arm
// silently diverging from production when no override is supplied.
func TestRankerSeamDefaultsToProduction(t *testing.T) {
	opts := Options{DatasetPath: testDatasetPath(t, "golden-questions.json"), ProjectRoot: testProjectRoot(t)}
	withNil, err := Run(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}
	opts.Ranker = func(c Corpus) Scorer { return retrieval.NewRanker(c) }
	withExplicit, err := Run(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}
	if withNil.RecallAtK != withExplicit.RecallAtK || withNil.PrecisionAtK != withExplicit.PrecisionAtK {
		t.Fatalf("nil ranker (%v/%v) diverged from explicit production ranker (%v/%v)",
			withNil.RecallAtK, withNil.PrecisionAtK, withExplicit.RecallAtK, withExplicit.PrecisionAtK)
	}
}
