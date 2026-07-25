package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/check"
	"github.com/bmanson/backstop-core/pkg/pack"
	"github.com/bmanson/backstop-core/pkg/pack/engine"
	"github.com/bmanson/backstop-core/pkg/recipe"
)

// SPEC-054 CLM-025 — the transform trust gate, tested at the layer where it
// actually lives.
//
// It cannot be tested in pkg/recipe: no type there carries an engine tool or a
// pinned version, so the tool name and the compared version come from the PACK's
// engines: block, which only cmd/backstop can see. The allowlist driving these
// cases is the REAL one (the production resolveTrustedToolAllowlist path) — a
// permissive injected map would prove nothing, and stubbing an allowlist OPEN on
// the dispatch path is precisely the failure this claim guards.
//
// The compared version is the pack.yml `engines.<name>.provision.version`.
// backstop.lock carries NO tool version at all (LockEntry has no such field), so
// nothing here is a "lock pin" assertion.
const (
	recipeE2ERejectPack   = "demo-org/reject-pack"
	recipeE2ERejectID     = "reject"
	recipeE2EMismatchPack = "demo-org/mismatch-pack"
	recipeE2EMismatchID   = "mismatch"
)

// provisionedBinding returns the staged pack's SINGLE provisioned engine binding —
// the same declared datum the CLI selects the transform engine from. Zero or many
// is a fixture defect, not a test condition.
func provisionedBinding(t *testing.T, projectRoot string, packName string) engine.EngineBinding {
	t.Helper()

	packRoot := filepath.Join(projectRoot, ".backstop", "packs", filepath.FromSlash(packName))
	manifest, err := pack.ParseManifestFile(filepath.Join(packRoot, "pack.yml"))
	if err != nil {
		t.Fatalf("parse installed pack %q: %v", packName, err)
	}

	var found []engine.EngineBinding
	for _, spec := range manifest.Engines {
		if spec.Binding.Provision != nil {
			found = append(found, spec.Binding)
		}
	}
	if len(found) != 1 {
		t.Fatalf("pack %q declares %d provisioned engine bindings, want exactly 1", packName, len(found))
	}
	return found[0]
}

// stageTrustGateRun stages the fixture project, writes the CAPTURED before-fixture
// at the recipe's declared target, and returns the ref to drive, the target path and
// its untouched bytes. The captured payload comes from the PASS pack (the only
// recipe shipping a fixture pair); the reject fixtures declare the same target, so
// the byte-untouched assertion is over the same file a successful run would rewrite.
func stageTrustGateRun(t *testing.T, packName string, recipeID string) (string, string, string, []byte) {
	t.Helper()

	projectRoot := stageRecipeE2EProject(t)
	_, passManifest, passDir := stagedRecipe(t, projectRoot, recipeE2EPassPack, recipeE2EPassID)
	before := capturedFixture(t, passDir, passManifest.Ops[0].Target, recipeStageBefore)

	ref, manifest, _ := stagedRecipe(t, projectRoot, packName, recipeID)
	targetPath := stageDeclaredTarget(t, projectRoot, manifest.Ops[0].Target, before)

	return projectRoot, ref, targetPath, before
}

// assertRejectedBeforeAnyCommand runs the CLI over a recipe whose pack declares an
// engine the trust gate must refuse, and pins the whole rejection contract: a
// *check.ConfigError (the exit-2 shape) naming the offending tool, a target left
// byte-identical (no command ran), and no adoption recorded.
func assertRejectedBeforeAnyCommand(t *testing.T, packName string, recipeID string, wantInMessage []string) {
	t.Helper()

	projectRoot, ref, targetPath, before := stageTrustGateRun(t, packName, recipeID)

	output, err := runRecipeApplyCLI(t, projectRoot, ref)
	if err == nil {
		t.Fatalf("backstop recipe apply %s succeeded; the trust gate must reject the pack's declared engine\noutput:\n%s", ref, output)
	}

	var configErr *check.ConfigError
	if !errors.As(err, &configErr) {
		t.Fatalf("error is %T (%v), want a *check.ConfigError — the exit-2 shape the engine trust gate returns", err, err)
	}
	for _, want := range wantInMessage {
		if !strings.Contains(configErr.Message, want) {
			t.Errorf("rejection message %q does not name %q", configErr.Message, want)
		}
	}

	after, readErr := os.ReadFile(targetPath)
	if readErr != nil {
		t.Fatalf("read declared target %q: %v", targetPath, readErr)
	}
	if !bytes.Equal(after, before) {
		t.Errorf("the declared target was modified despite the rejection: the gate ran AFTER a command instead of before one\ngot:\n%s", after)
	}

	if _, statErr := os.Stat(filepath.Join(projectRoot, recipe.AdoptionRecordName)); statErr == nil {
		t.Errorf("a rejected run wrote an adoption record; nothing was adopted")
	}
}

// TestApply_TransformOp_UnallowlistedEngineRejected proves CLM-025 over the shipped
// CLI: an engine whose provisioned tool is absent from the REAL trusted-tool
// allowlist — and, in the second arm, one whose tool IS allowlisted but is declared
// at a version the allowlist does not pin — is rejected as a *check.ConfigError
// BEFORE any command is constructed or run. The third case is the passing twin:
// without it a gate that rejected everything would look identical.
func TestApply_TransformOp_UnallowlistedEngineRejected(t *testing.T) {
	t.Run("un-allowlisted tool", func(t *testing.T) {
		projectRoot := stageRecipeE2EProject(t)
		binding := provisionedBinding(t, projectRoot, recipeE2ERejectPack)
		if _, allowed := engine.TrustedToolAllowlist()[binding.Provision.Tool]; allowed {
			t.Fatalf("fixture defect: %q IS on the trusted-tool allowlist, so this case cannot exercise the rejection", binding.Provision.Tool)
		}

		assertRejectedBeforeAnyCommand(t, recipeE2ERejectPack, recipeE2ERejectID, []string{
			binding.Provision.Tool,
			recipeE2ERejectPack,
		})
	})

	t.Run("allowlisted tool declared at an unpinned version", func(t *testing.T) {
		projectRoot := stageRecipeE2EProject(t)
		binding := provisionedBinding(t, projectRoot, recipeE2EMismatchPack)
		pinned, allowed := engine.TrustedToolAllowlist()[binding.Provision.Tool]
		if !allowed {
			t.Fatalf("fixture defect: %q is NOT on the allowlist, so this case would exercise the first arm, not the version arm", binding.Provision.Tool)
		}
		if binding.Provision.Version == pinned {
			t.Fatalf("fixture defect: %q declares the allowlisted version %q, so there is no mismatch to reject", binding.Provision.Tool, pinned)
		}

		assertRejectedBeforeAnyCommand(t, recipeE2EMismatchPack, recipeE2EMismatchID, []string{
			binding.Provision.Tool,
			binding.Provision.Version,
			recipeE2EMismatchPack,
		})
	})

	t.Run("allowlisted tool at the pinned version clears the gate", func(t *testing.T) {
		projectRoot, ref, _, _ := stageTrustGateRun(t, recipeE2EPassPack, recipeE2EPassID)

		output, err := runRecipeApplyCLI(t, projectRoot, ref)
		if err != nil {
			t.Fatalf("backstop recipe apply %s was rejected: %v\noutput:\n%s\na gate that rejects every pack proves nothing", ref, err, output)
		}
	})
}
