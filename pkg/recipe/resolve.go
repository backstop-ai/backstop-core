package recipe

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmanson/backstop-core/pkg/pack"
)

// The two fail-loud markers resolution uses to keep "no such pack" and "that pack
// indexes no such recipe" DISTINCT diagnoses (CLM-046/CLM-047). They are exported
// nowhere and matched by the resolution tests so the two paths can never collapse
// into one indistinguishable message.
const (
	missingPackMarker      = "is not among the installed packs"
	undeclaredRecipeMarker = "declares no recipe"
)

// RecipeRef is a parsed `<pack>:<recipe>@<recipe_version>` reference (REQ-010).
// Every field is required: there is no unpinned form, so a RecipeRef in hand is
// always a fully qualified pin.
type RecipeRef struct {
	Pack    string
	Recipe  string
	Version string
}

// String renders the ref back into its canonical wire shape, so a diagnostic can
// echo what the operator wrote.
func (r RecipeRef) String() string {
	return fmt.Sprintf("%s:%s@%s", r.Pack, r.Recipe, r.Version)
}

// ResolvedRecipe is a ref resolved to its per-recipe DIRECTORY plus the parsed
// colocated recipe.yml, alongside the PACK ROOT that directory sits under.
//
// Both bases are retained because a recipe declares paths against BOTH, and they are
// not interchangeable: a payload and a fragment are COLOCATED with the recipe and
// resolve under Dir, while a transform's rule file is declared PACK-relative and
// resolves under PackDir — which is what lets one pack share a rule across several of
// its recipes. Resolving a pack-relative rule under Dir doubles the recipe segment and
// points at nothing. The applier itself contributes no path either way, which is what
// keeps core free of any layout knowledge.
type ResolvedRecipe struct {
	Ref      RecipeRef
	Dir      string
	PackDir  string
	Manifest *RecipeManifest
}

// ParseRecipeRef parses `<pack>:<recipe>@<recipe_version>`.
//
// The pin is MANDATORY and must be strict semver: there is no "latest", no
// default version, and no tolerance branch (CLM-049). Anything else is an error
// quoting the offending ref, never a best-effort resolution to whatever version
// happens to be on disk.
func ParseRecipeRef(raw string) (RecipeRef, error) {
	if strings.TrimSpace(raw) == "" {
		return RecipeRef{}, fmt.Errorf("recipe ref is empty; expected <pack>:<recipe>@<version>")
	}

	nameAndRecipe, version, pinned := strings.Cut(raw, "@")
	if !pinned {
		return RecipeRef{}, fmt.Errorf("recipe ref %q is unpinned; expected <pack>:<recipe>@<version> with an explicit semver version", raw)
	}
	if !recipeSemverRe.MatchString(version) {
		return RecipeRef{}, fmt.Errorf("recipe ref %q pins version %q, which must be semver MAJOR.MINOR.PATCH", raw, version)
	}

	packName, recipeID, separated := strings.Cut(nameAndRecipe, ":")
	if !separated {
		return RecipeRef{}, fmt.Errorf("recipe ref %q is missing the ':' separating the pack from the recipe id; expected <pack>:<recipe>@<version>", raw)
	}
	if strings.TrimSpace(packName) == "" {
		return RecipeRef{}, fmt.Errorf("recipe ref %q names an empty pack; expected <pack>:<recipe>@<version>", raw)
	}
	if strings.TrimSpace(recipeID) == "" {
		return RecipeRef{}, fmt.Errorf("recipe ref %q names an empty recipe id; expected <pack>:<recipe>@<version>", raw)
	}
	if strings.Contains(recipeID, ":") {
		return RecipeRef{}, fmt.Errorf("recipe ref %q has an extra ':' in its recipe id %q; expected <pack>:<recipe>@<version>", raw, recipeID)
	}

	return RecipeRef{Pack: packName, Recipe: recipeID, Version: version}, nil
}

// ResolveRecipe resolves a pinned ref against the supplied pack corpus (REQ-010).
//
// It looks the pack up in packs, reads that pack's `recipes:` index for the
// recipe id, joins packDir onto the INDEXED directory, parses the colocated
// recipe.yml, and requires the parsed manifest's declared version to equal the
// pin. Each of the four failure modes is a distinct fail-loud error naming what
// was missing or mismatched; none degrades into a partial or approximate result.
//
// This is the APPLY-TIME half only: nothing here guards a publish-time revision
// or compares against a previously applied revision.
func ResolveRecipe(ref RecipeRef, packs map[string]*pack.Manifest, packDir string) (*ResolvedRecipe, error) {
	if ref.Pack == "" || ref.Recipe == "" || !recipeSemverRe.MatchString(ref.Version) {
		return nil, fmt.Errorf("resolve recipe: ref %+v is not a fully pinned <pack>:<recipe>@<version> reference", ref)
	}

	manifest, installed := packs[ref.Pack]
	if !installed || manifest == nil {
		return nil, fmt.Errorf("resolve recipe %q: pack %q %s (installed: %v)", ref, ref.Pack, missingPackMarker, sortedPackNames(packs))
	}

	recipeDir, indexed := manifest.Recipes[ref.Recipe]
	if !indexed || recipeDir == "" {
		return nil, fmt.Errorf("resolve recipe %q: pack %q %s %q in its recipes: index (indexed: %v)", ref, ref.Pack, undeclaredRecipeMarker, ref.Recipe, sortedRecipeIDs(manifest.Recipes))
	}

	dir := filepath.Join(packDir, recipeDir)
	manifestPath := filepath.Join(dir, "recipe.yml")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("resolve recipe %q: read colocated recipe manifest %q: %w", ref, manifestPath, err)
	}
	parsed, err := ParseRecipeManifest(data)
	if err != nil {
		return nil, fmt.Errorf("resolve recipe %q: %w", ref, err)
	}

	if parsed.Version != ref.Version {
		return nil, fmt.Errorf("resolve recipe %q: ref pins version %q, but recipe %q in pack %q declares version %q", ref, ref.Version, ref.Recipe, ref.Pack, parsed.Version)
	}

	return &ResolvedRecipe{Ref: ref, Dir: dir, PackDir: packDir, Manifest: parsed}, nil
}

// sortedPackNames renders the corpus's pack names in a stable order so a
// missing-pack diagnostic reads the same on every run.
func sortedPackNames(packs map[string]*pack.Manifest) []string {
	names := make([]string, 0, len(packs))
	for name := range packs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// sortedRecipeIDs renders a pack's indexed recipe ids in a stable order so an
// undeclared-recipe diagnostic shows the operator what the pack DOES index.
func sortedRecipeIDs(index map[string]string) []string {
	ids := make([]string, 0, len(index))
	for id := range index {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
