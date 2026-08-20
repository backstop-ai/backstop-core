package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// fakeGhAnsweringRunsEndpoint installs a fake `gh` on PATH that answers any
// actions/runs query with payload. The arm is a GLOB, not the exact endpoint, so
// these tests stay sensitive to WHICH RUN is selected rather than to the query
// string — the window width is CLM-004's separate channel.
func fakeGhAnsweringRunsEndpoint(t *testing.T, payload string) {
	t.Helper()
	binDir := t.TempDir()
	writeFakeGh(t, filepath.Join(binDir, "gh"), `#!/bin/sh
if [ "$1" != "api" ]; then
  echo "unexpected command: $*" >&2
  exit 1
fi
case "$2" in
  *actions/runs*)
    printf '`+payload+`'
    ;;
  *)
    echo "unexpected endpoint: $2" >&2
    exit 1
    ;;
esac
`)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestResolveLatestSuccessfulMainRun_RejectsNewerForeignWorkflowRun(t *testing.T) {
	projectRoot := t.TempDir()
	fakeGhAnsweringRunsEndpoint(t, `{"workflow_runs":[`+
		`{"id":7,"name":"pages build and deployment","conclusion":"success","head_branch":"main"},`+
		`{"id":42,"name":"CI","conclusion":"success","head_branch":"main"}]}`)

	runID, err := resolveLatestSuccessfulMainRun(projectRoot, "owner/repo")
	if err != nil {
		t.Fatalf("resolve run: %v", err)
	}
	if runID != 42 {
		t.Fatalf("selected run id = %d, want 42; a newer foreign-workflow run was not rejected", runID)
	}
}

func TestResolveLatestSuccessfulMainRun_AcceptsExactlyTheNameCIWorkflowDeclares(t *testing.T) {
	workflowPath := filepath.Join(repoRoot(t), ".github", "workflows", "ci.yml")
	data, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("read %s: %v", workflowPath, err)
	}
	var workflow struct {
		Name string `yaml:"name"`
	}
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		t.Fatalf("parse %s: %v", workflowPath, err)
	}
	if strings.TrimSpace(workflow.Name) == "" {
		t.Fatalf("%s declares no `name:`; this test cannot pin the selector against an empty value", workflowPath)
	}

	projectRoot := t.TempDir()
	fakeGhAnsweringRunsEndpoint(t, `{"workflow_runs":[`+
		`{"id":7,"name":"pages build and deployment","conclusion":"success","head_branch":"main"},`+
		`{"id":42,"name":"`+workflow.Name+`","conclusion":"success","head_branch":"main"}]}`)

	runID, err := resolveLatestSuccessfulMainRun(projectRoot, "owner/repo")
	if err != nil {
		t.Fatalf("resolve run: %v", err)
	}
	if runID != 42 {
		t.Fatalf("selected run id = %d, want 42; the selector did not accept the name %q that %s declares", runID, workflow.Name, workflowPath)
	}
}

func TestResolveLatestSuccessfulMainRun_SelectionMissNamesWorkflowAndRejectedCandidates(t *testing.T) {
	projectRoot := t.TempDir()
	fakeGhAnsweringRunsEndpoint(t, `{"workflow_runs":[`+
		`{"id":7,"name":"pages build and deployment","conclusion":"success","head_branch":"main"},`+
		`{"id":8,"name":"pages build and deployment","conclusion":"success","head_branch":"main"}]}`)

	runID, err := resolveLatestSuccessfulMainRun(projectRoot, "owner/repo")
	if err == nil {
		t.Fatalf("expected a selection miss, got run id %d and a nil error", runID)
	}
	for _, want := range []string{"CI", "pages build and deployment"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("selection miss message %q does not name %q", err.Error(), want)
		}
	}
}

func TestResolveLatestSuccessfulMainRun_QueriesAHundredRunWindow(t *testing.T) {
	projectRoot := t.TempDir()
	binDir := t.TempDir()
	recordPath := filepath.Join(t.TempDir(), "endpoints.txt")
	writeFakeGh(t, filepath.Join(binDir, "gh"), `#!/bin/sh
echo "$2" >> "`+recordPath+`"
printf '{"workflow_runs":[{"id":42,"name":"CI","conclusion":"success","head_branch":"main"}]}'
`)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	if _, err := resolveLatestSuccessfulMainRun(projectRoot, "owner/repo"); err != nil {
		t.Fatalf("resolve run: %v", err)
	}
	recorded, err := os.ReadFile(recordPath)
	if err != nil {
		t.Fatalf("read recorded endpoints: %v", err)
	}
	if !strings.Contains(string(recorded), "per_page=100") {
		t.Fatalf("queried endpoint %q does not request a 100-run window", strings.TrimSpace(string(recorded)))
	}
}
