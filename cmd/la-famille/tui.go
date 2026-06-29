package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
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
	RunE: func(cmd *cobra.Command, args []string) error {
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

type screen int

const (
	screenMenu screen = iota
	screenRaoul
	screenStats
	screenWorking
	screenServe
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

type model struct {
	cfg     config.Config
	screen  screen
	choices []menuOption
	cursor  int
	frame   int
	workMsg string
	workErr error
	server  *http.Server
	stats   *generator.BuildResult
}

func initialModel(cfg config.Config) model {
	return model{
		cfg:    cfg,
		screen: screenMenu,
		choices: []menuOption{
			{"Build Site"},
			{"RAG Export"},
			{"Serve Site"},
			{"Serve Site with Watch"},
			{"Stats"},
			{"Just Raoul"},
			{"Quit"},
		},
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*500, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			if m.screen == screenMenu {
				return m, tea.Quit
			} else if m.screen != screenWorking || strings.Contains(m.workMsg, "complete") || m.screen == screenServe {
				if m.screen == screenServe && m.server != nil {
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
					m.server.Shutdown(ctx)
					m.server = nil
				}
				m.screen = screenMenu
				return m, nil
			}
		case "esc":
			if m.screen != screenWorking || strings.Contains(m.workMsg, "complete") || m.screen == screenServe {
				if m.screen == screenServe && m.server != nil {
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
					m.server.Shutdown(ctx)
					m.server = nil
				}
				m.screen = screenMenu
				return m, nil
			}
		case "up", "k":
			if m.screen == screenMenu {
				if m.cursor > 0 {
					m.cursor--
				}
			}
		case "down", "j":
			if m.screen == screenMenu {
				if m.cursor < len(m.choices)-1 {
					m.cursor++
				}
			}
		case "enter", " ":
			if m.screen == screenMenu {
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
				case "Build Site":
					m.screen = screenWorking
					m.workMsg = "Building site..."
					m.workErr = nil

					// Re-assigning to avoid capturing loop variable problem, though we don't have a loop here
					cfg := m.cfg
					return m, func() tea.Msg {
						res, err := generator.Build(cfg)
						return workResultMsg{err: err, msg: "Build complete", res: &res}
					}
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
						port = 8080
					}

					if choice == "Serve Site with Watch" {
						go watcher.Watch(m.cfg, func(res generator.BuildResult) {
							if p != nil {
								p.Send(statsUpdateMsg{res: res})
							}
						})
					}

					m.server = &http.Server{
						Addr:              fmt.Sprintf("127.0.0.1:%d", port),
						Handler:           http.FileServer(http.Dir(m.cfg.OutputDir)),
						ReadHeaderTimeout: 5 * time.Second,
					}
					go func() {
						_ = m.server.ListenAndServe()
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
		if msg.res != nil {
			m.stats = msg.res
		}

	}

	return m, nil
}

func (m model) View() string {
	switch m.screen {
	case screenMenu:
		s := lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Render(staticRaoul()) + "\n\n"
		s += lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).Render("Welcome to La Famille TUI") + "\n\n"

		for i, choice := range m.choices {
			cursor := "  "
			style := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
			if m.cursor == i {
				cursor = "> "
				style = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
			}
			s += fmt.Sprintf("%s %s\n", cursor, style.Render(choice.label))
		}
		s += "\nPress q to quit."
		return s

	case screenRaoul:
		s := lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Render(animatedRaoul(m.frame))
		s += "\n\nPress Esc or q to go back."
		return s

	case screenStats:
		s := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).Render("Stats Dashboard") + "\n\n"
		if m.stats == nil {
			s += "No build has been run yet in this session.\n"
		} else {
			s += fmt.Sprintf("Last Build Time: %d ms\n", m.stats.Duration.Milliseconds())
			s += fmt.Sprintf("Total Pages Generated: %d\n", m.stats.PageCount)
			s += fmt.Sprintf("Error Count: %d\n", m.stats.ErrorCount)
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
						tokens := size / 4
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
		return s

	case screenWorking:
		s := m.workMsg + "\n"
		if m.workErr != nil {
			s += lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(fmt.Sprintf("Error: %v", m.workErr)) + "\n"
		} else if strings.Contains(m.workMsg, "complete") {
			s += lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render("Success!") + "\n"
		}
		s += "\nPress Enter or Esc to return to the menu."
		return s

	case screenServe:
		port := m.cfg.Port
		if port == 0 {
			port = 8080
		}
		s := lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Render(animatedRaoul(m.frame))
		s += "\n\n"
		s += lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true).Render(fmt.Sprintf("Serving site on http://localhost:%d", port))
		s += "\n\nPress Esc or q to stop serving and go back."
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
