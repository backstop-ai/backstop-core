package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/artifact"
	"github.com/backstop-ai/backstop-core/pkg/gate"
)

// gateOutputInDir executes the real gate command with the process CWD set to dir, and
// returns its stdout plus the error the command returned. Driving the COMMAND rather
// than assembling steps by hand is what makes these tests statements about what an
// operator actually sees.
//
// It takes explicit args (unlike the waiver e2e helper of a similar shape, which is
// pinned to a human --all sweep) because these claims are about BOTH renderings.
func gateOutputInDir(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir to %s: %v", dir, err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Fatalf("chdir back: %v", err)
		}
	}()

	root := NewRootCommand()
	out, execErr := executeCommand(root, append([]string{"gate"}, args...)...)
	return out, execErr
}

// gateJSONInDir runs `gate --all --json` in dir and returns the RAW output alongside
// the decoded map. The raw text matters: a key dropped by omitempty is invisible once
// the JSON has been unmarshalled into a struct, and present-with-a-zero-value once it
// has been unmarshalled into a map.
func gateJSONInDir(t *testing.T, dir string) (string, map[string]interface{}) {
	t.Helper()
	out, _ := gateOutputInDir(t, dir, "--all", "--json")
	start := strings.Index(out, "{")
	if start < 0 {
		t.Fatalf("gate --json produced no JSON object:\n%s", out)
	}
	raw := out[start:]
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		t.Fatalf("gate --json output is not valid JSON: %v\noutput: %s", err, raw)
	}
	return raw, parsed
}

// TestGate_ArtifactValidation_UncoveredSchemaVersionRefusesGreen pins CLM-011: the
// gate INHERITS phase 8's refusal through its artifact_validation step rather than
// passing where the CLI would refuse. It delegates to the same ValidateArtifacts, so
// what this test proves is that the delegation is intact.
func TestGate_ArtifactValidation_UncoveredSchemaVersionRefusesGreen(t *testing.T) {
	dir := layoutProfileDir(t, "repo-root")
	if err := os.WriteFile(filepath.Join(dir, "specs", "SPEC-777-uncovered.spec.md"), []byte(uncoveredSpecContent), 0o644); err != nil {
		t.Fatalf("planting the uncovered spec: %v", err)
	}

	v := &realArtifactValidator{projectRoot: dir, root: rootAtDir(t, dir)}
	_, err := v.ValidateAll(t.Context())
	if err == nil {
		t.Fatal("the gate's artifact validator accepted an artifact declaring an uncovered schema_version; the CLI refuses, and the gate delegates to the same code")
	}
	if !strings.Contains(err.Error(), "spec/v99") {
		t.Errorf("the gate's refusal does not name the uncovered schema_version: %v", err)
	}

	// The control: the same fixture WITHOUT the planted spec validates cleanly, so the
	// refusal above is attributable to the uncovered version and not to the fixture.
	clean := layoutProfileDir(t, "repo-root")
	cleanValidator := &realArtifactValidator{projectRoot: clean, root: rootAtDir(t, clean)}
	if _, err := cleanValidator.ValidateAll(t.Context()); err != nil {
		t.Fatalf("the unmodified fixture failed the gate's artifact validation: %v", err)
	}
}

// TestGateOutput_JSONRecordsSchemaIdentities pins CLM-016: gate --json carries the
// per-SCHEMA identities its artifact validation asserted against.
//
// This is a DIFFERENT claim from the validate envelope's per-ARTIFACT records — one
// entry per schema in the cohort, not one per artifact on disk — and conflating them
// would let a per-schema list stand in for a binding it cannot express.
func TestGateOutput_JSONRecordsSchemaIdentities(t *testing.T) {
	_, parsed := gateJSONInDir(t, layoutProfileDir(t, "repo-root"))

	raw, ok := parsed["schema_identities"].([]interface{})
	if !ok {
		t.Fatalf("gate --json carries no schema_identities array (got %#v)", parsed["schema_identities"])
	}
	if len(raw) == 0 {
		t.Fatal("gate --json carries an EMPTY schema_identities array; the binary embeds schemas, so an empty list means none were recorded")
	}
	for _, entry := range raw {
		id, _ := entry.(string)
		if !strings.Contains(id, "@") {
			t.Errorf("schema identity %q is not of the form <schema_version>@<digest>", id)
		}
	}
}

// TestGate_JSONOutputCarriesBinaryIdentity pins CLM-020.
func TestGate_JSONOutputCarriesBinaryIdentity(t *testing.T) {
	_, parsed := gateJSONInDir(t, layoutProfileDir(t, "repo-root"))

	identity := effectiveBuildIdentity()
	if got, _ := parsed["binary_version"].(string); got != identity.Version {
		t.Errorf("gate --json binary_version = %q, want %q", got, identity.Version)
	}
	if got, _ := parsed["schema_cohort"].(string); got == "" {
		t.Error("gate --json carries an empty schema_cohort")
	}
	if got, _ := parsed["schema_version"].(string); got != "gate/v1" {
		t.Errorf("gate --json schema_version = %q, want exactly %q — the identity fields are additive", got, "gate/v1")
	}
}

// TestGate_HumanOutputPrintsBinaryIdentity pins CLM-021. The lines must come from the
// gate's own formatter, so the assertion is made against FormatHuman's output over a
// result carrying the fields — not against whatever the command happens to print
// beside it.
func TestGate_HumanOutputPrintsBinaryIdentity(t *testing.T) {
	_, parsed := gateJSONInDir(t, layoutProfileDir(t, "repo-root"))
	jsonVersion, _ := parsed["binary_version"].(string)
	jsonCohort, _ := parsed["schema_cohort"].(string)
	if jsonVersion == "" || jsonCohort == "" {
		t.Fatalf("the JSON rendering carries an empty identity (version=%q cohort=%q); the comparison below would be vacuous", jsonVersion, jsonCohort)
	}

	human, _ := gateOutputInDir(t, layoutProfileDir(t, "repo-root"), "--all")
	if !strings.Contains(human, jsonVersion) {
		t.Errorf("the gate's human output does not carry the binary version %q the JSON output does:\n%s", jsonVersion, human)
	}
	if !strings.Contains(human, jsonCohort) {
		t.Errorf("the gate's human output does not carry the schema cohort %q the JSON output does:\n%s", jsonCohort, human)
	}

	// And prove the lines are the FORMATTER's, not the command's: FormatHuman alone,
	// over a result carrying the fields, must render them.
	result := gate.NewGateResult(nil)
	result.BinaryVersion = "v9.9.9"
	result.SchemaCohort = "cohort-xyz"
	formatted := gate.FormatHuman(result, true)
	if !strings.Contains(formatted, "v9.9.9") || !strings.Contains(formatted, "cohort-xyz") {
		t.Errorf("gate.FormatHuman does not render the binary identity; a print beside the FormatHuman call would be a SECOND rendering path that can drift from the JSON one:\n%s", formatted)
	}
}

// TestGate_HumanAndJSONOutputNameScannedArtifactRoot pins CLM-054: both renderings name
// the artifact root actually scanned.
func TestGate_HumanAndJSONOutputNameScannedArtifactRoot(t *testing.T) {
	dir := layoutProfileDir(t, "repo-root")
	_, parsed := gateJSONInDir(t, dir)

	jsonRoot, _ := parsed["artifact_root"].(string)
	if jsonRoot == "" {
		t.Fatal("gate --json carries an empty artifact_root")
	}

	human, _ := gateOutputInDir(t, dir, "--all")
	if !strings.Contains(human, jsonRoot) {
		t.Errorf("the gate's human output does not name the scanned artifact root %q the JSON output does:\n%s", jsonRoot, human)
	}

	result := gate.NewGateResult(nil)
	result.ArtifactRoot = "/some/scanned/root"
	if !strings.Contains(gate.FormatHuman(result, true), "/some/scanned/root") {
		t.Error("gate.FormatHuman does not render the scanned artifact root")
	}
}

// TestGate_ConfiguredAbsentArtifactRootFailsLoudly pins CLM-055, and it lives in the
// same file as its twin below deliberately: the distinction between ABSENT (a loud
// failure) and EXISTING-BUT-EMPTY (a legitimate pass) drifts the moment the two are
// tested apart.
func TestGate_ConfiguredAbsentArtifactRootFailsLoudly(t *testing.T) {
	dir := layoutProfileDir(t, "configured-absent-root")

	// The fixture really does declare a root that is not on disk — otherwise the
	// assertion below would hold for the wrong reason.
	if _, err := os.Stat(filepath.Join(dir, ".backstop")); !os.IsNotExist(err) {
		t.Fatalf("the configured-absent-root fixture has a .backstop directory on disk (stat err=%v); it must be absent", err)
	}

	out, err := gateOutputInDir(t, dir, "--all")
	if err == nil {
		t.Fatalf("a CONFIGURED artifact root that is absent from disk did not fail the run; the gate walked absent directories and reported a passing dimension.\noutput:\n%s", out)
	}

	diagnostic := err.Error() + "\n" + out
	if !strings.Contains(diagnostic, ".backstop") {
		t.Errorf("the failure does not name the declared root, so an operator cannot act on it: %s", diagnostic)
	}
}

// TestGate_ConfiguredEmptyArtifactRootPasses pins CLM-056 as NARROWED by spec v1.2.2:
// root RESOLUTION succeeds and the artifact_validation DIMENSION reports clean over the
// zero-artifact corpus.
//
// IT DELIBERATELY DOES NOT ASSERT THE AGGREGATE VERDICT. Other dimensions may
// legitimately fail over this fixture for reasons that have nothing to do with root
// resolution, and an aggregate assertion here would land in direct conflict with
// CLM-069. "The empty root reads cleanly" is a per-dimension fact, not a promise about
// the whole run.
func TestGate_ConfiguredEmptyArtifactRootPasses(t *testing.T) {
	dir := layoutProfileDir(t, "configured-empty-root")

	// The existing-but-empty spec directory IS the point — it is what real `backstop
	// init` produces, and an ABSENT one models a shape init cannot reach.
	specDir := filepath.Join(dir, ".backstop", "specs")
	info, err := os.Stat(specDir)
	if err != nil || !info.IsDir() {
		t.Fatalf("the configured-empty-root fixture has no .backstop/specs directory (stat err=%v); the existing-but-empty case is what this test is about", err)
	}

	// Root RESOLUTION succeeds: an existing-but-empty configured root is not the
	// RootMissingError case.
	root, resolveErr := artifact.ResolveRoot(dir, ".backstop")
	if resolveErr != nil {
		t.Fatalf("resolving an existing-but-empty configured artifact root failed: %v", resolveErr)
	}
	if !root.Configured {
		t.Fatalf("the resolved root at %s is not marked configured", root.Path)
	}

	// And the artifact_validation DIMENSION reports clean over the zero-artifact corpus.
	v := &realArtifactValidator{projectRoot: dir, root: root}
	violations, err := v.ValidateAll(t.Context())
	if err != nil {
		t.Fatalf("artifact validation over an existing-but-empty artifact root errored: %v", err)
	}
	if len(violations) != 0 {
		t.Errorf("artifact validation over an existing-but-empty artifact root reported %d violations: %#v", len(violations), violations)
	}
}

// TestGate_JSONCarriesArtifactRootConfiguredFalseWhenUnconfigured pins CLM-068.
//
// The assertion is on the RAW JSON TEXT. Unmarshalling into a map reintroduces nothing,
// but unmarshalling into a struct would give the field its zero value whether or not it
// was serialized — and the bug this guards against is an omitempty that DROPS the key
// for exactly the false case that matters.
func TestGate_JSONCarriesArtifactRootConfiguredFalseWhenUnconfigured(t *testing.T) {
	raw, parsed := gateJSONInDir(t, layoutProfileDir(t, "repo-root"))

	if !strings.Contains(raw, `"artifact_root_configured"`) {
		t.Fatalf("gate --json OMITS artifact_root_configured on an UNCONFIGURED project — the exact case REQ-008 was written for.\nraw output:\n%s", raw)
	}
	configured, present := parsed["artifact_root_configured"]
	if !present {
		t.Fatal("artifact_root_configured is absent from the decoded gate JSON")
	}
	if configured != false {
		t.Errorf("artifact_root_configured = %#v on a project declaring no artifact_root, want false", configured)
	}
}
