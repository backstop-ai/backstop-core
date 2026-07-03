package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	yaml "gopkg.in/yaml.v3"
)

// SPEC-048 REQ-004b / CLM-013 — the phantom SPEC-040 contract retire. SPEC-040's
// contracts block declares `dispatchBuildViolationProjectWide` (kind: variable), a
// named symbol that NEVER existed: SPEC-041 REQ-004 superseded that transitional
// approach with the shipped EngineBinding.ExemptFromScopeFilter field mapping
// per-violation to gate.Violation.ProjectWide. The stale entry must be REALIGNED so
// contract_signature reports an HONEST contract for SPEC-040 — never silenced by a
// baseline/waiver. This test parses SPEC-040's contracts frontmatter and asserts the
// phantom symbol is gone and the realigned contract references the real mechanism.
// RED while the phantom entry is present (the retire is routed through the
// spec-author agent, not a hand-edit).

// specContractsFrontmatter is the minimal shape of a spec's `contracts:` frontmatter
// block the retirement test inspects.
type specContractsFrontmatter struct {
	Contracts []struct {
		File     string `yaml:"file"`
		Provides []struct {
			Name      string `yaml:"name"`
			Kind      string `yaml:"kind"`
			Signature string `yaml:"signature"`
			Notes     string `yaml:"notes"`
		} `yaml:"provides"`
	} `yaml:"contracts"`
}

// parseSpecContracts reads a spec's YAML frontmatter (between the first two `---`
// fences) and unmarshals its contracts block.
func parseSpecContracts(t *testing.T, specPath string) specContractsFrontmatter {
	t.Helper()
	raw, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("read spec %s: %v", specPath, err)
	}
	s := string(raw)
	if !strings.HasPrefix(s, "---") {
		t.Fatalf("spec %s must open with a --- frontmatter fence", specPath)
	}
	rest := s[len("---"):]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		t.Fatalf("spec %s must have a closing --- frontmatter fence", specPath)
	}
	var fm specContractsFrontmatter
	if err := yaml.Unmarshal([]byte(rest[:end]), &fm); err != nil {
		t.Fatalf("unmarshal spec %s frontmatter: %v", specPath, err)
	}
	return fm
}

// TestSpec040Contract_PhantomBuildViolationSymbolRetired asserts SPEC-040's contracts
// no longer declare the phantom `dispatchBuildViolationProjectWide` (kind: variable)
// symbol, and that the realigned contract references the shipped
// EngineBinding.ExemptFromScopeFilter -> gate.Violation.ProjectWide mechanism
// (SPEC-041 REQ-004) so contract_signature is honest — not silenced by a
// baseline/waiver (CLM-013).
func TestSpec040Contract_PhantomBuildViolationSymbolRetired(t *testing.T) {
	specPath := filepath.Join(repoRoot(t), "specs", "SPEC-040-toolchain-pack-cutover.spec.md")
	fm := parseSpecContracts(t, specPath)

	if len(fm.Contracts) == 0 {
		t.Fatal("SPEC-040 must declare a contracts block")
	}

	referencesExemptMechanism := false
	for _, c := range fm.Contracts {
		for _, p := range c.Provides {
			if p.Name == "dispatchBuildViolationProjectWide" {
				t.Errorf("SPEC-040 still declares the phantom symbol %q (kind: %q) — a symbol that never existed; it must be REALIGNED to the shipped ExemptFromScopeFilter -> Violation.ProjectWide mechanism, not baselined/waived", p.Name, p.Kind)
			}
			blob := p.Name + " " + p.Signature + " " + p.Notes
			if strings.Contains(blob, "ExemptFromScopeFilter") && strings.Contains(blob, "ProjectWide") {
				referencesExemptMechanism = true
			}
		}
	}

	if !referencesExemptMechanism {
		t.Errorf("SPEC-040's realigned contract must reference the shipped EngineBinding.ExemptFromScopeFilter -> gate.Violation.ProjectWide mechanism (SPEC-041 REQ-004) so contract_signature is honest")
	}
}
