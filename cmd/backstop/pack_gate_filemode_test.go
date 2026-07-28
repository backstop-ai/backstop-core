package main

import (
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/gate"
)

// SPEC-035 REQ-006b/CLM-024 — file-mode PACKAGE scoping keys off the binding's
// DECLARED PackageScoped flag, NOT a "go test" command-prefix sniff (Sharp Edge
// 6). filemode_scoping_test.go pins the real go-test engine's scoping behavior;
// this file pins the DECLARED-vs-NAME divergence the retirement requires, using
// the divergent-flags fixture whose flags disagree with their command names.

// TestPackageScoped_KeyedOnDeclaredFlagNotName proves fileModeTestTarget applies
// the file-mode package scoping based on the binding's declared PackageScoped
// flag, NOT a "go test" command-prefix sniff (CLM-024, Sharp Edge 6):
//
//   - a NON-"go test" command (acme-test) with PackageScoped:true IS scoped — a
//     name-sniff would have skipped it, and
//   - a "go test"-NAMED command with PackageScoped:false is NOT scoped — a
//     name-sniff would have wrongly scoped it.
func TestPackageScoped_KeyedOnDeclaredFlagNotName(t *testing.T) {
	m := divergentFlagsManifest(t)
	// A genuine file-mode scope so the override is eligible to apply.
	scope := &gate.GateScope{Mode: gate.GateScopeModeFile, Files: []string{"pkg/widget/widget_test.go"}}

	// Positive half: non-"go test" command, PackageScoped TRUE -> package-scoped.
	// A "go test" prefix sniff would have skipped this binding (ok=false).
	pkgDivergent := m.Engines["package-scoped-divergent"].Binding
	if strings.HasPrefix(strings.TrimSpace(pkgDivergent.Command), "go test") {
		t.Fatalf("fixture invariant: package-scoped-divergent must NOT be a `go test` command, got %q", pkgDivergent.Command)
	}
	if !pkgDivergent.PackageScoped {
		t.Fatal("fixture invariant: package-scoped-divergent must declare PackageScoped:true")
	}
	target, ok := fileModeTestTarget(pkgDivergent, scope)
	if !ok {
		t.Error("a non-`go test` command with PackageScoped:true must be package-scoped (keyed off the flag, not the name)")
	}
	if ok && target != "./pkg/widget" {
		t.Errorf("file-mode package target must be the changed file's package ./pkg/widget, got %q", target)
	}

	// Negative half: "go test"-NAMED command, PackageScoped FALSE -> NOT scoped.
	// A "go test" prefix sniff would have wrongly scoped it (ok=true).
	goTestNamed := m.Engines["go-test-named-no-pkgscope"].Binding
	if !strings.HasPrefix(strings.TrimSpace(goTestNamed.Command), "go test") {
		t.Fatalf("fixture invariant: go-test-named-no-pkgscope must be a `go test` command, got %q", goTestNamed.Command)
	}
	if goTestNamed.PackageScoped {
		t.Fatal("fixture invariant: go-test-named-no-pkgscope must declare PackageScoped:false")
	}
	if _, ok := fileModeTestTarget(goTestNamed, scope); ok {
		t.Error("a `go test`-named command with PackageScoped:false must NOT be package-scoped — the scoping keys off the declared flag, not the name")
	}
}
