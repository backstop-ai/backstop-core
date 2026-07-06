package pack_test

import (
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack"
)

func TestValidateToolConfigTrace_Standalone(t *testing.T) {
	m := makeMinimalManifest()
	m.ToolConfig = []pack.ToolConfigEntry{
		{ID: "golangci", Tool: "golangci-lint", File: ".golangci.yml"},
	}

	errs := pack.ValidateManifest(m, baseTestRegistry())

	requireNoError(t, errs, "CLM-031")
}

func TestValidateToolConfigTrace_Supporting(t *testing.T) {
	m := makeMinimalManifest()
	m.ToolConfig = []pack.ToolConfigEntry{
		{RequiredBy: "demo-rule", Tool: "rego", File: "policy.rego"},
	}

	errs := pack.ValidateManifest(m, baseTestRegistry())

	requireNoError(t, errs, "CLM-032")
}

func TestValidateToolConfigTrace_NeitherIdNorRequiredBy(t *testing.T) {
	m := makeMinimalManifest()
	m.ToolConfig = []pack.ToolConfigEntry{
		{Tool: "rego", File: "policy.rego"},
	}

	errs := pack.ValidateManifest(m, baseTestRegistry())

	requireError(t, errs, "CLM-033")
}
