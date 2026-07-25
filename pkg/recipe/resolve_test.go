package recipe

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack"
)

// fixturePackName is the demo pack in the committed corpus. Its pack.yml carries
// a `recipes:` index with two per-recipe DIRECTORIES (starter, second) alongside
// an independent content.scaffolds entry.
const fixturePackName = "demo-org/demo-pack"

// loadFixtureCorpus builds the packs map from the REAL pack manifest type by
// parsing the committed fixture pack.yml through pack.ParseManifestFile — not a
// hand-built stand-in — so resolution is exercised against the same value the
// production install path produces. It returns the corpus and the pack's root
// directory (the base every indexed recipe directory is joined onto).
func loadFixtureCorpus(t *testing.T) (map[string]*pack.Manifest, string) {
	t.Helper()

	packDir := filepath.Join("testdata", "packs", "demo-org", "demo-pack")
	manifest, err := pack.ParseManifestFile(filepath.Join(packDir, "pack.yml"))
	if err != nil {
		t.Fatalf("parse fixture pack manifest: %v", err)
	}
	if manifest.Name != fixturePackName {
		t.Fatalf("fixture pack name = %q, want %q", manifest.Name, fixturePackName)
	}
	if len(manifest.Recipes) == 0 {
		t.Fatalf("fixture pack declares no recipes: index; the corpus cannot exercise resolution")
	}

	return map[string]*pack.Manifest{manifest.Name: manifest}, packDir
}

// TestRecipeDir_ParsesColocatedManifestAndPayload proves the per-recipe DIRECTORY
// is what resolution yields (CLM-031): the result carries the recipe Dir and the
// parsed recipe.yml, AND the payload the manifest names is reachable RELATIVE to
// that Dir. Reading the payload off Dir is the colocation proof — a manifest-only
// read would pass a Dir assertion but fail this one.
func TestRecipeDir_ParsesColocatedManifestAndPayload(t *testing.T) {
	packs, packDir := loadFixtureCorpus(t)

	ref, err := ParseRecipeRef(fixturePackName + ":starter@1.2.0")
	if err != nil {
		t.Fatalf("ParseRecipeRef: unexpected error: %v", err)
	}

	resolved, err := ResolveRecipe(ref, packs, packDir)
	if err != nil {
		t.Fatalf("ResolveRecipe: unexpected error: %v", err)
	}

	wantDir := filepath.Join(packDir, "recipes", "starter")
	if resolved.Dir != wantDir {
		t.Errorf("resolved Dir = %q, want %q", resolved.Dir, wantDir)
	}
	if resolved.Manifest == nil {
		t.Fatalf("resolved Manifest is nil; the colocated recipe.yml was not parsed")
	}
	if resolved.Manifest.Kind != KindScaffolding {
		t.Errorf("resolved Manifest.Kind = %q, want %q", resolved.Manifest.Kind, KindScaffolding)
	}
	if resolved.Manifest.Version != "1.2.0" {
		t.Errorf("resolved Manifest.Version = %q, want %q", resolved.Manifest.Version, "1.2.0")
	}

	// The parsed ops must be the recipe directory's own, in DECLARED order.
	gotIDs := opIDs(resolved.Manifest.Ops)
	wantIDs := []string{"create-config", "merge-settings", "rename-config-key", "register-app", "confirm-adoption"}
	if len(gotIDs) != len(wantIDs) {
		t.Fatalf("resolved op ids = %v, want %v", gotIDs, wantIDs)
	}
	for i := range wantIDs {
		if gotIDs[i] != wantIDs[i] {
			t.Fatalf("resolved op ids = %v, want %v", gotIDs, wantIDs)
		}
	}

	// Colocation: the create op's payload path is recipe-directory-relative and
	// must be readable at Dir, with the recipe's own template markers intact.
	var payloadRel string
	for _, op := range resolved.Manifest.Ops {
		if op.Kind == OpCreate {
			payloadRel = op.Payload
			break
		}
	}
	if payloadRel == "" {
		t.Fatalf("fixture recipe declares no create op with a payload; colocation cannot be proven")
	}
	payloadBytes, err := os.ReadFile(filepath.Join(resolved.Dir, payloadRel))
	if err != nil {
		t.Fatalf("read colocated payload %q relative to resolved Dir %q: %v", payloadRel, resolved.Dir, err)
	}
	payload := string(payloadBytes)
	for _, want := range []string{"legacy_name", "{{ app_name }}", "registrations"} {
		if !strings.Contains(payload, want) {
			t.Errorf("colocated payload does not contain %q; got:\n%s", want, payload)
		}
	}
}

// TestResolveRef_ResolvesPinnedRecipe proves a pinned ref resolves through the
// pack's recipes index to that recipe's directory + parsed manifest (CLM-045),
// and that the parsed Ref round-trips Pack/Recipe/Version. Both fixture recipes
// are resolved so the lookup is proven to key on the recipe ID rather than
// returning whatever entry the index happens to yield first.
func TestResolveRef_ResolvesPinnedRecipe(t *testing.T) {
	packs, packDir := loadFixtureCorpus(t)

	cases := []struct {
		raw         string
		wantRecipe  string
		wantVersion string
		wantSubdir  string
		wantKind    string
	}{
		{
			raw:         fixturePackName + ":starter@1.2.0",
			wantRecipe:  "starter",
			wantVersion: "1.2.0",
			wantSubdir:  "starter",
			wantKind:    KindScaffolding,
		},
		{
			raw:         fixturePackName + ":second@2.0.1",
			wantRecipe:  "second",
			wantVersion: "2.0.1",
			wantSubdir:  "second",
			wantKind:    KindTemplating,
		},
	}

	for _, tc := range cases {
		t.Run(tc.raw, func(t *testing.T) {
			ref, err := ParseRecipeRef(tc.raw)
			if err != nil {
				t.Fatalf("ParseRecipeRef(%q): unexpected error: %v", tc.raw, err)
			}
			if ref.Pack != fixturePackName {
				t.Errorf("ref.Pack = %q, want %q", ref.Pack, fixturePackName)
			}
			if ref.Recipe != tc.wantRecipe {
				t.Errorf("ref.Recipe = %q, want %q", ref.Recipe, tc.wantRecipe)
			}
			if ref.Version != tc.wantVersion {
				t.Errorf("ref.Version = %q, want %q", ref.Version, tc.wantVersion)
			}

			resolved, err := ResolveRecipe(ref, packs, packDir)
			if err != nil {
				t.Fatalf("ResolveRecipe(%q): unexpected error: %v", tc.raw, err)
			}
			if resolved.Ref != ref {
				t.Errorf("resolved.Ref = %+v, want %+v", resolved.Ref, ref)
			}
			wantDir := filepath.Join(packDir, "recipes", tc.wantSubdir)
			if resolved.Dir != wantDir {
				t.Errorf("resolved Dir = %q, want %q", resolved.Dir, wantDir)
			}
			if resolved.Manifest == nil {
				t.Fatalf("resolved Manifest is nil")
			}
			if resolved.Manifest.Version != tc.wantVersion {
				t.Errorf("resolved Manifest.Version = %q, want %q", resolved.Manifest.Version, tc.wantVersion)
			}
			if resolved.Manifest.Kind != tc.wantKind {
				t.Errorf("resolved Manifest.Kind = %q, want %q", resolved.Manifest.Kind, tc.wantKind)
			}
			if len(resolved.Manifest.Ops) == 0 {
				t.Errorf("resolved Manifest declares no ops; the wrong recipe.yml was read")
			}
		})
	}
}

// TestResolveRef_MissingPackFailsLoud proves a ref naming a pack absent from the
// supplied corpus errors, naming the pack (CLM-046). Resolution never falls back
// to scanning the filesystem for a pack the corpus does not carry.
func TestResolveRef_MissingPackFailsLoud(t *testing.T) {
	packs, packDir := loadFixtureCorpus(t)

	ref, err := ParseRecipeRef("absent-org/absent-pack:starter@1.2.0")
	if err != nil {
		t.Fatalf("ParseRecipeRef: unexpected error: %v", err)
	}

	resolved, err := ResolveRecipe(ref, packs, packDir)
	if err == nil {
		t.Fatalf("ResolveRecipe with an absent pack returned no error (resolved = %+v)", resolved)
	}
	if resolved != nil {
		t.Errorf("ResolveRecipe returned a non-nil result alongside its error: %+v", resolved)
	}
	if !strings.Contains(err.Error(), "absent-org/absent-pack") {
		t.Errorf("missing-pack error does not name the pack: %v", err)
	}
	if !strings.Contains(err.Error(), missingPackMarker) {
		t.Errorf("missing-pack error does not carry the missing-pack marker %q: %v", missingPackMarker, err)
	}
}

// TestResolveRef_UndeclaredRecipeFailsLoud proves a recipe id absent from the
// pack's recipes index errors naming the id (CLM-047), with a message DISTINCT
// from the missing-pack one — the operator must be able to tell "no such pack"
// from "that pack indexes no such recipe".
func TestResolveRef_UndeclaredRecipeFailsLoud(t *testing.T) {
	packs, packDir := loadFixtureCorpus(t)

	undeclaredRef, parseErr := ParseRecipeRef(fixturePackName + ":nonexistent@1.2.0")
	if parseErr != nil {
		t.Fatalf("ParseRecipeRef: unexpected error: %v", parseErr)
	}
	resolved, undeclaredErr := ResolveRecipe(undeclaredRef, packs, packDir)
	if undeclaredErr == nil {
		t.Fatalf("ResolveRecipe with an undeclared recipe id returned no error (resolved = %+v)", resolved)
	}
	if resolved != nil {
		t.Errorf("ResolveRecipe returned a non-nil result alongside its error: %+v", resolved)
	}
	if !strings.Contains(undeclaredErr.Error(), "nonexistent") {
		t.Errorf("undeclared-recipe error does not name the recipe id: %v", undeclaredErr)
	}
	if !strings.Contains(undeclaredErr.Error(), undeclaredRecipeMarker) {
		t.Errorf("undeclared-recipe error does not carry the undeclared-recipe marker %q: %v", undeclaredRecipeMarker, undeclaredErr)
	}

	// Distinctness: the undeclared-recipe failure must not be reported using the
	// missing-pack wording, and vice versa.
	missingPackRef, missingParseErr := ParseRecipeRef("absent-org/absent-pack:starter@1.2.0")
	if missingParseErr != nil {
		t.Fatalf("ParseRecipeRef: unexpected error: %v", missingParseErr)
	}
	_, missingPackErr := ResolveRecipe(missingPackRef, packs, packDir)
	if missingPackErr == nil {
		t.Fatalf("ResolveRecipe with an absent pack returned no error")
	}
	if undeclaredErr.Error() == missingPackErr.Error() {
		t.Errorf("undeclared-recipe and missing-pack failures share one message: %v", undeclaredErr)
	}
	if strings.Contains(undeclaredErr.Error(), missingPackMarker) {
		t.Errorf("undeclared-recipe error reuses the missing-pack wording: %v", undeclaredErr)
	}
	if strings.Contains(missingPackErr.Error(), undeclaredRecipeMarker) {
		t.Errorf("missing-pack error reuses the undeclared-recipe wording: %v", missingPackErr)
	}
}

// TestResolveRef_NonexistentVersionFailsLoud proves a pin that does not match the
// recipe's DECLARED version errors naming both versions (CLM-048). Resolution
// never degrades a mismatched pin into "whatever version the directory holds".
func TestResolveRef_NonexistentVersionFailsLoud(t *testing.T) {
	packs, packDir := loadFixtureCorpus(t)

	ref, err := ParseRecipeRef(fixturePackName + ":starter@9.9.9")
	if err != nil {
		t.Fatalf("ParseRecipeRef: unexpected error: %v", err)
	}

	resolved, err := ResolveRecipe(ref, packs, packDir)
	if err == nil {
		t.Fatalf("ResolveRecipe with a mismatched pin returned no error (resolved = %+v)", resolved)
	}
	if resolved != nil {
		t.Errorf("ResolveRecipe returned a non-nil result alongside its error: %+v", resolved)
	}
	if !strings.Contains(err.Error(), "9.9.9") {
		t.Errorf("version-mismatch error does not name the pinned version: %v", err)
	}
	if !strings.Contains(err.Error(), "1.2.0") {
		t.Errorf("version-mismatch error does not name the declared version: %v", err)
	}
}

// TestResolveRef_UnpinnedRefFailsLoud proves the parser rejects every malformed
// or unpinned ref shape (CLM-049). An unpinned ref must NEVER silently resolve to
// "whatever version is there" — there is no latest, no default, no tolerance.
func TestResolveRef_UnpinnedRefFailsLoud(t *testing.T) {
	packs, packDir := loadFixtureCorpus(t)

	cases := []struct {
		name string
		raw  string
	}{
		{name: "no @ at all", raw: fixturePackName + ":starter"},
		{name: "@ with a non-semver tail", raw: fixturePackName + ":starter@latest"},
		{name: "@ with a partial-semver tail", raw: fixturePackName + ":starter@1.2"},
		{name: "@ with a v-prefixed tail", raw: fixturePackName + ":starter@v1.2.0"},
		{name: "@ with an empty tail", raw: fixturePackName + ":starter@"},
		{name: "missing :", raw: "demo-org-demo-pack-starter@1.2.0"},
		{name: "empty pack", raw: ":starter@1.2.0"},
		{name: "empty recipe", raw: fixturePackName + ":@1.2.0"},
		{name: "empty ref", raw: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ref, err := ParseRecipeRef(tc.raw)
			if err == nil {
				t.Fatalf("ParseRecipeRef(%q) returned no error (ref = %+v)", tc.raw, ref)
			}
			if ref != (RecipeRef{}) {
				t.Errorf("ParseRecipeRef(%q) returned a populated ref alongside its error: %+v", tc.raw, ref)
			}
			if !strings.Contains(err.Error(), tc.raw) && tc.raw != "" {
				t.Errorf("ParseRecipeRef(%q) error does not quote the offending ref: %v", tc.raw, err)
			}

			// The rejected ref must not resolve through some other path either.
			if resolved, resolveErr := ResolveRecipe(ref, packs, packDir); resolveErr == nil {
				t.Errorf("ResolveRecipe accepted the zero ref from malformed input %q: %+v", tc.raw, resolved)
			}
		})
	}
}
