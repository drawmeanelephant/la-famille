with open("internal/generator/generator.go", "r") as f:
    content = f.read()

# Instead of lock/unlock on appending to searchIndex, we can pre-allocate an array based on len(keys) and write to index i.
import os
os.system("git checkout HEAD -- internal/generator/generator.go")

with open("internal/generator/generator.go", "r") as f:
    content = f.read()

import re

if '"sync"' not in content:
    content = re.sub(r'("strings")', r'\1\n\t"sync"\n\t"runtime"', content)
elif '"runtime"' not in content:
    content = re.sub(r'("sync")', r'\1\n\t"runtime"', content)

new_loop = """
	var mu sync.Mutex
	numWorkers := runtime.NumCPU()
	if numWorkers < 1 {
		numWorkers = 1
	}

	searchIndexItems := make([]search.SearchItem, len(keys))

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

					searchIndexItems[idx] = search.SearchItem{
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
					if err := os.WriteFile(outPath, meta.Content, 0644); err != nil {
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

				md := goldmark.New(
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
					Content:         template.HTML(sanitizedHTML),
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
"""

start_str = "	for _, relPath := range keys {"
end_str = "	if len(errs) > 0 {"

start_idx = content.find(start_str)
end_idx = content.find(end_str)

buf_str = "\n\tvar buf bytes.Buffer\n"
content = content.replace(buf_str, "\n")
buf_str2 = "\tvar buf bytes.Buffer\n"
content = content.replace(buf_str2, "")

start_idx = content.find(start_str)
end_idx = content.find(end_str)

new_content = content[:start_idx] + new_loop + "\n\t// Sort errs for deterministic order\n\tif len(errs) > 0 {\n\t\tsort.Slice(errs, func(i, j int) bool {\n\t\t\treturn errs[i].Error() < errs[j].Error()\n\t\t})\n\t}\n" + content[end_idx:]

with open("internal/generator/generator.go", "w") as f:
    f.write(new_content)
