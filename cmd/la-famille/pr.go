package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/github"
)

// runPRSync is a test seam: replacing it lets a test capture the SyncConfig
// the command built (in particular which directory the git work targets)
// without a GitHub token or network access (#640).
var runPRSync = github.RunSync

func setupPrCmd(cfg config.Config) *cobra.Command {
	prCmd := &cobra.Command{
		Use:   "pr",
		Short: "Manage GitHub Pull Requests",
	}
	prCmd.AddCommand(setupPrSyncCmd(cfg))
	return prCmd
}

func setupPrSyncCmd(cfg config.Config) *cobra.Command {
	prSyncCmd := &cobra.Command{
		Use:   "sync",
		Short: "Automated PR management (clear the litterbox)",
		Long: `Clear the Litterbox — inspect open automation PRs and apply an explicit merge policy.

By default this command is a complete dry run: it prints a deterministic decision
for every relevant PR and performs no mutations.

Mutations require --apply. Merges require the configured required label
(default: litterbox-approved). Conflicts are reported but not closed unless
--close-conflicts is also supplied. Local working-tree publishing is disabled
unless --publish-local-changes is supplied (and still requires --apply).

All local git operations — including the origin lookup that decides which
repository is synced — run against --project-root (or the directory containing
the selected config.yaml), never the process working directory.

Zero check runs are not treated as passing unless --allow-no-checks is set.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			token := os.Getenv("GITHUB_TOKEN")
			if token == "" {
				return fmt.Errorf("GITHUB_TOKEN environment variable must be set")
			}

			baseBranch, err := cmd.Flags().GetString("base")
			if err != nil {
				return err
			}
			apply, err := cmd.Flags().GetBool("apply")
			if err != nil {
				return err
			}
			requiredLabel, err := cmd.Flags().GetString("required-label")
			if err != nil {
				return err
			}
			if requiredLabel == "" {
				return fmt.Errorf("--required-label must not be empty")
			}
			closeConflicts, err := cmd.Flags().GetBool("close-conflicts")
			if err != nil {
				return err
			}
			allowNoChecks, err := cmd.Flags().GetBool("allow-no-checks")
			if err != nil {
				return err
			}
			publishLocal, err := cmd.Flags().GetBool("publish-local-changes")
			if err != nil {
				return err
			}
			botAuthors, err := cmd.Flags().GetStringSlice("bot-author")
			if err != nil {
				return err
			}
			headPrefixes, err := cmd.Flags().GetStringSlice("head-prefix")
			if err != nil {
				return err
			}

			syncCfg := github.SyncConfig{
				Token:               token,
				BotAuthors:          botAuthors,
				BaseBranch:          baseBranch,
				RequiredLabel:       requiredLabel,
				HeadPrefixes:        headPrefixes,
				Apply:               apply,
				CloseConflicts:      closeConflicts,
				AllowNoChecks:       allowNoChecks,
				PublishLocalChanges: publishLocal,
				// The bootstrapper already resolved --project-root (or the
				// config.yaml directory) to an absolute path; binding the
				// sync's git work to it is what keeps `pr sync` off the CWD
				// repo (#640). Resolving the flag again here would nest a
				// relative root under itself.
				WorkDir: cfg.ProjectRoot,
			}

			result, syncErr := runPRSync(syncCfg)
			if formatErr := github.FormatSyncResult(cmd.OutOrStdout(), result); formatErr != nil {
				return formatErr
			}
			if syncErr != nil {
				return fmt.Errorf("sync failed: %w", syncErr)
			}
			return nil
		},
	}

	prSyncCmd.Flags().String("base", "", "Target base branch (empty: resolve repository default_branch via GitHub API)")
	prSyncCmd.Flags().Bool("apply", false, "Perform mutations (default is dry-run)")
	prSyncCmd.Flags().String("required-label", github.DefaultRequiredLabel, "Label required to merge or close a PR")
	prSyncCmd.Flags().Bool("close-conflicts", false, "Allow closing conflicting PRs that pass identity gates")
	prSyncCmd.Flags().Bool("allow-no-checks", false, "Treat PRs with zero check runs as eligible for merge")
	prSyncCmd.Flags().Bool("publish-local-changes", false, "Allow staging/committing/pushing local working-tree changes and opening a PR")
	prSyncCmd.Flags().StringSlice("bot-author", append([]string(nil), github.DefaultBotAuthors...), "Allowlisted bot author logins (repeatable)")
	prSyncCmd.Flags().StringSlice("head-prefix", nil, "Optional required head ref prefixes (repeatable)")
	return prSyncCmd
}
