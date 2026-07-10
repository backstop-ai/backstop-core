package gate

import "testing"

// TestEnrichViolationIdentity_ScopePathForms_ByteIdenticalIdentity is the crux
// regression test for ISSUE-046 (CLM-002): the SAME finding (same Rule,
// RegionHash, Severity, SourcePack, same underlying file) whose File arrives in
// the several textual forms the SARIF engine emits by invocation scope — the
// diff-scope explicit-arg form "pkg/pack/manifest.go" vs the full-scope
// directory-walk form "./pkg/pack/manifest.go" (and interior "./" / trailing
// variants) — must yield a BYTE-IDENTICAL Identity and IdentityHash. It FAILS
// today because EnrichViolationIdentity folds File in raw; it passes once File is
// canonicalized before entering identity.
func TestEnrichViolationIdentity_ScopePathForms_ByteIdenticalIdentity(t *testing.T) {
	base := Violation{
		Rule:       "backstop/go-standards/error-wrapping",
		Message:    "wrap errors with context",
		Severity:   "error",
		SourcePack: "backstop/go-standards",
		RegionHash: "region-abc-123",
	}

	forms := []string{
		"pkg/pack/manifest.go",     // diff-scope explicit repo-relative arg form
		"./pkg/pack/manifest.go",   // full-scope directory-walk "./"-prefixed form
		"pkg/pack/./manifest.go",   // interior "./" noise from a walk join
		"pkg/pack/../pack/manifest.go", // redundant traversal that Clean collapses
	}

	want := EnrichViolationIdentity(withFile(base, forms[0]))
	for _, form := range forms {
		got := EnrichViolationIdentity(withFile(base, form))
		if got.Identity != want.Identity {
			t.Fatalf("Identity differs by File form %q:\n got %q\nwant %q", form, got.Identity, want.Identity)
		}
		if got.IdentityHash != want.IdentityHash {
			t.Fatalf("IdentityHash differs by File form %q:\n got %s\nwant %s", form, got.IdentityHash, want.IdentityHash)
		}
	}
}

// TestEnrichViolationIdentity_ShiftedUnchangedFinding_KeepsIdentity (CLM-004):
// a finding whose surrounding file shifted (an unrelated edit moved line numbers
// or reordered the file list) but whose own Rule+File+content is unchanged KEEPS
// its identity, so it grandfathers instead of over-blocking. gate.Violation
// carries NO line/position field, and RegionHash is content-based and
// line-INDEPENDENT (pkg/check.sarifFingerprint), so two violations equal in every
// identity-participating field hash identically regardless of any positional
// shift. This pins that structural property.
func TestEnrichViolationIdentity_ShiftedUnchangedFinding_KeepsIdentity(t *testing.T) {
	before := EnrichViolationIdentity(Violation{
		Rule:       "backstop/go-standards/error-wrapping",
		File:       "pkg/pack/manifest.go",
		Message:    "wrap errors with context",
		Severity:   "error",
		SourcePack: "backstop/go-standards",
		RegionHash: "region-abc-123",
	})
	// Same finding after an unrelated edit shifted its position: identical
	// Rule+File+RegionHash+Severity+SourcePack, only the (non-carried) line moved.
	after := EnrichViolationIdentity(Violation{
		Rule:       "backstop/go-standards/error-wrapping",
		File:       "pkg/pack/manifest.go",
		Message:    "wrap errors with context",
		Severity:   "error",
		SourcePack: "backstop/go-standards",
		RegionHash: "region-abc-123",
	})
	if before.IdentityHash != after.IdentityHash {
		t.Fatalf("shifted-but-unchanged finding lost its identity: %s != %s", before.IdentityHash, after.IdentityHash)
	}
}

// TestEnrichViolationIdentity_ChangedContent_GetsNewIdentity (CLM-005) is the
// anti-false-grandfather / silent-green guard: normalization must NOT collapse
// genuinely distinct findings into one stale baseline entry. A different
// RegionHash (content changed) and a genuinely different File must each produce a
// DISTINCT IdentityHash — so a real new violation still blocks.
func TestEnrichViolationIdentity_ChangedContent_GetsNewIdentity(t *testing.T) {
	baseV := EnrichViolationIdentity(Violation{
		Rule:       "backstop/go-standards/error-wrapping",
		File:       "pkg/pack/manifest.go",
		Message:    "wrap errors with context",
		Severity:   "error",
		SourcePack: "backstop/go-standards",
		RegionHash: "region-abc-123",
	})

	// (a) same Rule+File, DIFFERENT RegionHash → DIFFERENT identity.
	changedContent := EnrichViolationIdentity(Violation{
		Rule:       "backstop/go-standards/error-wrapping",
		File:       "pkg/pack/manifest.go",
		Message:    "wrap errors with context",
		Severity:   "error",
		SourcePack: "backstop/go-standards",
		RegionHash: "region-xyz-999",
	})
	if baseV.IdentityHash == changedContent.IdentityHash {
		t.Fatalf("changed content collapsed to the same identity %s — silent green", baseV.IdentityHash)
	}

	// (b) same Rule+RegionHash, genuinely DIFFERENT File → DIFFERENT identity.
	otherFile := EnrichViolationIdentity(Violation{
		Rule:       "backstop/go-standards/error-wrapping",
		File:       "pkg/pack/loader.go",
		Message:    "wrap errors with context",
		Severity:   "error",
		SourcePack: "backstop/go-standards",
		RegionHash: "region-abc-123",
	})
	if baseV.IdentityHash == otherFile.IdentityHash {
		t.Fatalf("distinct files collapsed to the same identity %s — silent green", baseV.IdentityHash)
	}
}

// TestCompareBaseline_GenerateThenDiffScope_PreexistingGrandfathers (CLM-003)
// exercises the real generate→gate cycle at the identity layer: a baseline whose
// single violation carries the full-scope File form, compared against a current
// run carrying the SAME violation in the diff-scope File form under a diff
// GateScope that Contains the file. IDENTITY STABILITY (ISSUE-046) is asserted via
// FixedViolations == 0: a stable identity means the baseline finding MATCHED a
// current finding (not orphaned), so it is not mis-counted as Fixed — a broken
// identity would surface it as both net-new AND fixed. Under the strict file-level
// ratchet (ISSUE-050) the touched file's grandfather is REVOKED, so the matched
// finding now reports as exactly one NEW violation (revocation, not a net-new
// identity miss). The FixedViolations == 0 assertion is what still pins the
// ISSUE-046 canonicalization guarantee.
func TestCompareBaseline_GenerateThenDiffScope_PreexistingGrandfathers(t *testing.T) {
	// Baseline captured under full scope: File in the directory-walk "./"-form.
	baseline := &BaselineArtifact{
		SchemaVersion: BaselineSchemaV1,
		Violations: []Violation{{
			Rule:       "backstop/go-standards/error-wrapping",
			File:       "./pkg/pack/manifest.go",
			Message:    "wrap errors with context",
			Severity:   "error",
			SourcePack: "backstop/go-standards",
			RegionHash: "region-abc-123",
		}},
	}
	// Current diff-scope run: SAME finding, File in the explicit repo-relative form.
	current := []Violation{{
		Rule:       "backstop/go-standards/error-wrapping",
		File:       "pkg/pack/manifest.go",
		Message:    "wrap errors with context",
		Severity:   "error",
		SourcePack: "backstop/go-standards",
		RegionHash: "region-abc-123",
	}}

	comparison := CompareBaseline(current, baseline, BaselineCompareOptions{
		Scope: newGateScope("", GateScopeModeDiff, []string{"pkg/pack/manifest.go"}, nil),
	})

	// Revoked-on-touch (ISSUE-050): the matched finding is on the touched file, so it
	// reports as exactly one NEW violation via the scope-touch path — NOT a net-new
	// identity miss.
	if len(comparison.NewViolations) != 1 {
		t.Fatalf("touched-file finding must be revoked as exactly one NEW across scope forms: got %d new %#v", len(comparison.NewViolations), comparison.NewViolations)
	}
	// Identity stability (ISSUE-046): stable identity ⇒ baseline finding matched
	// current ⇒ NOT orphaned as Fixed. A broken identity would surface it as fixed.
	if len(comparison.FixedViolations) != 0 {
		t.Fatalf("pre-existing finding mis-counted as FIXED across scope forms (identity unstable): got %d fixed %#v", len(comparison.FixedViolations), comparison.FixedViolations)
	}
}

// withFile returns a copy of v with File set to file (test-local helper; keeps the
// scope-form table above readable without re-declaring Violation literals).
func withFile(v Violation, file string) Violation {
	v.File = file
	return v
}
