package retrieval

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/tbuddy/la-famille/internal/ragfmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// LoadResult is the success envelope from Load. It records what we found
// (and what we couldn't find) so the orchestrator can produce an actionable
// error if the archive is stale or malformed.
type LoadResult struct {
	Corpus            Corpus
	MissingArtifacts  []string // e.g. ["rag-content.md"] when not found
	MalformedArtifact string   // one of the artifact paths that could not be parsed (empty if all OK)
}

// LoadOptions configures where Load reads from.
type LoadOptions struct {
	RagDir    string // directory containing rag-{system,config,content}.md
	OutputDir string // directory containing meta.json/graph.json/search.json (optional)
}

// Load reads the RAG archive and any optional generated-site metadata,
// returning a built Corpus. It performs the following checks in order:
//
//  1. RagDir exists and is a directory.
//  2. At least one of the RAG bundles is present; missing ones are reported
//     in MissingArtifacts (callers decide whether that's fatal).
//  3. Each present bundle is well-formed XML-tagged markdown; malformed inputs
//     set MalformedArtifact and short-circuit the load.
//
// Load never returns an empty corpus — if nothing could be parsed, it returns
// an error explaining which artifact was at fault.
func Load(opts LoadOptions) (LoadResult, error) {
	if strings.TrimSpace(opts.RagDir) == "" {
		return LoadResult{}, errors.New("retrieval: RagDir is required")
	}
	info, err := os.Stat(opts.RagDir)
	if err != nil {
		return LoadResult{}, fmt.Errorf("retrieval: read RagDir: %w", err)
	}
	if !info.IsDir() {
		return LoadResult{}, fmt.Errorf("retrieval: %s is not a directory", opts.RagDir)
	}

	result := LoadResult{
		Corpus: Corpus{
			Version:   "v1",
			SourceDir: opts.RagDir,
		},
	}

	var bundles []parsedBundle
	for _, name := range []string{"rag-content.md", "rag-system.md", "rag-config.md"} {
		bundlePath := filepath.Join(opts.RagDir, name)
		b, err := parseRAGBundle(bundlePath)
		if errors.Is(err, os.ErrNotExist) {
			result.MissingArtifacts = append(result.MissingArtifacts, name)
			continue
		}
		if err != nil {
			result.MalformedArtifact = name
			return result, fmt.Errorf("retrieval: parse %s: %w", name, err)
		}
		result.Corpus.DocumentCount += len(b.files)
		bundles = append(bundles, b)
	}

	// Optional: enrich from generated-site metadata
	if opts.OutputDir != "" {
		if err := enrichCorpusWithSiteMeta(&result.Corpus, opts.OutputDir); err != nil {
			// Non-fatal: log via returned warning but keep the corpus usable
			result.MalformedArtifact = "meta.json"
		}
	}

	for _, b := range bundles {
		for _, f := range b.files {
			chunks := chunkFile(f.text, f.path)
			result.Corpus.Chunks = append(result.Corpus.Chunks, chunks...)
		}
	}

	result.Corpus.ChunkCount = len(result.Corpus.Chunks)

	// Stable ordering for determinism across runs.
	sort.SliceStable(result.Corpus.Chunks, func(i, j int) bool {
		return result.Corpus.Chunks[i].ID < result.Corpus.Chunks[j].ID
	})

	if result.Corpus.ChunkCount == 0 {
		return result, fmt.Errorf("retrieval: %s produced no chunks (missing or empty)", opts.RagDir)
	}
	return result, nil
}

// parsedBundle is the in-memory representation of a single RAG file.
type parsedBundle struct {
	files []parsedFile
}

type parsedFile struct {
	path string
	text string
}

// parseRAGBundle reads a single rag-*.md file and splits it into the
// <file path="..."><content>...</content></file> blocks written by
// internal/ragexport. The parser is intentionally tolerant: it ignores
// lines that don't fit the structure and surfaces only the most egregious
// malformation cases (e.g. an unterminated <file> block).
func parseRAGBundle(p string) (parsedBundle, error) {
	f, err := os.Open(p)
	if err != nil {
		return parsedBundle{}, err
	}
	defer f.Close()

	var out parsedBundle
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1<<16), 1<<24) // 16 MB

	var (
		currentFile *parsedFile
		buf         bytes.Buffer
		currentPath string
	)

	scanned := false
	for scanner.Scan() {
		scanned = true
		line := scanner.Text()

		switch {
		case strings.HasPrefix(line, "<file "):
			// <file path="...">
			if currentFile != nil {
				// A second <file> opened before the previous closed means the
				// previous block was malformed. Surface that as an error
				// rather than silently swallowing blocks.
				return parsedBundle{}, fmt.Errorf("%s: unterminated <file> block before path=%q", p, currentPath)
			}
			currentFile = &parsedFile{}
			buf.Reset()
			attr := extractPathAttr(line)
			if attr == "" {
				return parsedBundle{}, fmt.Errorf("%s: missing path attribute on <file>", p)
			}
			currentPath = attr
		case strings.HasPrefix(line, "<content>") && currentFile != nil:
			buf.Reset()
			// Re-init the current file so we don't leak partial state from
			// any earlier lines that pre-date <content>.
			currentFile = &parsedFile{path: currentPath}
		case strings.HasPrefix(line, "</content>") && currentFile != nil:
			currentFile.text = buf.String()
			out.files = append(out.files, *currentFile)
			currentFile = nil
			currentPath = ""
		case strings.HasPrefix(line, "</file>"):
			if currentFile != nil {
				return parsedBundle{}, fmt.Errorf("%s: </file> appeared inside an unclosed <content> block (path=%q)", p, currentPath)
			}
		case strings.HasPrefix(line, "<content>") && currentFile == nil:
			return parsedBundle{}, fmt.Errorf("%s: <content> outside of <file>", p)
		case currentFile != nil:
			if buf.Len() > 0 {
				buf.WriteByte('\n')
			}
			// Undo the write-time escaping so a body line that looks like
			// archive structure is restored verbatim.
			buf.WriteString(ragfmt.UnescapeLine(line))
		}
	}
	if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
		return parsedBundle{}, fmt.Errorf("%s: scan: %w", p, err)
	}
	if !scanned {
		return parsedBundle{}, fmt.Errorf("%s: empty file", p)
	}
	if currentFile != nil {
		return parsedBundle{}, fmt.Errorf("%s: unterminated <file> block at EOF (last path=%q)", p, currentPath)
	}
	return out, nil
}

// extractPathAttr pulls the path="..." value out of a <file ...> line.
// Returns empty string if absent or malformed. It supports double quotes only
// because that's the only form ragexport produces.
func extractPathAttr(line string) string {
	const key = `path="`
	i := strings.Index(line, key)
	if i < 0 {
		return ""
	}
	rest := line[i+len(key):]
	j := strings.IndexByte(rest, '"')
	if j < 0 {
		return ""
	}
	return rest[:j]
}

// metaEntry is a minimal projection of the meta.json schema. We deliberately
// don't import the generator package to avoid a circular dependency with the
// asset pipeline.
type metaEntry struct {
	ID    string `json:"id"`
	URL   string `json:"url"`
	Title string `json:"title"`
	Type  string `json:"type"`
}

// searchIndexEntry is a minimal projection of search.json. We use it to
// skim titles/tags for corpus enrichment.
type searchIndexEntry struct {
	Title string   `json:"t"`
	URL   string   `json:"u"`
	Tags  []string `json:"g,omitempty"`
}

func enrichCorpusWithSiteMeta(c *Corpus, outputDir string) error {
	// meta.json: {pages:[{id,url,title}]} – apply URL/title back to chunks.
	metaPath := filepath.Join(outputDir, "meta.json")
	if data, err := os.ReadFile(metaPath); err == nil {
		var doc struct {
			Pages []metaEntry `json:"pages"`
		}
		if err := json.Unmarshal(data, &doc); err == nil {
			urlByID := map[string]string{}
			titleByID := map[string]string{}
			for _, p := range doc.Pages {
				if p.ID != "" {
					urlByID[p.ID] = p.URL
					titleByID[p.ID] = p.Title
				}
			}
			for i, ch := range c.Chunks {
				if ch.PageID == "" {
					continue
				}
				if u, ok := urlByID[ch.PageID]; ok && ch.URL == "" {
					ch.URL = u
				}
				if t, ok := titleByID[ch.PageID]; ok && ch.Title == "" {
					ch.Title = t
				}
				c.Chunks[i] = ch
			}
		}
	}

	// search.json: list of items w/ title/URL/tags. Currently only used to
	// backfill titles when the corpus has none. Cheap to read.
	searchPath := filepath.Join(outputDir, "search.json")
	if data, err := os.ReadFile(searchPath); err == nil {
		var entries []searchIndexEntry
		if err := json.Unmarshal(data, &entries); err == nil {
			titleByURL := map[string]string{}
			for _, e := range entries {
				if e.URL != "" {
					titleByURL[normaliseURL(e.URL)] = e.Title
				}
			}
			for i, ch := range c.Chunks {
				if ch.Title == "" && ch.URL != "" {
					if t, ok := titleByURL[normaliseURL(ch.URL)]; ok {
						ch.Title = t
						c.Chunks[i] = ch
					}
				}
			}
		}
	}
	return nil
}

func normaliseURL(u string) string {
	u = strings.TrimSpace(u)
	u = strings.TrimRight(u, "/")
	return strings.ToLower(u)
}
