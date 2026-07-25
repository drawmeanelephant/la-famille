package github

import (
	"fmt"
	"strings"
)

// PRAction is the deterministic action decided for a pull request.
type PRAction string

const (
	ActionSkip       PRAction = "skip"
	ActionWouldMerge PRAction = "would_merge"
	ActionMerge      PRAction = "merge"
	ActionWouldClose PRAction = "would_close"
	ActionClose      PRAction = "close"
)

// PRDecision is the policy outcome for a single PR.
type PRDecision struct {
	Number int
	Action PRAction
	Reason string
}

// PolicyConfig is the explicit litterbox policy evaluated without side effects.
type PolicyConfig struct {
	BaseBranch     string
	RequiredLabel  string
	BotAuthors     []string
	HeadPrefixes   []string
	Apply          bool
	CloseConflicts bool
	AllowNoChecks  bool
}

// PRState is the snapshot of PR + check state used for pure policy evaluation.
type PRState struct {
	Number    int
	Title     string
	Author    string
	Draft     bool
	Labels    []string
	HeadRef   string
	HeadSHA   string
	BaseRef   string
	Mergeable *bool
	Checks    CheckSummary
}

// EvaluatePR decides what to do with a PR. It never performs I/O.
func EvaluatePR(pr PRState, cfg PolicyConfig) PRDecision {
	if !authorAllowed(pr.Author, cfg.BotAuthors) {
		return skip(pr.Number, fmt.Sprintf("author %q not on bot allowlist", pr.Author))
	}
	if pr.BaseRef != cfg.BaseBranch {
		return skip(pr.Number, fmt.Sprintf("base branch %q does not match target %q", pr.BaseRef, cfg.BaseBranch))
	}
	if pr.Draft {
		return skip(pr.Number, "draft PR")
	}
	if !hasRequiredLabel(pr.Labels, cfg.RequiredLabel) {
		return skip(pr.Number, fmt.Sprintf("missing required label %q", cfg.RequiredLabel))
	}
	if !headPrefixAllowed(pr.HeadRef, cfg.HeadPrefixes) {
		return skip(pr.Number, fmt.Sprintf("head ref %q does not match configured prefixes", pr.HeadRef))
	}

	if pr.Mergeable == nil {
		return skip(pr.Number, "mergeable status still computing")
	}

	if !*pr.Mergeable {
		if !cfg.CloseConflicts {
			return skip(pr.Number, "conflict detected; use --close-conflicts to authorize closure")
		}
		if cfg.Apply {
			return PRDecision{Number: pr.Number, Action: ActionClose, Reason: "conflict detected"}
		}
		return PRDecision{Number: pr.Number, Action: ActionWouldClose, Reason: "conflict detected"}
	}

	// Merge path: checks are required.
	switch pr.Checks.State {
	case CheckStateNone:
		if !cfg.AllowNoChecks {
			return skip(pr.Number, "no checks reported")
		}
	case CheckStatePending:
		return skip(pr.Number, "checks pending")
	case CheckStateFailed:
		return skip(pr.Number, "checks failed")
	case CheckStatePassing:
		// ok
	default:
		return skip(pr.Number, fmt.Sprintf("unknown check state %q", pr.Checks.State))
	}

	if cfg.Apply {
		return PRDecision{Number: pr.Number, Action: ActionMerge, Reason: "all policy gates passed"}
	}
	return PRDecision{Number: pr.Number, Action: ActionWouldMerge, Reason: "all policy gates passed"}
}

func skip(number int, reason string) PRDecision {
	return PRDecision{Number: number, Action: ActionSkip, Reason: reason}
}

func authorAllowed(author string, allowlist []string) bool {
	login := strings.ToLower(author)
	for _, a := range allowlist {
		if strings.ToLower(a) == login {
			return true
		}
	}
	return false
}

func hasRequiredLabel(labels []string, required string) bool {
	want := strings.ToLower(required)
	for _, l := range labels {
		if strings.ToLower(l) == want {
			return true
		}
	}
	return false
}

func headPrefixAllowed(headRef string, prefixes []string) bool {
	if len(prefixes) == 0 {
		return true
	}
	for _, p := range prefixes {
		if strings.HasPrefix(headRef, p) {
			return true
		}
	}
	return false
}

// DisplayAction returns the stable uppercase token used in CLI output.
func DisplayAction(a PRAction) string {
	switch a {
	case ActionSkip:
		return "SKIP"
	case ActionWouldMerge:
		return "WOULD_MERGE"
	case ActionMerge:
		return "MERGED"
	case ActionWouldClose:
		return "WOULD_CLOSE"
	case ActionClose:
		return "CLOSED"
	default:
		return strings.ToUpper(string(a))
	}
}
