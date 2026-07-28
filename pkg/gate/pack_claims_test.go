package gate

// ISSUE-098 Phase 1 (CLM-001/002/008): the pack-claim half of the mandated-test
// vocabulary, proven in isolation. These drive PackClaimIndex and
// ResolvePresentTestNames over LITERAL index and MandatedTest values — no manifest
// reading (that is Phase 2's job, in cmd/backstop) and no filesystem access, so the
// union semantics are pinned independently of where the index comes from.

import "testing"

// TestPackClaimIndex_DeclaredClaimIDResolvesPresent (CLM-001): an id a pack declares
// resolves present, and carries a non-empty evidence label — the label is what a future
// diagnostic surfaces to say WHERE the evidence lives.
func TestPackClaimIndex_DeclaredClaimIDResolvesPresent(t *testing.T) {
	idx := PackClaimIndex{"type-signature-go": "backstop-ai/go-contracts:contract-signature"}

	if !idx.Has("type-signature-go") {
		t.Fatalf("Has(%q) = false, want true — a declared claim id must resolve present", "type-signature-go")
	}
	if label := idx["type-signature-go"]; label == "" {
		t.Errorf("evidence label for %q is empty; the index must record where the claim is declared", "type-signature-go")
	}
}

// TestPackClaimIndex_FabricatedClaimIDStaysAbsent (CLM-008): the no-vacuous-presence
// falsifier at the unit grain. An id no pack declares never resolves, and the nil index
// resolves nothing without panicking (Go nil-map read semantics, relied on deliberately).
func TestPackClaimIndex_FabricatedClaimIDStaysAbsent(t *testing.T) {
	idx := PackClaimIndex{
		"type-signature-go":      "backstop-ai/go-contracts:contract-signature",
		"const-signature-go":     "backstop-ai/go-contracts:contract-signature",
		"var-signature-go":       "backstop-ai/go-contracts:contract-signature",
		"method-signature-go":    "backstop-ai/go-contracts:contract-signature",
		"interface-signature-go": "backstop-ai/go-contracts:contract-signature",
	}

	if idx.Has("no-such-claim-go") {
		t.Errorf("Has(%q) = true, want false — presence must never be vacuous", "no-such-claim-go")
	}

	var nilIdx PackClaimIndex
	if nilIdx.Has("type-signature-go") {
		t.Errorf("nil index reported %q present, want absent", "type-signature-go")
	}
}

// TestResolvePresentTestNames_UnionsSourcePathAndPackClaim (CLM-002): the whole fix in
// one assertion. A source-resolved test (FilePath != "") is present; a name with NO
// source file but a pack-declared claim id is ALSO present; a name in neither vocabulary
// stays absent.
func TestResolvePresentTestNames_UnionsSourcePathAndPackClaim(t *testing.T) {
	mandatedTests := []MandatedTest{
		{FuncName: "TestSourceResolved", FilePath: "pkg/gate/some_test.go"},
		{FuncName: "type-signature-go"},
		{FuncName: "no-such-claim-go"},
	}
	idx := PackClaimIndex{"type-signature-go": "backstop-ai/go-contracts:contract-signature"}

	present := ResolvePresentTestNames(mandatedTests, idx)

	if !present["TestSourceResolved"] {
		t.Errorf("source-resolved test absent from the present set")
	}
	if !present["type-signature-go"] {
		t.Errorf("pack-declared claim id absent from the present set — this is the ISSUE-098 defect")
	}
	if present["no-such-claim-go"] {
		t.Errorf("name in neither vocabulary reported present; presence must never be vacuous")
	}

	// The union must not write back into the caller's records: FilePath feeds the
	// substantiveness noTarget join, which would then interrogate a pack.yml.
	if mandatedTests[1].FilePath != "" {
		t.Errorf("FilePath = %q for a pack-claim-resolved test, want empty", mandatedTests[1].FilePath)
	}
}

// TestResolvePresentTestNames_SourceOnlyBehaviorUnchanged (CLM-002 regression guard):
// with no pack claims in play the result is EXACTLY the FilePath != "" set, proving the
// union is purely additive and can never make a source-resolved name absent.
func TestResolvePresentTestNames_SourceOnlyBehaviorUnchanged(t *testing.T) {
	mandatedTests := []MandatedTest{
		{FuncName: "TestResolved", FilePath: "cmd/backstop/gate_test.go"},
		{FuncName: "TestUnresolved"},
	}

	for _, tc := range []struct {
		name string
		idx  PackClaimIndex
	}{
		{name: "nil index", idx: nil},
		{name: "empty index", idx: PackClaimIndex{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			present := ResolvePresentTestNames(mandatedTests, tc.idx)

			if len(present) != 1 {
				t.Fatalf("present set has %d entries, want exactly 1 (the FilePath-resolved name): %v", len(present), present)
			}
			if !present["TestResolved"] {
				t.Errorf("source-resolved test missing from the present set")
			}
			if present["TestUnresolved"] {
				t.Errorf("unresolved test reported present with no pack evidence")
			}
		})
	}
}
