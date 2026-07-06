package engine

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// contracts_kind_signature_test.go (ISSUE-036): the contracts pack signature
// compiler must be DECLARATION-KIND-AWARE. Today compile-signature.sh is
// func-ONLY: fed a type/const/var/interface/method-with-receiver signature it
// emits a garbage ast-grep pattern that can never match real Go, so five of the
// schema's six contract kinds read as vacuous green. These tests shell the
// DURABLE compiler source (packs/contracts/scripts/compile-signature.sh — the
// real fix target, source_type: local) over each declared signature and run REAL
// ast-grep against the per-kind fixtures. Substantive: assert match COUNTS
// (>0 present vs ==0 absent) AND the emitted pattern shape — never canned
// plumbing.

// durablePackRel is the TRACKED, DURABLE compiler source (source_type: local in
// backstop.lock) — NOT the disposable installed copy under .backstop/packs/ and
// NOT the test-harness copy under pkg/gate/testdata/. This is the fix target, so
// the test proves the real durable script is kind-aware.
const durablePackRel = "packs/contracts"

// compileDurableSignature runs the DURABLE compile-signature.sh over a declared
// signature and returns the emitted ast-grep pattern.
func compileDurableSignature(t *testing.T, root, sig string) string {
	t.Helper()
	script := filepath.Join(root, durablePackRel, "scripts", "compile-signature.sh")
	if _, err := os.Stat(script); err != nil {
		t.Fatalf("durable compile-signature.sh must exist (ISSUE-036 fix target): %v", err)
	}
	cmd := exec.Command("/bin/sh", script, sig)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("compile-signature.sh failed for %q: %v (stderr: %s)", sig, err, stderr.String())
	}
	return strings.TrimSpace(stdout.String())
}

// kindFixture resolves a per-kind fixture file that lives WITH the pack.
func kindFixture(root, name string) string {
	return filepath.Join(root, durablePackRel, "testdata", "fixtures", name)
}

// TestContractCompiler_FuncSignatureUnchanged (CLM-001): a func signature still
// compiles to the existing `func $NAME($$$PARAMS) …` structural pattern and
// matches the present func fixture / does NOT match the mismatch fixture. NO
// REGRESSION of the func path.
func TestContractCompiler_FuncSignatureUnchanged(t *testing.T) {
	root := repoRoot(t)
	pattern := compileDurableSignature(t, root, "func RouteFile(path string, mode int) (string, error)")
	if !strings.HasPrefix(pattern, "func RouteFile(") {
		t.Errorf("func signature must still compile to a func pattern, got %q", pattern)
	}
	if !strings.Contains(pattern, "$$$PARAMS") {
		t.Errorf("func pattern must metavar the parameter list, got %q", pattern)
	}
	if got := astGrepMatchCount(t, root, pattern, kindFixture(root, "sig-present.go")); got == 0 {
		t.Fatalf("func pattern must MATCH the present func fixture, got 0 with pattern %q", pattern)
	}
	if got := astGrepMatchCount(t, root, pattern, kindFixture(root, "sig-mismatch.go")); got != 0 {
		t.Fatalf("func pattern must NOT match the mismatch fixture, got %d with pattern %q", got, pattern)
	}
}

// TestContractCompiler_KindInferredFromSignatureText (CLM-007): the runtime
// passes ONLY the signature string (no kind field), so the compiler must infer
// the declaration kind from the leading token. Compiling a type signature must
// NOT produce a func-wrapped pattern.
func TestContractCompiler_KindInferredFromSignatureText(t *testing.T) {
	root := repoRoot(t)
	pattern := compileDurableSignature(t, root, "type CheckType int")
	if strings.HasPrefix(pattern, "func ") {
		t.Fatalf("a type signature must NOT compile to a func pattern (kind inferred from text), got %q", pattern)
	}
	if !strings.HasPrefix(pattern, "type ") {
		t.Fatalf("a type signature must compile to a type pattern, got %q", pattern)
	}
}

// TestContractCompiler_TypeSignatureMatchesTypeDecl (CLM-002): "type CheckType
// int" compiles to a `type $NAME …` pattern that MATCHES the real type decl in
// the present fixture.
func TestContractCompiler_TypeSignatureMatchesTypeDecl(t *testing.T) {
	root := repoRoot(t)
	pattern := compileDurableSignature(t, root, "type CheckType int")
	if !strings.HasPrefix(pattern, "type CheckType") {
		t.Errorf("type pattern must preserve the type name, got %q", pattern)
	}
	if got := astGrepMatchCount(t, root, pattern, kindFixture(root, "sig-kinds-present.go")); got == 0 {
		t.Fatalf("type pattern must MATCH the present type decl, got 0 with pattern %q", pattern)
	}
}

// TestContractCompiler_TypeSignatureNoMatchAbsent (CLM-002): the same type
// pattern finds ZERO matches in the mismatch fixture (which omits CheckType).
func TestContractCompiler_TypeSignatureNoMatchAbsent(t *testing.T) {
	root := repoRoot(t)
	pattern := compileDurableSignature(t, root, "type CheckType int")
	if got := astGrepMatchCount(t, root, pattern, kindFixture(root, "sig-kinds-mismatch.go")); got != 0 {
		t.Fatalf("type pattern must find ZERO matches where CheckType is absent, got %d with pattern %q", got, pattern)
	}
}

// TestContractCompiler_ConstSignatureMatches (CLM-003): a constant signature
// compiles to a `const $NAME = $$$` pattern that MATCHES the present const decl
// and does NOT match the mismatch fixture.
func TestContractCompiler_ConstSignatureMatches(t *testing.T) {
	root := repoRoot(t)
	pattern := compileDurableSignature(t, root, `const CheckTypeFindings = "findings"`)
	if !strings.HasPrefix(pattern, "const CheckTypeFindings") {
		t.Errorf("const pattern must preserve the const name, got %q", pattern)
	}
	if !strings.Contains(pattern, "=") {
		t.Errorf("const pattern must preserve the `=` (a bare const $NAME $$$ is an ast-grep ERROR node), got %q", pattern)
	}
	if got := astGrepMatchCount(t, root, pattern, kindFixture(root, "sig-kinds-present.go")); got == 0 {
		t.Fatalf("const pattern must MATCH the present const decl, got 0 with pattern %q", pattern)
	}
	if got := astGrepMatchCount(t, root, pattern, kindFixture(root, "sig-kinds-mismatch.go")); got != 0 {
		t.Fatalf("const pattern must find ZERO matches where CheckTypeFindings is absent, got %d with pattern %q", got, pattern)
	}
}

// TestContractCompiler_VarSignatureMatches (CLM-004): a variable signature
// compiles to a `var $NAME …` pattern that MATCHES the present var decl and does
// NOT match the mismatch fixture.
func TestContractCompiler_VarSignatureMatches(t *testing.T) {
	root := repoRoot(t)
	pattern := compileDurableSignature(t, root, "var GlobalRegistry = map[string]int{}")
	if !strings.HasPrefix(pattern, "var GlobalRegistry") {
		t.Errorf("var pattern must preserve the var name, got %q", pattern)
	}
	if got := astGrepMatchCount(t, root, pattern, kindFixture(root, "sig-kinds-present.go")); got == 0 {
		t.Fatalf("var pattern must MATCH the present var decl, got 0 with pattern %q", pattern)
	}
	if got := astGrepMatchCount(t, root, pattern, kindFixture(root, "sig-kinds-mismatch.go")); got != 0 {
		t.Fatalf("var pattern must find ZERO matches where GlobalRegistry is absent, got %d with pattern %q", got, pattern)
	}
}

// TestContractCompiler_MethodPreservesReceiverType (CLM-005): "func (ct
// CheckType) String() string" compiles to a pattern that PRESERVES the receiver
// TYPE (metavar only the receiver name + params, keep the return clause) and
// MATCHES the real method in the present fixture.
func TestContractCompiler_MethodPreservesReceiverType(t *testing.T) {
	root := repoRoot(t)
	pattern := compileDurableSignature(t, root, "func (ct CheckType) String() string")
	if !strings.Contains(pattern, "CheckType") {
		t.Fatalf("method pattern must PRESERVE the receiver type (current bug drops it), got %q", pattern)
	}
	if !strings.Contains(pattern, "String(") {
		t.Errorf("method pattern must preserve the method name, got %q", pattern)
	}
	if got := astGrepMatchCount(t, root, pattern, kindFixture(root, "sig-kinds-present.go")); got == 0 {
		t.Fatalf("method pattern must MATCH the real (ct CheckType) String() method, got 0 with pattern %q", pattern)
	}
}

// TestContractCompiler_MethodRejectsFreeFunctionSameName (CLM-005): the SAME
// compiled method pattern finds ZERO matches against the mismatch fixture, which
// holds a FREE func String() AND a String() method on a DIFFERENT receiver
// (Other) — proving the receiver type is no longer dropped.
func TestContractCompiler_MethodRejectsFreeFunctionSameName(t *testing.T) {
	root := repoRoot(t)
	pattern := compileDurableSignature(t, root, "func (ct CheckType) String() string")
	if got := astGrepMatchCount(t, root, pattern, kindFixture(root, "sig-kinds-mismatch.go")); got != 0 {
		t.Fatalf("method pattern must REJECT a same-named free func and a method on a different receiver, got %d with pattern %q", got, pattern)
	}
}

// TestContractCompiler_InterfaceSignatureMatches (CLM-006): an interface
// signature compiles to a `type $NAME interface { $$$ }` pattern that MATCHES the
// interface decl in the present fixture and does NOT match the mismatch fixture.
func TestContractCompiler_InterfaceSignatureMatches(t *testing.T) {
	root := repoRoot(t)
	pattern := compileDurableSignature(t, root, "type Stringer interface { String() string }")
	if !strings.Contains(pattern, "interface") {
		t.Fatalf("interface pattern must emit an interface shape, got %q", pattern)
	}
	if got := astGrepMatchCount(t, root, pattern, kindFixture(root, "sig-kinds-present.go")); got == 0 {
		t.Fatalf("interface pattern must MATCH the present interface decl, got 0 with pattern %q", pattern)
	}
	if got := astGrepMatchCount(t, root, pattern, kindFixture(root, "sig-kinds-mismatch.go")); got != 0 {
		t.Fatalf("interface pattern must find ZERO matches where Stringer is absent, got %d with pattern %q", got, pattern)
	}
}
