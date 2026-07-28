package packval

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// THE GUARD PAIR THAT MAKES THE COVERAGE EXCLUSION AUDITABLE.
//
// .backstop/coverage-exclusions excludes pkg/packval/sandbox_linux_helper.go WHOLE,
// on the evidence in testdata/sandbox-linux-coverage-profile.txt: every function in
// that file runs inside the re-exec helper, which ends in unix.Exec, so the process
// image is replaced and Go never flushes its counters. Five functions there measured
// exactly 0/79 on a run where the sandbox demonstrably installed and held.
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

// execSideFunctions returns the set the exclusion is justified FOR. Each one either
// calls unix.Exec or is only reachable from something that does, so its counters cannot
// flush. This list and the exclusion entry rise and fall together — changing one
// without the other is the drift these tests exist to catch.
//
// A function rather than a package-level var so the list cannot be mutated by one test
// and read by another: both tests below assert an EXACT set, and a shared slice they
// could each append to is a guard that can be silently widened.
func execSideFunctions() []string {
	return []string{
		"runSandboxHelper",
		"applyRestrictionsAndExec",
		"applyLandlock",
		"applySeccomp",
		"seccompAuditArch",
		"seccompSyscallNumbers",
	}
}

// TestSandboxLinuxHelperFile_ContainsEveryExecSideStatement asserts the exec-side
// functions live in the EXCLUDED file and nowhere else, and that the excluded file
// holds NOTHING BUT them.
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
		sanctionedCaller = "MaybeRunSandboxHelper"
		sanctionedCallee = "runSandboxHelper"
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

	// The exclusion's justification, asserted where it lives. If the helper no longer
	// execs, its counters flush like anything else and the whole-file exemption in
	// .backstop/coverage-exclusions is no longer an evidenced claim.
	if !fileCallsUnixExec(t, "sandbox_linux_helper.go") {
		t.Error("sandbox_linux_helper.go no longer calls unix.Exec. The coverage exclusion for that file " +
			"rests entirely on exec replacing the process image before counters flush; with the exec gone " +
			"the file is measurable and the exemption must be removed, not inherited")
	}
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
func fileCallsUnixExec(t *testing.T, filename string) bool {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, nil, 0)
	if err != nil {
		t.Fatalf("parsing %s: %v", filename, err)
	}

	found := false
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		pkg, ok := sel.X.(*ast.Ident)
		if ok && pkg.Name == "unix" && sel.Sel.Name == "Exec" {
			found = true
		}
		return true
	})
	return found
}
