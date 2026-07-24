package main

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tbuddy/la-famille/internal/ask"
	"github.com/tbuddy/la-famille/internal/config"
)

// askServerReadyMsg is emitted by the launchAskServer command once the ask
// server has been started in a background goroutine. The Update loop is
// the only place that mutates model state from this message, which
// preserves Bubble Tea's single-writer threading model.
type askServerReadyMsg struct {
	server *ask.Server
	cancel context.CancelFunc
	host   string
	port   int
	err    error
}

// askServerFailedMsg is dispatched when the background goroutine running
// the ask server reports a non-cancellation error (for example, port
// already in use). The Update loop routes it into the diagnostics drawer
// so the user sees why the assistant stopped rather than staring at a
// "Server Status: RUNNING" line for a port that is actually dead.
type askServerFailedMsg struct {
	err error
}

// launchAskServer creates the ask.Config from the TUI's current config and
// starts the assistant in a background goroutine, returning an
// askServerReadyMsg so the typed Update switch can wire the lifecycle
// fields safely. Returning the message (rather than mutating model inside
// the cmd) avoids the canonical Bubble Tea data race.
//
// The goroutine also sends an askServerFailedMsg via `p.Send` if the
// HTTP listener errors out (e.g. "address already in use") so the
// TUI surface remains honest about server health.
func launchAskServer(cfg config.Config) tea.Cmd {
	return func() tea.Msg {
		host := "127.0.0.1"
		port := askFlagBundle.port
		if port == 0 {
			port = ask.PortDefault
		}
		askCfg := ask.Config{
			ProviderName: askFlagBundle.provider,
			Model:        askFlagBundle.model,
			Host:         host,
			Port:         port,
			RagDir:       firstNonEmpty(askFlagBundle.ragDir, cfg.RagDir, "rag-archive"),
			OutputDir:    firstNonEmpty(askFlagBundle.outputDir, cfg.OutputDir, "public"),
			LoopbackOnly: true,
		}

		server, err := ask.NewServer(askCfg)
		if err != nil {
			return askServerReadyMsg{
				err:  fmt.Errorf("ask: %w", err),
				host: host, port: port,
			}
		}
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			if runErr := server.Start(ctx); runErr != nil && runErr != context.Canceled && p != nil {
				p.Send(askServerFailedMsg{err: runErr})
			}
		}()
		// The runtime will dispatch this to Update; the lifecycle fields
		// are assigned there under the framework's serialization rules.
		return askServerReadyMsg{server: server, cancel: cancel, host: host, port: port}
	}
}
