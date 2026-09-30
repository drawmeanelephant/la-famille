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
	cmd := &cobra.Command{
		Use:   "diff <a> <b>",
		Short: "Compare two site manifests",
		Long: strings.TrimSpace(`Compare two built sites using their site manifests.

Each input can be an output directory, a site-manifest.json file, or a Git
revision. Git revisions are read from <output_dir>/site-manifest.json in the
selected project root. Use ref:<revision> to force Git-ref interpretation when
the name also exists as a local path.`),
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			before, err := sitediff.LoadInput(args[0], cfg.OutputDir, cfg.ProjectRoot)
			if err != nil {
				return fmt.Errorf("load first diff input: %w", err)
			}
			after, err := sitediff.LoadInput(args[1], cfg.OutputDir, cfg.ProjectRoot)
			if err != nil {
				return fmt.Errorf("load second diff input: %w", err)
			}
			report, err := sitediff.Compare(before, after)
			if err != nil {
				return err
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
				return encoder.Encode(output)
			}

			_, err = fmt.Fprint(cmd.OutOrStdout(), report.Summary(args[0], args[1]))
			return err
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Write the structured diff as JSON")
	return cmd
}
