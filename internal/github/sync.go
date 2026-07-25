package github

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/tbuddy/la-famille/internal/git"
)

// DefaultBotAuthors is the default allowlist for automated PR authors.
var DefaultBotAuthors = []string{"google-labs-jules", "google-labs-code"}

// DefaultRequiredLabel is the mandatory merge/close label.
const DefaultRequiredLabel = "litterbox-approved"

// LocalAction describes what the sync did (or would do) with local working-tree changes.
type LocalAction string

const (
	LocalNone           LocalAction = ""
	LocalWouldCreatePR  LocalAction = "would_create_pr"
	LocalCreatedPR      LocalAction = "created_pr"
	LocalPresentNotAuth LocalAction = "present_not_authorized"
	LocalNoChanges      LocalAction = "no_changes"
)

// SyncConfig holds configuration for the PR sync process.
type SyncConfig struct {
	Token               string
	BotAuthors          []string
	BaseBranch          string // empty: resolve repository default_branch via API
	RequiredLabel       string
	HeadPrefixes        []string
	Apply               bool
	CloseConflicts      bool
	AllowNoChecks       bool
	PublishLocalChanges bool

	// Optional injection for tests / advanced callers.
	Client APIClient
	Git    GitRunner
	Now    func() time.Time
	Sleep  func(time.Duration)
	Owner  string
	Repo   string
}

// GitRunner abstracts git operations used by local-change publishing.
type GitRunner interface {
	HasUncommittedChanges() (bool, error)
	GetRemoteURL(remote string) (string, error)
	CurrentBranch() (string, error)
	CheckoutNewBranch(branchName string) error
	Checkout(branchName string) error
	AddAll() error
	Commit(message, authorName, authorEmail string) error
	Push(remote, branchName string) error
}

type realGit struct{}

func (realGit) HasUncommittedChanges() (bool, error) { return git.HasUncommittedChanges() }
func (realGit) GetRemoteURL(remote string) (string, error) {
	return git.GetRemoteURL(remote)
}
func (realGit) CurrentBranch() (string, error)            { return git.CurrentBranch() }
func (realGit) CheckoutNewBranch(branchName string) error { return git.CheckoutBranch(branchName) }
func (realGit) Checkout(branchName string) error          { return git.Checkout(branchName) }
func (realGit) AddAll() error                             { return git.AddAll() }
func (realGit) Commit(message, authorName, authorEmail string) error {
	return git.Commit(message, authorName, authorEmail)
}
func (realGit) Push(remote, branchName string) error { return git.Push(remote, branchName) }

// SyncResult is the structured outcome of a litterbox run.
type SyncResult struct {
	Owner       string
	Repo        string
	BaseBranch  string
	Apply       bool
	Decisions   []PRDecision
	Inspected   int
	Skipped     int
	WouldMerge  int
	Merged      int
	WouldClose  int
	Closed      int
	LocalAction LocalAction
	LocalReason string
	PRErrors    []error
}

// Validate checks configuration before any mutations.
func (cfg SyncConfig) Validate() error {
	if strings.TrimSpace(cfg.Token) == "" && cfg.Client == nil {
		return fmt.Errorf("GITHUB_TOKEN is not set")
	}
	if strings.TrimSpace(cfg.RequiredLabel) == "" {
		return fmt.Errorf("--required-label must not be empty")
	}
	return nil
}

// RunSync executes the litterbox policy engine.
// Policy skips are not errors. Operational failures are returned (possibly joined)
// while still populating SyncResult for decisions already made.
func RunSync(cfg SyncConfig) (SyncResult, error) {
	result := SyncResult{Apply: cfg.Apply}

	if err := cfg.Validate(); err != nil {
		return result, err
	}

	if cfg.RequiredLabel == "" {
		cfg.RequiredLabel = DefaultRequiredLabel
	}
	if len(cfg.BotAuthors) == 0 {
		cfg.BotAuthors = append([]string(nil), DefaultBotAuthors...)
	}
	if cfg.Git == nil {
		cfg.Git = realGit{}
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.Sleep == nil {
		cfg.Sleep = time.Sleep
	}

	owner, repo, client, err := resolveClient(cfg)
	if err != nil {
		return result, err
	}
	result.Owner = owner
	result.Repo = repo

	baseBranch := strings.TrimSpace(cfg.BaseBranch)
	if baseBranch == "" {
		baseBranch, err = client.GetDefaultBranch()
		if err != nil {
			return result, fmt.Errorf("failed to resolve repository default branch: %w", err)
		}
	}
	result.BaseBranch = baseBranch
	slog.Info("Starting litterbox sync", "owner", owner, "repo", repo, "base", baseBranch, "apply", cfg.Apply)

	prs, err := client.ListOpenPRs(cfg.BotAuthors, baseBranch)
	if err != nil {
		return result, fmt.Errorf("failed to list PRs: %w", err)
	}
	result.Inspected = len(prs)
	slog.Info("Found open bot-authored PRs", "count", len(prs))

	policy := PolicyConfig{
		BaseBranch:     baseBranch,
		RequiredLabel:  cfg.RequiredLabel,
		BotAuthors:     cfg.BotAuthors,
		HeadPrefixes:   cfg.HeadPrefixes,
		Apply:          cfg.Apply,
		CloseConflicts: cfg.CloseConflicts,
		AllowNoChecks:  cfg.AllowNoChecks,
	}

	for _, listed := range prs {
		decision, opErr := evaluateAndMaybeMutate(client, listed, policy)
		result.Decisions = append(result.Decisions, decision)
		tallyDecision(&result, decision)
		if opErr != nil {
			result.PRErrors = append(result.PRErrors, opErr)
			slog.Error("Operational error for PR", "pr", listed.Number, "error", opErr)
		}
	}

	if err := handleLocalChanges(cfg, client, baseBranch, &result); err != nil {
		result.PRErrors = append(result.PRErrors, err)
	}

	if len(result.PRErrors) > 0 {
		return result, errors.Join(result.PRErrors...)
	}
	return result, nil
}

func resolveClient(cfg SyncConfig) (owner, repo string, client APIClient, err error) {
	if cfg.Client != nil {
		owner = cfg.Owner
		repo = cfg.Repo
		if owner == "" || repo == "" {
			return "", "", nil, fmt.Errorf("Owner and Repo must be set when injecting Client")
		}
		return owner, repo, cfg.Client, nil
	}

	remoteURL, err := cfg.Git.GetRemoteURL("origin")
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to get git remote url: %w", err)
	}
	owner, repo, err = git.ParseOwnerRepo(remoteURL)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to parse owner/repo from remote URL %s: %w", remoteURL, err)
	}
	return owner, repo, NewClient(cfg.Token, owner, repo), nil
}

func evaluateAndMaybeMutate(client APIClient, listed PullRequest, policy PolicyConfig) (PRDecision, error) {
	fullPR, err := client.GetPR(listed.Number)
	if err != nil {
		return PRDecision{
			Number: listed.Number,
			Action: ActionSkip,
			Reason: "failed to fetch PR details",
		}, fmt.Errorf("PR #%d: failed to get details: %w", listed.Number, err)
	}

	checks, err := client.GetCheckSummary(fullPR.Head.Sha)
	if err != nil {
		return PRDecision{
			Number: fullPR.Number,
			Action: ActionSkip,
			Reason: "failed to fetch check runs",
		}, fmt.Errorf("PR #%d: failed to get check runs: %w", fullPR.Number, err)
	}

	state := PRState{
		Number:    fullPR.Number,
		Title:     fullPR.Title,
		Author:    fullPR.User.Login,
		Draft:     fullPR.Draft,
		Labels:    fullPR.LabelNames(),
		HeadRef:   fullPR.Head.Ref,
		HeadSHA:   fullPR.Head.Sha,
		BaseRef:   fullPR.Base.Ref,
		Mergeable: fullPR.Mergeable,
		Checks:    checks,
	}
	decision := EvaluatePR(state, policy)

	switch decision.Action {
	case ActionMerge:
		if err := client.MergePR(fullPR.Number, fullPR.Head.Sha); err != nil {
			decision.Action = ActionSkip
			decision.Reason = "merge request failed"
			return decision, fmt.Errorf("PR #%d: merge failed: %w", fullPR.Number, err)
		}
		decision.Reason = "squash merge completed at expected head SHA"
		slog.Info("Merged PR", "pr", fullPR.Number, "sha", fullPR.Head.Sha)
	case ActionClose:
		if err := client.ClosePR(fullPR.Number); err != nil {
			decision.Action = ActionSkip
			decision.Reason = "close request failed"
			return decision, fmt.Errorf("PR #%d: close failed: %w", fullPR.Number, err)
		}
		decision.Reason = "conflict detected"
		slog.Info("Closed conflicting PR", "pr", fullPR.Number)
	}

	return decision, nil
}

func tallyDecision(result *SyncResult, d PRDecision) {
	switch d.Action {
	case ActionSkip:
		result.Skipped++
	case ActionWouldMerge:
		result.WouldMerge++
	case ActionMerge:
		result.Merged++
	case ActionWouldClose:
		result.WouldClose++
	case ActionClose:
		result.Closed++
	}
}

func handleLocalChanges(cfg SyncConfig, client APIClient, baseBranch string, result *SyncResult) error {
	hasChanges, err := cfg.Git.HasUncommittedChanges()
	if err != nil {
		return fmt.Errorf("failed to check for uncommitted changes: %w", err)
	}
	if !hasChanges {
		result.LocalAction = LocalNoChanges
		result.LocalReason = "no local uncommitted changes"
		return nil
	}

	if !cfg.PublishLocalChanges {
		result.LocalAction = LocalPresentNotAuth
		result.LocalReason = "local changes present; use --publish-local-changes to authorize publishing"
		slog.Info(result.LocalReason)
		return nil
	}

	timestamp := cfg.Now().UTC().Format("20060102150405")
	branchName := fmt.Sprintf("jules-auto-%s", timestamp)
	prTitle := fmt.Sprintf("Automated Routine Execution: %s", timestamp)
	prBody := "This PR was generated automatically by the la-famille GitHub sync feature to commit routine artifacts.\n\n" +
		"Note: this PR does not receive the litterbox-approved label automatically. " +
		"A human or separate trusted automation must apply that label before merge."

	if !cfg.Apply {
		result.LocalAction = LocalWouldCreatePR
		result.LocalReason = fmt.Sprintf("would create branch %s and open PR against %s (stages all working-tree changes)", branchName, baseBranch)
		slog.Info("Dry-run local publish", "branch", branchName, "base", baseBranch)
		return nil
	}

	originalBranch, origErr := cfg.Git.CurrentBranch()
	if origErr != nil {
		slog.Warn("Could not determine current branch for restore", "error", origErr)
	}

	restore := func() {
		if originalBranch == "" {
			return
		}
		if err := cfg.Git.Checkout(originalBranch); err != nil {
			slog.Warn("Failed to restore original branch", "branch", originalBranch, "error", err)
		}
	}

	if err := cfg.Git.CheckoutNewBranch(branchName); err != nil {
		return fmt.Errorf("failed to checkout branch: %w", err)
	}

	if err := cfg.Git.AddAll(); err != nil {
		restore()
		return fmt.Errorf("failed to stage changes: %w", err)
	}

	commitMsg := "chore: automated routine execution"
	if err := cfg.Git.Commit(commitMsg, "google-labs-jules", "jules-bot@users.noreply.github.com"); err != nil {
		restore()
		return fmt.Errorf("failed to commit: %w", err)
	}

	slog.Info("Pushing branch", "branch", branchName)
	if err := cfg.Git.Push("origin", branchName); err != nil {
		restore()
		return fmt.Errorf("failed to push: %w", err)
	}

	maxAttempts := 5
	backoff := 2 * time.Second
	var errPR error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		cfg.Sleep(backoff)
		errPR = client.CreatePR(prTitle, prBody, branchName, baseBranch)
		if errPR == nil {
			break
		}
		slog.Warn("Attempt to create PR failed. Retrying.", "attempt", attempt, "error", errPR, "retry_in", backoff*2)
		backoff *= 2
	}

	restore()

	if errPR != nil {
		return fmt.Errorf("failed to create PR after %d attempts: %w", maxAttempts, errPR)
	}

	result.LocalAction = LocalCreatedPR
	result.LocalReason = fmt.Sprintf("created branch %s and opened PR against %s", branchName, baseBranch)
	slog.Info("Successfully created PR for branch", "branch", branchName)
	return nil
}

// FormatSyncResult writes the stable human-readable litterbox summary.
func FormatSyncResult(w io.Writer, result SyncResult) error {
	mode := "dry-run"
	if result.Apply {
		mode = "apply"
	}
	if _, err := fmt.Fprintf(w, "🐙 Clear the Litterbox\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Repository: %s/%s\n", result.Owner, result.Repo); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Base: %s\n", result.BaseBranch); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Mode: %s\n\n", mode); err != nil {
		return err
	}

	for _, d := range result.Decisions {
		action := DisplayAction(d.Action)
		if _, err := fmt.Fprintf(w, "PR #%-6d %-12s %s\n", d.Number, action, d.Reason); err != nil {
			return err
		}
	}

	if len(result.Decisions) > 0 {
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(w,
		"Summary: inspected=%d skipped=%d would_merge=%d merged=%d would_close=%d closed=%d\n",
		result.Inspected, result.Skipped, result.WouldMerge, result.Merged, result.WouldClose, result.Closed,
	); err != nil {
		return err
	}

	switch result.LocalAction {
	case LocalWouldCreatePR:
		if _, err := fmt.Fprintf(w, "Local: WOULD_CREATE_PR  %s\n", result.LocalReason); err != nil {
			return err
		}
	case LocalCreatedPR:
		if _, err := fmt.Fprintf(w, "Local: CREATED_PR  %s\n", result.LocalReason); err != nil {
			return err
		}
	case LocalPresentNotAuth:
		if _, err := fmt.Fprintf(w, "Local: SKIP  %s\n", result.LocalReason); err != nil {
			return err
		}
	}

	return nil
}
