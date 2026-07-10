package gate

import (
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/waiver"
)

// TestGateWaiver_Prefill_WaivableFindingEmitsToken proves blocking on a WAIVABLE
// finding emits a pre-filled @waiver token carrying that finding's own rule-id
// and a reason-code-defaulted expiry (CLM-059).
func TestGateWaiver_Prefill_WaivableFindingEmitsToken(t *testing.T) {
	v := Violation{Rule: "go-standards/line-length", File: "app.go", Line: 5, Severity: "warning"}
	token, ok := PrefilledWaiverToken(StepPackEngines, v, nil, waiverTestNow)
	if !ok {
		t.Fatal("a blocked waivable finding must yield a pre-filled token")
	}
	if !strings.Contains(token, "@waiver:go-standards/line-length:") {
		t.Errorf("the pre-filled token must carry the finding's own rule-id, got %q", token)
	}
	// The token must carry a concrete future ISO-8601 expiry, not a blank.
	if !strings.Contains(token, "20") {
		t.Errorf("the pre-filled token must carry a defaulted expiry date, got %q", token)
	}
}

// TestGateWaiver_Prefill_NonWaivableFindingNoToken proves a non-waivable finding
// does NOT get a pre-filled token — it cannot be waived (CLM-060).
func TestGateWaiver_Prefill_NonWaivableFindingNoToken(t *testing.T) {
	policy := waiver.NewDeclaredPolicy([]string{"backstop/self/no-baked-language"}, []string{"critical"})
	v := Violation{Rule: "backstop/self/no-baked-language", File: "app.go", Line: 5, Severity: "error"}
	if _, ok := PrefilledWaiverToken(StepPackEngines, v, policy, waiverTestNow); ok {
		t.Fatal("a non-waivable finding must NOT get a pre-filled token")
	}
}

// TestGateWaiver_Prefill_StructuralFindingNoToken proves a structural / non-code
// finding does NOT get a pre-filled token — it is outside the waivable surface
// (CLM-061).
func TestGateWaiver_Prefill_StructuralFindingNoToken(t *testing.T) {
	v := Violation{Rule: "contract_signature", File: "specs/x.spec.md", Line: 1, Severity: "error"}
	if _, ok := PrefilledWaiverToken(StepContractSignature, v, nil, waiverTestNow); ok {
		t.Fatal("a structural finding must NOT get a pre-filled token")
	}
}
