package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/check"
	"github.com/bmanson/backstop-core/pkg/gate"
	"github.com/bmanson/backstop-core/pkg/pack"
)

// bunFixtureDir returns the in-repo static bun-toolchain fixture root (the project
// the live gate runs over: backstop.yml + src/*.ts + coverage/lcov.info).
func bunFixtureDir(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRoot(t), "cmd", "backstop", "testdata", "bun-toolchain")
}

// bunFixturePacksDir returns the fixture's .backstop/packs dir holding the bun pack.
func bunFixturePacksDir(t *testing.T) string {
	t.Helper()
	return filepath.Join(bunFixtureDir(t), ".backstop", "packs")
}

// TestBunFixture_StaticTestdataExistsWithPackTsFilesAndPrecapturedLcov proves the
// in-repo static fixture exists: a backstop.yml declaring backstop/bun-toolchain as
// an ordinary pack with NO language field (SPEC-046), the bun pack.yml, .ts source
// + .test.ts files, and a pre-captured lcov .info (CLM-022).
func TestBunFixture_StaticTestdataExistsWithPackTsFilesAndPrecapturedLcov(t *testing.T) {
	dir := bunFixtureDir(t)
	for _, rel := range []string{
		"backstop.yml",
		filepath.Join(".backstop", "packs", "backstop", "bun-toolchain", "pack.yml"),
		filepath.Join("src", "app.ts"),
		filepath.Join("src", "app.test.ts"),
		filepath.Join("coverage", "lcov.info"),
	} {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			t.Errorf("fixture must contain %s: %v", rel, err)
		}
	}
	cfg, err := os.ReadFile(filepath.Join(dir, "backstop.yml"))
	if err != nil {
		t.Fatalf("read backstop.yml: %v", err)
	}
	if !strings.Contains(string(cfg), "backstop/bun-toolchain") {
		t.Error("backstop.yml must declare backstop/bun-toolchain as a pack")
	}
	// SPEC-046: toolchain packs are ordinary declared packs — NO language field.
	for _, line := range strings.Split(string(cfg), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "language:") {
			t.Errorf("backstop.yml must NOT declare a language field (SPEC-046), got %q", strings.TrimSpace(line))
		}
	}
}

// bunCoverageRecords runs the LIVE coverage dispatch over the bun pack with the
// runner STUBBED (the real convert runs over the lcov FILE the engine declares as
// its stdout_artifact). projectRoot is where coverage/lcov.info is read from; the
// returned spy receives the convert's stdin so a test can prove the POSIX convert
// actually ran. No real bun/oxlint/tsc/prettier process is invoked — the
// fixtureRunner intercepts the engine command.
func bunCoverageRecords(t *testing.T, projectRoot string, runner *fixtureRunner, convertStdin *[]byte) []check.CoverageRecord {
	t.Helper()
	stubSandboxedRunStdout(t, convertStdin)
	records, err := dispatchPackCoverage([]*pack.Manifest{bunToolchainManifest(t)}, bunFixturePacksDir(t), projectRoot, nil, runner)
	if err != nil {
		t.Fatalf("dispatchPackCoverage over the bun fixture: %v", err)
	}
	return records
}

// TestBunFixture_GateMeasuresTsCoverageFromPrecapturedLcovRunnerStubbed is the
// load-bearing consumer proof (CLM-023): an executed gate over the fixture with the
// runner STUBBED runs the REAL convert over the pre-captured lcov and measures the
// .ts source file's line AND branch coverage end-to-end. The pre-captured lcov
// (app.ts line 9/10, branch 2/2) flows through dispatchPackCoverage's real convert
// into canonical records, then through the live coverage step to a PASS.
func TestBunFixture_GateMeasuresTsCoverageFromPrecapturedLcovRunnerStubbed(t *testing.T) {
	var convertStdin []byte
	runner := &fixtureRunner{}
	records := bunCoverageRecords(t, bunFixtureDir(t), runner, &convertStdin)

	if len(convertStdin) == 0 {
		t.Fatal("the real POSIX convert never ran over the lcov DATA — the e2e did not exercise the production convert path")
	}
	// The consumer reads BOTH bun metrics for the .ts source, end-to-end.
	line := recordFor(t, records, "src/app.ts", "line")
	branch := recordFor(t, records, "src/app.ts", "branch")
	if line.Covered != 9 || line.Total != 10 || !line.Measured {
		t.Errorf("line coverage must be measured 9/10 from the pre-captured lcov, got %#v", line)
	}
	if branch.Covered != 2 || branch.Total != 2 || !branch.Measured {
		t.Errorf("branch coverage must be measured 2/2 from the pre-captured lcov, got %#v", branch)
	}

	// Run the LIVE coverage step over the measured records: the .ts source's line+
	// branch both clear an 80% threshold, so the gate measures it green end-to-end.
	classifier := mergeSourceClassifier([]*pack.Manifest{bunToolchainManifest(t)})
	scope := &gate.GateScope{Mode: gate.GateScopeModeDiff, Files: []string{"src/app.ts"}, ProjectRoot: bunFixtureDir(t)}
	specs := []gate.SpecVerification{{SpecID: "FIXTURE", CoverageThreshold: 80}}
	res := gate.StepCoverageThresholdScopedFunc(records, specs, scope, classifier)(context.Background())
	if res.Status != "pass" {
		t.Errorf("the live coverage step must PASS for the measured .ts source (line 9/10, branch 2/2 ≥ 80%%), got %s: %#v", res.Status, res.Violations)
	}
}

// TestBunFixture_TsClassifiedMeasurableSourceViaDeclaredGlobsEndToEnd proves the
// .ts source is classified MEASURABLE source via the bun pack's declared globs in
// the LIVE gate path — the SPEC-043 merged classifier consumes the bun pack's globs
// (CLM-026). The classifier is built exactly as the live gate builds it
// (mergeSourceClassifier over the declared pack), and it drives which file the
// coverage step measures.
func TestBunFixture_TsClassifiedMeasurableSourceViaDeclaredGlobsEndToEnd(t *testing.T) {
	classifier := mergeSourceClassifier([]*pack.Manifest{bunToolchainManifest(t)})
	if !classifier.IsMeasurableSource("src/app.ts") {
		t.Error("the merged classifier must classify src/app.ts as measurable source via the bun pack's **/*.ts glob")
	}
	if classifier.IsMeasurableSource("src/app.test.ts") {
		t.Error("src/app.test.ts must NOT be measurable (test-wins-on-overlap) in the live classifier")
	}

	// End-to-end: the classifier drives coverage scope. With the green records and a
	// scope naming the .ts source, the step MEASURES it (the measurable-source
	// decision came from the declared globs, not a baked extension literal).
	var convertStdin []byte
	records := bunCoverageRecords(t, bunFixtureDir(t), &fixtureRunner{}, &convertStdin)
	scope := &gate.GateScope{Mode: gate.GateScopeModeDiff, Files: []string{"src/app.ts"}, ProjectRoot: bunFixtureDir(t)}
	res := gate.StepCoverageThresholdScopedFunc(records, []gate.SpecVerification{{SpecID: "FIXTURE", CoverageThreshold: 80}}, scope, classifier)(context.Background())
	if res.Status != "pass" {
		t.Errorf("the .ts source classified measurable must be measured-and-passing end-to-end, got %s: %#v", res.Status, res.Violations)
	}
}

// TestBunFixture_SeededUncoveredTsSourceRedsGateNotVacuousGreen proves the
// seeded-defect variant (a changed .ts source with NO coverage record in the lcov)
// REDs the gate loudly — the SPEC-043 anti-vacuous-green guard fires for the non-Go
// file (CLM-024). The seeded lcov measures only src/util.ts, so the changed
// src/app.ts has no record and must produce a blocking coverage_unmeasured
// violation, NOT a silent pass.
func TestBunFixture_SeededUncoveredTsSourceRedsGateNotVacuousGreen(t *testing.T) {
	// A temp project whose coverage/lcov.info IS the seeded-defect lcov (app.ts
	// absent). The convert reads projectRoot/coverage/lcov.info (the engine's
	// declared stdout_artifact), so swapping the file swaps the measured set.
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "coverage"), 0o755); err != nil {
		t.Fatal(err)
	}
	seeded, err := os.ReadFile(filepath.Join(bunFixtureDir(t), "coverage", "lcov-seeded-defect.info"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "coverage", "lcov.info"), seeded, 0o644); err != nil {
		t.Fatal(err)
	}
	// Materialize the changed source ON DISK: src/app.ts is a genuinely changed
	// (present) source with no coverage record — NOT a deleted file. The coverage
	// step's ISSUE-034 existence guard excludes only not-on-disk (deleted) paths, so
	// a faithful "changed but unmeasured" fixture must put the file on disk exactly
	// as the real working tree would; its absence-of-record (not absence-of-file) is
	// what must RED the gate.
	if err := os.MkdirAll(filepath.Join(tmp, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "src", "app.ts"), []byte("export const app = (n: number) => (n > 0 ? n : -n);\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var convertStdin []byte
	records := bunCoverageRecords(t, tmp, &fixtureRunner{}, &convertStdin)
	// The seeded lcov measures util.ts but NOT the changed app.ts.
	if _, ok := findRecordBySuffix(indexFirstRecordByPath(records), "app.ts"); ok {
		t.Fatal("the seeded-defect fixture must NOT carry a record for the changed src/app.ts")
	}

	classifier := mergeSourceClassifier([]*pack.Manifest{bunToolchainManifest(t)})
	scope := &gate.GateScope{Mode: gate.GateScopeModeDiff, Files: []string{"src/app.ts"}, ProjectRoot: tmp}
	res := gate.StepCoverageThresholdScopedFunc(records, []gate.SpecVerification{{SpecID: "FIXTURE", CoverageThreshold: 80}}, scope, classifier)(context.Background())
	if res.Status != "fail" {
		t.Fatalf("a changed measurable .ts source with NO coverage record must RED the gate (anti-vacuous-green), got %s", res.Status)
	}
	foundUnmeasured := false
	for _, v := range res.Violations {
		if v.Rule == "coverage_unmeasured" && strings.HasSuffix(v.File, "app.ts") {
			foundUnmeasured = true
		}
	}
	if !foundUnmeasured {
		t.Errorf("the RED must be a loud coverage_unmeasured violation for src/app.ts (not vacuous-green), got %#v", res.Violations)
	}
}

// TestBunFixture_NoRealBunOxlintTscPrettierInvokedInGoCI proves the fixture test
// invokes NO real bun/oxlint/tsc/prettier process — only the POSIX convert runs
// over the pre-captured lcov DATA, keeping backstop-core's Go CI bun-free (CLM-025).
// The fixtureRunner intercepts every engine command (no real binary is exec'd), the
// SARIF findings channel (dispatchPackEngines) is never touched, and the only tool
// the coverage dispatch tries to run is `bun` — through the stub.
func TestBunFixture_NoRealBunOxlintTscPrettierInvokedInGoCI(t *testing.T) {
	var convertStdin []byte
	runner := &fixtureRunner{}
	_ = bunCoverageRecords(t, bunFixtureDir(t), runner, &convertStdin)

	// The real POSIX convert ran over the lcov DATA (this is the ONLY process the
	// fixture exercises).
	if len(convertStdin) == 0 {
		t.Fatal("the POSIX convert must run over the lcov DATA")
	}
	// Every command the dispatch issued went through the STUB runner (no real exec),
	// and the ONLY tool was bun — never a real oxlint/tsc/prettier and never a real
	// bun binary (the fixtureRunner returns canned bytes, it spawns nothing).
	if len(runner.calls) == 0 {
		t.Fatal("the coverage dispatch must have issued its command through the stub runner")
	}
	for _, c := range runner.calls {
		if c.name != "bun" {
			t.Errorf("the coverage channel must only dispatch `bun` through the stub; saw %q — no real oxlint/tsc/prettier may run in Go CI", c.name)
		}
	}
}

// indexFirstRecordByPath collapses records to a path->record map (first record per
// path) for suffix lookups in the seeded-defect assertion.
func indexFirstRecordByPath(records []check.CoverageRecord) map[string]check.CoverageRecord {
	out := map[string]check.CoverageRecord{}
	for _, r := range records {
		if _, ok := out[r.Path]; !ok {
			out[r.Path] = r
		}
	}
	return out
}
