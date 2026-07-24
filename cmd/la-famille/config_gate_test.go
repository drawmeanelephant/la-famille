package main

import (
	"errors"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/tbuddy/la-famille/internal/config"
)

// --- unit level: the two pieces main() is now built from ---------------------

func TestLoadSiteConfig(t *testing.T) {
	writeSite := func(t *testing.T, body string) string {
		t.Helper()
		dir := t.TempDir()
		p := filepath.Join(dir, "config.yaml")
		if err := os.WriteFile(p, []byte(body), 0600); err != nil {
			t.Fatalf("write config: %v", err)
		}
		return p
	}

	t.Run("missing file uses defaults", func(t *testing.T) {
		cfg, err := loadSiteConfig(filepath.Join(t.TempDir(), "config.yaml"))
		if err != nil {
			t.Fatalf("loadSiteConfig(missing) = %v, want nil", err)
		}
		if !reflect.DeepEqual(cfg, config.DefaultConfig()) {
			t.Errorf("loadSiteConfig(missing) = %+v, want DefaultConfig()", cfg)
		}
	})

	t.Run("unparsable file yields no config", func(t *testing.T) {
		p := writeSite(t, "site_name: \"X\"\noutput_dir: content\nport: \"not-a-number\"\n")
		cfg, err := loadSiteConfig(p)
		if err == nil {
			t.Fatal("loadSiteConfig(unparsable) = nil error, want a failure")
		}
		if !reflect.DeepEqual(cfg, config.Config{}) {
			t.Errorf("loadSiteConfig(unparsable) returned a usable config: %+v", cfg)
		}
	})

	t.Run("invalid file yields no config", func(t *testing.T) {
		p := writeSite(t, "output_dir: \"/etc\"\n")
		cfg, err := loadSiteConfig(p)
		if err == nil {
			t.Fatal("loadSiteConfig(invalid) = nil error, want a failure")
		}
		if !reflect.DeepEqual(cfg, config.Config{}) {
			t.Errorf("loadSiteConfig(invalid) returned a usable config: %+v", cfg)
		}
	})

	t.Run("valid file is returned intact", func(t *testing.T) {
		p := writeSite(t, "siteurl: \"https://example.com/my-site\"\n")
		cfg, err := loadSiteConfig(p)
		if err != nil {
			t.Fatalf("loadSiteConfig(valid) = %v, want nil", err)
		}
		if cfg.SiteURL != "https://example.com/my-site" {
			t.Errorf("SiteURL = %q, want the value from the file", cfg.SiteURL)
		}
	})
}

func TestGuardUnusableConfigBlocksOnlyConfigConsumers(t *testing.T) {
	configErr := errors.New("config.yaml is broken")

	// Names are checked against the live command tree so a renamed or newly
	// added command cannot silently drift out of the blocked set.
	blocked := []string{"build", "serve", "rag", "check", "new", "ask", "tui"}
	allowed := []string{"init", "pr", "help"}

	root := setupRootCmd(config.Config{})
	byName := map[string]*cobra.Command{}
	for _, c := range root.Commands() {
		byName[c.Name()] = c
	}
	for _, name := range append(append([]string{}, blocked...), "pr") {
		if byName[name] == nil {
			t.Fatalf("command %q is not registered on the root command; update this test", name)
		}
	}

	for _, name := range blocked {
		if !requiresSiteConfig(byName[name]) {
			t.Errorf("requiresSiteConfig(%q) = false, want true", name)
		}
	}
	for _, name := range allowed {
		cmd := byName[name]
		if cmd == nil {
			// "help" is added lazily by cobra; synthesise it for the check.
			cmd = &cobra.Command{Use: name}
			root.AddCommand(cmd)
		}
		if requiresSiteConfig(cmd) {
			t.Errorf("requiresSiteConfig(%q) = true, want false", name)
		}
	}

	// A subcommand inherits its parent's exemption ("la-famille pr sync").
	prSync := &cobra.Command{Use: "sync"}
	byName["pr"].AddCommand(prSync)
	if requiresSiteConfig(prSync) {
		t.Error("requiresSiteConfig(pr sync) = true, want false")
	}

	// The guard must chain, not replace, the existing persistent hook.
	guarded := setupRootCmd(config.Config{})
	guardUnusableConfig(guarded, configErr)
	if guarded.PersistentPreRunE == nil {
		t.Fatal("guardUnusableConfig removed the persistent pre-run hook")
	}
	build, _, err := guarded.Find([]string{"build"})
	if err != nil {
		t.Fatalf("Find(build): %v", err)
	}
	if err := guarded.PersistentPreRunE(build, nil); err == nil {
		t.Error("guarded pre-run allowed `build` to proceed on a broken config")
	} else if !errors.Is(err, configErr) {
		t.Errorf("guarded pre-run error = %v, want it to wrap the config error", err)
	}
	initCmd, _, err := guarded.Find([]string{"init"})
	if err != nil {
		t.Fatalf("Find(init): %v", err)
	}
	if err := guarded.PersistentPreRunE(initCmd, nil); err != nil {
		t.Errorf("guarded pre-run blocked `init`: %v", err)
	}

	// With a usable config the guard must be inert.
	ungated := setupRootCmd(config.Config{})
	guardUnusableConfig(ungated, nil)
	if err := ungated.PersistentPreRunE(build, nil); err != nil {
		t.Errorf("guard with nil configErr blocked `build`: %v", err)
	}
}

// tui.go is the other caller of config.Load. It used to swap in a hardcoded
// partial config on a load error, which reported the wrong cause.
func TestTUIRefusesToRunOnAnUnusableConfig(t *testing.T) {
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(brokenYAML), 0600); err != nil {
		t.Fatalf("write config.yaml: %v", err)
	}
	t.Chdir(dir)

	err := tuiCmd.RunE(tuiCmd, nil)
	if err == nil {
		t.Fatal("tui accepted an unparsable config.yaml")
	}
	if !strings.Contains(err.Error(), "failed to load config.yaml") {
		t.Errorf("tui reported %q; want the actual load failure rather than a symptom of substituted defaults", err)
	}
}

// --- end to end: real binary, real exit codes, real output tree --------------

type gateSite struct {
	dir string
	exe string
}

func newGateSite(t *testing.T, exe, configYAML string) gateSite {
	t.Helper()
	dir := t.TempDir()
	if configYAML != "" {
		if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(configYAML), 0600); err != nil {
			t.Fatalf("write config.yaml: %v", err)
		}
	}
	if err := os.MkdirAll(filepath.Join(dir, "content"), 0755); err != nil {
		t.Fatalf("mkdir content: %v", err)
	}
	page := "---\ntitle: Home\n---\n\n# Home\n"
	if err := os.WriteFile(filepath.Join(dir, "content", "index.md"), []byte(page), 0600); err != nil {
		t.Fatalf("write index.md: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "templates"), 0755); err != nil {
		t.Fatalf("mkdir templates: %v", err)
	}
	layout := "<!DOCTYPE html>\n<html lang=\"en\"><body>{{.Content}}</body></html>\n"
	if err := os.WriteFile(filepath.Join(dir, "templates", "layout.html"), []byte(layout), 0600); err != nil {
		t.Fatalf("write layout.html: %v", err)
	}
	return gateSite{dir: dir, exe: exe}
}

func (s gateSite) run(t *testing.T, args ...string) (int, string) {
	t.Helper()
	// The binary under test is one this test compiled itself into t.TempDir,
	// and the arguments are fixed literals from the cases below.
	cmd := exec.Command(s.exe, args...) // #nosec G204
	cmd.Dir = s.dir
	out, err := cmd.CombinedOutput()
	code := 0
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		code = exitErr.ExitCode()
	} else if err != nil {
		t.Fatalf("running %v: %v", args, err)
	}
	return code, string(out)
}

func buildGateBinary(t *testing.T) string {
	t.Helper()
	exe := filepath.Join(t.TempDir(), "la-famille.bin")
	build := exec.Command("go", "build", "-o", exe, "../../cmd/la-famille")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("failed to build la-famille: %v\n%s", err, out)
	}
	return exe
}

const brokenYAML = "site_name: \"X\"\noutput_dir: content\nport: \"not-a-number\"\n"

func TestBrokenConfigFailsTheBuildInsteadOfSilentlyMisconfiguringIt(t *testing.T) {
	exe := buildGateBinary(t)
	site := newGateSite(t, exe, brokenYAML)

	code, out := site.run(t, "build")
	if code == 0 {
		t.Fatalf("build with an unparsable config.yaml exited 0:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(site.dir, "content", "index.md")); err != nil {
		t.Errorf("the half-applied output_dir was honoured and destroyed the source content: %v", err)
	}
	if _, err := os.Stat(filepath.Join(site.dir, "public")); !os.IsNotExist(err) {
		t.Errorf("a site was generated from a config that failed to load (stat public: %v)", err)
	}
}

func TestBrokenConfigStillAllowsTheCommandsThatRepairIt(t *testing.T) {
	exe := buildGateBinary(t)

	t.Run("init regenerates config.yaml", func(t *testing.T) {
		site := newGateSite(t, exe, brokenYAML)
		if code, out := site.run(t, "init"); code != 0 {
			t.Fatalf("`init` exited %d on a broken config.yaml; the repair path is blocked by the thing it repairs:\n%s", code, out)
		}
		cfg, err := config.Load(filepath.Join(site.dir, "config.yaml"))
		if err != nil {
			t.Fatalf("config.yaml is still unloadable after `init`: %v", err)
		}
		if err := cfg.Validate(); err != nil {
			t.Fatalf("config.yaml is still invalid after `init`: %v", err)
		}
		if code, out := site.run(t, "build"); code != 0 {
			t.Fatalf("build after `init` exited %d:\n%s", code, out)
		}
	})

	t.Run("help still works", func(t *testing.T) {
		site := newGateSite(t, exe, brokenYAML)
		code, out := site.run(t, "--help")
		if code != 0 {
			t.Fatalf("`--help` exited %d on a broken config.yaml:\n%s", code, out)
		}
		if !strings.Contains(out, "init") {
			t.Errorf("`--help` output does not list the repair command:\n%s", out)
		}
	})

	t.Run("an invalid but parsable config also leaves init reachable", func(t *testing.T) {
		site := newGateSite(t, exe, "port: 0\n")
		if code, out := site.run(t, "init"); code != 0 {
			t.Fatalf("`init` exited %d on an invalid config.yaml:\n%s", code, out)
		}
		if code, out := site.run(t, "build"); code != 0 {
			t.Fatalf("build after `init` exited %d:\n%s", code, out)
		}
	})
}

func TestUnreadableConfigDoesNotSilentlyFallBackToDefaults(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root: mode 0000 is still readable")
	}
	exe := buildGateBinary(t)
	site := newGateSite(t, exe, "siteurl: \"https://example.com/my-site\"\n")
	cfgPath := filepath.Join(site.dir, "config.yaml")
	if err := os.Chmod(cfgPath, 0000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(cfgPath, 0600) })

	code, out := site.run(t, "build")
	if code == 0 {
		t.Fatalf("build with an unreadable config.yaml exited 0, silently losing siteurl:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(site.dir, "public")); !os.IsNotExist(err) {
		t.Errorf("a site was generated from an unreadable config (stat public: %v)", err)
	}
}

// The paths that must keep working exactly as before: no config.yaml at all,
// and a good config.yaml whose siteurl has to survive into the output.
func TestUsableConfigsStillBuild(t *testing.T) {
	exe := buildGateBinary(t)

	t.Run("no config.yaml builds on defaults", func(t *testing.T) {
		site := newGateSite(t, exe, "")
		if code, out := site.run(t, "build"); code != 0 {
			t.Fatalf("build without config.yaml exited %d:\n%s", code, out)
		}
		if _, err := os.Stat(filepath.Join(site.dir, "public", "index.html")); err != nil {
			t.Fatalf("no site generated: %v", err)
		}
	})

	t.Run("siteurl survives into the generated site", func(t *testing.T) {
		site := newGateSite(t, exe, "siteurl: \"https://example.com/my-site\"\n")
		if code, out := site.run(t, "build"); code != 0 {
			t.Fatalf("build exited %d:\n%s", code, out)
		}
		sitemap, err := os.ReadFile(filepath.Join(site.dir, "public", "sitemap.xml"))
		if err != nil {
			t.Fatalf("read sitemap.xml: %v", err)
		}
		if !strings.Contains(string(sitemap), "https://example.com/my-site/") {
			t.Errorf("siteurl lost from sitemap.xml:\n%s", sitemap)
		}
	})
}
