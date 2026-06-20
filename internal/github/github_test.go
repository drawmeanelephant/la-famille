package github

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
			json.NewEncoder(w).Encode(resp)
			return
		}
		if r.URL.Path == "/repos/owner/repo/commits/sha456/check-runs" {
			resp := CheckRunsResponse{
				TotalCount: 1,
				CheckRuns: []CheckRun{
					{Status: "in_progress"},
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		if r.URL.Path == "/repos/owner/repo/commits/sha789/check-runs" {
			resp := CheckRunsResponse{
				TotalCount: 1,
				CheckRuns: []CheckRun{
					{Status: "completed", Conclusion: "failure"},
				},
			}
			json.NewEncoder(w).Encode(resp)
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

type redirectTransport struct {
	baseURL string
}

func (t *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// e.g. /repos/owner/repo/commits/... -> /commits/...
	path := req.URL.Path[len("/repos/owner/repo"):]
	urlStr := t.baseURL + path
	newReq, _ := http.NewRequest(req.Method, urlStr, req.Body)
	newReq.Header = req.Header

	return http.DefaultTransport.RoundTrip(newReq)
}
