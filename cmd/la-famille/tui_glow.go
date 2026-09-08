package main

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
)

// cookSpinner is the octoburger cooking show: the burger assembles itself
// while the fam waits for builds, exports, and the oracle to wake up.
var cookSpinner = spinner.Spinner{
	Frames: []string{"🍞", "🥬", "🥩", "🍔"},
	FPS:    time.Second / 4,
}

// rainbowColors sweeps red → orange → yellow → green → cyan → blue → violet →
// magenta. Applied per line, indexed by frame + line, it gives Raoul a color
// wave that rolls across him while he dances.
var rainbowColors = []string{"196", "208", "226", "46", "51", "39", "135", "199"}

var (
	spinnerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("213")).
			Bold(true)

	flavorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("212")).
			Italic(true)

	pulseStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39"))

	pulseStyleAlt = lipgloss.NewStyle().
			Foreground(lipgloss.Color("213"))
)

// raoulPoses is Raoul's little performance: blinks, squints, excitement, and
// feet that sway left and right. Every pose is the same 6-line, 7-column
// grid so frames never jitter when they swap.
var raoulPoses = []string{
	"  .---.\n ( @ @ )\n  )   (\n (v|v|v)\n  \\ | /\n   \\|/",
	"  .---.\n ( - @ )\n  )   (\n (v|v|v)\n  \\ | /\n   \\|/",
	"  .---.\n ( ^ ^ )\n  )   (\n (v|v|v)\n  \\ | /\n   \\|/",
	"  .---.\n ( > < )\n  )   (\n (v|v|v)\n  / | \\\n   /|\\",
	"  .---.\n ( O O )\n  )   (\n (v|v|v)\n  \\ | /\n   \\|/",
	"  .---.\n ( @ ^ )\n  )   (\n (v|v|v)\n  / | \\\n   /|\\",
}

func staticRaoul() string {
	return tintRainbow(raoulPoses[0], 0)
}

func animatedRaoul(frame int) string {
	return tintRainbow(raoulPoses[frame%len(raoulPoses)], frame)
}

// tintRainbow paints each line of s with a rainbow color, offset by start so
// the wave travels as frames advance. (start+i) mod len wraps the palette so
// any start index lands on the same sequence 8 lines later.
func tintRainbow(s string, start int) string {
	lines := strings.Split(s, "\n")
	out := make([]string, len(lines))
	for i, line := range lines {
		color := rainbowColors[(start+i)%len(rainbowColors)]
		out[i] = lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(line)
	}
	return strings.Join(out, "\n")
}

// confettiTotalFrames is how many ticks the success confetti rains for.
const confettiTotalFrames = 16

// confettiGlyphs are the party pieces. All single-cell glyphs so the grid
// stays aligned on every terminal.
var confettiGlyphs = []string{"✦", "*", "·", "+", "✧"}

// confettiRain draws a deterministic 4-row sprinkle of colored particles.
// Positions are pure arithmetic on the frame index (no rand) so the same
// frame always renders the same rain — testable and race-free.
func confettiRain(frame, width int) string {
	width = clampGlowWidth(width)
	var sb strings.Builder
	for r := 0; r < 4; r++ {
		cells := make([]string, width)
		for i := range cells {
			cells[i] = " "
		}
		for g, glyph := range confettiGlyphs {
			x := ((g*13 + r*31 + frame*7) * (g + 3)) % width
			color := rainbowColors[(frame+r+g)%len(rainbowColors)]
			cells[x] = lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(glyph)
		}
		sb.WriteString(strings.Join(cells, ""))
		if r < 3 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// confettiWidth fits the rain to the terminal with breathing room on both
// sides, degrading gracefully on narrow panes.
func confettiWidth(termWidth int) int {
	switch {
	case termWidth <= 0:
		return 40
	case termWidth < 20:
		return termWidth - 2
	case termWidth > 70:
		return 70
	default:
		return termWidth - 10
	}
}

// phaseFlavor maps build phases to flavor lines that rotate as work events
// stream in, so long builds get a little variety show.
var phaseFlavor = map[string][]string{
	"Preparing build":            {"stretching, hydrating, manifesting…"},
	"Rendering pages":            {"serving looks and pages…", "dusting off the divs…"},
	"Writing assets and indexes": {"folding fresh fits (assets)…", "tucking in the tags…"},
	"Checking provider & corpus": {"waking up the oracle…", "counting the spells…"},
}

var defaultFlavor = []string{"cooking the octoburger…", "zhuzhing pixels…", "summoning the fam…"}

// funFlavor picks a deterministic flavor line: same phase + event count
// always yields the same line.
func funFlavor(phase string, events int) string {
	lines := defaultFlavor
	if l, ok := phaseFlavor[phase]; ok && len(l) > 0 {
		lines = l
	}
	return lines[events%len(lines)]
}

func flavorView(phase string, events int) string {
	return flavorStyle.Render(funFlavor(phase, events))
}

var pulseFrames = []string{"●●●", "◉◉◉", "○○○"}

// pulseDots cycles a live-server heartbeat, alternating pink and cyan so it
// shimmers while the site is being served.
func pulseDots(frame int) string {
	style := pulseStyleAlt
	if frame%2 == 0 {
		style = pulseStyle
	}
	return style.Render(pulseFrames[frame%len(pulseFrames)])
}

// successBanner is the confetti's finale.
func successBanner() string {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("236")).
		Background(lipgloss.Color("213")).
		Padding(0, 1).
		Render("✨ SUCCESS! ✨")
}

// clampGlowWidth keeps the progress bar and confetti inside a sane width no
// matter what the terminal reports.
func clampGlowWidth(w int) int {
	switch {
	case w <= 0:
		return 40
	case w < 12:
		return 12
	case w > 60:
		return 60
	default:
		return w
	}
}

func newCookSpinner() spinner.Model {
	return spinner.New(
		spinner.WithSpinner(cookSpinner),
		spinner.WithStyle(spinnerStyle),
	)
}

func newGlowProgress(width int) progress.Model {
	p := progress.New(
		progress.WithScaledGradient("#FF34D8", "#7C3AED"),
		progress.WithoutPercentage(),
	)
	p.Width = clampGlowWidth(width)
	return p
}
