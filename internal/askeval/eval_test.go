package askeval

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDatasetValidate(t *testing.T) {
	valid := Dataset{
		Version: 1,
		Name:    "unit",
		K:       5,
		Sites: []Site{{
			ID:         "site",
			Fixture:    "assets/site",
			ContentDir: "content",
			Questions: []Question{{
				ID:               "question",
				Text:             "Where is the answer?",
				AcceptablePages:  []string{"guide"},
				MinimumRecallAtK: 1,
			}},
		}},
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid dataset rejected: %v", err)
	}

	tests := []struct {
		name   string
		mutate func(*Dataset)
	}{
		{
			name:   "unsupported version",
			mutate: func(d *Dataset) { d.Version = 2 },
		},
		{
			name:   "k must be positive",
			mutate: func(d *Dataset) { d.K = 0 },
		},
		{
			name: "baseline requires k five",
			mutate: func(d *Dataset) {
				value := 0.5
				d.BaselineRecallAt5 = &value
				d.K = 3
			},
		},
		{
			name: "fixture must stay in project",
			mutate: func(d *Dataset) {
				d.Sites[0].Fixture = "../outside"
			},
		},
		{
			name: "answerable questions need a relevant page",
			mutate: func(d *Dataset) {
				d.Sites[0].Questions[0].AcceptablePages = nil
			},
		},
		{
			name: "page IDs cannot traverse directories",
			mutate: func(d *Dataset) {
				d.Sites[0].Questions[0].AcceptablePages = []string{"../private"}
			},
		},
		{
			name: "unanswerable questions cannot name pages",
			mutate: func(d *Dataset) {
				d.Sites[0].Questions[0].Unanswerable = true
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dataset := cloneDataset(valid)
			tt.mutate(&dataset)
			if err := dataset.Validate(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestRecallAtK(t *testing.T) {
	got := recallAtK([]string{"a", "b"}, []string{"b", "c"})
	if got != 0.5 {
		t.Fatalf("recallAtK = %v, want 0.5", got)
	}
	if got := recallAtK(nil, []string{"a"}); got != 0 {
		t.Fatalf("recallAtK with no retrieved pages = %v, want 0", got)
	}
}

func TestRunGoldenDataset(t *testing.T) {
	projectRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	report, err := Run(context.Background(), Options{
		DatasetPath: filepath.Join(projectRoot, "assets", "testdata", "ask-eval", "golden-questions.json"),
		ProjectRoot: projectRoot,
		Provider:    "fake",
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !report.Passed {
		t.Fatalf("golden dataset has %d failed questions: %+v", report.FailedCount, report.Questions)
	}
	if report.AnswerableCount != 8 {
		t.Errorf("answerable question count = %d, want 8", report.AnswerableCount)
	}
	if report.RecallAtK != 1 {
		t.Errorf("recall@%d = %.2f, want 1.00", report.K, report.RecallAtK)
	}
	if report.BaselineRecallAt5 == nil || *report.BaselineRecallAt5 != 1 {
		t.Errorf("recorded recall@5 baseline = %v, want 1.00", report.BaselineRecallAt5)
	}
	var noAnswerFound bool
	for _, result := range report.Questions {
		if result.Unanswerable {
			noAnswerFound = result.NoAnswer && result.NoAnswerMessage == "This site does not provide enough information to answer that question."
		}
	}
	if !noAnswerFound {
		t.Fatal("unanswerable golden question did not verify the no-answer fallback")
	}
}

func TestRunRequiresValidDataset(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dataset.json")
	if err := os.WriteFile(path, []byte(`{"version":1,"name":"broken","k":0,"sites":[]}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(context.Background(), Options{DatasetPath: path, ProjectRoot: t.TempDir()}); err == nil {
		t.Fatal("Run accepted an invalid dataset")
	}
}

func TestWriteReport(t *testing.T) {
	var output bytes.Buffer
	baseline := 1.0
	report := Report{
		DatasetName:       "sample",
		K:                 5,
		Provider:          "fake",
		RecallAtK:         1,
		AnswerableCount:   1,
		PassedCount:       1,
		BaselineRecallAt5: &baseline,
		Passed:            true,
		Questions: []QuestionResult{{
			SiteID:           "site",
			QuestionID:       "q1",
			RecallAtK:        1,
			MinimumRecallAtK: 1,
			RetrievedPages:   []string{"guide"},
			Passed:           true,
		}},
	}
	if err := WriteReport(&output, report); err != nil {
		t.Fatalf("WriteReport: %v", err)
	}
	for _, want := range []string{"q1", "PASS", "Recall@5: 1.00", "recorded BM25-lite recall@5 baseline: 1.00"} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("report missing %q:\n%s", want, output.String())
		}
	}
}

func cloneDataset(in Dataset) Dataset {
	out := in
	out.Sites = append([]Site(nil), in.Sites...)
	for i := range out.Sites {
		out.Sites[i].Questions = append([]Question(nil), in.Sites[i].Questions...)
		for j := range out.Sites[i].Questions {
			out.Sites[i].Questions[j].AcceptablePages = append([]string(nil), in.Sites[i].Questions[j].AcceptablePages...)
		}
	}
	return out
}
