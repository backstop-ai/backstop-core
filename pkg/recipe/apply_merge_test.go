package recipe

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	toml "github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

// mergeFixtureDir holds the TASK-002 / TASK-003 capture corpus: real project files
// copied byte-for-byte under neutral names (see the CAPTURE-SOURCE-merge-*.md
// fragments beside it), paired with the recipe-authored fragments that merge into
// them. Every expectation below is DERIVED from those two inputs — no expected
// merged file is stored, and nothing is copied back from what the merger emits.
var mergeFixtureDir = filepath.Join("testdata", "captured", "merge") // nosemgrep: go.core.no-global-mutable-state — immutable path constant expressed as a var only because filepath.Join is not constant-foldable

// mergeCoverage counts the three properties the captured fixtures were built to make
// falsifiable. A merger that returned the fragment wholesale, or the target
// untouched, or that overwrote a nested object shallowly, drives at least one of
// these to zero, so asserting all three are non-zero keeps the deep-merge walk from
// passing vacuously over an empty or degenerate union.
type mergeCoverage struct {
	newTopLevelNames      int // a name only the fragment declares, at the top level
	deepFragmentWins      int // a nested leaf (depth >= 2) the fragment added or overrode
	deepSurvivingSiblings int // a nested name (depth >= 2) only the target declares
}

// readMergeFixture returns a captured fixture's bytes.
func readMergeFixture(t *testing.T, fixture string) []byte {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(mergeFixtureDir, fixture))
	if err != nil {
		t.Fatalf("read captured merge fixture %q: %v", fixture, err)
	}

	return data
}

// structuredMergeCase is one row of the structured half of the merge-format matrix:
// a recipe declaring a single merge op, the captured target and authored fragment it
// names, the params the fragment is substituted through, and the codec the test
// decodes all three trees with.
type structuredMergeCase struct {
	recipeYAML      string
	targetFixture   string
	fragmentFixture string
	params          map[string]string
	decode          func(data []byte) (map[string]any, error)
}

// structuredMergeTrees is the three decoded trees a structured case compares.
type structuredMergeTrees struct {
	target   map[string]any
	fragment map[string]any
	merged   map[string]any
}

// runStructuredMerge applies the case's single merge op over copies of the captured
// fixtures in a temp project, then decodes the target, the substituted fragment, and
// the merged output with the SAME codec so the deep-merge law can be checked against
// the inputs rather than against a stored expectation.
func runStructuredMerge(t *testing.T, testCase structuredMergeCase) structuredMergeTrees {
	t.Helper()

	recipeDir := t.TempDir()
	projectRoot := t.TempDir()

	resolved := resolvedFromManifest(t, recipeDir, testCase.recipeYAML)
	if len(resolved.Manifest.Ops) != 1 {
		t.Fatalf("test recipe declares %d ops, want exactly one merge op", len(resolved.Manifest.Ops))
	}
	op := resolved.Manifest.Ops[0]

	copyUnder(t, projectRoot, op.Target, filepath.Join(mergeFixtureDir, testCase.targetFixture))
	copyUnder(t, recipeDir, op.Fragment, filepath.Join(mergeFixtureDir, testCase.fragmentFixture))

	result, err := Apply(resolved, ApplyOptions{Mode: ModeDirect, ProjectRoot: projectRoot, Params: testCase.params})
	if err != nil {
		t.Fatalf("Apply: unexpected error: %v", err)
	}
	if !reflect.DeepEqual(result.Written, []string{op.Target}) {
		t.Errorf("result.Written = %v, want exactly the declared target [%q]", result.Written, op.Target)
	}

	mergedBytes, err := os.ReadFile(filepath.Join(projectRoot, op.Target))
	if err != nil {
		t.Fatalf("read merged target: %v", err)
	}

	renderedFragment, err := Substitute(string(readMergeFixture(t, testCase.fragmentFixture)), testCase.params)
	if err != nil {
		t.Fatalf("substitute captured fragment %q: %v", testCase.fragmentFixture, err)
	}

	return structuredMergeTrees{
		target:   decodeTree(t, testCase.decode, readMergeFixture(t, testCase.targetFixture), "captured target"),
		fragment: decodeTree(t, testCase.decode, []byte(renderedFragment), "substituted fragment"),
		merged:   decodeTree(t, testCase.decode, mergedBytes, "merged output"),
	}
}

// decodeTree decodes one document with the case's codec, failing loud with the role
// of the document so a codec error is attributable.
func decodeTree(t *testing.T, decode func(data []byte) (map[string]any, error), data []byte, role string) map[string]any {
	t.Helper()

	tree, err := decode(data)
	if err != nil {
		t.Fatalf("decode %s: %v", role, err)
	}
	if len(tree) == 0 {
		t.Fatalf("decode %s: empty tree", role)
	}

	return tree
}

func decodeJSONTree(data []byte) (map[string]any, error) {
	var tree map[string]any
	if err := json.Unmarshal(data, &tree); err != nil {
		return nil, fmt.Errorf("decode %s: %w", mergeFormatJSON, err)
	}

	return tree, nil
}

func decodeYAMLTree(data []byte) (map[string]any, error) {
	var tree map[string]any
	if err := yaml.Unmarshal(data, &tree); err != nil {
		return nil, fmt.Errorf("decode %s: %w", mergeFormatYAML, err)
	}

	return tree, nil
}

func decodeTOMLTree(data []byte) (map[string]any, error) {
	var tree map[string]any
	if err := toml.Unmarshal(data, &tree); err != nil {
		return nil, fmt.Errorf("decode %s: %w", mergeFormatTOML, err)
	}

	return tree, nil
}

// assertDeepMerged checks the merged tree against the deep-merge law derived from the
// two inputs, then requires that all three falsifiable properties were actually
// exercised.
func assertDeepMerged(t *testing.T, trees structuredMergeTrees) {
	t.Helper()

	var coverage mergeCoverage
	walkDeepMerge(t, nil, trees.target, trees.fragment, trees.merged, &coverage)

	if coverage.newTopLevelNames == 0 {
		t.Error("no NEW top-level key was exercised; the fragment must add one the target lacks")
	}
	if coverage.deepFragmentWins == 0 {
		t.Error("no nested addition/override was exercised; the fragment must reach INSIDE an existing object")
	}
	if coverage.deepSurvivingSiblings == 0 {
		t.Error("no nested sibling survival was exercised; a shallow overwrite would be indistinguishable from a deep merge")
	}
}

// walkDeepMerge recursively checks one level of the merged tree against the target
// and fragment that produced it: a name only the target declares keeps its value, a
// name only the fragment declares appears with the fragment's value, two maps merge
// RECURSIVELY (never replace), any other conflict resolves to the fragment, and the
// merged tree carries nothing neither input declared.
func walkDeepMerge(t *testing.T, path []string, target, fragment, merged map[string]any, coverage *mergeCoverage) {
	t.Helper()

	for name, targetValue := range target {
		where := mergePath(path, name)

		mergedValue, present := merged[name]
		if !present {
			t.Errorf("%s: declared by the target but absent from the merged tree", where)
			continue
		}

		fragmentValue, inFragment := fragment[name]
		if !inFragment {
			if !reflect.DeepEqual(mergedValue, targetValue) {
				t.Errorf("%s: merged value %#v, want the target's untouched %#v", where, mergedValue, targetValue)
			}
			if len(path) > 0 {
				coverage.deepSurvivingSiblings++
			}
			continue
		}

		targetChild, targetIsMap := targetValue.(map[string]any)
		fragmentChild, fragmentIsMap := fragmentValue.(map[string]any)
		if targetIsMap && fragmentIsMap {
			mergedChild, mergedIsMap := mergedValue.(map[string]any)
			if !mergedIsMap {
				t.Errorf("%s: merged value is %T, want a map merged from both inputs", where, mergedValue)
				continue
			}
			walkDeepMerge(t, mergePath(path, name), targetChild, fragmentChild, mergedChild, coverage)
			continue
		}

		if !reflect.DeepEqual(mergedValue, fragmentValue) {
			t.Errorf("%s: merged value %#v, want the fragment's %#v", where, mergedValue, fragmentValue)
		}
		if len(path) > 0 && !reflect.DeepEqual(targetValue, fragmentValue) {
			coverage.deepFragmentWins++
		}
	}

	for name, fragmentValue := range fragment {
		if _, inTarget := target[name]; inTarget {
			continue
		}
		where := mergePath(path, name)

		mergedValue, present := merged[name]
		if !present {
			t.Errorf("%s: declared by the fragment but absent from the merged tree", where)
			continue
		}
		if !reflect.DeepEqual(mergedValue, fragmentValue) {
			t.Errorf("%s: merged value %#v, want the fragment's %#v", where, mergedValue, fragmentValue)
		}
		if len(path) == 0 {
			coverage.newTopLevelNames++
		} else {
			coverage.deepFragmentWins++
		}
	}

	for name := range merged {
		_, inTarget := target[name]
		_, inFragment := fragment[name]
		if !inTarget && !inFragment {
			t.Errorf("%s: present in the merged tree but declared by neither input", mergePath(path, name))
		}
	}
}

// mergePath returns a fresh slash-joined ancestor chain, copied rather than appended
// in place so sibling recursions cannot share backing storage.
func mergePath(path []string, name string) []string {
	extended := make([]string, 0, len(path)+1)
	extended = append(extended, path...)

	return append(extended, name)
}

// TestApply_MergeOp_Json proves a merge op DEEP-merges a declared fragment into a
// captured structured JSON target (CLM-005). The captured target's nested object
// holds fourteen sibling keys (two of them arrays) that a shallow overwrite would
// destroy, so the sibling-survival assertion discriminates deep from shallow. The op
// declares its format explicitly, exercising the declared-format route.
func TestApply_MergeOp_Json(t *testing.T) {
	const jsonMergeRecipe = `
kind: scaffolding
version: 1.0.0
ops:
  - id: op-merge-structured
    kind: merge
    target: config/target.json
    fragment: fragment.json
    format: json
`

	trees := runStructuredMerge(t, structuredMergeCase{
		recipeYAML:      jsonMergeRecipe,
		targetFixture:   "target.json",
		fragmentFixture: "fragment.json",
		decode:          decodeJSONTree,
	})
	assertDeepMerged(t, trees)
}

// TestApply_MergeOp_Yaml proves the same deep-merge law over the captured YAML pair
// (CLM-006). The op declares NO format, so the route falls back to the target's
// extension. The fragment reaches three levels deep (an addition inside an existing
// member of an existing object) and carries a {{ param }} placeholder, so this case
// also pins that a fragment is substituted before it is decoded.
func TestApply_MergeOp_Yaml(t *testing.T) {
	const yamlMergeRecipe = `
kind: scaffolding
version: 1.0.0
params:
  - name: github.ref
    default: refs/heads/main
ops:
  - id: op-merge-structured
    kind: merge
    target: workflows/target.yml
    fragment: fragment.yml
`

	trees := runStructuredMerge(t, structuredMergeCase{
		recipeYAML:      yamlMergeRecipe,
		targetFixture:   "target.yml",
		fragmentFixture: "fragment.yml",
		params:          map[string]string{"github.ref": "refs/heads/main"},
		decode:          decodeYAMLTree,
	})
	assertDeepMerged(t, trees)

	// The substitution pin, scoped to the fragment's OWN contribution. Only the
	// fragment is substituted: the captured target carries platform placeholders of
	// its own that must pass through untouched, so scanning the whole merged tree
	// would flag the target's text rather than the fragment's. Re-encoding just the
	// top-level names the fragment introduced isolates what the fragment wrote.
	var contributed strings.Builder
	for name := range trees.fragment {
		if _, inTarget := trees.target[name]; inTarget {
			continue
		}
		encoded, err := yaml.Marshal(trees.merged[name])
		if err != nil {
			t.Fatalf("re-encode the merged value of %q: %v", name, err)
		}
		contributed.Write(encoded)
	}
	if contributed.Len() == 0 {
		t.Fatal("the fragment introduced no new top-level key; the substitution pin would be vacuous")
	}
	if strings.Contains(contributed.String(), placeholderOpen) {
		t.Errorf("the fragment's contribution still carries an unsubstituted %s placeholder:\n%s", placeholderOpen, contributed.String())
	}
	if !strings.Contains(contributed.String(), "refs/heads/main") {
		t.Errorf("the fragment's contribution does not carry the substituted param default:\n%s", contributed.String())
	}
}

// TestApply_MergeOp_Toml proves the same law over the captured TOML pair (CLM-007),
// including a genuinely NESTED table: the fragment sets one key inside an existing
// nested table whose two other keys must survive, which a shallow table overwrite
// would drop. The op declares no format, so the extension routes it.
func TestApply_MergeOp_Toml(t *testing.T) {
	const tomlMergeRecipe = `
kind: scaffolding
version: 1.0.0
ops:
  - id: op-merge-structured
    kind: merge
    target: deploy/config.toml
    fragment: config.fragment.toml
`

	trees := runStructuredMerge(t, structuredMergeCase{
		recipeYAML:      tomlMergeRecipe,
		targetFixture:   "config.toml",
		fragmentFixture: "config.fragment.toml",
		decode:          decodeTOMLTree,
	})
	assertDeepMerged(t, trees)
}

// envEntry is one line of a key/value file as the TEST reads it: the raw bytes plus,
// for an assignment, the declared name and the opaque remainder after the first '='.
type envEntry struct {
	raw   string
	name  string
	value string
	isSet bool
}

// splitEnvEntries splits a key/value document into lines, dropping the empty tail a
// final newline would otherwise produce so an appended key can be positioned exactly
// against the target's line indices. A line is an assignment when it is neither blank nor a full-line comment and carries an
// '='; everything after that '=' is opaque, INCLUDING any trailing '#' text — the
// position this suite pins (see TestApply_MergeOp_DotEnv).
func splitEnvEntries(content string) []envEntry {
	raws := strings.Split(strings.TrimSuffix(content, "\n"), "\n")
	entries := make([]envEntry, 0, len(raws))
	for _, raw := range raws {
		entry := envEntry{raw: raw}
		trimmed := strings.TrimSpace(raw)
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			if at := strings.Index(raw, "="); at >= 0 {
				entry.name = strings.TrimSpace(raw[:at])
				entry.value = raw[at+1:]
				entry.isSet = true
			}
		}
		entries = append(entries, entry)
	}

	return entries
}

// TestApply_MergeOp_DotEnv proves a merge op merges a declared fragment into a
// captured .env target by KEY (CLM-008): an overriding key takes the fragment's
// value IN PLACE, a new key is appended, and every untouched line — full-line
// comments included — survives byte-identical at its original index.
//
// The captured target carries a trailing '#' comment on nearly every assignment,
// which forces a position. The one taken here, and pinned below: '#' opens a comment
// only at the START of a line. On an assignment line everything after the first '='
// is an OPAQUE value, so overriding a key replaces the WHOLE line and its trailing
// comment goes with it. A thin executor has no dialect knowledge to tell a comment
// from a value that contains '#', and untouched lines are copied verbatim rather than
// re-rendered, so no comment is lost anywhere the merge did not have to write.
func TestApply_MergeOp_DotEnv(t *testing.T) {
	const dotEnvMergeRecipe = `
kind: scaffolding
version: 1.0.0
ops:
  - id: op-merge-keyvalue
    kind: merge
    target: env/dotenv.env
    fragment: dotenv.fragment.env
    format: env
`

	recipeDir := t.TempDir()
	projectRoot := t.TempDir()

	resolved := resolvedFromManifest(t, recipeDir, dotEnvMergeRecipe)
	op := resolved.Manifest.Ops[0]
	copyUnder(t, projectRoot, op.Target, filepath.Join(mergeFixtureDir, "dotenv.env"))
	copyUnder(t, recipeDir, op.Fragment, filepath.Join(mergeFixtureDir, "dotenv.fragment.env"))

	targetContent := string(readMergeFixture(t, "dotenv.env"))
	targetEntries := splitEnvEntries(targetContent)
	fragmentEntries := splitEnvEntries(string(readMergeFixture(t, "dotenv.fragment.env")))

	// Derive the two roles from the inputs alone: an overriding name is one both
	// declare, a new name is one only the fragment declares.
	overrides := make(map[string]string)
	var appended []envEntry
	for _, entry := range fragmentEntries {
		if !entry.isSet {
			continue
		}
		found := false
		for _, targetEntry := range targetEntries {
			if targetEntry.isSet && targetEntry.name == entry.name {
				found = true
				break
			}
		}
		if found {
			overrides[entry.name] = entry.value
		} else {
			appended = append(appended, entry)
		}
	}
	if len(overrides) == 0 || len(appended) == 0 {
		t.Fatalf("fixture pair is degenerate: %d overriding and %d new keys, want at least one of each", len(overrides), len(appended))
	}

	if _, err := Apply(resolved, ApplyOptions{Mode: ModeDirect, ProjectRoot: projectRoot}); err != nil {
		t.Fatalf("Apply: unexpected error: %v", err)
	}

	mergedBytes, err := os.ReadFile(filepath.Join(projectRoot, op.Target))
	if err != nil {
		t.Fatalf("read merged target: %v", err)
	}
	mergedEntries := splitEnvEntries(string(mergedBytes))

	if endsWithNewline(string(mergedBytes)) != endsWithNewline(targetContent) {
		t.Errorf("merged trailing newline = %t, want the target's %t", endsWithNewline(string(mergedBytes)), endsWithNewline(targetContent))
	}
	if len(mergedEntries) != len(targetEntries)+len(appended) {
		t.Fatalf("merged line count = %d, want the target's %d plus %d appended", len(mergedEntries), len(targetEntries), len(appended))
	}

	// Every target line holds its index. Untouched lines are byte-identical; an
	// overridden line is exactly `NAME=<fragment value>` — no trailing comment
	// re-attached, which is the position this test pins.
	preservedInlineComments := 0
	overriddenWithComment := 0
	preservedFullLineComments := 0
	for index, entry := range targetEntries {
		got := mergedEntries[index].raw

		newValue, overridden := overrides[entry.name]
		if entry.isSet && overridden {
			want := entry.name + "=" + newValue
			if got != want {
				t.Errorf("line %d: merged %q, want the overridden %q", index, got, want)
			}
			if strings.Contains(entry.raw, " #") {
				overriddenWithComment++
			}
			continue
		}

		if got != entry.raw {
			t.Errorf("line %d: merged %q, want the untouched target line %q", index, got, entry.raw)
			continue
		}
		if entry.isSet && strings.Contains(entry.raw, " #") {
			preservedInlineComments++
		}
		if !entry.isSet && strings.HasPrefix(strings.TrimSpace(entry.raw), "#") {
			preservedFullLineComments++
		}
	}

	for offset, entry := range appended {
		got := mergedEntries[len(targetEntries)+offset].raw
		want := entry.name + "=" + entry.value
		if got != want {
			t.Errorf("appended line %d: merged %q, want %q", offset, got, want)
		}
	}

	// The fragment's own commentary is authoring text, not target content: merging by
	// KEY carries the fragment's assignments and nothing else.
	for _, entry := range fragmentEntries {
		if entry.isSet || strings.TrimSpace(entry.raw) == "" {
			continue
		}
		if strings.Contains(string(mergedBytes), entry.raw) {
			t.Errorf("fragment comment line %q leaked into the merged target; a key/value merge carries assignments only", entry.raw)
		}
	}

	if overriddenWithComment == 0 {
		t.Error("no overridden line carried a trailing comment; the comment position this test pins was never exercised")
	}
	if preservedInlineComments == 0 {
		t.Error("no untouched assignment carried a trailing comment; inline-comment survival was never exercised")
	}
	if preservedFullLineComments == 0 {
		t.Error("no full-line comment was exercised; the captured target declares several")
	}
}

// TestApply_MergeOp_UnsupportedFormatFailsLoud proves a merge op aimed at the
// captured UNSTRUCTURED file fails loud naming the target and the format it could not
// handle (CLM-009), leaves the target byte-identical, and reports no success. A
// best-effort text append here would silently corrupt a real document, so the closed
// format allowlist has to reject rather than degrade.
func TestApply_MergeOp_UnsupportedFormatFailsLoud(t *testing.T) {
	const unsupportedMergeRecipe = `
kind: scaffolding
version: 1.0.0
ops:
  - id: op-merge-unsupported
    kind: merge
    target: docs/unsupported-target.md
    fragment: fragment.json
`

	recipeDir := t.TempDir()
	projectRoot := t.TempDir()

	resolved := resolvedFromManifest(t, recipeDir, unsupportedMergeRecipe)
	op := resolved.Manifest.Ops[0]
	copyUnder(t, projectRoot, op.Target, filepath.Join(mergeFixtureDir, "unsupported-target.md"))
	copyUnder(t, recipeDir, op.Fragment, filepath.Join(mergeFixtureDir, "fragment.json"))

	before := readMergeFixture(t, "unsupported-target.md")

	result, err := Apply(resolved, ApplyOptions{Mode: ModeDirect, ProjectRoot: projectRoot})
	if err == nil {
		t.Fatal("Apply: merging into an unsupported format returned no error; the closed format allowlist must reject it")
	}
	if len(result.Written) != 0 || len(result.Preserved) != 0 {
		t.Errorf("failed Apply reported a verdict: %+v, want the zero result", result)
	}

	message := err.Error()
	if !strings.Contains(message, op.Target) {
		t.Errorf("error %q does not name the declared target %q", message, op.Target)
	}
	unsupported := strings.TrimPrefix(filepath.Ext(op.Target), ".")
	if !strings.Contains(message, unsupported) {
		t.Errorf("error %q does not name the unsupported format %q", message, unsupported)
	}

	after, err := os.ReadFile(filepath.Join(projectRoot, op.Target))
	if err != nil {
		t.Fatalf("read the untouched target: %v", err)
	}
	if !reflect.DeepEqual(after, before) {
		t.Errorf("target was modified by a failed merge:\ngot  %q\nwant %q", after, before)
	}
}

// TestApply_MergeOp_FailureLeavesTargetUntouched covers the merge executor's
// fail-loud paths (CLM-005..CLM-009). Each one must abort with a diagnostic that
// locates the offending declaration, report no verdict, and — where a target was
// already on disk — leave it BYTE-IDENTICAL: a merge that decoded half a document
// and wrote the remains would corrupt a real project file while looking like a
// partial success.
func TestApply_MergeOp_FailureLeavesTargetUntouched(t *testing.T) {
	const declaredTarget = "config/target.json"
	captured := readMergeFixture(t, "target.json")

	cases := []struct {
		name        string
		target      string
		targetBytes []byte
		fragment    string
		wantMessage string
	}{
		{
			name:        "the declared target escapes the project root",
			target:      "../escaped.json",
			fragment:    `{"added": true}`,
			wantMessage: "escapes",
		},
		{
			name:        "the declared fragment is absent from the recipe",
			target:      declaredTarget,
			targetBytes: captured,
			wantMessage: "read declared fragment",
		},
		{
			name:        "the fragment carries an undeclared placeholder",
			target:      declaredTarget,
			targetBytes: captured,
			fragment:    `{"added": "{{ undeclared }}"}`,
			wantMessage: "substitute declared fragment",
		},
		{
			name:        "the declared target is absent from the project",
			target:      declaredTarget,
			fragment:    `{"added": true}`,
			wantMessage: "read declared target",
		},
		{
			name:        "the fragment does not parse as the declared format",
			target:      declaredTarget,
			targetBytes: captured,
			fragment:    "{ this is not a structured document",
			wantMessage: "decode fragment",
		},
		{
			name:        "the target does not parse as the declared format",
			target:      declaredTarget,
			targetBytes: []byte("{ this is not a structured document"),
			fragment:    `{"added": true}`,
			wantMessage: "decode target",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			recipeDir := t.TempDir()
			projectRoot := t.TempDir()

			resolved := resolvedFromManifest(t, recipeDir, fmt.Sprintf(`
kind: scaffolding
version: 1.0.0
ops:
  - id: op-merge-failing
    kind: merge
    target: %s
    fragment: fragment.json
    format: json
`, testCase.target))
			op := resolved.Manifest.Ops[0]

			if testCase.fragment != "" {
				writeUnder(t, recipeDir, op.Fragment, testCase.fragment)
			}
			if testCase.targetBytes != nil {
				writeUnder(t, projectRoot, op.Target, string(testCase.targetBytes))
			}

			result, err := Apply(resolved, ApplyOptions{Mode: ModeDirect, ProjectRoot: projectRoot})
			if err == nil {
				t.Fatal("Apply: expected a fail-loud error, got none")
			}
			if !strings.Contains(err.Error(), testCase.wantMessage) {
				t.Errorf("error %q does not explain the failure with %q", err, testCase.wantMessage)
			}
			if len(result.Written) != 0 || len(result.Preserved) != 0 {
				t.Errorf("failed Apply reported a verdict: %+v, want the zero result", result)
			}

			if testCase.targetBytes == nil {
				return
			}
			after, readErr := os.ReadFile(filepath.Join(projectRoot, op.Target))
			if readErr != nil {
				t.Fatalf("read the target after the failed merge: %v", readErr)
			}
			if !reflect.DeepEqual(after, testCase.targetBytes) {
				t.Errorf("target changed under a failed merge:\ngot  %q\nwant %q", after, testCase.targetBytes)
			}
		})
	}
}

// TestApply_MergeOp_UnreadableFragmentPathNamesThePathOnlyContract pins the
// SECOND layer of ISSUE-081's path-only canon (CLM-004).
//
// Manifest validation refuses a fragment that is empty, spans lines, or carries a
// `{{` placeholder. It deliberately does NOT refuse a single-line, placeholder-
// free value that is really inline content but is syntactically a valid relative
// filename — nothing at parse can tell `{"adopted_by": "app"}` from a file of
// that name without a CONTENT SNIFF, which would mean guessing at document syntax
// the manifest reader has no business knowing.
//
// So that residual case survives to APPLY, and this is what it must do there:
// name the path-only contract and the declared value, rather than leaking a bare
// "no such file or directory". The bare ENOENT is exactly what the live
// 2026-07-25 dogfood produced and could not be acted on — an author who wrote
// inline content had no way to learn the field is a path. Asserting only that the
// error is non-nil would leave that undiagnosable message passing.
func TestApply_MergeOp_UnreadableFragmentPathNamesThePathOnlyContract(t *testing.T) {
	const declaredTarget = "config/target.json"

	cases := []struct {
		name     string
		fragment string
	}{
		{
			// THE residual case: inline content that parse cannot distinguish
			// from a filename.
			name:     "a single-line inline document that is syntactically a filename",
			fragment: `{"adopted_by": "app"}`,
		},
		{
			name:     "a declared path naming nothing under the recipe directory",
			fragment: "payload/absent.fragment.json",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			recipeDir := t.TempDir()
			projectRoot := t.TempDir()
			captured := readMergeFixture(t, "target.json")

			// The manifest must PARSE: this value is single-line, non-empty and
			// placeholder-free, so the parse-time check lets it through by design.
			resolved := resolvedFromManifest(t, recipeDir, fmt.Sprintf(`
kind: scaffolding
version: 1.0.0
ops:
  - id: op-merge-unreadable-fragment
    kind: merge
    target: %s
    fragment: '%s'
    format: json
`, declaredTarget, testCase.fragment))
			op := resolved.Manifest.Ops[0]
			if op.Fragment != testCase.fragment {
				t.Fatalf("Fragment = %q, want the declared value %q verbatim; the case is not exercising what it claims", op.Fragment, testCase.fragment)
			}

			writeUnder(t, projectRoot, op.Target, string(captured))

			result, err := Apply(resolved, ApplyOptions{Mode: ModeDirect, ProjectRoot: projectRoot})
			if err == nil {
				t.Fatal("Apply: a fragment naming nothing on disk must fail loud, got none")
			}
			msg := err.Error()

			// The operation that failed, the value that failed it, and the CANON
			// the author needs in order to fix it.
			for _, want := range []string{
				"read declared fragment",
				testCase.fragment,
				"recipe-directory-relative path",
				"read from disk",
			} {
				if !strings.Contains(msg, want) {
					t.Errorf("error must carry %q so the author learns the fragment field is a PATH, got: %v", want, err)
				}
			}
			// The underlying ENOENT stays wrapped — it is still useful, it just
			// must not be the whole story.
			if !strings.Contains(msg, "no such file or directory") {
				t.Errorf("error must keep the underlying read failure wrapped, got: %v", err)
			}

			if len(result.Written) != 0 || len(result.Preserved) != 0 {
				t.Errorf("failed Apply reported a verdict: %+v, want the zero result", result)
			}
			after, readErr := os.ReadFile(filepath.Join(projectRoot, op.Target))
			if readErr != nil {
				t.Fatalf("read the target after the failed merge: %v", readErr)
			}
			if !reflect.DeepEqual(after, captured) {
				t.Errorf("target changed under a failed merge:\ngot  %q\nwant %q", after, captured)
			}
		})
	}
}
