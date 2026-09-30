package askeval

import (
	"bytes"
	"context"
	"math"
	"slices"
	"strings"
	"testing"
)

func TestGraphGoldenComparisonGroundsEveryRoute(t *testing.T) {
	report, err := Run(context.Background(), Options{
		ProjectRoot: testProjectRoot(t), DatasetPath: testDatasetPath(t, "golden-questions-graph.json"),
		CompareGraph: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	c := report.Comparison
	if c == nil || !report.Passed || !c.Passed || c.SinglePageCount != 2 ||
		c.Graph.RecallAtK <= c.Lexical.RecallAtK || c.Graph.PrecisionAtK < c.Lexical.PrecisionAtK {
		t.Fatalf("graph did not improve without dilution: %+v %+v", report, c)
	}
	routes := 0
	for _, result := range report.Questions {
		if result.Class == "multi-hop" {
			routes++
			if !result.Passed || !result.GroundedPath || result.PathCoverage == nil ||
				*result.PathCoverage != 1 || len(result.Paths) != 1 {
				t.Fatalf("bridge retrieval was mistaken for answer grounding: %+v", result)
			}
		}
	}
	if routes != 3 {
		t.Fatalf("expected three grounded multi-hop questions, got %d", routes)
	}
	var out bytes.Buffer
	if err := WriteReport(&out, report); err != nil {
		t.Fatal(err)
	}
	for _, label := range []string{"graph off", "graph on", "Single-page non-dilution: true",
		"sensor → registry → assay", "cited/named all pages: true", "Synthetic graph-grounding"} {
		if !strings.Contains(out.String(), label) {
			t.Fatalf("report missing %q:\n%s", label, out.String())
		}
	}
	t.Log("\n" + out.String())
}

func TestGraphKeepsAllFrozenSinglePagePrecision(t *testing.T) {
	report, err := Run(context.Background(), Options{
		ProjectRoot: testProjectRoot(t), DatasetPath: testDatasetPath(t, "golden-questions.json"),
		CompareGraph: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	c := report.Comparison
	if !report.Passed || c == nil || !c.SinglePageOK || c.SinglePageCount != 8 ||
		c.Lexical.RecallAtK != 1 || math.Abs(c.Lexical.PrecisionAtK-0.4375) > 1e-9 {
		t.Fatalf("frozen single-page baseline changed: %+v %+v", report, c)
	}
	for i, before := range c.Lexical.Questions {
		after := c.Graph.Questions[i]
		if after.PrecisionAtK < before.PrecisionAtK || after.RecallAtK < before.RecallAtK {
			t.Fatalf("single-page precision/recall diluted: %+v -> %+v", before, after)
		}
	}
}

func TestGraphHardLabelsRemainHonest(t *testing.T) {
	report, err := Run(context.Background(), Options{
		ProjectRoot: testProjectRoot(t), DatasetPath: testDatasetPath(t, "golden-questions-hard.json"),
		CompareGraph: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	c := report.Comparison
	if c == nil || c.Lexical.RecallAtK != *c.Lexical.BaselineRecallAt5 || c.Lexical.FailedCount != 1 {
		t.Fatal("lexical baseline changed")
	}
	for _, q := range report.Questions {
		if q.QuestionID == "sensor-warranty-absent" && (q.Passed || q.NoAnswer || len(q.RetrievedPages) == 0) {
			t.Fatalf("graph hid known abstention failure: %+v", q)
		}
		if q.QuestionID == "sensor-to-assay" && q.PathCoverage != nil && *q.PathCoverage == 1 &&
			slices.Contains(q.RetrievedPages, "registry") && q.PrecisionAtK == 1 {
			t.Fatal("bridge was relabeled as acceptable evidence in the frozen hard set")
		}
	}
	if report.Passed {
		t.Fatal("hard dataset must retain its known failure")
	}
}

func TestGraphEvalCannotUseEmbeddingsOrCustomBaseline(t *testing.T) {
	for _, opts := range []Options{
		{GraphExpansion: true, Embeddings: true},
		{CompareGraph: true, Embeddings: true},
		{CompareGraph: true, Ranker: func(Corpus) Scorer { return nil }},
	} {
		if _, err := Run(context.Background(), opts); err == nil {
			t.Fatal("graph comparison allowed a non-lexical baseline")
		}
	}
	d := followUpDataset()
	d.Sites[0].Questions[0].RequireGroundedPath = true
	if err := d.Validate(); err == nil {
		t.Fatal("grounded-path gate accepted missing route labels")
	}
}
