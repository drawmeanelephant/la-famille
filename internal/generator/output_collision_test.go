package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/config"
)

func setupCollisionSite(t *testing.T, files map[string]string) config.Config {
	t.Helper()
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	templateDir := filepath.Join(tempDir, "templates")
	templatePath := filepath.Join(templateDir, "layout.html")

	if err := os.MkdirAll(templateDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(templatePath, []byte("{{.Content}}"), 0600); err != nil {
		t.Fatal(err)
	}
	for name, body := range files {
		path := filepath.Join(contentDir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}

	cfg := config.DefaultConfig()
	cfg.ContentDir = contentDir
	cfg.OutputDir = filepath.Join(tempDir, "public")
	cfg.Template = templatePath
	cfg.ProjectRoot = tempDir
	return cfg
}

func readOutput(t *testing.T, cfg config.Config, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(cfg.OutputDir, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(data)
}

// An ignored slug has to be ignored by the collision guard too, otherwise two
// pages that really do render to the same file pass validation and one of them
// is silently overwritten by whichever worker writes last.
func TestBuild_IgnoredSlugCollisionIsDetected(t *testing.T) {
	cfg := setupCollisionSite(t, map[string]string{
		"alpha.md":       "---\nslug: \"renamed.v2\"\n---\nALPHA_MD_BODY\n",
		"alpha/index.md": "---\ntitle: Alpha Index\n---\nALPHA_INDEX_BODY\n",
	})

	_, err := Build(cfg)
	if err == nil {
		t.Fatal("Build() error = nil, want a collision for alpha.md and alpha/index.md")
	}
	if !strings.Contains(err.Error(), "output path collision") {
		t.Fatalf("Build() error = %v, want an output path collision", err)
	}
	if !strings.Contains(err.Error(), "alpha.md") || !strings.Contains(err.Error(), "alpha/index.md") {
		t.Fatalf("Build() error = %v, want both colliding sources named", err)
	}
}

// The mirror case: two pages sharing the same invalid slug never share an
// output path, because the slug is discarded before rendering. Failing there
// reports a collision the generator would never create.
func TestBuild_IgnoredSlugIsNotACollision(t *testing.T) {
	cfg := setupCollisionSite(t, map[string]string{
		"one.md": "---\nslug: my.page\n---\nONE_BODY\n",
		"two.md": "---\nslug: my.page\n---\nTWO_BODY\n",
	})

	if _, err := Build(cfg); err != nil {
		t.Fatalf("Build() error = %v, want success: the invalid slug is ignored for both pages", err)
	}

	for rel, want := range map[string]string{
		"one/index.html": "ONE_BODY",
		"two/index.html": "TWO_BODY",
	} {
		if got := readOutput(t, cfg, rel); !strings.Contains(got, want) {
			t.Errorf("%s = %q, want it to contain %q", rel, got, want)
		}
	}
}

// Case-insensitive filesystems (macOS, Windows) treat Foo/ and foo/ as one
// directory, so two pages differing only in case race for the same file.
// The probe is pinned to "insensitive" so the behavior is asserted on every
// host, CI included.
func TestBuild_CaseOnlyCollisionIsDetected(t *testing.T) {
	pinCaseSensitivity(t, false)
	cfg := setupCollisionSite(t, map[string]string{
		"one.md": "---\nslug: Foo\n---\nUPPER_BODY\n",
		"two.md": "---\nslug: foo\n---\nlower_body\n",
	})

	_, err := Build(cfg)
	if err == nil {
		t.Fatal("Build() error = nil, want a collision for slugs Foo and foo")
	}
	if !strings.Contains(err.Error(), "output path collision") {
		t.Fatalf("Build() error = %v, want an output path collision", err)
	}
	if !strings.Contains(err.Error(), "one.md") || !strings.Contains(err.Error(), "two.md") {
		t.Fatalf("Build() error = %v, want both colliding sources named", err)
	}
	if !strings.Contains(err.Error(), "case-insensitive") {
		t.Fatalf("Build() error = %v, want the case-folding reason explained", err)
	}
}

// On a case-sensitive filesystem the same two pages are genuinely distinct
// files, so the build must publish both and warn instead of refusing.
func TestBuild_CaseOnlyCollisionAllowedOnCaseSensitiveFS(t *testing.T) {
	pinCaseSensitivity(t, true)
	cfg := setupCollisionSite(t, map[string]string{
		"one.md": "---\nslug: Foo\n---\nUPPER_BODY\n",
		"two.md": "---\nslug: foo\n---\nlower_body\n",
	})

	result, err := Build(cfg)
	if err != nil {
		t.Fatalf("Build() error = %v, want success on a case-sensitive filesystem", err)
	}
	warned := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "differ only in case") || strings.Contains(w, "case-insensitive filesystem") {
			warned = true
		}
	}
	if !warned {
		t.Errorf("Warnings = %v, want a case-only collision warning", result.Warnings)
	}
	// The registry admits both writers, which is the point of the test — but
	// whether the staging directory can physically hold both pages depends on
	// the host filesystem. Assert the distinct outputs only where the host can
	// represent them.
	if !probeCaseSensitivity(t.TempDir()) {
		return
	}
	if got := readOutput(t, cfg, "Foo/index.html"); !strings.Contains(got, "UPPER_BODY") {
		t.Errorf("Foo/index.html = %q, want the upper-case page", got)
	}
	if got := readOutput(t, cfg, "foo/index.html"); !strings.Contains(got, "lower_body") {
		t.Errorf("foo/index.html = %q, want the lower-case page", got)
	}
}

// The same policy covers assets: a downloaded theme's assets/CSS/ next to the
// generator's assets/css/ is an ordinary layout on a case-sensitive host.
func TestBuild_CaseOnlyAssetCollisionHonoursFilesystem(t *testing.T) {
	for _, tc := range []struct {
		name          string
		caseSensitive bool
		wantErr       bool
	}{
		{name: "insensitive filesystem refuses", caseSensitive: false, wantErr: true},
		{name: "sensitive filesystem warns and builds", caseSensitive: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pinCaseSensitivity(t, tc.caseSensitive)
			tempDir := t.TempDir()
			contentDir := filepath.Join(tempDir, "content")
			assetDir := filepath.Join(tempDir, "assets")
			templateDir := filepath.Join(tempDir, "templates")
			outputDir := filepath.Join(tempDir, "public")

			for _, d := range []string{filepath.Join(contentDir, "assets", "help"), filepath.Join(assetDir, "help"), templateDir} {
				if err := os.MkdirAll(d, 0o755); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(filepath.Join(templateDir, "layout.html"), []byte("{{.Content}}"), 0o600); err != nil {
				t.Fatal(err)
			}
			// Renders to assets/help/index.html.
			page := []byte("---\ntitle: Help Page\n---\n# AUTHORED_PAGE_CONTENT\n")
			if err := os.WriteFile(filepath.Join(contentDir, "assets", "help", "index.md"), page, 0o600); err != nil {
				t.Fatal(err)
			}
			// Copies to assets/help/INDEX.html: same file on an insensitive FS.
			if err := os.WriteFile(filepath.Join(assetDir, "help", "INDEX.html"), []byte("ASSET_BYTES"), 0o600); err != nil {
				t.Fatal(err)
			}

			cfg := config.DefaultConfig()
			cfg.ContentDir = contentDir
			cfg.AssetDir = assetDir
			cfg.OutputDir = outputDir
			cfg.Template = filepath.Join(templateDir, "layout.html")
			cfg.ProjectRoot = tempDir

			result, err := Build(cfg)
			if tc.wantErr {
				if err == nil {
					t.Fatal("Build() error = nil, want a case-only collision error")
				}
				if !strings.Contains(err.Error(), "case-insensitive") {
					t.Fatalf("Build() error = %v, want the case reason explained", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Build() error = %v, want success on a case-sensitive filesystem", err)
			}
			for _, w := range result.Warnings {
				if strings.Contains(w, "case-insensitive filesystem") {
					return
				}
			}
			t.Errorf("Warnings = %v, want a case-only collision warning", result.Warnings)
		})
	}
}

// pinCaseSensitivity forces the filesystem-sensitivity probe to either answer
// for the rest of the test, so collision behavior is asserted on every host.
func pinCaseSensitivity(t *testing.T, sensitive bool) {
	t.Helper()
	old := detectCaseSensitivity
	detectCaseSensitivity = func(string) bool { return sensitive }
	t.Cleanup(func() { detectCaseSensitivity = old })
}

// Taxonomy listings are written before the content workers run, so a content
// file mapping onto one of them destroys the listing without a warning.
func TestBuild_TaxonomyIndexCollisionIsDetected(t *testing.T) {
	cfg := setupCollisionSite(t, map[string]string{
		"post.md":       "---\ntitle: Post\ntags: [go]\n---\nPOST_BODY\n",
		"tags/index.md": "---\ntitle: Hand Written Tags\n---\nHANDWRITTEN_TAGS_INDEX\n",
	})

	_, err := Build(cfg)
	if err == nil {
		t.Fatal("Build() error = nil, want a collision between content/tags/index.md and the generated tag index")
	}
	if !strings.Contains(err.Error(), "output path collision") {
		t.Fatalf("Build() error = %v, want an output path collision", err)
	}
	if !strings.Contains(err.Error(), "tags/index.md") {
		t.Fatalf("Build() error = %v, want the colliding content file named", err)
	}
}

// A term page collides the same way as the taxonomy index page.
func TestBuild_TaxonomyTermCollisionIsDetected(t *testing.T) {
	cfg := setupCollisionSite(t, map[string]string{
		"post.md":    "---\ntitle: Post\ntags: [go]\n---\nPOST_BODY\n",
		"tags/go.md": "---\ntitle: Hand Written Go\n---\nHANDWRITTEN_GO_PAGE\n",
	})

	_, err := Build(cfg)
	if err == nil {
		t.Fatal("Build() error = nil, want a collision between content/tags/go.md and the generated go term page")
	}
	if !strings.Contains(err.Error(), "output path collision") {
		t.Fatalf("Build() error = %v, want an output path collision", err)
	}
}

// Stubs are generated after the taxonomy listings and write with os.Create, so
// a single dangling link used to replace the tag index with "Under
// Construction" at exit 0.
func TestBuild_StubDoesNotOverwriteTaxonomyPage(t *testing.T) {
	cfg := setupCollisionSite(t, map[string]string{
		"post.md":  "---\ntitle: Post\ntags: [go]\n---\nPOST_BODY\n",
		"index.md": "---\ntitle: Index\n---\nSee [all tags](tags.md).\n",
	})

	if _, err := Build(cfg); err != nil {
		t.Fatalf("Build() error = %v, want success: a dangling link is not a fatal error", err)
	}

	got := readOutput(t, cfg, "tags/index.html")
	if strings.Contains(got, "Under Construction") {
		t.Errorf("tags/index.html = %q, want the generated tag listing, not a stub", got)
	}
	if !strings.Contains(got, "go") {
		t.Errorf("tags/index.html = %q, want it to list the go term", got)
	}
}

// The same hazard with a rendered page as the victim: content/foo/index.md and
// a dangling link to foo.md both target foo/index.html.
func TestBuild_StubDoesNotOverwriteRenderedPage(t *testing.T) {
	cfg := setupCollisionSite(t, map[string]string{
		"foo/index.md": "---\ntitle: Foo\n---\nREAL_FOO_BODY\n",
		"index.md":     "---\ntitle: Index\n---\nSee [foo](foo.md).\n",
	})

	if _, err := Build(cfg); err != nil {
		t.Fatalf("Build() error = %v, want success", err)
	}

	got := readOutput(t, cfg, "foo/index.html")
	if !strings.Contains(got, "REAL_FOO_BODY") {
		t.Errorf("foo/index.html = %q, want the rendered page, not a stub", got)
	}
}

// Two dangling links can name the same output file: ghost.md and
// ghost/index.md both render to ghost/index.html. Whoever claims it first keeps
// it, so the surviving page is a function of the source rather than of which
// stub happened to be written last.
func TestBuild_StubDoesNotOverwriteAnotherStub(t *testing.T) {
	cfg := setupCollisionSite(t, map[string]string{
		"alpha.md": "---\ntitle: Alpha\n---\n[bare](ghost.md)\n",
		"beta.md":  "---\ntitle: Beta\n---\n[nested](ghost/index.md)\n",
	})

	if _, err := Build(cfg); err != nil {
		t.Fatalf("Build() error = %v, want success", err)
	}

	got := readOutput(t, cfg, "ghost/index.html")
	if !strings.Contains(got, "Under Construction") {
		t.Fatalf("ghost/index.html = %q, want a stub page", got)
	}
	// Stubs are generated in sorted order, so "ghost.md" claims the path and
	// keeps its referrer list. Before the claim, "ghost/index.md" was written
	// second and replaced it.
	if !strings.Contains(got, "alpha.md") {
		t.Errorf("ghost/index.html = %q, want the first claimant's referrer alpha.md", got)
	}
	if strings.Contains(got, "beta.md") {
		t.Errorf("ghost/index.html = %q, want it not replaced by the later stub for ghost/index.md", got)
	}
}

// TestBuild_AssetDoesNotOverwriteRenderedPage covers the last writer into the
// output tree that had no ownership check. Assets are copied after the pages
// are rendered, so an asset whose destination matches a rendered page silently
// replaced it: the site published the asset bytes while search.json, graph.json
// and meta.json all still described the page.
func TestBuild_AssetDoesNotOverwriteRenderedPage(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	assetDir := filepath.Join(tempDir, "assets")
	templateDir := filepath.Join(tempDir, "templates")
	outputDir := filepath.Join(tempDir, "public")

	for _, d := range []string{filepath.Join(contentDir, "assets", "help"), filepath.Join(assetDir, "help"), templateDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(templateDir, "layout.html"), []byte("{{.Content}}"), 0o600); err != nil {
		t.Fatal(err)
	}
	// Renders to assets/help/index.html.
	page := []byte("---\ntitle: Help Page\n---\n# AUTHORED_PAGE_CONTENT\n")
	if err := os.WriteFile(filepath.Join(contentDir, "assets", "help", "index.md"), page, 0o600); err != nil {
		t.Fatal(err)
	}
	// Copies to the very same place.
	if err := os.WriteFile(filepath.Join(assetDir, "help", "index.html"), []byte("ASSET_BYTES"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	cfg.ContentDir = contentDir
	cfg.AssetDir = assetDir
	cfg.OutputDir = outputDir
	cfg.Template = filepath.Join(templateDir, "layout.html")
	cfg.ProjectRoot = tempDir

	_, err := Build(cfg)
	if err == nil {
		t.Fatal("expected a collision between an asset and a rendered page to fail the build")
	}
	if !strings.Contains(err.Error(), "collision") {
		t.Errorf("error should name the collision, got: %v", err)
	}

	// The build stages its output, so a failure must leave nothing published
	// rather than a tree where the asset won.
	if body, readErr := os.ReadFile(filepath.Join(outputDir, "assets", "help", "index.html")); readErr == nil {
		if strings.Contains(string(body), "ASSET_BYTES") {
			t.Error("the asset was published over the rendered page")
		}
	}
}

// TestOutputClaimsStillRejectsAnExactDuplicate keeps the relaxation narrow: the
// case rule changed, the collision rule did not.
func TestOutputClaimsStillRejectsAnExactDuplicate(t *testing.T) {
	claims := newOutputClaims(t.TempDir(), 2, false)

	if _, ok := claims.claim("the page \"a.md\"", "docs/index.html"); !ok {
		t.Fatal("first claim should succeed")
	}
	previous, ok := claims.claim("the asset \"docs/index.html\"", "docs/index.html")
	if ok {
		t.Fatal("an exact duplicate path must always collide")
	}
	if previous.source == "" {
		t.Error("the collision should name the previous owner")
	}
}

// TestOutputClaimsAllowsCaseOnlyOnCaseSensitiveFS pins the registry-level rule:
// on a case-sensitive filesystem the second, case-differing writer is admitted
// and recorded as a warning.
func TestOutputClaimsAllowsCaseOnlyOnCaseSensitiveFS(t *testing.T) {
	claims := newOutputClaims(t.TempDir(), 2, true)

	if _, ok := claims.claim("the page \"one.md\"", "Foo/index.html"); !ok {
		t.Fatal("first claim should succeed")
	}
	if _, ok := claims.claim("the page \"two.md\"", "foo/index.html"); !ok {
		t.Fatal("case-differing claim should succeed on a case-sensitive filesystem")
	}
	if warns := claims.Warnings(); len(warns) != 1 {
		t.Fatalf("Warnings() = %v, want exactly one warning", warns)
	}
	// A third writer must still collide against the earlier owners, not slip
	// past a one-entry map.
	if _, ok := claims.claim("the asset \"docs/foo/index.html\"", "foo/index.html"); ok {
		t.Fatal("an exact duplicate of an admitted case-differing owner must collide")
	}
}
