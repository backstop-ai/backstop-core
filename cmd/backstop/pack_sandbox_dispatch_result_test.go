package main

import (
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/packval"
)

func TestPackSandbox_DispatchResultsCarryDeterministicEvidence(t *testing.T) {
	runner := &recordingSandboxRunner{mode: packval.SandboxModeExternal}
	engines, err := dispatchPackEnginesWithEvidence(nil, t.TempDir(), t.TempDir(), nil, emptySarifRunner{}, runner)
	if err != nil {
		t.Fatal(err)
	}
	if engines.Violations == nil || engines.NativeSandboxApplied {
		t.Fatalf("engine dispatch result = %#v", engines)
	}

	coverage, err := dispatchPackCoverageWithEvidence(nil, t.TempDir(), t.TempDir(), nil, emptySarifRunner{}, runner)
	if err != nil {
		t.Fatal(err)
	}
	if coverage.Records == nil || coverage.NativeSandboxApplied {
		t.Fatalf("coverage dispatch result = %#v", coverage)
	}
}
