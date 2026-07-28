package gate

// ISSUE-098 (CLM-001/002/005): the SECOND half of the mandated-test vocabulary.
//
// ResolveMandatedTestPaths answers "does a test function by this name exist in source?"
// by walking pack-declared test-file globs and applying the pack-declared
// TestNameMatcher. That question is categorically unanswerable for a mandated name that
// is a PACK CLAIM id: a pack manifest is not a test file to any classifier, and a claim
// id is not a line shape any test-name matcher recognizes. A pack's claims ARE that
// pack's tests, so the drift resolver needs a second index to consult before calling a
// name absent.

// PackClaimIndex maps a pack-DECLARED claim id to a human-readable evidence label
// identifying where that claim is declared (conventionally "<pack>:<rule>"). The label
// is diagnostic only; presence is decided by key membership alone.
//
// PRESENCE SEMANTICS — DECLARATION, NOT EXECUTION (CLM-005). A key in this index means
// the claim id is declared by an installed pack manifest that the gate has already
// vouched for: pack_lock_verification runs first in the kill chain and halts the gate on
// failure, the installed tree only got there through the full validation pipeline (which
// executes every claim's fixtures), and a manifest cannot parse a claim that declares no
// positive AND negative fixture pair. So a hit means: a claim by that name exists,
// carries a falsifying fixture pair, and passed it at the content now on disk.
//
// It deliberately does NOT mean those fixtures pass right now. Re-running them here is
// not merely expensive (it would drag engine provisioning and subprocess execution into
// what is a pure filesystem sweep) — it would hold pack evidence to a STRICTLY HIGHER
// bar than source evidence, inside a dimension whose whole design is existence-only.
// ResolveMandatedTestPaths proves only that a test function EXISTS; ClassifyStatusDrift
// takes no pass/fail parameter by design. Execution proof lives where it already lives:
// the pack_engines dimension and `backstop pack test`.
type PackClaimIndex map[string]string

// Has reports whether an installed pack declares a claim with this id. A nil index
// reports nothing present, via ordinary nil-map read semantics.
func (i PackClaimIndex) Has(name string) bool {
	_, ok := i[name]
	return ok
}

// ResolvePresentTestNames returns the present-name set ClassifyStatusDrift consumes: the
// UNION of source-resolved mandated tests (those ResolveMandatedTestPaths gave a
// FilePath) and pack-declared claim ids. A name in NEITHER vocabulary stays absent, so
// the union is purely additive and can never turn a real broken promise green.
//
// The union lives HERE, at the drift consumer, and deliberately NOT inside
// ResolveMandatedTestPaths: MandatedTest.FilePath is a shared signal, and the test
// substantiveness Q2 noTarget join skips a mandated test only when FilePath is empty.
// Filling a pack-claim-resolved test's FilePath with its manifest path would make that
// join ask whether a pack.yml references the target package and emit a false "does not
// call package X" violation against a pack manifest. This function never writes
// FilePath, never mutates its input, and never reads the filesystem.
func ResolvePresentTestNames(mandated []MandatedTest, packClaims PackClaimIndex) map[string]bool {
	present := make(map[string]bool, len(mandated))
	for _, mt := range mandated {
		if mt.FilePath != "" || packClaims.Has(mt.FuncName) {
			present[mt.FuncName] = true
		}
	}
	return present
}
