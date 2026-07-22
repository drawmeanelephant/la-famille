package generator

import (
	"sort"
	"strings"

	"github.com/tbuddy/la-famille/internal/content"
	"github.com/tbuddy/la-famille/internal/graph"
)

// TagCount records the frequency of a tag across content files.
type TagCount struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}

// ContentHealth encapsulates observability and health metrics calculated from build data.
type ContentHealth struct {
	TotalWordCount      int        `json:"total_word_count"`
	AvgWordsPerPage     float64    `json:"avg_words_per_page"`
	TopTags             []TagCount `json:"top_tags"`
	OrphanedPages       []string   `json:"orphaned_pages"`
	NodeCount           int        `json:"node_count"`
	EdgeCount           int        `json:"edge_count"`
	MissingDescriptions []string   `json:"missing_descriptions"`
	MissingDates        []string   `json:"missing_dates"`
}

// ComputeContentHealth calculates content health metrics from metadata, graph data, and backlinks.
func ComputeContentHealth(fileMap map[string]*content.FileMeta, g graph.Graph, backlinks map[string][]string) ContentHealth {
	var health ContentHealth

	health.NodeCount = len(g.Nodes)
	health.EdgeCount = len(g.Edges)

	tagCounts := make(map[string]int)
	renderedCount := 0

	for relPath, meta := range fileMap {
		if meta == nil {
			continue
		}
		shouldRender := true
		if meta.Render != nil && !*meta.Render {
			shouldRender = false
		}

		id := strings.TrimSuffix(relPath, ".md")
		if !shouldRender {
			id = relPath
		}

		if shouldRender {
			renderedCount++
			words := len(strings.Fields(string(meta.Rest)))
			health.TotalWordCount += words

			if strings.TrimSpace(meta.Description) == "" {
				health.MissingDescriptions = append(health.MissingDescriptions, id)
			}
			if strings.TrimSpace(meta.Date) == "" {
				health.MissingDates = append(health.MissingDates, id)
			}

			if len(backlinks[id]) == 0 {
				health.OrphanedPages = append(health.OrphanedPages, id)
			}
		}

		for _, tag := range meta.Tags {
			cleanTag := strings.TrimSpace(tag)
			if cleanTag != "" {
				tagCounts[cleanTag]++
			}
		}
	}

	if renderedCount > 0 {
		health.AvgWordsPerPage = float64(health.TotalWordCount) / float64(renderedCount)
	}

	topTags := make([]TagCount, 0, len(tagCounts))
	for tag, count := range tagCounts {
		topTags = append(topTags, TagCount{Tag: tag, Count: count})
	}
	sort.Slice(topTags, func(i, j int) bool {
		if topTags[i].Count != topTags[j].Count {
			return topTags[i].Count > topTags[j].Count
		}
		return topTags[i].Tag < topTags[j].Tag
	})
	health.TopTags = topTags

	sort.Strings(health.OrphanedPages)
	sort.Strings(health.MissingDescriptions)
	sort.Strings(health.MissingDates)

	return health
}
