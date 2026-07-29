package gate

import "context"

// step_contract.go — POST-SPEC-038. The baked Go contract analyzer (go/parser
// symbol extraction + Go-source signature rendering + whitespace-normalized
// string-equality compare, plus ISSUE-013's gate-side parser absence probe) is
// DELETED (REQ-001). Contract verification is now done by PACKS on the structural
// engines (ast-grep signature presence + grep symbol absence); this file keeps
// ONLY the language-agnostic gate-side surface:
//   - ContractEntry: the pure declared-contract data record (no probe behavior),
//   - StepContractSignatureScopedFunc: the SOLE retained contract entrypoint,
//     which consumes pack-produced []ContractEngineResult and applies the
//     language-agnostic verdict (VerifyContractVerdict, contract_verdict.go).
// It imports NO go/parser, go/ast, or go/printer. The non-scoped
// StepContractSignatureFunc wrapper is DELETED (REQ-011) — not a shim.

// ContractEntry represents a declared symbol in a spec's contracts section. It is a
// PURE DATA RECORD fed to the pack engines via pattern-arg; it no longer drives a
// go/parser probe. File/Name/Kind/Signature/Absent keep their declared meaning;
// Scope is the absence file-OR-path parameter (REQ-012).
type ContractEntry struct {
	File      string // path to the file containing the symbol
	Name      string // symbol name
	Kind      string // "function", "type", "variable", "interface"
	Signature string // declared signature (passed to the pack compiler unmodified)
	// Scope is the absence file-OR-path parameter (REQ-012/CLM-010/CLM-040): the
	// declared scope the grep absence probe scans for the forbidden symbol. It may
	// be a single file or a directory path. Populated by ExtractContractEntries
	// from the declared provides entry; empty for a present-signature contract (the
	// absence verdict falls back to File when Scope is empty).
	Scope string
	// Absent inverts the assertion: when true, the entry asserts the named symbol
	// must NOT exist in the declared Scope. It passes iff the symbol is genuinely
	// absent from a confirmed-SCANNED scope and fails if the symbol reappears (a
	// deletion regression guard). An absent entry ignores Signature. An absence
	// over an UNSCANNED/missing scope is a loud config error (not a silent pass) —
	// the gate-side file-scanned guard, not an extension check.
	Absent bool
}

// StepContractSignatureScopedFunc is the SOLE retained contract entrypoint
// (REQ-011/CLM-038). It consumes the pack-produced []ContractEngineResult (one per
// contract entry, from real ast-grep signature presence + real grep symbol absence)
// and applies the pure language-agnostic verdict (VerifyContractVerdict): ast-grep
// match = SATISFIED / no-match = VIOLATION (REQ-002); grep present-match = absence
// VIOLATION (REQ-003); unscanned scope = loud config error (REQ-004). It imports no
// go/parser/go/ast/go/printer — all language work lives in the pack. The scope
// parameter filters which results are evaluated by the entry's declared File.
func StepContractSignatureScopedFunc(results []ContractEngineResult, scope *GateScope) StepFunc {
	return func(_ context.Context) StepResult {
		var violations []Violation
		for _, r := range results {
			if scope != nil && scope.Mode != GateScopeModeAll && r.Entry.File != "" && !scope.Contains(r.Entry.File) {
				continue
			}
			if v, raised := VerifyContractVerdict(r); raised {
				violations = append(violations, v)
			}
		}

		status := StepVerdict(violations)
		if violations == nil {
			violations = []Violation{}
		}
		return StepResult{
			StepName:   StepContractSignature,
			Status:     status,
			Violations: violations,
		}
	}
}
