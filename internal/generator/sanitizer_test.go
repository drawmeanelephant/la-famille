package generator

import (
	"strings"
	"testing"
)

func TestContentSanitizerKeepsFigureMarkup(t *testing.T) {
	p := newContentSanitizer()

	input := []byte(`<figure class="lf-figure">` +
		`<img src="/assets/img/a.png" alt="A chart" loading="lazy" decoding="async" title="Caption text">` +
		`<figcaption>Caption text</figcaption></figure>`)

	got := string(p.SanitizeBytes(input))

	for _, want := range []string{
		`<figure class="lf-figure">`,
		`loading="lazy"`,
		`decoding="async"`,
		`title="Caption text"`,
		`<figcaption>Caption text</figcaption>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("sanitizer dropped %q\noutput: %s", want, got)
		}
	}
}

func TestContentSanitizerStripsHostileFigureContent(t *testing.T) {
	p := newContentSanitizer()

	input := []byte(`<figure onclick="steal()"><img src="javascript:alert(1)" ` +
		`loading="eagerish" onerror="boom()" alt="x"><figcaption><script>evil()</script>ok</figcaption></figure>`)

	got := string(p.SanitizeBytes(input))

	for _, banned := range []string{"onclick", "onerror", "javascript:", "<script>", "eagerish"} {
		if strings.Contains(got, banned) {
			t.Errorf("sanitizer kept %q\noutput: %s", banned, got)
		}
	}
	if !strings.Contains(got, "<figure>") || !strings.Contains(got, "ok") {
		t.Errorf("sanitizer should keep the figure shell and safe text\noutput: %s", got)
	}
}

func TestContentSanitizerStillAllowsTaskListsAndSVG(t *testing.T) {
	p := newContentSanitizer()

	got := string(p.SanitizeBytes([]byte(
		`<input type="checkbox" disabled checked> <svg viewBox="0 0 24 24"><path d="M4 6h16"></path></svg>`,
	)))

	for _, want := range []string{`type="checkbox"`, "disabled", "checked", `<svg viewbox="0 0 24 24"`} {
		if !strings.Contains(got, want) {
			t.Errorf("sanitizer dropped %q\noutput: %s", want, got)
		}
	}
}
