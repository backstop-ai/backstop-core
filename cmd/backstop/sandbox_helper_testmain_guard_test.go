package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// sandbox_helper_testmain_guard_test.go (PLAN-ISSUE-163, CLM-001..CLM-003).
//
// WHAT THIS PINS. The Linux sandbox is a re-exec trampoline: SandboxedRun spawns
// /proc/self/exe in a hidden helper mode, and under `go test` /proc/self/exe is
// THE TEST BINARY. A TestMain that does real work before calling
// packval.MaybeRunSandboxHelper therefore runs that work in the re-exec'd copy —
// for cmd/backstop that is an unconditional `go build` in the pack directory,
// which dies with "no Go files in <dir>" before the helper is ever recognised.
// That is ISSUE-163.
//
// WHY STRUCTURAL AND NOT BEHAVIOURAL. The behaviour these tests protect exists
// ONLY on Linux. pkg/packval/sandbox_linux.go and sandbox_linux_helper.go carry
// //go:build linux; on darwin they are not compiled at all, no re-exec can
// happen, and MaybeRunSandboxHelper resolves to the !linux stub
// (pkg/packval/sandbox_nonlinux.go: `func MaybeRunSandboxHelper() error { return
// nil }`). Setting BACKSTOP_SANDBOX_HELPER_SPEC by hand on darwin proves nothing
// — the stub never reads it — so a "set the env var and watch the helper take
// over" test cannot be written honestly here and would be a vacuous green if
// written anyway. The AST pin is what IS falsifiable on every platform: delete
// the guard and these tests go red wherever they run.
//
// The other two members of this wiring family are cmd/backstop/main.go's runWith
// (the shipped binary's half) and pkg/packval/main_test.go's TestMain (the
// packval test binary's half).

// sandboxHelperGateName is the function every member of the wiring family calls
// as its first statement. cmd/backstop reaches it through a selector
// (packval.MaybeRunSandboxHelper); pkg/packval's own TestMain calls it bare.
const sandboxHelperGateName = "MaybeRunSandboxHelper"

// packvalImportPath is the import that makes a package able to become the
// re-exec target: it is the package that owns the trampoline.
const packvalImportPath = "github.com/backstop-ai/backstop-core/pkg/packval"

// TestIntegrationTestMain_RunsSandboxHelperGateAsItsFirstStatement pins CLM-001
// and CLM-002: cmd/backstop's TestMain calls packval.MaybeRunSandboxHelper as
// the FIRST statement of its body, CHECKS the returned error, and exits with the
// package's existing sandboxHelperExitCode constant.
//
// Position zero is the assertion, not "somewhere above the go build" — the
// helper has to run before anything else consumes argv or does work, which is
// the invariant both existing call sites document. A guard inserted above the
// build but below the os.MkdirTemp must fail this test.
func TestIntegrationTestMain_RunsSandboxHelperGateAsItsFirstStatement(t *testing.T) {
	path := filepath.Join(repoRoot(t), "cmd", "backstop", "integration_test.go")

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}

	decl := findTestMainDecl(file)
	if decl == nil {
		t.Fatalf("%s declares no `func TestMain(m *testing.M)`; ISSUE-163's guard has no home. "+
			"Locate the function by name — line numbers in this file move across lanes.", path)
	}

	// The guard's SHAPE. This deliberately requires the `if err := ...; err != nil`
	// form rather than merely "the call is present": PLAN-ISSUE-020 records that
	// this exact gate shipped as a BARE statement in six places before being
	// corrected. A bare packval.MaybeRunSandboxHelper() compiles, satisfies any
	// naive presence scan, and silently runs the whole suite as a broken helper.
	ifStmt, errName, shapeErr := sandboxHelperGateShape(decl)
	if shapeErr != "" {
		t.Fatalf("%s: TestMain does not open with the sandbox-helper gate: %s\n\n"+
			"Expected as the FIRST statement of the body:\n"+
			"    if err := packval.%s(); err != nil {\n"+
			"        fmt.Fprintf(os.Stderr, \"backstop sandbox helper: %%v\\n\", err)\n"+
			"        os.Exit(sandboxHelperExitCode)\n"+
			"    }",
			path, shapeErr, sandboxHelperGateName)
	}

	// CLM-002: the exit reuses the package's existing constant. sandboxHelperExitCode
	// is declared once, in cmd/backstop/main.go, with the reasoning for 126 attached
	// to it. A bare 126 spelled here is the drift that lets the two halves of the
	// wiring pair disagree about what a failed sandbox install reports.
	//
	// This assertion is cmd/backstop-only ON PURPOSE and does NOT belong in the
	// roster test below: pkg/packval/main_test.go correctly spells os.Exit(126) as a
	// literal, because it is a different package with no such constant in reach.
	exitArg, found := exitCallArgument(ifStmt.Body)
	if !found {
		t.Fatalf("%s: the sandbox-helper guard's body calls no os.Exit; a helper whose sandbox "+
			"failed to install must NOT fall through into the suite — the parent would read the "+
			"suite's output as the sandboxed command's output", path)
	}
	ident, ok := exitArg.(*ast.Ident)
	if !ok || ident.Name != "sandboxHelperExitCode" {
		t.Fatalf("%s: the guard exits with %s; it must exit with the identifier "+
			"sandboxHelperExitCode (declared in cmd/backstop/main.go, same package), not a literal",
			path, describeExpr(exitArg))
	}

	// A qualified call needs the import. A missing one would not compile, but naming
	// it here is what makes a partially-applied edit legible instead of cryptic.
	if usesQualifiedGateCall(ifStmt) && !importsPath(file, packvalImportPath) {
		t.Fatalf("%s: TestMain calls packval.%s but the file does not import %q",
			path, sandboxHelperGateName, packvalImportPath)
	}

	// The bound error must actually be the one tested. sandboxHelperGateShape has
	// already established that; assert the name is non-empty so a future refactor
	// that loosens the helper cannot leave this test asserting nothing.
	if errName == "" {
		t.Fatalf("%s: the sandbox-helper guard binds no error variable to check", path)
	}
}

// TestSandboxHelperGate_PresentInEveryPackvalReachingTestMain pins CLM-003, the
// PROPAGATION check. ISSUE-163 exists because a known, documented pattern was
// simply never propagated to a second site; this is the test that makes the next
// omission loud instead of silent.
//
// IT PINS BOTH HALVES:
//
//	(1) every packval-reaching package that compiles a test binary DECLARES a
//	    TestMain at all, and
//	(2) every such TestMain OPENS with the sandbox-helper gate.
//
// HALF (1) WAS ADDED BY ISSUE-180, AFTER THE HALF-(2)-ONLY VERSION FAILED TO
// CATCH EXACTLY THIS DEFECT. The original roster was built by
// `if pkg.testMain == nil { continue }` followed by the packval predicate, so a
// packval-reaching package with NO TestMain AT ALL was invisible BY CONSTRUCTION
// — this test was measurably GREEN while pkg/pack/distribution shipped without
// one and broke on Linux CI. ISSUE-164 raised the generalization as a question;
// it is delivered here. Membership is now derived with NO reference to TestMain
// presence, so a FIFTH packval-reaching package added later is caught with no
// edit to this file and no exemption list.
//
// It asserts THE GUARD SHAPE AND ONLY THE GUARD SHAPE. The exit-code assertion
// from the test above is deliberately absent here: pkg/packval/main_test.go and
// pkg/pack/distribution/main_test.go both legitimately write os.Exit(126) as a
// bare literal, so a roster carrying that assertion would red the very files this
// fix is modelled on.
func TestSandboxHelperGate_PresentInEveryPackvalReachingTestMain(t *testing.T) {
	root := repoRoot(t)
	packages := scanGoPackages(t, root)

	// ── STEP 1: DERIVE THE PACKVAL-REACHING SET, INDEPENDENT OF TestMain ────────
	//
	// THE PREDICATE, stated as the deliberate approximation it is: a package can
	// become the re-exec target when it DIRECTLY imports pkg/packval, or when it IS
	// pkg/packval (a package cannot import itself). Direct import, not transitive
	// reach — widen this on measurement, not on a hunch. Membership is a real
	// *ast.ImportSpec recorded by scanGoPackages, NEVER a grep for the import path:
	// a grep reports pkg/pack/engine as a packval importer and is WRONG, because
	// the hits are forbidden-import STRING LITERALS inside that package's own
	// leaf-invariant tests (TestEngineBinding_NoImportCycle, which resolves the real
	// transitive set via `go list -deps`, and TestEngine_NoForbiddenImports) — tests
	// that assert the exact opposite of what the grep implies.
	//
	// ★ NOTHING HERE MAY CONSULT pkg.testMain. That skip is what made both ISSUE-163
	// and ISSUE-180 invisible; STEP 3a below is the only assertion allowed to look
	// at TestMain presence.
	//
	// NARROWED BY DERIVATION, NOT BY EXEMPTION: a member must have at least one
	// *_test.go file, because a package that compiles no test binary can never be
	// the trampoline's re-exec target under `go test`. This changes nothing today —
	// all three current members have tests — and exists so a future production-only
	// importer is excluded by the predicate rather than by the hand-written
	// exemption this check exists to eliminate.
	//
	// tests/smoke has a TestMain and is correctly absent from the roster: its tests
	// drive the BUILT BINARY as a subprocess, so on Linux the re-exec target is that
	// binary (already guarded through runWith), never its test binary. It does not
	// import pkg/packval, so the predicate excludes it by derivation rather than by
	// an exception.
	var roster []string
	for dir, pkg := range packages {
		if !pkg.hasTestFile {
			continue
		}
		if pkg.importsPackval || dir == "pkg/packval" {
			roster = append(roster, dir)
		}
	}
	sort.Strings(roster)

	// ── STEP 2: ANTI-VACUOUS FLOOR, FIRST AND LOAD-BEARING ──────────────────────
	//
	// A predicate that quietly narrows to nothing would otherwise pass while
	// checking nothing — the vacuous-green shape DIR-032 exists to prevent.
	//
	// pkg/pack/distribution belongs here because validator.go calls
	// packval.NewPipeline(packDir, ...).Run(); it is a member BEFORE the fix exists,
	// which is exactly why STEP 2 is green pre-fix and only STEP 3a reds.
	// pkg/pack/engine deliberately does NOT belong: it is not a packval importer at
	// all, and TestEngineBinding_NoImportCycle pins that structurally.
	if len(roster) == 0 {
		t.Fatalf("derived roster is EMPTY: no package under %s both compiles a test binary and "+
			"reaches %s. The scan or the predicate is broken — this test would otherwise pass "+
			"while asserting nothing.", root, packvalImportPath)
	}
	for _, want := range []string{"cmd/backstop", "pkg/pack/distribution", "pkg/packval"} {
		if !containsString(roster, want) {
			t.Fatalf("derived roster %v is missing the known member %q; the scan or the predicate "+
				"is broken (all three packages compile a test binary and reach %s)",
				roster, want, packvalImportPath)
		}
	}

	for _, dir := range roster {
		pkg := packages[dir]

		// ── STEP 3a: EVERY MEMBER DECLARES A TestMain (ISSUE-180) ───────────────
		//
		// THE LINE THAT REMOVES THE BLIND SPOT. It does not need to know WHICH
		// packages exist, so a package added to the reaching set later is caught
		// here with no edit to this file.
		if pkg.testMain == nil {
			t.Errorf("%s reaches %s and compiles a test binary but declares NO "+
				"`func TestMain(m *testing.M)` (ISSUE-180).\n\n"+
				"On Linux the sandbox trampoline re-execs that package's test binary with "+
				"BACKSTOP_SANDBOX_HELPER_SPEC set and its working directory pointed at the pack "+
				"directory. Without a TestMain, Go's DEFAULT generated test main does not "+
				"recognise helper mode and reruns the package's WHOLE SUITE in the pack's scratch "+
				"directory instead of installing the sandbox — it dies off any go.mod ancestry, "+
				"exits 1, and writes all of it to stdout, which foldHelperStderrIntoError never "+
				"reads.\n\n"+
				"Add one, modelled on pkg/packval/main_test.go:\n"+
				"    func TestMain(m *testing.M) {\n"+
				"        if err := packval.%s(); err != nil {\n"+
				"            fmt.Fprintf(os.Stderr, \"backstop sandbox helper: %%v\\n\", err)\n"+
				"            os.Exit(126)\n"+
				"        }\n"+
				"        os.Exit(m.Run())\n"+
				"    }", dir, packvalImportPath, sandboxHelperGateName)
			continue
		}

		// ── STEP 3b: AND THAT TestMain OPENS WITH THE GATE ──────────────────────
		_, _, shapeErr := sandboxHelperGateShape(pkg.testMain)
		if shapeErr != "" {
			t.Errorf("%s: TestMain does not open with the sandbox-helper gate: %s\n\n"+
				"Every package that can become the sandbox re-exec target must open its "+
				"TestMain with:\n"+
				"    if err := %s(); err != nil { ... }\n"+
				"(qualified as packval.%s outside pkg/packval). Without it the re-exec'd copy "+
				"runs this package's TestMain body instead of the helper.",
				pkg.testMainFile, shapeErr, sandboxHelperGateName, sandboxHelperGateName)
		}
	}
}

// goPackage is what the scan records per directory: whether the package declares
// a TestMain (and where), whether it reaches pkg/packval, and whether it compiles
// a test binary at all.
//
// hasTestFile exists so the roster's membership predicate can be NARROWED BY
// DERIVATION rather than by a hand-written exemption: a package that compiles no
// test binary can never be `go test`'s re-exec target, so it cannot be handed
// BACKSTOP_SANDBOX_HELPER_SPEC by the trampoline. It changes nothing today — all
// three current members have tests — and exists so a future production-only
// packval importer is excluded by the predicate instead of by a name list.
type goPackage struct {
	testMain       *ast.FuncDecl
	testMainFile   string
	importsPackval bool
	hasTestFile    bool
}

// scanGoPackages walks the module from root and records, per directory, the
// TestMain declaration (if any), whether any file in it imports pkg/packval, and
// whether it contains at least one *_test.go file.
//
// Exclusions are DERIVED rather than hand-listed, so a new fixture cannot silently
// join the roster: any `testdata` directory (Go's own tooling ignores those, and
// both fixture TestMains in this repo live under one), any hidden directory
// (.backstop holds installed pack trees), and vendor if present.
func scanGoPackages(t *testing.T, root string) map[string]*goPackage {
	t.Helper()

	packages := map[string]*goPackage{}
	fset := token.NewFileSet()

	walkErr := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := entry.Name()
		if entry.IsDir() {
			if path == root {
				return nil
			}
			if name == "testdata" || name == "vendor" || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(name, ".go") {
			return nil
		}

		file, parseErr := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if parseErr != nil {
			return parseErr
		}

		rel, relErr := filepath.Rel(root, filepath.Dir(path))
		if relErr != nil {
			return relErr
		}
		dir := filepath.ToSlash(rel)

		pkg := packages[dir]
		if pkg == nil {
			pkg = &goPackage{}
			packages[dir] = pkg
		}
		if strings.HasSuffix(name, "_test.go") {
			pkg.hasTestFile = true
		}
		if importsPath(file, packvalImportPath) {
			pkg.importsPackval = true
		}
		if decl := findTestMainDecl(file); decl != nil {
			pkg.testMain = decl
			pkg.testMainFile = filepath.ToSlash(mustRel(t, root, path))
		}
		return nil
	})
	if walkErr != nil {
		t.Fatalf("walk %s: %v", root, walkErr)
	}
	return packages
}

// mustRel reports path relative to root, failing the test rather than returning
// an error the caller would have to swallow.
func mustRel(t *testing.T, root, path string) string {
	t.Helper()
	rel, err := filepath.Rel(root, path)
	if err != nil {
		t.Fatalf("relativize %s against %s: %v", path, root, err)
	}
	return rel
}

// findTestMainDecl returns the `func TestMain(m *testing.M)` declaration in file,
// or nil. It matches BY NAME — never by line number, which moves across lanes.
func findTestMainDecl(file *ast.File) *ast.FuncDecl {
	for _, d := range file.Decls {
		decl, ok := d.(*ast.FuncDecl)
		if !ok || decl.Recv != nil || decl.Name == nil {
			continue
		}
		if decl.Name.Name == "TestMain" && decl.Body != nil {
			return decl
		}
	}
	return nil
}

// sandboxHelperGateShape reports whether decl's body OPENS with an error-checked
// call to MaybeRunSandboxHelper, in either spelling. It returns the guard's
// *ast.IfStmt, the name of the bound error, and an empty string on success; on
// failure it returns a legible reason.
//
// This is the shared half of CLM-001 and CLM-003. The exit-code assertion is NOT
// here on purpose (see the roster test's docstring).
func sandboxHelperGateShape(decl *ast.FuncDecl) (*ast.IfStmt, string, string) {
	if decl.Body == nil || len(decl.Body.List) == 0 {
		return nil, "", "its body is empty"
	}

	// POSITION ZERO. "Above the build but below the temp dir" is not first.
	ifStmt, ok := decl.Body.List[0].(*ast.IfStmt)
	if !ok {
		return nil, "", "its first statement is " + describeStmt(decl.Body.List[0]) +
			", not an `if err := ...; err != nil` guard"
	}

	assign, ok := ifStmt.Init.(*ast.AssignStmt)
	if !ok || len(assign.Lhs) != 1 || len(assign.Rhs) != 1 {
		return nil, "", "its first statement is an `if` that does not bind a single value in its init"
	}
	call, ok := assign.Rhs[0].(*ast.CallExpr)
	if !ok || calleeName(call.Fun) != sandboxHelperGateName {
		return nil, "", "its first statement binds " + describeExpr(assign.Rhs[0]) +
			", not a call to " + sandboxHelperGateName
	}
	errIdent, ok := assign.Lhs[0].(*ast.Ident)
	if !ok {
		return nil, "", "the value returned by " + sandboxHelperGateName + " is not bound to a plain identifier"
	}

	// The error is CHECKED, and checked in the right direction. Pinning the OPERATOR
	// (and not merely "compares against nil") makes the assertion say what it means.
	cond, ok := ifStmt.Cond.(*ast.BinaryExpr)
	if !ok || cond.Op != token.NEQ {
		return nil, "", "the guard's condition is " + describeExpr(ifStmt.Cond) +
			", not `" + errIdent.Name + " != nil` — a bare or unchecked call runs the suite as a broken helper"
	}
	if !comparesIdentToNil(cond, errIdent.Name) {
		return nil, "", "the guard's condition does not compare " + errIdent.Name + " against nil"
	}

	return ifStmt, errIdent.Name, ""
}

// calleeName resolves a call's function name in BOTH spellings: a selector
// (packval.MaybeRunSandboxHelper, an *ast.SelectorExpr) and a bare identifier
// (MaybeRunSandboxHelper, how pkg/packval's own TestMain calls it). A
// selector-only matcher would red the already-correct precedent.
func calleeName(fun ast.Expr) string {
	switch f := fun.(type) {
	case *ast.SelectorExpr:
		if f.Sel == nil {
			return ""
		}
		return f.Sel.Name
	case *ast.Ident:
		return f.Name
	default:
		return ""
	}
}

// usesQualifiedGateCall reports whether the guard calls the gate through a
// package selector, which is what makes the packval import load-bearing.
func usesQualifiedGateCall(ifStmt *ast.IfStmt) bool {
	assign, ok := ifStmt.Init.(*ast.AssignStmt)
	if !ok || len(assign.Rhs) != 1 {
		return false
	}
	call, ok := assign.Rhs[0].(*ast.CallExpr)
	if !ok {
		return false
	}
	_, isSelector := call.Fun.(*ast.SelectorExpr)
	return isSelector
}

// comparesIdentToNil reports whether cond puts name on one side and nil on the other.
func comparesIdentToNil(cond *ast.BinaryExpr, name string) bool {
	return isIdentNamed(cond.X, name) && isIdentNamed(cond.Y, "nil") ||
		isIdentNamed(cond.Y, name) && isIdentNamed(cond.X, "nil")
}

func isIdentNamed(expr ast.Expr, name string) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == name
}

// exitCallArgument returns the single argument of the first os.Exit call in body.
func exitCallArgument(body *ast.BlockStmt) (ast.Expr, bool) {
	var arg ast.Expr
	var found bool
	ast.Inspect(body, func(n ast.Node) bool {
		if found {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel == nil || sel.Sel.Name != "Exit" || !isIdentNamed(sel.X, "os") {
			return true
		}
		if len(call.Args) != 1 {
			return true
		}
		arg = call.Args[0]
		found = true
		return false
	})
	return arg, found
}

// importsPath reports whether file imports the given path.
func importsPath(file *ast.File, path string) bool {
	quoted := `"` + path + `"`
	for _, imp := range file.Imports {
		if imp.Path != nil && imp.Path.Value == quoted {
			return true
		}
	}
	return false
}

func containsString(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

// describeStmt and describeExpr render a node kind for failure messages. A
// structural assertion is only as useful as the message it fails with.
func describeStmt(stmt ast.Stmt) string {
	switch s := stmt.(type) {
	case *ast.AssignStmt:
		if len(s.Lhs) > 0 {
			return "an assignment to " + describeExpr(s.Lhs[0])
		}
		return "an assignment"
	case *ast.ExprStmt:
		return "the bare expression " + describeExpr(s.X)
	case *ast.IfStmt:
		return "an if statement"
	default:
		return "a statement of type " + typeName(stmt)
	}
}

func describeExpr(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.BasicLit:
		return "the literal " + e.Value
	case *ast.SelectorExpr:
		if e.Sel != nil {
			return describeExpr(e.X) + "." + e.Sel.Name
		}
		return describeExpr(e.X) + ".?"
	case *ast.CallExpr:
		return "a call to " + describeExpr(e.Fun)
	case *ast.BinaryExpr:
		return describeExpr(e.X) + " " + e.Op.String() + " " + describeExpr(e.Y)
	case nil:
		return "nothing"
	default:
		return "an expression of type " + typeName(expr)
	}
}

// typeName renders a node's dynamic type without pulling in reflect for a
// failure message.
func typeName(node ast.Node) string {
	switch node.(type) {
	case *ast.DeclStmt:
		return "*ast.DeclStmt"
	case *ast.ReturnStmt:
		return "*ast.ReturnStmt"
	case *ast.RangeStmt:
		return "*ast.RangeStmt"
	case *ast.ForStmt:
		return "*ast.ForStmt"
	case *ast.SwitchStmt:
		return "*ast.SwitchStmt"
	case *ast.BlockStmt:
		return "*ast.BlockStmt"
	default:
		return "unrecognised ast node"
	}
}
