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

// spec072ProbeRequireNoEntryNamed fails when the extracted contract set still carries a
// provides-derived entry for symbol name.
func spec072ProbeRequireNoEntryNamed(t *testing.T, entries []gate.ContractEntry, name string) {
	t.Helper()
	for _, entry := range entries {
		if entry.Name == name {
			t.Errorf("contract extraction still yields a provides entry for %s over %s (signature %q); a non-Go file cannot satisfy a Go ast-grep pattern, so the entry must be declared as a consumes of scripts/verify-public-product-model.sh",
				entry.Name, entry.File, entry.Signature)
		}
	}
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
	return step(context.Background())
}

// spec072ProbeRequireNoContractViolation fails when the step result carries a
// contract_signature violation whose File or Message names any of the given needles.
func spec072ProbeRequireNoContractViolation(t *testing.T, result gate.StepResult, needles ...string) {
	t.Helper()
	if result.ConfigErr {
		t.Fatalf("contract step reported a config error rather than a verdict: %+v", result.Violations)
	}
	for _, violation := range result.Violations {
		if violation.Rule != gate.StepContractSignature {
			continue
		}
		for _, needle := range needles {
			if strings.Contains(violation.Message, needle) || strings.Contains(violation.File, needle) {
				t.Errorf("contract step raised %s naming %q: file=%s message=%s",
					gate.StepContractSignature, needle, violation.File, violation.Message)
				break
			}
		}
	}
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
	spec072ProbeRequireNoEntryNamed(t, spec072ProbeEntries(t, root), "legacy_content_disposition_inventory")
	spec072ProbeRequireNoContractViolation(t,
		spec072ProbeStep(t, root, "docs/_data/content-inventory.yml"),
		"legacy_content_disposition_inventory",
		"docs/_data/content-inventory.yml",
		"sources[]",
	)
}

// TestGate_ProductModelContractSignatureProbeClears is CLM-003: gating
// docs/_data/product-model.yml produces no contract_signature failure for
// canonical_product_model.
func TestGate_ProductModelContractSignatureProbeClears(t *testing.T) {
	root := repoRoot(t)

	spec072ProbeControlProvesPackIsLive(t, root)

	entries := spec072ProbeEntries(t, root)
	spec072ProbeRequireNoEntryNamed(t, entries, "canonical_product_model")
	const proseSignature = "concepts[] + architecture_views[] + boundaries[state|explanation_markdown|continuation|guarantee_denial_markdown]"
	for _, entry := range entries {
		if entry.Signature == proseSignature {
			t.Errorf("contract extraction still yields the prose product-model signature %q on %s", proseSignature, entry.File)
		}
	}

	spec072ProbeRequireNoContractViolation(t,
		spec072ProbeStep(t, root, "docs/_data/product-model.yml"),
		"canonical_product_model",
		"docs/_data/product-model.yml",
		proseSignature,
	)
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
	for _, diagram := range diagrams {
		t.Run(diagram.symbol, func(t *testing.T) {
			spec072ProbeRequireNoEntryNamed(t, entries, diagram.symbol)
			spec072ProbeRequireNoContractViolation(t,
				spec072ProbeStep(t, root, diagram.file),
				diagram.symbol,
				diagram.file,
			)
		})
	}
}
