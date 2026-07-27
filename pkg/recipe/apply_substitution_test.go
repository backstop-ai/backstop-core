package recipe

import (
	"io/fs"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// SUBSTITUTION SCOPE (ISSUE-079). Every test in this file drives the REAL Apply.
// None of them calls Substitute directly, and that is the point: the three
// mandated Substitute-level tests in substitute_test.go all pass today while the
// applier writes at the literal `{{ … }}` text, so a Substitute-direct assertion
// cannot prove that a declared SITE is resolved. Only an apply can.
//
// Every expectation below is read back off the PARSED manifest and rendered
// through the applier's own param scope (effectiveParams), so no expected path,
// anchor, or snippet is a re-typed Go literal that a broken applier could agree
// with by coincidence.

// templatedSitesRecipeID is the committed sibling recipe whose create, merge,
// transform and insert ops ALL declare templated sites. It exists beside the
// byte-intact `starter` recipe because starter's merge op declares an INLINE
// fragment, which the shipped applier reads as a recipe-relative PATH.
const templatedSitesRecipeID = "templated-sites"

// templatedSitesPin is the committed recipe's declared version, supplied as the
// mandatory pin in the reference. It is a ref input, not an expectation.
const templatedSitesPin = "1.0.0"

// starterPin is the byte-intact committed starter recipe's declared version.
const starterPin = "1.2.0"

// suppliedAppName is the caller-supplied value for the fixture's REQUIRED
// app_name param, which declares no default. It is deliberately unlike any
// placeholder text, so a site that still carries `{{ app_name }}` is obvious.
const suppliedAppName = "supplied-app-name"

// seededSiblingValue is the name the seeded consumer document carries alongside
// the insert anchor, so the merge is provably additive rather than a rewrite.
const seededSiblingValue = "kept across the merge"

// templatedSitesSeededSettings is the consumer document the merge and insert ops
// both address.
const templatedSitesSeededSettings = `{
  "existing_sibling": "kept across the merge",
  "registrations": []
}
`

// templatedSitesRun is one end-to-end apply of the committed fixture recipe: the
// project root it wrote into, the resolved recipe, the effective param scope, the
// verdict, and the (rule, target) pairs the injected dispatch was handed.
type templatedSitesRun struct {
	root       string
	resolved   *ResolvedRecipe
	params     map[string]string
	result     ApplyResult
	dispatched [][2]string
	tree       map[string]string
}

// resolveCommittedRecipe resolves one recipe out of the COMMITTED fixture pack
// through the production resolution path, so the manifest under test is the
// tracked artifact rather than a Go literal.
func resolveCommittedRecipe(t *testing.T, recipeID string, pin string) *ResolvedRecipe {
	t.Helper()

	packs, packDir := loadFixtureCorpus(t)
	ref, err := ParseRecipeRef(fixturePackName + ":" + recipeID + "@" + pin)
	if err != nil {
		t.Fatalf("ParseRecipeRef for %q: unexpected error: %v", recipeID, err)
	}
	resolved, err := ResolveRecipe(ref, packs, packDir)
	if err != nil {
		t.Fatalf("ResolveRecipe %q: unexpected error: %v", ref, err)
	}

	return resolved
}

// renderDeclared renders one DECLARED field through the same substitution the
// applier owes it, so an expectation is derived from the artifact rather than
// re-typed. A field the test expects to resolve must resolve here too.
func renderDeclared(t *testing.T, declared string, params map[string]string) string {
	t.Helper()

	rendered, err := Substitute(declared, params)
	if err != nil {
		t.Fatalf("render declared field %q against the effective params: %v", declared, err)
	}

	return rendered
}

// applyTemplatedSitesFixture drives the committed templated-sites recipe end to
// end in direct mode with a recording dispatch, and returns everything the
// assertions need.
func applyTemplatedSitesFixture(t *testing.T) templatedSitesRun {
	t.Helper()

	resolved := resolveCommittedRecipe(t, templatedSitesRecipeID, templatedSitesPin)
	supplied := map[string]string{"app_name": suppliedAppName}
	params := effectiveParams(resolved.Manifest, supplied)

	byID := opsByID(resolved)
	mergeOp, declared := byID["merge-settings"]
	if !declared {
		t.Fatalf("fixture defect: the committed recipe declares no merge op; parsed ids = %v", opIDs(resolved.Manifest.Ops))
	}

	root := t.TempDir()
	writeUnder(t, root, renderDeclared(t, mergeOp.Target, params), templatedSitesSeededSettings)

	var dispatched [][2]string
	result, err := Apply(resolved, ApplyOptions{
		Mode:        ModeDirect,
		ProjectRoot: root,
		Params:      supplied,
		Dispatch: func(rule string, target string) error {
			dispatched = append(dispatched, [2]string{rule, target})
			return nil
		},
	})
	if err != nil {
		t.Fatalf("Apply of the committed %q recipe: unexpected error: %v", templatedSitesRecipeID, err)
	}

	return templatedSitesRun{
		root:       root,
		resolved:   resolved,
		params:     params,
		result:     result,
		dispatched: dispatched,
		tree:       snapshotTree(t, root),
	}
}

// TestApply_CommittedFixtureRecipe_SubstitutesEveryTemplatedSite is THE crux
// regression for ISSUE-079: a COMMITTED, resolution-driven recipe whose every
// declared site is templated applies end to end, with each site resolved BEFORE
// it is used to locate a file, match an anchor, or splice content — and with the
// transform's declared rule left alone.
func TestApply_CommittedFixtureRecipe_SubstitutesEveryTemplatedSite(t *testing.T) {
	run := applyTemplatedSitesFixture(t)
	byID := opsByID(run.resolved)

	createTarget := renderDeclared(t, byID["create-config"].Target, run.params)
	mergeTarget := renderDeclared(t, byID["merge-settings"].Target, run.params)
	insertTarget := renderDeclared(t, byID["register-app"].Target, run.params)
	transformTarget := renderDeclared(t, byID["rename-config-key"].Target, run.params)
	snippet := renderDeclared(t, byID["register-app"].Snippet, run.params)

	if createTarget == byID["create-config"].Target {
		t.Fatalf("fixture defect: the create op's declared target %q carries no placeholder, so this test could not falsify a raw-target applier", createTarget)
	}

	// The create op materialized the payload at the SUBSTITUTED target, with the
	// payload's own placeholders resolved.
	created, materialized := run.tree[createTarget]
	if !materialized {
		t.Fatalf("the create op did not write at the substituted target %q; project tree = %v", createTarget, pathSet(run.tree))
	}
	createdTree, decodeErr := decodeJSONTree([]byte(created))
	if decodeErr != nil {
		t.Fatalf("decode the created document %q: %v", created, decodeErr)
	}
	if createdTree["legacy_name"] != run.params["app_name"] {
		t.Errorf("created document carries legacy_name = %v, want the substituted param %q", createdTree["legacy_name"], run.params["app_name"])
	}
	if createdTree["config_dir"] != run.params["config_dir"] {
		t.Errorf("created document carries config_dir = %v, want the substituted param %q", createdTree["config_dir"], run.params["config_dir"])
	}

	// The merge op merged the FILE fragment into the substituted target, additively.
	if mergeTarget != insertTarget {
		t.Fatalf("fixture defect: the merge (%q) and insert (%q) ops must address ONE document for the insert to observe the merged output", mergeTarget, insertTarget)
	}
	settings, merged := run.tree[mergeTarget]
	if !merged {
		t.Fatalf("the merge op did not write at the substituted target %q; project tree = %v", mergeTarget, pathSet(run.tree))
	}
	settingsTree, decodeErr := decodeJSONTree([]byte(settings))
	if decodeErr != nil {
		t.Fatalf("decode the merged document %q: %v", settings, decodeErr)
	}
	if settingsTree["adopted_by"] != run.params["app_name"] {
		t.Errorf("merged document carries adopted_by = %v, want the substituted param %q", settingsTree["adopted_by"], run.params["app_name"])
	}
	if settingsTree["existing_sibling"] != seededSiblingValue {
		t.Errorf("merged document lost the seeded sibling: %v", settingsTree)
	}

	// The insert op spliced the SUBSTITUTED snippet immediately after the anchor
	// in that same document. The declared snippet is the JSON string form of the
	// param, so the spliced value decodes as the registrations list's sole entry.
	registrations, isList := settingsTree["registrations"].([]any)
	if !isList || len(registrations) != 1 {
		t.Fatalf("registrations = %v, want exactly one spliced entry", settingsTree["registrations"])
	}
	if registrations[0] != run.params["app_name"] {
		t.Errorf("spliced registration = %v, want the SUBSTITUTED snippet value %q (declared snippet %q rendered to %q)", registrations[0], run.params["app_name"], byID["register-app"].Snippet, snippet)
	}

	// The transform op was dispatched at the SUBSTITUTED target with the DECLARED
	// rule — resolved under the pack root and byte-identical to the declaration.
	// Op.Rule is deliberately NOT substituted (CLM-009).
	if len(run.dispatched) != 1 {
		t.Fatalf("the transform dispatch ran %d times, want exactly once: %v", len(run.dispatched), run.dispatched)
	}
	absPack, absErr := filepath.Abs(run.resolved.PackDir)
	if absErr != nil {
		t.Fatalf("resolve the fixture pack root %q: %v", run.resolved.PackDir, absErr)
	}
	wantRule := filepath.Join(absPack, filepath.FromSlash(byID["rename-config-key"].Rule))
	if run.dispatched[0][0] != wantRule {
		t.Errorf("dispatched rule = %q, want the DECLARED rule under the pack root %q", run.dispatched[0][0], wantRule)
	}
	wantTransformTarget := filepath.Join(run.root, filepath.FromSlash(transformTarget))
	if run.dispatched[0][1] != wantTransformTarget {
		t.Errorf("dispatched target = %q, want the SUBSTITUTED target %q", run.dispatched[0][1], wantTransformTarget)
	}

	// The verdict echoes the SUBSTITUTED project-relative paths — the recipe's own
	// path form, resolved, never a `{{`-bearing string and never an absolute path.
	wantWritten := []string{createTarget, mergeTarget}
	sort.Strings(wantWritten)
	gotWritten := append([]string(nil), run.result.Written...)
	sort.Strings(gotWritten)
	if !reflect.DeepEqual(gotWritten, wantWritten) {
		t.Errorf("result.Written = %v, want the substituted project-relative targets %v", run.result.Written, wantWritten)
	}
	for _, written := range run.result.Written {
		if strings.Contains(written, placeholderOpen) {
			t.Errorf("result.Written entry %q still carries a literal placeholder", written)
		}
	}

	// The reserved step op executed nothing and did not break the sequence: the
	// tree holds exactly the two substituted targets plus the adoption record.
	wantPaths := []string{createTarget, mergeTarget, AdoptionRecordName}
	sort.Strings(wantPaths)
	gotPaths := pathSet(run.tree)
	sort.Strings(gotPaths)
	if !reflect.DeepEqual(gotPaths, wantPaths) {
		t.Errorf("project tree = %v, want exactly %v", gotPaths, wantPaths)
	}
}

// TestApply_CommittedStarterRecipe_SubstitutesCreateTargetThenHaltsOnInlineFragment
// drives the BYTE-INTACT committed `starter` recipe — the one ISSUE-079 names as
// never having been driven through Apply — and pins both halves of what happens.
//
// Its create op runs FIRST and its templated target must resolve. Its merge op
// then declares `fragment:` as INLINE content while the shipped applier reads
// Fragment as a recipe-relative PATH, so the apply halts there.
//
// ISSUE-081 ("Recipe Authoring Surface Underspecified: Merge Fragment Form + No
// CLI Param Input") owns that second half. This assertion is a live, falsifiable
// marker for it rather than a comment: whoever pins the fragment form updates
// this test, and the committed recipe.yml itself is never touched.
func TestApply_CommittedStarterRecipe_SubstitutesCreateTargetThenHaltsOnInlineFragment(t *testing.T) {
	resolved := resolveCommittedRecipe(t, "starter", starterPin)
	supplied := map[string]string{"app_name": suppliedAppName}
	params := effectiveParams(resolved.Manifest, supplied)

	byID := opsByID(resolved)
	createOp, declared := byID["create-config"]
	if !declared {
		t.Fatalf("fixture defect: the committed starter recipe declares no create op; parsed ids = %v", opIDs(resolved.Manifest.Ops))
	}
	mergeOp := byID["merge-settings"]
	createTarget := renderDeclared(t, createOp.Target, params)
	if createTarget == createOp.Target {
		t.Fatalf("fixture defect: the starter create target %q carries no placeholder", createOp.Target)
	}

	root := t.TempDir()
	_, err := Apply(resolved, ApplyOptions{
		Mode:        ModeDirect,
		ProjectRoot: root,
		Params:      supplied,
		Dispatch:    func(string, string) error { return nil },
	})

	// The templated create target resolved before the halt.
	tree := snapshotTree(t, root)
	if _, materialized := tree[createTarget]; !materialized {
		t.Errorf("the starter create op did not write at the substituted target %q; project tree = %v", createTarget, pathSet(tree))
	}
	for _, path := range pathSet(tree) {
		if strings.Contains(path, placeholderOpen) || strings.Contains(path, placeholderClose) {
			t.Errorf("the apply created %q, a path carrying a literal placeholder", path)
		}
	}

	// ...and then the apply halted on the merge op's inline fragment.
	if err == nil {
		t.Fatalf("Apply of the committed starter recipe succeeded; its merge op declares an INLINE fragment the shipped applier reads as a path, so it must fail loud (ISSUE-081)")
	}
	for _, want := range []string{mergeOp.ID, "fragment"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not identify the halting merge op via %q", err, want)
		}
	}
}

// TestApply_NoLiteralPlaceholderReachesTheFilesystem is the INVARIANT, and the
// assertion that would have caught the live repro generically: after a successful
// apply, no entry anywhere beneath the project root may carry a placeholder
// delimiter in any path segment. It asserts on a WALK rather than on a known-good
// path list, so a future op family that forgets to substitute its site is caught
// without anyone remembering to extend a list.
func TestApply_NoLiteralPlaceholderReachesTheFilesystem(t *testing.T) {
	run := applyTemplatedSitesFixture(t)

	var walked int
	walkErr := filepath.WalkDir(run.root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(run.root, path)
		if relErr != nil {
			return relErr
		}
		if rel == "." {
			return nil
		}
		walked++
		for _, segment := range strings.Split(filepath.ToSlash(rel), "/") {
			if strings.Contains(segment, placeholderOpen) || strings.Contains(segment, placeholderClose) {
				t.Errorf("path segment %q of %q carries a literal placeholder; a declared site reached the filesystem unsubstituted", segment, rel)
			}
		}
		return nil
	})
	if walkErr != nil {
		t.Fatalf("walk the project root %q: %v", run.root, walkErr)
	}
	if walked == 0 {
		t.Fatal("the walk visited nothing; the invariant would be vacuous")
	}
}

// TestApply_UndeclaredPlaceholderInSiteFieldFailsLoud proves REQ-002 across every
// substituted SITE/CONTENT position: an undeclared placeholder fails the apply
// with Substitute's own unresolvable-placeholder diagnostic, wrapped by opFailure
// so it locates the recipe, the op index and the op id — and the project root is
// left exactly as it was. A partially-applied failure is as bad as a silent
// success, which is why the tree is compared before and after.
func TestApply_UndeclaredPlaceholderInSiteFieldFailsLoud(t *testing.T) {
	const undeclaredParam = "nope"

	cases := []struct {
		name       string
		recipeYAML string
		field      string
	}{
		{
			name:  "create_target",
			field: "target",
			recipeYAML: `
kind: scaffolding
version: 1.0.0
ops:
  - id: op-create-templated-target
    kind: create
    target: "generated/{{ nope }}/out.conf"
    payload: body.txt
`,
		},
		{
			name:  "merge_target",
			field: "target",
			recipeYAML: `
kind: scaffolding
version: 1.0.0
ops:
  - id: op-merge-templated-target
    kind: merge
    target: "generated/{{ nope }}.json"
    format: json
    fragment: fragment.json
`,
		},
		{
			name:  "insert_anchor",
			field: "anchor",
			recipeYAML: `
kind: scaffolding
version: 1.0.0
ops:
  - id: op-insert-templated-anchor
    kind: insert
    target: hosts/site.txt
    anchor: "# {{ nope }}"
    snippet: "\nspliced"
    manual: "Splice the snippet at the marker by hand."
`,
		},
		{
			name:  "insert_snippet",
			field: "snippet",
			recipeYAML: `
kind: scaffolding
version: 1.0.0
ops:
  - id: op-insert-templated-snippet
    kind: insert
    target: hosts/site.txt
    anchor: "# SLOTS"
    snippet: "\n{{ nope }}"
    manual: "Splice the snippet at the SLOTS marker by hand."
`,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			recipeDir := t.TempDir()
			projectRoot := t.TempDir()

			resolved := resolvedFromManifest(t, recipeDir, testCase.recipeYAML)
			op := resolved.Manifest.Ops[0]
			if !strings.Contains(op.Target+op.Anchor+op.Snippet, undeclaredParam) {
				t.Fatalf("fixture defect: op %q declares no %q placeholder", op.ID, undeclaredParam)
			}
			switch op.Kind {
			case OpCreate:
				writeUnder(t, recipeDir, op.Payload, "a payload with no placeholder of its own\n")
			case OpMerge:
				writeUnder(t, recipeDir, op.Fragment, `{"added": true}`)
			case OpInsert:
				writeUnder(t, projectRoot, op.Target, seedPayload)
			}

			before := snapshotTree(t, projectRoot)
			result, err := Apply(resolved, ApplyOptions{Mode: ModeDirect, ProjectRoot: projectRoot})
			if err == nil {
				t.Fatalf("Apply with an undeclared placeholder in the declared %s: expected a fail-loud error, got nil (result %+v)", testCase.field, result)
			}

			for _, want := range []string{"unresolvable placeholder", undeclaredParam, resolved.Ref.String(), "ops[0]", op.ID} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q does not carry %q", err, want)
				}
			}
			if len(result.Written) != 0 || len(result.Preserved) != 0 {
				t.Errorf("result = %+v, want the zero verdict alongside the error", result)
			}

			after := snapshotTree(t, projectRoot)
			gotPaths, wantPaths := pathSet(after), pathSet(before)
			sort.Strings(gotPaths)
			sort.Strings(wantPaths)
			if !reflect.DeepEqual(gotPaths, wantPaths) {
				t.Fatalf("project tree = %v, want the untouched %v — an op the applier cannot resolve writes nothing", gotPaths, wantPaths)
			}
			for path, content := range before {
				if after[path] != content {
					t.Errorf("file %q changed: before %q, after %q", path, content, after[path])
				}
			}
		})
	}
}

// suppliedSiteRecipe declares an insert whose own site is fine, so the ONLY thing
// under test is the sdlc-mediated supplied override. Both of the supplied site's
// halves are templated against params the recipe declares.
const suppliedSiteRecipe = `
kind: implementing
version: 1.0.0
params:
  - name: site_dir
    default: supplied
  - name: marker
    default: "# END"
ops:
  - id: op-insert
    kind: insert
    target: hosts/declared.txt
    anchor: "# SLOTS"
    snippet: "\nspliced-by-the-recipe"
    manual: "Splice the declared snippet at the marker by hand."
`

// TestApply_SDLCMediatedMode_SuppliedInjectionSiteIsSubstituted proves the gap at
// siteFor closes on BOTH halves of a supplied site (CLM-005). The override is the
// only way to reach that code path, so it needs its own test.
//
// The supplied anchor resolves to the target's SECOND marker, so honoring it is
// distinguishable from falling back to the recipe-declared anchor; and the
// declared target must stay untouched, so an applier that ignored the override
// cannot pass. The "#" separator is split off the DECLARED text BEFORE
// substitution, which is why a substituted anchor may itself contain "#".
func TestApply_SDLCMediatedMode_SuppliedInjectionSiteIsSubstituted(t *testing.T) {
	resolved := resolvedFromManifest(t, t.TempDir(), suppliedSiteRecipe)
	insertOp := resolved.Manifest.Ops[0]
	params := effectiveParams(resolved.Manifest, nil)

	const suppliedTargetHalf = "{{ site_dir }}/target.txt"
	const suppliedAnchorHalf = "{{ marker }}"
	suppliedSite := suppliedTargetHalf + injectionSiteSeparator + suppliedAnchorHalf
	suppliedTarget := renderDeclared(t, suppliedTargetHalf, params)
	suppliedAnchor := renderDeclared(t, suppliedAnchorHalf, params)
	if suppliedAnchor == insertOp.Anchor {
		t.Fatalf("fixture defect: the supplied anchor resolves to the declared anchor %q, so honoring the override is not falsifiable", insertOp.Anchor)
	}
	if !strings.Contains(suppliedAnchor, injectionSiteSeparator) {
		t.Fatalf("fixture defect: the substituted anchor %q carries no %q, so the split-before-substitute ordering is not under test", suppliedAnchor, injectionSiteSeparator)
	}

	projectRoot := t.TempDir()
	writeUnder(t, projectRoot, insertOp.Target, seedPayload)
	writeUnder(t, projectRoot, suppliedTarget, seedPayload)

	if _, err := Apply(resolved, ApplyOptions{
		Mode:           ModeSDLCMediated,
		ProjectRoot:    projectRoot,
		InjectionSites: map[string]string{insertOp.ID: suppliedSite},
	}); err != nil {
		t.Fatalf("Apply in sdlc-mediated mode with a templated supplied site: unexpected error: %v", err)
	}

	tree := snapshotTree(t, projectRoot)
	spliceAt := strings.Index(seedPayload, suppliedAnchor) + len(suppliedAnchor)
	want := seedPayload[:spliceAt] + insertOp.Snippet + seedPayload[spliceAt:]
	if got := tree[suppliedTarget]; got != want {
		t.Errorf("supplied site %q =\n%q\nwant the splice at the SUBSTITUTED anchor %q\n%q", suppliedTarget, got, suppliedAnchor, want)
	}
	// The falsifier: an applier that ignored the override would have written here.
	if got := tree[insertOp.Target]; got != seedPayload {
		t.Errorf("the recipe-declared target %q = %q, want the untouched seed: the supplied site owns the WHERE", insertOp.Target, got)
	}
}

// templatedTransformRecipe templates its transform TARGET and declares a LITERAL
// rule that is also listed in transform_rules. The two fields are deliberately
// asymmetric.
const templatedTransformRecipe = `
kind: implementing
version: 1.0.0
params:
  - name: variant
    default: "resolved-variant"
transform_rules:
  - rules/rename-key.yml
ops:
  - id: op-transform
    kind: transform
    target: "{{ variant }}/site.conf"
    rule: rules/rename-key.yml
    manual: "Re-point the superseded call sites in the declared target by hand."
`

// TestApply_TransformOp_DeclaredRuleIsNotSubstituted guards the DELIBERATE
// asymmetry (CLM-009) against a future "fix" that routes every field through
// Substitute uniformly: the dispatch receives the templated target RESOLVED, and
// the declared rule path byte-identical, resolved under the PACK root.
//
// Op.Rule is validated at parse by exact string equality against the recipe's
// declared transform_rules, and it selects which pack asset an allowlisted engine
// executes in place over the consumer's tree. Substituting it would break the
// declared-vs-executed correspondence the allowlist exists to guarantee and would
// hand a consumer-supplied param authority over the recipe's own code.
func TestApply_TransformOp_DeclaredRuleIsNotSubstituted(t *testing.T) {
	resolved := resolvedFromManifest(t, t.TempDir(), templatedTransformRecipe)
	resolved.PackDir = t.TempDir()
	transformOp := resolved.Manifest.Ops[0]
	params := effectiveParams(resolved.Manifest, nil)

	target := renderDeclared(t, transformOp.Target, params)
	if target == transformOp.Target {
		t.Fatalf("fixture defect: the transform target %q carries no placeholder", transformOp.Target)
	}
	if strings.Contains(transformOp.Rule, placeholderOpen) {
		t.Fatalf("fixture defect: the declared rule %q carries a placeholder; a templated rule is refused at parse", transformOp.Rule)
	}

	projectRoot := t.TempDir()
	writeUnder(t, projectRoot, target, unreachableSiteContent)

	var dispatched [][2]string
	if _, err := Apply(resolved, ApplyOptions{
		Mode:        ModeDirect,
		ProjectRoot: projectRoot,
		Dispatch: func(rule string, dispatchedTarget string) error {
			dispatched = append(dispatched, [2]string{rule, dispatchedTarget})
			return nil
		},
	}); err != nil {
		t.Fatalf("Apply of a templated-target transform: unexpected error: %v", err)
	}

	if len(dispatched) != 1 {
		t.Fatalf("the transform dispatch ran %d times, want exactly once: %v", len(dispatched), dispatched)
	}
	wantRule := filepath.Join(resolved.PackDir, filepath.FromSlash(transformOp.Rule))
	if dispatched[0][0] != wantRule {
		t.Errorf("dispatched rule = %q, want the DECLARED rule under the pack root %q — Op.Rule is never substituted", dispatched[0][0], wantRule)
	}
	wantTarget := filepath.Join(projectRoot, filepath.FromSlash(target))
	if dispatched[0][1] != wantTarget {
		t.Errorf("dispatched target = %q, want the SUBSTITUTED target %q", dispatched[0][1], wantTarget)
	}
}

// The two injection-limit manual cases: an insert whose anchor is absent from the
// target, whose declared `manual` templates a param the recipe DOES declare, and
// the same op templating one it does NOT. The committed fixtures template their
// manual text, which is why the resolved case matters.
const (
	declaredManualRecipe = `
kind: scaffolding
version: 1.0.0
params:
  - name: app_name
    default: "resolved-app"
ops:
  - id: op-missing-anchor
    kind: insert
    target: hosts/site.txt
    anchor: "# THE ANCHOR THE TARGET DOES NOT CARRY"
    snippet: "\nspliced"
    manual: 'Add "{{ app_name }}" to the registrations list in the settings file by hand.'
`

	undeclaredManualRecipe = `
kind: scaffolding
version: 1.0.0
ops:
  - id: op-missing-anchor
    kind: insert
    target: hosts/site.txt
    anchor: "# THE ANCHOR THE TARGET DOES NOT CARRY"
    snippet: "\nspliced"
    manual: 'Add "{{ never_declared }}" to the registrations list in the settings file by hand.'
`
)

// TestApply_InjectionLimit_RelaysSubstitutedManualFailingSoftToRaw pins the one
// field where fail-loud is the WRONG policy (CLM-006).
//
// Op.Manual is operator-facing text telling a human what to type into their own
// file, and the committed fixtures template it — relaying literal braces to the
// operator is the same defect class in the diagnostic channel. So it IS
// substituted. But it is emitted ONLY on an error path, so a substitution failure
// there must not replace the operator's instruction with a second error: the RAW
// declared text is relayed instead. Case (b) is the falsifier for a naive
// "substitute everything, fail loud" implementation.
func TestApply_InjectionLimit_RelaysSubstitutedManualFailingSoftToRaw(t *testing.T) {
	cases := []struct {
		name       string
		recipeYAML string
		resolves   bool
	}{
		{name: "declared_param_resolves_in_the_relayed_manual", recipeYAML: declaredManualRecipe, resolves: true},
		{name: "undeclared_param_falls_soft_back_to_the_raw_manual", recipeYAML: undeclaredManualRecipe, resolves: false},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			resolved := resolvedFromManifest(t, t.TempDir(), testCase.recipeYAML)
			insertOp := resolved.Manifest.Ops[0]
			if !strings.Contains(insertOp.Manual, placeholderOpen) {
				t.Fatalf("fixture defect: the declared manual %q carries no placeholder", insertOp.Manual)
			}

			projectRoot := t.TempDir()
			writeUnder(t, projectRoot, insertOp.Target, unreachableSiteContent)

			_, err := Apply(resolved, ApplyOptions{Mode: ModeDirect, ProjectRoot: projectRoot})
			if err == nil {
				t.Fatalf("Apply of an insert whose anchor is absent: expected a fail-loud error, got nil")
			}
			// Either way the operator is told WHICH site could not be reached.
			for _, want := range []string{insertOp.ID, insertOp.Target} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q does not locate the unreachable site via %q", err, want)
				}
			}

			if !testCase.resolves {
				if !strings.Contains(err.Error(), insertOp.Manual) {
					t.Errorf("error\n%q\ndoes not relay the RAW declared manual verbatim\n%q — a manual whose own substitution fails must not be replaced by a substitution error", err, insertOp.Manual)
				}
				return
			}

			params := effectiveParams(resolved.Manifest, nil)
			wantManual := renderDeclared(t, insertOp.Manual, params)
			if wantManual == insertOp.Manual {
				t.Fatal("fixture defect: the declared manual renders to itself, so the resolved case is not falsifiable")
			}
			if !strings.Contains(err.Error(), wantManual) {
				t.Errorf("error\n%q\ndoes not relay the SUBSTITUTED manual\n%q", err, wantManual)
			}
			if strings.Contains(err.Error(), placeholderOpen) {
				t.Errorf("error %q still carries a literal placeholder in the relayed instruction", err)
			}
		})
	}
}
