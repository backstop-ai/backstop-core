package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/artifact"
	"github.com/backstop-ai/backstop-core/pkg/gate"
)

// noncorpus_wiring_test.go closes a STRUCTURAL gap, not a hypothetical one.
//
// Every other ISSUE-122 test — the walk tests in phase 3 and the discovery e2e tests
// beside this file — assembles the exclusion set ITSELF and then calls DiscoverArtifacts
// or FindUngatedArtifacts DIRECTLY. An implementer who silently passed
// artifact.NonCorpusDirs{} at all four production call sites would keep every one of
// those tests GREEN while reintroducing exactly the regression this change prevents.
// The tests below drive REAL production entry points over the SAME fixture instead.

// plantedFixtureBasenames are the artifact-shaped files planted inside declared
// dependency trees, by BASENAME. Assertions use basenames deliberately: the ungated
// findings carry full paths but a resolution violation carries only the citing
// artifact's basename, and a full-path assertion would silently never match it.
func plantedFixtureBasenames() []string {
	return []string{
		"SPEC-902-vendored-citer.spec.md",
		"ISSUE-902-node-dependency.issue.md",
		"SPEC-903-nested-vendored.spec.md",
		"SPEC-904-invented-ecosystem.spec.md",
	}
}

// TestNonCorpusWiring_RealCommandsHonorPackDeclarationAcrossPackLoadOutcomes (CLM-005):
// ONE TABLE, FOUR ROWS — {doctor, artifact validate} x {packs load, packs do NOT load}.
//
// Each row copies the fixture to a temp dir and drives the REAL root command there
// through the existing doctor harness, which chdirs and calls NewRootCommand().Execute().
// Assertions are on DECODED/RENDERED OUTPUT, never on an internal helper's return value:
// the helper is already known to work, and it is the wiring TO it that is unverified.
//
// THE TABLE CATCHES BOTH DIRECTIONS BY CONSTRUCTION, which is why it is one table and
// not two unrelated tests. A call site that hardcodes the zero value fails the HAPPY
// rows. A call site that "softens" the degraded path by defaulting the ecosystem names
// back into core fails the DEGRADED rows. Neither branch needs a preferred behavior
// chosen; the table pins that the wiring does not regress in either direction.
func TestNonCorpusWiring_RealCommandsHonorPackDeclarationAcrossPackLoadOutcomes(t *testing.T) {
	for _, row := range []struct {
		name string
		// packsLoad false deletes the installed pack tree while LEAVING backstop.yml's
		// declaration in place, so loadInstalledPacks fails for real ("declared pack %s
		// is missing from %s") — the production failure, not a stub.
		packsLoad bool
		command   string
	}{
		{name: "doctor with packs loaded", packsLoad: true, command: "doctor"},
		{name: "doctor with packs failing to load", packsLoad: false, command: "doctor"},
		{name: "artifact validate with packs loaded", packsLoad: true, command: "validate"},
		{name: "artifact validate with packs failing to load", packsLoad: false, command: "validate"},
	} {
		t.Run(row.name, func(t *testing.T) {
			projectRoot := copyFixtureToTemp(t, noncorpusFixtureRoot(t))
			if !row.packsLoad {
				breakPackInstall(t, projectRoot)
			}

			switch row.command {
			case "doctor":
				assertDoctorLayoutRow(t, projectRoot, row.packsLoad)
			case "validate":
				assertArtifactValidateRow(t, projectRoot, row.packsLoad)
			}
		})
	}
}

// breakPackInstall deletes the INSTALLED pack tree while leaving backstop.yml's
// declaration alone, reproducing the production pack-load failure.
func breakPackInstall(t *testing.T, projectRoot string) {
	t.Helper()
	installed := filepath.Join(projectRoot, ".backstop", "packs", "backstop", "noncorpus-fixture")
	if err := os.RemoveAll(installed); err != nil {
		t.Fatalf("removing the installed pack at %s: %v", installed, err)
	}
	if _, err := loadInstalledPacks(projectRoot); err == nil {
		t.Fatalf("loadInstalledPacks still succeeded after removing %s; the degraded row would not be degraded", installed)
	}
}

// assertDoctorLayoutRow drives the REAL `doctor --json` command and asserts on the
// rendered artifact-layout entry.
func assertDoctorLayoutRow(t *testing.T, projectRoot string, packsLoad bool) {
	t.Helper()

	payload, _ := runDoctorJSON(t, projectRoot, "--check", doctorCheckArtifactLayout)

	entry := payload.find(t, doctorCheckArtifactLayout)
	rendered, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("re-encoding the artifact-layout entry: %v", err)
	}
	text := string(rendered)

	for _, basename := range plantedFixtureBasenames() {
		mentioned := strings.Contains(text, basename)
		switch {
		case packsLoad && mentioned:
			t.Errorf("doctor's artifact-layout check named %s as a deviation even though the pack declares its dependency tree; the exclusion set is not reaching FindUngatedArtifacts from ctx.Packs.\nentry: %s", basename, text)
		case !packsLoad && !mentioned:
			t.Errorf("with the pack load FAILING, doctor's artifact-layout check did NOT name %s. On that path the set is the tool-agnostic base only, so these files MUST surface — anything else means core is defaulting the ecosystem nouns back in.\nentry: %s", basename, text)
		}
	}
}

// assertArtifactValidateRow drives the REAL `artifact validate --all --json` command
// and asserts on its decoded per-artifact records.
func assertArtifactValidateRow(t *testing.T, projectRoot string, packsLoad bool) {
	t.Helper()

	stdout, _ := runDoctorInProject(t, projectRoot, "artifact", "validate", "--all", "--json")

	var payload struct {
		Records []struct {
			Path string `json:"path"`
		} `json:"artifacts"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("decoding `artifact validate --all --json` output: %v\noutput: %s", err, stdout)
	}

	// The command must RUN AND REPORT on both paths. A pack-load failure must not fail
	// `artifact validate` — it has to keep working in a project with no packs at all.
	if len(payload.Records) == 0 {
		t.Fatalf("`artifact validate --all --json` reported NO per-artifact records; it must still run and report regardless of the pack-load outcome.\noutput: %s", stdout)
	}

	recorded := map[string]bool{}
	for _, r := range payload.Records {
		recorded[filepath.Base(r.Path)] = true
	}

	for _, basename := range plantedFixtureBasenames() {
		switch {
		case packsLoad && recorded[basename]:
			t.Errorf("`artifact validate` carried a per-artifact record for %s, which sits inside a pack-declared dependency tree; ValidateConfig.NonCorpus is not populated from the installed packs at the CLI construction site.\nrecords: %v", basename, payload.Records)
		case !packsLoad && !recorded[basename]:
			t.Errorf("with the pack load FAILING, `artifact validate` carried NO record for %s. On that path the exclusion set is the tool-agnostic base only, so these files MUST be discovered.\nrecords: %v", basename, payload.Records)
		}
	}
}

// TestGateSteps_ExclusionSetWiredIntoArtifactValidation (CLM-005): THE GATE'S
// ARTIFACT-VALIDATION HOP, end to end through the real builder.
//
// A step's identity is only knowable by CALLING it — there is no way to invoke one step
// in isolation — so this executes the returned slice IN ORDER, keys results by
// StepName, and asserts ONLY on the artifact_validation entry. Nothing else is
// asserted: the other steps' verdicts have nothing to do with the exclusion set, and a
// test that can fail for unrelated reasons is a test nobody trusts.
//
// TWO INDEPENDENT PRODUCTION SITES FALSIFY HERE, BY TWO DIFFERENT MECHANISMS:
//
//	THE gate.FindUngatedArtifacts CALL — reverting its argument to
//	  artifact.NonCorpusDirs{} makes every planted file an ungated finding. That walk
//	  reads the PROJECT root and reports every artifact-shaped file whose parent is not
//	  root.Dir(kind), REGARDLESS of the file's validity.
//	THE ValidateConfig{…}.NonCorpus FIELD — subtler. A planted file that is VALID
//	  produces no per-artifact schema violation, so if the per-artifact pass were this
//	  field's only consumer, reverting it would be invisible. It is not: cfg.NonCorpus
//	  also reaches buildResolutionViolations, whose corpus resolution pass runs
//	  UNCONDITIONALLY. With the field reverted, vendor/SPEC-902 is discovered as a
//	  citer, its supports ref to a bundle absent from the fixture fails ResolveSupports,
//	  and an error violation naming SPEC-902's BASENAME lands in this step's output.
func TestGateSteps_ExclusionSetWiredIntoArtifactValidation(t *testing.T) {
	projectRoot := noncorpusFixtureRoot(t)

	scope, err := gate.ComputeGateScope(projectRoot, gate.GateScopeModeAll, nil)
	if err != nil {
		t.Fatalf("ComputeGateScope: %v", err)
	}

	steps := buildGateSteps(projectRoot, rootAtDir(t, projectRoot), scope)
	results := map[string]gate.StepResult{}
	for _, step := range steps {
		res := step(context.Background())
		results[res.StepName] = res
	}

	validation, ok := results[gate.StepArtifactValidation]
	if !ok {
		t.Fatalf("the assembled gate steps produced no %q result; got %v", gate.StepArtifactValidation, stepNamesOf(results))
	}

	for _, v := range validation.Violations {
		for _, basename := range plantedFixtureBasenames() {
			if strings.Contains(v.File, basename) || strings.Contains(v.Message, basename) {
				t.Errorf("the gate's artifact_validation step produced a violation naming %s, which sits inside a pack-declared dependency tree — the exclusion set is not threaded into this hop.\nrule=%s file=%s message=%s", basename, v.Rule, v.File, v.Message)
			}
		}
	}
}

func stepNamesOf(results map[string]gate.StepResult) []string {
	out := make([]string, 0, len(results))
	for name := range results {
		out = append(out, name)
	}
	return out
}

// TestCollectTraceRefs_HonorsPackDeclaredDependencyDirs (CLM-005): THE TRACEABILITY
// HOP, asserted at the only observable that can actually go red.
//
// THIS DELIBERATELY DOES NOT GO THROUGH THE requirement_traceability STEP. That step's
// output is STRUCTURALLY INSENSITIVE to the exclusion set, so a test written against it
// would pass identically with the wiring reverted — a falsifier that cannot fire. The
// mechanism: computeRequirementTraceabilitySurfaces emits violations ONLY through
// ClassifyRequirementTraceability(res.Records, refs); res.Records comes from
// ResolveArtifactStatus, which reads only root.Dir(kind) through a NON-RECURSIVE
// os.ReadDir that skips subdirectories outright; and ClassifyRequirementTraceability
// then DROPS any ref whose CitingPath is absent from those same records. A file planted
// at ANY nested path is therefore never a record, so its refs are discarded before they
// can add a violation OR suppress one. The violation set is identical either way.
//
// The observable that DOES move is the ref set collectTraceRefs returns.
//
// THE ZERO-VALUE HALF IS A BUILT-IN FALSIFIER: it proves the hop is genuinely sensitive
// to its parameter without anyone hand-reverting production code. It is also why the
// planted spec must carry a supports ref at all — with none, CollectSupportRefs yields
// zero refs, both halves would assert emptiness, and the test would be vacuous in
// precisely the way it exists to prevent.
//
// THE RESIDUAL, STATED RATHER THAN HIDDEN: this proves the hop CONSUMES the set it is
// handed. It CANNOT prove that buildGateSteps hands that hop the pack-derived set
// rather than a literal zero value, because — per the mechanism above — that difference
// has no behavioral consequence anywhere downstream, so no test in this repository can
// observe it. What guards that seam instead is the explicit-parameter rule (no
// variadic, no package-level default, so every call site must pass something a reviewer
// can see) plus the requirement that the set be derived ONCE into a single local in
// buildGateSteps and that SAME variable threaded to all five sites.
func TestCollectTraceRefs_HonorsPackDeclaredDependencyDirs(t *testing.T) {
	projectRoot := noncorpusFixtureRoot(t)
	root := rootAtDir(t, projectRoot)

	t.Run("with the production-built set the vendored citer contributes no ref", func(t *testing.T) {
		refs, err := collectTraceRefs(root, productionExclusionSet(t, projectRoot))
		if err != nil {
			t.Fatalf("collectTraceRefs: %v", err)
		}
		for _, ref := range refs {
			for _, tree := range []string{"vendor", "node_modules", "_thirdparty_deps"} {
				if pathHasDirSegment(ref.CitingPath, tree) {
					t.Errorf("collectTraceRefs returned a ref citing %s, which sits under the pack-declared dependency tree %q", ref.CitingPath, tree)
				}
			}
		}
	})

	t.Run("with the zero value the vendored citer's ref IS present", func(t *testing.T) {
		refs, err := collectTraceRefs(root, artifact.NonCorpusDirs{})
		if err != nil {
			t.Fatalf("collectTraceRefs: %v", err)
		}
		found := false
		for _, ref := range refs {
			if ref.BundleName == "absent-fixture-bundle" {
				found = true
			}
		}
		if !found {
			t.Errorf("with the zero value collectTraceRefs returned no ref naming bundle %q; the hop is not sensitive to its parameter, so the half above proves nothing. refs: %+v", "absent-fixture-bundle", refs)
		}
	})
}

// pathHasDirSegment reports whether path contains segment as a whole directory
// component, so `vendor` matches `a/vendor/b` but not a file merely named `vendored`.
func pathHasDirSegment(path, segment string) bool {
	for _, part := range strings.Split(filepath.ToSlash(path), "/") {
		if part == segment {
			return true
		}
	}
	return false
}
