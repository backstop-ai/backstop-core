package main

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/backstop-ai/backstop-core/pkg/artifact"
	"github.com/backstop-ai/backstop-core/pkg/gate"
	"github.com/backstop-ai/backstop-core/pkg/pack"
	"github.com/backstop-ai/backstop-core/pkg/pack/distribution"
	"gopkg.in/yaml.v3"
)

// gate_substantiveness_e2e.go is the cmd/backstop E2E harness the provisioning + REAL
// over-installed-pack tests drive (SPEC-037 REQ-009/REQ-010). It installs the
// packs/substantiveness/ SOURCE as a LOCAL pack via the REAL distribution.Add path into
// a temp workspace, then runs the PRODUCTION substantiveness gate path (the re-wired
// buildTestSubstantivenessStep → real resolveDispatchPackEngines → dispatchPackEngines →
// real ast-grep → real convert-under-sandbox → route + set-join) over a hollow backstop
// *_test.go fixture, returning the produced violations for the tests to assert. It
// resolves the pack from the INSTALLED (local) declaration — NEVER from testdata — and
// adds NO //go:embed and NO baked path. It reuses the Phase-5 wiring + Phase-2 helpers;
// it does NOT re-implement dispatch.

// substantivenessSourceDir returns the in-repo installable pack SOURCE at
// packs/substantiveness/ (location B), relative to the repo root.
func substantivenessSourceDir(repoRoot string) string {
	return filepath.Join(repoRoot, "packs", "substantiveness")
}

// e2eSpecLayout resolves a workspace's spec directory and the shared spec extension
// through the artifact layout authority (ISSUE-124), so neither builder below holds a
// private "specs" directory literal or ".spec.md" fixture-name literal.
//
// ★ THE DIRECTORY COMES FROM ResolveRoot + Root.Dir, NOT FROM A HAND-BUILT Root STRUCT
// LITERAL. Root.Path is absolute-and-cleaned by GUARANTEE of the resolver, and a literal
// skips that absolutization — which is a misuse PLAN-SPEC-068 already paid a review round
// for.
//
// Both callers return (*e2eWorkspace, error), so an unrecognized kind is reported as an
// error naming it rather than degrading to a zero-value KindLayout whose empty Extension
// would silently produce an extensionless fixture.
func e2eSpecLayout(tmp string) (specDir string, specExt string, err error) {
	root, rootErr := artifact.ResolveRoot(tmp, "")
	if rootErr != nil {
		return "", "", fmt.Errorf("resolving e2e artifact root at %s: %w", tmp, rootErr)
	}
	layout, ok := artifact.LayoutFor(artifact.KindSpec)
	if !ok {
		return "", "", fmt.Errorf("resolving e2e spec layout: artifact kind %q is unrecognized", artifact.KindSpec)
	}
	return root.Dir(artifact.KindSpec), layout.Extension, nil
}

// e2eWorkspace is a temp workspace with a backstop.yml + specs/ + a hollow mandated
// test file, into which the substantiveness pack is installed as a local pack.
type e2eWorkspace struct {
	root        string
	specDir     string
	hollowFile  string
	installed   bool
	installInfo *distribution.AddResult
}

// newE2EWorkspace scaffolds a temp workspace: a minimal backstop.yml, a spec mandating
// a hollow test, and the hollow *_test.go source. It does NOT install the pack — the
// caller decides (so the negative twin can run the SAME workspace uninstalled).
func newE2EWorkspace(tmp string) (*e2eWorkspace, error) {
	specDir, specExt, layoutErr := e2eSpecLayout(tmp)
	if layoutErr != nil {
		return nil, fmt.Errorf("scaffolding e2e workspace: %w", layoutErr)
	}
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating e2e spec dir: %w", err)
	}
	// Minimal backstop.yml (no packs yet). distribution.Add appends the
	// substantiveness pack to this when the workspace is installed. SPEC-046: no
	// `language:` key — a project is described by its declared packs.
	ymlContent := "project: e2e\npacks: {}\n"
	if err := os.WriteFile(filepath.Join(tmp, "backstop.yml"), []byte(ymlContent), 0o644); err != nil {
		return nil, fmt.Errorf("writing e2e backstop.yml: %w", err)
	}
	// A spec mandating a hollow test in pkg/gate (target package "gate").
	spec := `---
title: "E2E Sub Spec"
number: E2E-001
created: "2026-01-01"
status: implemented
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: e2e
  package: pkg/gate

verification:
  level: integration
  test_command: go test ./pkg/gate/
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: req
    supports: x:REQ-001

claims:
  - id: CLM-001
    requirement: REQ-001
    text: claim
    tests:
      - TestE2EHollowSubject
---

# E2E Sub Spec
`
	if err := os.WriteFile(filepath.Join(specDir, "e2e"+specExt), []byte(spec), 0o644); err != nil {
		return nil, fmt.Errorf("writing e2e spec: %w", err)
	}
	// A genuinely HOLLOW backstop *_test.go source: calls a subject, asserts nothing.
	hollowFile := filepath.Join(tmp, "subject_test.go")
	hollow := "package sample_test\n\nimport \"testing\"\n\n" +
		"func TestE2EHollowSubject(t *testing.T) {\n\tdoSubject()\n}\n"
	if err := os.WriteFile(hollowFile, []byte(hollow), 0o644); err != nil {
		return nil, fmt.Errorf("writing e2e hollow fixture: %w", err)
	}
	return &e2eWorkspace{root: tmp, specDir: specDir, hollowFile: hollowFile}, nil
}

// installSubstantivenessLocalPack installs the packs/substantiveness/ SOURCE as a LOCAL
// pack via the REAL pack add path (declared `backstop/substantiveness: local` in
// backstop.yml + a `local` lockfile entry) — the install path itself is REAL, not mocked.
//
// The command comes from the PRODUCTION assembly (SPEC-055 REQ-013), so pack check and
// pack test run over the source exactly as they do for a consumer. Assembling here is
// legitimate because cmd/backstop IS the assembly layer; the same construction inside
// pkg/pack/distribution would be the internal defaulting that makes a test double
// indistinguishable from production wiring.
//
// Its receiver and signature are pinned: the four call sites in
// gate_substantiveness_e2e_test.go depend on this exact shape.
func (w *e2eWorkspace) installSubstantivenessLocalPack(repoRoot string) error {
	add, err := newProductionAddCommand()
	if err != nil {
		return fmt.Errorf("assembling the pack add command: %w", err)
	}

	res, err := add.Run(substantivenessSourceDir(repoRoot), distribution.AddOptions{
		ProjectDir: w.root,
	})
	if err != nil {
		return fmt.Errorf("installing substantiveness local pack: %w", err)
	}
	w.installed = true
	w.installInfo = res
	return nil
}

// ── ISSUE-113 ZERO-MATCH FIXTURE ───────────────────────────────────────────────
// The two helpers below reproduce the bclabs-portal incident: a REAL, healthy-looking,
// installable substantiveness pack whose Q2 referenced-symbol classification matches
// ZERO test files, so the noTarget set-join is starved and the step raises one FALSE
// "does not call package X" violation per mandated test.
//
// WHY THIS FIXTURE BREAKS THE PACK'S GLOBS AND NOT THE PATH: ISSUE-112 owns the OTHER
// root cause of an empty evidence set (a missing engine tool that returns empty SARIF
// instead of failing loud) and is fixing it in its own lane. A tool-present fixture —
// ast-grep is installed, the engine runs, the scan exits 0 — stays valid and
// red-provable on BOTH sides of that lane's change. Do not rewrite it into a
// missing-tool fixture.

// newZeroMatchE2EWorkspace scaffolds the ISSUE-113 workspace: a minimal backstop.yml,
// ONE spec mandating THREE tests against target package `gate`, and THREE genuinely
// substantive *_test.go sources that really do call that package.
//
// It is a SIBLING of newE2EWorkspace, not a modification of it — four call sites in
// gate_substantiveness_e2e_test.go depend on that helper's exact current shape.
//
// The fixture sources never compile or run as Go. They are ast-grep INPUT only, which
// is why the unresolved `gate` import is fine (newE2EWorkspace's shipped fixture calls
// an undefined doSubject() for exactly the same reason).
func newZeroMatchE2EWorkspace(tmp string) (*e2eWorkspace, error) {
	specDir, specExt, layoutErr := e2eSpecLayout(tmp)
	if layoutErr != nil {
		return nil, fmt.Errorf("scaffolding zero-match e2e workspace: %w", layoutErr)
	}
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating zero-match spec dir: %w", err)
	}
	ymlContent := "project: e2e\npacks: {}\n"
	if err := os.WriteFile(filepath.Join(tmp, "backstop.yml"), []byte(ymlContent), 0o644); err != nil {
		return nil, fmt.Errorf("writing zero-match backstop.yml: %w", err)
	}
	// status `implemented` is REQUIRED: ContractsAreDue filters every other status out
	// of the Q2 join, and this workspace would silently carry ZERO due mandated tests.
	// implementation.package `pkg/gate` reduces (TargetPackageName's last-segment op) to
	// the target token `gate`.
	spec := `---
title: "E2E ZeroMatch Spec"
number: E2E-113
created: "2026-01-01"
status: implemented
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: e2e zero-match
  package: pkg/gate

verification:
  level: integration
  test_command: go test ./pkg/gate/
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: req
    supports: x:REQ-001

claims:
  - id: CLM-001
    requirement: REQ-001
    text: claim alpha
    tests:
      - TestZeroMatchAlpha
  - id: CLM-002
    requirement: REQ-001
    text: claim bravo
    tests:
      - TestZeroMatchBravo
  - id: CLM-003
    requirement: REQ-001
    text: claim charlie
    tests:
      - TestZeroMatchCharlie
---

# E2E ZeroMatch Spec
`
	if err := os.WriteFile(filepath.Join(specDir, "zeromatch"+specExt), []byte(spec), 0o644); err != nil {
		return nil, fmt.Errorf("writing zero-match spec: %w", err)
	}

	// THREE tests, one per file, at the workspace ROOT.
	//
	// ROOT PLACEMENT IS LOAD-BEARING: testFileColocatedWithTarget compares
	// filepath.Base(filepath.Dir(path)) against the target token, and a root-level file's
	// directory leaf is the temp dir name — never "gate" — so samePackage is false and
	// all three tests are genuinely JOIN-ELIGIBLE. Moving them into a `gate/`
	// subdirectory would make them same-package and silently disable the whole fixture.
	//
	// EACH BODY IS LOAD-BEARING TWICE OVER:
	//   - gate.Build() is a package-QUALIFIED call, so the pack's Q2 referenced-symbol
	//     rule extracts symbol=gate and the noTarget set-join is SATISFIED whenever the
	//     pack works. That is what makes the zero-match run's violations provably FALSE
	//     (the CLM-006 control), rather than merely asserted to be.
	//   - t.Fatalf matches the Q1 hollow rule's assertion-vocabulary regex, so these
	//     tests are NOT hollow and the routed hollow partition stays EMPTY. This is
	//     STRUCTURAL, not count hygiene: the refusal requires hollow == 0, so a single
	//     hollow test in this workspace would BLOCK the refusal and the falsifier would
	//     never fire. Removing t.Fatalf does not weaken this fixture — it disables it.
	for _, name := range []string{"Alpha", "Bravo", "Charlie"} {
		src := "package sample_test\n\nimport \"testing\"\n\n" +
			"func TestZeroMatch" + name + "(t *testing.T) {\n" +
			"\tif gate.Build() == \"\" {\n\t\tt.Fatalf(\"Build returned empty\")\n\t}\n}\n"
		file := filepath.Join(tmp, "subject_"+strings.ToLower(name)+"_test.go")
		if err := os.WriteFile(file, []byte(src), 0o644); err != nil {
			return nil, fmt.Errorf("writing zero-match fixture %s: %w", name, err)
		}
	}
	return &e2eWorkspace{root: tmp, specDir: specDir}, nil
}

// zeroMatchRuleID is the ONE spelling of the rule the zero-match harness patches. It
// addresses BOTH the rule file on disk AND the manifest entry the patched scope is
// derived from; two spellings would let the derivation silently patch a rule whose
// declared fixtures it never read.
const zeroMatchRuleID = "referenced-symbol-go"

// installZeroMatchSubstantivenessPack installs a PATCHED COPY of packs/substantiveness
// through the SAME real newProductionAddCommand() path installSubstantivenessLocalPack
// uses. The copy's Q2 referenced-symbol rule gains a `files:` restriction that matches
// nothing in the consumer's workspace — the bclabs-portal defect verbatim: a real,
// valid, installable pack whose classification globs are baked to the PACK AUTHOR's own
// repo layout rather than the consumer's.
//
// Everything else — hollow-test-go.yml, sgconfig.yml, the convert script, pack.yml — is
// left UNTOUCHED, so the engine still runs, still exits 0, and the pack still installs
// clean. The point is a HEALTHY-LOOKING pack producing zero Q2 evidence, not a broken
// install.
//
// The in-repo pack source is NEVER mutated: editing packs/substantiveness from a test
// would corrupt this repo's own dogfood install.
//
// ★ WHY THE PATCHED COPY STILL VALIDATES (ISSUE-158). `pack add` runs the FULL packval
// pipeline unconditionally on a scratch copy, and packval phase3 RUNS this rule's
// declared fixtures — it resolves them through `rule.RuleSourcePath()`, so a `rule_path:`
// declaration is no longer invisible to it (that was the hole ISSUE-092 CLOSED). A scope
// that took the pack's own negative fixture out of range would therefore make phase3
// refuse the copy, and every test driving this harness would die at install before
// reaching the code under test. So the scope is DERIVED FROM THE COPY'S OWN pack.yml —
// the exact pack-relative fixture paths that manifest declares for this rule — and the
// pack's negative fixture still triggers under packval.
//
// ★ AND IT IS ROOT-ANCHORED, WHICH IS THE WHOLE MECHANISM. ast-grep has no scan-root
// concept: it resolves a rule's `files:` globs against the INVOKING PROCESS'S WORKING
// DIRECTORY. packval runs the engine from the PACK directory (DefaultExecutor.RunEngine
// sets cmd.Dir = packDir), while the consumer gate runs it from the PROJECT directory
// (buildTestSubstantivenessStep's ExecCommandRunner Dir = projectRoot). One anchored,
// pack-relative path therefore matches the fixture in one context and NOTHING in the
// other — which is precisely the ISSUE-113 property this fixture exists to demonstrate.
//
// ★ DO NOT "SIMPLIFY" THESE PATHS INTO A `**/`-PREFIXED GLOB. A wildcard-led variant
// also passes `pack test`, so nothing in the ordinary loop catches it, but it is
// consumer-dark only by ACCIDENT of ast-grep skipping hidden directories (`.backstop/`
// is hidden) — force hidden directories into the scan and it leaks findings from the
// installed pack's own fixtures. The derivation refuses a wildcard-led path for that
// reason, and TestZeroMatchHarnessGlob_LeavesReferencedSymbolDarkEvenInHiddenDirs pins
// the difference by scanning hidden directories on purpose.
func (w *e2eWorkspace) installZeroMatchSubstantivenessPack(repoRoot string) error {
	// Beside w.root, never inside it: a pack source tree inside the workspace would be
	// swept into the engine's own scan targets.
	packCopy, err := os.MkdirTemp(filepath.Dir(w.root), "zeromatch-pack-")
	if err != nil {
		return fmt.Errorf("creating the zero-match pack copy dir: %w", err)
	}
	if err := copyPackSourceTree(substantivenessSourceDir(repoRoot), packCopy); err != nil {
		return fmt.Errorf("copying the substantiveness pack source: %w", err)
	}

	// Derived from the COPY's own manifest, and derived BEFORE anything is installed, so a
	// manifest that cannot yield an anchored scope refuses with nothing written to the
	// consumer workspace.
	scope, err := zeroMatchClassificationScope(filepath.Join(packCopy, "pack.yml"))
	if err != nil {
		return fmt.Errorf("deriving the zero-match classification scope: %w", err)
	}

	rulePath := filepath.Join(packCopy, "ast-grep", "rules", zeroMatchRuleID+".yml")
	pristine, err := os.ReadFile(rulePath)
	if err != nil {
		return fmt.Errorf("reading the referenced-symbol rule for patching: %w", err)
	}
	// The rule ships with NO top-level `files:` key, so a plain top-level append is
	// well-formed — but the result is parsed back before `pack add` sees it rather than
	// trusted.
	var patched strings.Builder
	patched.WriteString(string(pristine))
	patched.WriteString("\n# ISSUE-113 FIXTURE PATCH (mechanism corrected by ISSUE-158): classification\n" +
		"# scope anchored at the PACK directory, derived from this pack's own declared\n" +
		"# fixture paths. packval runs the engine from the pack dir, so the fixtures still\n" +
		"# match; the gate runs it from the consumer's project dir, where these paths exist\n" +
		"# nowhere — so this rule matches ZERO of the consumer's test files.\n")
	patched.WriteString("files:\n")
	for _, p := range scope {
		patched.WriteString("  - \"" + p + "\"\n")
	}
	var probe map[string]any
	if err := yaml.Unmarshal([]byte(patched.String()), &probe); err != nil {
		return fmt.Errorf("the patched %s rule is not parseable YAML: %w", zeroMatchRuleID, err)
	}
	if err := os.WriteFile(rulePath, []byte(patched.String()), 0o644); err != nil {
		return fmt.Errorf("writing the patched referenced-symbol rule: %w", err)
	}

	add, err := newProductionAddCommand()
	if err != nil {
		return fmt.Errorf("assembling the pack add command: %w", err)
	}
	res, err := add.Run(packCopy, distribution.AddOptions{ProjectDir: w.root})
	if err != nil {
		return fmt.Errorf("installing the zero-match substantiveness pack: %w", err)
	}
	w.installed = true
	w.installInfo = res
	return nil
}

// zeroMatchClassificationScope reads a pack manifest and returns the pack-relative
// fixture paths it declares for zeroMatchRuleID, in declaration order, de-duplicated.
// Those paths ARE the patched rule's `files:` scope.
//
// DERIVATION, NOT A LITERAL, IS THE POINT: a hardcoded fixture path in this harness is
// the same silent-drift class as the defect ISSUE-158 fixed — it goes stale the moment
// the pack's fixture layout moves, and a stale scope takes the pack's own fixtures out of
// its own rule's range, which makes `pack add` refuse the copy.
//
// It REFUSES rather than patching blind, because both degenerate inputs still install
// clean and still look green:
//   - no declared fixture paths at all, which would write an empty `files:` block;
//   - a path that is not cleanly pack-relative, which would silently UN-ANCHOR the scope
//     and make the fixture's consumer-darkness accidental rather than structural.
//
// Each refusal carries its own distinguishing phrase, so a caller's test can tell the
// branch apart from a manifest PARSE failure — which also names a rule id and a path.
func zeroMatchClassificationScope(manifestPath string) ([]string, error) {
	manifest, err := pack.ParseManifestFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("parsing %s for rule %q: %w", manifestPath, zeroMatchRuleID, err)
	}

	var paths []string
	seen := map[string]bool{}
	for _, rule := range manifest.Content.Ruleset.Rules {
		if rule.ID != zeroMatchRuleID {
			continue
		}
		for _, claim := range rule.Claims {
			for _, entry := range append(append([]pack.FixtureEntry{}, claim.Fixtures.Positive...), claim.Fixtures.Negative...) {
				if seen[entry.Path] {
					continue
				}
				seen[entry.Path] = true
				paths = append(paths, entry.Path)
			}
		}
	}

	if len(paths) == 0 {
		return nil, fmt.Errorf("rule %q in %s declares no fixture paths to anchor the zero-match scope on; "+
			"writing an empty files: block would leave the rule's scope at the tool's discretion rather than "+
			"pinned to this pack's own layout", zeroMatchRuleID, manifestPath)
	}
	for _, p := range paths {
		if reason := zeroMatchUnanchoredReason(p); reason != "" {
			return nil, fmt.Errorf("rule %q in %s declares fixture path %q, which is not a clean pack-relative "+
				"fixture path (%s); the zero-match scope must stay anchored at the pack directory, because that "+
				"anchoring is the only reason the same path matches under packval and matches nothing in the "+
				"consumer's tree", zeroMatchRuleID, manifestPath, p, reason)
		}
	}
	return paths, nil
}

// zeroMatchUnanchoredReason reports why a declared fixture path cannot anchor the scope,
// or "" when it can. Slash-normalized first so the check reads the same on any host.
func zeroMatchUnanchoredReason(declared string) string {
	p := filepath.ToSlash(declared)
	switch {
	case strings.TrimSpace(p) == "":
		return "it is empty"
	case strings.HasPrefix(p, "/") || filepath.IsAbs(declared):
		return "it is absolute"
	case p == "." || p == ".." || strings.HasPrefix(p, "./") || strings.HasPrefix(p, "../"):
		return "it is relative to something other than the pack root"
	case strings.Contains(p, "**"):
		return "it carries a `**` segment"
	}
	if first, _, _ := strings.Cut(p, "/"); strings.Contains(first, "*") {
		return "its first segment is a wildcard"
	}
	return ""
}

// copyPackSourceTree recursively copies a pack source tree, preserving the executable
// bit the convert script needs.
func copyPackSourceTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode().Perm())
	})
}

// runProductionSubstantivenessStep runs the PRODUCTION substantiveness gate step over
// the workspace through the REAL dispatch path: buildTestSubstantivenessStep resolves
// the pack from the INSTALLED (local) declaration via loadInstalledPacks, dispatches
// through the real resolveDispatchPackEngines (NOT the seam stub) → real ast-grep → real
// convert-under-sandbox → route + set-join. It returns the step result so the test can
// assert a real test_substantiveness violation (or its absence when uninstalled).
//
// NOTE: this deliberately does NOT install a dispatch seam stub and does NOT override
// resolveSubstantivenessPacksFn — the pack is resolved from the real installed
// declaration and dispatched through the real engine, so the proof is unstubbable.
func (w *e2eWorkspace) runProductionSubstantivenessStep() gate.StepResult {
	// The workspace is a Go project; build the pack-shaped Go classifier + matcher the
	// production substantiveness step now consumes to resolve mandated test file paths
	// (mirrors the go-toolchain pack DATA — this Go self-toolchain harness is not on
	// the language-neutral gate spine).
	classifier := gate.NewSourceClassifier([]string{"**/*.go"}, []string{"**/*_test.go", "**/testdata/**"})
	matcher, err := gate.NewTestNameMatcher([]string{`^\s*func\s+(Test\w+)\s*\(`})
	if err != nil {
		return gate.StepResult{StepName: gate.StepTestSubstantiveness, Status: "fail", Violations: []gate.Violation{{Rule: gate.StepTestSubstantiveness, Message: "compiling go test-name pattern: " + err.Error(), Severity: "error"}}}
	}
	step := buildTestSubstantivenessStep(w.specDir, w.root, w.root, nil, classifier, matcher)
	return step(context.Background())
}
