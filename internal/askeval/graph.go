package askeval

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/tbuddy/la-famille/internal/ask"
	"github.com/tbuddy/la-famille/internal/retrieval"
)

// GraphComparison retains both arms instead of hiding changed pages behind
// aggregate averages. Every arm uses identical labels, K and lexical scoring.
type GraphComparison struct {
	Lexical         Report
	Graph           Report
	RecallOK        bool
	PrecisionOK     bool
	SinglePageOK    bool
	SinglePageCount int
	Passed          bool
}

func compareGraph(ctx context.Context, opts Options) (Report, error) {
	if opts.Embeddings || opts.Ranker != nil {
		return Report{}, errors.New("ask eval: graph comparison requires the production lexical baseline, without embeddings or ranker overrides")
	}
	opts.CompareGraph = false
	opts.GraphExpansion = false
	off, err := Run(ctx, opts)
	if err != nil {
		return Report{}, err
	}
	opts.GraphExpansion = true
	on, err := Run(ctx, opts)
	if err != nil {
		return Report{}, err
	}
	comparison := &GraphComparison{
		Lexical: off, Graph: on,
		RecallOK:     on.RecallAtK+1e-9 >= off.RecallAtK,
		PrecisionOK:  on.PrecisionAtK+1e-9 >= off.PrecisionAtK,
		SinglePageOK: true,
	}
	for i, before := range off.Questions {
		if before.Unanswerable || len(before.AcceptablePages) != 1 {
			continue
		}
		comparison.SinglePageCount++
		after := on.Questions[i]
		comparison.SinglePageOK = comparison.SinglePageOK &&
			after.PrecisionAtK+1e-9 >= before.PrecisionAtK &&
			after.RecallAtK+1e-9 >= before.RecallAtK
	}
	comparison.Passed = on.Passed && comparison.RecallOK && comparison.PrecisionOK && comparison.SinglePageOK
	on.Comparison = comparison
	on.Passed = comparison.Passed
	return on, nil
}

func groundedPathMatches(answer ask.AnswerResponse, expected []string) bool {
	if answer.NoAnswer || answer.Status != "answered" {
		return false
	}
	reverse := slices.Clone(expected)
	slices.Reverse(reverse)
	for _, path := range answer.Paths {
		var pages []string
		complete := true
		for _, node := range path.Nodes {
			pages = append(pages, node.PageID)
			complete = complete && strings.Contains(answer.Answer, node.Title) &&
				slices.ContainsFunc(answer.Sources, func(source retrieval.SourceCard) bool {
					return source.ChunkID == node.ChunkID && source.Key == "["+node.Key+"]"
				})
		}
		if complete && (slices.Equal(pages, expected) || slices.Equal(pages, reverse)) {
			return true
		}
	}
	return false
}

func pathLabel(path retrieval.GraphPath) string {
	var pages []string
	for _, node := range path.Nodes {
		pages = append(pages, node.PageID)
	}
	return strings.Join(pages, " → ")
}

func writeGraphComparison(w io.Writer, c *GraphComparison) {
	fmt.Fprintln(w, "\nSame-run graph comparison (lexical only; embeddings disabled):")
	fmt.Fprintf(w, "MODE       RECALL@%d  PRECISION@%d  QUESTION GATES\n", c.Graph.K, c.Graph.K)
	for _, arm := range []struct {
		name string
		r    Report
	}{{"graph off", c.Lexical}, {"graph on", c.Graph}} {
		fmt.Fprintf(w, "%-10s %.4f    %.4f       %d/%d\n", arm.name, arm.r.RecallAtK,
			arm.r.PrecisionAtK, arm.r.PassedCount, len(arm.r.Questions))
	}
	fmt.Fprintf(w, "Delta: recall=%+.4f precision=%+.4f\nSingle-page non-dilution: %t (%d questions); comparison passed: %t\n",
		c.Graph.RecallAtK-c.Lexical.RecallAtK, c.Graph.PrecisionAtK-c.Lexical.PrecisionAtK,
		c.SinglePageOK, c.SinglePageCount, c.Passed)
	for i, before := range c.Lexical.Questions {
		after := c.Graph.Questions[i]
		fmt.Fprintf(w, "  %s/%s: off=%v on=%v; recall %.4f -> %.4f; precision %.4f -> %.4f\n",
			before.SiteID, before.QuestionID, before.RetrievedPages, after.RetrievedPages,
			before.RecallAtK, after.RecallAtK, before.PrecisionAtK, after.PrecisionAtK)
	}
}
