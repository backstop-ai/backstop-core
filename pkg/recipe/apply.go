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

// WaiverReader is the waiver-adjudication seam (REQ-004): given the recipe's
// declared enforcement rule and a diverged path, it reports whether a covering
// @waiver is ACTIVE. It is a READ path only — the applier never authors a token,
// because a waiver's reason and expiry are human judgments.
//
// A nil reader is not a disabled check: it selects adjudicateDivergence, this
// package's own reader over the REAL pkg/waiver read path. The seam exists so a
// caller can supply a different source, never so the mechanism can be stubbed
// out of the tests that matter.
type WaiverReader func(rule string, file string) (covered bool)

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

// ApplyResult records what one apply did: the files it WROTE (as the
// recipe-declared targets, so the result echoes the recipe's own paths rather than
// the applier's resolution of them), the files it left in place, and the thin
// adoption entry. A returned error yields the ZERO result: an apply either produces
// a verdict or it fails, never both.
type ApplyResult struct {
	Written   []string
	Preserved []PreservedDivergence
	Adoption  AdoptionEntry
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
			err = applyInsert(op, opts, &result)
		case OpStep:
			// RESERVED, not executed: the step keeps its sequence position and
			// contributes nothing to the result. Its payload is opaque here and is
			// deliberately not round-tripped.
		case OpMerge:
			err = applyMerge(resolved, op, opts, params, &result)
		case OpTransform:
			err = applyTransform(resolved, op, opts, &result)
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
func applyCreate(resolved *ResolvedRecipe, op Op, opts ApplyOptions, params map[string]string, own *ownership, result *ApplyResult) error {
	target, err := resolveUnder(opts.ProjectRoot, op.Target)
	if err != nil {
		return fmt.Errorf("resolve declared target: %w", err)
	}

	_, writtenThisRun := own.writtenNow[op.Target]
	present, err := targetExists(op.Target, target)
	if err != nil {
		return err
	}

	if present && !writtenThisRun {
		preserved, done, err := preserveOrRegenerate(resolved, op, opts, params, own, target)
		if err != nil {
			return err
		}
		if done {
			if preserved != nil {
				result.Preserved = append(result.Preserved, *preserved)
			}
			return nil
		}
	}

	rendered, err := renderPayload(resolved, op, params)
	if err != nil {
		return err
	}
	if err := writeRendered(op.Target, target, rendered); err != nil {
		return err
	}

	own.writtenNow[op.Target] = struct{}{}
	own.materialized = true
	recordWritten(result, op.Target)

	return nil
}

// preserveOrRegenerate decides what happens to a target that is ALREADY on disk.
// It reports the divergence to record (nil when there is nothing to report) and
// whether the decision is final — done=false means the caller regenerates.
func preserveOrRegenerate(resolved *ResolvedRecipe, op Op, opts ApplyOptions, params map[string]string, own *ownership, target string) (*PreservedDivergence, bool, error) {
	if !own.adopted {
		// USER-OWNED: no apply of this recipe ever produced this file, so it is
		// the consumer's outright. Rule and CoveringWaiver stay empty — nothing
		// was adjudicated, and the applier never authors the token that would
		// account for one.
		return &PreservedDivergence{Path: op.Target}, true, nil
	}

	if own.kind == KindTemplating {
		// ONE-SHOT / consumer-owned (REQ-012). Reached without reading the file
		// or consulting the waiver seam: templating output carries no
		// regeneration obligation, so there is no divergence to account for.
		own.materialized = true
		return &PreservedDivergence{Path: op.Target}, true, nil
	}

	rendered, err := renderPayload(resolved, op, params)
	if err != nil {
		return nil, false, fmt.Errorf("compute the would-be regenerated output for %q: %w", op.Target, err)
	}
	onDisk, err := os.ReadFile(target)
	if err != nil {
		return nil, false, fmt.Errorf("read recipe-owned target %q: %w", op.Target, err)
	}
	if string(onDisk) == rendered {
		own.materialized = true
		return nil, true, nil
	}

	if rule, covering, covered := coveredDivergence(resolved, opts, target); covered {
		own.materialized = true
		return &PreservedDivergence{Path: op.Target, Rule: rule, CoveringWaiver: covering}, true, nil
	}

	return nil, false, nil
}

// coveredDivergence adjudicates a diverged recipe-owned file against the recipe's
// DECLARED enforcement rules and reports the rule and the covering token when one
// of them is waived. A recipe that declares no enforcement rule has no rule id a
// consumer could waive against, so its divergences are always regenerated.
func coveredDivergence(resolved *ResolvedRecipe, opts ApplyOptions, target string) (string, string, bool) {
	for _, rule := range enforcementRules(resolved) {
		read := opts.ReadWaivers
		if read == nil {
			read = adjudicateDivergence
		}
		if !read(rule, target) {
			continue
		}
		return rule, coveringWaiverText(target, rule), true
	}

	return "", "", false
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
func adjudicateDivergence(rule string, file string) bool {
	lines, err := rawFileLines(file)
	if err != nil {
		return false
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

	return len(waiver.Adjudicate(findings, read, nil, time.Now()).Suppressed) > 0
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
func applyInsert(op Op, opts ApplyOptions, result *ApplyResult) error {
	site := siteFor(op, opts)
	if strings.TrimSpace(site.target) == "" {
		return missingSite(op, "target")
	}
	if strings.TrimSpace(site.anchor) == "" {
		return missingSite(op, "anchor")
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
		return injectionLimit(site.target, op, fmt.Errorf("anchor %q is absent from the target", site.anchor))
	}

	spliceAt := anchorAt + len(site.anchor)
	updated := content[:spliceAt] + op.Snippet + content[spliceAt:]
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
func applyTransform(resolved *ResolvedRecipe, op Op, opts ApplyOptions, result *ApplyResult) error {
	if opts.Dispatch == nil {
		return errors.New("no transform dispatch was supplied; the transform seam is injected and has no default")
	}

	site := siteFor(op, opts)
	if strings.TrimSpace(site.target) == "" {
		return missingSite(op, "target")
	}

	rule, err := resolveUnder(resolved.PackDir, op.Rule)
	if err != nil {
		return fmt.Errorf("resolve declared rule: %w", err)
	}
	target, err := resolveUnder(opts.ProjectRoot, site.target)
	if err != nil {
		return fmt.Errorf("resolve rewrite site: %w", err)
	}

	if _, statErr := os.Stat(target); statErr != nil {
		return injectionLimit(site.target, op, fmt.Errorf("the target could not be opened: %w", statErr))
	}

	if dispatchErr := opts.Dispatch(rule, target); dispatchErr != nil {
		return injectionLimit(site.target, op, dispatchErr)
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
func siteFor(op Op, opts ApplyOptions) injectionSite {
	site := injectionSite{target: op.Target, anchor: op.Anchor}
	if opts.Mode != ModeSDLCMediated {
		return site
	}

	supplied, present := opts.InjectionSites[op.ID]
	if !present {
		return site
	}

	target, anchor := supplied, ""
	if at := strings.Index(supplied, injectionSiteSeparator); at >= 0 {
		target, anchor = supplied[:at], supplied[at+len(injectionSiteSeparator):]
	}
	if strings.TrimSpace(target) != "" {
		site.target = target
	}
	if strings.TrimSpace(anchor) != "" {
		site.anchor = anchor
	}

	return site
}

// missingSite renders the failure for an injection-accepting op that has NEITHER a
// recipe-declared half NOR a supplied one (REQ-003). There is no fallback: the
// applier never defaults a target, never appends at the end of a file, and never
// guesses an anchor, because a guessed site writes into the consumer codebase. The
// DECLARED manual instruction is relayed VERBATIM (REQ-011), so the operator is left
// with the same actionable text an unreachable site would have produced.
func missingSite(op Op, part string) error {
	return fmt.Errorf(
		"the op declares no %s and no injection site supplied one; the applier never guesses a site. Apply the declared instruction by hand: %s",
		part, op.Manual,
	)
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
func injectionLimit(target string, op Op, cause error) error {
	return fmt.Errorf(
		"the declared site in target %q could not be reached (%v); apply the recipe's declared instruction by hand: %s",
		target, cause, op.Manual,
	)
}

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
	target, err := resolveUnder(opts.ProjectRoot, op.Target)
	if err != nil {
		return fmt.Errorf("resolve declared target: %w", err)
	}

	format, err := mergeFormatFor(op)
	if err != nil {
		return err
	}

	fragmentPath, err := resolveUnder(resolved.Dir, op.Fragment)
	if err != nil {
		return fmt.Errorf("resolve declared fragment: %w", err)
	}
	rawFragment, err := os.ReadFile(fragmentPath)
	if err != nil {
		return fmt.Errorf("read declared fragment %q: %w", op.Fragment, err)
	}

	// The fragment is substituted before it is decoded, so a "{{ param }}" works in a
	// fragment exactly as it does in a create payload.
	fragment, err := Substitute(string(rawFragment), params)
	if err != nil {
		return fmt.Errorf("substitute declared fragment %q: %w", op.Fragment, err)
	}

	rawTarget, err := os.ReadFile(target)
	if err != nil {
		return fmt.Errorf("read declared target %q: %w", op.Target, err)
	}

	merged, err := mergeDocuments(format, string(rawTarget), fragment)
	if err != nil {
		return fmt.Errorf("merge declared fragment %q into %q as %s: %w", op.Fragment, op.Target, format, err)
	}

	if err := os.WriteFile(target, []byte(merged), 0o644); err != nil {
		return fmt.Errorf("write declared target %q: %w", op.Target, err)
	}

	recordWritten(result, op.Target)

	return nil
}

// mergeFormatFor resolves the op's merge format against the closed allowlist. An
// unsupported format is a fail-loud error naming the target and the format it could
// not handle, never a degraded write.
func mergeFormatFor(op Op) (string, error) {
	declared := strings.ToLower(strings.TrimSpace(op.Format))
	if declared == "" {
		declared = strings.ToLower(strings.TrimPrefix(filepath.Ext(op.Target), "."))
	}

	switch declared {
	case mergeFormatJSON, mergeFormatTOML, mergeFormatEnv:
		return declared, nil
	case mergeFormatYAML, "yml":
		return mergeFormatYAML, nil
	}

	return "", fmt.Errorf(
		"merge target %q has format %q, which is outside the supported set {%s, %s, %s, %s}; a merge never falls back to a text append",
		op.Target, declared, mergeFormatJSON, mergeFormatYAML, mergeFormatTOML, mergeFormatEnv,
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
