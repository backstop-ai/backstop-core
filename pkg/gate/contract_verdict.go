package gate

import "fmt"

// contract_verdict.go is the language-agnostic, gate-side half of the SPEC-038
// contracts pack split (BUNDLE-009 Seed 4). The PACK does the language-specific
// work — the contract->ast-grep-pattern compiler (signature presence) and the
// grep forbidden-pattern probe (symbol absence) — and emits SARIF. This file
// holds the ONLY contract logic that stays gate-side:
//   - the match-verdict (ast-grep match = SATISFIED, no-match = VIOLATION),
//   - the absence-polarity inversion (grep present-match = absence VIOLATION),
//   - the file-scanned guard (a declared scope with no scan record is a LOUD
//     config error, preserving ISSUE-013's loud-on-empty — REPLACING the old
//     non-.go/missing config-error).
// It imports NO language package: the verdict is a pure function of the engine
// result's {Absent, Matched, Scanned} fields. A scanned non-.go scope gets a
// normal verdict — extension/language is no longer a config-error axis (CLM-034).

// SarifLocation is the minimal physicalLocation carrier the gate keeps from a pack
// engine probe — enough to name the file and line in a violation message without
// re-parsing source.
type SarifLocation struct {
	File string
	Line int
}

// ContractEngineResult is the gate-side, language-agnostic carrier of ONE pack
// engine probe result for ONE contract entry. Matched is whether the ast-grep
// signature query / grep absence query matched; Scanned is the file-scanned guard
// signal (did the engine actually scan the declared scope — REQ-004/CLM-013);
// Locations carries the match locations for the violation message. The gate
// verdicts PURELY off these fields; it never re-parses source.
type ContractEngineResult struct {
	Entry     ContractEntry
	Matched   bool
	Scanned   bool
	Locations []SarifLocation
}

// VerifyContractVerdict is the pure, language-agnostic verdict function (REQ-002/
// REQ-003/REQ-004). It returns (violation, true) for a genuine violation/config
// error and (Violation{}, false) for a satisfied/passing contract. The decision
// table:
//
//	present-contract (Absent=false):
//	  Matched           -> satisfied (no violation)
//	  !Matched          -> blocking contract VIOLATION (signature absent/mismatched)
//	absence-contract (Absent=true):
//	  !Scanned          -> LOUD config error (missing/unscanned scope; never silent)
//	  Matched           -> blocking absence VIOLATION (forbidden symbol present)
//	  !Matched && Scanned -> PASS (genuinely absent from a confirmed-scanned scope)
//
// The unscanned-scope branch is the language-agnostic REPLACEMENT for the old
// non-.go/missing config-error: it keys ONLY on scanned-vs-unscanned, so a
// SCANNED non-Go scope gets a normal verdict (CLM-034). No language imports
// (CLM-014).
func VerifyContractVerdict(r ContractEngineResult) (Violation, bool) {
	// Canonical repo-relative contract File, computed ONCE (ISSUE-046).
	// VerifyContractVerdict is a pure function with no ProjectRoot in scope; a
	// spec-declared contract path is authored repo-relative, so NormalizePath("", …)
	// — the idempotent Clean+ToSlash+strip-"./" subset of the SINGLE helper —
	// canonicalizes it. It feeds BOTH the Violation.File AND the report Message: a
	// contract violation carries no RegionHash, so baseline identity falls back to
	// the Message, and a non-canonical path leaking into the Message would make the
	// identity scope-unstable even with a canonical File field. An absolute spec
	// path is not expected here; the Phase-1 identity chokepoint is the backstop.
	file := NormalizePath("", r.Entry.File)
	if r.Entry.Absent {
		// Absence contract. The file-scanned guard fires FIRST: a scope that was
		// not scanned (missing file / no scan record) is a loud config error, not a
		// silent pass — empty-because-not-scanned is not absent (REQ-004/CLM-012/013).
		if !r.Scanned {
			return Violation{
				Rule:     StepContractSignature,
				File:     file,
				Message:  fmt.Sprintf("absence assertion for symbol %s: declared scope %s was not scanned (missing or unscanned) — cannot confirm absence, refusing to pass silently", r.Entry.Name, contractScope(r.Entry)),
				Severity: "error",
			}, true
		}
		if r.Matched {
			return Violation{
				Rule:     StepContractSignature,
				File:     NormalizePath("", locationFile(r, r.Entry.File)),
				Message:  fmt.Sprintf("symbol %s expected absent but present%s (forbidden symbol regression)", r.Entry.Name, locationSuffix(r)),
				Severity: "error",
			}, true
		}
		// Scanned and not matched: genuinely absent — PASS.
		return Violation{}, false
	}

	// Present contract: an ast-grep match means the declared signature is present
	// (SATISFIED); no match is a blocking violation.
	if r.Matched {
		return Violation{}, false
	}
	return Violation{
		Rule:     StepContractSignature,
		File:     file,
		Message:  fmt.Sprintf("symbol %s signature not found or mismatched in %s: expected %q", r.Entry.Name, file, r.Entry.Signature),
		Severity: "error",
	}, true
}

// contractScope returns the entry's declared absence scope, falling back to File
// when Scope is empty (a file-scoped absence may declare only File).
func contractScope(e ContractEntry) string {
	if e.Scope != "" {
		return e.Scope
	}
	return e.File
}

// locationFile returns the first match location's file, falling back to fallback.
func locationFile(r ContractEngineResult, fallback string) string {
	if len(r.Locations) > 0 && r.Locations[0].File != "" {
		return r.Locations[0].File
	}
	return fallback
}

// locationSuffix renders " in <file>:<line>" for the first match location, or "".
func locationSuffix(r ContractEngineResult) string {
	if len(r.Locations) == 0 {
		return ""
	}
	loc := r.Locations[0]
	if loc.Line > 0 {
		return fmt.Sprintf(" in %s:%d", loc.File, loc.Line)
	}
	return fmt.Sprintf(" in %s", loc.File)
}
