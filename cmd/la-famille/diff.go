package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tbuddy/la-famille/internal/config"
	sitediff "github.com/tbuddy/la-famille/internal/diff"
)

func setupDiffCmd(cfg config.Config) *cobra.Command {
	var jsonOutput bool
	var gate bool
	var reportDir string
	cmd := &cobra.Command{
		Use:   "diff <a> <b>",
		Short: "Compare two site manifests",
		Long: strings.TrimSpace(`Compare two built sites using their site manifests.

Each input can be an output directory, a site-manifest.json file, or a Git
revision. Git revisions use a committed manifest when available, otherwise
build an isolated source snapshot. Use ref:<revision> to force interpretation when
the name also exists as a local path.`),
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			before, err := loadDiffInput(args[0], cfg)
			if err != nil {
				return fmt.Errorf("load first diff input: %w", err)
			}
			after, err := loadDiffInput(args[1], cfg)
			if err != nil {
				return fmt.Errorf("load second diff input: %w", err)
			}
			report, err := sitediff.Compare(before, after)
			if err != nil {
				return err
			}
			if reportDir != "" {
				if err := sitediff.Write(resolveProjectPath(cfg.ProjectRoot, reportDir),
					sitediff.Ledger{Version: 1, Changes: report}); err != nil {
					return err
				}
			}

			if jsonOutput {
				output := struct {
					Before  string          `json:"before"`
					After   string          `json:"after"`
					Changes sitediff.Report `json:"changes"`
				}{
					Before:  args[0],
					After:   args[1],
					Changes: report,
				}
				encoder := json.NewEncoder(cmd.OutOrStdout())
				encoder.SetIndent("", "  ")
				if err := encoder.Encode(output); err != nil {
					return err
				}
			} else {
				if _, err := fmt.Fprint(cmd.OutOrStdout(), report.Summary(args[0], args[1])); err != nil {
					return err
				}
			}
			if gate {
				return report.GateError()
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Write the structured diff as JSON")
	cmd.Flags().BoolVar(&gate, "gate", false, "Fail on regressions (requires complete v2 snapshots)")
	cmd.Flags().StringVar(&reportDir, "report-dir", "", "Save diff.json and diff.txt to this directory")
	return cmd
}
