package pack_test

import (
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack"
)

// TestRunGroup_IncoherentCommandRejected asserts a run-group whose members diverge
// on a run-shaping field (here Command) is a blocking broken-pack error at parse
// time (ISSUE-068 CLM-002/CLM-003). Two engines that declare the SAME run_group are
// promising to share ONE memoized run; if their commands differ the memoized
// payload would be whichever ran first, silently reused by the other — latent
// verdict drift. The error must name the pack and the run_group key. Byte-equality
// on the raw declared field, never normalized.
func TestRunGroup_IncoherentCommandRejected(t *testing.T) {
	body := enginesPackYAML(`engines:
  acme-test:
    command: acme-suite run
    input_mode: none
    scope_kind: project-wide
    category: mechanism
    gate_type: test
    run_group: acme-suite
  acme-coverage:
    command: acme-suite run --coverage
    input_mode: none
    scope_kind: project-wide
    category: mechanism
    gate_type: coverage
    convert: acme-coverage/to-records.sh
    run_group: acme-suite`)

	_, err := pack.ParseManifestFile(writePackYAML(t, body))
	if err == nil {
		t.Fatal("a run-group diverging on command must fail loud at parse time, got nil error")
	}
	if !strings.Contains(err.Error(), "acme-suite") {
		t.Errorf("error must name the run_group key, got: %v", err)
	}
	if !strings.Contains(err.Error(), "acme/engines-demo") {
		t.Errorf("error must name the pack, got: %v", err)
	}
}

// TestRunGroup_MismatchedScopeOrTargetRejected asserts a run-group is ALSO rejected
// when members share an identical Command but diverge on ScopeKind or ProjectTarget
// (ISSUE-068 CLM-002/CLM-003). The actual run depends on scope/target shaping — a
// project-wide member appends its ProjectTarget while a file-args member appends
// diff-scoped file inputs — so identical Command alone does NOT guarantee identical
// runs. Without ScopeKind/ProjectTarget parity whichever engine executes first
// shapes a payload the other wrongly reuses.
func TestRunGroup_MismatchedScopeOrTargetRejected(t *testing.T) {
	// Same command, divergent ProjectTarget.
	targetBody := enginesPackYAML(`engines:
  acme-test:
    command: acme-suite run
    input_mode: none
    scope_kind: project-wide
    category: mechanism
    gate_type: test
    project_target: ./...
    run_group: acme-suite
  acme-coverage:
    command: acme-suite run
    input_mode: none
    scope_kind: project-wide
    category: mechanism
    gate_type: coverage
    convert: acme-coverage/to-records.sh
    project_target: ./cmd/...
    run_group: acme-suite`)

	_, err := pack.ParseManifestFile(writePackYAML(t, targetBody))
	if err == nil {
		t.Fatal("a run-group diverging on project_target must fail loud, got nil error")
	}
	if !strings.Contains(err.Error(), "acme-suite") {
		t.Errorf("project_target divergence error must name the run_group key, got: %v", err)
	}

	// Same command + target, divergent ScopeKind (file-args vs project-wide).
	scopeBody := enginesPackYAML(`engines:
  acme-test:
    command: acme-suite run
    input_mode: none
    scope_kind: project-wide
    category: mechanism
    gate_type: test
    run_group: acme-suite
  acme-coverage:
    command: acme-suite run
    input_mode: none
    scope_kind: file-args
    category: mechanism
    gate_type: coverage
    convert: acme-coverage/to-records.sh
    run_group: acme-suite`)

	_, err = pack.ParseManifestFile(writePackYAML(t, scopeBody))
	if err == nil {
		t.Fatal("a run-group diverging on scope_kind must fail loud, got nil error")
	}
	if !strings.Contains(err.Error(), "acme-suite") {
		t.Errorf("scope_kind divergence error must name the run_group key, got: %v", err)
	}
}

// TestRunGroup_CoherentGroupAccepted asserts a run-group whose members agree on ALL
// run-shaping fields (Command, Producer, StdoutArtifact, ScopeKind, ProjectTarget)
// is accepted (ISSUE-068 CLM-002/CLM-003) — this is the only coherent, run-once use.
// The two engines keep DISTINCT gate_types and converts; they merely share the run.
func TestRunGroup_CoherentGroupAccepted(t *testing.T) {
	body := enginesPackYAML(`engines:
  acme-test:
    command: acme-suite run --coverage --reporter=json
    input_mode: none
    scope_kind: project-wide
    category: mechanism
    gate_type: test
    stdout_artifact: acme-report.json
    convert: acme-test/to-sarif.sh
    run_group: acme-suite
  acme-coverage:
    command: acme-suite run --coverage --reporter=json
    input_mode: none
    scope_kind: project-wide
    category: mechanism
    gate_type: coverage
    stdout_artifact: acme-report.json
    convert: acme-coverage/to-records.sh
    run_group: acme-suite`)

	m, err := pack.ParseManifestFile(writePackYAML(t, body))
	if err != nil {
		t.Fatalf("a fully-parity run-group must be accepted, got: %v", err)
	}
	if got := m.Engines["acme-test"].Binding.RunGroup; got != "acme-suite" {
		t.Errorf("acme-test RunGroup = %q, want acme-suite", got)
	}
	if got := m.Engines["acme-coverage"].Binding.RunGroup; got != "acme-suite" {
		t.Errorf("acme-coverage RunGroup = %q, want acme-suite", got)
	}
}

// TestRunGroup_SingletonGroupIsNoOp asserts a run-group with exactly ONE
// participant is accepted as a documented no-op (ISSUE-068 CLM-002/CLM-003): it
// dedupes nothing and behaves as if the key were unset. Only groups of >1 need the
// coherence parity check, so a lone declared key must never fail-loud.
func TestRunGroup_SingletonGroupIsNoOp(t *testing.T) {
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
		t.Fatalf("a singleton run-group must be accepted as a no-op, got: %v", err)
	}
	if got := m.Engines["acme-test"].Binding.RunGroup; got != "acme-suite" {
		t.Errorf("acme-test RunGroup = %q, want acme-suite", got)
	}
}
