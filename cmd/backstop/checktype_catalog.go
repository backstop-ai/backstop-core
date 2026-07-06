package main

// CheckTypeConsumer is one catalog row: a single GATE-SEMANTIC site where
// lint/build/test/findings identity drives gate behavior the cutover could
// strand (scope-filtering, engine dispatch, or the violation verdict). It is
// DELIBERATELY NOT a row for cosmetic .Pass.String() display/serialization sites
// — those carry no gate-semantic decision and would make the catalog red on every
// added log line (SPEC-041 REQ-005/CLM-020).
type CheckTypeConsumer struct {
	// Site is the file:symbol the keying happens at (the reconciliation key the
	// completeness guard verifies still exists, REQ-006/CLM-024).
	Site string
	// KeysOn names the identity the site keys on: build/lint/test/findings/all.
	KeysOn string
	// PostCutoverSource is the post-cutover source of the behavior, one of:
	// declared-engine-property | findings-pack-engine | surviving-checktype-dispatch.
	// (The genuinely-DELETED sites C-2/C-3 are ABSENT from the live catalog — their
	// rows are removed once their sites are gone, so the stale-entry guard stays
	// green; REQ-005/CLM-021.)
	PostCutoverSource string
}

// Post-cutover source classifications.
const (
	// SourceDeclaredEngineProperty is the NEW declared exempt_from_scope_filter →
	// gate.Violation.ProjectWide bridge (C-1, REQ-004) — the SOLE live ProjectWide
	// locus after the cutover. No CheckType enum identity drives scope.
	SourceDeclaredEngineProperty = "declared-engine-property"
	// SourceFindingsPackEngine is the findings-pack-engine source: findings run
	// through the pack engine; the CheckTypeFindings stamping persists, sourced from
	// that engine (C-5). C-4 (code_check.go:CheckTypeFindings) was REMOVED by
	// ISSUE-018 when the `backstop code check` command was deleted.
	SourceFindingsPackEngine = "findings-pack-engine"
	// SourceSurvivingCheckTypeDispatch tagged the pkg/check CheckType
	// labeling/dispatch sites (C-6/C-7/C-8). Those sites were DELETED by ISSUE-018
	// with the in-process check engine; the constant is retained as the guard-test
	// vocabulary so a re-introduced surviving-dispatch row can be classified.
	SourceSurvivingCheckTypeDispatch = "surviving-checktype-dispatch"
)

// CheckTypeConsumerCatalog returns the machine-readable CheckType-consumer catalog
// scoped to GATE-SEMANTIC consumers, mirroring the SPEC-041 table. C-1 is the
// engine-path declared bridge; C-5 is the surviving findings stamping inside the
// LIVE ParsePackFindings. The genuinely-DELETED sites are ABSENT from the live
// catalog so the stale-entry guard stays green: C-2/C-3 (orphaned gate.go
// exemption + shared-runner feeds) and — removed by ISSUE-018 with the `backstop
// code check` command + in-process check engine — C-4 (code_check.go), C-6
// (check.go:passOrder), C-7 (registry.go:Entries), C-8 (manifest.go:parseCheckType).
func CheckTypeConsumerCatalog() []CheckTypeConsumer {
	return []CheckTypeConsumer{
		// C-1: the declared engine-property bridge (the sole live ProjectWide locus).
		{
			Site:              "cmd/backstop/pack_gate.go:runFindingsEngine",
			KeysOn:            "build",
			PostCutoverSource: SourceDeclaredEngineProperty,
		},
		// C-5: findings stamping in the LIVE ParsePackFindings (parsers.go). Survives.
		{
			Site:              "pkg/check/parsers.go:CheckTypeFindings",
			KeysOn:            "findings",
			PostCutoverSource: SourceFindingsPackEngine,
		},
	}
}
