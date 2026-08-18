package packval

// ⚠ THIS FILE CARRIES NO BUILD TAG, AND ITS NAME CARRIES NO _linux/_darwin
// COMPONENT, ON PURPOSE. Go applies an IMPLICIT build constraint from a filename
// suffix, so naming this sandbox_wiring_guard_linux_test.go would silently re-gate
// the guard to linux and quietly undo the durable half of ISSUE-165's fix while
// every local command still passed. The guard below parses sandbox_linux.go as
// TEXT — it executes no Linux syscall and names no linux-only symbol
// (LandlockABIProbe appears here only as a STRING compared against an AST node's
// spelling) — so it can, and must, run on every platform. Being //go:build linux
// is exactly why this guard's own mechanism defect survived undetected for its
// whole life until real Linux CI reached it (ISSUE-165).

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

const (
	// proberWiringRealProber is the package-level prober in sandbox_linux.go. It is
	// a STRING here, never a value: this file compiles on darwin, where that
	// function does not exist.
	proberWiringRealProber = "probeLandlockABI"
	// proberWiringProberType is the injected prober's declared type, likewise as a
	// spelling rather than as a type.
	proberWiringProberType = "LandlockABIProbe"

	proberWiringDispatch       = "platformSandboxedRun"
	proberWiringDispatchStdout = "platformSandboxedRunStdout"
	proberWiringRunWith        = "linuxSandboxedRunWith"
	proberWiringRunStdoutWith  = "linuxSandboxedRunStdoutWith"
	proberWiringHelperCommand  = "newSandboxHelperCommand"
)

// proberWiringIsTracked reports whether name is one of the three prober-carrying
// callees. A call to anything outside this set is invisible to the per-site rules —
// see the checker's header comment on why the COUNTS are what catch that.
func proberWiringIsTracked(name string) bool {
	switch name {
	case proberWiringRunWith, proberWiringRunStdoutWith, proberWiringHelperCommand:
		return true
	default:
		return false
	}
}

// ─── THE CHECKER, AND HONESTLY WHAT IT REACHES ──────────────────────────────────
//
// proberWiringViolations reports every way the parsed sandbox_linux.go fails to
// carry the real ABI prober from the platform-neutral dispatch seams down to
// newSandboxHelperCommand. It REPORTS; it never calls t.Errorf. That separation is
// what makes the falsification table in this file possible at all — a checker
// welded to *testing.T can only ever be exercised against the one correct source
// on disk, which tells you nothing about whether it would notice a wrong one.
//
// WHAT IT IS: an IDENTIFIER-BINDING check. At the injectable seams it resolves the
// forwarded name against the ENCLOSING DECL's OWN TYPED prober parameter, which is
// strictly stronger than the flat package-scope spelling match it replaces.
//
// WHAT IT IS NOT: dataflow provenance. A body that re-binds that identifier
// (probeABI = someFake) still forwards the right NAME, which is exactly why rule
// B3 flags the ASSIGNMENT — a mitigation, not a proof. Real provenance would need
// go/types, which this guard deliberately does not use: it must stay a text-and-AST
// check over a file that does not even build for the host platform. A value
// substitution that never touches an identifier this walk can see remains out of
// reach of any AST-only guard, and this checker does not pretend otherwise.
//
// AND WHAT THE PER-SITE RULES CANNOT SEE AT ALL, which is why the COUNTS the guard
// asserts are load-bearing rather than redundant: a call this walk does not
// recognise as tracked stops being COUNTED. A selector-form call
// (x.newSandboxHelperCommand(...)) has no *ast.Ident for its Fun; a renamed or
// aliased callee is not in the tracked set. Either way the site vanishes from
// dispatch/forward and the guard's count assertion is the thing that fails. Do not
// delete the counts because the per-site rules look sufficient — they are not.
// (A tracked call nested inside a FuncLit IS still seen, and is attributed to its
// enclosing FuncDecl, because this walk descends into function literals.)
func proberWiringViolations(fset *token.FileSet, file *ast.File) (violations []string, dispatch, forward int) {
	at := func(n ast.Node) string { return fset.Position(n.Pos()).String() }
	report := func(format string, args ...any) {
		violations = append(violations, fmt.Sprintf(format, args...))
	}

	for _, decl := range file.Decls {
		fn, isFunc := decl.(*ast.FuncDecl)
		if !isFunc || fn.Body == nil {
			// A tracked call outside any function body has no enclosing seam to be
			// classified against, so it is reported rather than skipped.
			for _, call := range proberWiringTrackedCalls(decl) {
				report("%s: unclassified prober-carrying call site: a call to %s at FILE SCOPE, outside any "+
					"function declaration, belongs to neither the dispatch bucket nor the forward bucket. "+
					"Every prober-carrying call must be classified rather than silently trusted",
					at(call), proberWiringCallee(call))
			}
			continue
		}

		enclosing := fn.Name.Name

		// Classify first, so bucket B's DECL-level rules fire once per seam rather
		// than once per call site inside it.
		var dispatchCalls, forwardCalls, unclassifiedCalls []*ast.CallExpr
		for _, call := range proberWiringTrackedCalls(fn.Body) {
			callee := proberWiringCallee(call)
			switch {
			case enclosing == proberWiringDispatch && callee == proberWiringRunWith,
				enclosing == proberWiringDispatchStdout && callee == proberWiringRunStdoutWith:
				dispatchCalls = append(dispatchCalls, call)
			case (enclosing == proberWiringRunWith || enclosing == proberWiringRunStdoutWith) &&
				callee == proberWiringHelperCommand:
				forwardCalls = append(forwardCalls, call)
			default:
				unclassifiedCalls = append(unclassifiedCalls, call)
			}
		}
		dispatch += len(dispatchCalls)
		forward += len(forwardCalls)

		// ── BUCKET A — THE PRODUCTION DISPATCH SEAMS (CLM-001) ──────────────────
		// platformSandboxedRun and platformSandboxedRunStdout take NO prober
		// parameter: they are the platform-neutral seam shared with
		// sandbox_nonlinux.go, and threading a Landlock-only dependency through
		// them would leak a linux concept into a contract darwin also implements.
		// So this is the ONE place the real prober enters the chain, and nothing
		// downstream can recover from getting it wrong here.
		for _, call := range dispatchCalls {
			last, empty := proberWiringLastArg(call)
			if empty {
				report("%s: the platform-neutral dispatch seam %s calls %s with no arguments at all, so it "+
					"carries no prober; a tracked call with an empty argument list is a defect, not a site to skip",
					at(call), enclosing, proberWiringCallee(call))
				continue
			}
			if ident, isIdent := last.(*ast.Ident); !isIdent || ident.Name != proberWiringRealProber {
				report("%s: the platform-neutral dispatch seam %s calls %s passing %s as its prober, but this "+
					"is the ONE hop where the real prober enters the chain, so the last argument must be the "+
					"package-level identifier %s. Anything else and the sandbox negotiates its ABI through "+
					"something other than the kernel",
					at(call), enclosing, proberWiringCallee(call),
					proberWiringDescribeArg(last), proberWiringRealProber)
			}
		}

		// ★ THE DISPATCH SEAM GETS RE-BIND PROTECTION TOO, AND IT IS NOT OPTIONAL.
		// The comment above calls this the ONE hop where the real prober enters the
		// chain and says nothing downstream can recover from getting it wrong — and
		// yet for one review round this was the WEAKER seam, because the re-bind scan
		// lived only inside bucket B. A local
		//     probeLandlockABI := someFake
		// in the dispatch body makes the forwarded identifier spell correctly while
		// naming something else entirely, and that produced ZERO violations
		// (reviewer's repro, ISSUE-165 impl-review). The strictest seam must not be
		// the least defended one.
		if len(dispatchCalls) > 0 {
			for _, rebind := range proberWiringRebindsOf(fn.Body, proberWiringRealProber) {
				report("%s: the platform-neutral dispatch seam %s RE-BINDS or SHADOWS the identifier %s inside "+
					"its own body, so the name it forwards is no longer the package-level prober even though it "+
					"still spells correctly. This is the ONE hop where the real prober enters the chain, so "+
					"nothing downstream can recover from it. This guard checks IDENTIFIER BINDING, not dataflow "+
					"provenance — real provenance would need go/types, which it deliberately does not use",
					at(rebind), enclosing, proberWiringRealProber)
			}
		}

		// ── BUCKET B — THE INJECTABLE SEAMS (CLM-002) ───────────────────────────
		if len(forwardCalls) > 0 {
			// RESOLVE THE PROBER PARAMETER BY TYPE, NEVER POSITIONALLY. A trailing
			// decoy (..., probeABI LandlockABIProbe, decoy LandlockABIProbe)
			// forwarding decoy produces ZERO violations under a positionally-last
			// rule, in both the separate-field and grouped-field spellings. By-type
			// resolution closes that and makes a two-prober signature itself loud.
			names := proberWiringProberParamNames(fn)
			resolved := ""
			switch len(names) {
			case 1:
				resolved = names[0]
			case 0:
				report("%s: the injectable seam %s calls %s but declares NO parameter of type %s; this guard "+
					"requires exactly one, and without it the injection seam it describes no longer exists",
					at(fn), enclosing, proberWiringHelperCommand, proberWiringProberType)
			default:
				report("%s: the injectable seam %s declares %d parameters of type %s (%s); this guard requires "+
					"exactly one. A second prober in scope is a decoy — it is what defeats any "+
					"positionally-last resolution rule, and it is a defect in its own right",
					at(fn), enclosing, len(names), proberWiringProberType, strings.Join(names, ", "))
			}

			// ★ B2 IS EVALUATED FIRST AND UNCONDITIONALLY, AND IT IS NEVER AN ELSE
			// BRANCH OF B1. Written as `if last != want { ... } else if last ==
			// probeLandlockABI { ... }`, the full option-(a) shape — parameter
			// RENAMED to probeLandlockABI and forwarded correctly — yields ZERO
			// violations, because B1 passes trivially when the name matches itself
			// and the else branch never fires. That is the vacuous green this guard
			// exists to refuse, reinstated by a single `else`. The falsification
			// table's option-(a) case is what enforces this ordering.
			if resolved == proberWiringRealProber {
				report("%s: the injectable seam %s names its %s parameter %s, which SHADOWS the package-level "+
					"prober inside this body and makes the forwarding assertion satisfiable by ANY "+
					"caller-supplied fake. That is option (a), refused by PLAN-ISSUE-165 and DIR-024 item 21: "+
					"it retires the exact test/production divergence this guard exists to catch",
					at(fn), enclosing, proberWiringProberType, proberWiringRealProber)
			}

			// A BLANK-IDENTIFIER PROBER PARAMETER IS ITS OWN DEFECT, and reporting it
			// as one is also what stops the B1 message below degenerating into the
			// nonsense advice "it must forward its OWN prober parameter _". A seam
			// that accepts an injection and names it `_` has thrown the injection
			// away at the signature: it cannot forward it, and every caller-supplied
			// fake is silently discarded.
			if resolved == "_" {
				report("%s: the injectable seam %s names its %s parameter _ (the blank identifier), so it "+
					"DISCARDS the prober its caller injected and can never forward it. An injection seam that "+
					"throws its injection away at the signature is a defect, not a naming choice",
					at(fn), enclosing, proberWiringProberType)
				resolved = ""
			}

			for _, call := range forwardCalls {
				last, empty := proberWiringLastArg(call)
				if empty {
					report("%s: the injectable seam %s calls %s with no arguments at all, so it carries no "+
						"prober; a tracked call with an empty argument list is a defect, not a site to skip",
						at(call), enclosing, proberWiringHelperCommand)
					continue
				}
				ident, isIdent := last.(*ast.Ident)

				// B2, second half — also unconditional.
				if isIdent && ident.Name == proberWiringRealProber {
					report("%s: the injectable seam %s calls %s passing the literal package-level identifier "+
						"%s, BYPASSING the prober its caller injected. The seam would still be there in the "+
						"signature and do nothing, so every fake-prober test in this package would silently "+
						"exercise the real kernel probe",
						at(call), enclosing, proberWiringHelperCommand, proberWiringRealProber)
				}

				// B1 — the forwarded identifier must be the resolved parameter.
				if resolved != "" && (!isIdent || ident.Name != resolved) {
					report("%s: the injectable seam %s calls %s passing %s, but it must forward its OWN prober "+
						"parameter %s. Anything else is a hardcoded prober standing where an injected one is "+
						"required, which is the test/production divergence this guard exists to prevent",
						at(call), enclosing, proberWiringHelperCommand,
						proberWiringDescribeArg(last), resolved)
				}
			}

			// B3 — the honest mitigation for the provenance this checker does not do.
			if resolved != "" {
				for _, rebind := range proberWiringRebindsOf(fn.Body, resolved) {
					report("%s: the injectable seam %s RE-BINDS or SHADOWS its prober parameter %s inside its own "+
						"body, so the forwarded name is no longer the injected value even though it still spells "+
						"correctly. This guard checks IDENTIFIER BINDING, not dataflow provenance — real "+
						"provenance would need go/types, which it deliberately does not use",
						at(rebind), enclosing, resolved)
				}
			}
		}

		// ── ANY OTHER SITE (CLM-003) ────────────────────────────────────────────
		for _, call := range unclassifiedCalls {
			report("%s: unclassified prober-carrying call site: %s calls %s, which belongs to neither the "+
				"dispatch bucket (%s / %s) nor the forward bucket (%s / %s calling %s). A new delegation must "+
				"be classified into one of the two buckets rather than silently trusted",
				at(call), enclosing, proberWiringCallee(call),
				proberWiringDispatch, proberWiringDispatchStdout,
				proberWiringRunWith, proberWiringRunStdoutWith, proberWiringHelperCommand)
		}
	}

	return violations, dispatch, forward
}

// proberWiringTrackedCalls collects every call to one of the three tracked
// functions inside n, in source order. Inspecting PER-FuncDecl is what makes the
// enclosing function known without building a parent map.
func proberWiringTrackedCalls(n ast.Node) []*ast.CallExpr {
	var found []*ast.CallExpr
	ast.Inspect(n, func(node ast.Node) bool {
		call, isCall := node.(*ast.CallExpr)
		if !isCall {
			return true
		}
		ident, isIdent := call.Fun.(*ast.Ident)
		if !isIdent {
			return true
		}
		if proberWiringIsTracked(ident.Name) {
			found = append(found, call)
		}
		return true
	})
	return found
}

// proberWiringCallee names the tracked callee. Every caller has already filtered
// for an *ast.Ident Fun, so the assertion cannot fail here.
func proberWiringCallee(call *ast.CallExpr) string {
	if ident, isIdent := call.Fun.(*ast.Ident); isIdent {
		return ident.Name
	}
	return "an unnamed callee"
}

// proberWiringProberParamNames resolves the decl's prober parameters BY TYPE —
// every *ast.Field whose type is the identifier LandlockABIProbe — and returns
// EVERY name those fields declare. A single field may declare several names, so
// the grouped spelling `probeABI, decoy LandlockABIProbe` correctly yields two.
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

// proberWiringLastArg returns the call's last argument, reporting empty=true for a
// zero-argument call. Indexing that case directly is an index-out-of-range panic,
// which reads as a broken test rather than as the defect it is.
func proberWiringLastArg(call *ast.CallExpr) (arg ast.Expr, empty bool) {
	if len(call.Args) == 0 {
		return nil, true
	}
	return call.Args[len(call.Args)-1], false
}

// proberWiringRebindsOf finds every place inside body that binds name to something
// new — the AST-visible ways a correctly-SPELLED forward can be carrying the wrong
// VALUE.
//
// TWO NODE SHAPES, AND THE SECOND ONE IS EASY TO MISS. An *ast.AssignStmt covers
// `name = x` and `name := x`. It does NOT cover the DECLARATION form
// `var name LandlockABIProbe = someFake`, which the parser gives as an
// *ast.DeclStmt wrapping a *ast.GenDecl of *ast.ValueSpec — a different node type
// entirely, and in a nested block it is legal Go that shadows the parameter for
// every statement after it. An assignment-only scan reported ZERO violations on
// exactly that shape (reviewer's repro, ISSUE-165 impl-review), so both are matched
// here. Matching the ValueSpec also flags a shadow declared in an inner block even
// when it never reaches the forward: this guard resolves no scopes, so it reports
// the ambiguity rather than guessing which binding wins.
//
// STILL OUT OF REACH, DELIBERATELY: a FuncLit whose own PARAMETER shadows the
// prober identifier. That needs FuncLit-parameter-shadow detection and is tracked
// as a follow-on to ISSUE-165, not silently absorbed.
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

// proberWiringDescribeArg renders an argument for a failure message.
func proberWiringDescribeArg(arg ast.Expr) string {
	if ident, isIdent := arg.(*ast.Ident); isIdent {
		return ident.Name
	}
	return "a non-identifier expression"
}

// TestSandboxLinux_ProductionPathUsesTheRealABIProbe is the WIRING GUARD for the
// injectable ABI-prober seam.
//
// A seam that a test can fill is a seam PRODUCTION can be left holding the wrong
// thing in. That divergence — test and production taking different paths through
// the same code — caused two of ISSUE-020's three runner failures, so the seam
// ships with a structural assertion. Parsing, not executing, because these lines
// only run on a host with a sandbox.
//
// ─── WHY THE INNER HOP IS ASSERTED AGAINST A TYPED PARAMETER, NOT A SPELLING ────
// This guard originally required the literal identifier probeLandlockABI at all
// four call sites. The two INNER sites cannot satisfy that and never could: they
// forward their own injected parameter, which is the whole point of the seam. That
// false positive is ISSUE-165. The obvious repair — rename the parameter to
// probeLandlockABI so the flat check happens to match — is REFUSED here and by
// DIR-024 item 21, because such a parameter SHADOWS the package-level function
// inside both bodies and the assertion then passes for ANY caller-supplied prober,
// including the fakes this package's own tests inject. That is green without being
// true. The checker therefore resolves each inner hop against its enclosing decl's
// exactly-one LandlockABIProbe parameter, and refuses that rename mechanically.
//
// ─── WHY THIS FILE IS UNTAGGED ─────────────────────────────────────────────────
// This guard lived in a //go:build linux file for its whole life, so it never
// compiled — let alone ran — on the darwin machine this suite is authored on, and
// its own mechanism defect stayed invisible until real Linux CI reached it. It
// parses source TEXT and executes no Linux syscall, so there was never a reason for
// it to be gated. It now runs everywhere.
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

	// TWO dispatch delegations + TWO inner forwards = four prober-carrying sites.
	// The counts are asserted PER BUCKET and in total so the guard cannot pass
	// vacuously after a rename or a deletion: it asserts a property OF those sites
	// and is meaningless if they moved. They are also the only thing that catches a
	// site the per-site rules cannot see as tracked at all — a selector-form call,
	// or a callee renamed out of the tracked set — because such a site stops being
	// counted.
	if dispatch != 2 || forward != 2 || dispatch+forward != 4 {
		t.Fatalf("expected 2 dispatch delegations (%s -> %s, %s -> %s) and 2 inner forwards to %s, "+
			"found dispatch=%d forward=%d total=%d — this guard asserts a property OF those sites and is "+
			"meaningless if they moved",
			proberWiringDispatch, proberWiringRunWith,
			proberWiringDispatchStdout, proberWiringRunStdoutWith,
			proberWiringHelperCommand, dispatch, forward, dispatch+forward)
	}
}

// ─── THE FALSIFICATION TABLE ────────────────────────────────────────────────────
//
// The guard above only ever sees ONE source: the correct sandbox_linux.go on disk.
// So it can tell us the production wiring is right, and nothing at all about
// whether the checker would notice if it were wrong. Every rule the checker claims
// is proved — or not — HERE, against doctored-but-PARSING miniature sources.
//
// Each fixture declares the four real seams with their real names and signatures,
// bodies trimmed to the call under test. They are never type-checked, so undeclared
// identifiers (someOtherProbe, someFake, ...) are fine; a SYNTAX error is not,
// because a fixture that fails to parse never reaches the checker and its subtest
// would green for entirely the wrong reason. Every case therefore asserts its own
// parse succeeded before it asserts anything else.

// proberWiringCorrectShape is the POSITIVE CONTROL, copied faithfully from
// sandbox_linux.go. It is LOAD-BEARING: without a correct-shape fixture asserted to
// yield ZERO violations, a checker that flags absolutely everything would pass
// every negative case in this table.
const proberWiringCorrectShape = `package packval

func platformSandboxedRun(command string, args []string, packDir string) ([]byte, error) {
	return linuxSandboxedRunWith(command, args, packDir, probeLandlockABI)
}

func linuxSandboxedRunWith(command string, args []string, packDir string, probeABI LandlockABIProbe) ([]byte, error) {
	helper, err := newSandboxHelperCommand(command, args, packDir, probeABI)
	if err != nil {
		return nil, err
	}
	return helper.CombinedOutput()
}

func platformSandboxedRunStdout(command string, args []string, packDir string, stdin []byte) ([]byte, error) {
	return linuxSandboxedRunStdoutWith(command, args, packDir, stdin, probeLandlockABI)
}

func linuxSandboxedRunStdoutWith(command string, args []string, packDir string, stdin []byte, probeABI LandlockABIProbe) ([]byte, error) {
	helper, err := newSandboxHelperCommand(command, args, packDir, probeABI)
	if err != nil {
		return nil, err
	}
	return helper.CombinedOutput()
}
`

// proberWiringFakeProberAtDispatch: the platform-neutral dispatch seam hands
// linuxSandboxedRunWith something other than the real package-level prober. This is
// the ONE hop where the real prober enters the chain, so nothing downstream can
// recover from it.
const proberWiringFakeProberAtDispatch = `package packval

func platformSandboxedRun(command string, args []string, packDir string) ([]byte, error) {
	return linuxSandboxedRunWith(command, args, packDir, someOtherProbe)
}

func linuxSandboxedRunWith(command string, args []string, packDir string, probeABI LandlockABIProbe) ([]byte, error) {
	helper, err := newSandboxHelperCommand(command, args, packDir, probeABI)
	if err != nil {
		return nil, err
	}
	return helper.CombinedOutput()
}

func platformSandboxedRunStdout(command string, args []string, packDir string, stdin []byte) ([]byte, error) {
	return linuxSandboxedRunStdoutWith(command, args, packDir, stdin, probeLandlockABI)
}

func linuxSandboxedRunStdoutWith(command string, args []string, packDir string, stdin []byte, probeABI LandlockABIProbe) ([]byte, error) {
	helper, err := newSandboxHelperCommand(command, args, packDir, probeABI)
	if err != nil {
		return nil, err
	}
	return helper.CombinedOutput()
}
`

// proberWiringLiteralAtInnerSeam: the injectable seam bypasses its own injected
// parameter and reaches for the package-level probeLandlockABI directly. This is
// the state in which every fake-prober test in this package silently drives the
// REAL kernel probe — the injection seam is still there in the signature and does
// nothing at all.
const proberWiringLiteralAtInnerSeam = `package packval

func platformSandboxedRun(command string, args []string, packDir string) ([]byte, error) {
	return linuxSandboxedRunWith(command, args, packDir, probeLandlockABI)
}

func linuxSandboxedRunWith(command string, args []string, packDir string, probeABI LandlockABIProbe) ([]byte, error) {
	helper, err := newSandboxHelperCommand(command, args, packDir, probeLandlockABI)
	if err != nil {
		return nil, err
	}
	return helper.CombinedOutput()
}

func platformSandboxedRunStdout(command string, args []string, packDir string, stdin []byte) ([]byte, error) {
	return linuxSandboxedRunStdoutWith(command, args, packDir, stdin, probeLandlockABI)
}

func linuxSandboxedRunStdoutWith(command string, args []string, packDir string, stdin []byte, probeABI LandlockABIProbe) ([]byte, error) {
	helper, err := newSandboxHelperCommand(command, args, packDir, probeABI)
	if err != nil {
		return nil, err
	}
	return helper.CombinedOutput()
}
`

// ★ proberWiringFullOptionAShape IS THE MOST IMPORTANT FIXTURE IN THIS TABLE.
//
// It is option (a) in full: the injectable seam's parameter RENAMED to
// probeLandlockABI and then forwarded perfectly correctly. Every name matches
// itself, so a checker whose literal-name refusal is written as the natural `else
// if` of the forwarding check returns ZERO violations here — the plan reviewer
// reproduced exactly that against a prototype. The parameter SHADOWS the
// package-level prober inside the body, so the forwarding assertion becomes
// satisfiable by ANY caller-supplied fake: green without being true, and the exact
// shape PLAN-ISSUE-165 and DIR-024 item 21 went to the trouble of refusing.
//
// This case is what makes the rule ORDERING enforced by a test rather than by a
// comment. If it ever passes, the lane has silently shipped option (a).
const proberWiringFullOptionAShape = `package packval

func platformSandboxedRun(command string, args []string, packDir string) ([]byte, error) {
	return linuxSandboxedRunWith(command, args, packDir, probeLandlockABI)
}

func linuxSandboxedRunWith(command string, args []string, packDir string, probeLandlockABI LandlockABIProbe) ([]byte, error) {
	helper, err := newSandboxHelperCommand(command, args, packDir, probeLandlockABI)
	if err != nil {
		return nil, err
	}
	return helper.CombinedOutput()
}

func platformSandboxedRunStdout(command string, args []string, packDir string, stdin []byte) ([]byte, error) {
	return linuxSandboxedRunStdoutWith(command, args, packDir, stdin, probeLandlockABI)
}

func linuxSandboxedRunStdoutWith(command string, args []string, packDir string, stdin []byte, probeABI LandlockABIProbe) ([]byte, error) {
	helper, err := newSandboxHelperCommand(command, args, packDir, probeABI)
	if err != nil {
		return nil, err
	}
	return helper.CombinedOutput()
}
`

// proberWiringTrailingDecoySeparateFields: a second LandlockABIProbe parameter is
// appended and forwarded in place of the real one. Under a positionally-last rule
// ("the enclosing decl's LAST parameter") this yields ZERO violations — measured by
// the plan reviewer — which is why the prober parameter is resolved BY TYPE and why
// a two-prober signature is itself a violation.
const proberWiringTrailingDecoySeparateFields = `package packval

func platformSandboxedRun(command string, args []string, packDir string) ([]byte, error) {
	return linuxSandboxedRunWith(command, args, packDir, probeLandlockABI)
}

func linuxSandboxedRunWith(command string, args []string, packDir string, probeABI LandlockABIProbe, decoy LandlockABIProbe) ([]byte, error) {
	helper, err := newSandboxHelperCommand(command, args, packDir, decoy)
	if err != nil {
		return nil, err
	}
	return helper.CombinedOutput()
}

func platformSandboxedRunStdout(command string, args []string, packDir string, stdin []byte) ([]byte, error) {
	return linuxSandboxedRunStdoutWith(command, args, packDir, stdin, probeLandlockABI)
}

func linuxSandboxedRunStdoutWith(command string, args []string, packDir string, stdin []byte, probeABI LandlockABIProbe) ([]byte, error) {
	helper, err := newSandboxHelperCommand(command, args, packDir, probeABI)
	if err != nil {
		return nil, err
	}
	return helper.CombinedOutput()
}
`

// proberWiringTrailingDecoyGroupedField is the same defect in the GROUPED spelling
// (probeABI, decoy LandlockABIProbe), where one *ast.Field carries two names. It is
// listed separately because a by-type resolver that read only Field.Names[0] would
// pass this one while catching the separate-field form.
const proberWiringTrailingDecoyGroupedField = `package packval

func platformSandboxedRun(command string, args []string, packDir string) ([]byte, error) {
	return linuxSandboxedRunWith(command, args, packDir, probeLandlockABI)
}

func linuxSandboxedRunWith(command string, args []string, packDir string, probeABI, decoy LandlockABIProbe) ([]byte, error) {
	helper, err := newSandboxHelperCommand(command, args, packDir, decoy)
	if err != nil {
		return nil, err
	}
	return helper.CombinedOutput()
}

func platformSandboxedRunStdout(command string, args []string, packDir string, stdin []byte) ([]byte, error) {
	return linuxSandboxedRunStdoutWith(command, args, packDir, stdin, probeLandlockABI)
}

func linuxSandboxedRunStdoutWith(command string, args []string, packDir string, stdin []byte, probeABI LandlockABIProbe) ([]byte, error) {
	helper, err := newSandboxHelperCommand(command, args, packDir, probeABI)
	if err != nil {
		return nil, err
	}
	return helper.CombinedOutput()
}
`

// proberWiringForeignIdentifierAtInnerSeam: the injectable seam forwards some other
// in-scope identifier instead of its own prober parameter. The injected prober is
// accepted and then thrown away.
const proberWiringForeignIdentifierAtInnerSeam = `package packval

func platformSandboxedRun(command string, args []string, packDir string) ([]byte, error) {
	return linuxSandboxedRunWith(command, args, packDir, probeLandlockABI)
}

func linuxSandboxedRunWith(command string, args []string, packDir string, probeABI LandlockABIProbe) ([]byte, error) {
	helper, err := newSandboxHelperCommand(command, args, packDir, somethingElse)
	if err != nil {
		return nil, err
	}
	return helper.CombinedOutput()
}

func platformSandboxedRunStdout(command string, args []string, packDir string, stdin []byte) ([]byte, error) {
	return linuxSandboxedRunStdoutWith(command, args, packDir, stdin, probeLandlockABI)
}

func linuxSandboxedRunStdoutWith(command string, args []string, packDir string, stdin []byte, probeABI LandlockABIProbe) ([]byte, error) {
	helper, err := newSandboxHelperCommand(command, args, packDir, probeABI)
	if err != nil {
		return nil, err
	}
	return helper.CombinedOutput()
}
`

// proberWiringReboundProber: the seam re-binds its own prober parameter before
// forwarding it.
//
// THE HONEST LIMIT LIVES HERE. The forward itself still looks perfectly correct to
// a name check — probeABI is passed in probeABI's position — which is precisely why
// the checker flags the ASSIGNMENT rather than claiming dataflow provenance it does
// not have. A value substitution that never touches this identifier remains out of
// reach of any AST-only guard, and this table does not pretend otherwise.
const proberWiringReboundProber = `package packval

func platformSandboxedRun(command string, args []string, packDir string) ([]byte, error) {
	return linuxSandboxedRunWith(command, args, packDir, probeLandlockABI)
}

func linuxSandboxedRunWith(command string, args []string, packDir string, probeABI LandlockABIProbe) ([]byte, error) {
	probeABI = someFake
	helper, err := newSandboxHelperCommand(command, args, packDir, probeABI)
	if err != nil {
		return nil, err
	}
	return helper.CombinedOutput()
}

func platformSandboxedRunStdout(command string, args []string, packDir string, stdin []byte) ([]byte, error) {
	return linuxSandboxedRunStdoutWith(command, args, packDir, stdin, probeLandlockABI)
}

func linuxSandboxedRunStdoutWith(command string, args []string, packDir string, stdin []byte, probeABI LandlockABIProbe) ([]byte, error) {
	helper, err := newSandboxHelperCommand(command, args, packDir, probeABI)
	if err != nil {
		return nil, err
	}
	return helper.CombinedOutput()
}
`

// ★ proberWiringReboundProberAtDispatch is the DISPATCH-SEAM twin of the case
// above, and it is here because for one review round it was the hole the guard's
// own prose said could not exist.
//
// The forwarded identifier spells probeLandlockABI exactly, so the bucket-A
// last-argument rule passes — while a local binding one line earlier has made that
// name mean someFake. The re-bind scan originally ran only inside the bucket-B
// branch, so this produced ZERO violations at the seam the checker calls "the ONE
// place the real prober enters the chain, and nothing downstream can recover from
// getting it wrong here." The strictest seam must not be the least defended one.
const proberWiringReboundProberAtDispatch = `package packval

func platformSandboxedRun(command string, args []string, packDir string) ([]byte, error) {
	probeLandlockABI := someFake
	return linuxSandboxedRunWith(command, args, packDir, probeLandlockABI)
}

func linuxSandboxedRunWith(command string, args []string, packDir string, probeABI LandlockABIProbe) ([]byte, error) {
	helper, err := newSandboxHelperCommand(command, args, packDir, probeABI)
	if err != nil {
		return nil, err
	}
	return helper.CombinedOutput()
}

func platformSandboxedRunStdout(command string, args []string, packDir string, stdin []byte) ([]byte, error) {
	return linuxSandboxedRunStdoutWith(command, args, packDir, stdin, probeLandlockABI)
}

func linuxSandboxedRunStdoutWith(command string, args []string, packDir string, stdin []byte, probeABI LandlockABIProbe) ([]byte, error) {
	helper, err := newSandboxHelperCommand(command, args, packDir, probeABI)
	if err != nil {
		return nil, err
	}
	return helper.CombinedOutput()
}
`

// ★ proberWiringVarFormShadowedProber is the DECLARATION-form re-bind, and it is
// legal, compiling Go rather than a contrivance: an inner block declares
// `var probeABI LandlockABIProbe = someFake`, shadowing the injected parameter, and
// the forward inside that block picks up the shadow. The forwarded name still
// spells probeABI, so B1 passes; the parameter is still correctly named, so B2
// passes. An assignment-only re-bind scan sees an *ast.AssignStmt and this is an
// *ast.DeclStmt/*ast.ValueSpec, so it reported ZERO violations.
const proberWiringVarFormShadowedProber = `package packval

func platformSandboxedRun(command string, args []string, packDir string) ([]byte, error) {
	return linuxSandboxedRunWith(command, args, packDir, probeLandlockABI)
}

func linuxSandboxedRunWith(command string, args []string, packDir string, probeABI LandlockABIProbe) ([]byte, error) {
	if command != "" {
		var probeABI LandlockABIProbe = someFake
		helper, err := newSandboxHelperCommand(command, args, packDir, probeABI)
		if err != nil {
			return nil, err
		}
		return helper.CombinedOutput()
	}
	return nil, nil
}

func platformSandboxedRunStdout(command string, args []string, packDir string, stdin []byte) ([]byte, error) {
	return linuxSandboxedRunStdoutWith(command, args, packDir, stdin, probeLandlockABI)
}

func linuxSandboxedRunStdoutWith(command string, args []string, packDir string, stdin []byte, probeABI LandlockABIProbe) ([]byte, error) {
	helper, err := newSandboxHelperCommand(command, args, packDir, probeABI)
	if err != nil {
		return nil, err
	}
	return helper.CombinedOutput()
}
`

// proberWiringBlankProberParameter: the injectable seam names its prober parameter
// `_`. The seam still ADVERTISES an injection point in its signature and then
// discards it, so every caller-supplied fake is silently thrown away — and before
// this was its own rule, the B1 message degenerated into the nonsense advice "it
// must forward its OWN prober parameter _", which is itself evidence of the defect
// rather than a diagnosis of it.
const proberWiringBlankProberParameter = `package packval

func platformSandboxedRun(command string, args []string, packDir string) ([]byte, error) {
	return linuxSandboxedRunWith(command, args, packDir, probeLandlockABI)
}

func linuxSandboxedRunWith(command string, args []string, packDir string, _ LandlockABIProbe) ([]byte, error) {
	helper, err := newSandboxHelperCommand(command, args, packDir, someFake)
	if err != nil {
		return nil, err
	}
	return helper.CombinedOutput()
}

func platformSandboxedRunStdout(command string, args []string, packDir string, stdin []byte) ([]byte, error) {
	return linuxSandboxedRunStdoutWith(command, args, packDir, stdin, probeLandlockABI)
}

func linuxSandboxedRunStdoutWith(command string, args []string, packDir string, stdin []byte, probeABI LandlockABIProbe) ([]byte, error) {
	helper, err := newSandboxHelperCommand(command, args, packDir, probeABI)
	if err != nil {
		return nil, err
	}
	return helper.CombinedOutput()
}
`

// proberWiringUnclassifiedSite: a third delegation appears that nobody classified.
// A silently skipped site is the failure mode this arm exists to prevent — the four
// known sites stay correct and a fifth quietly does whatever it likes.
const proberWiringUnclassifiedSite = `package packval

func platformSandboxedRun(command string, args []string, packDir string) ([]byte, error) {
	return linuxSandboxedRunWith(command, args, packDir, probeLandlockABI)
}

func linuxSandboxedRunWith(command string, args []string, packDir string, probeABI LandlockABIProbe) ([]byte, error) {
	helper, err := newSandboxHelperCommand(command, args, packDir, probeABI)
	if err != nil {
		return nil, err
	}
	return helper.CombinedOutput()
}

func platformSandboxedRunStdout(command string, args []string, packDir string, stdin []byte) ([]byte, error) {
	return linuxSandboxedRunStdoutWith(command, args, packDir, stdin, probeLandlockABI)
}

func linuxSandboxedRunStdoutWith(command string, args []string, packDir string, stdin []byte, probeABI LandlockABIProbe) ([]byte, error) {
	helper, err := newSandboxHelperCommand(command, args, packDir, probeABI)
	if err != nil {
		return nil, err
	}
	return helper.CombinedOutput()
}

func someOtherHelperPath(command string, args []string, packDir string) ([]byte, error) {
	helper, err := newSandboxHelperCommand(command, args, packDir, probeLandlockABI)
	if err != nil {
		return nil, err
	}
	return helper.CombinedOutput()
}
`

// proberWiringDeletedDispatchSeam: one dispatch delegation is gone. NOTHING here is
// individually wrong, which is the point — the per-site rules have nothing to say
// and only the COUNTS notice. This is why the guard asserts per-bucket counts and
// why they must never be deleted as redundant.
const proberWiringDeletedDispatchSeam = `package packval

func platformSandboxedRun(command string, args []string, packDir string) ([]byte, error) {
	return nil, nil
}

func linuxSandboxedRunWith(command string, args []string, packDir string, probeABI LandlockABIProbe) ([]byte, error) {
	helper, err := newSandboxHelperCommand(command, args, packDir, probeABI)
	if err != nil {
		return nil, err
	}
	return helper.CombinedOutput()
}

func platformSandboxedRunStdout(command string, args []string, packDir string, stdin []byte) ([]byte, error) {
	return linuxSandboxedRunStdoutWith(command, args, packDir, stdin, probeLandlockABI)
}

func linuxSandboxedRunStdoutWith(command string, args []string, packDir string, stdin []byte, probeABI LandlockABIProbe) ([]byte, error) {
	helper, err := newSandboxHelperCommand(command, args, packDir, probeABI)
	if err != nil {
		return nil, err
	}
	return helper.CombinedOutput()
}
`

// proberWiringZeroArgTrackedCall: a tracked call with NO arguments at all.
// Indexing call.Args[len(call.Args)-1] on this is an index-out-of-range PANIC,
// which reads as a broken test rather than as the defect it is.
const proberWiringZeroArgTrackedCall = `package packval

func platformSandboxedRun(command string, args []string, packDir string) ([]byte, error) {
	return linuxSandboxedRunWith(command, args, packDir, probeLandlockABI)
}

func linuxSandboxedRunWith(command string, args []string, packDir string, probeABI LandlockABIProbe) ([]byte, error) {
	helper, err := newSandboxHelperCommand()
	if err != nil {
		return nil, err
	}
	return helper.CombinedOutput()
}

func platformSandboxedRunStdout(command string, args []string, packDir string, stdin []byte) ([]byte, error) {
	return linuxSandboxedRunStdoutWith(command, args, packDir, stdin, probeLandlockABI)
}

func linuxSandboxedRunStdoutWith(command string, args []string, packDir string, stdin []byte, probeABI LandlockABIProbe) ([]byte, error) {
	helper, err := newSandboxHelperCommand(command, args, packDir, probeABI)
	if err != nil {
		return nil, err
	}
	return helper.CombinedOutput()
}
`

// TestSandboxLinux_ABIProbeWiringGuardFalsifiesEachDefectShape drives
// proberWiringViolations against one POSITIVE CONTROL plus one fixture per defect
// shape the guard claims to catch, and asserts each is reported — by a message that
// NAMES ITS SITE, not merely by a nonzero count.
//
// This is the only mechanical falsifier this lane has for the checker's rules. The
// guard proper reads the one correct sandbox_linux.go on disk and therefore cannot
// tell a working checker from one that has been quietly weakened.
func TestSandboxLinux_ABIProbeWiringGuardFalsifiesEachDefectShape(t *testing.T) {
	cases := []struct {
		name string
		// why states what real defect this shape stands for, so a future reader
		// deleting a case has to argue with the reason rather than the fixture.
		why string
		src string
		// wantViolation is false ONLY for the positive control and for the deleted
		// seam, where the counts — not the per-site rules — are what notice.
		wantViolation bool
		// wantMentions are substrings (matched case-insensitively) that must each
		// appear in at least ONE violation. They pin the SPECIFIC rule that must
		// fire, so a checker returning generic strings cannot satisfy this table.
		wantMentions []string
		wantDispatch int
		wantForward  int
	}{
		{
			name:          "positive control: the correct four-site shape",
			why:           "without this, a checker that flags everything passes every negative case below",
			src:           proberWiringCorrectShape,
			wantViolation: false,
			wantDispatch:  2,
			wantForward:   2,
		},
		{
			name:          "fake prober at a dispatch seam",
			why:           "the dispatch seams are the ONE place the real prober enters the chain",
			src:           proberWiringFakeProberAtDispatch,
			wantViolation: true,
			wantMentions:  []string{"platformSandboxedRun", "someOtherProbe", "probeLandlockABI"},
			wantDispatch:  2,
			wantForward:   2,
		},
		{
			name:          "literal probeLandlockABI at an inner seam",
			why:           "the injection seam still exists in the signature but does nothing; every fake-prober test would drive the real kernel probe",
			src:           proberWiringLiteralAtInnerSeam,
			wantViolation: true,
			wantMentions:  []string{"linuxSandboxedRunWith", "probeLandlockABI"},
			wantDispatch:  2,
			wantForward:   2,
		},
		{
			name:          "the full option-(a) shape: parameter renamed to probeLandlockABI AND forwarded correctly",
			why:           "a literal-name refusal written as an else-branch of the forwarding check returns ZERO violations here; this case is what stops the lane shipping the shape it refused",
			src:           proberWiringFullOptionAShape,
			wantViolation: true,
			wantMentions:  []string{"linuxSandboxedRunWith", "shadow"},
			wantDispatch:  2,
			wantForward:   2,
		},
		{
			name:          "trailing decoy LandlockABIProbe parameter, separate fields",
			why:           "a positionally-last rule yields ZERO violations here; by-type resolution is what closes it",
			src:           proberWiringTrailingDecoySeparateFields,
			wantViolation: true,
			wantMentions:  []string{"linuxSandboxedRunWith", "exactly one"},
			wantDispatch:  2,
			wantForward:   2,
		},
		{
			name:          "trailing decoy LandlockABIProbe parameter, grouped field",
			why:           "one *ast.Field carries two names here; a resolver reading only Names[0] would miss it",
			src:           proberWiringTrailingDecoyGroupedField,
			wantViolation: true,
			wantMentions:  []string{"linuxSandboxedRunWith", "exactly one"},
			wantDispatch:  2,
			wantForward:   2,
		},
		{
			name:          "foreign identifier at an inner seam",
			why:           "the injected prober is accepted and then thrown away",
			src:           proberWiringForeignIdentifierAtInnerSeam,
			wantViolation: true,
			wantMentions:  []string{"linuxSandboxedRunWith", "somethingElse"},
			wantDispatch:  2,
			wantForward:   2,
		},
		{
			name:          "re-bound prober identifier inside the seam body",
			why:           "the forward still looks correct to a name check; the ASSIGNMENT is what is flagged, and that is a mitigation rather than dataflow provenance",
			src:           proberWiringReboundProber,
			wantViolation: true,
			wantMentions:  []string{"linuxSandboxedRunWith", "re-bind"},
			wantDispatch:  2,
			wantForward:   2,
		},
		{
			name:          "re-bound prober identifier inside a DISPATCH seam body",
			why:           "the forwarded name spells probeLandlockABI exactly while meaning someFake; the strictest seam must not be the least defended one",
			src:           proberWiringReboundProberAtDispatch,
			wantViolation: true,
			wantMentions:  []string{"platformSandboxedRun", "re-bind"},
			wantDispatch:  2,
			wantForward:   2,
		},
		{
			name:          "declaration-form shadow of the prober parameter",
			why:           "var probeABI LandlockABIProbe = someFake is an *ast.DeclStmt/*ast.ValueSpec, not an *ast.AssignStmt, so an assignment-only scan sees nothing",
			src:           proberWiringVarFormShadowedProber,
			wantViolation: true,
			wantMentions:  []string{"linuxSandboxedRunWith", "shadow"},
			wantDispatch:  2,
			wantForward:   2,
		},
		{
			name:          "blank-identifier prober parameter",
			why:           "the seam advertises an injection point and discards it; without its own rule the forwarding message degenerates into advice to forward `_`",
			src:           proberWiringBlankProberParameter,
			wantViolation: true,
			wantMentions:  []string{"linuxSandboxedRunWith", "blank identifier"},
			wantDispatch:  2,
			wantForward:   2,
		},
		{
			name:          "a tracked call in an unclassified enclosing function",
			why:           "a silently skipped site is the failure mode this arm exists to prevent",
			src:           proberWiringUnclassifiedSite,
			wantViolation: true,
			wantMentions:  []string{"unclassified", "someOtherHelperPath"},
			wantDispatch:  2,
			wantForward:   2,
		},
		{
			name:          "a deleted dispatch seam shows up in the counts",
			why:           "nothing is individually wrong; only the per-bucket counts notice, which is why they are asserted",
			src:           proberWiringDeletedDispatchSeam,
			wantViolation: false,
			wantDispatch:  1,
			wantForward:   2,
		},
		{
			name:          "a zero-argument tracked call is a violation, not a panic",
			why:           "indexing the last argument of an empty argument list reads as a broken test rather than as the defect it is",
			src:           proberWiringZeroArgTrackedCall,
			wantViolation: true,
			wantMentions:  []string{"linuxSandboxedRunWith", "no arguments"},
			wantDispatch:  2,
			wantForward:   2,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if recovered := recover(); recovered != nil {
					t.Fatalf("proberWiringViolations PANICKED on the %q fixture (%s): %v.\n"+
						"A panic here reads as a broken test rather than as the defect the fixture "+
						"encodes; the checker must report every malformed shape as a violation instead",
						tc.name, tc.why, recovered)
				}
			}()

			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "sandbox_linux.go", tc.src, 0)
			if err != nil {
				t.Fatalf("the %q fixture does not PARSE: %v.\n"+
					"A fixture with a syntax error never reaches the checker, so this subtest would "+
					"otherwise green for entirely the wrong reason. Fix the fixture source, not the assertion",
					tc.name, err)
			}
			if file == nil {
				t.Fatalf("parsing the %q fixture returned a nil *ast.File with no error", tc.name)
			}

			violations, dispatch, forward := proberWiringViolations(fset, file)

			if tc.wantViolation && len(violations) == 0 {
				t.Errorf("the %q fixture produced NO violations.\nWhy this shape must be caught: %s\n"+
					"counts: dispatch=%d forward=%d", tc.name, tc.why, dispatch, forward)
			}
			if !tc.wantViolation && len(violations) != 0 {
				t.Errorf("the %q fixture is CORRECT-BY-CONSTRUCTION and must yield zero violations, got %d:\n  %s\n"+
					"Why: %s", tc.name, len(violations), strings.Join(violations, "\n  "), tc.why)
			}

			for _, mention := range tc.wantMentions {
				if !proberWiringAnyMentions(violations, mention) {
					t.Errorf("no violation from the %q fixture mentions %q, so the message does not name the "+
						"offending site or the rule that fired.\nWhy this shape must be caught: %s\nviolations:\n  %s",
						tc.name, mention, tc.why, strings.Join(violations, "\n  "))
				}
			}

			// Every violation must carry a position. A message that cannot be traced
			// back to a line is a message a reader will waive.
			for _, violation := range violations {
				if !strings.Contains(violation, "sandbox_linux.go:") {
					t.Errorf("a violation from the %q fixture names no source position: %q.\n"+
						"Every violation must carry fset.Position so the site is locatable", tc.name, violation)
				}
			}

			if dispatch != tc.wantDispatch {
				t.Errorf("the %q fixture yielded dispatch=%d, want %d — the per-bucket counts are what catch a "+
					"seam that was deleted, renamed, or turned into a shape the identifier match cannot see",
					tc.name, dispatch, tc.wantDispatch)
			}
			if forward != tc.wantForward {
				t.Errorf("the %q fixture yielded forward=%d, want %d — same reason as dispatch above",
					tc.name, forward, tc.wantForward)
			}
		})
	}
}

// proberWiringAnyMentions reports whether at least one violation contains needle,
// compared case-insensitively so a message may capitalise a rule word for emphasis
// without breaking the table.
func proberWiringAnyMentions(violations []string, needle string) bool {
	lowered := strings.ToLower(needle)
	for _, violation := range violations {
		if strings.Contains(strings.ToLower(violation), lowered) {
			return true
		}
	}
	return false
}
