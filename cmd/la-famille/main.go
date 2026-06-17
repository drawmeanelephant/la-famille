package main

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"

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
			// Read content
			content, err := os.ReadFile(filepath.Join(contentDir, file.Name()))
			if err != nil {
				log.Printf("Error reading %s: %v", file.Name(), err)
				continue
			}

			// Convert Markdown to HTML
			var buf bytes.Buffer
			if err := goldmark.Convert(content, &buf); err != nil {
				log.Printf("Error converting %s: %v", file.Name(), err)
				continue
			}

			// Render
			page := Page{
				Title:   file.Name(),
				Content: template.HTML(buf.String()),
			}

			outFile, err := os.Create(filepath.Join(outputDir, file.Name()+".html"))
			if err != nil {
				log.Printf("Error creating %s: %v", file.Name()+".html", err)
				continue
			}
			err = tmpl.Execute(outFile, page)
			outFile.Close()
			if err != nil {
				log.Printf("Error executing template for %s: %v", file.Name(), err)
				continue
			}
		}
	}
	return nil
}
