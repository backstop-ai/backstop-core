package check

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// TestCheck_NeverStartedIsTheSingleAuthority pins CLM-006: after ISSUE-140 there is
// exactly ONE implementation of "did the process start", and both consumers DELEGATE
// to it. It lives beside the authority because the authority is what it protects.
//
// It reads EXACTLY TWO consumer sources — the two files that each carried a
// never-started check. CLM-006 names FILES, not packages, deliberately: a
// package-wide sweep would false-positive on the legitimate *exec.ExitError handling
// scattered through both packages (a DIFFERENT type answering a DIFFERENT question —
// did it exit non-zero) and would decay into an exclusion list. Two files genuinely
// read beats a sweep talked out of its own hits.
//
// Sources are parsed with go/ast rather than grepped, so the call-site-local comment
// in runCoverageEngine that legitimately MENTIONS *exec.Error (explaining why that
// shape is unreachable on the os.Stat-guarded producer branch) cannot false-positive:
// parsing without ParseComments drops comments entirely, leaving only real code.
func TestCheck_NeverStartedIsTheSingleAuthority(t *testing.T) {
	consumers := []struct {
		label string
		path  string
	}{
		{"cmd/backstop/pack_gate.go", "../../cmd/backstop/pack_gate.go"},
		{"pkg/packval/executor.go", "../packval/executor.go"},
	}

	for _, c := range consumers {
		src, err := os.ReadFile(c.path)
		if err != nil {
			// LOUD on a moved/renamed file. A tripwire that vacuously passes on
			// unreadable content is a decoration, not a tripwire.
			t.Fatalf("reading consumer source %s (%s): %v — if this file moved, RE-POINT this tripwire; do not delete it",
				c.label, c.path, err)
		}
		if len(bytes.TrimSpace(src)) == 0 {
			t.Fatalf("consumer source %s is empty; the tripwire has nothing to read", c.label)
		}

		fset := token.NewFileSet()
		file, parseErr := parser.ParseFile(fset, c.path, src, 0)
		if parseErr != nil {
			t.Fatalf("parsing consumer source %s: %v", c.label, parseErr)
		}

		// 1. Neither file may declare a PRIVATE never-started predicate. A predicate
		//    is a func whose name mentions never-started AND returns a single bool —
		//    which spares neverStartedError, the gate-specific *check.ConfigError
		//    refusal RENDERER that legitimately stays in cmd/backstop.
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name == nil {
				continue
			}
			if !strings.Contains(strings.ToLower(fn.Name.Name), "neverstarted") {
				continue
			}
			if !returnsSingleBool(fn) {
				continue
			}
			t.Errorf("%s declares a private never-started predicate %q; check.NeverStarted is the single authority — DELETE the local copy and call it instead (a wrapper is still a place a future edit can diverge, which is how ISSUE-140 happened)",
				c.label, fn.Name.Name)
		}

		// 2. Neither file may classify a run error on *exec.Error alone — the exact
		//    narrow shape ISSUE-140 reports, which misses the path-ful fork/exec
		//    *fs.PathError. *exec.ExitError is a different type answering a different
		//    question and MUST survive; this check cannot reach it.
		ast.Inspect(file, func(n ast.Node) bool {
			star, ok := n.(*ast.StarExpr)
			if !ok {
				return true
			}
			sel, ok := star.X.(*ast.SelectorExpr)
			if !ok || sel.Sel == nil || sel.Sel.Name != "Error" {
				return true
			}
			pkgIdent, ok := sel.X.(*ast.Ident)
			if !ok || pkgIdent.Name != "exec" {
				return true
			}
			t.Errorf("%s references *exec.Error in code; that is the NARROW never-started classification ISSUE-140 reports — it misses the path-ful *fs.PathError{Op: \"fork/exec\"} shape. Call check.NeverStarted instead",
				c.label)
			return true
		})

		// 3. Absence alone is not enough — it would also pass if a future edit deleted
		//    the check entirely, which is the vacuous version of this test. Both files
		//    must DELEGATE.
		if !bytes.Contains(src, []byte("check.NeverStarted(")) {
			t.Errorf("%s does not call check.NeverStarted; the tripwire proves DELEGATION, not merely the absence of a local copy",
				c.label)
		}
	}
}

// returnsSingleBool reports whether fn's result list is exactly one unnamed-or-named
// bool — the shape of a predicate, as opposed to an error-returning renderer.
func returnsSingleBool(fn *ast.FuncDecl) bool {
	if fn.Type == nil || fn.Type.Results == nil || len(fn.Type.Results.List) != 1 {
		return false
	}
	ident, ok := fn.Type.Results.List[0].Type.(*ast.Ident)
	return ok && ident.Name == "bool"
}
