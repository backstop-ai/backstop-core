package main

import (
	"path/filepath"
	"strings"

	"github.com/bmanson/backstop-core/pkg/gate"
	"github.com/bmanson/backstop-core/pkg/pack/engine"
)

// File-mode go-test PACKAGE scoping across the bridge (SPEC-034 REQ-010 / N1,
// Sharp Edge 8). The `code check --file` standalone-hook path scopes `go test` to
// the changed file's PACKAGE — not `./...` — so the hook stays within its tight
// time budget. The bespoke testExecutor did this via goPackageSelector +
// testExecutor.fileMode; the engine path must PRESERVE it, or a single-file hook
// would silently run the whole module. This file is the file-mode half of the
// scope-kind-aware arg-shaping the project-wide half (ProjectTarget) started.
//
// The DECISION (REQ-010): file-mode test scoping is PRESERVED, not dropped. A
// whole-module run in file mode is a regression (CLM-035), asserted by
// filemode_scoping_test.go.

// fileModeTestTarget returns the file-scoped `go test` package target for a
// project-wide toolchain test engine when the gate scope is file-mode, and
// reports whether the override applies. It applies ONLY to the native go-test
// engine (a project-wide findings engine whose command runs `go test`) under a
// GateScopeModeFile scope with at least one changed file — exactly the
// `code check --file` hook path. In every other case ok is false and the caller
// keeps the engine's project-wide ProjectTarget (./...), so unchanged-file
// breakage still fails a full gate run (the override is surgical, not a leak).
func fileModeTestTarget(binding engine.EngineBinding, scope *gate.GateScope) (string, bool) {
	if scope == nil || scope.Mode != gate.GateScopeModeFile || len(scope.Files) == 0 {
		return "", false
	}
	if !isNativeGoTestEngine(binding) {
		return "", false
	}
	return goTestPackageSelector(scope.Files[0]), true
}

// isNativeGoTestEngine reports whether a binding is the native go-test pass: a
// project-wide findings engine whose command runs `go test`. Keyed off the
// declared command/shape so no extra EngineBinding field is needed (the binding
// table stays an immutable lookup table). go-build is project-wide too but is NOT
// file-scoped: only `go test` carried the file-mode package scoping in the
// bespoke path (REQ-010).
func isNativeGoTestEngine(binding engine.EngineBinding) bool {
	return binding.ScopeKind == engine.ScopeKindProjectWide &&
		strings.HasPrefix(strings.TrimSpace(binding.Command), "go test")
}

// goTestPackageSelector returns the `go test` package selector for a single
// changed file: the file's directory as a module-relative ./-prefixed path,
// mirroring the retired pkg/check.goPackageSelector so the engine path scopes
// identically (REQ-010). A file at the module root resolves to ".".
func goTestPackageSelector(file string) string {
	dir := filepath.Dir(file)
	if dir == "" || dir == "." {
		return "."
	}
	dir = filepath.ToSlash(dir)
	if strings.HasPrefix(dir, "/") || strings.HasPrefix(dir, "./") {
		return dir
	}
	return "./" + dir
}
