package github

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// goRunModulePath matches a `go run <module path>@<version>` invocation.
var goRunModulePath = regexp.MustCompile(`go run\s+([^\s"']+)@[^\s"']+`)

// The composite action must be able to build this module. Fetching it through the
// module proxy under the repository path fails whenever that path differs from the
// module path declared in go.mod, so the action builds from its own checkout instead.
func TestCompositeActionBuildsThisModule(t *testing.T) {
	modData, err := os.ReadFile("../../go.mod")
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	var modulePath string
	for _, line := range strings.Split(string(modData), "\n") {
		if strings.HasPrefix(line, "module ") {
			modulePath = strings.TrimSpace(strings.TrimPrefix(line, "module "))
			break
		}
	}
	if modulePath == "" {
		t.Fatal("go.mod does not declare a module path")
	}

	actionData, err := os.ReadFile("../../action.yml")
	if err != nil {
		t.Fatalf("read action.yml: %v", err)
	}
	action := string(actionData)

	for _, match := range goRunModulePath.FindAllStringSubmatch(action, -1) {
		pkg := match[1]
		if !strings.HasPrefix(pkg, modulePath) {
			t.Errorf("action.yml runs %q, which cannot resolve: go.mod declares module %q", pkg, modulePath)
		}
	}

	if !strings.Contains(action, "$GITHUB_ACTION_PATH") {
		t.Error("action.yml does not build la-famille from the action's own checkout ($GITHUB_ACTION_PATH)")
	}
}
