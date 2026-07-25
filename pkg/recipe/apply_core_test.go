package recipe

import (
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// The applier-core recipes are declared as recipe.yml DATA and parsed through the
// real ParseRecipeManifest, never hand-built as typed Go literals: every target,
// payload, anchor, and param below is read back off the parsed manifest, so no
// fixture path is a Go-source literal the applier could accidentally agree with.

// orderedRecipeForward declares three ops whose DECLARED order is
// seed -> alpha -> bravo. Sorting the ops by id would run op-seed LAST (it sorts
// after op-alpha/op-bravo), so a sorted implementation cannot even reach the
// inserts — and a map-ranged one is non-deterministic. Both inserts share ONE
// anchor, which makes the surviving order observable in the output bytes.
const orderedRecipeForward = `
kind: scaffolding
version: 1.0.0
ops:
  - id: op-seed
    kind: create
    target: generated/manifest.txt
    payload: seed.txt
  - id: op-alpha
    kind: insert
    target: generated/manifest.txt
    anchor: "# SLOTS"
    snippet: "\nalpha"
    manual: "Add the alpha line directly beneath the SLOTS marker."
  - id: op-bravo
    kind: insert
    target: generated/manifest.txt
    anchor: "# SLOTS"
    snippet: "\nbravo"
    manual: "Add the bravo line directly beneath the SLOTS marker."
`

// orderedRecipeReversed is orderedRecipeForward with the two inserts REVERSED and
// nothing else changed. It is the falsifier: an applier that honors the declared
// sequence produces the mirrored output, while any implementation that imposes its
// own order produces the same bytes for both recipes.
const orderedRecipeReversed = `
kind: scaffolding
version: 1.0.0
ops:
  - id: op-seed
    kind: create
    target: generated/manifest.txt
    payload: seed.txt
  - id: op-bravo
    kind: insert
    target: generated/manifest.txt
    anchor: "# SLOTS"
    snippet: "\nbravo"
    manual: "Add the bravo line directly beneath the SLOTS marker."
  - id: op-alpha
    kind: insert
    target: generated/manifest.txt
    anchor: "# SLOTS"
    snippet: "\nalpha"
    manual: "Add the alpha line directly beneath the SLOTS marker."
`

// seedPayload is the create op's payload: an anchor line and a trailing marker, so
// a snippet spliced at the anchor is distinguishable from one appended at EOF.
const seedPayload = "# SLOTS\n# END\n"

// capturedExpectedName is the neutral fixture name of the UNMODIFIED capture the
// templated payload must materialize back into (see
// testdata/captured/CAPTURE-SOURCE-create.md). The capture corpus deliberately
// uses neutral names so no test has to name a real project-manifest filename.
const capturedExpectedName = "payload.expected.json"

// resolvedFromManifest parses recipeYAML through the production parser and pairs it
// with dir, producing the same ResolvedRecipe shape ResolveRecipe returns.
func resolvedFromManifest(t *testing.T, dir string, recipeYAML string) *ResolvedRecipe {
	t.Helper()

	manifest, err := ParseRecipeManifest([]byte(recipeYAML))
	if err != nil {
		t.Fatalf("parse test recipe manifest: %v", err)
	}

	return &ResolvedRecipe{
		Ref:      RecipeRef{Pack: fixturePackName, Recipe: "starter", Version: manifest.Version},
		Dir:      dir,
		Manifest: manifest,
	}
}

// writeUnder writes content at relPath under root, creating parent directories.
func writeUnder(t *testing.T, root string, relPath string, content string) string {
	t.Helper()

	path := filepath.Join(root, relPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir for %q: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %q: %v", path, err)
	}

	return path
}

// copyUnder copies srcPath to relPath under root, so a CAPTURED fixture reaches the
// recipe directory as bytes rather than as a re-typed literal.
func copyUnder(t *testing.T, root string, relPath string, srcPath string) {
	t.Helper()

	data, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatalf("read captured fixture %q: %v", srcPath, err)
	}
	writeUnder(t, root, relPath, string(data))
}

// snapshotTree walks the WHOLE root and returns every file's slash-relative path
// mapped to its contents. Assertions compare against this rather than probing known
// paths, so a write to an unexpected location is caught instead of ignored.
func snapshotTree(t *testing.T, root string) map[string]string {
	t.Helper()

	tree := make(map[string]string)
	walkErr := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
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
		tree[filepath.ToSlash(rel)] = string(data)
		return nil
	})
	if walkErr != nil {
		t.Fatalf("walk project root %q: %v", root, walkErr)
	}

	return tree
}

// pathSet reduces a tree snapshot to its path set for set-equality assertions.
func pathSet(tree map[string]string) []string {
	paths := make([]string, 0, len(tree))
	for path := range tree {
		paths = append(paths, path)
	}
	return paths
}

// TestApply_RunsOpsInDeclaredOrder proves the applier executes ops in the recipe's
// DECLARED sequence (CLM-001). Both inserts splice at one shared anchor, so the
// later op's snippet ends up nearest the anchor and the output records which ran
// first. The reversed sub-case is the falsifier: an applier that sorted or
// map-ranged the ops would emit identical bytes for both recipes (and, since
// op-seed sorts last, could not even reach the inserts).
func TestApply_RunsOpsInDeclaredOrder(t *testing.T) {
	cases := []struct {
		name       string
		recipeYAML string
		want       string
	}{
		{name: "declared_seed_alpha_bravo", recipeYAML: orderedRecipeForward, want: "# SLOTS\nbravo\nalpha\n# END\n"},
		{name: "declared_seed_bravo_alpha", recipeYAML: orderedRecipeReversed, want: "# SLOTS\nalpha\nbravo\n# END\n"},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			recipeDir := t.TempDir()
			projectRoot := t.TempDir()

			resolved := resolvedFromManifest(t, recipeDir, testCase.recipeYAML)
			createOp := resolved.Manifest.Ops[0]
			writeUnder(t, recipeDir, createOp.Payload, seedPayload)

			result, err := Apply(resolved, ApplyOptions{Mode: ModeDirect, ProjectRoot: projectRoot})
			if err != nil {
				t.Fatalf("Apply: unexpected error: %v", err)
			}

			tree := snapshotTree(t, projectRoot)
			got, materialized := tree[createOp.Target]
			if !materialized {
				t.Fatalf("declared target %q was not written; project tree = %v", createOp.Target, pathSet(tree))
			}
			if got != testCase.want {
				t.Errorf("applied content =\n%q\nwant\n%q", got, testCase.want)
			}
			if len(result.Written) != 1 || result.Written[0] != createOp.Target {
				t.Errorf("result.Written = %v, want exactly [%q]", result.Written, createOp.Target)
			}
		})
	}
}

// TestApply_TargetComesFromRecipeNotApplier proves the applier contributes NO path
// (CLM-002): the whole project root is walked and its file set must equal exactly
// the recipe-declared target — no sibling, no default location, no implied
// directory convention, no stray marker file.
func TestApply_TargetComesFromRecipeNotApplier(t *testing.T) {
	const singleCreateRecipe = `
kind: scaffolding
version: 1.0.0
ops:
  - id: op-only
    kind: create
    target: deeply/nested/declared/out.conf
    payload: only.txt
`

	recipeDir := t.TempDir()
	projectRoot := t.TempDir()

	resolved := resolvedFromManifest(t, recipeDir, singleCreateRecipe)
	createOp := resolved.Manifest.Ops[0]
	writeUnder(t, recipeDir, createOp.Payload, "declared payload body\n")

	if _, err := Apply(resolved, ApplyOptions{Mode: ModeDirect, ProjectRoot: projectRoot}); err != nil {
		t.Fatalf("Apply: unexpected error: %v", err)
	}

	tree := snapshotTree(t, projectRoot)
	// The apply also writes the tracked adoption record (REQ-005) — the applier's
	// own record of what this project adopted, not a recipe-declared target. Both
	// expected files are NAMED rather than filtered out by count, so a third file
	// appearing anywhere under the root still fails this assertion.
	wantFiles := []string{createOp.Target, AdoptionRecordName}
	sort.Strings(wantFiles)
	gotFiles := pathSet(tree)
	sort.Strings(gotFiles)
	if !reflect.DeepEqual(gotFiles, wantFiles) {
		t.Fatalf("project root holds %v, want exactly the declared target + adoption record and nothing else (%v)", gotFiles, wantFiles)
	}
	if _, atDeclaredTarget := tree[createOp.Target]; !atDeclaredTarget {
		t.Errorf("written file set = %v, want exactly the recipe-declared target %q", pathSet(tree), createOp.Target)
	}
}

// TestApply_EmptyOpsNoop proves a recipe with zero ops applies as a clean no-op
// (CLM-003): no error, and the project tree — paths AND contents — is byte-identical
// afterwards. A pre-seeded consumer file makes "wrote nothing" distinguishable from
// "wrote over an empty directory".
func TestApply_EmptyOpsNoop(t *testing.T) {
	projectRoot := t.TempDir()
	writeUnder(t, projectRoot, "existing/consumer.conf", "consumer content the applier must not touch\n")
	before := snapshotTree(t, projectRoot)

	resolved := &ResolvedRecipe{
		Ref:      RecipeRef{Pack: fixturePackName, Recipe: "empty", Version: "1.0.0"},
		Dir:      t.TempDir(),
		Manifest: &RecipeManifest{Kind: KindScaffolding, Version: "1.0.0"},
	}

	result, err := Apply(resolved, ApplyOptions{Mode: ModeDirect, ProjectRoot: projectRoot})
	if err != nil {
		t.Fatalf("Apply over a zero-op recipe: unexpected error: %v", err)
	}
	if len(result.Written) != 0 {
		t.Errorf("result.Written = %v, want empty — a zero-op recipe writes nothing", result.Written)
	}

	after := snapshotTree(t, projectRoot)
	if len(after) != len(before) {
		t.Fatalf("project tree changed: before %v, after %v", pathSet(before), pathSet(after))
	}
	for path, content := range before {
		if after[path] != content {
			t.Errorf("file %q changed: before %q, after %q", path, content, after[path])
		}
	}
}

// TestApply_CreateOp_MaterializesCapturedFixture proves a create op materializes the
// TASK-001 CAPTURED payload with its {{ }} params resolved (CLM-004). The comparison
// is BYTE-for-byte against the unmodified capture, so an applier that skipped
// substitution (leaving the placeholders in place) or re-wrapped the content fails.
func TestApply_CreateOp_MaterializesCapturedFixture(t *testing.T) {
	const capturedCreateRecipe = `
kind: scaffolding
version: 1.0.0
params:
  - name: cron_path
    required: true
    default: "/api/reconcile"
  - name: cron_schedule
    required: true
    default: "0 * * * *"
ops:
  - id: op-captured
    kind: create
    target: generated/captured-output
    payload: payload.json.tmpl
`

	capturedDir := filepath.Join("testdata", "captured", "create")
	recipeDir := t.TempDir()
	projectRoot := t.TempDir()

	resolved := resolvedFromManifest(t, recipeDir, capturedCreateRecipe)
	createOp := resolved.Manifest.Ops[0]
	copyUnder(t, recipeDir, createOp.Payload, filepath.Join(capturedDir, createOp.Payload))

	want, err := os.ReadFile(filepath.Join(capturedDir, capturedExpectedName))
	if err != nil {
		t.Fatalf("read captured expected materialization: %v", err)
	}

	if _, applyErr := Apply(resolved, ApplyOptions{Mode: ModeDirect, ProjectRoot: projectRoot}); applyErr != nil {
		t.Fatalf("Apply: unexpected error: %v", applyErr)
	}

	got, err := os.ReadFile(filepath.Join(projectRoot, createOp.Target))
	if err != nil {
		t.Fatalf("read materialized output: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("materialized output =\n%q\nwant the captured bytes\n%q", string(got), string(want))
	}
}

// TestApply_UserOwnedFileNeverClobbered proves a create op never overwrites a
// consumer-owned file (CLM-018) — one this recipe did not produce and has no
// recipe-owned provenance for. The on-disk bytes must survive untouched, and the
// result must REPORT the file rather than claim it was written: silently succeeding
// as if written is the failure this guards.
func TestApply_UserOwnedFileNeverClobbered(t *testing.T) {
	const collidingCreateRecipe = `
kind: scaffolding
version: 1.0.0
ops:
  - id: op-collides
    kind: create
    target: existing/consumer-owned.conf
    payload: replacement.txt
`

	recipeDir := t.TempDir()
	projectRoot := t.TempDir()

	resolved := resolvedFromManifest(t, recipeDir, collidingCreateRecipe)
	createOp := resolved.Manifest.Ops[0]
	writeUnder(t, recipeDir, createOp.Payload, "payload the recipe would have written\n")

	const consumerContent = "hand-written consumer content the applier must never clobber\n"
	writeUnder(t, projectRoot, createOp.Target, consumerContent)

	result, err := Apply(resolved, ApplyOptions{Mode: ModeDirect, ProjectRoot: projectRoot})
	if err != nil {
		t.Fatalf("Apply: unexpected error: %v", err)
	}

	tree := snapshotTree(t, projectRoot)
	if tree[createOp.Target] != consumerContent {
		t.Errorf("consumer-owned file was modified: got %q, want %q", tree[createOp.Target], consumerContent)
	}
	for _, written := range result.Written {
		if written == createOp.Target {
			t.Errorf("result.Written reports %q as written, but the consumer-owned file was (correctly) left alone", written)
		}
	}

	var reported bool
	for _, preserved := range result.Preserved {
		if preserved.Path != createOp.Target {
			continue
		}
		reported = true
		if preserved.CoveringWaiver != "" {
			t.Errorf("preserved entry carries CoveringWaiver %q; a user-owned file is protected outright, and the applier never authors a waiver", preserved.CoveringWaiver)
		}
	}
	if !reported {
		t.Errorf("result.Preserved = %+v, want an entry reporting the untouched consumer-owned target %q", result.Preserved, createOp.Target)
	}
}

// TestApply_FailsLoudOnUnusableInput proves the applier refuses to run without the
// two things it cannot infer: a resolved recipe and a project root. Neither has a
// default — an applier that invented a root would write outside the caller's
// control, and one that tolerated a nil manifest would report a vacuous success.
func TestApply_FailsLoudOnUnusableInput(t *testing.T) {
	cases := []struct {
		name     string
		resolved *ResolvedRecipe
		opts     ApplyOptions
	}{
		{name: "nil_resolved_recipe", resolved: nil, opts: ApplyOptions{ProjectRoot: t.TempDir()}},
		{name: "nil_manifest", resolved: &ResolvedRecipe{Ref: RecipeRef{Pack: fixturePackName, Recipe: "starter", Version: "1.0.0"}}, opts: ApplyOptions{ProjectRoot: t.TempDir()}},
		{name: "no_project_root", resolved: &ResolvedRecipe{Manifest: &RecipeManifest{Kind: KindScaffolding, Version: "1.0.0"}}, opts: ApplyOptions{ProjectRoot: "  "}},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := Apply(testCase.resolved, testCase.opts)
			if err == nil {
				t.Fatalf("Apply: expected a fail-loud error, got nil")
			}
			if len(result.Written) != 0 || len(result.Preserved) != 0 {
				t.Errorf("result = %+v, want the zero verdict alongside the error", result)
			}
		})
	}
}

// TestApply_CreateOp_FailsLoudOnUnreachablePayloadOrTarget proves every create-op
// path the applier cannot honor is an error rather than a guess: a target that
// escapes the project root, an undeclared path, a payload the recipe directory does
// not hold, a payload whose placeholder no param declares, and a target whose parent
// is not a directory. In every case the project root must stay empty — the applier
// never falls back to a nearby location.
func TestApply_CreateOp_FailsLoudOnUnreachablePayloadOrTarget(t *testing.T) {
	cases := []struct {
		name         string
		recipeYAML   string
		payloadBody  string
		writePayload bool
		blockerPath  string
		wantMessage  string
	}{
		{
			name: "target_escapes_project_root",
			recipeYAML: `
kind: scaffolding
version: 1.0.0
ops:
  - id: op-escape
    kind: create
    target: ../escaped.conf
    payload: body.txt
`,
			payloadBody:  "body\n",
			writePayload: true,
			wantMessage:  "escapes",
		},
		{
			name: "target_not_declared",
			recipeYAML: `
kind: scaffolding
version: 1.0.0
ops:
  - id: op-no-target
    kind: create
    payload: body.txt
`,
			payloadBody:  "body\n",
			writePayload: true,
			wantMessage:  "declares no path",
		},
		{
			name: "payload_absent_from_recipe_dir",
			recipeYAML: `
kind: scaffolding
version: 1.0.0
ops:
  - id: op-missing-payload
    kind: create
    target: generated/out.conf
    payload: absent.txt
`,
			writePayload: false,
			wantMessage:  "absent.txt",
		},
		{
			name: "payload_placeholder_no_param_declares",
			recipeYAML: `
kind: scaffolding
version: 1.0.0
ops:
  - id: op-undeclared-param
    kind: create
    target: generated/out.conf
    payload: body.txt
`,
			payloadBody:  "value = {{ never_declared }}\n",
			writePayload: true,
			wantMessage:  "never_declared",
		},
		{
			name: "target_parent_is_not_a_directory",
			recipeYAML: `
kind: scaffolding
version: 1.0.0
ops:
  - id: op-blocked-parent
    kind: create
    target: blocker/out.conf
    payload: body.txt
`,
			payloadBody:  "body\n",
			writePayload: true,
			blockerPath:  "blocker",
			wantMessage:  "blocker",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			recipeDir := t.TempDir()
			projectRoot := t.TempDir()

			resolved := resolvedFromManifest(t, recipeDir, testCase.recipeYAML)
			createOp := resolved.Manifest.Ops[0]
			if testCase.writePayload {
				writeUnder(t, recipeDir, createOp.Payload, testCase.payloadBody)
			}
			if testCase.blockerPath != "" {
				writeUnder(t, projectRoot, testCase.blockerPath, "an ordinary file where a directory would have to be\n")
			}

			result, err := Apply(resolved, ApplyOptions{Mode: ModeDirect, ProjectRoot: projectRoot})
			if err == nil {
				t.Fatalf("Apply: expected a fail-loud error, got nil (result %+v)", result)
			}
			if !strings.Contains(err.Error(), testCase.wantMessage) {
				t.Errorf("error %q does not mention %q", err, testCase.wantMessage)
			}
			if !strings.Contains(err.Error(), createOp.ID) {
				t.Errorf("error %q does not locate the offending op %q", err, createOp.ID)
			}

			tree := snapshotTree(t, projectRoot)
			delete(tree, testCase.blockerPath)
			if len(tree) != 0 {
				t.Errorf("project root gained %v; a create the applier cannot honor must write nothing", pathSet(tree))
			}
		})
	}
}

// TestApply_InsertOp_FailsLoudOnUnreachableSite proves an insert whose site cannot be
// reached is an error, never a silent skip and never an append at EOF: an absent
// target, an undeclared anchor, and an anchor the target does not contain all fail,
// and the target's bytes are left exactly as they were.
func TestApply_InsertOp_FailsLoudOnUnreachableSite(t *testing.T) {
	const seedContent = "# SLOTS\n# END\n"

	cases := []struct {
		name        string
		recipeYAML  string
		seedTarget  bool
		wantMessage string
	}{
		{
			name: "target_absent",
			recipeYAML: `
kind: scaffolding
version: 1.0.0
ops:
  - id: op-absent-target
    kind: insert
    target: generated/absent.txt
    anchor: "# SLOTS"
    snippet: "\nline"
    manual: "Add the line beneath the SLOTS marker by hand."
`,
			seedTarget:  false,
			wantMessage: "generated/absent.txt",
		},
		{
			name: "anchor_not_declared",
			recipeYAML: `
kind: scaffolding
version: 1.0.0
ops:
  - id: op-no-anchor
    kind: insert
    target: generated/seeded.txt
    snippet: "\nline"
    manual: "Add the line beneath the SLOTS marker by hand."
`,
			seedTarget:  true,
			wantMessage: "no anchor",
		},
		{
			name: "anchor_absent_from_target",
			recipeYAML: `
kind: scaffolding
version: 1.0.0
ops:
  - id: op-unmatched-anchor
    kind: insert
    target: generated/seeded.txt
    anchor: "# NOT PRESENT"
    snippet: "\nline"
    manual: "Add the line beneath the SLOTS marker by hand."
`,
			seedTarget:  true,
			wantMessage: "# NOT PRESENT",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			projectRoot := t.TempDir()
			resolved := resolvedFromManifest(t, t.TempDir(), testCase.recipeYAML)
			insertOp := resolved.Manifest.Ops[0]
			if testCase.seedTarget {
				writeUnder(t, projectRoot, insertOp.Target, seedContent)
			}

			result, err := Apply(resolved, ApplyOptions{Mode: ModeDirect, ProjectRoot: projectRoot})
			if err == nil {
				t.Fatalf("Apply: expected a fail-loud error, got nil (result %+v)", result)
			}
			if !strings.Contains(err.Error(), testCase.wantMessage) {
				t.Errorf("error %q does not mention %q", err, testCase.wantMessage)
			}
			if len(result.Written) != 0 {
				t.Errorf("result.Written = %v, want empty alongside the error", result.Written)
			}

			tree := snapshotTree(t, projectRoot)
			if testCase.seedTarget && tree[insertOp.Target] != seedContent {
				t.Errorf("target content = %q, want the untouched seed %q", tree[insertOp.Target], seedContent)
			}
			if !testCase.seedTarget && len(tree) != 0 {
				t.Errorf("project root gained %v; an unreachable insert writes nothing", pathSet(tree))
			}
		})
	}
}

// TestApply_SuppliedParamsOverrideDeclaredDefaults proves the substitution scope is
// the recipe's declared defaults OVERRIDDEN by the caller's params — the default
// supplies the value when the caller says nothing, and the supplied value wins when
// it does. A declared REQUIRED param with no default and no supplied value stays
// ABSENT so its placeholder fails loud rather than resolving to an empty string.
func TestApply_SuppliedParamsOverrideDeclaredDefaults(t *testing.T) {
	const paramRecipe = `
kind: scaffolding
version: 1.0.0
params:
  - name: defaulted
    default: "from-the-default"
  - name: overridden
    default: "from-the-default"
  - name: required_no_default
    required: true
ops:
  - id: op-params
    kind: create
    target: generated/params.conf
    payload: body.txt
`

	recipeDir := t.TempDir()
	resolved := resolvedFromManifest(t, recipeDir, paramRecipe)
	createOp := resolved.Manifest.Ops[0]
	writeUnder(t, recipeDir, createOp.Payload, "a={{ defaulted }} b={{ overridden }}\n")

	projectRoot := t.TempDir()
	if _, err := Apply(resolved, ApplyOptions{
		Mode:        ModeDirect,
		ProjectRoot: projectRoot,
		Params:      map[string]string{"overridden": "from-the-caller"},
	}); err != nil {
		t.Fatalf("Apply: unexpected error: %v", err)
	}

	got := snapshotTree(t, projectRoot)[createOp.Target]
	want := "a=from-the-default b=from-the-caller\n"
	if got != want {
		t.Errorf("substituted output = %q, want %q", got, want)
	}

	// The required param declares no default and is supplied by nobody, so its
	// placeholder must fail loud rather than blank out.
	requiredRoot := t.TempDir()
	writeUnder(t, recipeDir, createOp.Payload, "c={{ required_no_default }}\n")
	_, err := Apply(resolved, ApplyOptions{Mode: ModeDirect, ProjectRoot: requiredRoot})
	if err == nil {
		t.Fatalf("Apply with an unsatisfied required param: expected a fail-loud error, got nil")
	}
	if !strings.Contains(err.Error(), "required_no_default") {
		t.Errorf("error %q does not name the unsatisfied param", err)
	}
}

// TestApply_TransformWithoutDispatchFailsLoud proves the transform seam has no
// in-package default: reaching a transform op with a nil dispatch is a fail-loud
// configuration error naming the op, never a silent no-op. A no-op would let every
// transform assertion pass while nothing ran — the failure mode the injected seam
// exists to prevent.
func TestApply_TransformWithoutDispatchFailsLoud(t *testing.T) {
	const transformRecipe = `
kind: implementing
version: 1.0.0
transform_rules:
  - rules/rewrite.yml
ops:
  - id: op-transform
    kind: transform
    target: generated/target.conf
    rule: rules/rewrite.yml
    manual: "Apply the rewrite by hand as described in the pack."
`

	projectRoot := t.TempDir()
	resolved := resolvedFromManifest(t, t.TempDir(), transformRecipe)
	transformOp := resolved.Manifest.Ops[0]

	result, err := Apply(resolved, ApplyOptions{Mode: ModeDirect, ProjectRoot: projectRoot})
	if err == nil {
		t.Fatalf("Apply with a nil transform dispatch: expected a fail-loud error, got nil (result %+v)", result)
	}
	if !strings.Contains(err.Error(), transformOp.ID) {
		t.Errorf("error %q does not name the offending op %q", err, transformOp.ID)
	}
	if len(result.Written) != 0 {
		t.Errorf("result.Written = %v, want empty alongside the error", result.Written)
	}
	if len(snapshotTree(t, projectRoot)) != 0 {
		t.Errorf("project root was written to despite the missing dispatch")
	}
}

// TestApplyAll_RunsRecipesSequentiallyInGivenOrder proves multi-recipe apply is
// strictly sequential in the ORDER GIVEN: the second recipe inserts into the file
// the first creates, so it can only succeed downstream of it. The reversed order is
// the falsifier — an implementation that reordered or interleaved the recipes to
// make the run "work" would pass the forward case and hide the dependency; here it
// fails loud and returns no partial results.
func TestApplyAll_RunsRecipesSequentiallyInGivenOrder(t *testing.T) {
	const creatingRecipe = `
kind: scaffolding
version: 1.0.0
ops:
  - id: op-create
    kind: create
    target: generated/shared.txt
    payload: seed.txt
`
	const insertingRecipe = `
kind: scaffolding
version: 1.0.0
ops:
  - id: op-insert
    kind: insert
    target: generated/shared.txt
    anchor: "# SLOTS"
    snippet: "\nfrom-the-second-recipe"
    manual: "Add the line beneath the SLOTS marker by hand."
`

	creatingDir := t.TempDir()
	creating := resolvedFromManifest(t, creatingDir, creatingRecipe)
	writeUnder(t, creatingDir, creating.Manifest.Ops[0].Payload, seedPayload)
	inserting := resolvedFromManifest(t, t.TempDir(), insertingRecipe)
	target := creating.Manifest.Ops[0].Target

	projectRoot := t.TempDir()
	results, err := ApplyAll([]*ResolvedRecipe{creating, inserting}, ApplyOptions{Mode: ModeDirect, ProjectRoot: projectRoot})
	if err != nil {
		t.Fatalf("ApplyAll: unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("ApplyAll returned %d results, want one per recipe", len(results))
	}
	for i, result := range results {
		if len(result.Written) != 1 || result.Written[0] != target {
			t.Errorf("results[%d].Written = %v, want exactly [%q]", i, result.Written, target)
		}
	}

	got := snapshotTree(t, projectRoot)[target]
	want := "# SLOTS\nfrom-the-second-recipe\n# END\n"
	if got != want {
		t.Errorf("composed content =\n%q\nwant\n%q", got, want)
	}

	reversedRoot := t.TempDir()
	reversed, err := ApplyAll([]*ResolvedRecipe{inserting, creating}, ApplyOptions{Mode: ModeDirect, ProjectRoot: reversedRoot})
	if err == nil {
		t.Fatalf("ApplyAll in the reversed order: expected a fail-loud error, got nil")
	}
	if reversed != nil {
		t.Errorf("ApplyAll returned %+v alongside the error, want no partial results", reversed)
	}
	if len(snapshotTree(t, reversedRoot)) != 0 {
		t.Errorf("the reversed run wrote files; the recipes must not be reordered to make the run succeed")
	}
}
