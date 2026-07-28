package github

import (
	"testing"
)

func eligiblePR() PRState {
	mergeable := true
	return PRState{
		Number:    414,
		Title:     "Eligible bot PR",
		Author:    "google-labs-jules",
		Draft:     false,
		Labels:    []string{"litterbox-approved"},
		HeadRef:   "jules/feature",
		HeadSHA:   "abc123",
		BaseRef:   "master",
		Mergeable: &mergeable,
		Checks:    CheckSummary{State: CheckStatePassing, Total: 2},
	}
}

func basePolicy() PolicyConfig {
	return PolicyConfig{
		BaseBranch:    "master",
		RequiredLabel: "litterbox-approved",
		BotAuthors:    []string{"google-labs-jules", "google-labs-code"},
		Apply:         false,
	}
}

func TestEvaluatePR(t *testing.T) {
	mergeableTrue := true
	mergeableFalse := false

	tests := []struct {
		name   string
		action PRAction
		reason string
		pr     PRState
		cfg    PolicyConfig
	}{
		{
			name:   "1 dry-run eligible would merge",
			pr:     eligiblePR(),
			cfg:    basePolicy(),
			action: ActionWouldMerge,
			reason: "all policy gates passed",
		},
		{
			name: "1b apply eligible merge",
			pr:   eligiblePR(),
			cfg: func() PolicyConfig {
				c := basePolicy()
				c.Apply = true
				return c
			}(),
			action: ActionMerge,
			reason: "all policy gates passed",
		},
		{
			name: "2 author not allowlisted",
			pr: func() PRState {
				p := eligiblePR()
				p.Author = "random-human"
				return p
			}(),
			cfg:    basePolicy(),
			action: ActionSkip,
			reason: `author "random-human" not on bot allowlist`,
		},
		{
			name: "3 base branch mismatch",
			pr: func() PRState {
				p := eligiblePR()
				p.BaseRef = "develop"
				return p
			}(),
			cfg:    basePolicy(),
			action: ActionSkip,
			reason: `base branch "develop" does not match target "master"`,
		},
		{
			name: "4 draft PR",
			pr: func() PRState {
				p := eligiblePR()
				p.Draft = true
				return p
			}(),
			cfg:    basePolicy(),
			action: ActionSkip,
			reason: "draft PR",
		},
		{
			name: "5 required label missing",
			pr: func() PRState {
				p := eligiblePR()
				p.Labels = []string{"other"}
				return p
			}(),
			cfg:    basePolicy(),
			action: ActionSkip,
			reason: `missing required label "litterbox-approved"`,
		},
		{
			name: "6 required label case-insensitive",
			pr: func() PRState {
				p := eligiblePR()
				p.Labels = []string{"Litterbox-Approved"}
				return p
			}(),
			cfg:    basePolicy(),
			action: ActionWouldMerge,
			reason: "all policy gates passed",
		},
		{
			name: "7 head prefix matches",
			pr:   eligiblePR(),
			cfg: func() PolicyConfig {
				c := basePolicy()
				c.HeadPrefixes = []string{"jules/"}
				return c
			}(),
			action: ActionWouldMerge,
			reason: "all policy gates passed",
		},
		{
			name: "8 head prefix does not match",
			pr:   eligiblePR(),
			cfg: func() PolicyConfig {
				c := basePolicy()
				c.HeadPrefixes = []string{"codex/"}
				return c
			}(),
			action: ActionSkip,
			reason: `head ref "jules/feature" does not match configured prefixes`,
		},
		{
			name: "9 mergeable null",
			pr: func() PRState {
				p := eligiblePR()
				p.Mergeable = nil
				return p
			}(),
			cfg:    basePolicy(),
			action: ActionSkip,
			reason: "mergeable status still computing",
		},
		{
			name: "10 conflict without close-conflicts",
			pr: func() PRState {
				p := eligiblePR()
				p.Mergeable = &mergeableFalse
				return p
			}(),
			cfg:    basePolicy(),
			action: ActionSkip,
			reason: "conflict detected; use --close-conflicts to authorize closure",
		},
		{
			name: "11 conflict with close-conflicts dry-run",
			pr: func() PRState {
				p := eligiblePR()
				p.Mergeable = &mergeableFalse
				return p
			}(),
			cfg: func() PolicyConfig {
				c := basePolicy()
				c.CloseConflicts = true
				return c
			}(),
			action: ActionWouldClose,
			reason: "conflict detected",
		},
		{
			name: "12 conflict with close-conflicts apply",
			pr: func() PRState {
				p := eligiblePR()
				p.Mergeable = &mergeableFalse
				return p
			}(),
			cfg: func() PolicyConfig {
				c := basePolicy()
				c.CloseConflicts = true
				c.Apply = true
				return c
			}(),
			action: ActionClose,
			reason: "conflict detected",
		},
		{
			name: "13 checks pending",
			pr: func() PRState {
				p := eligiblePR()
				p.Checks = CheckSummary{State: CheckStatePending, Total: 1}
				return p
			}(),
			cfg:    basePolicy(),
			action: ActionSkip,
			reason: "checks pending",
		},
		{
			name: "14 checks failed",
			pr: func() PRState {
				p := eligiblePR()
				p.Checks = CheckSummary{State: CheckStateFailed, Total: 1}
				return p
			}(),
			cfg:    basePolicy(),
			action: ActionSkip,
			reason: "checks failed",
		},
		{
			name: "15 no checks default policy",
			pr: func() PRState {
				p := eligiblePR()
				p.Checks = CheckSummary{State: CheckStateNone, Total: 0}
				return p
			}(),
			cfg:    basePolicy(),
			action: ActionSkip,
			reason: "no checks reported",
		},
		{
			name: "16 no checks with allow-no-checks",
			pr: func() PRState {
				p := eligiblePR()
				p.Checks = CheckSummary{State: CheckStateNone, Total: 0}
				p.Mergeable = &mergeableTrue
				return p
			}(),
			cfg: func() PolicyConfig {
				c := basePolicy()
				c.AllowNoChecks = true
				return c
			}(),
			action: ActionWouldMerge,
			reason: "all policy gates passed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EvaluatePR(tt.pr, tt.cfg)
			if got.Action != tt.action {
				t.Errorf("Action = %q, want %q", got.Action, tt.action)
			}
			if got.Reason != tt.reason {
				t.Errorf("Reason = %q, want %q", got.Reason, tt.reason)
			}
			if got.Number != tt.pr.Number {
				t.Errorf("Number = %d, want %d", got.Number, tt.pr.Number)
			}
		})
	}
}

func TestAuthorAllowlistCaseInsensitive(t *testing.T) {
	pr := eligiblePR()
	pr.Author = "Google-Labs-Jules"
	got := EvaluatePR(pr, basePolicy())
	if got.Action != ActionWouldMerge {
		t.Fatalf("expected would_merge for case-insensitive author, got %s (%s)", got.Action, got.Reason)
	}
}

func TestConflictCloseStillRequiresLabel(t *testing.T) {
	mergeableFalse := false
	pr := eligiblePR()
	pr.Mergeable = &mergeableFalse
	pr.Labels = nil
	cfg := basePolicy()
	cfg.CloseConflicts = true
	cfg.Apply = true
	got := EvaluatePR(pr, cfg)
	if got.Action != ActionSkip {
		t.Fatalf("expected skip without label, got %s", got.Action)
	}
}
