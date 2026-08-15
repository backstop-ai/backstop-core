package initialize

import (
	"os"
	"path/filepath"
	"testing"
)

// ONE shared set of fakes for the five seams, used by every step test in this
// package and by the runner test. One set, in one file, so the package has no
// duplicate fake types and no per-file drift.
//
// ★ WHAT THESE FAKES ARE FOR, AND WHAT THEY CANNOT PROVE.
//
// They exist so each STEP's own logic — ordering, subtraction, classification,
// report wording, the converge-never-clobber posture — is testable without a
// network, a clone or a real gate run. They prove NOTHING about the machinery behind
// the seam: a fake installer writes no `backstop.lock`, a fake applier lands no byte,
// a fake gate runner invents its own dimension counts.
//
// The five claims that assert REAL machinery (CLM-046, CLM-047, CLM-086, CLM-088,
// CLM-130) are therefore written in cmd/backstop/init_seams_test.go against the REAL
// adapters, over a pack installed by genuine git ref. If a step test in this package
// ever seems to prove one of those five, the fake has grown behavior it should not
// have.

// fakePackInstaller records the refs it was asked to install, in call order.
type fakePackInstaller struct {
	// calls holds one entry per Install call, in order, as "<projectRoot>|<ref>".
	calls []string
	// refs holds just the refs, in order — the shape most assertions want.
	refs []string
	// failures maps a ref to the error Install should return for it.
	failures map[string]error
	// onInstall runs at Install time. It is what lets a test MATERIALIZE a pack the
	// way the real installer does — a manifest under .backstop/packs/ plus the
	// backstop.yml entry — so a downstream step has something real to read. Without
	// it an ordering claim could only assert that the fake was called, which says
	// nothing about the order two steps actually ran in.
	onInstall func(projectRoot, ref string)
}

func (f *fakePackInstaller) Install(projectRoot string, ref string) error {
	f.calls = append(f.calls, projectRoot+"|"+ref)
	f.refs = append(f.refs, ref)
	if err, failing := f.failures[ref]; failing {
		return err
	}
	if f.onInstall != nil {
		f.onInstall(projectRoot, ref)
	}
	return nil
}

// applyCall is one recorded RecipeApplier invocation.
type applyCall struct {
	ProjectRoot string
	Ref         string
}

// fakeRecipeApplier records every apply and returns a per-ref scripted outcome.
//
// It is deliberately keyed by REF rather than by step: ONE seam serves both the CI
// step and the scaffold step, so a fake that could tell them apart would let a test
// assert something the production seam cannot see.
type fakeRecipeApplier struct {
	calls    []applyCall
	outcomes map[string]ApplyOutcome
	failures map[string]error
	// defaultOutcome is returned for a ref with no scripted entry.
	defaultOutcome ApplyOutcome
	// onApply runs at Apply time, so a test can LAND the recipe's declared target on
	// disk and then observe when a later step sees it.
	onApply func(projectRoot, ref string)
}

func (f *fakeRecipeApplier) Apply(projectRoot string, ref string) (ApplyOutcome, error) {
	f.calls = append(f.calls, applyCall{ProjectRoot: projectRoot, Ref: ref})
	if err, failing := f.failures[ref]; failing {
		return ApplyOutcome{}, err
	}
	if f.onApply != nil {
		f.onApply(projectRoot, ref)
	}
	if outcome, scripted := f.outcomes[ref]; scripted {
		return outcome, nil
	}
	return f.defaultOutcome, nil
}

// fakeGateRunner returns scripted dimension counts and records its call count.
type fakeGateRunner struct {
	calls  int
	counts []DimensionCount
	err    error
}

func (f *fakeGateRunner) Run(projectRoot string) ([]DimensionCount, error) {
	f.calls++
	return f.counts, f.err
}

// fakeToolchainProber returns scripted step reports and records its call count.
//
// onProbe runs at Probe time, which is what lets a test observe the FILESYSTEM STATE
// at the toolchain step's boundary — the only way to assert that the scaffolded file
// is already on disk when the toolchain step executes (CLM-138). Asserting the
// step-name list alone is satisfiable vacuously by the very refactor that claim
// exists to catch.
type fakeToolchainProber struct {
	calls   int
	reports []StepReport
	err     error
	onProbe func(projectRoot string)
}

func (f *fakeToolchainProber) Probe(projectRoot string) ([]StepReport, error) {
	f.calls++
	if f.onProbe != nil {
		f.onProbe(projectRoot)
	}
	return f.reports, f.err
}

// fakeBaselineSeeder returns a scripted seed path or error and counts its calls.
type fakeBaselineSeeder struct {
	calls int
	path  string
	err   error
}

func (f *fakeBaselineSeeder) Seed(projectRoot string) (string, error) {
	f.calls++
	return f.path, f.err
}

// unavailableSeeder is the fake standing in for production's
// unavailableBaselineSeeder: it returns the sentinel and nothing else.
//
// "No seeder available" CANNOT be a nil field — NewRunner is fail-closed, so a nil
// seam is unconstructable by design — so it is a VALUE the seam returns.
type unavailableSeeder struct{ calls int }

func (u *unavailableSeeder) Seed(projectRoot string) (string, error) {
	u.calls++
	return "", ErrBaselineSeedingUnavailable
}

// newTestRunner assembles a Runner over the five fakes, failing the test when the
// fail-closed constructor refuses. Every step test builds its runner through this so
// no test accidentally exercises a half-wired one.
func newTestRunner(t *testing.T, packs PackInstaller, recipes RecipeApplier, gates GateRunner, tools ToolchainProber, seeds BaselineSeeder) *Runner {
	t.Helper()
	runner, err := NewRunner(packs, recipes, gates, tools, seeds)
	if err != nil {
		t.Fatalf("NewRunner over the shared fakes errored: %v", err)
	}
	return runner
}

// defaultFakes returns a fully-wired set of inert fakes: nothing installed, nothing
// applied, no findings, no entrypoints, no seeder. A test overrides only the seam it
// is about.
func defaultFakes() (*fakePackInstaller, *fakeRecipeApplier, *fakeGateRunner, *fakeToolchainProber, *unavailableSeeder) {
	return &fakePackInstaller{failures: map[string]error{}},
		&fakeRecipeApplier{outcomes: map[string]ApplyOutcome{}, failures: map[string]error{}},
		&fakeGateRunner{},
		&fakeToolchainProber{},
		&unavailableSeeder{}
}

// allCapabilities is the resolved default set, for a test that wants a bare run.
func allCapabilities(t *testing.T) map[Capability]bool {
	t.Helper()
	set, err := ResolveCapabilities(nil, nil)
	if err != nil {
		t.Fatalf("resolving the default capability set: %v", err)
	}
	return set
}

// capabilitiesExcept is the default set minus the named capabilities.
func capabilitiesExcept(t *testing.T, excluded ...string) map[Capability]bool {
	t.Helper()
	set, err := ResolveCapabilities(nil, excluded)
	if err != nil {
		t.Fatalf("resolving the default set minus %v: %v", excluded, err)
	}
	return set
}

// capabilitiesOnly is exactly the named capabilities.
func capabilitiesOnly(t *testing.T, only ...string) map[Capability]bool {
	t.Helper()
	set, err := ResolveCapabilities(only, nil)
	if err != nil {
		t.Fatalf("resolving --only %v: %v", only, err)
	}
	return set
}

// stepNames renders a Result's step reports as the ordered list of step names.
func stepNames(steps []StepReport) []string {
	names := make([]string, 0, len(steps))
	for _, step := range steps {
		names = append(names, step.Step)
	}
	return names
}

// findStep returns the first report for the named step, and whether one exists.
func findStep(steps []StepReport, name string) (StepReport, bool) {
	for _, step := range steps {
		if step.Step == name {
			return step, true
		}
	}
	return StepReport{}, false
}

// requireStep returns the named step report or fails the test.
func requireStep(t *testing.T, steps []StepReport, name string) StepReport {
	t.Helper()
	step, found := findStep(steps, name)
	if !found {
		t.Fatalf("no report for step %q; the run reported %v", name, stepNames(steps))
	}
	return step
}

// snapshotTree reads every regular file under root and returns path -> bytes, so a
// test can assert that a second run changed nothing.
func snapshotTree(t *testing.T, root string) map[string]string {
	t.Helper()
	snapshot := map[string]string{}
	walkErr := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		body, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		snapshot[rel] = string(body)
		return nil
	})
	if walkErr != nil {
		t.Fatalf("snapshotting %s: %v", root, walkErr)
	}
	return snapshot
}

// writeFile writes body at root/rel, creating parents, and fails the test on error.
func writeFile(t *testing.T, root, rel, body string) string {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("creating parent of %s: %v", full, err)
	}
	if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
		t.Fatalf("writing %s: %v", full, err)
	}
	return full
}

// readFile reads root/rel, failing the test on error.
func readFile(t *testing.T, root, rel string) string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("reading %s: %v", rel, err)
	}
	return string(body)
}

// exists reports whether root/rel is present on disk.
func exists(root, rel string) bool {
	_, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel)))
	return err == nil
}
