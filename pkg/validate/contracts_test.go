package validate_test

import (
	"testing"

	"github.com/bmanson/backstop-core/pkg/artifact"
	"github.com/bmanson/backstop-core/pkg/validate"
)

// These tests use the spec validator since contracts are required on specs always.
// The shared validateContracts function is tested through both spec and issue validators.

func specWithContracts(contracts interface{}) *artifact.ParsedArtifact {
	art := validSpecArtifact()
	art.Frontmatter["contracts"] = contracts
	return art
}

// --- Contracts required ---

func TestContracts_MissingFromSpec(t *testing.T) {
	art := validSpecArtifact()
	delete(art.Frontmatter, "contracts")
	result := validate.ValidateSpec(art, specSchema())
	assertHasViolation(t, result, "spec/contracts-required")
}

func TestContracts_NotAnArray(t *testing.T) {
	art := specWithContracts("not-array")
	result := validate.ValidateSpec(art, specSchema())
	assertHasViolation(t, result, "spec/contracts-required")
}

func TestContracts_EmptyArray(t *testing.T) {
	art := specWithContracts([]interface{}{})
	result := validate.ValidateSpec(art, specSchema())
	assertHasViolation(t, result, "spec/contracts-empty")
}

// --- Contract entry validation ---

func TestContracts_EntryNotMap(t *testing.T) {
	art := specWithContracts([]interface{}{"not-a-map"})
	result := validate.ValidateSpec(art, specSchema())
	assertHasViolation(t, result, "spec/contract-format")
}

func TestContracts_MissingFile(t *testing.T) {
	art := specWithContracts([]interface{}{
		map[string]interface{}{
			"provides": []interface{}{
				map[string]interface{}{"name": "Foo", "kind": "function", "signature": "func Foo()"},
			},
		},
	})
	result := validate.ValidateSpec(art, specSchema())
	assertHasViolation(t, result, "spec/contract-file-required")
}

func TestContracts_EmptyFile(t *testing.T) {
	art := specWithContracts([]interface{}{
		map[string]interface{}{
			"file": "",
			"provides": []interface{}{
				map[string]interface{}{"name": "Foo", "kind": "function", "signature": "func Foo()"},
			},
		},
	})
	result := validate.ValidateSpec(art, specSchema())
	assertHasViolation(t, result, "spec/contract-file-required")
}

func TestContracts_DuplicateFile(t *testing.T) {
	art := specWithContracts([]interface{}{
		map[string]interface{}{
			"file": "pkg/foo.go",
			"provides": []interface{}{
				map[string]interface{}{"name": "Foo", "kind": "function", "signature": "func Foo()"},
			},
		},
		map[string]interface{}{
			"file": "pkg/foo.go",
			"provides": []interface{}{
				map[string]interface{}{"name": "Bar", "kind": "function", "signature": "func Bar()"},
			},
		},
	})
	result := validate.ValidateSpec(art, specSchema())
	assertHasViolation(t, result, "spec/contract-file-duplicate")
}

func TestContracts_NoProvidesOrConsumes(t *testing.T) {
	art := specWithContracts([]interface{}{
		map[string]interface{}{"file": "pkg/foo.go"},
	})
	result := validate.ValidateSpec(art, specSchema())
	assertHasViolation(t, result, "spec/contract-empty")
}

// --- Provides validation ---

func TestContracts_ProvidesNotArray(t *testing.T) {
	art := specWithContracts([]interface{}{
		map[string]interface{}{
			"file":     "pkg/foo.go",
			"provides": "not-array",
		},
	})
	result := validate.ValidateSpec(art, specSchema())
	assertHasViolation(t, result, "spec/contract-provides-format")
}

func TestContracts_ProvidesEntryNotMap(t *testing.T) {
	art := specWithContracts([]interface{}{
		map[string]interface{}{
			"file":     "pkg/foo.go",
			"provides": []interface{}{"not-a-map"},
		},
	})
	result := validate.ValidateSpec(art, specSchema())
	assertHasViolation(t, result, "spec/provides-format")
}

func TestContracts_ProvidesMissingName(t *testing.T) {
	art := specWithContracts([]interface{}{
		map[string]interface{}{
			"file": "pkg/foo.go",
			"provides": []interface{}{
				map[string]interface{}{"kind": "function", "signature": "func Foo()"},
			},
		},
	})
	result := validate.ValidateSpec(art, specSchema())
	assertHasViolation(t, result, "spec/provides-name-required")
}

func TestContracts_ProvidesMissingKind(t *testing.T) {
	art := specWithContracts([]interface{}{
		map[string]interface{}{
			"file": "pkg/foo.go",
			"provides": []interface{}{
				map[string]interface{}{"name": "Foo", "signature": "func Foo()"},
			},
		},
	})
	result := validate.ValidateSpec(art, specSchema())
	assertHasViolation(t, result, "spec/provides-kind-required")
}

func TestContracts_ProvidesBadKind(t *testing.T) {
	art := specWithContracts([]interface{}{
		map[string]interface{}{
			"file": "pkg/foo.go",
			"provides": []interface{}{
				map[string]interface{}{"name": "Foo", "kind": "class", "signature": "func Foo()"},
			},
		},
	})
	result := validate.ValidateSpec(art, specSchema())
	assertHasViolation(t, result, "spec/provides-kind-enum")
}

func TestContracts_ProvidesMissingSignature(t *testing.T) {
	art := specWithContracts([]interface{}{
		map[string]interface{}{
			"file": "pkg/foo.go",
			"provides": []interface{}{
				map[string]interface{}{"name": "Foo", "kind": "function"},
			},
		},
	})
	result := validate.ValidateSpec(art, specSchema())
	assertHasViolation(t, result, "spec/provides-signature-required")
}

func TestContracts_ProvidesAllKindsValid(t *testing.T) {
	kinds := []string{"function", "type", "interface", "method", "constant", "variable"}
	for _, kind := range kinds {
		art := specWithContracts([]interface{}{
			map[string]interface{}{
				"file": "pkg/foo.go",
				"provides": []interface{}{
					map[string]interface{}{"name": "Foo", "kind": kind, "signature": "sig"},
				},
			},
		})
		result := validate.ValidateSpec(art, specSchema())
		assertNoViolationRule(t, result, "spec/provides-kind-enum")
	}
}

// --- Consumes validation ---

func TestContracts_ConsumesNotArray(t *testing.T) {
	art := specWithContracts([]interface{}{
		map[string]interface{}{
			"file":     "pkg/foo.go",
			"consumes": "not-array",
		},
	})
	result := validate.ValidateSpec(art, specSchema())
	assertHasViolation(t, result, "spec/contract-consumes-format")
}

func TestContracts_ConsumesEntryNotMap(t *testing.T) {
	art := specWithContracts([]interface{}{
		map[string]interface{}{
			"file":     "pkg/foo.go",
			"consumes": []interface{}{"not-a-map"},
		},
	})
	result := validate.ValidateSpec(art, specSchema())
	assertHasViolation(t, result, "spec/consumes-format")
}

func TestContracts_ConsumesMissingSource(t *testing.T) {
	art := specWithContracts([]interface{}{
		map[string]interface{}{
			"file": "pkg/foo.go",
			"consumes": []interface{}{
				map[string]interface{}{"name": "Bar", "kind": "type"},
			},
		},
	})
	result := validate.ValidateSpec(art, specSchema())
	assertHasViolation(t, result, "spec/consumes-source-required")
}

func TestContracts_ConsumesMissingName(t *testing.T) {
	art := specWithContracts([]interface{}{
		map[string]interface{}{
			"file": "pkg/foo.go",
			"consumes": []interface{}{
				map[string]interface{}{"source": "pkg/bar.go", "kind": "type"},
			},
		},
	})
	result := validate.ValidateSpec(art, specSchema())
	assertHasViolation(t, result, "spec/consumes-name-required")
}

func TestContracts_ConsumesMissingKind(t *testing.T) {
	art := specWithContracts([]interface{}{
		map[string]interface{}{
			"file": "pkg/foo.go",
			"consumes": []interface{}{
				map[string]interface{}{"source": "pkg/bar.go", "name": "Bar"},
			},
		},
	})
	result := validate.ValidateSpec(art, specSchema())
	assertHasViolation(t, result, "spec/consumes-kind-required")
}

func TestContracts_ConsumesBadKind(t *testing.T) {
	art := specWithContracts([]interface{}{
		map[string]interface{}{
			"file": "pkg/foo.go",
			"consumes": []interface{}{
				map[string]interface{}{"source": "pkg/bar.go", "name": "Bar", "kind": "module"},
			},
		},
	})
	result := validate.ValidateSpec(art, specSchema())
	assertHasViolation(t, result, "spec/consumes-kind-enum")
}

// --- Empty-string edge cases ---

func TestContracts_ProvidesEmptyName(t *testing.T) {
	art := specWithContracts([]interface{}{
		map[string]interface{}{
			"file": "pkg/foo.go",
			"provides": []interface{}{
				map[string]interface{}{"name": "", "kind": "function", "signature": "func Foo()"},
			},
		},
	})
	result := validate.ValidateSpec(art, specSchema())
	assertHasViolation(t, result, "spec/provides-name-required")
}

func TestContracts_ProvidesEmptySignature(t *testing.T) {
	art := specWithContracts([]interface{}{
		map[string]interface{}{
			"file": "pkg/foo.go",
			"provides": []interface{}{
				map[string]interface{}{"name": "Foo", "kind": "function", "signature": "  "},
			},
		},
	})
	result := validate.ValidateSpec(art, specSchema())
	assertHasViolation(t, result, "spec/provides-signature-required")
}

func TestContracts_ConsumesEmptySource(t *testing.T) {
	art := specWithContracts([]interface{}{
		map[string]interface{}{
			"file": "pkg/foo.go",
			"consumes": []interface{}{
				map[string]interface{}{"source": "", "name": "Bar", "kind": "type"},
			},
		},
	})
	result := validate.ValidateSpec(art, specSchema())
	assertHasViolation(t, result, "spec/consumes-source-required")
}

func TestContracts_ConsumesEmptyName(t *testing.T) {
	art := specWithContracts([]interface{}{
		map[string]interface{}{
			"file": "pkg/foo.go",
			"consumes": []interface{}{
				map[string]interface{}{"source": "pkg/bar.go", "name": "  ", "kind": "type"},
			},
		},
	})
	result := validate.ValidateSpec(art, specSchema())
	assertHasViolation(t, result, "spec/consumes-name-required")
}

// --- Consumes-only contract is valid ---

func TestContracts_ConsumesOnlyValid(t *testing.T) {
	art := specWithContracts([]interface{}{
		map[string]interface{}{
			"file": "pkg/foo.go",
			"consumes": []interface{}{
				map[string]interface{}{"source": "pkg/bar.go", "name": "Bar", "kind": "type"},
			},
		},
	})
	result := validate.ValidateSpec(art, specSchema())
	assertNoViolationRule(t, result, "spec/contract-empty")
}

// --- Issue contracts enforced from ready onward ---

func TestContracts_IssueOpen_NotRequired(t *testing.T) {
	art := validIssueArtifact()
	result := validate.ValidateIssue(art, issueSchema())
	assertNoViolationRule(t, result, "issue/contracts-required")
}

func TestContracts_IssueReady_Required(t *testing.T) {
	art := validClosedIssueArtifact()
	art.Frontmatter["issue"].(map[string]interface{})["status"] = "ready"
	delete(art.Frontmatter["issue"].(map[string]interface{}), "closed")
	delete(art.Frontmatter, "contracts")
	result := validate.ValidateIssue(art, issueSchema())
	assertHasViolation(t, result, "issue/contracts-required")
}

func TestContracts_IssueClosed_Valid(t *testing.T) {
	art := validClosedIssueArtifact()
	result := validate.ValidateIssue(art, issueSchema())
	assertNoViolationRule(t, result, "issue/contracts-required")
}
