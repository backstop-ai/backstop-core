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
	// that engine (C-4/C-5).
	SourceFindingsPackEngine = "findings-pack-engine"
	// SourceSurvivingCheckTypeDispatch is the surviving CheckType labeling/dispatch
	// lingua franca — the CheckType type stays the gate's pass-identity vocabulary;
	// NO sibling spec deletes it (C-6/C-7/C-8).
	SourceSurvivingCheckTypeDispatch = "surviving-checktype-dispatch"
)

// CheckTypeConsumerCatalog returns the machine-readable CheckType-consumer catalog
// scoped to GATE-SEMANTIC consumers, mirroring the SPEC-041 C-1..C-8 table. C-1 is
// the NEW engine-path declared bridge; C-2/C-3 are the genuinely-DELETED sites
// (orphaned gate.go:1173 exemption + shared-runner feeds, removed in Phases 2/4) —
// these are ABSENT from the live catalog so the stale-entry guard stays green.
// C-4..C-8 are the SURVIVING pkg/check CheckType sites tagged with their real
// post-cutover role, NOT mis-tagged DELETED (REQ-005/CLM-020/CLM-021).
func CheckTypeConsumerCatalog() []CheckTypeConsumer {
	return []CheckTypeConsumer{
		// C-1: the NEW declared engine-property bridge (the sole live ProjectWide locus).
		{
			Site:              "cmd/backstop/pack_gate.go:runFindingsEngine",
			KeysOn:            "build",
			PostCutoverSource: SourceDeclaredEngineProperty,
		},
		// C-4: findings stamping in code_check.go.
		{
			Site:              "cmd/backstop/code_check.go:CheckTypeFindings",
			KeysOn:            "findings",
			PostCutoverSource: SourceFindingsPackEngine,
		},
		// C-5: findings stamping in parsers.go.
		{
			Site:              "pkg/check/parsers.go:CheckTypeFindings",
			KeysOn:            "findings",
			PostCutoverSource: SourceFindingsPackEngine,
		},
		// C-6: passOrder + Violation.Pass/PassResult.Pass + Executors/applicableChecks dispatch.
		{
			Site:              "pkg/check/check.go:passOrder",
			KeysOn:            "all",
			PostCutoverSource: SourceSurvivingCheckTypeDispatch,
		},
		// C-7: registry Entries keyed by CheckType + executor builders.
		{
			Site:              "pkg/check/registry.go:Entries",
			KeysOn:            "all",
			PostCutoverSource: SourceSurvivingCheckTypeDispatch,
		},
		// C-8: manifest enum + parseCheckType + RouteFile/routeFileDefaults routing.
		{
			Site:              "pkg/check/manifest.go:parseCheckType",
			KeysOn:            "all",
			PostCutoverSource: SourceSurvivingCheckTypeDispatch,
		},
	}
}
