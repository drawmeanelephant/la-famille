package pack

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

const (
	MaxFeedRequests  = 64
	MaxFeedRedirects = 3
)

type remoteNetwork struct {
	lookup func(context.Context, string) ([]netip.Addr, error)
	dial   func(context.Context, string, string) (net.Conn, error)
	tls    *tls.Config
}

func defaultRemoteNetwork() remoteNetwork {
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	return remoteNetwork{
		lookup: func(ctx context.Context, host string) ([]netip.Addr, error) {
			return net.DefaultResolver.LookupNetIP(ctx, "ip", host)
		},
		dial: dialer.DialContext,
	}
}

// Deny non-public and special-purpose ranges in addition to Go's private,
// loopback and link-local classifications. Reject IPv6 translation/tunnel
// ranges so an embedded forbidden IPv4 address cannot bypass the policy.
var forbiddenNetworks = func() []netip.Prefix {
	var prefixes []netip.Prefix
	for _, value := range []string{
		"0.0.0.0/8", "100.64.0.0/10", "192.0.0.0/24", "192.0.2.0/24",
		"192.88.99.0/24", "198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24",
		"240.0.0.0/4", "::/96", "64:ff9b::/96", "64:ff9b:1::/48",
		"100::/64", "2001::/23", "2001:db8::/32", "2002::/16", "3fff::/20",
	} {
		prefixes = append(prefixes, netip.MustParsePrefix(value))
	}
	return prefixes
}()

func publicAddress(ip netip.Addr) bool {
	if ip.Zone() != "" {
		return false
	}
	ip = ip.Unmap()
	if !ip.IsValid() || !ip.IsGlobalUnicast() || ip.IsPrivate() ||
		ip.IsLoopback() || ip.IsLinkLocalUnicast() {
		return false
	}
	// Go's IsGlobalUnicast includes deprecated site-local and unallocated
	// IPv6 space. Public IPv6 unicast is confined to the allocated GUA block.
	if ip.Is6() && !netip.MustParsePrefix("2000::/3").Contains(ip) {
		return false
	}
	for _, prefix := range forbiddenNetworks {
		if prefix.Contains(ip) {
			return false
		}
	}
	return true
}

func safeFeedURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid HTTPS feed URL")
	}
	if u.Scheme != "https" || u.Opaque != "" || u.Hostname() == "" ||
		u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" ||
		u.RawPath != "" || strings.ContainsAny(u.Path, "%\\\x00\r\n") {
		return nil, fmt.Errorf("feed URL must be credential-free HTTPS with a canonical path and no query or fragment")
	}
	if u.Port() != "" && u.Port() != "443" {
		return nil, fmt.Errorf("HTTPS feeds use port 443 only")
	}
	if u.Path != path.Clean(u.Path) || path.Base(u.Path) != FeedName {
		return nil, fmt.Errorf("HTTPS source must name a canonical pack-feed.json")
	}
	u.Host = strings.ToLower(u.Host)
	return u, nil
}

func confinedURL(u, origin *url.URL, directory string) bool {
	return u.Scheme == "https" && u.Host == origin.Host && u.User == nil &&
		u.Opaque == "" && u.RawQuery == "" && !u.ForceQuery && u.Fragment == "" &&
		u.RawPath == "" && !strings.ContainsAny(u.Path, "%\\\x00\r\n") &&
		u.Path == path.Clean(u.Path) && strings.HasPrefix(u.Path, directory)
}

func dialPublic(ctx context.Context, network, address string, dependencies remoteNetwork) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil || port != "443" {
		return nil, fmt.Errorf("invalid HTTPS destination")
	}
	var ips []netip.Addr
	if ip, err := netip.ParseAddr(host); err == nil {
		ips = []netip.Addr{ip}
	} else {
		ips, err = dependencies.lookup(ctx, host)
		if err != nil {
			return nil, fmt.Errorf("HTTPS DNS resolution failed")
		}
	}
	if len(ips) == 0 || len(ips) > 32 {
		return nil, fmt.Errorf("HTTPS DNS result is empty or exceeds 32 addresses")
	}
	for _, ip := range ips {
		if !publicAddress(ip) {
			return nil, fmt.Errorf("HTTPS destination is not a public network address")
		}
	}
	// Pin the actual socket to a validated IP, never resolve the hostname twice.
	for _, ip := range ips {
		conn, err := dependencies.dial(ctx, network, net.JoinHostPort(ip.String(), port))
		if err == nil {
			return conn, nil
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
	}
	return nil, fmt.Errorf("HTTPS connection failed")
}

func openHTTPSFeed(ctx context.Context, raw string, options RemoteOptions) (*feedSource, error) {
	origin, err := safeFeedURL(raw)
	if err != nil {
		return nil, err
	}
	timeout := options.Timeout
	if timeout == 0 {
		timeout = DefaultRequestTimeout
	}
	if timeout < 0 {
		return nil, fmt.Errorf("request timeout must be positive")
	}
	dependencies := defaultRemoteNetwork()
	if options.network != nil {
		dependencies = *options.network
	}
	directory := path.Dir(origin.Path) + "/"
	if directory == "//" {
		directory = "/"
	}
	requests := 0
	transport := &http.Transport{
		Proxy: nil, DisableCompression: true, TLSClientConfig: dependencies.tls,
		TLSHandshakeTimeout: 10 * time.Second, ResponseHeaderTimeout: 15 * time.Second,
		MaxResponseHeaderBytes: 32 << 10, MaxConnsPerHost: 1,
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			return dialPublic(ctx, network, address, dependencies)
		},
	}
	client := &http.Client{Transport: transport, Timeout: timeout}
	if options.OnTransfer != nil {
		client.Transport = tracedTransport{base: transport, report: options.OnTransfer}
	}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		requests++
		if len(via) > MaxFeedRedirects || requests > MaxFeedRequests ||
			!confinedURL(req.URL, origin, directory) {
			return fmt.Errorf("HTTPS redirect violates feed confinement or request limit")
		}
		return nil
	}
	var temporary []string
	closeSource := func() {
		transport.CloseIdleConnections()
		for _, name := range temporary {
			_ = os.Remove(name)
		}
	}
	open := func(name string, limit int64) (*os.File, error) {
		if err := validatePath(name); err != nil {
			return nil, fmt.Errorf("invalid HTTPS feed member path")
		}
		if strings.ContainsAny(name, "%?#") {
			return nil, fmt.Errorf("HTTPS member paths cannot contain URL escapes, queries or fragments")
		}
		member := *origin
		member.Path = directory + name
		if !confinedURL(&member, origin, directory) {
			return nil, fmt.Errorf("HTTPS member escapes feed directory")
		}
		requests++
		if requests > MaxFeedRequests {
			return nil, fmt.Errorf("HTTPS feed request limit exceeded")
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, member.String(), nil)
		if err != nil {
			return nil, fmt.Errorf("invalid HTTPS request")
		}
		req.Header.Set("Accept-Encoding", "identity")
		req.Header.Set("Cache-Control", "no-cache")
		resp, err := client.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			var timeoutError net.Error
			if errors.As(err, &timeoutError) && timeoutError.Timeout() {
				return nil, fmt.Errorf("HTTPS request timed out")
			}
			// Never include URL-bearing transport errors or server response text.
			return nil, fmt.Errorf("HTTPS request failed (TLS, network or redirect policy)")
		}
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("HTTPS artifact unavailable: %w", fs.ErrNotExist)
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("HTTPS response status %d", resp.StatusCode)
		}
		if resp.ContentLength > limit || resp.Header.Get("Content-Encoding") != "" {
			return nil, fmt.Errorf("HTTPS artifact exceeds size bound or uses unsupported encoding")
		}
		f, err := os.CreateTemp("", "la-famille-feed-*")
		if err != nil {
			return nil, err
		}
		temporary = append(temporary, f.Name())
		size, copyErr := io.Copy(f, io.LimitReader(resp.Body, limit+1))
		if copyErr != nil || size > limit {
			_ = f.Close()
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			return nil, fmt.Errorf("HTTPS artifact interrupted or exceeds %s-byte bound", strconv.FormatInt(limit, 10))
		}
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			_ = f.Close()
			return nil, err
		}
		return f, nil
	}
	return &feedSource{
		open: open, Close: closeSource,
		load: func() (Feed, error) {
			f, err := open(FeedName, MaxFeedSize)
			if err != nil {
				return Feed{}, err
			}
			defer f.Close()
			data, err := io.ReadAll(f)
			if err != nil {
				return Feed{}, err
			}
			feed, err := parseFeed(data)
			if err != nil {
				return Feed{}, fmt.Errorf("invalid HTTPS feed manifest")
			}
			for _, name := range append([]string{feed.Full.Path}, feedDeltaPaths(feed)...) {
				if strings.ContainsAny(name, "%?#") {
					return Feed{}, fmt.Errorf("invalid HTTPS feed member path")
				}
			}
			return feed, nil
		},
	}, nil
}
