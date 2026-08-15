package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/schema"
)

// validateRepoRootFixture validates the repo-root layout fixture and returns the
// result. It is the shared corpus for the envelope claims: a small, fully covered
// corpus with more than one artifact type.
func validateRepoRootFixture(t *testing.T) (ValidateResult, string) {
	t.Helper()
	dir := layoutProfileDir(t, "repo-root")
	result, err := ValidateArtifacts(assertionConfig(t, dir))
	if err != nil {
		t.Fatalf("validating the repo-root fixture: %v", err)
	}
	if result.ArtifactsFound == 0 {
		t.Fatal("the repo-root fixture validated zero artifacts; every assertion below would be vacuous")
	}
	return result, dir
}

// jsonEnvelopeOf renders a ValidateResult through the JSON formatter and decodes it.
func jsonEnvelopeOf(t *testing.T, result ValidateResult) map[string]interface{} {
	t.Helper()
	f := &JSONFormatter{}
	out, err := f.FormatValidateResult(result)
	if err != nil {
		t.Fatalf("formatting the validate result as JSON: %v", err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("the validate JSON envelope is not valid JSON: %v\noutput: %s", err, out)
	}
	return parsed
}

// TestValidateOutput_JSONRecordsSchemaIdentityPerArtifact pins CLM-013. The envelope
// carries ONE record PER VALIDATED ARTIFACT, each with path, type and a
// `<schema_version>@<digest>` identity.
//
// The CARDINALITY assertion is what makes this claim distinguishable from the gate
// side's per-SCHEMA list (CLM-016): a flat list of schema identities is a genuinely
// different fact and cannot discharge a per-artifact binding.
func TestValidateOutput_JSONRecordsSchemaIdentityPerArtifact(t *testing.T) {
	result, _ := validateRepoRootFixture(t)
	env := jsonEnvelopeOf(t, result)

	raw, ok := env["artifacts"].([]interface{})
	if !ok {
		t.Fatalf("the JSON envelope carries no `artifacts` array (keys: %v)", envelopeKeys(env))
	}
	if len(raw) != result.ArtifactsFound {
		t.Fatalf("the envelope carries %d artifact records for %d validated artifacts; the binding is per artifact, not per schema", len(raw), result.ArtifactsFound)
	}

	cohort := realCohort(t)
	schemaless := 0
	for _, entry := range raw {
		rec, ok := entry.(map[string]interface{})
		if !ok {
			t.Fatalf("an artifacts entry is not an object: %#v", entry)
		}
		path, _ := rec["path"].(string)
		if path == "" {
			t.Errorf("an artifact record carries no path: %#v", rec)
		}
		if kind, _ := rec["type"].(string); kind == "" {
			t.Errorf("artifact record %q carries no type", path)
		}
		if isSchemaless, _ := rec["schemaless"].(bool); isSchemaless {
			schemaless++
			if id, _ := rec["schema_identity"].(string); id != "" {
				t.Errorf("schema-less record %q also carries schema_identity %q", path, id)
			}
			continue
		}
		id, _ := rec["schema_identity"].(string)
		if !strings.Contains(id, "@") {
			t.Errorf("artifact record %q carries schema_identity %q, want the form <schema_version>@<digest>", path, id)
		}
		version, digest, _ := strings.Cut(id, "@")
		if want, ok := cohort.DigestFor(version); !ok || digest != want {
			t.Errorf("artifact record %q reports digest %q for schema_version %q; the cohort says %q (covered=%v)", path, digest, version, want, ok)
		}
	}
	if schemaless == 0 {
		t.Error("no record was schema-less; the repo-root fixture carries a plan, so the schema-less branch was never exercised")
	}
}

// TestValidateOutput_HumanRecordsSameSchemaIdentityAsJSON pins CLM-014: the two
// renderings of ONE result read ONE resolved value. The identity SETS are compared for
// equality — asserting only that the human output is non-empty would pass for a
// rendering that recomputes and disagrees.
func TestValidateOutput_HumanRecordsSameSchemaIdentityAsJSON(t *testing.T) {
	result, _ := validateRepoRootFixture(t)

	env := jsonEnvelopeOf(t, result)
	jsonIdentities := identitiesFromEnvelope(t, env)
	if len(jsonIdentities) == 0 {
		t.Fatal("the JSON envelope carried no schema identities; the comparison below would be vacuous")
	}

	h := &HumanFormatter{}
	human, err := h.FormatValidateResult(result)
	if err != nil {
		t.Fatalf("formatting the validate result for humans: %v", err)
	}

	var missing []string
	for _, id := range jsonIdentities {
		if !strings.Contains(human, id) {
			missing = append(missing, id)
		}
	}
	if len(missing) > 0 {
		t.Errorf("the human rendering does not carry these identities the JSON rendering does: %v\nhuman output:\n%s", missing, human)
	}
}

// TestValidateOutput_SchemaIdentityChangesOnInPlaceRevision pins CLM-015 and is the
// BUNDLE-014 incident reproduced as a test: the schema_version const is BYTE-IDENTICAL
// on both sides and only the schema's CONTENT changed, so a path-derived identity would
// report the two runs as the same. The recorded identity must differ.
func TestValidateOutput_SchemaIdentityChangesOnInPlaceRevision(t *testing.T) {
	baseline := cohortFromFixtureTree(t, "baseline")
	revised := cohortFromFixtureTree(t, "revised")

	const revisedSchemaVersion = "bundle/v2"

	baseIdentity, baseOK := baseline.SchemaIdentity(revisedSchemaVersion)
	revIdentity, revOK := revised.SchemaIdentity(revisedSchemaVersion)
	if !baseOK || !revOK {
		t.Fatalf("the fixture cohorts do not both cover %q (baseline=%v revised=%v); the comparison below would be vacuous", revisedSchemaVersion, baseOK, revOK)
	}

	baseVersion, _, _ := strings.Cut(baseIdentity, "@")
	revVersion, _, _ := strings.Cut(revIdentity, "@")
	if baseVersion != revVersion {
		t.Fatalf("the declared schema_version differs between the two trees (%q vs %q); this test is about an IN-PLACE revision under a constant version", baseVersion, revVersion)
	}
	if baseIdentity == revIdentity {
		t.Errorf("an in-place schema revision under an unchanged schema_version produced the SAME identity %q; the identity is not content-derived", baseIdentity)
	}
}

// TestValidateOutput_JSONCarriesBinaryIdentity pins CLM-022: a result names the binary
// that produced it — its version and its schema cohort identifier.
func TestValidateOutput_JSONCarriesBinaryIdentity(t *testing.T) {
	result, _ := validateRepoRootFixture(t)
	env := jsonEnvelopeOf(t, result)

	identity := effectiveBuildIdentity()
	if got, _ := env["binary_version"].(string); got != identity.Version {
		t.Errorf("envelope binary_version = %q, want %q", got, identity.Version)
	}
	if got, _ := env["schema_cohort"].(string); got != realCohort(t).ID {
		t.Errorf("envelope schema_cohort = %q, want the computed cohort ID %q", got, realCohort(t).ID)
	}

	// The pre-existing envelope keys are additive-safe.
	for _, field := range []string{"schema_version", "pass", "violations_count", "violations"} {
		if _, ok := env[field]; !ok {
			t.Errorf("the widened envelope dropped the pre-existing field %q", field)
		}
	}
}

// TestValidateOutput_HumanCarriesSameBinaryIdentityAsJSON pins CLM-023: both
// renderings report the SAME binary identity, read from one resolved value.
func TestValidateOutput_HumanCarriesSameBinaryIdentityAsJSON(t *testing.T) {
	result, _ := validateRepoRootFixture(t)
	env := jsonEnvelopeOf(t, result)

	h := &HumanFormatter{}
	human, err := h.FormatValidateResult(result)
	if err != nil {
		t.Fatalf("formatting the validate result for humans: %v", err)
	}

	jsonVersion, _ := env["binary_version"].(string)
	jsonCohort, _ := env["schema_cohort"].(string)
	if jsonVersion == "" || jsonCohort == "" {
		t.Fatalf("the JSON envelope carries an empty binary identity (version=%q cohort=%q); the comparison below would be vacuous", jsonVersion, jsonCohort)
	}
	if !strings.Contains(human, jsonVersion) {
		t.Errorf("the human rendering does not carry the binary version %q the JSON rendering does:\n%s", jsonVersion, human)
	}
	if !strings.Contains(human, jsonCohort) {
		t.Errorf("the human rendering does not carry the schema cohort %q the JSON rendering does:\n%s", jsonCohort, human)
	}
}

// TestResultBinaryIdentity_MatchesVersionCommandOutput pins CLM-024. The two outputs
// are compared DIRECTLY, in one test — asserting a shared constant twice would prove
// nothing about whether the two surfaces agree.
func TestResultBinaryIdentity_MatchesVersionCommandOutput(t *testing.T) {
	result, _ := validateRepoRootFixture(t)
	env := jsonEnvelopeOf(t, result)

	root := NewRootCommand()
	versionOut, err := executeCommand(root, "version", "--json")
	if err != nil {
		t.Fatalf("version --json error: %v", err)
	}
	var version map[string]interface{}
	if err := json.Unmarshal([]byte(versionOut), &version); err != nil {
		t.Fatalf("version --json output is not valid JSON: %v", err)
	}

	resultVersion, _ := env["binary_version"].(string)
	commandVersion, _ := version["version"].(string)
	if resultVersion == "" || commandVersion == "" {
		t.Fatalf("one side reported an empty version (result=%q command=%q); the comparison would be vacuous", resultVersion, commandVersion)
	}
	if resultVersion != commandVersion {
		t.Errorf("a validate result reports version %q while `backstop version` reports %q; the two surfaces must read one resolved identity", resultVersion, commandVersion)
	}

	resultCohort, _ := env["schema_cohort"].(string)
	commandCohort, _ := version["schema_cohort"].(string)
	if resultCohort != commandCohort {
		t.Errorf("a validate result reports cohort %q while `backstop version` reports %q", resultCohort, commandCohort)
	}
}

// cohortFromFixtureTree computes a cohort over one of the pkg/schema cohort fixture
// trees, copied here so the in-place-revision claim is provable without rewriting the
// real embedded schemas.
func cohortFromFixtureTree(t *testing.T, tree string) schema.Cohort {
	t.Helper()
	dir := filepath.Join("..", "..", "pkg", "schema", "testdata", "cohort", tree)
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("cohort fixture tree %q not found: %v", tree, err)
	}
	c, err := schema.ComputeCohort(os.DirFS(dir))
	if err != nil {
		t.Fatalf("computing the cohort over fixture tree %q: %v", tree, err)
	}
	return c
}

// identitiesFromEnvelope extracts the sorted schema_identity values an envelope carries.
func identitiesFromEnvelope(t *testing.T, env map[string]interface{}) []string {
	t.Helper()
	raw, ok := env["artifacts"].([]interface{})
	if !ok {
		t.Fatalf("the JSON envelope carries no `artifacts` array (keys: %v)", envelopeKeys(env))
	}
	var out []string
	for _, entry := range raw {
		rec, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		if id, _ := rec["schema_identity"].(string); id != "" {
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out
}

// envelopeKeys lists an envelope's keys for a failure message.
func envelopeKeys(env map[string]interface{}) []string {
	out := make([]string, 0, len(env))
	for k := range env {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
