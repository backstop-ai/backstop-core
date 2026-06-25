package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/bmanson/backstop-core/pkg/gate"
)

// goldenViolation is the normalized, serialized shape of one gate.Violation the
// golden-equivalence fixture round-trips (SPEC-040 REQ-003/CLM-009). It captures
// exactly the fields the equivalence comparison is keyed on — the namespaced
// rule, file, message, severity, and source pack — so the golden set and the
// reproduced set compare like-for-like independent of unexported identity hashes.
type goldenViolation struct {
	Rule       string `json:"Rule"`
	File       string `json:"File"`
	Message    string `json:"Message"`
	Severity   string `json:"Severity"`
	SourcePack string `json:"SourcePack"`
}

// loadGoldenViolations reads and parses the serialized golden violation set
// captured from the legacy pkg/check engine's normalization of the backstop
// repo's captured tool output (SPEC-040 REQ-003/CLM-009). It is the one-shot
// equivalence evidence the Phase-6 deletion is gated on.
func loadGoldenViolations(path string) ([]goldenViolation, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading golden fixture %s: %w", path, err)
	}
	var out []goldenViolation
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("parsing golden fixture %s: %w", path, err)
	}
	return out, nil
}

// normalizeViolations projects a []gate.Violation onto the comparable
// goldenViolation shape and sorts it deterministically (by rule then message),
// so the reproduced engine-path set can be compared field-for-field against the
// golden set regardless of dispatch ordering (SPEC-040 REQ-003/CLM-010).
func normalizeViolations(vs []gate.Violation) []goldenViolation {
	out := make([]goldenViolation, 0, len(vs))
	for _, v := range vs {
		out = append(out, goldenViolation{
			Rule:       v.Rule,
			File:       v.File,
			Message:    v.Message,
			Severity:   v.Severity,
			SourcePack: v.SourcePack,
		})
	}
	sortGoldenViolations(out)
	return out
}

// sortGoldenViolations orders a golden set canonically (rule, then message, then
// file) so two sets that are equal as multisets compare equal element-by-element.
func sortGoldenViolations(vs []goldenViolation) {
	sort.Slice(vs, func(i, j int) bool {
		if vs[i].Rule != vs[j].Rule {
			return vs[i].Rule < vs[j].Rule
		}
		if vs[i].Message != vs[j].Message {
			return vs[i].Message < vs[j].Message
		}
		return vs[i].File < vs[j].File
	})
}

// goldenViolationsEqual reports whether two normalized golden sets are
// element-for-element equal (both must already be sorted via
// normalizeViolations / sortGoldenViolations). It is the equivalence predicate
// the golden-equivalence proof asserts (SPEC-040 REQ-003/CLM-010).
func goldenViolationsEqual(a, b []goldenViolation) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
