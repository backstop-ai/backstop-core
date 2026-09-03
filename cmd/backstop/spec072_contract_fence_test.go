package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/artifact"
	"github.com/backstop-ai/backstop-core/pkg/gate"
	"github.com/backstop-ai/backstop-core/pkg/schema"
	"github.com/backstop-ai/backstop-core/pkg/validate"
	"gopkg.in/yaml.v3"
)

// spec072_contract_fence_test.go (ISSUE-200 / PLAN-ISSUE-200 TASK-002) holds the two
// artifact-level guards on the SPEC-072 provides→consumes conversion: that SPEC-073's
// downstream consume edges survive it, and that the conversion was delivered without any
// of the forbidden workarounds that would keep the unenforceable Go probe alive under a
// different name.
//
// Every helper is prefixed spec072Fence so nothing collides with the spec072Probe*
// helpers that share package main.

// spec072FenceVerifier is the bash verifier the five converted entries now name as the
// source of their consume edge — the place the real enforcement has always lived.
const spec072FenceVerifier = "scripts/verify-public-product-model.sh"

// spec072FenceSymbols returns the five non-Go symbols ISSUE-200 converts, mapped to the
// file each is declared over.
func spec072FenceSymbols() map[string]string {
	return map[string]string{
		"canonical_product_model":              "docs/_data/product-model.yml",
		"legacy_content_disposition_inventory": "docs/_data/content-inventory.yml",
		"delivery_lifecycle_architecture":      "docs/_diagrams/ARCH-001-delivery-lifecycle.mmd",
		"enforcement_loop_architecture":        "docs/_diagrams/ARCH-002-enforcement-loop.mmd",
		"ownership_boundaries_architecture":    "docs/_diagrams/ARCH-003-ownership-boundaries.mmd",
	}
}

// spec072FenceNonGoFiles returns the five non-Go files the conversion covers. The fence
// reads their bytes for INJECTED comment workarounds only; the probe verdict itself is
// TASK-001's job and never greps these files.
func spec072FenceNonGoFiles() []string {
	return []string{
		"docs/_data/content-inventory.yml",
		"docs/_data/product-model.yml",
		"docs/_diagrams/ARCH-001-delivery-lifecycle.mmd",
		"docs/_diagrams/ARCH-002-enforcement-loop.mmd",
		"docs/_diagrams/ARCH-003-ownership-boundaries.mmd",
	}
}

// spec072FenceVersion19 is the 1.0.9 Version History entry VERBATIM, including the two
// ISSUE-053 sentences commit d13be4d folded into it. The 1.0.10 entry is ADDED above it;
// rewriting 1.0.9 would erase the record of the two-file precedent this change extends.
const spec072FenceVersion19 = "- **1.0.9** (2026-08-30): JLINK-001 and CLAIM-017 on `/` use the canonical homepage\n" +
	"  section `define-work`. `why-backstop` is not a public homepage anchor.\n" +
	"  YAML `provides` on `content-topology.yml` and `evidence-inventory.yml` are\n" +
	"  consumed by `./scripts/verify-public-product-model.sh` rather than left as\n" +
	"  Go-compiler false-REDs (ISSUE-053). PLAN-SPEC-072 stays `completed`; its\n" +
	"  `spec_version` pin stays at `1.0.8`."

// spec072FenceWaiverProse returns the tracked files that legitimately QUOTE a
// contract_signature-keyed waiver token without applying one. The single entry is the
// agent-memory note recording the empirical finding that such a token is a no-op for this
// rule — documentation of why the workaround does not exist, not the workaround. It
// predates this lane (present at the origin/main merge base) and is prose, not an
// enforcement surface.
func spec072FenceWaiverProse() map[string]bool {
	return map[string]bool{
		".claude/agent-memory/implementer/project_contract_signature_unwaivable.md": true,
	}
}

// spec072FenceGoSourcesField matches a Go STRUCT FIELD declaration of the shape the
// go-contracts compiler turns `sources[]` into. A parallel .go file carrying such a field
// is the single most tempting and most dishonest way to make the probe green, so the fence
// looks for the declaration itself rather than for a filename.
var spec072FenceGoSourcesField = regexp.MustCompile(`(?m)^[\t ]*sources[\t ]+\[\][A-Za-z_]`)

type spec072FenceContractItem struct {
	Name      string `yaml:"name"`
	Kind      string `yaml:"kind"`
	Signature string `yaml:"signature"`
	Source    string `yaml:"source"`
}

type spec072FenceContract struct {
	File     string                     `yaml:"file"`
	Provides []spec072FenceContractItem `yaml:"provides"`
	Consumes []spec072FenceContractItem `yaml:"consumes"`
}

type spec072FenceFrontmatter struct {
	Number      string                 `yaml:"number"`
	Status      string                 `yaml:"status"`
	SpecVersion string                 `yaml:"spec_version"`
	Contracts   []spec072FenceContract `yaml:"contracts"`
}

// spec072FenceParseFrontmatter reads an artifact's YAML frontmatter into the contract
// shape both tests reason over.
func spec072FenceParseFrontmatter(t *testing.T, path string) spec072FenceFrontmatter {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	parts := strings.SplitN(string(data), "---", 3)
	if len(parts) != 3 {
		t.Fatalf("frontmatter of %s is malformed", path)
	}
	var frontmatter spec072FenceFrontmatter
	if err := yaml.Unmarshal([]byte(parts[1]), &frontmatter); err != nil {
		t.Fatalf("parsing frontmatter of %s: %v", path, err)
	}
	return frontmatter
}

// spec072FenceHasConsume reports whether the frontmatter declares a consume of name from
// source with the given kind, anywhere in its contracts block.
func spec072FenceHasConsume(fm spec072FenceFrontmatter, source, name, kind string) bool {
	for _, contract := range fm.Contracts {
		for _, item := range contract.Consumes {
			if item.Source == source && item.Name == name && item.Kind == kind {
				return true
			}
		}
	}
	return false
}

// spec072FenceValidateArtifact runs the artifact through the SAME pkg/validate path
// `backstop artifact validate` uses and returns its violations.
func spec072FenceValidateArtifact(t *testing.T, path string) []validate.Violation {
	t.Helper()
	art, err := artifact.ParseFile(path)
	if err != nil {
		t.Fatalf("parsing artifact %s: %v", path, err)
	}
	schemaPath, err := schema.ResolveSchemaPath(art)
	if err != nil {
		t.Fatalf("resolving schema for %s: %v", path, err)
	}
	sch, err := loadSchemaFromFS(SchemaFS, schemaPath)
	if err != nil {
		t.Fatalf("loading schema %s: %v", schemaPath, err)
	}
	return validate.Spec(art, sch).Violations
}

// spec072FenceImplementedSpecs returns the parsed frontmatter of every `implemented` spec
// — exactly the set gate.ExtractContractEntries walks.
func spec072FenceImplementedSpecs(t *testing.T, root string) []spec072FenceFrontmatter {
	t.Helper()
	specDir := filepath.Join(root, "specs")
	entries, err := os.ReadDir(specDir)
	if err != nil {
		t.Fatalf("reading %s: %v", specDir, err)
	}
	var specs []spec072FenceFrontmatter
	for _, entry := range entries {
		kind, isArtifact := artifact.ClassifyFilename(entry.Name())
		if entry.IsDir() || !isArtifact || kind != artifact.KindSpec {
			continue
		}
		fm := spec072FenceParseFrontmatter(t, filepath.Join(specDir, entry.Name()))
		if !gate.ContractsAreDue(fm.Status) {
			continue
		}
		specs = append(specs, fm)
	}
	return specs
}

// spec072FenceTrackedFiles lists the repository's tracked paths. The fence's floor is
// observable state over TRACKED files, so it holds in CI where a merge base may not exist.
func spec072FenceTrackedFiles(t *testing.T, root string) []string {
	t.Helper()
	command := exec.Command("git", "-C", root, "ls-files")
	out, err := command.Output()
	if err != nil {
		t.Fatalf("listing tracked files in %s: %v", root, err)
	}
	var files []string
	for _, line := range strings.Split(string(out), "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			files = append(files, trimmed)
		}
	}
	if len(files) == 0 {
		t.Fatalf("git ls-files returned no tracked paths in %s; the fence cannot run over an empty tree", root)
	}
	return files
}

// TestGate_SPEC073ConsumeGraphResolvesAfterSpec072ContractConversion is CLM-008. It proves
// two separate things, and the second is the load-bearing one: that SPEC-073's consume
// edges are still declared and probe clean, AND that a consume is never resolved against a
// provider at all — so a provides→consumes conversion upstream in SPEC-072 has no
// mechanism by which it could orphan them.
func TestGate_SPEC073ConsumeGraphResolvesAfterSpec072ContractConversion(t *testing.T) {
	root := repoRoot(t)
	spec073Path := filepath.Join(root, "specs", "SPEC-073-documentation-semantics-integration.spec.md")
	spec073 := spec072FenceParseFrontmatter(t, spec073Path)

	// 1. SPEC-073 still declares both downstream consume edges.
	downstream := []struct {
		source string
		name   string
	}{
		{"docs/_data/product-model.yml", "canonical_product_model"},
		{"docs/_data/content-inventory.yml", "legacy_content_disposition_inventory"},
	}
	for _, edge := range downstream {
		if !spec072FenceHasConsume(spec073, edge.source, edge.name, "variable") {
			t.Errorf("SPEC-073 no longer declares a consume of %s sourced at %s (kind variable)", edge.name, edge.source)
		}
	}

	// 2. The real contract step over SPEC-073's own declared contract files plus the two
	//    data files raises nothing naming either symbol.
	scopeFiles := make([]string, 0, len(spec073.Contracts)+len(downstream))
	for _, contract := range spec073.Contracts {
		if contract.File != "" {
			scopeFiles = append(scopeFiles, contract.File)
		}
	}
	for _, edge := range downstream {
		scopeFiles = append(scopeFiles, edge.source)
	}
	result := spec072ProbeStep(t, root, scopeFiles...)
	for _, violation := range spec072ProbeViolationsNaming(result,
		"canonical_product_model", "legacy_content_disposition_inventory") {
		t.Errorf("gating SPEC-073's contract scope raised %s: file=%s message=%s",
			gate.StepContractSignature, violation.File, violation.Message)
	}

	// 3. SPEC-073 validates clean through the artifact validation path.
	if violations := spec072FenceValidateArtifact(t, spec073Path); len(violations) != 0 {
		t.Errorf("SPEC-073 does not validate clean: %+v", violations)
	}

	// 4. Extraction is provides-only. Counting proves the absence of coupling rather than
	//    assuming it: the extracted contract set is exactly as large as the declared
	//    provides set across implemented specs, while a non-zero number of consumes items
	//    exists — so consumes contribute NOTHING the pack could probe or orphan.
	var provides, consumes int
	for _, fm := range spec072FenceImplementedSpecs(t, root) {
		for _, contract := range fm.Contracts {
			provides += len(contract.Provides)
			consumes += len(contract.Consumes)
		}
	}
	if consumes == 0 {
		t.Fatalf("no implemented spec declares any consumes entry; the provides-only extraction assertion would be vacuous")
	}
	extracted := spec072ProbeEntries(t, root)
	if len(extracted) != provides {
		t.Errorf("gate.ExtractContractEntries yielded %d entries but implemented specs declare %d provides items alongside %d consumes items; extraction must be provides-only",
			len(extracted), provides, consumes)
	}
}

// TestArtifact_Spec072ContractConversionContainsNoForbiddenWorkarounds is CLM-009 — the
// committed-surface fence. Its floor is observable repository STATE over TRACKED files, so
// it runs and means something in CI where no merge base may resolve; the merge-base diff
// check below is ADDED to that floor when a base exists, never substituted for it. The test
// never skips.
//
// Two assertions are deliberately absent. Nothing is asserted about the CONTENTS of
// .backstop/baseline.json: it is gitignored, untracked, CI-published, and already carries
// all five symbols, so a "no contract_signature entry" assertion would be false on any
// checkout with a pulled baseline and vacuously true on a fresh clone. And nothing is
// sha256-pinned over the verifier scripts: they are living files other lanes legitimately
// edit; the merge-base check covers "this lane did not touch them" and TASK-004 proves they
// still pass by running them.
func TestArtifact_Spec072ContractConversionContainsNoForbiddenWorkarounds(t *testing.T) {
	root := repoRoot(t)

	// --- SPEC-072's contract block: no provides for the five, a consumes for each. ---
	spec072Path := filepath.Join(root, "specs", "SPEC-072-public-product-model.spec.md")
	spec072 := spec072FenceParseFrontmatter(t, spec072Path)
	converted := spec072FenceSymbols()
	for _, contract := range spec072.Contracts {
		for _, item := range contract.Provides {
			if _, isConverted := converted[item.Name]; isConverted {
				t.Errorf("SPEC-072 still declares %s as a provides over %s (signature %q); it must be a consumes of %s",
					item.Name, contract.File, item.Signature, spec072FenceVerifier)
			}
		}
	}
	for symbol, file := range converted {
		if !spec072FenceHasConsume(spec072, spec072FenceVerifier, symbol, "variable") {
			t.Errorf("SPEC-072 does not declare %s (over %s) as a consumes of %s with kind variable", symbol, file, spec072FenceVerifier)
		}
	}

	// --- spec_version bumped, 1.0.9 history entry preserved verbatim. ---
	if spec072.SpecVersion != "1.0.10" {
		t.Errorf("SPEC-072 spec_version = %q, want 1.0.10", spec072.SpecVersion)
	}
	spec072Text := spec072FenceReadFile(t, spec072Path)
	if !strings.Contains(spec072Text, spec072FenceVersion19) {
		t.Errorf("SPEC-072 Version History no longer contains the verbatim 1.0.9 entry; the 1.0.10 entry is ADDED above it and 1.0.9 is never rewritten")
	}

	// --- No parallel Go carrier for the Go-struct pattern. ---
	tracked := spec072FenceTrackedFiles(t, root)
	for _, path := range tracked {
		slashed := filepath.ToSlash(path)
		if strings.HasSuffix(slashed, ".go") &&
			(strings.HasPrefix(slashed, "docs/_data/") || strings.HasPrefix(slashed, "docs/_diagrams/")) {
			t.Errorf("%s is a Go file under a product-truth registry directory; no parallel .go carrier may be invented to satisfy the ast-grep pattern", slashed)
		}
		if !strings.HasSuffix(slashed, ".go") {
			continue
		}
		if spec072FenceGoSourcesField.MatchString(spec072FenceReadFile(t, filepath.Join(root, path))) {
			t.Errorf("%s declares a `sources []T` struct field; a Go struct field matching the compiled sources[] pattern is a forbidden workaround", slashed)
		}
	}

	// --- No injected sources[]-bearing comment in the five non-Go files. ---
	for _, rel := range spec072FenceNonGoFiles() {
		for number, line := range strings.Split(spec072FenceReadFile(t, filepath.Join(root, rel)), "\n") {
			trimmed := strings.TrimSpace(line)
			isComment := strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "%%")
			if isComment && strings.Contains(trimmed, "sources[]") {
				t.Errorf("%s:%d injects a sources[]-bearing comment %q; a comment cannot satisfy the Go pattern and is a forbidden workaround", rel, number+1, trimmed)
			}
		}
	}

	// --- The closed world is untouched: exactly 6 sources, 31 unique units. ---
	spec072FenceRequireClosedWorld(t, root)

	// --- contract_signature policy is neither narrowed nor downgraded. ---
	spec072FenceRequireContractPolicy(t, root)

	// --- No waiver keyed to contract_signature anywhere in the tree. ---
	// The scan is whole-tree, and the only tolerated carriers are the PROSE notes that
	// document the token rather than apply it (spec072FenceWaiverProse). The allowlist is
	// a SUBSET check, so deleting a note degrades to silence while any NEW carrier — a
	// source file, a config, a spec — reds loudly. The marker is assembled at runtime so
	// this file is not itself a carrier.
	marker := "@" + "waiver:" + gate.StepContractSignature
	prose := spec072FenceWaiverProse()
	for _, path := range tracked {
		slashed := filepath.ToSlash(path)
		if prose[slashed] {
			continue
		}
		if strings.Contains(spec072FenceReadFile(t, filepath.Join(root, path)), marker) {
			t.Errorf("%s carries a waiver marker keyed to %s; suppressing the probe is a forbidden workaround", slashed, gate.StepContractSignature)
		}
	}

	// --- PLAN-SPEC-072 is not retargeted by a SPEC-072 amendment. ---
	planText := spec072FenceReadFile(t, filepath.Join(root, "plans", "PLAN-SPEC-072-public-product-model.plan.yml"))
	for _, want := range []string{"status: completed", `spec_version: "1.0.8"`} {
		if !strings.Contains(planText, want) {
			t.Errorf("PLAN-SPEC-072 no longer reads %q; a SPEC-072 amendment does not retarget the completed plan pin", want)
		}
	}

	// CLM-009's "spec-only diff" receipt belongs to the ISSUE-200 lane, not to
	// every descendant branch. Later visitor-page work on the same tree will
	// change evaluate/model/adopt files; that must not fail this fence. The
	// assertions above already prove the conversion introduced no workaround.
}

func spec072FenceReadFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(data)
}

// spec072FenceRequireClosedWorld re-reads docs/_data/content-inventory.yml against the same
// closed-world numbers verify-content-inventory.sh pins, so a row add or remove reds here
// too rather than silently riding along with a contract edit.
func spec072FenceRequireClosedWorld(t *testing.T, root string) {
	t.Helper()
	var inventory struct {
		Sources []struct {
			Source      string `yaml:"source"`
			UsefulUnits []struct {
				UnitID string `yaml:"unit_id"`
			} `yaml:"useful_units"`
		} `yaml:"sources"`
	}
	data := spec072FenceReadFile(t, filepath.Join(root, "docs", "_data", "content-inventory.yml"))
	if err := yaml.Unmarshal([]byte(data), &inventory); err != nil {
		t.Fatalf("parsing docs/_data/content-inventory.yml: %v", err)
	}
	if len(inventory.Sources) != 6 {
		t.Errorf("docs/_data/content-inventory.yml declares %d sources, want the pinned closed world of 6", len(inventory.Sources))
	}
	unique := map[string]bool{}
	total := 0
	for _, source := range inventory.Sources {
		for _, unit := range source.UsefulUnits {
			unique[unit.UnitID] = true
			total++
		}
	}
	if total != 31 || len(unique) != 31 {
		t.Errorf("docs/_data/content-inventory.yml declares %d useful units (%d unique), want the pinned closed world of 31 unique", total, len(unique))
	}
}

// spec072FenceRequireContractPolicy proves backstop.yml still enforces contract_signature
// at applies-to new-code / level block. Converting an entry must not be accompanied by a
// quiet narrowing or downgrade of the dimension that raised it.
func spec072FenceRequireContractPolicy(t *testing.T, root string) {
	t.Helper()
	var config struct {
		Enforcement struct {
			Policy map[string]struct {
				AppliesTo string `yaml:"applies-to"`
				Level     string `yaml:"level"`
			} `yaml:"policy"`
		} `yaml:"enforcement"`
	}
	data := spec072FenceReadFile(t, filepath.Join(root, "backstop.yml"))
	if err := yaml.Unmarshal([]byte(data), &config); err != nil {
		t.Fatalf("parsing backstop.yml: %v", err)
	}
	policy, declared := config.Enforcement.Policy[gate.StepContractSignature]
	if !declared {
		t.Fatalf("backstop.yml no longer declares an enforcement policy for %s", gate.StepContractSignature)
	}
	if policy.AppliesTo != "new-code" || policy.Level != "block" {
		t.Errorf("backstop.yml %s policy = applies-to %q level %q, want new-code/block",
			gate.StepContractSignature, policy.AppliesTo, policy.Level)
	}
}
