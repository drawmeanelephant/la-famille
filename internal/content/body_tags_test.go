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
