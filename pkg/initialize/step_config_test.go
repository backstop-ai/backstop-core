package initialize

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/config"
	"gopkg.in/yaml.v3"
)

// theFiveSdlcDimensions are the dimensions REQ-003 names, returned fresh so every
// assertion below is about the same five and none can mutate the list for the others.
//
// They are the five that HARD-ERROR on a missing `specs/` directory rather than
// skipping, which is why the pack-only profile must turn them off explicitly instead
// of relying on them to notice there is nothing to check.
func theFiveSdlcDimensions() []string {
	return []string{
		"test_verification",
		"coverage_threshold",
		"contract_signature",
		"test_substantiveness",
		"artifact_status_drift",
	}
}

// runConfigStep runs the config step over a fresh temp project under the supplied
// capability set and returns the project root and the report.
func runConfigStep(t *testing.T, capabilities map[Capability]bool) (string, StepReport) {
	t.Helper()
	root := t.TempDir()
	return root, stepConfig(root, capabilities)
}

// generatedConfigMap parses the generated backstop.yml into an untyped map, so an
// assertion can speak about the SET of top-level keys — which is what "no other
// top-level key" requires and what a typed decode into config.Config would hide.
func generatedConfigMap(t *testing.T, root string) map[string]any {
	t.Helper()
	var parsed map[string]any
	if err := yaml.Unmarshal([]byte(readFile(t, root, "backstop.yml")), &parsed); err != nil {
		t.Fatalf("the generated backstop.yml is not parseable YAML: %v", err)
	}
	return parsed
}

// policyLevel returns the generated enforcement.policy level for one dimension.
func policyLevel(t *testing.T, root, dimension string) (string, bool) {
	t.Helper()
	loaded, err := config.LoadConfigFromPath(filepath.Join(root, "backstop.yml"))
	if err != nil {
		t.Fatalf("the generated config did not load: %v", err)
	}
	entry, declared := loaded.Enforcement.Policy[dimension]
	if !declared {
		return "", false
	}
	return entry.Level, true
}

// TestInit_FullSdlcConfigLeavesAllFiveSdlcDimensionsEnforced (SPEC-069 CLM-015).
//
// The full-SDLC config carries `project:` plus `artifact_root: .backstop` and NO
// other top-level key, and sets NONE of the five SDLC dimensions to `level: off`.
func TestInit_FullSdlcConfigLeavesAllFiveSdlcDimensionsEnforced(t *testing.T) {
	root, report := runConfigStep(t, allCapabilities(t))

	if report.Outcome != OutcomeDelivered {
		t.Fatalf("config step reported %v (%s), want OutcomeDelivered", report.Outcome, report.Detail)
	}

	parsed := generatedConfigMap(t, root)
	want := map[string]bool{"project": true, "artifact_root": true}
	for key := range parsed {
		if !want[key] {
			t.Fatalf("the full-SDLC config carries the unexpected top-level key %q; it must carry exactly project and artifact_root", key)
		}
	}
	for key := range want {
		if _, present := parsed[key]; !present {
			t.Fatalf("the full-SDLC config is missing the top-level key %q", key)
		}
	}

	for _, dimension := range theFiveSdlcDimensions() {
		level, declared := policyLevel(t, root, dimension)
		if declared && level == "off" {
			t.Fatalf("the full-SDLC config turned %s off; the greenfield profile enforces all five SDLC dimensions", dimension)
		}
	}
}

// TestInit_PackOnlyDisablesTestVerification (SPEC-069 CLM-016).
func TestInit_PackOnlyDisablesTestVerification(t *testing.T) {
	assertPackOnlyDisables(t, "test_verification")
}

// TestInit_PackOnlyDisablesCoverageThreshold (SPEC-069 CLM-017).
func TestInit_PackOnlyDisablesCoverageThreshold(t *testing.T) {
	assertPackOnlyDisables(t, "coverage_threshold")
}

// TestInit_PackOnlyDisablesContractSignature (SPEC-069 CLM-018).
func TestInit_PackOnlyDisablesContractSignature(t *testing.T) {
	assertPackOnlyDisables(t, "contract_signature")
}

// TestInit_PackOnlyDisablesTestSubstantiveness (SPEC-069 CLM-019).
func TestInit_PackOnlyDisablesTestSubstantiveness(t *testing.T) {
	assertPackOnlyDisables(t, "test_substantiveness")
}

// TestInit_PackOnlyDisablesArtifactStatusDrift (SPEC-069 CLM-020).
func TestInit_PackOnlyDisablesArtifactStatusDrift(t *testing.T) {
	assertPackOnlyDisables(t, "artifact_status_drift")
}

// assertPackOnlyDisables is the body the five one-dimension claims share.
//
// They are FIVE claims and five tests rather than one table, deliberately: a matrix
// asserted in aggregate hides a missing member, and each of the five is separately
// load-bearing because each hard-errors on a missing specs/ directory.
func assertPackOnlyDisables(t *testing.T, dimension string) {
	t.Helper()
	root, report := runConfigStep(t, capabilitiesExcept(t, "sdlc"))

	if report.Outcome != OutcomeDelivered {
		t.Fatalf("config step reported %v (%s), want OutcomeDelivered", report.Outcome, report.Detail)
	}

	level, declared := policyLevel(t, root, dimension)
	if !declared {
		t.Fatalf("the pack-only config declares no enforcement.policy entry for %s; that dimension hard-errors on a missing specs/ directory rather than skipping, so it must be turned off explicitly", dimension)
	}
	if level != "off" {
		t.Fatalf("the pack-only config sets %s to %q, want \"off\"", dimension, level)
	}
}

// TestInit_FullSdlcConfigRoundTripsThroughConfigLoad (SPEC-069 CLM-021).
//
// The assertion is against the SHIPPED config.Load AND the shipped JSON-schema pass,
// never a hand-rolled unmarshal: a config init writes must never be one the binary
// refuses to read. Both halves matter — the strict typed decode rejects an unknown
// field, and the schema's additionalProperties:false rejects an undeclared key —
// and LoadConfigFromPath runs both.
func TestInit_FullSdlcConfigRoundTripsThroughConfigLoad(t *testing.T) {
	root, _ := runConfigStep(t, allCapabilities(t))

	loaded, err := config.LoadConfigFromPath(filepath.Join(root, "backstop.yml"))
	if err != nil {
		t.Fatalf("the full-SDLC config init generated does not load through the shipped loader: %v\n---\n%s", err, readFile(t, root, "backstop.yml"))
	}
	if loaded.Project == "" {
		t.Fatal("the loaded config carries an empty project name")
	}
	if loaded.ArtifactRoot != ".backstop" {
		t.Fatalf("the loaded config carries artifact_root %q, want \".backstop\"; discovery must resolve the layout init created", loaded.ArtifactRoot)
	}
}

// TestInit_PackOnlyConfigRoundTripsThroughConfigLoad (SPEC-069 CLM-022).
//
// The pack-only profile writes no `artifact_root`, and this claim is listed anyway on
// purpose: it exercises the same loader and the same schema pass as its sibling, so a
// loader-level failure is diagnosed as one failure rather than looking like two
// distinct ones.
func TestInit_PackOnlyConfigRoundTripsThroughConfigLoad(t *testing.T) {
	root, _ := runConfigStep(t, capabilitiesExcept(t, "sdlc"))

	loaded, err := config.LoadConfigFromPath(filepath.Join(root, "backstop.yml"))
	if err != nil {
		t.Fatalf("the pack-only config init generated does not load through the shipped loader: %v\n---\n%s", err, readFile(t, root, "backstop.yml"))
	}
	if loaded.ArtifactRoot != "" {
		t.Fatalf("the pack-only config declares artifact_root %q; that profile scaffolds no artifact directories, so declaring a root would point discovery at a layout that does not exist", loaded.ArtifactRoot)
	}
	if len(loaded.Enforcement.Policy) != len(theFiveSdlcDimensions()) {
		t.Fatalf("the pack-only config declares %d policy entries, want exactly the five SDLC dimensions", len(loaded.Enforcement.Policy))
	}
}

// TestInit_FullSdlcWritesArtifactRootPointingAtBackstopDir (SPEC-069 CLM-026).
//
// The key's spelling is CONFIRMED from SPEC-068 REQ-006 (`Config.ArtifactRoot`, yaml
// tag `artifact_root`), not guessed. Init that scaffolds `.backstop/` without
// declaring it as the artifact root manufactures exactly the silent-undiscovery false
// green DD-15 exists to prevent.
func TestInit_FullSdlcWritesArtifactRootPointingAtBackstopDir(t *testing.T) {
	root, _ := runConfigStep(t, allCapabilities(t))

	raw := readFile(t, root, "backstop.yml")
	if !strings.Contains(raw, "artifact_root:") {
		t.Fatalf("the generated config carries no artifact_root key.\n---\n%s", raw)
	}

	loaded, err := config.LoadConfigFromPath(filepath.Join(root, "backstop.yml"))
	if err != nil {
		t.Fatalf("the generated config does not load: %v", err)
	}
	if loaded.ArtifactRoot != ".backstop" {
		t.Fatalf("artifact_root is %q, want \".backstop\" — the directory REQ-004's layout step creates", loaded.ArtifactRoot)
	}
}

// TestInit_GeneratedConfigNeverWritesTheRetiredLanguageKey (SPEC-069 CLM-023).
//
// `language:` was retired by SPEC-046. pkg/config absorbs it as a legacy key and
// ignores it, so writing it would emit a DEAD key that bakes a language into the very
// first file a consumer sees — the opposite of what a thin executor's onboarding
// should hand someone.
func TestInit_GeneratedConfigNeverWritesTheRetiredLanguageKey(t *testing.T) {
	for _, profile := range []struct {
		name         string
		capabilities map[Capability]bool
	}{
		{"full-sdlc", allCapabilities(t)},
		{"pack-only", capabilitiesExcept(t, "sdlc")},
	} {
		t.Run(profile.name, func(t *testing.T) {
			root, _ := runConfigStep(t, profile.capabilities)
			if _, present := generatedConfigMap(t, root)["language"]; present {
				t.Fatalf("the %s config writes the retired language: key", profile.name)
			}
			if strings.Contains(readFile(t, root, "backstop.yml"), "language") {
				t.Fatalf("the %s config's TEXT mentions language; the retired key must not appear even as a comment", profile.name)
			}
		})
	}
}

// TestInit_ProjectNameComesFromDirectoryBasenameOnly (SPEC-069 CLM-024).
//
// The fixture POPULATES the directory with differing project-identity files and
// asserts the `project:` value is unchanged: nothing in the project is read to name
// it. Reading one would be detection, and detection is the one thing init does not do.
func TestInit_ProjectNameComesFromDirectoryBasenameOnly(t *testing.T) {
	root := t.TempDir()
	expected := filepath.Base(root)

	// Two files that a detecting implementation would be tempted by, each naming
	// something OTHER than the directory. They live in the fixture, so they put no
	// literal in the init source set.
	writeFile(t, root, "identity-a.manifest", "{\"name\": \"name-from-manifest-a\"}\n")
	writeFile(t, root, "identity-b.manifest", "name = \"name-from-manifest-b\"\n")

	report := stepConfig(root, allCapabilities(t))
	if report.Outcome != OutcomeDelivered {
		t.Fatalf("config step reported %v (%s)", report.Outcome, report.Detail)
	}

	loaded, err := config.LoadConfigFromPath(filepath.Join(root, "backstop.yml"))
	if err != nil {
		t.Fatalf("the generated config does not load: %v", err)
	}
	if loaded.Project != expected {
		t.Fatalf("project is %q, want the directory basename %q; nothing on disk may influence it", loaded.Project, expected)
	}
	for _, planted := range []string{"name-from-manifest-a", "name-from-manifest-b"} {
		if strings.Contains(readFile(t, root, "backstop.yml"), planted) {
			t.Fatalf("the generated config carries %q, which came from a file in the project — that is detection", planted)
		}
	}
}

// TestInit_InventsNoCoverageFloorEnforcementKey (SPEC-069 CLM-093).
//
// REQ-033's tripwire. The shipped schema declares `enforcement` with
// additionalProperties:false over exactly security, waiver_warning_days,
// semgrep_version, baseline_ttl, test_command, toolchain, policy — so a
// `coverage.min_pct`-shaped key would be REJECTED at load. The tempting "fix" —
// adding the key to the schema to make REQ-033 satisfiable — is init designing an
// enforcement-policy surface the bundle excludes, from the wrong artifact.
func TestInit_InventsNoCoverageFloorEnforcementKey(t *testing.T) {
	schemaDeclared := map[string]bool{
		"security": true, "waiver_warning_days": true, "semgrep_version": true,
		"baseline_ttl": true, "test_command": true, "toolchain": true, "policy": true,
	}

	for _, profile := range []struct {
		name         string
		capabilities map[Capability]bool
	}{
		{"full-sdlc", allCapabilities(t)},
		{"pack-only", capabilitiesExcept(t, "sdlc")},
	} {
		t.Run(profile.name, func(t *testing.T) {
			root, _ := runConfigStep(t, profile.capabilities)
			parsed := generatedConfigMap(t, root)

			block, present := parsed["enforcement"]
			if !present {
				return
			}
			enforcement, ok := block.(map[string]any)
			if !ok {
				t.Fatalf("the generated enforcement block is not a mapping: %T", block)
			}
			for key := range enforcement {
				if !schemaDeclared[key] {
					t.Fatalf("the %s config writes enforcement.%s, which the shipped schema does not declare; additionalProperties:false would reject it at load, and adding it to the schema would be init designing an enforcement-policy surface",
						profile.name, key)
				}
			}
			if _, invented := enforcement["coverage"]; invented {
				t.Fatalf("the %s config invented an enforcement.coverage key; REQ-033 is a DOCUMENTED, UNSATISFIED gap and init reports the forfeiture rather than wiring a floor", profile.name)
			}
		})
	}
}

// TestInit_PackOnlyReportsTheCoverageFloorGapWithoutFailing (SPEC-069 CLM-094).
//
// Turning `coverage_threshold` off forfeits coverage enforcement for a consumer whose
// packs still emit coverage records. The spec-independent floor that would replace it
// does not exist and has no owner, so the pack-only profile REPORTS the forfeiture and
// does not fail the run — loud, not blocking.
func TestInit_PackOnlyReportsTheCoverageFloorGapWithoutFailing(t *testing.T) {
	_, report := runConfigStep(t, capabilitiesExcept(t, "sdlc"))

	if report.Outcome == OutcomeBrokenPromise {
		t.Fatalf("the pack-only config step failed the run over the coverage-floor gap: %s", report.Detail)
	}
	detail := strings.ToLower(report.Detail)
	if !strings.Contains(detail, "coverage") {
		t.Fatalf("the pack-only report does not name the forfeited coverage enforcement.\ngot: %s", report.Detail)
	}
	if !strings.Contains(detail, "does not exist") && !strings.Contains(detail, "no owner") && !strings.Contains(detail, "not available") {
		t.Fatalf("the pack-only report names the gap but does not state that the spec-independent floor knob does not exist, which is the half that tells a consumer not to go looking for it.\ngot: %s", report.Detail)
	}
}

// TestInit_FullSdlcEmitsNoCoverageFloorGapNotice (SPEC-069 CLM-095).
//
// The falsifying half of CLM-094: the greenfield profile forfeits nothing, so a
// notice there would be init reporting a gap that does not exist. An implementation
// that always emits the line passes CLM-094 and fails here.
func TestInit_FullSdlcEmitsNoCoverageFloorGapNotice(t *testing.T) {
	_, report := runConfigStep(t, allCapabilities(t))

	if strings.Contains(strings.ToLower(report.Detail), "coverage") {
		t.Fatalf("the full-SDLC config report mentions coverage; that profile enforces coverage_threshold and forfeits nothing.\ngot: %s", report.Detail)
	}
}

// TestInit_GeneratedConfigCarriesNoScopeOverride (SPEC-069 CLM-071).
//
// REQ-015 is satisfied by NOT REGRESSING the shipped diff-scope default, so the
// generated config must write nothing that changes, overrides or pins gate scope.
func TestInit_GeneratedConfigCarriesNoScopeOverride(t *testing.T) {
	for _, profile := range []struct {
		name         string
		capabilities map[Capability]bool
	}{
		{"full-sdlc", allCapabilities(t)},
		{"pack-only", capabilitiesExcept(t, "sdlc")},
	} {
		t.Run(profile.name, func(t *testing.T) {
			root, _ := runConfigStep(t, profile.capabilities)
			raw := readFile(t, root, "backstop.yml")
			for _, forbidden := range []string{"scope", "--all", "--file", "--base", "diff"} {
				if strings.Contains(raw, forbidden) {
					t.Fatalf("the %s config mentions %q; init must write no configuration that changes, overrides or pins the gate's default scope.\n---\n%s",
						profile.name, forbidden, raw)
				}
			}
		})
	}
}

// TestInit_ExistingBackstopYmlIsPreservedNotOverwritten (SPEC-069 CLM-038).
//
// An existing config is preserved BYTE-FOR-BYTE and reported as already present.
// Converge, never clobber: the consumer's policy decisions are theirs.
func TestInit_ExistingBackstopYmlIsPreservedNotOverwritten(t *testing.T) {
	root := t.TempDir()
	// Deliberately unlike anything init would generate — different project name,
	// a key init never writes, and a comment a rewrite would drop.
	original := "# hand-written by the consumer\nproject: a-name-init-would-never-choose\nruntimes:\n  - some-runtime\n"
	writeFile(t, root, "backstop.yml", original)

	report := stepConfig(root, allCapabilities(t))

	if report.Outcome != OutcomeConverged {
		t.Fatalf("config step reported %v over an existing backstop.yml, want OutcomeConverged (%s)", report.Outcome, report.Detail)
	}
	if got := readFile(t, root, "backstop.yml"); got != original {
		t.Fatalf("the existing backstop.yml was rewritten.\nbefore:\n%s\nafter:\n%s", original, got)
	}
}

// TestInit_UnnameableProjectDirectoryIsABrokenPromiseNotASilentDefault covers the
// generation-failure arm of the config step.
//
// Config generation is UNCONDITIONAL, so the one thing that must never happen when a
// project cannot be named is a quiet substitution — an empty `project:`, a placeholder,
// or a report that says the config was delivered when nothing was written. The failure
// travels as OutcomeBrokenPromise carrying the naming reason, and NO file lands.
//
// The filesystem root is the directory whose basename is genuinely unusable: `filepath.
// Base("/")` is the separator itself, which is exactly the case projectName refuses.
func TestInit_UnnameableProjectDirectoryIsABrokenPromiseNotASilentDefault(t *testing.T) {
	unnameable := string(filepath.Separator)

	// Precondition, stated loudly rather than assumed: the step returns early and
	// reports convergence if a backstop.yml is already there, which would make the
	// assertion below about the wrong branch.
	if _, err := os.Stat(filepath.Join(unnameable, "backstop.yml")); err == nil {
		t.Skipf("%s exists on this machine, so stepConfig would converge instead of reaching generation", filepath.Join(unnameable, "backstop.yml"))
	}

	report := stepConfig(unnameable, allCapabilities(t))

	if report.Outcome != OutcomeBrokenPromise {
		t.Fatalf("config step reported %v (%s) for an unnameable project directory, want OutcomeBrokenPromise: an init that cannot name the project must say so, never substitute a default",
			report.Outcome, report.Detail)
	}
	if report.Step != stepConfigName {
		t.Fatalf("the failing report is attributed to step %q, want %q", report.Step, stepConfigName)
	}
	// The detail must carry the GENERATION reason, not a bare "failed": the whole point
	// of a broken-promise report is that the consumer can act on it.
	for _, fragment := range []string{"generating backstop.yml", "basename"} {
		if !strings.Contains(report.Detail, fragment) {
			t.Fatalf("the broken-promise detail does not mention %q, so it does not explain what failed.\ngot: %s", fragment, report.Detail)
		}
	}
	if _, err := os.Stat(filepath.Join(unnameable, "backstop.yml")); err == nil {
		t.Fatalf("a backstop.yml was written at %s despite generation failing; a failed generation writes nothing", unnameable)
	}
}

// TestInit_UnwritableProjectDirectoryIsABrokenPromiseNamingTheWrite covers the
// write-failure arm, which is a DIFFERENT arm from generation failing.
//
// Generation succeeding and the write failing must not be reported as success. The two
// arms are distinguished by what the detail says — "writing backstop.yml" rather than
// "generating backstop.yml" — because a consumer's remedy differs: one is a permissions
// problem in their own directory, the other is not.
func TestInit_UnwritableProjectDirectoryIsABrokenPromiseNamingTheWrite(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: a read-only directory does not refuse a write, so the arm under test is unreachable")
	}

	project := filepath.Join(t.TempDir(), "unwritable-project")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatalf("creating the project directory: %v", err)
	}
	if err := os.Chmod(project, 0o555); err != nil {
		t.Fatalf("making the project directory read-only: %v", err)
	}
	// Restored so t.TempDir's cleanup can remove the tree.
	t.Cleanup(func() { _ = os.Chmod(project, 0o755) })

	report := stepConfig(project, allCapabilities(t))

	if report.Outcome != OutcomeBrokenPromise {
		t.Fatalf("config step reported %v (%s) when backstop.yml could not be written, want OutcomeBrokenPromise",
			report.Outcome, report.Detail)
	}
	if !strings.Contains(report.Detail, "writing backstop.yml") {
		t.Fatalf("the detail does not attribute the failure to the WRITE, so it is indistinguishable from a generation failure.\ngot: %s", report.Detail)
	}
	if strings.Contains(report.Detail, "generating backstop.yml") {
		t.Fatalf("a write failure is reported as a generation failure; the two arms point a consumer at different remedies.\ngot: %s", report.Detail)
	}
	if _, err := os.Stat(filepath.Join(project, "backstop.yml")); err == nil {
		t.Fatal("a backstop.yml exists after the write was refused")
	}
}
