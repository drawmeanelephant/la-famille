package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const (
	defaultAPIBase    = "https://api.github.com"
	githubAPIVersion  = "2022-11-28"
	userAgent         = "la-famille"
	maxErrorBodyBytes = 4096
	defaultPerPage    = 100
	maxPRListPages    = 20
	maxCheckRunPages  = 20
	httpClientTimeout = 10 * time.Second
)

// Client is a minimal GitHub REST API client used by litterbox sync.
type Client struct {
	Token      string
	Owner      string
	Repo       string
	BaseURL    string // optional; defaults to https://api.github.com (overridable in tests)
	HTTPClient *http.Client
}

// NewClient constructs a Client with the project-standard timeout.
func NewClient(token, owner, repo string) *Client {
	return &Client{
		Token: token,
		Owner: owner,
		Repo:  repo,
		HTTPClient: &http.Client{
			Timeout: httpClientTimeout,
		},
	}
}

func (c *Client) apiBase() string {
	if c.BaseURL != "" {
		return strings.TrimRight(c.BaseURL, "/")
	}
	return defaultAPIBase
}

func (c *Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: httpClientTimeout}
}

func (c *Client) doRequest(method, path string, body interface{}, response interface{}) error {
	fullURL := c.apiBase() + "/repos/" + c.Owner + "/" + c.Repo + path

	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, fullURL, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", githubAPIVersion)
	req.Header.Set("User-Agent", userAgent)
	if c.Token != "" {
		req.Header.Set("Authorization", "token "+c.Token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: request failed: %w", method, path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, readErr := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes))
		if readErr != nil {
			return fmt.Errorf("%s %s: API error status=%d (also failed to read body: %v)", method, path, resp.StatusCode, readErr)
		}
		msg := strings.TrimSpace(string(raw))
		if msg == "" {
			msg = resp.Status
		}
		return fmt.Errorf("%s %s: API error status=%d: %s", method, path, resp.StatusCode, msg)
	}

	if response == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("%s %s: failed to read response body: %w", method, path, err)
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, response); err != nil {
		return fmt.Errorf("%s %s: failed to decode response: %w", method, path, err)
	}
	return nil
}

// User is a GitHub user reference.
type User struct {
	Login string `json:"login"`
}

// Label is a GitHub issue/PR label.
type Label struct {
	Name string `json:"name"`
}

// Ref is a git ref with SHA.
type Ref struct {
	Ref string `json:"ref"`
	Sha string `json:"sha"`
}

// PullRequest is the subset of the GitHub PR model needed for policy evaluation.
type PullRequest struct {
	Number    int     `json:"number"`
	Title     string  `json:"title"`
	State     string  `json:"state"`
	Draft     bool    `json:"draft"`
	User      User    `json:"user"`
	Labels    []Label `json:"labels"`
	Head      Ref     `json:"head"`
	Base      Ref     `json:"base"`
	Mergeable *bool   `json:"mergeable"`
}

// LabelNames returns the label names on the PR.
func (pr PullRequest) LabelNames() []string {
	names := make([]string, 0, len(pr.Labels))
	for _, l := range pr.Labels {
		names = append(names, l.Name)
	}
	return names
}

// ListOpenPRs returns open pull requests, optionally filtered by base branch and authors.
// Results are sorted by PR number ascending for deterministic evaluation.
func (c *Client) ListOpenPRs(authors []string, base string) ([]PullRequest, error) {
	var all []PullRequest

	for page := 1; page <= maxPRListPages; page++ {
		q := url.Values{}
		q.Set("state", "open")
		q.Set("per_page", fmt.Sprintf("%d", defaultPerPage))
		q.Set("page", fmt.Sprintf("%d", page))
		if base != "" {
			q.Set("base", base)
		}
		path := "/pulls?" + q.Encode()

		var pagePRs []PullRequest
		if err := c.doRequest("GET", path, nil, &pagePRs); err != nil {
			return nil, err
		}
		if len(pagePRs) == 0 {
			break
		}
		all = append(all, pagePRs...)
		if len(pagePRs) < defaultPerPage {
			break
		}
		if page == maxPRListPages {
			return nil, fmt.Errorf("GET /pulls: pagination truncated after %d pages", maxPRListPages)
		}
	}

	filtered := filterByAuthors(all, authors)
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Number < filtered[j].Number
	})
	return filtered, nil
}

func filterByAuthors(prs []PullRequest, authors []string) []PullRequest {
	if len(authors) == 0 {
		return prs
	}
	authorMap := make(map[string]struct{}, len(authors))
	for _, a := range authors {
		authorMap[strings.ToLower(a)] = struct{}{}
	}
	var filtered []PullRequest
	for _, pr := range prs {
		if _, ok := authorMap[strings.ToLower(pr.User.Login)]; ok {
			filtered = append(filtered, pr)
		}
	}
	return filtered
}

// GetPR fetches a single pull request by number (includes mergeable when available).
func (c *Client) GetPR(number int) (*PullRequest, error) {
	var pr PullRequest
	path := fmt.Sprintf("/pulls/%d", number)
	if err := c.doRequest("GET", path, nil, &pr); err != nil {
		return nil, err
	}
	return &pr, nil
}

// CheckRun is a single GitHub check run.
type CheckRun struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
}

// CheckRunsResponse is the list check-runs API payload.
type CheckRunsResponse struct {
	TotalCount int        `json:"total_count"`
	CheckRuns  []CheckRun `json:"check_runs"`
}

// CheckState classifies aggregate check-run status for policy decisions.
type CheckState string

const (
	CheckStateNone    CheckState = "none"
	CheckStatePending CheckState = "pending"
	CheckStateFailed  CheckState = "failed"
	CheckStatePassing CheckState = "passing"
)

// CheckSummary is the policy-facing view of check runs for a commit.
type CheckSummary struct {
	State CheckState
	Total int
}

// GetCheckSummary inspects all check-run pages for ref and classifies them.
func (c *Client) GetCheckSummary(ref string) (CheckSummary, error) {
	var all []CheckRun
	totalCount := 0

	for page := 1; page <= maxCheckRunPages; page++ {
		q := url.Values{}
		q.Set("per_page", fmt.Sprintf("%d", defaultPerPage))
		q.Set("page", fmt.Sprintf("%d", page))
		path := fmt.Sprintf("/commits/%s/check-runs?%s", url.PathEscape(ref), q.Encode())

		var resp CheckRunsResponse
		if err := c.doRequest("GET", path, nil, &resp); err != nil {
			return CheckSummary{}, err
		}
		if page == 1 {
			totalCount = resp.TotalCount
		}
		all = append(all, resp.CheckRuns...)

		if totalCount == 0 || len(all) >= totalCount || len(resp.CheckRuns) == 0 {
			break
		}
		if page == maxCheckRunPages && len(all) < totalCount {
			return CheckSummary{}, fmt.Errorf(
				"GET /commits/%s/check-runs: pagination truncated: got %d of %d",
				ref, len(all), totalCount,
			)
		}
	}

	if totalCount > 0 && len(all) < totalCount {
		return CheckSummary{}, fmt.Errorf(
			"GET /commits/%s/check-runs: pagination truncated: got %d of %d",
			ref, len(all), totalCount,
		)
	}

	return summarizeChecks(all, totalCount), nil
}

func summarizeChecks(runs []CheckRun, totalCount int) CheckSummary {
	total := totalCount
	if total == 0 {
		total = len(runs)
	}
	if total == 0 {
		return CheckSummary{State: CheckStateNone, Total: 0}
	}

	for _, check := range runs {
		if check.Status != "completed" {
			return CheckSummary{State: CheckStatePending, Total: total}
		}
	}
	for _, check := range runs {
		if check.Conclusion != "success" && check.Conclusion != "skipped" && check.Conclusion != "neutral" {
			return CheckSummary{State: CheckStateFailed, Total: total}
		}
	}
	return CheckSummary{State: CheckStatePassing, Total: total}
}

// AreChecksPassing reports whether checks are in the passing state.
// Prefer GetCheckSummary when a richer explanation is needed.
func (c *Client) AreChecksPassing(ref string) (bool, error) {
	summary, err := c.GetCheckSummary(ref)
	if err != nil {
		return false, err
	}
	return summary.State == CheckStatePassing, nil
}

// ClosePR closes a pull request.
func (c *Client) ClosePR(number int) error {
	body := map[string]string{"state": "closed"}
	return c.doRequest("PATCH", fmt.Sprintf("/pulls/%d", number), body, nil)
}

// MergeResult is the GitHub merge endpoint response.
type MergeResult struct {
	Merged  bool   `json:"merged"`
	Message string `json:"message"`
	SHA     string `json:"sha"`
}

// MergePR squash-merges a pull request at the expected head SHA.
func (c *Client) MergePR(number int, sha string) error {
	body := map[string]string{
		"merge_method": "squash",
		"sha":          sha,
	}
	var result MergeResult
	if err := c.doRequest("PUT", fmt.Sprintf("/pulls/%d/merge", number), body, &result); err != nil {
		return err
	}
	if !result.Merged {
		msg := result.Message
		if msg == "" {
			msg = "merged=false"
		}
		return fmt.Errorf("merge of PR #%d was not confirmed: %s", number, msg)
	}
	return nil
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

// Repository is the subset of repository metadata used by litterbox.
type Repository struct {
	DefaultBranch string `json:"default_branch"`
}

// GetDefaultBranch returns the repository default branch from the GitHub API.
func (c *Client) GetDefaultBranch() (string, error) {
	var repo Repository
	// Path "" hits GET /repos/{owner}/{repo}
	if err := c.doRequest("GET", "", nil, &repo); err != nil {
		return "", err
	}
	if strings.TrimSpace(repo.DefaultBranch) == "" {
		return "", fmt.Errorf("repository default_branch is empty")
	}
	return repo.DefaultBranch, nil
}

// APIClient is the GitHub surface used by RunSync (real client or test fake).
type APIClient interface {
	ListOpenPRs(authors []string, base string) ([]PullRequest, error)
	GetPR(number int) (*PullRequest, error)
	GetCheckSummary(ref string) (CheckSummary, error)
	MergePR(number int, sha string) error
	ClosePR(number int) error
	CreatePR(title, body, head, base string) error
	GetDefaultBranch() (string, error)
}
