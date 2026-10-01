package sitedata

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/content"
	"github.com/tbuddy/la-famille/internal/graph"
	"github.com/tbuddy/la-famille/internal/jsonutil"
)

const (
	ManifestFileName = "site-manifest.json"
	ManifestVersion  = 2
)

// Manifest is the canonical, per-build projection of the site's page and
// link data. It intentionally has no build timestamp so identical inputs
// serialize to identical bytes.
type Manifest struct {
	Version        int            `json:"version"`
	Pages          []ManifestPage `json:"pages"`
	Files          []ManifestFile `json:"files,omitempty"`
	Sitemap        []string       `json:"sitemap,omitempty"`
	OutputCaptured bool           `json:"output_captured,omitempty"`
}

// ManifestFile fingerprints a published file other than a content page or the
// ledger itself. This includes assets and derived metadata.
type ManifestFile struct {
	Path string `json:"path"`
	Hash string `json:"hash"`
}

// ManifestPage records the stable identity and build-time relationships of a
// content page. SourcePath retains the content-tree key so check can resolve
// source Markdown links without reconstructing identities from URLs.
type ManifestPage struct {
	Identity         string            `json:"identity"`
	SourcePath       string            `json:"source_path"`
	Rendered         bool              `json:"rendered"`
	URL              string            `json:"url"`
	Title            string            `json:"title"`
	Date             string            `json:"date"`
	Tags             []string          `json:"tags"`
	Categories       []string          `json:"categories"`
	OutboundLinks    []string          `json:"outbound_links"`
	InboundLinkCount int               `json:"inbound_link_count"`
	Links            []ManifestLink    `json:"links"`
	AssetReferences  []string          `json:"asset_references"`
	ContentHash      string            `json:"content_hash,omitempty"`
	OutputHash       string            `json:"output_hash,omitempty"`
	Frontmatter      map[string]string `json:"frontmatter,omitempty"`
}

// ManifestLink records an internal Markdown or generated-output link used by
// check. GraphTarget is set for source Markdown links represented in graph
// edges; Target is the normalized source/output path used by diagnostics.
type ManifestLink struct {
	Destination string `json:"destination"`
	Target      string `json:"target"`
	GraphTarget string `json:"graph_target,omitempty"`
	Line        int    `json:"line"`
	Resolved    bool   `json:"resolved"`
}

// NewManifest projects existing content, graph, backlink, and output-path
// data into one sorted manifest. Link and asset references are supplied by
// checker, which owns the existing Markdown reference rules.
func NewManifest(
	siteCfg config.Config,
	fileMap map[string]*content.FileMeta,
	g graph.Graph,
	backlinks map[string][]string,
	pageOutputs map[string]string,
	links map[string][]ManifestLink,
	assetReferences map[string][]string,
) Manifest {
	adjacency := graph.Adjacency(g)
	pages := make([]ManifestPage, 0, len(fileMap))
	for sourcePath, meta := range fileMap {
		if meta == nil {
			continue
		}

		rendered := meta.Render == nil || *meta.Render
		identity := strings.TrimSuffix(sourcePath, ".md")
		if !rendered {
			identity = sourcePath
		}

		title := meta.Title
		if title == "" {
			title = filepath.Base(sourcePath)
		}

		url := ""
		outputPath := pageOutputs[identity]
		if !rendered {
			outputPath = sourcePath
		}
		if outputPath != "" {
			url = siteCfg.PublicPathForOutput(outputPath)
		}

		neighbors := adjacency[identity]
		pages = append(pages, ManifestPage{
			Identity:         identity,
			SourcePath:       sourcePath,
			Rendered:         rendered,
			URL:              url,
			Title:            title,
			Date:             meta.Date,
			Tags:             sortedStrings(meta.Tags),
			Categories:       sortedStrings(meta.Categories),
			OutboundLinks:    sortedStrings(neighbors.Outbound),
			InboundLinkCount: len(backlinks[identity]),
			Links:            sortedManifestLinks(links[sourcePath]),
			AssetReferences:  sortedUniqueStrings(assetReferences[sourcePath]),
			ContentHash:      fmt.Sprintf("%x", sha256.Sum256(meta.Rest)),
			Frontmatter:      effectiveFrontmatter(meta),
		})
	}

	sort.Slice(pages, func(i, j int) bool {
		if pages[i].Identity != pages[j].Identity {
			return pages[i].Identity < pages[j].Identity
		}
		return pages[i].SourcePath < pages[j].SourcePath
	})

	return Manifest{Version: ManifestVersion, Pages: pages}
}

// WriteManifest writes the manifest to the site's output directory.
func WriteManifest(outputDir string, manifest Manifest) error {
	if err := jsonutil.WriteJSON(filepath.Join(outputDir, ManifestFileName), manifest); err != nil {
		return fmt.Errorf("failed to write %s: %w", ManifestFileName, err)
	}
	return nil
}

// ReadManifest reads a manifest and rejects unsupported schema versions.
func ReadManifest(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, err
	}
	return ParseManifest(data)
}

// ParseManifest parses serialized manifest data and rejects unsupported schema
// versions.
func ParseManifest(data []byte) (Manifest, error) {
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return manifest, fmt.Errorf("failed to parse site manifest: %w", err)
	}
	if manifest.Version != 1 && manifest.Version != ManifestVersion {
		return manifest, fmt.Errorf("unsupported site manifest version %d", manifest.Version)
	}
	if manifest.Pages == nil {
		manifest.Pages = []ManifestPage{}
	}
	for i := range manifest.Pages {
		page := &manifest.Pages[i]
		if page.Tags == nil {
			page.Tags = []string{}
		}
		if page.Categories == nil {
			page.Categories = []string{}
		}
		if page.OutboundLinks == nil {
			page.OutboundLinks = []string{}
		}
		if page.Links == nil {
			page.Links = []ManifestLink{}
		}
		if page.AssetReferences == nil {
			page.AssetReferences = []string{}
		}
	}
	return manifest, nil
}

func effectiveFrontmatter(meta *content.FileMeta) map[string]string {
	publish := "true"
	if meta.Publish != nil && !*meta.Publish {
		publish = "false"
	}
	return map[string]string{
		"author": meta.Author, "description": meta.Description,
		"image": meta.Image, "layout": meta.Layout, "slug": meta.Slug,
		"publish": publish, "video_script": meta.VideoScript,
		"animation_cues": meta.AnimationCues, "soundtrack_theme": meta.SoundtrackTheme,
		"compliance_modal": meta.ComplianceModal,
	}
}

func sortedStrings(values []string) []string {
	sorted := append([]string{}, values...)
	sort.Strings(sorted)
	return sorted
}

func sortedUniqueStrings(values []string) []string {
	sorted := sortedStrings(values)
	if len(sorted) < 2 {
		return sorted
	}
	unique := sorted[:1]
	for _, value := range sorted[1:] {
		if value != unique[len(unique)-1] {
			unique = append(unique, value)
		}
	}
	return unique
}

func sortedManifestLinks(values []ManifestLink) []ManifestLink {
	sorted := append([]ManifestLink{}, values...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Line != sorted[j].Line {
			return sorted[i].Line < sorted[j].Line
		}
		if sorted[i].Destination != sorted[j].Destination {
			return sorted[i].Destination < sorted[j].Destination
		}
		if sorted[i].Target != sorted[j].Target {
			return sorted[i].Target < sorted[j].Target
		}
		return sorted[i].GraphTarget < sorted[j].GraphTarget
	})
	return sorted
}
