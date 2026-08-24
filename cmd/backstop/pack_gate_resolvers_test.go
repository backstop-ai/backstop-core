package main

import (
	"context"
	"reflect"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/check"
	"github.com/backstop-ai/backstop-core/pkg/gate"
	"github.com/backstop-ai/backstop-core/pkg/pack"
)

// funcPtr returns the underlying code pointer of a function value so two
// function values can be compared for identity (Go forbids == on funcs).
func funcPtr(fn interface{}) uintptr {
	return reflect.ValueOf(fn).Pointer()
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
