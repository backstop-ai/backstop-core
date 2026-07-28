package main

import (
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

// goTestCall returns the recorded `go test <target>` invocation from a fixture
// runner, failing if there is not exactly one.
func goTestCall(t *testing.T, runner *fixtureRunner) fixtureCall {
	t.Helper()
	var found []fixtureCall
	for _, c := range runner.calls {
		if c.name == "go" && len(c.args) > 0 && c.args[0] == "test" {
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
	stubSandboxedRunStdout(t, nil)
	runner := &fixtureRunner{byCmd: map[string][]byte{"go test": readFixture(t, "go-test-failures.txt")}}

	// File-mode scope: a single changed file in pkg/widget.
	scope := &gate.GateScope{Mode: gate.GateScopeModeFile, Files: []string{"pkg/widget/widget_test.go"}}

	if _, err := dispatchPackEngines([]*pack.Manifest{m}, goToolchainPacksDir(t), t.TempDir(), scope, runner); err != nil {
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
	stubSandboxedRunStdout(t, nil)
	runner := &fixtureRunner{byCmd: map[string][]byte{"go test": readFixture(t, "go-test-failures.txt")}}

	scope := &gate.GateScope{Mode: gate.GateScopeModeFile, Files: []string{"pkg/gadget/gadget_test.go"}}
	if _, err := dispatchPackEngines([]*pack.Manifest{m}, goToolchainPacksDir(t), t.TempDir(), scope, runner); err != nil {
		t.Fatalf("dispatchPackEngines: %v", err)
	}
	call := goTestCall(t, runner)
	for _, a := range call.args {
		if a == "./..." {
			t.Fatalf("file-mode test run silently regressed to whole-module ./...; args=%v", call.args)
		}
	}
	if strings.Join(call.args, " ") != "test ./pkg/gadget" {
		t.Errorf("file-mode test must target exactly ./pkg/gadget, got args=%v", call.args)
	}
}

// TestFileMode_ProjectWideModeStillWholeModule proves the file-mode scoping is
// SURGICAL: outside file mode (nil scope / mode all), the go-test engine still
// runs project-wide ./... so unchanged-file breakage keeps failing the gate. The
// file-mode notion must not leak into the whole-module path.
func TestFileMode_ProjectWideModeStillWholeModule(t *testing.T) {
	m := onlyRules(goToolchainManifest(t), "go-test")
	stubSandboxedRunStdout(t, nil)
	runner := &fixtureRunner{byCmd: map[string][]byte{"go test": readFixture(t, "go-test-failures.txt")}}

	// nil scope == whole-repo escape hatch (gate --all / code check).
	if _, err := dispatchPackEngines([]*pack.Manifest{m}, goToolchainPacksDir(t), t.TempDir(), nil, runner); err != nil {
		t.Fatalf("dispatchPackEngines: %v", err)
	}
	call := goTestCall(t, runner)
	if strings.Join(call.args, " ") != "test ./..." {
		t.Errorf("project-wide test must target ./..., got args=%v", call.args)
	}
}
