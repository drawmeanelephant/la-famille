package pack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const (
	FeedVersion   = 1
	FeedName      = "pack-feed.json"
	MaxFeedSize   = 64 << 10
	MaxFeedDeltas = 128
)

type FeedPack struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type FeedDelta struct {
	Path       string `json:"path"`
	BaseSHA256 string `json:"base_sha256"`
}

// Feed lists deltas to one current target. Their exact base identities must be
// unique; the full archive digest authenticates neither publisher nor content.
type Feed struct {
	SchemaVersion int         `json:"schema_version"`
	Full          FeedPack    `json:"full"`
	Deltas        []FeedDelta `json:"deltas"`
}

func parseFeed(data []byte) (Feed, error) {
	var feed Feed
	if len(data) > MaxFeedSize || !utf8.Valid(data) {
		return feed, fmt.Errorf("feed must be UTF-8 and at most %d bytes", MaxFeedSize)
	}
	tokens := json.NewDecoder(bytes.NewReader(data))
	if err := checkJSON(tokens, 0); err != nil {
		return feed, fmt.Errorf("malformed feed: %w", err)
	}
	if _, err := tokens.Token(); err != io.EOF {
		return feed, fmt.Errorf("malformed feed: trailing JSON data")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&feed); err != nil {
		return feed, fmt.Errorf("malformed feed: %w", err)
	}
	var fields map[string]json.RawMessage
	_ = json.Unmarshal(data, &fields)
	if err := checkFields(fields, []string{"schema_version", "full", "deltas"}); err != nil {
		return feed, fmt.Errorf("feed: %w", err)
	}
	var full map[string]json.RawMessage
	_ = json.Unmarshal(fields["full"], &full)
	if err := checkFields(full, []string{"path", "sha256"}); err != nil {
		return feed, fmt.Errorf("feed full pack: %w", err)
	}
	var deltas []map[string]json.RawMessage
	_ = json.Unmarshal(fields["deltas"], &deltas)
	for _, delta := range deltas {
		if err := checkFields(delta, []string{"path", "base_sha256"}); err != nil {
			return feed, fmt.Errorf("feed delta: %w", err)
		}
	}
	if feed.SchemaVersion != FeedVersion {
		return feed, fmt.Errorf("unsupported feed schema version %d", feed.SchemaVersion)
	}
	if !validHash(feed.Full.SHA256) {
		return feed, fmt.Errorf("malformed full archive SHA256")
	}
	if feed.Deltas == nil || len(feed.Deltas) > MaxFeedDeltas {
		return feed, fmt.Errorf("feed deltas must be an array of at most %d entries", MaxFeedDeltas)
	}
	seen := map[string]bool{FeedName: true}
	checkPath := func(name string) error {
		if err := validatePath(name); err != nil {
			return fmt.Errorf("feed source: %w", err)
		}
		if seen[name] {
			return fmt.Errorf("duplicate or reserved feed source path %q", name)
		}
		seen[name] = true
		return nil
	}
	if err := checkPath(feed.Full.Path); err != nil {
		return feed, err
	}
	bases := make(map[string]bool, len(feed.Deltas))
	for _, delta := range feed.Deltas {
		if err := checkPath(delta.Path); err != nil {
			return feed, err
		}
		if !validHash(delta.BaseSHA256) || bases[delta.BaseSHA256] {
			return feed, fmt.Errorf("malformed or duplicate delta base SHA256 %q", delta.BaseSHA256)
		}
		bases[delta.BaseSHA256] = true
	}
	return feed, nil
}

func openFeedRoot(directory string) (*os.Root, error) {
	if directory == "" {
		return nil, fmt.Errorf("feed directory must not be empty")
	}
	directory = filepath.Clean(directory)
	info, err := os.Lstat(directory)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("feed directory must not be a symlink or non-directory")
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return nil, err
	}
	pinned, err := root.Stat(".")
	after, afterErr := os.Lstat(directory)
	if err != nil || afterErr != nil || !after.IsDir() ||
		!os.SameFile(info, pinned) || !os.SameFile(info, after) {
		_ = root.Close()
		return nil, fmt.Errorf("feed directory changed while opening")
	}
	return root, nil
}

// Walk through pinned directory handles, not re-resolved parent paths.
// Lstat/identity checks reject links and swapped directories; os.Root confines
// traversal even if a source tree changes concurrently.
func feedParent(root *os.Root, name string) (*os.Root, string, func(), error) {
	if err := validatePath(name); err != nil {
		return nil, "", nil, err
	}
	parts := strings.Split(name, "/")
	var opened []*os.Root
	closeParents := func() {
		for _, parent := range opened {
			_ = parent.Close()
		}
	}
	for _, part := range parts[:len(parts)-1] {
		info, err := root.Lstat(part)
		if err != nil {
			closeParents()
			return nil, "", nil, err
		}
		if !info.IsDir() {
			closeParents()
			return nil, "", nil, fmt.Errorf("symlink or non-directory feed source parent %q", name)
		}
		next, err := root.OpenRoot(part)
		if err != nil {
			closeParents()
			return nil, "", nil, err
		}
		opened = append(opened, next)
		pinned, err := next.Stat(".")
		after, afterErr := root.Lstat(part)
		if err != nil || afterErr != nil || !after.IsDir() ||
			!os.SameFile(info, pinned) || !os.SameFile(info, after) {
			closeParents()
			return nil, "", nil, fmt.Errorf("feed source parent changed while opening %q", name)
		}
		root = next
	}
	return root, parts[len(parts)-1], closeParents, nil
}

// Unselected archives can be absent, but any existing declared source must be
// regular and link-free. This inspects metadata only, never target payloads.
func checkFeedSource(root *os.Root, name string) error {
	parent, leaf, closeParents, err := feedParent(root, name)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer closeParents()
	info, err := parent.Lstat(leaf)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("symlink or non-regular feed source %q", name)
	}
	return nil
}

func openFeedSource(root *os.Root, name string, maxSize int64) (*os.File, error) {
	parent, leaf, closeParents, err := feedParent(root, name)
	if err != nil {
		return nil, err
	}
	defer closeParents()
	info, err := parent.Lstat(leaf)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > maxSize {
		return nil, fmt.Errorf("feed source %q must be regular, not a symlink, and at most %d bytes", name, maxSize)
	}
	f, err := openFeedLeaf(parent, leaf)
	if err != nil {
		return nil, err
	}
	pinned, err := f.Stat()
	after, afterErr := parent.Lstat(leaf)
	if err != nil || afterErr != nil || !after.Mode().IsRegular() || !pinned.Mode().IsRegular() ||
		pinned.Size() > maxSize || !os.SameFile(info, pinned) || !os.SameFile(info, after) {
		_ = f.Close()
		return nil, fmt.Errorf("feed source changed while opening %q", name)
	}
	return f, nil
}

func loadFeed(root *os.Root) (Feed, error) {
	f, err := openFeedSource(root, FeedName, MaxFeedSize)
	if err != nil {
		return Feed{}, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, MaxFeedSize+1))
	if err != nil {
		return Feed{}, err
	}
	feed, err := parseFeed(data)
	if err != nil {
		return feed, err
	}
	for _, name := range append([]string{feed.Full.Path}, feedDeltaPaths(feed)...) {
		if err := checkFeedSource(root, name); err != nil {
			return Feed{}, fmt.Errorf("feed source %q: %w", name, err)
		}
	}
	return feed, nil
}

func feedDeltaPaths(feed Feed) []string {
	paths := make([]string, 0, len(feed.Deltas))
	for _, delta := range feed.Deltas {
		paths = append(paths, delta.Path)
	}
	return paths
}
