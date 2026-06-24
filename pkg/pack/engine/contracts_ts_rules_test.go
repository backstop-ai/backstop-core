package engine

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// contracts_ts_rules_test.go (SPEC-038 TASK-014, REQ-007): the TypeScript contract
// rules in the SHARED ts-proof pack, over REAL ast-grep + REAL grep. A present .ts
// signature MATCHES (SATISFIED) and an absent/mismatched one VIOLATES (CLM-023); a
// present forbidden .ts symbol -> grep violation, absent -> PASS (CLM-024); the rules
// ride STRUCTURAL engines only — no eslint/tsc for contracts (CLM-025); and they are
// ADDED to the SAME pack that holds Seed 3's substantiveness rules, not a second pack
// (CLM-026).

const tsPackRel = "pkg/gate/testdata/ts-proof-pack"

func tsFixture(root, name string) string {
	return filepath.Join(root, tsPackRel, "fixtures", "ts", name)
}

// compileTSSignature runs the pack's TS compile-signature-ts.sh.
func compileTSSignature(t *testing.T, root, sig string) string {
	t.Helper()
	script := filepath.Join(root, tsPackRel, "scripts", "compile-signature-ts.sh")
	if _, err := os.Stat(script); err != nil {
		t.Fatalf("TS signature compiler must exist in the shared pack (CLM-006): %v", err)
	}
	cmd := exec.Command("/bin/sh", script, sig)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("compile-signature-ts.sh failed: %v (stderr: %s)", err, stderr.String())
	}
	return strings.TrimSpace(stdout.String())
}

func tsAstGrepMatchCount(t *testing.T, root, pattern, file string) int {
	t.Helper()
	if _, err := exec.LookPath("ast-grep"); err != nil {
		t.Fatalf("real ast-grep is required (no t.Skip): %v", err)
	}
	raw := runEngineStdout(t, "ast-grep", "run", "--pattern", pattern, "--lang", "typescript", "--json", file)
	convert := filepath.Join(root, tsPackRel, "ast-grep", "to-sarif.sh")
	return sarifResultCount(t, pipeConvert(t, convert, raw))
}

func tsGrepMatchCount(t *testing.T, root, symbol, scope string) int {
	t.Helper()
	if _, err := exec.LookPath("grep"); err != nil {
		t.Fatalf("real grep is required (no t.Skip): %v", err)
	}
	raw := runEngineStdout(t, "grep", "-rn", "-e", symbol, scope)
	convert := filepath.Join(root, tsPackRel, "grep", "to-sarif.sh")
	return sarifResultCount(t, pipeConvert(t, convert, raw))
}

// TestTSPack_ContractSignaturePresenceAstGrep: a present .ts signature MATCHES
// (SATISFIED) and an absent/mismatched one yields a VIOLATION (CLM-023), via real
// ast-grep over real .ts fixtures.
func TestTSPack_ContractSignaturePresenceAstGrep(t *testing.T) {
	root := repoRoot(t)
	pattern := compileTSSignature(t, root, "function routeFile(path: string, mode: number): string")
	if got := tsAstGrepMatchCount(t, root, pattern, tsFixture(root, "contract-sig-present.ts")); got == 0 {
		t.Fatalf("present .ts signature must MATCH (SATISFIED), got 0 with pattern %q", pattern)
	}
	if got := tsAstGrepMatchCount(t, root, pattern, tsFixture(root, "contract-sig-absent.ts")); got != 0 {
		t.Fatalf("absent/mismatched .ts signature must NOT match (VIOLATION), got %d", got)
	}
}

// TestTSPack_ContractAbsenceGrep: a present forbidden .ts symbol -> grep violation,
// absent -> PASS (CLM-024), real grep over real .ts fixtures.
func TestTSPack_ContractAbsenceGrep(t *testing.T) {
	root := repoRoot(t)
	if got := tsGrepMatchCount(t, root, "legacyTsHelper", tsFixture(root, "contract-absence-present.ts")); got == 0 {
		t.Fatal("present forbidden .ts symbol must produce a grep match (absence violation), got 0")
	}
	if got := tsGrepMatchCount(t, root, "legacyTsHelper", tsFixture(root, "contract-absence-clean.ts")); got != 0 {
		t.Fatalf("absent forbidden .ts symbol must PASS (empty grep), got %d", got)
	}
}

// tsManifest is the minimal view of the shared pack needed for the structural-only
// and shared-pack assertions.
type tsManifest struct {
	Name    string `yaml:"name"`
	Engines map[string]struct {
		Command string `yaml:"command"`
	} `yaml:"engines"`
	Content struct {
		Ruleset struct {
			Rules []struct {
				ID     string `yaml:"id"`
				Engine string `yaml:"engine"`
			} `yaml:"rules"`
		} `yaml:"ruleset"`
	} `yaml:"content"`
}

func loadTSManifest(t *testing.T, root string) tsManifest {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, tsPackRel, "pack.yml"))
	if err != nil {
		t.Fatalf("reading ts-proof pack.yml: %v", err)
	}
	var m tsManifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshalling ts-proof pack.yml: %v", err)
	}
	return m
}

// TestTSPack_ContractRulesUseStructuralEnginesOnly: the TS contract rules bind ONLY
// structural engines (ast-grep / grep) — NO TS toolchain engine (eslint/tsc) is
// bound for the contract rules (CLM-025).
func TestTSPack_ContractRulesUseStructuralEnginesOnly(t *testing.T) {
	m := loadTSManifest(t, repoRoot(t))
	allowed := map[string]bool{"ast-grep": true, "grep": true}
	var sawContractRule bool
	for _, r := range m.Content.Ruleset.Rules {
		if !strings.HasPrefix(r.ID, "contract-") {
			continue
		}
		sawContractRule = true
		if !allowed[r.Engine] {
			t.Errorf("contract rule %q binds engine %q — contracts must ride structural engines only (ast-grep/grep), never a TS toolchain engine (CLM-025)", r.ID, r.Engine)
		}
	}
	if !sawContractRule {
		t.Fatal("expected at least one contract-* rule in the shared TS pack")
	}
	// And the pack declares no eslint/tsc engine at all.
	for name := range m.Engines {
		if name == "eslint" || name == "tsc" {
			t.Errorf("shared TS pack must not declare a %q toolchain engine for contracts (CLM-025)", name)
		}
	}
}

// TestTSPack_ContractRulesShareSubstantivenessPack: the TS contract rules are added
// to the SAME shared TS proof pack that holds Seed 3's substantiveness (hollow-test-ts)
// rule — one stack-locked pack, not a second TS pack (CLM-026).
func TestTSPack_ContractRulesShareSubstantivenessPack(t *testing.T) {
	root := repoRoot(t)
	m := loadTSManifest(t, root)
	if m.Name != "backstop/ts-proof" {
		t.Fatalf("contract rules must live in the shared backstop/ts-proof pack, got name %q (CLM-026)", m.Name)
	}
	ids := map[string]bool{}
	for _, r := range m.Content.Ruleset.Rules {
		ids[r.ID] = true
	}
	// Seed 3's substantiveness rule must STILL be present (not clobbered).
	if !ids["hollow-test-ts"] {
		t.Error("the shared pack must still carry Seed 3's hollow-test-ts rule (additive, not clobbered — CLM-026)")
	}
	// Seed 4's contract rules must be present in the SAME pack.
	if !ids["contract-signature-ts"] || !ids["contract-absence-ts"] {
		t.Error("the shared pack must carry the new TS contract rules in the SAME manifest (CLM-026)")
	}
	// There must be exactly ONE ts-proof pack on disk (no second/relocated TS pack).
	other := filepath.Join(root, ".backstop", "packs", "backstop", "ts-proof")
	if _, err := os.Stat(other); err == nil {
		t.Errorf("a second/relocated TS pack at %s would violate CLM-026 — there must be one shared TS pack", other)
	}
}
