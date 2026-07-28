package distribution_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack/distribution"
)

// The REQ-003 / REQ-006 / REQ-007 add suite: the MANIFEST name is the pack's install and
// runtime identity, divergence is loud but never fatal, and nothing is written before the
// identity gate passes.
//
// The divergence that motivates all of it is real: a pack legitimately named
// backstop/harness-toolchain lives at backstop-ai/backstop-harness-toolchain-pack. Before
// this spec, add took the pack name from the requested COORDINATE, so it installed under
// the repository name while the gate later resolved the pack's rules, producers and
// converters under its MANIFEST name — the add reported success and the gate failed
// looking for assets that were installed somewhere else.

const (
	divergentAddRef       = "acme/divergent"
	divergentManifestName = "other/renamed-pack"
	convergentName        = "acme/valid-pack"
)

// identityMockCloner materializes a manifest of the caller's choosing and RECORDS every
// invocation, so a test can prove a refusal happened BEFORE git rather than after it.
type identityMockCloner struct {
	manifest string
	calls    []string
	failWith error
}

func (c *identityMockCloner) Clone(url, version, destDir string) error {
	c.calls = append(c.calls, url+" @ "+version)
	if c.failWith != nil {
		return c.failWith
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(destDir, "pack.yml"), []byte(c.manifest), 0o644)
}

func (c *identityMockCloner) ListTags(_ string) ([]string, error) {
	return []string{"v1.0.0"}, nil
}

// identityManifestYAML is a minimal manifest carrying just the two identity fields the
// gate reads. No tool_config, so the merge stage is a no-op and these tests stay about
// identity.
func identityManifestYAML(name, version string) string {
	return "name: " + name + "\nversion: \"" + version + "\"\narchetype: rule-pack\n"
}

// addWithManifest runs an add whose clone materializes the given manifest.
func addWithManifest(t *testing.T, projectDir, ref, manifest string) (*distribution.AddResult, *identityMockCloner, error) {
	t.Helper()
	cloner := &identityMockCloner{manifest: manifest}
	add := newTestAddCommand(t, cloner, &mockValidator{})
	result, err := add.Run(ref, distribution.AddOptions{ProjectDir: projectDir, Version: "1.0.0"})
	return result, cloner, err
}

// ── REQ-003: the manifest name is the identity ──────────────────────────────────

func TestAddCommand_Run_InstallPathComesFromManifestName(t *testing.T) {
	projectDir := setupAddProject(t)

	if _, _, err := addWithManifest(t, projectDir, divergentAddRef+"@1.0.0",
		identityManifestYAML(divergentManifestName, "1.0.0")); err != nil {
		t.Fatalf("a divergent add must succeed: %v", err)
	}

	underManifest := filepath.Join(projectDir, ".backstop", "packs", divergentManifestName)
	if _, err := os.Stat(underManifest); err != nil {
		t.Errorf("the pack must install under the MANIFEST name at %s: %v", underManifest, err)
	}
	// The coordinate path must NOT exist — that is where the old behavior put it, and it
	// is where the gate would fail to find the pack's assets.
	underCoordinate := filepath.Join(projectDir, ".backstop", "packs", divergentAddRef)
	if _, err := os.Stat(underCoordinate); !os.IsNotExist(err) {
		t.Errorf("the pack was installed under the COORDINATE at %s (stat error: %v); that is the defect this requirement removes", underCoordinate, err)
	}
}

func TestAddCommand_Run_ManifestKeyComesFromManifestName(t *testing.T) {
	projectDir := setupAddProject(t)

	if _, _, err := addWithManifest(t, projectDir, divergentAddRef+"@1.0.0",
		identityManifestYAML(divergentManifestName, "1.0.0")); err != nil {
		t.Fatalf("Add: %v", err)
	}

	yml := string(mustReadFile(t, filepath.Join(projectDir, "backstop.yml")))
	if !strings.Contains(yml, divergentManifestName) {
		t.Errorf("backstop.yml's packs key must be the manifest name %q; got:\n%s", divergentManifestName, yml)
	}
	if strings.Contains(yml, divergentAddRef) {
		t.Errorf("backstop.yml carries the COORDINATE %q as a packs key; the key is the runtime identity and must be the manifest name.\nGot:\n%s", divergentAddRef, yml)
	}
}

// TestAddCommand_Run_LockKeyComesFromManifestName asserts BOTH the map key and the
// entry's own name field (CLM-030). Checking the key alone would let a mismatched `name`
// field through, and the two are read by different consumers.
func TestAddCommand_Run_LockKeyComesFromManifestName(t *testing.T) {
	projectDir := setupAddProject(t)

	if _, _, err := addWithManifest(t, projectDir, divergentAddRef+"@1.0.0",
		identityManifestYAML(divergentManifestName, "1.0.0")); err != nil {
		t.Fatalf("Add: %v", err)
	}

	lf, err := distribution.ReadLockfile(filepath.Join(projectDir, "backstop.lock"))
	if err != nil {
		t.Fatalf("ReadLockfile: %v", err)
	}
	entry, ok := lf.Packs[divergentManifestName]
	if !ok {
		t.Fatalf("the lock has no entry keyed by the manifest name %q; keys are: %v", divergentManifestName, lockKeys(lf))
	}
	if entry.Name != divergentManifestName {
		t.Errorf("lock entry name = %q, want the manifest name %q", entry.Name, divergentManifestName)
	}
	if _, wrong := lf.Packs[divergentAddRef]; wrong {
		t.Errorf("the lock is keyed by the COORDINATE %q", divergentAddRef)
	}
}

func lockKeys(lf *distribution.Lockfile) []string {
	out := make([]string, 0, len(lf.Packs))
	for k := range lf.Packs {
		out = append(out, k)
	}
	return out
}

// TestAddCommand_Run_EqualNameAndCoordinateInstallsUnchanged is the fleet-convention
// case: the ten packs published under backstop-ai hold name == coordinate, so the
// overwhelmingly common path must behave exactly as it did before (CLM-032).
func TestAddCommand_Run_EqualNameAndCoordinateInstallsUnchanged(t *testing.T) {
	projectDir := setupAddProject(t)

	result, _, err := addWithManifest(t, projectDir, convergentName+"@1.0.0",
		identityManifestYAML(convergentName, "1.0.0"))
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	if result.PackName != convergentName {
		t.Errorf("PackName = %q, want %q", result.PackName, convergentName)
	}
	if _, statErr := os.Stat(filepath.Join(projectDir, ".backstop", "packs", convergentName)); statErr != nil {
		t.Errorf("the pack must install under its one name: %v", statErr)
	}
	if len(result.Warnings) != 0 {
		t.Errorf("an equal name and coordinate must emit NO divergence diagnostic; got %v", result.Warnings)
	}
}

// TestAddCommand_Run_LocalPathIdentityComesFromSharedResolution asserts BOTH halves of
// CLM-038: behaviorally the name arrives from the manifest, and STRUCTURALLY there is no
// second manifest reader left in command.go.
//
// The structural half is the one that carries the claim. "The name arrives" would pass
// against a second inline reader that happened to agree; only a source-level check can
// see that a duplicate implementation exists at all.
func TestAddCommand_Run_LocalPathIdentityComesFromSharedResolution(t *testing.T) {
	projectDir := setupAddProject(t)

	localDir := filepath.Join(t.TempDir(), "my-local-pack")
	writeFile(t, filepath.Join(localDir, "pack.yml"), identityManifestYAML("internal/local-rules", "1.0.0"))

	add := newTestAddCommand(t, defaultTestPackCloner(), &mockValidator{})
	result, err := add.Run(localDir, distribution.AddOptions{ProjectDir: projectDir})
	if err != nil {
		t.Fatalf("local-path Add: %v", err)
	}
	if result.PackName != "internal/local-rules" {
		t.Errorf("PackName = %q, want the manifest's %q", result.PackName, "internal/local-rules")
	}

	// STRUCTURAL: the inline yaml.Unmarshal into map[string]interface{} must be retired.
	source := mustReadFile(t, "command.go")
	if strings.Contains(string(source), "map[string]interface{}") {
		t.Error("command.go still contains a map[string]interface{} manifest read; local and remote identity must come from ONE implementation (ReadManifestIdentity), and a second reader is exactly what CLM-038 forbids")
	}
}

// TestRemove_DivergentNamePackRemovesEverySurfaceByManifestName proves Remove needs no
// change of its own (CLM-039): it looks a pack up by the key REQ-003 sets, so keying
// everything by the manifest name makes removal work by construction.
func TestRemove_DivergentNamePackRemovesEverySurfaceByManifestName(t *testing.T) {
	projectDir := setupAddProject(t)

	if _, _, err := addWithManifest(t, projectDir, divergentAddRef+"@1.0.0",
		identityManifestYAML(divergentManifestName, "1.0.0")); err != nil {
		t.Fatalf("Add: %v", err)
	}

	if _, err := distribution.Remove(divergentManifestName, distribution.RemoveOptions{ProjectDir: projectDir}); err != nil {
		t.Fatalf("Remove by manifest name: %v", err)
	}

	if _, err := os.Stat(filepath.Join(projectDir, ".backstop", "packs", divergentManifestName)); !os.IsNotExist(err) {
		t.Errorf("the installed tree survived removal (stat error: %v)", err)
	}
	if yml := string(mustReadFile(t, filepath.Join(projectDir, "backstop.yml"))); strings.Contains(yml, divergentManifestName) {
		t.Errorf("backstop.yml still declares the pack:\n%s", yml)
	}
	lf, err := distribution.ReadLockfile(filepath.Join(projectDir, "backstop.lock"))
	if err != nil {
		t.Fatalf("ReadLockfile: %v", err)
	}
	if _, still := lf.Packs[divergentManifestName]; still {
		t.Error("the lock entry survived removal")
	}
}

// ── REQ-001: refusal happens BEFORE git ─────────────────────────────────────────

// TestAddCommand_Run_UnresolvableVersionRefusesBeforeCloning requires the cloner to
// record ZERO invocations (CLM-010).
//
// Asserting only that an error came back cannot distinguish "refused before git" from
// "cloned, then refused" — and the difference is the entire requirement. Today a bare
// org/name clones the ref "v" and the operator's diagnostic is a git error about a
// nonexistent branch.
func TestAddCommand_Run_UnresolvableVersionRefusesBeforeCloning(t *testing.T) {
	projectDir := setupAddProject(t)

	cloner := &identityMockCloner{manifest: identityManifestYAML(convergentName, "1.0.0")}
	add := newTestAddCommand(t, cloner, &mockValidator{})

	// No @version suffix and no --version: nothing can resolve.
	_, err := add.Run("acme/versionless", distribution.AddOptions{ProjectDir: projectDir})
	if err == nil {
		t.Fatal("a reference carrying no version and no --version flag must refuse")
	}

	var unresolved *distribution.VersionUnresolvedError
	if !errors.As(err, &unresolved) {
		t.Errorf("error is %T (%v), want *VersionUnresolvedError", err, err)
	}
	if len(cloner.calls) != 0 {
		t.Errorf("the cloner was invoked %d time(s) before the refusal: %v — the check must complete BEFORE any git subprocess runs", len(cloner.calls), cloner.calls)
	}
}

// ── REQ-007: nothing is written before the gate ─────────────────────────────────

// driftingAdd runs an add whose manifest version disagrees with the requested tag, which
// is the refusal every REQ-007 test drives.
func driftingAdd(t *testing.T, projectDir string) (*identityMockCloner, *recordingScratchValidator, error) {
	t.Helper()
	cloner := &identityMockCloner{manifest: identityManifestYAML(convergentName, "9.9.9")}
	validator := &recordingScratchValidator{}
	add := newTestAddCommand(t, cloner, validator)
	_, err := add.Run(convergentName+"@1.0.0", distribution.AddOptions{ProjectDir: projectDir, Version: "1.0.0"})
	if err == nil {
		t.Fatal("a manifest version disagreeing with the requested tag must refuse")
	}
	return cloner, validator, err
}

// recordingScratchValidator notes whether validation was reached and what directory it
// was handed, so a test can prove the identity gate ran FIRST and can locate the scratch
// directory precisely rather than scanning the shared OS temp area.
type recordingScratchValidator struct {
	handedDir string
	reached   bool
}

func (v *recordingScratchValidator) RunPackCheck(packDir string) error {
	v.reached = true
	v.handedDir = packDir
	return nil
}
func (v *recordingScratchValidator) RunPackTest(_ string) error { return nil }

func TestAddCommand_Run_IdentityRefusalWritesNoInstalledContent(t *testing.T) {
	projectDir := setupAddProject(t)
	_, _, _ = driftingAdd(t, projectDir)

	// BOTH paths: a refusal that created the coordinate directory would pass a
	// manifest-name-only check.
	for _, name := range []string{convergentName, divergentManifestName} {
		p := filepath.Join(projectDir, ".backstop", "packs", name)
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("a refusal created installed content at %s (stat error: %v)", p, err)
		}
	}
}

func TestAddCommand_Run_IdentityRefusalLeavesManifestUntouched(t *testing.T) {
	projectDir := setupAddProject(t)
	ymlPath := filepath.Join(projectDir, "backstop.yml")
	before := mustReadFile(t, ymlPath)

	_, _, _ = driftingAdd(t, projectDir)

	after := mustReadFile(t, ymlPath)
	if string(before) != string(after) {
		t.Errorf("backstop.yml changed across a refusal.\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

// TestAddCommand_Run_IdentityRefusalLeavesLockUntouched covers the ABSENT case too — the
// one a naive read-modify-write breaks, because it creates the file on the way to
// failing (CLM-073).
func TestAddCommand_Run_IdentityRefusalLeavesLockUntouched(t *testing.T) {
	t.Run("lock absent stays absent", func(t *testing.T) {
		projectDir := setupAddProject(t)
		lockPath := filepath.Join(projectDir, "backstop.lock")

		_, _, _ = driftingAdd(t, projectDir)

		if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
			t.Errorf("a refusal created backstop.lock where none existed (stat error: %v)", err)
		}
	})

	t.Run("existing lock byte-identical", func(t *testing.T) {
		projectDir := setupAddProject(t)
		lockPath := filepath.Join(projectDir, "backstop.lock")
		writeFile(t, lockPath, "packs: {}\n")
		before := mustReadFile(t, lockPath)

		_, _, _ = driftingAdd(t, projectDir)

		after := mustReadFile(t, lockPath)
		if string(before) != string(after) {
			t.Errorf("backstop.lock changed across a refusal.\nbefore:\n%s\nafter:\n%s", before, after)
		}
	})
}

func TestAddCommand_Run_IdentityRefusalWritesNoProvenanceOrToolConfig(t *testing.T) {
	projectDir := setupAddProject(t)
	provPath := filepath.Join(projectDir, ".backstop", "pack-config-provenance.json")
	toolConfigPath := filepath.Join(projectDir, ".golangci.yml")

	_, _, _ = driftingAdd(t, projectDir)

	if _, err := os.Stat(provPath); !os.IsNotExist(err) {
		t.Errorf("a refusal wrote provenance at %s (stat error: %v)", provPath, err)
	}
	if _, err := os.Stat(toolConfigPath); !os.IsNotExist(err) {
		t.Errorf("a refusal created a tool-config file at %s (stat error: %v)", toolConfigPath, err)
	}
}

func TestAddCommand_Run_IdentityRefusalLeavesGitignoreUntouched(t *testing.T) {
	t.Run("gitignore absent stays absent", func(t *testing.T) {
		projectDir := setupAddProject(t)
		_, _, _ = driftingAdd(t, projectDir)

		if _, err := os.Stat(filepath.Join(projectDir, ".gitignore")); !os.IsNotExist(err) {
			t.Errorf("a refusal created .gitignore (stat error: %v)", err)
		}
	})

	t.Run("existing gitignore byte-identical", func(t *testing.T) {
		projectDir := setupAddProject(t)
		gitignorePath := filepath.Join(projectDir, ".gitignore")
		writeFile(t, gitignorePath, "node_modules/\n")
		before := mustReadFile(t, gitignorePath)

		_, _, _ = driftingAdd(t, projectDir)

		after := mustReadFile(t, gitignorePath)
		if string(before) != string(after) {
			t.Errorf(".gitignore changed across a refusal.\nbefore:\n%s\nafter:\n%s", before, after)
		}
	})
}

// TestAddCommand_Run_IdentityGatePrecedesValidation pins the ORDER, not merely the
// outcome (CLM-076).
//
// It would FAIL if the gate ran after validation: the validator here RECORDS whether it
// was reached, so "the identity diagnostic came back" and "validation never ran" are
// asserted separately. Without the second half, an implementation that validated first
// and then refused on identity would still produce an identity error and pass.
func TestAddCommand_Run_IdentityGatePrecedesValidation(t *testing.T) {
	projectDir := setupAddProject(t)

	_, validator, err := driftingAdd(t, projectDir)

	var mismatch *distribution.VersionMismatchError
	if !errors.As(err, &mismatch) {
		t.Fatalf("want the identity diagnostic (the cheaper, more specific failure), got %v (%T)", err, err)
	}
	if validator.reached {
		t.Error("validation ran before the identity gate refused; the gate must run FIRST so a pack that is both identity-invalid and validation-failing reports the identity problem")
	}
}

// TestAddCommand_Run_IdentityRefusalLeavesNoTemporaryDirectories captures the directories
// from the doubles rather than scanning the OS temp area, which is shared between
// concurrent tests and would make this assertion flaky rather than strong (CLM-077).
func TestAddCommand_Run_IdentityRefusalLeavesNoTemporaryDirectories(t *testing.T) {
	projectDir := setupAddProject(t)

	cloner := &recordingCloneDirValidator{manifest: identityManifestYAML(convergentName, "9.9.9")}
	validator := &recordingScratchValidator{}
	add := newTestAddCommand(t, cloner, validator)

	if _, err := add.Run(convergentName+"@1.0.0", distribution.AddOptions{ProjectDir: projectDir, Version: "1.0.0"}); err == nil {
		t.Fatal("expected an identity refusal")
	}

	if cloner.destDir == "" {
		t.Fatal("the cloner was never invoked, so this test cannot observe the clone temp dir")
	}
	if _, err := os.Stat(cloner.destDir); !os.IsNotExist(err) {
		t.Errorf("the clone temp directory %q survived the refusal (stat error: %v)", cloner.destDir, err)
	}
	// The gate refuses before validation, so no scratch directory should ever have been
	// created; if one was, it must still be gone.
	if validator.handedDir != "" {
		if _, err := os.Stat(validator.handedDir); !os.IsNotExist(err) {
			t.Errorf("the validation scratch directory %q survived the refusal (stat error: %v)", validator.handedDir, err)
		}
	}
}

// recordingCloneDirValidator records the destination directory it was handed.
type recordingCloneDirValidator struct {
	manifest string
	destDir  string
}

func (c *recordingCloneDirValidator) Clone(_, _, destDir string) error {
	c.destDir = destDir
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(destDir, "pack.yml"), []byte(c.manifest), 0o644)
}

func (c *recordingCloneDirValidator) ListTags(_ string) ([]string, error) {
	return []string{"v1.0.0"}, nil
}

// ── REQ-006: divergence is loud and never fatal ─────────────────────────────────

func TestAddCommand_Run_DivergentIdentitySucceedsWithWarning(t *testing.T) {
	projectDir := setupAddProject(t)

	result, _, err := addWithManifest(t, projectDir, divergentAddRef+"@1.0.0",
		identityManifestYAML(divergentManifestName, "1.0.0"))
	if err != nil {
		t.Fatalf("divergence must NEVER refuse — OQ-9 resolved to option (b) and rejected requiring equality by name: %v", err)
	}
	if len(result.Warnings) == 0 {
		t.Fatal("a divergent add must report the divergence; silence would leave an operator to discover it when the gate cannot find the pack's assets")
	}
}

func TestAddCommand_Run_DivergenceWarningNamesAllThreeIdentities(t *testing.T) {
	projectDir := setupAddProject(t)

	result, _, err := addWithManifest(t, projectDir, divergentAddRef+"@1.0.0",
		identityManifestYAML(divergentManifestName, "1.0.0"))
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	joined := strings.Join(result.Warnings, "\n")
	// All THREE. Two of three leaves an operator unable to find where the pack went.
	for _, want := range []string{divergentAddRef, divergentManifestName} {
		if !strings.Contains(joined, want) {
			t.Errorf("the divergence diagnostic does not name %q; got:\n%s", want, joined)
		}
	}
	installPath := filepath.Join(".backstop", "packs", divergentManifestName)
	if !strings.Contains(joined, installPath) {
		t.Errorf("the diagnostic does not name the resolved install path %q; got:\n%s", installPath, joined)
	}
}

// TestAddCommand_Run_EqualIdentitiesEmitNoWarning makes SILENCE the signal that the
// fleet convention still holds (CLM-065). An always-warning implementation reds here.
func TestAddCommand_Run_EqualIdentitiesEmitNoWarning(t *testing.T) {
	projectDir := setupAddProject(t)

	result, _, err := addWithManifest(t, projectDir, convergentName+"@1.0.0",
		identityManifestYAML(convergentName, "1.0.0"))
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if len(result.Warnings) != 0 {
		t.Errorf("an equal name and coordinate must emit NO diagnostic, or the signal stops meaning anything; got %v", result.Warnings)
	}
}

// TestAddCommand_Run_CaseOnlyDifferenceCountsAsDivergent pins byte-exactness (CLM-066).
// GitHub would treat Acme/Valid-Pack and acme/valid-pack as one repository; another host
// would not, and DD-31 removed that host assumption deliberately.
func TestAddCommand_Run_CaseOnlyDifferenceCountsAsDivergent(t *testing.T) {
	projectDir := setupAddProject(t)

	result, _, err := addWithManifest(t, projectDir, "Acme/Valid-Pack@1.0.0",
		identityManifestYAML(convergentName, "1.0.0"))
	if err != nil {
		t.Fatalf("a case-only difference must not refuse: %v", err)
	}
	if len(result.Warnings) == 0 {
		t.Error("a coordinate differing from the manifest name only in CASE is divergent; comparison is byte-exact because case-insensitivity is a property of one host")
	}
}

// TestAddCommand_Run_DivergenceRecordsBothIdentities requires the lock to answer "what is
// it called here?" and "where did it come from?" INDEPENDENTLY (CLM-067).
func TestAddCommand_Run_DivergenceRecordsBothIdentities(t *testing.T) {
	projectDir := setupAddProject(t)

	if _, _, err := addWithManifest(t, projectDir, divergentAddRef+"@1.0.0",
		identityManifestYAML(divergentManifestName, "1.0.0")); err != nil {
		t.Fatalf("Add: %v", err)
	}

	lf, err := distribution.ReadLockfile(filepath.Join(projectDir, "backstop.lock"))
	if err != nil {
		t.Fatalf("ReadLockfile: %v", err)
	}
	entry := lf.Packs[divergentManifestName]
	if entry.Name != divergentManifestName {
		t.Errorf("lock name = %q, want the manifest name %q", entry.Name, divergentManifestName)
	}
	if entry.SourceCoordinate != divergentAddRef {
		t.Errorf("source_coordinate = %q, want the requested coordinate %q recorded independently", entry.SourceCoordinate, divergentAddRef)
	}
}
