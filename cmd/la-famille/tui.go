package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/tbuddy/la-famille/internal/logger"
	"log/slog"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/tbuddy/la-famille/internal/config"
	sitediff "github.com/tbuddy/la-famille/internal/diff"
	"github.com/tbuddy/la-famille/internal/generator"
	"github.com/tbuddy/la-famille/internal/ragexport"
	"github.com/tbuddy/la-famille/internal/watcher"
)

var p *tea.Program

// setupTUICmd builds the tui command. The --log-file value and the bootstrapped
// runtime configuration are passed through cliState and parameters rather than
// package globals (#547): setupRootCmdState hands the real config over when the
// binary builds the tree, and tests exercise the direct-CWD loading path with
// an empty config.
func setupTUICmd(st *cliState, runtimeCfg config.Config, runtimeCfgSet bool) *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Launch the semi-graphical user interface",
		RunE: func(_ *cobra.Command, _ []string) error {
			logTarget := st.globalLogFile
			if logTarget == "" {
				logTarget = "la-famille.log"
			}
			f, _ := logger.Setup(logTarget, true)
			defer func() {
				if f != nil {
					f.Close()
				}
			}()

			var cfg config.Config
			if runtimeCfgSet {
				cfg = runtimeCfg
				if err := cfg.ValidateResolved(); err != nil {
					return fmt.Errorf("configuration validation failed: %w", err)
				}
			} else {
				// No defaults fallback here: config.Load only errors when config.yaml
				// exists but is unreadable or unparsable, and substituting defaults
				// for it would silently discard the operator's real settings.
				var err error
				cfg, err = config.Load("config.yaml")
				if err != nil {
					return fmt.Errorf("failed to load config.yaml: %w", err)
				}
				if err := cfg.Validate(); err != nil {
					return fmt.Errorf("configuration validation failed: %w", err)
				}
			}

			p = tea.NewProgram(initialModel(cfg), tea.WithAltScreen())
			if _, err := p.Run(); err != nil {
				return fmt.Errorf("tui error: %w", err)
			}
			return nil
		},
	}
}

// Rough approximation: 1 token ≈ 4 bytes (OpenAI tokenizer heuristic)
const bytesPerToken = 4

type screen int

const (
	screenMenu screen = iota
	screenRaoul
	screenStats
	screenWorking
	screenServe
	screenDiagnostics
	screenHelp
	screenChanges
)

type menuOption struct {
	label string
}

type tickMsg time.Time

type statsUpdateMsg struct {
	res generator.BuildResult
	err error
}

type workResultMsg struct {
	err error
	res *generator.BuildResult
	msg string
}

type workProgressMsg struct {
	phase     string
	detail    string
	completed int
	total     int
}

type serverErrorMsg struct {
	err error
}

type diagnostic struct {
	level      string
	message    string
	source     string
	nextAction string
}

type model struct {
	ledger            *sitediff.Ledger
	changesCursor     int
	regressionsOnly   bool
	workErr           error
	stats             *generator.BuildResult
	server            *http.Server
	serverCancel      context.CancelFunc
	watcherCancel     context.CancelFunc
	cfg               config.Config
	workPhase         string
	workMsg           string
	choices           []menuOption
	diagnostics       []diagnostic
	workEvents        []string
	cursor            int
	frame             int
	screen            screen
	workCompleted     int
	workTotal         int
	diagnosticCursor  int
	diagnosticsReturn screen
	helpReturn        screen
	width             int
	height            int
	menuOpen          bool
	working           bool
	servePending      bool
	spinner           spinner.Model
	progress          progress.Model
	confetti          int
}

func (m model) workDone() bool {
	return strings.Contains(m.workMsg, "complete") || m.workErr != nil
}

func initialModel(cfg config.Config) model {
	m := model{
		cfg:    cfg,
		screen: screenMenu,
		choices: []menuOption{
			{"Build Site"},
			{"Serve Site"},
			{"Toggle Watch Mode"},
			{"Stats"},
			{"Changes"},
			{"Diagnostics"},
			{"RAG Export"},
			{"Help"},
			{"Just Raoul"},
		},
		menuOpen: true,
		spinner:  newCookSpinner(),
		progress: newGlowProgress(40),
	}
	m.loadLedger()
	return m
}

func getRecoveryGuidance(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "address already in use") || strings.Contains(msg, "bind"):
		return "Port conflict: update 'port' in config.yaml or stop the existing process using the port."
	case strings.Contains(msg, "template") || strings.Contains(msg, "layout"):
		return "Template syntax error: inspect layout templates for missing tags or invalid formatting."
	case strings.Contains(msg, "frontmatter") || strings.Contains(msg, "yaml") || strings.Contains(msg, "markdown"):
		return "Content syntax error: check YAML frontmatter formatting and key delimiters in content files."
	case strings.Contains(msg, "no such file") || strings.Contains(msg, "not found"):
		return "Path missing: ensure content, template, and asset directories exist as configured."
	default:
		return "Check configuration in config.yaml and inspect detailed logs in Diagnostics drawer (press 'd')."
	}
}

var diagnosticSourceRE = regexp.MustCompile(`(?:^|[[:space:]([])([^[:space:]]+:[0-9]+(?::[0-9]+)?)`)
var frontmatterPathRE = regexp.MustCompile(`frontmatter (?:parse )?warning in ([^:]+):`)

func diagnosticNextAction(message string) string {
	lower := strings.ToLower(message)
	switch {
	case strings.Contains(lower, "case-insensitive filesystem") || strings.Contains(lower, "would be the same file on a case-insensitive"):
		return "see docs/issues-420-422.md — case-only collision is same file on case-insensitive filesystem; rename one path"
	case strings.Contains(lower, "frontmatter"):
		if m := frontmatterPathRE.FindStringSubmatch(message); len(m) == 2 {
			rel := strings.TrimSpace(m[1])
			if rel != "" {
				return fmt.Sprintf("fix frontmatter in %s", rel)
			}
		}
		// fallback: try generic "in <path>:" extraction
		if idx := strings.Index(lower, " in "); idx != -1 {
			rest := message[idx+4:]
			if end := strings.Index(rest, ":"); end != -1 {
				rel := strings.TrimSpace(rest[:end])
				if rel != "" && strings.Contains(rel, ".md") {
					return fmt.Sprintf("fix frontmatter in %s", rel)
				}
			}
		}
		return "fix frontmatter in offending file"
	case strings.Contains(lower, "broken internal link"):
		return "run `la-famille check` to list broken links"
	case strings.Contains(lower, "missing referenced asset") || strings.Contains(lower, "missing asset") || (strings.Contains(lower, "asset") && strings.Contains(lower, "missing")):
		return "run `la-famille check --asset-health` to diagnose asset issues"
	case strings.Contains(lower, "unusually large raster"):
		return "run `la-famille check --asset-health` to diagnose large raster assets (consider optimizing)"
	case strings.Contains(lower, "suspicious image extension") || strings.Contains(lower, "unsupported or suspicious image"):
		return "run `la-famille check --asset-health` to diagnose unsupported image formats"
	case strings.Contains(lower, "case mismatch") || strings.Contains(lower, "case-collision") || strings.Contains(lower, "case collision"):
		return "run `la-famille check --asset-health` — asset case mismatch"
	case strings.Contains(lower, "output path collision"):
		if strings.Contains(lower, "case-insensitive") {
			return "see docs/issues-420-422.md — case-only collision"
		}
		return "run `la-famille check` — output collision: rename one source or slug"
	}
	return ""
}

func (m *model) addDiagnostic(level string, err error) {
	if err == nil {
		return
	}
	message := err.Error()
	source := ""
	if match := diagnosticSourceRE.FindStringSubmatch(message); len(match) == 2 {
		source = match[1]
	}
	next := diagnosticNextAction(message)
	m.diagnostics = append(m.diagnostics, diagnostic{level: level, message: message, source: source, nextAction: next})
	m.diagnosticCursor = len(m.diagnostics) - 1
}

func (m *model) addDiagnosticWarning(message string) {
	if strings.TrimSpace(message) == "" {
		return
	}
	source := ""
	if match := diagnosticSourceRE.FindStringSubmatch(message); len(match) == 2 {
		source = match[1]
	}
	next := diagnosticNextAction(message)
	m.diagnostics = append(m.diagnostics, diagnostic{level: "warning", message: message, source: source, nextAction: next})
	m.diagnosticCursor = len(m.diagnostics) - 1
}

func (m *model) showDiagnostics() {
	m.diagnosticsReturn = m.screen
	m.screen = screenDiagnostics
	if len(m.diagnostics) > 0 && m.diagnosticCursor >= len(m.diagnostics) {
		m.diagnosticCursor = len(m.diagnostics) - 1
	}
}

func (m *model) showHelp() {
	m.helpReturn = m.screen
	m.screen = screenHelp
}

func (m *model) toggleWatchMode() {
	m.cfg.WatchMode = !m.cfg.WatchMode
	m.workMsg = fmt.Sprintf("Watch mode %s", map[bool]string{true: "enabled", false: "disabled"}[m.cfg.WatchMode])
}

func (m model) Init() tea.Cmd {
	return nil
}

// stopServing stops any server and watcher started by the TUI. It deliberately
// leaves screen selection to its caller so the same cleanup works for both a
// user-initiated exit and an unexpected server failure.
func (m *model) stopServing() {
	if m.watcherCancel != nil {
		m.watcherCancel()
		m.watcherCancel = nil
	}
	if m.serverCancel != nil {
		m.serverCancel()
		m.serverCancel = nil
	}
	if m.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = m.server.Shutdown(ctx)
		m.server = nil
	}
}

func runServer(server *http.Server, report func(tea.Msg)) {
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		report(serverErrorMsg{err: err})
	}
}

func (m *model) startWatcher() {
	watchCtx, cancelWatch := context.WithCancel(context.Background())
	m.watcherCancel = cancelWatch

	go func(ctx context.Context, c config.Config) {
		if err := watcher.Watch(ctx, c, func(res generator.BuildResult, buildErr error) {
			if p != nil {
				p.Send(statsUpdateMsg{res: res, err: buildErr})
			}
		}); err != nil {
			slog.Error("Watcher thread exited", "error", err)
		}
	}(watchCtx, m.cfg)
}

func serveMux(cfg config.Config, watch bool) *http.ServeMux {
	mux := http.NewServeMux()
	base := cfg.BasePath()
	if base != "" && base != "/" {
		cleanBase := strings.TrimSuffix(base, "/")
		mux.Handle(cleanBase+"/", http.StripPrefix(cleanBase, http.FileServer(http.Dir(cfg.OutputDir))))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/" {
				http.Redirect(w, r, cleanBase+"/", http.StatusFound)
				return
			}
			http.NotFound(w, r)
		})
	} else {
		mux.Handle("/", http.FileServer(http.Dir(cfg.OutputDir)))
	}
	if watch {
		mux.HandleFunc(strings.TrimSuffix(base, "/")+"/livereload", watcher.LiveReloadHandler)
	}
	return mux
}

func (m *model) startServing() tea.Cmd {
	if m.cfg.WatchMode {
		m.startWatcher()
	}

	port := m.cfg.Port
	if port == 0 {
		port = config.DefaultConfig().Port
	}

	serverCtx, serverCancel := context.WithCancel(context.Background())
	m.serverCancel = serverCancel

	server := &http.Server{
		Addr:              fmt.Sprintf("127.0.0.1:%d", port),
		Handler:           serveMux(m.cfg, m.cfg.WatchMode),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		BaseContext: func(net.Listener) context.Context {
			return serverCtx
		},
	}
	m.server = server
	go func() {
		runServer(server, func(msg tea.Msg) {
			if p != nil {
				p.Send(msg)
			}
		})
	}()
	m.screen = screenServe
	m.frame = 0
	return tickCmd()
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*250, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func buildProgressCmd(cfg config.Config) tea.Cmd {
	progress := func(phase string, completed int) tea.Cmd {
		return func() tea.Msg {
			return workProgressMsg{phase: phase, completed: completed, total: 4}
		}
	}
	return tea.Sequence(
		progress("Preparing build", 1),
		progress("Rendering pages", 2),
		progress("Writing assets and indexes", 3),
		func() tea.Msg {
			res, err := generator.Build(cfg)
			msg := "Build complete"
			if err == nil {
				if res.CacheHit {
					msg = "Build complete (cache hit)"
				} else {
					msg = "Build complete (cache miss)"
				}
			}
			return workResultMsg{err: err, msg: msg, res: &res}
		},
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.progress.Width = clampGlowWidth(msg.Width - 18)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.stopServing()
			return m, tea.Quit
		case "d":
			if m.screen == screenDiagnostics {
				m.screen = m.diagnosticsReturn
			} else {
				m.showDiagnostics()
			}
			return m, nil
		case "l":
			if m.screen != screenWorking || m.workDone() {
				m.screen = screenChanges
			}
			return m, nil
		case "r":
			if m.screen == screenChanges {
				m.regressionsOnly = !m.regressionsOnly
				m.changesCursor = 0
				return m, nil
			}
		case "c":
			if m.screen == screenDiagnostics {
				m.diagnostics = nil
				m.diagnosticCursor = 0
				return m, nil
			}
		case "m":
			if m.screen == screenMenu {
				m.menuOpen = !m.menuOpen
				return m, nil
			}
		case "w", "W":
			if m.screen == screenMenu || m.screen == screenStats || m.screen == screenDiagnostics || m.screen == screenHelp {
				m.toggleWatchMode()
				return m, nil
			}
		case "?", "h", "H":
			if m.screen == screenHelp {
				if m.helpReturn != screenHelp {
					m.screen = m.helpReturn
				} else {
					m.screen = screenMenu
				}
				return m, nil
			}
			m.showHelp()
			return m, nil
		case "q", "esc":
			if m.screen == screenHelp {
				if m.helpReturn != screenHelp {
					m.screen = m.helpReturn
				} else {
					m.screen = screenMenu
				}
				return m, nil
			}
			if m.screen == screenDiagnostics {
				m.screen = m.diagnosticsReturn
				return m, nil
			}
			if m.screen == screenMenu && msg.String() == "esc" {
				m.menuOpen = false
				return m, nil
			}
			if msg.String() == "q" && m.screen == screenMenu {
				m.stopServing()
				return m, tea.Quit
			}
			if m.screen != screenWorking || m.workDone() || m.screen == screenServe {
				m.stopServing()
				m.screen = screenMenu
				return m, nil
			}
		case "up", "k":
			if m.screen == screenChanges {
				if m.changesCursor > 0 {
					m.changesCursor--
				}
			} else if m.screen == screenDiagnostics {
				if m.diagnosticCursor > 0 {
					m.diagnosticCursor--
				}
			} else if m.screen == screenMenu && m.menuOpen {
				if m.cursor > 0 {
					m.cursor--
				}
			}
		case "down", "j":
			if m.screen == screenChanges {
				if m.changesCursor < len(m.changeRows())-1 {
					m.changesCursor++
				}
			} else if m.screen == screenDiagnostics {
				if m.diagnosticCursor < len(m.diagnostics)-1 {
					m.diagnosticCursor++
				}
			} else if m.screen == screenMenu && m.menuOpen {
				if m.cursor < len(m.choices)-1 {
					m.cursor++
				}
			}
		case "enter", " ":
			if m.screen == screenMenu && m.menuOpen {
				choice := m.choices[m.cursor].label
				switch choice {
				case "Quit":
					return m, tea.Quit
				case "Just Raoul":
					m.screen = screenRaoul
					m.frame = 0
					return m, tickCmd()
				case "Stats":
					m.screen = screenStats
					return m, nil
				case "Changes":
					m.screen = screenChanges
					return m, nil
				case "Diagnostics":
					m.showDiagnostics()
					return m, nil
				case "Help":
					m.showHelp()
					return m, nil
				case "Toggle Watch Mode":
					m.toggleWatchMode()
					m.workMsg = fmt.Sprintf("Watch mode %s", map[bool]string{true: "enabled", false: "disabled"}[m.cfg.WatchMode])
					return m, nil
				case "Build Site":
					if m.working {
						m.screen = screenWorking
						return m, m.spinner.Tick
					}
					m.screen = screenWorking
					m.working = true
					m.workMsg = "Building site..."
					m.workErr = nil
					m.workPhase = "Preparing build"
					m.workCompleted, m.workTotal = 0, 4
					m.workEvents = nil
					m.confetti = 0
					m.progress.SetPercent(0)
					return m, tea.Batch(buildProgressCmd(m.cfg), m.spinner.Tick)
				case "RAG Export":
					if m.working {
						m.screen = screenWorking
						return m, m.spinner.Tick
					}
					m.screen = screenWorking
					m.working = true
					m.workMsg = "Exporting RAG data..."
					m.workErr = nil
					m.workPhase = ""
					m.workCompleted, m.workTotal = 0, 0
					m.workEvents = nil
					m.confetti = 0
					m.progress.SetPercent(0)
					return m, tea.Batch(func() tea.Msg {
						err := ragexport.RunExport(m.cfg)
						return workResultMsg{err: err, msg: "RAG Export complete"}
					}, m.spinner.Tick)
				case "Serve Site", "Serve Site with Watch":
					if m.working {
						m.screen = screenWorking
						return m, m.spinner.Tick
					}
					if choice == "Serve Site with Watch" {
						m.cfg.WatchMode = true
					}
					m.screen = screenWorking
					m.working = true
					m.servePending = true
					m.workMsg = "Building site..."
					m.workErr = nil
					m.workPhase = "Preparing build"
					m.workCompleted, m.workTotal = 0, 4
					m.workEvents = nil
					m.confetti = 0
					m.progress.SetPercent(0)
					return m, tea.Batch(buildProgressCmd(m.cfg), m.spinner.Tick)
				}
			} else if m.screen == screenWorking {
				if m.workDone() {
					m.screen = screenMenu
				}
			}
		}

	case spinner.TickMsg:
		if m.screen == screenWorking && !m.workDone() {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case progress.FrameMsg:
		pm, cmd := m.progress.Update(msg)
		m.progress = pm.(progress.Model)
		return m, cmd

	case tickMsg:
		switch m.screen {
		case screenRaoul, screenServe:
			m.frame = (m.frame + 1) % len(raoulPoses)
			return m, tickCmd()
		case screenWorking:
			if m.confetti > 0 {
				m.confetti--
				return m, tickCmd()
			}
		}

	case statsUpdateMsg:
		if msg.err != nil {
			m.addDiagnostic("error", fmt.Errorf("watch rebuild failed: %w", msg.err))
			return m, nil
		}
		newRes := msg.res
		m.stats = &newRes
		m.recordLedger(newRes)
		return m, nil

	case workResultMsg:
		m.working = false
		m.workMsg = msg.msg
		m.workErr = msg.err
		if msg.err != nil {
			m.addDiagnostic("error", msg.err)
		}
		if msg.err == nil && msg.res != nil && msg.res.ErrorCount > 0 {
			m.addDiagnosticWarning(fmt.Sprintf("Build completed with %d error(s)", msg.res.ErrorCount))
		}
		if msg.res != nil {
			for _, w := range msg.res.Warnings {
				m.addDiagnosticWarning(w)
			}
		}
		m.workCompleted = m.workTotal
		m.workPhase = "Complete"
		if msg.err != nil {
			m.workPhase = "Build failed"
			m.workEvents = append(m.workEvents, fmt.Sprintf("Error: %v", msg.err))
		}
		if msg.res != nil {
			m.stats = msg.res
			if msg.err == nil {
				m.recordLedger(*msg.res)
			}
			if msg.res.ErrorCount > 0 {
				m.workEvents = append(m.workEvents, fmt.Sprintf("Warning: %d build errors reported", msg.res.ErrorCount))
			}
			if len(msg.res.Warnings) > 0 {
				m.workEvents = append(m.workEvents, fmt.Sprintf("Warning: %d warning(s) — open diagnostics (d) for next actions", len(msg.res.Warnings)))
			}
		}
		if m.servePending {
			m.servePending = false
			if msg.err != nil {
				m.workMsg = "Unable to start serve (initial build failed)"
				return m, nil
			}
			return m, m.startServing()
		}
		if msg.err == nil {
			m.confetti = confettiTotalFrames
			return m, tickCmd()
		}

	case workProgressMsg:
		m.workPhase = msg.phase
		m.workCompleted = msg.completed
		m.workTotal = msg.total
		if msg.detail != "" {
			m.workEvents = append(m.workEvents, msg.detail)
		}
		if m.workTotal > 0 {
			return m, m.progress.SetPercent(float64(m.workCompleted) / float64(m.workTotal))
		}

	case serverErrorMsg:
		m.addDiagnostic("error", msg.err)
		m.stopServing()
		// Only the serve screen gets yanked to the error view — anywhere
		// else the diagnostics drawer records the failure without tearing
		// the user away from what they were doing.
		if m.screen == screenServe {
			m.screen = screenWorking
			m.workMsg = fmt.Sprintf("Server error: %v", msg.err)
			m.workErr = msg.err
		}

	}

	return m, nil
}

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205"))

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("228"))

	accentStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39"))

	subtleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244"))

	successBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")).
			Bold(true)

	warningBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color("215")).
			Bold(true)

	errorBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")).
			Bold(true)

	infoBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Bold(true)

	offBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	panelBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(0, 1)

	boxBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)
)

func (m model) renderStatusPanel(maxWidth int) string {
	var sb strings.Builder

	sb.WriteString(headerStyle.Render("📊 DASHBOARD STATUS ✨") + "\n")
	sb.WriteString(accentStyle.Render(strings.Repeat("─", 24)) + "\n\n")

	// 1. Watch Mode
	watchStr := offBadge.Render("DISABLED")
	if m.cfg.WatchMode {
		watchStr = successBadge.Render("ENABLED")
	}
	sb.WriteString(fmt.Sprintf("%s %s\n", subtleStyle.Render("Watch Mode:"), watchStr))

	// 2. Server Status
	serverStr := offBadge.Render("OFF")
	if m.server != nil {
		port := m.cfg.Port
		if port == 0 {
			port = config.DefaultConfig().Port
		}
		serverStr = infoBadge.Render(fmt.Sprintf("RUNNING (127.0.0.1:%d)", port))
	}
	sb.WriteString(fmt.Sprintf("%s %s\n", subtleStyle.Render("Server Status:"), serverStr))

	// 3. Build Phase
	phaseStr := subtleStyle.Render("Idle")
	if m.workPhase != "" {
		if m.workPhase == "Complete" {
			phaseStr = successBadge.Render("Complete")
		} else if strings.Contains(m.workPhase, "failed") || strings.Contains(m.workPhase, "Error") {
			phaseStr = errorBadge.Render(m.workPhase)
		} else {
			phaseStr = warningBadge.Render(m.workPhase)
		}
	}
	sb.WriteString(fmt.Sprintf("%s %s\n", subtleStyle.Render("Build Phase:"), phaseStr))

	// 4. Cache Status
	cacheStr := subtleStyle.Render("N/A")
	if m.stats != nil {
		if m.stats.CacheHit {
			cacheStr = successBadge.Render("HIT")
		} else {
			cacheStr = warningBadge.Render("MISS")
		}
	}
	sb.WriteString(fmt.Sprintf("%s %s\n", subtleStyle.Render("Cache Status:"), cacheStr))

	// 5. Diagnostics
	diagStr := successBadge.Render("OK")
	if m.workErr != nil {
		diagStr = errorBadge.Render(fmt.Sprintf("1 error (%v)", m.workErr))
	} else if m.stats != nil && m.stats.ErrorCount > 0 {
		diagStr = warningBadge.Render(fmt.Sprintf("%d build warnings/errors", m.stats.ErrorCount))
	}
	sb.WriteString(fmt.Sprintf("%s %s\n", subtleStyle.Render("Diagnostics:"), diagStr))

	// 6. Build Stats Summary
	if m.stats != nil {
		sb.WriteString(fmt.Sprintf("%s %d ms | %s %d pages\n",
			subtleStyle.Render("Duration:"), m.stats.Duration.Milliseconds(),
			subtleStyle.Render("Generated:"), m.stats.PageCount))
	}

	// 7. RAG Estimates Summary
	ragDir := m.cfg.RagDir
	if ragDir == "" {
		ragDir = "rag-archive"
	}
	totalTokens := 0
	files, err := os.ReadDir(ragDir)
	if err == nil {
		for _, file := range files {
			if !file.IsDir() && strings.HasSuffix(file.Name(), ".md") {
				if info, err := file.Info(); err == nil {
					totalTokens += int(info.Size() / bytesPerToken)
				}
			}
		}
		sb.WriteString(fmt.Sprintf("%s ~%d tokens\n", subtleStyle.Render("RAG Tokens:"), totalTokens))
	} else {
		sb.WriteString(fmt.Sprintf("%s Not exported\n", subtleStyle.Render("RAG Archive:")))
	}

	style := panelBorder
	if maxWidth > 4 {
		style = style.MaxWidth(maxWidth)
	}
	return style.Render(sb.String())
}

func (m model) View() string {
	switch m.screen {
	case screenMenu:
		isWide := m.width >= 80 || m.width <= 0
		effectiveWidth := m.width
		if effectiveWidth <= 0 {
			effectiveWidth = 80
		}

		var leftBuf strings.Builder
		leftBuf.WriteString(accentStyle.Render(staticRaoul()) + "\n\n")
		leftBuf.WriteString(titleStyle.Render("Welcome to La Famille TUI ✨") + "\n\n")
		leftBuf.WriteString(headerStyle.Render("🍔 OCTOBURGER MENU 🍔") + "\n")

		if !m.menuOpen {
			leftBuf.WriteString("\nMenu closed. Press m to open • d: Diagnostics • w: Watch • ?: Help • q: Quit")
		} else {
			for i, choice := range m.choices {
				cursor := "  "
				style := subtleStyle
				if m.cursor == i {
					cursor = "✦ "
					// focus-visible: mirror templates/layout.html focus-visible:outline pattern with underline + background highlight for a11y
					style = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true).Underline(true).Background(lipgloss.Color("236"))
				}
				leftBuf.WriteString(fmt.Sprintf("%s %s\n", cursor, style.Render(choice.label)))
			}
			leftBuf.WriteString("\n↑/k, ↓/j: Navigate • Enter/Space: Select • m: Menu • d: Diagnostics • w: Watch • ?: Help • q: Quit")
		}

		if isWide {
			leftColWidth := 38
			if effectiveWidth > 90 {
				leftColWidth = 42
			}
			rightColWidth := effectiveWidth - leftColWidth - 4
			if rightColWidth < 35 {
				rightColWidth = 35
			}

			leftView := lipgloss.NewStyle().Width(leftColWidth).Render(leftBuf.String())
			rightView := m.renderStatusPanel(rightColWidth)
			return lipgloss.JoinHorizontal(lipgloss.Top, leftView, "  ", rightView)
		}

		// Compact stacked layout for narrow screens
		statusView := m.renderStatusPanel(effectiveWidth - 2)
		stacked := lipgloss.JoinVertical(lipgloss.Left, leftBuf.String(), "\n", statusView)
		return lipgloss.NewStyle().MaxWidth(effectiveWidth).Render(stacked)

	case screenRaoul:
		s := accentStyle.Render(animatedRaoul(m.frame))
		s += "\n\nJust Raoul doing mascot work.\n\nPress d for diagnostics • Press ?/h for help • Press Esc or q to go back to main menu."
		if m.width > 0 {
			return lipgloss.NewStyle().MaxWidth(m.width).Render(s)
		}
		return s

	case screenStats:
		s := titleStyle.Render("Stats Dashboard") + "\n\n"
		if m.stats == nil {
			s += "No build has been run yet in this session.\n"
		} else {
			cacheStatus := "Miss"
			if m.stats.CacheHit {
				cacheStatus = "Hit"
			}
			s += fmt.Sprintf("Last Build Time: %d ms\n", m.stats.Duration.Milliseconds())
			s += fmt.Sprintf("Total Pages Generated: %d\n", m.stats.PageCount)
			s += fmt.Sprintf("Error Count: %d\n", m.stats.ErrorCount)
			s += fmt.Sprintf("Cache Status: %s\n", cacheStatus)

			h := m.stats.Health
			s += "\n" + headerStyle.Render("Content Health & Observability") + "\n"
			s += fmt.Sprintf("Total Word Count: %d\n", h.TotalWordCount)
			s += fmt.Sprintf("Average Words per Page: %.1f\n", h.AvgWordsPerPage)

			s += "Top Tags: "
			if len(h.TopTags) == 0 {
				s += "None\n"
			} else {
				var tagStrs []string
				for _, t := range h.TopTags {
					tagStrs = append(tagStrs, fmt.Sprintf("%s (%d)", t.Tag, t.Count))
				}
				s += strings.Join(tagStrs, ", ") + "\n"
			}

			s += fmt.Sprintf("Graph Nodes: %d | Graph Edges: %d\n", h.NodeCount, h.EdgeCount)

			s += fmt.Sprintf("Orphaned Pages (%d): ", len(h.OrphanedPages))
			if len(h.OrphanedPages) == 0 {
				s += "None\n"
			} else {
				s += strings.Join(h.OrphanedPages, ", ") + "\n"
			}

			s += fmt.Sprintf("Missing Descriptions (%d): ", len(h.MissingDescriptions))
			if len(h.MissingDescriptions) == 0 {
				s += "None\n"
			} else {
				s += strings.Join(h.MissingDescriptions, ", ") + "\n"
			}

			s += fmt.Sprintf("Missing Dates (%d): ", len(h.MissingDates))
			if len(h.MissingDates) == 0 {
				s += "None\n"
			} else {
				s += strings.Join(h.MissingDates, ", ") + "\n"
			}

			hasHealthIssues := len(h.OrphanedPages) > 0 || len(h.MissingDescriptions) > 0 || len(h.MissingDates) > 0 || m.stats.ErrorCount > 0
			s += "\n" + headerStyle.Render("Next-Step Guidance") + "\n"
			if hasHealthIssues {
				if len(h.OrphanedPages) > 0 {
					s += subtleStyle.Render("• Orphaned pages detected: Add internal links to connect them to the site graph.") + "\n"
				}
				if len(h.MissingDescriptions) > 0 {
					s += subtleStyle.Render("• Missing descriptions: Add 'description:' frontmatter for better SEO.") + "\n"
				}
				if len(h.MissingDates) > 0 {
					s += subtleStyle.Render("• Missing dates: Add 'date:' frontmatter for proper post ordering.") + "\n"
				}
				if m.stats.ErrorCount > 0 {
					s += subtleStyle.Render("• Build warnings/errors: Press 'd' to open Diagnostics drawer for details.") + "\n"
				}
			} else {
				s += successBadge.Render("• Content health is optimal! All pages have valid metadata and graph links.") + "\n"
			}
		}
		s += "\nRAG Token Estimations:\n"
		ragDir := m.cfg.RagDir
		if ragDir == "" {
			ragDir = "rag-archive"
		}
		totalTokens := 0
		files, err := os.ReadDir(ragDir)
		if err == nil {
			for _, file := range files {
				if !file.IsDir() && strings.HasSuffix(file.Name(), ".md") {
					info, err := file.Info()
					if err == nil {
						size := info.Size()
						tokens := size / bytesPerToken
						totalTokens += int(tokens)
						s += fmt.Sprintf("- %s: ~%d tokens\n", file.Name(), tokens)
					}
				}
			}
			s += fmt.Sprintf("\nTotal Estimated Tokens: ~%d (Note: 1 token ≈ 4 bytes)\n", totalTokens)
		} else {
			s += "RAG archive not found. Run 'RAG Export' to generate bundles.\n"
		}
		s += "\nPress d for diagnostics • Press w to toggle watch • Press ?/h for help • Press Esc or q to go back to main menu."
		if m.width > 0 {
			return boxBorder.MaxWidth(m.width).Render(s)
		}
		return s

	case screenChanges:
		return m.changesView()

	case screenDiagnostics:
		info := currentBuildInfo()
		s := titleStyle.Render(fmt.Sprintf("Diagnostics & Recovery Guidance [%s (commit: %s)]", info.Version, info.Commit)) + "\n\n"
		if len(m.diagnostics) == 0 {
			s += "No diagnostics recorded. All system checks passed cleanly.\n\n"
			s += "Press d, Esc, or q to return • ?: Help • w: Watch"
			if m.width > 0 {
				return boxBorder.MaxWidth(m.width).Render(s)
			}
			return s
		}
		for i, item := range m.diagnostics {
			cursor := "  "
			style := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
			if i == m.diagnosticCursor {
				cursor = "> "
				style = lipgloss.NewStyle().Bold(true)
			}
			color := "9"
			if strings.EqualFold(item.level, "warning") || strings.EqualFold(item.level, "warn") {
				color = "11"
			}
			label := lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(strings.ToUpper(item.level))
			s += fmt.Sprintf("%s%s %s\n", cursor, label, style.Render(item.message))
			if item.source != "" {
				s += fmt.Sprintf("    Source: %s\n", item.source)
			}
			if item.nextAction != "" {
				s += fmt.Sprintf("    Next: %s\n", item.nextAction)
				if g := getRecoveryGuidance(errors.New(item.message)); g != "" && g != item.nextAction {
					s += fmt.Sprintf("    Action: %s\n", g)
				}
			} else {
				if next := diagnosticNextAction(item.message); next != "" {
					s += fmt.Sprintf("    Next: %s\n", next)
				} else if guidance := getRecoveryGuidance(errors.New(item.message)); guidance != "" {
					s += fmt.Sprintf("    Action: %s\n", guidance)
				}
			}
		}
		s += "\nUse ↑/↓ to navigate • c: Clear • d/Esc/q: Return • ?: Help • w: Watch"
		if m.width > 0 {
			return boxBorder.MaxWidth(m.width).Render(s)
		}
		return s

	case screenHelp:
		s := titleStyle.Render("La Famille Help & Keybindings") + "\n\n"
		s += headerStyle.Render("Navigation & Commands") + "\n"
		s += "  ↑ / k        Move selection up\n"
		s += "  ↓ / j        Move selection down\n"
		s += "  Enter / Space Select command\n"
		s += "  m            Toggle octoburger menu open/closed\n\n"
		s += headerStyle.Render("Global Shortcuts") + "\n"
		s += "  d            Toggle Diagnostics drawer\n"
		s += "  w            Toggle Watch Mode (menu/stats/diagnostics)\n"
		s += "  ? / h        Open/close this help (legend)\n"
		s += "  c            Clear diagnostics (in Diagnostics drawer)\n"
		s += "  l            Open Changes (last successful build)\n"
		s += "  r            Regressions-only filter (in Changes)\n"
		s += "  Esc / q      Go back / close menu / quit\n"
		s += "  Ctrl+C       Force quit application\n\n"
		s += headerStyle.Render("Workflow Hints") + "\n"
		s += "  • Build Site: Compiles site content & static assets to public directory.\n"
		s += "  • Serve Site: Runs HTTP web server on 127.0.0.1 (with live reload if Watch Mode is on).\n"
		s += "  • Toggle Watch Mode: Automatically rebuilds site on content change (w).\n"
		s += "  • Diagnostics: Inspect error logs, warnings, and next CLI actions (d → Next: …).\n\n"
		s += "Press d for diagnostics • Press Esc, q, ? or h to return • w: Toggle watch"
		s += "\n" + subtleStyle.Render("Raoul is watching. Always. 💅")
		if m.width > 0 {
			return boxBorder.MaxWidth(m.width).Render(s)
		}
		return s

	case screenWorking:
		s := titleStyle.Render("Task Progress") + "\n\n"
		if m.confetti > 0 && m.workDone() {
			s = confettiRain(confettiTotalFrames-m.confetti, confettiWidth(m.width)) + "\n" + s
		}
		if !m.workDone() {
			s += m.spinner.View() + " " + flavorView(m.workPhase, len(m.workEvents)) + "\n"
		}
		s += m.workMsg + "\n"
		if m.workPhase != "" && m.workTotal > 0 {
			s += fmt.Sprintf("Phase: %s (%d/%d)\n", m.workPhase, m.workCompleted, m.workTotal)
			if !m.workDone() {
				s += m.progress.ViewAs(float64(m.workCompleted)/float64(m.workTotal)) + "\n"
			}
		}
		if len(m.workEvents) > 0 {
			s += "\nEvents:\n"
			for _, event := range m.workEvents {
				s += "- " + event + "\n"
			}
		}
		if m.workErr != nil {
			s += "\n" + errorBadge.Render(fmt.Sprintf("Error: %v", m.workErr)) + "\n"
			guidance := getRecoveryGuidance(m.workErr)
			if guidance != "" {
				s += warningBadge.Render("Recovery Guidance: ") + guidance + "\n"
			}
		} else if strings.Contains(m.workMsg, "complete") {
			s += "\n" + successBanner() + "\n"
			if m.stats != nil && m.stats.ErrorCount > 0 {
				s += warningBadge.Render(fmt.Sprintf("Warning: Build completed with %d error(s). Press 'd' to view diagnostics.", m.stats.ErrorCount)) + "\n"
			}
		}
		if m.workDone() {
			s += "\nPress d for diagnostics • Press ?/h for help • Press Enter or Esc to return to menu"
		} else {
			s += "\nPress d for diagnostics • Press ?/h for help • Working…"
		}
		if m.width > 0 {
			return boxBorder.MaxWidth(m.width).Render(s)
		}
		return s

	case screenServe:
		port := m.cfg.Port
		if port == 0 {
			port = config.DefaultConfig().Port
		}
		s := accentStyle.Render(animatedRaoul(m.frame))
		s += "\n\n"
		s += titleStyle.Render(fmt.Sprintf("Serving site on http://127.0.0.1:%d", port)) + "\n"
		if m.cfg.WatchMode {
			s += successBadge.Render("Watch Mode: ENABLED (Live Reload active)") + "\n"
		} else {
			s += subtleStyle.Render("Watch Mode: DISABLED") + "\n"
		}
		s += infoBadge.Render("Server Status: RUNNING") + " " + pulseDots(m.frame) + "\n\n"
		s += "Press d for diagnostics • Press ?/h for help • Press Esc or q to stop serving and return to menu"
		if m.width > 0 {
			return lipgloss.NewStyle().MaxWidth(m.width).Render(s)
		}
		return s

	}

	return "Unknown screen"
}
