package gate

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// contract_verdict_test.go (SPEC-038 TASK-008, REQ-002/003/004): the pure,
// language-agnostic verdict function over ContractEngineResult. The gate keeps
// ONLY the match-verdict, the absence-polarity inversion, and the file-scanned
// guard; all language-specific work lives in the pack. These tests pin every
// branch and prove the guard imports no language package (CLM-014).

// presentEntry / absenceEntry build a ContractEntry of the right polarity.
func presentEntry(file, name string) ContractEntry {
	return ContractEntry{File: file, Name: name, Kind: "function", Signature: "func " + name + "()", Absent: false}
}

func absenceEntry(file, name string) ContractEntry {
	return ContractEntry{File: file, Name: name, Kind: "function", Absent: true, Scope: file}
}

// TestContract_AbsenceMissingFileLoudConfigError: an absence contract whose
// declared file is MISSING (no scan record / not scanned) is a LOUD config error
// (severity error, blocking), never a silent pass (CLM-012). empty-because-not-
// there must not read as absent.
func TestContract_AbsenceMissingFileLoudConfigError(t *testing.T) {
	r := ContractEngineResult{
		Entry:   absenceEntry("/nonexistent/missing.go", "legacyProbeSymbol"),
		Matched: false,
		Scanned: false, // the file was never scanned (missing)
	}
	v, raised := VerifyContractVerdict(r)
	if !raised {
		t.Fatal("missing-file absence must raise a loud config-error violation, got none")
	}
	if v.Severity != "error" {
		t.Errorf("config error must be severity error (exit 2), got %q", v.Severity)
	}
	if !strings.Contains(strings.ToLower(v.Message), "scan") && !strings.Contains(strings.ToLower(v.Message), "missing") {
		t.Errorf("config error must explain the unscanned/missing scope, got: %q", v.Message)
	}
}

// TestContract_AbsenceUnscannedScopeLoudConfigError: an absence contract whose
// declared scope was NOT scanned by the engine (no scan record) is a LOUD config
// error, never a silent pass — empty-because-not-scanned is distinguished from
// absent (CLM-013).
func TestContract_AbsenceUnscannedScopeLoudConfigError(t *testing.T) {
	r := ContractEngineResult{
		Entry:   absenceEntry("/some/real/looking/path.go", "forbidden"),
		Matched: false,
		Scanned: false,
	}
	v, raised := VerifyContractVerdict(r)
	if !raised {
		t.Fatal("unscanned-scope absence must raise a loud config error, got none")
	}
	if v.Severity != "error" {
		t.Errorf("unscanned config error must be severity error, got %q", v.Severity)
	}
}

// TestContract_FileScannedGuardIsLanguageAgnostic: the file-scanned guard asserts
// a scan record exists for the declared scope and does NOT parse, AST-walk, or
// import any language package (CLM-014). We assert behavior across BOTH the
// scanned-and-absent (pass) and the unscanned (loud) cases for a present-contract
// AND an absence-contract, with no language-specific input — the verdict is a
// pure function of {Absent, Matched, Scanned}.
func TestContract_FileScannedGuardIsLanguageAgnostic(t *testing.T) {
	cases := []struct {
		name        string
		r           ContractEngineResult
		wantRaised  bool
		wantSeverit string
	}{
		{
			name:       "present scanned matched -> satisfied",
			r:          ContractEngineResult{Entry: presentEntry("x.go", "F"), Matched: true, Scanned: true},
			wantRaised: false,
		},
		{
			name:        "present scanned no-match -> violation",
			r:           ContractEngineResult{Entry: presentEntry("x.go", "F"), Matched: false, Scanned: true},
			wantRaised:  true,
			wantSeverit: "error",
		},
		{
			name:        "absence scanned matched -> absence violation",
			r:           ContractEngineResult{Entry: absenceEntry("x.go", "F"), Matched: true, Scanned: true},
			wantRaised:  true,
			wantSeverit: "error",
		},
		{
			name:       "absence scanned no-match -> pass",
			r:          ContractEngineResult{Entry: absenceEntry("x.go", "F"), Matched: false, Scanned: true},
			wantRaised: false,
		},
		{
			name:        "absence UNSCANNED -> loud config error",
			r:           ContractEngineResult{Entry: absenceEntry("x.go", "F"), Matched: false, Scanned: false},
			wantRaised:  true,
			wantSeverit: "error",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v, raised := VerifyContractVerdict(tc.r)
			if raised != tc.wantRaised {
				t.Fatalf("raised = %v, want %v (verdict over {Absent:%v Matched:%v Scanned:%v})",
					raised, tc.wantRaised, tc.r.Entry.Absent, tc.r.Matched, tc.r.Scanned)
			}
			if tc.wantRaised && v.Severity != tc.wantSeverit {
				t.Errorf("severity = %q, want %q", v.Severity, tc.wantSeverit)
			}
		})
	}

	// Guard the language-agnostic claim structurally: contract_verdict.go must not
	// import a Go language package. (We parse the source file itself — that is a
	// test using go/parser, NOT the verdict importing it.)
	src := mustReadVerdictSource(t)
	for _, banned := range []string{`"go/parser"`, `"go/ast"`, `"go/printer"`, `"go/token"`} {
		if strings.Contains(src, banned) {
			t.Errorf("contract_verdict.go must not import %s — the guard is language-agnostic (CLM-014)", banned)
		}
	}
}

// TestContract_AbsenceNonGoScannedScopeIsNotConfigError: an absence contract
// targeting a NON-.go file that WAS scanned is NOT a config error — the dissolved
// "non-Go is an error" clause no longer fires. A scanned non-Go scope produces a
// normal absence verdict: present (Matched) -> violation, absent -> pass (CLM-034).
func TestContract_AbsenceNonGoScannedScopeIsNotConfigError(t *testing.T) {
	tsFile := "/some/dir/contract-absence-nongo-scanned.ts"

	// Present forbidden symbol in a scanned .ts scope -> normal absence violation,
	// NOT an extension-based config error.
	present := ContractEngineResult{
		Entry:   ContractEntry{File: tsFile, Name: "legacyTsHelper", Absent: true, Scope: tsFile},
		Matched: true,
		Scanned: true,
	}
	v, raised := VerifyContractVerdict(present)
	if !raised {
		t.Fatal("scanned non-Go scope with a present forbidden symbol must raise a normal absence violation")
	}
	if strings.Contains(strings.ToLower(v.Message), "non-go") || strings.Contains(strings.ToLower(v.Message), "only .go") {
		t.Errorf("a scanned non-Go scope must NOT produce an extension-based error, got: %q", v.Message)
	}

	// Absent forbidden symbol in a scanned .ts scope -> PASS (not a config error).
	absent := ContractEngineResult{
		Entry:   ContractEntry{File: tsFile, Name: "legacyTsHelper", Absent: true, Scope: tsFile},
		Matched: false,
		Scanned: true,
	}
	if _, raised := VerifyContractVerdict(absent); raised {
		t.Error("scanned non-Go scope with the forbidden symbol absent must PASS (CLM-034)")
	}
}

// mustReadVerdictSource locates and reads pkg/gate/contract_verdict.go for the
// import-guard assertion. Uses go/parser ONLY in the test (allowed), proving the
// SOURCE file does not.
func mustReadVerdictSource(t *testing.T) string {
	t.Helper()
	path := filepath.Join(".", "contract_verdict.go")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading contract_verdict.go: %v", err)
	}
	// Sanity: it must parse as Go (a real file, not a stub).
	if _, perr := parser.ParseFile(token.NewFileSet(), path, data, parser.ImportsOnly); perr != nil {
		t.Fatalf("contract_verdict.go does not parse: %v", perr)
	}
	return string(data)
}
