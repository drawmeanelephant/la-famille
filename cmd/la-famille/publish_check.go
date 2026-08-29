package main

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/publisher"
)

type publishCheckReport struct {
	Valid  bool     `json:"valid"`
	Files  []string `json:"files,omitempty"`
	Stubs  []string `json:"stubs,omitempty"`
	Errors []string `json:"errors,omitempty"`
}

func setupPublishCheckCmd(cfg config.Config) *cobra.Command {
	var output string
	var asJSON bool
	var strict bool
	var siteURLFlag string
	cmd := &cobra.Command{
		Use:   "publish-check",
		Short: "Validate the static publish artifact and emit its file manifest",
		Long: "Validate the static publish artifact and emit its file manifest.\n" +
			"Generated \"Missing Page\" stubs are reported as warnings; pass --strict\n" +
			"to fail on them so a typo'd internal link cannot reach deploy.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()
			// A project site (siteurl with a subpath, e.g. a GitHub Pages
			// project page) renders root-relative links as /repo/...; the check
			// needs that prefix to resolve them against the on-disk artifact.
			basePath := config.Config{SiteURL: siteURLFlag}.BasePath()
			manifest, err := publisher.Check(resolveProjectPath(cfg.ProjectRoot, output), basePath)

			if asJSON {
				report := publishCheckReport{
					Valid: err == nil,
					Files: manifest.Files,
					Stubs: manifest.Stubs,
				}
				var valErr *publisher.ValidationError
				switch {
				case errors.As(err, &valErr):
					report.Errors = valErr.Problems
				case err != nil:
					report.Valid = false
					report.Errors = []string{err.Error()}
				}
				if strict && len(report.Stubs) > 0 {
					for _, stub := range report.Stubs {
						report.Errors = append(report.Errors, fmt.Sprintf("generated Missing Page stub %q is present in the artifact", stub))
					}
					report.Valid = false
				}
				if encErr := json.NewEncoder(out).Encode(report); encErr != nil {
					return encErr
				}
				// The report above is the machine-readable result; returning the
				// error keeps the process exit code non-zero without printing a
				// second, plain-text copy of the failure over it.
				cmd.SilenceErrors = true
				if err != nil {
					return err
				}
				if strict && len(manifest.Stubs) > 0 {
					return fmt.Errorf("publish artifact contains %d generated Missing Page stub(s)", len(manifest.Stubs))
				}
				return nil
			}

			if err != nil {
				return err
			}
			fmt.Fprintf(out, "Publish artifact is valid: %d files\n", len(manifest.Files))
			for _, file := range manifest.Files {
				fmt.Fprintln(out, file)
			}
			for _, stub := range manifest.Stubs {
				fmt.Fprintf(out, "warning: %s is a generated \"Missing Page\" stub; fix or remove the internal link that produced it (use --strict to fail on this)\n", stub)
			}
			if strict && len(manifest.Stubs) > 0 {
				return fmt.Errorf("publish artifact contains %d generated Missing Page stub(s)", len(manifest.Stubs))
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&output, "output", "o", cfg.OutputDir, "Directory containing the generated publish artifact")
	cmd.Flags().StringVar(&siteURLFlag, "site-url", cfg.SiteURL, "Public base URL of the site (used to resolve base-path links in the artifact)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Emit the manifest as JSON")
	cmd.Flags().BoolVar(&strict, "strict", false, "Treat generated Missing Page stubs as validation failures")
	return cmd
}
