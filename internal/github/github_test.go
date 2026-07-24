package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestAreChecksPassing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/owner/repo/commits/sha123/check-runs" {
			resp := CheckRunsResponse{
				TotalCount: 2,
				CheckRuns: []CheckRun{
					{Status: "completed", Conclusion: "success"},
					{Status: "completed", Conclusion: "skipped"},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		if r.URL.Path == "/repos/owner/repo/commits/sha456/check-runs" {
			resp := CheckRunsResponse{
				TotalCount: 1,
				CheckRuns: []CheckRun{
					{Status: "in_progress"},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		if r.URL.Path == "/repos/owner/repo/commits/sha789/check-runs" {
			resp := CheckRunsResponse{
				TotalCount: 1,
				CheckRuns: []CheckRun{
					{Status: "completed", Conclusion: "failure"},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// Intercept the default HTTPClient used by Client to point to our test server
	c := NewClient("token", "owner", "repo")

	// Hack to replace base URL for tests:
	// We'll wrap the transport to redirect
	c.HTTPClient.Transport = &redirectTransport{
		baseURL: server.URL + "/repos/owner/repo",
	}

	t.Run("Passing checks", func(t *testing.T) {
		passing, err := c.AreChecksPassing("sha123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !passing {
			t.Errorf("expected passing=true, got false")
		}
	})

	t.Run("In progress checks", func(t *testing.T) {
		passing, err := c.AreChecksPassing("sha456")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if passing {
			t.Errorf("expected passing=false, got true")
		}
	})

	t.Run("Failed checks", func(t *testing.T) {
		passing, err := c.AreChecksPassing("sha789")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if passing {
			t.Errorf("expected passing=false, got true")
		}
	})
}

func TestAreChecksPassing_NoCheckRunsReported(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/owner/repo/commits/shaNoChecks/check-runs" {
			_ = json.NewEncoder(w).Encode(CheckRunsResponse{TotalCount: 0})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	c := NewClient("token", "owner", "repo")
	c.HTTPClient.Transport = &redirectTransport{baseURL: server.URL + "/repos/owner/repo"}

	passing, err := c.AreChecksPassing("shaNoChecks")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if passing {
		t.Error("expected passing=false when no check runs are reported, got true")
	}
}

func TestAreChecksPassing_FailureBeyondFirstPage(t *testing.T) {
	// total_count exceeds one page, and the only failing run lives on the second page.
	const total = 105

	var requestedPages []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/repo/commits/shaPaged/check-runs" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		requestedPages = append(requestedPages, r.URL.Query().Get("page"))

		perPage, err := strconv.Atoi(r.URL.Query().Get("per_page"))
		if err != nil || perPage < 1 || perPage > 100 {
			perPage = 30
		}
		page, err := strconv.Atoi(r.URL.Query().Get("page"))
		if err != nil || page < 1 {
			page = 1
		}

		start := (page - 1) * perPage
		end := start + perPage
		if start > total {
			start = total
		}
		if end > total {
			end = total
		}

		runs := []CheckRun{}
		for i := start; i < end; i++ {
			run := CheckRun{Name: fmt.Sprintf("check-%d", i), Status: "completed", Conclusion: "success"}
			if i == total-1 {
				run.Conclusion = "failure"
			}
			runs = append(runs, run)
		}
		_ = json.NewEncoder(w).Encode(CheckRunsResponse{TotalCount: total, CheckRuns: runs})
	}))
	defer server.Close()

	c := NewClient("token", "owner", "repo")
	c.HTTPClient.Transport = &redirectTransport{baseURL: server.URL + "/repos/owner/repo"}

	passing, err := c.AreChecksPassing("shaPaged")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if passing {
		t.Errorf("expected passing=false, got true (pages requested: %v)", requestedPages)
	}
}

func TestAreChecksPassing_AllPagesSucceed(t *testing.T) {
	const total = 105

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/repo/commits/shaPagedOK/check-runs" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		perPage, err := strconv.Atoi(r.URL.Query().Get("per_page"))
		if err != nil || perPage < 1 || perPage > 100 {
			perPage = 30
		}
		page, err := strconv.Atoi(r.URL.Query().Get("page"))
		if err != nil || page < 1 {
			page = 1
		}

		start := (page - 1) * perPage
		end := start + perPage
		if start > total {
			start = total
		}
		if end > total {
			end = total
		}

		runs := []CheckRun{}
		for i := start; i < end; i++ {
			runs = append(runs, CheckRun{Name: fmt.Sprintf("check-%d", i), Status: "completed", Conclusion: "success"})
		}
		_ = json.NewEncoder(w).Encode(CheckRunsResponse{TotalCount: total, CheckRuns: runs})
	}))
	defer server.Close()

	c := NewClient("token", "owner", "repo")
	c.HTTPClient.Transport = &redirectTransport{baseURL: server.URL + "/repos/owner/repo"}

	passing, err := c.AreChecksPassing("shaPagedOK")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !passing {
		t.Error("expected passing=true when every page of check runs succeeded, got false")
	}
}

// TestAreChecksPassing_PagingIsBounded guards the paging loop against a
// server-reported total it can never satisfy. The loop is driven by
// total_count, so without a ceiling an inflated count turns a single call into
// thousands of sequential requests against the API.
func TestAreChecksPassing_PagingIsBounded(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		// Always one more run, never reaching the advertised total.
		_ = json.NewEncoder(w).Encode(CheckRunsResponse{
			TotalCount: 1000000,
			CheckRuns:  []CheckRun{{Status: "completed", Conclusion: "success"}},
		})
	}))
	defer server.Close()

	c := NewClient("token", "owner", "repo")
	c.HTTPClient.Transport = &redirectTransport{baseURL: server.URL + "/repos/owner/repo"}

	passing, err := c.AreChecksPassing("shaEndless")
	if passing {
		t.Error("expected passing=false when the reported total is never reached")
	}
	if err == nil {
		t.Error("expected an error rather than an unbounded walk")
	}
	if requests > checkRunsMaxPages {
		t.Errorf("made %d requests, want at most %d — the paging loop is unbounded", requests, checkRunsMaxPages)
	}
}

func TestAreChecksPassing_TruncatedPagesAreNotPassing(t *testing.T) {
	// A server that reports more runs than it ever returns must not yield a passing verdict.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/repo/commits/shaTruncated/check-runs" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.URL.Query().Get("page") == "1" {
			_ = json.NewEncoder(w).Encode(CheckRunsResponse{
				TotalCount: 5,
				CheckRuns:  []CheckRun{{Status: "completed", Conclusion: "success"}},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(CheckRunsResponse{TotalCount: 5})
	}))
	defer server.Close()

	c := NewClient("token", "owner", "repo")
	c.HTTPClient.Transport = &redirectTransport{baseURL: server.URL + "/repos/owner/repo"}

	passing, err := c.AreChecksPassing("shaTruncated")
	if passing {
		t.Error("expected passing=false when the API never returned every reported check run")
	}
	if err == nil {
		t.Error("expected an error when the check-run listing is truncated")
	}
}

type redirectTransport struct {
	baseURL string
}

func (t *redirectTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	// e.g. /repos/owner/repo/commits/... -> /commits/...
	path := r.URL.Path[len("/repos/owner/repo"):]
	urlStr := t.baseURL + path
	if r.URL.RawQuery != "" {
		urlStr += "?" + r.URL.RawQuery
	}
	newReq, _ := http.NewRequest(r.Method, urlStr, r.Body) //nolint:gosec // test helper
	newReq.Header = r.Header

	return http.DefaultTransport.RoundTrip(newReq)
}
