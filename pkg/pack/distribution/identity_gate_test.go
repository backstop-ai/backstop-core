package distribution_test

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack/distribution"
)

// The REQ-002 / REQ-003 gate suite. Every test builds a pack directory in t.TempDir()
// carrying a hand-written pack.yml and calls ValidateRemoteIdentity directly — no clone,
// no command, no consumer project. The gate is a pure function over a materialized tree,
// which is exactly what lets REQ-007 place it before the first byte of consumer state is
// written.

const identityTestCoordinate = "acme/pack"

// writeIdentityPack materializes a pack directory whose pack.yml is the given literal,
// and returns the directory. A raw literal rather than a struct so a test can express a
// manifest that does not parse at all (CLM-021).
func writeIdentityPack(t *testing.T, manifest string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pack.yml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("writing pack.yml: %v", err)
	}
	return dir
}

// identityManifest builds a minimal well-formed manifest with the given name and version.
func identityManifest(name, version string) string {
	return "name: " + name + "\nversion: " + version + "\nlanguage: neutral\narchetype: enforcement\n"
}

// treeSnapshot records every file path under root and its contents, so a caller can prove
// a function did not write. Sorted for a stable comparison.
func treeSnapshot(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
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
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		out = append(out, rel+"\x00"+string(data))
		return nil
	})
	if err != nil {
		t.Fatalf("snapshotting %s: %v", root, err)
	}
	sort.Strings(out)
	return out
}

// assertNoMutation proves ValidateRemoteIdentity wrote nothing. REQ-007 makes this
// function the thing that runs BEFORE anything is safe to write, so a gate that mutated
// its input would defeat its own purpose — and would do so invisibly, since every caller
// invokes it on a directory it is about to copy and hash.
func assertNoMutation(t *testing.T, packDir string, before []string) {
	t.Helper()
	after := treeSnapshot(t, packDir)
	if len(before) != len(after) {
		t.Fatalf("ValidateRemoteIdentity changed the file set: %d files before, %d after", len(before), len(after))
	}
	for i := range before {
		if before[i] != after[i] {
			t.Errorf("ValidateRemoteIdentity mutated the pack directory; entry %d differs", i)
		}
	}
}

// validateIdentityUnchanged runs the gate and proves it left packDir alone either way.
func validateIdentityUnchanged(t *testing.T, coordinate, effectiveVersion, packDir string) (*distribution.RemoteIdentity, error) {
	t.Helper()
	before := treeSnapshot(t, packDir)
	identity, err := distribution.ValidateRemoteIdentity(coordinate, effectiveVersion, packDir)
	assertNoMutation(t, packDir, before)
	return identity, err
}

// ── Version equality: the passing shapes ────────────────────────────────────────

func TestValidateRemoteIdentity_PrefixedTagUnprefixedManifestPasses(t *testing.T) {
	dir := writeIdentityPack(t, identityManifest(identityTestCoordinate, "1.0.0"))

	identity, err := validateIdentityUnchanged(t, identityTestCoordinate, "v1.0.0", dir)
	if err != nil {
		t.Fatalf("a v-prefixed tag against an unprefixed manifest must pass, got: %v", err)
	}
	if identity.EffectiveVersion != "1.0.0" {
		t.Errorf("EffectiveVersion = %q, want the normalized %q", identity.EffectiveVersion, "1.0.0")
	}
	if identity.Tag != "v1.0.0" {
		t.Errorf("Tag = %q, want %q — the version re-prefixed with exactly one v", identity.Tag, "v1.0.0")
	}
	// Diverged is a FIELD, checked against a byte-exact comparison rather than
	// recomputed by the assertion: a test that recomputed it could not catch a wrong rule.
	if identity.Diverged != (identity.ManifestName != identity.Coordinate) {
		t.Errorf("Diverged = %v but ManifestName(%q) != Coordinate(%q) is %v",
			identity.Diverged, identity.ManifestName, identity.Coordinate, identity.ManifestName != identity.Coordinate)
	}
	if identity.Diverged {
		t.Errorf("Diverged = true for an identical name and coordinate (%q)", identity.Coordinate)
	}
	if identity.InstallName != identity.ManifestName {
		t.Errorf("InstallName = %q, want it to alias ManifestName %q — the install path, backstop.yml key, lock key and asset root all read from this one field",
			identity.InstallName, identity.ManifestName)
	}
}

func TestValidateRemoteIdentity_UnprefixedBothSidesPasses(t *testing.T) {
	dir := writeIdentityPack(t, identityManifest(identityTestCoordinate, "1.0.0"))

	identity, err := validateIdentityUnchanged(t, identityTestCoordinate, "1.0.0", dir)
	if err != nil {
		t.Fatalf("an unprefixed effective version against an unprefixed manifest must pass, got: %v", err)
	}
	if identity.EffectiveVersion != "1.0.0" {
		t.Errorf("EffectiveVersion = %q, want %q", identity.EffectiveVersion, "1.0.0")
	}
	if identity.Tag != "v1.0.0" {
		t.Errorf("Tag = %q, want %q even though the effective version arrived unprefixed", identity.Tag, "v1.0.0")
	}
}

// TestValidateRemoteIdentity_PrefixedManifestPasses covers the side the other two do not:
// the MANIFEST carrying the prefix. One prefix normalizes on EACH side independently.
func TestValidateRemoteIdentity_PrefixedManifestPasses(t *testing.T) {
	dir := writeIdentityPack(t, identityManifest(identityTestCoordinate, "v1.0.0"))

	identity, err := validateIdentityUnchanged(t, identityTestCoordinate, "v1.0.0", dir)
	if err != nil {
		t.Fatalf("a v-prefixed manifest version against the same tag must pass, got: %v", err)
	}
	if identity.EffectiveVersion != "1.0.0" {
		t.Errorf("EffectiveVersion = %q, want the normalized %q", identity.EffectiveVersion, "1.0.0")
	}
}

// ── Version equality: the mismatches ────────────────────────────────────────────

func TestValidateRemoteIdentity_ManifestVersionMismatchIsTypedRefusal(t *testing.T) {
	dir := writeIdentityPack(t, identityManifest(identityTestCoordinate, "1.0.1"))

	_, err := validateIdentityUnchanged(t, identityTestCoordinate, "v1.0.0", dir)
	var mismatch *distribution.VersionMismatchError
	if !errors.As(err, &mismatch) {
		t.Fatalf("a manifest version differing from the effective version must refuse with *VersionMismatchError, got %v (%T)", err, err)
	}

	// ALL FOUR fields, because an operator seeing only "version mismatch" cannot tell
	// whether to retag the repository or request a different version.
	msg := err.Error()
	for _, want := range []string{identityTestCoordinate, "v1.0.0", "1.0.1"} {
		if !strings.Contains(msg, want) {
			t.Errorf("diagnostic %q does not name %q", msg, want)
		}
	}
	if mismatch.ManifestVersion != "1.0.1" {
		t.Errorf("ManifestVersion = %q, want %q", mismatch.ManifestVersion, "1.0.1")
	}
	if mismatch.Coordinate != identityTestCoordinate {
		t.Errorf("Coordinate = %q, want %q", mismatch.Coordinate, identityTestCoordinate)
	}
}

// TestValidateRemoteIdentity_HarnessPackVersionDriftRefuses is the PUBLISHED shape, not a
// synthetic one: the harness toolchain pack's manifest declares 0.1.3 while its tags stop
// at v0.1.1 (DIR-027 item 2). This check's first real customer is already broken in
// exactly this way.
func TestValidateRemoteIdentity_HarnessPackVersionDriftRefuses(t *testing.T) {
	const coordinate = "backstop-ai/backstop-harness-toolchain-pack"
	dir := writeIdentityPack(t, identityManifest("backstop/harness-toolchain", "0.1.3"))

	_, err := validateIdentityUnchanged(t, coordinate, "v0.1.1", dir)
	var mismatch *distribution.VersionMismatchError
	if !errors.As(err, &mismatch) {
		t.Fatalf("the published harness pack's drift must refuse with *VersionMismatchError, got %v (%T)", err, err)
	}

	// BOTH versions side by side. An operator debugging this pack needs to see 0.1.3 and
	// 0.1.1 together to know the fix is a RETAG of the repository, not a reinstall.
	msg := err.Error()
	if !strings.Contains(msg, "0.1.3") {
		t.Errorf("diagnostic %q does not name the manifest's declared 0.1.3", msg)
	}
	if !strings.Contains(msg, "0.1.1") {
		t.Errorf("diagnostic %q does not name the requested 0.1.1", msg)
	}
}

// ── Manifest defects ────────────────────────────────────────────────────────────

func TestValidateRemoteIdentity_MissingManifestVersionIsIdentityError(t *testing.T) {
	dir := writeIdentityPack(t, "name: "+identityTestCoordinate+"\nlanguage: neutral\narchetype: enforcement\n")

	_, err := validateIdentityUnchanged(t, identityTestCoordinate, "v1.0.0", dir)
	var identityErr *distribution.IdentityError
	if !errors.As(err, &identityErr) {
		t.Fatalf("a manifest declaring no version must refuse with *IdentityError, got %v (%T)", err, err)
	}
	if !strings.Contains(strings.ToLower(identityErr.Field), "version") {
		t.Errorf("IdentityError.Field = %q, want it to name the version field so the pack author knows what to fix", identityErr.Field)
	}
}

func TestValidateRemoteIdentity_NonStrictManifestVersionIsIdentityError(t *testing.T) {
	dir := writeIdentityPack(t, identityManifest(identityTestCoordinate, "1.0"))

	_, err := validateIdentityUnchanged(t, identityTestCoordinate, "v1.0.0", dir)
	var identityErr *distribution.IdentityError
	if !errors.As(err, &identityErr) {
		t.Fatalf("a manifest version of \"1.0\" must refuse with *IdentityError, got %v (%T)", err, err)
	}
}

// TestValidateRemoteIdentity_PrereleaseManifestVersionIsIdentityError carries the
// contrast a future reader needs.
//
// pkg/pack's semverPattern (manifest.go:460) ACCEPTS prerelease and build-metadata
// suffixes, so `version: 1.0.0-rc1` is a perfectly VALID manifest and `pack check` passes
// it. It is refused HERE because identity is narrower on purpose: no strict release tag
// can ever equal it, so such a pack is not installable BY TAG.
//
// Without that contrast stated, the natural "fix" for this test is to reuse
// pack.validateSemver — which would delete the check entirely.
func TestValidateRemoteIdentity_PrereleaseManifestVersionIsIdentityError(t *testing.T) {
	dir := writeIdentityPack(t, identityManifest(identityTestCoordinate, "1.0.0-rc1"))

	_, err := validateIdentityUnchanged(t, identityTestCoordinate, "v1.0.0", dir)
	var identityErr *distribution.IdentityError
	if !errors.As(err, &identityErr) {
		t.Fatalf("a prerelease manifest version must refuse with *IdentityError even though pkg/pack accepts it, got %v (%T)", err, err)
	}
}

func TestValidateRemoteIdentity_DoubledPrefixManifestVersionIsIdentityError(t *testing.T) {
	dir := writeIdentityPack(t, identityManifest(identityTestCoordinate, "vv1.0.0"))

	_, err := validateIdentityUnchanged(t, identityTestCoordinate, "v1.0.0", dir)
	var identityErr *distribution.IdentityError
	if !errors.As(err, &identityErr) {
		t.Fatalf("a doubled-prefix manifest version must refuse with *IdentityError, got %v (%T)", err, err)
	}
}

// TestValidateRemoteIdentity_UnreadableManifestIsIdentityError covers BOTH unreadable
// shapes, and requires the coordinate and tag in the message because at this point the
// pack.yml path is a temporary clone directory the operator cannot inspect — naming the
// path alone would be useless to them.
func TestValidateRemoteIdentity_UnreadableManifestIsIdentityError(t *testing.T) {
	cases := []struct {
		name     string
		manifest string
		absent   bool
	}{
		{name: "absent pack.yml", absent: true},
		{name: "unparseable pack.yml", manifest: "\x00\x01\x02 binary garbage\n\tname: [unclosed"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var dir string
			if tc.absent {
				dir = t.TempDir()
			} else {
				dir = writeIdentityPack(t, tc.manifest)
			}

			_, err := validateIdentityUnchanged(t, identityTestCoordinate, "v1.0.0", dir)
			var identityErr *distribution.IdentityError
			if !errors.As(err, &identityErr) {
				t.Fatalf("an %s must refuse with *IdentityError, got %v (%T)", tc.name, err, err)
			}
			msg := err.Error()
			if !strings.Contains(msg, identityTestCoordinate) {
				t.Errorf("diagnostic %q does not name the coordinate; the pack.yml path is a temp dir the operator cannot inspect", msg)
			}
			if !strings.Contains(msg, "v1.0.0") {
				t.Errorf("diagnostic %q does not name the tag that was cloned", msg)
			}
		})
	}
}

// ── Name validity ───────────────────────────────────────────────────────────────

func TestValidateRemoteIdentity_EmptyManifestNameIsIdentityError(t *testing.T) {
	cases := map[string]string{
		"absent name key":  "version: 1.0.0\nlanguage: neutral\narchetype: enforcement\n",
		"empty name value": "name: \"\"\nversion: 1.0.0\nlanguage: neutral\narchetype: enforcement\n",
	}

	for name, manifest := range cases {
		t.Run(name, func(t *testing.T) {
			dir := writeIdentityPack(t, manifest)

			_, err := validateIdentityUnchanged(t, identityTestCoordinate, "v1.0.0", dir)
			var identityErr *distribution.IdentityError
			if !errors.As(err, &identityErr) {
				t.Fatalf("a manifest with an %s must refuse with *IdentityError — a pack that cannot be addressed cannot be installed; got %v (%T)", name, err, err)
			}
			if !strings.Contains(strings.ToLower(identityErr.Field), "name") {
				t.Errorf("IdentityError.Field = %q, want it to name the name field", identityErr.Field)
			}
		})
	}
}

func TestValidateRemoteIdentity_UnqualifiedManifestNameIsIdentityError(t *testing.T) {
	dir := writeIdentityPack(t, identityManifest("validpack", "1.0.0"))

	_, err := validateIdentityUnchanged(t, identityTestCoordinate, "v1.0.0", dir)
	var identityErr *distribution.IdentityError
	if !errors.As(err, &identityErr) {
		t.Fatalf("a manifest name with no slash must refuse with *IdentityError, got %v (%T)", err, err)
	}
}

func TestValidateRemoteIdentity_EmptyNamePartIsIdentityError(t *testing.T) {
	for _, name := range []string{"hermetic/", "/valid-pack"} {
		t.Run(name, func(t *testing.T) {
			dir := writeIdentityPack(t, identityManifest(`"`+name+`"`, "1.0.0"))

			_, err := validateIdentityUnchanged(t, identityTestCoordinate, "v1.0.0", dir)
			var identityErr *distribution.IdentityError
			if !errors.As(err, &identityErr) {
				t.Fatalf("manifest name %q must refuse with *IdentityError, got %v (%T)", name, err, err)
			}
		})
	}
}

func TestValidateRemoteIdentity_InvalidNameCharactersIsIdentityError(t *testing.T) {
	dir := writeIdentityPack(t, identityManifest(`"hermetic/valid pack"`, "1.0.0"))

	_, err := validateIdentityUnchanged(t, identityTestCoordinate, "v1.0.0", dir)
	var identityErr *distribution.IdentityError
	if !errors.As(err, &identityErr) {
		t.Fatalf("a manifest name containing a space must refuse with *IdentityError, got %v (%T)", err, err)
	}
}

// ── Check ORDER, asserted rather than merely arranged ───────────────────────────

// TestValidateRemoteIdentity_VersionCheckPrecedesNameCheck feeds a manifest that is BOTH
// version-mismatched AND name-invalid, and requires the VERSION problem to be reported.
//
// The spec fixes the order: version available → version strict → manifest readable with a
// name → manifest version equals effective → name valid. Without a pack that fails two
// checks at once, an implementation could run them in any order and every other test in
// this file would still pass.
func TestValidateRemoteIdentity_VersionCheckPrecedesNameCheck(t *testing.T) {
	dir := writeIdentityPack(t, identityManifest(`"bad name/with spaces"`, "9.9.9"))

	_, err := validateIdentityUnchanged(t, identityTestCoordinate, "v1.0.0", dir)
	if err == nil {
		t.Fatal("a pack that is both version-mismatched and name-invalid must refuse")
	}

	var mismatch *distribution.VersionMismatchError
	if !errors.As(err, &mismatch) {
		t.Fatalf("want the VERSION diagnostic (the cheaper, more specific failure) to win the ordering, got %v (%T)", err, err)
	}
}

// ── Divergence is recorded, never a refusal ─────────────────────────────────────

// TestValidateRemoteIdentity_DivergentNameSetsDivergedAndInstallsUnderManifestName is the
// positive half of REQ-006 at the gate level: a name that differs from its coordinate is
// a DIAGNOSTIC, not a refusal, and the identity that comes back installs under the
// MANIFEST name while remembering the coordinate independently.
func TestValidateRemoteIdentity_DivergentNameSetsDivergedAndInstallsUnderManifestName(t *testing.T) {
	const (
		coordinate   = "backstop-ai/backstop-harness-toolchain-pack"
		manifestName = "backstop/harness-toolchain"
	)
	dir := writeIdentityPack(t, identityManifest(manifestName, "1.0.0"))

	identity, err := validateIdentityUnchanged(t, coordinate, "v1.0.0", dir)
	if err != nil {
		t.Fatalf("divergence must never refuse, got: %v", err)
	}
	if !identity.Diverged {
		t.Errorf("Diverged = false for coordinate %q vs manifest name %q", coordinate, manifestName)
	}
	if identity.InstallName != manifestName {
		t.Errorf("InstallName = %q, want the MANIFEST name %q — it is what builds the install path and the asset root", identity.InstallName, manifestName)
	}
	if identity.Coordinate != coordinate {
		t.Errorf("Coordinate = %q, want the requested %q recorded independently", identity.Coordinate, coordinate)
	}
}

// TestValidateRemoteIdentity_CaseOnlyDifferenceIsDivergent pins the byte-exactness of the
// divergence rule. strings.EqualFold or ToLower anywhere in this path would silently
// reintroduce the GitHub host assumption DD-31 removed — and it would do so while every
// other test in this file kept passing.
func TestValidateRemoteIdentity_CaseOnlyDifferenceIsDivergent(t *testing.T) {
	const (
		coordinate   = "Acme/Pack"
		manifestName = "acme/pack"
	)
	dir := writeIdentityPack(t, identityManifest(manifestName, "1.0.0"))

	identity, err := validateIdentityUnchanged(t, coordinate, "v1.0.0", dir)
	if err != nil {
		t.Fatalf("a case-only difference must not refuse, got: %v", err)
	}
	if !identity.Diverged {
		t.Errorf("Diverged = false for %q vs %q — comparison must be byte-exact; case-insensitivity is a GitHub property and packs may be hosted anywhere",
			coordinate, manifestName)
	}
}

// TestValidateRemoteIdentity_NonStrictEffectiveVersionIsTypedRefusal covers the gate's
// own guard on the version it is HANDED, which the ordinary pipeline can never trip
// because ResolveEffectiveVersion has already normalized it.
//
// It exists because the guard is real code with a real branch, and an uncovered branch in
// this file is indistinguishable from a claim whose test was forgotten. ValidateRemoteIdentity
// is exported, so a future caller can reach it without going through the resolver — and
// when one does, the refusal must be typed rather than a nil-deref on a half-built tag.
func TestValidateRemoteIdentity_NonStrictEffectiveVersionIsTypedRefusal(t *testing.T) {
	dir := writeIdentityPack(t, identityManifest(identityTestCoordinate, "1.0.0"))

	for _, effective := range []string{"1.0", "vv1.0.0", "1.0.0-rc1", ""} {
		t.Run("effective="+effective, func(t *testing.T) {
			_, err := validateIdentityUnchanged(t, identityTestCoordinate, effective, dir)
			var unresolved *distribution.VersionUnresolvedError
			if !errors.As(err, &unresolved) {
				t.Fatalf("effective version %q must refuse with *VersionUnresolvedError, got %v (%T)", effective, err, err)
			}
		})
	}
}
