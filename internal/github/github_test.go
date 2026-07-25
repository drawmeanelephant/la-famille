package github

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func testClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	c := NewClient("secret-token-value", "owner", "repo")
	c.BaseURL = server.URL
	return c
}

func TestGetCheckSummaryStates(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("per_page") != "100" {
			t.Errorf("expected per_page=100, got %q", r.URL.Query().Get("per_page"))
		}
		switch {
		case strings.Contains(r.URL.Path, "/commits/sha123/check-runs"):
			_ = json.NewEncoder(w).Encode(CheckRunsResponse{
				TotalCount: 2,
				CheckRuns: []CheckRun{
					{Status: "completed", Conclusion: "success"},
					{Status: "completed", Conclusion: "skipped"},
				},
			})
		case strings.Contains(r.URL.Path, "/commits/sha456/check-runs"):
			_ = json.NewEncoder(w).Encode(CheckRunsResponse{
				TotalCount: 1,
				CheckRuns:  []CheckRun{{Status: "in_progress"}},
			})
		case strings.Contains(r.URL.Path, "/commits/sha789/check-runs"):
			_ = json.NewEncoder(w).Encode(CheckRunsResponse{
				TotalCount: 1,
				CheckRuns:  []CheckRun{{Status: "completed", Conclusion: "failure"}},
			})
		case strings.Contains(r.URL.Path, "/commits/sha000/check-runs"):
			_ = json.NewEncoder(w).Encode(CheckRunsResponse{TotalCount: 0, CheckRuns: []CheckRun{}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	t.Run("Passing checks", func(t *testing.T) {
		s, err := c.GetCheckSummary("sha123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.State != CheckStatePassing || s.Total != 2 {
			t.Errorf("got %+v, want passing total=2", s)
		}
		ok, err := c.AreChecksPassing("sha123")
		if err != nil || !ok {
			t.Errorf("AreChecksPassing = %v, %v", ok, err)
		}
	})

	t.Run("In progress checks", func(t *testing.T) {
		s, err := c.GetCheckSummary("sha456")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.State != CheckStatePending {
			t.Errorf("got %s, want pending", s.State)
		}
	})

	t.Run("Failed checks", func(t *testing.T) {
		s, err := c.GetCheckSummary("sha789")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.State != CheckStateFailed {
			t.Errorf("got %s, want failed", s.State)
		}
	})

	t.Run("Zero checks distinct from passing", func(t *testing.T) {
		s, err := c.GetCheckSummary("sha000")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.State != CheckStateNone || s.Total != 0 {
			t.Errorf("got %+v, want none total=0", s)
		}
		ok, err := c.AreChecksPassing("sha000")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ok {
			t.Error("zero checks must not count as passing")
		}
	})
}

func TestCheckRunPagination(t *testing.T) {
	pages := 0
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		pages++
		page := r.URL.Query().Get("page")
		if r.URL.Query().Get("per_page") != "100" {
			t.Errorf("per_page not preserved: %s", r.URL.RawQuery)
		}
		switch page {
		case "1", "":
			runs := make([]CheckRun, 100)
			for i := range runs {
				runs[i] = CheckRun{Status: "completed", Conclusion: "success", Name: "c1"}
			}
			_ = json.NewEncoder(w).Encode(CheckRunsResponse{TotalCount: 101, CheckRuns: runs})
		case "2":
			_ = json.NewEncoder(w).Encode(CheckRunsResponse{
				TotalCount: 101,
				CheckRuns:  []CheckRun{{Status: "completed", Conclusion: "neutral", Name: "last"}},
			})
		default:
			t.Errorf("unexpected page %s", page)
			w.WriteHeader(http.StatusNotFound)
		}
	})

	s, err := c.GetCheckSummary("paginated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pages != 2 {
		t.Errorf("expected 2 pages, got %d", pages)
	}
	if s.State != CheckStatePassing || s.Total != 101 {
		t.Errorf("got %+v", s)
	}
}

func TestCheckRunPaginationTruncated(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, _ *http.Request) {
		// Always claim more results than returned, never complete.
		_ = json.NewEncoder(w).Encode(CheckRunsResponse{
			TotalCount: 5000,
			CheckRuns:  []CheckRun{{Status: "completed", Conclusion: "success"}},
		})
	})
	_, err := c.GetCheckSummary("trunc")
	if err == nil {
		t.Fatal("expected truncation error")
	}
	if !strings.Contains(err.Error(), "truncated") {
		t.Errorf("error should mention truncated: %v", err)
	}
}

func TestListOpenPRsPaginationAndFiltering(t *testing.T) {
	var seenQueries []string
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		seenQueries = append(seenQueries, r.URL.RawQuery)
		if r.URL.Query().Get("state") != "open" {
			t.Errorf("state=%q", r.URL.Query().Get("state"))
		}
		if r.URL.Query().Get("base") != "master" {
			t.Errorf("base=%q", r.URL.Query().Get("base"))
		}
		if r.URL.Query().Get("per_page") != "100" {
			t.Errorf("per_page=%q", r.URL.Query().Get("per_page"))
		}
		page := r.URL.Query().Get("page")
		switch page {
		case "1":
			prs := make([]PullRequest, 100)
			for i := 0; i < 100; i++ {
				prs[i] = PullRequest{
					Number: 200 - i, // reverse order to test sort
					User:   User{Login: "google-labs-jules"},
					Title:  "bot",
				}
			}
			// inject a non-bot that should be filtered
			prs[0].User.Login = "human"
			prs[0].Number = 999
			_ = json.NewEncoder(w).Encode(prs)
		case "2":
			_ = json.NewEncoder(w).Encode([]PullRequest{
				{Number: 5, User: User{Login: "Google-Labs-Code"}, Title: "case"},
				{Number: 3, User: User{Login: "google-labs-jules"}, Title: "early"},
			})
		default:
			_ = json.NewEncoder(w).Encode([]PullRequest{})
		}
	})

	prs, err := c.ListOpenPRs([]string{"google-labs-jules", "google-labs-code"}, "master")
	if err != nil {
		t.Fatalf("ListOpenPRs: %v", err)
	}
	if len(seenQueries) < 2 {
		t.Fatalf("expected multi-page queries, got %v", seenQueries)
	}
	for _, q := range seenQueries {
		if !strings.Contains(q, "state=open") || !strings.Contains(q, "base=master") || !strings.Contains(q, "per_page=100") {
			t.Errorf("query missing required params: %s", q)
		}
	}

	// human filtered out; case-insensitive author kept; sorted by number
	for i := 1; i < len(prs); i++ {
		if prs[i-1].Number > prs[i].Number {
			t.Fatalf("PRs not sorted: %d then %d", prs[i-1].Number, prs[i].Number)
		}
	}
	for _, pr := range prs {
		if strings.EqualFold(pr.User.Login, "human") {
			t.Error("human author should be filtered")
		}
	}
	if len(prs) == 0 {
		t.Fatal("expected some PRs")
	}
	// number 5 from page 2 with case-variant author should be present
	found := false
	for _, pr := range prs {
		if pr.Number == 5 {
			found = true
		}
	}
	if !found {
		t.Error("expected case-insensitive author match for PR #5")
	}
}

func TestMergePRIncludesSquashAndSHA(t *testing.T) {
	var gotBody map[string]string
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/repos/owner/repo/pulls/42/merge" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		_ = json.NewEncoder(w).Encode(MergeResult{Merged: true, Message: "ok", SHA: "deadbeef"})
	})
	if err := c.MergePR(42, "abc123"); err != nil {
		t.Fatalf("MergePR: %v", err)
	}
	if gotBody["merge_method"] != "squash" {
		t.Errorf("merge_method=%q", gotBody["merge_method"])
	}
	if gotBody["sha"] != "abc123" {
		t.Errorf("sha=%q", gotBody["sha"])
	}
}

func TestMergePRMergedFalse(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(MergeResult{Merged: false, Message: "not mergeable"})
	})
	err := c.MergePR(7, "sha")
	if err == nil {
		t.Fatal("expected error when merged=false")
	}
	if !strings.Contains(err.Error(), "not confirmed") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestClosePR(t *testing.T) {
	var method, path string
	var body map[string]string
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		path = r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		w.WriteHeader(http.StatusOK)
	})
	if err := c.ClosePR(99); err != nil {
		t.Fatalf("ClosePR: %v", err)
	}
	if method != http.MethodPatch || path != "/repos/owner/repo/pulls/99" {
		t.Errorf("%s %s", method, path)
	}
	if body["state"] != "closed" {
		t.Errorf("body=%v", body)
	}
}

func TestGetDefaultBranch(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/repo" {
			t.Errorf("path=%s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(Repository{DefaultBranch: "main"})
	})
	got, err := c.GetDefaultBranch()
	if err != nil {
		t.Fatalf("GetDefaultBranch: %v", err)
	}
	if got != "main" {
		t.Errorf("got %q", got)
	}
}

func TestAPIErrorContainsMethodPathStatusNotToken(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			t.Error("expected Authorization header on request")
		}
		if r.Header.Get("Accept") != "application/vnd.github+json" {
			t.Errorf("Accept=%q", r.Header.Get("Accept"))
		}
		if r.Header.Get("X-GitHub-Api-Version") == "" {
			t.Error("missing API version header")
		}
		if r.Header.Get("User-Agent") != userAgent {
			t.Errorf("User-Agent=%q", r.Header.Get("User-Agent"))
		}
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"nope"}`))
	})
	err := c.ClosePR(1)
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "PATCH") || !strings.Contains(msg, "/pulls/1") || !strings.Contains(msg, "403") {
		t.Errorf("error missing method/path/status: %v", err)
	}
	if strings.Contains(msg, "secret-token-value") {
		t.Error("error must not contain token")
	}
}

func TestEmptySuccessBody(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	if err := c.ClosePR(1); err != nil {
		t.Fatalf("empty body should be ok: %v", err)
	}
}
