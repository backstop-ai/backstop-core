package pack_test

import "github.com/bmanson/backstop-core/pkg/pack/engine"

// baseTestRegistry constructs the four generic base engine bindings (semgrep,
// ast-grep, sandbox, config-file) as a test-only engine.Registry, mirroring the
// embedded base-engines pack's inline field_contracts. It is the injected `base`
// the pkg/pack validation tests thread into ValidateManifest / ExpectedLayout after
// ISSUE-027 deleted the baked engine.DefaultRegistry() seed.
//
// A Go engine literal is compliant HERE because this is a *_test.go file: the
// backstop/self no-baked scan excludes test files (the PRODUCTION bake is gone —
// the real built-ins load from packs/base-engines/pack.yml via pkg/baseengines).
// The production validation path receives this same shape by injection from
// baseengines.Registry() at the CLI; pkg/pack never imports the embed loader.
func baseTestRegistry() engine.Registry {
	return engine.Registry{
		"semgrep": {
			Command:   "semgrep --sarif --quiet",
			InputMode: engine.InputModeRuleFlags,
			InputFlag: "--config",
			ScopeKind: engine.ScopeKindFileArgs,
			Category:  engine.EngineCategoryOpinion,
			Provision: &engine.Provision{Tool: "semgrep", Version: "1.96.0"},
			FieldContract: engine.FieldContract{
				Requires: []string{engine.FieldRulePath, engine.FieldStandard},
				Forbids:  []string{engine.FieldCategory, engine.FieldInputScope, engine.FieldValidator},
			},
		},
		"ast-grep": {
			Command:   "ast-grep scan --json",
			InputMode: engine.InputModeConfigFile,
			InputFlag: "--config",
			ScopeKind: engine.ScopeKindFileArgs,
			Convert:   "ast-grep/to-sarif.sh",
			Category:  engine.EngineCategoryOpinion,
			Provision: &engine.Provision{Tool: "ast-grep", Version: "0.43.0"},
			FieldContract: engine.FieldContract{
				Requires: []string{engine.FieldRulePath},
				Forbids:  []string{engine.FieldCategory, engine.FieldInputScope, engine.FieldValidator},
			},
		},
		"sandbox": {
			Command:   "",
			InputMode: engine.InputModeNone,
			ScopeKind: engine.ScopeKindFileArgs,
			FieldContract: engine.FieldContract{
				Requires: []string{engine.FieldValidator, engine.FieldInputScope, engine.FieldCategory},
				Forbids:  []string{engine.FieldRulePath},
			},
		},
		"config-file": {
			Command:   "",
			InputMode: engine.InputModeConfigFile,
			InputFlag: "--config",
			ScopeKind: engine.ScopeKindProjectWide,
			FieldContract: engine.FieldContract{
				Forbids: []string{engine.FieldRulePath, engine.FieldCategory, engine.FieldInputScope, engine.FieldValidator},
			},
		},
	}
}
