package spine

// Positive fixture for Family B6 (no-pack-name-keyed-capability, ISSUE-063 REQ-005):
// capability is derived from a DECLARED gate_type engine, not from a pack name/coordinate.
// No cfg.Packs name-literal map key and no HasSuffix/HasPrefix/Contains test on a pack
// name — nothing here may trigger the rule.

type engineSpec struct {
	GateType string
}

type manifest struct {
	Engines map[string]engineSpec
}

// packDeclaresGateType keys a capability on whether some installed pack DECLARES an
// engine with the matching gate_type — the correct, org-agnostic source.
func packDeclaresGateType(packs []manifest, dim string) bool {
	for _, m := range packs {
		for _, e := range m.Engines {
			if e.GateType == dim {
				return true
			}
		}
	}
	return false
}
