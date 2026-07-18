package pack_test

import (
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack"
)

// TestRunGroup_FieldParsesFromEngineSpec asserts a pack.yml declaring
// `run_group: <key>` on an engine parses through parseEngineSpec onto
// engine.EngineBinding.RunGroup — the opaque declared shared-run key core dedupes
// runs by (ISSUE-068 Option C, CLM-001). Without the field being threaded through
// parseEngineSpec the non-strict YAML decode silently drops the key (the same
// drop-hazard the Producer field guards against), so the binding would carry an
// empty RunGroup and the dedup would never fire.
func TestRunGroup_FieldParsesFromEngineSpec(t *testing.T) {
	body := enginesPackYAML(`engines:
  acme-test:
    command: acme-suite run
    input_mode: none
    scope_kind: project-wide
    category: mechanism
    gate_type: test
    run_group: acme-suite`)

	m, err := pack.ParseManifestFile(writePackYAML(t, body))
	if err != nil {
		t.Fatalf("parse engines block with run_group: %v", err)
	}

	spec, ok := m.Engines["acme-test"]
	if !ok {
		t.Fatalf("engine acme-test not parsed into Engines map; got %#v", m.Engines)
	}
	if got := spec.Binding.RunGroup; got != "acme-suite" {
		t.Errorf("RunGroup = %q, want %q — the declared run_group key was dropped by the non-strict decode", got, "acme-suite")
	}
}

// TestRunGroup_AbsentKeyLeavesBindingEmpty asserts an engine that declares NO
// run_group parses with an empty RunGroup — the SAFE DEFAULT (ISSUE-068 CLM-001):
// no declared key => unchanged two-run behavior, no dedup. An accidental non-empty
// default would silently start memoizing runs for every separate-build toolchain.
func TestRunGroup_AbsentKeyLeavesBindingEmpty(t *testing.T) {
	body := enginesPackYAML(`engines:
  acme-coverage:
    command: acme-suite run --coverage
    input_mode: none
    scope_kind: project-wide
    category: mechanism
    gate_type: coverage
    convert: acme-coverage/to-records.sh`)

	m, err := pack.ParseManifestFile(writePackYAML(t, body))
	if err != nil {
		t.Fatalf("parse engines block without run_group: %v", err)
	}

	spec, ok := m.Engines["acme-coverage"]
	if !ok {
		t.Fatalf("engine acme-coverage not parsed into Engines map; got %#v", m.Engines)
	}
	if got := spec.Binding.RunGroup; got != "" {
		t.Errorf("RunGroup = %q, want empty — an absent run_group must leave the binding empty (safe default)", got)
	}
}
