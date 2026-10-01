package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestFlagshipWorkflowPublishesOnlyTrustedMasterBuilds(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "website.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var workflow struct {
		On struct {
			Dispatch struct {
				Inputs map[string]struct {
					Type    string
					Default bool
				}
			} `yaml:"workflow_dispatch"`
		}
		Permissions map[string]string
		Env         map[string]string
		Jobs        map[string]struct {
			If    string
			Needs string
			Steps []struct {
				Uses string
				Run  string
				With map[string]string
			}
		}
	}
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		t.Fatal(err)
	}
	input, ok := workflow.On.Dispatch.Inputs["deploy"]
	if !ok || input.Type != "boolean" || input.Default {
		t.Error("production deployment must be an explicit boolean input defaulting to false")
	}
	deploy := workflow.Jobs["deploy"]
	for _, guard := range []string{
		"github.event_name == 'push'", "github.event_name == 'workflow_dispatch'", "inputs.deploy",
		"github.ref == 'refs/heads/master'",
	} {
		if !strings.Contains(deploy.If, guard) {
			t.Errorf("production deployment missing guard %q", guard)
		}
	}
	if deploy.Needs != "build" {
		t.Error("deployment must depend on a validated build")
	}
	if deploy.If != "github.ref == 'refs/heads/master' && (github.event_name == 'push' || (github.event_name == 'workflow_dispatch' && inputs.deploy))" {
		t.Error("deployment must reject PR events, feature branches, and unapproved manual runs")
	}
	if workflow.Permissions["contents"] != "read" || len(workflow.Permissions) != 1 {
		t.Error("workflow must not request unnecessary GitHub write permissions")
	}
	if workflow.Env["SITE_URL"] != "https://la-famille.filed.fyi" ||
		workflow.Env["PAGES_PROJECT"] != "la-famille-go" {
		t.Error("workflow must target the existing project and canonical domain")
	}
	var buildCommands strings.Builder
	for _, step := range workflow.Jobs["build"].Steps {
		buildCommands.WriteString(step.Run)
		for _, value := range step.With {
			if strings.Contains(value, "secrets.") {
				t.Error("build/PR validation must not receive deployment secrets")
			}
		}
	}
	for _, marker := range []string{
		"website.yaml", "publish-check", "--site-url", `"$RUNNER_TEMP/flagship-rag"`,
		`cp "$RUNNER_TEMP/flagship-rag/rag-content.md"`,
	} {
		if !strings.Contains(buildCommands.String(), marker) {
			t.Errorf("build missing publishing contract %q", marker)
		}
	}
	if strings.Contains(buildCommands.String(), "rag --output \"$GITHUB_WORKSPACE/public") {
		t.Error("website must not export system/config bundles directly into public")
	}
	foundPublisher := false
	for _, step := range deploy.Steps {
		if step.Uses != "cloudflare/wrangler-action@v3" {
			continue
		}
		foundPublisher = true
		if step.With["apiToken"] != "${{ secrets.CLOUDFLARE_API_TOKEN }}" ||
			step.With["accountId"] != "${{ secrets.CLOUDFLARE_ACCOUNT_ID }}" {
			t.Error("publisher must use existing secret references, never literal values")
		}
		if step.With["command"] != `pages deploy public --project-name "${{ env.PAGES_PROJECT }}" --branch "${{ steps.pages_project.outputs.production_branch }}"` {
			t.Error("publisher must use the existing project's detected production branch")
		}
	}
	if !foundPublisher {
		t.Error("workflow has no Cloudflare publisher")
	}
	foundLookup := false
	for _, step := range deploy.Steps {
		if strings.Contains(step.Run, ".github/scripts/website/production_branch.py") {
			foundLookup = true
		}
	}
	if !foundLookup {
		t.Error("workflow must detect the production branch without changing the project")
	}
}
