package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/tbuddy/la-famille/internal/ask"
	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/ragexport"
)

// All ask-related flags live as package-level so the small TUI integration
// can read them through AskFlagSnapshot().
var askFlagBundle = struct {
	provider  string
	model     string
	host      string
	port      int
	ragDir    string
	outputDir string
	rebuild   bool
	noBrowser bool
	maxCtx    int
	verbose   bool
	expose    bool
}{}

// setupAskCmd wires the `la-famille ask` cobra subcommand. It mirrors the
// registration shape used by build/serve/check.
func setupAskCmd(cfg config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ask",
		Short: "Serve a local citation-grounded question-answering UI for this site",
		Long: strings.TrimSpace(`ask serves a small loopback-only web UI that answers
questions using only the contents of your RAG archive plus generated site
metadata. It runs entirely on the local machine and never sends your content
to a remote service. Use --provider ollama with a local Ollama daemon to get
an LLM in front of the corpus. Set --provider fake to exercise the pipeline
without a model.`),
		RunE: runAsk(cfg),
	}

	cmd.Flags().StringVar(&askFlagBundle.provider, "provider", "ollama",
		"Local provider name (ollama, fake). Defaults to ollama.")
	cmd.Flags().StringVar(&askFlagBundle.model, "model", "",
		"Model identifier (e.g. llama3.2). Empty uses the provider's default.")
	cmd.Flags().StringVar(&askFlagBundle.host, "host", "127.0.0.1",
		"Bind address. Defaults to loopback. Use --expose-host at your own risk.")
	cmd.Flags().IntVarP(&askFlagBundle.port, "port", "p", ask.PortDefault,
		"HTTP port to serve the assistant on.")
	cmd.Flags().StringVar(&askFlagBundle.ragDir, "rag-dir", cfg.RagDir,
		"Directory containing the RAG archive (rag-content.md, rag-system.md, rag-config.md).")
	cmd.Flags().StringVar(&askFlagBundle.outputDir, "output", cfg.OutputDir,
		"Generated site output directory, used for citation URLs.")
	cmd.Flags().BoolVar(&askFlagBundle.rebuild, "rebuild", false,
		"Regenerate the RAG archive before starting the assistant.")
	cmd.Flags().BoolVar(&askFlagBundle.noBrowser, "no-browser", false,
		"Do not try to open the local UI in a browser.")
	cmd.Flags().IntVar(&askFlagBundle.maxCtx, "max-context", 0,
		"Maximum context characters fed to the provider per request (overrides the default).")
	cmd.Flags().BoolVar(&askFlagBundle.verbose, "verbose", false,
		"Verbose logging of retrieval/generation timings.")
	cmd.Flags().BoolVar(&askFlagBundle.expose, "expose-host", false,
		"Allow binding to a non-loopback address (you accept your prompt data may become reachable on your network).")

	// The flags describe optional behaviour and the standard validity
	// checks run before the server starts, so no flag groups are required.
	cmd.SilenceUsage = true
	return cmd
}

// runAsk returns a Cobra RunE that loads flags, validates them, optionally
// rebuilds the RAG archive, and starts the ask server.
func runAsk(cfg config.Config) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		host := strings.TrimSpace(askFlagBundle.host)
		if host == "" {
			host = "127.0.0.1"
		}

		port := askFlagBundle.port
		if port == 0 {
			port = ask.PortDefault
		}

		// Strong guard against accidental exposure.
		if !askFlagBundle.expose && !ask.IsLoopbackHost(host) {
			return fmt.Errorf("refusing to start: --host=%q is not loopback (re-run with --expose-host if you understand the implications)", host)
		}

		ragDir := askFlagBundle.ragDir
		if ragDir == "" {
			ragDir = cfg.RagDir
		}
		if ragDir == "" {
			ragDir = "rag-archive"
		}

		outputDir := askFlagBundle.outputDir
		if outputDir == "" {
			outputDir = cfg.OutputDir
		}

		if askFlagBundle.rebuild {
			slog.Info("rebuilding RAG archive", "dir", ragDir)
			reb := cfg
			reb.RagDir = ragDir
			reb.OutputDir = outputDir
			if err := ragexport.RunExport(reb); err != nil {
				return fmt.Errorf("ask: rag rebuild failed: %w", err)
			}
		}

		askCfg := ask.Config{
			ProviderName: askFlagBundle.provider,
			Model:        askFlagBundle.model,
			Host:         host,
			Port:         port,
			RagDir:       ragDir,
			OutputDir:    outputDir,
			Rebuild:      askFlagBundle.rebuild,
			NoBrowser:    askFlagBundle.noBrowser,
			MaxContext:   askFlagBundle.maxCtx,
			Verbose:      askFlagBundle.verbose,
			LoopbackOnly: !askFlagBundle.expose,
		}

		// Path safety: ensure configured dirs escape no upward traversal.
		for _, p := range []struct {
			name, value string
		}{{"rag-dir", ragDir}, {"output", outputDir}} {
			if !filepath.IsLocal(p.value) {
				return fmt.Errorf("ask: %s must be a local path, got %s", p.name, p.value)
			}
		}

		server, err := ask.NewServer(askCfg)
		if err != nil {
			return err
		}

		url := "http://" + net.JoinHostPort(host, strconv.Itoa(port))
		slog.Info("Ask This Site is starting", "url", url,
			"provider", askFlagBundle.provider,
			"model", firstNonEmpty(askFlagBundle.model, "(provider default)"),
			"corpus", ragDir,
		)
		fmt.Fprintln(cmd.OutOrStdout(), "🐙 Ask This Site")
		fmt.Fprintf(cmd.OutOrStdout(), "  ↪ open: %s\n", url)
		fmt.Fprintf(cmd.OutOrStdout(), "  provider: %s · model: %s\n",
			askFlagBundle.provider, firstNonEmpty(askFlagBundle.model, "(provider default)"))
		fmt.Fprintln(cmd.OutOrStdout(), "  privacy: runs on this machine only. Do not expose this port to the public internet.")
		fmt.Fprintln(cmd.OutOrStdout(), "  stop:    Ctrl+C")

		if !askFlagBundle.noBrowser {
			tryOpenBrowser(url)
		}

		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		return server.Start(ctx)
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// isLoopbackAddress was removed in favour of ask.IsLoopbackHost (internal/ask),
// which understands IPv6 brackets and uses net.SplitHostPort internally.

// tryOpenBrowser politely asks the OS to open a URL. It is best-effort and
// never blocks the assistant startup — errors collapse silently into the
// informational banner above.
func tryOpenBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
	// On Linux, xdg-open returns immediately even though the browser is
	// launching. Do not wait on it.
	go func() {
		if cmd.Process != nil {
			done := make(chan struct{})
			go func() {
				_ = cmd.Wait()
				close(done)
			}()
			select {
			case <-done:
			case <-time.After(2 * time.Second):
				_ = cmd.Process.Kill()
			}
		}
	}()
}
