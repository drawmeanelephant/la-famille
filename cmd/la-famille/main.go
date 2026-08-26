package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/tbuddy/la-famille/internal/logger"

	"github.com/spf13/cobra"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/generator"
	"github.com/tbuddy/la-famille/internal/ragexport"
	"github.com/tbuddy/la-famille/internal/runtimeassets"
	"github.com/tbuddy/la-famille/internal/watcher"
)

var (
	globalLogFile string
	contentDir    string
	outputDir     string
	assetDir      string
	templateFile  string
	siteURL       string
	projectRoot   string
	configPath    string
	showVersion   bool
	versionJSON   bool
)

func setupRootCmd(cfg config.Config) *cobra.Command {
	// The TUI command is shared for historical reasons, so pass the same
	// bootstrapped configuration to it when the real binary constructs the
	// command tree. Unit tests that call setupRootCmd with an ad-hoc Config keep
	// the old direct-CWD loading behavior.
	if cfg.ConfigPath != "" {
		tuiRuntimeConfig = cfg
		tuiRuntimeConfigSet = true
	} else {
		tuiRuntimeConfig = config.Config{}
		tuiRuntimeConfigSet = false
	}

	var rootCmd = &cobra.Command{
		Use:          "la-famille",
		Short:        "La Famille is a static site generator",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !showVersion && versionJSON {
				return fmt.Errorf("--json is only valid together with --version")
			}
			if showVersion {
				return writeBuildInfo(cmd.OutOrStdout(), versionJSON)
			}
			return cmd.Help()
		},
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if cmd.Name() != "tui" {
				_, _ = logger.Setup(globalLogFile, false)
			}
			return nil
		},
	}

	var buildCmd = &cobra.Command{
		Use:   "build",
		Short: "Build the static site",
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Update config from flags
			if projectRoot != "" {
				cfg.ProjectRoot = resolveProjectPath(cfg.ProjectRoot, projectRoot)
			}
			cfg.ContentDir = resolveProjectPath(cfg.ProjectRoot, contentDir)
			cfg.OutputDir = resolveProjectPath(cfg.ProjectRoot, outputDir)
			cfg.AssetDir = resolveProjectPath(cfg.ProjectRoot, assetDir)
			cfg.Template = resolveProjectPath(cfg.ProjectRoot, templateFile)
			if cmd.Flags().Changed("site-url") || cmd.Flags().Changed("siteurl") {
				cfg.SiteURL = siteURL
				if err := cfg.ValidateSiteURL(); err != nil {
					return fmt.Errorf("invalid configuration: %w", err)
				}
			} else if cfg.SiteURL == "" {
				if envURL := os.Getenv("SITE_URL"); envURL != "" {
					cfg.SiteURL = envURL
					if err := cfg.ValidateSiteURL(); err != nil {
						return fmt.Errorf("invalid configuration: %w", err)
					}
				} else if envURL := os.Getenv("LA_FAMILLE_SITE_URL"); envURL != "" {
					cfg.SiteURL = envURL
					if err := cfg.ValidateSiteURL(); err != nil {
						return fmt.Errorf("invalid configuration: %w", err)
					}
				}
			}
			if err := cfg.ValidateResolved(); err != nil {
				return fmt.Errorf("invalid configuration: %w", err)
			}
			res, err := generator.Build(cfg)
			if err != nil {
				return err
			}
			cacheStatus := "miss"
			if res.CacheHit {
				cacheStatus = "hit"
			}
			// Content warnings are collected into BuildResult and cached, but
			// nothing reported them: a page whose frontmatter failed to parse
			// was published with its metadata cleared and its YAML showing as
			// body text, and the build said only "Build complete". On a cache
			// hit not even the per-file parse warning was re-emitted, so the
			// problem became invisible after the first run.
			for _, w := range res.Warnings {
				slog.Warn("Content warning", "detail", w)
			}
			slog.Info("Build complete", "pages", res.PageCount, "duration", res.Duration,
				"cache", cacheStatus, "warnings", len(res.Warnings))
			if len(res.Warnings) > 0 {
				slog.Info("Run `la-famille check` to see these as validation errors")
			}
			return nil
		},
	}

	buildCmd.Flags().StringVarP(&contentDir, "content", "c", cfg.ContentDir, "Directory containing markdown files")
	buildCmd.Flags().StringVarP(&outputDir, "output", "o", cfg.OutputDir, "Directory for generated static site")
	buildCmd.Flags().StringVar(&assetDir, "asset-dir", cfg.AssetDir, "Directory containing static assets")
	buildCmd.Flags().StringVarP(&templateFile, "template", "t", cfg.Template, "Path to HTML layout template")
	buildCmd.Flags().StringVarP(&siteURL, "site-url", "s", cfg.SiteURL, "Public base URL of the site")
	buildCmd.Flags().StringVar(&siteURL, "siteurl", cfg.SiteURL, "Public base URL of the site (alias for --site-url)")

	var initForce bool
	var initTheme string
	var initCmd = &cobra.Command{
		Use:   "init",
		Short: "Initialize default configuration",
		RunE: func(_ *cobra.Command, _ []string) error {
			root := cfg.ProjectRoot
			if root == "" {
				root = "."
			}
			root, err := absolutePath(".", root)
			if err != nil {
				return fmt.Errorf("resolve project root: %w", err)
			}
			configFile := cfg.ConfigPath
			if configFile == "" {
				configFile = filepath.Join(root, "config.yaml")
			}
			if err := os.MkdirAll(filepath.Dir(configFile), 0755); err != nil {
				return fmt.Errorf("failed to create configuration directory: %w", err)
			}
			// Writing unconditionally replaced a customized config.yaml with
			// defaults and reported success, losing siteurl, output_dir and
			// every other setting the operator had chosen. Overwriting is now
			// something you ask for.
			if err := writeInitialConfig(configFile, initForce, initTheme); err != nil {
				return err
			}

			// --force replaces the selected config with the defaults. Use that
			// same default configuration for the scaffold paths below; otherwise
			// a previously customized or malformed config could cause init to
			// create directories that the newly written config does not use.
			initCfg, err := config.DefaultConfig().ResolvePaths(root)
			if err != nil {
				return fmt.Errorf("resolve default project paths: %w", err)
			}

			// The log names the template the chosen configuration actually
			// points at: a themed init used to say "Created default template"
			// while config.yaml referenced the theme layout (#513).
			tmplPath, themed := initTemplateTarget(initCfg.Template, initTheme)
			if _, err := os.Stat(tmplPath); os.IsNotExist(err) {
				if _, readErr := runtimeassets.DefaultTemplate(); readErr != nil {
					return fmt.Errorf("read embedded default template: %w", readErr)
				}
				if err := os.MkdirAll(filepath.Dir(tmplPath), 0755); err != nil {
					return fmt.Errorf("failed to create template directory: %w", err)
				}
				if themed {
					slog.Info("Created template", "theme", initTheme, "path", tmplPath)
				} else {
					slog.Info("Created default template", "path", tmplPath)
				}
			} else if err != nil {
				return fmt.Errorf("inspect default template %s: %w", tmplPath, err)
			}
			// The whole bundled packet is installed missing-only so frontmatter
			// `layout:` switching works across themes without a source checkout.
			bundled, err := runtimeassets.CuratedLayouts()
			if err != nil {
				return fmt.Errorf("read embedded bundled layouts: %w", err)
			}
			bundledFiles := make(map[string][]byte, len(bundled))
			for name, data := range bundled {
				bundledFiles[name+".html"] = data
			}
			if err := runtimeassets.InstallMissing(filepath.Dir(tmplPath), bundledFiles, 0600); err != nil {
				return fmt.Errorf("install bundled layouts: %w", err)
			}
			partials, err := runtimeassets.DefaultPartials()
			if err != nil {
				return fmt.Errorf("read embedded template partials: %w", err)
			}
			if err := runtimeassets.InstallMissing(filepath.Dir(tmplPath), partials, 0600); err != nil {
				return fmt.Errorf("install default template partials: %w", err)
			}

			cDir := initCfg.ContentDir
			if err := os.MkdirAll(cDir, 0755); err != nil {
				return fmt.Errorf("failed to create content directory: %w", err)
			}

			aDir := initCfg.AssetDir
			if err := os.MkdirAll(aDir, 0755); err != nil {
				return fmt.Errorf("failed to create assets directory: %w", err)
			}
			assetFiles, err := runtimeassets.DefaultAssetFiles()
			if err != nil {
				return fmt.Errorf("read embedded default assets: %w", err)
			}
			if !initCfg.GraphExplorer {
				delete(assetFiles, "graph/explorer.css")
				delete(assetFiles, "graph/explorer.js")
			}
			if err := runtimeassets.InstallMissing(aDir, assetFiles, 0644); err != nil {
				return fmt.Errorf("install default assets: %w", err)
			}

			// Starter pages are missing-only: a re-run or an init over an
			// existing site must never touch content the operator wrote.
			demos := demoContentFiles(initTheme, time.Now())
			created, err := scaffoldDemoContent(cDir, demos)
			if err != nil {
				return err
			}
			if len(created) > 0 {
				slog.Info("Scaffolded demo content", "files", strings.Join(created, ", "))
			}

			return nil
		},
	}

	var ragOutputDir, ragContentDir, ragAssetDir, ragTemplateFile string
	var ragCmd = &cobra.Command{
		Use:   "rag",
		Short: "Export project files into RAG-friendly markdown bundles",
		RunE: func(_ *cobra.Command, _ []string) error {
			ragCfg := cfg
			if projectRoot != "" {
				ragCfg.ProjectRoot = resolveProjectPath(cfg.ProjectRoot, projectRoot)
			}
			if ragContentDir != "" {
				ragCfg.ContentDir = resolveProjectPath(ragCfg.ProjectRoot, ragContentDir)
			}
			if ragAssetDir != "" {
				ragCfg.AssetDir = resolveProjectPath(ragCfg.ProjectRoot, ragAssetDir)
			}
			if ragTemplateFile != "" {
				ragCfg.Template = resolveProjectPath(ragCfg.ProjectRoot, ragTemplateFile)
			}
			if ragOutputDir != "" {
				ragCfg.RagDir = resolveProjectPath(ragCfg.ProjectRoot, ragOutputDir)
			}
			if err := ragCfg.ValidateResolved(); err != nil {
				return fmt.Errorf("invalid configuration: %w", err)
			}
			slog.Info("Writing RAG archive", "output", ragCfg.RagDir)
			return ragexport.RunExport(ragCfg)
		},
	}
	ragCmd.Flags().StringVar(&ragOutputDir, "output", cfg.RagDir, "Directory for the RAG archive")
	ragCmd.Flags().StringVar(&ragContentDir, "content", cfg.ContentDir, "Directory containing Markdown files")
	ragCmd.Flags().StringVar(&ragAssetDir, "asset-dir", cfg.AssetDir, "Directory containing static assets")
	ragCmd.Flags().StringVar(&ragTemplateFile, "template", cfg.Template, "Path to the HTML layout template")

	var servePort int
	var watchMode bool
	var serveCmd = &cobra.Command{
		Use:   "serve",
		Short: "Start a local web server to serve the generated site",
		RunE: func(_ *cobra.Command, _ []string) error {
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
			defer stop()

			// Serve OutputDir
			dir := cfg.OutputDir
			port := servePort
			if port == 0 {
				port = cfg.Port
				if port == 0 {
					port = config.DefaultConfig().Port
				}
			}

			if watchMode {
				slog.Info("Starting watch mode...")
				cfg.WatchMode = true
			}

			slog.Info("Building site...")
			if _, err := generator.Build(cfg); err != nil {
				slog.Error("Initial build failed", "error", err)
				return fmt.Errorf("initial build failed: %w", err)
			}

			if watchMode {
				go func() {
					// The error was discarded, so a watcher that stopped —
					// failing to register a directory, say — left the server
					// running and serving while silently no longer rebuilding
					// anything. Losing the watch is worth saying out loud.
					if err := watcher.Watch(ctx, cfg, nil); err != nil && !errors.Is(err, context.Canceled) {
						slog.Error("File watcher stopped; edits will no longer rebuild the site", "error", err)
					}
				}()
			}

			slog.Info(fmt.Sprintf("Serving %s on http://localhost:%d", dir, port))
			slog.Info("Press Ctrl+C to stop")

			mux := http.NewServeMux()
			mux.Handle("/", http.FileServer(http.Dir(dir)))

			if watchMode {
				mux.HandleFunc("/livereload", watcher.LiveReloadHandler)
			}

			server := &http.Server{
				Addr:              fmt.Sprintf("127.0.0.1:%d", port),
				Handler:           mux,
				ReadHeaderTimeout: 5 * time.Second,
				ReadTimeout:       10 * time.Second,
				WriteTimeout:      10 * time.Second,
			}

			errChan := make(chan error, 1)
			go func() {
				if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					errChan <- err
				}
			}()

			select {
			case err := <-errChan:
				return err
			case <-ctx.Done():
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				return server.Shutdown(shutdownCtx)
			}
		},
	}
	serveCmd.Flags().IntVarP(&servePort, "port", "p", 0, "Port to run the server on (overrides config)")
	serveCmd.Flags().BoolVarP(&watchMode, "watch", "w", false, "Watch for file changes and auto-rebuild")

	initCmd.Flags().BoolVarP(&initForce, "force", "f", false, "Replace an existing config.yaml, keeping the current one as config.yaml.bak")
	initCmd.Flags().StringVar(&initTheme, "theme", "", fmt.Sprintf("Bundled theme to set as the site default (one of: %s; run 'la-famille themes' for descriptions)", strings.Join(runtimeassets.CuratedLayoutNames, ", ")))

	themesCmd := &cobra.Command{
		Use:   "themes",
		Short: "List the bundled themes with one-line descriptions",
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()
			for _, theme := range runtimeassets.CuratedThemes() {
				fmt.Fprintf(out, "%s\t%s\n", theme.Name, theme.Description)
			}
			return nil
		},
	}

	rootCmd.AddCommand(buildCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(themesCmd)
	rootCmd.AddCommand(ragCmd)
	rootCmd.AddCommand(prCmd)
	rootCmd.AddCommand(tuiCmd)
	rootCmd.PersistentFlags().StringVar(&globalLogFile, "log-file", "", "Path to log file (default is stderr for CLI, la-famille.log for TUI)")
	rootCmd.PersistentFlags().StringVar(&projectRoot, "project-root", cfg.ProjectRoot, "Project root for config-relative paths")
	rootCmd.PersistentFlags().StringVar(&configPath, "config", cfg.ConfigPath, "Path to config.yaml (default: <project-root>/config.yaml)")
	rootCmd.PersistentFlags().BoolVar(&showVersion, "version", false, "Print build identity and exit")
	rootCmd.PersistentFlags().BoolVar(&versionJSON, "json", false, "Print machine-readable output (use with --version)")

	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(setupCheckCmd(cfg))
	rootCmd.AddCommand(setupPublishCheckCmd(cfg))
	rootCmd.AddCommand(setupNewCmd(cfg))
	rootCmd.AddCommand(setupAskCmd(cfg))

	return rootCmd
}

// configIndependentCommands lists the top-level commands that stay available
// when config.yaml is unusable. "init" is how a broken config.yaml gets
// regenerated, so blocking it would block the repair path; "pr" never reads
// the site configuration at all. The rest are cobra's own built-ins.
var configIndependentCommands = map[string]bool{
	"init":             true,
	"pr":               true,
	"themes":           true,
	"help":             true,
	"completion":       true,
	"__complete":       true,
	"__completeNoDesc": true,
}

func requiresSiteConfig(cmd *cobra.Command) bool {
	for c := cmd; c != nil && c.HasParent(); c = c.Parent() {
		if configIndependentCommands[c.Name()] {
			return false
		}
	}
	return true
}

// loadSiteConfig resolves config.yaml into a configuration that is safe to
// build with. A missing file is not an error (defaults apply), but a file that
// cannot be read, cannot be parsed, or does not validate yields the zero
// Config and a reason. It never substitutes defaults for a config file that
// exists: that is precisely how a malformed config.yaml used to produce a
// silently mis-configured but exit-0 build.
func loadSiteConfig(path string) (config.Config, error) {
	cfg, err := config.Load(path)
	if err != nil {
		return config.Config{}, fmt.Errorf("failed to load %s: %w", path, err)
	}
	if err := cfg.Validate(); err != nil {
		return config.Config{}, fmt.Errorf("invalid %s: %w", path, err)
	}
	return cfg, nil
}

// guardUnusableConfig makes every command that consumes the site configuration
// fail loudly when configErr is non-nil, while leaving the commands that
// repair or do not need config.yaml reachable. Cobra handles --help and a bare
// invocation before persistent hooks run, so help output stays available too.
func guardUnusableConfig(rootCmd *cobra.Command, configErr error) {
	if configErr == nil {
		return
	}
	inner := rootCmd.PersistentPreRunE
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if inner != nil {
			if err := inner(cmd, args); err != nil {
				return err
			}
		}
		if showVersion {
			return nil
		}
		if requiresSiteConfig(cmd) {
			return fmt.Errorf("%w (run `la-famille init` to regenerate config.yaml, or fix it by hand)", configErr)
		}
		return nil
	}
}

func main() {
	args := os.Args[1:]
	// Version output is intentionally independent of project state. This is
	// what makes an unpacked release archive identifiable in an empty directory
	// with no source tree, Go module, or network access.
	if argsRequestVersion(args) {
		rootCmd := setupRootCmd(config.DefaultConfig())
		versionArgs := []string{"--version"}
		for _, arg := range args {
			if arg == "--json" {
				versionArgs = append(versionArgs, "--json")
				break
			}
		}
		rootCmd.SetArgs(versionArgs)
		if err := rootCmd.Execute(); err != nil {
			slog.Error("Application error", "error", err)
			os.Exit(1)
		}
		return
	}

	// Load config first to set path-aware command defaults. On failure the
	// bootstrapper still returns the selected project root/config path so
	// `init --force` remains available as the repair path.
	cfg, configErr := loadProjectConfig(args)
	if configErr != nil {
		slog.Error("selected config is unusable; only `la-famille init` and help remain available", "error", configErr)
		fallback := config.DefaultConfig()
		fallback.ProjectRoot = cfg.ProjectRoot
		fallback.ConfigPath = cfg.ConfigPath
		cfg = fallback
	}

	rootCmd := setupRootCmd(cfg)
	guardUnusableConfig(rootCmd, configErr)

	if err := rootCmd.Execute(); err != nil {
		slog.Error("Application error", "error", err)
		os.Exit(1)
	}
}

// initTemplateTarget resolves which installed layout file the selected theme
// points at and whether that theme was explicitly named. A themed init must
// report its own template rather than the default one (#513).
func initTemplateTarget(defaultTemplate, theme string) (path string, themed bool) {
	if theme == "" {
		return defaultTemplate, false
	}
	return filepath.Join(filepath.Dir(defaultTemplate), theme+".html"), true
}

// initConfigBackup is where `init --force` moves an existing config.yaml. The
// name is fixed rather than timestamped so repeated runs stay predictable and
// the workspace does not accumulate copies.
const initConfigBackup = "config.yaml.bak"

// writeInitialConfig writes the default configuration, refusing by default to
// replace one that is already there.
//
// `init` used to write unconditionally, so running it a second time replaced a
// customized config.yaml with defaults and reported success — siteurl,
// output_dir and theme silently gone, and the next build publishing to the
// wrong place. The CLI guide had always described it as creating the file only
// when absent; this makes the behaviour match.
//
// The recovery path stays open: a config.yaml too broken to parse is repaired
// with `init --force`, which keeps the old file as config.yaml.bak so even a
// broken one can still be read back by hand.
func writeInitialConfig(path string, force bool, theme string) error {
	layoutPath := ""
	if theme != "" && !isBundledTheme(theme) {
		return fmt.Errorf("unknown theme %q; available themes:\n%s", theme, formatThemeChoices())
	}
	if theme != "" {
		layoutPath = "templates/" + theme + ".html"
	}
	backupPath := filepath.Join(filepath.Dir(path), initConfigBackup)
	_, statErr := os.Stat(path)
	switch {
	case statErr == nil && !force:
		return fmt.Errorf("%s already exists; refusing to overwrite it. Edit it directly, or run `la-famille init --force` to replace it (the current file is kept as %s)", path, initConfigBackup)
	case statErr == nil:
		existing, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read existing %s before replacing it: %w", path, err)
		}
		if err := os.WriteFile(backupPath, existing, 0600); err != nil {
			return fmt.Errorf("failed to back up %s to %s: %w", path, backupPath, err)
		}
		slog.Info("Backed up existing configuration", "from", path, "to", backupPath)
	case !os.IsNotExist(statErr):
		return fmt.Errorf("failed to inspect %s: %w", path, statErr)
	}

	if layoutPath == "" {
		if err := config.WriteDefault(path); err != nil {
			return fmt.Errorf("failed to write %s: %w", path, err)
		}
		return nil
	}
	if err := config.WriteDefaultWithLayout(path, layoutPath); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}
	return nil
}

// isBundledTheme reports whether name is one of the release packet layouts.
func isBundledTheme(name string) bool {
	for _, candidate := range runtimeassets.CuratedLayoutNames {
		if candidate == name {
			return true
		}
	}
	return false
}
