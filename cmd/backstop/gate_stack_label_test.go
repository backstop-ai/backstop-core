package main

import (
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/gate"
	"github.com/bmanson/backstop-core/pkg/pack"
)

// TestPolarity_StackLabelFromDeclaredToolchainPacksNotLanguage (CLM-016): the
// traceability stack label for a repo declaring backstop/go-toolchain is derived from
// the declared toolchain-pack SET (membership by declared mechanism engine), and its
// VALUE is the pack's DECLARED language (go-toolchain declares language: go -> "go";
// ISSUE-064 rehomed the value onto manifest.Language, off the "-toolchain" name suffix).
// The rendered fail-loud message names the "go" stack via the CapabilityState.Stack carrier.
func TestPolarity_StackLabelFromDeclaredToolchainPacksNotLanguage(t *testing.T) {
	label := declaredToolchainStackLabel([]*pack.Manifest{spec046ToolchainManifest("backstop/go-toolchain")})
	if label != "go" {
		t.Fatalf("the stack label for a declared backstop/go-toolchain must be the declared-language \"go\", got %q", label)
	}

	// The label flows to the rendered fail-loud message via CapabilityState.Stack
	// (no cfg.Language anywhere in the chain).
	cap := gate.CapabilityState{Present: false, PackOrCommand: "a coverage pack", Stack: label}
	res := gate.PolarityStepResult(gate.StepCoverageThreshold, gate.DimensionCoverage, gate.ClassCapabilityAbsent, nil, cap)
	if len(res.Violations) == 0 || !strings.Contains(res.Violations[0].Message, "go") {
		t.Errorf("the rendered fail-loud message must name the \"go\" stack derived from the declared toolchain pack; got %v", res.Violations)
	}
}

// TestPolarity_PolyglotStackLabelIsUnionNoPrecedence (CLM-017): a repo declaring
// backstop/go-toolchain AND backstop/bun-toolchain yields a SET-valued label naming
// BOTH stacks (merged union), with NO single-pack precedence or overlap winner.
func TestPolarity_PolyglotStackLabelIsUnionNoPrecedence(t *testing.T) {
	label := declaredToolchainStackLabel([]*pack.Manifest{
		spec046ToolchainManifest("backstop/go-toolchain"),
		spec046ToolchainManifest("backstop/bun-toolchain"),
	})
	// Both stacks named, as a sorted SET (no precedence, no single winner).
	if !strings.Contains(label, "go") || !strings.Contains(label, "bun") {
		t.Fatalf("the polyglot label must name BOTH declared stacks, got %q", label)
	}
	if label != "bun, go" {
		t.Fatalf("the polyglot label must be the merged SET (sorted, no precedence), got %q want %q", label, "bun, go")
	}
	// Declaration order must not change the result (set, not precedence).
	reversed := declaredToolchainStackLabel([]*pack.Manifest{
		spec046ToolchainManifest("backstop/bun-toolchain"),
		spec046ToolchainManifest("backstop/go-toolchain"),
	})
	if reversed != label {
		t.Errorf("the polyglot label must be order-independent (no precedence): %q vs %q", reversed, label)
	}
}

// TestPolarity_NoToolchainPackStackLabelUnspecified (CLM-018): a repo declaring NO
// toolchain pack yields "unspecified", driven by the SINGLE authoritative signal —
// the empty declared-language set (membership by declared mechanism engine) — NOT by
// SourceClassifier.HasSourceGlobs() (corroborating only, and divergent). The divergence
// case is asserted: a mechanism pack shipping NO `classification` source globs has
// HasSourceGlobs()==false yet still labels its declared-language stack.
func TestPolarity_NoToolchainPackStackLabelUnspecified(t *testing.T) {
	// Empty declared toolchain-pack set -> "unspecified" (the authoritative signal).
	if got := declaredToolchainStackLabel(nil); got != "unspecified" {
		t.Fatalf("an empty declared toolchain-pack set must yield \"unspecified\", got %q", got)
	}
	// A declared NON-mechanism pack does not count toward the label set (no mechanism engine).
	nonToolchain := &pack.Manifest{Name: "backstop/go-standards", NormalizedName: "backstop/go-standards"}
	if got := declaredToolchainStackLabel([]*pack.Manifest{nonToolchain}); got != "unspecified" {
		t.Fatalf("a non-mechanism declared pack must not produce a stack label, got %q", got)
	}

	// DIVERGENCE: a mechanism pack with NO classification source globs has a non-empty
	// declared-language set (label "go") but HasSourceGlobs()==false — proving the label
	// keys on the declared-language set, NOT on HasSourceGlobs (corroborating only).
	noGlobs := spec046ToolchainManifest("backstop/go-toolchain") // no Classification block
	if got := declaredToolchainStackLabel([]*pack.Manifest{noGlobs}); got != "go" {
		t.Fatalf("a mechanism pack with no source globs must STILL label its declared-language stack, got %q", got)
	}
	if mergeSourceClassifier([]*pack.Manifest{noGlobs}).HasSourceGlobs() {
		t.Fatal("test premise broken: the stub toolchain pack must declare NO source globs (HasSourceGlobs must be false) to prove the divergence")
	}
}

// TestPolarity_RehomeIntroducesNoForkedGlobClassifier (CLM-019): a source guard
// asserts NO new glob-classifier type is defined in this seed — SPEC-043's single
// gate.SourceClassifier remains the ONLY glob classifier; the sole new symbol is
// declaredToolchainStackLabel (a name-set helper, not a glob matcher).
func TestPolarity_RehomeIntroducesNoForkedGlobClassifier(t *testing.T) {
	src := readFileStr(t, "gate.go")
	// No locally-defined classifier type, and no re-construction of a classifier
	// other than SPEC-043's mergeSourceClassifier / gate.NewSourceClassifier seam.
	for _, banned := range []string{"Classifier struct", "func NewSourceClassifier", "type SourceClassifier"} {
		if strings.Contains(src, banned) {
			t.Errorf("cmd/backstop/gate.go must not fork a glob classifier (%q) — reuse SPEC-043's gate.SourceClassifier (CLM-019)", banned)
		}
	}
	// The sole new label symbol is the name-set helper.
	if !strings.Contains(src, "func declaredToolchainStackLabel") {
		t.Error("declaredToolchainStackLabel (the name-set label helper) must be the only new label symbol (CLM-019)")
	}
}
