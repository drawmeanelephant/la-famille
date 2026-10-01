package llm

import (
	"context"
	"strings"
	"testing"
)

func TestFakeGraphResponseNamesAndCitesOrderedPath(t *testing.T) {
	req := Request{GroundingPaths: [][]CitationHint{{
		{Key: "2", Title: "A"}, {Key: "4", Title: "C"}, {Key: "1", Title: "B"},
	}}}
	for _, mode := range []string{"", "cite"} {
		resp, err := (&FakeProvider{EchoMode: mode}).Complete(context.Background(), req)
		if err != nil || !strings.Contains(resp.Answer, "A [2] → C [4] → B [1]") ||
			!strings.Contains(resp.Answer, "Synthetic") {
			t.Fatalf("fake hid synthetic provenance or dropped bridge: %+v %v", resp, err)
		}
	}
	miss, err := (&FakeProvider{EchoMode: "miss"}).Complete(context.Background(), req)
	if err != nil || !strings.Contains(miss.Answer, "[99]") {
		t.Fatal("path short-circuited the citation mutation control")
	}
}
