package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/tbuddy/la-famille/internal/checker"
	"github.com/tbuddy/la-famille/internal/config"
)

var (
	checkContentDir  string
	checkAssetDir    string
	checkAssetHealth bool
	checkSummary     bool
)

func setupCheckCmd(cfg config.Config) *cobra.Command {
	var checkCmd = &cobra.Command{
		Use:   "check",
		Short: "Validate frontmatter, dates, tags, slugs, internal markdown links, and optional asset health",
		RunE: func(cmd *cobra.Command, _ []string) error {
			checkCfg := cfg
			if checkContentDir != "" {
				checkCfg.ContentDir = resolveProjectPath(cfg.ProjectRoot, checkContentDir)
			}
			if checkAssetDir != "" {
				checkCfg.AssetDir = resolveProjectPath(cfg.ProjectRoot, checkAssetDir)
			}
			if cmd.Flags().Changed("asset-health") {
				checkCfg.CheckAssetHealth = checkAssetHealth
			}

			res, err := checker.Validate(checkCfg)
			if err != nil {
				return fmt.Errorf("content check failed: %w", err)
			}

			out := cmd.OutOrStdout()
			errOut := cmd.ErrOrStderr()

			info := currentBuildInfo()
			fmt.Fprintf(out, "La Famille Diagnostics [%s (commit: %s)]\n", info.Version, info.Commit)

			for _, finding := range res.Findings {
				if finding.Level == checker.LevelError {
					fmt.Fprintln(errOut, finding.String())
				} else {
					fmt.Fprintln(out, finding.String())
				}
			}

			if checkSummary {
				fmt.Fprintln(out, formatCheckSummary(res))
			}

			if res.ErrorCount() > 0 {
				return fmt.Errorf("content validation failed with %d error(s)", res.ErrorCount())
			}

			if len(res.Findings) == 0 && !checkSummary {
				fmt.Fprintln(out, "All content validation checks passed.")
			}

			return nil
		},
	}

	checkCmd.Flags().StringVarP(&checkContentDir, "content", "c", cfg.ContentDir, "Directory containing markdown files")
	checkCmd.Flags().StringVarP(&checkAssetDir, "asset", "a", cfg.AssetDir, "Directory containing static asset files")
	checkCmd.Flags().BoolVar(&checkAssetHealth, "asset-health", cfg.CheckAssetHealth, "Enable asset health diagnostics")
	checkCmd.Flags().BoolVar(&checkSummary, "summary", true, "Show summary footer")
	return checkCmd
}

func formatCheckSummary(res *checker.Result) string {
	errors := res.ErrorCount()
	warnings := res.WarnCount()
	orphans := res.CountByCategory(checker.CategoryOrphan)

	missingDesc := 0
	missingDates := 0
	for _, f := range res.Findings {
		msgLower := strings.ToLower(f.Message)
		if strings.Contains(msgLower, "missing description") {
			missingDesc++
		}
		if strings.Contains(msgLower, "missing date") {
			missingDates++
		}
	}

	symbol := "✓"
	if warnings > 0 {
		// A clean bill of health keeps ✓; warnings get ⚠ so a CI log scan
		// reads pass/warn/fail from the leading symbol alone (#512).
		symbol = "⚠"
	}
	if errors > 0 {
		symbol = "✗"
	}

	return fmt.Sprintf("%s %d %s, %d %s | %d %s, %d %s, %d %s",
		symbol,
		errors, plural(errors, "error", "errors"),
		warnings, plural(warnings, "warning", "warnings"),
		orphans, plural(orphans, "orphaned page", "orphaned pages"),
		missingDesc, plural(missingDesc, "missing description", "missing descriptions"),
		missingDates, plural(missingDates, "missing date", "missing dates"),
	)
}

func plural(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}
