package main

import (
	"bufio"
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

	buildFlags := []string{"content", "output", "template", "site-url", "siteurl"}
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

	exePath := filepath.Join(tmpDir, "la-famille.bin")
	cmdBuildExe := exec.Command("go", "build", "-o", exePath, "../../cmd/la-famille")
	if err := cmdBuildExe.Run(); err != nil {
		t.Fatalf("failed to build la-famille: %v", err)
	}

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
