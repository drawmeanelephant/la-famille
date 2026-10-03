package pack

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

const CurrentName = "current.json"

type SubscriberState struct {
	SchemaVersion int    `json:"schema_version"`
	SHA256        string `json:"sha256"`
	Path          string `json:"path"`
}

type WatchOptions struct {
	Source   string
	StateDir string
	Interval time.Duration
	Remote   RemoteOptions
	wait     func(context.Context, time.Duration) error
}

type WatchEvent struct {
	Unchanged bool
	Current   string
	Result    PullResult
	Err       error
}

// Watch polls immediately, then at the explicit interval. Poll failures are
// reported and retried; cancellation is a clean exit. One writer owns state.
// Stale .watch-lock files after an unclean process exit require manual removal
// only after confirming that no watch process still owns the directory.
func Watch(ctx context.Context, options WatchOptions, report func(WatchEvent)) error {
	if options.Interval <= 0 || options.StateDir == "" {
		return fmt.Errorf("watch requires a positive interval and a state directory")
	}
	if err := os.MkdirAll(options.StateDir, 0700); err != nil {
		return err
	}
	root, err := openFeedRoot(options.StateDir)
	if err != nil {
		return fmt.Errorf("subscriber state: %w", err)
	}
	defer root.Close()
	lock, err := root.OpenFile(".watch-lock", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("watch state is locked; check for another or interrupted watcher")
	}
	defer func() {
		_ = lock.Close()
		_ = root.Remove(".watch-lock")
	}()
	if err := root.Mkdir("packs", 0700); err != nil && !os.IsExist(err) {
		return err
	}
	if err := checkFeedSource(root, "packs/unused.tar"); err != nil {
		return fmt.Errorf("subscriber versions: %w", err)
	}
	wait := options.wait
	if wait == nil {
		wait = waitPoll
	}
	for {
		if ctx.Err() != nil {
			return nil
		}
		event := pollWatch(ctx, options, root)
		if ctx.Err() != nil {
			return nil
		}
		if report != nil {
			report(event)
		}
		if err := wait(ctx, options.Interval); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
	}
}

func waitPoll(ctx context.Context, interval time.Duration) error {
	timer := time.NewTimer(interval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func readSubscriberState(root *os.Root) (*SubscriberState, error) {
	f, err := openFeedSource(root, CurrentName, MaxFeedSize)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, MaxFeedSize+1))
	if err != nil {
		return nil, err
	}
	tokens := json.NewDecoder(bytes.NewReader(data))
	if err := checkJSON(tokens, 0); err != nil {
		return nil, fmt.Errorf("invalid subscriber metadata")
	}
	if _, err := tokens.Token(); err != io.EOF {
		return nil, fmt.Errorf("invalid subscriber metadata")
	}
	var fields map[string]json.RawMessage
	_ = json.Unmarshal(data, &fields)
	if err := checkFields(fields, []string{"schema_version", "sha256", "path"}); err != nil {
		return nil, fmt.Errorf("invalid subscriber metadata")
	}
	var state SubscriberState
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&state); err != nil || state.SchemaVersion != 1 ||
		!validHash(state.SHA256) || state.Path != versionPath(state.SHA256) {
		return nil, fmt.Errorf("invalid subscriber metadata")
	}
	return &state, nil
}

func verifiedStatePack(root *os.Root, state SubscriberState) error {
	f, err := openFeedSource(root, state.Path, MaxArchiveSize)
	if err != nil {
		return err
	}
	defer f.Close()
	return verifyTargetArchive(f, state.SHA256)
}

func captureStatePack(root *os.Root, state SubscriberState) (*snapshot, error) {
	f, err := openFeedSource(root, state.Path, MaxArchiveSize)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	s, err := captureOpenSnapshot(f, verifyPackSnapshot)
	if err != nil {
		return nil, err
	}
	if s.hash != state.SHA256 {
		s.close()
		return nil, fmt.Errorf("subscriber version SHA256 mismatch")
	}
	return s, nil
}

func pollWatch(ctx context.Context, options WatchOptions, root *os.Root) (event WatchEvent) {
	state, err := readSubscriberState(root)
	if err != nil {
		event.Err = fmt.Errorf("subscriber metadata: %w", err)
		return
	}
	if state != nil {
		event.Current = filepath.Join(options.StateDir, filepath.FromSlash(state.Path))
	}
	source, err := acquireFeed(ctx, options.Source, options.Remote)
	if err != nil {
		event.Err = err
		return
	}
	defer source.Close()
	feed, err := source.load()
	if err != nil {
		event.Err = err
		return
	}
	if state != nil && state.SHA256 == feed.Full.SHA256 {
		if err := verifiedStatePack(root, *state); err != nil {
			event.Err = fmt.Errorf("current subscriber pack: %w", err)
			return
		}
		event.Unchanged = true
		return
	}
	var base *snapshot
	if state != nil {
		base, err = captureStatePack(root, *state)
		if err != nil {
			event.Err = fmt.Errorf("current subscriber pack: %w", err)
			return
		}
		defer base.close()
	}
	current := SubscriberState{SchemaVersion: 1, SHA256: feed.Full.SHA256, Path: versionPath(feed.Full.SHA256)}
	destination := filepath.Join(options.StateDir, filepath.FromSlash(current.Path))
	// An interruption after pack publication but before pointer publication can
	// leave an orphan. Verify it, never overwrite it, and reuse it on restart.
	if _, err := root.Lstat(current.Path); err == nil {
		after, err := captureStatePack(root, current)
		if err != nil {
			event.Err = fmt.Errorf("existing subscriber version: %w", err)
			return
		}
		defer after.close()
		event.Result = PullResult{Mode: "recovered", TargetSHA256: current.SHA256, Manifest: after.manifest}
		if state != nil {
			changes := compareMembers(base.manifest, after.manifest)
			changes.BaseSHA256 = base.hash
			addLedger(&changes, base, after)
			event.Result.Manifest, event.Result.Changes = after.manifest, &changes
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		event.Err = err
		return
	} else {
		event.Result, err = pullVerifiedFeed(ctx, feed, source.open, base, destination)
		if err != nil {
			event.Err = err
			return
		}
	}
	if err := ctx.Err(); err != nil {
		event.Err = err
		return
	}
	if err := syncDirectory(filepath.Join(options.StateDir, "packs")); err != nil {
		event.Err = err
		return
	}
	data, err := json.MarshalIndent(current, "", "  ")
	if err == nil {
		err = writeAtomicJSON(options.StateDir, CurrentName, append(data, '\n'))
	}
	if err != nil {
		event.Err = err
		return
	}
	event.Current = destination
	return
}
