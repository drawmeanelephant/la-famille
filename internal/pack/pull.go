package pack

import (
	"context"
	"fmt"
	"io"

	"github.com/tbuddy/la-famille/internal/sitedata"
)

type PullResult struct {
	Mode         string
	Fallback     bool
	TargetSHA256 string
	Manifest     Manifest
	Changes      *Comparison
}

// Pull retains the local-directory, no-overwrite contract.
func Pull(directory, baseFile, destination string) (PullResult, error) {
	return PullContext(context.Background(), directory, baseFile, destination, RemoteOptions{})
}

// PullContext permits HTTPS only with explicit opt-in. Only an absent matching
// base identity permits full fallback; errors in a selected delta are fatal.
func PullContext(ctx context.Context, source, baseFile, destination string, options RemoteOptions) (PullResult, error) {
	root, err := acquireFeed(ctx, source, options)
	if err != nil {
		return PullResult{}, fmt.Errorf("open feed: %w", err)
	}
	defer root.Close()
	feed, err := root.load()
	if err != nil {
		return PullResult{}, fmt.Errorf("feed manifest: %w", err)
	}
	return pullFeed(ctx, feed, root.open, baseFile, destination)
}

func pullFeed(ctx context.Context, feed Feed, open archiveOpener, baseFile, destination string) (PullResult, error) {
	if err := ctx.Err(); err != nil {
		return PullResult{}, err
	}
	var base *snapshot
	var err error
	if baseFile != "" {
		base, err = loadPack(baseFile)
		if err != nil {
			return PullResult{}, fmt.Errorf("base pack: %w", err)
		}
		defer base.close()
	}
	return pullVerifiedFeed(ctx, feed, open, base, destination)
}

func pullVerifiedFeed(ctx context.Context, feed Feed, open archiveOpener, base *snapshot, destination string) (PullResult, error) {
	if err := ctx.Err(); err != nil {
		return PullResult{}, err
	}
	result := PullResult{Mode: "full", TargetSHA256: feed.Full.SHA256}
	if base != nil {
		for _, entry := range feed.Deltas {
			if entry.BaseSHA256 == base.hash {
				return pullDelta(ctx, open, base, entry, destination, result)
			}
		}
		result.Fallback = true
	}
	input, err := open(feed.Full.Path, MaxArchiveSize)
	if err != nil {
		return PullResult{}, fmt.Errorf("full pack: %w", err)
	}
	defer input.Close()
	target, err := captureOpenSnapshot(input, verifyPackSnapshot)
	if err != nil {
		return PullResult{}, fmt.Errorf("full pack: %w", err)
	}
	defer target.close()
	if target.hash != feed.Full.SHA256 {
		return PullResult{}, fmt.Errorf("target archive SHA256 mismatch: expected %s, actual %s",
			feed.Full.SHA256, target.hash)
	}
	if base != nil {
		changes := compareMembers(base.manifest, target.manifest)
		changes.BaseSHA256 = base.hash
		addLedger(&changes, base, target)
		result.Changes = &changes
	}
	// Copy the whole captured archive, including its original legal headers.
	err = publishArchive(destination, func(w io.Writer) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		info, err := target.file.Stat()
		if err != nil {
			return err
		}
		_, err = io.Copy(w, io.NewSectionReader(target.file, 0, info.Size()))
		return err
	}, func(r io.Reader) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		return verifyTargetArchive(r, feed.Full.SHA256)
	})
	if err != nil {
		return PullResult{}, err
	}
	result.Manifest = target.manifest
	return result, nil
}

func pullDelta(ctx context.Context, open archiveOpener, base *snapshot, entry FeedDelta, destination string, result PullResult) (PullResult, error) {
	input, err := open(entry.Path, MaxArchiveSize)
	if err != nil {
		return PullResult{}, fmt.Errorf("selected delta %q: %w", entry.Path, err)
	}
	defer input.Close()
	delta, metadata, err := loadOpenDelta(input)
	if err != nil {
		return PullResult{}, fmt.Errorf("selected delta %q: %w", entry.Path, err)
	}
	defer delta.close()
	if err := ctx.Err(); err != nil {
		return PullResult{}, err
	}
	manifest, err := applySnapshots(base, delta, metadata, destination, result.TargetSHA256)
	if err != nil {
		return PullResult{}, fmt.Errorf("selected delta %q: %w", entry.Path, err)
	}
	changes := compareMembers(base.manifest, manifest)
	changes.BaseSHA256 = base.hash
	ledgerTarget := delta
	if _, present := memberIndex(manifest)[sitedata.ManifestFileName]; present {
		if _, replaced := delta.offsets[sitedata.ManifestFileName]; !replaced {
			ledgerTarget = base
		}
	}
	addLedger(&changes, base, ledgerTarget)
	result.Mode, result.Manifest, result.Changes = "delta", manifest, &changes
	return result, nil
}
