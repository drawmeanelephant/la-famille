package pack

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
)

const DefaultRequestTimeout = 2 * time.Minute

// RemoteOptions never permits HTTP, custom trust roots, credentials, proxies,
// or private destinations. Timeout bounds each complete request, including body.
type RemoteOptions struct {
	AllowHTTPS bool
	Timeout    time.Duration
	OnTransfer func(Transfer)
	network    *remoteNetwork // Same-package tests only; no CLI security bypass.
}

type archiveOpener func(string, int64) (*os.File, error)

type feedSource struct {
	load  func() (Feed, error)
	open  archiveOpener
	Close func()
}

func acquireFeed(ctx context.Context, source string, options RemoteOptions) (*feedSource, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.Contains(source, "://") {
		if !options.AllowHTTPS {
			return nil, fmt.Errorf("remote feeds require explicit --allow-https")
		}
		return openHTTPSFeed(ctx, source, options)
	}
	root, err := openFeedRoot(source)
	if err != nil {
		return nil, err
	}
	return &feedSource{
		load: func() (Feed, error) { return loadFeed(root) },
		open: func(name string, limit int64) (*os.File, error) {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			return openFeedSource(root, name, limit)
		},
		Close: func() { _ = root.Close() },
	}, nil
}
