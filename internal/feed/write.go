// Package feed writes an RSS feed for dated rendered pages.
package feed

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/tbuddy/la-famille/internal/config"
)

// Item is one rendered page to publish in the feed.
type Item struct {
	Title       string
	URL         string
	Date        string
	Description string
}

// LocalURL converts a generated output path to the site's root-relative URL.
func LocalURL(outputPath string) string {
	p := filepath.ToSlash(outputPath)
	if p == "index.html" {
		return "/"
	}
	if strings.HasSuffix(p, "/index.html") {
		return "/" + strings.TrimSuffix(p, "index.html")
	}
	return "/" + strings.TrimSuffix(p, ".html") + "/"
}

type rss struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr"`
	Channel channel  `xml:"channel"`
}

type channel struct {
	Title string  `xml:"title"`
	Link  string  `xml:"link"`
	Items []entry `xml:"item"`
}

type entry struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	GUID        string `xml:"guid"`
	PubDate     string `xml:"pubDate"`
	Description string `xml:"description,omitempty"`
}

// Write writes feed.xml when items are present. An existing feed is removed when
// a build has no dated rendered pages, preventing stale generated output.
func Write(cfg config.Config, items []Item) error {
	feedPath := filepath.Join(cfg.OutputDir, "feed.xml")
	if len(items) == 0 {
		if err := os.Remove(feedPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove empty RSS feed: %w", err)
		}
		return nil
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Date != items[j].Date {
			return items[i].Date > items[j].Date
		}
		return items[i].URL < items[j].URL
	})

	entries := make([]entry, 0, len(items))
	for _, item := range items {
		date, err := time.Parse(time.DateOnly, item.Date)
		if err != nil {
			return fmt.Errorf("parse RSS date %q: %w", item.Date, err)
		}
		link := item.URL
		entries = append(entries, entry{
			Title:       item.Title,
			Link:        link,
			GUID:        link,
			PubDate:     date.UTC().Format(time.RFC1123Z),
			Description: item.Description,
		})
	}

	channelTitle := cfg.SiteName
	if strings.TrimSpace(channelTitle) == "" {
		channelTitle = "Site feed"
	}
	link := cfg.URLForOutputPath("index.html")
	if link == "" {
		link = "/"
	}
	doc := rss{Version: "2.0", Channel: channel{Title: channelTitle, Link: link, Items: entries}}
	contents, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal RSS feed: %w", err)
	}
	contents = append([]byte(xml.Header), append(contents, '\n')...)
	if err := os.WriteFile(feedPath, contents, 0600); err != nil {
		return fmt.Errorf("write RSS feed: %w", err)
	}
	return nil
}
