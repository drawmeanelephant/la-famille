package main

import (
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir, err := os.MkdirTemp("", "la-famille-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Set up content directory
	contentDir := filepath.Join(tmpDir, "content")
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatalf("Failed to create content dir: %v", err)
	}

	// Create a mock markdown file
	mockMD := filepath.Join(contentDir, "test.md")
	if err := os.WriteFile(mockMD, []byte("# Hello Test\nThis is a test."), 0644); err != nil {
		t.Fatalf("Failed to write mock markdown file: %v", err)
	}

	// Set up templates directory
	templatesDir := filepath.Join(tmpDir, "templates")
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		t.Fatalf("Failed to create templates dir: %v", err)
	}

	// Create a mock template file
	mockTemplate := filepath.Join(templatesDir, "layout.html")
	templateContent := `<html><head><title>{{.Title}}</title></head><body>{{.Content}}</body></html>`
	if err := os.WriteFile(mockTemplate, []byte(templateContent), 0644); err != nil {
		t.Fatalf("Failed to write mock template file: %v", err)
	}

	// Output directory
	outputDir := filepath.Join(tmpDir, "public")

	// Execute run function
	if err := run(contentDir, mockTemplate, outputDir); err != nil {
		t.Fatalf("run() returned an error: %v", err)
	}

	// Verify output
	outputFile := filepath.Join(outputDir, "test.md.html")
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Errorf("Expected output file %s was not created", outputFile)
	}

	// Verify content
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	outputStr := string(content)
	if !strings.Contains(outputStr, "<title>test.md</title>") {
		t.Errorf("Output HTML does not contain expected title, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "<h1>Hello Test</h1>") {
		t.Errorf("Output HTML does not contain expected heading, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "<p>This is a test.</p>") {
		t.Errorf("Output HTML does not contain expected paragraph, got: %s", outputStr)
	}
}

func TestProcessFile(t *testing.T) {
	// Setup mock input and output directories
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	outputDir := filepath.Join(tempDir, "public")
	err := os.MkdirAll(contentDir, 0755)
	if err != nil {
		t.Fatalf("failed to create content dir: %v", err)
	}
	err = os.MkdirAll(outputDir, 0755)
	if err != nil {
		t.Fatalf("failed to create output dir: %v", err)
	}

	// Create a dummy template
	tmpl, err := template.New("layout").Parse("<html><body>{{.Content}}</body></html>")
	if err != nil {
		t.Fatalf("failed to parse template: %v", err)
	}

	// Create a dummy markdown file
	fileName := "test.md"
	content := []byte("# Hello World\nThis is a test.")
	err = os.WriteFile(filepath.Join(contentDir, fileName), content, 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Run processFile
	err = processFile(fileName, contentDir, outputDir, tmpl)
	if err != nil {
		t.Errorf("processFile failed: %v", err)
	}

	// Assert the output file exists
	outFileName := fileName + ".html"
	_, err = os.Stat(filepath.Join(outputDir, outFileName))
	if os.IsNotExist(err) {
		t.Errorf("expected output file %s does not exist", outFileName)
	} else if err != nil {
		t.Errorf("failed to check output file: %v", err)
	}
}

func TestMainXSS(t *testing.T) {
	// Create temporary directories
	tempDir, err := os.MkdirTemp("", "la-famille-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	contentDir := filepath.Join(tempDir, "content")
	templateDir := filepath.Join(tempDir, "templates")
	outputDir := filepath.Join(tempDir, "public")

	os.MkdirAll(contentDir, 0755)
	os.MkdirAll(templateDir, 0755)
	os.MkdirAll(outputDir, 0755)

	// Create test template
	templateFile := filepath.Join(templateDir, "layout.html")
	err = os.WriteFile(templateFile, []byte("<html><body>{{.Content}}</body></html>"), 0644)
	if err != nil {
		t.Fatalf("Failed to write template file: %v", err)
	}

	// Create test markdown with XSS payload
	testFile := filepath.Join(contentDir, "test.md")
	err = os.WriteFile(testFile, []byte("## Title\n\n<script>alert('xss')</script>"), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Change working directory to temp dir
	originalWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalWd)

	// Run main
	main()

	// Check output
	outputFile := filepath.Join(outputDir, "test.md.html")
	outputContent, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	outputStr := string(outputContent)
	if strings.Contains(outputStr, "<script>") {
		t.Errorf("XSS payload was not sanitized: %s", outputStr)
	}

	if !strings.Contains(outputStr, "<h2>Title</h2>") {
		t.Errorf("Markdown was not rendered correctly: %s", outputStr)
	}
}

func TestRun_TemplateParsingError(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatalf("Failed to create content dir: %v", err)
	}

	outputDir := filepath.Join(tempDir, "public")
	templateFile := "non_existent_template.html"

	err := run(contentDir, templateFile, outputDir)
	if err == nil {
		t.Fatalf("Expected error for non-existent template file, got nil")
	}

	if !strings.Contains(err.Error(), "failed to parse template file") {
		t.Errorf("Expected error to mention 'failed to parse template file', got: %v", err)
	}
}

func TestRun_MkdirAllError(t *testing.T) {
	tempDir := t.TempDir()

	// Create a read-only directory
	readOnlyDir := filepath.Join(tempDir, "readonly")
	if err := os.Mkdir(readOnlyDir, 0555); err != nil {
		t.Fatalf("failed to create read-only dir: %v", err)
	}

	// Try to use a subdirectory of the read-only directory as outputDir
	outputDir := filepath.Join(readOnlyDir, "public")

	err := run("content", "templates/layout.html", outputDir)
	if err == nil {
		t.Errorf("expected error when output directory cannot be created, got nil")
	}
}

func TestRun_ReadDirError(t *testing.T) {
	tempDir := t.TempDir()

	// Use a non-existent directory as contentDir
	contentDir := filepath.Join(tempDir, "non_existent_content_dir")
	outputDir := filepath.Join(tempDir, "public")

	// We need a valid template so that template.ParseFiles succeeds before ReadDir is called.
	// Oh wait, ReadDir is called BEFORE template.ParseFiles in the code!
	// Let's create a valid output dir and mock template anyway just in case.
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatalf("Failed to create output dir: %v", err)
	}
	templateFile := filepath.Join(tempDir, "layout.html")
	if err := os.WriteFile(templateFile, []byte("<html></html>"), 0644); err != nil {
		t.Fatalf("Failed to write mock template file: %v", err)
	}

	err := run(contentDir, templateFile, outputDir)
	if err == nil {
		t.Fatalf("Expected error when contentDir does not exist, got nil")
	}

	if !strings.Contains(err.Error(), "failed to read content directory") {
		t.Errorf("Expected error to mention 'failed to read content directory', got: %v", err)
	}
}

func TestProcessFile_ReadFileError(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	outputDir := filepath.Join(tempDir, "public")

	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatalf("Failed to create content dir: %v", err)
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatalf("Failed to create output dir: %v", err)
	}

	tmpl, err := template.New("layout").Parse("<html><body>{{.Content}}</body></html>")
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	// Create a directory that ends with .md so os.ReadFile will fail
	badFileName := "bad_file.md"
	if err := os.Mkdir(filepath.Join(contentDir, badFileName), 0755); err != nil {
		t.Fatalf("Failed to create bad markdown dir: %v", err)
	}

	err = processFile(badFileName, contentDir, outputDir, tmpl)
	if err == nil {
		t.Fatalf("Expected processFile to fail when reading a directory, got nil")
	}
}

func TestProcessFile_CreateOutputError(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	outputDir := filepath.Join(tempDir, "public")

	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatalf("Failed to create content dir: %v", err)
	}
	// Make output dir read-only
	if err := os.MkdirAll(outputDir, 0555); err != nil {
		t.Fatalf("Failed to create read-only output dir: %v", err)
	}

	tmpl, err := template.New("layout").Parse("<html><body>{{.Content}}</body></html>")
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	fileName := "test.md"
	if err := os.WriteFile(filepath.Join(contentDir, fileName), []byte("# Hello"), 0644); err != nil {
		t.Fatalf("Failed to write mock file: %v", err)
	}

	err = processFile(fileName, contentDir, outputDir, tmpl)
	if err == nil {
		t.Fatalf("Expected processFile to fail when creating output in read-only dir, got nil")
	}
}

func TestRun_ProcessFileErrorLog(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	outputDir := filepath.Join(tempDir, "public")
	templateFile := filepath.Join(tempDir, "layout.html")

	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatalf("Failed to create content dir: %v", err)
	}
	// Make output dir read-only
	if err := os.MkdirAll(outputDir, 0555); err != nil {
		t.Fatalf("Failed to create output dir: %v", err)
	}
	if err := os.WriteFile(templateFile, []byte("<html><body>{{.Content}}</body></html>"), 0644); err != nil {
		t.Fatalf("Failed to write template file: %v", err)
	}

	fileName := "test.md"
	if err := os.WriteFile(filepath.Join(contentDir, fileName), []byte("# Hello"), 0644); err != nil {
		t.Fatalf("Failed to write mock file: %v", err)
	}

	// run should not return an error but should log it.
	err := run(contentDir, templateFile, outputDir)
	if err != nil {
		t.Fatalf("Expected run to not return error when processFile fails, got %v", err)
	}
}
