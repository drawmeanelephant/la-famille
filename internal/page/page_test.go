package page

import (
	"html/template"
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/config"
)

// allFieldsTemplate references every exported Page field plus the Site config
// subfields the shipped layout actually uses.
const allFieldsTemplate = `{{.AnimationCues}}|{{.Content}}|{{.Title}}|{{.Author}}|{{.Date}}|{{.VideoScript}}|{{.SoundtrackTheme}}|{{.Layout}}|{{.ComplianceModal}}|{{.Description}}|{{.Image}}|{{.CanonicalURL}}|{{.Site.SiteName}}|{{.Site.Theme}}|{{range .Site.SiteLinks}}{{.Label}}={{.URL}};{{end}}`

func render(t *testing.T, tmplSrc string, p Page) string {
	t.Helper()
	tmpl, err := template.New("t").Parse(tmplSrc)
	if err != nil {
		t.Fatalf("parse template: %v", err)
	}
	var out strings.Builder
	if err := tmpl.Execute(&out, p); err != nil {
		t.Fatalf("execute template: %v", err)
	}
	return out.String()
}

// A zero-value Page is the initial state every build starts from: every string
// field empty, Content empty, and the embedded site config zeroed. Rendering it
// must not error and must produce empty output for every field.
func TestPageZeroValueRendersEmpty(t *testing.T) {
	got := render(t, allFieldsTemplate, Page{})
	// 15 values joined by 14 separators; the SiteLinks range renders nothing.
	want := strings.Repeat("|", 14)
	if got != want {
		t.Errorf("zero-value render = %q, want %q", got, want)
	}
}

// All exported fields must survive the template round-trip, including the Site
// config fields (SiteName, Theme, SiteLinks) that page authors configure.
func TestPageRendersAllFields(t *testing.T) {
	p := Page{
		AnimationCues:   "fade-in",
		Content:         template.HTML("<p>hello</p>"),
		Title:           "My Page",
		Author:          "Ada",
		Date:            "2026-08-29",
		VideoScript:     "voiceover",
		SoundtrackTheme: "ambient",
		Layout:          "editorial",
		ComplianceModal: "consent",
		Description:     "A page",
		Image:           "/img/cover.png",
		CanonicalURL:    "https://example.com/my-page",
		Site: config.Config{
			SiteName: "La Famille",
			Theme:    "midnight",
			SiteLinks: []config.SiteLink{
				{Label: "Home", URL: "/"},
				{Label: "About", URL: "/about"},
			},
		},
	}

	got := render(t, allFieldsTemplate, p)
	for _, want := range []string{
		"fade-in", "<p>hello</p>", "My Page", "Ada", "2026-08-29",
		"voiceover", "ambient", "editorial", "consent", "A page",
		"/img/cover.png", "https://example.com/my-page",
		"La Famille", "midnight", "Home=/;", "About=/about;",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("render missing %q; full output: %q", want, got)
		}
	}
}

// The one field that must NOT be escaped is Content (it is the rendered
// markdown); everything else is author/operator text and html/template must
// escape it. Pinning this is what keeps XSS out of titles and site names.
func TestPageEscapesTextButNotContent(t *testing.T) {
	p := Page{
		Content: template.HTML("<b>raw</b>"),
		Title:   "<script>alert(1)</script>",
		Site: config.Config{
			SiteName: `"><img src=x onerror=alert(2)>`,
		},
	}

	got := render(t, allFieldsTemplate, p)
	if !strings.Contains(got, "<b>raw</b>") {
		t.Errorf("Content must render as raw HTML, got %q", got)
	}
	if strings.Contains(got, "<script>alert(1)</script>") {
		t.Errorf("Title must be HTML-escaped, got %q", got)
	}
	if strings.Contains(got, `<img src=x onerror=alert(2)>`) {
		t.Errorf("Site.SiteName must be HTML-escaped, got %q", got)
	}
	if !strings.Contains(got, "&lt;script&gt;") {
		t.Errorf("expected escaped title markup, got %q", got)
	}
}

// The Site field is a value type, not a pointer: mutating the page's copy must
// not leak into a shared config the caller still holds. This is the contract
// callers like the generator rely on when they build one Page per file.
func TestPageSiteIsACopy(t *testing.T) {
	cfg := config.Config{SiteName: "original"}
	p := Page{Site: cfg}

	cfg.SiteName = "mutated"
	p.Site.Theme = "editorial"

	if p.Site.SiteName != "original" {
		t.Errorf("page.Site.SiteName = %q, want %q (must be a copy, not an alias)", p.Site.SiteName, "original")
	}
	if cfg.Theme != "" {
		t.Errorf("config.Theme = %q, want %q (mutating the page must not mutate the caller's config)", cfg.Theme, "")
	}
}
