package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// substantiveness_assembly_test.go pins the one non-test helper in cmd/backstop that
// installed a pack through the free distribution.Add and leaned on the nil-validator
// skip (SPEC-055 REQ-013).
//
// installSubstantivenessLocalPack is the exact shape the helper's four existing call
// sites depend on. Declaring it as a TYPE rather than a variable keeps the pin at
// compile time without introducing package-level state.
type installSubstantivenessLocalPack func(*e2eWorkspace, string) error

// TestInstallSubstantivenessLocalPack_UsesProductionAssembly (CLM-095) — the helper
// draws its command from the production assembly, so validation genuinely runs.
//
// The two halves are both necessary. The positive half proves packs/substantiveness
// passes pack check and pack test through this path — a green install that skipped
// validation would satisfy "installs" but not "with validation running". The negative
// half is what makes the positive half falsifiable: pointed at a pack that FAILS pack
// check, the helper must refuse. Against the pre-spec implementation — a bare
// AddOptions with no validator in it — that pack installed cleanly, so the negative
// half is the assertion the old code cannot pass.
func TestInstallSubstantivenessLocalPack_UsesProductionAssembly(t *testing.T) {
	// A COMPILE-TIME pin on the method's receiver and signature: the conversion below
	// does not compile unless the method still takes `(w *e2eWorkspace)` and
	// `(repoRoot string) error`.
	//
	// THE RECEIVER IS THE POINT. That exact shape is what lets the four existing call
	// sites in gate_substantiveness_e2e_test.go need no edits at all. A "helpful"
	// conversion to a free function would compile everywhere else and quietly cascade
	// into four unnecessary test edits, breaking the property this claim asserts.
	//
	// The pin is then USED to drive the positive case, so it is load-bearing rather
	// than a declaration someone could delete as unused.
	install := installSubstantivenessLocalPack((*e2eWorkspace).installSubstantivenessLocalPack)

	t.Run("validation runs and the real pack passes it", func(t *testing.T) {
		workspace, err := newE2EWorkspace(t.TempDir())
		if err != nil {
			t.Fatalf("scaffolding the e2e workspace: %v", err)
		}

		if installErr := install(workspace, repoRoot(t)); installErr != nil {
			t.Fatalf("packs/substantiveness must pass pack check and pack test through the production assembly: %v", installErr)
		}
		if !workspace.installed || workspace.installInfo == nil {
			t.Fatal("the helper reported success without recording an install; its callers read both")
		}
		if _, statErr := os.Stat(filepath.Join(workspace.root, ".backstop", "packs", workspace.installInfo.PackName, "pack.yml")); statErr != nil {
			t.Errorf("the pack was not materialized in the workspace: %v", statErr)
		}
	})

	t.Run("a pack that fails validation is refused", func(t *testing.T) {
		workspace, err := newE2EWorkspace(t.TempDir())
		if err != nil {
			t.Fatalf("scaffolding the e2e workspace: %v", err)
		}

		// A repo root whose packs/substantiveness source is the fixture that fails
		// pack check in phase1-structural and nothing else: everything in it parses,
		// so validation is the only thing that can reject it.
		failingRoot := t.TempDir()
		copyTree(t,
			filepath.Join("testdata", "hermetic-remote", invalidPackFixture),
			substantivenessSourceDir(failingRoot))

		installErr := install(workspace, failingRoot)
		if installErr == nil {
			t.Fatal("a pack that fails pack check must not install; validation is not running on this path")
		}
		if !strings.Contains(installErr.Error(), "validation") {
			t.Errorf("the refusal must name the validation failure, got: %v", installErr)
		}
		if _, statErr := os.Stat(filepath.Join(workspace.root, ".backstop", "packs")); !os.IsNotExist(statErr) {
			t.Errorf("a pack that failed validation must leave no installed content behind (stat error: %v)", statErr)
		}
	})
}
