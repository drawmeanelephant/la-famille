package main

import (
	"bytes"
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
			err := processFile(file.Name(), contentDir, outputDir, tmpl)
			if err != nil {
				log.Printf("Error processing %s: %v", file.Name(), err)
			}
		}
	}
}

func processFile(fileName, contentDir, outputDir string, tmpl *template.Template) error {
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

	// Render
	page := Page{
		Title:   fileName,
		Content: template.HTML(buf.String()),
	}

	outFile, err := os.Create(filepath.Join(outputDir, fileName+".html"))
	if err != nil {
		return err
	}
	defer outFile.Close()
	return tmpl.Execute(outFile, page)
}
