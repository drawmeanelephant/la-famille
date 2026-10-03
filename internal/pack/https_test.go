package pack

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func httpsFixture(t *testing.T, handler http.Handler) RemoteOptions {
	t.Helper()
	server := httptest.NewTLSServer(handler)
	t.Cleanup(server.Close)
	roots := x509.NewCertPool()
	roots.AddCert(server.Certificate())
	dialer := &net.Dialer{}
	return RemoteOptions{AllowHTTPS: true, network: &remoteNetwork{
		lookup: func(context.Context, string) ([]netip.Addr, error) {
			return []netip.Addr{netip.MustParseAddr("8.8.8.8")}, nil
		},
		dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			if address != "8.8.8.8:443" {
				t.Errorf("socket not pinned: %s", address)
			}
			return dialer.DialContext(ctx, network, server.Listener.Addr().String())
		},
		tls: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12},
	}}
}

func TestHTTPSPullColdDeltaAndFallback(t *testing.T) {
	dir, base, before, target, feed := pullFixture(t)
	var mu sync.Mutex
	var paths []string
	var transfers []Transfer
	options := httpsFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		paths = append(paths, r.URL.Path)
		mu.Unlock()
		if r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" {
			t.Error("sent credentials")
		}
		http.ServeFile(w, r, filepath.Join(dir, strings.TrimPrefix(r.URL.Path, "/feed/")))
	}))
	options.OnTransfer = func(transfer Transfer) { transfers = append(transfers, transfer) }
	cold := filepath.Join(t.TempDir(), "cold.tar")
	result, err := PullContext(context.Background(), "https://example.com/feed/pack-feed.json", "", cold, options)
	if err != nil || result.Mode != "full" || !bytes.Equal(readTestFile(t, cold), target) {
		t.Fatalf("cold: %+v %v", result, err)
	}
	if err := os.Remove(filepath.Join(dir, feed.Full.Path)); err != nil {
		t.Fatal(err)
	}
	result, err = PullContext(context.Background(), "https://example.com/feed/pack-feed.json", base,
		filepath.Join(t.TempDir(), "updated.tar"), options)
	if err != nil || result.Mode != "delta" || !bytes.Equal(readTestFile(t, base), before) {
		t.Fatalf("delta: %+v %v", result, err)
	}
	mu.Lock()
	if !reflect.DeepEqual(paths, []string{"/feed/pack-feed.json", "/feed/current.tar", "/feed/pack-feed.json", "/feed/delta.tar"}) {
		t.Errorf("requests: %v", paths)
	}
	mu.Unlock()
	if len(transfers) != 4 || transfers[1].Bytes != int64(len(target)) || transfers[3].Bytes != int64(len(readTestFile(t, filepath.Join(dir, "delta.tar")))) ||
		transfers[3].URL != "https://example.com/feed/delta.tar" || transfers[3].Status != 200 {
		t.Fatalf("transfer evidence: %+v", transfers)
	}
	writeInput(t, filepath.Join(dir, feed.Full.Path), target)
	feed.Deltas = []FeedDelta{}
	writeTestFeed(t, dir, feed)
	result, err = PullContext(context.Background(), "https://example.com/feed/pack-feed.json", base,
		filepath.Join(t.TempDir(), "fallback.tar"), options)
	if err != nil || !result.Fallback || result.Mode != "full" {
		t.Fatalf("fallback: %+v %v", result, err)
	}
}

func TestHTTPSURLAndAddressPolicy(t *testing.T) {
	for _, source := range []string{
		"http://example.com/feed/pack-feed.json", "https://user:secret@example.com/pack-feed.json",
		"https://example.com:8443/pack-feed.json", "https://example.com/pack-feed.json?secret",
		"https://example.com/pack-feed.json#fragment", "https://example.com/a/../pack-feed.json",
		"https://example.com/%2e%2e/pack-feed.json", "https://example.com/feed/",
		"https://example.com/pack%2dfeed.json", "https://example.com/a%252fb/pack-feed.json",
	} {
		_, err := PullContext(context.Background(), source, "", filepath.Join(t.TempDir(), "out"), RemoteOptions{AllowHTTPS: true})
		if err == nil || strings.Contains(err.Error(), "secret") {
			t.Fatalf("unsafe URL accepted or leaked: %q %v", source, err)
		}
	}
	if _, err := acquireFeed(context.Background(), "https://example.com/pack-feed.json", RemoteOptions{}); err == nil {
		t.Fatal("missing opt-in accepted")
	}
	for _, address := range []string{"127.0.0.1", "::1", "::ffff:127.0.0.1", "10.0.0.1", "172.16.1.1",
		"192.168.0.1", "169.254.169.254", "fe80::1", "fc00::1", "0.0.0.0", "100.64.0.1",
		"192.0.2.1", "198.18.0.1", "224.0.0.1", "240.0.0.1", "64:ff9b::7f00:1", "2002:7f00:1::",
		"fec0::1", "2001:db8::1", "4000::1", "2606:4700:4700::1111%lo0", "::ffff:8.8.8.8%lo0"} {
		if publicAddress(netip.MustParseAddr(address)) {
			t.Errorf("forbidden IP accepted: %s", address)
		}
	}
	for _, address := range []string{"8.8.8.8", "1.1.1.1", "2606:4700:4700::1111"} {
		if !publicAddress(netip.MustParseAddr(address)) {
			t.Errorf("public IP rejected: %s", address)
		}
	}
	called := false
	dependencies := remoteNetwork{
		lookup: func(context.Context, string) ([]netip.Addr, error) {
			return []netip.Addr{netip.MustParseAddr("8.8.8.8"), netip.MustParseAddr("127.0.0.1")}, nil
		},
		dial: func(context.Context, string, string) (net.Conn, error) { called = true; return nil, io.EOF },
	}
	if _, err := dialPublic(context.Background(), "tcp", "example.com:443", dependencies); err == nil || called {
		t.Fatal("mixed public/private DNS reached a socket")
	}
	if _, err := dialPublic(context.Background(), "tcp", "[::ffff:127.0.0.1]:443", dependencies); err == nil || called {
		t.Fatal("mapped literal reached a socket")
	}
	for _, address := range []string{"fec0::1", "2001:db8::1", "4000::1"} {
		dependencies.lookup = func(context.Context, string) ([]netip.Addr, error) {
			return []netip.Addr{netip.MustParseAddr(address)}, nil
		}
		if _, err := dialPublic(context.Background(), "tcp", "example.com:443", dependencies); err == nil || called {
			t.Fatalf("nonpublic IPv6 DNS answer reached a socket: %s", address)
		}
	}
}

func TestHTTPSRedirectConfinementAndLimits(t *testing.T) {
	for _, redirect := range []string{
		"http://example.com/feed/pack-feed.json", "https://other.example/feed/pack-feed.json",
		"https://user:secret@example.com/feed/pack-feed.json", "https://example.com/outside/pack-feed.json",
		"https://example.com/feed/%2e%2e/outside", "https://127.0.0.1/feed/pack-feed.json",
		"https://example.com/feed/pack-feed.json?secret", "/feed/pack-feed.json",
	} {
		t.Run(redirect, func(t *testing.T) {
			options := httpsFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, redirect, http.StatusFound)
			}))
			_, err := PullContext(context.Background(), "https://example.com/feed/pack-feed.json", "",
				filepath.Join(t.TempDir(), "out"), options)
			if err == nil || strings.Contains(err.Error(), "secret") {
				t.Fatalf("redirect accepted or leaked: %v", err)
			}
		})
	}
	// A confined redirect succeeds, and relative members still use the original
	// feed root rather than a relocated manifest's directory.
	dir, _, _, target, _ := pullFixture(t)
	options := httpsFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/feed/pack-feed.json" {
			http.Redirect(w, r, "/feed/version/pack-feed.json", http.StatusFound)
			return
		}
		name := filepath.Base(r.URL.Path)
		http.ServeFile(w, r, filepath.Join(dir, name))
	}))
	out := filepath.Join(t.TempDir(), "out")
	if _, err := PullContext(context.Background(), "https://example.com/feed/pack-feed.json", "", out, options); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(readTestFile(t, out), target) {
		t.Fatal("redirect changed archive")
	}
}

func TestHTTPSFailureBoundsCancellationAndTLS(t *testing.T) {
	for _, name := range []string{"feed size", "content length", "encoding", "status", "interrupted", "timeout", "cancel", "untrusted TLS"} {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			options := httpsFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch name {
				case "feed size":
					_, _ = io.WriteString(w, strings.Repeat(" ", MaxFeedSize+1))
				case "content length":
					w.Header().Set("Content-Length", "999999999")
				case "encoding":
					w.Header().Set("Content-Encoding", "gzip")
				case "status":
					http.Error(w, "secret corpus text", http.StatusForbidden)
				case "interrupted":
					w.Header().Set("Content-Length", "100")
					_, _ = io.WriteString(w, "{")
				case "cancel":
					cancel()
					<-r.Context().Done()
				case "timeout":
					<-r.Context().Done()
				}
			}))
			if name == "timeout" {
				options.Timeout = 20 * time.Millisecond
			}
			if name == "untrusted TLS" {
				options.network.tls = nil
			}
			out := filepath.Join(t.TempDir(), "out")
			_, err := PullContext(ctx, "https://example.com/feed/pack-feed.json", "", out, options)
			if err == nil || strings.Contains(err.Error(), "secret corpus text") {
				t.Fatalf("failure accepted or body leaked: %v", err)
			}
			if name == "cancel" && !errors.Is(err, context.Canceled) {
				t.Fatalf("cancellation: %v", err)
			}
			if _, err := os.Stat(out); !os.IsNotExist(err) {
				t.Fatal("failed network request published output")
			}
		})
	}
}

func TestHTTPSRequestBudgetAndDeclaredMemberEscapes(t *testing.T) {
	dir, _, _, _, feed := pullFixture(t)
	options := httpsFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(dir, filepath.Base(r.URL.Path)))
	}))
	source, err := acquireFeed(context.Background(), "https://example.com/feed/pack-feed.json", options)
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	for range MaxFeedRequests {
		f, err := source.open(FeedName, MaxFeedSize)
		if err != nil {
			t.Fatal(err)
		}
		_ = f.Close()
	}
	if _, err := source.open(FeedName, MaxFeedSize); err == nil {
		t.Fatal("request budget was not enforced")
	}
	for _, name := range []string{"packs/%2e%2e.tar", "packs/full.tar?query", "packs/full.tar#fragment"} {
		feed.Full.Path = name
		writeTestFeed(t, dir, feed)
		out := filepath.Join(t.TempDir(), "out")
		if _, err := PullContext(context.Background(), "https://example.com/feed/pack-feed.json", "", out, options); err == nil {
			t.Fatalf("declared URL escape accepted: %s", name)
		}
	}
}
