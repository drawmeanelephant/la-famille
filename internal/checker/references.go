package checker

import (
	"net/url"
	"path"
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"

	"github.com/tbuddy/la-famille/internal/content"
	"github.com/tbuddy/la-famille/internal/markdown"
	"github.com/tbuddy/la-famille/internal/sitedata"
	"github.com/tbuddy/la-famille/internal/transform"
)

// ExtractManifestReferences returns the internal links and local asset
// references found in the supplied source pages. Link resolution follows the
// same source/output rules as Validate; graph targets follow LinkTransformer's
// .md-only graph contract.
func ExtractManifestReferences(
	fileMap map[string]*content.FileMeta,
	graphExplorer bool,
) (map[string][]sitedata.ManifestLink, map[string][]string) {
	published := content.PublishedFiles(fileMap)
	linksByPage := make(map[string][]sitedata.ManifestLink, len(published))
	assetsByPage := make(map[string][]string, len(published))
	// A stub output the build will emit resolves an output-tree link the same
	// as a real page (#647).
	expectedOutputs, _ := buildExpectedOutputs(published, graphExplorer, collectStubTargets(published, fileMap))
	engine := markdown.NewEngine(nil)

	for relPath, meta := range published {
		if meta == nil {
			continue
		}
		if assetRef, ok := manifestAssetReference(relPath, meta.Image); ok {
			assetsByPage[relPath] = append(assetsByPage[relPath], assetRef)
		}
		if len(meta.Rest) == 0 {
			continue
		}
		doc := engine.Parser().Parse(text.NewReader(meta.Rest))
		_ = ast.Walk(doc, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
			if !entering {
				return ast.WalkContinue, nil
			}

			switch current := node.(type) {
			case *ast.Link:
				dest := string(current.Destination)
				if ref, ok := manifestLinkReference(relPath, meta, fileMap, expectedOutputs, dest, current); ok {
					linksByPage[relPath] = append(linksByPage[relPath], ref)
				}
				if isAssetLink(dest) {
					if assetRef, ok := manifestAssetReference(relPath, dest); ok {
						assetsByPage[relPath] = append(assetsByPage[relPath], assetRef)
					}
				}
			case *ast.Image:
				if assetRef, ok := manifestAssetReference(relPath, string(current.Destination)); ok {
					assetsByPage[relPath] = append(assetsByPage[relPath], assetRef)
				}
			}

			return ast.WalkContinue, nil
		})
	}

	return linksByPage, assetsByPage
}

func manifestLinkReference(
	relPath string,
	meta *content.FileMeta,
	fileMap map[string]*content.FileMeta,
	expectedOutputs map[string]bool,
	dest string,
	node ast.Node,
) (sitedata.ManifestLink, bool) {
	if wikiTarget, heading, ok := transform.ParseWikiLinkDestination(dest); ok {
		targetRelPath, targetMeta, resolved := transform.ResolveWikiTarget(relPath, wikiTarget, fileMap)
		if resolved && !content.IsPublished(targetMeta) {
			return sitedata.ManifestLink{}, false
		}
		if !resolved {
			targetRelPath = transform.UnresolvedWikiTargetPath(relPath, wikiTarget)
		}
		targetID := strings.TrimSuffix(targetRelPath, ".md")
		if resolved && targetMeta != nil && targetMeta.Render != nil && !*targetMeta.Render {
			targetID = targetRelPath
		}
		destination := wikiTarget
		if heading != "" {
			destination += "#" + heading
		}
		return sitedata.ManifestLink{
			Destination: destination,
			Target:      targetRelPath,
			GraphTarget: targetID,
			Line:        findLinkLine(meta.Content, meta.Rest, node, dest),
			Resolved:    resolved,
		}, true
	}

	u, err := url.Parse(dest)
	if err != nil || u.IsAbs() || strings.HasPrefix(dest, "//") || u.Path == "" {
		return sitedata.ManifestLink{}, false
	}

	ext := strings.ToLower(path.Ext(u.Path))
	line := findLinkLine(meta.Content, meta.Rest, node, dest)
	if ext == ".md" {
		targetRelPath := sourceTreeTarget(relPath, u.Path)
		if !filepath.IsLocal(filepath.FromSlash(targetRelPath)) {
			// A link whose target escapes the content root ships verbatim and
			// 404s (#648); record it unresolved so check --manifest reports it.
			return sitedata.ManifestLink{
				Destination: dest,
				Target:      targetRelPath,
				Line:        line,
				Resolved:    false,
			}, true
		}

		targetID := strings.TrimSuffix(targetRelPath, ".md")
		target, resolved := fileMap[targetRelPath]
		if resolved && !content.IsPublished(target) {
			return sitedata.ManifestLink{}, false
		}
		if resolved && target.Render != nil && !*target.Render {
			targetID = targetRelPath
		}
		graphTarget := ""
		if strings.HasSuffix(u.Path, ".md") {
			graphTarget = targetID
		}
		return sitedata.ManifestLink{
			Destination: dest,
			Target:      targetRelPath,
			GraphTarget: graphTarget,
			Line:        line,
			Resolved:    resolved,
		}, true
	}

	if ext != "" && ext != ".html" {
		return sitedata.ManifestLink{}, false
	}

	target := normalizeOutputCandidate(outputTreeTarget(relPath, u.Path, meta))
	return sitedata.ManifestLink{
		Destination: dest,
		Target:      target,
		Line:        line,
		Resolved:    expectedOutputFor(expectedOutputs, target),
	}, true
}

func isAssetLink(dest string) bool {
	ext := strings.ToLower(filepath.Ext(dest))
	return strings.HasPrefix(dest, "/assets/") ||
		strings.HasPrefix(dest, "assets/") ||
		rasterExts[ext] ||
		suspiciousImageExts[ext] ||
		ext == ".svg"
}

func manifestAssetReference(relPath, dest string) (string, bool) {
	u, err := url.Parse(dest)
	if err != nil || u.IsAbs() || strings.HasPrefix(dest, "//") || u.Path == "" {
		return "", false
	}

	refPath := u.Path
	var assetRel string
	switch {
	case strings.HasPrefix(refPath, "/assets/"):
		assetRel = strings.TrimPrefix(refPath, "/assets/")
	case strings.HasPrefix(refPath, "assets/"):
		assetRel = strings.TrimPrefix(refPath, "assets/")
	case strings.HasPrefix(refPath, "/"):
		assetRel = strings.TrimPrefix(refPath, "/")
	default:
		dir := filepath.Dir(relPath)
		assetRel = refPath
		if dir != "." {
			assetRel = dir + "/" + refPath
		}
		assetRel = strings.TrimPrefix(assetRel, "assets/")
	}
	return filepath.ToSlash(filepath.Clean(assetRel)), true
}
