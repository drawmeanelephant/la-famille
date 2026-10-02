package retrieval

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/tbuddy/la-famille/internal/sitedata"
)

// LoadPayload builds a content-only corpus from supplied bytes. It never reads
// the filesystem. Metadata is optional, but any supplied artifact must parse.
func LoadPayload(source string, content io.Reader, artifacts map[string][]byte) (LoadResult, error) {
	result := LoadResult{Corpus: Corpus{Version: "v1", SourceDir: source}}
	bundle, err := parseRAGReader("rag-content.md", content, true)
	if err != nil {
		result.MalformedArtifact = "rag-content.md"
		return result, fmt.Errorf("retrieval: parse rag-content.md: %w", err)
	}
	bundle.name = "rag-content.md"
	result.Corpus.DocumentCount = len(bundle.files)

	var manifest sitedata.Manifest
	if data, ok := artifacts[sitedata.ManifestFileName]; ok {
		manifest, err = sitedata.ParseManifest(data)
		if err != nil {
			return result, fmt.Errorf("retrieval: %s: %w", sitedata.ManifestFileName, err)
		}
	}
	pageIDs, err := payloadPageIDs(bundle.files, manifest)
	if err != nil {
		return result, fmt.Errorf("retrieval: site-manifest.json: %w", err)
	}
	appendBundle(&result.Corpus, bundle, func(f parsedFile) []Chunk {
		identity := pageIDs[f.path]
		if identity == "" {
			identity = derivePageID(f.path, "")
		}
		return chunkFileForPage(f.text, f.path, identity, bundle.name)
	})
	read := func(name string) ([]byte, error) {
		data, ok := artifacts[name]
		if !ok {
			return nil, os.ErrNotExist
		}
		if strings.TrimSpace(string(data)) == "null" {
			return nil, fmt.Errorf("%s: null artifact is not supported", name)
		}
		return data, nil
	}
	if err := enrichCorpus(&result.Corpus, read, true); err != nil {
		return result, fmt.Errorf("retrieval: metadata: %w", err)
	}
	if warnings := loadLinkGraphArtifacts(&result.Corpus, read); len(warnings) > 0 {
		return result, fmt.Errorf("retrieval: graph metadata: %s", strings.Join(warnings, "; "))
	}
	return finishLoad(result)
}

// Pack v1 has no content_dir field. Manifest sources are content-tree-relative;
// RAG paths include the root. Resolve a single root shared by all documents,
// not independent longest suffixes that can collapse distinct nested pages.
func payloadPageIDs(files []parsedFile, manifest sitedata.Manifest) (map[string]string, error) {
	if len(manifest.Pages) == 0 {
		return nil, nil
	}
	pages := make(map[string]string, len(manifest.Pages))
	for _, page := range manifest.Pages {
		if page.SourcePath == "" || page.Identity == "" {
			return nil, fmt.Errorf("page requires source_path and identity")
		}
		if _, exists := pages[page.SourcePath]; exists {
			return nil, fmt.Errorf("duplicate source_path %q", page.SourcePath)
		}
		pages[page.SourcePath] = page.Identity
	}
	var roots map[string]bool
	for _, f := range files {
		if strings.HasSuffix(f.path, "/") {
			continue
		}
		if roots == nil {
			roots = make(map[string]bool)
			for source := range pages {
				if f.path == source {
					roots[""] = true
				} else if strings.HasSuffix(f.path, "/"+source) {
					roots[strings.TrimSuffix(f.path, source)] = true
				}
			}
		} else {
			for root := range roots {
				if !strings.HasPrefix(f.path, root) || pages[strings.TrimPrefix(f.path, root)] == "" {
					delete(roots, root)
				}
			}
		}
		if len(roots) == 0 {
			return nil, fmt.Errorf("rag-content.md paths do not match a consistent content root")
		}
	}
	if len(roots) != 1 {
		return nil, fmt.Errorf("rag-content.md content root is ambiguous")
	}
	var root string
	for candidate := range roots {
		root = candidate
	}
	ids := make(map[string]string, len(files))
	for _, f := range files {
		ids[f.path] = pages[strings.TrimPrefix(f.path, root)]
	}
	return ids, nil
}
