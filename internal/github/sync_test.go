package github

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeClient records mutating calls and serves scripted responses.
type fakeClient struct {
	mu sync.Mutex

	prs           []PullRequest
	prByNumber    map[int]*PullRequest
	checks        map[string]CheckSummary
	checkErrs     map[string]error
	getPRErrs     map[int]error
	defaultBranch string
	listErr       error

	merges  []mergeCall
	closes  []int
	creates []createCall

	mergeErr        error
	closeErr        error
	createErr       error
	createFailTimes int
	createAttempts  int
}

type mergeCall struct {
	Number int
	SHA    string
}

type createCall struct {
	Title, Body, Head, Base string
}

func (f *fakeClient) ListOpenPRs(authors []string, _ string) ([]PullRequest, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return filterByAuthors(f.prs, authors), nil
}

func (f *fakeClient) GetPR(number int) (*PullRequest, error) {
	if err, ok := f.getPRErrs[number]; ok {
		return nil, err
	}
	if pr, ok := f.prByNumber[number]; ok {
		cp := *pr
		return &cp, nil
	}
	return nil, fmt.Errorf("PR %d not found", number)
}

func (f *fakeClient) GetCheckSummary(ref string) (CheckSummary, error) {
	if err, ok := f.checkErrs[ref]; ok {
		return CheckSummary{}, err
	}
	if s, ok := f.checks[ref]; ok {
		return s, nil
	}
	return CheckSummary{State: CheckStatePassing, Total: 1}, nil
}

func (f *fakeClient) MergePR(number int, sha string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.merges = append(f.merges, mergeCall{Number: number, SHA: sha})
	return f.mergeErr
}

func (f *fakeClient) ClosePR(number int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closes = append(f.closes, number)
	return f.closeErr
}

func (f *fakeClient) CreatePR(title, body, head, base string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.createAttempts++
	if f.createFailTimes > 0 && f.createAttempts <= f.createFailTimes {
		return fmt.Errorf("transient create failure")
	}
	f.creates = append(f.creates, createCall{title, body, head, base})
	return f.createErr
}

func (f *fakeClient) GetDefaultBranch() (string, error) {
	if f.defaultBranch == "" {
		return "", fmt.Errorf("no default branch")
	}
	return f.defaultBranch, nil
}

type fakeGit struct {
	mu sync.Mutex

	hasChanges    bool
	remoteURL     string
	currentBranch string
	branchStack   []string

	checkoutNewCalls []string
	checkoutCalls    []string
	addAllCalls      int
	commitCalls      int
	pushCalls        []string
}

func (g *fakeGit) HasUncommittedChanges() (bool, error) { return g.hasChanges, nil }
func (g *fakeGit) GetRemoteURL(_ string) (string, error) {
	if g.remoteURL == "" {
		return "", fmt.Errorf("no remote")
	}
	return g.remoteURL, nil
}
func (g *fakeGit) CurrentBranch() (string, error) {
	if g.currentBranch == "" {
		return "", fmt.Errorf("unknown branch")
	}
	return g.currentBranch, nil
}
func (g *fakeGit) CheckoutNewBranch(branchName string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.checkoutNewCalls = append(g.checkoutNewCalls, branchName)
	g.branchStack = append(g.branchStack, branchName)
	return nil
}
func (g *fakeGit) Checkout(branchName string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.checkoutCalls = append(g.checkoutCalls, branchName)
	g.currentBranch = branchName
	return nil
}
func (g *fakeGit) AddAll() error {
	g.addAllCalls++
	return nil
}
func (g *fakeGit) Commit(_, _, _ string) error {
	g.commitCalls++
	return nil
}
func (g *fakeGit) Push(remote, branchName string) error {
	g.pushCalls = append(g.pushCalls, remote+"/"+branchName)
	return nil
}

func boolPtr(b bool) *bool { return &b }

func eligibleListedPR(n int, sha string) PullRequest {
	return PullRequest{
		Number:    n,
		Title:     fmt.Sprintf("PR %d", n),
		User:      User{Login: "google-labs-jules"},
		Draft:     false,
		Labels:    []Label{{Name: "litterbox-approved"}},
		Head:      Ref{Ref: "jules/x", Sha: sha},
		Base:      Ref{Ref: "master"},
		Mergeable: boolPtr(true),
	}
}

func baseSyncConfig(client *fakeClient, g *fakeGit) SyncConfig {
	return SyncConfig{
		Token:         "tok",
		BotAuthors:    DefaultBotAuthors,
		BaseBranch:    "master",
		RequiredLabel: DefaultRequiredLabel,
		Client:        client,
		Git:           g,
		Owner:         "drawmeanelephant",
		Repo:          "la-famille",
		Now:           func() time.Time { return time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC) },
		Sleep:         func(time.Duration) {},
	}
}

func TestRunSyncDryRunNoMutations(t *testing.T) {
	pr := eligibleListedPR(414, "sha414")
	client := &fakeClient{
		prs:        []PullRequest{pr},
		prByNumber: map[int]*PullRequest{414: &pr},
		checks:     map[string]CheckSummary{"sha414": {State: CheckStatePassing, Total: 1}},
	}
	g := &fakeGit{hasChanges: true, currentBranch: "master"}
	cfg := baseSyncConfig(client, g)
	// dry-run default; publish flag set so we can see WOULD_CREATE without mutation
	cfg.PublishLocalChanges = true

	result, err := RunSync(cfg)
	if err != nil {
		t.Fatalf("RunSync: %v", err)
	}
	if len(client.merges) != 0 || len(client.closes) != 0 || len(client.creates) != 0 {
		t.Fatalf("dry-run performed mutations: merges=%v closes=%v creates=%v", client.merges, client.closes, client.creates)
	}
	if len(g.checkoutNewCalls) != 0 || g.addAllCalls != 0 || g.commitCalls != 0 || len(g.pushCalls) != 0 {
		t.Fatalf("dry-run performed git mutations")
	}
	if result.WouldMerge != 1 || result.Merged != 0 {
		t.Errorf("result counts: %+v", result)
	}
	if result.LocalAction != LocalWouldCreatePR {
		t.Errorf("LocalAction=%q", result.LocalAction)
	}
	if result.Decisions[0].Action != ActionWouldMerge {
		t.Errorf("decision=%s", result.Decisions[0].Action)
	}
}

func TestRunSyncApplyMergesEligibleOnly(t *testing.T) {
	good := eligibleListedPR(10, "sha10")
	bad := eligibleListedPR(11, "sha11")
	bad.Labels = nil // ineligible

	client := &fakeClient{
		prs: []PullRequest{bad, good}, // unsorted listing; evaluation uses listed order from ListOpenPRs which sorts after filter
		prByNumber: map[int]*PullRequest{
			10: &good,
			11: &bad,
		},
		checks: map[string]CheckSummary{
			"sha10": {State: CheckStatePassing, Total: 1},
			"sha11": {State: CheckStatePassing, Total: 1},
		},
	}
	// ensure list returns unsorted then our ListOpenPRs on real client sorts —
	// fake returns as-is, so sort in test setup by using numbers already ordered or sort fake.
	// Filter doesn't sort; RunSync iterates list order. For fake, pre-sort.
	client.prs = []PullRequest{good, bad}

	cfg := baseSyncConfig(client, &fakeGit{})
	cfg.Apply = true

	result, err := RunSync(cfg)
	if err != nil {
		t.Fatalf("RunSync: %v", err)
	}
	if len(client.merges) != 1 || client.merges[0].Number != 10 || client.merges[0].SHA != "sha10" {
		t.Fatalf("merges=%v", client.merges)
	}
	if result.Merged != 1 || result.Skipped != 1 {
		t.Errorf("merged=%d skipped=%d", result.Merged, result.Skipped)
	}
}

func TestRunSyncConflictNeverClosedWithoutFlag(t *testing.T) {
	pr := eligibleListedPR(15, "sha15")
	pr.Mergeable = boolPtr(false)
	client := &fakeClient{
		prs:        []PullRequest{pr},
		prByNumber: map[int]*PullRequest{15: &pr},
		checks:     map[string]CheckSummary{"sha15": {State: CheckStatePassing, Total: 1}},
	}
	cfg := baseSyncConfig(client, &fakeGit{})
	cfg.Apply = true

	result, err := RunSync(cfg)
	if err != nil {
		t.Fatalf("RunSync: %v", err)
	}
	if len(client.closes) != 0 {
		t.Fatalf("unexpected closes: %v", client.closes)
	}
	if result.Skipped != 1 {
		t.Errorf("expected skip, got %+v", result)
	}
}

func TestRunSyncConflictCloseWithFlag(t *testing.T) {
	pr := eligibleListedPR(16, "sha16")
	pr.Mergeable = boolPtr(false)
	client := &fakeClient{
		prs:        []PullRequest{pr},
		prByNumber: map[int]*PullRequest{16: &pr},
		checks:     map[string]CheckSummary{"sha16": {State: CheckStateNone, Total: 0}},
	}
	cfg := baseSyncConfig(client, &fakeGit{})
	cfg.Apply = true
	cfg.CloseConflicts = true

	result, err := RunSync(cfg)
	if err != nil {
		t.Fatalf("RunSync: %v", err)
	}
	if len(client.closes) != 1 || client.closes[0] != 16 {
		t.Fatalf("closes=%v", client.closes)
	}
	if result.Closed != 1 {
		t.Errorf("Closed=%d", result.Closed)
	}
}

func TestRunSyncAggregatesPRErrors(t *testing.T) {
	pr1 := eligibleListedPR(1, "s1")
	pr2 := eligibleListedPR(2, "s2")
	client := &fakeClient{
		prs: []PullRequest{pr1, pr2},
		prByNumber: map[int]*PullRequest{
			1: &pr1,
			2: &pr2,
		},
		getPRErrs: map[int]error{1: errors.New("boom")},
		checks: map[string]CheckSummary{
			"s2": {State: CheckStatePassing, Total: 1},
		},
	}
	cfg := baseSyncConfig(client, &fakeGit{})
	cfg.Apply = true

	result, err := RunSync(cfg)
	if err == nil {
		t.Fatal("expected aggregated error")
	}
	if !strings.Contains(err.Error(), "PR #1") {
		t.Errorf("error should mention PR #1: %v", err)
	}
	// PR 2 should still be evaluated and merged
	if len(client.merges) != 1 || client.merges[0].Number != 2 {
		t.Fatalf("expected PR #2 merged, got %v", client.merges)
	}
	if result.Merged != 1 {
		t.Errorf("Merged=%d", result.Merged)
	}
}

func TestRunSyncLocalChangesRequireFlag(t *testing.T) {
	client := &fakeClient{prs: nil}
	g := &fakeGit{hasChanges: true, currentBranch: "master"}
	cfg := baseSyncConfig(client, g)
	cfg.Apply = true

	result, err := RunSync(cfg)
	if err != nil {
		t.Fatalf("RunSync: %v", err)
	}
	if result.LocalAction != LocalPresentNotAuth {
		t.Errorf("LocalAction=%q", result.LocalAction)
	}
	if len(g.checkoutNewCalls) != 0 || len(client.creates) != 0 {
		t.Fatal("should not publish without flag")
	}
}

func TestRunSyncLocalPublishApplyRestoresBranch(t *testing.T) {
	client := &fakeClient{prs: nil}
	g := &fakeGit{hasChanges: true, currentBranch: "feature-x"}
	cfg := baseSyncConfig(client, g)
	cfg.Apply = true
	cfg.PublishLocalChanges = true

	result, err := RunSync(cfg)
	if err != nil {
		t.Fatalf("RunSync: %v", err)
	}
	if result.LocalAction != LocalCreatedPR {
		t.Errorf("LocalAction=%q reason=%q", result.LocalAction, result.LocalReason)
	}
	if len(g.checkoutNewCalls) != 1 || !strings.HasPrefix(g.checkoutNewCalls[0], "jules-auto-") {
		t.Errorf("checkoutNew=%v", g.checkoutNewCalls)
	}
	if len(g.checkoutCalls) == 0 || g.checkoutCalls[len(g.checkoutCalls)-1] != "feature-x" {
		t.Errorf("expected restore to feature-x, checkoutCalls=%v", g.checkoutCalls)
	}
	if len(client.creates) != 1 {
		t.Fatalf("creates=%v", client.creates)
	}
	// CreatePR has no labels field; body may document the label but must not claim auto-application.
	if strings.Contains(strings.ToLower(client.creates[0].Body), "automatically applied") {
		t.Error("local PR body must not claim the approval label was automatically applied")
	}
}

func TestRunSyncLocalPublishRetryUsesInjectedSleeper(t *testing.T) {
	client := &fakeClient{prs: nil, createFailTimes: 2}
	g := &fakeGit{hasChanges: true, currentBranch: "master"}
	sleeps := 0
	cfg := baseSyncConfig(client, g)
	cfg.Apply = true
	cfg.PublishLocalChanges = true
	cfg.Sleep = func(d time.Duration) {
		sleeps++
		if d <= 0 {
			t.Errorf("unexpected sleep duration %v", d)
		}
	}

	_, err := RunSync(cfg)
	if err != nil {
		t.Fatalf("RunSync: %v", err)
	}
	if sleeps != 3 { // 3 attempts until success (fail, fail, success)
		t.Errorf("sleeps=%d, createAttempts=%d", sleeps, client.createAttempts)
	}
	if len(client.creates) != 1 {
		t.Errorf("creates=%v", client.creates)
	}
}

func TestRunSyncResolvesDefaultBranch(t *testing.T) {
	client := &fakeClient{prs: nil, defaultBranch: "trunk"}
	cfg := baseSyncConfig(client, &fakeGit{})
	cfg.BaseBranch = ""

	result, err := RunSync(cfg)
	if err != nil {
		t.Fatalf("RunSync: %v", err)
	}
	if result.BaseBranch != "trunk" {
		t.Errorf("BaseBranch=%q", result.BaseBranch)
	}
}

func TestRunSyncDefaultBranchFailureIsOperational(t *testing.T) {
	client := &fakeClient{prs: nil, defaultBranch: ""}
	cfg := baseSyncConfig(client, &fakeGit{})
	cfg.BaseBranch = ""
	_, err := RunSync(cfg)
	if err == nil {
		t.Fatal("expected error resolving default branch")
	}
}

func TestValidateRejectsEmptyRequiredLabel(t *testing.T) {
	cfg := SyncConfig{Token: "t", RequiredLabel: ""}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestFormatSyncResultStable(t *testing.T) {
	var buf bytes.Buffer
	result := SyncResult{
		Owner:      "drawmeanelephant",
		Repo:       "la-famille",
		BaseBranch: "master",
		Apply:      false,
		Decisions: []PRDecision{
			{Number: 412, Action: ActionSkip, Reason: `missing required label "litterbox-approved"`},
			{Number: 414, Action: ActionWouldMerge, Reason: "all policy gates passed"},
		},
		Inspected:  2,
		Skipped:    1,
		WouldMerge: 1,
	}
	if err := FormatSyncResult(&buf, result); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "🐙 Clear the Litterbox") {
		t.Error("missing header")
	}
	if !strings.Contains(out, "Mode: dry-run") {
		t.Error("missing dry-run mode")
	}
	if !strings.Contains(out, "WOULD_MERGE") {
		t.Error("missing WOULD_MERGE")
	}
	if !strings.Contains(out, "inspected=2 skipped=1 would_merge=1 merged=0 would_close=0 closed=0") {
		t.Errorf("summary line wrong:\n%s", out)
	}
}

func TestDefaultBranchHelperRemoved(t *testing.T) {
	// Ensure we no longer hardcode master via a package helper for empty base;
	// resolution is API-driven. This test documents intentional removal.
	cfg := SyncConfig{
		Token:         "t",
		RequiredLabel: DefaultRequiredLabel,
		Client:        &fakeClient{defaultBranch: "from-api"},
		Git:           &fakeGit{},
		Owner:         "o",
		Repo:          "r",
		Sleep:         func(time.Duration) {},
		Now:           time.Now,
	}
	result, err := RunSync(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if result.BaseBranch != "from-api" {
		t.Errorf("got %q", result.BaseBranch)
	}
}
