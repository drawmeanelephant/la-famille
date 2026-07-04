package github

import (
	"fmt"
	"log"

	"time"

	"github.com/tbuddy/la-famille/internal/git"
)

// SyncConfig holds configuration for the PR sync process.
type SyncConfig struct {
	Token         string
	BotAuthors    []string
	DefaultBranch string
}

// RunSync executes the automated PR management routine.
func RunSync(cfg SyncConfig) error {
	if cfg.Token == "" {
		return fmt.Errorf("GITHUB_TOKEN is not set")
	}

	// 1. Infer owner/repo from git config
	remoteURL, err := git.GetRemoteURL("origin")
	if err != nil {
		return fmt.Errorf("failed to get git remote url: %w", err)
	}

	owner, repo, err := git.ParseOwnerRepo(remoteURL)
	if err != nil {
		return fmt.Errorf("failed to parse owner/repo from remote URL %s: %w", remoteURL, err)
	}

	client := NewClient(cfg.Token, owner, repo)
	log.Printf("Starting sync for %s/%s", owner, repo)

	// 2. Fetch and process existing PRs
	prs, err := client.ListOpenPRs(cfg.BotAuthors)
	if err != nil {
		return fmt.Errorf("failed to list PRs: %w", err)
	}

	log.Printf("Found %d open PRs authored by bots", len(prs))

	for _, pr := range prs {
		// We need to fetch the PR individually to reliably get the `mergeable` status.
		// The list endpoint sometimes omits it or caches old values.
		fullPR, err := client.GetPR(pr.Number)
		if err != nil {
			log.Printf("Failed to get details for PR #%d: %v", pr.Number, err)
			continue
		}

		if fullPR.Mergeable == nil {
			log.Printf("PR #%d mergeable status is computing (null), skipping for now", pr.Number)
			continue
		}

		if !*fullPR.Mergeable {
			log.Printf("PR #%d has conflicts (mergeable=false), closing", pr.Number)
			if err := client.ClosePR(pr.Number); err != nil {
				log.Printf("Failed to close PR #%d: %v", pr.Number, err)
			} else {
				log.Printf("Successfully closed PR #%d", pr.Number)
			}
			continue
		}

		// PR is mergeable, check CI status
		passing, err := client.AreChecksPassing(fullPR.Head.Sha)
		if err != nil {
			log.Printf("Failed to get check runs for PR #%d (sha: %s): %v", pr.Number, fullPR.Head.Sha, err)
			continue
		}

		if passing {
			log.Printf("PR #%d checks are passing and mergeable=true, merging", pr.Number)
			if err := client.MergePR(pr.Number); err != nil {
				log.Printf("Failed to merge PR #%d: %v", pr.Number, err)
			} else {
				log.Printf("Successfully merged PR #%d", pr.Number)
			}
		} else {
			log.Printf("PR #%d checks are not yet fully passing, skipping", pr.Number)
		}
	}

	// 3. Handle local changes
	hasChanges, err := git.HasUncommittedChanges()
	if err != nil {
		return fmt.Errorf("failed to check for uncommitted changes: %w", err)
	}

	if !hasChanges {
		log.Println("No local changes detected. Sync complete.")
		return nil
	}

	log.Println("Local changes detected. Creating a new automated PR.")
	timestamp := time.Now().Format("20060102150405")
	branchName := fmt.Sprintf("jules-auto-%s", timestamp)

	if err := git.CheckoutBranch(branchName); err != nil {
		return fmt.Errorf("failed to checkout branch: %w", err)
	}

	if err := git.AddAll(); err != nil {
		return fmt.Errorf("failed to stage changes: %w", err)
	}

	commitMsg := "chore: automated routine execution"
	if err := git.Commit(commitMsg, "google-labs-jules", "jules-bot@users.noreply.github.com"); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	log.Printf("Pushing branch %s...", branchName)
	if err := git.Push("origin", branchName); err != nil {
		return fmt.Errorf("failed to push: %w", err)
	}

	prTitle := fmt.Sprintf("Automated Routine Execution: %s", timestamp)
	prBody := "This PR was generated automatically by the la-famille GitHub sync feature to commit routine artifacts."

	baseBranch := cfg.DefaultBranch
	if baseBranch == "" {
		baseBranch = "main" // Default fallback
	}

	maxAttempts := 5
	backoff := 2 * time.Second
	var errPR error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// Wait before each attempt to give GitHub time to register the branch
		time.Sleep(backoff)

		errPR = client.CreatePR(prTitle, prBody, branchName, baseBranch)
		if errPR == nil {
			break
		}

		log.Printf("Attempt %d to create PR failed: %v. Retrying in %v...", attempt, errPR, backoff*2)
		backoff *= 2
	}

	if errPR != nil {
		return fmt.Errorf("failed to create PR after %d attempts: %w", maxAttempts, errPR)
	}

	log.Printf("Successfully created PR for branch %s", branchName)

	// Switch back to original branch? Let's just stay here or we'd need to know what we were on.
	// For automation containers, it usually doesn't matter since it's transient.

	return nil
}
