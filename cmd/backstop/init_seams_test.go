package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/check"
	"github.com/backstop-ai/backstop-core/pkg/initialize"
)

// init_seams_test.go drives initialize.NewRunner / initialize.Options with the REAL
// production adapters. NO fake from pkg/initialize appears anywhere in it.
//
// ★ WHY THIS FILE EXISTS AT ALL. The step tests in pkg/initialize prove each STEP's
// logic over fakes, which is the right shape for step logic and is genuinely not
// enough: a fake PackInstaller writes no backstop.lock, a fake RecipeApplier lands no
// byte, a fake GateRunner invents its own dimension counts. Every one of those seams
// reaches real, already-shipped machinery in production, and "unit tests nail the
// package while the cross-package wiring stays unproven" is this project's recurring
// integration-gap failure. THESE TESTS ARE THE WIRING PROOF.
//
// Referencing `initialize` here is also what satisfies the substantiveness subject join
// for the five claims whose declared subject stays pkg/initialize while their mandated
// tests live in cmd/backstop, where the production adapters are forced to live.

// initSeamsRunner assembles a Runner over the FOUR production adapters plus the real
// toolchain prober, exactly as newInitCommand does.
func initSeamsRunner(t *testing.T, projectRoot string) *initialize.Runner {
	t.Helper()
	runner, err := initialize.NewRunner(
		initPackInstaller{},
		initRecipeApplier{},
		initGateRunner{},
		&packToolchainProber{Runner: &check.ExecCommandRunner{Dir: projectRoot}},
		unavailableBaselineSeeder{},
	)
	if err != nil {
		t.Fatalf("assembling the production runner: %v", err)
	}
	return runner
}

// initSeamsHermeticPack publishes a fixture pack SOURCE as a hermetic remote and
// returns the pinned git ref plus a FRESH EMPTY project directory.
//
// The directory is empty on purpose: init writes the config itself, so a pre-seeded
// consumer project would hide whether it does.
func initSeamsHermeticPack(t *testing.T, fixture string) (packRef, projectRoot string) {
	t.Helper()

	source, err := filepath.Abs(filepath.Join("testdata", "hermetic-remote", fixture))
	if err != nil {
		t.Fatalf("resolving the %s fixture: %v", fixture, err)
	}
	remote := newHermeticRemote(t, source, "v1.0.0")
	redirectPackURL(t, remoteE2EOrg, fixture, remote.Path)
	// PROVE the redirect reached a child process before asserting on anything else. A
	// redirect that silently missed would be a green test talking to the network.
	assertPackURLRedirected(t, remoteE2EOrg, fixture, remote)

	return remoteE2EOrg + "/" + fixture + "@1.0.0", t.TempDir()
}

// initSeamsRun runs init over the production adapters and fails on an unexpected error.
func initSeamsRun(t *testing.T, projectRoot string, options initialize.Options) initialize.Result {
	t.Helper()
	options.ProjectRoot = projectRoot
	if options.Capabilities == nil {
		set, err := initialize.ResolveCapabilities(nil, nil)
		if err != nil {
			t.Fatalf("resolving capabilities: %v", err)
		}
		options.Capabilities = set
	}
	result, err := initSeamsRunner(t, projectRoot).Run(options)
	if err != nil {
		t.Fatalf("the production init run errored: %v", err)
	}
	return result
}

// ═══════════════════════════════════════════════════════════════════════════════
// REAL PackInstaller — a REAL backstop.lock, read off disk
// ═══════════════════════════════════════════════════════════════════════════════

// TestInit_GitRefPackInstallLandsAsAPortableLockEntry (SPEC-069 CLM-086).
//
// A pack installed by GENUINE GIT REF lands as a git-source lock entry carrying its
// source coordinate. The assertion is against THE FILE, not a returned struct: the
// COMMITTED lock is what REQ-018 is about, and a struct in memory proves nothing about
// what the next machine to clone this repository will read.
func TestInit_GitRefPackInstallLandsAsAPortableLockEntry(t *testing.T) {
	ref, project := initSeamsHermeticPack(t, "acceptance-lint-pack")

	result := initSeamsRun(t, project, initialize.Options{PackRefs: []string{ref}})

	packs := requireSeamStep(t, result, "packs")
	if packs.Outcome != initialize.OutcomeDelivered {
		t.Fatalf("the packs step reported %v: %s", packs.Outcome, packs.Detail)
	}

	entry := remoteE2ELockEntry(t, project, remoteE2EOrg+"/acceptance-lint-pack")
	if entry.SourceType != "git" {
		t.Fatalf("lock entry source_type = %q, want \"git\" — a remote install that recorded a local source proves nothing was cloned", entry.SourceType)
	}
	if entry.Version != "1.0.0" {
		t.Fatalf("lock entry version = %q, want 1.0.0", entry.Version)
	}
	if entry.GitRef == nil || *entry.GitRef != "v1.0.0" {
		t.Fatalf("lock entry git_ref = %v, want v1.0.0", entry.GitRef)
	}
	if entry.SourceCoordinate == "" {
		t.Fatal("the lock entry carries no source_coordinate; without it a fresh clone cannot resolve where the pack came from")
	}
	if entry.ContentHash == "" {
		t.Fatal("the lock entry carries no content hash; there would be nothing for a later install to verify against")
	}
}

// TestInit_ProducesNoLockEntryCarryingALocalPath (SPEC-069 CLM-088, denylist).
//
// Over the same REAL run, NO entry in the lock carries a `local_path`. The distribution
// lock records local_path only for a local source, so this is the assertion that init's
// refusal and the real installer's behavior actually COMPOSE into a portable lock — the
// property a fake installer cannot demonstrate, because it writes no lock at all.
func TestInit_ProducesNoLockEntryCarryingALocalPath(t *testing.T) {
	ref, project := initSeamsHermeticPack(t, "acceptance-lint-pack")

	initSeamsRun(t, project, initialize.Options{PackRefs: []string{ref}})

	body, err := os.ReadFile(filepath.Join(project, "backstop.lock"))
	if err != nil {
		t.Fatalf("reading backstop.lock: %v", err)
	}
	if len(body) == 0 {
		t.Fatal("backstop.lock is empty; an absence claim over an empty file asserts nothing")
	}
	if strings.Contains(string(body), "local_path") {
		t.Fatalf("the lock records a local_path, which is machine-specific and would not resolve on any other checkout.\n---\n%s", body)
	}
	if !strings.Contains(string(body), remoteE2EOrg+"/acceptance-lint-pack") {
		t.Fatalf("the lock does not record the installed pack at all, so the absence above is vacuous.\n---\n%s", body)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// REAL RecipeApplier — a REAL apply, landing REAL bytes
// ═══════════════════════════════════════════════════════════════════════════════

// recipePackPayload reads the pack's OWN declared payload out of the installed tree, so
// a byte comparison is against what the PACK shipped rather than against a literal this
// test invented.
func recipePackPayload(t *testing.T, project, packName string) []byte {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(project, ".backstop", "packs",
		filepath.FromSlash(packName), "recipes", "first-source", "payload.txt"))
	if err != nil {
		t.Fatalf("reading the installed pack's declared payload: %v", err)
	}
	return body
}

// TestInit_AppliesRecipesOnlyThroughTheShippedResolveApplyPath (SPEC-069 CLM-046).
//
// The target files written are EXACTLY the ones the RECIPE declared. Init contributes
// no path: the recipe names `src/first-source.txt` and that is precisely where the file
// lands, with nothing else appearing anywhere.
func TestInit_AppliesRecipesOnlyThroughTheShippedResolveApplyPath(t *testing.T) {
	ref, project := initSeamsHermeticPack(t, "scaffold-recipe-pack")
	packName := remoteE2EOrg + "/scaffold-recipe-pack"

	result := initSeamsRun(t, project, initialize.Options{
		PackRefs:    []string{ref},
		CIRecipeRef: packName + ":first-source@1.0.0",
	})

	ci := requireSeamStep(t, result, "ci")
	if ci.Outcome != initialize.OutcomeDelivered {
		t.Fatalf("the CI step reported %v over a real apply: %s", ci.Outcome, ci.Detail)
	}

	// The RECIPE's declared target, and it landed there.
	declared := filepath.Join(project, "src", "first-source.txt")
	if _, err := os.Stat(declared); err != nil {
		t.Fatalf("the recipe's declared target %s did not land: %v", declared, err)
	}
	// And nothing ELSE was written under the recipe's directory: init contributes no
	// path of its own, so a second file would mean core invented one.
	entries, err := os.ReadDir(filepath.Join(project, "src"))
	if err != nil {
		t.Fatalf("reading the recipe's target directory: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "first-source.txt" {
		names := []string{}
		for _, entry := range entries {
			names = append(names, entry.Name())
		}
		t.Fatalf("the apply wrote %v; the recipe declares exactly one target and init contributes no path", names)
	}
}

// TestInit_RecipePayloadLandsByteIdentical (SPEC-069 CLM-047).
//
// The bytes on disk equal the PACK's payload bytes EXACTLY. The comparison is over raw
// byte slices with NO normalization, NO trimming and NO line-ending tolerance —
// tolerance here would hide precisely the rendering or rewriting this claim forbids.
func TestInit_RecipePayloadLandsByteIdentical(t *testing.T) {
	ref, project := initSeamsHermeticPack(t, "scaffold-recipe-pack")
	packName := remoteE2EOrg + "/scaffold-recipe-pack"

	initSeamsRun(t, project, initialize.Options{
		PackRefs:    []string{ref},
		CIRecipeRef: packName + ":first-source@1.0.0",
	})

	landed, err := os.ReadFile(filepath.Join(project, "src", "first-source.txt"))
	if err != nil {
		t.Fatalf("reading the landed target: %v", err)
	}
	declared := recipePackPayload(t, project, packName)

	if len(declared) == 0 {
		t.Fatal("the pack's declared payload is empty; a byte-identity claim over empty bytes asserts nothing")
	}
	if string(landed) != string(declared) {
		t.Fatalf("the payload did not land byte-identical.\ndeclared (%d bytes): %q\nlanded   (%d bytes): %q",
			len(declared), declared, len(landed), landed)
	}
}

// TestInit_ScaffoldRecipeTargetsLandAtTheRecipeDeclaredPaths (SPEC-069 CLM-130).
//
// The SAME two properties, reached through the SCAFFOLD step's ref rather than the CI
// step's — proving ONE seam with TWO callers reaches the same shipped path. A scaffold
// step that had grown its own apply would pass every CI-side claim and fail here.
func TestInit_ScaffoldRecipeTargetsLandAtTheRecipeDeclaredPaths(t *testing.T) {
	ref, project := initSeamsHermeticPack(t, "scaffold-recipe-pack")
	packName := remoteE2EOrg + "/scaffold-recipe-pack"

	result := initSeamsRun(t, project, initialize.Options{
		PackRefs:          []string{ref},
		ScaffoldRecipeRef: packName + ":first-source@1.0.0",
	})

	scaffold := requireSeamStep(t, result, "scaffold")
	if scaffold.Outcome != initialize.OutcomeDelivered {
		t.Fatalf("the scaffold step reported %v over a real apply: %s", scaffold.Outcome, scaffold.Detail)
	}

	landed, err := os.ReadFile(filepath.Join(project, "src", "first-source.txt"))
	if err != nil {
		t.Fatalf("the recipe's declared target did not land through the SCAFFOLD step: %v", err)
	}
	if string(landed) != string(recipePackPayload(t, project, packName)) {
		t.Fatal("the scaffold step's apply did not land the payload byte-identical, so the two callers do not reach the same shipped path")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// REAL GateRunner, REAL BaselineSeeder, and the fail-closed constructor
// ═══════════════════════════════════════════════════════════════════════════════

// TestInitSeams_GateRunnerReducesARealGateRunToDimensionCounts is deliberately NOT a
// mandated name: CLM-064's mandated test covers the REDUCTION logic over a fake.
//
// This one proves the other half — that the production adapter runs the SAME assembled
// kill chain `backstop gate` runs, and that its DimensionCounts are derived from a real
// GateResult's steps with dimension names that are backstop's own vocabulary. It runs
// over a project with a pack genuinely installed, so at least one dimension is really
// populated: a reduction asserted over an empty gate result asserts nothing.
func TestInitSeams_GateRunnerReducesARealGateRunToDimensionCounts(t *testing.T) {
	ref, project := initSeamsHermeticPack(t, "acceptance-lint-pack")

	result := initSeamsRun(t, project, initialize.Options{PackRefs: []string{ref}})

	if len(result.Observations) == 0 {
		t.Fatal("a real gate run produced no dimension counts at all")
	}

	// The dimension names are the gate's OWN step vocabulary, which is what keeps this
	// reduction free of any tool or language noun.
	seen := map[string]bool{}
	for _, observation := range result.Observations {
		if strings.TrimSpace(observation.Dimension) == "" {
			t.Fatalf("a dimension count carries an empty name: %+v", observation)
		}
		if seen[observation.Dimension] {
			t.Fatalf("dimension %q appears twice; the reduction is one count PER dimension", observation.Dimension)
		}
		seen[observation.Dimension] = true
		if observation.Count < 0 {
			t.Fatalf("dimension %q reports a negative count", observation.Dimension)
		}
	}
	// Anchored on dimensions the shipped chain always assembles, so a reduction over
	// some other step list would not pass.
	for _, expected := range []string{"artifact_validation", "waiver_resolution"} {
		if !seen[expected] {
			t.Fatalf("the reduction is missing the shipped gate dimension %q; it did not observe the chain `backstop gate` runs.\nsaw: %v", expected, seen)
		}
	}
}

// TestInitSeams_ProductionBaselineSeederReportsCapabilityAbsent is also not a mandated
// name. It is the PRODUCTION half of CLM-062.
//
// The production seeder returns the sentinel and NOTHING ELSE happens. This is what
// keeps the ISSUE-056 boundary honest now that a concrete type exists to be tempted
// into filling it.
func TestInitSeams_ProductionBaselineSeederReportsCapabilityAbsent(t *testing.T) {
	project := t.TempDir()

	path, err := unavailableBaselineSeeder{}.Seed(project)
	if !errors.Is(err, initialize.ErrBaselineSeedingUnavailable) {
		t.Fatalf("the production seeder returned %v, want ErrBaselineSeedingUnavailable (matched with errors.Is, never a string compare)", err)
	}
	if path != "" {
		t.Fatalf("the production seeder returned the path %q; it seeds nothing, so it has no path to report", path)
	}

	// And over a FULL init run: nothing wrote a baseline or a fingerprint of any kind.
	initSeamsRun(t, project, initialize.Options{})

	if _, statErr := os.Stat(filepath.Join(project, ".backstop", "baseline.json")); !os.IsNotExist(statErr) {
		t.Fatalf(".backstop/baseline.json exists after a full init run (stat error: %v); this command builds no seeding machinery", statErr)
	}
	walkErr := filepath.Walk(project, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		name := strings.ToLower(filepath.Base(path))
		if strings.Contains(name, "baseline") || strings.Contains(name, "fingerprint") {
			t.Fatalf("init wrote %s; nothing in the init source set may write a baseline or compute a fingerprint", path)
		}
		return nil
	})
	if walkErr != nil {
		t.Fatalf("walking the initialized project: %v", walkErr)
	}
}

// TestInitSeams_NewRunnerRefusesEveryNilDependencyByName asserts the fail-closed
// constructor holds IN PRODUCTION SHAPE.
//
// Five sub-cases, one per seam, each passing the four real adapters and one nil. This
// is the property that makes a half-wired runner unconstructable, and it is worth
// asserting where the real adapters are rather than only over fakes.
func TestInitSeams_NewRunnerRefusesEveryNilDependencyByName(t *testing.T) {
	project := t.TempDir()
	prober := &packToolchainProber{Runner: &check.ExecCommandRunner{Dir: project}}

	cases := []struct {
		name    string
		build   func() (*initialize.Runner, error)
		mustSay string
	}{
		{"nil PackInstaller", func() (*initialize.Runner, error) {
			return initialize.NewRunner(nil, initRecipeApplier{}, initGateRunner{}, prober, unavailableBaselineSeeder{})
		}, "PackInstaller"},
		{"nil RecipeApplier", func() (*initialize.Runner, error) {
			return initialize.NewRunner(initPackInstaller{}, nil, initGateRunner{}, prober, unavailableBaselineSeeder{})
		}, "RecipeApplier"},
		{"nil GateRunner", func() (*initialize.Runner, error) {
			return initialize.NewRunner(initPackInstaller{}, initRecipeApplier{}, nil, prober, unavailableBaselineSeeder{})
		}, "GateRunner"},
		{"nil ToolchainProber", func() (*initialize.Runner, error) {
			return initialize.NewRunner(initPackInstaller{}, initRecipeApplier{}, initGateRunner{}, nil, unavailableBaselineSeeder{})
		}, "ToolchainProber"},
		{"nil BaselineSeeder", func() (*initialize.Runner, error) {
			return initialize.NewRunner(initPackInstaller{}, initRecipeApplier{}, initGateRunner{}, prober, nil)
		}, "BaselineSeeder"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			runner, err := tc.build()
			if err == nil {
				t.Fatalf("NewRunner accepted a nil %s; a half-wired runner must be unconstructable, not a runtime nil-deref", tc.mustSay)
			}
			if runner != nil {
				t.Fatal("NewRunner returned a runner alongside its error")
			}
			if !strings.Contains(err.Error(), tc.mustSay) {
				t.Fatalf("the refusal does not NAME the nil dependency.\nwant to contain: %s\ngot:             %s", tc.mustSay, err.Error())
			}
		})
	}
}

// requireSeamStep returns the named step's report or fails.
func requireSeamStep(t *testing.T, result initialize.Result, name string) initialize.StepReport {
	t.Helper()
	for _, step := range result.Steps {
		if step.Step == name {
			return step
		}
	}
	names := []string{}
	for _, step := range result.Steps {
		names = append(names, step.Step)
	}
	t.Fatalf("no report for step %q; the run reported %v", name, names)
	return initialize.StepReport{}
}
