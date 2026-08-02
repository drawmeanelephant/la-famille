package main

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/publisher"
)

func setupPublishCheckCmd(cfg config.Config) *cobra.Command {
	var output string
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "publish-check",
		Short: "Validate the static publish artifact and emit its file manifest",
		RunE: func(cmd *cobra.Command, _ []string) error {
			manifest, err := publisher.Check(resolveProjectPath(cfg.ProjectRoot, output))
			if err != nil {
				return err
			}
			if asJSON {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(manifest)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Publish artifact is valid: %d files\n", len(manifest.Files))
			for _, file := range manifest.Files {
				fmt.Fprintln(cmd.OutOrStdout(), file)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&output, "output", "o", cfg.OutputDir, "Directory containing the generated publish artifact")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Emit the manifest as JSON")
	return cmd
}
