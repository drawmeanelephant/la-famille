package main

import (
	"context"
	"fmt"
	"net/http"
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
			}
		}

		p := tea.NewProgram(initialModel(cfg), tea.WithAltScreen())
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

type workResultMsg struct {
	err error
	msg string
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
					m.server.Shutdown(context.Background())
					m.server = nil
				}
				m.screen = screenMenu
				return m, nil
			}
		case "esc":
			if m.screen != screenWorking || strings.Contains(m.workMsg, "complete") || m.screen == screenServe {
				if m.screen == screenServe && m.server != nil {
					m.server.Shutdown(context.Background())
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
						err := generator.Build(cfg)
						return workResultMsg{err: err, msg: "Build complete"}
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
						go watcher.Watch(m.cfg)
					}

					m.server = &http.Server{Addr: fmt.Sprintf(":%d", port), Handler: http.FileServer(http.Dir(m.cfg.OutputDir))}
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

	case workResultMsg:
		m.workMsg = msg.msg
		m.workErr = msg.err
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
		s := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).Render("Stats (Coming Soon)") + "\n\n"
		s += "We don't have stats yet, but we ultimately should have shit like:\n"
		s += "- Build time\n"
		s += "- RAG sizes\n"
		s += "- LLM context window representations\n"
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
