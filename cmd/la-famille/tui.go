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

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/generator"
	"github.com/tbuddy/la-famille/internal/ragexport"
	"github.com/tbuddy/la-famille/internal/watcher"
)

var p *tea.Program

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch the semi-graphical user interface",
	RunE: func(_ *cobra.Command, _ []string) error {
		logTarget := globalLogFile
		if logTarget == "" {
			logTarget = "la-famille.log"
		}
		f, _ := logger.Setup(logTarget, true)
		defer func() {
			if f != nil {
				f.Close()
			}
		}()

		cfg, err := config.Load("config.yaml")
		if err != nil {
			// use defaults if config fails
			cfg = config.Config{
				ContentDir: "content",
				OutputDir:  "public",
				Template:   "templates/layout.html",
				AssetDir:   "assets",
				RagDir:     "rag-archive",
			}
		}
		if err := cfg.Validate(); err != nil {
			return fmt.Errorf("configuration validation failed: %w", err)
		}

		p = tea.NewProgram(initialModel(cfg), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("tui error: %w", err)
		}
		return nil
	},
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
)

type menuOption struct {
	label string
}

type tickMsg time.Time

type statsUpdateMsg struct {
	res generator.BuildResult
}

type workResultMsg struct {
	err error
	msg string
	res *generator.BuildResult
}

type workProgressMsg struct {
	phase     string
	completed int
	total     int
	detail    string
}

type serverErrorMsg struct {
	err error
}

type diagnostic struct {
	level   string
	message string
	source  string
}

type model struct {
	cfg               config.Config
	screen            screen
	choices           []menuOption
	cursor            int
	menuOpen          bool
	frame             int
	workMsg           string
	workErr           error
	workPhase         string
	workCompleted     int
	workTotal         int
	workEvents        []string
	server            *http.Server
	serverCancel      context.CancelFunc
	watcherCancel     context.CancelFunc
	stats             *generator.BuildResult
	diagnostics       []diagnostic
	diagnosticCursor  int
	diagnosticsReturn screen
	width             int
	height            int
}

func initialModel(cfg config.Config) model {
	return model{
		cfg:    cfg,
		screen: screenMenu,
		choices: []menuOption{
			{"Build Site"},
			{"Serve Site"},
			{"Toggle Watch Mode"},
			{"Stats"},
			{"Diagnostics"},
			{"RAG Export"},
			{"Help"},
			{"Quit"},
		},
		menuOpen: true,
	}
}

var diagnosticSourceRE = regexp.MustCompile(`(?:^|[[:space:]([])([^[:space:]]+:[0-9]+(?::[0-9]+)?)`)

func (m *model) addDiagnostic(level string, err error) {
	if err == nil {
		return
	}
	message := err.Error()
	source := ""
	if match := diagnosticSourceRE.FindStringSubmatch(message); len(match) == 2 {
		source = match[1]
	}
	m.diagnostics = append(m.diagnostics, diagnostic{level: level, message: message, source: source})
	m.diagnosticCursor = len(m.diagnostics) - 1
}

func (m *model) showDiagnostics() {
	m.diagnosticsReturn = m.screen
	m.screen = screenDiagnostics
	if len(m.diagnostics) > 0 && m.diagnosticCursor >= len(m.diagnostics) {
		m.diagnosticCursor = len(m.diagnostics) - 1
	}
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

func tickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*500, func(t time.Time) tea.Msg {
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
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "d":
			if m.screen == screenDiagnostics {
				m.screen = m.diagnosticsReturn
			} else {
				m.showDiagnostics()
			}
			return m, nil
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
		case "q", "esc":
			if m.screen == screenDiagnostics {
				m.screen = m.diagnosticsReturn
				return m, nil
			}
			if m.screen == screenMenu && msg.String() == "esc" {
				m.menuOpen = false
				return m, nil
			}
			if msg.String() == "q" && m.screen == screenMenu {
				return m, tea.Quit
			}
			if m.screen != screenWorking || strings.Contains(m.workMsg, "complete") || m.screen == screenServe {
				m.stopServing()
				m.screen = screenMenu
				return m, nil
			}
		case "up", "k":
			if m.screen == screenDiagnostics {
				if m.diagnosticCursor > 0 {
					m.diagnosticCursor--
				}
			} else if m.screen == screenMenu && m.menuOpen {
				if m.cursor > 0 {
					m.cursor--
				}
			}
		case "down", "j":
			if m.screen == screenDiagnostics {
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
				case "Diagnostics":
					m.screen = screenDiagnostics
					return m, nil
				case "Help":
					m.screen = screenHelp
					return m, nil
				case "Toggle Watch Mode":
					m.cfg.WatchMode = !m.cfg.WatchMode
					m.workMsg = fmt.Sprintf("Watch mode %s", map[bool]string{true: "enabled", false: "disabled"}[m.cfg.WatchMode])
					return m, nil
				case "Build Site":
					m.screen = screenWorking
					m.workMsg = "Building site..."
					m.workErr = nil
					m.workPhase = "Preparing build"
					m.workCompleted, m.workTotal = 0, 4
					m.workEvents = nil
					return m, buildProgressCmd(m.cfg)
				case "RAG Export":
					m.screen = screenWorking
					m.workMsg = "Exporting RAG data..."
					m.workErr = nil
					return m, func() tea.Msg {
						err := ragexport.RunExport(m.cfg)
						return workResultMsg{err: err, msg: "RAG Export complete"}
					}
				case "Serve Site", "Serve Site with Watch":
					m.screen = screenServe
					m.frame = 0
					port := m.cfg.Port
					if port == 0 {
						port = config.DefaultConfig().Port
					}

					if choice == "Serve Site with Watch" {
						m.cfg.WatchMode = true
						if _, err := generator.Build(m.cfg); err != nil {
							slog.Error("Initial build failed", "error", err)
						}

						watchCtx, cancelWatch := context.WithCancel(context.Background())
						m.watcherCancel = cancelWatch

						go func(ctx context.Context, c config.Config) {
							if err := watcher.Watch(ctx, c, func(res generator.BuildResult) {
								if p != nil {
									p.Send(statsUpdateMsg{res: res})
								}
							}); err != nil {
								slog.Error("Watcher thread exited", "error", err)
							}
						}(watchCtx, m.cfg)
					}

					mux := http.NewServeMux()
					mux.Handle("/", http.FileServer(http.Dir(m.cfg.OutputDir)))
					if m.cfg.WatchMode {
						mux.HandleFunc("/livereload", watcher.LiveReloadHandler)
					}

					serverCtx, serverCancel := context.WithCancel(context.Background())
					m.serverCancel = serverCancel

					server := &http.Server{
						Addr:              fmt.Sprintf("127.0.0.1:%d", port),
						Handler:           mux,
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
					return m, tickCmd()
				}
			} else if m.screen == screenWorking {
				if strings.Contains(m.workMsg, "complete") || m.workErr != nil {
					m.screen = screenMenu
				}
			}
		}

	case tickMsg:
		if m.screen == screenRaoul || m.screen == screenServe {
			m.frame = (m.frame + 1) % 2
			return m, tickCmd()
		}

	case statsUpdateMsg:
		newRes := msg.res
		m.stats = &newRes
		return m, nil

	case workResultMsg:
		m.workMsg = msg.msg
		m.workErr = msg.err
		if msg.err != nil {
			m.addDiagnostic("error", msg.err)
		}
		if msg.err == nil && msg.res != nil && msg.res.ErrorCount > 0 {
			m.diagnostics = append(m.diagnostics, diagnostic{level: "warning", message: fmt.Sprintf("Build completed with %d error(s)", msg.res.ErrorCount)})
		}
		m.workCompleted = m.workTotal
		m.workPhase = "Complete"
		if msg.err != nil {
			m.workPhase = "Build failed"
			m.workEvents = append(m.workEvents, fmt.Sprintf("Error: %v", msg.err))
		}
		if msg.res != nil {
			m.stats = msg.res
			if msg.res.ErrorCount > 0 {
				m.workEvents = append(m.workEvents, fmt.Sprintf("Warning: %d build errors reported", msg.res.ErrorCount))
			}
		}

	case workProgressMsg:
		m.workPhase = msg.phase
		m.workCompleted = msg.completed
		m.workTotal = msg.total
		if msg.detail != "" {
			m.workEvents = append(m.workEvents, msg.detail)
		}

	case serverErrorMsg:
		m.addDiagnostic("error", msg.err)
		m.stopServing()
		m.screen = screenWorking
		m.workMsg = "Unable to start server"
		m.workErr = msg.err

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

	sb.WriteString(headerStyle.Render("📊 DASHBOARD STATUS") + "\n\n")

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
		leftBuf.WriteString(titleStyle.Render("Welcome to La Famille TUI") + "\n\n")
		leftBuf.WriteString(headerStyle.Render("🍔 OCTOBURGER MENU") + "\n")

		if !m.menuOpen {
			leftBuf.WriteString("\nMenu closed. Press m to open commands, q to quit.")
		} else {
			for i, choice := range m.choices {
				cursor := "  "
				style := subtleStyle
				if m.cursor == i {
					cursor = "> "
					style = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
				}
				leftBuf.WriteString(fmt.Sprintf("%s %s\n", cursor, style.Render(choice.label)))
			}
			leftBuf.WriteString("\nPress q to quit.")
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
		s += "\n\nPress Esc or q to go back."
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
		s += "\nPress Esc or q to go back."
		if m.width > 0 {
			return boxBorder.MaxWidth(m.width).Render(s)
		}
		return s

	case screenDiagnostics:
		s := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).Render("Diagnostics") + "\n\n"
		if len(m.diagnostics) == 0 {
			return s + "No diagnostics recorded.\n\nUse d, Esc, or q to return."
		}
		for i, item := range m.diagnostics {
			cursor := "  "
			style := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
			if i == m.diagnosticCursor {
				cursor = "> "
				style = lipgloss.NewStyle().Bold(true)
			}
			color := "9"
			if item.level == "warning" {
				color = "11"
			}
			label := lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(strings.ToUpper(item.level))
			s += fmt.Sprintf("%s%s %s\n", cursor, label, style.Render(item.message))
			if item.source != "" {
				s += fmt.Sprintf("    %s\n", item.source)
			}
		}
		s += "\nUse ↑/↓ to navigate, c to clear, d/Esc/q to return."
		if m.width > 0 {
			return boxBorder.MaxWidth(m.width).Render(s)
		}
		return s

	case screenHelp:
		s := titleStyle.Render("La Famille Help") +
			"\n\n↑/k and ↓/j  Navigate\nEnter/Space  Select\nm             Toggle octoburger menu\nEsc           Close menu or go back\nq             Quit\n\nPress Esc or q to go back."
		if m.width > 0 {
			return boxBorder.MaxWidth(m.width).Render(s)
		}
		return s

	case screenWorking:
		s := m.workMsg + "\n"
		if m.workPhase != "" && m.workTotal > 0 {
			s += fmt.Sprintf("Phase: %s (%d/%d)\n", m.workPhase, m.workCompleted, m.workTotal)
		}
		if len(m.workEvents) > 0 {
			s += "\nEvents:\n"
			for _, event := range m.workEvents {
				s += "- " + event + "\n"
			}
		}
		if m.workErr != nil {
			s += errorBadge.Render(fmt.Sprintf("Error: %v", m.workErr)) + "\n"
		} else if strings.Contains(m.workMsg, "complete") {
			s += successBadge.Render("Success!") + "\n"
		}
		s += "\nPress Enter or Esc to return to the menu."
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
		s += titleStyle.Render(fmt.Sprintf("Serving site on http://localhost:%d", port))
		s += "\n\nPress Esc or q to stop serving and go back."
		if m.width > 0 {
			return lipgloss.NewStyle().MaxWidth(m.width).Render(s)
		}
		return s
	}

	return "Unknown screen"
}

func staticRaoul() string {
	return `       .---.
      ( o o )
       \_-_/
      / | | \
     / / \ \ \`
}

func animatedRaoul(frame int) string {
	if frame == 0 {
		return `       .---.
      ( o o )
       \_-_/
      / | | \
     / / \ \ \`
	}
	return `       .---.
      ( - - )
       \_-_/
      \ \ / /
       \ | | /`
}
