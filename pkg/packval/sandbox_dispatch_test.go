package packval

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"testing"
)

// Platform selection for the sandbox moved from a RUN-time `switch runtime.GOOS`
// to BUILD tags. These two tests are the pair that makes that refactor safe:
// one locks the exported contract so the move cannot become a behavior change,
// the other locks the structure so the switch cannot come back.
//
// Both run on every platform — that is the point. A test that only held on the
// platform it was written on would leave the other one unguarded.

// TestSandboxDispatch_ExportedSignaturesUnchanged is the CLM-017 lock: the
// exported sandbox interface is unchanged, so no caller in
// pkg/packval/executor.go or cmd/backstop/pack_gate.go has to move.
//
// Signatures are asserted through reflection rather than by calling the
// functions, because calling them requires a real sandbox and this assertion is
// about the CONTRACT, which must hold identically on a host with no sandbox at
// all. A refactor that quietly widened a parameter or dropped the ([]byte,
// error) pair would compile at every call site inside this package and break
// the gate's dispatch at run time.
func TestSandboxDispatch_ExportedSignaturesUnchanged(t *testing.T) {
	cases := []struct {
		name string
		fn   interface{}
		want string
	}{
		{
			name: "SandboxedRun",
			fn:   SandboxedRun,
			want: "func(string, []string, string) ([]uint8, error)",
		},
		{
			name: "SandboxedRunStdout",
			fn:   SandboxedRunStdout,
			want: "func(string, []string, string, []uint8) ([]uint8, error)",
		},
	}

	for _, tc := range cases {
		got := reflect.TypeOf(tc.fn).String()
		if got != tc.want {
			t.Errorf("%s has signature %s, want %s — the dispatch refactor must not change the exported "+
				"contract; every caller outside this package depends on it", tc.name, got, tc.want)
		}
	}
}

// TestSandboxDispatch_NoRuntimeGOOSSwitchRemains asserts structurally that no
// run-time platform branch survives in sandbox.go.
//
// Structural, not textual: the whole point of the refactor is that platform
// selection happens at BUILD time, so each platform compiles and measures only
// code it can execute. A `switch runtime.GOOS` reintroduced later "for clarity"
// would restore the dead arms — the darwin bodies compiled into the Linux binary
// and vice versa — which is exactly the unreachable code this phase deletes.
//
// The two presence assertions are not decoration. Without them this test passes
// against a sandbox.go that was renamed, emptied, or never parsed at all, which
// is the classic way an absence test becomes vacuous.
func TestSandboxDispatch_NoRuntimeGOOSSwitchRemains(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sandbox.go", nil, 0)
	if err != nil {
		t.Fatalf("parsing sandbox.go: %v", err)
	}

	found := map[string]bool{}
	var runtimeGOOSRefs []string

	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			found[node.Name.Name] = true
		case *ast.SelectorExpr:
			ident, ok := node.X.(*ast.Ident)
			if ok && ident.Name == "runtime" && node.Sel.Name == "GOOS" {
				runtimeGOOSRefs = append(runtimeGOOSRefs, fset.Position(node.Pos()).String())
			}
		}
		return true
	})

	for _, want := range []string{"SandboxedRun", "SandboxedRunStdout"} {
		if !found[want] {
			t.Fatalf("sandbox.go does not declare %s; this test asserts an ABSENCE, so it is only meaningful "+
				"when the file it parsed is the one that carries the dispatch", want)
		}
	}

	if len(runtimeGOOSRefs) != 0 {
		t.Errorf("sandbox.go still references runtime.GOOS at %v — platform selection belongs in build-tagged "+
			"files (sandbox_darwin.go, sandbox_linux.go, and the unsupported-platform file), so that neither "+
			"platform compiles the other's unreachable arms", runtimeGOOSRefs)
	}
}
