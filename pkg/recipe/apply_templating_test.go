package recipe

import "testing"

// The templating suite holds the OTHER half of the kind gate (REQ-012). A
// templating recipe is ONE-SHOT: it produces its output once and the output is
// CONSUMER-OWNED from that moment on. An applier that regenerates by default
// uniformly — the wrong-but-plausible implementation — clobbers consumer work,
// and every case below is written to catch exactly that.

// templatingRecipe declares the same shape the regenerate suite uses (a create op
// and an enforcement rule) so the ONLY difference between the two suites is the
// kind. The enforcement rule is present deliberately: a templating divergence must
// not be adjudicated against it at all.
const templatingRecipe = `
kind: templating
version: 1.0.0
enforcement:
  rules:
    - recipe.output.divergence
ops:
  - id: op-template
    kind: create
    target: generated/from-template.conf
    payload: template.txt
`

// templatedPayload is the recipe's would-be output on a first apply.
const templatedPayload = "# rendered once from the recipe's template\nalpha\nbravo\ncharlie\n"

// applyTemplatingOnce applies the templating recipe into a fresh project root and
// returns the recipe dir (so a pack upgrade can rewrite the payload), the project
// root, and the create op.
func applyTemplatingOnce(t *testing.T) (*ResolvedRecipe, string, string, Op) {
	t.Helper()

	recipeDir := t.TempDir()
	projectRoot := t.TempDir()

	resolved := resolvedFromManifest(t, recipeDir, templatingRecipe)
	if resolved.Manifest.Kind != KindTemplating {
		t.Fatalf("parsed recipe kind = %q, want %q", resolved.Manifest.Kind, KindTemplating)
	}
	createOp := resolved.Manifest.Ops[0]
	writeUnder(t, recipeDir, createOp.Payload, templatedPayload)

	result, err := Apply(resolved, ApplyOptions{Mode: ModeDirect, ProjectRoot: projectRoot})
	if err != nil {
		t.Fatalf("first Apply: unexpected error: %v", err)
	}
	if got := snapshotTree(t, projectRoot)[createOp.Target]; got != templatedPayload {
		t.Fatalf("first apply produced %q at %q, want the rendered template %q", got, createOp.Target, templatedPayload)
	}
	if len(result.Written) != 1 || result.Written[0] != createOp.Target {
		t.Fatalf("first apply result.Written = %v, want exactly [%q]", result.Written, createOp.Target)
	}
	if len(result.Preserved) != 0 {
		t.Fatalf("first apply result.Preserved = %+v, want empty — nothing existed to preserve", result.Preserved)
	}

	return resolved, recipeDir, projectRoot, createOp
}

// spyWaiverReader records every adjudication request. A templating recipe must
// make NONE: its output carries no regeneration obligation, so there is no
// divergence to account for and nothing to adjudicate.
type spyWaiverReader struct {
	calls []string
}

func (s *spyWaiverReader) read(rule string, file string) DivergenceVerdict {
	s.calls = append(s.calls, rule+" "+file)
	return DivergenceVerdict{}
}

// TestApply_Templating_OutputConsumerOwned proves a templating recipe's output is
// CONSUMER-OWNED once produced (CLM-054). The re-apply below touches an
// UNCHANGED file, and still must not diff it, adjudicate it, or rewrite it: the
// result reports it as left-in-place with no rule and no covering waiver — the
// same outright protection a user-owned file gets — and the waiver seam is never
// consulted.
func TestApply_Templating_OutputConsumerOwned(t *testing.T) {
	resolved, _, projectRoot, createOp := applyTemplatingOnce(t)

	spy := &spyWaiverReader{}
	result, err := Apply(resolved, ApplyOptions{Mode: ModeDirect, ProjectRoot: projectRoot, ReadWaivers: spy.read})
	if err != nil {
		t.Fatalf("re-apply: unexpected error: %v", err)
	}

	if len(spy.calls) != 0 {
		t.Errorf("the waiver seam was consulted %v; templating output carries no regeneration obligation to account for", spy.calls)
	}
	preserved, reported := preservedFor(result, createOp.Target)
	if !reported {
		t.Fatalf("result.Preserved = %+v, want an entry reporting the consumer-owned template output %q", result.Preserved, createOp.Target)
	}
	if preserved.Rule != "" || preserved.CoveringWaiver != "" {
		t.Errorf("preserved entry = %+v, want no rule and no covering waiver — consumer-owned output is protected outright, not by adjudication", preserved)
	}
	for _, written := range result.Written {
		if written == createOp.Target {
			t.Errorf("result.Written reports %q; a one-shot recipe writes its output once", written)
		}
	}
}

// TestApply_Templating_NotRegeneratedOnReapply proves the output survives a
// re-apply even when the recipe's WOULD-BE output has changed (CLM-055) — the
// pack-upgrade case, simulated by rewriting the recipe's payload between the two
// applies. An applier that diffed against the new would-be bytes and rewrote
// would replace consumer-owned content on every pack upgrade.
func TestApply_Templating_NotRegeneratedOnReapply(t *testing.T) {
	resolved, recipeDir, projectRoot, createOp := applyTemplatingOnce(t)

	const upgradedPayload = "# rendered by a NEWER version of the pack's template\ndelta\necho\n"
	writeUnder(t, recipeDir, createOp.Payload, upgradedPayload)

	result, err := Apply(resolved, ApplyOptions{Mode: ModeDirect, ProjectRoot: projectRoot})
	if err != nil {
		t.Fatalf("re-apply after a pack upgrade: unexpected error: %v", err)
	}

	got := snapshotTree(t, projectRoot)[createOp.Target]
	if got == upgradedPayload {
		t.Fatalf("the upgraded template overwrote consumer-owned output %q; templating is one-shot", createOp.Target)
	}
	if got != templatedPayload {
		t.Errorf("output after a pack upgrade = %q, want the originally rendered bytes %q", got, templatedPayload)
	}
	for _, written := range result.Written {
		if written == createOp.Target {
			t.Errorf("result.Written reports %q; the pack upgrade must not rewrite consumer-owned output", written)
		}
	}
}

// TestApply_Templating_ConsumerEditSurvivesReapply is the DISCRIMINATING case
// against the scaffolding/implementing model (CLM-056): the consumer edits the
// output, no waiver exists anywhere, and the edit must still be byte-identical
// after a re-apply. Under regenerate-by-default this exact input is overwritten
// (CLM-061), so an applier that applies one model uniformly fails one of the two.
func TestApply_Templating_ConsumerEditSurvivesReapply(t *testing.T) {
	resolved, _, projectRoot, createOp := applyTemplatingOnce(t)

	const consumerEdit = "# rendered once, then made the consumer's own\nalpha\nHAND-WRITTEN by the consumer\ncharlie\n"
	writeUnder(t, projectRoot, createOp.Target, consumerEdit)

	spy := &spyWaiverReader{}
	result, err := Apply(resolved, ApplyOptions{Mode: ModeDirect, ProjectRoot: projectRoot, ReadWaivers: spy.read})
	if err != nil {
		t.Fatalf("re-apply over an edited template output: unexpected error: %v", err)
	}

	tree := snapshotTree(t, projectRoot)
	if got := tree[createOp.Target]; got != consumerEdit {
		t.Errorf("edited template output = %q, want the consumer's bytes byte-identical %q", got, consumerEdit)
	}
	if len(spy.calls) != 0 {
		t.Errorf("the waiver seam was consulted %v; a consumer edit to templating output is not an accountable divergence", spy.calls)
	}
	if tokens := countWaiverTokens(tree); tokens != 0 {
		t.Errorf("project tree holds %d @waiver tokens, want 0 — a consumer edit to one-shot output needs no waiver", tokens)
	}
	preserved, reported := preservedFor(result, createOp.Target)
	if !reported {
		t.Fatalf("result.Preserved = %+v, want an entry for the untouched consumer-owned output %q", result.Preserved, createOp.Target)
	}
	if preserved.Rule != "" || preserved.CoveringWaiver != "" {
		t.Errorf("preserved entry = %+v, want no rule and no covering waiver", preserved)
	}
}
