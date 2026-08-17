package packval

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
)

// exemptDecisionCheck is the Check string the advisory carries. Declared once so
// every test in this file agrees on the identifier under test.
const exemptDecisionCheck = "exempt-scope-decision"

// exemptFixtureDir is the tri-state fixture corpus. Every test here parses through
// the real ParseManifest and drives the real RunCoherence — never a hand-built
// PackManifest literal, which carries no declared-key record and would make these
// tests vacuously green (SE-6).
func exemptFixtureDir(name string) string {
	return filepath.Join("testdata", "exempt-decision", name)
}

func parseExemptFixture(t *testing.T, name string) (*PackManifest, string) {
	t.Helper()
	dir := exemptFixtureDir(name)
	pack, err := ParseManifest(filepath.Join(dir, "pack.yml"))
	if err != nil {
		t.Fatalf("ParseManifest(%s): %v", name, err)
	}
	return pack, dir
}

// exemptWarnings returns the exempt-scope-decision warnings RunCoherence emitted
// for the named fixture, plus the whole phase result for status assertions.
func exemptWarnings(t *testing.T, name string) ([]ValidationWarning, *PhaseResult) {
	t.Helper()
	pack, dir := parseExemptFixture(t, name)
	res := RunCoherence(pack, dir)
	var hits []ValidationWarning
	for _, w := range res.Warnings {
		if w.Check == exemptDecisionCheck {
			hits = append(hits, w)
		}
	}
	return hits, res
}

func TestExemptDecision_ProjectWideEngineOmittingKeyWarns(t *testing.T) {
	hits, res := exemptWarnings(t, "projectwide-undeclared")
	if len(hits) != 1 {
		t.Fatalf("want exactly 1 %s warning, got %d: %+v", exemptDecisionCheck, len(hits), res.Warnings)
	}
	w := hits[0]
	if !strings.Contains(w.Message, "onlyengine") {
		t.Errorf("warning message must name the engine; got %q", w.Message)
	}
	if w.Phase != res.Phase {
		t.Errorf("warning Phase = %q, want %q (an empty phase renders as WARN [/%s])", w.Phase, res.Phase, exemptDecisionCheck)
	}
	if w.FixHint == "" {
		t.Errorf("warning must carry a FixHint telling the author which key to add")
	}
}

func TestExemptDecision_ExplicitFalseIsSilent(t *testing.T) {
	hits, res := exemptWarnings(t, "projectwide-explicit-false")
	if len(hits) != 0 {
		t.Fatalf("explicit false is a RECORDED decision and must be silent; got %d warnings: %+v", len(hits), hits)
	}
	// Same effective boolean as projectwide-undeclared, opposite outcome: the
	// check demonstrably reads key PRESENCE, not the value.
	pack, _ := parseExemptFixture(t, "projectwide-explicit-false")
	if pack.Engines["onlyengine"].ExemptFromScopeFilter {
		t.Fatalf("fixture drift: projectwide-explicit-false must resolve ExemptFromScopeFilter=false")
	}
	if res.Status == "fail" {
		t.Errorf("phase status = fail on a clean fixture")
	}
}

func TestExemptDecision_ExplicitTrueIsSilent(t *testing.T) {
	hits, _ := exemptWarnings(t, "projectwide-explicit-true")
	if len(hits) != 0 {
		t.Fatalf("explicit true is a RECORDED decision and must be silent; got %d warnings: %+v", len(hits), hits)
	}
	pack, _ := parseExemptFixture(t, "projectwide-explicit-true")
	if !pack.Engines["onlyengine"].ExemptFromScopeFilter {
		t.Fatalf("fixture drift: projectwide-explicit-true must resolve ExemptFromScopeFilter=true")
	}
}

func TestExemptDecision_FileArgsEngineNeverWarns(t *testing.T) {
	hits, _ := exemptWarnings(t, "filearg-undeclared")
	if len(hits) != 0 {
		t.Fatalf("a file-args engine only reports on files handed to it, so no decision is owed and no warning may fire; got %d: %+v", len(hits), hits)
	}
	pack, _ := parseExemptFixture(t, "filearg-undeclared")
	b := pack.Engines["onlyengine"]
	if b.ScopeKind != engine.ScopeKindFileArgs {
		t.Fatalf("fixture drift: filearg-undeclared must declare scope_kind: file-args")
	}
	// The falsifier: this fixture shares gate_type with projectwide-undeclared,
	// so any gate_type-keyed implementation warns here and fails this test.
	pw, _ := parseExemptFixture(t, "projectwide-undeclared")
	if b.GateType != pw.Engines["onlyengine"].GateType {
		t.Fatalf("fixture drift: filearg-undeclared must carry the SAME gate_type as projectwide-undeclared (%v vs %v) or CLM-004's guard goes vacuous",
			b.GateType, pw.Engines["onlyengine"].GateType)
	}
}

func TestExemptDecision_PromptsWithoutDerivingTheValue(t *testing.T) {
	fixtures := []string{
		"projectwide-undeclared",
		"projectwide-explicit-false",
		"projectwide-explicit-true",
		"filearg-undeclared",
	}
	warnedBy := map[string]bool{}
	gateTypes := map[string]engine.GateType{}
	for _, name := range fixtures {
		pack, dir := parseExemptFixture(t, name)
		before := pack.Engines["onlyengine"].ExemptFromScopeFilter
		res := RunCoherence(pack, dir)
		after := pack.Engines["onlyengine"].ExemptFromScopeFilter
		if before != after {
			t.Errorf("%s: validation REWROTE ExemptFromScopeFilter (%v -> %v); the advisory must prompt, never derive", name, before, after)
		}
		gateTypes[name] = pack.Engines["onlyengine"].GateType
		for _, w := range res.Warnings {
			if w.Check == exemptDecisionCheck {
				warnedBy[name] = true
			}
		}
	}

	// Only scope_kind + key presence may separate the warn set from the silent set.
	want := map[string]bool{
		"projectwide-undeclared":     true,
		"projectwide-explicit-false": false,
		"projectwide-explicit-true":  false,
		"filearg-undeclared":         false,
	}
	for name, expect := range want {
		if warnedBy[name] != expect {
			t.Errorf("%s: warned=%v, want %v", name, warnedBy[name], expect)
		}
	}

	// The corpus must remain a real falsifier for a gate_type-keyed rule: the two
	// undeclared-key fixtures share a gate_type while differing only in scope_kind,
	// and the three project-wide fixtures carry three distinct gate_types.
	if gateTypes["filearg-undeclared"] != gateTypes["projectwide-undeclared"] {
		t.Errorf("corpus drift: the two undeclared fixtures must share a gate_type")
	}
	distinct := map[engine.GateType]bool{
		gateTypes["projectwide-undeclared"]:     true,
		gateTypes["projectwide-explicit-false"]: true,
		gateTypes["projectwide-explicit-true"]:  true,
	}
	if len(distinct) != 3 {
		t.Errorf("corpus drift: the three project-wide fixtures must carry three DISTINCT gate_types, got %d", len(distinct))
	}
}

func TestExemptDecision_ParseManifestAlwaysRecordsDeclaredKeys(t *testing.T) {
	// A manifest with NO engines: block must still produce a non-nil record, so
	// "not parsed" stays distinguishable from "no keys declared" (SE-6).
	noEngines, _ := parseExemptFixture(t, "no-engines")
	if noEngines.declaredEngineKeys == nil {
		t.Fatalf("ParseManifest on a manifest with no engines: block returned a NIL declared-key record; the advisory could then go vacuously silent")
	}
	if len(noEngines.Engines) != 0 {
		t.Fatalf("fixture drift: no-engines must declare no engines, got %d", len(noEngines.Engines))
	}

	for _, name := range []string{"projectwide-undeclared", "projectwide-explicit-false", "projectwide-explicit-true", "filearg-undeclared"} {
		pack, _ := parseExemptFixture(t, name)
		if pack.declaredEngineKeys == nil {
			t.Errorf("%s: declared-key record is nil", name)
		}
	}

	recorded, _ := parseExemptFixture(t, "projectwide-explicit-false")
	if !recorded.declaredEngineKeys["onlyengine"]["exempt_from_scope_filter"] {
		t.Errorf("projectwide-explicit-false: declared-key record must contain exempt_from_scope_filter, got %+v", recorded.declaredEngineKeys["onlyengine"])
	}
	omitted, _ := parseExemptFixture(t, "projectwide-undeclared")
	if omitted.declaredEngineKeys["onlyengine"]["exempt_from_scope_filter"] {
		t.Errorf("projectwide-undeclared: declared-key record must NOT contain exempt_from_scope_filter, got %+v", omitted.declaredEngineKeys["onlyengine"])
	}
	// The record is a real key set, not an empty stub that would make presence
	// unknowable: a key the fixture DOES declare is present.
	if !omitted.declaredEngineKeys["onlyengine"]["scope_kind"] {
		t.Errorf("projectwide-undeclared: declared-key record lost scope_kind, got %+v", omitted.declaredEngineKeys["onlyengine"])
	}
}

func TestExemptDecision_AdvisoryNeverFailsTheRun(t *testing.T) {
	hits, res := exemptWarnings(t, "projectwide-undeclared")
	if len(hits) == 0 {
		t.Fatalf("precondition: expected the advisory to fire on projectwide-undeclared")
	}
	if res.Status == "fail" {
		t.Errorf("phase status = %q; an advisory must never fail the phase (SE-5)", res.Status)
	}
	for _, e := range res.Errors {
		if e.Check == exemptDecisionCheck {
			t.Errorf("advisory leaked into Errors: %+v", e)
		}
	}

	// A Result carrying ONLY this warning finalises to pass through the real
	// FinalizeStatus, which keys status purely on len(Errors).
	r := &Result{Warnings: hits}
	r.FinalizeStatus()
	if r.Status != "pass" {
		t.Errorf("Result.Status = %q with only advisory warnings, want pass", r.Status)
	}
}
