package content

import (
	"reflect"
	"testing"
)

func TestExtractBodyTagsFromMarkdownTextOnly(t *testing.T) {
	source := []byte("# Heading with #heading-only\n\n" +
		"#ceramics\n\n" +
		"A paragraph has #pottery, #red-clay and (#round-pot). " +
		"It also mentions word#attached, path/#fragment, and C#.\n\n" +
		"`#inline-code`\n\n" +
		"```go\n// #fenced-code\n```\n\n" +
		"[#linked-text](https://example.com/#linked-fragment) " +
		"https://example.com/#autolink-fragment <https://example.org/#angle-fragment>\n")

	got := extractBodyTags(newBodyTagParser(), source)
	want := []string{"ceramics", "pottery", "red-clay", "round-pot"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("extractBodyTags() = %v, want %v", got, want)
	}
}

func TestExtractBodyTagsIgnoreNumericReferences(t *testing.T) {
	source := []byte("Issues #599, (#617), #１２３ and #٢٠٢٦ are references, " +
		"not #123-456 or #123_456 tags.\n\n" +
		"Keep #3d-printing, #v2, #2026-release, #起始２ and #cafe\u0301.\n")

	got := extractBodyTags(newBodyTagParser(), source)
	want := []string{"3d-printing", "v2", "2026-release", "起始２", "cafe\u0301"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("extractBodyTags() = %v, want %v", got, want)
	}
}

func TestGatherMetadataMergesBodyTagsWithFrontmatter(t *testing.T) {
	meta := writeContentFile(t, `---
title: Tagged note
tags: [frontmatter, shared]
---
# Heading with #heading-only

This note has #shared and #body-only.
`)

	want := []string{"frontmatter", "shared", "body-only"}
	if !reflect.DeepEqual(meta.Tags, want) {
		t.Errorf("Tags = %v, want %v", meta.Tags, want)
	}
}

func TestGatherMetadataPreservesExplicitNumericTags(t *testing.T) {
	meta := writeContentFile(t, `---
title: Issue references
tags: ["2026", ceramics]
---
Merged #599 and #617 in #2026. Keep #3d-printing.
`)

	want := []string{"2026", "ceramics", "3d-printing"}
	if !reflect.DeepEqual(meta.Tags, want) {
		t.Errorf("Tags = %v, want %v", meta.Tags, want)
	}
}
