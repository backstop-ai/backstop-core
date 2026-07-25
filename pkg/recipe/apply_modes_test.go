package recipe

import (
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// The two application MODES (REQ-003). Both drive the SAME op executors over the
// SAME recipe artifact — the mode selects ONLY where each op's WHERE comes from:
// direct reads the recipe's own declarations, sdlc-mediated additionally consults
// the sites the plan supplies for the injection-accepting families.
//
// Every declared value below is read back off the PARSED manifest rather than
// re-typed, so no expectation can agree with the applier by coincidence.

// directModeRecipe declares defaults for both of its params and supplies its own
// insertion site. Applied in direct mode with NO caller params and NO caller
// sites, the declared defaults and the declared anchor are the only inputs.
const directModeRecipe = `
kind: scaffolding
version: 1.0.0
params:
  - name: greeting
    default: "declared-default-greeting"
  - name: audience
    default: "declared-default-audience"
ops:
  - id: op-create
    kind: create
    target: generated/rendered.txt
    payload: rendered.txt.tmpl
  - id: op-insert
    kind: insert
    target: generated/rendered.txt
    anchor: "# SLOTS"
    snippet: "\ndeclared-snippet"
    manual: "Add the declared snippet beneath the SLOTS marker by hand."
`

// directModePayload is the create op's template: an anchor line, a line built
// from BOTH declared params, and a trailing marker.
const directModePayload = "# SLOTS\n{{ greeting }} {{ audience }}\n# END\n"

// TestApply_DirectMode_SelfAppliesFromDefaults proves direct mode self-applies from
// the recipe's DECLARED defaults and declared sites (CLM-015): with no caller params
// the defaults render the output, and an InjectionSites map supplied ANYWAY is
// ignored outright — direct mode never consults it. The decoy site is the falsifier:
// an applier that honored sites regardless of mode would move the insert to
// decoy/op-insert.txt, which the exact path-set assertion catches.
func TestApply_DirectMode_SelfAppliesFromDefaults(t *testing.T) {
	recipeDir := t.TempDir()
	resolved := resolvedFromManifest(t, recipeDir, directModeRecipe)
	createOp := resolved.Manifest.Ops[0]
	insertOp := resolved.Manifest.Ops[1]
	writeUnder(t, recipeDir, createOp.Payload, directModePayload)

	// A decoy site for EVERY declared op, none of which direct mode may honor.
	sites := make(map[string]string, len(resolved.Manifest.Ops))
	for _, op := range resolved.Manifest.Ops {
		sites[op.ID] = decoySite(op.ID)
	}

	projectRoot := t.TempDir()
	result, err := Apply(resolved, ApplyOptions{
		Mode:           ModeDirect,
		ProjectRoot:    projectRoot,
		InjectionSites: sites,
	})
	if err != nil {
		t.Fatalf("Apply in direct mode: unexpected error: %v", err)
	}

	defaults := declaredDefaults(resolved)
	want := "# SLOTS" + insertOp.Snippet + "\n" + defaults["greeting"] + " " + defaults["audience"] + "\n# END\n"

	tree := snapshotTree(t, projectRoot)
	if got := tree[createOp.Target]; got != want {
		t.Errorf("direct-mode output =\n%q\nwant the declared defaults rendered at the declared site\n%q", got, want)
	}
	if strings.Contains(tree[createOp.Target], "{{") {
		t.Errorf("direct-mode output still carries an unresolved placeholder: %q", tree[createOp.Target])
	}

	wantPaths := []string{AdoptionRecordName, createOp.Target}
	gotPaths := pathSet(tree)
	sort.Strings(gotPaths)
	sort.Strings(wantPaths)
	if !reflect.DeepEqual(gotPaths, wantPaths) {
		t.Errorf("project tree = %v, want exactly %v — direct mode must ignore the supplied sites", gotPaths, wantPaths)
	}
	if len(result.Written) != 1 || result.Written[0] != createOp.Target {
		t.Errorf("result.Written = %v, want exactly [%q]", result.Written, createOp.Target)
	}
}

// declaredDefaults reads the recipe's DECLARED param defaults back off the parsed
// manifest, so the expectation is built from the artifact rather than re-typed.
func declaredDefaults(resolved *ResolvedRecipe) map[string]string {
	defaults := make(map[string]string, len(resolved.Manifest.Params))
	for _, spec := range resolved.Manifest.Params {
		defaults[spec.Name] = spec.Default
	}

	return defaults
}

// deterministicRecipe exercises every write path that could leak nondeterminism:
// two templated creates, a structured merge in each of two codecs (whose merged
// trees are re-encoded from Go maps — the classic map-iteration-order hazard), and
// an insert. If any of them ranged a map into the output, two runs diverge.
const deterministicRecipe = `
kind: scaffolding
version: 1.0.0
params:
  - name: project_name
    default: "declared-project"
ops:
  - id: op-create-doc
    kind: create
    target: generated/doc.json
    payload: doc.json.tmpl
  - id: op-merge-doc
    kind: merge
    target: generated/doc.json
    fragment: fragment.json
  - id: op-create-conf
    kind: create
    target: generated/conf.yml
    payload: conf.yml.tmpl
  - id: op-merge-conf
    kind: merge
    target: generated/conf.yml
    fragment: fragment.yml
  - id: op-create-notes
    kind: create
    target: generated/notes.txt
    payload: notes.txt.tmpl
  - id: op-insert-notes
    kind: insert
    target: generated/notes.txt
    anchor: "# SLOTS"
    snippet: "\nspliced-note"
    manual: "Add the note beneath the SLOTS marker by hand."
`

// The templated targets and the fragments merged into them. Both fragments carry
// several names, at two depths, in a NON-sorted declaration order, so a merger that
// emitted its tree in map-iteration order would produce different bytes per run.
const (
	deterministicDocPayload = `{
  "name": "{{ project_name }}",
  "zulu": 1,
  "nested": { "inner_zulu": true, "inner_alpha": false }
}
`
	deterministicDocFragment = `{
  "yankee": "y",
  "bravo": "b",
  "mike": "m",
  "delta": "d",
  "nested": { "inner_mike": 1, "inner_bravo": 2 }
}
`
	deterministicConfPayload = `name: "{{ project_name }}"
zulu: 1
nested:
  inner_zulu: true
  inner_alpha: false
`
	deterministicConfFragment = `yankee: y
bravo: b
mike: m
delta: d
nested:
  inner_mike: 1
  inner_bravo: 2
`
	deterministicNotesPayload = "# SLOTS\n{{ project_name }}\n# END\n"
)

// TestApply_DirectMode_Deterministic proves the same recipe + params applied into
// two FRESH roots yields byte-identical output (CLM-016). The comparison is over the
// WHOLE tree of both roots, not a probed path, and every file is compared as raw
// BYTES — with exactly one documented exception: the adoption record's adopted
// timestamp, the single time-bearing value in the model, which is asserted PRESENT
// in both and then excluded by name.
func TestApply_DirectMode_Deterministic(t *testing.T) {
	params := map[string]string{"project_name": "supplied-project"}

	first := deterministicRun(t, params)
	second := deterministicRun(t, params)

	if len(first) < 4 {
		t.Fatalf("the deterministic run produced only %v; the comparison would be near-vacuous", pathSet(first))
	}

	firstPaths, secondPaths := pathSet(first), pathSet(second)
	sort.Strings(firstPaths)
	sort.Strings(secondPaths)
	if !reflect.DeepEqual(firstPaths, secondPaths) {
		t.Fatalf("the two runs wrote different path sets: %v vs %v", firstPaths, secondPaths)
	}

	for _, path := range firstPaths {
		if path == AdoptionRecordName {
			assertAdoptionsMatchModuloTimestamp(t, first[path], second[path])
			continue
		}
		if first[path] != second[path] {
			t.Errorf("%s diverged across two runs of the same recipe+params:\nfirst:\n%q\nsecond:\n%q", path, first[path], second[path])
		}
	}
}

// deterministicRun applies deterministicRecipe into a FRESH recipe directory and a
// FRESH project root, returning the whole resulting tree. A fresh recipe directory
// per run keeps the payload/fragment bytes from being shared state that could mask a
// nondeterministic render.
func deterministicRun(t *testing.T, params map[string]string) map[string]string {
	t.Helper()

	recipeDir := t.TempDir()
	resolved := resolvedFromManifest(t, recipeDir, deterministicRecipe)
	contents := map[string]string{
		"op-create-doc":   deterministicDocPayload,
		"op-merge-doc":    deterministicDocFragment,
		"op-create-conf":  deterministicConfPayload,
		"op-merge-conf":   deterministicConfFragment,
		"op-create-notes": deterministicNotesPayload,
	}
	for _, op := range resolved.Manifest.Ops {
		content, declared := contents[op.ID]
		if !declared {
			continue
		}
		declaredPath := op.Payload
		if op.Kind == OpMerge {
			declaredPath = op.Fragment
		}
		writeUnder(t, recipeDir, declaredPath, content)
	}

	projectRoot := t.TempDir()
	if _, err := Apply(resolved, ApplyOptions{Mode: ModeDirect, ProjectRoot: projectRoot, Params: params}); err != nil {
		t.Fatalf("Apply in direct mode: unexpected error: %v", err)
	}

	return snapshotTree(t, projectRoot)
}

// assertAdoptionsMatchModuloTimestamp compares two adoption records for equality
// EXCEPT the adopted timestamp — and first asserts that field is populated in
// both, so excluding it cannot hide a record that silently stopped recording it.
func assertAdoptionsMatchModuloTimestamp(t *testing.T, first string, second string) {
	t.Helper()

	decode := func(raw string) AdoptionRecord {
		var record AdoptionRecord
		if err := yaml.Unmarshal([]byte(raw), &record); err != nil {
			t.Fatalf("parse adoption record %q: %v", raw, err)
		}
		if len(record.Recipes) == 0 {
			t.Fatalf("adoption record %q carries no entries; the exclusion below would be vacuous", raw)
		}
		for key, entry := range record.Recipes {
			if strings.TrimSpace(entry.Adopted) == "" {
				t.Fatalf("adoption entry %q carries no adopted timestamp; excluding it would hide a regression", key)
			}
			entry.Adopted = ""
			record.Recipes[key] = entry
		}
		return record
	}

	if !reflect.DeepEqual(decode(first), decode(second)) {
		t.Errorf("the adoption records diverged beyond their timestamps:\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

// mediatedRecipe declares one op of every family. The injection-accepting two
// (insert, transform) declare their own sites, which a supplied site overrides; the
// other three (create, merge, step) must ignore any site supplied for them.
const mediatedRecipe = `
kind: implementing
version: 1.0.0
transform_rules:
  - rules/rewrite.yml
ops:
  - id: op-create
    kind: create
    target: generated/created.txt
    payload: created.txt
  - id: op-merge
    kind: merge
    target: hosts/declared-merge.json
    fragment: fragment.json
  - id: op-step
    kind: step
  - id: op-insert
    kind: insert
    target: hosts/declared-insert.txt
    anchor: "# SLOTS"
    snippet: "\nthe-recipe-supplies-the-what"
    manual: "Splice the declared snippet beneath the SLOTS marker by hand."
  - id: op-transform
    kind: transform
    target: hosts/declared-transform.txt
    rule: rules/rewrite.yml
    manual: "Re-point the superseded call sites in the declared target by hand."
`

// The recipe-side files the mediated recipe declares. The rule body is opaque to
// the applier — the dispatch seam is spied, so only the PATH handed to it matters.
const (
	mediatedCreatePayload = "created-by-the-recipe\n"
	mediatedFragment      = `{ "added_by_the_recipe": true }`
	mediatedRuleBody      = "# an opaque rule body; the applier only routes its path\n"
)

// mediatedOutcome is one mediated run: the resulting tree and the (rule, target)
// pairs the injected transform dispatch was handed.
type mediatedOutcome struct {
	tree       map[string]string
	dispatched [][2]string
}

// The site paths a mediated run supplies. The two host families are per-run so
// the write location is observable; the decoy paths exist only to be ignored.
func suppliedInsertSite(run string) string {
	return "hosts/" + run + "-insert.txt"
}

func suppliedTransformSite(run string) string {
	return "hosts/" + run + "-transform.txt"
}

func decoySite(opID string) string {
	return "decoy/" + opID + ".txt"
}

// mediatedRuns names every run the mediated suite seeds host files for.
var mediatedRuns = []string{"alpha", "beta", "gamma"} // nosemgrep: go.core.no-global-mutable-state — immutable fixture roster expressed as a var only because a slice literal is not constant-foldable

// opsByID indexes the parsed manifest ops by their declared id, the same key
// InjectionSites routes by.
func opsByID(resolved *ResolvedRecipe) map[string]Op {
	byID := make(map[string]Op, len(resolved.Manifest.Ops))
	for _, op := range resolved.Manifest.Ops {
		byID[op.ID] = op
	}

	return byID
}

// TestApply_SDLCMediatedMode_AppliesAtSuppliedInjectionSite proves SDLC-mediated
// mode takes the WHERE from InjectionSites[op-id] while the recipe still supplies
// the WHAT (CLM-017).
//
// The falsifier is the PAIR of runs: the SAME recipe applied with DIFFERENT
// supplied sites must write to DIFFERENT places, with byte-identical
// contributions. An applier that ignored the supplied site would write to the same
// declared target in both runs and fail the differs-assertion; one that honored
// sites for every family would move the create/merge/step decoys too.
func TestApply_SDLCMediatedMode_AppliesAtSuppliedInjectionSite(t *testing.T) {
	recipeDir := t.TempDir()
	resolved := resolvedFromManifest(t, recipeDir, mediatedRecipe)
	byID := opsByID(resolved)
	writeUnder(t, recipeDir, byID["op-create"].Payload, mediatedCreatePayload)
	writeUnder(t, recipeDir, byID["op-merge"].Fragment, mediatedFragment)
	rulePath := writeUnder(t, recipeDir, byID["op-transform"].Rule, mediatedRuleBody)

	alpha := runMediated(t, resolved, "alpha", suppliedInsertSite("alpha"))
	beta := runMediated(t, resolved, "beta", suppliedInsertSite("beta"))

	wantSpliced := "# SLOTS" + byID["op-insert"].Snippet + "\n# END\n"
	capturedTarget := string(readMergeFixture(t, "target.json"))

	for _, run := range []struct {
		name  string
		this  string
		other string
		out   mediatedOutcome
	}{
		{name: "alpha", this: "alpha", other: "beta", out: alpha},
		{name: "beta", this: "beta", other: "alpha", out: beta},
	} {
		t.Run(run.name, func(t *testing.T) {
			// The insert landed at the SUPPLIED site...
			if got := run.out.tree[suppliedInsertSite(run.this)]; got != wantSpliced {
				t.Errorf("supplied insert site %q =\n%q\nwant\n%q", suppliedInsertSite(run.this), got, wantSpliced)
			}
			// ...and NOT at the recipe-declared one, nor at the site of the other run.
			if got := run.out.tree[byID["op-insert"].Target]; got != seedPayload {
				t.Errorf("declared insert target = %q, want the untouched seed %q: the supplied site owns the WHERE", got, seedPayload)
			}
			if got := run.out.tree[suppliedInsertSite(run.other)]; got != seedPayload {
				t.Errorf("the site of the other run %q = %q, want the untouched seed", suppliedInsertSite(run.other), got)
			}

			// The transform was dispatched at the SUPPLIED target with the
			// rule the RECIPE declares.
			if len(run.out.dispatched) != 1 {
				t.Fatalf("transform dispatched %d times, want exactly once: %v", len(run.out.dispatched), run.out.dispatched)
			}
			gotRule, gotTarget := run.out.dispatched[0][0], run.out.dispatched[0][1]
			if gotRule != rulePath {
				t.Errorf("dispatched rule = %q, want the recipe-declared %q: the recipe supplies the WHAT", gotRule, rulePath)
			}
			if !strings.HasSuffix(filepath.ToSlash(gotTarget), suppliedTransformSite(run.this)) {
				t.Errorf("dispatched target = %q, want the supplied site %q", gotTarget, suppliedTransformSite(run.this))
			}

			// create / merge / step are NOT injection-accepting: their decoys are
			// untouched and their declared targets carry the output.
			if got := run.out.tree[byID["op-create"].Target]; got != mediatedCreatePayload {
				t.Errorf("create wrote %q at its declared target, want %q: a create ignores InjectionSites", got, mediatedCreatePayload)
			}
			for _, decoy := range []string{decoySite("op-create"), decoySite("op-step")} {
				if _, moved := run.out.tree[decoy]; moved {
					t.Errorf("a non-injection-accepting op wrote at its supplied decoy %q; only transform/insert accept a site", decoy)
				}
			}
			if got := run.out.tree[decoySite("op-merge")]; got != capturedTarget {
				t.Errorf("the merge decoy %q was modified; a merge ignores InjectionSites", decoySite("op-merge"))
			}
			merged, decodeErr := decodeJSONTree([]byte(run.out.tree[byID["op-merge"].Target]))
			if decodeErr != nil {
				t.Fatalf("decode the merged declared target: %v", decodeErr)
			}
			if merged["added_by_the_recipe"] != true {
				t.Errorf("the declared merge target did not receive the fragment: %v", merged)
			}
		})
	}

	// The cross-run falsifier: DIFFERENT supplied sites, DIFFERENT write locations,
	// IDENTICAL contribution.
	if suppliedInsertSite("alpha") == suppliedInsertSite("beta") {
		t.Fatal("the two runs supplied the same site; the comparison would be a tautology")
	}
	if alpha.dispatched[0][1] == beta.dispatched[0][1] {
		t.Errorf("both runs dispatched the transform at %q; a different supplied site must move the target", alpha.dispatched[0][1])
	}
	if alpha.tree[suppliedInsertSite("alpha")] != beta.tree[suppliedInsertSite("beta")] {
		t.Errorf("the WHAT differed across runs:\n%q\nvs\n%q\nonly the WHERE may change", alpha.tree[suppliedInsertSite("alpha")], beta.tree[suppliedInsertSite("beta")])
	}
	if alpha.tree[byID["op-merge"].Target] != beta.tree[byID["op-merge"].Target] {
		t.Error("the merge output differed across runs; a merge ignores InjectionSites entirely")
	}

	// A supplied site may carry the ANCHOR half as well: keeping a target and
	// naming a different anchor moves the splice WITHIN one file. The seed carries
	// two markers, so splicing at the second one is observable against the first.
	const secondMarker = "# END"
	gammaSite := suppliedInsertSite("gamma") + "#" + secondMarker
	gamma := runMediated(t, resolved, "gamma", gammaSite)
	wantAtSecondMarker := seedPayload[:strings.Index(seedPayload, secondMarker)+len(secondMarker)] +
		byID["op-insert"].Snippet + "\n"
	if got := gamma.tree[suppliedInsertSite("gamma")]; got != wantAtSecondMarker {
		t.Errorf("supplied anchor site =\n%q\nwant the splice at the supplied anchor\n%q", got, wantAtSecondMarker)
	}
	if wantAtSecondMarker == wantSpliced {
		t.Fatal("the two anchors expect the same bytes; the anchor half of the site is not falsifiable here")
	}
}

// runMediated seeds a fresh project root with every declared and supplied host
// file and applies the recipe in SDLC-MEDIATED mode with a site for EVERY op.
func runMediated(t *testing.T, resolved *ResolvedRecipe, run string, insertSite string) mediatedOutcome {
	t.Helper()

	byID := opsByID(resolved)
	projectRoot := t.TempDir()
	seeds := []string{byID["op-insert"].Target, byID["op-transform"].Target}
	for _, named := range mediatedRuns {
		seeds = append(seeds, suppliedInsertSite(named), suppliedTransformSite(named))
	}
	for _, seeded := range seeds {
		writeUnder(t, projectRoot, seeded, seedPayload)
	}
	capturedTarget := string(readMergeFixture(t, "target.json"))
	writeUnder(t, projectRoot, byID["op-merge"].Target, capturedTarget)
	writeUnder(t, projectRoot, decoySite("op-merge"), capturedTarget)

	sites := map[string]string{
		"op-create":    decoySite("op-create"),
		"op-merge":     decoySite("op-merge"),
		"op-step":      decoySite("op-step"),
		"op-insert":    insertSite,
		"op-transform": suppliedTransformSite(run),
	}

	var dispatched [][2]string
	if _, err := Apply(resolved, ApplyOptions{
		Mode:           ModeSDLCMediated,
		ProjectRoot:    projectRoot,
		InjectionSites: sites,
		Dispatch: func(rule string, target string) error {
			dispatched = append(dispatched, [2]string{rule, target})
			return nil
		},
	}); err != nil {
		t.Fatalf("Apply in sdlc-mediated mode (%s): unexpected error: %v", run, err)
	}

	return mediatedOutcome{tree: snapshotTree(t, projectRoot), dispatched: dispatched}
}

// TestApply_SDLCMediatedMode_MissingInjectionSiteFailsLoud proves an
// injection-accepting op with NEITHER a recipe-declared site NOR a supplied one
// fails loud and writes NOTHING (CLM-060, REQ-011). There is no fallback: no
// appending at EOF, no defaulted target, no guess. The whole root is walked, so a
// write anywhere is caught, and a REAL dispatch is supplied that must never run.
func TestApply_SDLCMediatedMode_MissingInjectionSiteFailsLoud(t *testing.T) {
	const insertNoAnchor = `
kind: implementing
version: 1.0.0
ops:
  - id: op-insert-no-site
    kind: insert
    target: hosts/present.txt
    snippet: "\nunreachable"
    manual: "Splice the snippet at the site you choose, by hand."
`
	const insertNoTarget = `
kind: implementing
version: 1.0.0
ops:
  - id: op-insert-no-target
    kind: insert
    anchor: "# SLOTS"
    snippet: "\nunreachable"
    manual: "Splice the snippet into the file you choose, by hand."
`
	const transformNoTarget = `
kind: implementing
version: 1.0.0
transform_rules:
  - rules/rewrite.yml
ops:
  - id: op-transform-no-target
    kind: transform
    rule: rules/rewrite.yml
    manual: "Re-point the superseded call sites by hand."
`

	cases := []struct {
		name       string
		recipeYAML string
		seedTarget bool
		wantPart   string
	}{
		{name: "insert_without_anchor_or_site", recipeYAML: insertNoAnchor, seedTarget: true, wantPart: "no anchor"},
		{name: "insert_without_target_or_site", recipeYAML: insertNoTarget, wantPart: "no target"},
		{name: "transform_without_target_or_site", recipeYAML: transformNoTarget, wantPart: "no target"},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			recipeDir := t.TempDir()
			resolved := resolvedFromManifest(t, recipeDir, testCase.recipeYAML)
			op := resolved.Manifest.Ops[0]
			if op.Kind == OpTransform {
				writeUnder(t, recipeDir, op.Rule, mediatedRuleBody)
			}

			projectRoot := t.TempDir()
			var seeded string
			if testCase.seedTarget {
				seeded = op.Target
				writeUnder(t, projectRoot, seeded, seedPayload)
			}

			dispatchedAnyway := false
			result, err := Apply(resolved, ApplyOptions{
				Mode:           ModeSDLCMediated,
				ProjectRoot:    projectRoot,
				InjectionSites: map[string]string{"some-other-op": "hosts/not-this-op.txt"},
				Dispatch: func(string, string) error {
					dispatchedAnyway = true
					return nil
				},
			})
			if err == nil {
				t.Fatalf("Apply: expected a fail-loud error for a site-less %s op, got nil (result %+v)", op.Kind, result)
			}
			for _, want := range []string{testCase.wantPart, op.ID, op.Manual} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q does not carry %q", err, want)
				}
			}
			if dispatchedAnyway {
				t.Error("the transform dispatch ran despite there being no site; a site-less op must fail before any engine runs")
			}
			if len(result.Written) != 0 || len(result.Preserved) != 0 {
				t.Errorf("result = %+v, want the zero verdict alongside the error", result)
			}

			after := snapshotTree(t, projectRoot)
			if testCase.seedTarget && after[seeded] != seedPayload {
				t.Errorf("the declared target = %q, want the untouched seed: a site-less op must not fall back to a guessed site", after[seeded])
			}
			delete(after, seeded)
			if len(after) != 0 {
				t.Errorf("the project root gained %v; a site-less op writes nothing anywhere", pathSet(after))
			}
		})
	}
}
