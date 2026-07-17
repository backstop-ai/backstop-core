package spine

import "strings"

// Negative fixture for Family B6 (no-pack-name-keyed-capability, ISSUE-063 REQ-005):
// capability/behavior detection keyed on a baked distribution identity — an org/pack
// coordinate literal used as a cfg.Packs map key, and a name-convention suffix test on a
// pack name. Both MUST be flagged. The correct source is a declared gate_type engine
// (see the valid fixture gate-type-declaration-scan.go).

type cfg struct {
	Packs map[string]string
}

// contractsInstalled keys the contracts capability on the BAKED org/pack coordinate
// "backstop/contracts" used as a map key — binds the capability to one GitHub org.
func contractsInstalled(c *cfg) bool {
	_, ok := c.Packs["backstop/contracts"]
	return ok
}

// coverageInstalled keys the coverage capability on a pack-NAME CONVENTION (the
// "-toolchain" suffix) instead of a declared coverage engine.
func coverageInstalled(c *cfg) bool {
	for name := range c.Packs {
		if strings.HasSuffix(name, "-toolchain") {
			return true
		}
	}
	return false
}
