package retrieval

import (
	"reflect"
	"testing"
)

func TestRankerAbstainsOnUnsupportedQueryEvidence(t *testing.T) {
	c := graphCorpus()
	c.Chunks = append(c.Chunks, Chunk{
		ID: "telescope", PageID: "telescope", Title: "Telescope Log",
		Text: "The telescope recorded lunar craters during the observation.",
	})
	for _, query := range []string{
		"salinity sensor warranty duration",
		"telescope insurance deductible",
		"sensor sensor sensor refund eligibility",
		"What is the telescope insurance deductible?",
		"sensor warranty",
	} {
		t.Run(query, func(t *testing.T) {
			if got := NewRanker(c).Rank(query, 5); len(got) != 0 {
				t.Fatalf("topical evidence is not enough for %q: %+v", query, got)
			}
			got := NewGraphRanker(c).Retrieve(query, 5)
			if len(got.Scored) != 0 || len(got.Paths) != 0 {
				t.Fatalf("graph expansion bypassed abstention for %q: %+v", query, got)
			}
		})
	}
}

func TestRankerKeepsSupportedQueriesAndMinorVocabularyGaps(t *testing.T) {
	c := graphCorpus()
	c.Chunks = append(c.Chunks,
		Chunk{ID: "warranty", PageID: "warranty", Title: "Sensor Terms",
			Text: "The salinity sensor warranty duration is three years."},
		Chunk{ID: "insurance", PageID: "insurance", Title: "Telescope Insurance",
			Text: "The telescope insurance deductible is fifty credits."})
	r := NewRanker(c)
	for _, query := range []string{
		"salinity sensor warranty duration",
		"What is the telescope insurance deductible?",
		"What is the heading on the sensor page?",
		"Who says they found elevated concentration?",
		"salinity sensor anomaly unexplained",
		"salinity sensor anomaly unexplained unexplained unexplained",
		"tidal anomaly",
	} {
		t.Run(query, func(t *testing.T) {
			if got := r.Rank(query, 5); len(got) == 0 {
				t.Fatalf("supported query was rejected: %q", query)
			}
		})
	}
	short := r.Rank("salinity sensor anomaly unexplained", 5)
	repeated := r.Rank("salinity sensor anomaly unexplained unexplained unexplained", 5)
	if !reflect.DeepEqual(short, repeated) {
		t.Fatal("repeating an absent word changed accepted ranking")
	}
}
