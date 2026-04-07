package gate

import (
	"context"
	"testing"
)

// --- Baseline comparison tests ---

// TestGate_Baseline_SkippedWhenNoFile verifies baseline step reports skipped
// when no baseline file exists.
func TestGate_Baseline_SkippedWhenNoFile(t *testing.T) {
	step := StepBaselineComparisonFunc()
	result := step(context.Background())
	if result.StepName != StepBaselineComparison {
		t.Errorf("expected step_name %q, got %q", StepBaselineComparison, result.StepName)
	}
	if result.Status != "skipped" {
		t.Errorf("expected status %q, got %q", "skipped", result.Status)
	}
	// Deferred implementation returns "baseline not implemented" for all cases
	if result.Reason == "" {
		t.Error("expected non-empty reason for skipped step")
	}
}

// TestGate_Baseline_SkippedWhenNotImplemented verifies baseline step reports
// skipped when subsystem is not implemented.
func TestGate_Baseline_SkippedWhenNotImplemented(t *testing.T) {
	step := StepBaselineComparisonFunc()
	result := step(context.Background())
	if result.Status != "skipped" {
		t.Errorf("expected status %q, got %q", "skipped", result.Status)
	}
	if result.Reason != "baseline not implemented" {
		t.Errorf("expected reason %q, got %q", "baseline not implemented", result.Reason)
	}
}

// TestGate_Baseline_FailOnNewViolation — deferred: test verifies the interface
// returns skipped for now (when baseline subsystem is implemented, this will
// test actual fail behavior).
func TestGate_Baseline_FailOnNewViolation(t *testing.T) {
	step := StepBaselineComparisonFunc()
	result := step(context.Background())
	// Currently deferred — should be skipped
	if result.Status != "skipped" {
		t.Errorf("expected deferred status %q, got %q", "skipped", result.Status)
	}
}

// TestGate_Baseline_PassWhenClean — deferred: test verifies the interface
// returns skipped for now.
func TestGate_Baseline_PassWhenClean(t *testing.T) {
	step := StepBaselineComparisonFunc()
	result := step(context.Background())
	if result.Status != "skipped" {
		t.Errorf("expected deferred status %q, got %q", "skipped", result.Status)
	}
}

// --- Waiver resolution tests ---

// TestGate_Waiver_SkippedWhenNotImplemented verifies waiver step reports
// skipped when subsystem is not implemented.
func TestGate_Waiver_SkippedWhenNotImplemented(t *testing.T) {
	step := StepWaiverResolutionFunc()
	result := step(context.Background())
	if result.StepName != StepWaiverResolution {
		t.Errorf("expected step_name %q, got %q", StepWaiverResolution, result.StepName)
	}
	if result.Status != "skipped" {
		t.Errorf("expected status %q, got %q", "skipped", result.Status)
	}
	if result.Reason != "waivers not implemented" {
		t.Errorf("expected reason %q, got %q", "waivers not implemented", result.Reason)
	}
}

// --- Ledger integrity tests ---

// TestGate_Ledger_SkippedWhenNoLedger verifies ledger step reports skipped
// when no ledger file exists.
func TestGate_Ledger_SkippedWhenNoLedger(t *testing.T) {
	step := StepLedgerIntegrityFunc()
	result := step(context.Background())
	if result.StepName != StepLedgerIntegrity {
		t.Errorf("expected step_name %q, got %q", StepLedgerIntegrity, result.StepName)
	}
	if result.Status != "skipped" {
		t.Errorf("expected status %q, got %q", "skipped", result.Status)
	}
	if result.Reason == "" {
		t.Error("expected non-empty reason for skipped step")
	}
}

// TestGate_Ledger_SkippedWhenNotImplemented verifies ledger step reports
// skipped when subsystem is not implemented.
func TestGate_Ledger_SkippedWhenNotImplemented(t *testing.T) {
	step := StepLedgerIntegrityFunc()
	result := step(context.Background())
	if result.Status != "skipped" {
		t.Errorf("expected status %q, got %q", "skipped", result.Status)
	}
	if result.Reason != "ledger not implemented" {
		t.Errorf("expected reason %q, got %q", "ledger not implemented", result.Reason)
	}
}

// TestGate_Ledger_FailOnBrokenChain — deferred: test verifies the interface
// returns skipped for now.
func TestGate_Ledger_FailOnBrokenChain(t *testing.T) {
	step := StepLedgerIntegrityFunc()
	result := step(context.Background())
	if result.Status != "skipped" {
		t.Errorf("expected deferred status %q, got %q", "skipped", result.Status)
	}
}

// TestGate_Ledger_PassWhenIntact — deferred: test verifies the interface
// returns skipped for now.
func TestGate_Ledger_PassWhenIntact(t *testing.T) {
	step := StepLedgerIntegrityFunc()
	result := step(context.Background())
	if result.Status != "skipped" {
		t.Errorf("expected deferred status %q, got %q", "skipped", result.Status)
	}
}
