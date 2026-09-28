package ask

import (
	"strings"
	"testing"
)

func TestScrubLogTextRedactsSensitiveValues(t *testing.T) {
	raw := "user jane@example.com bearer=super-secret-token password=hunter2 credential=abcdefghijklmnopqrstuvwxyz012345"
	got := scrubLogText(raw)

	for _, unwanted := range []string{
		"jane@example.com",
		"super-secret-token",
		"hunter2",
		"abcdefghijklmnopqrstuvwxyz012345",
	} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("scrubLogText leaked %q in %q", unwanted, got)
		}
	}
	for _, expected := range []string{"[REDACTED_EMAIL]", "bearer=[REDACTED]", "password=[REDACTED]", "[REDACTED_TOKEN]"} {
		if !strings.Contains(got, expected) {
			t.Fatalf("scrubLogText missing %q in %q", expected, got)
		}
	}
}

func TestScrubLogTextPreservesSafeContext(t *testing.T) {
	const raw = "provider unavailable: ollama status 503"
	if got := scrubLogText(raw); got != raw {
		t.Fatalf("scrubLogText changed safe context: got %q, want %q", got, raw)
	}
}
