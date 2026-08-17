package validate

import (
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/artifact"
)

// layout_literals_test.go pins the pkg/validate half of ISSUE-124: slug and stem
// extraction derive from the shared artifact layout table rather than from private
// extension constants, and the terminal exemption decides "is this an issue" through
// ClassifyFilename.
//
// ★ GREEN BEFORE AND AFTER THE SWAP, BY DESIGN. Like the pkg/gate pins, these do not
// falsify the duplication — the source scan in pkg/artifact does that. They falsify a
// BOTCHED swap: an implementer who discards LayoutFor's ok, or who "simplifies" the
// bundle stem strip in a way that is not behavior-preserving.
//
// Every input filename below is COMPOSED from LayoutFor and every expected output is
// written as the bare slug, so a private literal that drifts from the table goes red
// here rather than silently disagreeing with it.

// layoutTestExtension returns a kind's extension or fails the test.
func layoutTestExtension(t *testing.T, kind artifact.Kind) string {
	t.Helper()
	layout, ok := artifact.LayoutFor(kind)
	if !ok {
		t.Fatalf("artifact.LayoutFor(%q) reported ok=false; the fixture filenames cannot be composed", kind)
	}
	return layout.Extension
}

// TestValidate_SlugAndStemExtractionAreDerivedFromSharedLayout (ISSUE-124 CLM-004,
// CLM-006).
//
// extractSpecSlug, extractSlug and the bundle stem strip all produce byte-identical
// results to their pre-swap selves.
//
// ★ THE ADR ROW IS ALSO THE MAGIC-7 PIN. adr.go used to close with
// `rest[:len(rest)-7]`, where 7 is len(".adr.md") written as a bare integer — a private
// copy of the same fact in a form NO STRING SCAN CAN SEE. The source scan goes green on
// the HasSuffix line while that arithmetic keeps its own copy, which is exactly why this
// claim pins length-derivation BEHAVIORALLY as well.
func TestValidate_SlugAndStemExtractionAreDerivedFromSharedLayout(t *testing.T) {
	specExt := layoutTestExtension(t, artifact.KindSpec)
	adrExt := layoutTestExtension(t, artifact.KindADR)
	bundleExt := layoutTestExtension(t, artifact.KindBundle)

	t.Run("extractSpecSlug", func(t *testing.T) {
		cases := []struct {
			filename string
			want     string
		}{
			{"SPEC-023-my-slug" + specExt, "my-slug"},
			// A filename carrying the SPEC- prefix but NOT the spec extension keeps the
			// existing "" failure mode, which is also what ok=false must degrade to.
			{"SPEC-023-my-slug.txt", ""},
			{"short", ""},
		}
		for _, tc := range cases {
			if got := extractSpecSlug(tc.filename); got != tc.want {
				t.Errorf("extractSpecSlug(%q) = %q, want %q", tc.filename, got, tc.want)
			}
		}
	})

	t.Run("extractSlug", func(t *testing.T) {
		cases := []struct {
			filename string
			want     string
		}{
			// The length-derivation pin: a slug of a different length than the one the
			// magic 7 was written for would come back truncated or over-long if the
			// arithmetic ever disagreed with the extension it is derived from.
			{"ADR-0001-my-slug" + adrExt, "my-slug"},
			{"ADR-0002-a" + adrExt, "a"},
			{"ADR-0003-a-much-longer-slug-here" + adrExt, "a-much-longer-slug-here"},
			{"ADR-0001-my-slug.txt", ""},
			{"short", ""},
		}
		for _, tc := range cases {
			if got := extractSlug(tc.filename); got != tc.want {
				t.Errorf("extractSlug(%q) = %q, want %q", tc.filename, got, tc.want)
			}
		}
	})

	// ★ THE BUNDLE STEM ROWS CALL validateNameFilenameConsistency DIRECTLY, AND THAT IS
	// THE ONLY REACHABLE OPTION — do not "simplify" it back to driving Bundle().
	//
	// This file is `package validate`, so the unexported helper is in scope and NOTHING
	// NEW BECOMES EXPORTED, which is what the "do not export the private helper" rule is
	// actually protecting.
	//
	// The helper has EXACTLY ONE caller (bundle.go, step 5) and it is gated on
	// filenameOK — the filename must ALREADY match the schema pattern
	// `^[a-z0-9-]+(\.epic)?\.bundle\.md$` before the stem code runs at all. Two rows
	// below cannot get through that gate: "foo.epic" has no bundle extension, and
	// "BUNDLE-007-baseline..." is rejected by `[a-z0-9-]+` for its uppercase. Driving
	// those through the exported Bundle() validator would produce a red for a DIFFERENT
	// check (bundle/filename-pattern), which cannot distinguish "the stem bug fired" from
	// "an earlier check fired" — a vacuous row, and one whose mutation could not go red
	// at all. Calling the helper directly makes every row genuinely diagnostic.
	t.Run("bundle stem strip", func(t *testing.T) {
		cases := []struct {
			name     string
			filename string
			bundle   string
			why      string
		}{
			{
				name:     "bare bundle extension",
				filename: "baseline" + bundleExt,
				bundle:   "baseline",
			},
			{
				name:     "epic modifier",
				filename: "baseline.epic" + bundleExt,
				bundle:   "baseline",
				why:      "THE ORDER PIN. `.epic` sits INSIDE the stem, before the kind extension, so the epic-suffixed branch must be tested FIRST. Swap the two branches and the bare-extension branch matches first, leaving the stem as \"baseline.epic\".",
			},
			{
				name:     "numbered bundle prefix",
				filename: "BUNDLE-007-baseline" + bundleExt,
				bundle:   "baseline",
				why:      "The BUNDLE-NNN- prefix strip runs AFTER the extension strip and must be unaffected by it.",
			},
			{
				name:     "matches neither suffix",
				filename: "foo.epic",
				bundle:   "foo.epic",
				// ★ THE SHORT-CIRCUIT PIN, and the row that is otherwise untested
				// anywhere in this repo.
				//
				// HEAD's code is an if/ELSE chain, so a filename matching NEITHER suffix
				// falls through COMPLETELY UNTOUCHED. The tempting rewrite —
				//
				//     stem = strings.TrimSuffix(art.Filename, bundleLayout.Extension)
				//     stem = strings.TrimSuffix(stem, ".epic")
				//
				// — is equivalent for every input that DOES match a branch, and wrong
				// for this one: the first trim is a no-op and the second strips ".epic",
				// yielding "foo".
				//
				// ★ STATE THE REACHABILITY HONESTLY. This divergence is NOT observable
				// through the exported Bundle() validator today: the schema filename
				// pattern filters "foo.epic" out before the stem code is ever reached, so
				// the fall-through branch is DEAD in production. This row does NOT emit
				// bundle/name-filename-mismatch through Bundle(), and claiming otherwise
				// would be false.
				//
				// The row exists BECAUSE that harmlessness is an accident. It rests on a
				// real, load-bearing coupling between artifacts/bundle/v1/schema.json's
				// filename_pattern and this function's fall-through — a coupling nothing
				// in the code states and nothing enforces, which becomes a live defect
				// the moment the pattern loosens or a second, ungated caller appears.
				// That is a reason to pin it more tightly, not less.
				why: "The fall-through must leave the stem BYTE-IDENTICAL to the input. See the note above: this is a coupling to the schema's filename pattern, not a guarantee.",
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				art := &artifact.ParsedArtifact{
					Filename: tc.filename,
					Frontmatter: map[string]interface{}{
						"bundle": map[string]interface{}{
							"name": tc.bundle,
						},
					},
				}
				// An empty violation slice means the derived stem equalled bundle.name.
				if violations := validateNameFilenameConsistency(art); len(violations) != 0 {
					t.Fatalf("validateNameFilenameConsistency(%q with bundle.name %q) returned %d violation(s): %v\nThe stem did not reduce to the declared name.\n%s",
						tc.filename, tc.bundle, len(violations), violations, tc.why)
				}
			})
		}
	})
}

// TestValidate_CitingStatusClassifiesIssuesThroughSharedLayout (ISSUE-124 CLM-005).
//
// citingStatus reads the issue.* block for an ISSUE filename and top-level metadata for
// every other kind, deciding which through the shared table rather than a private suffix
// test. This is the cleanest ClassifyFilename fit in the whole change: the bool IS the
// answer, so there is no zero-value hazard to handle at all.
func TestValidate_CitingStatusClassifiesIssuesThroughSharedLayout(t *testing.T) {
	t.Run("an issue filename reads the issue block", func(t *testing.T) {
		art := &artifact.ParsedArtifact{
			Filename: "ISSUE-001-example" + layoutTestExtension(t, artifact.KindIssue),
			// Metadata carries a DIFFERENT status on purpose: reading the wrong source
			// must be visible rather than coincidentally correct.
			Metadata: map[string]string{"status": "open"},
			Frontmatter: map[string]interface{}{
				"issue": map[string]interface{}{
					"id":     "ISSUE-001",
					"status": "resolved",
				},
			},
		}
		if got := citingStatus(art); got != "resolved" {
			t.Fatalf("citingStatus for an issue filename = %q, want %q (the issue.* block, not top-level metadata)", got, "resolved")
		}
	})

	t.Run("a spec filename reads top-level metadata", func(t *testing.T) {
		art := &artifact.ParsedArtifact{
			Filename: "SPEC-001-example" + layoutTestExtension(t, artifact.KindSpec),
			Metadata: map[string]string{"status": "implemented"},
			// An issue block that must NOT be consulted for a non-issue kind.
			Frontmatter: map[string]interface{}{
				"issue": map[string]interface{}{
					"status": "resolved",
				},
			},
		}
		if got := citingStatus(art); got != "implemented" {
			t.Fatalf("citingStatus for a spec filename = %q, want %q (top-level metadata, not the issue.* block)", got, "implemented")
		}
	})

	t.Run("a non-artifact filename reads top-level metadata", func(t *testing.T) {
		art := &artifact.ParsedArtifact{
			Filename: "notes.txt",
			Metadata: map[string]string{"status": "open"},
		}
		if got := citingStatus(art); got != "open" {
			t.Fatalf("citingStatus for an unclassifiable filename = %q, want %q; an unrecognized name must fall through to metadata rather than being treated as an issue", got, "open")
		}
	})
}
