package main

// SPEC-067 — the shared machinery behind the 62 mandated CI-recipe tests.
//
// This file carries NO mandated test name. It exists so the six test files that
// do can each read the REAL INSTALLED PACK rather than a committed copy under
// testdata/, and drive the SHIPPED root command rather than a stub.
//
// THE ONE SUBSTITUTION THAT MATTERS, relative to recipe_apply_e2e_test.go's
// shape (stagedRecipe / executeCommand / copyTree): the pack comes from
// `.backstop/packs/backstop-ai/ci-workflows/` — the directory `backstop pack
// install` writes — not from a fixture. A stub or a committed copy would pass
// whether or not the pack exists, which is precisely the failure mode SPEC-067
// is the acceptance test against.
//
// NOTHING IN THIS SUITE EVER SKIPS. The pack's absence is a LOUD failure. A skip
// here would recreate exactly the vacuous green the pack exists to prevent, and
// it is the single most likely way this whole suite silently stops testing
// anything.
//
// NO t.Parallel ANYWHERE IN THIS SUITE: ciStageConsumer calls t.Chdir, which is
// process-global.

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack"
	"github.com/backstop-ai/backstop-core/pkg/recipe"
	"gopkg.in/yaml.v3"
)

const (
	// ciPackName is the pack's MANIFEST NAME, which IS its install identity: the
	// install path, the backstop.yml key and the lock key all read it.
	ciPackName = "backstop-ai/ci-workflows"

	// The four recipe ids. They are the ONLY input that differs between the four
	// platform applies (REQ-008(b)), which is why they are collected here rather
	// than scattered as literals through four test files.
	ciRecipeGitHubActions      = "github-actions-gate"
	ciRecipeGitLabCI           = "gitlab-ci-gate"
	ciRecipeBitbucketPipelines = "bitbucket-pipelines-gate"
	ciRecipeJenkins            = "jenkins-gate"

	// ciProbeVersion is the distinctive `backstop_version` value the render
	// assertions supply. It is deliberately not a plausible release number: a
	// payload carrying a hardcoded default cannot masquerade as a substitution
	// when the value asserted is one no default would ever be.
	ciProbeVersion = "9.9.9"
)

// ciAllRecipeIDs is the declared four, in a stable order.
func ciAllRecipeIDs() []string {
	return []string{
		ciRecipeGitHubActions,
		ciRecipeGitLabCI,
		ciRecipeBitbucketPipelines,
		ciRecipeJenkins,
	}
}

// ciRepoRoot walks UP from the test's working directory until it finds the
// backstop.yml that marks the repository root.
//
// It must be called BEFORE any t.Chdir in the same test. Every helper below that
// chdirs (ciStageConsumer) resolves everything it needs from here FIRST, so a
// test never has to remember the ordering itself.
func ciRepoRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("resolve the test working directory: %v", err)
	}
	for {
		if _, statErr := os.Stat(filepath.Join(dir, "backstop.yml")); statErr == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("no backstop.yml found walking up from the test working directory; the repository root is unresolvable")
		}
		dir = parent
	}
}

// ciPackRoot returns the INSTALLED pack directory and FAILS LOUDLY when it, or
// its pack.yml, is absent — naming the resolved path and the fix.
func ciPackRoot(t *testing.T) string {
	t.Helper()

	root := filepath.Join(ciRepoRoot(t), ".backstop", "packs", filepath.FromSlash(ciPackName))
	manifestPath := filepath.Join(root, "pack.yml")
	if _, err := os.Stat(manifestPath); err != nil {
		t.Fatalf("installed pack not found at %s: %v\n"+
			"SPEC-067's claims are asserted against the REAL installed pack, never a committed copy.\n"+
			"Fix: declare %s in backstop.yml/backstop.lock and run `backstop pack install`.",
			manifestPath, err, ciPackName)
	}
	return root
}

// ciParsedPack parses the installed pack.yml, so every id, version, directory
// and target below is read from DATA rather than retyped as a test literal. A
// test that hardcodes a recipe version has broken the property that the pack can
// be revved without editing core.
func ciParsedPack(t *testing.T) *pack.Manifest {
	t.Helper()

	manifest, err := pack.ParseManifestFile(filepath.Join(ciPackRoot(t), "pack.yml"))
	if err != nil {
		t.Fatalf("parse the installed pack manifest: %v", err)
	}
	return manifest
}

// ciRecipeDir resolves one indexed recipe's directory inside the installed pack.
func ciRecipeDir(t *testing.T, recipeID string) string {
	t.Helper()

	manifest := ciParsedPack(t)
	indexed, declared := manifest.Recipes[recipeID]
	if !declared {
		t.Fatalf("pack %s indexes no recipe %q (indexed: %v)", ciPackName, recipeID, manifest.Recipes)
	}
	return filepath.Join(ciPackRoot(t), filepath.FromSlash(indexed))
}

// ciRecipeManifest parses one indexed recipe's recipe.yml.
func ciRecipeManifest(t *testing.T, recipeID string) *recipe.RecipeManifest {
	t.Helper()

	path := filepath.Join(ciRecipeDir(t, recipeID), "recipe.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read recipe manifest %s: %v", path, err)
	}
	parsed, parseErr := recipe.ParseRecipeManifest(data)
	if parseErr != nil {
		t.Fatalf("parse recipe manifest %s: %v", path, parseErr)
	}
	return parsed
}

// ciPayloadBytes reads one recipe op's declared payload out of the installed
// pack — the SOURCE template, before substitution.
func ciPayloadBytes(t *testing.T, recipeID string, payload string) []byte {
	t.Helper()

	path := filepath.Join(ciRecipeDir(t, recipeID), filepath.FromSlash(payload))
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read payload %s: %v", path, err)
	}
	return body
}

// ciStageConsumer copies the INSTALLED pack into a fresh t.TempDir() at
// .backstop/packs/<pack name>, writes a minimal backstop.yml declaring it, and
// t.Chdir's into that root.
//
// recipeProjectRoot() resolves the project from the DISCOVERED backstop.yml, so
// the apply writes into the scratch tree and never into the working repository.
// Everything read out of the real repository is resolved BEFORE the chdir.
func ciStageConsumer(t *testing.T) string {
	t.Helper()

	installed := ciPackRoot(t)
	version := ciParsedPack(t).Version

	root := t.TempDir()
	staged := filepath.Join(root, ".backstop", "packs", filepath.FromSlash(ciPackName))
	if err := os.MkdirAll(staged, 0o755); err != nil {
		t.Fatalf("create the staged pack directory: %v", err)
	}
	copyTree(t, installed, staged)

	config := fmt.Sprintf("project: ci-recipes-scratch\npacks:\n    %s: %s\n", ciPackName, version)
	if err := os.WriteFile(filepath.Join(root, "backstop.yml"), []byte(config), 0o644); err != nil {
		t.Fatalf("write the scratch project config: %v", err)
	}

	t.Chdir(root)
	return root
}

// ciApply drives the SHIPPED root command with the ONE invocation shape all four
// platforms share: `recipe apply <pack>:<id>@<version> --param k=v ...`. The
// reference argument is the only input that differs between platforms.
//
// It returns stdout AND the error, because CLM-049 must inspect both.
func ciApply(t *testing.T, root string, recipeID string, version string, params ...string) (string, error) {
	t.Helper()
	t.Chdir(root)

	args := []string{"recipe", "apply", ciApplyRef(recipeID, version)}
	for _, param := range params {
		args = append(args, "--param", param)
	}

	out := new(bytes.Buffer)
	cmd := NewRootCommand()
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

// ciApplyRef builds the fully pinned reference. There is no "latest" form.
func ciApplyRef(recipeID string, version string) string {
	return ciPackName + ":" + recipeID + "@" + version
}

// ciApplyProbe is the shape every render assertion uses: stage a consumer, apply
// one recipe at its OWN declared version with the distinctive probe version, and
// return the project root, the rendered target's path and its bytes.
func ciApplyProbe(t *testing.T, recipeID string) (string, string, []byte) {
	t.Helper()

	manifest := ciRecipeManifest(t, recipeID)
	if len(manifest.Ops) != 1 {
		t.Fatalf("recipe %q declares %d ops, want exactly 1", recipeID, len(manifest.Ops))
	}
	target := manifest.Ops[0].Target
	version := manifest.Version

	root := ciStageConsumer(t)
	out, err := ciApply(t, root, recipeID, version, "backstop_version="+ciProbeVersion)
	if err != nil {
		t.Fatalf("apply %s failed: %v\noutput:\n%s", ciApplyRef(recipeID, version), err, out)
	}

	path := filepath.Join(root, filepath.FromSlash(target))
	rendered, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("read the rendered target %s: %v\napply output:\n%s", path, readErr, out)
	}
	return root, path, rendered
}

// ciBasenameMatch models semgrep's slashless-`paths: include:` semantics.
//
// SHARP EDGE 10 — filepath.Match does NOT reproduce those semantics. semgrep
// matches a slashless include against the file's BASENAME anywhere in the tree,
// whereas filepath.Match requires the WHOLE path to match. Measured:
//
//	filepath.Match("backstop-gate*.yml", ".github/workflows/backstop-gate.yml")
//	  => false, nil
//	filepath.Match("backstop-gate*.yml", filepath.Base(".github/workflows/backstop-gate.yml"))
//	  => true, nil
//
// Written the naive way, "the glob matches the deployed target" FAILS against a
// CORRECT pack — and the likely repair, widening the mandated pattern until
// filepath.Match accepts a full path, replaces a working include with one
// semgrep matches nothing against (Sharp Edge 8).
//
// THIS HELPER IS A MODEL OF SEMGREP'S BEHAVIOUR, NOT THE BEHAVIOUR ITSELF. The
// only authority is semgrep, which is what the pack's scripts/falsify.sh runs.
func ciBasenameMatch(t *testing.T, pattern string, path string) bool {
	t.Helper()

	matched, err := filepath.Match(pattern, filepath.Base(filepath.FromSlash(path)))
	if err != nil {
		t.Fatalf("include pattern %q is malformed: %v", pattern, err)
	}
	return matched
}

// ciByteOffset returns the index of the SINGLE occurrence of needle in content,
// failing when the count is not exactly 1 — so an ordering claim asserts the
// COUNT and the ORDER in one place.
func ciByteOffset(t *testing.T, content []byte, needle string) int {
	t.Helper()

	count := bytes.Count(content, []byte(needle))
	if count != 1 {
		t.Fatalf("%q appears %d time(s) in the rendered file, want exactly 1\nrendered:\n%s", needle, count, content)
	}
	return bytes.Index(content, []byte(needle))
}

// ciTrackedFiles lists every file git tracks in backstop-core, for CLM-006's
// union check. It must be called before any chdir.
func ciTrackedFiles(t *testing.T) []string {
	t.Helper()

	cmd := exec.Command("git", "ls-files")
	cmd.Dir = ciRepoRoot(t)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git ls-files in the repository root: %v", err)
	}
	files := []string{}
	for _, line := range strings.Split(string(out), "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			files = append(files, trimmed)
		}
	}
	if len(files) == 0 {
		t.Fatalf("git ls-files returned nothing; the union check would pass vacuously")
	}
	return files
}

// ciGitShow reads one object out of git — used to compare a tracked file against
// its committed content without keeping a second copy that could drift.
func ciGitShow(t *testing.T, repoRoot string, object string) []byte {
	t.Helper()

	cmd := exec.Command("git", "show", object)
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git show %s: %v", object, err)
	}
	return out
}

// ciRuleFile is the slice of a semgrep rule document these claims assert on: the
// declared rule ids and each rule's `paths: include:` set.
type ciRuleFile struct {
	Rules []struct {
		ID    string `yaml:"id"`
		Paths struct {
			Include []string `yaml:"include"`
			Exclude []string `yaml:"exclude"`
		} `yaml:"paths"`
		Severity  string   `yaml:"severity"`
		Languages []string `yaml:"languages"`
	} `yaml:"rules"`
}

// ciParseRuleFile parses one of the pack's semgrep rule files.
func ciParseRuleFile(t *testing.T, packRelPath string) ciRuleFile {
	t.Helper()

	path := filepath.Join(ciPackRoot(t), filepath.FromSlash(packRelPath))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read rule file %s: %v", path, err)
	}
	parsed := ciRuleFile{}
	if unmarshalErr := yaml.Unmarshal(data, &parsed); unmarshalErr != nil {
		t.Fatalf("parse rule file %s: %v", path, unmarshalErr)
	}
	if len(parsed.Rules) == 0 {
		t.Fatalf("rule file %s declares no rules", path)
	}
	return parsed
}

// ciRulePathsFor returns the DISTINCT rule-file paths the pack's ruleset declares
// for the rule ids a given recipe's enforcement list names.
func ciRulePathsFor(t *testing.T, recipeID string) []string {
	t.Helper()

	manifest := ciParsedPack(t)
	declared := map[string]string{}
	for _, rule := range manifest.Content.Ruleset.Rules {
		declared[rule.ID] = rule.RulePath
	}

	enforcement := ciRecipeManifest(t, recipeID).Enforcement
	if enforcement == nil || len(enforcement.Rules) == 0 {
		t.Fatalf("recipe %q declares no enforcement.rules", recipeID)
	}

	seen := map[string]bool{}
	paths := []string{}
	for _, id := range enforcement.Rules {
		rulePath, resolved := declared[id]
		if !resolved {
			t.Fatalf("recipe %q names enforcement rule %q, which the pack ruleset does not declare", recipeID, id)
		}
		if rulePath == "" {
			t.Fatalf("pack ruleset declares rule %q with no rule_path", id)
		}
		if !seen[rulePath] {
			seen[rulePath] = true
			paths = append(paths, rulePath)
		}
	}
	sort.Strings(paths)
	return paths
}

// ciIncludeSetFor returns the SORTED, DEDUPLICATED union of `paths: include:`
// patterns across every rule the named recipe enforces.
func ciIncludeSetFor(t *testing.T, recipeID string) []string {
	t.Helper()

	enforced := map[string]bool{}
	for _, id := range ciRecipeManifest(t, recipeID).Enforcement.Rules {
		enforced[id] = true
	}

	seen := map[string]bool{}
	patterns := []string{}
	for _, rulePath := range ciRulePathsFor(t, recipeID) {
		for _, rule := range ciParseRuleFile(t, rulePath).Rules {
			if !enforced[rule.ID] {
				continue
			}
			if len(rule.Paths.Include) == 0 {
				t.Fatalf("rule %q in %s declares no `paths: include:` set; an unscoped rule polices files no recipe wrote", rule.ID, rulePath)
			}
			for _, pattern := range rule.Paths.Include {
				if !seen[pattern] {
					seen[pattern] = true
					patterns = append(patterns, pattern)
				}
			}
		}
	}
	sort.Strings(patterns)
	return patterns
}

// ciAllIncludePatterns is the union of every include pattern in every rule file
// the pack ships — the set CLM-006 runs over `git ls-files`.
func ciAllIncludePatterns(t *testing.T) map[string]string {
	t.Helper()

	patterns := map[string]string{}
	for _, rule := range ciParsedPack(t).Content.Ruleset.Rules {
		if rule.RulePath == "" {
			t.Fatalf("pack ruleset declares rule %q with no rule_path", rule.ID)
		}
		for _, declared := range ciParseRuleFile(t, rule.RulePath).Rules {
			if len(declared.Paths.Include) == 0 {
				t.Fatalf("rule %q in %s declares no `paths: include:` set", declared.ID, rule.RulePath)
			}
			for _, pattern := range declared.Paths.Include {
				patterns[pattern] = declared.ID
			}
		}
	}
	if len(patterns) == 0 {
		t.Fatalf("the pack declares no include patterns at all; the neutrality check would pass vacuously")
	}
	return patterns
}

// ciFixturePathsFor returns every fixture path (positive and negative) the
// pack's ruleset declares for the rules the named recipe enforces.
func ciFixturePathsFor(t *testing.T, recipeID string) []string {
	t.Helper()

	enforced := map[string]bool{}
	for _, id := range ciRecipeManifest(t, recipeID).Enforcement.Rules {
		enforced[id] = true
	}

	seen := map[string]bool{}
	paths := []string{}
	for _, rule := range ciParsedPack(t).Content.Ruleset.Rules {
		if !enforced[rule.ID] {
			continue
		}
		for _, claim := range rule.Claims {
			for _, fixture := range append(claim.Fixtures.Positive, claim.Fixtures.Negative...) {
				if fixture.Path != "" && !seen[fixture.Path] {
					seen[fixture.Path] = true
					paths = append(paths, fixture.Path)
				}
			}
		}
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		t.Fatalf("recipe %q's rules declare no fixtures at all", recipeID)
	}
	return paths
}

// ciOtherPlatformTargets returns the DECLARED targets of the three recipes that
// are NOT recipeID, so a scoping claim can assert it matches none of them.
func ciOtherPlatformTargets(t *testing.T, recipeID string) []string {
	t.Helper()

	targets := []string{}
	for _, id := range ciAllRecipeIDs() {
		if id == recipeID {
			continue
		}
		for _, op := range ciRecipeManifest(t, id).Ops {
			if op.Target != "" {
				targets = append(targets, op.Target)
			}
		}
	}
	sort.Strings(targets)
	return targets
}

// ciGlobScopingProblems REPORTS every way one platform's include set departs
// from CLM-053..056.
//
// wantInclude is the EXACT set the spec fixes for this platform. The comparison
// is over the WHOLE set, so a "tightened" multi-segment pattern added alongside
// is a failure: under the gate's DEFAULT diff-scoped dispatch, which hands
// semgrep EXPLICIT FILE targets, a multi-segment include matches ZERO files —
// not even the file it names. Since ISSUE-091 collapsed every scope carrying a
// file list onto that one explicit-file dispatch, `--all` has no directory
// dispatch left either, so a rule tightened that way matches ZERO files under
// EVERY such scope and is uniformly dead — it does not even have the misleading
// full-sweep liveness it had before.
func ciGlobScopingProblems(t *testing.T, recipeID string, wantInclude []string) []string {
	t.Helper()

	got := ciIncludeSetFor(t, recipeID)
	want := append([]string{}, wantInclude...)
	sort.Strings(want)
	if strings.Join(got, "|") != strings.Join(want, "|") {
		return []string{"declares the include set [" + strings.Join(got, " ") +
			"], want exactly [" + strings.Join(want, " ") + "]"}
	}

	problems := []string{}
	for _, pattern := range got {
		if strings.Contains(pattern, "/") {
			problems = append(problems, "include pattern "+pattern+
				" is multi-segment; semgrep matches ZERO files against it under the gate's default explicit-file-target dispatch")
		}
	}

	target := ciRecipeManifest(t, recipeID).Ops[0].Target
	if !ciAnyPatternMatches(t, got, target) {
		problems = append(problems, "the include set matches none of its own deployed target "+target)
	}
	for _, fixture := range ciFixturePathsFor(t, recipeID) {
		if !ciAnyPatternMatches(t, got, fixture) {
			problems = append(problems, "the include set does not match its own fixture "+fixture+
				"; the fixture would be filtered out the day packval phase 3 executes fixtures")
		}
	}
	for _, other := range ciOtherPlatformTargets(t, recipeID) {
		if ciAnyPatternMatches(t, got, other) {
			problems = append(problems, "the include set matches another platform's target "+other+"; the scoping is not tight")
		}
	}
	for _, file := range ciTrackedFiles(t) {
		if ciAnyPatternMatches(t, got, file) {
			problems = append(problems, "the include set matches the tracked backstop-core file "+file)
		}
	}
	return problems
}

// ciAnyPatternMatches reports whether any pattern in the set matches path's
// basename.
func ciAnyPatternMatches(t *testing.T, patterns []string, path string) bool {
	t.Helper()

	for _, pattern := range patterns {
		if ciBasenameMatch(t, pattern, path) {
			return true
		}
	}
	return false
}

// ciFileSet snapshots every file beneath root, project-relative, so an
// only-its-declared-target claim can diff before against after at ANY depth.
func ciFileSet(t *testing.T, root string) map[string]bool {
	t.Helper()

	files := map[string]bool{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		files[filepath.ToSlash(rel)] = true
		return nil
	})
	if err != nil {
		t.Fatalf("snapshot the staged project file set: %v", err)
	}
	return files
}

// ciAddedFiles reports what appeared between two ciFileSet snapshots, sorted.
func ciAddedFiles(before map[string]bool, after map[string]bool) []string {
	added := []string{}
	for path := range after {
		if !before[path] {
			added = append(added, path)
		}
	}
	sort.Strings(added)
	return added
}

// ciAppliedFileDelta applies one recipe into a fresh staged consumer and
// RETURNS what the apply ADDED alongside what it was allowed to add — the
// declared target plus the adoption record, and nothing else at any depth.
func ciAppliedFileDelta(t *testing.T, recipeID string) ([]string, []string) {
	t.Helper()

	manifest := ciRecipeManifest(t, recipeID)
	if len(manifest.Ops) != 1 {
		t.Fatalf("recipe %q declares %d ops, want exactly 1", recipeID, len(manifest.Ops))
	}
	target := filepath.ToSlash(manifest.Ops[0].Target)

	root := ciStageConsumer(t)
	before := ciFileSet(t, root)

	out, err := ciApply(t, root, recipeID, manifest.Version, "backstop_version="+ciProbeVersion)
	if err != nil {
		t.Fatalf("apply %s failed: %v\noutput:\n%s", recipeID, err, out)
	}

	want := []string{recipe.AdoptionRecordName, target}
	sort.Strings(want)
	return ciAddedFiles(before, ciFileSet(t, root)), want
}

// ciSwallowFormsFound REPORTS the swallow forms present in rendered bytes; it
// does not assert. Every mandated test carries its own assertion, so the test
// that proves a claim is the place the claim's failure is raised — a helper
// that both computed and asserted left each mandated test body with nothing in
// it, which is indistinguishable from a hollow test to anything reading the
// source.
//
// The denylist is passed in per platform, verbatim from that platform's claim,
// because a rule and its claim asserting different sets is silent
// disagreement.
func ciSwallowFormsFound(rendered []byte, forms []string) []string {
	text := string(rendered)
	found := []string{}

	for _, form := range forms {
		if strings.Contains(text, form) {
			found = append(found, "carries the swallow form "+form)
		}
	}
	// A TRAILING `exit 0` is a STATEMENT, not a substring: `exit 0` inside a
	// comment swallows nothing, and the claim is about the former.
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") {
			continue
		}
		if trimmed == "exit 0" || strings.HasSuffix(trimmed, "; exit 0") || strings.HasSuffix(trimmed, "&& exit 0") {
			found = append(found, "carries a trailing `exit 0` statement: "+trimmed)
		}
	}
	return found
}

// ciToolchainDenylist is REQ-010's language-runtime / package-manager denylist,
// identical for all four platforms.
func ciToolchainDenylist() []string {
	return []string{
		"actions/setup-", "go install", "go build", "npm ", "pnpm ", "yarn ",
		"pip ", "bundle install", "cargo ", "mvn ", "gradle",
	}
}

// ciBackstopReleaseCoordinate is the ONE download literal REQ-010 permits. It is
// the coordinate .goreleaser.yml actually publishes against.
const ciBackstopReleaseCoordinate = "backstop-ai/backstop-core"

// ciTravelProblems REPORTS what stops a rendered file from travelling: a
// language-runtime/package-manager literal, or an organization-or-repository
// literal beyond what this platform permits.
//
// permittedOrgRepo lists the org/repo literals this platform may carry BEYOND
// the backstop release coordinate. For the three non-GitHub platforms it is
// EMPTY, and that emptiness is the claim.
func ciTravelProblems(rendered []byte, permittedOrgRepo []string) []string {
	text := string(rendered)
	problems := []string{}

	for _, banned := range ciToolchainDenylist() {
		if strings.Contains(text, banned) {
			problems = append(problems, "carries the toolchain literal "+banned+"; a generic template must install no language runtime")
		}
	}

	for _, literal := range ciOrgRepoLiterals(text) {
		permitted := false
		for _, allowed := range permittedOrgRepo {
			// Any version suffix is permitted on a permitted literal
			// (`actions/checkout@v4`), and nothing else is.
			if literal == allowed || strings.HasPrefix(literal, allowed+"@") {
				permitted = true
				break
			}
		}
		if !permitted {
			problems = append(problems, "carries the organization-or-repository literal "+literal+
				", which is neither the backstop release coordinate nor one of "+strings.Join(permittedOrgRepo, ","))
		}
	}
	return problems
}

// ciOrgRepoLiterals extracts every ORGANIZATION-OR-REPOSITORY literal from
// rendered text, so REQ-010's permitted-literal check is decidable from the
// bytes rather than from a judgment about what "looks like" an action
// reference.
//
// THE SCAN IS TOKEN-BASED, AND THE THREE EXCLUSIONS BELOW ARE WHAT MAKE IT
// DECIDE THE RIGHT THING. A naive "any `<word>/<word>` span" reading — the
// shape REQ-010 states informally — flags things that are plainly not
// repository names, and every one of them appears in these payloads:
//
//   - the backstop RELEASE DOWNLOAD COORDINATE, which REQ-010 permits IN FULL.
//     Left unmasked, one URL yields `github.com/backstop-ai`,
//     `backstop-core/releases`, `releases/download` and `download/v9.9.9` —
//     four "violations" for the one literal the requirement explicitly allows.
//   - FILESYSTEM PATHS: `./backstop` (the extracted CLI) and `/dev/null` (a
//     redirect target) are not organizations. A leading `.`, `..` or `/`
//     segment is the mechanical mark of a path.
//   - SHELL EXPANSIONS: `origin/$BASE_BRANCH` names a git ref built at
//     runtime, not a repository. A `$` disqualifies the token.
//
// What survives is exactly the class the requirement is about: a bare
// `org/repo` (with an optional `@version` suffix), which is how an action, a
// pipe, a plugin coordinate or a consumer's own repository would appear. A
// second `actions/*`, or any `myorg/myrepo`, is still caught.
func ciOrgRepoLiterals(text string) []string {
	found := map[string]bool{}

	for _, raw := range strings.FieldsFunc(text, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\n' || r == '\r' ||
			r == '"' || r == '\'' || r == '`' || r == ',' || r == ';'
	}) {
		// The release coordinate is permitted IN FULL, so the whole token
		// carrying it is dropped before anything is extracted from it.
		if strings.Contains(raw, ciBackstopReleaseCoordinate) {
			continue
		}
		token := strings.Trim(raw, "()[]{}<>:.")
		if !strings.Contains(token, "/") || strings.Contains(token, "$") {
			continue
		}
		if strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "./") ||
			strings.HasPrefix(raw, "../") || strings.HasPrefix(raw, "~") ||
			strings.Contains(raw, ">/") {
			continue
		}
		segments := strings.Split(token, "/")
		if len(segments) != 2 || segments[0] == "" || segments[1] == "" {
			continue
		}
		if !ciIsCoordinateSegment(segments[0]) || !ciIsCoordinateSegment(segments[1]) {
			continue
		}
		found[token] = true
	}

	literals := []string{}
	for literal := range found {
		literals = append(literals, literal)
	}
	sort.Strings(literals)
	return literals
}

// ciIsCoordinateSegment reports whether one half of a candidate looks like an
// organization or repository name: word characters, with `@` tolerated so a
// pinned `actions/checkout@v4` is recognized rather than skipped.
func ciIsCoordinateSegment(segment string) bool {
	if segment == "." || segment == ".." {
		return false
	}
	for index := 0; index < len(segment); index++ {
		char := segment[index]
		switch {
		case char >= 'a' && char <= 'z', char >= 'A' && char <= 'Z',
			char >= '0' && char <= '9',
			char == '.', char == '-', char == '_', char == '@':
		default:
			return false
		}
	}
	return segment != ""
}

// ciReapplyProblems drives a SECOND apply with identical params and REPORTS
// what a deterministic, regenerate-by-default recipe would not do.
//
// preserveOrRegenerate short-circuits on byte-equality and returns Final: true
// WITHOUT writing, which the CLI reports as "nothing was written or preserved".
// Both halves matter: identical bytes catch nondeterministic rendering, and the
// short-circuit message catches a re-apply that rewrote the file to the same
// content (which is not the same property).
func ciReapplyProblems(t *testing.T, recipeID string) []string {
	t.Helper()

	manifest := ciRecipeManifest(t, recipeID)
	target := manifest.Ops[0].Target

	root := ciStageConsumer(t)
	firstOut, firstErr := ciApply(t, root, recipeID, manifest.Version, "backstop_version="+ciProbeVersion)
	if firstErr != nil {
		t.Fatalf("first apply of %s failed: %v\noutput:\n%s", recipeID, firstErr, firstOut)
	}
	path := filepath.Join(root, filepath.FromSlash(target))
	first, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("read the first rendered target: %v", readErr)
	}

	secondOut, secondErr := ciApply(t, root, recipeID, manifest.Version, "backstop_version="+ciProbeVersion)
	if secondErr != nil {
		t.Fatalf("re-apply of %s failed: %v\noutput:\n%s", recipeID, secondErr, secondOut)
	}
	second, reReadErr := os.ReadFile(path)
	if reReadErr != nil {
		t.Fatalf("read the re-applied target: %v", reReadErr)
	}

	problems := []string{}
	if !bytes.Equal(first, second) {
		problems = append(problems, "the re-apply CHANGED the target's bytes, so rendering is not deterministic")
	}
	const shortCircuit = "nothing was written or preserved"
	if !strings.Contains(secondOut, shortCircuit) {
		problems = append(problems, "the second apply did not report "+shortCircuit+
			"; the byte-equality short-circuit did not fire. output:\n"+secondOut)
	}
	return problems
}

// ciResidualPlaceholders REPORTS any placeholder delimiter that survived
// substitution and reached the consumer's filesystem.
func ciResidualPlaceholders(rendered []byte) []string {
	text := string(rendered)
	found := []string{}
	if strings.Contains(text, "{{") {
		found = append(found, "carries a residual `{{`")
	}
	if strings.Contains(text, "}}") {
		found = append(found, "carries a residual `}}`")
	}
	return found
}

// ciPinnedInstallProblems REPORTS what stops the rendered install from being
// pinned to the version the apply actually supplied.
func ciPinnedInstallProblems(rendered []byte) []string {
	text := string(rendered)
	problems := []string{}

	if !strings.Contains(text, ciProbeVersion) {
		problems = append(problems, "does not carry the supplied backstop_version "+ciProbeVersion+
			"; the install is not pinned to what the apply asked for")
	}
	if !strings.Contains(text, "releases/download/v"+ciProbeVersion+"/") {
		problems = append(problems, "does not install from the release archive at v"+ciProbeVersion)
	}
	for _, unpinned := range []string{"releases/latest", "/latest/download", "@latest"} {
		if strings.Contains(text, unpinned) {
			problems = append(problems, "carries the unpinned install form "+unpinned)
		}
	}
	return problems
}

// ciRecipeShapeProblems REPORTS every way one recipe's declaration departs from
// REQ-003: scaffolding kind, its OWN semver distinct from the pack's, a
// non-empty enforcement.rules, and EXACTLY ONE `create` op at the declared
// target.
func ciRecipeShapeProblems(t *testing.T, recipeID string, wantTarget string) []string {
	t.Helper()

	manifest := ciRecipeManifest(t, recipeID)
	problems := []string{}

	if manifest.Kind != recipe.KindScaffolding {
		problems = append(problems, "declares kind "+manifest.Kind+", want "+recipe.KindScaffolding)
	}
	if manifest.Version == "" {
		problems = append(problems, "declares no version")
	}
	if manifest.Version == ciParsedPack(t).Version {
		problems = append(problems, "its version "+manifest.Version+" equals the pack version; REQ-003 requires its OWN semver")
	}
	if manifest.Enforcement == nil || len(manifest.Enforcement.Rules) == 0 {
		problems = append(problems, "declares no enforcement.rules; a scaffolding recipe with no paired enforcement is a suggestion")
	}
	if len(manifest.Ops) != 1 {
		problems = append(problems, fmt.Sprintf("declares %d ops, want exactly 1", len(manifest.Ops)))
		return problems
	}

	op := manifest.Ops[0]
	if op.Kind != recipe.OpCreate {
		problems = append(problems, "its only op is kind "+op.Kind+", want "+recipe.OpCreate+
			"; merge/transform/insert/step would put a recipe-owned promise inside consumer-owned bytes")
	}
	if op.Target != wantTarget {
		problems = append(problems, "targets "+op.Target+", want "+wantTarget)
	}
	return problems
}

// ciBaseResolutionProblems REPORTS every way the rendered file fails REQ-004(e):
// a base variable assigned from the platform's OWN environment plus
// default_branch, the SAME variable passed to `backstop gate --base`, and a
// non-zero exit when the base does not resolve.
//
// The variable NAME is checked to match between assignment and use. A base
// resolved into one name and a gate reading another is exactly the silent
// degradation this claim exists for, and it is invisible to any check that only
// asks whether both strings appear.
func ciBaseResolutionProblems(t *testing.T, rendered []byte, platformEnvVars []string, defaultBranchValue string) []string {
	t.Helper()

	text := string(rendered)
	problems := []string{}

	assigned := ciAssignedBaseVariable(text)
	if assigned == "" {
		return []string{"assigns no shell variable that is later passed to --base"}
	}
	used := "--base \"$" + assigned + "\""
	if !strings.Contains(text, used) {
		problems = append(problems, "assigns "+assigned+" but does not pass it as "+used+
			"; the gate would read a different variable than the resolution wrote")
	}
	for _, envVar := range platformEnvVars {
		if !strings.Contains(text, envVar) {
			problems = append(problems, "does not read the platform environment variable "+envVar+" when resolving the base")
		}
	}
	if !strings.Contains(text, defaultBranchValue) {
		problems = append(problems, "does not fall back to the default_branch value "+defaultBranchValue)
	}
	if !strings.Contains(text, "exit 1") {
		problems = append(problems, "never exits non-zero, so an unresolvable base degrades silently into an unscoped run")
	}
	return problems
}

// ciAssignedBaseVariable finds the name of the shell variable the rendered file
// passes to `--base "$NAME"`, so the assignment/use correspondence is checked
// against DATA in the file rather than a name this test picked.
func ciAssignedBaseVariable(text string) string {
	const marker = `--base "$`
	index := strings.Index(text, marker)
	if index < 0 {
		return ""
	}
	rest := text[index+len(marker):]
	end := strings.IndexAny(rest, `"`)
	if end <= 0 {
		return ""
	}
	name := rest[:end]
	if !strings.Contains(text, name+"=") {
		return ""
	}
	return name
}

// ciDecodeYAML decodes rendered bytes into a generic map so a claim can walk the
// document's KEYS AS WRITTEN.
//
// The YAML 1.1 `on:` pitfall is why this returns map[string]any via yaml.Node
// rather than decoding into a struct: `on` unquoted decodes to the BOOLEAN true
// as a key under some decoders, and a struct field named On would silently paper
// over an absent key.
func ciDecodeYAML(t *testing.T, rendered []byte) map[string]any {
	t.Helper()

	var node yaml.Node
	if err := yaml.Unmarshal(rendered, &node); err != nil {
		t.Fatalf("the rendered file does not parse as YAML: %v\nrendered:\n%s", err, rendered)
	}
	if node.Kind != yaml.DocumentNode || len(node.Content) != 1 {
		t.Fatalf("the rendered file is not a single YAML document\nrendered:\n%s", rendered)
	}
	root := node.Content[0]
	if root.Kind != yaml.MappingNode {
		t.Fatalf("the rendered file's top level is not a mapping\nrendered:\n%s", rendered)
	}

	doc := map[string]any{}
	for i := 0; i+1 < len(root.Content); i += 2 {
		key := root.Content[i].Value
		var value any
		if err := root.Content[i+1].Decode(&value); err != nil {
			t.Fatalf("decode the value of top-level key %q: %v", key, err)
		}
		doc[key] = value
	}
	return doc
}

// ciWalkYAMLValue reports whether any node anywhere in the decoded document is a
// mapping carrying key with a value satisfying accept. It is how the full-history
// claims assert a DIRECTIVE rather than a substring, so an indented comment
// cannot satisfy them.
func ciWalkYAMLValue(node any, key string, accept func(any) bool) bool {
	switch typed := node.(type) {
	case map[string]any:
		for k, v := range typed {
			if k == key && accept(v) {
				return true
			}
			if ciWalkYAMLValue(v, key, accept) {
				return true
			}
		}
	case map[any]any:
		for k, v := range typed {
			if name, ok := k.(string); ok && name == key && accept(v) {
				return true
			}
			if ciWalkYAMLValue(v, key, accept) {
				return true
			}
		}
	case []any:
		for _, item := range typed {
			if ciWalkYAMLValue(item, key, accept) {
				return true
			}
		}
	}
	return false
}

// ciIsZero accepts both the YAML-typed integer 0 and the string "0", which are
// equally valid spellings of a full-history depth directive.
func ciIsZero(value any) bool {
	switch typed := value.(type) {
	case int:
		return typed == 0
	case string:
		return strings.TrimSpace(typed) == "0"
	default:
		return false
	}
}

// ciExitCodeOf classifies a CLI error the way main.go does, so a claim can assert
// EXIT 1 (an op failure through the normal apply path) as distinct from the
// exit-2 *check.ConfigError shape reserved for malformed --param input.
func ciExitCodeOf(t *testing.T, err error) (int, string) {
	t.Helper()

	var exitErr *ExitCodeError
	if errors.As(err, &exitErr) {
		return exitErr.Code, exitErr.Message
	}
	return 0, ""
}
