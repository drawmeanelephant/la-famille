<file path=".github/workflows/ci.yml">
<content>
name: CI

on:
  push:
    branches: [ master, main ]
  pull_request:
    branches: [ master, main ]

jobs:
  run-tests:
    name: Go Test Suite
    runs-on: ubuntu-latest

    steps:
      - name: Checkout Code
        uses: actions/checkout@v7

      - name: Setup Go
        uses: actions/setup-go@v6
        with:
          go-version: 'stable'

      - name: Verify Code and Run Tests
        run: |
          go test -v ./...

</content>
</file>

<file path=".github/workflows/cron-sync.yml">
<content>
name: Nightly PR Sync

on:
  schedule:
    # Runs at 08:00 UTC (4:00 AM EDT) every day
    - cron: '0 8 * * *'
  workflow_dispatch:

jobs:
  sync:
    name: Clear the Litterbox
    runs-on: ubuntu-latest
    permissions:
      contents: write
      pull-requests: write

    steps:
      - name: Checkout Code
        uses: actions/checkout@v7
        with:
          fetch-depth: 0 # Needed for branch checkouts

      - name: Setup Go
        uses: actions/setup-go@v6
        with:
          go-version: 'stable'

      - name: Run PR Sync
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          go run ./cmd/la-famille pr sync --base main

</content>
</file>

<file path=".github/workflows/deploy.yml">
<content>
name: Deploy La Famille Site

on:
  push:
    branches: [ "main", "master" ]
  workflow_dispatch: # Allows manual trigger

# Sets permissions of the GITHUB_TOKEN to allow deployment to GitHub Pages
permissions:
  contents: write
  pages: write
  id-token: write

concurrency:
  group: "pages"
  cancel-in-progress: false

jobs:
  build-and-deploy:
    environment:
      name: github-pages
      url: ${{ steps.deployment.outputs.page_url }}
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v7

      - name: Setup Go
        uses: actions/setup-go@v6
        with:
          go-version: 'stable'

      - name: Build Static Site
        run: |
          # Executes the La Famille compiler
          go run ./cmd/la-famille build

      - name: Generate RAG Export
        run: |
          go run ./cmd/la-famille rag
          cp -r rag-archive public/rag-archive


      - name: Commit and Push RAG Export
        run: |
          git config --global user.name 'github-actions[bot]'
          git config --global user.email 'github-actions[bot]@users.noreply.github.com'
          git add rag-archive/
          git commit -m "chore: auto-update rag export [skip ci]" || echo "No changes to commit"
          git push origin HEAD:${GITHUB_REF#refs/heads/}
      - name: Setup Pages
        uses: actions/configure-pages@v6
        with:
          enablement: true

      - name: Upload artifact
        uses: actions/upload-pages-artifact@v5
        with:
          path: './public' # Target the generator's output directory

      - name: Deploy to GitHub Pages
        id: deployment
        uses: actions/deploy-pages@v5

</content>
</file>

<file path=".github/workflows/jules-ci.yml">
<content>
name: La Famille Integration Pipeline

on:
  pull_request:
    branches: [ main ]

jobs:
  verify-and-merge:
    # Only run this if the PR was created by the Jules bot / integration
    if: github.actor == 'google-labs-code' || contains(github.head_ref, 'jules')
    runs-on: ubuntu-latest

    steps:
      - name: Checkout Code
        uses: actions/checkout@v7

      - name: Setup Go
        uses: actions/setup-go@v6
        with:
          go-version: 'stable'

      - name: Run Verification Suite
        run: |
          go test -v ./...

      - name: Run golangci-lint
        uses: golangci/golangci-lint-action@v6
        with:
          version: v1.64.5

      - name: Auto-Merge Green PRs
        if: success()
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          PR_URL: ${{ github.event.pull_request.html_url }}
        run: |
          gh pr merge --auto --squash "$PR_URL"

</content>
</file>

<file path=".github/workflows/lint.yml">
<content>
name: Lint

on:
  pull_request:
    branches: [ master, main ]

jobs:
  golangci:
    name: lint
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: 'go.mod'
      - name: golangci-lint
        uses: golangci/golangci-lint-action@v6
        with:
          version: v1.64.5

</content>
</file>

<file path="README.md">
<content>
# La Famille 🐙

[![GitHub Repository](https://img.shields.io/badge/GitHub-Repository-blue?logo=github)](https://github.com/drawmeanelephant/la-famille/)

La Famille is a fast, feature-rich static site generator written in Go. It goes beyond simple markdown-to-HTML conversion by offering powerful developer tools, an interactive Terminal UI (TUI), and AI-ready RAG (Retrieval-Augmented Generation) exports.

This project is built and maintained primarily by **Jules** (AI assistant) alongside an eight-legged friend, Raoul(s) the Octopus. We take a "Jules-forward" approach to development. If you are opening a Pull Request, please make sure to tag Jules in the comments to keep the AI looped in.

## Features ✨

*   **Lightning-Fast Static Generation:** Converts Markdown content into clean, semantic HTML using the `goldmark` library.
*   **Interactive TUI:** A sleek Bubbletea-powered terminal interface for managing builds, serving the site locally, and viewing project stats.
*   **Robust CLI:** A powerful command-line interface built with `cobra` for tasks like initialization, building, serving, and RAG generation.
*   **RAG Export:** Native tools to extract your site's content and metadata into clean archives optimized for LLM context windows (`rag-system.md`, `rag-content.md`, etc.).
*   **Flexible Templating:** Support for multiple HTML layouts (e.g., standard, cyberpunk, minimal) easily overridden via YAML frontmatter.
*   **Built-in Local Server:** Instantly preview your site with `go run ./cmd/la-famille serve`.
*   **Smart Graphing:** Automatically generates `graph.json`, `backlinks.json`, and handles non-existent internal links by generating helpful stub pages.

## Quickstart 🚀

### Prerequisites
*   [Go](https://go.dev/doc/install) installed on your machine.

### Build & Run
To run the static site generator using the CLI:
```bash
go run ./cmd/la-famille build
```

To launch the interactive TUI:
```bash
go run ./cmd/la-famille tui
```

### TUI Navigation & Controls
The TUI uses standard, frictionless keybindings for easy navigation:
*   **Navigation:** Use `up`/`down` arrows or Unix-centric `j`/`k` primitives to move through the menus.
*   **Selection & Exit:** Press `Enter` or `Space` to execute a command. Press `q` or `Esc` to safely drop back to the main menu screen buffer.
*   **Active Server Views:** When you select "Serve Site" (or "Serve Site with Watch"), the TUI locks into an alternate screen buffer, displaying the dancing mascot animation (Raoul!). To gracefully tear down the network handle and exit back to the main menu, press `q` or `Esc`.

To serve the generated site locally (defaults to port 8080):
```bash
go run ./cmd/la-famille serve
```

## Documentation 📚

The commands above will get you started, but La Famille has a lot more to offer. For deep-dive guides on how to use all the features, please explore our documentation:

*   **[Setup & Getting Started](content/docs/setup.md)**
*   **[CLI Reference](content/docs/cli.md)**
*   **[Using the TUI](content/docs/tui.md)**
*   **[Templating Guide](content/docs/templates.md)**
*   **[RAG Export Guide](content/docs/rag.md)**
*   **[How the Generator Works](content/docs/generator.md)**

---
*Generated with ❤️ by Jules*


### CI/Testing
La Famille uses a comprehensive automated testing pipeline. All code merges are gated by passing `go test` and static analysis provided by `golangci-lint` to ensure security and code quality.

## GitHub Action 🤖

You can easily build your La Famille site in CI using our GitHub Action:

```yaml
steps:
  - uses: actions/checkout@v4
  - name: Build with La Famille
    uses: drawmeanelephant/la-famille@main
```

</content>
</file>

<file path="cmd/la-famille/fixture_test.go">
<content>
package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/generator"
)

func TestFixtures(t *testing.T) {
	fixtureDir := filepath.Join("..", "..", "assets", "testdata", "sites")
	fixtures, err := os.ReadDir(fixtureDir)
	if err != nil {
		t.Fatalf("failed to read fixtures directory: %v", err)
	}

	templateFile := filepath.Join("..", "..", "templates", "layout.html")

	for _, f := range fixtures {
		if !f.IsDir() {
			continue
		}

		t.Run(f.Name(), func(t *testing.T) {
			contentDir := filepath.Join(fixtureDir, f.Name(), "content")
			expectedDir := filepath.Join(fixtureDir, f.Name(), "expected")

			outputDir := t.TempDir()

			cfg := config.DefaultConfig()
			cfg.ContentDir = contentDir
			cfg.OutputDir = outputDir
			cfg.Template = templateFile

			if _, err := generator.Build(cfg); err != nil {
				t.Fatalf("run failed: %v", err)
			}

			// Check all files in expectedDir exist in outputDir and match
			err = filepath.WalkDir(expectedDir, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() {
					return nil
				}

				relPath, err := filepath.Rel(expectedDir, path)
				if err != nil {
					return err
				}

				actualPath := filepath.Join(outputDir, relPath)
				// If the expected file is under 'pages/', it maps to the root of the output directory
				if strings.HasPrefix(relPath, "pages"+string(filepath.Separator)) {
					actualPath = filepath.Join(outputDir, relPath[len("pages")+1:])
				}

				actualContent, err := os.ReadFile(actualPath)
				if err != nil {
					t.Errorf("missing expected file %s (checked %s): %v", relPath, actualPath, err)
					return nil
				}

				expectedContent, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("failed to read expected file %s: %v", relPath, err)
				}

				if string(actualContent) != string(expectedContent) {
					t.Errorf("content mismatch in %s:\nExpected:\n%s\nActual:\n%s\n", relPath, string(expectedContent), string(actualContent))
				}

				return nil
			})

			if err != nil {
				t.Fatalf("walk failed: %v", err)
			}
		})
	}
}

</content>
</file>

<file path="cmd/la-famille/main.go">
<content>
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/spf13/cobra"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/generator"
	"github.com/tbuddy/la-famille/internal/ragexport"
	"github.com/tbuddy/la-famille/internal/watcher"
)

var (
	contentDir   string
	outputDir    string
	templateFile string
)

func setupRootCmd(cfg config.Config) *cobra.Command {
	var rootCmd = &cobra.Command{
		Use:   "la-famille",
		Short: "La Famille is a static site generator",
	}

	var buildCmd = &cobra.Command{
		Use:   "build",
		Short: "Build the static site",
		RunE: func(_ *cobra.Command, _ []string) error {
			// Update config from flags
			cfg.ContentDir = contentDir
			cfg.OutputDir = outputDir
			cfg.Template = templateFile
			_, err := generator.Build(cfg)
			return err
		},
	}

	buildCmd.Flags().StringVarP(&contentDir, "content", "c", cfg.ContentDir, "Directory containing markdown files")
	buildCmd.Flags().StringVarP(&outputDir, "output", "o", cfg.OutputDir, "Directory for generated static site")
	buildCmd.Flags().StringVarP(&templateFile, "template", "t", cfg.Template, "Path to HTML layout template")

	var initCmd = &cobra.Command{
		Use:   "init",
		Short: "Initialize default configuration",
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := config.WriteDefault("config.yaml"); err != nil {
				return fmt.Errorf("failed to write config.yaml: %w", err)
			}
			fmt.Println("Created default config.yaml")
			return nil
		},
	}

	var ragCmd = &cobra.Command{
		Use:   "rag",
		Short: "Export project files into RAG-friendly markdown bundles",
		RunE: func(_ *cobra.Command, _ []string) error {
			return ragexport.RunExport(cfg)
		},
	}

	var servePort int
	var watchMode bool
	var serveCmd = &cobra.Command{
		Use:   "serve",
		Short: "Start a local web server to serve the generated site",
		RunE: func(_ *cobra.Command, _ []string) error {
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
				fmt.Println("Starting watch mode...")
				cfg.WatchMode = true
			}

			fmt.Println("Building site...")
			if _, err := generator.Build(cfg); err != nil {
				log.Printf("Initial build failed: %v", err)
			}

			if watchMode {
				go func() { _ = watcher.Watch(context.Background(), cfg, nil) }()
			}

			fmt.Printf("Serving %s on http://localhost:%d\n", dir, port)
			fmt.Printf("Press Ctrl+C to stop\n")

			mux := http.NewServeMux()
			mux.Handle("/", http.FileServer(http.Dir(dir)))

			if watchMode {
				mux.HandleFunc("/livereload", watcher.LiveReloadHandler)
			}

			server := &http.Server{
				Addr:              fmt.Sprintf("127.0.0.1:%d", port),
				Handler:           mux,
				ReadHeaderTimeout: 5 * time.Second,
			}
			return server.ListenAndServe()
		},
	}
	serveCmd.Flags().IntVarP(&servePort, "port", "p", 0, "Port to run the server on (overrides config)")
	serveCmd.Flags().BoolVarP(&watchMode, "watch", "w", false, "Watch for file changes and auto-rebuild")

	rootCmd.AddCommand(buildCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(ragCmd)
	rootCmd.AddCommand(prCmd)
	rootCmd.AddCommand(tuiCmd)
	rootCmd.AddCommand(serveCmd)

	return rootCmd
}

func main() {
	// Load config first to set defaults for flags
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Printf("Warning: failed to load config.yaml: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
	}

	rootCmd := setupRootCmd(cfg)

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

</content>
</file>

<file path="cmd/la-famille/main_test.go">
<content>
package main

import (
	"bytes"
	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/generator"
	"github.com/tbuddy/la-famille/internal/stub"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCLIOverrides(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()

	// Write a config.yaml to the temp dir
	configFile := filepath.Join(tmpDir, "config.yaml")
	yamlContent := []byte(`
site_name: "Test Site From Config"
output_dir: "default_output_from_config"
content_dir: "default_content_from_config"
theme: "dark"
`)
	if err := os.WriteFile(configFile, yamlContent, 0600); err != nil {
		t.Fatalf("Failed to write config.yaml: %v", err)
	}

	// Create content dir
	contentDir := filepath.Join(tmpDir, "content")
	if err := os.Mkdir(contentDir, 0755); err != nil {
		t.Fatalf("Failed to create content dir: %v", err)
	}

	// Write a test markdown file
	mdContent := []byte(`---
title: Test Page
---
# Hello World
<script>alert('xss')</script>
`)
	if err := os.WriteFile(filepath.Join(contentDir, "index.md"), mdContent, 0600); err != nil {
		t.Fatalf("Failed to write index.md: %v", err)
	}

	// Create templates dir and layout
	templateDir := filepath.Join(tmpDir, "templates")
	if err := os.Mkdir(templateDir, 0755); err != nil {
		t.Fatalf("Failed to create templates dir: %v", err)
	}
	htmlContent := []byte(`<!DOCTYPE html>
<html lang="en" data-theme="{{.Site.Theme}}">
<body>
<h1>{{.Title}} - {{.Site.SiteName}}</h1>
{{.Content}}
</body>
</html>`)
	if err := os.WriteFile(filepath.Join(templateDir, "layout.html"), htmlContent, 0600); err != nil {
		t.Fatalf("Failed to write layout.html: %v", err)
	}

	// Build la-famille executable first
	exePath := filepath.Join(tmpDir, "la-famille.bin")
	cmdBuild := exec.Command("go", "build", "-o", exePath, "../../cmd/la-famille")
	if err := cmdBuild.Run(); err != nil {
		t.Fatalf("failed to build la-famille: %v", err)
	}

	cmdRun := exec.Command(exePath, "build",
		"--content", contentDir,
		"--output", filepath.Join(tmpDir, "cli_output"),
		"--template", filepath.Join(templateDir, "layout.html"))

	// Run from tmpDir so it picks up config.yaml
	cmdRun.Dir = tmpDir

	var stderr bytes.Buffer
	cmdRun.Stderr = &stderr
	if err := cmdRun.Run(); err != nil {
		t.Fatalf("la-famille run failed: %v, stderr: %s", err, stderr.String())
	}

	// Check if output went to `cli_output` instead of `default_output_from_config`
	outputFile := filepath.Join(tmpDir, "cli_output", "index.html")
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Fatalf("Output file was not created in the CLI-specified directory. Did CLI flag override fail?")
	}

	// Read output to ensure config vars (like Theme and SiteName) were still loaded
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read generated html: %v", err)
	}
	htmlStr := string(content)

	if !strings.Contains(htmlStr, `data-theme="dark"`) {
		t.Errorf("Expected config data-theme='dark' to be present, but it wasn't")
	}
	if !strings.Contains(htmlStr, `Test Page - Test Site From Config`) {
		t.Errorf("Expected SiteName from config to be present, but it wasn't")
	}
	if strings.Contains(htmlStr, "<script>") {
		t.Errorf("XSS payload was not sanitized: %s", htmlStr)
	}

	// Test serve command defaults to 8080 when no port flag is provided
	cmdServe := exec.Command(exePath, "serve")
	cmdServe.Dir = tmpDir

	stdoutPipe, err := cmdServe.StdoutPipe()
	if err != nil {
		t.Fatalf("failed to create stdout pipe: %v", err)
	}

	if err := cmdServe.Start(); err != nil {
		t.Fatalf("failed to start serve command: %v", err)
	}

	outputChan := make(chan string)
	go func() {
		buf := new(bytes.Buffer)
		// Read just enough to verify the port
		b := make([]byte, 1024)
		n, _ := stdoutPipe.Read(b)
		buf.Write(b[:n])
		outputChan <- buf.String()
	}()

	select {
	case serveOut := <-outputChan:
		if !strings.Contains(serveOut, "http://localhost:8080") {
			t.Errorf("Expected serve command to default to port 8080, got output: %s", serveOut)
		}
	case <-time.After(2 * time.Second):
		t.Errorf("Timed out waiting for serve command output")
	}

	if err := cmdServe.Process.Kill(); err != nil {
		t.Fatalf("failed to kill serve command: %v", err)
	}

	// Wait for process to clean up
	_ = cmdServe.Wait()
}


func TestInitCommand(t *testing.T) {
	tmpDir := t.TempDir()

	exePath := filepath.Join(tmpDir, "la-famille.bin")
	cmdBuild := exec.Command("go", "build", "-o", exePath, "../../cmd/la-famille")
	if err := cmdBuild.Run(); err != nil {
		t.Fatalf("failed to build la-famille: %v", err)
	}

	cmdRun := exec.Command(exePath, "init")
	cmdRun.Dir = tmpDir

	if err := cmdRun.Run(); err != nil {
		t.Fatalf("la-famille init failed: %v", err)
	}

	configFile := filepath.Join(tmpDir, "config.yaml")
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		t.Fatalf("la-famille init did not create config.yaml")
	}
}

func TestStubRelPathFromToFallback(t *testing.T) {
	tests := []struct {
		name     string
		base     string
		target   string
		expected string
		wantErr  bool
	}{
		{
			name:     "same directory",
			base:     "a.md",
			wantErr:  false,
			target:   "b.html",
			expected: "b.html",
		},
		{
			name:     "target in subdirectory",
			base:     "a.md",
			wantErr:  false,
			target:   "dir/b.html",
			expected: "dir/b.html",
		},
		{
			name:     "base in subdirectory",
			base:     "dir/a.md",
			wantErr:  false,
			target:   "b.html",
			expected: "../b.html",
		},
		{
			name:     "absolute and relative paths (error)",
			base:     "/absolute/path/base.md",
			target:   "relative/target.html",
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := stub.RelPathFromTo(tt.base, tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("stub.RelPathFromTo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("stub.RelPathFromTo() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestProcessFile_PathTraversalPrevented(t *testing.T) {
	tempDir := t.TempDir()

	// Create mock config
	cfg := config.Config{
		ContentDir: filepath.Join(tempDir, "content"),
		OutputDir:  filepath.Join(tempDir, "public"),
		Template:   filepath.Join(tempDir, "layout.html"),
	}

	_ = os.MkdirAll(cfg.ContentDir, 0755)
	_ = os.MkdirAll(cfg.OutputDir, 0755)
	_ = os.WriteFile(cfg.Template, []byte("<html><body>{{.Content}}</body></html>"), 0600)

	fileName := "index.md"
	// Path traverses out of the content directory to a theoretical /tmp directory
	content := []byte("# Home\n[Malicious](../../../../../tmp/hack.md)")
	_ = os.WriteFile(filepath.Join(cfg.ContentDir, fileName), content, 0600)

	_, err := generator.Build(cfg)
	if err != nil {
		t.Errorf("run failed: %v", err)
	}

	// Make sure the index file is generated but doesn't rewrite to .html (stays as original destination because traversal was blocked)
	indexFile := filepath.Join(cfg.OutputDir, "index.html")
	indexContent, _ := os.ReadFile(indexFile)
	if strings.Contains(string(indexContent), `href="../../../../../tmp/hack.html"`) {
		t.Errorf("Malicious link was incorrectly rewritten to .html: %s", string(indexContent))
	}

	// Verify that the malicious file stub is not created anywhere
	maliciousFile := filepath.Join(tempDir, "tmp", "hack.html")
	if _, err := os.Stat(maliciousFile); !os.IsNotExist(err) {
		t.Errorf("Malicious stub was incorrectly generated outside the output directory at: %s", maliciousFile)
	}
}

func TestRun_WalkError(t *testing.T) {
	tempDir := t.TempDir()

	// Create mock config
	cfg := config.Config{
		ContentDir: filepath.Join(tempDir, "does-not-exist"),
		OutputDir:  filepath.Join(tempDir, "public"),
		Template:   filepath.Join(tempDir, "layout.html"),
	}

	// Create valid output dir and template file so it only fails on content dir
	_ = os.MkdirAll(cfg.OutputDir, 0755)
	_ = os.WriteFile(cfg.Template, []byte("<html><body>{{.Content}}</body></html>"), 0600)

	_, err := generator.Build(cfg)
	if err == nil {
		t.Fatalf("expected an error when walking a non-existent directory, but got nil")
	}

	if !strings.Contains(err.Error(), "failed to walk content directory") {
		t.Errorf("expected error message to contain 'failed to walk content directory', got: %v", err)
	}
}

func TestCommandFlags(t *testing.T) {
	// This prevents flag names from silently drifting from documentation again.
	cfg := config.Config{}
	rootCmd := setupRootCmd(cfg)

	// Test build command flags
	buildCmd, _, err := rootCmd.Find([]string{"build"})
	if err != nil {
		t.Fatalf("Failed to find build command: %v", err)
	}

	buildFlags := []string{"content", "output", "template"}
	for _, flag := range buildFlags {
		if buildCmd.Flags().Lookup(flag) == nil {
			t.Errorf("buildCmd is missing expected flag: %s", flag)
		}
	}

	// Test serve command flags
	serveCmd, _, err := rootCmd.Find([]string{"serve"})
	if err != nil {
		t.Fatalf("Failed to find serve command: %v", err)
	}

	serveFlags := []string{"port", "watch"}
	for _, flag := range serveFlags {
		if serveCmd.Flags().Lookup(flag) == nil {
			t.Errorf("serveCmd is missing expected flag: %s", flag)
		}
	}
}

</content>
</file>

<file path="cmd/la-famille/pr.go">
<content>
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tbuddy/la-famille/internal/github"
)

var prCmd = &cobra.Command{
	Use:   "pr",
	Short: "Manage GitHub Pull Requests",
}

var prSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Automated PR management (clear the litterbox)",
	Long: `Fetches open pull requests by automation agents.
Closes stale/conflicting PRs and merges passing ones.
If there are local uncommitted changes, branches and creates a new PR.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		token := os.Getenv("GITHUB_TOKEN")
		if token == "" {
			return fmt.Errorf("GITHUB_TOKEN environment variable must be set")
		}

		baseBranch, _ := cmd.Flags().GetString("base")

		cfg := github.SyncConfig{
			Token:         token,
			BotAuthors:    []string{"google-labs-jules", "google-labs-code"},
			DefaultBranch: baseBranch,
		}

		fmt.Println("Starting automated PR sync...")
		if err := github.RunSync(cfg); err != nil {
			return fmt.Errorf("sync failed: %w", err)
		}
		fmt.Println("Sync completed successfully.")
		return nil
	},
}

func init() {
	prSyncCmd.Flags().String("base", "main", "The base branch to target for new PRs")
	prCmd.AddCommand(prSyncCmd)
	// We need to add prCmd to rootCmd in main.go
}

</content>
</file>

<file path="cmd/la-famille/tui.go">
<content>
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/generator"
	"github.com/tbuddy/la-famille/internal/ragexport"
	"github.com/tbuddy/la-famille/internal/watcher"
)

var p *tea.Program

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch the semi-graphical user interface",
	RunE: func(_ *cobra.Command, _ []string) error {
		cfg, err := config.Load("config.yaml")
		if err != nil {
			// use defaults if config fails
			cfg = config.Config{
				ContentDir: "content",
				OutputDir:  "public",
				Template:   "templates/layout.html",
				AssetDir:   "assets",
				RagDir:     "rag-archive",
			}
		}
		if err := cfg.Validate(); err != nil {
			return fmt.Errorf("configuration validation failed: %w", err)
		}

		p = tea.NewProgram(initialModel(cfg), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("tui error: %w", err)
		}
		return nil
	},
}

// Rough approximation: 1 token ≈ 4 bytes (OpenAI tokenizer heuristic)
const bytesPerToken = 4

type screen int

const (
	screenMenu screen = iota
	screenRaoul
	screenStats
	screenWorking
	screenServe
)

type menuOption struct {
	label string
}

type tickMsg time.Time

type statsUpdateMsg struct {
	res generator.BuildResult
}

type workResultMsg struct {
	err error
	msg string
	res *generator.BuildResult
}

type model struct {
	cfg           config.Config
	screen        screen
	choices       []menuOption
	cursor        int
	frame         int
	workMsg       string
	workErr       error
	server        *http.Server
	watcherCancel context.CancelFunc
	stats         *generator.BuildResult
}

func initialModel(cfg config.Config) model {
	return model{
		cfg:    cfg,
		screen: screenMenu,
		choices: []menuOption{
			{"Build Site"},
			{"RAG Export"},
			{"Serve Site"},
			{"Serve Site with Watch"},
			{"Stats"},
			{"Just Raoul"},
			{"Quit"},
		},
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*500, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			if m.screen == screenMenu {
				return m, tea.Quit
			} else if m.screen != screenWorking || strings.Contains(m.workMsg, "complete") || m.screen == screenServe {
				if m.watcherCancel != nil {
					m.watcherCancel()
					m.watcherCancel = nil
				}
				if m.screen == screenServe && m.server != nil {
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
					_ = m.server.Shutdown(ctx)
					m.server = nil
				}
				m.screen = screenMenu
				return m, nil
			}
		case "esc":
			if m.screen != screenWorking || strings.Contains(m.workMsg, "complete") || m.screen == screenServe {
				if m.watcherCancel != nil {
					m.watcherCancel()
					m.watcherCancel = nil
				}
				if m.screen == screenServe && m.server != nil {
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
					_ = m.server.Shutdown(ctx)
					m.server = nil
				}
				m.screen = screenMenu
				return m, nil
			}
		case "up", "k":
			if m.screen == screenMenu {
				if m.cursor > 0 {
					m.cursor--
				}
			}
		case "down", "j":
			if m.screen == screenMenu {
				if m.cursor < len(m.choices)-1 {
					m.cursor++
				}
			}
		case "enter", " ":
			if m.screen == screenMenu {
				choice := m.choices[m.cursor].label
				switch choice {
				case "Quit":
					return m, tea.Quit
				case "Just Raoul":
					m.screen = screenRaoul
					m.frame = 0
					return m, tickCmd()
				case "Stats":
					m.screen = screenStats
					return m, nil
				case "Build Site":
					m.screen = screenWorking
					m.workMsg = "Building site..."
					m.workErr = nil

					// Re-assigning to avoid capturing loop variable problem, though we don't have a loop here
					cfg := m.cfg
					return m, func() tea.Msg {
						res, err := generator.Build(cfg)
						return workResultMsg{err: err, msg: "Build complete", res: &res}
					}
				case "RAG Export":
					m.screen = screenWorking
					m.workMsg = "Exporting RAG data..."
					m.workErr = nil
					return m, func() tea.Msg {
						err := ragexport.RunExport(m.cfg)
						return workResultMsg{err: err, msg: "RAG Export complete"}
					}
				case "Serve Site", "Serve Site with Watch":
					m.screen = screenServe
					m.frame = 0
					port := m.cfg.Port
					if port == 0 {
						port = config.DefaultConfig().Port
					}

					if choice == "Serve Site with Watch" {
						m.cfg.WatchMode = true
						if _, err := generator.Build(m.cfg); err != nil {
							log.Printf("Initial build failed: %v", err)
						}

						watchCtx, cancelWatch := context.WithCancel(context.Background())
						m.watcherCancel = cancelWatch

						go func(ctx context.Context, c config.Config) {
							if err := watcher.Watch(ctx, c, func(res generator.BuildResult) {
								if p != nil {
									p.Send(statsUpdateMsg{res: res})
								}
							}); err != nil {
								log.Printf("Watcher thread exited with: %v", err)
							}
						}(watchCtx, m.cfg)
					}

					mux := http.NewServeMux()
					mux.Handle("/", http.FileServer(http.Dir(m.cfg.OutputDir)))
					if m.cfg.WatchMode {
						mux.HandleFunc("/livereload", watcher.LiveReloadHandler)
					}

					m.server = &http.Server{
						Addr:              fmt.Sprintf("127.0.0.1:%d", port),
						Handler:           mux,
						ReadHeaderTimeout: 5 * time.Second,
					}
					go func() {
						_ = m.server.ListenAndServe()
					}()
					return m, tickCmd()
				}
			} else if m.screen == screenWorking {
				if strings.Contains(m.workMsg, "complete") || m.workErr != nil {
					m.screen = screenMenu
				}
			}
		}

	case tickMsg:
		if m.screen == screenRaoul || m.screen == screenServe {
			m.frame = (m.frame + 1) % 2
			return m, tickCmd()
		}

	case statsUpdateMsg:
		newRes := msg.res
		m.stats = &newRes
		return m, nil

	case workResultMsg:
		m.workMsg = msg.msg
		m.workErr = msg.err
		if msg.res != nil {
			m.stats = msg.res
		}

	}

	return m, nil
}

func (m model) View() string {
	switch m.screen {
	case screenMenu:
		s := lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Render(staticRaoul()) + "\n\n"
		s += lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).Render("Welcome to La Famille TUI") + "\n\n"

		for i, choice := range m.choices {
			cursor := "  "
			style := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
			if m.cursor == i {
				cursor = "> "
				style = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
			}
			s += fmt.Sprintf("%s %s\n", cursor, style.Render(choice.label))
		}
		s += "\nPress q to quit."
		return s

	case screenRaoul:
		s := lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Render(animatedRaoul(m.frame))
		s += "\n\nPress Esc or q to go back."
		return s

	case screenStats:
		s := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).Render("Stats Dashboard") + "\n\n"
		if m.stats == nil {
			s += "No build has been run yet in this session.\n"
		} else {
			s += fmt.Sprintf("Last Build Time: %d ms\n", m.stats.Duration.Milliseconds())
			s += fmt.Sprintf("Total Pages Generated: %d\n", m.stats.PageCount)
			s += fmt.Sprintf("Error Count: %d\n", m.stats.ErrorCount)
		}
		s += "\nRAG Token Estimations:\n"
		ragDir := m.cfg.RagDir
		if ragDir == "" {
			ragDir = "rag-archive"
		}
		totalTokens := 0
		files, err := os.ReadDir(ragDir)
		if err == nil {
			for _, file := range files {
				if !file.IsDir() && strings.HasSuffix(file.Name(), ".md") {
					info, err := file.Info()
					if err == nil {
						size := info.Size()
						tokens := size / bytesPerToken
						totalTokens += int(tokens)
						s += fmt.Sprintf("- %s: ~%d tokens\n", file.Name(), tokens)
					}
				}
			}
			s += fmt.Sprintf("\nTotal Estimated Tokens: ~%d (Note: 1 token ≈ 4 bytes)\n", totalTokens)
		} else {
			s += "RAG archive not found. Run 'RAG Export' to generate bundles.\n"
		}
		s += "\nPress Esc or q to go back."
		return s

	case screenWorking:
		s := m.workMsg + "\n"
		if m.workErr != nil {
			s += lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(fmt.Sprintf("Error: %v", m.workErr)) + "\n"
		} else if strings.Contains(m.workMsg, "complete") {
			s += lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render("Success!") + "\n"
		}
		s += "\nPress Enter or Esc to return to the menu."
		return s

	case screenServe:
		port := m.cfg.Port
		if port == 0 {
			port = config.DefaultConfig().Port
		}
		s := lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Render(animatedRaoul(m.frame))
		s += "\n\n"
		s += lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true).Render(fmt.Sprintf("Serving site on http://localhost:%d", port))
		s += "\n\nPress Esc or q to stop serving and go back."
		return s
	}

	return "Unknown screen"
}

func staticRaoul() string {
	return `       .---.
      ( o o )
       \_-_/
      / | | \
     / / \ \ \`
}

func animatedRaoul(frame int) string {
	if frame == 0 {
		return `       .---.
      ( o o )
       \_-_/
      / | | \
     / / \ \ \`
	}
	return `       .---.
      ( - - )
       \_-_/
      \ \ / /
       \ | | /`
}

</content>
</file>

<file path="go.mod">
<content>
module github.com/tbuddy/la-famille

go 1.24.0

toolchain go1.24.3

require (
	github.com/adrg/frontmatter v0.2.0
	github.com/charmbracelet/bubbletea v1.3.10
	github.com/charmbracelet/lipgloss v1.1.0
	github.com/fsnotify/fsnotify v1.10.1
	github.com/microcosm-cc/bluemonday v1.0.27
	github.com/spf13/cobra v1.10.2
	github.com/yuin/goldmark v1.8.2
	gopkg.in/yaml.v2 v2.4.0
)

require (
	github.com/BurntSushi/toml v0.3.1 // indirect
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/aymerick/douceur v0.2.0 // indirect
	github.com/charmbracelet/colorprofile v0.2.3-0.20250311203215-f60798e515dc // indirect
	github.com/charmbracelet/x/ansi v0.10.1 // indirect
	github.com/charmbracelet/x/cellbuf v0.0.13-0.20250311204145-2c3ea96c31dd // indirect
	github.com/charmbracelet/x/term v0.2.1 // indirect
	github.com/erikgeiser/coninput v0.0.0-20211004153227-1c3628e74d0f // indirect
	github.com/gorilla/css v1.0.1 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-localereader v0.0.1 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/muesli/ansi v0.0.0-20230316100256-276c6243b2f6 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/muesli/termenv v0.16.0 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	golang.org/x/net v0.26.0 // indirect
	golang.org/x/sys v0.36.0 // indirect
	golang.org/x/text v0.16.0 // indirect
)

</content>
</file>

<file path="go.sum">
<content>
github.com/BurntSushi/toml v0.3.1 h1:WXkYYl6Yr3qBf1K79EBnL4mak0OimBfB0XUf9Vl28OQ=
github.com/BurntSushi/toml v0.3.1/go.mod h1:xHWCNGjB5oqiDr8zfno3MHue2Ht5sIBksp03qcyfWMU=
github.com/adrg/frontmatter v0.2.0 h1:/DgnNe82o03riBd1S+ZDjd43wAmC6W35q67NHeLkPd4=
github.com/adrg/frontmatter v0.2.0/go.mod h1:93rQCj3z3ZlwyxxpQioRKC1wDLto4aXHrbqIsnH9wmE=
github.com/aymanbagabas/go-osc52/v2 v2.0.1 h1:HwpRHbFMcZLEVr42D4p7XBqjyuxQH5SMiErDT4WkJ2k=
github.com/aymanbagabas/go-osc52/v2 v2.0.1/go.mod h1:uYgXzlJ7ZpABp8OJ+exZzJJhRNQ2ASbcXHWsFqH8hp8=
github.com/aymerick/douceur v0.2.0 h1:Mv+mAeH1Q+n9Fr+oyamOlAkUNPWPlA8PPGR0QAaYuPk=
github.com/aymerick/douceur v0.2.0/go.mod h1:wlT5vV2O3h55X9m7iVYN0TBM0NH/MmbLnd30/FjWUq4=
github.com/charmbracelet/bubbletea v1.3.10 h1:otUDHWMMzQSB0Pkc87rm691KZ3SWa4KUlvF9nRvCICw=
github.com/charmbracelet/bubbletea v1.3.10/go.mod h1:ORQfo0fk8U+po9VaNvnV95UPWA1BitP1E0N6xJPlHr4=
github.com/charmbracelet/colorprofile v0.2.3-0.20250311203215-f60798e515dc h1:4pZI35227imm7yK2bGPcfpFEmuY1gc2YSTShr4iJBfs=
github.com/charmbracelet/colorprofile v0.2.3-0.20250311203215-f60798e515dc/go.mod h1:X4/0JoqgTIPSFcRA/P6INZzIuyqdFY5rm8tb41s9okk=
github.com/charmbracelet/lipgloss v1.1.0 h1:vYXsiLHVkK7fp74RkV7b2kq9+zDLoEU4MZoFqR/noCY=
github.com/charmbracelet/lipgloss v1.1.0/go.mod h1:/6Q8FR2o+kj8rz4Dq0zQc3vYf7X+B0binUUBwA0aL30=
github.com/charmbracelet/x/ansi v0.10.1 h1:rL3Koar5XvX0pHGfovN03f5cxLbCF2YvLeyz7D2jVDQ=
github.com/charmbracelet/x/ansi v0.10.1/go.mod h1:3RQDQ6lDnROptfpWuUVIUG64bD2g2BgntdxH0Ya5TeE=
github.com/charmbracelet/x/cellbuf v0.0.13-0.20250311204145-2c3ea96c31dd h1:vy0GVL4jeHEwG5YOXDmi86oYw2yuYUGqz6a8sLwg0X8=
github.com/charmbracelet/x/cellbuf v0.0.13-0.20250311204145-2c3ea96c31dd/go.mod h1:xe0nKWGd3eJgtqZRaN9RjMtK7xUYchjzPr7q6kcvCCs=
github.com/charmbracelet/x/term v0.2.1 h1:AQeHeLZ1OqSXhrAWpYUtZyX1T3zVxfpZuEQMIQaGIAQ=
github.com/charmbracelet/x/term v0.2.1/go.mod h1:oQ4enTYFV7QN4m0i9mzHrViD7TQKvNEEkHUMCmsxdUg=
github.com/cpuguy83/go-md2man/v2 v2.0.6/go.mod h1:oOW0eioCTA6cOiMLiUPZOpcVxMig6NIQQ7OS05n1F4g=
github.com/erikgeiser/coninput v0.0.0-20211004153227-1c3628e74d0f h1:Y/CXytFA4m6baUTXGLOoWe4PQhGxaX0KpnayAqC48p4=
github.com/erikgeiser/coninput v0.0.0-20211004153227-1c3628e74d0f/go.mod h1:vw97MGsxSvLiUE2X8qFplwetxpGLQrlU1Q9AUEIzCaM=
github.com/fsnotify/fsnotify v1.10.1 h1:b0/UzAf9yR5rhf3RPm9gf3ehBPpf0oZKIjtpKrx59Ho=
github.com/fsnotify/fsnotify v1.10.1/go.mod h1:TLheqan6HD6GBK6PrDWyDPBaEV8LspOxvPSjC+bVfgo=
github.com/gorilla/css v1.0.1 h1:ntNaBIghp6JmvWnxbZKANoLyuXTPZ4cAMlo6RyhlbO8=
github.com/gorilla/css v1.0.1/go.mod h1:BvnYkspnSzMmwRK+b8/xgNPLiIuNZr6vbZBTPQ2A3b0=
github.com/inconshreveable/mousetrap v1.1.0 h1:wN+x4NVGpMsO7ErUn/mUI3vEoE6Jt13X2s0bqwp9tc8=
github.com/inconshreveable/mousetrap v1.1.0/go.mod h1:vpF70FUmC8bwa3OWnCshd2FqLfsEA9PFc4w1p2J65bw=
github.com/lucasb-eyer/go-colorful v1.2.0 h1:1nnpGOrhyZZuNyfu1QjKiUICQ74+3FNCN69Aj6K7nkY=
github.com/lucasb-eyer/go-colorful v1.2.0/go.mod h1:R4dSotOR9KMtayYi1e77YzuveK+i7ruzyGqttikkLy0=
github.com/mattn/go-isatty v0.0.20 h1:xfD0iDuEKnDkl03q4limB+vH+GxLEtL/jb4xVJSWWEY=
github.com/mattn/go-isatty v0.0.20/go.mod h1:W+V8PltTTMOvKvAeJH7IuucS94S2C6jfK/D7dTCTo3Y=
github.com/mattn/go-localereader v0.0.1 h1:ygSAOl7ZXTx4RdPYinUpg6W99U8jWvWi9Ye2JC/oIi4=
github.com/mattn/go-localereader v0.0.1/go.mod h1:8fBrzywKY7BI3czFoHkuzRoWE9C+EiG4R1k4Cjx5p88=
github.com/mattn/go-runewidth v0.0.16 h1:E5ScNMtiwvlvB5paMFdw9p4kSQzbXFikJ5SQO6TULQc=
github.com/mattn/go-runewidth v0.0.16/go.mod h1:Jdepj2loyihRzMpdS35Xk/zdY8IAYHsh153qUoGf23w=
github.com/microcosm-cc/bluemonday v1.0.27 h1:MpEUotklkwCSLeH+Qdx1VJgNqLlpY2KXwXFM08ygZfk=
github.com/microcosm-cc/bluemonday v1.0.27/go.mod h1:jFi9vgW+H7c3V0lb6nR74Ib/DIB5OBs92Dimizgw2cA=
github.com/muesli/ansi v0.0.0-20230316100256-276c6243b2f6 h1:ZK8zHtRHOkbHy6Mmr5D264iyp3TiX5OmNcI5cIARiQI=
github.com/muesli/ansi v0.0.0-20230316100256-276c6243b2f6/go.mod h1:CJlz5H+gyd6CUWT45Oy4q24RdLyn7Md9Vj2/ldJBSIo=
github.com/muesli/cancelreader v0.2.2 h1:3I4Kt4BQjOR54NavqnDogx/MIoWBFa0StPA8ELUXHmA=
github.com/muesli/cancelreader v0.2.2/go.mod h1:3XuTXfFS2VjM+HTLZY9Ak0l6eUKfijIfMUZ4EgX0QYo=
github.com/muesli/termenv v0.16.0 h1:S5AlUN9dENB57rsbnkPyfdGuWIlkmzJjbFf0Tf5FWUc=
github.com/muesli/termenv v0.16.0/go.mod h1:ZRfOIKPFDYQoDFF4Olj7/QJbW60Ol/kL1pU3VfY/Cnk=
github.com/rivo/uniseg v0.2.0/go.mod h1:J6wj4VEh+S6ZtnVlnTBMWIodfgj8LQOQFoIToxlJtxc=
github.com/rivo/uniseg v0.4.7 h1:WUdvkW8uEhrYfLC4ZzdpI2ztxP1I582+49Oc5Mq64VQ=
github.com/rivo/uniseg v0.4.7/go.mod h1:FN3SvrM+Zdj16jyLfmOkMNblXMcoc8DfTHruCPUcx88=
github.com/russross/blackfriday/v2 v2.1.0/go.mod h1:+Rmxgy9KzJVeS9/2gXHxylqXiyQDYRxCVz55jmeOWTM=
github.com/spf13/cobra v1.10.2 h1:DMTTonx5m65Ic0GOoRY2c16WCbHxOOw6xxezuLaBpcU=
github.com/spf13/cobra v1.10.2/go.mod h1:7C1pvHqHw5A4vrJfjNwvOdzYu0Gml16OCs2GRiTUUS4=
github.com/spf13/pflag v1.0.9 h1:9exaQaMOCwffKiiiYk6/BndUBv+iRViNW+4lEMi0PvY=
github.com/spf13/pflag v1.0.9/go.mod h1:McXfInJRrz4CZXVZOBLb0bTZqETkiAhM9Iw0y3An2Bg=
github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e h1:JVG44RsyaB9T2KIHavMF/ppJZNG9ZpyihvCd0w101no=
github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e/go.mod h1:RbqR21r5mrJuqunuUZ/Dhy/avygyECGrLceyNeo4LiM=
github.com/yuin/goldmark v1.8.2 h1:kEGpgqJXdgbkhcOgBxkC0X0PmoPG1ZyoZ117rDVp4zE=
github.com/yuin/goldmark v1.8.2/go.mod h1:ip/1k0VRfGynBgxOz0yCqHrbZXhcjxyuS66Brc7iBKg=
go.yaml.in/yaml/v3 v3.0.4/go.mod h1:DhzuOOF2ATzADvBadXxruRBLzYTpT36CKvDb3+aBEFg=
golang.org/x/exp v0.0.0-20220909182711-5c715a9e8561 h1:MDc5xs78ZrZr3HMQugiXOAkSZtfTpbJLDr/lwfgO53E=
golang.org/x/exp v0.0.0-20220909182711-5c715a9e8561/go.mod h1:cyybsKvd6eL0RnXn6p/Grxp8F5bW7iYuBgsNCOHpMYE=
golang.org/x/net v0.26.0 h1:soB7SVo0PWrY4vPW/+ay0jKDNScG2X9wFeYlXIvJsOQ=
golang.org/x/net v0.26.0/go.mod h1:5YKkiSynbBIh3p6iOc/vibscux0x38BZDkn8sCUPxHE=
golang.org/x/sys v0.0.0-20210809222454-d867a43fc93e/go.mod h1:oPkhp1MJrh7nUepCBck5+mAzfO9JrbApNNgaTdGDITg=
golang.org/x/sys v0.6.0/go.mod h1:oPkhp1MJrh7nUepCBck5+mAzfO9JrbApNNgaTdGDITg=
golang.org/x/sys v0.36.0 h1:KVRy2GtZBrk1cBYA7MKu5bEZFxQk4NIDV6RLVcC8o0k=
golang.org/x/sys v0.36.0/go.mod h1:OgkHotnGiDImocRcuBABYBEXf8A9a87e/uXjp9XT3ks=
golang.org/x/text v0.16.0 h1:a94ExnEXNtEwYLGJSIUxnWoxoRz/ZcCsV63ROupILh4=
golang.org/x/text v0.16.0/go.mod h1:GhwF1Be+LQoKShO3cGOHzqOgRrGaYc9AvblQOmPVHnI=
gopkg.in/check.v1 v0.0.0-20161208181325-20d25e280405 h1:yhCVgyC4o1eVCa2tZl7eS0r+SDo693bJlVdllGtEeKM=
gopkg.in/check.v1 v0.0.0-20161208181325-20d25e280405/go.mod h1:Co6ibVJAznAaIkqp8huTwlJQCZ016jof/cbN4VW5Yz0=
gopkg.in/yaml.v2 v2.3.0/go.mod h1:hI93XBmqTisBFMUTm0b8Fm+jr3Dg1NNxqwp+5A1VGuI=
gopkg.in/yaml.v2 v2.4.0 h1:D8xgwECY7CYvx+Y2n4sBz93Jn9JRvxdiyyo8CTfuKaY=
gopkg.in/yaml.v2 v2.4.0/go.mod h1:RDklbk79AGWmwhnvt/jBztapEOGDOx6ZbXqjP6csGnQ=

</content>
</file>

<file path="internal/asset/copy.go">
<content>
package asset

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/tbuddy/la-famille/internal/config"
)

// CopyAssets copies files from the configured AssetDir to OutputDir/assets,
// skipping testdata directories and checking for path traversal.
func CopyAssets(cfg config.Config) error {
	if cfg.AssetDir != "" {
		ignorePatterns := []string{}
		if gitignore, err := os.ReadFile(".gitignore"); err == nil {
			lines := strings.Split(string(gitignore), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line != "" && !strings.HasPrefix(line, "#") {
					// Normalize pattern for filepath.Match
					if strings.HasSuffix(line, "/") {
						line = strings.TrimSuffix(line, "/")
					}
					if strings.HasPrefix(line, "/") {
						line = strings.TrimPrefix(line, "/")
					}
					ignorePatterns = append(ignorePatterns, line)
					_ = ignorePatterns // use variable
				}
			}
		}

		if _, err := os.Stat(cfg.AssetDir); err == nil {
			targetDir := filepath.Join(cfg.OutputDir, "assets")
			if err := os.MkdirAll(targetDir, 0755); err != nil {
				return err
			}

			var paths []string
			err = filepath.WalkDir(cfg.AssetDir, func(path string, d os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if !d.IsDir() {
					paths = append(paths, path)
				}
				return nil
			})
			if err != nil {
				return err
			}

			// Batch check gitignore
			ignoredPaths := make(map[string]bool)
			if len(paths) > 0 {
				cmd := exec.Command("git", "check-ignore", "--stdin")
					projectRoot, _ := filepath.Abs(".")
					cmd.Dir = projectRoot
				cmd.Stdin = strings.NewReader(strings.Join(paths, "\n"))
				out, err := cmd.Output()
				if err != nil {
					if _, lookErr := exec.LookPath("git"); lookErr != nil {
						log.Printf("Warning: git not found in environment, skipping check-ignore")
					} else {
						var exitErr *exec.ExitError
						if errors.As(err, &exitErr) {
							// exit code 1 means none of the paths are ignored, which is a normal case
							// exit code 128 means outside repository, which happens in tests
								if exitErr.ExitCode() != 1 && exitErr.ExitCode() != 128 {
								log.Printf("Error running git check-ignore: %v (stderr: %q)", err, string(exitErr.Stderr))
							}
						} else {
							log.Printf("Error running git check-ignore: %v", err)
						}
					}
				}

				if len(out) > 0 {
					lines := strings.Split(strings.TrimSpace(string(out)), "\n")
					for _, line := range lines {
						if line != "" {
							// check-ignore returns absolute or relative paths depending on input. Since we passed relative, it should return relative.
							// let's use the exact string returned to populate the map.
							ignoredPaths[line] = true
						}
					}
				}
			}

			for _, path := range paths {
				if ignoredPaths[path] {
					continue
				}

				if filepath.Ext(path) == ".go" {
					continue
				}

				// Skip testdata in the path
				if strings.Contains(path, "/testdata/") || strings.Contains(path, "\\testdata\\") || strings.HasSuffix(path, "/testdata") || strings.HasSuffix(path, "\\testdata") {
					continue
				}

				relPath, err := filepath.Rel(cfg.AssetDir, path)
				if err != nil {
					return err
				}

				outDirClean := filepath.Clean(filepath.Join(cfg.OutputDir, "assets"))
				destPath := filepath.Join(outDirClean, filepath.FromSlash(relPath))
				if !strings.HasPrefix(destPath, outDirClean+string(filepath.Separator)) && destPath != outDirClean {
					log.Printf("Warning: Potential path traversal in asset copying detected: %s. Skipping.", relPath)
					continue
				}
				if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
					return err
				}

				if err := CopyFile(path, destPath); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// CopyFile streams the contents of src to dst using a buffer.
func CopyFile(src, dst string) (err error) {
	source, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer func() {
		cerr := destination.Close()
		if err == nil {
			err = cerr
		}
	}()

	if _, err = io.Copy(destination, source); err != nil {
		return fmt.Errorf("failed to copy data: %w", err)
	}

	// Ensure the write is flushed to disk
	if err = destination.Sync(); err != nil {
		return fmt.Errorf("failed to sync destination: %w", err)
	}

	return nil
}

</content>
</file>

<file path="internal/asset/copy_test.go">
<content>
package asset

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tbuddy/la-famille/internal/config"
)

func TestCopyAssets(t *testing.T) {
	tempDir := t.TempDir()

	assetDir := filepath.Join(tempDir, "assets")
	outputDir := filepath.Join(tempDir, "public")

	// Create asset dir and some files
	_ = os.MkdirAll(filepath.Join(assetDir, "css"), 0755)
	_ = os.MkdirAll(filepath.Join(assetDir, "testdata"), 0755)

	_ = os.WriteFile(filepath.Join(assetDir, "main.css"), []byte("body { color: red; }"), 0600)
	_ = os.WriteFile(filepath.Join(assetDir, "css", "style.css"), []byte("h1 { color: blue; }"), 0600)
	_ = os.WriteFile(filepath.Join(assetDir, "testdata", "ignore.txt"), []byte("ignore me"), 0600)

	cfg := config.Config{
		AssetDir:  assetDir,
		OutputDir: outputDir,
	}

	err := CopyAssets(cfg)
	if err != nil {
		t.Fatalf("CopyAssets failed: %v", err)
	}

	// Verify copied files
	if _, err := os.Stat(filepath.Join(outputDir, "assets", "main.css")); os.IsNotExist(err) {
		t.Errorf("main.css was not copied")
	}
	if _, err := os.Stat(filepath.Join(outputDir, "assets", "css", "style.css")); os.IsNotExist(err) {
		t.Errorf("style.css was not copied")
	}

	// Verify skipped testdata
	if _, err := os.Stat(filepath.Join(outputDir, "assets", "testdata")); !os.IsNotExist(err) {
		t.Errorf("testdata was copied, but should have been skipped")
	}
}

func TestCopyAssets_EmptyAssetDir(t *testing.T) {
	cfg := config.Config{
		AssetDir:  "",
		OutputDir: t.TempDir(),
	}
	err := CopyAssets(cfg)
	if err != nil {
		t.Errorf("Expected nil error for empty AssetDir, got: %v", err)
	}
}
func TestCopyAssets_SkipGoAndGitignore(t *testing.T) {
	tempDir := t.TempDir()

	assetDir := filepath.Join(tempDir, "assets")
	outputDir := filepath.Join(tempDir, "public")

	// Create asset dir and some files
	_ = os.MkdirAll(assetDir, 0755)

	_ = os.WriteFile(filepath.Join(assetDir, "main.go"), []byte("package main"), 0600)
	_ = os.WriteFile(filepath.Join(assetDir, "main.css"), []byte("body { color: red; }"), 0600)

	cfg := config.Config{
		AssetDir:  assetDir,
		OutputDir: outputDir,
	}

	err := CopyAssets(cfg)
	if err != nil {
		t.Fatalf("CopyAssets failed: %v", err)
	}

	// Verify copied files
	if _, err := os.Stat(filepath.Join(outputDir, "assets", "main.css")); os.IsNotExist(err) {
		t.Errorf("main.css was not copied")
	}
	if _, err := os.Stat(filepath.Join(outputDir, "assets", "main.go")); !os.IsNotExist(err) {
		t.Errorf("main.go was copied, but should have been skipped")
	}
}

func TestCopyAssetsGitNotAvailable(t *testing.T) {
	// Temporarily stub PATH to make git unavailable
	originalPath := os.Getenv("PATH")
	defer os.Setenv("PATH", originalPath)
	os.Setenv("PATH", "")

	tempDir := t.TempDir()

	assetDir := filepath.Join(tempDir, "assets")
	outputDir := filepath.Join(tempDir, "public")

	// Create asset dir and some files
	_ = os.MkdirAll(filepath.Join(assetDir, "css"), 0755)

	_ = os.WriteFile(filepath.Join(assetDir, "main.css"), []byte("body { color: red; }"), 0600)
	_ = os.WriteFile(filepath.Join(assetDir, "css", "style.css"), []byte("h1 { color: blue; }"), 0600)

	cfg := config.Config{
		AssetDir:  assetDir,
		OutputDir: outputDir,
	}

	err := CopyAssets(cfg)
	if err != nil {
		t.Fatalf("CopyAssets failed when git is not available: %v", err)
	}

	// Verify copied files even when git check-ignore is skipped
	if _, err := os.Stat(filepath.Join(outputDir, "assets", "main.css")); os.IsNotExist(err) {
		t.Errorf("main.css was not copied")
	}
	if _, err := os.Stat(filepath.Join(outputDir, "assets", "css", "style.css")); os.IsNotExist(err) {
		t.Errorf("style.css was not copied")
	}
}

</content>
</file>

<file path="internal/content/metadata.go">
<content>
package content

import (
	"bytes"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/adrg/frontmatter"
)

type FileMeta struct {
	RelPath         string
	Title           string
	Author          string
	Date            string
	Render          *bool
	VideoScript     string
	AnimationCues   string
	SoundtrackTheme string
	Layout          string
	ComplianceModal string
	Slug            string
	Tags            []string
	Content         []byte
	Rest            []byte // The content after frontmatter
	Description     string
	Image           string
}

// GatherMetadata walks the content directory and parses the frontmatter for each markdown file.
func GatherMetadata(contentDir string) (map[string]*FileMeta, error) {
	fileMap := make(map[string]*FileMeta)

	err := filepath.WalkDir(contentDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".md" {
			return nil
		}

		relPath, err := filepath.Rel(contentDir, path)
		if err != nil {
			return err
		}
		// Always use forward slashes for internal map keys to match web links
		relPath = filepath.ToSlash(relPath)

		contentBytes, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		// Parse into a generic map to normalize casing first
		var rawMatter map[string]interface{}
		rest, err := frontmatter.Parse(bytes.NewReader(contentBytes), &rawMatter)
		if err != nil {
			// If frontmatter parsing fails, treat the whole file as content
			rest = contentBytes
		}

		var matter struct {
			Title           string   `yaml:"title"`
			Author          string   `yaml:"author"`
			Date            string   `yaml:"date"`
			Render          *bool    `yaml:"render"`
			VideoScript     string   `yaml:"video_script"`
			AnimationCues   string   `yaml:"animation_cues"`
			SoundtrackTheme string   `yaml:"soundtrack_theme"`
			Layout          string   `yaml:"layout"`
			ComplianceModal string   `yaml:"compliance_modal"`
			Slug            string   `yaml:"slug"`
			Tags            []string `yaml:"tags"`
			Description     string   `yaml:"description"`
			Image           string   `yaml:"image"`
		}

		if rawMatter != nil {
			// Lowercase keys
			normalizedMatter := make(map[string]interface{})
			for k, v := range rawMatter {
				// Convert to lower case, but preserve underscores for things like video_script
				normalizedMatter[strings.ToLower(k)] = v
			}

			yamlBytes, err := yaml.Marshal(normalizedMatter)
			if err == nil {
				_ = yaml.Unmarshal(yamlBytes, &matter)
			}
		}

		// Date validation
		if matter.Date != "" {
			if _, err := time.Parse(time.DateOnly, matter.Date); err != nil {
				log.Printf("Warning: Invalid date format in %s: %s", relPath, matter.Date)
				matter.Date = ""
			}
		}

		// Tag validation and normalization
		var normalizedTags []string
		for _, tag := range matter.Tags {
			lower := strings.ToLower(tag)
			var sb strings.Builder
			for _, r := range lower {
				if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
					sb.WriteRune(r)
				}
			}
			normalized := sb.String()
			if normalized != tag {
				log.Printf("Warning: Normalized tag '%s' to '%s' in %s", tag, normalized, relPath)
			}
			if normalized != "" {
				normalizedTags = append(normalizedTags, normalized)
			}
		}

		fileMap[relPath] = &FileMeta{
			RelPath:         relPath,
			Title:           matter.Title,
			Author:          matter.Author,
			Date:            matter.Date,
			Render:          matter.Render,
			VideoScript:     matter.VideoScript,
			AnimationCues:   matter.AnimationCues,
			SoundtrackTheme: matter.SoundtrackTheme,
			Layout:          matter.Layout,
			ComplianceModal: matter.ComplianceModal,
			Slug:            matter.Slug,
			Tags:            normalizedTags,
			Content:         contentBytes,
			Rest:            rest,
			Description:     matter.Description,
			Image:           matter.Image,
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk content directory: %w", err)
	}

	return fileMap, nil
}

</content>
</file>

<file path="internal/content/metadata_test.go">
<content>
package content

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGatherMetadata(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// 1. Create a markdown file with frontmatter
	mdWithFrontmatter := `---
title: "Test Title"
author: "Test Author"
---
# Content here
`
	if err := os.WriteFile(filepath.Join(tmpDir, "frontmatter.md"), []byte(mdWithFrontmatter), 0600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// 2. Create a markdown file without frontmatter
	mdWithoutFrontmatter := `# Just content`
	if err := os.WriteFile(filepath.Join(tmpDir, "no_frontmatter.md"), []byte(mdWithoutFrontmatter), 0600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// 3. Create a non-markdown file
	txtFile := `Just a text file`
	if err := os.WriteFile(filepath.Join(tmpDir, "ignore.txt"), []byte(txtFile), 0600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// 4. Create a nested directory with a markdown file
	nestedDir := filepath.Join(tmpDir, "nested")
	if err := os.Mkdir(nestedDir, 0755); err != nil {
		t.Fatalf("failed to create nested dir: %v", err)
	}
	nestedMd := `---
title: "Nested File"
---
# Nested content
`
	if err := os.WriteFile(filepath.Join(nestedDir, "nested.md"), []byte(nestedMd), 0600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Run GatherMetadata
	fileMap, err := GatherMetadata(tmpDir)
	if err != nil {
		t.Fatalf("GatherMetadata returned error: %v", err)
	}

	// Verify results
	if len(fileMap) != 3 {
		t.Errorf("expected 3 files in map, got %d", len(fileMap))
	}

	// Check frontmatter.md
	fmFile, ok := fileMap["frontmatter.md"]
	if !ok {
		t.Errorf("frontmatter.md missing from map")
	} else {
		if fmFile.Title != "Test Title" {
			t.Errorf("expected title 'Test Title', got '%s'", fmFile.Title)
		}
		if fmFile.Author != "Test Author" {
			t.Errorf("expected author 'Test Author', got '%s'", fmFile.Author)
		}
		if string(fmFile.Rest) != "# Content here\n" {
			t.Errorf("expected rest content '# Content here\\n', got '%s'", string(fmFile.Rest))
		}
	}

	// Check no_frontmatter.md
	noFmFile, ok := fileMap["no_frontmatter.md"]
	if !ok {
		t.Errorf("no_frontmatter.md missing from map")
	} else {
		if noFmFile.Title != "" {
			t.Errorf("expected empty title, got '%s'", noFmFile.Title)
		}
		if string(noFmFile.Rest) != "# Just content" {
			t.Errorf("expected rest content '# Just content', got '%s'", string(noFmFile.Rest))
		}
	}

	// Check nested.md
	nestedFile, ok := fileMap["nested/nested.md"]
	if !ok {
		t.Errorf("nested/nested.md missing from map")
	} else {
		if nestedFile.Title != "Nested File" {
			t.Errorf("expected title 'Nested File', got '%s'", nestedFile.Title)
		}
	}

	// Check that text file was ignored
	if _, ok := fileMap["ignore.txt"]; ok {
		t.Errorf("ignore.txt should not be in map")
	}

	t.Run("Mixed case frontmatter", func(t *testing.T) {
		content := `---
Title: "Uppercase Title"
author: "lowercase author"
Render: false
---
Some body text.`
		fileName := "mixed.md"
		if err := os.WriteFile(filepath.Join(tmpDir, fileName), []byte(content), 0600); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		fileMap, err := GatherMetadata(tmpDir)
		if err != nil {
			t.Fatalf("GatherMetadata failed: %v", err)
		}

		meta, ok := fileMap["mixed.md"]
		if !ok {
			t.Fatalf("Expected 'mixed.md' in fileMap, got none")
		}

		if meta.Title != "Uppercase Title" {
			t.Errorf("Expected Title to be 'Uppercase Title', got '%s'", meta.Title)
		}
		if meta.Author != "lowercase author" {
			t.Errorf("Expected Author to be 'lowercase author', got '%s'", meta.Author)
		}
		if meta.Render == nil || *meta.Render != false {
			t.Errorf("Expected Render to be false, got %v", meta.Render)
		}
	})

	t.Run("All uppercase frontmatter", func(t *testing.T) {
		content := `---
TITLE: "All Uppercase Title"
AUTHOR: "UPPERCASE AUTHOR"
DATE: "2024-01-01"
RENDER: false
LAYOUT: "blog"
---
Uppercase body.`
		fileName := "uppercase.md"
		if err := os.WriteFile(filepath.Join(tmpDir, fileName), []byte(content), 0600); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		fileMap, err := GatherMetadata(tmpDir)
		if err != nil {
			t.Fatalf("GatherMetadata failed: %v", err)
		}

		meta, ok := fileMap["uppercase.md"]
		if !ok {
			t.Fatalf("Expected 'uppercase.md' in fileMap, got none")
		}

		if meta.Title != "All Uppercase Title" {
			t.Errorf("Expected Title to be 'All Uppercase Title', got '%s'", meta.Title)
		}
		if meta.Author != "UPPERCASE AUTHOR" {
			t.Errorf("Expected Author to be 'UPPERCASE AUTHOR', got '%s'", meta.Author)
		}
		if meta.Date != "2024-01-01" {
			t.Errorf("Expected Date to be '2024-01-01', got '%s'", meta.Date)
		}
		if meta.Render == nil || *meta.Render != false {
			t.Errorf("Expected Render to be false, got %v", meta.Render)
		}
		if meta.Layout != "blog" {
			t.Errorf("Expected Layout to be 'blog', got '%s'", meta.Layout)
		}
	})

}

func TestGatherMetadataValidation(t *testing.T) {
	tmpDir := t.TempDir()

	mdContent := `---
title: "Test Title"
tags: ["Valid-Tag", "Inv@lid_Tag"]
date: "invalid-date"
---
Content
`
	if err := os.WriteFile(filepath.Join(tmpDir, "test.md"), []byte(mdContent), 0600); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	fileMap, err := GatherMetadata(tmpDir)
	if err != nil {
		t.Fatalf("GatherMetadata failed: %v", err)
	}

	meta, ok := fileMap["test.md"]
	if !ok {
		t.Fatalf("Expected test.md in fileMap")
	}

	if meta.Date != "" {
		t.Errorf("Expected date to be cleared due to invalid format, got: %s", meta.Date)
	}

	if len(meta.Tags) != 2 {
		t.Fatalf("Expected 2 tags, got %d", len(meta.Tags))
	}
	if meta.Tags[0] != "valid-tag" {
		t.Errorf("Expected tag 0 to be 'valid-tag', got: %s", meta.Tags[0])
	}
	if meta.Tags[1] != "invlidtag" {
		t.Errorf("Expected tag 1 to be 'invlidtag', got: %s", meta.Tags[1])
	}
}

</content>
</file>

<file path="internal/generator/generator.go">
<content>
package generator

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"

	"github.com/tbuddy/la-famille/internal/asset"
	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/content"
	"github.com/tbuddy/la-famille/internal/graph"
	"github.com/tbuddy/la-famille/internal/markdown"
	"github.com/tbuddy/la-famille/internal/page"
	"github.com/tbuddy/la-famille/internal/render"
	"github.com/tbuddy/la-famille/internal/search"
	"github.com/tbuddy/la-famille/internal/sitedata"
	"github.com/tbuddy/la-famille/internal/stub"
	"github.com/tbuddy/la-famille/internal/taxonomy"
	"github.com/tbuddy/la-famille/internal/transform"
)

// convertMarkdown is a variable to allow mocking in tests.
var convertMarkdown = func(md goldmark.Markdown, source []byte, w *bytes.Buffer) error {
	return md.Convert(source, w)
}

// BuildResult contains statistics about the build process.
type BuildResult struct {
	Duration   time.Duration
	PageCount  int
	ErrorCount int
}

// Build generates the static site based on the given configuration.
func Build(cfg config.Config) (BuildResult, error) {
	start := time.Now()
	var result BuildResult

	// 1. Pass 1: Walk content dir and gather metadata
	fileMap, err := content.GatherMetadata(cfg.ContentDir)
	if err != nil {
		return result, fmt.Errorf("failed to gather metadata: %w", err)
	}

	// Track missing files that need stubs. map[missingPath][]parentFiles
	missingFiles := make(map[string][]string)
	backlinks := make(map[string][]string)
	g := graph.Graph{
		Nodes: make(map[string]graph.Node),
		Edges: [][2]string{},
	}
	metaData := make(map[string]map[string]interface{})
	var searchIndex []search.Item

	// 2. Pass 2: Process files in deterministic order
	var keys []string
	for k := range fileMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Reusable buffer for markdown conversion
	renderer := render.New(filepath.Dir(cfg.Template))

	var errs []error

	p := bluemonday.UGCPolicy()
	p.AllowAttrs("class").Globally()
	p.AllowElements("svg", "path")
	p.AllowAttrs("xmlns", "fill", "viewBox", "stroke-linecap", "stroke-linejoin", "stroke-width", "d", "stroke", "class").OnElements("svg", "path")

	if err := taxonomy.GenerateTags(cfg, fileMap, renderer, p); err != nil {
		return result, err
	}

	var mu sync.Mutex
	numWorkers := runtime.NumCPU()
	if numWorkers < 1 {
		numWorkers = 1
	}

	searchIndexItems := make([]search.Item, len(keys))

	type job struct {
		index   int
		relPath string
	}

	jobs := make(chan job, len(keys))
	for i, k := range keys {
		jobs <- job{index: i, relPath: k}
	}
	close(jobs)

	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var buf bytes.Buffer
			for j := range jobs {
				relPath := j.relPath
				idx := j.index
				meta := fileMap[relPath]
				shouldRender := true
				if meta.Render != nil && !*meta.Render {
					shouldRender = false
				}

				id := strings.TrimSuffix(relPath, ".md")

				mu.Lock()
				g.Nodes[id] = graph.Node{
					Type:   "page",
					Render: shouldRender,
				}
				mu.Unlock()

				m := make(map[string]interface{})
				title := meta.Title
				if title == "" {
					title = filepath.Base(relPath)
				}
				m["title"] = title
				if meta.Author != "" {
					m["author"] = meta.Author
				}
				if meta.Date != "" {
					m["date"] = meta.Date
				}
				if meta.Tags != nil {
					m["tags"] = meta.Tags
				}
				m["word_count"] = len(strings.Fields(string(meta.Rest)))

				mu.Lock()
				metaData[id] = m
				mu.Unlock()

				if shouldRender {
					urlOut := transform.GetOutputURL(relPath, meta.Slug)
					urlPath := "/" + filepath.ToSlash(urlOut)

					searchIndexItems[idx] = search.Item{
						Title:   title,
						URL:     urlPath,
						Tags:    meta.Tags,
						Snippet: search.ExtractSnippet(meta.Rest),
					}
				}

				outDirClean := filepath.Clean(cfg.OutputDir)
				outPath := filepath.Join(outDirClean, filepath.FromSlash(relPath))
				if !strings.HasPrefix(outPath, outDirClean+string(filepath.Separator)) && outPath != outDirClean {
					mu.Lock()
					result.ErrorCount++
					mu.Unlock()
					log.Printf("Warning: Potential path traversal in page loading detected: %s. Skipping.", relPath)
					continue
				}
				if shouldRender {
					slug := meta.Slug
					if slug != "" {
						if !filepath.IsLocal(slug) || strings.Contains(slug, ".") || strings.Contains(slug, string(filepath.Separator)) || strings.Contains(slug, "/") {
							log.Printf("Warning: Invalid slug %q for %s. Ignoring.", slug, relPath)
							slug = ""
						}
					}
					relOut := transform.GetOutputURL(relPath, slug)
					outPath = filepath.Join(outDirClean, filepath.FromSlash(relOut))
				}

				if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
					mu.Lock()
					errs = append(errs, err)
					mu.Unlock()
					continue
				}

				if !shouldRender {
					// Just copy the file
					if err := os.WriteFile(outPath, meta.Content, 0600); err != nil {
						mu.Lock()
						errs = append(errs, err)
						mu.Unlock()
					}
					continue
				}

				// Set up goldmark with AST transformer
				transformer := &transform.LinkTransformer{
					CurrentFile:  relPath,
					FileMap:      fileMap,
					MissingFiles: missingFiles,
					Backlinks:    backlinks,
					Graph:        &g,
					Mu:           &mu,
				}

				md := markdown.NewEngine(transformer)

				buf.Reset()
				if err := convertMarkdown(md, meta.Rest, &buf); err != nil {
					mu.Lock()
					result.ErrorCount++
					errs = append(errs, fmt.Errorf("error converting %s: %w", relPath, err))
					mu.Unlock()
					continue
				}

				sanitizedHTML := p.SanitizeBytes(buf.Bytes())

				desc := meta.Description
				if desc == "" {
					desc = cfg.DefaultDescription
				}
				img := meta.Image
				if img == "" {
					img = cfg.DefaultOGImage
				}

				page := page.Page{
					Site:            cfg,
					Title:           title,
					Author:          meta.Author,
					Date:            meta.Date,
					VideoScript:     meta.VideoScript,
					AnimationCues:   meta.AnimationCues,
					SoundtrackTheme: meta.SoundtrackTheme,
					Layout:          meta.Layout,
					ComplianceModal: meta.ComplianceModal,
					Content:         template.HTML(sanitizedHTML), // #nosec G203
					Description:     desc,
					Image:           img,
				}

				if err := renderer.HTML(cfg, page, meta.Layout, outPath); err != nil {
					mu.Lock()
					errs = append(errs, err)
					mu.Unlock()
					continue
				}
				mu.Lock()
				result.PageCount++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	for _, item := range searchIndexItems {
		if item.URL != "" {
			searchIndex = append(searchIndex, item)
		}
	}

	// Sort searchIndex, edges, and other outputs to ensure deterministic output
	sort.SliceStable(g.Edges, func(i, j int) bool {
		return g.Edges[i][0] < g.Edges[j][0]
	})

	for k := range backlinks {
		sort.Strings(backlinks[k])
	}

	// Sort errs for deterministic order
	if len(errs) > 0 {
		sort.Slice(errs, func(i, j int) bool {
			return errs[i].Error() < errs[j].Error()
		})
	}
	if len(errs) > 0 {
		return result, errors.Join(errs...)
	}
	// 3. Generate stubs for missing files in deterministic order
	if err := stub.GenerateStubs(cfg, missingFiles, &g, p, fileMap); err != nil {
		return result, err
	}

	// 4. Verbatim Asset Copy Step
	if err := asset.CopyAssets(cfg); err != nil {
		return result, err
	}

	// Write graph structures via internal/graph
	// 5. Write JSON outputs
	if err := graph.WriteGraphFiles(cfg.OutputDir, g, backlinks); err != nil {
		return result, err
	}

	if err := sitedata.Write(cfg.OutputDir, metaData); err != nil {
		return result, err
	}

	if err := search.WriteMinifiedJSON(filepath.Join(cfg.OutputDir, "search.json"), searchIndex); err != nil {
		return result, err
	}

	result.Duration = time.Since(start)
	return result, nil
}

</content>
</file>

<file path="internal/generator/generator_test.go">
<content>
package generator

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/yuin/goldmark"
)

func TestBuild_MarkdownConversionError(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	outputDir := filepath.Join(tempDir, "public")
	templateDir := filepath.Join(tempDir, "templates")

	_ = os.MkdirAll(contentDir, 0755)
	_ = os.MkdirAll(templateDir, 0755)

	templatePath := filepath.Join(templateDir, "layout.html")
	_ = os.WriteFile(templatePath, []byte("{{.Content}}"), 0600)

	cfg := config.DefaultConfig()
	cfg.ContentDir = contentDir
	cfg.OutputDir = outputDir
	cfg.Template = templatePath

	_ = os.WriteFile(filepath.Join(contentDir, "test1.md"), []byte("# Hello 1"), 0600)
	_ = os.WriteFile(filepath.Join(contentDir, "test2.md"), []byte("# Hello 2"), 0600)

	// Mock convertMarkdown to always fail
	originalConvert := convertMarkdown
	defer func() { convertMarkdown = originalConvert }()

	convertMarkdown = func(_ goldmark.Markdown, _ []byte, _ *bytes.Buffer) error {
		return errors.New("simulated conversion error")
	}

	res, err := Build(cfg)
	if err == nil {
		t.Fatalf("expected error from Build, got nil")
	}

	if !strings.Contains(err.Error(), "simulated conversion error") {
		t.Errorf("expected error string to contain 'simulated conversion error', got: %v", err)
	}

	if res.ErrorCount != 2 {
		t.Errorf("expected 2 errors, got %d", res.ErrorCount)
	}
}

func TestBuild_Success(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	outputDir := filepath.Join(tempDir, "public")
	templateDir := filepath.Join(tempDir, "templates")

	_ = os.MkdirAll(contentDir, 0755)
	_ = os.MkdirAll(templateDir, 0755)

	templatePath := filepath.Join(templateDir, "layout.html")
	_ = os.WriteFile(templatePath, []byte("{{.Content}}"), 0600)

	cfg := config.DefaultConfig()
	cfg.ContentDir = contentDir
	cfg.OutputDir = outputDir
	cfg.Template = templatePath

	_ = os.WriteFile(filepath.Join(contentDir, "test.md"), []byte("# Hello"), 0600)

	_, err := Build(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGeneratorSEO(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	outDir := filepath.Join(tempDir, "public")
	assetDir := filepath.Join(tempDir, "assets")
	ragDir := filepath.Join(tempDir, "rag-archive")
	tmplDir := filepath.Join(tempDir, "templates")

	_ = os.MkdirAll(contentDir, 0755)
	_ = os.MkdirAll(outDir, 0755)
	_ = os.MkdirAll(assetDir, 0755)
	_ = os.MkdirAll(ragDir, 0755)
	_ = os.MkdirAll(tmplDir, 0755)

	tmplPath := filepath.Join(tmplDir, "layout.html")
	tmplContent := `<!DOCTYPE html><html><head><title>{{.Title}}</title><meta name="description" content="{{.Description}}"><meta property="og:image" content="{{.Image}}"></head><body>{{.Content}}</body></html>`
	if err := os.WriteFile(tmplPath, []byte(tmplContent), 0600); err != nil {
		t.Fatalf("failed to write template: %v", err)
	}

	mdContent := `---
title: Test SEO
description: "Test SEO Description"
image: "/images/test-seo.png"
---
# Hello SEO`
	mdPath := filepath.Join(contentDir, "test.md")
	if err := os.WriteFile(mdPath, []byte(mdContent), 0600); err != nil {
		t.Fatalf("failed to write markdown file: %v", err)
	}

	cfg := config.Config{
		SiteName:           "Test Site",
		Template:           tmplPath,
		ContentDir:         contentDir,
		OutputDir:          outDir,
		AssetDir:           assetDir,
		RagDir:             ragDir,
		Theme:              "retro",
		Port:               8080,
		DefaultDescription: "Default Desc",
		DefaultOGImage:     "/default.png",
	}

	_, err := Build(cfg)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	outHTMLPath := filepath.Join(outDir, "test", "index.html")
	outBytes, err := os.ReadFile(outHTMLPath)
	if err != nil {
		t.Fatalf("failed to read output HTML: %v", err)
	}

	outHTML := string(outBytes)

	expectedDesc := `<meta name="description" content="Test SEO Description">`
	if !strings.Contains(outHTML, expectedDesc) {
		t.Errorf("output HTML missing expected description meta tag.\nGot: %s", outHTML)
	}

	expectedImage := `<meta property="og:image" content="/images/test-seo.png">`
	if !strings.Contains(outHTML, expectedImage) {
		t.Errorf("output HTML missing expected image meta tag.\nGot: %s", outHTML)
	}
}

</content>
</file>

<file path="internal/git/git.go">
<content>
package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// HasUncommittedChanges returns true if there are uncommitted changes in the working directory.
func HasUncommittedChanges() (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return false, fmt.Errorf("failed to check git status: %w", err)
	}
	return strings.TrimSpace(out.String()) != "", nil
}

// GetRemoteURL returns the URL of the specified remote (usually "origin").
func GetRemoteURL(remote string) (string, error) {
	cmd := exec.Command("git", "remote", "get-url", remote)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to get remote url for %s: %w", remote, err)
	}
	return strings.TrimSpace(out.String()), nil
}

// ParseOwnerRepo extracts the owner and repository name from a git remote URL.
// It handles both HTTPS and SSH formats.
func ParseOwnerRepo(url string) (string, string, error) {
	// Examples:
	// https://github.com/owner/repo.git
	// git@github.com:owner/repo.git
	// https://github.com/owner/repo

	url = strings.TrimSuffix(url, ".git")

	var pathPart string
	if strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://") {
		// http(s)://github.com/owner/repo
		parts := strings.SplitN(url, "github.com/", 2)
		if len(parts) != 2 {
			return "", "", fmt.Errorf("could not parse HTTPS github URL: %s", url)
		}
		pathPart = parts[1]
	} else if strings.HasPrefix(url, "git@") {
		// git@github.com:owner/repo
		parts := strings.SplitN(url, ":", 2)
		if len(parts) != 2 {
			return "", "", fmt.Errorf("could not parse SSH github URL: %s", url)
		}
		pathPart = parts[1]
	} else {
		return "", "", fmt.Errorf("unsupported git remote URL format: %s", url)
	}

	parts := strings.SplitN(pathPart, "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("could not extract owner/repo from path: %s", pathPart)
	}

	return parts[0], parts[1], nil
}

// CheckoutBranch creates and checks out a new branch.
func CheckoutBranch(branchName string) error {
	cmd := exec.Command("git", "checkout", "-b", branchName)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to checkout branch %s: %s: %w", branchName, stderr.String(), err)
	}
	return nil
}

// AddAll stages all changes.
func AddAll() error {
	cmd := exec.Command("git", "add", ".")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to git add: %s: %w", stderr.String(), err)
	}
	return nil
}

// Commit creates a commit with the specified message and author.
func Commit(message string, authorName string, authorEmail string) error {
	author := fmt.Sprintf("%s <%s>", authorName, authorEmail)
	cmd := exec.Command("git", "commit", "-m", message, "--author", author)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to commit: %s: %w", stderr.String(), err)
	}
	return nil
}

// Push pushes the specified branch to the remote.
func Push(remote string, branchName string) error {
	// Set upstream so that the branch tracks correctly.
	cmd := exec.Command("git", "push", "--set-upstream", remote, branchName)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to push branch %s to %s: %s: %w", branchName, remote, stderr.String(), err)
	}
	return nil
}

</content>
</file>

<file path="internal/git/git_test.go">
<content>
package git

import (
	"testing"
)

func TestParseOwnerRepo(t *testing.T) {
	tests := []struct {
		url           string
		expectedOwner string
		expectedRepo  string
		expectError   bool
	}{
		{"https://github.com/tbuddy/la-famille.git", "tbuddy", "la-famille", false},
		{"https://github.com/tbuddy/la-famille", "tbuddy", "la-famille", false},
		{"http://github.com/owner/repo.git", "owner", "repo", false},
		{"git@github.com:tbuddy/la-famille.git", "tbuddy", "la-famille", false},
		{"git@github.com:owner/repo", "owner", "repo", false},
		{"https://gitlab.com/owner/repo", "", "", true},
		{"invalid-url", "", "", true},
	}

	for _, tt := range tests {
		owner, repo, err := ParseOwnerRepo(tt.url)
		if tt.expectError {
			if err == nil {
				t.Errorf("expected error for url %q, got none", tt.url)
			}
			continue
		}
		if err != nil {
			t.Errorf("unexpected error for url %q: %v", tt.url, err)
			continue
		}
		if owner != tt.expectedOwner {
			t.Errorf("for url %q expected owner %q, got %q", tt.url, tt.expectedOwner, owner)
		}
		if repo != tt.expectedRepo {
			t.Errorf("for url %q expected repo %q, got %q", tt.url, tt.expectedRepo, repo)
		}
	}
}

</content>
</file>

<file path="internal/github/github.go">
<content>
package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	Token      string
	Owner      string
	Repo       string
	HTTPClient *http.Client
}

func NewClient(token, owner, repo string) *Client {
	return &Client{
		Token: token,
		Owner: owner,
		Repo:  repo,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) doRequest(method, path string, body interface{}, response interface{}) error {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s%s", c.Owner, c.Repo, path)

	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if c.Token != "" {
		req.Header.Set("Authorization", "token "+c.Token)
	}
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: status=%d %s", resp.StatusCode, string(b))
	}

	if response != nil {
		if err := json.NewDecoder(resp.Body).Decode(response); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}
	return nil
}

type User struct {
	Login string `json:"login"`
}

type PullRequest struct {
	Number    int    `json:"number"`
	Title     string `json:"title"`
	State     string `json:"state"`
	User      User   `json:"user"`
	Head      Ref    `json:"head"`
	Mergeable *bool  `json:"mergeable"` // Using pointer because it can be null
}

type Ref struct {
	Ref string `json:"ref"`
	Sha string `json:"sha"`
}

// ListOpenPRs returns open pull requests, filtered by authors if provided.
func (c *Client) ListOpenPRs(authors []string) ([]PullRequest, error) {
	var prs []PullRequest
	// For simplicity, just get the first page. For a robust implementation, handle pagination.
	err := c.doRequest("GET", "/pulls?state=open", nil, &prs)
	if err != nil {
		return nil, err
	}

	if len(authors) == 0 {
		return prs, nil
	}

	var filtered []PullRequest
	authorMap := make(map[string]bool)
	for _, a := range authors {
		authorMap[strings.ToLower(a)] = true
	}

	for _, pr := range prs {
		if authorMap[strings.ToLower(pr.User.Login)] {
			filtered = append(filtered, pr)
		}
	}

	return filtered, nil
}

// GetPR fetches a single pull request by number.
// Useful to get the up-to-date `mergeable` status which might not be in the list view.
func (c *Client) GetPR(number int) (*PullRequest, error) {
	var pr PullRequest
	err := c.doRequest("GET", fmt.Sprintf("/pulls/%d", number), nil, &pr)
	if err != nil {
		return nil, err
	}
	return &pr, nil
}

type CheckRun struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
}

type CheckRunsResponse struct {
	TotalCount int        `json:"total_count"`
	CheckRuns  []CheckRun `json:"check_runs"`
}

// AreChecksPassing returns true if all check runs for the given ref are completed and successful/skipped.
func (c *Client) AreChecksPassing(ref string) (bool, error) {
	var resp CheckRunsResponse
	err := c.doRequest("GET", fmt.Sprintf("/commits/%s/check-runs", ref), nil, &resp)
	if err != nil {
		return false, err
	}

	if resp.TotalCount == 0 {
		// No checks defined means they technically didn't fail.
		// Depending on strictness, this might be considered passing.
		// For our purposes, we'll consider it true.
		return true, nil
	}

	for _, check := range resp.CheckRuns {
		if check.Status != "completed" {
			return false, nil
		}
		if check.Conclusion != "success" && check.Conclusion != "skipped" && check.Conclusion != "neutral" {
			return false, nil
		}
	}
	return true, nil
}

// ClosePR closes a pull request.
func (c *Client) ClosePR(number int) error {
	body := map[string]string{
		"state": "closed",
	}
	return c.doRequest("PATCH", fmt.Sprintf("/pulls/%d", number), body, nil)
}

// MergePR merges a pull request.
func (c *Client) MergePR(number int) error {
	// The API returns a response, but we don't strictly need to parse it unless we want to check merged status
	return c.doRequest("PUT", fmt.Sprintf("/pulls/%d/merge", number), nil, nil)
}

// CreatePR opens a new pull request.
func (c *Client) CreatePR(title, body, head, base string) error {
	reqBody := map[string]string{
		"title": title,
		"body":  body,
		"head":  head,
		"base":  base,
	}
	return c.doRequest("POST", "/pulls", reqBody, nil)
}

</content>
</file>

<file path="internal/github/github_test.go">
<content>
package github

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAreChecksPassing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/owner/repo/commits/sha123/check-runs" {
			resp := CheckRunsResponse{
				TotalCount: 2,
				CheckRuns: []CheckRun{
					{Status: "completed", Conclusion: "success"},
					{Status: "completed", Conclusion: "skipped"},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		if r.URL.Path == "/repos/owner/repo/commits/sha456/check-runs" {
			resp := CheckRunsResponse{
				TotalCount: 1,
				CheckRuns: []CheckRun{
					{Status: "in_progress"},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		if r.URL.Path == "/repos/owner/repo/commits/sha789/check-runs" {
			resp := CheckRunsResponse{
				TotalCount: 1,
				CheckRuns: []CheckRun{
					{Status: "completed", Conclusion: "failure"},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// Intercept the default HTTPClient used by Client to point to our test server
	c := NewClient("token", "owner", "repo")

	// Hack to replace base URL for tests:
	// We'll wrap the transport to redirect
	c.HTTPClient.Transport = &redirectTransport{
		baseURL: server.URL + "/repos/owner/repo",
	}

	t.Run("Passing checks", func(t *testing.T) {
		passing, err := c.AreChecksPassing("sha123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !passing {
			t.Errorf("expected passing=true, got false")
		}
	})

	t.Run("In progress checks", func(t *testing.T) {
		passing, err := c.AreChecksPassing("sha456")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if passing {
			t.Errorf("expected passing=false, got true")
		}
	})

	t.Run("Failed checks", func(t *testing.T) {
		passing, err := c.AreChecksPassing("sha789")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if passing {
			t.Errorf("expected passing=false, got true")
		}
	})
}

type redirectTransport struct {
	baseURL string
}

func (t *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// e.g. /repos/owner/repo/commits/... -> /commits/...
	path := req.URL.Path[len("/repos/owner/repo"):]
	urlStr := t.baseURL + path
	newReq, _ := http.NewRequest(req.Method, urlStr, req.Body)
	newReq.Header = req.Header

	return http.DefaultTransport.RoundTrip(newReq)
}

</content>
</file>

<file path="internal/github/sync.go">
<content>
package github

import (
	"fmt"
	"log"

	"time"

	"github.com/tbuddy/la-famille/internal/git"
)

// SyncConfig holds configuration for the PR sync process.
type SyncConfig struct {
	Token         string
	BotAuthors    []string
	DefaultBranch string
}

// RunSync executes the automated PR management routine.
func RunSync(cfg SyncConfig) error {
	if cfg.Token == "" {
		return fmt.Errorf("GITHUB_TOKEN is not set")
	}

	// 1. Infer owner/repo from git config
	remoteURL, err := git.GetRemoteURL("origin")
	if err != nil {
		return fmt.Errorf("failed to get git remote url: %w", err)
	}

	owner, repo, err := git.ParseOwnerRepo(remoteURL)
	if err != nil {
		return fmt.Errorf("failed to parse owner/repo from remote URL %s: %w", remoteURL, err)
	}

	client := NewClient(cfg.Token, owner, repo)
	log.Printf("Starting sync for %s/%s", owner, repo)

	// 2. Fetch and process existing PRs
	prs, err := client.ListOpenPRs(cfg.BotAuthors)
	if err != nil {
		return fmt.Errorf("failed to list PRs: %w", err)
	}

	log.Printf("Found %d open PRs authored by bots", len(prs))

	for _, pr := range prs {
		// We need to fetch the PR individually to reliably get the `mergeable` status.
		// The list endpoint sometimes omits it or caches old values.
		fullPR, err := client.GetPR(pr.Number)
		if err != nil {
			log.Printf("Failed to get details for PR #%d: %v", pr.Number, err)
			continue
		}

		if fullPR.Mergeable == nil {
			log.Printf("PR #%d mergeable status is computing (null), skipping for now", pr.Number)
			continue
		}

		if !*fullPR.Mergeable {
			log.Printf("PR #%d has conflicts (mergeable=false), closing", pr.Number)
			if err := client.ClosePR(pr.Number); err != nil {
				log.Printf("Failed to close PR #%d: %v", pr.Number, err)
			} else {
				log.Printf("Successfully closed PR #%d", pr.Number)
			}
			continue
		}

		// PR is mergeable, check CI status
		passing, err := client.AreChecksPassing(fullPR.Head.Sha)
		if err != nil {
			log.Printf("Failed to get check runs for PR #%d (sha: %s): %v", pr.Number, fullPR.Head.Sha, err)
			continue
		}

		if passing {
			log.Printf("PR #%d checks are passing and mergeable=true, merging", pr.Number)
			if err := client.MergePR(pr.Number); err != nil {
				log.Printf("Failed to merge PR #%d: %v", pr.Number, err)
			} else {
				log.Printf("Successfully merged PR #%d", pr.Number)
			}
		} else {
			log.Printf("PR #%d checks are not yet fully passing, skipping", pr.Number)
		}
	}

	// 3. Handle local changes
	hasChanges, err := git.HasUncommittedChanges()
	if err != nil {
		return fmt.Errorf("failed to check for uncommitted changes: %w", err)
	}

	if !hasChanges {
		log.Println("No local changes detected. Sync complete.")
		return nil
	}

	log.Println("Local changes detected. Creating a new automated PR.")
	timestamp := time.Now().Format("20060102150405")
	branchName := fmt.Sprintf("jules-auto-%s", timestamp)

	if err := git.CheckoutBranch(branchName); err != nil {
		return fmt.Errorf("failed to checkout branch: %w", err)
	}

	if err := git.AddAll(); err != nil {
		return fmt.Errorf("failed to stage changes: %w", err)
	}

	commitMsg := "chore: automated routine execution"
	if err := git.Commit(commitMsg, "google-labs-jules", "jules-bot@users.noreply.github.com"); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	log.Printf("Pushing branch %s...", branchName)
	if err := git.Push("origin", branchName); err != nil {
		return fmt.Errorf("failed to push: %w", err)
	}

	// Give GitHub a tiny moment to register the branch before creating the PR.
	time.Sleep(2 * time.Second)

	prTitle := fmt.Sprintf("Automated Routine Execution: %s", timestamp)
	prBody := "This PR was generated automatically by the la-famille GitHub sync feature to commit routine artifacts."

	baseBranch := cfg.DefaultBranch
	if baseBranch == "" {
		baseBranch = "main" // Default fallback
	}

	if err := client.CreatePR(prTitle, prBody, branchName, baseBranch); err != nil {
		return fmt.Errorf("failed to create PR: %w", err)
	}

	log.Printf("Successfully created PR for branch %s", branchName)

	// Switch back to original branch? Let's just stay here or we'd need to know what we were on.
	// For automation containers, it usually doesn't matter since it's transient.

	return nil
}

</content>
</file>

<file path="internal/graph/graph.go">
<content>
package graph

type Node struct {
	Type         string   `json:"type"`
	Render       bool     `json:"render"`
	Missing      bool     `json:"missing,omitempty"`
	ReferencedBy []string `json:"referenced_by,omitempty"`
}

type Graph struct {
	Nodes map[string]Node `json:"nodes"`
	Edges [][2]string     `json:"edges"`
}

</content>
</file>

<file path="internal/graph/write.go">
<content>
package graph

import (
	"path/filepath"
	"sort"

	"github.com/tbuddy/la-famille/internal/jsonutil"
)

// WriteGraphFiles writes the graph and backlinks data to the output directory.
func WriteGraphFiles(outputDir string, g Graph, backlinks map[string][]string) error {
	// Sort backlinks for deterministic output
	for _, parents := range backlinks {
		sort.Strings(parents)
	}

	if err := jsonutil.WriteJSON(filepath.Join(outputDir, "graph.json"), g); err != nil {
		return err
	}
	if err := jsonutil.WriteJSON(filepath.Join(outputDir, "backlinks.json"), backlinks); err != nil {
		return err
	}

	return nil
}

</content>
</file>

<file path="internal/graph/write_test.go">
<content>
package graph

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteGraphFiles(t *testing.T) {
	tempDir := t.TempDir()

	g := Graph{
		Nodes: map[string]Node{
			"index": {Type: "page", Render: true},
		},
		Edges: [][2]string{
			{"index", "about"},
		},
	}

	backlinks := map[string][]string{
		"about": {"index", "home", "a_test"}, // Intentionally unordered
	}

	err := WriteGraphFiles(tempDir, g, backlinks)
	if err != nil {
		t.Fatalf("WriteGraphFiles failed: %v", err)
	}

	// 1. Check graph.json
	graphContent, err := os.ReadFile(filepath.Join(tempDir, "graph.json"))
	if err != nil {
		t.Fatalf("Failed to read graph.json: %v", err)
	}
	var readGraph Graph
	if err := json.Unmarshal(graphContent, &readGraph); err != nil {
		t.Fatalf("Failed to parse graph.json: %v", err)
	}
	if len(readGraph.Nodes) != 1 || readGraph.Nodes["index"].Type != "page" {
		t.Errorf("Unexpected graph content: %+v", readGraph)
	}

	// 2. Check backlinks.json (and ensure sorting happened)
	backlinksContent, err := os.ReadFile(filepath.Join(tempDir, "backlinks.json"))
	if err != nil {
		t.Fatalf("Failed to read backlinks.json: %v", err)
	}
	var readBacklinks map[string][]string
	if err := json.Unmarshal(backlinksContent, &readBacklinks); err != nil {
		t.Fatalf("Failed to parse backlinks.json: %v", err)
	}

	aboutBacklinks := readBacklinks["about"]
	if len(aboutBacklinks) != 3 {
		t.Errorf("Expected 3 backlinks for 'about', got %d", len(aboutBacklinks))
	}
	if aboutBacklinks[0] != "a_test" || aboutBacklinks[1] != "home" || aboutBacklinks[2] != "index" {
		t.Errorf("Backlinks were not sorted correctly: %v", aboutBacklinks)
	}
}

</content>
</file>

<file path="internal/jsonutil/write.go">
<content>
package jsonutil

import (
	"encoding/json"
	"os"
)

// WriteJSON writes the given data to the specified path as a formatted JSON file.
func WriteJSON(path string, data interface{}) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

</content>
</file>

<file path="internal/jsonutil/write_test.go">
<content>
package jsonutil

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "test.json")

	type TestData struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	data := TestData{
		Name:  "Test",
		Value: 123,
	}

	err := WriteJSON(tempFile, data)
	if err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}

	fileContent, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}

	var readData TestData
	err = json.Unmarshal(fileContent, &readData)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if readData.Name != data.Name || readData.Value != data.Value {
		t.Errorf("Expected %+v, got %+v", data, readData)
	}
}

</content>
</file>

<file path="internal/markdown/markdown.go">
<content>
package markdown

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"

	"github.com/tbuddy/la-famille/internal/transform"
)

// NewEngine creates a new configured goldmark.Markdown instance
func NewEngine(transformer *transform.LinkTransformer) goldmark.Markdown {
	return goldmark.New(
		goldmark.WithParserOptions(
			parser.WithASTTransformers(
				util.Prioritized(transformer, 100),
			),
			parser.WithInlineParsers(
				util.Prioritized(&transform.EmojiKitchenParser{}, 100),
			),
		),
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
		),
	)
}

</content>
</file>

<file path="internal/markdown/markdown_test.go">
<content>
package markdown

import (
	"bytes"
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/transform"
)

func TestNewEngine(t *testing.T) {
	// Provide a dummy transformer
	transformer := &transform.LinkTransformer{}
	engine := NewEngine(transformer)

	if engine == nil {
		t.Fatal("expected engine to not be nil")
	}

	// Test a simple conversion to ensure it is configured properly
	source := []byte("# Hello World\n\nThis is a test.")
	var buf bytes.Buffer
	if err := engine.Convert(source, &buf); err != nil {
		t.Fatalf("failed to convert markdown: %v", err)
	}

	result := buf.String()
	if !strings.Contains(result, "<h1>Hello World</h1>") {
		t.Errorf("expected output to contain <h1>Hello World</h1>, got: %s", result)
	}
	if !strings.Contains(result, "<p>This is a test.</p>") {
		t.Errorf("expected output to contain <p>This is a test.</p>, got: %s", result)
	}
}

</content>
</file>

<file path="internal/page/page.go">
<content>
package page

import (
	"html/template"

	"github.com/tbuddy/la-famille/internal/config"
)

type Page struct {
	Site            config.Config
	Title           string
	Author          string
	Date            string
	VideoScript     string
	AnimationCues   string
	SoundtrackTheme string
	Layout          string
	ComplianceModal string
	Content         template.HTML
	Description     string
	Image           string
}

</content>
</file>

<file path="internal/ragexport/export.go">
<content>
package ragexport

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tbuddy/la-famille/internal/config"
)

// RunExport exports project files into RAG-friendly markdown bundles
func RunExport(cfg config.Config) error {
	outDir := cfg.RagDir
	if outDir == "" {
		outDir = "rag-archive"
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	fmt.Printf("RAG archive directory created at %s\n", outDir)

	// 1. System Bundle
	if err := writeBundle(
		filepath.Join(outDir, "rag-system.md"),
		[]string{
			"cmd/**/*.go",
			"internal/**/*.go",
			"pkg/**/*.go",
			"*.go",
			"go.mod",
			"go.sum",
			"README.md",
			"playwright_test.js",
			".github/workflows/*.yml",
		},
		[]string{"internal/config"},
		nil,
		outDir,
		cfg.ProjectRoot,
	); err != nil {
		return fmt.Errorf("failed to write system bundle: %w", err)
	}
	fmt.Println("Created rag-system.md")

	// 2. Config/Templates Bundle
	if err := writeBundle(
		filepath.Join(outDir, "rag-config.md"),
		[]string{
			"internal/config/**/*.go",
			".jules/**/*.md",
		},
		nil,
		nil,
		outDir,
		cfg.ProjectRoot,
	); err != nil {
		return fmt.Errorf("failed to write config bundle: %w", err)
	}

	// Append assets listing to Config/Templates Bundle
	cfgFile, err := os.OpenFile(filepath.Join(outDir, "rag-config.md"), os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open config bundle for appending assets: %w", err)
	}
	defer cfgFile.Close()

	_, _ = cfgFile.WriteString("<file path=\"assets/\">\n<content>\n")
	_ = filepath.WalkDir(filepath.Join(cfg.ProjectRoot, "assets"), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // ignore missing assets dir
		}
		// if it's a directory, just print the path with a trailing slash
		if d.IsDir() {
			_, _ = cfgFile.WriteString(filepath.ToSlash(getRel(cfg.ProjectRoot, path)) + "/\n")
		} else {
			// for files, print size and name
			info, err := d.Info()
			size := int64(0)
			if err == nil {
				size = info.Size()
			}
			_, _ = cfgFile.WriteString(fmt.Sprintf("%s (size: %d bytes)\n", filepath.ToSlash(getRel(cfg.ProjectRoot, path)), size))
		}
		return nil
	})
	_, _ = cfgFile.WriteString("</content>\n</file>\n\n")

	_, _ = cfgFile.WriteString("<file path=\"templates/\">\n<content>\n")
	_ = filepath.WalkDir(filepath.Join(cfg.ProjectRoot, "templates"), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // ignore missing templates dir
		}
		// if it's a directory, just print the path with a trailing slash
		if d.IsDir() {
			_, _ = cfgFile.WriteString(filepath.ToSlash(getRel(cfg.ProjectRoot, path)) + "/\n")
		} else {
			// for files, print size and name
			info, err := d.Info()
			size := int64(0)
			if err == nil {
				size = info.Size()
			}
			_, _ = cfgFile.WriteString(fmt.Sprintf("%s (size: %d bytes)\n", filepath.ToSlash(getRel(cfg.ProjectRoot, path)), size))
		}
		return nil
	})
	_, _ = cfgFile.WriteString("</content>\n</file>\n\n")

	fmt.Println("Created rag-config.md")

	// 3. Content Bundle
	if err :=
		writeBundle(
			filepath.Join(outDir, "rag-content.md"),
			[]string{
				"content/**/*.md",
			},
			[]string{"content/jules"},
			nil, // Default formatting is verbatim with XML tags, which preserves the YAML frontmatter
			outDir,
			cfg.ProjectRoot,
		); err != nil {
		return fmt.Errorf("failed to write content bundle: %w", err)
	}
	fmt.Println("Created rag-content.md")

	return nil
}

func writeBundle(outPath string, patterns []string, excludes []string, formatFunc func(path string, content []byte) string, outDir string, projectRoot string) error {
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()

	var matchedFiles []string
	for _, pattern := range patterns {
		err := filepath.WalkDir(projectRoot, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if d.Name() == ".git" || d.Name() == "test-results" || d.Name() == "public" || d.Name() == "vendor" || d.Name() == "node_modules" {
					return filepath.SkipDir
				}
				return nil
			}

			relPath := getRel(projectRoot, path)
			if pathMatch(pattern, filepath.ToSlash(relPath)) {
				if strings.Contains(filepath.ToSlash(relPath), filepath.ToSlash(outDir)) {
					return nil
				}
				// Check excludes
				isExcluded := false
				for _, exclude := range excludes {
					if strings.HasPrefix(filepath.ToSlash(relPath), filepath.ToSlash(exclude)) {
						isExcluded = true
						break
					}
				}
				if isExcluded {
					return nil
				}
				found := false
				for _, mf := range matchedFiles {
					if mf == path { // keep path for reading file later
						found = true
						break
					}
				}
				if !found {
					matchedFiles = append(matchedFiles, path)
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	sort.Strings(matchedFiles)

	for _, path := range matchedFiles {
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var output string
		if formatFunc != nil {
			output = formatFunc(path, content)
		} else {
			output = fmt.Sprintf("<file path=\"%s\">\n<content>\n%s\n</content>\n</file>\n\n", filepath.ToSlash(getRel(projectRoot, path)), string(content))
		}
		if _, err := f.WriteString(output); err != nil {
			return err
		}
	}

	return nil
}

func pathMatch(pattern, path string) bool {
	if strings.Contains(pattern, "**/") {
		prefix := strings.Split(pattern, "**/")[0]
		suffix := strings.Split(pattern, "**/")[1]
		if prefix != "" && !strings.HasPrefix(path, prefix) {
			return false
		}
		match, _ := filepath.Match(suffix, filepath.Base(path))
		return match
	}
	match, _ := filepath.Match(pattern, path)
	return match
}

func getRel(base, target string) string {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return target
	}
	return rel
}

</content>
</file>

<file path="internal/ragexport/export_test.go">
<content>
package ragexport

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/config"
)

func TestRunExport_ProjectRoot(t *testing.T) {
	// Create a temp directory to represent our project
	tempDir := t.TempDir()

	// Create some files inside the project
	err := os.MkdirAll(filepath.Join(tempDir, "internal", "foo"), 0755)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(tempDir, "internal", "foo", "foo.go"), []byte("package foo"), 0600)
	if err != nil {
		t.Fatal(err)
	}

	err = os.MkdirAll(filepath.Join(tempDir, "assets"), 0755)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(tempDir, "assets", "logo.png"), []byte("PNG"), 0600)
	if err != nil {
		t.Fatal(err)
	}

	// We'll run the export from a DIFFERENT working directory
	invokeDir := t.TempDir()

	cfg := config.Config{
		ProjectRoot: tempDir,
		RagDir:      filepath.Join(invokeDir, "my-rag"),
	}

	err = RunExport(cfg)
	if err != nil {
		t.Fatalf("RunExport failed: %v", err)
	}

	// Verify the output exists in my-rag
	systemBundlePath := filepath.Join(invokeDir, "my-rag", "rag-system.md")
	content, err := os.ReadFile(systemBundlePath)
	if err != nil {
		t.Fatalf("Failed to read system bundle: %v", err)
	}

	// The path in the bundle should be relative to ProjectRoot
	expectedPath := "<file path=\"internal/foo/foo.go\">"
	if !strings.Contains(string(content), expectedPath) {
		t.Errorf("Expected system bundle to contain %q, but it didn't.\nContent:\n%s", expectedPath, content)
	}

	configBundlePath := filepath.Join(invokeDir, "my-rag", "rag-config.md")
	cfgContent, err := os.ReadFile(configBundlePath)
	if err != nil {
		t.Fatalf("Failed to read config bundle: %v", err)
	}

	expectedAssetPath := "assets/logo.png"
	if !strings.Contains(string(cfgContent), expectedAssetPath) {
		t.Errorf("Expected config bundle to contain %q, but it didn't.\nContent:\n%s", expectedAssetPath, cfgContent)
	}
}

func TestRunExport_RootLevelMatch(t *testing.T) {
	tempDir := t.TempDir()

	// Should be included (root)
	err := os.WriteFile(filepath.Join(tempDir, "README.md"), []byte("Root README"), 0600)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(tempDir, "root.go"), []byte("package main"), 0600)
	if err != nil {
		t.Fatal(err)
	}

	// Should be excluded (nested)
	err = os.MkdirAll(filepath.Join(tempDir, "nested"), 0755)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(tempDir, "nested", "README.md"), []byte("Nested README"), 0600)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(tempDir, "nested", "nested.go"), []byte("package nested"), 0600)
	if err != nil {
		t.Fatal(err)
	}

	invokeDir := t.TempDir()
	cfg := config.Config{
		ProjectRoot: tempDir,
		RagDir:      filepath.Join(invokeDir, "my-rag"),
	}

	err = RunExport(cfg)
	if err != nil {
		t.Fatalf("RunExport failed: %v", err)
	}

	systemBundlePath := filepath.Join(invokeDir, "my-rag", "rag-system.md")
	content, err := os.ReadFile(systemBundlePath)
	if err != nil {
		t.Fatalf("Failed to read system bundle: %v", err)
	}

	contentStr := string(content)

	if !strings.Contains(contentStr, "<file path=\"README.md\">") {
		t.Errorf("Expected system bundle to contain root README.md")
	}
	if strings.Contains(contentStr, "<file path=\"nested/README.md\">") {
		t.Errorf("Expected system bundle NOT to contain nested/README.md")
	}

	if !strings.Contains(contentStr, "<file path=\"root.go\">") {
		t.Errorf("Expected system bundle to contain root.go")
	}
	if strings.Contains(contentStr, "<file path=\"nested/nested.go\">") {
		t.Errorf("Expected system bundle NOT to contain nested/nested.go")
	}
}

</content>
</file>

<file path="internal/render/render.go">
<content>
package render

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/page"
)

type Renderer struct {
	cache     map[string]*template.Template
	allowlist map[string]bool
	mu        sync.Mutex
}

func New(templateDir string) *Renderer {
	allowlist, err := DiscoverLayouts(templateDir)
	if err != nil {
		allowlist = make(map[string]bool)
	}
	return &Renderer{
		cache:     make(map[string]*template.Template),
		allowlist: allowlist,
	}
}

// DiscoverLayouts walks the templates directory to find available layouts.
func DiscoverLayouts(templateDir string) (map[string]bool, error) {
	allowlist := make(map[string]bool)
	entries, err := os.ReadDir(templateDir)
	if err != nil {
		return allowlist, err
	}
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".html" {
			allowlist[strings.TrimSuffix(e.Name(), ".html")] = true
		}
	}
	return allowlist, nil
}

func findPartials() (map[string]string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	var templatesDir string
	for {
		potential := filepath.Join(wd, "templates")
		if stat, err := os.Stat(potential); err == nil && stat.IsDir() {
			templatesDir = potential
			break
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			// Reached root without finding it, just return empty to not break existing flow
			return nil, nil
		}
		wd = parent
	}

	partialsDir := filepath.Join(templatesDir, "partials")
	if _, err := os.Stat(partialsDir); os.IsNotExist(err) {
		return nil, nil
	}

	partials := make(map[string]string)
	err = filepath.WalkDir(partialsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Ext(d.Name()) == ".html" {
			rel, err := filepath.Rel(templatesDir, path)
			if err != nil {
				return err
			}
			partials[filepath.ToSlash(rel)] = path
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return partials, nil
}

// HTML renders a page struct using the specified layout template.
func (r *Renderer) HTML(cfg config.Config, p page.Page, layout, outPath string) error {
	outFile, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	templatePath := cfg.Template
	if layout != "" {
		if !r.allowlist[layout] {
			log.Printf("Warning: Layout %q not found in allowlist. Falling back to default %s", layout, cfg.Template)
		} else {
			layoutPath := filepath.Join("templates", layout+".html")
			// If we are running tests, the templates directory is relative to the root, but the test might run from cmd/la-famille
			if _, err := os.Stat(layoutPath); os.IsNotExist(err) {
				layoutPathFallback := filepath.Join("..", "..", "templates", layout+".html")
				if _, err2 := os.Stat(layoutPathFallback); err2 == nil {
					layoutPath = layoutPathFallback
				}
			}
			if _, err := os.Stat(layoutPath); err == nil {
				templatePath = layoutPath
			} else {
				log.Printf("Warning: layout template %s not found, falling back to %s", layoutPath, cfg.Template)
			}
		}
	}

	r.mu.Lock()
	cachedTmpl, exists := r.cache[templatePath]
	if !exists {
		partials, _ := findPartials()

		b, err := os.ReadFile(templatePath)
		if err != nil {
			r.mu.Unlock()
			return fmt.Errorf("failed to read template %s: %w", templatePath, err)
		}

		parsedTmpl := template.New(filepath.Base(templatePath))
		parsedTmpl, err = parsedTmpl.Parse(string(b))
		if err != nil {
			r.mu.Unlock()
			return fmt.Errorf("failed to parse template %s: %w", templatePath, err)
		}

		for name, path := range partials {
			pb, err := os.ReadFile(path)
			if err != nil {
				r.mu.Unlock()
				return fmt.Errorf("failed to read partial %s: %w", path, err)
			}
			_, err = parsedTmpl.New(name).Parse(string(pb))
			if err != nil {
				r.mu.Unlock()
				return fmt.Errorf("failed to parse partial %s: %w", path, err)
			}
		}
		r.cache[templatePath] = parsedTmpl
		cachedTmpl = parsedTmpl
	}
	r.mu.Unlock()

	clonedTmpl, err := cachedTmpl.Clone()
	if err != nil {
		return fmt.Errorf("failed to clone template %s: %w", templatePath, err)
	}

	// Use ExecuteTemplate with the base name to avoid the ParseFiles name trap
	templateName := filepath.Base(templatePath)
	if cfg.WatchMode {
		var buf bytes.Buffer
		if err := clonedTmpl.ExecuteTemplate(&buf, templateName, p); err != nil {
			return err
		}

		script := `<script>
		if (window.EventSource) {
			var source = new EventSource('/livereload');
			source.onmessage = function(e) {
				if (e.data === 'reload') {
					window.location.reload();
				}
			};
		}
		</script>
</body>`

		htmlStr := buf.String()
		htmlStr = strings.Replace(htmlStr, "</body>", script, 1)

		_, err = outFile.WriteString(htmlStr)
		return err
	}

	if err := clonedTmpl.ExecuteTemplate(outFile, templateName, p); err != nil {
		return err
	}
	return nil
}

</content>
</file>

<file path="internal/render/render_test.go">
<content>
package render

import (
	"html/template"
	"os"
	"path/filepath"
	"testing"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/page"
)

func TestHTML(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "out.html")

	// Setup a fake template
	tmplDir := filepath.Join(tmpDir, "templates")
	_ = os.MkdirAll(tmplDir, 0755)
	tmplPath := filepath.Join(tmplDir, "layout.html")
	err := os.WriteFile(tmplPath, []byte("Hello {{.Title}}"), 0600)
	if err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{Template: tmplPath}
	p := page.Page{Title: "World", Content: template.HTML("")}

	renderer := New(filepath.Dir(cfg.Template))
	err = renderer.HTML(cfg, p, "", outPath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	content, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}

	if string(content) != "Hello World" {
		t.Errorf("expected 'Hello World', got '%s'", string(content))
	}
}

func TestHTMLLayoutSelection(t *testing.T) {
	tmpDir := t.TempDir()

	// Setup fake templates
	tmplDir := filepath.Join(tmpDir, "templates")
	_ = os.MkdirAll(tmplDir, 0755)

	defaultTmplPath := filepath.Join(tmplDir, "layout.html")
	err := os.WriteFile(defaultTmplPath, []byte("Default: {{.Title}}"), 0600)
	if err != nil {
		t.Fatal(err)
	}

	customTmplPath := filepath.Join(tmplDir, "custom.html")
	err = os.WriteFile(customTmplPath, []byte("Custom: {{.Title}}"), 0600)
	if err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{Template: defaultTmplPath}

	// Temporarily change directory to tmpDir so that filepath.Join("templates", layout+".html")
	// resolves to our mocked templates directory.
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origWd) }()
	err = os.Chdir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("no layout specified uses default", func(t *testing.T) {
		outPath := filepath.Join(tmpDir, "out_default.html")
		p := page.Page{Title: "Page One"}

		renderer := New(filepath.Dir(cfg.Template))
		err = renderer.HTML(cfg, p, "", outPath)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		content, err := os.ReadFile(outPath)
		if err != nil {
			t.Fatal(err)
		}

		if string(content) != "Default: Page One" {
			t.Errorf("expected 'Default: Page One', got '%s'", string(content))
		}
	})

	t.Run("layout specified uses custom", func(t *testing.T) {
		outPath := filepath.Join(tmpDir, "out_custom.html")
		p := page.Page{Title: "Page Two"}

		renderer := New(filepath.Dir(cfg.Template))
		err = renderer.HTML(cfg, p, "custom", outPath)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		content, err := os.ReadFile(outPath)
		if err != nil {
			t.Fatal(err)
		}

		if string(content) != "Custom: Page Two" {
			t.Errorf("expected 'Custom: Page Two', got '%s'", string(content))
		}
	})

	t.Run("back-to-back renders using different layouts", func(t *testing.T) {
		renderer := New(filepath.Dir(cfg.Template))

		// First render
		outPath1 := filepath.Join(tmpDir, "out_bb_1.html")
		p1 := page.Page{Title: "First"}
		err = renderer.HTML(cfg, p1, "", outPath1)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Second render
		outPath2 := filepath.Join(tmpDir, "out_bb_2.html")
		p2 := page.Page{Title: "Second"}
		err = renderer.HTML(cfg, p2, "custom", outPath2)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		content1, err := os.ReadFile(outPath1)
		if err != nil {
			t.Fatal(err)
		}
		if string(content1) != "Default: First" {
			t.Errorf("expected 'Default: First', got '%s'", string(content1))
		}

		content2, err := os.ReadFile(outPath2)
		if err != nil {
			t.Fatal(err)
		}
		if string(content2) != "Custom: Second" {
			t.Errorf("expected 'Custom: Second', got '%s'", string(content2))
		}
	})
}

func TestDiscoverLayouts(t *testing.T) {
	tmpDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tmpDir, "layout1.html"), []byte("<html></html>"), 0600); err != nil {
		t.Fatalf("Failed to write layout1.html: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "not-a-layout.txt"), []byte("text"), 0600); err != nil {
		t.Fatalf("Failed to write not-a-layout.txt: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "partials"), 0755); err != nil {
		t.Fatalf("Failed to create partials dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "partials", "partial.html"), []byte("<div></div>"), 0600); err != nil {
		t.Fatalf("Failed to write partial.html: %v", err)
	}

	allowlist, err := DiscoverLayouts(tmpDir)
	if err != nil {
		t.Fatalf("DiscoverLayouts failed: %v", err)
	}

	if len(allowlist) != 1 {
		t.Fatalf("Expected 1 layout, got %d", len(allowlist))
	}
	if !allowlist["layout1"] {
		t.Errorf("Expected layout1 to be in allowlist")
	}
}

func TestHTMLWithPartial(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "out.html")

	// Setup fake templates directory structure
	tmplDir := filepath.Join(tmpDir, "templates")
	_ = os.MkdirAll(tmplDir, 0755)

	// Layout using a partial
	tmplPath := filepath.Join(tmplDir, "layout.html")
	err := os.WriteFile(tmplPath, []byte("Layout: {{template \"partials/footer.html\" .}}"), 0600)
	if err != nil {
		t.Fatal(err)
	}

	// Partial file
	partialsDir := filepath.Join(tmplDir, "partials")
	_ = os.MkdirAll(partialsDir, 0755)
	partialPath := filepath.Join(partialsDir, "footer.html")
	err = os.WriteFile(partialPath, []byte("Footer - {{.Title}}"), 0600)
	if err != nil {
		t.Fatal(err)
	}

	// We need to change the working directory so findPartials can locate "templates"
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origWd) }()
	err = os.Chdir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{Template: tmplPath}
	p := page.Page{Title: "World", Content: template.HTML("")}

	renderer := New(filepath.Dir(cfg.Template))
	err = renderer.HTML(cfg, p, "", outPath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	content, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}

	expected := "Layout: Footer - World"
	if string(content) != expected {
		t.Errorf("expected %q, got %q", expected, string(content))
	}
}

</content>
</file>

<file path="internal/search/search.go">
<content>
package search

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"unicode"
)

type Item struct {
	Title   string   `json:"t"`
	URL     string   `json:"u"`
	Tags    []string `json:"g"`
	Snippet string   `json:"s"`
}

var linkRe = regexp.MustCompile(`\[([^\]]+)\]\([^\)]+\)`)

func ExtractSnippet(rest []byte) string {
	s := string(rest)
	s = linkRe.ReplaceAllString(s, "$1")
	var sb strings.Builder
	for _, r := range s {
		if r == '#' || r == '*' || r == '[' || r == ']' || r == '`' || r == '>' {
			continue
		}
		if unicode.IsSpace(r) {
			sb.WriteRune(' ')
		} else {
			sb.WriteRune(r)
		}
	}
	cleaned := strings.Join(strings.Fields(sb.String()), " ")
	runes := []rune(cleaned)
	if len(runes) > 160 {
		return string(runes[:160]) + "..."
	}
	if len(runes) > 0 {
		return string(runes)
	}
	return ""
}

func WriteMinifiedJSON(path string, data interface{}) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "")
	return enc.Encode(data)
}

</content>
</file>

<file path="internal/search/search_test.go">
<content>
package search

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractSnippet(t *testing.T) {
	md := []byte(`
# Hello World
This is a **bold** and *italic* text.
Here is a [link](https://example.com).
And a code block:
` + "```\nfoo = bar\n```" + `
Inline code: ` + "`fmt.Println()`" + `
> Blockquote text!

Let's see if this works nicely without those characters.
This text needs to be long enough to exceed the one hundred and sixty character limit so that we can verify the truncation logic correctly appends the ellipsis at the very end of the string.
`)
	snippet := ExtractSnippet(md)
	expected := "Hello World This is a bold and italic text. Here is a link. And a code block: foo = bar Inline code: fmt.Println() Blockquote text! Let's see if this works nice..."
	if snippet != expected {
		t.Errorf("expected %q, got %q", expected, snippet)
	}
}

func TestWriteMinifiedJSON(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "search.json")
	items := []Item{
		{Title: "Test", URL: "/test", Tags: []string{"a"}, Snippet: "snip"},
	}
	err := WriteMinifiedJSON(path, items)
	if err != nil {
		t.Fatalf("WriteMinifiedJSON failed: %v", err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	str := string(b)
	if str != `[{"t":"Test","u":"/test","g":["a"],"s":"snip"}]`+"\n" {
		t.Errorf("unexpected json output: %q", str)
	}
}

</content>
</file>

<file path="internal/sitedata/write.go">
<content>
package sitedata

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tbuddy/la-famille/internal/jsonutil"
	"github.com/tbuddy/la-famille/internal/transform"
)

// Write writes the meta data to the output directory.
func Write(outputDir string, metaData map[string]map[string]interface{}) error {
	if err := jsonutil.WriteJSON(filepath.Join(outputDir, "meta.json"), metaData); err != nil {
		return err
	}

	// Generate sitemap.xml
	outDirClean := filepath.Clean(outputDir)
	sitemapPath := filepath.Join(outDirClean, "sitemap.xml")

	// Safeguard against path traversal
	if !strings.HasPrefix(sitemapPath, outDirClean+string(filepath.Separator)) && sitemapPath != outDirClean {
		return fmt.Errorf("potential path traversal writing sitemap: %s", sitemapPath)
	}

	var keys []string
	for k := range metaData {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sitemapBuilder strings.Builder
	sitemapBuilder.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	sitemapBuilder.WriteString("<urlset xmlns=\"http://www.sitemaps.org/schemas/sitemap/0.9\">\n")

	for _, k := range keys {
		// Calculate the output URL
		meta := metaData[k]
		slug := ""
		if slugVal, ok := meta["slug"].(string); ok {
			slug = slugVal
		}

		relOut := transform.GetOutputURL(k+".md", slug)
		urlPath := "/" + filepath.ToSlash(relOut)

		sitemapBuilder.WriteString(fmt.Sprintf("\t<url>\n\t\t<loc>%s</loc>\n\t</url>\n", urlPath))
	}
	sitemapBuilder.WriteString("</urlset>\n")

	if err := os.WriteFile(sitemapPath, []byte(sitemapBuilder.String()), 0600); err != nil {
		return fmt.Errorf("failed to write sitemap.xml: %w", err)
	}

	return nil
}

</content>
</file>

<file path="internal/sitedata/write_test.go">
<content>
package sitedata

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWrite(t *testing.T) {
	tempDir := t.TempDir()

	metaData := map[string]map[string]interface{}{
		"index": {
			"title": "Home Page",
		},
		"about/me": {
			"title": "About Me",
			"slug": "jules",
		},
	}

	err := Write(tempDir, metaData)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// 1. Check meta.json
	metaContent, err := os.ReadFile(filepath.Join(tempDir, "meta.json"))
	if err != nil {
		t.Fatalf("Failed to read meta.json: %v", err)
	}
	var readMeta map[string]map[string]interface{}
	if err := json.Unmarshal(metaContent, &readMeta); err != nil {
		t.Fatalf("Failed to parse meta.json: %v", err)
	}

	if readMeta["index"]["title"] != "Home Page" {
		t.Errorf("Unexpected meta content: %+v", readMeta)
	}

	// 2. Check sitemap.xml
	sitemapContentBytes, err := os.ReadFile(filepath.Join(tempDir, "sitemap.xml"))
	if err != nil {
		t.Fatalf("Failed to read sitemap.xml: %v", err)
	}
	sitemapContent := string(sitemapContentBytes)

	if !strings.Contains(sitemapContent, "<urlset xmlns=\"http://www.sitemaps.org/schemas/sitemap/0.9\">") {
		t.Errorf("sitemap.xml missing root urlset tag")
	}

	if !strings.Contains(sitemapContent, "<loc>/index.html</loc>") {
		t.Errorf("sitemap.xml missing loc for index.html")
	}

	if !strings.Contains(sitemapContent, "<loc>/about/jules/index.html</loc>") {
		t.Errorf("sitemap.xml missing loc for about/jules/index.html")
	}
}

</content>
</file>

<file path="internal/stub/stub.go">
<content>
package stub

import (
	"fmt"
	"html"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/microcosm-cc/bluemonday"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/content"
	"github.com/tbuddy/la-famille/internal/graph"
	"github.com/tbuddy/la-famille/internal/page"
	"github.com/tbuddy/la-famille/internal/transform"
)

func findPartials() (map[string]string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	var templatesDir string
	for {
		potential := filepath.Join(wd, "templates")
		if stat, err := os.Stat(potential); err == nil && stat.IsDir() {
			templatesDir = potential
			break
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			// Reached root without finding it, just return empty to not break existing flow
			return nil, nil
		}
		wd = parent
	}

	partialsDir := filepath.Join(templatesDir, "partials")
	if _, err := os.Stat(partialsDir); os.IsNotExist(err) {
		return nil, nil
	}

	partials := make(map[string]string)
	err = filepath.WalkDir(partialsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Ext(d.Name()) == ".html" {
			rel, err := filepath.Rel(templatesDir, path)
			if err != nil {
				return err
			}
			partials[filepath.ToSlash(rel)] = path
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return partials, nil
}

func GenerateStubs(cfg config.Config, missingFiles map[string][]string, g *graph.Graph, p *bluemonday.Policy, fileMap map[string]*content.FileMeta) error {
	var missingKeys []string
	for k := range missingFiles {
		missingKeys = append(missingKeys, k)
	}
	sort.Strings(missingKeys)

	for _, missingRelPath := range missingKeys {
		outDirClean := filepath.Clean(cfg.OutputDir)
		outPath := filepath.Join(outDirClean, filepath.FromSlash(missingRelPath))
		if !strings.HasPrefix(outPath, outDirClean+string(filepath.Separator)) && outPath != outDirClean {
			continue
		}

		parents := missingFiles[missingRelPath]
		sort.Strings(parents)
		id := strings.TrimSuffix(missingRelPath, ".md")
		g.Nodes[id] = graph.Node{
			Type:         "stub",
			Render:       true,
			Missing:      true,
			ReferencedBy: parents,
		}

		// derive outPath using clean URL logic
		relOut := transform.GetOutputURL(missingRelPath, "")
		outPath = filepath.Join(outDirClean, filepath.FromSlash(relOut))

		if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
			return err
		}

		// build simple HTML stub
		var htmlContent strings.Builder
		htmlContent.WriteString("<div class=\"alert alert-warning shadow-lg mb-8\">\n")
		htmlContent.WriteString("  <div>\n")
		htmlContent.WriteString("    <svg xmlns=\"http://www.w3.org/2000/svg\" class=\"stroke-current flex-shrink-0 h-6 w-6\" fill=\"none\" viewBox=\"0 0 24 24\"><path stroke-linecap=\"round\" stroke-linejoin=\"round\" stroke-width=\"2\" d=\"M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z\" /></svg>\n")
		htmlContent.WriteString("    <div>\n")
		htmlContent.WriteString("      <h3 class=\"font-bold\">🚧 Under Construction</h3>\n")
		htmlContent.WriteString("      <div class=\"text-xs\">We are still working on this content. Please check back later!</div>\n")
		htmlContent.WriteString("    </div>\n")
		htmlContent.WriteString("  </div>\n")
		htmlContent.WriteString("</div>\n")
		htmlContent.WriteString("<h3>Where did you come from?</h3>\n")
		htmlContent.WriteString("<p>You can return to the previous context by visiting one of these pages that link here:</p>\n")
		htmlContent.WriteString("<ul class=\"menu bg-base-100 border border-base-300 rounded-box w-full\">\n")
		for _, parent := range parents {
			parentSlug := ""
			if meta, ok := fileMap[parent]; ok && meta != nil {
				parentSlug = meta.Slug
				if parentSlug != "" {
					if !filepath.IsLocal(parentSlug) || strings.Contains(parentSlug, ".") || strings.Contains(parentSlug, string(filepath.Separator)) || strings.Contains(parentSlug, "/") {
						parentSlug = ""
					}
				}
			}

			currOut := transform.GetOutputURL(missingRelPath, "")
			parentOut := transform.GetOutputURL(parent, parentSlug)

			currDir := filepath.Dir(currOut)
			if currDir == "." {
				currDir = ""
			}

			relParent, err := filepath.Rel(currDir, parentOut)
			if err == nil {
				relParentSlash := filepath.ToSlash(relParent)
				if strings.HasSuffix(relParentSlash, "index.html") {
					if relParentSlash == "index.html" {
						relParentSlash = "./"
					} else {
						relParentSlash = strings.TrimSuffix(relParentSlash, "index.html")
					}
				}
				htmlContent.WriteString(fmt.Sprintf("<li><a href=\"%s\">%s</a></li>\n", html.EscapeString(relParentSlash), html.EscapeString(parent)))
			} else {
				htmlContent.WriteString(fmt.Sprintf("<li>%s</li>\n", html.EscapeString(parent)))
			}
		}
		htmlContent.WriteString("</ul>\n")

		pageStruct := page.Page{
			Site:    cfg,
			Title:   "Missing Page",
			Content: template.HTML(p.SanitizeBytes([]byte(htmlContent.String()))), // #nosec G203
		}

		outFile, err := os.Create(outPath)
		if err != nil {
			return err
		}

		partials, _ := findPartials()
		b, err := os.ReadFile(cfg.Template)
		if err != nil {
			outFile.Close()
			return fmt.Errorf("failed to read default template file for stubs: %w", err)
		}

		defaultTmpl := template.New(filepath.Base(cfg.Template))
		defaultTmpl, err = defaultTmpl.Parse(string(b))
		if err != nil {
			outFile.Close()
			return fmt.Errorf("failed to parse default template file for stubs: %w", err)
		}

		for name, path := range partials {
			pb, err := os.ReadFile(path)
			if err != nil {
				outFile.Close()
				return fmt.Errorf("failed to read partial %s for stubs: %w", path, err)
			}
			_, err = defaultTmpl.New(name).Parse(string(pb))
			if err != nil {
				outFile.Close()
				return fmt.Errorf("failed to parse partial %s for stubs: %w", path, err)
			}
		}

		if err := defaultTmpl.ExecuteTemplate(outFile, filepath.Base(cfg.Template), pageStruct); err != nil {
			outFile.Close()
			return err
		}
		outFile.Close()
	}
	return nil
}

// RelPathFromTo computes the relative URL path from base (e.g. dir1/missing.md) to target (e.g. index.html)
func RelPathFromTo(base, target string) (string, error) {
	baseDir := filepath.Dir(base)
	rel, err := filepath.Rel(baseDir, target)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(rel), nil
}

</content>
</file>

<file path="internal/stub/stub_test.go">
<content>
package stub

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/microcosm-cc/bluemonday"
	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/content"
	"github.com/tbuddy/la-famille/internal/graph"
)

func TestRelPathFromTo(t *testing.T) {
	tests := []struct {
		name     string
		base     string
		target   string
		expected string
	}{
		{
			name:     "same directory",
			base:     "dir/missing.md",
			target:   "dir/parent.html",
			expected: "parent.html",
		},
		{
			name:     "target in parent directory",
			base:     "dir/subdir/missing.md",
			target:   "dir/parent.html",
			expected: "../parent.html",
		},
		{
			name:     "target in child directory",
			base:     "dir/missing.md",
			target:   "dir/subdir/parent.html",
			expected: "subdir/parent.html",
		},
		{
			name:     "different branch",
			base:     "dir1/missing.md",
			target:   "dir2/parent.html",
			expected: "../dir2/parent.html",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rel, err := RelPathFromTo(tt.base, tt.target)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if rel != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, rel)
			}
		})
	}
}

func TestGenerateStubs(t *testing.T) {
	// Setup a temporary directory for output
	tempDir, err := os.MkdirTemp("", "stub-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a dummy template file since GenerateStubs requires it
	templatePath := filepath.Join(tempDir, "layout.html")
	templateContent := `<html><body>{{.Content}}</body></html>`
	if err := os.WriteFile(templatePath, []byte(templateContent), 0600); err != nil {
		t.Fatalf("failed to write dummy template: %v", err)
	}

	cfg := config.Config{
		OutputDir: tempDir,
		Template:  templatePath,
	}

	missingFiles := map[string][]string{
		"missing.md":      {"parent1.md"},
		"dir/missing2.md": {"parent2.md", "dir/parent3.md"},
	}

	g := &graph.Graph{
		Nodes: make(map[string]graph.Node),
	}

	p := bluemonday.UGCPolicy()
	p.AllowAttrs("class").Globally()

	// Execute GenerateStubs

	fileMap := make(map[string]*content.FileMeta)
	if err := GenerateStubs(cfg, missingFiles, g, p, fileMap); err != nil {
		t.Fatalf("unexpected error from GenerateStubs: %v", err)
	}

	// Verify graph node updates
	if node, ok := g.Nodes["missing"]; !ok || node.Type != "stub" {
		t.Errorf("expected missing node in graph with type 'stub', got %v", node)
	}
	if node, ok := g.Nodes["dir/missing2"]; !ok || node.Type != "stub" {
		t.Errorf("expected dir/missing2 node in graph with type 'stub', got %v", node)
	}

	// Verify output files are created with expected content
	checkFile := func(relPath string, expectedSubstrings []string) {
		fullPath := filepath.Join(tempDir, relPath)
		contentBytes, err := os.ReadFile(fullPath)
		if err != nil {
			t.Fatalf("failed to read expected stub file %q: %v", fullPath, err)
		}
		contentStr := string(contentBytes)
		for _, substr := range expectedSubstrings {
			if !strings.Contains(contentStr, substr) {
				t.Errorf("file %q did not contain expected substring %q. Content:\n%s", relPath, substr, contentStr)
			}
		}
	}

	checkFile("missing/index.html", []string{
		"🚧 Under Construction",
		"alert alert-warning",
		"menu bg-base-100",
		`<a href="../parent1/" rel="nofollow">parent1.md</a>`,
	})

	checkFile("dir/missing2/index.html", []string{
		"🚧 Under Construction",
		"alert alert-warning",
		"menu bg-base-100",
		`<a href="../../parent2/" rel="nofollow">parent2.md</a>`,
		`<a href="../parent3/" rel="nofollow">dir/parent3.md</a>`,
	})
}

</content>
</file>

<file path="internal/taxonomy/taxonomy.go">
<content>
package taxonomy

import (
	"fmt"
	"html"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/microcosm-cc/bluemonday"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/content"
	"github.com/tbuddy/la-famille/internal/page"
	"github.com/tbuddy/la-famille/internal/render"
	"github.com/tbuddy/la-famille/internal/transform"
)

func GenerateTags(cfg config.Config, fileMap map[string]*content.FileMeta, renderer *render.Renderer, p *bluemonday.Policy) error {
	tagMap := make(map[string][]string)

	for relPath, meta := range fileMap {
		if meta.Render != nil && !*meta.Render {
			continue
		}
		for _, tag := range meta.Tags {
			tagMap[tag] = append(tagMap[tag], relPath)
		}
	}

	var tags []string
	for tag := range tagMap {
		tags = append(tags, tag)
	}
	sort.Strings(tags)

	outDirClean := filepath.Clean(cfg.OutputDir)

	for _, tag := range tags {
		pages := tagMap[tag]
		sort.Strings(pages)

		tagRelPath := fmt.Sprintf("tags/%s/index.md", tag)
		tagOut := transform.GetOutputURL(tagRelPath, "")
		outPath := filepath.Join(outDirClean, filepath.FromSlash(tagOut))

		if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
			return err
		}

		var htmlContent strings.Builder
		htmlContent.WriteString(fmt.Sprintf("<h2>Tag: %s</h2>\n", html.EscapeString(tag)))
		htmlContent.WriteString("<ul>\n")

		for _, relPath := range pages {
			meta := fileMap[relPath]

			title := meta.Title
			if title == "" {
				title = filepath.Base(relPath)
			}

			pageOut := transform.GetOutputURL(relPath, meta.Slug)

			currDir := filepath.Dir(tagOut)
			if currDir == "." {
				currDir = ""
			}

			relOut, err := filepath.Rel(currDir, pageOut)
			if err == nil {
				relOutSlash := filepath.ToSlash(relOut)
				if strings.HasSuffix(relOutSlash, "index.html") {
					if relOutSlash == "index.html" {
						relOutSlash = "./"
					} else {
						relOutSlash = strings.TrimSuffix(relOutSlash, "index.html")
					}
				}
				htmlContent.WriteString(fmt.Sprintf("<li><a href=\"%s\">%s</a></li>\n", html.EscapeString(relOutSlash), html.EscapeString(title)))
			}
		}
		htmlContent.WriteString("</ul>\n")

		sanitizedHTML := p.SanitizeBytes([]byte(htmlContent.String()))

		pageStruct := page.Page{
			Site:    cfg,
			Title:   fmt.Sprintf("Tag: %s", tag),
			Content: template.HTML(sanitizedHTML), // #nosec G203
		}

		if err := renderer.HTML(cfg, pageStruct, "", outPath); err != nil {
			return err
		}
	}
	return nil
}

</content>
</file>

<file path="internal/taxonomy/taxonomy_test.go">
<content>
package taxonomy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/microcosm-cc/bluemonday"
	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/content"
	"github.com/tbuddy/la-famille/internal/render"
)

func TestGenerateTags(t *testing.T) {
	tempDir := t.TempDir()
	outDir := filepath.Join(tempDir, "public")
	tmplDir := filepath.Join(tempDir, "templates")

	_ = os.MkdirAll(outDir, 0755)
	_ = os.MkdirAll(tmplDir, 0755)

	tmplPath := filepath.Join(tmplDir, "layout.html")
	_ = os.WriteFile(tmplPath, []byte("{{.Content}}"), 0600)

	cfg := config.Config{
		OutputDir: outDir,
		Template:  tmplPath,
	}

	renderTrue := true
	fileMap := map[string]*content.FileMeta{
		"post1.md": {Title: "Post 1", Tags: []string{"go", "web"}, Render: &renderTrue},
		"post2.md": {Title: "Post 2", Tags: []string{"go"}, Render: &renderTrue},
	}

	renderer := render.New(tmplDir)
	p := bluemonday.UGCPolicy()

	err := GenerateTags(cfg, fileMap, renderer, p)
	if err != nil {
		t.Fatalf("GenerateTags failed: %v", err)
	}

	// Check if go tag page was created
	goTagPath := filepath.Join(outDir, "tags", "go", "index.html")
	b, err := os.ReadFile(goTagPath)
	if err != nil {
		t.Fatalf("expected tags/go/index.html to exist: %v", err)
	}
	html := string(b)
	if !strings.Contains(html, "<h2>Tag: go</h2>") {
		t.Errorf("expected tag title, got: %s", html)
	}
	if !strings.Contains(html, `href="../../post1/"`) {
		t.Errorf("expected link to post1, got: %s", html)
	}
	if !strings.Contains(html, `href="../../post2/"`) {
		t.Errorf("expected link to post2, got: %s", html)
	}
}

</content>
</file>

<file path="internal/transform/emoji_kitchen.go">
<content>
package transform

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

type EmojiKitchenParser struct{}

func (p *EmojiKitchenParser) Trigger() []byte {
	return []byte{'!'}
}

func (p *EmojiKitchenParser) Parse(_ ast.Node, block text.Reader, _ parser.Context) ast.Node {
	line, _ := block.PeekLine()
	re := regexp.MustCompile(`^!ek\[([^\+\]]+)\+([^\]]+)\]`)
	match := re.FindSubmatchIndex(line)
	if match == nil {
		return nil
	}

	leftStr := strings.TrimSpace(string(line[match[2]:match[3]]))
	rightStr := strings.TrimSpace(string(line[match[4]:match[5]]))

	leftRunes := []rune(leftStr)
	rightRunes := []rune(rightStr)

	if len(leftRunes) == 0 || len(rightRunes) == 0 {
		return nil
	}

	left := leftRunes[0]
	right := rightRunes[0]

	leftHex := fmt.Sprintf("u%x", left)
	rightHex := fmt.Sprintf("u%x", right)

	url := fmt.Sprintf("https://www.gstatic.com/android/keyboard/emojikitchen/20201001/%s/%s_%s.png", leftHex, leftHex, rightHex)

	img := ast.NewImage(ast.NewLink())
	img.Destination = []byte(url)
	img.Title = []byte(fmt.Sprintf("Emoji Kitchen combination of %s and %s", string(left), string(right)))

	altText := fmt.Sprintf("Emoji Kitchen combination of %s and %s", string(left), string(right))
	img.AppendChild(img, ast.NewString([]byte(altText)))

	block.Advance(match[1])

	return img
}

</content>
</file>

<file path="internal/transform/emoji_kitchen_test.go">
<content>
package transform

import (
	"bytes"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/util"
)

func TestEmojiKitchenParser(t *testing.T) {
	md := goldmark.New(
		goldmark.WithParserOptions(
			parser.WithInlineParsers(
				util.Prioritized(&EmojiKitchenParser{}, 100),
			),
		),
	)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Turtle and Fire",
			input:    "Look at this !ek[🐢+🔥]",
			expected: "<p>Look at this <img src=\"https://www.gstatic.com/android/keyboard/emojikitchen/20201001/u1f422/u1f422_u1f525.png\" alt=\"Emoji Kitchen combination of 🐢 and 🔥\" title=\"Emoji Kitchen combination of 🐢 and 🔥\"></p>\n",
		},
		{
			name:     "Turtle and Turtle",
			input:    "!ek[🐢+🐢]",
			expected: "<p><img src=\"https://www.gstatic.com/android/keyboard/emojikitchen/20201001/u1f422/u1f422_u1f422.png\" alt=\"Emoji Kitchen combination of 🐢 and 🐢\" title=\"Emoji Kitchen combination of 🐢 and 🐢\"></p>\n",
		},
		{
			name:     "Invalid syntax",
			input:    "!ek[🐢+]",
			expected: "<p>!ek[🐢+]</p>\n",
		},
		{
			name:     "Spaces around emojis",
			input:    "!ek[ 🐢 + 🔥 ]",
			expected: "<p><img src=\"https://www.gstatic.com/android/keyboard/emojikitchen/20201001/u1f422/u1f422_u1f525.png\" alt=\"Emoji Kitchen combination of 🐢 and 🔥\" title=\"Emoji Kitchen combination of 🐢 and 🔥\"></p>\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := md.Convert([]byte(tc.input), &buf); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if buf.String() != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, buf.String())
			}
		})
	}
}

</content>
</file>

<file path="internal/transform/link_transformer.go">
<content>
package transform

import (
	"net/url"
	"path/filepath"
	"strings"
	"sync"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"

	"github.com/tbuddy/la-famille/internal/content"
	"github.com/tbuddy/la-famille/internal/graph"
)

type LinkTransformer struct {
	CurrentFile  string // The current file being processed (e.g., docs/index.md)
	FileMap      map[string]*content.FileMeta
	MissingFiles map[string][]string // map[targetFile]parents
	Backlinks    map[string][]string
	Graph        *graph.Graph
	Mu           *sync.Mutex
}

func (t *LinkTransformer) Transform(node *ast.Document, _ text.Reader, _ parser.Context) {
	sourceID := strings.TrimSuffix(t.CurrentFile, ".md")

	_ = ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		if link, ok := n.(*ast.Link); ok {
			dest := string(link.Destination)
			u, err := url.Parse(dest)
			// Ignore if parse fails, or it's an absolute url (like http://...), or not a .md file
			if err != nil || u.IsAbs() || !strings.HasSuffix(u.Path, ".md") {
				return ast.WalkContinue, nil
			}

			// Path is relative, like "../file.md" or "file.md"
			// Need to resolve it relative to the directory of CurrentFile
			dir := filepath.Dir(t.CurrentFile)
			// filepath.Join uses OS separators, but we want to stick to slashes
			targetRelPath := filepath.ToSlash(filepath.Clean(dir + "/" + u.Path))
			if dir == "." {
				targetRelPath = filepath.ToSlash(filepath.Clean(u.Path))
			}

			// Prevent path traversal
			if !filepath.IsLocal(filepath.FromSlash(targetRelPath)) {
				return ast.WalkContinue, nil
			}

			targetID := strings.TrimSuffix(targetRelPath, ".md")
			if t.Mu != nil {
				t.Mu.Lock()
			}
			t.Graph.Edges = append(t.Graph.Edges, [2]string{sourceID, targetID})
			t.Backlinks[targetID] = append(t.Backlinks[targetID], sourceID)
			if t.Mu != nil {
				t.Mu.Unlock()
			}

			// Check file map
			meta, exists := t.FileMap[targetRelPath]

			// If target exists and render is explicitly false, keep as .md
			if exists && meta.Render != nil && !*meta.Render {
				// keep it as .md, no change needed
				_ = meta
			} else {
				slug := ""
				if exists && meta != nil {
					slug = meta.Slug
					if slug != "" {
						if !filepath.IsLocal(slug) || strings.Contains(slug, ".") || strings.Contains(slug, string(filepath.Separator)) || strings.Contains(slug, "/") {
							slug = ""
						}
					}
				}

				currOut := GetOutputURL(t.CurrentFile, "")
				targetOut := GetOutputURL(targetRelPath, slug)

				currDir := filepath.Dir(currOut)
				if currDir == "." {
					currDir = ""
				}

				relOut, err := filepath.Rel(currDir, targetOut)
				if err == nil {
					relOutSlash := filepath.ToSlash(relOut)
					if strings.HasSuffix(relOutSlash, "index.html") {
						if relOutSlash == "index.html" {
							relOutSlash = "./"
						} else {
							relOutSlash = strings.TrimSuffix(relOutSlash, "index.html")
						}
					}
					u.Path = relOutSlash
					link.Destination = []byte(u.String())
				}
			}

			if !exists {
				// record target as missing so we can generate stub
				if t.Mu != nil {
					t.Mu.Lock()
				}
				parents := t.MissingFiles[targetRelPath]
				found := false
				for _, p := range parents {
					if p == t.CurrentFile {
						found = true
						break
					}
				}
				if !found {
					t.MissingFiles[targetRelPath] = append(parents, t.CurrentFile)
				}
				if t.Mu != nil {
					t.Mu.Unlock()
				}
			}
		}

		return ast.WalkContinue, nil
	})
}

</content>
</file>

<file path="internal/transform/link_transformer_test.go">
<content>
package transform

import (
	"bytes"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/util"

	"github.com/tbuddy/la-famille/internal/content"
	"github.com/tbuddy/la-famille/internal/graph"
)

func TestLinkTransformer(t *testing.T) {
	renderTrue := true
	renderFalse := false

	tests := []struct {
		name         string
		currentFile  string
		markdown     string
		fileMap      map[string]*content.FileMeta
		expectedHTML string
		expectedMiss map[string][]string
	}{
		{
			name:        "internal link rewritten",
			currentFile: "index.md",
			markdown:    "[Link](page.md)",
			fileMap: map[string]*content.FileMeta{
				"page.md": {Render: &renderTrue},
			},
			expectedHTML: "<p><a href=\"page/\">Link</a></p>\n",
			expectedMiss: map[string][]string{},
		},
		{
			name:         "external link ignored",
			currentFile:  "index.md",
			markdown:     "[Link](http://example.com/page.md)",
			fileMap:      map[string]*content.FileMeta{},
			expectedHTML: "<p><a href=\"http://example.com/page.md\">Link</a></p>\n",
			expectedMiss: map[string][]string{},
		},
		{
			name:         "non-markdown link ignored",
			currentFile:  "index.md",
			markdown:     "[Link](page.txt)",
			fileMap:      map[string]*content.FileMeta{},
			expectedHTML: "<p><a href=\"page.txt\">Link</a></p>\n",
			expectedMiss: map[string][]string{},
		},
		{
			name:        "link to unrendered md file kept as .md",
			currentFile: "index.md",
			markdown:    "[Link](raw.md)",
			fileMap: map[string]*content.FileMeta{
				"raw.md": {Render: &renderFalse},
			},
			expectedHTML: "<p><a href=\"raw.md\">Link</a></p>\n",
			expectedMiss: map[string][]string{},
		},
		{
			name:         "missing link rewritten and tracked",
			currentFile:  "index.md",
			markdown:     "[Link](missing.md)",
			fileMap:      map[string]*content.FileMeta{},
			expectedHTML: "<p><a href=\"missing/\">Link</a></p>\n",
			expectedMiss: map[string][]string{
				"missing.md": {"index.md"},
			},
		},
		{
			name:        "relative subdirectory link",
			currentFile: "sub/index.md",
			markdown:    "[Link](../page.md)",
			fileMap: map[string]*content.FileMeta{
				"page.md": {Render: &renderTrue},
			},
			expectedHTML: "<p><a href=\"../page/\">Link</a></p>\n",
			expectedMiss: map[string][]string{},
		},
		{
			name:         "path traversal link ignored",
			currentFile:  "index.md",
			markdown:     "[Link](../../../etc/passwd.md)",
			fileMap:      map[string]*content.FileMeta{},
			expectedHTML: "<p><a href=\"../../../etc/passwd.md\">Link</a></p>\n",
			expectedMiss: map[string][]string{},
		},
		{
			name:         "multiple identical missing links deduplicate parent",
			currentFile:  "index.md",
			markdown:     "[Link](missing.md) and [Link2](missing.md)",
			fileMap:      map[string]*content.FileMeta{},
			expectedHTML: "<p><a href=\"missing/\">Link</a> and <a href=\"missing/\">Link2</a></p>\n",
			expectedMiss: map[string][]string{
				"missing.md": {"index.md"},
			},
		},
		{
			name:         "empty target path ignored",
			currentFile:  "index.md",
			markdown:     "[Link](#test)",
			fileMap:      map[string]*content.FileMeta{},
			expectedHTML: "<p><a href=\"#test\">Link</a></p>\n",
			expectedMiss: map[string][]string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			missingFiles := make(map[string][]string)
			backlinks := make(map[string][]string)
			g := &graph.Graph{
				Nodes: make(map[string]graph.Node),
				Edges: [][2]string{},
			}

			transformer := &LinkTransformer{
				CurrentFile:  tc.currentFile,
				FileMap:      tc.fileMap,
				MissingFiles: missingFiles,
				Backlinks:    backlinks,
				Graph:        g,
			}

			md := goldmark.New(
				goldmark.WithParserOptions(
					parser.WithASTTransformers(
						util.Prioritized(transformer, 100),
					),
				),
			)

			var buf bytes.Buffer
			if err := md.Convert([]byte(tc.markdown), &buf); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if buf.String() != tc.expectedHTML {
				t.Errorf("expected HTML %q, got %q", tc.expectedHTML, buf.String())
			}

			if len(missingFiles) != len(tc.expectedMiss) {
				t.Errorf("expected %d missing files, got %d", len(tc.expectedMiss), len(missingFiles))
			}
			for k, v := range tc.expectedMiss {
				if len(missingFiles[k]) != len(v) {
					t.Errorf("missing file %s: expected %d parents, got %d", k, len(v), len(missingFiles[k]))
				}
			}
		})
	}
}

</content>
</file>

<file path="internal/transform/url.go">
<content>
package transform

import (
	"path"
	"strings"
)

// GetOutputURL calculates the output URL (with index.html) for a given .md relative path and optional slug override.
func GetOutputURL(relPath string, slug string) string {
	dir := path.Dir(relPath)
	if dir == "." {
		dir = ""
	}

	base := path.Base(relPath)
	name := strings.TrimSuffix(base, ".md")

	if slug != "" {
		name = slug
	}

	if name == "index" {
		if dir == "" {
			return "index.html"
		}
		return path.Join(dir, "index.html")
	}

	return path.Join(dir, name, "index.html")
}

</content>
</file>

<file path="internal/transform/url_test.go">
<content>
package transform

import "testing"

func TestGetOutputURL(t *testing.T) {
	tests := []struct {
		name     string
		relPath  string
		slug     string
		expected string
	}{
		{
			name:     "standard md file",
			relPath:  "about.md",
			slug:     "",
			expected: "about/index.html",
		},
		{
			name:     "index md file",
			relPath:  "index.md",
			slug:     "",
			expected: "index.html",
		},
		{
			name:     "standard md file with slug",
			relPath:  "about.md",
			slug:     "bio",
			expected: "bio/index.html",
		},
		{
			name:     "index md file with slug",
			relPath:  "index.md",
			slug:     "home",
			expected: "home/index.html",
		},
		{
			name:     "nested standard md file",
			relPath:  "blog/post.md",
			slug:     "",
			expected: "blog/post/index.html",
		},
		{
			name:     "nested index md file",
			relPath:  "blog/index.md",
			slug:     "",
			expected: "blog/index.html",
		},
		{
			name:     "nested md file with slug",
			relPath:  "blog/post.md",
			slug:     "my-post",
			expected: "blog/my-post/index.html",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := GetOutputURL(tc.relPath, tc.slug)
			if actual != tc.expected {
				t.Errorf("GetOutputURL(%q, %q) = %q; expected %q", tc.relPath, tc.slug, actual, tc.expected)
			}
		})
	}
}

</content>
</file>

<file path="internal/watcher/livereload.go">
<content>
package watcher

import (
	"fmt"
	"net/http"
	"sync"
)

var (
	clients   = make(map[chan struct{}]bool)
	clientsMu sync.Mutex
)

// LiveReloadHandler handles SSE connections from the browser.
func LiveReloadHandler(w http.ResponseWriter, r *http.Request) {
	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	// Allow CORS just in case
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create a channel for this client
	clientChan := make(chan struct{})

	clientsMu.Lock()
	clients[clientChan] = true
	clientsMu.Unlock()

	defer func() {
		clientsMu.Lock()
		delete(clients, clientChan)
		clientsMu.Unlock()
	}()

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	// Wait for a message or client disconnect
	for {
		select {
		case <-clientChan:
			fmt.Fprintf(w, "data: reload\n\n")
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

// BroadcastReload sends a reload signal to all connected SSE clients.
func BroadcastReload() {
	clientsMu.Lock()
	defer clientsMu.Unlock()
	for clientChan := range clients {
		// Non-blocking send
		select {
		case clientChan <- struct{}{}:
		default:
		}
	}
}

</content>
</file>

<file path="internal/watcher/watcher.go">
<content>
package watcher

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/generator"
)

// Watch starts an fsnotify watcher on the given config's ContentDir, Templates, and Assets dir.
// It explicitly unbinds and tears down resources once the passed context registers Done.
func Watch(ctx context.Context, cfg config.Config, onBuild func(generator.BuildResult)) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	// Debounce timer management
	var buildTimer *time.Timer
	defer func() {
		if buildTimer != nil {
			buildTimer.Stop()
		}
	}()

	// Orchestrate directories to monitor
	dirsToWatch := []string{cfg.ContentDir}

	templateDir := filepath.Dir(cfg.Template)
	if _, err := os.Stat(templateDir); err == nil {
		dirsToWatch = append(dirsToWatch, templateDir)
	}
	if _, err := os.Stat("assets"); err == nil {
		dirsToWatch = append(dirsToWatch, "assets")
	}

	for _, dir := range dirsToWatch {
		err = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return watcher.Add(path)
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	log.Println("Context-aware file watcher initialized successfully.")

	for {
		select {
		case <-ctx.Done():
			log.Println("Halting file watcher: Context canceled.")
			return ctx.Err()

		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}

			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) || event.Has(fsnotify.Remove) {
				if event.Has(fsnotify.Create) {
					stat, err := os.Stat(event.Name)
					if err == nil && stat.IsDir() {
						log.Printf("Dynamic directory tracking added: %s", event.Name)
						_ = watcher.Add(event.Name)
					}
				}

				log.Printf("Change caught in %s, scheduling build pass...", event.Name)
				if buildTimer != nil {
					buildTimer.Stop()
				}

				buildTimer = time.AfterFunc(500*time.Millisecond, func() {
					log.Println("Executing pipeline rebuild...")
					start := time.Now()
					if res, err := generator.Build(cfg); err != nil {
						if onBuild != nil {
							onBuild(res)
						}
						BroadcastReload()
						log.Printf("Pipeline compilation failed: %v", err)
					} else {
						log.Printf("Rebuild complete in %v.", time.Since(start))
						if onBuild != nil {
							onBuild(res)
						}
						BroadcastReload()
					}
				})
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			log.Printf("Watcher filesystem interruption error: %v", err)
		}
	}
}

</content>
</file>
