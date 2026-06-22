package main

import (
	"context"
	"reflect"
	"testing"

	"github.com/bmanson/backstop-core/pkg/check"
	"github.com/bmanson/backstop-core/pkg/gate"
	"github.com/bmanson/backstop-core/pkg/pack"
	"github.com/bmanson/backstop-core/pkg/packval"
)

// funcPtr returns the underlying code pointer of a function value so two
// function values can be compared for identity (Go forbids == on funcs).
func funcPtr(fn interface{}) uintptr {
	return reflect.ValueOf(fn).Pointer()
}

// TestResolveSandboxedRun_FallsBackToConcreteWhenSeamNil proves the production
// path: with the sandboxedRun seam unset (nil), resolveSandboxedRun returns the
// concrete packval.SandboxedRun, and with the seam set it returns the override.
// This pins the lazy-resolution wiring that lets the seam hold no package-level
// mutable default while production behavior is unchanged.
func TestResolveSandboxedRun_FallsBackToConcreteWhenSeamNil(t *testing.T) {
	orig := sandboxedRun
	t.Cleanup(func() { sandboxedRun = orig })

	sandboxedRun = nil
	if got := resolveSandboxedRun(); funcPtr(got) != funcPtr(packval.SandboxedRun) {
		t.Fatalf("nil seam must resolve to packval.SandboxedRun")
	}

	called := false
	stub := func(string, []string, string) ([]byte, error) { called = true; return nil, nil }
	sandboxedRun = stub
	resolved := resolveSandboxedRun()
	if funcPtr(resolved) != funcPtr(stub) {
		t.Fatalf("set seam must resolve to the override")
	}
	if _, err := resolved("x", nil, "d"); err != nil {
		t.Fatalf("resolved override returned err: %v", err)
	}
	if !called {
		t.Fatal("resolved override was not the stub")
	}
}

// TestResolveSandboxedRunStdout_FallsBackToConcreteWhenSeamNil proves the
// clean-stdout convert seam resolves to packval.SandboxedRunStdout when unset and
// to the override when set.
func TestResolveSandboxedRunStdout_FallsBackToConcreteWhenSeamNil(t *testing.T) {
	orig := sandboxedRunStdout
	t.Cleanup(func() { sandboxedRunStdout = orig })

	sandboxedRunStdout = nil
	if funcPtr(resolveSandboxedRunStdout()) != funcPtr(packval.SandboxedRunStdout) {
		t.Fatalf("nil seam must resolve to packval.SandboxedRunStdout")
	}

	called := false
	stub := func(string, []string, string, []byte) ([]byte, error) { called = true; return []byte("ok"), nil }
	sandboxedRunStdout = stub
	resolved := resolveSandboxedRunStdout()
	out, err := resolved("c", nil, "d", []byte("in"))
	if err != nil || string(out) != "ok" || !called {
		t.Fatalf("set seam must resolve to the override; out=%q err=%v called=%v", out, err, called)
	}
}

// TestResolveEngineRegistry_FallsBackToDefaultWhenSeamNil proves the engine
// registry resolves to engine.DefaultRegistry when the seam is nil (so the
// built-in semgrep/ast-grep/sandbox bindings are present in production) and to
// the injected registry when set.
func TestResolveEngineRegistry_FallsBackToDefaultWhenSeamNil(t *testing.T) {
	orig := engineRegistry
	t.Cleanup(func() { engineRegistry = orig })

	engineRegistry = nil
	got := resolveEngineRegistry(nil)
	if _, err := got.Lookup("semgrep"); err != nil {
		t.Fatalf("nil seam must resolve to the default registry with semgrep registered: %v", err)
	}

	engineRegistry = resolveEngineRegistry(nil)
	engineRegistry["custom-engine"] = orig["semgrep"]
	if _, err := resolveEngineRegistry(nil).Lookup("custom-engine"); err != nil {
		t.Fatalf("set seam must resolve to the injected registry: %v", err)
	}
}

// TestResolveCheckRun_FallsBackToConcreteWhenSeamNil proves the check-run seam
// resolves to check.Run when unset and to the override when set.
func TestResolveCheckRun_FallsBackToConcreteWhenSeamNil(t *testing.T) {
	orig := checkRunFn
	t.Cleanup(func() { checkRunFn = orig })

	checkRunFn = nil
	if funcPtr(resolveCheckRun()) != funcPtr(check.Run) {
		t.Fatalf("nil seam must resolve to check.Run")
	}

	called := false
	stub := func(context.Context, check.Options) (*check.Result, error) { called = true; return &check.Result{}, nil }
	checkRunFn = stub
	if _, err := resolveCheckRun()(context.Background(), check.Options{}); err != nil || !called {
		t.Fatalf("set seam must resolve to the override; err=%v called=%v", err, called)
	}
}

// TestResolveLoadInstalledPacks_FallsBackToConcreteWhenSeamNil proves the pack
// loader seam resolves to loadInstalledPacks when unset and to the override when
// set.
func TestResolveLoadInstalledPacks_FallsBackToConcreteWhenSeamNil(t *testing.T) {
	orig := loadInstalledPacksFn
	t.Cleanup(func() { loadInstalledPacksFn = orig })

	loadInstalledPacksFn = nil
	if funcPtr(resolveLoadInstalledPacks()) != funcPtr(loadInstalledPacks) {
		t.Fatalf("nil seam must resolve to loadInstalledPacks")
	}

	called := false
	stub := func(string) ([]*pack.Manifest, error) { called = true; return nil, nil }
	loadInstalledPacksFn = stub
	if _, err := resolveLoadInstalledPacks()("root"); err != nil || !called {
		t.Fatalf("set seam must resolve to the override; err=%v called=%v", err, called)
	}
}

// TestResolveDispatchPackEngines_FallsBackToConcreteWhenSeamNil proves the
// dispatch seam (shared by code check and the gate) resolves to the concrete
// dispatchPackEngines when unset and to the override when set.
func TestResolveDispatchPackEngines_FallsBackToConcreteWhenSeamNil(t *testing.T) {
	orig := dispatchPackEnginesFn
	t.Cleanup(func() { dispatchPackEnginesFn = orig })

	dispatchPackEnginesFn = nil
	if funcPtr(resolveDispatchPackEngines()) != funcPtr(dispatchPackEngines) {
		t.Fatalf("nil seam must resolve to dispatchPackEngines")
	}

	called := false
	var gotScope *gate.GateScope
	stub := func(_ []*pack.Manifest, _, _ string, s *gate.GateScope, _ check.CommandRunner) ([]gate.Violation, error) {
		called = true
		gotScope = s
		return nil, nil
	}
	dispatchPackEnginesFn = stub
	scope := &gate.GateScope{Mode: gate.GateScopeModeDiff, Files: []string{"a.go"}}
	if _, err := resolveDispatchPackEngines()(nil, "", "", scope, nil); err != nil || !called {
		t.Fatalf("set seam must resolve to the override; err=%v called=%v", err, called)
	}
	if gotScope != scope {
		t.Fatal("resolved dispatch override did not receive the threaded scope")
	}
}

// TestFirstNonNil covers the error-collapsing helper used by the gate flag guard:
// it returns the first non-nil error, or nil when all are nil.
func TestFirstNonNil(t *testing.T) {
	if err := firstNonNil(nil, nil, nil); err != nil {
		t.Errorf("all-nil must yield nil, got %v", err)
	}
	first := context.Canceled
	second := context.DeadlineExceeded
	if got := firstNonNil(nil, first, second); got != first {
		t.Errorf("expected the FIRST non-nil error, got %v", got)
	}
	if got := firstNonNil(first); got != first {
		t.Errorf("single non-nil must be returned, got %v", got)
	}
	if got := firstNonNil(); got != nil {
		t.Errorf("no args must yield nil, got %v", got)
	}
}
