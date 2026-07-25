package watcher

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/generator"
)

// TestWatchCoalescesChangesDuringABuild pins the property that debouncing alone
// cannot provide. Timer.Stop is a no-op once the timer has fired, so each event
// arriving after that point used to schedule its own rebuild: three edits made
// while a build was running produced three further builds, each republishing
// the whole site, when one pass would have picked up all three.
//
// The test gates on build entry rather than on sleeps, so it fails for the
// right reason on a loaded machine instead of flaking.
func TestWatchCoalescesChangesDuringABuild(t *testing.T) {
	cfg := testConfig(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var mu sync.Mutex
	builds := 0

	started := make(chan struct{}, 8)
	release := make(chan struct{})
	finished := make(chan struct{}, 8)

	debounce := 20 * time.Millisecond
	done := make(chan error, 1)

	go func() {
		done <- watch(ctx, cfg, nil, func(config.Config) (generator.BuildResult, error) {
			mu.Lock()
			builds++
			n := builds
			mu.Unlock()

			started <- struct{}{}
			// Only the first build blocks; it is the window during which the
			// other edits arrive.
			if n == 1 {
				<-release
			}
			finished <- struct{}{}
			return generator.BuildResult{}, nil
		}, debounce)
	}()

	waitFor := func(c chan struct{}, what string) {
		t.Helper()
		select {
		case <-c:
		case <-time.After(10 * time.Second):
			t.Fatalf("timed out waiting for %s", what)
		}
	}

	write := func(name string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(cfg.ContentDir, name), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	// Let the watcher register its directories before touching anything.
	time.Sleep(2 * debounce)

	write("first.md")
	waitFor(started, "the first build to start")

	// Three separate edits, each given long enough for its debounce timer to
	// fire, all while the first build is still blocked.
	for _, name := range []string{"second.md", "third.md", "fourth.md"} {
		write(name)
		time.Sleep(3 * debounce)
	}

	close(release)
	waitFor(finished, "the first build to finish")

	// Exactly one coalesced pass should follow, covering all three edits.
	waitFor(started, "the coalesced rebuild to start")
	waitFor(finished, "the coalesced rebuild to finish")

	// Give any surplus rebuild the chance to appear before asserting.
	time.Sleep(5 * debounce)

	mu.Lock()
	got := builds
	mu.Unlock()

	if got != 2 {
		t.Errorf("three edits during a build produced %d builds in total, want 2 (the original plus one coalesced pass)", got)
	}

	cancel()
	<-done
}

// TestWatchReturnsOnStartupErrorWithoutHanging guards the interaction between
// the builder goroutine and watch()'s early returns. The builder is waited on
// in a defer; if it only stopped when the CALLER's context was cancelled, every
// path that returns without cancelling — a missing content directory, a failed
// watcher.Add, a closed event channel — would block there forever, holding all
// registered watch descriptors and never reporting the error.
func TestWatchReturnsOnStartupErrorWithoutHanging(t *testing.T) {
	cfg := testConfig(t)
	// A content directory that does not exist makes registration fail, which is
	// one of the non-context return paths.
	cfg.ContentDir = filepath.Join(t.TempDir(), "definitely-not-here")

	// The caller's context stays ALIVE for the whole test: that is the point.
	// If watch() needs it cancelled in order to return, this fails.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- watch(ctx, cfg, nil, func(config.Config) (generator.BuildResult, error) {
			return generator.BuildResult{}, nil
		}, 20*time.Millisecond)
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Error("expected watch to report the registration failure")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("watch did not return on a startup error; it is blocked waiting for the builder goroutine")
	}
}
