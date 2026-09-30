package search

import (
	"reflect"
	"testing"
)

func TestExtractWikiLinkTargets(t *testing.T) {
	source := []byte(`---
title: Source
---

Read [[Secret Note|the visible alias]] and [[docs/Another Note#Setup]].
Repeated [[Secret Note]].

` + "```md\n[[Code Example]]\n```\n\n" + `[Ordinary link](other.md)`)

	got := ExtractWikiLinkTargets(source)
	want := []string{"Secret Note", "docs/Another Note"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ExtractWikiLinkTargets() = %q, want %q", got, want)
	}
}
