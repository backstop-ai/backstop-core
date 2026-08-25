package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/gate"
)

func TestPackSandbox_HumanOutputReportsModeAndNativeApplication(t *testing.T) {
	result := gate.NewGateResult(nil)
	result.PackSandboxMode = "external"
	result.NativeSandboxApplied = false
	output := gate.FormatHuman(result, true)
	if !strings.Contains(output, "pack sandbox: external (native applied: false)") {
		t.Fatalf("human output lacks sandbox metadata:\n%s", output)
	}
}

func TestPackSandbox_JSONReportsModeAndNativeApplication(t *testing.T) {
	result := gate.NewGateResult(nil)
	result.PackSandboxMode = "external"
	result.NativeSandboxApplied = false
	data, err := gate.FormatJSON(result)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	if document["pack_sandbox_mode"] != "external" {
		t.Fatalf("pack_sandbox_mode = %#v", document["pack_sandbox_mode"])
	}
	if applied, present := document["native_sandbox_applied"]; !present || applied != false {
		t.Fatalf("native_sandbox_applied = %#v, present=%v", applied, present)
	}
}
