package packval

// THIS FILE CARRIES NO BUILD TAG OR OS SUFFIX, ON PURPOSE. It parses
// sandbox_linux.go as text and must guard the Linux wiring on every platform.

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/check"
)

const (
	proberWiringRealProber = "probeLandlockABI"
	proberWiringProberType = "LandlockABIProbe"

	proberWiringDispatch        = "platformSandboxedRun"
	proberWiringDispatchStdout  = "platformSandboxedRunStdout"
	proberWiringDispatchExecute = "platformSandboxedExecute"
	proberWiringRunWith         = "linuxSandboxedRunWith"
	proberWiringRunStdoutWith   = "linuxSandboxedRunStdoutWith"
	proberWiringExecuteWith     = "linuxSandboxedExecuteWith"
	proberWiringInvocation      = "newSandboxHelperInvocation"
)

func proberWiringIsTracked(name string) bool {
	return name == proberWiringExecuteWith || name == proberWiringInvocation
}

func proberWiringIsDispatch(name string) bool {
	switch name {
	case proberWiringDispatch, proberWiringDispatchStdout, proberWiringDispatchExecute:
		return true
	default:
		return false
	}
}

func proberWiringIsForward(enclosing, callee string) bool {
	return enclosing == proberWiringRunWith && callee == proberWiringExecuteWith ||
		(enclosing == proberWiringRunStdoutWith || enclosing == proberWiringExecuteWith) && callee == proberWiringInvocation
}

// proberWiringViolations is an identifier-binding check, not a dataflow proof.
// Rebinding checks cover the AST-visible assignment and declaration forms that can
// make a correctly spelled forwarded identifier carry the wrong value.
func proberWiringViolations(fset *token.FileSet, file *ast.File) (violations []string, dispatch, forward int) {
	at := func(n ast.Node) string { return fset.Position(n.Pos()).String() }
	report := func(format string, args ...any) {
		violations = append(violations, fmt.Sprintf(format, args...))
	}

	for _, decl := range file.Decls {
		fn, isFunc := decl.(*ast.FuncDecl)
		if !isFunc || fn.Body == nil {
			for _, call := range proberWiringTrackedCalls(decl) {
				report("%s: unclassified prober-carrying call to %s at FILE SCOPE", at(call), proberWiringCallee(call))
			}
			continue
		}

		enclosing := fn.Name.Name
		var dispatchCalls, forwardCalls, unclassifiedCalls []*ast.CallExpr
		for _, call := range proberWiringTrackedCalls(fn.Body) {
			callee := proberWiringCallee(call)
			switch {
			case proberWiringIsDispatch(enclosing) && callee == proberWiringExecuteWith:
				dispatchCalls = append(dispatchCalls, call)
			case proberWiringIsForward(enclosing, callee):
				forwardCalls = append(forwardCalls, call)
			default:
				unclassifiedCalls = append(unclassifiedCalls, call)
			}
		}
		dispatch += len(dispatchCalls)
		forward += len(forwardCalls)

		for _, call := range dispatchCalls {
			last, empty := proberWiringLastArg(call)
			if empty {
				report("%s: dispatch seam %s calls %s with no arguments and carries no prober", at(call), enclosing, proberWiringCallee(call))
				continue
			}
			if ident, isIdent := last.(*ast.Ident); !isIdent || ident.Name != proberWiringRealProber {
				report("%s: dispatch seam %s calls %s passing %s; the last argument must be package-level %s",
					at(call), enclosing, proberWiringCallee(call), proberWiringDescribeArg(last), proberWiringRealProber)
			}
		}
		if len(dispatchCalls) > 0 {
			for _, rebind := range proberWiringRebindsOf(fn.Body, proberWiringRealProber) {
				report("%s: dispatch seam %s RE-BINDS or SHADOWS %s, so the forwarded name no longer proves the real prober",
					at(rebind), enclosing, proberWiringRealProber)
			}
		}

		if len(forwardCalls) > 0 {
			if want, tracked := proberWiringExpectedParamCount(enclosing); tracked {
				if got := proberWiringParamCount(fn); got != want {
					report("%s: injectable seam %s declares %d parameters, want exact active signature count %d; extra parameters create a BYPASS around the guarded prober/invocation path",
						at(fn), enclosing, got, want)
				}
			}
			names := proberWiringProberParamNames(fn)
			resolved := ""
			switch len(names) {
			case 1:
				resolved = names[0]
			case 0:
				report("%s: injectable seam %s declares NO parameter of type %s", at(fn), enclosing, proberWiringProberType)
			default:
				report("%s: injectable seam %s declares %d parameters of type %s (%s); exactly one is required",
					at(fn), enclosing, len(names), proberWiringProberType, strings.Join(names, ", "))
			}

			// This must not become an else branch of the forwarding check. The full
			// option-(a) fixture relies on both checks running independently.
			if resolved == proberWiringRealProber {
				report("%s: injectable seam %s names its %s parameter %s, which SHADOWS the package-level prober",
					at(fn), enclosing, proberWiringProberType, proberWiringRealProber)
			}
			if resolved == "_" {
				report("%s: injectable seam %s uses the blank identifier for its %s parameter and discards the injected prober",
					at(fn), enclosing, proberWiringProberType)
				resolved = ""
			}

			for _, call := range forwardCalls {
				last, empty := proberWiringLastArg(call)
				if empty {
					report("%s: injectable seam %s calls %s with no arguments and carries no prober", at(call), enclosing, proberWiringCallee(call))
					continue
				}
				ident, isIdent := last.(*ast.Ident)
				if isIdent && ident.Name == proberWiringRealProber {
					report("%s: injectable seam %s calls %s with literal %s, bypassing its injected prober",
						at(call), enclosing, proberWiringCallee(call), proberWiringRealProber)
				}
				if resolved != "" && (!isIdent || ident.Name != resolved) {
					report("%s: injectable seam %s calls %s passing %s, but must forward its own prober parameter %s",
						at(call), enclosing, proberWiringCallee(call), proberWiringDescribeArg(last), resolved)
				}
			}
			if resolved != "" {
				for _, rebind := range proberWiringRebindsOf(fn.Body, resolved) {
					report("%s: injectable seam %s RE-BINDS or SHADOWS its prober parameter %s",
						at(rebind), enclosing, resolved)
				}
			}
		}

		for _, call := range unclassifiedCalls {
			report("%s: unclassified prober-carrying call site: %s calls %s", at(call), enclosing, proberWiringCallee(call))
		}
	}

	return violations, dispatch, forward
}

func proberWiringTrackedCalls(n ast.Node) []*ast.CallExpr {
	var found []*ast.CallExpr
	ast.Inspect(n, func(node ast.Node) bool {
		call, isCall := node.(*ast.CallExpr)
		if !isCall {
			return true
		}
		ident, isIdent := call.Fun.(*ast.Ident)
		if isIdent && proberWiringIsTracked(ident.Name) {
			found = append(found, call)
		}
		return true
	})
	return found
}

func proberWiringCallee(call *ast.CallExpr) string {
	if ident, isIdent := call.Fun.(*ast.Ident); isIdent {
		return ident.Name
	}
	return "an unnamed callee"
}

func proberWiringProberParamNames(fn *ast.FuncDecl) []string {
	var names []string
	if fn.Type == nil || fn.Type.Params == nil {
		return names
	}
	for _, field := range fn.Type.Params.List {
		ident, isIdent := field.Type.(*ast.Ident)
		if !isIdent || ident.Name != proberWiringProberType {
			continue
		}
		for _, name := range field.Names {
			names = append(names, name.Name)
		}
	}
	return names
}

func proberWiringExpectedParamCount(name string) (int, bool) {
	switch name {
	case proberWiringRunWith:
		return 4, true
	case proberWiringRunStdoutWith:
		return 5, true
	case proberWiringExecuteWith:
		return 6, true
	default:
		return 0, false
	}
}

func proberWiringParamCount(fn *ast.FuncDecl) int {
	if fn.Type == nil || fn.Type.Params == nil {
		return 0
	}
	count := 0
	for _, field := range fn.Type.Params.List {
		if len(field.Names) == 0 {
			count++
			continue
		}
		count += len(field.Names)
	}
	return count
}

func proberWiringLastArg(call *ast.CallExpr) (ast.Expr, bool) {
	if len(call.Args) == 0 {
		return nil, true
	}
	return call.Args[len(call.Args)-1], false
}

func proberWiringRebindsOf(body *ast.BlockStmt, name string) []ast.Node {
	var found []ast.Node
	ast.Inspect(body, func(node ast.Node) bool {
		switch bound := node.(type) {
		case *ast.AssignStmt:
			for _, lhs := range bound.Lhs {
				if ident, isIdent := lhs.(*ast.Ident); isIdent && ident.Name == name {
					found = append(found, bound)
					return true
				}
			}
		case *ast.ValueSpec:
			for _, declared := range bound.Names {
				if declared.Name == name {
					found = append(found, bound)
					return true
				}
			}
		}
		return true
	})
	return found
}

func proberWiringDescribeArg(arg ast.Expr) string {
	if ident, isIdent := arg.(*ast.Ident); isIdent {
		return ident.Name
	}
	return "a non-identifier expression"
}

func TestSandboxLinux_ProductionPathUsesTheRealABIProbe(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sandbox_linux.go", nil, 0)
	if err != nil {
		t.Fatalf("parsing sandbox_linux.go: %v", err)
	}
	violations, dispatch, forward := proberWiringViolations(fset, file)
	for _, violation := range violations {
		t.Error(violation)
	}
	if dispatch != 3 || forward != 3 || dispatch+forward != 6 {
		t.Fatalf("expected 3 exported dispatch paths to %s and 3 injected forwards to %s/%s; got dispatch=%d forward=%d total=%d",
			proberWiringExecuteWith, proberWiringExecuteWith, proberWiringInvocation, dispatch, forward, dispatch+forward)
	}
}

const proberWiringCorrectShape = `package packval

func platformSandboxedRun(command string, args []string, packDir string) ([]byte, error) {
	result, err := linuxSandboxedExecuteWith(command, args, packDir, nil, false, probeLandlockABI)
	return result.Output, err
}

func linuxSandboxedRunWith(command string, args []string, packDir string, probeABI LandlockABIProbe) ([]byte, error) {
	result, err := linuxSandboxedExecuteWith(command, args, packDir, nil, false, probeABI)
	return result.Output, err
}

func platformSandboxedRunStdout(command string, args []string, packDir string, stdin []byte) ([]byte, error) {
	result, err := linuxSandboxedExecuteWith(command, args, packDir, stdin, true, probeLandlockABI)
	return result.Output, err
}

func linuxSandboxedRunStdoutWith(command string, args []string, packDir string, stdin []byte, probeABI LandlockABIProbe) ([]byte, error) {
	invocation, err := newSandboxHelperInvocation(command, args, packDir, probeABI)
	return invocation, err
}

func platformSandboxedExecute(command string, args []string, packDir string, stdin []byte, stdoutOnly bool) (SandboxRunResult, error) {
	return linuxSandboxedExecuteWith(command, args, packDir, stdin, stdoutOnly, probeLandlockABI)
}

func linuxSandboxedExecuteWith(command string, args []string, packDir string, stdin []byte, stdoutOnly bool, probeABI LandlockABIProbe) (SandboxRunResult, error) {
	invocation, err := newSandboxHelperInvocation(command, args, packDir, probeABI)
	return invocation, err
}
`

const proberWiringFakeProberAtDispatch = `package packval
func platformSandboxedRun(command string, args []string, packDir string) {
	linuxSandboxedExecuteWith(command, args, packDir, nil, false, someOtherProbe)
}`

const proberWiringFakeProberAtExportedExecute = `package packval
func platformSandboxedExecute(command string, args []string, packDir string, stdin []byte, stdoutOnly bool) {
	linuxSandboxedExecuteWith(command, args, packDir, stdin, stdoutOnly, someOtherProbe)
}`

const proberWiringLiteralAtInnerSeam = `package packval
func linuxSandboxedRunWith(command string, args []string, packDir string, probeABI LandlockABIProbe) {
	linuxSandboxedExecuteWith(command, args, packDir, nil, false, probeLandlockABI)
}`

const proberWiringFullOptionAShape = `package packval
func linuxSandboxedRunWith(command string, args []string, packDir string, probeLandlockABI LandlockABIProbe) {
	linuxSandboxedExecuteWith(command, args, packDir, nil, false, probeLandlockABI)
}`

const proberWiringTrailingDecoySeparateFields = `package packval
func linuxSandboxedRunWith(command string, args []string, packDir string, probeABI LandlockABIProbe, decoy LandlockABIProbe) {
	linuxSandboxedExecuteWith(command, args, packDir, nil, false, decoy)
}`

const proberWiringTrailingDecoyGroupedField = `package packval
func linuxSandboxedRunWith(command string, args []string, packDir string, probeABI, decoy LandlockABIProbe) {
	linuxSandboxedExecuteWith(command, args, packDir, nil, false, decoy)
}`

const proberWiringForeignIdentifierAtInnerSeam = `package packval
func linuxSandboxedRunWith(command string, args []string, packDir string, probeABI LandlockABIProbe) {
	linuxSandboxedExecuteWith(command, args, packDir, nil, false, somethingElse)
}`

const proberWiringReboundProber = `package packval
func linuxSandboxedRunWith(command string, args []string, packDir string, probeABI LandlockABIProbe) {
	probeABI = someFake
	linuxSandboxedExecuteWith(command, args, packDir, nil, false, probeABI)
}`

const proberWiringReboundProberAtDispatch = `package packval
func platformSandboxedRun(command string, args []string, packDir string) {
	probeLandlockABI := someFake
	linuxSandboxedExecuteWith(command, args, packDir, nil, false, probeLandlockABI)
}`

const proberWiringVarFormShadowedProber = `package packval
func linuxSandboxedRunWith(command string, args []string, packDir string, probeABI LandlockABIProbe) {
	if command != "" {
		var probeABI LandlockABIProbe = someFake
		linuxSandboxedExecuteWith(command, args, packDir, nil, false, probeABI)
	}
}`

const proberWiringBlankProberParameter = `package packval
func linuxSandboxedRunWith(command string, args []string, packDir string, _ LandlockABIProbe) {
	linuxSandboxedExecuteWith(command, args, packDir, nil, false, someFake)
}`

const proberWiringUnclassifiedSite = `package packval
func someOtherHelperPath(command string, args []string, packDir string) {
	newSandboxHelperInvocation(command, args, packDir, probeLandlockABI)
}`

const proberWiringDeletedDispatchSeam = `package packval
func platformSandboxedRun(command string, args []string, packDir string) {}`

const proberWiringZeroArgTrackedCall = `package packval
func linuxSandboxedRunStdoutWith(command string, args []string, packDir string, stdin []byte, probeABI LandlockABIProbe) {
	newSandboxHelperInvocation()
}`

const proberWiringBypassParameter = `package packval
func linuxSandboxedRunStdoutWith(command string, args []string, packDir string, stdin []byte, probeABI LandlockABIProbe, prepared *sandboxHelperInvocation) {
	newSandboxHelperInvocation(command, args, packDir, probeABI)
}`

// TestSandboxLinux_ABIProbeWiringGuardFalsifiesEachDefectShape is the load-bearing
// checker mutation falsifier mandated by PLAN-ISSUE-165. The positive control
// prevents an always-failing checker from satisfying every negative case.
func TestSandboxLinux_ABIProbeWiringGuardFalsifiesEachDefectShape(t *testing.T) {
	cases := []struct {
		name          string
		why           string
		src           string
		wantViolation bool
		wantMentions  []string
		wantDispatch  int
		wantForward   int
	}{
		{"positive control: current six-site shape", "an always-failing checker must not pass", proberWiringCorrectShape, false, nil, 3, 3},
		{"fake prober at dispatch", "the real prober enters at exported dispatch", proberWiringFakeProberAtDispatch, true, []string{"platformSandboxedRun", "someOtherProbe", "probeLandlockABI"}, 1, 0},
		{"fake prober at exported execute", "the direct exported execution path is load-bearing", proberWiringFakeProberAtExportedExecute, true, []string{"platformSandboxedExecute", "someOtherProbe", "probeLandlockABI"}, 1, 0},
		{"literal real prober at inner seam", "the injected parameter must not be bypassed", proberWiringLiteralAtInnerSeam, true, []string{"linuxSandboxedRunWith", "probeLandlockABI"}, 0, 1},
		{"full option-a shadow shape", "the literal-name refusal must not be an else branch", proberWiringFullOptionAShape, true, []string{"linuxSandboxedRunWith", "shadow"}, 0, 1},
		{"trailing decoy separate fields", "typed resolution must reject multiple probers", proberWiringTrailingDecoySeparateFields, true, []string{"linuxSandboxedRunWith", "exactly one"}, 0, 1},
		{"trailing decoy grouped field", "all names on a typed field must be counted", proberWiringTrailingDecoyGroupedField, true, []string{"linuxSandboxedRunWith", "exactly one"}, 0, 1},
		{"foreign identifier at inner seam", "the injected prober must be forwarded", proberWiringForeignIdentifierAtInnerSeam, true, []string{"linuxSandboxedRunWith", "somethingElse"}, 0, 1},
		{"assigned prober", "a correctly spelled identifier can be rebound", proberWiringReboundProber, true, []string{"linuxSandboxedRunWith", "re-bind"}, 0, 1},
		{"dispatch prober shadow", "the strictest seam must detect local shadowing", proberWiringReboundProberAtDispatch, true, []string{"platformSandboxedRun", "re-bind"}, 1, 0},
		{"declaration-form shadow", "ValueSpec shadowing is distinct from assignment", proberWiringVarFormShadowedProber, true, []string{"linuxSandboxedRunWith", "shadow"}, 0, 1},
		{"blank prober parameter", "an injection seam cannot discard its injection", proberWiringBlankProberParameter, true, []string{"linuxSandboxedRunWith", "blank identifier"}, 0, 1},
		{"unclassified invocation", "new delegation sites cannot be silently trusted", proberWiringUnclassifiedSite, true, []string{"unclassified", "someOtherHelperPath"}, 0, 0},
		{"deleted exported dispatch", "counts catch a vanished direct platform path", proberWiringDeletedDispatchSeam, false, nil, 0, 0},
		{"zero-argument invocation", "malformed calls must report rather than panic", proberWiringZeroArgTrackedCall, true, []string{"linuxSandboxedRunStdoutWith", "no arguments"}, 0, 1},
		{"prepared-invocation bypass parameter", "an extra parameter can bypass production invocation construction", proberWiringBypassParameter, true, []string{"linuxSandboxedRunStdoutWith", "bypass", "exact active signature"}, 0, 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if recovered := recover(); recovered != nil {
					t.Fatalf("proberWiringViolations panicked on %q (%s): %v", tc.name, tc.why, recovered)
				}
			}()
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "sandbox_linux.go", tc.src, 0)
			if err != nil {
				t.Fatalf("fixture %q does not parse: %v", tc.name, err)
			}
			violations, dispatch, forward := proberWiringViolations(fset, file)
			if tc.wantViolation != (len(violations) > 0) {
				t.Errorf("fixture %q violations=%d, want violation=%v (%s):\n%s", tc.name, len(violations), tc.wantViolation, tc.why, strings.Join(violations, "\n"))
			}
			for _, mention := range tc.wantMentions {
				if !proberWiringAnyMentions(violations, mention) {
					t.Errorf("fixture %q has no violation mentioning %q:\n%s", tc.name, mention, strings.Join(violations, "\n"))
				}
			}
			for _, violation := range violations {
				if !strings.Contains(violation, "sandbox_linux.go:") {
					t.Errorf("fixture %q violation has no source position: %q", tc.name, violation)
				}
			}
			if dispatch != tc.wantDispatch || forward != tc.wantForward {
				t.Errorf("fixture %q counts dispatch/forward=%d/%d, want %d/%d", tc.name, dispatch, forward, tc.wantDispatch, tc.wantForward)
			}
		})
	}
}

func proberWiringAnyMentions(violations []string, needle string) bool {
	for _, violation := range violations {
		if strings.Contains(strings.ToLower(violation), strings.ToLower(needle)) {
			return true
		}
	}
	return false
}

func TestPackSandbox_NativeRunnerPlumbingPreservesConfinementPolicy(t *testing.T) {
	var nativeCalls, externalCalls int
	native := func(command string, args []string, _ string, stdin []byte, stdoutOnly bool) (SandboxRunResult, error) {
		nativeCalls++
		if command != "validator" || len(args) != 1 || args[0] != "rule.yml" || stdin != nil || stdoutOnly {
			t.Fatalf("native invocation changed: command=%q args=%q stdin=%q stdoutOnly=%v", command, args, stdin, stdoutOnly)
		}
		return sandboxRunResult([]byte("confined"), true), nil
	}
	external := func(string, []string, string, []byte, bool) (SandboxRunResult, error) {
		externalCalls++
		return SandboxRunResult{}, nil
	}
	runner, err := newSandboxRunnerWithExecution(SandboxModeNative, native, external)
	if err != nil {
		t.Fatal(err)
	}
	result, err := runner.Run("validator", []string{"rule.yml"}, t.TempDir())
	if err != nil || string(result.Output) != "confined" || !result.NativeSandboxApplied {
		t.Fatalf("native result=%#v err=%v", result, err)
	}
	if nativeCalls != 1 || externalCalls != 0 {
		t.Fatalf("native/external calls=%d/%d, want 1/0", nativeCalls, externalCalls)
	}
}

func TestPackSandbox_NativeEnvironmentSanitizationPreservesConfinement(t *testing.T) {
	input := []string{"KEEP=first", PackSandboxEnvVar + "=native", "EMPTY=", "KEEP=last", sandboxHelperEnvVar + "=secret", "META=$() * ;"}
	got := check.WithoutEnvironment(input, sandboxHelperEnvVar, PackSandboxEnvVar)
	want := []string{"KEEP=first", "EMPTY=", "KEEP=last", "META=$() * ;"}
	if !slices.Equal(got, want) {
		t.Fatalf("sanitized environment=%q, want exact ordered survivors %q", got, want)
	}
}

func TestPackSandbox_NativeEvidenceInstrumentationWiringGuard(t *testing.T) {
	source, err := os.ReadFile("sandbox_linux.go")
	if err != nil {
		t.Fatal(err)
	}
	body := string(source)
	filesystem := strings.Index(body, "applyFilesystem(restrictions)")
	syscalls := strings.Index(body, "applySeccompPolicy(restrictions, applySyscalls)")
	ack := strings.Index(body, "acknowledge(request.AckFD)")
	exec := strings.Index(body, "execTarget(resolved, argv, request.Environment)")
	if filesystem < 0 || syscalls < filesystem || ack < syscalls || exec < ack {
		t.Fatalf("injected orchestration order changed: filesystem=%d syscalls=%d ack=%d exec=%d", filesystem, syscalls, ack, exec)
	}
}
