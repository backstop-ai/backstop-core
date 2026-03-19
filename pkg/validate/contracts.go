package validate

import (
	"fmt"
	"strings"
)

var contractKinds = map[string]bool{
	"function": true, "type": true, "interface": true,
	"method": true, "constant": true, "variable": true,
}

// contractEntry represents a single file's provides/consumes contract.
type contractEntry struct {
	file     string
	provides []contractItem
	consumes []contractItem
}

type contractItem struct {
	name      string
	kind      string
	signature string // provides only
	source    string // consumes only
}

// validateContracts checks the contracts array for well-formed entries.
// Each contract must specify a file and at least one provides or consumes entry.
// rulePrefix should be "spec" or "issue" for violation rule naming.
func validateContracts(fm map[string]interface{}, filename string, rulePrefix string) []Violation {
	var violations []Violation

	contractsVal, ok := fm["contracts"]
	if !ok {
		violations = append(violations, Violation{
			Rule:     rulePrefix + "/contracts-required",
			File:     filename,
			Message:  "contracts array is missing from frontmatter",
			Severity: "error",
		})
		return violations
	}

	contracts, ok := contractsVal.([]interface{})
	if !ok {
		violations = append(violations, Violation{
			Rule:     rulePrefix + "/contracts-required",
			File:     filename,
			Message:  "contracts is not a valid array",
			Severity: "error",
		})
		return violations
	}

	if len(contracts) == 0 {
		violations = append(violations, Violation{
			Rule:     rulePrefix + "/contracts-empty",
			File:     filename,
			Message:  "contracts array must contain at least one contract",
			Severity: "error",
		})
		return violations
	}

	seenFiles := make(map[string]bool)
	for i, item := range contracts {
		contract, ok := item.(map[string]interface{})
		if !ok {
			violations = append(violations, Violation{
				Rule:     rulePrefix + "/contract-format",
				File:     filename,
				Message:  fmt.Sprintf("contracts[%d] is not a valid map", i),
				Severity: "error",
			})
			continue
		}

		label := fmt.Sprintf("contracts[%d]", i)

		// file (required)
		fileVal, hasFile := contract["file"]
		contractFile := ""
		if !hasFile {
			violations = append(violations, Violation{
				Rule:     rulePrefix + "/contract-file-required",
				File:     filename,
				Message:  fmt.Sprintf("%s is missing 'file'", label),
				Severity: "error",
			})
		} else if f, ok := fileVal.(string); !ok || strings.TrimSpace(f) == "" {
			violations = append(violations, Violation{
				Rule:     rulePrefix + "/contract-file-required",
				File:     filename,
				Message:  fmt.Sprintf("%s 'file' is empty", label),
				Severity: "error",
			})
		} else {
			contractFile = f
			if seenFiles[f] {
				violations = append(violations, Violation{
					Rule:     rulePrefix + "/contract-file-duplicate",
					File:     filename,
					Message:  fmt.Sprintf("duplicate contract file '%s'", f),
					Severity: "error",
				})
			} else {
				seenFiles[f] = true
			}
		}

		// provides and consumes
		hasProvides := false
		hasConsumes := false

		if pVal, ok := contract["provides"]; ok {
			provides, ok := pVal.([]interface{})
			if !ok {
				violations = append(violations, Violation{
					Rule:     rulePrefix + "/contract-provides-format",
					File:     filename,
					Message:  fmt.Sprintf("%s 'provides' is not a valid array", label),
					Severity: "error",
				})
			} else {
				hasProvides = len(provides) > 0
				for j, p := range provides {
					violations = append(violations,
						validateProvidesItem(p, filename, rulePrefix, fmt.Sprintf("%s.provides[%d]", label, j), contractFile)...)
				}
			}
		}

		if cVal, ok := contract["consumes"]; ok {
			consumes, ok := cVal.([]interface{})
			if !ok {
				violations = append(violations, Violation{
					Rule:     rulePrefix + "/contract-consumes-format",
					File:     filename,
					Message:  fmt.Sprintf("%s 'consumes' is not a valid array", label),
					Severity: "error",
				})
			} else {
				hasConsumes = len(consumes) > 0
				for j, c := range consumes {
					violations = append(violations,
						validateConsumesItem(c, filename, rulePrefix, fmt.Sprintf("%s.consumes[%d]", label, j))...)
				}
			}
		}

		if !hasProvides && !hasConsumes {
			violations = append(violations, Violation{
				Rule:     rulePrefix + "/contract-empty",
				File:     filename,
				Message:  fmt.Sprintf("%s must have at least one provides or consumes entry", label),
				Severity: "error",
			})
		}
	}

	return violations
}

// validateProvidesItem checks a single provides entry: name, kind, signature.
func validateProvidesItem(item interface{}, filename string, rulePrefix string, label string, contractFile string) []Violation {
	var violations []Violation

	p, ok := item.(map[string]interface{})
	if !ok {
		violations = append(violations, Violation{
			Rule:     rulePrefix + "/provides-format",
			File:     filename,
			Message:  fmt.Sprintf("%s is not a valid map", label),
			Severity: "error",
		})
		return violations
	}

	// name
	if name, ok := p["name"]; !ok {
		violations = append(violations, Violation{
			Rule:     rulePrefix + "/provides-name-required",
			File:     filename,
			Message:  fmt.Sprintf("%s is missing 'name'", label),
			Severity: "error",
		})
	} else if n, ok := name.(string); !ok || strings.TrimSpace(n) == "" {
		violations = append(violations, Violation{
			Rule:     rulePrefix + "/provides-name-required",
			File:     filename,
			Message:  fmt.Sprintf("%s 'name' is empty", label),
			Severity: "error",
		})
	}

	// kind
	if kindVal, ok := p["kind"]; !ok {
		violations = append(violations, Violation{
			Rule:     rulePrefix + "/provides-kind-required",
			File:     filename,
			Message:  fmt.Sprintf("%s is missing 'kind'", label),
			Severity: "error",
		})
	} else if k, ok := kindVal.(string); !ok || !contractKinds[k] {
		violations = append(violations, Violation{
			Rule:     rulePrefix + "/provides-kind-enum",
			File:     filename,
			Message:  fmt.Sprintf("%s kind '%v' is not valid (allowed: function, type, interface, method, constant, variable)", label, kindVal),
			Severity: "error",
		})
	}

	// signature (required for provides)
	if sigVal, ok := p["signature"]; !ok {
		violations = append(violations, Violation{
			Rule:     rulePrefix + "/provides-signature-required",
			File:     filename,
			Message:  fmt.Sprintf("%s is missing 'signature'", label),
			Severity: "error",
		})
	} else if s, ok := sigVal.(string); !ok || strings.TrimSpace(s) == "" {
		violations = append(violations, Violation{
			Rule:     rulePrefix + "/provides-signature-required",
			File:     filename,
			Message:  fmt.Sprintf("%s 'signature' is empty", label),
			Severity: "error",
		})
	}

	return violations
}

// validateConsumesItem checks a single consumes entry: source, name, kind.
func validateConsumesItem(item interface{}, filename string, rulePrefix string, label string) []Violation {
	var violations []Violation

	c, ok := item.(map[string]interface{})
	if !ok {
		violations = append(violations, Violation{
			Rule:     rulePrefix + "/consumes-format",
			File:     filename,
			Message:  fmt.Sprintf("%s is not a valid map", label),
			Severity: "error",
		})
		return violations
	}

	// source
	if src, ok := c["source"]; !ok {
		violations = append(violations, Violation{
			Rule:     rulePrefix + "/consumes-source-required",
			File:     filename,
			Message:  fmt.Sprintf("%s is missing 'source'", label),
			Severity: "error",
		})
	} else if s, ok := src.(string); !ok || strings.TrimSpace(s) == "" {
		violations = append(violations, Violation{
			Rule:     rulePrefix + "/consumes-source-required",
			File:     filename,
			Message:  fmt.Sprintf("%s 'source' is empty", label),
			Severity: "error",
		})
	}

	// name
	if name, ok := c["name"]; !ok {
		violations = append(violations, Violation{
			Rule:     rulePrefix + "/consumes-name-required",
			File:     filename,
			Message:  fmt.Sprintf("%s is missing 'name'", label),
			Severity: "error",
		})
	} else if n, ok := name.(string); !ok || strings.TrimSpace(n) == "" {
		violations = append(violations, Violation{
			Rule:     rulePrefix + "/consumes-name-required",
			File:     filename,
			Message:  fmt.Sprintf("%s 'name' is empty", label),
			Severity: "error",
		})
	}

	// kind
	if kindVal, ok := c["kind"]; !ok {
		violations = append(violations, Violation{
			Rule:     rulePrefix + "/consumes-kind-required",
			File:     filename,
			Message:  fmt.Sprintf("%s is missing 'kind'", label),
			Severity: "error",
		})
	} else if k, ok := kindVal.(string); !ok || !contractKinds[k] {
		violations = append(violations, Violation{
			Rule:     rulePrefix + "/consumes-kind-enum",
			File:     filename,
			Message:  fmt.Sprintf("%s kind '%v' is not valid (allowed: function, type, interface, method, constant, variable)", label, kindVal),
			Severity: "error",
		})
	}

	return violations
}
