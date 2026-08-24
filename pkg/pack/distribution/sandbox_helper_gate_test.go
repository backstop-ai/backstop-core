package distribution_test

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/packval"
)

// sandbox_helper_gate_test.go (PLAN-ISSUE-180, CLM-001, CLM-002, CLM-004).
//
// WHAT THIS PINS. The Linux sandbox is a RE-EXEC TRAMPOLINE:
// newSandboxHelperInvocation (pkg/packval/sandbox_linux.go) spawns os.Executable()
// with BACKSTOP_SANDBOX_HELPER_SPEC set and helper.Dir pointed at the pack
// directory — and under `go test` os.Executable() is THIS PACKAGE'S OWN COMPILED
// TEST BINARY. Without a TestMain that recognises helper mode, Go's DEFAULT
// generated test main does not know that env var, so the re-exec'd process
// reruns this package's ENTIRE suite from a directory off any go.mod ancestry,
// dies, and exits 1. Because Go's testing framework writes to STDOUT while
// foldHelperStderrIntoError (pkg/packval/sandbox_diagnostic.go) reads only
// STDERR, the parent reports "the sandboxed command wrote no diagnostic" and the
// real output vanishes. That is ISSUE-180, and it is what broke
// TestInstallContractsLocalPack_InstallsWithSuppliedCommand on Linux CI.
//
// This package is a re-exec target at all because validator.go reaches the real
// pipeline through packval.NewPipeline(packDir, ...).Run().
//
// WHY STRUCTURAL AND NOT BEHAVIOURAL. The behaviour this guard fixes exists ONLY
// on Linux. pkg/packval/sandbox_linux.go and sandbox_linux_helper.go carry
// //go:build linux; on the development machine they are not compiled at all, no
// re-exec can happen, and MaybeRunSandboxHelper resolves to
// pkg/packval/sandbox_nonlinux.go's unconditional `return nil` stub. Setting
// BACKSTOP_SANDBOX_HELPER_SPEC by hand on darwin proves nothing — the stub never
// reads it — and the failing test measurably PASSES there. The AST pin is what
// IS falsifiable on every platform: delete the guard and this test goes red
// wherever it runs.
//
// WHY THIS DUPLICATES cmd/backstop's ROSTER TEST, AND MUST NOT BE
// "DEDUPLICATED" (CLM-004). cmd/backstop/sandbox_helper_testmain_guard_test.go's
// TestSandboxHelperGate_PresentInEveryPackvalReachingTestMain covers every
// packval-reaching package, this one included — but it can never RUN on the diff
// that matters. go-toolchain's `go-test` binding declares `package_scoped: true`,
// so under the diff-scoped gate CI actually runs, fileModeTestTargets
// (cmd/backstop/pack_gate.go) narrows the run to the CHANGED FILES' PACKAGES. The
// regression this pin exists for — someone deletes or mangles THIS package's
// TestMain — is by definition a diff in THIS package, which narrows the run here
// and never executes cmd/backstop's roster. The roster is the PROPAGATION pin;
// this is the one that actually FIRES.
//
// DO NOT CONFUSE package_scoped WITH THE FLAG THREE LINES BELOW IT. The same
// binding also declares `exempt_from_scope_filter: true`, which governs VIOLATION
// FILTERING (findings survive activeScope.FilterViolations instead of being
// dropped for landing outside the diff). It does NOT affect TARGET DERIVATION.
// An out-of-scope package's tests are never RUN in the first place, and a finding
// that is never produced cannot be exempted from filtering.
//
// The ~40 lines of AST helpers below are duplicated from cmd/backstop's guard on
// purpose: unexported helpers cannot cross a package boundary, and a new exported
// test-helper package is a worse trade than this.

// sandboxGateName is the packval function every member of this wiring family
// calls as the first statement of its TestMain.
const sandboxGateName = "MaybeRunSandboxHelper"

// sandboxGatePackvalImport is the import that makes a package able to become the
// re-exec target: it is the package that owns the trampoline.
const sandboxGatePackvalImport = "github.com/backstop-ai/backstop-core/pkg/packval"

// sandboxGateExitLiteral is the exit code a failed sandbox install must report.
//
// THE BARE LITERAL IS CORRECT HERE and is NOT the drift PLAN-ISSUE-163 forbade in
// cmd/backstop. That plan's "use the constant, never a bare 126" rule was SCOPED
// to cmd/backstop, where sandboxHelperExitCode (cmd/backstop/main.go) is in
// reach. It is unexported and lives in `package main`, so pkg/pack/distribution
// cannot reference it, and pkg/packval exports no equivalent. pkg/packval's own
// main_test.go spells this same literal for this same reason. The single
// underlying rule is "use the constant where one is in reach; spell the literal
// where none is" — and 126 is the fail-closed "refused to run pack code it could
// not confine" code documented on pkg/packval/sandbox_diagnostic.go.
const sandboxGateExitLiteral = "126"

// TestDistributionTestMain_OpensWithSandboxHelperGate pins CLM-001, CLM-002 and
// CLM-004: this package's test binary declares EXACTLY ONE
// `func TestMain(m *testing.M)`, and its FIRST statement is an error-CHECKED call
// to packval.MaybeRunSandboxHelper that exits 126 on failure.
//
// Under `go test` the working directory IS the package directory, so the file set
// is reachable as filepath.Glob("*_test.go") — no repo-root resolution and no
// shelling out. (cmd/backstop's repoRoot helper lives in a different package and
// is not reachable from here.)
func TestDistributionTestMain_OpensWithSandboxHelperGate(t *testing.T) {
	matches, err := filepath.Glob("*_test.go")
	if err != nil {
		t.Fatalf("glob *_test.go in the package directory: %v", err)
	}
	if len(matches) == 0 {
		t.Fatalf("glob *_test.go matched nothing in the package directory; this test would " +
			"otherwise pass while parsing no source at all")
	}

	fset := token.NewFileSet()
	type testMainDecl struct {
		file *ast.File
		path string
		decl *ast.FuncDecl
	}
	var found []testMainDecl
	for _, path := range matches {
		parsed, parseErr := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if parseErr != nil {
			t.Fatalf("parse %s: %v", path, parseErr)
		}
		if decl := sandboxGateFindTestMain(parsed); decl != nil {
			found = append(found, testMainDecl{file: parsed, path: path, decl: decl})
		}
	}

	// THE ASSERTION THAT IS RED BEFORE THE FIX. Zero means Go's DEFAULT generated
	// test main is what the Linux sandbox trampoline re-execs, so instead of
	// installing the sandbox and exec'ing the pack's command the re-exec'd copy
	// reruns this package's whole suite in the pack's scratch directory, dies off
	// any go.mod ancestry, and exits 1 with all of its output on stdout — which
	// foldHelperStderrIntoError never reads.
	if len(found) == 0 {
		t.Fatalf("pkg/pack/distribution declares NO `func TestMain(m *testing.M)` (ISSUE-180). "+
			"This package imports %s through validator.go, so on Linux the sandbox trampoline "+
			"re-execs this test binary; without a TestMain that opens with %s, Go's default test "+
			"main reruns the entire suite in the pack's scratch directory instead of installing "+
			"the sandbox, and the parent reports \"the sandboxed command wrote no diagnostic\".",
			sandboxGatePackvalImport, sandboxGateName)
	}
	// More than one is also a failure: pkg/pack/distribution compiles `package
	// distribution` and `package distribution_test` into ONE test binary, and Go
	// permits TestMain in exactly one of them. A second declaration breaks the build.
	if len(found) > 1 {
		var paths []string
		for _, f := range found {
			paths = append(paths, f.path)
		}
		t.Fatalf("pkg/pack/distribution declares %d TestMain functions (%v); Go compiles this "+
			"package's internal and external test packages into ONE test binary and permits "+
			"TestMain in exactly one of them", len(found), paths)
	}

	only := found[0]

	// POSITION ZERO. "Above the other setup" is not first: the helper has to run
	// before anything else consumes argv or does work.
	if len(only.decl.Body.List) == 0 {
		t.Fatalf("%s: TestMain's body is empty; it must open with the sandbox-helper gate", only.path)
	}
	ifStmt, ok := only.decl.Body.List[0].(*ast.IfStmt)
	if !ok {
		t.Fatalf("%s: TestMain's FIRST statement is %s, not an `if err := packval.%s(); err != nil` "+
			"guard. Position zero is the assertion — a guard inserted below any other setup fails "+
			"this pin.", only.path, sandboxGateDescribeStmt(only.decl.Body.List[0]), sandboxGateName)
	}

	// The gate is CALLED and its result is BOUND. A bare
	// packval.MaybeRunSandboxHelper() compiles and satisfies any naive "is the call
	// present" scan while silently running the suite as a broken helper —
	// PLAN-ISSUE-020 records that this exact gate shipped as a bare statement in six
	// places before being corrected. This pin asserts the `if err := ...; err != nil`
	// SHAPE, never mere presence.
	assign, ok := ifStmt.Init.(*ast.AssignStmt)
	if !ok || len(assign.Lhs) != 1 || len(assign.Rhs) != 1 {
		t.Fatalf("%s: TestMain's first statement is an `if` that does not bind a single value in "+
			"its init; expected `if err := packval.%s(); err != nil`", only.path, sandboxGateName)
	}
	call, ok := assign.Rhs[0].(*ast.CallExpr)
	if !ok || sandboxGateCalleeName(call.Fun) != sandboxGateName {
		t.Fatalf("%s: TestMain's first statement binds %s, not a call to %s",
			only.path, sandboxGateDescribeExpr(assign.Rhs[0]), sandboxGateName)
	}
	errIdent, ok := assign.Lhs[0].(*ast.Ident)
	if !ok {
		t.Fatalf("%s: the value returned by %s is not bound to a plain identifier",
			only.path, sandboxGateName)
	}

	// The error is CHECKED, and checked in the right direction. Pinning the OPERATOR
	// is what makes this say what it means rather than "compares against something".
	cond, ok := ifStmt.Cond.(*ast.BinaryExpr)
	if !ok || cond.Op != token.NEQ {
		t.Fatalf("%s: the guard's condition is %s, not `%s != nil` — an unchecked call runs the "+
			"suite as a broken helper", only.path, sandboxGateDescribeExpr(ifStmt.Cond), errIdent.Name)
	}
	if !sandboxGateComparesToNil(cond, errIdent.Name) {
		t.Fatalf("%s: the guard's condition does not compare %s against nil", only.path, errIdent.Name)
	}

	// CLM-002: the failure exit is the BARE LITERAL 126. See sandboxGateExitLiteral
	// for why the literal (and not an identifier) is correct in THIS package.
	exitArg, exitFound := sandboxGateExitArgument(ifStmt.Body)
	if !exitFound {
		t.Fatalf("%s: the sandbox-helper guard's body calls no os.Exit; a helper whose sandbox "+
			"failed to install must NOT fall through into the suite — the parent would read the "+
			"suite's output as the sandboxed command's output", only.path)
	}
	lit, ok := exitArg.(*ast.BasicLit)
	if !ok || lit.Kind != token.INT || lit.Value != sandboxGateExitLiteral {
		t.Fatalf("%s: the guard exits with %s; it must exit with the basic literal %s "+
			"(sandboxHelperExitCode is unexported in cmd/backstop's `package main` and pkg/packval "+
			"exports no equivalent, so the literal is correct here — pkg/packval/main_test.go "+
			"spells the same one)", only.path, sandboxGateDescribeExpr(exitArg), sandboxGateExitLiteral)
	}

	// A qualified call needs the import. A missing one would not compile, but naming
	// it here is what makes a partially-applied edit legible instead of cryptic.
	if _, qualified := call.Fun.(*ast.SelectorExpr); qualified && !sandboxGateImports(only.file, sandboxGatePackvalImport) {
		t.Fatalf("%s: TestMain calls packval.%s but the file does not import %q",
			only.path, sandboxGateName, sandboxGatePackvalImport)
	}
}

func TestDistributionTestMain_PropagatesDarwinHelperCompletion(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("sandbox-exec helper completion is Darwin-only")
	}
	runner, err := packval.NewSandboxRunner(packval.SandboxModeNative)
	if err != nil {
		t.Fatal(err)
	}
	packDir := t.TempDir()
	if result, err := runner.Run("/usr/bin/true", nil, packDir); err != nil {
		t.Fatalf("target exit 0 did not propagate through distribution TestMain: %v: %s", err, result.Output)
	}
	_, runErr := runner.Run("/bin/sh", []string{"-c", "exit 37"}, packDir)
	var exitErr *exec.ExitError
	if !errors.As(runErr, &exitErr) || exitErr.ExitCode() != 37 {
		t.Fatalf("completion error=%v, want exit 37", runErr)
	}
}

// sandboxGateFindTestMain returns the `func TestMain(m *testing.M)` declaration in
// file, or nil. It matches BY NAME — never by line number, which moves across lanes.
func sandboxGateFindTestMain(file *ast.File) *ast.FuncDecl {
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

// sandboxGateCalleeName resolves a call's function name in BOTH spellings: a
// selector (packval.MaybeRunSandboxHelper) and a bare identifier, so the assertion
// describes the INVARIANT rather than this one call site.
func sandboxGateCalleeName(fun ast.Expr) string {
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

// sandboxGateComparesToNil reports whether cond puts name on one side and nil on
// the other.
func sandboxGateComparesToNil(cond *ast.BinaryExpr, name string) bool {
	return sandboxGateIdentNamed(cond.X, name) && sandboxGateIdentNamed(cond.Y, "nil") ||
		sandboxGateIdentNamed(cond.Y, name) && sandboxGateIdentNamed(cond.X, "nil")
}

func sandboxGateIdentNamed(expr ast.Expr, name string) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == name
}

// sandboxGateExitArgument returns the single argument of the last os.Exit call in
// body. Typed completion exits first; the last exit is the ordinary setup error.
func sandboxGateExitArgument(body *ast.BlockStmt) (ast.Expr, bool) {
	var arg ast.Expr
	var found bool
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel == nil || sel.Sel.Name != "Exit" || !sandboxGateIdentNamed(sel.X, "os") {
			return true
		}
		if len(call.Args) != 1 {
			return true
		}
		arg = call.Args[0]
		found = true
		return true
	})
	return arg, found
}

// sandboxGateImports reports whether file imports the given path.
func sandboxGateImports(file *ast.File, path string) bool {
	quoted := `"` + path + `"`
	for _, imp := range file.Imports {
		if imp.Path != nil && imp.Path.Value == quoted {
			return true
		}
	}
	return false
}

// sandboxGateDescribeStmt and sandboxGateDescribeExpr render a node kind for
// failure messages. A structural assertion is only as useful as the message it
// fails with.
func sandboxGateDescribeStmt(stmt ast.Stmt) string {
	switch s := stmt.(type) {
	case *ast.AssignStmt:
		if len(s.Lhs) > 0 {
			return "an assignment to " + sandboxGateDescribeExpr(s.Lhs[0])
		}
		return "an assignment"
	case *ast.ExprStmt:
		return "the bare expression " + sandboxGateDescribeExpr(s.X)
	case *ast.IfStmt:
		return "an if statement"
	default:
		return "a statement of another kind"
	}
}

func sandboxGateDescribeExpr(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.BasicLit:
		return "the literal " + e.Value
	case *ast.SelectorExpr:
		if e.Sel != nil {
			return sandboxGateDescribeExpr(e.X) + "." + e.Sel.Name
		}
		return sandboxGateDescribeExpr(e.X) + ".?"
	case *ast.CallExpr:
		return "a call to " + sandboxGateDescribeExpr(e.Fun)
	case *ast.BinaryExpr:
		return sandboxGateDescribeExpr(e.X) + " " + e.Op.String() + " " + sandboxGateDescribeExpr(e.Y)
	case nil:
		return "nothing"
	default:
		return "an expression of another kind"
	}
}
