package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/pack"
)

func packProvenance() pack.Provenance {
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
	return provenance
}

func packFeedSource(cfg config.Config, source string) string {
	if strings.Contains(source, "://") {
		return source
	}
	return resolveProjectPath(cfg.ProjectRoot, source)
}

func packTransferReporter(cmd *cobra.Command, enabled bool) func(pack.Transfer) {
	if !enabled {
		return nil
	}
	return func(transfer pack.Transfer) {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "HTTPS GET %s: status %d, %d body bytes\n",
			transfer.URL, transfer.Status, transfer.Bytes)
	}
}

func setupPackPublishCmd(cfg config.Config) *cobra.Command {
	var outputDir, ragDir, previous, destination string
	var retain int
	cmd := &cobra.Command{
		Use: "publish --output <new-feed-directory>", Short: "Build a canonical feed with bounded retained version and delta history",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if destination == "" {
				return fmt.Errorf("--output requires a new feed directory")
			}
			if retain < 1 || retain > 8 {
				return fmt.Errorf("--retain must be between 1 and 8")
			}
			if cmd.Flags().Changed("previous") && previous == "" {
				return fmt.Errorf("--previous requires a feed directory")
			}
			feed, err := pack.Publish(pack.PublishOptions{
				Build: pack.BuildOptions{
					OutputDir: resolveProjectPath(cfg.ProjectRoot, outputDir),
					RagDir:    resolveProjectPath(cfg.ProjectRoot, ragDir),
					Site:      pack.Site{Name: cfg.SiteName, URL: cfg.SiteURL}, Provenance: packProvenance(),
				},
				Previous: resolveProjectPath(cfg.ProjectRoot, previous), Retain: retain,
			}, resolveProjectPath(cfg.ProjectRoot, destination))
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Published feed %s: %d retained deltas\nTarget archive SHA256: %s\n",
				destination, len(feed.Deltas), feed.Full.SHA256)
			return err
		},
	}
	cmd.Flags().StringVar(&outputDir, "site-output", cfg.OutputDir, "Existing generated public directory")
	cmd.Flags().StringVar(&ragDir, "rag-dir", cfg.RagDir, "Existing RAG directory (only rag-content.md is included)")
	cmd.Flags().StringVar(&previous, "previous", "", "Retained previous feed directory; absent history publishes full only")
	cmd.Flags().IntVar(&retain, "retain", pack.DefaultRetainedVersions, "Total full versions to retain (1–8)")
	cmd.Flags().StringVarP(&destination, "output", "o", "", "New feed directory, must not already exist")
	_ = cmd.MarkFlagRequired("output")
	return cmd
}

func setupPackWatchCmd(cfg config.Config) *cobra.Command {
	var state string
	var interval, timeout time.Duration
	var allowHTTPS, traceHTTP bool
	cmd := &cobra.Command{
		Use:   "watch <feed-directory-or-HTTPS-manifest> --state <directory> --interval <duration>",
		Short: "Poll a feed and atomically advance durable verified subscriber state",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if state == "" || interval <= 0 || timeout <= 0 {
				return fmt.Errorf("watch requires --state, a positive --interval, and a positive --timeout")
			}
			source := packFeedSource(cfg, args[0])
			if strings.Contains(source, "://") && !allowHTTPS {
				return fmt.Errorf("remote feeds require explicit --allow-https")
			}
			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			var outputErr error
			returnErr := pack.Watch(ctx, pack.WatchOptions{
				Source: source, StateDir: resolveProjectPath(cfg.ProjectRoot, state), Interval: interval,
				Remote: pack.RemoteOptions{AllowHTTPS: allowHTTPS, Timeout: timeout, OnTransfer: packTransferReporter(cmd, traceHTTP)},
			}, func(event pack.WatchEvent) {
				if outputErr != nil {
					return
				}
				if event.Err != nil {
					_, outputErr = fmt.Fprintf(cmd.ErrOrStderr(), "Pack poll failed; current version preserved: %v\n", event.Err)
				} else if !event.Unchanged {
					_, outputErr = fmt.Fprintf(cmd.OutOrStdout(), "Current pack: %s\nMode: %s\nTarget archive SHA256: %s\n",
						event.Current, event.Result.Mode, event.Result.TargetSHA256)
					if outputErr == nil && event.Result.Fallback {
						_, outputErr = fmt.Fprintln(cmd.OutOrStdout(), "No matching delta for the exact base SHA256; using full-pack fallback.")
					}
					if outputErr == nil && event.Result.Changes != nil {
						_, outputErr = fmt.Fprint(cmd.OutOrStdout(), event.Result.Changes.Summary())
					}
				}
				if outputErr != nil {
					stop()
				}
			})
			if outputErr != nil {
				return outputErr
			}
			return returnErr
		},
	}
	cmd.Flags().StringVar(&state, "state", "", "Dedicated durable subscriber directory (single writer)")
	cmd.Flags().DurationVar(&interval, "interval", 0, "Delay between polls, required and positive")
	cmd.Flags().DurationVar(&timeout, "timeout", pack.DefaultRequestTimeout, "Maximum duration of each HTTPS request including body")
	cmd.Flags().BoolVar(&allowHTTPS, "allow-https", false, "Explicitly permit public HTTPS feed acquisition")
	cmd.Flags().BoolVar(&traceHTTP, "trace-http", false, "Report credential-free requested URLs and received body bytes")
	_ = cmd.MarkFlagRequired("state")
	_ = cmd.MarkFlagRequired("interval")
	return cmd
}
