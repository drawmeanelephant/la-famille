package pack

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const DefaultRetainedVersions = 3

// PublishOptions produces a fresh deployment directory. Previous is retained
// feed history, not source input; a missing history permits full-only output.
type PublishOptions struct {
	Build    BuildOptions
	Previous string
	Retain   int // Total full versions, including the advertised current version.
}

// Publish builds only the existing content allowlist. Artifacts are addressed
// by exact archive hashes; the manifest is published last. The destination must
// be new and is owned by this invocation until successful completion.
func Publish(options PublishOptions, destination string) (Feed, error) {
	retain := options.Retain
	if retain == 0 {
		retain = DefaultRetainedVersions
	}
	if retain < 1 || retain > 8 {
		return Feed{}, fmt.Errorf("retained versions must be between 1 and 8")
	}
	if err := os.Mkdir(destination, 0700); err != nil {
		return Feed{}, fmt.Errorf("create feed (destination must not exist): %w", err)
	}
	complete := false
	defer func() {
		if !complete {
			_ = os.RemoveAll(destination)
		}
	}()
	for _, dir := range []string{"packs", "deltas"} {
		if err := os.Mkdir(filepath.Join(destination, dir), 0700); err != nil {
			return Feed{}, err
		}
	}
	working := filepath.Join(destination, ".building.tar")
	if _, err := Build(options.Build, working); err != nil {
		return Feed{}, err
	}
	target, err := loadPack(working)
	if err != nil {
		return Feed{}, err
	}
	hash := target.hash
	target.close()
	feed := Feed{SchemaVersion: FeedVersion, Full: FeedPack{Path: versionPath(hash), SHA256: hash}, Deltas: []FeedDelta{}}
	targetPath := filepath.Join(destination, filepath.FromSlash(feed.Full.Path))
	if err := os.Rename(working, targetPath); err != nil {
		return Feed{}, err
	}
	if options.Previous != "" && retain > 1 {
		if err := retainHistory(options.Previous, destination, &feed, retain); err != nil {
			return Feed{}, err
		}
	}
	data, err := json.MarshalIndent(feed, "", "  ")
	if err != nil {
		return Feed{}, err
	}
	data = append(data, '\n')
	if _, err := parseFeed(data); err != nil {
		return Feed{}, err
	}
	if err := writeAtomicJSON(destination, FeedName, data); err != nil {
		return Feed{}, err
	}
	complete = true
	return feed, nil
}

func versionPath(hash string) string { return "packs/" + hash + ".tar" }

func retainHistory(previous, destination string, feed *Feed, retain int) error {
	root, err := openFeedRoot(previous)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("previous feed: %w", err)
	}
	defer root.Close()
	old, err := loadFeed(root)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("previous feed: %w", err)
	}
	candidates := []FeedPack{old.Full}
	for _, delta := range old.Deltas {
		candidates = append(candidates, FeedPack{Path: versionPath(delta.BaseSHA256), SHA256: delta.BaseSHA256})
	}
	seen := map[string]bool{feed.Full.SHA256: true}
	for _, candidate := range candidates {
		if seen[candidate.SHA256] {
			continue
		}
		seen[candidate.SHA256] = true
		input, err := openFeedSource(root, candidate.Path, MaxArchiveSize)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return fmt.Errorf("previous version: %w", err)
		}
		base, err := captureOpenSnapshot(input, verifyPackSnapshot)
		_ = input.Close()
		if err != nil {
			return fmt.Errorf("previous version: %w", err)
		}
		if base.hash != candidate.SHA256 {
			base.close()
			return fmt.Errorf("previous version SHA256 mismatch")
		}
		basePath := filepath.Join(destination, filepath.FromSlash(versionPath(base.hash)))
		err = publishArchive(basePath, func(w io.Writer) error {
			info, err := base.file.Stat()
			if err != nil {
				return err
			}
			_, err = io.Copy(w, io.NewSectionReader(base.file, 0, info.Size()))
			return err
		}, func(r io.Reader) error { return verifyTargetArchive(r, candidate.SHA256) })
		base.close()
		if err != nil {
			return err
		}
		deltaPath := "deltas/" + candidate.SHA256 + "-" + feed.Full.SHA256 + ".tar"
		if _, err := Diff(basePath, filepath.Join(destination, filepath.FromSlash(feed.Full.Path)),
			filepath.Join(destination, filepath.FromSlash(deltaPath))); err != nil {
			return err
		}
		// Check canonical reconstruction before advertising a delta, even when
		// history contains alternate legal headers or unknown members.
		check := filepath.Join(destination, ".roundtrip.tar")
		if err := checkPublishedDelta(basePath, filepath.Join(destination, filepath.FromSlash(deltaPath)), check, feed.Full.SHA256); err != nil {
			return err
		}
		_ = os.Remove(check)
		feed.Deltas = append(feed.Deltas, FeedDelta{Path: deltaPath, BaseSHA256: candidate.SHA256})
		if len(feed.Deltas) == retain-1 {
			break
		}
	}
	return nil
}

func checkPublishedDelta(basePath, deltaPath, check, targetHash string) error {
	base, err := loadPack(basePath)
	if err != nil {
		return err
	}
	defer base.close()
	delta, metadata, err := loadDelta(deltaPath)
	if err != nil {
		return err
	}
	defer delta.close()
	_, err = applySnapshots(base, delta, metadata, check, targetHash)
	return err
}
