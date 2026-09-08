package main

import (
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/config"
)

func TestAnimatedRaoulCyclesPoses(t *testing.T) {
	if len(raoulPoses) < 4 {
		t.Fatalf("raoulPoses = %d frames, want at least 4", len(raoulPoses))
	}
	if animatedRaoul(0) == animatedRaoul(3) {
		t.Fatal("frames 0 and 3 should be different poses")
	}
	if animatedRaoul(7) != animatedRaoul(7%len(raoulPoses)) {
		t.Fatal("frames should wrap modulo len(raoulPoses)")
	}

	// Every pose must be the same height, and each line index must have the
	// same width across all poses so the mascot never jitters between frames.
	reference := strings.Split(raoulPoses[0], "\n")
	for i, pose := range raoulPoses {
		lines := strings.Split(pose, "\n")
		if len(lines) != len(reference) {
			t.Errorf("pose %d has %d lines, want %d", i, len(lines), len(reference))
			continue
		}
		for j, line := range lines {
			if len(line) != len(reference[j]) {
				t.Errorf("pose %d line %d is %d cols, want %d", i, j, len(line), len(reference[j]))
			}
		}
	}
}

func TestTintRainbowWrapsPalette(t *testing.T) {
	s := "line1\nline2"
	if tintRainbow(s, 0) != tintRainbow(s, len(rainbowColors)) {
		t.Fatal("tint start should wrap around the palette")
	}
	// Colors shift per line so adjacent lines differ.
	lines := strings.Split(tintRainbow(s, 0), "\n")
	if len(lines) != 2 || lines[0] == lines[1] {
		t.Fatalf("expected two differently colored lines, got %#v", lines)
	}
}

func TestConfettiRainDeterministic(t *testing.T) {
	a := confettiRain(3, 60)
	b := confettiRain(3, 60)
	if a != b {
		t.Fatal("same frame and width must render identical confetti")
	}
	if confettiRain(3, 60) == confettiRain(4, 60) {
		t.Fatal("different frames should move the particles")
	}
	rows := strings.Split(a, "\n")
	if len(rows) != 4 {
		t.Fatalf("confetti rows = %d, want 4", len(rows))
	}
	for _, row := range rows {
		if strings.TrimSpace(row) == "" {
			t.Fatalf("confetti row has no particles: %q", row)
		}
	}
}

func TestConfettiWidthClampsNarrowTerminals(t *testing.T) {
	if got := confettiWidth(0); got <= 0 {
		t.Fatalf("confettiWidth(0) = %d, want positive default", got)
	}
	if got := confettiWidth(10); got < 1 {
		t.Fatalf("confettiWidth(10) = %d, want at least 1", got)
	}
	if got := confettiWidth(200); got > 70 {
		t.Fatalf("confettiWidth(200) = %d, want capped at 70", got)
	}
}

func TestFunFlavorDeterministicAndCoversPhases(t *testing.T) {
	a := funFlavor("Rendering pages", 0)
	b := funFlavor("Rendering pages", 0)
	if a != b {
		t.Fatal("flavor must be deterministic")
	}
	for _, phase := range []string{"Preparing build", "Rendering pages", "Writing assets and indexes", "Checking provider & corpus", "", "Unknown"} {
		if funFlavor(phase, 0) == "" {
			t.Errorf("funFlavor(%q) returned empty string", phase)
		}
	}
	if funFlavor("Rendering pages", 0) == funFlavor("Rendering pages", 1) {
		t.Fatal("flavor should rotate with event count for multi-line phases")
	}
}

func TestPulseDotsCycle(t *testing.T) {
	if pulseDots(0) == pulseDots(1) {
		t.Fatal("pulse dots should change between frames")
	}
	a := pulseDots(2)
	b := pulseDots(2)
	if a != b {
		t.Fatal("pulse dots must be deterministic")
	}
}

func TestSuccessBannerContainsSparkle(t *testing.T) {
	if !strings.Contains(successBanner(), "SUCCESS!") {
		t.Fatalf("success banner missing SUCCESS!: %q", successBanner())
	}
}

func TestClampGlowWidthBounds(t *testing.T) {
	cases := []struct{ in, want int }{
		{0, 40}, {-5, 40}, {5, 12}, {12, 12}, {40, 40}, {100, 60},
	}
	for _, c := range cases {
		if got := clampGlowWidth(c.in); got != c.want {
			t.Errorf("clampGlowWidth(%d) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestGlowProgressRendersAtWidth(t *testing.T) {
	p := newGlowProgress(40)
	view := p.ViewAs(0.5)
	if view == "" {
		t.Fatal("progress bar view is empty")
	}
	plain := strings.ReplaceAll(view, "\n", "")
	if got := len([]rune(plain)); got < 12 || got > 60 {
		t.Fatalf("bar width = %d runes, want within [12,60]: %q", got, view)
	}
}

func TestNewCookSpinnerHasFrames(t *testing.T) {
	sp := newCookSpinner()
	if len(sp.Spinner.Frames) == 0 {
		t.Fatal("cook spinner has no frames")
	}
	if !strings.Contains(strings.Join(sp.Spinner.Frames, ""), "🍔") {
		t.Fatal("cook spinner should include the octoburger")
	}
}

func TestWorkingScreenShowsGlow(t *testing.T) {
	m := initialModel(config.Config{})
	m.screen = screenWorking
	m.workMsg = "Building site..."
	m.workPhase = "Rendering pages"
	m.workCompleted, m.workTotal = 1, 4
	m.width = 100

	view := m.View()
	if !strings.Contains(view, "Rendering pages") {
		t.Fatalf("working view missing phase: %q", view)
	}
	if !strings.Contains(view, "cooking the octoburger") && !strings.Contains(view, "serving looks and pages") && !strings.Contains(view, "dusting off the divs") {
		t.Fatalf("working view missing flavor line: %q", view)
	}
}

func TestWorkingScreenConfettiOnSuccess(t *testing.T) {
	m := initialModel(config.Config{})
	m.screen = screenWorking
	m.workTotal = 4
	updated, cmd := m.Update(workResultMsg{msg: "Build complete (cache miss)"})
	m = updated.(model)
	if m.confetti != confettiTotalFrames {
		t.Fatalf("confetti = %d, want %d on success", m.confetti, confettiTotalFrames)
	}
	if cmd == nil {
		t.Fatal("expected tick cmd to animate confetti")
	}
	if !strings.Contains(m.View(), "SUCCESS!") {
		t.Fatalf("success view missing banner: %q", m.View())
	}
	// Confetti decays per tick and stops asking for more ticks at zero.
	for i := 0; i < confettiTotalFrames+1; i++ {
		updated, _ = m.Update(tickMsg{})
		m = updated.(model)
	}
	if m.confetti != 0 {
		t.Fatalf("confetti = %d after decay loop, want 0", m.confetti)
	}
}
