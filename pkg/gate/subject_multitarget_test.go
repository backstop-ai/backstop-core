package gate

import (
	"os"
	"path/filepath"
	"testing"
)

// subject_multitarget_test.go drives the ISSUE-047 de-bake + per-claim subject
// resolution: ExtractMandatedTests stamps each MandatedTest.TargetPkg from the
// claim-level `subject` when present (multi-target specs), else the spec-level
// default; the parser reads the neutral `subject` with the legacy `package` as a
// deprecated alias; and the reduced OPAQUE token flows through the noTarget join
// with a POSITIVE and a NEGATIVE sharp control proving the guard stays non-vacuous
// after the "cmd/"/"pkg/" layout literals are removed.

// writeSpecDir writes a single spec file with the given body into a fresh temp
// dir and returns the dir, for driving ExtractMandatedTests over a controlled
// spec. The content is an in-test Go string literal (no on-disk fixture).
func writeSpecDir(t *testing.T, name, body string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatalf("writing spec fixture: %v", err)
	}
	return dir
}

// TestExtractMandatedTests_PerClaimSubjectOverridesDefault (CLM-004) — a spec whose
// implementation subject is `pkg/check` with ONE claim carrying `subject: pkg/gate`
// yields MandatedTests whose TargetPkg is the reduced OVERRIDE (`gate`) for that
// claim's tests and the reduced DEFAULT (`check`) for the other claims.
func TestExtractMandatedTests_PerClaimSubjectOverridesDefault(t *testing.T) {
	body := `---
title: Multi-Target Fixture
number: SPEC-901
status: draft
implementation:
  subject: pkg/check
claims:
  - id: CLM-001
    subject: pkg/gate
    tests:
      - TestOverriddenClaim
  - id: CLM-002
    tests:
      - TestDefaultClaim
---

# Multi-Target Fixture
`
	dir := writeSpecDir(t, "SPEC-901-multitarget.spec.md", body)
	tests, err := ExtractMandatedTests(dir)
	if err != nil {
		t.Fatalf("ExtractMandatedTests: %v", err)
	}

	overridden, ok := mandatedByName(tests, "TestOverriddenClaim")
	if !ok {
		t.Fatalf("TestOverriddenClaim not extracted; got %+v", tests)
	}
	if overridden.TargetPkg != "gate" {
		t.Errorf("per-claim subject override must reduce to 'gate'; got %q", overridden.TargetPkg)
	}

	def, ok := mandatedByName(tests, "TestDefaultClaim")
	if !ok {
		t.Fatalf("TestDefaultClaim not extracted; got %+v", tests)
	}
	if def.TargetPkg != "check" {
		t.Errorf("claim without per-claim subject must inherit the reduced spec default 'check'; got %q", def.TargetPkg)
	}
}

// TestExtractMandatedTests_LegacyPackageAliasStillResolves (CLM-002) — a spec
// declaring ONLY the legacy `package: pkg/gate` (no `subject`) still resolves
// TargetPkg to `gate`; the parser alias keeps the ~40 unmigrated specs working.
func TestExtractMandatedTests_LegacyPackageAliasStillResolves(t *testing.T) {
	body := `---
title: Legacy Alias Fixture
number: SPEC-902
status: draft
implementation:
  package: pkg/gate
claims:
  - id: CLM-001
    tests:
      - TestLegacyAliasResolves
---

# Legacy Alias Fixture
`
	dir := writeSpecDir(t, "SPEC-902-legacy.spec.md", body)
	tests, err := ExtractMandatedTests(dir)
	if err != nil {
		t.Fatalf("ExtractMandatedTests: %v", err)
	}
	mt, ok := mandatedByName(tests, "TestLegacyAliasResolves")
	if !ok {
		t.Fatalf("TestLegacyAliasResolves not extracted; got %+v", tests)
	}
	if mt.TargetPkg != "gate" {
		t.Errorf("legacy package alias must resolve TargetPkg to 'gate'; got %q", mt.TargetPkg)
	}
}

// TestExtractMandatedTests_AbsenceStillSkips (CLM-004) — a claim with `kind: absence`
// still sets IsAbsence (pre-join skip preserved) even when it ALSO carries a per-claim
// subject; the two per-claim signals are independent.
func TestExtractMandatedTests_AbsenceStillSkips(t *testing.T) {
	body := `---
title: Absence Plus Subject Fixture
number: SPEC-903
status: draft
implementation:
  subject: pkg/check
claims:
  - id: CLM-001
    kind: absence
    subject: pkg/gate
    tests:
      - TestAbsenceWithSubject
---

# Absence Plus Subject Fixture
`
	dir := writeSpecDir(t, "SPEC-903-absence-subject.spec.md", body)
	tests, err := ExtractMandatedTests(dir)
	if err != nil {
		t.Fatalf("ExtractMandatedTests: %v", err)
	}
	mt, ok := mandatedByName(tests, "TestAbsenceWithSubject")
	if !ok {
		t.Fatalf("TestAbsenceWithSubject not extracted; got %+v", tests)
	}
	if !mt.IsAbsence {
		t.Errorf("a kind:absence claim must set IsAbsence even with a per-claim subject present; got false")
	}
	// The per-claim subject is still resolved onto TargetPkg (the absence signal is
	// orthogonal — it governs the pre-join skip, not the target derivation).
	if mt.TargetPkg != "gate" {
		t.Errorf("per-claim subject must still resolve TargetPkg to 'gate'; got %q", mt.TargetPkg)
	}
	// And the pre-join skip is honored regardless of set membership.
	if v, raised := NoTargetViolationForTest(mt, ReferencedSymbolSet{}, false); raised {
		t.Errorf("absence test must skip the noTarget join; got raised violation %+v", v)
	}
}

// TestNoTarget_PositiveControl_ReferencedSubjectSatisfied (CLM-006) — the POSITIVE
// control: a cross-package test whose referenced set CONTAINS its declared subject
// leaf (`gate`, resolved from a per-claim `subject: pkg/gate`) is SATISFIED. Exercises
// the new resolution end-to-end into the join.
func TestNoTarget_PositiveControl_ReferencedSubjectSatisfied(t *testing.T) {
	body := `---
title: Positive Control Fixture
number: SPEC-904
status: draft
implementation:
  subject: pkg/check
claims:
  - id: CLM-001
    subject: pkg/gate
    tests:
      - TestReferencesGate
---

# Positive Control Fixture
`
	dir := writeSpecDir(t, "SPEC-904-positive.spec.md", body)
	tests, err := ExtractMandatedTests(dir)
	if err != nil {
		t.Fatalf("ExtractMandatedTests: %v", err)
	}
	mt, ok := mandatedByName(tests, "TestReferencesGate")
	if !ok {
		t.Fatalf("TestReferencesGate not extracted; got %+v", tests)
	}
	// Cross-package (samePackage=false) but the referenced set contains the subject leaf.
	referenced := ReferencedSymbolSet{"gate": true, "other": true}
	if v, raised := NoTargetViolationForTest(mt, referenced, false); raised {
		t.Errorf("a test that references its declared subject leaf must be SATISFIED; got raised %+v", v)
	}
}

// TestNoTarget_NegativeControl_WrongSubjectFiresLoud (CLM-006) — the NEGATIVE control
// (anti-vacuous): a cross-package test whose referenced set does NOT contain its
// declared subject leaf (declares `gate`, references only `other`) FIRES noTarget
// loudly, proving the guard stays sharp after the layout literals are removed.
func TestNoTarget_NegativeControl_WrongSubjectFiresLoud(t *testing.T) {
	body := `---
title: Negative Control Fixture
number: SPEC-905
status: draft
implementation:
  subject: pkg/check
claims:
  - id: CLM-001
    subject: pkg/gate
    tests:
      - TestReferencesWrongSubject
---

# Negative Control Fixture
`
	dir := writeSpecDir(t, "SPEC-905-negative.spec.md", body)
	tests, err := ExtractMandatedTests(dir)
	if err != nil {
		t.Fatalf("ExtractMandatedTests: %v", err)
	}
	mt, ok := mandatedByName(tests, "TestReferencesWrongSubject")
	if !ok {
		t.Fatalf("TestReferencesWrongSubject not extracted; got %+v", tests)
	}
	// Cross-package and the referenced set lacks the subject leaf `gate`.
	referenced := ReferencedSymbolSet{"other": true}
	v, raised := NoTargetViolationForTest(mt, referenced, false)
	if !raised {
		t.Fatalf("a test that does NOT reference its declared subject leaf MUST fire noTarget (guard stays sharp); got no violation")
	}
	if v.Message != "test function TestReferencesWrongSubject does not call package gate" {
		t.Errorf("unexpected noTarget message: %q", v.Message)
	}
}
