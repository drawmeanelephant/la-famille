package main

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
)

type Page struct {
	Title   string
	Content template.HTML
}

func main() {
	if err := run("content", "templates/layout.html", "public"); err != nil {
		log.Fatal(err)
	}
}

func run(contentDir, templateFile, outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	files, err := os.ReadDir(contentDir)
	if err != nil {
		return fmt.Errorf("failed to read content directory: %w", err)
	}

	tmpl, err := template.ParseFiles(templateFile)
	if err != nil {
		return fmt.Errorf("failed to parse template file: %w", err)
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".md" {
			err := processFile(file.Name(), contentDir, outputDir, tmpl)
			if err != nil {
				log.Printf("Error processing %s: %v", file.Name(), err)
			}
		}
	}
	return nil
}

func processFile(fileName, contentDir, outputDir string, tmpl *template.Template) (err error) {
	// Read content
	content, err := os.ReadFile(filepath.Join(contentDir, fileName))
	if err != nil {
		return err
	}

	// Convert Markdown to HTML
	var buf bytes.Buffer
	if err := goldmark.Convert(content, &buf); err != nil {
		return err
	}

	// Sanitize HTML
	p := bluemonday.UGCPolicy()
	sanitizedHTML := p.SanitizeBytes(buf.Bytes())

	// Render
	page := Page{
		Title:   fileName,
		Content: template.HTML(sanitizedHTML),
	}

	outFile, createErr := os.Create(filepath.Join(outputDir, fileName+".html"))
	if createErr != nil {
		return createErr
	}
	defer func() {
		if closeErr := outFile.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()
	return tmpl.Execute(outFile, page)
}
