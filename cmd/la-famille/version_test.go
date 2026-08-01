package main

import (
	"bytes"
	"encoding/json"
	"runtime"
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/config"
)

func TestWriteBuildInfoTextAndJSON(t *testing.T) {
	var textOut bytes.Buffer
	if err := writeBuildInfo(&textOut, false); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"La Famille", "commit:", "built:", "target: " + runtime.GOOS + "/" + runtime.GOARCH, "go: " + runtime.Version()} {
		if !strings.Contains(textOut.String(), want) {
			t.Errorf("text version output missing %q: %s", want, textOut.String())
		}
	}

	var jsonOut bytes.Buffer
	if err := writeBuildInfo(&jsonOut, true); err != nil {
		t.Fatal(err)
	}
	var got buildInfo
	if err := json.Unmarshal(jsonOut.Bytes(), &got); err != nil {
		t.Fatalf("JSON version output: %v\n%s", err, jsonOut.String())
	}
	if got.Target != runtime.GOOS+"/"+runtime.GOARCH || got.GoVersion != runtime.Version() {
		t.Errorf("build info = %+v", got)
	}
}

func TestVersionDoesNotRequireConfig(t *testing.T) {
	root := setupRootCmd(config.DefaultConfig())
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"--version", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("version command: %v", err)
	}
	if !strings.Contains(out.String(), `"version"`) {
		t.Errorf("version JSON output = %s", out.String())
	}
}
