package main

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/check"
	"github.com/backstop-ai/backstop-core/pkg/gate"
	"github.com/backstop-ai/backstop-core/pkg/pack"
)

// copyCoverageE2EFixture copies the static go-coverage-e2e fixture MODULE to a temp
// dir so the un-sandboxed producer's `go test -coverprofile` can write cover.out
// there without polluting the tracked testdata tree.
func copyCoverageE2EFixture(t *testing.T) string {
	t.Helper()
	src := filepath.Join(repoRoot(t), "cmd", "backstop", "testdata", "go-coverage-e2e")
	dst := t.TempDir()
	// Recursive copy via stdlib — no baked tool exec (backstop/self: no-baked-tool-exec).
	if err := filepath.WalkDir(src, func(p string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(src, p)
		if relErr != nil {
			return relErr
		}
		target := filepath.Join(dst, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		b, readErr := os.ReadFile(p)
		if readErr != nil {
			return readErr
		}
		return os.WriteFile(target, b, 0o644)
	}); err != nil {
		t.Fatalf("copy fixture module: %v", err)
	}
	return dst
}

// TestCoverageRealDispatch_E2E_FalsePositivesClearGenuineGapsFire is the acceptance-
// critical integration-gap closer (CLM-008): it drives the coverage path through the
// ACTUAL installed go-toolchain pack + the REAL sandbox-exec convert + a REAL runner
// running real `go test`/`go list` against a fixture Go MODULE — NOT direct record
// injection, NOT a fixtureRunner supplying a canned profile, NOT a stubbed convert.
// It reproduces both original false positives (a zero-statement file in a measured
// package; a measured root file colliding on basename with a nested file) plus an
// untested-with-statements file and a genuinely-unmeasured file, and asserts the false
// positives clear AND the genuine gaps still fire — proving the fix holds through the
// real path (CLM-001/002/003/004/006/008).
func TestCoverageRealDispatch_E2E_FalsePositivesClearGenuineGapsFire(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("the real sandbox-exec convert is macOS-only")
	}
	// Guards: the production seams must NOT be stubbed — a stub would game the only
	// safety net (mirrors dispatch_coverage_e2e_test.go).
	if dispatchPackEnginesFn != nil {
		t.Fatal("dispatchPackEnginesFn must be nil — the real-dispatch e2e must run the un-stubbed dispatch")
	}
	if sandboxedRunStdout != nil {
		t.Fatal("sandboxedRunStdout must be nil — the convert must run under the REAL sandbox, not a stub")
	}

	fixtureDir := copyCoverageE2EFixture(t)
	// A REAL runner (Dir = fixture module) so the producer's `go test`/`go list` see
	// the project — no fixtureRunner, no canned profile.
	runner := &check.ExecCommandRunner{Dir: fixtureDir}

	records, err := dispatchPackCoverage(
		[]*pack.Manifest{goToolchainCoverageManifest(t)},
		goToolchainPacksDir(t), fixtureDir, nil, runner,
	)
	if err != nil {
		t.Fatalf("real coverage dispatch over the installed pack: %v", err)
	}
	if len(records) == 0 {
		t.Fatal("the real producer+convert must yield coverage records")
	}

	// CLM-005/006: records are REPO-RELATIVE — the producer emitted #backstop-module
	// and the REAL sandboxed convert stripped it. No record carries the module prefix.
	for _, r := range records {
		if strings.Contains(r.Path, "example.com/covge") {
			t.Errorf("records must be repo-relative through the real path (module prefix stripped), got %q", r.Path)
		}
	}

	// Feed the real records into the LIVE coverage step with a diff scope covering the
	// fixture files and the live merged classifier (exactly as the gate builds it).
	classifier := mergeSourceClassifier([]*pack.Manifest{goToolchainManifest(t)})
	scope := &gate.GateScope{Mode: gate.GateScopeModeDiff, Files: []string{
		"embed.go", "cmd/x/embed.go", "types.go", "untested.go", "lonely/lonely.go",
	}, ProjectRoot: fixtureDir}
	specs := []gate.SpecVerification{{SpecID: "FIXTURE", CoverageThreshold: 50}}
	res := gate.StepCoverageThresholdScopedFunc(records, specs, scope, classifier)(context.Background())

	violationFor := func(file string) *gate.Violation {
		for i := range res.Violations {
			if res.Violations[i].File == file {
				return &res.Violations[i]
			}
		}
		return nil
	}
	unmeasuredFor := func(file string) bool {
		for _, v := range res.Violations {
			if v.Rule == "coverage_unmeasured" && v.File == file {
				return true
			}
		}
		return false
	}

	// CLM-003: the measured ROOT embed.go resolves to its OWN record under the
	// same-basename collision → ZERO coverage_unmeasured; 1/1 ≥ 50 so no threshold
	// violation either.
	if v := violationFor("embed.go"); v != nil {
		t.Errorf("measured root embed.go must produce NO violation (own 1/1 record under the collision), got [%s] %s", v.Rule, v.Message)
	}
	// The nested same-basename file is unaffected (measured 1/1).
	if v := violationFor("cmd/x/embed.go"); v != nil {
		t.Errorf("measured nested cmd/x/embed.go must produce NO violation, got [%s] %s", v.Rule, v.Message)
	}
	// CLM-001: the zero-statement file in a MEASURED package is N/A (total:0) → no
	// violation, through the REAL producer+convert.
	if v := violationFor("types.go"); v != nil {
		t.Errorf("zero-statement types.go must be N/A (no violation) through the real path, got [%s] %s", v.Rule, v.Message)
	}

	// CLM-002: the untested-with-statements file STILL fires (it has a 0/3 record, so
	// coverage_threshold, NOT coverage_unmeasured — it must NOT be N/A'd).
	if v := violationFor("untested.go"); v == nil {
		t.Errorf("untested-with-statements untested.go must STILL fire below-threshold, got %#v", res.Violations)
	} else if v.Rule != "coverage_threshold" {
		t.Errorf("untested.go has a 0/3 record so it must fire coverage_threshold, got %q", v.Rule)
	}

	// CLM-004: a genuinely-unmeasured file (its package emitted no profile block) STILL
	// fires coverage_unmeasured — the fix did not blind the genuine-gap check.
	if !unmeasuredFor("lonely/lonely.go") {
		t.Errorf("genuinely-unmeasured lonely/lonely.go must STILL fire coverage_unmeasured, got %#v", res.Violations)
	}

	// Non-vacuous: with genuine gaps present the step REDs.
	if res.Status != "fail" {
		t.Errorf("with genuine gaps present the coverage step must fail (non-vacuous), got %s: %#v", res.Status, res.Violations)
	}
}
