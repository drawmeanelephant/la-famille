package main

import (
	"bytes"
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
	contentDir := "content"
	templateFile := "templates/layout.html"
	outputDir := "public"

	os.MkdirAll(outputDir, 0755)

	files, err := os.ReadDir(contentDir)
	if err != nil {
		log.Fatal(err)
	}

	tmpl := template.Must(template.ParseFiles(templateFile))

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

			// Sanitize HTML
			p := bluemonday.UGCPolicy()
			sanitizedHTML := p.SanitizeBytes(buf.Bytes())

			// Render
			page := Page{
				Title:   file.Name(),
				Content: template.HTML(sanitizedHTML),
			}

			outFile, err := os.Create(filepath.Join(outputDir, file.Name()+".html"))
			if err != nil {
				log.Printf("Error creating %s: %v", file.Name()+".html", err)
				continue
			}
			tmpl.Execute(outFile, page)
			outFile.Close()
		}
	}
}
