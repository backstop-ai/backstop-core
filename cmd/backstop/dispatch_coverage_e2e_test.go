package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack"
)

// goToolchainCoverageManifest loads the INSTALLED go-toolchain pack and keeps only
// its go-coverage rule, so the e2e drives the coverage engine in isolation through
// the real installed-pack convert.
func goToolchainCoverageManifest(t *testing.T) *pack.Manifest {
	t.Helper()
	return onlyRules(goToolchainManifest(t), "go-coverage")
}

// TestGoToolchain_CoverageEngineRealEndToEndOverInstalledPack proves the INSTALLED
// go-toolchain pack's coverage engine runs through the UN-STUBBED sandboxed dispatch
// — the real dispatchPackCoverage executes the pack's REAL
// scripts/coverage-to-records.sh via resolveSandboxedRunStdout (convert NOT stubbed,
// NO parallel raw-exec) and produces real per-file CoverageRecords. The pack now
// declares a producer (ISSUE-045), so the engine's payload is its declared
// stdout_artifact (cover.out) — the producer would write it un-sandboxed; here the
// Phase-1 profile fixture is placed at cover.out to stand in for the producer's
// output while the CONVERT stays the real script on disk. A stubbed convert returning
// canned records would NOT satisfy this — the test asserts the real script actually
// ran by checking its real per-file aggregation (CLM-024).
func TestGoToolchain_CoverageEngineRealEndToEndOverInstalledPack(t *testing.T) {
	// Guard: the production dispatch seam must NOT be stubbed — a stubbed dispatcher
	// would game the only safety net (mirrors the golden-equivalence guard).
	if dispatchPackEnginesFn != nil {
		t.Fatal("dispatchPackEnginesFn must be nil — the coverage e2e must run the REAL un-stubbed dispatch, not a stub")
	}
	// The installed pack + its real convert script must exist ON DISK.
	convertScript := filepath.Join(goToolchainPackRoot(t), "scripts", "coverage-to-records.sh")
	if _, err := os.ReadFile(convertScript); err != nil {
		t.Fatalf("the go-toolchain coverage convert script must be installed on disk: %v", err)
	}

	// stubSandboxedRunStdout shells the REAL convert script directly (the same
	// production-equivalent seam the ast-grep e2e uses) — the convert is NOT replaced
	// by a canned-records stub. Spy the stdin it receives to prove the convert pipe ran.
	var convertStdin []byte
	sandboxRunner := directConvertSandboxRunner(&convertStdin)

	// The engine now sources its payload from the declared stdout_artifact the producer
	// writes; place the Phase-1 profile fixture at cover.out to stand in for the
	// producer's output. The fake runner intercepts the producer exec (returns nothing
	// and writes nothing), so the pre-placed cover.out is what the real convert reads.
	profile := readCoverageProfileFixture(t, "cover-combined.out")
	projectRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectRoot, "cover.out"), profile, 0o644); err != nil {
		t.Fatalf("seed cover.out artifact: %v", err)
	}
	runner := &fixtureRunner{}

	result, err := dispatchPackCoverageWithEvidence([]*pack.Manifest{goToolchainCoverageManifest(t)}, goToolchainPacksDir(t), projectRoot, nil, runner, sandboxRunner)
	if err != nil {
		t.Fatalf("real un-stubbed coverage dispatch over installed pack: %v", err)
	}

	// The real convert pipe was exercised — it received the raw Go profile on stdin.
	if len(convertStdin) == 0 {
		t.Fatal("the sandboxed-convert seam was never reached — the real coverage-to-records.sh did not run; the e2e did not exercise the production path")
	}
	if string(convertStdin) != string(profile) {
		t.Errorf("the real convert must receive the raw Go profile stdout (un-stubbed), got %q", string(convertStdin))
	}

	// Real per-file records came back, stamped statement (Go's granularity) — proof
	// the real aggregation ran, not a canned stub. The combined profile has 3 files.
	if len(result.Records) != 3 || !result.NativeSandboxApplied {
		t.Fatalf("expected 3 acknowledged records from the real convert, got %#v", result)
	}
	for _, r := range result.Records {
		if r.Metric != "statement" {
			t.Errorf("real go-coverage records must carry metric statement, got %q for %q", r.Metric, r.Path)
		}
		if !r.Measured {
			t.Errorf("every profiled file is measured; %q has Measured=false", r.Path)
		}
	}
}

// TestGoToolchain_RealEndToEndRecordsDriveCorrectGateVerdict proves the real-e2e
// records the gate consumes include a MEASURED-AND-PASSING file, a
// MEASURED-AND-FAILING (below-threshold) file, and a Total==0 (N/A) file, and the
// gate's verdict over them is correct (the failing file REDs; the passing and N/A
// files do not) — proving the producer feeds a NON-VACUOUS consumer over the real
// convert (CLM-025). The verdict is checked at the producer→consumer boundary
// established in Phase 3 (the minimal test-local consumer), NOT by re-authoring
// SPEC-041's step.
func TestGoToolchain_RealEndToEndRecordsDriveCorrectGateVerdict(t *testing.T) {
	sandboxRunner := directConvertSandboxRunner(nil)
	// The engine sources its payload from the declared stdout_artifact the producer
	// writes (ISSUE-045); seed cover.out with the Phase-1 profile fixture as the
	// producer's stand-in output. The fake runner intercepts the producer exec.
	profile := readCoverageProfileFixture(t, "cover-combined.out")
	projectRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectRoot, "cover.out"), profile, 0o644); err != nil {
		t.Fatalf("seed cover.out artifact: %v", err)
	}
	runner := &fixtureRunner{}

	result, err := dispatchPackCoverageWithEvidence([]*pack.Manifest{goToolchainCoverageManifest(t)}, goToolchainPacksDir(t), projectRoot, nil, runner, sandboxRunner)
	if err != nil {
		t.Fatalf("real un-stubbed coverage dispatch: %v", err)
	}

	consumer := minimalCoverageConsumer{thresholdPct: 80}
	lines := consumer.verdict(result.Records)

	verdictBySuffix := func(suffix string) (coverageReportLine, bool) {
		for _, ln := range lines {
			if strings.HasSuffix(ln.Path, suffix) {
				return ln, true
			}
		}
		return coverageReportLine{}, false
	}

	// Measured-and-PASSING (9/10 = 90% >= 80%): not a shortfall, not N/A.
	passing, ok := verdictBySuffix("passing.go")
	if !ok {
		t.Fatalf("the real records must include the measured-and-passing file, got %#v", result.Records)
	}
	if passing.Shortfall || passing.NA {
		t.Errorf("the measured-and-passing file must NOT red and is NOT N/A, got %+v", passing)
	}

	// Measured-and-FAILING (4/10 = 40% < 80%): a shortfall (REDs).
	failing, ok := verdictBySuffix("failing.go")
	if !ok {
		t.Fatalf("the real records must include the measured-and-failing file")
	}
	if !failing.Shortfall {
		t.Errorf("the measured-and-failing file MUST red (below threshold), got %+v", failing)
	}
	if failing.NA {
		t.Errorf("a below-threshold file is NOT N/A, got %+v", failing)
	}

	// Total==0 (no executable lines): N/A, never a 0%-fail.
	na, ok := verdictBySuffix("iface.go")
	if !ok {
		t.Fatalf("the real records must include the Total==0 (N/A) file")
	}
	if !na.NA {
		t.Errorf("the no-executable-lines file MUST be N/A, got %+v", na)
	}
	if na.Shortfall {
		t.Errorf("the Total==0 file must NEVER red as a 0%%-fail, got %+v", na)
	}

	// Non-vacuous: exactly one shortfall (the failing file), proving the consumer
	// actually discriminated rather than passing/failing everything uniformly.
	shortfalls := 0
	for _, ln := range lines {
		if ln.Shortfall {
			shortfalls++
		}
	}
	if shortfalls != 1 {
		t.Errorf("the verdict must be non-vacuous: exactly one shortfall (failing.go), got %d", shortfalls)
	}
}
