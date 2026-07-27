package recipe

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bmanson/backstop-core/pkg/waiver"
	toml "github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

// The CLOSED merge-format allowlist (REQ-002). These four are DATA SHAPES, not
// language or tool knowledge: a codec for a universal structured-config format
// carries no assumption about which language, framework, or toolchain wrote the
// file, and the recipe still supplies every path, fragment, and format token.
const (
	mergeFormatJSON = "json"
	mergeFormatYAML = "yaml"
	mergeFormatTOML = "toml"
	mergeFormatEnv  = "env"
)

// waiverMarker is the literal prefix a waiver token starts with. It is used for
// REPORTING only — to quote back the token that accounted for a preserved
// divergence. Adjudication's own scan lives in pkg/waiver and is the only thing
// that DECIDES anything.
const waiverMarker = "@waiver:"

// ApplyMode selects where an op's WHERE comes from (REQ-003). Both modes drive the
// SAME op executors over the same recipe artifact: direct reads the recipe-declared
// targets/anchors and param defaults, while sdlc-mediated takes the injection site
// for the injection-accepting families from ApplyOptions.InjectionSites.
type ApplyMode string

// The two application modes.
const (
	ModeDirect       ApplyMode = "direct"
	ModeSDLCMediated ApplyMode = "sdlc-mediated"
)

// TransformDispatch is the transform-engine seam (REQ-006). It is INJECTED-ONLY:
// pkg/recipe does not import pkg/pack/engine and runs no trust check of its own,
// because no declared type here carries an engine tool or a locked version to check.
// The production implementation — built in cmd/backstop, the layer that can see the
// pack's engines: block and backstop.lock — routes the rewrite through the same
// engine.CheckToolAllowed gate the enforcement dispatch uses.
type TransformDispatch func(rule string, target string) error

// DivergenceVerdict is what the waiver-adjudication seam REPORTS about one
// diverged file: whether an ACTIVE waiver covers it, and the diagnostics the
// adjudication produced on the way to that answer.
//
// Diagnostics carries what adjudication REPORTED — never anything the applier
// synthesized — so the seam stays a read path. It exists because narrowing the
// answer to a bool silently DROPPED waiver.DiagnosticMalformed, which made a
// malformed token and an absent token indistinguishable at the decision point and
// let a hand-edited file be regenerated over in silence (ISSUE-080).
type DivergenceVerdict struct {
	Covered     bool
	Diagnostics []waiver.Diagnostic
}

// WaiverReader is the waiver-adjudication seam (REQ-004): given the recipe's
// declared enforcement rule and a diverged path, it reports whether a covering
// @waiver is ACTIVE and what the adjudication found. It is a READ path only — the
// applier never authors a token, because a waiver's reason and expiry are human
// judgments.
//
// It returns a VERDICT rather than a bool plus an optional reporting hook, and the
// difference matters: an optional hook is nil by default, and nil means "drop it",
// which is precisely the silent drop this widening exists to close. A reader that
// computed a verdict cannot fail to hand back what it computed. The two facts also
// come from ONE waiver.Result, so splitting them across two seams would let them
// disagree with nothing to catch it.
//
// A nil reader is not a disabled check: it selects adjudicateDivergence, this
// package's own reader over the REAL pkg/waiver read path. The seam exists so a
// caller can supply a different source, never so the mechanism can be stubbed
// out of the tests that matter.
type WaiverReader func(rule string, file string) DivergenceVerdict

// ApplyOptions carries everything the applier is allowed to know that the recipe
// does not declare. Note what is absent: no target path, no extension, no default
// location, no language or tool noun. Those come from the recipe manifest alone.
type ApplyOptions struct {
	Mode           ApplyMode
	Params         map[string]string
	InjectionSites map[string]string
	ProjectRoot    string
	Dispatch       TransformDispatch
	ReadWaivers    WaiverReader
}

// PreservedDivergence is one file the apply left in place instead of writing. Two
// cases produce one: a USER-OWNED file a create op would otherwise have clobbered
// (REQ-004's outright protection — Rule and CoveringWaiver are empty, since nothing
// was adjudicated), and recipe-owned output whose divergence was adjudicated as
// covered by an ACTIVE waiver (CoveringWaiver is the token that was READ, never one
// the applier wrote).
type PreservedDivergence struct {
	Path           string
	Rule           string
	CoveringWaiver string
}

// ApplyResult records what one apply did: the files it WROTE, the files it left in
// place, which of the writes overwrote a divergence, what adjudication reported
// non-fatally, and the thin adoption entry. A returned error yields the ZERO
// result: an apply either produces a verdict or it fails, never both.
//
// Regenerated and Diagnostics are REPORTING CHANNELS the CLI reads, never inputs to
// a decision. Regenerated is a strict SUBSET of Written recorded in the same value
// form, so a caller can mark a CLOBBER — a write that overwrote an
// unaccounted-for local divergence — without re-deriving the distinction. Before
// it existed, a clobber and a clean re-apply printed identically.
//
// Written and PreservedDivergence.Path carry the recipe-declared target AFTER param
// substitution — still the recipe's own path FORM, never the applier's absolute
// resolution of it. The substituted form is what a consumer can act on: reporting
// "{{ config_dir }}/service.json" to an operator whose file landed at
// "config/service.json" names a path that does not exist. SPEC-054's ApplyResult
// contract states the same thing; the two must not drift apart.
type ApplyResult struct {
	Written     []string
	Preserved   []PreservedDivergence
	Regenerated []string
	Diagnostics []waiver.Diagnostic
	Adoption    AdoptionEntry
}

// Apply materializes one resolved recipe into the project at opts.ProjectRoot.
//
// Ops execute in the manifest's DECLARED SLICE ORDER — never sorted, never
// map-ranged — because the declared sequence is the recipe's contract: an insert
// that depends on an earlier create, or two inserts sharing one anchor, mean
// something different in any other order.
//
// The op-family allowlist is CLOSED (REQ-002/REQ-007): create, merge, transform and
// insert execute; step is recognized and holds its sequence position but is NOT
// executed here (its executor is a later capability's); anything else fails loud
// naming the kind and the op id rather than being silently skipped.
func Apply(resolved *ResolvedRecipe, opts ApplyOptions) (ApplyResult, error) {
	if resolved == nil || resolved.Manifest == nil {
		return ApplyResult{}, errors.New("apply recipe: no resolved recipe was supplied")
	}
	if strings.TrimSpace(opts.ProjectRoot) == "" {
		return ApplyResult{}, fmt.Errorf("apply recipe %q: no project root was supplied; the applier writes only beneath a caller-supplied root", resolved.Ref)
	}

	params := effectiveParams(resolved.Manifest, opts.Params)

	adoptions, err := ReadAdoptions(adoptionRecordPath(opts.ProjectRoot))
	if err != nil {
		return ApplyResult{}, fmt.Errorf("apply recipe %q: %w", resolved.Ref, err)
	}
	_, previouslyAdopted := adoptions.Recipes[adoptionKey(resolved.Ref)]

	var result ApplyResult
	own := &ownership{
		kind:       resolved.Manifest.Kind,
		adopted:    previouslyAdopted,
		writtenNow: make(map[string]struct{}, len(resolved.Manifest.Ops)),
	}

	for index, op := range resolved.Manifest.Ops {
		var err error

		switch op.Kind {
		case OpCreate:
			err = applyCreate(resolved, op, opts, params, own, &result)
		case OpInsert:
			err = applyInsert(op, opts, params, &result)
		case OpStep:
			// RESERVED, not executed: the step keeps its sequence position and
			// contributes nothing to the result. Its payload is opaque here and is
			// deliberately not round-tripped.
		case OpMerge:
			err = applyMerge(resolved, op, opts, params, &result)
		case OpTransform:
			err = applyTransform(resolved, op, opts, params, &result)
		default:
			err = fmt.Errorf("op kind %q is outside the closed allowlist {%s, %s, %s, %s, %s}", op.Kind, OpCreate, OpMerge, OpTransform, OpInsert, OpStep)
		}

		if err != nil {
			return ApplyResult{}, opFailure(resolved, index, op, err)
		}
	}

	result.Adoption = AdoptionEntry{
		Recipe:  adoptionKey(resolved.Ref),
		Version: resolved.Ref.Version,
		Adopted: time.Now().UTC().Format(time.RFC3339),
	}
	// The record is written ONLY when this apply left recipe-owned output in
	// place. The spec pins that Apply writes the entry, not when it declines to:
	// a zero-op recipe, or one whose every target turned out to be the
	// consumer's, has adopted nothing, and recording it anyway would tell the
	// NEXT apply that a user's own file is recipe-owned and safe to regenerate
	// over. The entry is still returned on the result either way, so a caller
	// always sees what would have been recorded.
	if own.materialized {
		if err := recordAdoption(adoptions, opts.ProjectRoot, result.Adoption); err != nil {
			return ApplyResult{}, fmt.Errorf("apply recipe %q: %w", resolved.Ref, err)
		}
	}

	return result, nil
}

// ApplyAll applies several resolved recipes strictly SEQUENTIALLY in the GIVEN
// order (REQ-013). It never sorts, never dedupes by reordering, never parallelizes
// and never interleaves: co-writes to one file compose precisely BECAUSE the order
// is preserved and merge is additive, so a co-write is composition rather than a
// conflict and there is no arbitration here.
//
// The first failure stops the run and returns the results accumulated SO FAR
// alongside the error, so a caller can see how far the sequence got instead of being
// told only that it failed. A run that fails on its FIRST recipe has accumulated
// nothing and returns no results at all.
func ApplyAll(resolved []*ResolvedRecipe, opts ApplyOptions) ([]ApplyResult, error) {
	var results []ApplyResult

	for index, one := range resolved {
		result, err := Apply(one, opts)
		if err != nil {
			return results, fmt.Errorf("apply recipe %d of %d: %w", index+1, len(resolved), err)
		}
		results = append(results, result)
	}

	return results, nil
}

// applyCreate materializes the op's payload at its declared target, gated on the
// recipe's KIND — the hinge that decides whether re-applying reproduces the
// recipe's output or leaves the consumer's.
//
// Three cases, in the order they are decided:
//
//  1. The target exists and NO previous apply of this recipe adopted it: it is
//     USER-OWNED and is never clobbered by any kind (REQ-004). This is why the
//     adoption record is read rather than inferred — a file the recipe DECLARES
//     and a file the recipe PRODUCED are indistinguishable on disk, and only the
//     tracked record tells them apart.
//  2. The target is recipe-owned and the kind is TEMPLATING: one-shot. The output
//     became consumer-owned the moment it was rendered, so it is not diffed, not
//     adjudicated, and not rewritten — not even when a pack upgrade changed the
//     would-be output (REQ-012).
//  3. The target is recipe-owned and the kind is SCAFFOLDING or IMPLEMENTING:
//     regenerate-by-default with the accountable-divergence hinge — compute the
//     would-be bytes, diff, and on a divergence PRESERVE only if a covering
//     @waiver is adjudicated ACTIVE, otherwise regenerate over it.
//
// The declared target is substituted ONCE, at the top, and that value is what every
// step below uses: the path resolution, the ownership key, the reported result, and
// every diagnostic. Keying ownership on the substituted path is required rather than
// incidental — two ops templating differently can legitimately resolve to one file,
// and the raw declarations would look like two distinct targets.
func applyCreate(resolved *ResolvedRecipe, op Op, opts ApplyOptions, params map[string]string, own *ownership, result *ApplyResult) error {
	declaredTarget, err := substituteField(op.Target, "target", params)
	if err != nil {
		return err
	}

	target, err := resolveUnder(opts.ProjectRoot, declaredTarget)
	if err != nil {
		return fmt.Errorf("resolve declared target: %w", err)
	}

	_, writtenThisRun := own.writtenNow[declaredTarget]
	present, err := targetExists(declaredTarget, target)
	if err != nil {
		return err
	}

	// Whether the pending write overwrites an unaccounted-for divergence. It is
	// decided here and recorded AFTER the write succeeds, so a failed write never
	// reports a clobber that did not happen.
	var overDivergence bool

	if present && !writtenThisRun {
		outcome, err := preserveOrRegenerate(resolved, op, opts, params, own, declaredTarget, target)
		if err != nil {
			return err
		}
		// Collected on EVERY branch that produced them, including the preserving
		// one: coverage and token hygiene are independent facts, and reporting the
		// diagnostics only when the apply proceeds would be the same silent drop
		// one `if` later.
		result.Diagnostics = appendDiagnostics(result.Diagnostics, outcome.Diagnostics)
		if outcome.Final {
			if outcome.Preserved != nil {
				result.Preserved = append(result.Preserved, *outcome.Preserved)
			}
			return nil
		}
		overDivergence = outcome.OverDivergence
	}

	rendered, err := renderPayload(resolved, op, params)
	if err != nil {
		return err
	}
	if err := writeRendered(declaredTarget, target, rendered); err != nil {
		return err
	}

	own.writtenNow[declaredTarget] = struct{}{}
	own.materialized = true
	recordWritten(result, declaredTarget)
	if overDivergence {
		// The SAME value recordWritten just recorded — the SUBSTITUTED declared
		// target. Appending the raw op.Target instead would break the
		// subset-of-Written property for every templated target, silently.
		recordRegenerated(result, declaredTarget)
	}

	return nil
}

// divergenceOutcome is preserveOrRegenerate's answer. The third case — the pending
// write would overwrite a divergence NOTHING accounts for — is an explicit field
// rather than something the caller infers from "not final", because that case is
// exactly what the CLI must be able to report as a CLOBBER.
type divergenceOutcome struct {
	Preserved      *PreservedDivergence
	Final          bool                // true: the caller must not write
	OverDivergence bool                // the pending write overwrites unaccounted-for divergence
	Diagnostics    []waiver.Diagnostic // non-fatal, surfaced by the caller
}

// preserveOrRegenerate decides what happens to a target that is ALREADY on disk:
// preserve it, regenerate over it, or REFUSE.
//
// The refusal is the ISSUE-080 case. An UNCOVERED divergence whose file carries a
// token that does not PARSE fails the apply and leaves the file byte-for-byte,
// because the two alternatives both lie: regenerate-but-warn still destroys the
// operator's edit and exits 0, and preserve-silently would turn a typo into a
// permanent opt-out of regeneration. Failing is the only outcome that destroys
// nothing AND claims nothing. Waiver semantics are intact either way — a malformed
// token still does not suppress; the apply refuses rather than accepting it.
func preserveOrRegenerate(resolved *ResolvedRecipe, op Op, opts ApplyOptions, params map[string]string, own *ownership, declaredTarget string, target string) (divergenceOutcome, error) {
	if !own.adopted {
		// USER-OWNED: no apply of this recipe ever produced this file, so it is
		// the consumer's outright. Rule and CoveringWaiver stay empty — nothing
		// was adjudicated, and the applier never authors the token that would
		// account for one.
		return divergenceOutcome{Preserved: &PreservedDivergence{Path: declaredTarget}, Final: true}, nil
	}

	if own.kind == KindTemplating {
		// ONE-SHOT / consumer-owned (REQ-012). Reached without reading the file
		// or consulting the waiver seam: templating output carries no
		// regeneration obligation, so there is no divergence to account for.
		own.materialized = true
		return divergenceOutcome{Preserved: &PreservedDivergence{Path: declaredTarget}, Final: true}, nil
	}

	rendered, err := renderPayload(resolved, op, params)
	if err != nil {
		return divergenceOutcome{}, fmt.Errorf("compute the would-be regenerated output for %q: %w", declaredTarget, err)
	}
	onDisk, err := os.ReadFile(target)
	if err != nil {
		return divergenceOutcome{}, fmt.Errorf("read recipe-owned target %q: %w", declaredTarget, err)
	}
	if string(onDisk) == rendered {
		own.materialized = true
		return divergenceOutcome{Final: true}, nil
	}

	rule, covering, diagnostics, covered := coveredDivergence(resolved, opts, target)
	if covered {
		// PRESERVED, and the diagnostics still ride out. The operator accounted
		// for this divergence with a valid token; an unrelated malformed token
		// elsewhere in the file is a hygiene problem to report, not grounds to
		// revoke the accountable path.
		own.materialized = true
		return divergenceOutcome{
			Preserved:   &PreservedDivergence{Path: declaredTarget, Rule: rule, CoveringWaiver: covering},
			Final:       true,
			Diagnostics: diagnostics,
		}, nil
	}

	if malformed, found := firstMalformed(diagnostics); found {
		// The target is named AS REPORTED — the substituted declared path the
		// operator sees on disk, never the raw declaration text.
		return divergenceOutcome{}, fmt.Errorf(
			"the local divergence in %q is not covered by any active waiver, and the @waiver on line %d does not parse (%s); the applier will not regenerate over an edit it cannot adjudicate. Either correct the reason code to one the waiver grammar declares, or remove the token and re-apply to accept the regeneration",
			declaredTarget, malformed.Line, malformed.Message,
		)
	}

	return divergenceOutcome{OverDivergence: true, Diagnostics: diagnostics}, nil
}

// appendDiagnostics appends src onto dst, dropping any diagnostic already present
// under TOKEN IDENTITY — {File, Line, Message} — and preserving first-seen order
// for determinism.
//
// The dedupe is required, not cosmetic: coveredDivergence adjudicates the same file
// once per DECLARED enforcement rule, so a single malformed token yields N
// identical entries for a recipe declaring N rules. RuleID is deliberately absent
// from the key — it is the one field that legitimately DIFFERS between those N,
// because each adjudication ran against a different rule's findings.
func appendDiagnostics(dst []waiver.Diagnostic, src []waiver.Diagnostic) []waiver.Diagnostic {
	for _, candidate := range src {
		if containsDiagnostic(dst, candidate) {
			continue
		}
		dst = append(dst, candidate)
	}

	return dst
}

// containsDiagnostic reports whether a diagnostic for the same token is already
// present.
func containsDiagnostic(diagnostics []waiver.Diagnostic, candidate waiver.Diagnostic) bool {
	for _, existing := range diagnostics {
		if existing.File == candidate.File && existing.Line == candidate.Line && existing.Message == candidate.Message {
			return true
		}
	}

	return false
}

// firstMalformed returns the first diagnostic reporting a token that did not parse,
// which is what turns an uncovered divergence into a refusal.
func firstMalformed(diagnostics []waiver.Diagnostic) (waiver.Diagnostic, bool) {
	for _, diagnostic := range diagnostics {
		if diagnostic.Kind == waiver.DiagnosticMalformed {
			return diagnostic, true
		}
	}

	return waiver.Diagnostic{}, false
}

// coveredDivergence adjudicates a diverged recipe-owned file against the recipe's
// DECLARED enforcement rules and reports the rule and the covering token when one
// of them is waived, together with every diagnostic the adjudications produced.
// A recipe that declares no enforcement rule has no rule id a consumer could waive
// against, so the seam is never consulted and its divergences are always
// regenerated — a malformed token in such a recipe's output cannot block anything.
//
// EVERY declared rule is consulted, including after one has covered. Returning
// early on the first covering verdict would drop the diagnostics the later rules
// reported, which is the same silent drop the bool seam produced, just one `if`
// later. The FIRST covering rule wins; the accumulation runs throughout.
func coveredDivergence(resolved *ResolvedRecipe, opts ApplyOptions, target string) (string, string, []waiver.Diagnostic, bool) {
	read := opts.ReadWaivers
	if read == nil {
		read = adjudicateDivergence
	}

	var diagnostics []waiver.Diagnostic
	covering := ""

	for _, rule := range enforcementRules(resolved) {
		verdict := read(rule, target)
		diagnostics = appendDiagnostics(diagnostics, verdict.Diagnostics)
		if verdict.Covered && covering == "" {
			covering = rule
		}
	}

	if covering == "" {
		return "", "", diagnostics, false
	}

	return covering, coveringWaiverText(target, covering), diagnostics, true
}

// adjudicateDivergence is this package's own WaiverReader over the REAL
// pkg/waiver read path, and it implements THE LINE CONTRACT.
//
// waiver.Adjudicate associates a token with a finding through a {Line, Line-1}
// window ONLY (pkg/waiver/adjudicate.go windowLines). A divergence, unlike an
// engine finding, has no single line — and recipe-owned output includes formats
// with no comment slot at a fixed position (a JSON document cannot carry a token
// on its first line). So the divergence becomes ONE finding PER LINE of the
// diverged file, all adjudicated in a SINGLE call, and it is covered iff at least
// one of them lands in Result.Suppressed. A single fixed-line finding would only
// ever see a token on that line or the one above it.
//
// The Policy is nil (every rule waivable here): the declared non-waivable tier is
// the GATE's adjudication to make, not the applier's.
func adjudicateDivergence(rule string, file string) DivergenceVerdict {
	lines, err := rawFileLines(file)
	if err != nil {
		// Unreadable file: nothing was adjudicated, so there is nothing to report
		// and nothing is covered.
		return DivergenceVerdict{}
	}

	findings := make([]waiver.Finding, 0, len(lines))
	for index := range lines {
		findings = append(findings, waiver.Finding{RuleID: rule, File: file, Line: index + 1})
	}

	read := func(named string, line int) (string, bool) {
		if named != file || line < 1 || line > len(lines) {
			return "", false
		}
		return lines[line-1], true
	}

	// ONE call, and the WHOLE Result is held: coverage and the diagnostics are two
	// facts read off the same adjudication, and computing them separately would let
	// them disagree.
	adjudicated := waiver.Adjudicate(findings, read, nil, time.Now())

	// BOTH diagnostic kinds are carried, even though the nil Policy above makes
	// NonWaivable structurally unreachable today. Narrowing to Malformed would
	// re-create ISSUE-080 the moment a Policy is supplied here.
	diagnostics := make([]waiver.Diagnostic, 0, len(adjudicated.Malformed)+len(adjudicated.NonWaivable))
	diagnostics = append(diagnostics, adjudicated.Malformed...)
	diagnostics = append(diagnostics, adjudicated.NonWaivable...)

	return DivergenceVerdict{
		Covered:     len(adjudicated.Suppressed) > 0,
		Diagnostics: diagnostics,
	}
}

// coveringWaiverText re-reads the covering token's raw text so the result can
// report the token that ACCOUNTED for the divergence.
//
// This is REPORTING ONLY. The decision to preserve was already made by
// waiver.Adjudicate — expiry and rule-id identity included — and this scan
// changes nothing about it; an expired or wrong-rule token never reaches here.
func coveringWaiverText(file string, rule string) string {
	lines, err := rawFileLines(file)
	if err != nil {
		return ""
	}

	marker := waiverMarker + rule + ":"
	for _, line := range lines {
		at := strings.Index(line, marker)
		if at < 0 {
			continue
		}
		return strings.TrimSpace(line[at:])
	}

	return ""
}

// enforcementRules returns the recipe's DECLARED enforcement rules — the rule ids
// a consumer's divergence waiver must name. Nothing here invents one.
func enforcementRules(resolved *ResolvedRecipe) []string {
	if resolved.Manifest.Enforcement == nil {
		return nil
	}

	return resolved.Manifest.Enforcement.Rules
}

// targetExists reports whether a declared target is already on disk. An
// inspection failure other than absence is fail-loud: treating it as "absent"
// would clobber a file the applier could not read.
func targetExists(declared string, target string) (bool, error) {
	if _, err := os.Stat(target); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("inspect declared target %q: %w", declared, err)
	}

	return true, nil
}

// substituteField resolves ONE declared field against the effective params before
// the applier uses it to locate a file, match an anchor, or splice content.
//
// It is the single door every consumer-facing site/content field goes through, so a
// field added to Op later cannot quietly skip substitution, and the wrapped error
// names WHICH declaration was unresolvable rather than only quoting the placeholder.
//
// Op.Rule does NOT go through it — see applyTransform for why.
func substituteField(declared string, label string, params map[string]string) (string, error) {
	rendered, err := Substitute(declared, params)
	if err != nil {
		return "", fmt.Errorf("substitute declared %s %q: %w", label, declared, err)
	}

	return rendered, nil
}

// renderPayload reads the op's declared payload from the recipe directory and
// substitutes the effective params, producing the WOULD-BE output bytes. Both the
// fresh write and the divergence diff go through it, so what is compared is
// exactly what would be written.
func renderPayload(resolved *ResolvedRecipe, op Op, params map[string]string) (string, error) {
	payload, err := resolveUnder(resolved.Dir, op.Payload)
	if err != nil {
		return "", fmt.Errorf("resolve declared payload: %w", err)
	}
	raw, err := os.ReadFile(payload)
	if err != nil {
		return "", fmt.Errorf("read declared payload %q: %w", op.Payload, err)
	}

	rendered, err := Substitute(string(raw), params)
	if err != nil {
		return "", fmt.Errorf("substitute declared payload %q: %w", op.Payload, err)
	}

	return rendered, nil
}

// writeRendered materializes rendered bytes at a declared target, creating the
// parent directories the declared path implies.
func writeRendered(declared string, target string, rendered string) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("create parent directory for %q: %w", declared, err)
	}
	if err := os.WriteFile(target, []byte(rendered), 0o644); err != nil {
		return fmt.Errorf("write declared target %q: %w", declared, err)
	}

	return nil
}

// applyInsert splices the op's declared snippet at its declared anchor, immediately
// after the anchor's first occurrence. An absent target or an absent anchor is a
// fail-loud error: the applier never guesses a site and never falls back to
// appending at the end of the file.
func applyInsert(op Op, opts ApplyOptions, params map[string]string, result *ApplyResult) error {
	site, err := siteFor(op, opts, params)
	if err != nil {
		return err
	}
	if strings.TrimSpace(site.target) == "" {
		return missingSite(op, "target", params)
	}
	if strings.TrimSpace(site.anchor) == "" {
		return missingSite(op, "anchor", params)
	}

	// The snippet is bytes spliced into the CONSUMER's file, so it is resolved
	// before anything is read: nothing is opened until every field this op needs
	// has resolved.
	snippet, err := substituteField(op.Snippet, "snippet", params)
	if err != nil {
		return err
	}

	target, err := resolveUnder(opts.ProjectRoot, site.target)
	if err != nil {
		return fmt.Errorf("resolve insertion site: %w", err)
	}
	raw, err := os.ReadFile(target)
	if err != nil {
		return fmt.Errorf("read insertion site %q: %w", site.target, err)
	}

	content := string(raw)
	anchorAt := strings.Index(content, site.anchor)
	if anchorAt < 0 {
		return injectionLimit(site.target, op, params, fmt.Errorf("anchor %q is absent from the target", site.anchor))
	}

	spliceAt := anchorAt + len(site.anchor)
	updated := content[:spliceAt] + snippet + content[spliceAt:]
	if err := os.WriteFile(target, []byte(updated), 0o644); err != nil {
		return fmt.Errorf("write insertion site %q: %w", site.target, err)
	}

	recordWritten(result, site.target)

	return nil
}

// applyTransform hands the op's declared rule and its declared target to the injected
// dispatch, both resolved against the bases they are declared relative to — the rule
// under the PACK ROOT (a rule file is declared pack-relative, so a pack can share one
// rule across several recipes; resolving it under the recipe directory instead doubles
// the recipe segment and reaches nothing), the target under the project root. Nothing
// else about the rewrite is known here: the rule is the recipe's, the engine is the
// caller's, and the applier carries no rewrite of its own.
//
// A nil dispatch is a fail-loud CONFIGURATION error, not a no-op: silently skipping
// the rewrite would let every transform assertion pass while nothing happened, which
// is exactly the failure the injected seam exists to make impossible.
//
// There is no trust check here. No type this package declares carries an engine tool
// or a locked version to check one against, so the dispatch is the WHOLE seam, and the
// production implementation gates the engine at the layer that can see the pack's
// declared engines and the lock file.
func applyTransform(resolved *ResolvedRecipe, op Op, opts ApplyOptions, params map[string]string, result *ApplyResult) error {
	if opts.Dispatch == nil {
		return errors.New("no transform dispatch was supplied; the transform seam is injected and has no default")
	}

	site, err := siteFor(op, opts, params)
	if err != nil {
		return err
	}
	if strings.TrimSpace(site.target) == "" {
		return missingSite(op, "target", params)
	}

	// Op.Rule is used RAW, and deliberately so. It is validated at PARSE time by
	// exact string equality against the recipe's declared transform_rules
	// (manifest.go validateRecipeOps), so substituting it here would execute a rule
	// path that is not the one validation approved. And it selects which pack asset
	// an allowlisted engine runs IN PLACE over the consumer's tree — a
	// consumer-supplied param must never carry that authority. Params are inputs to
	// the recipe's output, never a selector for the recipe's own code. A placeholder
	// in `rule:` is therefore refused at manifest validation instead.
	rule, err := resolveUnder(resolved.PackDir, op.Rule)
	if err != nil {
		return fmt.Errorf("resolve declared rule: %w", err)
	}
	target, err := resolveUnder(opts.ProjectRoot, site.target)
	if err != nil {
		return fmt.Errorf("resolve rewrite site: %w", err)
	}

	if _, statErr := os.Stat(target); statErr != nil {
		return injectionLimit(site.target, op, params, fmt.Errorf("the target could not be opened: %w", statErr))
	}

	if dispatchErr := opts.Dispatch(rule, target); dispatchErr != nil {
		return injectionLimit(site.target, op, params, dispatchErr)
	}

	recordWritten(result, site.target)

	return nil
}

// injectionSite is the resolved WHERE for one injection-accepting op: the file the
// write lands in and, for an insert, the anchor it splices at.
type injectionSite struct {
	target string
	anchor string
}

// injectionSiteSeparator splits a supplied site into its target and anchor halves.
// It is core-owned OPTION grammar, carrying no knowledge about the files a recipe
// touches.
const injectionSiteSeparator = "#"

// siteFor resolves the WHERE for an injection-accepting op (REQ-003).
//
// In DIRECT mode the answer is ALWAYS the recipe declaration: opts.InjectionSites is
// not consulted at all, so a caller that supplies one anyway cannot move a write.
// In SDLC-MEDIATED mode a site keyed by the op id overrides the declaration, and the
// value may carry either half: "<target>", "<target>#<anchor>", or "#<anchor>" to
// keep the declared target and place only the anchor. An empty half falls back to
// the declaration, so a supplied site never ERASES one.
//
// Only transform and insert reach here. create, merge and step are not
// injection-accepting and never consult a site.
//
// BOTH halves are substituted, in BOTH modes, and the sdlc-mediated override is
// substituted too — it locates a write in the consumer's tree exactly as the
// declaration does, so leaving it raw would reopen the same hole through the other
// door. The separator is split off the DECLARED text BEFORE substitution: the "#"
// grammar is core-owned and applies to the declaration, so a substituted value that
// happens to contain "#" can never be re-split into halves the caller did not write.
func siteFor(op Op, opts ApplyOptions, params map[string]string) (injectionSite, error) {
	target, anchor := op.Target, op.Anchor

	if supplied, present := opts.InjectionSites[op.ID]; present && opts.Mode == ModeSDLCMediated {
		suppliedTarget, suppliedAnchor := supplied, ""
		if at := strings.Index(supplied, injectionSiteSeparator); at >= 0 {
			suppliedTarget, suppliedAnchor = supplied[:at], supplied[at+len(injectionSiteSeparator):]
		}
		if strings.TrimSpace(suppliedTarget) != "" {
			target = suppliedTarget
		}
		if strings.TrimSpace(suppliedAnchor) != "" {
			anchor = suppliedAnchor
		}
	}

	substitutedTarget, err := substituteField(target, "target", params)
	if err != nil {
		return injectionSite{}, fmt.Errorf("resolve the injection site: %w", err)
	}
	substitutedAnchor, err := substituteField(anchor, "anchor", params)
	if err != nil {
		return injectionSite{}, fmt.Errorf("resolve the injection site: %w", err)
	}

	return injectionSite{target: substitutedTarget, anchor: substitutedAnchor}, nil
}

// missingSite renders the failure for an injection-accepting op that has NEITHER a
// recipe-declared half NOR a supplied one (REQ-003). There is no fallback: the
// applier never defaults a target, never appends at the end of a file, and never
// guesses an anchor, because a guessed site writes into the consumer codebase. The
// DECLARED manual instruction is relayed VERBATIM (REQ-011), so the operator is left
// with the same actionable text an unreachable site would have produced.
func missingSite(op Op, part string, params map[string]string) error {
	return fmt.Errorf(
		"the op declares no %s and no injection site supplied one; the applier never guesses a site. Apply the declared instruction by hand: %s",
		part, relayedManual(op, params),
	)
}

// relayedManual renders the op's DECLARED instruction for the operator, and falls
// SOFT to the raw declared text if its own substitution fails.
//
// The instruction is substituted because the recipe author writes it with the same
// params as everything else — relaying "{{ app_name }}" to a human is the same
// defect as writing it to disk, only in the diagnostic channel. REQ-011's "verbatim"
// forbids the applier COMPOSING, paraphrasing or re-wrapping the instruction;
// rendering the author's own declared params is not composition.
//
// The fail-soft is the deliberate exception to fail-loud everywhere else, and the
// reason is that this text is emitted ONLY on an error path: the loud failure has
// already happened, and replacing the operator's instruction with a second error
// about the instruction would leave them with nothing to act on.
func relayedManual(op Op, params map[string]string) string {
	rendered, err := Substitute(op.Manual, params)
	if err != nil {
		return op.Manual
	}

	return rendered
}

// injectionLimit renders the failure for an op whose declared site could not be
// reached (REQ-011): the transform whose target or match is not there, and the insert
// whose declared anchor is absent.
//
// The op's DECLARED manual instruction is emitted LAST and VERBATIM — never composed,
// paraphrased, re-wrapped, or templated. An actionable "wire it in by hand like THIS"
// is language- and framework-specific knowledge the applier does not have and must not
// invent, so the recipe supplies it as data and this only relays it. The target half of
// the locator is here; the op id half comes from opFailure, which wraps every op
// failure on the way out.
func injectionLimit(target string, op Op, params map[string]string, cause error) error {
	return fmt.Errorf(
		"the declared site in target %q could not be reached (%v); apply the recipe's declared instruction by hand: %s",
		target, cause, relayedManual(op, params),
	)
}

// fragmentPathContract states the canon a fragment failure must carry, held in
// one place so the resolve and read diagnostics cannot drift apart.
//
// It exists because the bare wrapped ENOENT these paths used to emit was
// unactionable: an author who declared inline content saw "no such file or
// directory" naming a filename they never wrote, with nothing to tell them the
// field is a PATH (ISSUE-081, the live 2026-07-25 dogfood). Manifest validation
// now refuses the obvious non-path shapes, but a single-line value that is
// syntactically a filename cannot be told from inline content at parse, so it
// arrives HERE — and this is the only place left to say what the field is.
const fragmentPathContract = "'fragment' is a recipe-directory-relative path to a file read from disk, resolved under the recipe directory; inline content is not a supported form"

// applyMerge merges the op's declared fragment into its declared target.
//
// The format is the op's DECLARED format, falling back to the target's extension only
// when the recipe declares none, and it is checked against the closed allowlist FIRST
// — before the fragment is even read — so an unsupported target fails on the format
// rather than on some later symptom. There is no text-append fallback: silently
// concatenating a fragment onto an unstructured document would corrupt it while
// reporting success.
//
// Nothing is written until the whole merge has succeeded, so a decode or encode
// failure leaves the target byte-identical.
func applyMerge(resolved *ResolvedRecipe, op Op, opts ApplyOptions, params map[string]string, result *ApplyResult) error {
	declaredTarget, err := substituteField(op.Target, "target", params)
	if err != nil {
		return err
	}

	target, err := resolveUnder(opts.ProjectRoot, declaredTarget)
	if err != nil {
		return fmt.Errorf("resolve declared target: %w", err)
	}

	format, err := mergeFormatFor(op, declaredTarget)
	if err != nil {
		return err
	}

	// Both diagnostics keep their "resolve/read declared fragment" PREFIX — which
	// names the failing OPERATION, and which apply_merge_test.go pins by substring
	// — and append the CONTRACT after it. The underlying error stays wrapped: the
	// ENOENT is still useful, it just must not be the whole story.
	fragmentPath, err := resolveUnder(resolved.Dir, op.Fragment)
	if err != nil {
		return fmt.Errorf("resolve declared fragment %q: %w; %s", op.Fragment, err, fragmentPathContract)
	}
	rawFragment, err := os.ReadFile(fragmentPath)
	if err != nil {
		return fmt.Errorf("read declared fragment %q: %w; %s", op.Fragment, err, fragmentPathContract)
	}

	// THE SPLIT that makes fragment path-only coherent: the declared PATH above is
	// resolved VERBATIM and never substituted, while the file's CONTENT is
	// substituted right here — so a "{{ param }}" works inside a fragment file
	// exactly as it does in a create payload, and only the path is placeholder-free.
	fragment, err := Substitute(string(rawFragment), params)
	if err != nil {
		return fmt.Errorf("substitute declared fragment %q: %w", op.Fragment, err)
	}

	rawTarget, err := os.ReadFile(target)
	if err != nil {
		return fmt.Errorf("read declared target %q: %w", declaredTarget, err)
	}

	merged, err := mergeDocuments(format, string(rawTarget), fragment)
	if err != nil {
		return fmt.Errorf("merge declared fragment %q into %q as %s: %w", op.Fragment, declaredTarget, format, err)
	}

	if err := os.WriteFile(target, []byte(merged), 0o644); err != nil {
		return fmt.Errorf("write declared target %q: %w", declaredTarget, err)
	}

	recordWritten(result, declaredTarget)

	return nil
}

// mergeFormatFor resolves the op's merge format against the closed allowlist. An
// unsupported format is a fail-loud error naming the target and the format it could
// not handle, never a degraded write.
//
// It takes the SUBSTITUTED target because the extension fallback is read off it: a
// templated path would otherwise yield the extension of the placeholder text and so
// the wrong format verdict.
func mergeFormatFor(op Op, declaredTarget string) (string, error) {
	declared := strings.ToLower(strings.TrimSpace(op.Format))
	if declared == "" {
		declared = strings.ToLower(strings.TrimPrefix(filepath.Ext(declaredTarget), "."))
	}

	switch declared {
	case mergeFormatJSON, mergeFormatTOML, mergeFormatEnv:
		return declared, nil
	case mergeFormatYAML, "yml":
		return mergeFormatYAML, nil
	}

	return "", fmt.Errorf(
		"merge target %q has format %q, which is outside the supported set {%s, %s, %s, %s}; a merge never falls back to a text append",
		declaredTarget, declared, mergeFormatJSON, mergeFormatYAML, mergeFormatTOML, mergeFormatEnv,
	)
}

// mergeDocuments merges fragment into target under an already-validated format. The
// three structured formats share one tree merge; the key/value format is merged over
// its LINE set instead, because its documents carry no tree to decode.
func mergeDocuments(format string, target string, fragment string) (string, error) {
	if format == mergeFormatEnv {
		return mergeKeyValue(target, fragment), nil
	}

	targetTree, err := decodeStructured(format, []byte(target))
	if err != nil {
		return "", fmt.Errorf("decode target: %w", err)
	}
	fragmentTree, err := decodeStructured(format, []byte(fragment))
	if err != nil {
		return "", fmt.Errorf("decode fragment: %w", err)
	}

	encoded, err := encodeStructured(format, deepMerge(targetTree, fragmentTree))
	if err != nil {
		return "", fmt.Errorf("encode merged document: %w", err)
	}

	return string(encoded), nil
}

// decodeStructured decodes one document into a generic tree with the format's codec.
func decodeStructured(format string, data []byte) (any, error) {
	var tree any
	var err error

	switch format {
	case mergeFormatJSON:
		err = json.Unmarshal(data, &tree)
	case mergeFormatYAML:
		err = yaml.Unmarshal(data, &tree)
	default:
		// mergeFormatTOML, the remaining structured member of the closed allowlist.
		// Its codec needs a concrete table to decode into rather than a bare any.
		table := make(map[string]any)
		err = toml.Unmarshal(data, &table)
		tree = table
	}

	if err != nil {
		return nil, fmt.Errorf("decode %s document: %w", format, err)
	}

	return tree, nil
}

// encodeStructured re-encodes the merged tree with the format's codec. The document
// is rebuilt from the tree rather than patched textually, so the merged output is the
// codec's canonical rendering of the union.
func encodeStructured(format string, tree any) ([]byte, error) {
	var encoded []byte
	var err error

	switch format {
	case mergeFormatJSON:
		encoded, err = json.MarshalIndent(tree, "", "  ")
		encoded = append(encoded, '\n')
	case mergeFormatYAML:
		encoded, err = yaml.Marshal(tree)
	default:
		// mergeFormatTOML, the remaining structured member of the closed allowlist.
		encoded, err = toml.Marshal(tree)
	}

	if err != nil {
		return nil, fmt.Errorf("encode %s document: %w", format, err)
	}

	return encoded, nil
}

// deepMerge merges overlay onto base RECURSIVELY: two tables merge name by name so a
// nested addition lands inside the existing table with its siblings intact, and any
// other pairing resolves to the overlay. A shallow overwrite of a nested table — the
// most likely wrong-but-plausible implementation — would drop exactly the siblings
// this recursion preserves.
//
// A list is replaced wholesale rather than concatenated or element-merged: a list has
// no name to merge by, so any element-wise rule would be a guess about the document's
// meaning that the recipe never declared.
func deepMerge(base any, overlay any) any {
	baseTable, baseIsTable := base.(map[string]any)
	overlayTable, overlayIsTable := overlay.(map[string]any)
	if !baseIsTable || !overlayIsTable {
		return overlay
	}

	merged := make(map[string]any, len(baseTable)+len(overlayTable))
	for name, value := range baseTable {
		merged[name] = value
	}
	for name, value := range overlayTable {
		if existing, present := merged[name]; present {
			merged[name] = deepMerge(existing, value)
			continue
		}
		merged[name] = value
	}

	return merged
}

// mergeKeyValue merges a key/value fragment into a key/value target by NAME over the
// target's line set: an overriding name is rewritten IN PLACE, a new name is appended,
// and every other line — comments and blanks included — is copied through verbatim.
// Only the fragment's assignments merge; its own commentary is authoring text for the
// recipe, not content for the target.
func mergeKeyValue(target string, fragment string) string {
	fragmentNames, fragmentValues := keyValueAssignments(fragment)

	lines := rawLines(target)
	applied := make(map[string]struct{}, len(fragmentNames))
	for index, line := range lines {
		name, _, isAssignment := splitKeyValueLine(line)
		if !isAssignment {
			continue
		}
		value, overrides := fragmentValues[name]
		if !overrides {
			continue
		}
		lines[index] = name + "=" + value
		applied[name] = struct{}{}
	}

	for _, name := range fragmentNames {
		if _, done := applied[name]; done {
			continue
		}
		lines = append(lines, name+"="+fragmentValues[name])
	}

	merged := strings.Join(lines, "\n")
	if endsWithNewline(target) {
		merged += "\n"
	}

	return merged
}

// keyValueAssignments reads a key/value document's assignments in DECLARED order, so
// several appended names land in the order the fragment wrote them.
func keyValueAssignments(content string) ([]string, map[string]string) {
	lines := rawLines(content)

	names := make([]string, 0, len(lines))
	values := make(map[string]string, len(lines))
	for _, line := range lines {
		name, value, isAssignment := splitKeyValueLine(line)
		if !isAssignment {
			continue
		}
		if _, seen := values[name]; !seen {
			names = append(names, name)
		}
		values[name] = value
	}

	return names, values
}

// rawLines splits a document into its lines, dropping the empty tail a final
// newline would otherwise produce. It is the one splitter behind both the
// key/value merge's line set and the divergence scan's 1-indexed line view.
func rawLines(content string) []string {
	return strings.Split(strings.TrimSuffix(content, "\n"), "\n")
}

// rawFileLines reads a file as its raw lines, the view waiver adjudication is
// given: no parse, no comment lexer, no language identifier anywhere.
func rawFileLines(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %q: %w", path, err)
	}

	return rawLines(string(data)), nil
}

// endsWithNewline reports whether a document ended with a newline, so the merged
// document can end exactly as the target did.
func endsWithNewline(content string) bool {
	return strings.HasSuffix(content, "\n")
}

// splitKeyValueLine reports an assignment line's declared name and its OPAQUE value.
//
// The position this takes on trailing "#" text: "#" opens a comment only at the START
// of a line. On an assignment, everything after the first "=" is the value, trailing
// "#" text included. A thin executor has no dialect knowledge with which to tell a
// comment apart from a value that legitimately contains "#", so overriding a name
// replaces the WHOLE line; untouched lines are copied verbatim, so no comment is lost
// anywhere the merge did not have to write.
func splitKeyValueLine(line string) (string, string, bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return "", "", false
	}

	at := strings.Index(line, "=")
	if at < 0 {
		return "", "", false
	}

	return strings.TrimSpace(line[:at]), line[at+1:], true
}

// effectiveParams builds the substitution scope: the recipe's declared defaults,
// overridden by the caller's supplied params. A REQUIRED param with no default and
// no supplied value is deliberately left ABSENT so its placeholder fails loud in
// Substitute rather than resolving to an empty string.
func effectiveParams(manifest *RecipeManifest, supplied map[string]string) map[string]string {
	params := make(map[string]string, len(manifest.Params)+len(supplied))

	for _, spec := range manifest.Params {
		if !spec.Required || spec.Default != "" {
			params[spec.Name] = spec.Default
		}
	}
	for name, value := range supplied {
		params[name] = value
	}

	return params
}

// resolveUnder joins a recipe-DECLARED relative path onto a base directory and
// refuses anything that escapes it. The declared path is the only source of the
// location: nothing here supplies a default name, extension, or directory.
func resolveUnder(base string, declared string) (string, error) {
	if strings.TrimSpace(declared) == "" {
		return "", errors.New("the op declares no path")
	}

	absBase, err := filepath.Abs(base)
	if err != nil {
		return "", fmt.Errorf("resolve base directory %q: %w", base, err)
	}

	joined := filepath.Join(absBase, declared)
	relative, err := filepath.Rel(absBase, joined)
	if err != nil {
		return "", fmt.Errorf("resolve declared path %q: %w", declared, err)
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("declared path %q escapes %q", declared, base)
	}

	return joined, nil
}

// recordWritten appends a declared target to the result once, so a file touched by
// several ops appears a single time in first-write order.
func recordWritten(result *ApplyResult, target string) {
	for _, written := range result.Written {
		if written == target {
			return
		}
	}
	result.Written = append(result.Written, target)
}

// recordRegenerated appends a declared target to the result's Regenerated list
// once. It is only ever called alongside recordWritten and with the SAME value, so
// Regenerated stays a strict subset of Written — for a templated target too.
func recordRegenerated(result *ApplyResult, target string) {
	for _, regenerated := range result.Regenerated {
		if regenerated == target {
			return
		}
	}
	result.Regenerated = append(result.Regenerated, target)
}

// ownership is one apply's ownership state: the recipe KIND (the
// regenerate-vs-one-shot switch), whether a PREVIOUS apply of this recipe was
// recorded as adopted, the declared targets THIS run wrote, and whether the run
// left any recipe-owned output in place.
type ownership struct {
	kind         string
	adopted      bool
	writtenNow   map[string]struct{}
	materialized bool
}

// adoptionKey is a recipe's VERSION-INDEPENDENT identity in the adoption record,
// so re-applying at a new pin updates one entry instead of accumulating a row per
// version.
func adoptionKey(ref RecipeRef) string {
	return ref.Pack + ":" + ref.Recipe
}

// adoptionRecordPath locates the tracked record at the caller-supplied project
// root — the only root the applier ever writes beneath.
func adoptionRecordPath(projectRoot string) string {
	return filepath.Join(projectRoot, AdoptionRecordName)
}

// recordAdoption upserts one entry into the record already read at the start of
// this apply and writes it back. It is reached only when the apply left
// recipe-owned output in place: a run that produced none (a zero-op recipe, or
// one whose every target turned out to be the consumer's) adopts nothing, and a
// run that FAILED never reaches it at all.
func recordAdoption(adoptions *AdoptionRecord, projectRoot string, entry AdoptionEntry) error {
	if adoptions.Recipes == nil {
		adoptions.Recipes = make(map[string]AdoptionEntry, 1)
	}
	adoptions.Recipes[entry.Recipe] = entry

	if err := WriteAdoptions(adoptionRecordPath(projectRoot), adoptions); err != nil {
		return fmt.Errorf("record adoption of %q: %w", entry.Recipe, err)
	}

	return nil
}

// opFailure names the recipe, the op's position, its id, and its kind on every
// failure, so a diagnostic locates the offending declaration rather than describing
// a symptom.
func opFailure(resolved *ResolvedRecipe, index int, op Op, err error) error {
	return fmt.Errorf("apply recipe %q: ops[%d] %q (kind %q): %w", resolved.Ref, index, op.ID, op.Kind, err)
}
