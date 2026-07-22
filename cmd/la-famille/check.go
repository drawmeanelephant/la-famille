package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/tbuddy/la-famille/internal/checker"
	"github.com/tbuddy/la-famille/internal/config"
)

var (
	checkContentDir  string
	checkAssetDir    string
	checkAssetHealth bool
)

func setupCheckCmd(cfg config.Config) *cobra.Command {
	var checkCmd = &cobra.Command{
		Use:   "check",
		Short: "Validate frontmatter, dates, tags, slugs, internal markdown links, and optional asset health",
		RunE: func(cmd *cobra.Command, _ []string) error {
			checkCfg := cfg
			if checkContentDir != "" {
				checkCfg.ContentDir = checkContentDir
			}
			if checkAssetDir != "" {
				checkCfg.AssetDir = checkAssetDir
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

			for _, finding := range res.Findings {
				if finding.Level == checker.LevelError {
					fmt.Fprintln(errOut, finding.String())
				} else {
					fmt.Fprintln(out, finding.String())
				}
			}

			if res.ErrorCount() > 0 {
				return fmt.Errorf("content validation failed with %d error(s)", res.ErrorCount())
			}

			if len(res.Findings) == 0 {
				fmt.Fprintln(out, "All content validation checks passed.")
			}

			return nil
		},
	}

	checkCmd.Flags().StringVarP(&checkContentDir, "content", "c", cfg.ContentDir, "Directory containing markdown files")
	checkCmd.Flags().StringVarP(&checkAssetDir, "asset", "a", cfg.AssetDir, "Directory containing static asset files")
	checkCmd.Flags().BoolVar(&checkAssetHealth, "asset-health", cfg.CheckAssetHealth, "Enable asset health diagnostics")
	return checkCmd
}
