package distribution_test

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack/distribution"
)

// The REQ-002 / REQ-006 / REQ-007 suite for the OTHER two cloning commands.
//
// Add got its identity gate in phase 8. Update and upgrade clone a tag they intend to
// install just as add does, so the same drift is possible on both — and until this phase
// neither checked. A pack whose manifest disagrees with its tag would update cleanly and
// then fail at gate time looking for assets under a name nothing installed.

// manifestServingCloner materializes a manifest of the caller's choosing, so a test can
// make the CLONED tree disagree with the tag that was requested.
type manifestServingCloner struct {
	manifest string
	extra    map[string]string
}

func (c *manifestServingCloner) Clone(_, _, destDir string) error {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}
	for rel, body := range c.extra {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(destDir, rel)), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(destDir, rel), []byte(body), 0o644); err != nil {
			return err
		}
	}
	return os.WriteFile(filepath.Join(destDir, "pack.yml"), []byte(c.manifest), 0o644)
}

func (c *manifestServingCloner) ListTags(_ string) ([]string, error) {
	return []string{"v1.0.0", "v1.1.0", "v2.0.0"}, nil
}

// ── Version drift on update and upgrade ─────────────────────────────────────────

func TestUpdateCommand_Run_ResolvedTagVersionDriftRefuses(t *testing.T) {
	projectDir := setupUpdateProject(t)

	// The resolver resolves 1.1.0; the cloned manifest declares something else.
	update := newTestUpdateCommand(t,
		&manifestServingCloner{manifest: identityManifestYAML("acme/valid-pack", "9.9.9")},
		&mockValidator{},
		&mockVersionResolver{latestMinor: "1.1.0"},
	)

	_, err := update.Run("acme/valid-pack", distribution.UpdateOptions{ProjectDir: projectDir})
	var mismatch *distribution.VersionMismatchError
	if !errors.As(err, &mismatch) {
		t.Fatalf("an update whose resolved tag's manifest disagrees must refuse with *VersionMismatchError, got %v (%T)", err, err)
	}
}

func TestUpgradeCommand_Run_TargetTagVersionDriftRefuses(t *testing.T) {
	projectDir := setupUpgradeProject(t)

	// The ref targets @2.0.0; the cloned manifest declares 1.1.0 — the exact shape
	// valid-pack-v3 exists so the other upgrade tests can avoid.
	upgrade := newTestUpgradeCommand(t,
		&manifestServingCloner{manifest: identityManifestYAML("acme/valid-pack", "1.1.0")},
		&mockValidator{}, &mockScanner{}, &mockRemediationGenerator{},
	)

	_, err := upgrade.Run("acme/valid-pack@2.0.0", distribution.UpgradeOptions{ProjectDir: projectDir})
	var mismatch *distribution.VersionMismatchError
	if !errors.As(err, &mismatch) {
		t.Fatalf("an upgrade whose target tag's manifest disagrees must refuse with *VersionMismatchError, got %v (%T)", err, err)
	}
}

// identityManifestWithToolConfig declares a tool_config block, so a command that reached
// the merge WOULD write .golangci.yml into the consumer project.
//
// CLM-079 names tool config as one of the surfaces a refusal must leave untouched, and a
// drift fixture without a tool_config block cannot exercise it: the merge runs, writes
// nothing, and the surface assertion passes no matter where the gate sits. Verified by
// mutation — with a tool_config-less manifest, moving the gate to AFTER MergeToolConfig
// still passed. This is the fixture that makes the ordering genuinely falsifiable.
func identityManifestWithToolConfig(name, version string) string {
	return "name: " + name + "\nversion: \"" + version + "\"\narchetype: rule-pack\n" +
		"tool_config:\n  - config_file: \".golangci.yml\"\n    settings:\n      linters.enable.revive: true\n"
}

// identityManifestPreservingRules declares the SAME rules testdata/valid-pack does, with
// a name of the caller's choosing.
//
// It exists because update runs DetectTamper after the identity gate: a manifest that
// merely renamed the pack while dropping its rules is refused for RULE REMOVAL before the
// divergence assertion is ever reached. Keeping the rules isolates the property under
// test — a name that diverges, and nothing else about the pack changing.
func identityManifestPreservingRules(name, version string) string {
	return "name: " + name + "\nversion: \"" + version + "\"\narchetype: rule-pack\n" +
		"rules:\n" +
		"  - id: RULE-001\n    severity: error\n    risk_class: high\n    description: Test rule one\n    pattern: \"test-pattern-1\"\n" +
		"  - id: RULE-002\n    severity: warning\n    risk_class: medium\n    description: Test rule two\n    pattern: \"test-pattern-2\"\n"
}

// ── Divergence carried out on the result ────────────────────────────────────────

func TestUpdateCommand_Run_CarriesDivergenceWarning(t *testing.T) {
	projectDir := setupUpdateProject(t)

	update := newTestUpdateCommand(t,
		&manifestServingCloner{manifest: identityManifestPreservingRules("other/renamed-pack", "1.1.0")},
		&mockValidator{},
		&mockVersionResolver{latestMinor: "1.1.0"},
	)

	result, err := update.Run("acme/valid-pack", distribution.UpdateOptions{ProjectDir: projectDir})
	if err != nil {
		t.Fatalf("divergence must never refuse an update: %v", err)
	}
	if !containsAny(result.Warnings, "other/renamed-pack") {
		t.Errorf("the divergence diagnostic must arrive on UpdateResult.Warnings; before this spec the type had no warning field at all, so it would have been computed and dropped. Got: %v", result.Warnings)
	}
}

func TestUpgradeCommand_Run_CarriesDivergenceWarning(t *testing.T) {
	projectDir := setupUpgradeProject(t)

	upgrade := newTestUpgradeCommand(t,
		&manifestServingCloner{manifest: identityManifestYAML("other/renamed-pack", "2.0.0")},
		&mockValidator{}, &mockScanner{}, &mockRemediationGenerator{},
	)

	result, err := upgrade.Run("acme/valid-pack@2.0.0", distribution.UpgradeOptions{ProjectDir: projectDir})
	if err != nil {
		t.Fatalf("divergence must never refuse an upgrade: %v", err)
	}
	if !containsAny(result.Warnings, "other/renamed-pack") {
		t.Errorf("the divergence diagnostic must arrive on UpgradeResult.Warnings. Got: %v", result.Warnings)
	}
}

// containsAny reports whether any warning mentions want.
func containsAny(warnings []string, want string) bool {
	for _, w := range warnings {
		if strings.Contains(w, want) {
			return true
		}
	}
	return false
}

// ── Nothing is written before the gate ──────────────────────────────────────────

// surfaceSnapshot records every consumer-state file under a project, so a refusal can be
// proven byte-wise rather than by existence.
func surfaceSnapshot(t *testing.T, projectDir string) []string {
	t.Helper()
	var out []string
	err := filepath.Walk(projectDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		rel, relErr := filepath.Rel(projectDir, path)
		if relErr != nil {
			return relErr
		}
		out = append(out, rel+"\x00"+string(data))
		return nil
	})
	if err != nil {
		t.Fatalf("snapshotting %s: %v", projectDir, err)
	}
	sort.Strings(out)
	return out
}

func assertSurfacesUnchanged(t *testing.T, before, after []string) {
	t.Helper()
	if len(before) != len(after) {
		t.Fatalf("the refusal changed the project's file set: %d before, %d after", len(before), len(after))
	}
	for i := range before {
		if before[i] != after[i] {
			t.Errorf("a refusal modified consumer state; entry %d differs", i)
		}
	}
}

// TestUpdateCommand_Run_IdentityRefusalLeavesEverySurfaceUntouched asserts the installed
// tree BYTE-WISE (CLM-078). A refusal that replaced it with the new content and then
// errored would still pass an existence check.
func TestUpdateCommand_Run_IdentityRefusalLeavesEverySurfaceUntouched(t *testing.T) {
	projectDir := setupUpdateProject(t)
	before := surfaceSnapshot(t, projectDir)

	update := newTestUpdateCommand(t,
		&manifestServingCloner{manifest: identityManifestYAML("acme/valid-pack", "9.9.9")},
		&mockValidator{},
		&mockVersionResolver{latestMinor: "1.1.0"},
	)
	if _, err := update.Run("acme/valid-pack", distribution.UpdateOptions{ProjectDir: projectDir}); err == nil {
		t.Fatal("expected an identity refusal")
	}

	assertSurfacesUnchanged(t, before, surfaceSnapshot(t, projectDir))
}

// TestUpgradeCommand_Run_IdentityRefusalLeavesEverySurfaceUntouched covers the same three
// surfaces plus provenance, tool config and any remediation artifact (CLM-079). Upgrade's
// first consumer write is the tool-config merge, so the gate must precede the violation
// scan, which already precedes it.
func TestUpgradeCommand_Run_IdentityRefusalLeavesEverySurfaceUntouched(t *testing.T) {
	projectDir := setupUpgradeProject(t)
	before := surfaceSnapshot(t, projectDir)

	upgrade := newTestUpgradeCommand(t,
		// The manifest declares tool_config so the merge WOULD write into the project;
		// that is what makes "nothing written before the gate" falsifiable here.
		&manifestServingCloner{manifest: identityManifestWithToolConfig("acme/valid-pack", "9.9.9")},
		&mockValidator{}, &mockScanner{}, &mockRemediationGenerator{},
	)
	if _, err := upgrade.Run("acme/valid-pack@2.0.0", distribution.UpgradeOptions{ProjectDir: projectDir}); err == nil {
		t.Fatal("expected an identity refusal")
	}

	assertSurfacesUnchanged(t, before, surfaceSnapshot(t, projectDir))
}
