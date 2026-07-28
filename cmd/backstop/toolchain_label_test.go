package main

import (
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack"
	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
)

// toolchain_label_test.go pins the ISSUE-064 REQ-003 rehome of the cosmetic toolchain
// stack label off the "-toolchain" name suffix and onto declaration: membership by
// declaresToolchainMechanism, value by manifest.Language. The isToolchainPack name-suffix
// helper is deleted (absence contract).

// mechanismManifest builds a declared-pack manifest with an enforcement-mechanism (build)
// engine and a declared language — the shape declaredToolchainStackLabel reads BY
// DECLARATION, independent of the pack name.
func mechanismManifest(normalizedName, language string) *pack.Manifest {
	return &pack.Manifest{
		Name:           normalizedName,
		NormalizedName: normalizedName,
		Language:       language,
		Engines: map[string]pack.EngineSpec{
			"build": {Binding: engine.EngineBinding{GateType: engine.GateTypeBuild}},
		},
	}
}

// TestToolchainStackLabel_ByDeclaredLanguageNotName (CLM-004): the label VALUE is the
// pack's DECLARED manifest.Language, not the name with a "-toolchain" suffix stripped. A
// mechanism pack named acme/rust-tooling (NO "-toolchain" suffix to strip) declaring
// language: rust labels "rust"; and when the declared language DIVERGES from the name, the
// declared language wins — proving the value is language-derived, not name-derived.
func TestToolchainStackLabel_ByDeclaredLanguageNotName(t *testing.T) {
	m := mechanismManifest("acme/rust-tooling", "rust")
	if got := declaredToolchainStackLabel([]*pack.Manifest{m}); got != "rust" {
		t.Fatalf("the label must be the DECLARED language %q, not name-derived; got %q", "rust", got)
	}

	// Divergence proof: a pack whose name would name-strip to "go" but declares
	// language: python labels "python" — the value follows the declaration, not the name.
	diverged := mechanismManifest("backstop/go-toolchain", "python")
	if got := declaredToolchainStackLabel([]*pack.Manifest{diverged}); got != "python" {
		t.Fatalf("the label must follow the DECLARED language even when it diverges from the name; got %q, want python", got)
	}

	// A mechanism pack that declares NO language contributes no token (empty -> unspecified).
	noLang := mechanismManifest("acme/quiet-tooling", "")
	if got := declaredToolchainStackLabel([]*pack.Manifest{noLang}); got != "unspecified" {
		t.Fatalf("a mechanism pack declaring no language must contribute no token (unspecified); got %q", got)
	}
}

// TestToolchainStackLabel_MembershipByMechanismDeclaration (CLM-005): membership in the
// label set is determined by declaresToolchainMechanism — the same by-declaration
// primitive countToolchainPacks uses — NOT the "-toolchain" name suffix. A "-toolchain"-
// named pack with NO mechanism engine contributes nothing; a mechanism pack NOT named
// "*-toolchain" contributes its declared language.
func TestToolchainStackLabel_MembershipByMechanismDeclaration(t *testing.T) {
	// A "-toolchain"-NAMED pack with NO mechanism engine: excluded (contributes nothing),
	// even though the old suffix convention would have counted it.
	namedNoMech := &pack.Manifest{
		Name:           "backstop/faux-toolchain",
		NormalizedName: "backstop/faux-toolchain",
		Language:       "faux",
	}
	if got := declaredToolchainStackLabel([]*pack.Manifest{namedNoMech}); got != "unspecified" {
		t.Fatalf("a -toolchain-NAMED pack with no mechanism engine must NOT label (membership is by declaration); got %q", got)
	}

	// A mechanism pack NOT named "*-toolchain": included, labels its declared language.
	mechNotNamed := mechanismManifest("acme/kotlin-quality", "kotlin")
	if got := declaredToolchainStackLabel([]*pack.Manifest{mechNotNamed}); got != "kotlin" {
		t.Fatalf("a mechanism pack not named *-toolchain must label its declared language; got %q, want kotlin", got)
	}

	// Together: only the mechanism-declaring pack contributes — membership is decided by
	// the declared engine, not the name suffix.
	if got := declaredToolchainStackLabel([]*pack.Manifest{namedNoMech, mechNotNamed}); got != "kotlin" {
		t.Fatalf("only the mechanism-declaring pack contributes to the label; got %q, want kotlin", got)
	}
}

// TestIsToolchainPackRemoved (CLM-006, kind: absence): the isToolchainPack "-toolchain"
// name-suffix helper — and its B6 waiver comment — is DELETED from cmd/backstop. No
// declaration and no caller survives.
func TestIsToolchainPackRemoved(t *testing.T) {
	code := stripGoLineComments(readFileStr(t, "gate.go"))
	if strings.Contains(code, "func isToolchainPack") {
		t.Error("isToolchainPack must be REMOVED from cmd/backstop (CLM-006); it is still declared")
	}
	if strings.Contains(code, "isToolchainPack(") {
		t.Error("no caller of isToolchainPack may remain after the rehome (CLM-006)")
	}
	// The B6 waiver that justified the name-suffix helper must be gone with it. Check the
	// RAW source (comments intact) for the waiver marker.
	raw := readFileStr(t, "gate.go")
	if strings.Contains(raw, "no-pack-name-keyed-capability:false-positive") {
		t.Error("the isToolchainPack B6 waiver comment must be REMOVED with the helper (CLM-006)")
	}
}
