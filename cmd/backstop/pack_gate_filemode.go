package main

import (
	"path/filepath"
	"strings"

	"github.com/backstop-ai/backstop-core/pkg/gate"
	"github.com/backstop-ai/backstop-core/pkg/pack"
	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
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
// filemode_scoping_test.go. That rationale is still why file-mode scoping exists
// at all and is not retracted by anything below.
//
// ── ISSUE-093: WHY THE DECISION IS THREE-STATE AND NOT TWO ──────────────────
// The original two-state form derived a package target for EVERY file-mode scope
// whose binding declared PackageScoped, with NO check that the pack dispatching
// the engine had any interest in the scoped file at all. A scope over a file in a
// directory the pack owns nothing in handed that directory to the engine, which
// had nothing to do there, exited non-zero, and — under a CrashGuard binding —
// had a legitimate no-op reported as an engine crash.
//
// The missing question is NOT "does this directory contain <language> code":
// asking that in core would BAKE A LANGUAGE, which the thin-executor first
// principle forbids outright. The missing question is
//
//	DOES THE PACK DISPATCHING THIS ENGINE CLAIM ANY FILE IN THIS SCOPE?
//
// and it is answerable from pack DATA that is already parsed — the top-level
// `classification:` block (SPEC-043's seam, reused here). Core carries no
// extension literal; the pack says what it owns.
//
// Two states could not express the answer, which is why the defect existed. The
// three-ness is STRUCTURAL below (a named state enum on a result struct) rather
// than two bools a future reader can collapse back into one.

// fileModeState names the three structurally-distinct answers the file-mode
// package-target decision can give. It is an enum rather than a pair of bools so
// the third state cannot be quietly folded back into the second.
type fileModeState int

const (
	// fileModeNotApplicable (A) — the scope is not file-mode, or the binding does
	// not declare PackageScoped. The caller keeps binding.ProjectTarget, so diff
	// scope and --all still run the engine's whole-project pass and unchanged-file
	// breakage still REDs a full run (CLM-013).
	fileModeNotApplicable fileModeState = iota

	// fileModeTargetsDerived (B) — the pack claims one or more scoped files (or,
	// under the capability-absent carve-out, has not declared what it claims). The
	// caller appends every derived target.
	fileModeTargetsDerived

	// fileModeClaimsNothing (C) — the pack DECLARED its classification globs and
	// none of the scoped files match. The engine is not dispatched at all and the
	// caller says so out loud. It must NEVER fall back to binding.ProjectTarget:
	// that would run the pack's entire project-wide pass because someone scoped
	// the gate to one unrelated file — both a scope lie and a large regression on
	// the fast per-file loop. This is the same ANTI-FALLBACK rule the rule-fed
	// branch already carries (ISSUE-010 CLM-003), extended to this branch.
	fileModeClaimsNothing
)

// fileModeDecision is the result of the file-mode package-target decision: the
// state, the derived targets (state B only), and the capability-absent carve-out
// flag.
type fileModeDecision struct {
	state   fileModeState
	targets []string

	// capabilityAbsent marks the (C') carve-out: the binding declares
	// PackageScoped but the pack declares NO classification globs at all, so
	// "claims nothing" is UNKNOWABLE rather than FALSE. Collapsing (C') into (C)
	// would turn a missing declaration into a silent skip — the exact vacuous-green
	// shape ISSUE-112 filed — so the derivation's SHAPE is preserved (no claim
	// check, no testdata drop) and the caller emits a DISTINCT capability-absent
	// advisory instead.
	capabilityAbsent bool
}

// fileModeTestTargets decides how a project-wide PackageScoped engine is targeted
// under the current gate scope, returning one of the three states above.
//
// The dispatching pack's OWN manifest is the input for the claim check — never
// the cross-pack merged classifier buildGateSteps assembles for coverage. Using
// the merged union here would be a defect in a multi-language repo: a file
// claimed by some OTHER installed pack would make THIS pack's package-scoped
// engine fire and fail exactly as it did before the fix.
//
// ORDER OF OPERATIONS IS LOAD-BEARING:
//
//  1. Not file-mode, or !PackageScoped                -> (A), unchanged.
//  2. The pack declares NO classification globs       -> (C'), derive from every
//     scoped file with today's SHAPE and signal capability-absent.
//  3. Drop testdata paths (the SHARED excludeTestdataPaths, not a second copy).
//  4. Retain the files the pack's own classifier CLAIMS.
//  5. Empty after 3+4 -> (C). Non-empty -> (B), one deduped selector per
//     retained file in stable input order.
//
// Step 5 derives from EVERY retained file, not scope.Files[0]. The old
// Files[0]-only form already meant that files 2..n of a multi-file scope were
// never tested; leaving it in place while making the scope easier to populate
// would make a multi-file invocation READ as thorough while testing exactly one
// package — a false green worse than the crash this fix removes.
func fileModeTestTargets(m *pack.Manifest, binding engine.EngineBinding, scope *gate.GateScope) fileModeDecision {
	if scope == nil || scope.Mode != gate.GateScopeModeFile || len(scope.Files) == 0 {
		return fileModeDecision{state: fileModeNotApplicable}
	}
	// File-mode package scoping applies to an engine that DECLARES PackageScoped
	// (REQ-006b/CLM-024), NOT a "go test" command-prefix sniff: a pack declares
	// which of its toolchain engines run per package, so the override rides the
	// declaration and no tool-name literal drives this control flow (Sharp Edge 5).
	// A project-wide engine that does NOT declare PackageScoped keeps its
	// ProjectTarget. This test is evaluated BEFORE any classification is consulted,
	// which is why the declared-flag-vs-name property is state-independent.
	if !binding.PackageScoped {
		return fileModeDecision{state: fileModeNotApplicable}
	}

	classifier := gate.NewSourceClassifier(m.Classification.Source, m.Classification.Test)

	// (C') THE CAPABILITY-ABSENT CARVE-OUT. The pack never said what it owns, so
	// the claim question has no answer — and "no answer" must not be read as "no".
	// Preserve the derivation's SHAPE: no claim check, and deliberately NO testdata
	// drop. The drop is sequenced with the claim check and belongs to it; applying
	// it here could leave an EMPTY target list with no honest answer available
	// (skipping would be the silent-skip shape this carve-out exists to prevent,
	// and ProjectTarget is the fallback the (C) contract forbids). So a scope over
	// a testdata path against an undeclared pack still derives that directory and
	// still fails exactly as it does today. That residual is PRE-EXISTING and named
	// rather than hidden; the testdata drop closes the door only for packs that
	// declare what they own.
	//
	// The ARITY, however, is NOT preserved: every scoped file yields a target, for
	// the same reason state (B) does.
	if !classifier.DeclaresAnyGlobs() {
		return fileModeDecision{
			state:            fileModeTargetsDerived,
			targets:          packageSelectorsFor(scope.Files),
			capabilityAbsent: true,
		}
	}

	// Testdata FIRST, then the claim check. A pack that folds its fixture
	// convention into its TEST globs genuinely CLAIMS a file under testdata, so
	// without the drop such a file would become a package target — the same false
	// RED through a different door. excludeTestdataPaths is the SHARED, shipped,
	// deliberately language-neutral filter the rule-fed branch already uses
	// (ISSUE-040: exact directory-SEGMENT match, so a look-alike name is not
	// dropped); a second implementation here would be a place for the two to drift.
	claimed := make([]string, 0, len(scope.Files))
	for _, f := range excludeTestdataPaths(scope.Files) {
		if classifier.ClaimsPath(f) {
			claimed = append(claimed, f)
		}
	}
	if len(claimed) == 0 {
		// (C) The pack declared what it owns and owns nothing here. A scope of only
		// testdata files lands here too, which is correct — testdata is inert data.
		return fileModeDecision{state: fileModeClaimsNothing}
	}
	return fileModeDecision{state: fileModeTargetsDerived, targets: packageSelectorsFor(claimed)}
}

// packageSelectorsFor maps files to their package selectors, DEDUPED and in
// stable input order. Order stability matters because the resulting args are
// compared in tests and read by operators; dedup matters because several files in
// one package must not run that package's pass repeatedly.
func packageSelectorsFor(files []string) []string {
	seen := map[string]bool{}
	targets := make([]string, 0, len(files))
	for _, f := range files {
		sel := goTestPackageSelector(f)
		if seen[sel] {
			continue
		}
		seen[sel] = true
		targets = append(targets, sel)
	}
	return targets
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
