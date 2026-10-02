package main

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/pack"
)

func setupPackCmd(cfg config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use: "pack", Short: "Build, verify, diff, apply, and pull content-only Corpus Packs",
	}
	var outputDir, ragDir, destination string
	build := &cobra.Command{
		Use: "build --output <pack.tar>", Short: "Package existing public artifacts and rag-content.md",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			info := currentBuildInfo()
			provenance := pack.Provenance{
				Generator: "la-famille", Version: info.Version, Target: info.Target, GoVersion: info.GoVersion,
			}
			if info.Commit != "unknown" {
				provenance.Commit = info.Commit
			}
			if info.BuildDate != "unknown" {
				provenance.BuildDate = info.BuildDate
			}
			manifest, err := pack.Build(pack.BuildOptions{
				OutputDir:  resolveProjectPath(cfg.ProjectRoot, outputDir),
				RagDir:     resolveProjectPath(cfg.ProjectRoot, ragDir),
				Site:       pack.Site{Name: cfg.SiteName, URL: cfg.SiteURL},
				Provenance: provenance,
			}, resolveProjectPath(cfg.ProjectRoot, destination))
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Built pack %s: %d members, content root %s\n",
				destination, len(manifest.Members), manifest.ContentRoot)
			return err
		},
	}
	build.Flags().StringVar(&outputDir, "site-output", cfg.OutputDir, "Existing generated public directory")
	build.Flags().StringVar(&ragDir, "rag-dir", cfg.RagDir, "Existing RAG directory (only rag-content.md is included)")
	build.Flags().StringVarP(&destination, "output", "o", "", "New pack file, must not already exist")
	_ = build.MarkFlagRequired("output")
	verify := &cobra.Command{
		Use: "verify <pack.tar>", Short: "Check schema and member integrity without extraction",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			manifest, err := pack.VerifyFile(resolveProjectPath(cfg.ProjectRoot, args[0]))
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Verified pack %s: %d members, content root %s\n",
				args[0], len(manifest.Members), manifest.ContentRoot)
			return err
		},
	}
	cmd.AddCommand(build, verify, setupPackDiffCmd(cfg), setupPackApplyCmd(cfg), setupPackPullCmd(cfg))
	return cmd
}

func setupPackPullCmd(cfg config.Config) *cobra.Command {
	var base, destination string
	cmd := &cobra.Command{
		Use:   "pull <feed-directory> --output <new-pack.tar>",
		Short: "Pull a verified local feed, optionally updating an exact base by delta",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().Changed("base") && base == "" {
				return fmt.Errorf("--base requires an existing pack path")
			}
			if destination == "" {
				return fmt.Errorf("--output requires a new pack path")
			}
			result, err := pack.Pull(
				resolveProjectPath(cfg.ProjectRoot, args[0]),
				resolveProjectPath(cfg.ProjectRoot, base),
				resolveProjectPath(cfg.ProjectRoot, destination))
			if err != nil {
				return err
			}
			if result.Fallback {
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), "No matching delta for the exact base SHA256; using full-pack fallback."); err != nil {
					return err
				}
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(),
				"Pulled pack %s: mode %s, %d members\nTarget archive SHA256: %s\nTarget content root: %s\n",
				destination, result.Mode, len(result.Manifest.Members), result.TargetSHA256, result.Manifest.ContentRoot); err != nil {
				return err
			}
			if result.Changes != nil {
				_, err = fmt.Fprint(cmd.OutOrStdout(), result.Changes.Summary())
			}
			return err
		},
	}
	cmd.Flags().StringVar(&base, "base", "", "Existing verified base pack (never modified)")
	cmd.Flags().StringVarP(&destination, "output", "o", "", "New result pack file, must not already exist")
	_ = cmd.MarkFlagRequired("output")
	return cmd
}

func setupPackDiffCmd(cfg config.Config) *cobra.Command {
	var destination string
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "diff <before.tar> <after.tar> --output <delta.tar>",
		Short: "Compare verified packs and write a member-level delta",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			report, err := pack.Diff(
				resolveProjectPath(cfg.ProjectRoot, args[0]),
				resolveProjectPath(cfg.ProjectRoot, args[1]),
				resolveProjectPath(cfg.ProjectRoot, destination))
			if err != nil {
				return err
			}
			if jsonOutput {
				encoder := json.NewEncoder(cmd.OutOrStdout())
				encoder.SetIndent("", "  ")
				return encoder.Encode(report)
			}
			_, err = fmt.Fprint(cmd.OutOrStdout(), report.Summary())
			return err
		},
	}
	cmd.Flags().StringVarP(&destination, "output", "o", "", "New delta file, must not already exist")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Write the structured member and Ledger comparison as JSON")
	_ = cmd.MarkFlagRequired("output")
	return cmd
}

func setupPackApplyCmd(cfg config.Config) *cobra.Command {
	var destination string
	cmd := &cobra.Command{
		Use:   "apply <base.tar> <delta.tar> --output <result.tar>",
		Short: "Verify and apply a local delta into a new pack",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			manifest, err := pack.Apply(
				resolveProjectPath(cfg.ProjectRoot, args[0]),
				resolveProjectPath(cfg.ProjectRoot, args[1]),
				resolveProjectPath(cfg.ProjectRoot, destination))
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Applied pack %s: %d members, content root %s\n",
				destination, len(manifest.Members), manifest.ContentRoot)
			return err
		},
	}
	cmd.Flags().StringVarP(&destination, "output", "o", "", "New result pack file, must not already exist")
	_ = cmd.MarkFlagRequired("output")
	return cmd
}
