package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/gate"
	"github.com/backstop-ai/backstop-core/pkg/pack"
)

// SPEC-034 REQ-010 / N1 — INVOCATION equivalence for the `code check --file`
// go-test file-mode PACKAGE scoping. Parser equivalence (equivalence_test.go) is
// necessary but not sufficient: the standalone-hook path scopes `go test` to the
// changed file's PACKAGE (goPackageSelector / testExecutor.fileMode) under a tight
// time budget, not `./...`. Routing test through the bridge could silently switch
// it to a whole-module run (Sharp Edge 8). These tests pin the chosen outcome:
// file-mode scoping is PRESERVED through the engine path.

// goTestCall returns the recorded test-pass invocation from a fixture runner,
// failing if there is not exactly one.
//
// SELECTION IS RE-GROUNDED ON THE PRODUCER (ISSUE-067). The go-test binding now
// declares a producer, so the dispatch invokes the packRoot-resolved script path
// rather than the tool: a selector keyed on `c.name == "go"` finds NOTHING and the
// helper's own t.Fatalf fires. Both forms are accepted so the helper keeps working
// whichever way the binding is declared — what it PROVES is unchanged, because the
// evidence lives in `c.args`, which the producer swap deliberately leaves alone.
// Those arg assertions are the direct evidence for the arg-preservation half of
// ISSUE-067 CLM-001 and must not be weakened.
func goTestCall(t *testing.T, runner *fixtureRunner) fixtureCall {
	t.Helper()
	var found []fixtureCall
	for _, c := range runner.calls {
		if len(c.args) == 0 || c.args[0] != "test" {
			continue
		}
		isTool := c.name == "go"
		isProducer := producerCommandAlias()[filepath.Base(c.name)] == "go test"
		if isTool || isProducer {
			found = append(found, c)
		}
	}
	if len(found) != 1 {
		t.Fatalf("expected exactly one `go test` invocation, got %d: %+v", len(found), runner.calls)
	}
	return found[0]
}

// TestFileMode_TestPassScopedToChangedFilePackage proves the --file hook path
// scopes the test pass to the changed file's PACKAGE through the new engine path,
// preserving goPackageSelector file-mode scoping — the invocation targets the
// package (./pkg/widget), NOT ./... (CLM-034).
func TestFileMode_TestPassScopedToChangedFilePackage(t *testing.T) {
	m := onlyRules(goToolchainManifest(t), "go-test")
	sandboxRunner := directConvertSandboxRunner(nil)
	runner := &fixtureRunner{byCmd: map[string][]byte{"go test": readFixture(t, "go-test-failures.txt")}}

	// File-mode scope: a single changed file in pkg/widget.
	scope := &gate.GateScope{Mode: gate.GateScopeModeFile, Files: []string{"pkg/widget/widget_test.go"}}

	if _, err := dispatchPackEnginesWithEvidence([]*pack.Manifest{m}, goToolchainPacksDir(t), t.TempDir(), scope, runner, sandboxRunner); err != nil {
		t.Fatalf("dispatchPackEngines (file-mode test): %v", err)
	}

	call := goTestCall(t, runner)
	joined := strings.Join(call.args, " ")
	if strings.Contains(joined, "./...") {
		t.Errorf("file-mode `go test` must NOT run the whole module (./...); it must scope to the changed file's package; args=%v", call.args)
	}
	// goPackageSelector("pkg/widget/widget_test.go") == "./pkg/widget".
	if !strings.Contains(joined, "./pkg/widget") {
		t.Errorf("file-mode `go test` must target the changed file's package ./pkg/widget, got args=%v", call.args)
	}
}

// TestFileMode_NoSilentWholeModuleRegression proves a whole-module run in file
// mode is REJECTED as a regression: with a file-mode scope present, the go-test
// engine must not fall back to ./... . The decision to preserve scoping is
// explicit and tested; a silent ./... in file mode is a CLM-035 regression.
func TestFileMode_NoSilentWholeModuleRegression(t *testing.T) {
	m := onlyRules(goToolchainManifest(t), "go-test")
	sandboxRunner := directConvertSandboxRunner(nil)
	runner := &fixtureRunner{byCmd: map[string][]byte{"go test": readFixture(t, "go-test-failures.txt")}}

	scope := &gate.GateScope{Mode: gate.GateScopeModeFile, Files: []string{"pkg/gadget/gadget_test.go"}}
	if _, err := dispatchPackEnginesWithEvidence([]*pack.Manifest{m}, goToolchainPacksDir(t), t.TempDir(), scope, runner, sandboxRunner); err != nil {
		t.Fatalf("dispatchPackEngines: %v", err)
	}
	call := goTestCall(t, runner)
	for _, a := range call.args {
		if a == "./..." {
			t.Fatalf("file-mode test run silently regressed to whole-module ./...; args=%v", call.args)
		}
	}
	// ISSUE-172: the declared command now carries -coverprofile=cover.out (the
	// single-run convention), so the vector gained that flag. The CLAIM is unchanged
	// and is about the TARGET: exactly ./pkg/gadget, never ./... . Note what the flag
	// means here — a file-mode run writes a PARTIAL profile, which is precisely why
	// test-produce.sh stamps a profile as reusable ONLY after a `./...` run.
	if strings.Join(call.args, " ") != "test -coverprofile=cover.out ./pkg/gadget" {
		t.Errorf("file-mode test must target exactly ./pkg/gadget, got args=%v", call.args)
	}
}

// TestFileMode_ProjectWideModeStillWholeModule proves the file-mode scoping is
// SURGICAL: outside file mode (nil scope / mode all), the go-test engine still
// runs project-wide ./... so unchanged-file breakage keeps failing the gate. The
// file-mode notion must not leak into the whole-module path.
func TestFileMode_ProjectWideModeStillWholeModule(t *testing.T) {
	m := onlyRules(goToolchainManifest(t), "go-test")
	sandboxRunner := directConvertSandboxRunner(nil)
	runner := &fixtureRunner{byCmd: map[string][]byte{"go test": readFixture(t, "go-test-failures.txt")}}

	// A nil scope is the whole-repo path: it carries no file list, so the engine
	// is handed the projectRoot directory (ISSUE-091).
	if _, err := dispatchPackEnginesWithEvidence([]*pack.Manifest{m}, goToolchainPacksDir(t), t.TempDir(), nil, runner, sandboxRunner); err != nil {
		t.Fatalf("dispatchPackEngines: %v", err)
	}
	call := goTestCall(t, runner)
	// ISSUE-172: -coverprofile=cover.out rides on the declared command (the
	// single-run convention). The CLAIM is unchanged and is about the TARGET: ./... .
	if strings.Join(call.args, " ") != "test -coverprofile=cover.out ./..." {
		t.Errorf("project-wide test must target ./..., got args=%v", call.args)
	}
}
