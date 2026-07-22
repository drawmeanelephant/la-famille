package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/tbuddy/la-famille/internal/checker"
	"github.com/tbuddy/la-famille/internal/config"
)

var checkContentDir string

func setupCheckCmd(cfg config.Config) *cobra.Command {
	var checkCmd = &cobra.Command{
		Use:   "check",
		Short: "Validate frontmatter, dates, tags, slugs, and internal markdown links",
		RunE: func(cmd *cobra.Command, _ []string) error {
			checkCfg := cfg
			if checkContentDir != "" {
				checkCfg.ContentDir = checkContentDir
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
	return checkCmd
}
