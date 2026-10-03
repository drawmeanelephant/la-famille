package pack

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tbuddy/la-famille/internal/sitedata"
)

func TestWatchDeterministicUpdateNoChangeAndDurableRestart(t *testing.T) {
	dir, base, before, target, feed := pullFixture(t)
	stateDir := filepath.Join(t.TempDir(), "subscriber")
	initial := feed
	initial.Full = FeedPack{Path: "base.tar", SHA256: archiveHash(before)}
	initial.Deltas = []FeedDelta{}
	writeInput(t, filepath.Join(dir, "base.tar"), before)
	writeTestFeed(t, dir, initial)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var events []WatchEvent
	options := WatchOptions{Source: dir, StateDir: stateDir, Interval: time.Hour}
	waits := 0
	options.wait = func(_ context.Context, interval time.Duration) error {
		if interval != time.Hour {
			t.Fatal("interval changed")
		}
		waits++
		switch waits {
		case 1:
			// If unchanged polls download again, removing the full makes them fail.
			_ = os.Remove(filepath.Join(dir, initial.Full.Path))
		case 2:
			writeTestFeed(t, dir, feed)
			_ = os.Remove(filepath.Join(dir, feed.Full.Path))
		case 3:
			cancel()
		}
		return nil
	}
	if err := Watch(ctx, options, func(event WatchEvent) { events = append(events, event) }); err != nil {
		t.Fatal(err)
	}
	if len(events) != 3 || events[0].Err != nil || !events[1].Unchanged ||
		events[2].Err != nil || events[2].Result.Mode != "delta" || events[2].Result.Changes == nil {
		t.Fatalf("events: %+v", events)
	}
	if events[2].Result.Changes.Ledger != nil || events[2].Result.Changes.LedgerWarning == "" {
		t.Fatal("unavailable page semantics not explicit")
	}
	if !bytes.Equal(readTestFile(t, base), before) ||
		!bytes.Equal(readTestFile(t, events[0].Current), before) ||
		!bytes.Equal(readTestFile(t, events[2].Current), target) {
		t.Fatal("watch changed immutable version bytes")
	}
	metadata := readTestFile(t, filepath.Join(stateDir, CurrentName))
	ctx, cancelRestart := context.WithCancel(context.Background())
	options.wait = func(context.Context, time.Duration) error { cancelRestart(); return nil }
	if err := Watch(ctx, options, func(event WatchEvent) {
		if event.Err != nil || !event.Unchanged || event.Current != events[2].Current {
			t.Fatalf("restart: %+v", event)
		}
	}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(metadata, readTestFile(t, filepath.Join(stateDir, CurrentName))) {
		t.Fatal("no-change rewrote pointer")
	}
}

func TestWatchFailedSelectedDeltaPreservesCurrentAndRecovers(t *testing.T) {
	for _, name := range []string{"missing", "corrupt", "truncated", "wrong base", "publication race"} {
		t.Run(name, func(t *testing.T) {
			dir, _, before, target, feed := pullFixture(t)
			validDelta := readTestFile(t, filepath.Join(dir, "delta.tar"))
			initial := Feed{SchemaVersion: 1, Full: FeedPack{Path: "base.tar", SHA256: archiveHash(before)}, Deltas: []FeedDelta{}}
			writeInput(t, filepath.Join(dir, "base.tar"), before)
			writeTestFeed(t, dir, initial)
			stateDir := filepath.Join(t.TempDir(), "state")
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			polls := 0
			var oldPointer []byte
			options := WatchOptions{Source: dir, StateDir: stateDir, Interval: time.Second}
			options.wait = func(context.Context, time.Duration) error {
				switch polls {
				case 1:
					oldPointer = readTestFile(t, filepath.Join(stateDir, CurrentName))
					writeTestFeed(t, dir, feed)
					switch name {
					case "missing":
						_ = os.Remove(filepath.Join(dir, "delta.tar"))
					case "corrupt":
						writeInput(t, filepath.Join(dir, "delta.tar"), []byte("bad"))
					case "truncated":
						writeInput(t, filepath.Join(dir, "delta.tar"), validDelta[:len(validDelta)-1])
					case "wrong base":
						entries := archiveEntries(t, validDelta)
						var d DeltaMetadata
						_ = json.Unmarshal(entries[0].data, &d)
						d.BaseSHA256 = strings.Repeat("0", 64)
						entries[0].data, _ = json.Marshal(d)
						writeInput(t, filepath.Join(dir, "delta.tar"), rawArchive(t, entries...))
					case "publication race":
						racing := feed
						racing.Full.SHA256 = strings.Repeat("1", 64)
						writeTestFeed(t, dir, racing)
					}
				case 2:
					writeInput(t, filepath.Join(dir, "delta.tar"), validDelta)
					writeTestFeed(t, dir, feed)
				case 3:
					cancel()
				}
				return nil
			}
			if err := Watch(ctx, options, func(event WatchEvent) {
				polls++
				switch polls {
				case 1:
					if event.Err != nil {
						t.Fatal(event.Err)
					}
				case 2:
					if event.Err == nil {
						t.Fatal("broken selected delta succeeded")
					}
					if !bytes.Equal(oldPointer, readTestFile(t, filepath.Join(stateDir, CurrentName))) ||
						!bytes.Equal(before, readTestFile(t, filepath.Join(stateDir, versionPath(initial.Full.SHA256)))) {
						t.Fatal("failed update changed last current pack")
					}
					if _, err := VerifyFile(event.Current); err != nil {
						t.Fatal(err)
					}
				case 3:
					if event.Err != nil || event.Result.Mode != "delta" ||
						!bytes.Equal(target, readTestFile(t, event.Current)) {
						t.Fatalf("recovery: %+v", event)
					}
				}
			}); err != nil || polls != 3 {
				t.Fatalf("watch: %v polls %d", err, polls)
			}
		})
	}
}

func TestWatchLedgerNamesAndOrphanRecovery(t *testing.T) {
	build := buildInputs(t)
	site := sitedata.Manifest{Version: 2, OutputCaptured: true, Pages: []sitedata.ManifestPage{
		{Identity: "birds", Title: "Birds"}, {Identity: "removed", Title: "Removed"},
	}}
	data, _ := json.Marshal(site)
	writeInput(t, filepath.Join(build.OutputDir, sitedata.ManifestFileName), data)
	first := filepath.Join(t.TempDir(), "first")
	a, err := Publish(PublishOptions{Build: build}, first)
	if err != nil {
		t.Fatal(err)
	}
	site.Pages[0].Title = "Updated Birds"
	site.Pages[1] = sitedata.ManifestPage{Identity: "added", Title: "Added"}
	data, _ = json.Marshal(site)
	writeInput(t, filepath.Join(build.OutputDir, sitedata.ManifestFileName), data)
	second := filepath.Join(t.TempDir(), "second")
	b, err := Publish(PublishOptions{Build: build, Previous: first}, second)
	if err != nil {
		t.Fatal(err)
	}
	state := t.TempDir()
	writeInput(t, filepath.Join(state, a.Full.Path), readTestFile(t, filepath.Join(first, a.Full.Path)))
	metadata, _ := json.Marshal(SubscriberState{SchemaVersion: 1, SHA256: a.Full.SHA256, Path: a.Full.Path})
	writeInput(t, filepath.Join(state, CurrentName), metadata)
	// Simulate a process stopping after publication, before pointer advancement.
	writeInput(t, filepath.Join(state, b.Full.Path), readTestFile(t, filepath.Join(second, b.Full.Path)))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	options := WatchOptions{Source: second, StateDir: state, Interval: time.Second,
		wait: func(context.Context, time.Duration) error { cancel(); return nil }}
	if err := Watch(ctx, options, func(event WatchEvent) {
		if event.Err != nil || event.Result.Mode != "recovered" || event.Result.Changes == nil ||
			event.Result.Changes.Ledger == nil || !strings.Contains(event.Result.Changes.Summary(), "birds") {
			t.Fatalf("Ledger orphan recovery: %+v", event)
		}
		for _, change := range []string{"+ added", "- removed", "~ birds"} {
			if !strings.Contains(event.Result.Changes.Summary(), change) {
				t.Fatalf("missing named page change %q", change)
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
}

func TestSubscriberStateStrictFieldsAndCapturedIdentity(t *testing.T) {
	stateDir := t.TempDir()
	archive := writeTestPack(t, filepath.Join(stateDir, "fixture.tar"), testEntry{"rag-content.md", []byte("fact")})
	hash := archiveHash(archive)
	writeInput(t, filepath.Join(stateDir, versionPath(hash)), archive)
	state := SubscriberState{SchemaVersion: 1, SHA256: hash, Path: versionPath(hash)}
	root, err := openFeedRoot(stateDir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	data, _ := json.Marshal(state)
	for _, malformed := range []string{
		strings.Replace(string(data), `"path":`, `"Path":`, 1),
		strings.Replace(string(data), `"sha256":`, `"SHA256":`, 1),
		strings.Replace(string(data), `"path":`, `"Path":"packs/other.tar","path":`, 1),
		strings.Replace(string(data), `"schema_version":1`, `"schema_version":1,"schema_version":1`, 1),
	} {
		writeInput(t, filepath.Join(stateDir, CurrentName), []byte(malformed))
		if _, err := readSubscriberState(root); err == nil {
			t.Fatal("subscriber metadata accepted aliased or duplicate fields")
		}
	}
	captured, err := captureStatePack(root, state)
	if err != nil {
		t.Fatal(err)
	}
	defer captured.close()
	// Snapshot reports and reconstruction retain the same verified bytes even
	// when the original version path is replaced after capture.
	writeInput(t, filepath.Join(stateDir, state.Path), alternateArchiveMode(archive))
	if captured.hash != hash {
		t.Fatal("snapshot identity changed")
	}
	if _, err := captureStatePack(root, state); err == nil {
		t.Fatal("legal pack with a different exact identity was accepted")
	}
}

func TestWatchRejectsBadStateAndConcurrentWriter(t *testing.T) {
	for _, name := range []string{"lock", "metadata", "corrupt current", "symlink versions"} {
		t.Run(name, func(t *testing.T) {
			dir, _, before, _, _ := pullFixture(t)
			state := t.TempDir()
			switch name {
			case "lock":
				writeInput(t, filepath.Join(state, ".watch-lock"), []byte("owned"))
			case "metadata":
				writeInput(t, filepath.Join(state, CurrentName), []byte(`{"path":"../secret"}`))
			case "corrupt current":
				hash := archiveHash(before)
				data, _ := json.Marshal(SubscriberState{SchemaVersion: 1, SHA256: hash, Path: versionPath(hash)})
				writeInput(t, filepath.Join(state, CurrentName), data)
				writeInput(t, filepath.Join(state, versionPath(hash)), []byte("bad"))
			case "symlink versions":
				if err := os.Symlink(t.TempDir(), filepath.Join(state, "packs")); err != nil {
					t.Fatal(err)
				}
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			failed := false
			err := Watch(ctx, WatchOptions{Source: dir, StateDir: state, Interval: time.Second,
				wait: func(context.Context, time.Duration) error { cancel(); return nil }},
				func(event WatchEvent) { failed = event.Err != nil })
			if err == nil && !failed {
				t.Fatal("unsafe subscriber state accepted")
			}
		})
	}
}
