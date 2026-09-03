package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/gate"
)

// gate_contract_spec072_nongo_test.go (ISSUE-200 / PLAN-ISSUE-200 TASK-001) probes the
// five SPEC-072 contract entries declared over NON-Go files — two YAML registries and
// three Mermaid architecture diagrams. The go-contracts pack compiles every declared
// `signature` into an ast-grep GO STRUCT pattern, so `ast-grep --lang go` over YAML or
// Mermaid can never match and VerifyContractVerdict raises `contract_signature` for each
// one the moment the file enters a Gate diff. There is no conforming content edit; the
// conforming fix is to stop declaring these as machine-probed `provides` and declare them
// as `consumes` of the bash verifier that actually enforces them.
//
// The tests read the EXTRACTED contract set and the PRODUCTION step verdict. They never
// read the bytes of the probed files: the signal is what the gate extracts and verdicts,
// never a grep for a signature literal in a YAML or Mermaid file.
//
// Every helper here is prefixed spec072Probe so it cannot collide with the spec072Fence*
// helpers that share package main.

// spec072ProbeEntries returns every ContractEntry the gate extracts from the REAL specs
// directory. Extraction reads `provides` only — a `consumes` entry produces no entry and
// therefore nothing for the pack to probe.
func spec072ProbeEntries(t *testing.T, root string) []gate.ContractEntry {
	t.Helper()
	entries, err := gate.ExtractContractEntries(filepath.Join(root, "specs"), root)
	if err != nil {
		t.Fatalf("extracting contract entries from %s: %v", filepath.Join(root, "specs"), err)
	}
	return entries
}

// spec072ProbeEntriesNamed returns every extracted entry carrying the given symbol name.
// A non-empty result IS the defect: extraction reads `provides` only, so a surviving entry
// means the symbol is still declared as a machine-probed promise over a file that can never
// satisfy a Go ast-grep pattern.
func spec072ProbeEntriesNamed(entries []gate.ContractEntry, name string) []gate.ContractEntry {
	var matched []gate.ContractEntry
	for _, entry := range entries {
		if entry.Name == name {
			matched = append(matched, entry)
		}
	}
	return matched
}

// spec072ProbeStep runs the PRODUCTION contract gate step — buildContractStep ->
// produceContractEngineResults -> the installed go-contracts pack -> ast-grep ->
// gate.StepContractSignatureScopedFunc -> gate.VerifyContractVerdict — over a file-mode
// gate scope covering the given repository-relative paths. No seam stub is installed, so
// contractEngineResultsFn stays nil and the dispatch is the real one.
func spec072ProbeStep(t *testing.T, root string, files ...string) gate.StepResult {
	t.Helper()
	scope, err := gate.ComputeGateScope(root, gate.GateScopeModeFile, files)
	if err != nil {
		t.Fatalf("computing file-mode gate scope over %v: %v", files, err)
	}
	step := buildContractStep(filepath.Join(root, "specs"), root, scope)
	result := step(context.Background())
	if result.ConfigErr {
		t.Fatalf("contract step over %v reported a config error rather than a verdict: %+v", files, result.Violations)
	}
	return result
}

// spec072ProbeViolationsNaming returns the contract_signature violations in a step result
// whose File or Message names any of the given needles.
func spec072ProbeViolationsNaming(result gate.StepResult, needles ...string) []gate.Violation {
	var matched []gate.Violation
	for _, violation := range result.Violations {
		if violation.Rule != gate.StepContractSignature {
			continue
		}
		for _, needle := range needles {
			if strings.Contains(violation.Message, needle) || strings.Contains(violation.File, needle) {
				matched = append(matched, violation)
				break
			}
		}
	}
	return matched
}

// spec072ProbeControlProvesPackIsLive is the no-vacuous-green control. An absent
// violation only means something when the pack path is known to run: when the contracts
// pack does not resolve, produceContractEngineResults returns NO results at all and every
// "no violation for this symbol" assertion above passes for the wrong reason.
//
// The control pushes ONE artificial present-signature contract over a synthetic non-Go
// file through the same production producer. Two preconditions make it a real control:
//
//   - projectRoot is the REAL repository root, never a temp dir. Pack resolution reads
//     <projectRoot>/backstop.yml, <projectRoot>/backstop.lock and
//     <projectRoot>/.backstop/packs/, so a temp root resolves no pack, returns no results,
//     and the control stops controlling anything.
//   - the target file EXISTS on disk and `Scanned` is asserted DIRECTLY. Because the
//     control is a PRESENT contract (Absent false), VerifyContractVerdict's !Scanned
//     config-error branch — which sits inside `if r.Entry.Absent` — is unreachable for it:
//     a missing or unscanned file degrades to Matched false and yields the ordinary
//     "signature not found or mismatched" violation, byte-indistinguishable from a genuine
//     scanned miss. A raised violation is therefore NOT evidence the engine ran; Scanned is.
//
// The synthetic file is written into a t.TempDir() and passed as an ABSOLUTE path, which
// extraction-style relative joining leaves untouched.
func spec072ProbeControlProvesPackIsLive(t *testing.T, root string) {
	t.Helper()
	controlFile := filepath.Join(t.TempDir(), "control.yml")
	if err := os.WriteFile(controlFile, []byte("sources:\n  - id: SRC-CONTROL\n    disposition: control\n"), 0o644); err != nil {
		t.Fatalf("writing control fixture %s: %v", controlFile, err)
	}
	control := gate.ContractEntry{
		File:      controlFile,
		Name:      "spec072ProbeControlSymbol",
		Kind:      "variable",
		Signature: "spec072ProbeControlSymbol[]",
	}

	results, err := produceContractEngineResults(root, []gate.ContractEntry{control})
	if err != nil {
		t.Fatalf("control dispatch through the production contract producer failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("control produced %d engine results, want 1: the contracts pack did not resolve from project root %s, so every absent-violation assertion in this test would be vacuously green", len(results), root)
	}
	result := results[0]
	if !result.Scanned {
		t.Fatalf("control result Scanned=false for %s: ast-grep never opened the file, so the pack dispatch is not actually live and an absent violation proves nothing", controlFile)
	}
	violation, raised := gate.VerifyContractVerdict(result)
	if !raised {
		t.Fatalf("control result raised no verdict for a Go-struct signature probed against a YAML file; the probe path is not live")
	}
	if violation.Rule != gate.StepContractSignature {
		t.Fatalf("control verdict rule = %q, want %q", violation.Rule, gate.StepContractSignature)
	}
}

// TestGate_ContentInventoryContractSignatureProbeClears is CLM-001: gating
// docs/_data/content-inventory.yml produces no contract_signature failure for
// legacy_content_disposition_inventory.
func TestGate_ContentInventoryContractSignatureProbeClears(t *testing.T) {
	root := repoRoot(t)
	spec072ProbeControlProvesPackIsLive(t, root)

	for _, entry := range spec072ProbeEntriesNamed(spec072ProbeEntries(t, root), "legacy_content_disposition_inventory") {
		t.Errorf("contract extraction still yields a provides entry for %s over %s (signature %q); a non-Go file cannot satisfy a Go ast-grep pattern, so the entry must be a consumes of scripts/verify-public-product-model.sh",
			entry.Name, entry.File, entry.Signature)
	}

	result := spec072ProbeStep(t, root, "docs/_data/content-inventory.yml")
	for _, violation := range spec072ProbeViolationsNaming(result,
		"legacy_content_disposition_inventory", "docs/_data/content-inventory.yml", "sources[]") {
		t.Errorf("gating docs/_data/content-inventory.yml raised %s: file=%s message=%s",
			gate.StepContractSignature, violation.File, violation.Message)
	}
}

// TestGate_ProductModelContractSignatureProbeClears is CLM-003: gating
// docs/_data/product-model.yml produces no contract_signature failure for
// canonical_product_model.
func TestGate_ProductModelContractSignatureProbeClears(t *testing.T) {
	root := repoRoot(t)
	spec072ProbeControlProvesPackIsLive(t, root)

	const proseSignature = "concepts[] + architecture_views[] + boundaries[state|explanation_markdown|continuation|guarantee_denial_markdown]"
	entries := spec072ProbeEntries(t, root)
	for _, entry := range spec072ProbeEntriesNamed(entries, "canonical_product_model") {
		t.Errorf("contract extraction still yields a provides entry for %s over %s (signature %q); a non-Go file cannot satisfy a Go ast-grep pattern, so the entry must be a consumes of scripts/verify-public-product-model.sh",
			entry.Name, entry.File, entry.Signature)
	}
	for _, entry := range entries {
		if entry.Signature == proseSignature {
			t.Errorf("contract extraction still yields the prose product-model signature %q on %s", proseSignature, entry.File)
		}
	}

	result := spec072ProbeStep(t, root, "docs/_data/product-model.yml")
	for _, violation := range spec072ProbeViolationsNaming(result,
		"canonical_product_model", "docs/_data/product-model.yml", proseSignature) {
		t.Errorf("gating docs/_data/product-model.yml raised %s: file=%s message=%s",
			gate.StepContractSignature, violation.File, violation.Message)
	}
}

// TestGate_ArchitectureDiagramContractSignatureProbeClearsOnEdit is CLM-005: gating any of
// the three Mermaid architecture diagrams produces no contract_signature failure for its
// architecture symbol. All three diagrams get a subtest — a single-diagram assertion would
// pass against a two-of-three conversion.
func TestGate_ArchitectureDiagramContractSignatureProbeClearsOnEdit(t *testing.T) {
	root := repoRoot(t)

	spec072ProbeControlProvesPackIsLive(t, root)
	entries := spec072ProbeEntries(t, root)

	diagrams := []struct {
		file   string
		symbol string
	}{
		{"docs/_diagrams/ARCH-001-delivery-lifecycle.mmd", "delivery_lifecycle_architecture"},
		{"docs/_diagrams/ARCH-002-enforcement-loop.mmd", "enforcement_loop_architecture"},
		{"docs/_diagrams/ARCH-003-ownership-boundaries.mmd", "ownership_boundaries_architecture"},
	}
	if len(diagrams) != 3 {
		t.Fatalf("architecture diagram table covers %d diagrams, want all 3: a two-of-three assertion would pass against a partial conversion", len(diagrams))
	}
	for _, diagram := range diagrams {
		t.Run(diagram.symbol, func(t *testing.T) {
			for _, entry := range spec072ProbeEntriesNamed(entries, diagram.symbol) {
				t.Errorf("contract extraction still yields a provides entry for %s over %s (signature %q); a Mermaid diagram cannot satisfy a Go ast-grep pattern, so the entry must be a consumes of scripts/verify-public-product-model.sh",
					entry.Name, entry.File, entry.Signature)
			}
			result := spec072ProbeStep(t, root, diagram.file)
			for _, violation := range spec072ProbeViolationsNaming(result, diagram.symbol, diagram.file) {
				t.Errorf("gating %s raised %s: file=%s message=%s",
					diagram.file, gate.StepContractSignature, violation.File, violation.Message)
			}
		})
	}
}
