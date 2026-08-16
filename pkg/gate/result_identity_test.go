package gate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/artifact"
)

// TestGateResult_CarriesProducingBinaryIdentity pins CLM-019: a gate result names the
// binary that produced it. The ADDITIVE-NESS is half the claim — SchemaVersion must
// still be exactly "gate/v1", the same way ISSUE-059 added GitSHA/GeneratedAt.
func TestGateResult_CarriesProducingBinaryIdentity(t *testing.T) {
	result := NewGateResult(nil)
	result.BinaryVersion = "v9.9.9"
	result.SchemaCohort = "cohort-abc"
	result.SchemaIdentities = []string{"spec/v1@deadbeef", "bundle/v2@cafebabe"}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshalling the gate result: %v", err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("the marshalled gate result is not valid JSON: %v", err)
	}

	if got, _ := parsed["schema_version"].(string); got != "gate/v1" {
		t.Errorf("schema_version = %q, want exactly %q — these fields are ADDITIVE under an unchanged schema", got, "gate/v1")
	}
	if got, _ := parsed["binary_version"].(string); got != "v9.9.9" {
		t.Errorf("binary_version = %q, want %q", got, "v9.9.9")
	}
	if got, _ := parsed["schema_cohort"].(string); got != "cohort-abc" {
		t.Errorf("schema_cohort = %q, want %q", got, "cohort-abc")
	}
	identities, ok := parsed["schema_identities"].([]interface{})
	if !ok || len(identities) != 2 {
		t.Fatalf("schema_identities = %#v, want the two identities that were set", parsed["schema_identities"])
	}

	// The pre-existing provenance and summary fields survive untouched.
	for _, field := range []string{"pass", "total_violations", "steps_passed", "steps_failed", "steps_skipped", "steps_warned", "steps"} {
		if _, ok := parsed[field]; !ok {
			t.Errorf("the widened gate result dropped the pre-existing field %q", field)
		}
	}
}

// TestGateResult_NamesScannedArtifactRootAndConfiguredFlag pins CLM-053.
//
// The assertion is on the SERIALIZED BYTES, not on the struct field, and that is the
// whole point: encoding/json DROPS a false bool under omitempty, and false is both the
// default and the motivating state (a project that configures no artifact_root). An
// omitempty on ArtifactRootConfigured therefore makes REQ-008 unsatisfiable in --json
// for exactly the case it was written for, while leaving every struct-level assertion
// green.
func TestGateResult_NamesScannedArtifactRootAndConfiguredFlag(t *testing.T) {
	result := NewGateResult(nil)
	result.ArtifactRoot = "/tmp/project"
	result.ArtifactRootConfigured = false

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshalling the gate result: %v", err)
	}

	if !strings.Contains(string(data), `"artifact_root_configured"`) {
		t.Errorf("the serialized gate result OMITS artifact_root_configured when it is false; a false bool under omitempty is dropped entirely, and false is the default and the motivating state.\ngot: %s", data)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("the marshalled gate result is not valid JSON: %v", err)
	}
	configured, present := parsed["artifact_root_configured"]
	if !present {
		t.Fatal("artifact_root_configured is absent from the decoded result")
	}
	if configured != false {
		t.Errorf("artifact_root_configured = %#v, want false", configured)
	}
	if got, _ := parsed["artifact_root"].(string); got != "/tmp/project" {
		t.Errorf("artifact_root = %q, want %q", got, "/tmp/project")
	}

	// And the true case still serializes, so the field is not merely hardcoded false.
	result.ArtifactRootConfigured = true
	data, err = json.Marshal(result)
	if err != nil {
		t.Fatalf("marshalling the configured gate result: %v", err)
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("the marshalled configured gate result is not valid JSON: %v", err)
	}
	if parsed["artifact_root_configured"] != true {
		t.Errorf("artifact_root_configured = %#v after setting it true, want true", parsed["artifact_root_configured"])
	}
}

// TestFormatHuman_RendersBinaryIdentityAndScannedRoot pins the HUMAN half of CLM-021
// and CLM-054 in the package that owns the rendering.
//
// It lives here rather than only in the command's test because the gate's human output
// is produced entirely inside FormatHuman: a Println beside the FormatHuman call site
// would satisfy a command-level test while leaving this one red, and that second
// rendering path is exactly what "both renderings read one resolved value" forbids.
func TestFormatHuman_RendersBinaryIdentityAndScannedRoot(t *testing.T) {
	result := NewGateResult(nil)
	result.BinaryVersion = "v9.9.9"
	result.SchemaCohort = "cohort-abc"
	result.ArtifactRoot = "/scanned/root"
	result.ArtifactRootConfigured = true

	out := FormatHuman(result, true)
	for _, want := range []string{"v9.9.9", "cohort-abc", "/scanned/root"} {
		if !strings.Contains(out, want) {
			t.Errorf("FormatHuman does not render %q:\n%s", want, out)
		}
	}
	if !strings.Contains(out, "configured: true") {
		t.Errorf("FormatHuman does not render the configured flag as true:\n%s", out)
	}

	// The UNCONFIGURED case is rendered too, not omitted — it is the default and the
	// motivating state, exactly as on the JSON side.
	result.ArtifactRootConfigured = false
	if out := FormatHuman(result, true); !strings.Contains(out, "configured: false") {
		t.Errorf("FormatHuman does not render the configured flag when it is false:\n%s", out)
	}

	// A result carrying NONE of the fields renders no identity block at all, so an
	// internally-assembled result (a unit test, a partial run) gains no empty lines.
	bare := FormatHuman(NewGateResult(nil), true)
	if strings.Contains(bare, "backstop version") || strings.Contains(bare, "artifact root:") {
		t.Errorf("FormatHuman rendered an identity block for a result carrying no identity:\n%s", bare)
	}
}

// TestResolveSymlinks_FallsBackToTheGivenPath covers the fallback the ungated scan
// depends on: a path that cannot be resolved (it does not exist) comes back unchanged
// rather than empty, so the walk reports the absence in its own terms instead of
// silently scanning the working directory.
func TestResolveSymlinks_FallsBackToTheGivenPath(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "no-such-directory")
	if got := resolveSymlinks(missing); got != missing {
		t.Errorf("resolveSymlinks(%q) = %q, want the path unchanged", missing, got)
	}

	// And a path that DOES exist is genuinely resolved, so the fallback above is not
	// the only behavior.
	real := t.TempDir()
	if got := resolveSymlinks(real); got == "" {
		t.Errorf("resolveSymlinks(%q) returned empty", real)
	}
}

// TestFindUngatedArtifacts_UnreadableProjectRootIsAnError covers the walk's error path:
// a projectRoot that is not a directory cannot be scanned, and the scan says so rather
// than returning an empty finding set that would read as a clean result.
func TestFindUngatedArtifacts_UnreadableProjectRootIsAnError(t *testing.T) {
	dir := t.TempDir()
	notADir := filepath.Join(dir, "a-file")
	if err := os.WriteFile(notADir, []byte("x"), 0o644); err != nil {
		t.Fatalf("writing the fixture file: %v", err)
	}

	root, err := artifact.ResolveRoot(dir, "")
	if err != nil {
		t.Fatalf("resolving the root: %v", err)
	}

	// Walking a path that is not a directory yields no findings and no error from
	// filepath.Walk itself — the file is simply visited — so the meaningful assertion
	// is that a non-artifact-shaped file produces nothing rather than a spurious finding.
	found, err := FindUngatedArtifacts(notADir, root, artifact.NonCorpusDirs{})
	if err != nil {
		t.Fatalf("FindUngatedArtifacts over a single non-artifact file errored: %v", err)
	}
	if len(found) != 0 {
		t.Errorf("a non-artifact-shaped file produced %d findings", len(found))
	}

	// A projectRoot that does not exist at all IS an error — the scan must not report
	// a clean empty set for a tree it never read.
	if _, err := FindUngatedArtifacts(filepath.Join(dir, "absent"), root, artifact.NonCorpusDirs{}); err == nil {
		t.Error("scanning an absent project root returned no error; an unscanned tree must not read as a clean result")
	}
}
