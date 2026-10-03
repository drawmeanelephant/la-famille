package pack

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestHTTPSWatchInterruptedUpdateRecoveryAndRequests(t *testing.T) {
	dir, _, before, target, feed := pullFixture(t)
	initial := Feed{SchemaVersion: 1, Full: FeedPack{Path: "base.tar", SHA256: archiveHash(before)}, Deltas: []FeedDelta{}}
	writeInput(t, filepath.Join(dir, initial.Full.Path), before)
	writeTestFeed(t, dir, initial)
	var mu sync.Mutex
	var requests []string
	stage := 0
	options := httpsFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requests = append(requests, r.URL.Path)
		interrupt := stage == 1 && strings.HasSuffix(r.URL.Path, "delta.tar")
		mu.Unlock()
		if interrupt {
			w.Header().Set("Content-Length", "100000")
			_, _ = io.WriteString(w, "interrupted")
			return
		}
		http.ServeFile(w, r, filepath.Join(dir, filepath.Base(r.URL.Path)))
	}))
	state := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	polls := 0
	var originalPointer []byte
	err := Watch(ctx, WatchOptions{
		Source: "https://example.com/feed/pack-feed.json", StateDir: state, Interval: time.Minute, Remote: options,
		wait: func(context.Context, time.Duration) error {
			mu.Lock()
			defer mu.Unlock()
			switch polls {
			case 1:
				stage = 1
				originalPointer = readTestFile(t, filepath.Join(state, CurrentName))
				writeTestFeed(t, dir, feed)
			case 2:
				stage = 2
			case 4:
				cancel()
			}
			return nil
		},
	}, func(event WatchEvent) {
		polls++
		switch polls {
		case 1:
			if event.Err != nil {
				t.Fatal(event.Err)
			}
		case 2:
			if event.Err == nil || !bytes.Equal(originalPointer, readTestFile(t, filepath.Join(state, CurrentName))) ||
				!bytes.Equal(before, readTestFile(t, event.Current)) {
				t.Fatalf("interrupted update lost current: %+v", event)
			}
		case 3:
			if event.Err != nil || event.Result.Mode != "delta" || !bytes.Equal(target, readTestFile(t, event.Current)) {
				t.Fatalf("network recovery: %+v", event)
			}
		case 4:
			if !event.Unchanged || event.Err != nil {
				t.Fatalf("unchanged: %+v", event)
			}
		}
	})
	if err != nil || polls != 4 {
		t.Fatalf("watch: %v polls %d", err, polls)
	}
	mu.Lock()
	defer mu.Unlock()
	var full, delta, manifest int
	for _, path := range requests {
		switch filepath.Base(path) {
		case "base.tar":
			full++
		case "delta.tar":
			delta++
		case FeedName:
			manifest++
		case "current.tar":
			t.Fatal("selected delta failure downloaded full target")
		}
	}
	if full != 1 || delta != 2 || manifest != 4 {
		t.Fatalf("requests: %v", requests)
	}
}

func TestHTTPSWatchCancellationDuringTransfer(t *testing.T) {
	dir, _, before, _, feed := pullFixture(t)
	state := t.TempDir()
	current := SubscriberState{SchemaVersion: 1, SHA256: archiveHash(before), Path: versionPath(archiveHash(before))}
	data, _ := json.Marshal(current)
	writeInput(t, filepath.Join(state, CurrentName), data)
	writeInput(t, filepath.Join(state, current.Path), before)
	pointer := readTestFile(t, filepath.Join(state, CurrentName))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	options := httpsFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "delta.tar") {
			cancel()
			<-r.Context().Done()
			return
		}
		http.ServeFile(w, r, filepath.Join(dir, filepath.Base(r.URL.Path)))
	}))
	called := false
	if err := Watch(ctx, WatchOptions{Source: "https://example.com/feed/pack-feed.json", StateDir: state,
		Interval: time.Hour, Remote: options}, func(WatchEvent) { called = true }); err != nil || called {
		t.Fatalf("cancellation must exit cleanly: %v, report %v", err, called)
	}
	if !bytes.Equal(pointer, readTestFile(t, filepath.Join(state, CurrentName))) ||
		!bytes.Equal(before, readTestFile(t, filepath.Join(state, current.Path))) {
		t.Fatal("cancellation changed durable current")
	}
	if _, err := os.Stat(filepath.Join(state, versionPath(feed.Full.SHA256))); !os.IsNotExist(err) {
		t.Fatal("interrupted update published target")
	}
}

func TestHTTPSDNSRebindingRejectedBeforeReconnect(t *testing.T) {
	dir, _, _, _, _ := pullFixture(t)
	options := httpsFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Force a second DNS resolution instead of reusing the validated socket.
		w.Header().Set("Connection", "close")
		http.ServeFile(w, r, filepath.Join(dir, filepath.Base(r.URL.Path)))
	}))
	resolutions, dials := 0, 0
	goodDial := options.network.dial
	options.network.lookup = func(context.Context, string) ([]netip.Addr, error) {
		resolutions++
		if resolutions > 1 {
			return []netip.Addr{netip.MustParseAddr("169.254.169.254")}, nil
		}
		return []netip.Addr{netip.MustParseAddr("8.8.8.8")}, nil
	}
	options.network.dial = func(ctx context.Context, network, address string) (net.Conn, error) {
		dials++
		return goodDial(ctx, network, address)
	}
	out := filepath.Join(t.TempDir(), "out")
	if _, err := PullContext(context.Background(), "https://example.com/feed/pack-feed.json", "", out, options); err == nil {
		t.Fatal("DNS rebinding reached target")
	}
	if resolutions != 2 || dials != 1 {
		t.Fatalf("rebinding attempted a socket: resolutions %d dials %d", resolutions, dials)
	}
}
