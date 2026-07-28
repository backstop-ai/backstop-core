package validate_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/artifact"
	"github.com/backstop-ai/backstop-core/pkg/schema"
	"github.com/backstop-ai/backstop-core/pkg/validate"
)

// terminalRepoRoot walks up from cwd to the directory containing go.mod.
func terminalRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find repo root (no go.mod)")
		}
		dir = parent
	}
}

// loadTerminalSchema loads the real artifact schema for the given type/version.
func loadTerminalSchema(t *testing.T, typ, ver string) *schema.Schema {
	t.Helper()
	root := terminalRepoRoot(t)
	artifactsRoot := filepath.Join(root, "artifacts")
	schemaPath := filepath.Join(artifactsRoot, typ, ver, "schema.json")
	sch, err := schema.LoadArtifactSchema(schemaPath, artifactsRoot)
	if err != nil {
		t.Fatalf("LoadArtifactSchema(%s/%s): %v", typ, ver, err)
	}
	return sch
}

// parseTerminalFixture parses a fixture under pkg/validate/testdata/terminal/.
func parseTerminalFixture(t *testing.T, name string) *artifact.ParsedArtifact {
	t.Helper()
	root := terminalRepoRoot(t)
	path := filepath.Join(root, "pkg", "validate", "testdata", "terminal", name)
	art, err := artifact.ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile(%s): %v", name, err)
	}
	return art
}

// hasViolationRule reports whether the result contains the given rule.
func hasViolationRule(result validate.ValidationResult, rule string) bool {
	for _, v := range result.Violations {
		if v.Rule == rule {
			return true
		}
	}
	return false
}

func TestValidateSpec_AcceptsTerminalStates(t *testing.T) {
	sch := loadTerminalSchema(t, "spec", "v1")

	// deprecated
	dep := parseTerminalFixture(t, "spec-deprecated.spec.md")
	res := validate.Spec(dep, sch)
	if hasViolationRule(res, "spec/invalid-status") {
		t.Errorf("deprecated spec raised spec/invalid-status: %v", res.Violations)
	}

	// canceled (mutate the parsed deprecated fixture)
	canceled := parseTerminalFixture(t, "spec-deprecated.spec.md")
	canceled.Metadata["status"] = "canceled"
	canceled.Frontmatter["status"] = "canceled"
	res = validate.Spec(canceled, sch)
	if hasViolationRule(res, "spec/invalid-status") {
		t.Errorf("canceled spec raised spec/invalid-status: %v", res.Violations)
	}

	// replaced WITH a valid replaced-by validates without invalid-status
	replaced := parseTerminalFixture(t, "spec-deprecated.spec.md")
	replaced.Metadata["status"] = "replaced"
	replaced.Frontmatter["status"] = "replaced"
	replaced.Frontmatter["replaced-by"] = "SPEC-902"
	res = validate.Spec(replaced, sch)
	if hasViolationRule(res, "spec/invalid-status") {
		t.Errorf("replaced spec raised spec/invalid-status: %v", res.Violations)
	}
}

func TestValidatePlan_AcceptsRetirementStates(t *testing.T) {
	sch := loadTerminalSchema(t, "plan", "v1")

	// replaced with valid replaced-by
	replaced := parseTerminalFixture(t, "plan-replaced.plan.yml")
	res := validate.Plan(replaced, sch)
	if hasViolationRule(res, "plan/invalid-status") {
		t.Errorf("replaced plan raised plan/invalid-status: %v", res.Violations)
	}

	// canceled
	canceled := parseTerminalFixture(t, "plan-replaced.plan.yml")
	canceled.Frontmatter["status"] = "canceled"
	delete(canceled.Frontmatter, "replaced-by")
	res = validate.Plan(canceled, sch)
	if hasViolationRule(res, "plan/invalid-status") {
		t.Errorf("canceled plan raised plan/invalid-status: %v", res.Violations)
	}
}

func TestValidateBundle_AcceptsDeliveredAndRetirementMaturity(t *testing.T) {
	sch := loadTerminalSchema(t, "bundle", "v1")

	cases := []struct {
		maturity   string
		replacedBy string
	}{
		{"delivered", ""},
		{"canceled", ""},
		{"deprecated", ""},
		{"replaced", "BUNDLE-902"},
	}
	for _, c := range cases {
		art := parseTerminalFixture(t, "bundle-delivered.bundle.md")
		status := art.Frontmatter["status"].(map[string]interface{})
		status["maturity"] = c.maturity
		if c.replacedBy != "" {
			art.Frontmatter["replaced-by"] = c.replacedBy
		}
		res := validate.Bundle(art, sch)
		if hasViolationRule(res, "bundle/maturity-enum") {
			t.Errorf("maturity %q raised bundle/maturity-enum: %v", c.maturity, res.Violations)
		}
	}
}

func TestValidateDirective_AcceptsRetirementStates(t *testing.T) {
	sch := loadTerminalSchema(t, "directive", "v1")

	// canceled
	canceled := parseTerminalFixture(t, "directive-canceled.directive.md")
	res := validate.Directive(canceled, sch)
	if hasViolationRule(res, "directive/invalid-status") {
		t.Errorf("canceled directive raised directive/invalid-status: %v", res.Violations)
	}

	// replaced with valid replaced-by
	replaced := parseTerminalFixture(t, "directive-canceled.directive.md")
	dir := replaced.Frontmatter["directive"].(map[string]interface{})
	dir["status"] = "replaced"
	replaced.Frontmatter["replaced-by"] = "DIR-902"
	res = validate.Directive(replaced, sch)
	if hasViolationRule(res, "directive/invalid-status") {
		t.Errorf("replaced directive raised directive/invalid-status: %v", res.Violations)
	}
}

func TestValidateIssue_AcceptsRetirementStatesKeepsClosed(t *testing.T) {
	sch := loadTerminalSchema(t, "issue", "v1")

	// replaced with valid replaced-by
	replaced := parseTerminalFixture(t, "issue-replaced.issue.md")
	res := validate.Issue(replaced, sch)
	if hasViolationRule(res, "issue/status-enum") {
		t.Errorf("replaced issue raised issue/status-enum: %v", res.Violations)
	}

	// canceled
	canceled := parseTerminalFixture(t, "issue-replaced.issue.md")
	iss := canceled.Frontmatter["issue"].(map[string]interface{})
	iss["status"] = "canceled"
	delete(canceled.Frontmatter, "replaced-by")
	res = validate.Issue(canceled, sch)
	if hasViolationRule(res, "issue/status-enum") {
		t.Errorf("canceled issue raised issue/status-enum: %v", res.Violations)
	}

	// closed still validates as a legal status
	closed := parseTerminalFixture(t, "issue-replaced.issue.md")
	iss2 := closed.Frontmatter["issue"].(map[string]interface{})
	iss2["status"] = "closed"
	iss2["closed"] = "2026-06-27"
	delete(closed.Frontmatter, "replaced-by")
	res = validate.Issue(closed, sch)
	if hasViolationRule(res, "issue/status-enum") {
		t.Errorf("closed issue raised issue/status-enum: %v", res.Violations)
	}
}
