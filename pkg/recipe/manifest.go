// Package recipe implements the recipe manifest and the generic applier that
// materializes a pack-declared recipe into a consumer project (SPEC-054). It
// carries ZERO language, framework, or platform knowledge: every path, payload,
// rule, and instruction is DATA read from the pack's recipe.yml.
package recipe

import (
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// The three valid recipe kinds (REQ-009). The kind is the regenerate-vs-one-shot
// switch: scaffolding and implementing regenerate by default (REQ-004), while
// templating is one-shot / consumer-owned (REQ-012).
const (
	KindScaffolding  = "scaffolding"
	KindImplementing = "implementing"
	KindTemplating   = "templating"
)

// The CLOSED op-family allowlist (REQ-002/REQ-007). A `step` op is recognized and
// sequenced here but never executed — BUNDLE-019 owns its executor.
const (
	OpCreate    = "create"
	OpMerge     = "merge"
	OpTransform = "transform"
	OpInsert    = "insert"
	OpStep      = "step"
)

// recipeSemverRe is the strict MAJOR.MINOR.PATCH grammar a recipe version must
// match — no prefix, no prerelease, no build metadata. It mirrors the artifact
// validators' semverRe so a recipe version and an artifact version are the same
// shape.
var recipeSemverRe = regexp.MustCompile(`^\d+\.\d+\.\d+$`) // nosemgrep: go.core.no-global-mutable-state — immutable compiled-regex singleton, package idiom

// RecipeManifest is a parsed recipe.yml. Compat and Variants are OPTIONAL and
// validated STRUCTURALLY only — their apply-time resolution behavior is a later
// spec's, so nothing here selects, resolves, or merges a variant. Enforcement is
// the paired-suite DECLARATION; its activation and scoping are likewise out of
// scope.
type RecipeManifest struct {
	Kind           string           `yaml:"kind"`
	Version        string           `yaml:"version"`
	Params         []ParamSpec      `yaml:"params"`
	Ops            []Op             `yaml:"ops"`
	TransformRules []string         `yaml:"transform_rules"`
	Enforcement    *EnforcementDecl `yaml:"enforcement"`
	Compat         []CompatSelector `yaml:"compat"`
	Variants       []Variant        `yaml:"variants"`
}

// Op is one declared operation. ID is the stable key the SDLC-mediated
// InjectionSites map routes the WHERE by (REQ-003), which is why it must be
// unique and non-empty on the injection-accepting families. Manual is the
// human-actionable fallback emitted VERBATIM when the injection limit is hit
// (REQ-011) — core cannot synthesize it without the language knowledge REQ-006
// forbids, so transform and insert must declare it.
//
// A `step` op carries only its ID and Kind here. Its future payload schema is NOT
// round-tripped by the non-strict decode below (unknown keys are dropped);
// BUNDLE-019, which owns the step executor, extends this contract with them.
type Op struct {
	ID       string `yaml:"id"`
	Kind     string `yaml:"kind"`
	Target   string `yaml:"target"`
	Payload  string `yaml:"payload"`
	Fragment string `yaml:"fragment"`
	Format   string `yaml:"format"`
	Rule     string `yaml:"rule"`
	Anchor   string `yaml:"anchor"`
	Snippet  string `yaml:"snippet"`
	Manual   string `yaml:"manual"`
}

// ParamSpec is one entry in the recipe's declared param schema. It feeds the
// {{ }} substitution and supplies direct mode's defaults.
type ParamSpec struct {
	Name     string `yaml:"name"`
	Required bool   `yaml:"required"`
	Default  string `yaml:"default"`
}

// EnforcementDecl is the recipe's paired enforcement suite DECLARATION: the gate
// rules scoped to the recipe's output. Adoption-gating and static scoping are a
// later spec's; this type only carries the declaration through parse.
type EnforcementDecl struct {
	Rules []string `yaml:"rules"`
}

// CompatSelector is one declared compatibility selector — a generic
// {file, path, range} read against the consumer's environment. Structural only
// here: nothing reads the file or evaluates the range.
type CompatSelector struct {
	File  string `yaml:"file"`
	Path  string `yaml:"path"`
	Range string `yaml:"range"`
}

// Variant is one version-keyed variant of a single logical recipe: a
// {compat range -> ops} pairing. Structural only here — no variant is selected,
// resolved, or merged into the recipe's own ops.
type Variant struct {
	Compat []CompatSelector `yaml:"compat"`
	Ops    []Op             `yaml:"ops"`
}

// ParseRecipeManifest parses and structurally validates recipe.yml bytes.
//
// The decode is deliberately NON-STRICT: unknown keys are dropped rather than
// rejected, so a recipe declaring a reserved `step` payload this spec does not
// yet model parses instead of failing.
//
// Ops keep their DECLARED ORDER — never sorted, deduped by reorder, or map-ified.
// Determinism downstream depends on it.
func ParseRecipeManifest(data []byte) (*RecipeManifest, error) {
	var manifest RecipeManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parse recipe yaml: %w", err)
	}

	if err := validateRecipeHeader(&manifest); err != nil {
		return nil, fmt.Errorf("validate recipe manifest: %w", err)
	}
	if err := validateRecipeOps(&manifest); err != nil {
		return nil, fmt.Errorf("validate recipe ops: %w", err)
	}

	return &manifest, nil
}

// recipeLabel names the recipe by its declared identity so an op-level error
// says WHICH recipe is at fault, not just which op.
func recipeLabel(m *RecipeManifest) string {
	return fmt.Sprintf("recipe (kind %q, version %q)", m.Kind, m.Version)
}

// validateRecipeHeader fail-louds on the recipe-level required fields: version
// present and well-formed semver, kind within the three, ops non-empty.
func validateRecipeHeader(m *RecipeManifest) error {
	if m.Version == "" {
		return fmt.Errorf("recipe is missing 'version' (semver MAJOR.MINOR.PATCH)")
	}
	if !recipeSemverRe.MatchString(m.Version) {
		return fmt.Errorf("recipe 'version' %q must be semver MAJOR.MINOR.PATCH", m.Version)
	}

	switch m.Kind {
	case KindScaffolding, KindImplementing, KindTemplating:
	default:
		return fmt.Errorf("recipe 'kind' %q must be one of %q, %q, %q", m.Kind, KindScaffolding, KindImplementing, KindTemplating)
	}

	if len(m.Ops) == 0 {
		return fmt.Errorf("%s: 'ops' is required and must declare at least one operation", recipeLabel(m))
	}

	return nil
}

// validateRecipeOps runs the op-level cross-checks: unique ids, a non-empty id on
// every injection-accepting op, a non-empty manual on the injection-limit
// families, and a transform rule that the recipe actually declared. Variant ops
// are NOT cross-checked here — variants validate structurally only.
func validateRecipeOps(m *RecipeManifest) error {
	label := recipeLabel(m)

	declaredRules := make(map[string]struct{}, len(m.TransformRules))
	for _, rule := range m.TransformRules {
		declaredRules[rule] = struct{}{}
	}

	seenAt := make(map[string]int, len(m.Ops))
	for i, op := range m.Ops {
		acceptsInjection := op.Kind == OpTransform || op.Kind == OpInsert

		if acceptsInjection && strings.TrimSpace(op.ID) == "" {
			return fmt.Errorf("%s: ops[%d] (kind %q) has an empty id; the op id is the injection-site routing key and must be non-empty", label, i, op.Kind)
		}
		if op.ID != "" {
			if prev, duplicate := seenAt[op.ID]; duplicate {
				return fmt.Errorf("%s: duplicate op id %q declared by ops[%d] and ops[%d]", label, op.ID, prev, i)
			}
			seenAt[op.ID] = i
		}

		if acceptsInjection && strings.TrimSpace(op.Manual) == "" {
			return fmt.Errorf("%s: ops[%d] %q (kind %q) is missing a non-empty 'manual'; the injection-limit families must declare the human-actionable fallback text", label, i, op.ID, op.Kind)
		}

		if op.Kind == OpTransform {
			if _, declared := declaredRules[op.Rule]; !declared {
				return fmt.Errorf("%s: ops[%d] %q declares rule %q, which is not among the recipe's declared transform_rules", label, i, op.ID, op.Rule)
			}
		}
	}

	return nil
}
