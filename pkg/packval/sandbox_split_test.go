package packval

import (
	"go/ast"
	"go/parser"
	"go/token"
	"slices"
	"testing"
)

// THE GUARD PAIR THAT MAKES THE COVERAGE EXCLUSION AUDITABLE.
//
// .backstop/coverage-exclusions excludes pkg/packval/sandbox_linux_helper.go WHOLE,
// on the evidence in testdata/sandbox-linux-coverage-profile.txt: every function in
// that file runs inside the re-exec helper, which ends in unix.Exec, so the process
// image is replaced and Go never flushes its counters. The final amendment narrows
// that file to the wrapper plus the two real production syscall installers.
//
// The exclusion covers a FILE. So the file's CONTENTS are the exclusion's real
// boundary, and without these two tests that boundary is enforced by nothing: a later
// edit could move measurable logic INTO the excluded file (silently retiring it from
// measurement) or move exec-side logic BACK INTO the measured one (silently
// re-creating an unmeasurable-but-not-excluded region whose number drops with no
// explanation). Either drift passes every other test in this package.
//
// Both tests parse source with go/ast and therefore run on darwin, where the local
// suite lives — go/parser reads a //go:build linux file regardless of its constraints.
// That is deliberate: a guard that only runs on the runner cannot stop the edit that
// needs stopping at the moment it is made.

// execSideFunctions returns the exact three-function set the exclusion is justified FOR.
// Pure decisions and mappings remain measured; only real syscall installation stays here.
// This list and the exclusion entry rise and fall together — changing one
// without the other is the drift these tests exist to catch.
//
// A function rather than a package-level var so the list cannot be mutated by one test
// and read by another: both tests below assert an EXACT set, and a shared slice they
// could each append to is a guard that can be silently widened.
func execSideFunctions() []string {
	return []string{
		"applyRestrictionsAndExec",
		"applyLandlock",
		"applySeccomp",
	}
}

// TestSandboxLinuxHelperFile_ContainsEveryExecSideStatement asserts the exec-side
// wrapper lives in the EXCLUDED file and nowhere else, and that the excluded file
// holds NOTHING BUT it.
//
// The set is asserted EXACTLY in both directions. A missing name means an exec-side
// function migrated into the measured file, where its unflushable counters will drag
// a real number down and look like untested code. An EXTRA name means a function
// gained a coverage exemption it was never argued for — the more dangerous direction,
// because it is silent: the file is excluded wholesale, so a measurable function
// parked here simply stops being measured and nothing anywhere goes red.
func TestSandboxLinuxHelperFile_ContainsEveryExecSideStatement(t *testing.T) {
	helperFuncs := topLevelFuncNames(t, "sandbox_linux_helper.go")
	measuredFuncs := topLevelFuncNames(t, "sandbox_linux.go")

	// PRESENCE FIRST. Every assertion below is over a map, and every one of them is
	// satisfied by an empty map — a file emptied out, renamed, or parsed from the
	// wrong directory would pass the whole test having proven nothing.
	if len(helperFuncs) == 0 {
		t.Fatalf("sandbox_linux_helper.go declares no functions. This test asserts a property OF its " +
			"contents and is vacuous against an empty file; if the exec-side split was dissolved, the " +
			"coverage exclusion in .backstop/coverage-exclusions must go with it")
	}
	if len(measuredFuncs) == 0 {
		t.Fatalf("sandbox_linux.go declares no functions — the MEASURED half of the split is empty, so " +
			"there is nothing left for the exclusion to be an exception to")
	}

	for _, name := range execSideFunctions() {
		if !helperFuncs[name] {
			t.Errorf("%s is not declared in sandbox_linux_helper.go. It runs after unix.Exec replaces the "+
				"process image, so wherever it now lives its counters never flush: if that is the measured "+
				"file, the coverage number for sandbox_linux.go is now wrong for a reason no reader can see",
				name)
		}
		if measuredFuncs[name] {
			t.Errorf("%s is declared in sandbox_linux.go, the MEASURED file. Exec-erased statements in a "+
				"measured file are unmeasurable-but-not-excluded — the exact hole the split was drawn to "+
				"close (evidence: testdata/sandbox-linux-coverage-profile.txt)", name)
		}
	}

	expected := make(map[string]bool, len(execSideFunctions()))
	for _, name := range execSideFunctions() {
		expected[name] = true
	}
	for name := range helperFuncs {
		if !expected[name] {
			t.Errorf("sandbox_linux_helper.go declares %s, which is not on the exec-side list. The whole "+
				"file is excluded from coverage, so this function is now unmeasured without an argument "+
				"for why it CANNOT be measured. Either move it to sandbox_linux.go and cover it, or add it "+
				"here together with the evidence that its counters cannot flush", name)
		}
	}
	for _, name := range []string{
		"runSandboxHelper",
		"applyRestrictionsAndExecWith",
		"applySeccompPolicy",
		"seccompAuditArch",
		"seccompSyscallNumbers",
		"writeSandboxAcknowledgement",
	} {
		if !measuredFuncs[name] || helperFuncs[name] {
			t.Errorf("returning helper %s ownership measured/excluded=%v/%v, want true/false", name, measuredFuncs[name], helperFuncs[name])
		}
	}
	assertProductionRestrictionDelegation(t)
	assertRealInstallerBodies(t)
}

// TestSandboxLinuxFile_HasNoExecErasedStatements is the inverse guard, and it catches
// the drift that matters most: exec-side logic moving BACK into the measured file.
//
// It looks for the mechanism rather than the declarations, because the damaging edit is
// rarely a whole function moving. It is a fragment: someone inlines the exec, or calls
// applyLandlock from a helper in the measured file to "keep it together". Either way the
// measured file acquires statements that execute and never report, and its number falls
// with no explanation available to whoever has to explain it.
//
// ⚠ EXACTLY ONE CROSSING IS SANCTIONED, AND THE TEST NAMES IT. The measured file DOES
// call an exec-side function: MaybeRunSandboxHelper dispatches to runSandboxHelper
// (sandbox_linux.go:124). That call is the seam itself and it is honestly measurable —
// TestMaybeRunSandboxHelper_DispatchesWhenTheEnvVarIsPresent drives it with a malformed
// spec so it returns at the decode step, before any restriction is installed and before
// any exec, and its counter flushes like any other in-process call. A blanket "no
// exec-side calls" rule would have condemned the seam it was written to protect. So the
// crossing is pinned instead: this callee, from this caller, exactly once.
//
// The final assertion is the anti-vacuity check: the helper file must STILL call
// unix.Exec. Without it this test passes trivially the day the exec goes away, which
// is also the day the exclusion stops being justified.
func TestSandboxLinuxFile_HasNoExecErasedStatements(t *testing.T) {
	const (
		sanctionedCaller = "runSandboxHelper"
		sanctionedCallee = "applyRestrictionsAndExec"
	)

	fset := token.NewFileSet()
	measured, err := parser.ParseFile(fset, "sandbox_linux.go", nil, 0)
	if err != nil {
		t.Fatalf("parsing sandbox_linux.go: %v", err)
	}

	execSide := make(map[string]bool, len(execSideFunctions()))
	for _, name := range execSideFunctions() {
		execSide[name] = true
	}

	inspected, crossings := 0, 0
	for _, decl := range measured.Decls {
		enclosing, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		ast.Inspect(enclosing, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			inspected++
			switch fn := call.Fun.(type) {
			case *ast.SelectorExpr:
				pkg, ok := fn.X.(*ast.Ident)
				if ok && pkg.Name == "unix" && fn.Sel.Name == "Exec" {
					t.Errorf("sandbox_linux.go calls unix.Exec at %s. The process image is replaced there, "+
						"so every counter in this MEASURED file that has not already flushed is lost — the "+
						"measurement failure the helper split exists to contain", fset.Position(call.Pos()))
				}
			case *ast.Ident:
				if !execSide[fn.Name] {
					return true
				}
				if enclosing.Name.Name == sanctionedCaller && fn.Name == sanctionedCallee {
					crossings++
					return true
				}
				t.Errorf("%s calls the exec-side function %s at %s. Only %s→%s is sanctioned: that one "+
					"returns before any restriction is installed and its counter flushes. This call reaches "+
					"statements that execute and never report, inside a file the exclusion does not cover",
					enclosing.Name.Name, fn.Name, fset.Position(call.Pos()), sanctionedCaller, sanctionedCallee)
			}
			return true
		})
	}

	if inspected == 0 {
		t.Fatal("found no call expressions in sandbox_linux.go — the file is empty, unparsed, or read " +
			"from the wrong directory, and the absence assertions above proved nothing")
	}
	if crossings != 1 {
		t.Errorf("expected exactly 1 sanctioned crossing (%s→%s), found %d. At zero the dispatch was "+
			"deleted or renamed and this guard no longer watches the seam it names; above one the measured "+
			"file has grown a second path into exec-erased code",
			sanctionedCaller, sanctionedCallee, crossings)
	}
	if fileReferencesSelector(t, "sandbox_linux.go", "unix", "Exec") {
		t.Error("sandbox_linux.go references unix.Exec directly; measured orchestration must call only its injected execTarget parameter")
	}

	// The exclusion's justification, asserted where it lives. If the helper no longer
	// execs, its counters flush like anything else and the whole-file exemption in
	// .backstop/coverage-exclusions is no longer an evidenced claim.
	references, calls, delegated := unixExecUsage(t, "sandbox_linux_helper.go")
	if references != 1 || calls != 0 || delegated != 1 {
		t.Errorf("sandbox_linux_helper.go unix.Exec usage references/calls/delegations=%d/%d/%d, want 1/0/1; "+
			"the excluded wrapper may only pass the real function once to applyRestrictionsAndExecWith",
			references, calls, delegated)
	}
}

func assertRealInstallerBodies(t *testing.T) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sandbox_linux_helper.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]*ast.FuncDecl{}
	for _, declaration := range file.Decls {
		if fn, ok := declaration.(*ast.FuncDecl); ok {
			found[fn.Name.Name] = fn
		}
	}
	for _, name := range []string{"applyLandlock", "applySeccomp"} {
		fn := found[name]
		if fn == nil || fn.Body == nil || len(fn.Body.List) < 2 {
			t.Errorf("excluded real installer %s is absent or vacuous", name)
		}
	}
	if !functionReferencesSelector(found["applyLandlock"], "unix", "SYS_LANDLOCK_CREATE_RULESET") ||
		!functionReferencesSelector(found["applyLandlock"], "unix", "SYS_LANDLOCK_RESTRICT_SELF") {
		t.Error("applyLandlock does not contain the concrete create/restrict syscalls")
	}
	if !functionReferencesSelector(found["applySeccomp"], "unix", "SYS_SECCOMP") {
		t.Error("applySeccomp does not contain the concrete seccomp install syscall")
	}
	if functionCallsLenSelector(found["applySeccomp"], "SeccompDenied") {
		t.Error("excluded applySeccomp contains the empty-policy decision; that pure branch belongs in measured applySeccompPolicy")
	}
}

func functionReferencesSelector(fn *ast.FuncDecl, packageName, selectorName string) bool {
	if fn == nil {
		return false
	}
	found := false
	ast.Inspect(fn, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != selectorName {
			return true
		}
		ident, ok := selector.X.(*ast.Ident)
		if ok && ident.Name == packageName {
			found = true
		}
		return true
	})
	return found
}

func functionCallsLenSelector(fn *ast.FuncDecl, selectorName string) bool {
	if fn == nil {
		return false
	}
	found := false
	ast.Inspect(fn, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		ident, ok := call.Fun.(*ast.Ident)
		if !ok || ident.Name != "len" || len(call.Args) != 1 {
			return true
		}
		selector, ok := call.Args[0].(*ast.SelectorExpr)
		if ok && selector.Sel.Name == selectorName {
			found = true
		}
		return true
	})
	return found
}

func assertProductionRestrictionDelegation(t *testing.T) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sandbox_linux_helper.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	calls := 0
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		ident, ok := call.Fun.(*ast.Ident)
		if !ok || ident.Name != "applyRestrictionsAndExecWith" {
			return true
		}
		calls++
		for _, arg := range call.Args {
			got = append(got, splitExprName(arg))
		}
		return true
	})
	want := []string{"request", "applyLandlock", "applySeccomp", "writeSandboxAcknowledgement", "unix.Chdir", "exec.LookPath", "unix.Exec"}
	if calls != 1 || !slices.Equal(got, want) {
		t.Fatalf("excluded wrapper delegation calls=%d args=%v, want exactly one with %v", calls, got, want)
	}
}

func splitExprName(expr ast.Expr) string {
	switch value := expr.(type) {
	case *ast.Ident:
		return value.Name
	case *ast.SelectorExpr:
		return splitExprName(value.X) + "." + value.Sel.Name
	default:
		return ""
	}
}

func fileReferencesSelector(t *testing.T, filename, packageName, selectorName string) bool {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), filename, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	ast.Inspect(file, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != selectorName {
			return true
		}
		ident, ok := selector.X.(*ast.Ident)
		if ok && ident.Name == packageName {
			found = true
		}
		return true
	})
	return found
}

// topLevelFuncNames returns the plain (receiverless) top-level function names declared
// in a file in this package directory. Methods are excluded deliberately: the exec-side
// split is about free functions, and a method would be reported as an unexpected name
// by the exact-set assertion rather than silently accepted.
func topLevelFuncNames(t *testing.T, filename string) map[string]bool {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, nil, 0)
	if err != nil {
		t.Fatalf("parsing %s: %v", filename, err)
	}

	names := make(map[string]bool)
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv != nil {
			continue
		}
		names[fn.Name.Name] = true
	}
	return names
}

// fileCallsUnixExec reports whether a file in this package directory calls unix.Exec.
func unixExecUsage(t *testing.T, filename string) (references, calls, delegated int) {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, nil, 0)
	if err != nil {
		t.Fatalf("parsing %s: %v", filename, err)
	}

	ast.Inspect(file, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		pkg, ok := sel.X.(*ast.Ident)
		if ok && pkg.Name == "unix" && sel.Sel.Name == "Exec" {
			references++
			if parentCallUses(file, sel, true) {
				calls++
			}
			if parentCallUses(file, sel, false) {
				delegated++
			}
		}
		return true
	})
	return references, calls, delegated
}

func parentCallUses(file *ast.File, target ast.Expr, asCallee bool) bool {
	found := false
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if asCallee && call.Fun == target {
			found = true
		}
		if !asCallee {
			if ident, ok := call.Fun.(*ast.Ident); !ok || ident.Name != "applyRestrictionsAndExecWith" {
				return true
			}
			for _, arg := range call.Args {
				if arg == target {
					found = true
				}
			}
		}
		return true
	})
	return found
}
