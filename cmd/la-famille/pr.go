package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tbuddy/la-famille/internal/github"
)

var prCmd = &cobra.Command{
	Use:   "pr",
	Short: "Manage GitHub Pull Requests",
}

var prSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Automated PR management (clear the litterbox)",
	Long: `Fetches open pull requests by automation agents.
Closes stale/conflicting PRs and merges passing ones.
If there are local uncommitted changes, branches and creates a new PR.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		token := os.Getenv("GITHUB_TOKEN")
		if token == "" {
			return fmt.Errorf("GITHUB_TOKEN environment variable must be set")
		}

		baseBranch, _ := cmd.Flags().GetString("base")

		cfg := github.SyncConfig{
			Token:         token,
			BotAuthors:    []string{"google-labs-jules", "google-labs-code"},
			DefaultBranch: baseBranch,
		}

		fmt.Println("Starting automated PR sync...")
		if err := github.RunSync(cfg); err != nil {
			return fmt.Errorf("sync failed: %w", err)
		}
		fmt.Println("Sync completed successfully.")
		return nil
	},
}

func init() {
	prSyncCmd.Flags().String("base", "main", "The base branch to target for new PRs")
	prCmd.AddCommand(prSyncCmd)
	// We need to add prCmd to rootCmd in main.go
}
