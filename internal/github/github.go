package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	Token      string
	Owner      string
	Repo       string
	HTTPClient *http.Client
}

func NewClient(token, owner, repo string) *Client {
	return &Client{
		Token: token,
		Owner: owner,
		Repo:  repo,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) doRequest(method, path string, body interface{}, response interface{}) error {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s%s", c.Owner, c.Repo, path)

	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if c.Token != "" {
		req.Header.Set("Authorization", "token "+c.Token)
	}
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: status=%d %s", resp.StatusCode, string(b))
	}

	if response != nil {
		if err := json.NewDecoder(resp.Body).Decode(response); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}
	return nil
}

type User struct {
	Login string `json:"login"`
}

type PullRequest struct {
	Number    int    `json:"number"`
	Title     string `json:"title"`
	State     string `json:"state"`
	User      User   `json:"user"`
	Head      Ref    `json:"head"`
	Mergeable *bool  `json:"mergeable"` // Using pointer because it can be null
}

type Ref struct {
	Ref string `json:"ref"`
	Sha string `json:"sha"`
}

// ListOpenPRs returns open pull requests, filtered by authors if provided.
func (c *Client) ListOpenPRs(authors []string) ([]PullRequest, error) {
	var prs []PullRequest
	// For simplicity, just get the first page. For a robust implementation, handle pagination.
	err := c.doRequest("GET", "/pulls?state=open", nil, &prs)
	if err != nil {
		return nil, err
	}

	if len(authors) == 0 {
		return prs, nil
	}

	var filtered []PullRequest
	authorMap := make(map[string]bool)
	for _, a := range authors {
		authorMap[strings.ToLower(a)] = true
	}

	for _, pr := range prs {
		if authorMap[strings.ToLower(pr.User.Login)] {
			filtered = append(filtered, pr)
		}
	}

	return filtered, nil
}

// GetPR fetches a single pull request by number.
// Useful to get the up-to-date `mergeable` status which might not be in the list view.
func (c *Client) GetPR(number int) (*PullRequest, error) {
	var pr PullRequest
	err := c.doRequest("GET", fmt.Sprintf("/pulls/%d", number), nil, &pr)
	if err != nil {
		return nil, err
	}
	return &pr, nil
}

type CheckRun struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
}

type CheckRunsResponse struct {
	TotalCount int        `json:"total_count"`
	CheckRuns  []CheckRun `json:"check_runs"`
}

// checkRunsPageSize is the largest page the check-runs API will serve.
const checkRunsPageSize = 100

// checkRunsMaxPages bounds the paging loop. The loop otherwise trusts the
// server-reported total_count to decide when it is done, so an inflated or
// wrong count could turn one call into thousands of sequential requests
// against the API. No real ref approaches this many check runs.
const checkRunsMaxPages = 20

// AreChecksPassing returns true if all check runs for the given ref are completed and successful/skipped.
// Every page of check runs is inspected, so a run that lands beyond the first page cannot hide a failure.
// A ref for which no check runs are reported is NOT passing: the runs may not have been created yet, or
// the repository may report CI through commit statuses, which this endpoint never returns.
func (c *Client) AreChecksPassing(ref string) (bool, error) {
	inspected, total := 0, 0
	for page := 1; page <= checkRunsMaxPages; page++ {
		var resp CheckRunsResponse
		path := fmt.Sprintf("/commits/%s/check-runs?per_page=%d&page=%d", ref, checkRunsPageSize, page)
		if err := c.doRequest("GET", path, nil, &resp); err != nil {
			return false, err
		}

		if page == 1 {
			total = resp.TotalCount
			if total == 0 {
				return false, nil
			}
		}

		for _, check := range resp.CheckRuns {
			if check.Status != "completed" {
				return false, nil
			}
			if check.Conclusion != "success" && check.Conclusion != "skipped" && check.Conclusion != "neutral" {
				return false, nil
			}
		}

		inspected += len(resp.CheckRuns)
		if inspected >= total {
			return true, nil
		}
		if len(resp.CheckRuns) == 0 {
			return false, fmt.Errorf("check runs for %s are truncated: inspected %d of %d", ref, inspected, total)
		}
	}
	return false, fmt.Errorf("check runs for %s exceed %d pages: inspected %d of %d", ref, checkRunsMaxPages, inspected, total)
}

// ClosePR closes a pull request.
func (c *Client) ClosePR(number int) error {
	body := map[string]string{
		"state": "closed",
	}
	return c.doRequest("PATCH", fmt.Sprintf("/pulls/%d", number), body, nil)
}

// MergePR merges a pull request.
func (c *Client) MergePR(number int) error {
	// The API returns a response, but we don't strictly need to parse it unless we want to check merged status
	return c.doRequest("PUT", fmt.Sprintf("/pulls/%d/merge", number), nil, nil)
}

// CreatePR opens a new pull request.
func (c *Client) CreatePR(title, body, head, base string) error {
	reqBody := map[string]string{
		"title": title,
		"body":  body,
		"head":  head,
		"base":  base,
	}
	return c.doRequest("POST", "/pulls", reqBody, nil)
}
