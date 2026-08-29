package main

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/generator"
	"github.com/tbuddy/la-famille/internal/runtimeassets"
	"github.com/tbuddy/la-famille/internal/stub"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
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
	defaultContentDir := filepath.Join(tmpDir, "default_content_from_config")
	if err := os.Mkdir(defaultContentDir, 0755); err != nil {
		t.Fatalf("Failed to create default content dir: %v", err)
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
	if err := os.WriteFile(filepath.Join(defaultContentDir, "index.md"), mdContent, 0600); err != nil {
		t.Fatalf("Failed to write index.md in default content dir: %v", err)
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

	// Build la-famille executable once per test run and share it across the
	// exec-based tests (#552).
	exePath := sharedGateBinary()

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

	stderrPipe, err := cmdServe.StderrPipe()
	if err != nil {
		t.Fatalf("failed to create stderr pipe: %v", err)
	}

	if err := cmdServe.Start(); err != nil {
		t.Fatalf("failed to start serve command: %v", err)
	}

	outputChan := make(chan string)
	go func() {
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "Serving") {
				outputChan <- line
				return
			}
		}
		if err := scanner.Err(); err != nil {
			outputChan <- err.Error()
		}
		close(outputChan)
	}()

	select {
	case serveOut, ok := <-outputChan:
		if !ok {
			t.Errorf("Serve command exited before outputting port")
		} else if !strings.Contains(serveOut, "msg=\"Serving") {
			t.Errorf("Expected serve command to log serving message, got output: %s", serveOut)
		}
	case <-time.After(5 * time.Second):
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

	exePath := sharedGateBinary()

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

// The init repair contract at the file level, tested without compiling a
// binary (#552). TestInitCommand covers the end-to-end invocation; these pin
// writeInitialConfig's create / refuse / force-backup / theme behavior.
func TestWriteInitialConfig(t *testing.T) {
	t.Run("creates a loadable default config when absent", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.yaml")
		if err := writeInitialConfig(path, false, ""); err != nil {
			t.Fatalf("writeInitialConfig: %v", err)
		}
		cfg, err := config.Load(path)
		if err != nil {
			t.Fatalf("created config is unloadable: %v", err)
		}
		if cfg.SiteName != "La Famille" {
			t.Errorf("SiteName = %q, want the default %q", cfg.SiteName, "La Famille")
		}
	})

	t.Run("refuses to overwrite an existing config and names the repair path", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.yaml")
		original := []byte("site_name: \"Customised\"\n")
		if err := os.WriteFile(path, original, 0600); err != nil {
			t.Fatal(err)
		}

		err := writeInitialConfig(path, false, "")
		if err == nil {
			t.Fatal("expected a refusal, got nil")
		}
		if !strings.Contains(err.Error(), initConfigBackup) || !strings.Contains(err.Error(), "--force") {
			t.Errorf("refusal must name the %s backup and the --force repair path, got: %v", initConfigBackup, err)
		}
		got, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if string(got) != string(original) {
			t.Errorf("refused init modified the config: %q", got)
		}
	})

	t.Run("--force replaces the config and keeps the original as a backup", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "config.yaml")
		original := []byte("site_name: \"Customised\"\n")
		if err := os.WriteFile(path, original, 0600); err != nil {
			t.Fatal(err)
		}

		if err := writeInitialConfig(path, true, ""); err != nil {
			t.Fatalf("writeInitialConfig(force): %v", err)
		}
		cfg, err := config.Load(path)
		if err != nil {
			t.Fatalf("replaced config is unloadable: %v", err)
		}
		if cfg.SiteName != "La Famille" {
			t.Errorf("SiteName = %q, want the default after --force", cfg.SiteName)
		}
		backup, err := os.ReadFile(filepath.Join(dir, initConfigBackup))
		if err != nil {
			t.Fatalf("expected %s backup: %v", initConfigBackup, err)
		}
		if string(backup) != string(original) {
			t.Errorf("backup = %q, want the original %q", backup, original)
		}
	})

	t.Run("a themed init selects the themed layout", func(t *testing.T) {
		if len(runtimeassets.CuratedLayoutNames) == 0 {
			t.Fatal("no curated themes to test with")
		}
		theme := runtimeassets.CuratedLayoutNames[0]
		path := filepath.Join(t.TempDir(), "config.yaml")
		if err := writeInitialConfig(path, false, theme); err != nil {
			t.Fatalf("writeInitialConfig(themed): %v", err)
		}
		cfg, err := config.Load(path)
		if err != nil {
			t.Fatalf("themed config is unloadable: %v", err)
		}
		if want := filepath.Join("templates", theme+".html"); cfg.Template != want {
			t.Errorf("Template = %q, want %q", cfg.Template, want)
		}
	})

	t.Run("an unknown theme is refused", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.yaml")
		err := writeInitialConfig(path, false, "not-a-real-theme")
		if err == nil {
			t.Fatal("expected an unknown-theme error, got nil")
		}
		if !strings.Contains(err.Error(), "unknown theme") {
			t.Errorf("error = %v, want an 'unknown theme' refusal", err)
		}
	})
}

// Issue #525: dying on a busy port should tell the operator how to recover,
// and unrelated serve errors must pass through unchanged.
func TestServeBindHint(t *testing.T) {
	busy := syscall.EADDRINUSE
	if got := serveBindHint(busy, 8080); !strings.Contains(got.Error(), "serve -p <port>") {
		t.Errorf("serveBindHint(EADDRINUSE) = %v, want a `serve -p <port>` recovery hint", got)
	}
	if got := serveBindHint(fmt.Errorf("listen tcp 127.0.0.1:8080: bind: address already in use"), 8123); !strings.Contains(got.Error(), "port 8123") {
		t.Errorf("serveBindHint(bind error) = %v, want the port named", got)
	}
	plain := fmt.Errorf("some other failure")
	if got := serveBindHint(plain, 8080); got.Error() != plain.Error() {
		t.Errorf("serveBindHint(non-bind error) = %v, want unchanged %v", got, plain)
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

	buildFlags := []string{"content", "output", "asset-dir", "template", "site-url", "siteurl"}
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

	// Test check command flags
	checkCmd, _, err := rootCmd.Find([]string{"check"})
	if err != nil {
		t.Fatalf("Failed to find check command: %v", err)
	}

	if checkCmd.Flags().Lookup("content") == nil {
		t.Errorf("checkCmd is missing expected flag: content")
	}
}

func TestCLICacheStatusLogging(t *testing.T) {
	tmpDir := t.TempDir()
	contentDir := filepath.Join(tmpDir, "content")
	outputDir := filepath.Join(tmpDir, "public")
	templateDir := filepath.Join(tmpDir, "templates")

	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(templateDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(templateDir, "layout.html"), []byte("<html><body>{{.Content}}</body></html>"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(contentDir, "index.md"), []byte("# Test Page"), 0600); err != nil {
		t.Fatal(err)
	}

	exePath := sharedGateBinary()

	// First run: should log cache=miss
	cmdRun1 := exec.Command(exePath, "build", "--content", contentDir, "--output", outputDir, "--template", filepath.Join(templateDir, "layout.html"))
	cmdRun1.Dir = tmpDir
	var stderr1 bytes.Buffer
	cmdRun1.Stderr = &stderr1
	if err := cmdRun1.Run(); err != nil {
		t.Fatalf("first build run failed: %v, stderr: %s", err, stderr1.String())
	}
	if !strings.Contains(stderr1.String(), "cache=miss") {
		t.Errorf("expected stderr to contain 'cache=miss' on initial build, got: %s", stderr1.String())
	}

	// Second run: should log cache=hit
	cmdRun2 := exec.Command(exePath, "build", "--content", contentDir, "--output", outputDir, "--template", filepath.Join(templateDir, "layout.html"))
	cmdRun2.Dir = tmpDir
	var stderr2 bytes.Buffer
	cmdRun2.Stderr = &stderr2
	if err := cmdRun2.Run(); err != nil {
		t.Fatalf("second build run failed: %v, stderr: %s", err, stderr2.String())
	}
	if !strings.Contains(stderr2.String(), "cache=hit") {
		t.Errorf("expected stderr to contain 'cache=hit' on repeated build, got: %s", stderr2.String())
	}
}

func TestCLISiteURLOverride(t *testing.T) {
	tmpDir := t.TempDir()
	contentDir := filepath.Join(tmpDir, "content")
	outputDir := filepath.Join(tmpDir, "public")
	templateDir := filepath.Join(tmpDir, "templates")

	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(templateDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(templateDir, "layout.html"), []byte("<html><body>{{.Content}}</body></html>"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(contentDir, "index.md"), []byte("# Test Page"), 0600); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()

	// Test valid site-url flag
	rootCmd := setupRootCmd(cfg)
	rootCmd.SetArgs([]string{
		"build",
		"--content", contentDir,
		"--output", outputDir,
		"--template", filepath.Join(templateDir, "layout.html"),
		"--site-url", "https://my-site.example.com",
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("build failed with valid site-url: %v", err)
	}

	// Test invalid site-url flag returns validation error
	rootCmdInvalid := setupRootCmd(cfg)
	rootCmdInvalid.SetArgs([]string{
		"build",
		"--content", contentDir,
		"--output", outputDir,
		"--template", filepath.Join(templateDir, "layout.html"),
		"--site-url", "not-a-valid-url",
	})
	if err := rootCmdInvalid.Execute(); err == nil {
		t.Fatalf("expected error for invalid site-url, got nil")
	} else if !strings.Contains(err.Error(), "SiteURL must be an absolute HTTP or HTTPS URL") {
		t.Errorf("expected SiteURL validation error message, got: %v", err)
	}
}

func TestInitCommand_FreshProject(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	cfg := config.DefaultConfig()
	rootCmd := setupRootCmd(cfg)
	rootCmd.SetArgs([]string{"init"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("expected init to succeed, got: %v", err)
	}

	if _, err := os.Stat("config.yaml"); os.IsNotExist(err) {
		t.Errorf("expected config.yaml to exist")
	}

	tmplPath := filepath.Join("templates", "layout.html")
	if _, err := os.Stat(tmplPath); os.IsNotExist(err) {
		t.Errorf("expected templates/layout.html to be created by init")
	}

	contentDir := "content"
	if info, err := os.Stat(contentDir); os.IsNotExist(err) || !info.IsDir() {
		t.Errorf("expected content directory to be created by init")
	}

	assetDir := "assets"
	if info, err := os.Stat(assetDir); os.IsNotExist(err) || !info.IsDir() {
		t.Errorf("expected assets directory to be created by init")
	}
	for _, rel := range []string{"css/search.css", "css/theme-foundations.css", "css/theme.css", "css/layout-editorial.css", "css/layout-midnight.css", "js/search.js", "graph/explorer.css", "graph/explorer.js", "img/mascot-default.jpeg"} {
		if _, err := os.Stat(filepath.Join(assetDir, filepath.FromSlash(rel))); err != nil {
			t.Errorf("expected embedded asset %s to be created by init: %v", rel, err)
		}
	}
}

func TestInitCommand_ThemeFlagSelectsBundledLayout(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	cfg := config.DefaultConfig()
	rootCmd := setupRootCmd(cfg)
	rootCmd.SetArgs([]string{"init", "--theme", "layout-octoburger"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("expected init --theme to succeed, got: %v", err)
	}

	configBytes, err := os.ReadFile("config.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(configBytes), `template: "templates/layout-octoburger.html"`) {
		t.Errorf("expected config.yaml template to point at the octoburger layout, got:\n%s", configBytes)
	}

	for _, rel := range []string{
		filepath.Join("templates", "layout-octoburger.html"),
		filepath.Join("templates", "layout-terminal.html"),
	} {
		if _, err := os.Stat(rel); os.IsNotExist(err) {
			t.Errorf("expected bundled layout %s to be installed by init --theme", rel)
		}
	}
}

func TestInitCommand_UnknownThemeListsChoices(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	cfg := config.DefaultConfig()
	rootCmd := setupRootCmd(cfg)
	rootCmd.SetArgs([]string{"init", "--theme", "definitely-not-a-theme"})

	err = rootCmd.Execute()
	if err == nil {
		t.Fatal("expected unknown theme to fail")
	}
	for _, want := range []string{"layout-octoburger", "layout-terminal"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("expected error to list bundled theme %q, got: %v", want, err)
		}
	}
}

func TestInitCommand_PlainInitInstallsBundledThemes(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	cfg := config.DefaultConfig()
	rootCmd := setupRootCmd(cfg)
	rootCmd.SetArgs([]string{"init"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("expected plain init to succeed, got: %v", err)
	}

	configBytes, err := os.ReadFile("config.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(configBytes), `template: "templates/layout.html"`) {
		t.Errorf("plain init must keep the default template in config.yaml, got:\n%s", configBytes)
	}
	for _, rel := range []string{
		filepath.Join("templates", "layout.html"),
		filepath.Join("templates", "layout-octoburger.html"),
		filepath.Join("templates", "layout-terminal.html"),
	} {
		if _, err := os.Stat(rel); os.IsNotExist(err) {
			t.Errorf("expected bundled layout %s to be installed by init", rel)
		}
	}
}

func TestThemesCommandListsCuratedThemes(t *testing.T) {
	cfg := config.DefaultConfig()
	rootCmd := setupRootCmd(cfg)
	var out strings.Builder
	rootCmd.SetOut(&out)
	rootCmd.SetArgs([]string{"themes"})

	// themes must stay runnable without a site config: discovery is exactly
	// what a binary-only operator needs before any project exists.
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("expected themes to succeed, got: %v", err)
	}
	for _, theme := range runtimeassets.CuratedLayoutNames {
		if !strings.Contains(out.String(), theme) {
			t.Errorf("expected themes output to list %q, got:\n%s", theme, out.String())
		}
	}
	if !strings.Contains(out.String(), "flagship") {
		t.Errorf("expected themes output to include descriptions, got:\n%s", out.String())
	}
}

func TestInitCommand_ScaffoldsDemoContent(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	cfg := config.DefaultConfig()
	rootCmd := setupRootCmd(cfg)
	rootCmd.SetArgs([]string{"init", "--theme", "layout-octoburger"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("expected init to succeed, got: %v", err)
	}

	indexBytes, err := os.ReadFile(filepath.Join("content", "index.md"))
	if err != nil {
		t.Fatalf("expected init to scaffold content/index.md: %v", err)
	}
	for _, want := range []string{"title:", "date:", "la-famille new", "la-famille serve --watch"} {
		if !strings.Contains(string(indexBytes), want) {
			t.Errorf("scaffolded index.md missing %q, got:\n%s", want, indexBytes)
		}
	}

	themingBytes, err := os.ReadFile(filepath.Join("content", "theming.md"))
	if err != nil {
		t.Fatalf("expected init to scaffold content/theming.md: %v", err)
	}
	// The demo must visibly switch away from the chosen site default.
	if !strings.Contains(string(themingBytes), "\nlayout: ") || strings.Contains(string(themingBytes), "layout: layout-octoburger") {
		t.Errorf("scaffolded theming.md should pin a non-default bundled layout, got:\n%s", themingBytes)
	}

	// Re-running init over the scaffolded site must not clobber it.
	if err := os.WriteFile(filepath.Join("content", "index.md"), []byte("---\ntitle: \"Mine\"\n---\n\n# Mine\n"), 0600); err != nil {
		t.Fatal(err)
	}
	reRun := setupRootCmd(cfg)
	reRun.SetArgs([]string{"init", "--force"})
	if err := reRun.Execute(); err != nil {
		t.Fatalf("expected re-run init --force to succeed, got: %v", err)
	}
	got, err := os.ReadFile(filepath.Join("content", "index.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "# Mine") {
		t.Errorf("re-run of init replaced operator content index.md, got:\n%s", got)
	}
}

func TestInitCommand_ExistingConfigRefusalAndForce(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	if err := os.WriteFile("config.yaml", []byte("site_name: Custom Site\n"), 0600); err != nil {
		t.Fatal(err)
	}

	// 1. Existing init without --force should fail
	cfg := config.DefaultConfig()
	rootCmd1 := setupRootCmd(cfg)
	rootCmd1.SetArgs([]string{"init"})
	if err := rootCmd1.Execute(); err == nil {
		t.Fatalf("expected init without --force to fail on existing config.yaml, got nil")
	}

	// 2. Existing init with --force should succeed and backup existing config
	rootCmd2 := setupRootCmd(cfg)
	rootCmd2.SetArgs([]string{"init", "--force"})
	if err := rootCmd2.Execute(); err != nil {
		t.Fatalf("expected init --force to succeed on existing config.yaml, got: %v", err)
	}

	if _, err := os.Stat("config.yaml.bak"); os.IsNotExist(err) {
		t.Errorf("expected config.yaml.bak backup file to exist after init --force")
	}
}

func TestServeCommand_InitialBuildFailure_ReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Clean directory with no template layout
	cfg := config.DefaultConfig()
	cfg.Template = filepath.Join(tmpDir, "templates", "layout.html")

	rootCmd := setupRootCmd(cfg)
	rootCmd.SetArgs([]string{"serve"})

	err = rootCmd.Execute()
	if err == nil {
		t.Fatalf("expected serve command to fail on initial build failure in clean directory, got nil")
	}
	if !strings.Contains(err.Error(), "initial build failed") {
		t.Errorf("expected error to contain 'initial build failed', got: %v", err)
	}
}

func TestBuildSiteURLEnvironmentVariables(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	if err := os.MkdirAll("content", 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join("content", "index.md"), []byte("---\ntitle: Home\n---\n# Home"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll("templates", 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join("templates", "layout.html"), []byte("<html><body>{{.Content}}</body></html>"), 0600); err != nil {
		t.Fatal(err)
	}

	t.Run("SITE_URL env variable sets SiteURL when flag is absent", func(t *testing.T) {
		t.Setenv("SITE_URL", "https://env.example.com")
		cfg := config.DefaultConfig()
		rootCmd := setupRootCmd(cfg)
		rootCmd.SetArgs([]string{"build"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("build failed: %v", err)
		}

		sitemap, err := os.ReadFile(filepath.Join("public", "sitemap.xml"))
		if err != nil {
			t.Fatalf("read sitemap.xml: %v", err)
		}
		if !strings.Contains(string(sitemap), "https://env.example.com/") {
			t.Errorf("expected sitemap to contain env SITE_URL, got:\n%s", sitemap)
		}
	})

	t.Run("CLI flag overrides SITE_URL env variable", func(t *testing.T) {
		t.Setenv("SITE_URL", "https://env.example.com")
		cfg := config.DefaultConfig()
		rootCmd := setupRootCmd(cfg)
		rootCmd.SetArgs([]string{"build", "--site-url", "https://flag.example.com"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("build failed: %v", err)
		}

		sitemap, err := os.ReadFile(filepath.Join("public", "sitemap.xml"))
		if err != nil {
			t.Fatalf("read sitemap.xml: %v", err)
		}
		if !strings.Contains(string(sitemap), "https://flag.example.com/") {
			t.Errorf("expected sitemap to contain CLI flag siteurl, got:\n%s", sitemap)
		}
	})
}

func TestGitHubPagesWorkflowAudit(t *testing.T) {
	// Root of repo is 2 directories up from cmd/la-famille
	workflowPath := filepath.Join("..", "..", ".github", "workflows", "deploy.yml")
	content, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("failed to read GitHub Pages deploy workflow at %s: %v", workflowPath, err)
	}

	sContent := string(content)

	// Regression check: configure-pages must run before build static site
	configurePagesIdx := strings.Index(sContent, "actions/configure-pages")
	buildSiteIdx := strings.Index(sContent, "go run ./cmd/la-famille build")

	if configurePagesIdx == -1 {
		t.Fatalf("deploy.yml missing actions/configure-pages step")
	}
	if buildSiteIdx == -1 {
		t.Fatalf("deploy.yml missing la-famille build step")
	}
	if configurePagesIdx >= buildSiteIdx {
		t.Errorf("deploy.yml must run actions/configure-pages BEFORE building static site so base_url is available")
	}

	// Regression check: build step must reference SITE_URL / base_url / site-url flag
	if !strings.Contains(sContent, "steps.pages.outputs.base_url") {
		t.Errorf("deploy.yml must reference steps.pages.outputs.base_url for public site URL")
	}
	if !strings.Contains(sContent, "--site-url") {
		t.Errorf("deploy.yml build step must pass --site-url flag")
	}
	for _, required := range []string{
		"LA_FAMILLE_VERSION",
		"SHA256SUMS",
		"--project-root",
		"--output \"$GITHUB_WORKSPACE/public\"",
		"publish-check",
		"actions/upload-pages-artifact",
	} {
		if !strings.Contains(sContent, required) {
			t.Errorf("deploy.yml missing publishing contract marker %q", required)
		}
	}
}
