package recipe

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

// AdoptionEntry is the thin, tracked adoption record entry {recipe ref, @version,
// adopted} (REQ-005) — deliberately carrying none of the rich per-op or per-region
// provenance that a downstream ledger owns.
type AdoptionEntry struct {
	Recipe  string
	Version string
	Adopted string
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

	var result ApplyResult
	owned := make(map[string]struct{}, len(resolved.Manifest.Ops))

	for index, op := range resolved.Manifest.Ops {
		var err error

		switch op.Kind {
		case OpCreate:
			err = applyCreate(resolved, op, opts, params, owned, &result)
		case OpInsert:
			err = applyInsert(op, opts, &result)
		case OpStep:
			// RESERVED, not executed: the step keeps its sequence position and
			// contributes nothing to the result. Its payload is opaque here and is
			// deliberately not round-tripped.
		case OpMerge:
			err = applyMerge(resolved, op, opts, params, &result)
		case OpTransform:
			err = applyTransform(op, opts)
		default:
			err = fmt.Errorf("op kind %q is outside the closed allowlist {%s, %s, %s, %s, %s}", op.Kind, OpCreate, OpMerge, OpTransform, OpInsert, OpStep)
		}

		if err != nil {
			return ApplyResult{}, opFailure(resolved, index, op, err)
		}
	}

	return result, nil
}

// ApplyAll applies several resolved recipes strictly SEQUENTIALLY in the given
// order, never reordering or interleaving them. The first failure stops the run and
// returns no partial verdict.
func ApplyAll(resolved []*ResolvedRecipe, opts ApplyOptions) ([]ApplyResult, error) {
	results := make([]ApplyResult, 0, len(resolved))

	for index, one := range resolved {
		result, err := Apply(one, opts)
		if err != nil {
			return nil, fmt.Errorf("apply recipe %d of %d: %w", index+1, len(resolved), err)
		}
		results = append(results, result)
	}

	return results, nil
}

// applyCreate materializes the op's payload at its declared target.
//
// A file already present at the target that this apply did not itself write is
// treated as USER-OWNED and is never clobbered (REQ-004): it is reported in the
// result instead, so the caller sees a protected file rather than a phantom write.
func applyCreate(resolved *ResolvedRecipe, op Op, opts ApplyOptions, params map[string]string, owned map[string]struct{}, result *ApplyResult) error {
	target, err := resolveUnder(opts.ProjectRoot, op.Target)
	if err != nil {
		return fmt.Errorf("resolve declared target: %w", err)
	}

	if _, statErr := os.Stat(target); statErr == nil {
		if _, recipeOwned := owned[op.Target]; !recipeOwned {
			result.Preserved = append(result.Preserved, PreservedDivergence{Path: op.Target})
			return nil
		}
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return fmt.Errorf("inspect declared target %q: %w", op.Target, statErr)
	}

	payload, err := resolveUnder(resolved.Dir, op.Payload)
	if err != nil {
		return fmt.Errorf("resolve declared payload: %w", err)
	}
	raw, err := os.ReadFile(payload)
	if err != nil {
		return fmt.Errorf("read declared payload %q: %w", op.Payload, err)
	}

	rendered, err := Substitute(string(raw), params)
	if err != nil {
		return fmt.Errorf("substitute declared payload %q: %w", op.Payload, err)
	}

	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("create parent directory for %q: %w", op.Target, err)
	}
	if err := os.WriteFile(target, []byte(rendered), 0o644); err != nil {
		return fmt.Errorf("write declared target %q: %w", op.Target, err)
	}

	owned[op.Target] = struct{}{}
	recordWritten(result, op.Target)

	return nil
}

// applyInsert splices the op's declared snippet at its declared anchor, immediately
// after the anchor's first occurrence. An absent target or an absent anchor is a
// fail-loud error: the applier never guesses a site and never falls back to
// appending at the end of the file.
func applyInsert(op Op, opts ApplyOptions, result *ApplyResult) error {
	if strings.TrimSpace(op.Anchor) == "" {
		return errors.New("insert op declares no anchor; the applier contributes no insertion site")
	}

	target, err := resolveUnder(opts.ProjectRoot, op.Target)
	if err != nil {
		return fmt.Errorf("resolve declared target: %w", err)
	}
	raw, err := os.ReadFile(target)
	if err != nil {
		return fmt.Errorf("read declared target %q: %w", op.Target, err)
	}

	content := string(raw)
	anchorAt := strings.Index(content, op.Anchor)
	if anchorAt < 0 {
		return fmt.Errorf("declared anchor %q is absent from target %q", op.Anchor, op.Target)
	}

	spliceAt := anchorAt + len(op.Anchor)
	updated := content[:spliceAt] + op.Snippet + content[spliceAt:]
	if err := os.WriteFile(target, []byte(updated), 0o644); err != nil {
		return fmt.Errorf("write declared target %q: %w", op.Target, err)
	}

	recordWritten(result, op.Target)

	return nil
}

// applyTransform hands the op's declared rule and target to the injected dispatch.
//
// A nil dispatch is a fail-loud CONFIGURATION error, not a no-op: silently skipping
// the rewrite would let every transform assertion pass while nothing happened, which
// is exactly the failure the injected seam exists to make impossible.
func applyTransform(op Op, opts ApplyOptions) error {
	if opts.Dispatch == nil {
		return errors.New("no transform dispatch was supplied; the transform seam is injected and has no default")
	}

	return errors.New("the transform op family is not implemented yet")
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

	lines := keyValueLines(target)
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
	lines := keyValueLines(content)

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

// keyValueLines splits a document into its lines, dropping the empty tail a final
// newline would otherwise produce.
func keyValueLines(content string) []string {
	return strings.Split(strings.TrimSuffix(content, "\n"), "\n")
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

// opFailure names the recipe, the op's position, its id, and its kind on every
// failure, so a diagnostic locates the offending declaration rather than describing
// a symptom.
func opFailure(resolved *ResolvedRecipe, index int, op Op, err error) error {
	return fmt.Errorf("apply recipe %q: ops[%d] %q (kind %q): %w", resolved.Ref, index, op.ID, op.Kind, err)
}
