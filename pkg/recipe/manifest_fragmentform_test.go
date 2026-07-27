package recipe

import (
	"strings"
	"testing"
)

// A merge op's `fragment:` is a RECIPE-DIRECTORY-RELATIVE PATH to a file read
// from disk. That is what applyMerge has always done — it resolves the declared
// value under the recipe directory and reads it — and ISSUE-081's Decision
// (2026-07-27) pins it as the canon, making an inline block a PARSE ERROR with a
// clear message rather than a silently-accepted alternate form.
//
// The refusal predicate is exactly three causes, each with its OWN message,
// because "invalid fragment" for three different defects is the same
// undiagnosable failure ISSUE-081 was filed about:
//
//	empty/whitespace  a merge op cannot merge nothing, and the apply-time
//	                  failure today is a bare "the op declares no path";
//	newline           a path never spans lines — this is the YAML block-scalar
//	                  shape the Decision names;
//	`{{`              guaranteed-dead syntax: a fragment PATH is handed to
//	                  resolveUnder verbatim and never substituted (only the
//	                  fragment FILE'S CONTENT is), so its only possible outcome
//	                  is an ENOENT at apply time.
//
// DELIBERATELY NOT REFUSED: a single-line, placeholder-free value that is really
// inline content but is syntactically a valid relative filename (e.g.
// `fragment: '{"a": 1}'`). Nothing at parse can tell that from a file named
// `{"a": 1}` without a CONTENT SNIFF — guessing at document syntax the manifest
// reader has no business knowing. That residual case is caught at APPLY, where
// the failure names the path-only contract instead of leaking a bare ENOENT
// (TestApply_MergeOp_UnreadableFragmentPathNamesThePathOnlyContract). Two
// layers, one canon, both stated.

// inlineTemplatedFragmentManifest declares the fragment as a YAML block scalar
// carrying templated JSON — the exact form the committed `starter` recipe used
// before ISSUE-081 pinned the canon.
const inlineTemplatedFragmentManifest = `
kind: scaffolding
version: 1.0.0
params:
  - name: app_name
    required: true
ops:
  - id: create-config
    kind: create
    target: config/app.settings
    payload: payload/app.settings
  - id: merge-settings
    kind: merge
    target: config/registry.json
    format: json
    fragment: |
      {"adopted_by": "{{ app_name }}"}
`

// inlineLiteralFragmentManifest carries the SAME block-scalar shape with NO
// placeholder anywhere. It proves the newline rule stands on its own rather than
// riding on the `{{` rule — without it, a check that only rejected placeholders
// would look like it had closed the inline form.
const inlineLiteralFragmentManifest = `
kind: scaffolding
version: 1.0.0
ops:
  - id: create-config
    kind: create
    target: config/app.settings
    payload: payload/app.settings
  - id: merge-settings
    kind: merge
    target: config/registry.json
    format: json
    fragment: |
      {"adopted_by": "the-literal-value"}
`

// templatedFragmentPathManifest declares a SINGLE-LINE, placeholder-bearing
// fragment path. Everything else is shaped so nothing ELSE could reject it: the
// transform op's `rule:` is literal AND listed in transform_rules, every op id is
// unique and non-empty, and the injection-limit op declares its manual. If this
// manifest is refused, the fragment check is the only thing that could have done
// it.
const templatedFragmentPathManifest = `
kind: scaffolding
version: 1.0.0
params:
  - name: variant
    default: settings
transform_rules:
  - rules/rename-key.yml
ops:
  - id: create-config
    kind: create
    target: config/app.settings
    payload: payload/app.settings
  - id: merge-settings
    kind: merge
    target: config/registry.json
    format: json
    fragment: "payload/{{ variant }}.json"
  - id: rewrite-entry
    kind: transform
    target: config/app.settings
    rule: rules/rename-key.yml
    manual: Apply the rewrite by hand.
`

// emptyFragmentManifest and whitespaceFragmentManifest are the two shapes of
// "merges nothing".
const emptyFragmentManifest = `
kind: scaffolding
version: 1.0.0
ops:
  - id: create-config
    kind: create
    target: config/app.settings
    payload: payload/app.settings
  - id: merge-settings
    kind: merge
    target: config/registry.json
    format: json
    fragment: ""
`

const whitespaceFragmentManifest = `
kind: scaffolding
version: 1.0.0
ops:
  - id: create-config
    kind: create
    target: config/app.settings
    payload: payload/app.settings
  - id: merge-settings
    kind: merge
    target: config/registry.json
    format: json
    fragment: "   "
`

// pathFormFragmentManifest is the CANON form and the companion accept case.
// Without it, every rejection test above would be satisfied by a check that
// refused every fragment outright.
const pathFormFragmentManifest = `
kind: scaffolding
version: 1.0.0
ops:
  - id: create-config
    kind: create
    target: config/app.settings
    payload: payload/app.settings
  - id: merge-settings
    kind: merge
    target: config/registry.json
    format: json
    fragment: payload/settings.fragment.json
`

// declaredFragmentPath is the exact string pathFormFragmentManifest declares, so
// the accept case can assert a byte-exact round trip rather than a containment.
const declaredFragmentPath = "payload/settings.fragment.json"

// assertLocatesTheMergeOp checks the diagnostic names the op INDEX, the op ID and
// the FIELD. A pack author who cannot tell WHICH declaration to change is exactly
// the complaint ISSUE-081 records.
func assertLocatesTheMergeOp(t *testing.T, err error) {
	t.Helper()
	msg := err.Error()
	for _, want := range []string{"ops[1]", "merge-settings", "fragment"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error must NAME %q so the pack author can locate the declaration, got: %v", want, err)
		}
	}
}

// assertStatesTheCanon checks the diagnostic tells the author what the field IS,
// not merely that it is wrong. The words are the ones ISSUE-081's Decision uses.
func assertStatesTheCanon(t *testing.T, err error) {
	t.Helper()
	msg := err.Error()
	for _, want := range []string{"recipe-directory-relative path", "read from disk"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error must state the canon (%q) so the author learns the form, got: %v", want, err)
		}
	}
}

// TestParseRecipeManifest_InlineFragmentRefused proves a merge op declaring its
// fragment as a YAML block scalar is refused at manifest validation (CLM-001),
// naming the op index, the op id and the field, and stating the path-only canon.
// The literal subtest proves the NEWLINE rule stands alone.
func TestParseRecipeManifest_InlineFragmentRefused(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		{name: "an inline block carrying a placeholder", src: inlineTemplatedFragmentManifest},
		{name: "an inline block carrying no placeholder at all", src: inlineLiteralFragmentManifest},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseRecipeManifest([]byte(tc.src))
			if err == nil {
				t.Fatal("a merge op declaring an INLINE fragment block must be a validation error; fragment is a path, not content, got nil")
			}
			assertLocatesTheMergeOp(t, err)
			assertStatesTheCanon(t, err)
			// The author needs the FIX, not just the diagnosis: move the content
			// into a file under the recipe directory and declare its path.
			if !strings.Contains(err.Error(), "inline") {
				t.Errorf("error must name the INLINE form it is refusing so the author recognizes their declaration, got: %v", err)
			}
		})
	}
}

// TestParseRecipeManifest_TemplatedFragmentPathRefused proves a single-line
// fragment path carrying a `{{` placeholder is refused at parse (CLM-002). Such a
// path is guaranteed-dead syntax: it is resolved verbatim and can only ENOENT.
func TestParseRecipeManifest_TemplatedFragmentPathRefused(t *testing.T) {
	_, err := ParseRecipeManifest([]byte(templatedFragmentPathManifest))
	if err == nil {
		t.Fatal("a merge op whose fragment PATH carries a placeholder must be a validation error; the path is never substituted, got nil")
	}
	assertLocatesTheMergeOp(t, err)

	msg := err.Error()
	if !strings.Contains(msg, placeholderOpen) {
		t.Errorf("error must quote the offending %s placeholder, got: %v", placeholderOpen, err)
	}
	// The author must not conclude templating is gone: the fragment FILE'S
	// CONTENT is still substituted. Only the PATH is verbatim.
	if !strings.Contains(msg, "never substituted") {
		t.Errorf("error must explain that a fragment PATH is never substituted, got: %v", err)
	}
	if !strings.Contains(msg, "content") {
		t.Errorf("error must say the fragment FILE'S CONTENT is still substituted, so the author does not think templating is gone, got: %v", err)
	}
	// The manifest declares its transform rule literally and lists it in
	// transform_rules, so a rule-related diagnosis would mean the fragment check
	// never ran and the test is passing for the wrong reason.
	if strings.Contains(msg, "transform_rules") {
		t.Errorf("the fragment defect was reported as a rule defect; the fragment check must be what rejects this manifest, got: %v", err)
	}
}

// TestParseRecipeManifest_EmptyFragmentOnMergeOpRefused proves an empty or
// whitespace-only fragment is refused at PARSE (CLM-003) rather than surviving to
// apply, where it surfaced as a bare "the op declares no path".
func TestParseRecipeManifest_EmptyFragmentOnMergeOpRefused(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		{name: "an empty fragment", src: emptyFragmentManifest},
		{name: "a whitespace-only fragment", src: whitespaceFragmentManifest},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseRecipeManifest([]byte(tc.src))
			if err == nil {
				t.Fatal("a merge op declaring no fragment must be a validation error; a merge cannot merge nothing, got nil")
			}
			assertLocatesTheMergeOp(t, err)
			assertStatesTheCanon(t, err)
		})
	}
}

// TestParseRecipeManifest_PathFormFragmentAccepted is the companion ACCEPT case:
// the canon form parses clean and round-trips its declared string byte-exactly.
// Without it, the three rejection tests above would be satisfied by a check that
// refused every fragment.
func TestParseRecipeManifest_PathFormFragmentAccepted(t *testing.T) {
	m, err := ParseRecipeManifest([]byte(pathFormFragmentManifest))
	if err != nil {
		t.Fatalf("a recipe-directory-relative fragment PATH is the canon form and must parse clean, got error: %v", err)
	}
	if len(m.Ops) != 2 {
		t.Fatalf("len(Ops) = %d, want 2", len(m.Ops))
	}
	merge := m.Ops[1]
	if merge.Kind != OpMerge {
		t.Fatalf("Ops[1].Kind = %q, want %q", merge.Kind, OpMerge)
	}
	if merge.Fragment != declaredFragmentPath {
		t.Errorf("Fragment = %q, want the declared path %q verbatim — the value is handed to resolveUnder unchanged", merge.Fragment, declaredFragmentPath)
	}
}
