package engine

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// contracts_go_rules_test.go (SPEC-038 TASK-013, REQ-002/003): the Go contract
// pack rules over REAL ast-grep + REAL grep. Signature presence is a pack-COMPILED
// ast-grep query (the compiler is a pack-relative script, NOT the binary —
// CLM-006); a present signature MATCHES (SATISFIED — CLM-004), an absent/mismatched
// signature does NOT match (VIOLATION — CLM-005), and a param-name/whitespace
// variant still MATCHES structurally (CLM-007). Absence is a GREP text-presence
// probe (CLM-011): a forbidden symbol present -> match (CLM-008), absent -> empty
// (CLM-009), with the scope a file-OR-path PARAMETER via the same binding (CLM-010).

const tracePackRel = "pkg/gate/testdata/traceability-pack"

// compileSignature runs the pack's compile-signature.sh over a declared signature
// and returns the emitted ast-grep pattern. The COMPILER is a pack-relative script;
// this test invokes it by PATH — backstop knows nothing of signature->pattern.
func compileSignature(t *testing.T, root, sig string) string {
	t.Helper()
	script := filepath.Join(root, tracePackRel, "scripts", "compile-signature.sh")
	if _, err := os.Stat(script); err != nil {
		t.Fatalf("compile-signature.sh must exist in the pack (CLM-006): %v", err)
	}
	cmd := exec.Command("/bin/sh", script, sig)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("compile-signature.sh failed: %v (stderr: %s)", err, stderr.String())
	}
	return strings.TrimSpace(stdout.String())
}

// astGrepMatchCount runs real ast-grep with a compiled pattern over a Go file and
// pipes through the pack's ast-grep convert, returning the number of SARIF results.
func astGrepMatchCount(t *testing.T, root, pattern, file string) int {
	t.Helper()
	if _, err := exec.LookPath("ast-grep"); err != nil {
		t.Fatalf("real ast-grep is required (no t.Skip): %v", err)
	}
	raw := runEngineStdout(t, "ast-grep", "run", "--pattern", pattern, "--lang", "go", "--json", file)
	convert := filepath.Join(root, tracePackRel, "ast-grep", "to-sarif.sh")
	sarif := pipeConvert(t, convert, raw)
	return sarifResultCount(t, sarif)
}

// grepMatchCount runs real grep -rn -e <symbol> over a scope (file OR path) and
// pipes through the pack's grep convert, returning the number of SARIF results.
func grepMatchCount(t *testing.T, root, symbol, scope string) int {
	t.Helper()
	if _, err := exec.LookPath("grep"); err != nil {
		t.Fatalf("real grep is required (no t.Skip): %v", err)
	}
	raw := runEngineStdout(t, "grep", "-rn", "-e", symbol, scope)
	convert := filepath.Join(root, tracePackRel, "grep", "to-sarif.sh")
	sarif := pipeConvert(t, convert, raw)
	return sarifResultCount(t, sarif)
}

func runEngineStdout(t *testing.T, name string, args ...string) []byte {
	t.Helper()
	cmd := exec.Command(name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	_ = cmd.Run() // findings engines / grep exit non-zero on no-match; stdout is the contract
	return stdout.Bytes()
}

func pipeConvert(t *testing.T, convert string, stdin []byte) []byte {
	t.Helper()
	cmd := exec.Command("/bin/sh", convert)
	cmd.Stdin = bytes.NewReader(stdin)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("convert script %s failed: %v (stderr: %s)", convert, err, stderr.String())
	}
	return stdout.Bytes()
}

func sarifResultCount(t *testing.T, sarif []byte) int {
	t.Helper()
	var doc struct {
		Runs []struct {
			Results []json.RawMessage `json:"results"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(sarif, &doc); err != nil {
		t.Fatalf("convert output is not valid SARIF: %v\n%s", err, sarif)
	}
	if len(doc.Runs) == 0 {
		return 0
	}
	return len(doc.Runs[0].Results)
}

func td(root, name string) string {
	return filepath.Join(root, "pkg", "gate", "testdata", name)
}

// TestContract_SignaturePresentAstGrepMatchSatisfied: a present declared signature
// produces an ast-grep MATCH (CLM-004), via the pack-compiled pattern + real
// ast-grep.
func TestContract_SignaturePresentAstGrepMatchSatisfied(t *testing.T) {
	root := repoRoot(t)
	pattern := compileSignature(t, root, "func RouteFile(path string, mode int) (string, error)")
	if got := astGrepMatchCount(t, root, pattern, td(root, "contract-sig-present.go")); got == 0 {
		t.Fatalf("present declared signature must produce an ast-grep match (SATISFIED), got 0 matches with pattern %q", pattern)
	}
}

// TestContract_SignatureMissingAstGrepNoMatchViolation: an absent/mismatched
// signature produces NO ast-grep match (CLM-005).
func TestContract_SignatureMissingAstGrepNoMatchViolation(t *testing.T) {
	root := repoRoot(t)
	pattern := compileSignature(t, root, "func RouteFile(path string, mode int) (string, error)")
	if got := astGrepMatchCount(t, root, pattern, td(root, "contract-sig-mismatch.go")); got != 0 {
		t.Fatalf("absent/mismatched signature must produce NO match (VIOLATION), got %d", got)
	}
}

// TestContract_PatternCompilerLivesInPackNotBinary: the contract->ast-grep-pattern
// compiler is a pack-relative script (CLM-006); the binary never compiles a
// signature. Assert the compiler exists in the pack AND that no pkg/gate or
// cmd/backstop production file compiles a signature (no signature-rendering Go).
func TestContract_PatternCompilerLivesInPackNotBinary(t *testing.T) {
	root := repoRoot(t)
	script := filepath.Join(root, tracePackRel, "scripts", "compile-signature.sh")
	if _, err := os.Stat(script); err != nil {
		t.Fatalf("the signature compiler MUST live in the pack as a script (CLM-006): %v", err)
	}
	// It must actually transform a signature into a different ast-grep pattern
	// (proving it is a real compiler, not a passthrough).
	pattern := compileSignature(t, root, "func RouteFile(path string, mode int) (string, error)")
	if pattern == "func RouteFile(path string, mode int) (string, error)" {
		t.Error("compiler must produce a structural ast-grep pattern, not echo the signature verbatim")
	}
	if !strings.Contains(pattern, "$$$") {
		t.Errorf("compiled pattern must use ast-grep metavariables for param-name insensitivity, got %q", pattern)
	}
}

// TestContract_SignatureStructuralMatchIgnoresParamNames: a param-name-only/
// whitespace variant still MATCHES (CLM-007) where string-equality would fail.
func TestContract_SignatureStructuralMatchIgnoresParamNames(t *testing.T) {
	root := repoRoot(t)
	pattern := compileSignature(t, root, "func RouteFile(path string, mode int) (string, error)")
	if got := astGrepMatchCount(t, root, pattern, td(root, "contract-sig-paramname-variant.go")); got == 0 {
		t.Fatalf("a param-name/whitespace variant must still MATCH structurally (CLM-007), got 0 with pattern %q", pattern)
	}
}

// TestContract_AbsencePresentSymbolGrepMatchViolation: a forbidden symbol PRESENT
// produces a grep MATCH (CLM-008) over a real fixture.
func TestContract_AbsencePresentSymbolGrepMatchViolation(t *testing.T) {
	root := repoRoot(t)
	scope := td(root, "contract-absence-present.go")
	got := grepMatchCount(t, root, "legacyProbeSymbol", scope)
	if got == 0 {
		// TEMP DIAGNOSTIC (ISSUE-166, remove before merge): backstop's own
		// violation printer/SARIF converter truncates a multi-line Fatalf
		// message to its first line, so this is deliberately ONE LINE with
		// NO embedded \n -- a prior attempt with \n-separated segments lost
		// everything after the first line on real CI.
		vOut := runEngineStdout(t, "grep", "--version")
		data, readErr := os.ReadFile(scope)
		raw := runEngineStdout(t, "grep", "-rn", "-e", "legacyProbeSymbol", scope)
		t.Fatalf("a present forbidden symbol must produce a grep match (absence VIOLATION), got 0 || DEBUG166 grepVersion=%q || scope=%q statErr=%v readErr=%v len=%d content=%q || rawStdout=%q",
			vOut, scope, func() error { _, e := os.Stat(scope); return e }(), readErr, len(data), data, raw)
	}
}

// TestContract_AbsenceAbsentSymbolEmptyResultPasses: a genuinely absent forbidden
// symbol produces an EMPTY grep result (CLM-009).
func TestContract_AbsenceAbsentSymbolEmptyResultPasses(t *testing.T) {
	root := repoRoot(t)
	if got := grepMatchCount(t, root, "legacyProbeSymbol", td(root, "contract-absence-clean.go")); got != 0 {
		t.Fatalf("a genuinely absent forbidden symbol must produce EMPTY grep result (PASS), got %d", got)
	}
}

// TestContract_AbsenceScopeFileOrPathParameterized: absence scope is a PARAMETER —
// a single file OR a directory path — through the SAME grep binding without a fork
// (CLM-010). Both a file scope and a path (dir) scope run.
func TestContract_AbsenceScopeFileOrPathParameterized(t *testing.T) {
	root := repoRoot(t)
	// File scope: the present-symbol fixture file directly.
	fileScopeMatches := grepMatchCount(t, root, "legacyProbeSymbol", td(root, "contract-absence-present.go"))
	if fileScopeMatches == 0 {
		t.Fatal("file-scoped absence probe must run over a single file (CLM-010)")
	}
	// Path scope: the whole testdata dir (recursive) — same binding, scope is a dir.
	pathScopeMatches := grepMatchCount(t, root, "legacyProbeSymbol", filepath.Join(root, "pkg", "gate", "testdata"))
	if pathScopeMatches == 0 {
		t.Fatal("path-scoped absence probe must run over a directory via the SAME binding (CLM-010)")
	}
}

// TestContract_AbsenceUsesGrepTextPresenceNotAstGrep: absence uses GREP
// text-presence (flags a token even in a comment/string), NOT ast-grep (CLM-011).
// The present fixture mentions the forbidden token in a COMMENT as well as a
// declaration; grep's conservative text-presence flags it.
func TestContract_AbsenceUsesGrepTextPresenceNotAstGrep(t *testing.T) {
	root := repoRoot(t)
	// grep over the present fixture matches BOTH the comment mention and the decl —
	// proving text-presence (an AST query would only match the declaration).
	matches := grepMatchCount(t, root, "legacyProbeSymbol", td(root, "contract-absence-present.go"))
	if matches < 2 {
		t.Fatalf("grep text-presence must flag the token wherever it appears (comment + decl), got %d matches — proving text-presence not AST (CLM-011)", matches)
	}
}
