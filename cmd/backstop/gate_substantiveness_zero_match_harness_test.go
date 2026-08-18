package main

// ISSUE-158 — executable pins on the ZERO-MATCH HARNESS's contract
// ((*e2eWorkspace).installZeroMatchSubstantivenessPack, gate_substantiveness_e2e.go).
//
// ★ THE NON-OBVIOUS PART, stated here because a future reader will otherwise "tidy"
// the patched glob straight back into the defect: ast-grep has no "scan root" concept.
// It resolves a rule's `files:` globs against the INVOKING PROCESS'S WORKING DIRECTORY,
// and the two contexts this harness straddles run the engine from DIFFERENT working
// directories:
//
//	packval phase3   DefaultExecutor.RunEngine (pkg/packval/executor.go) sets
//	                 cmd.Dir = packDir and passes the PACK-RELATIVE fixture path, so
//	                 `testdata/fixtures/rules/referenced-symbol-go/negative.go` resolves.
//	consumer gate    buildTestSubstantivenessStep (gate.go) builds an ExecCommandRunner
//	                 with Dir = projectRoot, so that same pack-relative path resolves
//	                 against the CONSUMER's tree — where it exists nowhere.
//
// One ROOT-ANCHORED glob is therefore fixture-visible under packval and consumer-dark
// under the gate. That single property is what lets the harness install a REAL, VALID,
// fully-validatable pack that still produces zero Q2 evidence — which is the whole
// point of the ISSUE-113 fixture. A `**/`-prefixed glob also passes `pack test`, so
// nothing in the ordinary loop catches it, but its consumer-darkness is an ACCIDENT of
// ast-grep skipping hidden directories (`.backstop/` is hidden) rather than a
// structural property. TestZeroMatchHarnessGlob_LeavesReferencedSymbolDarkEvenInHiddenDirs
// scans hidden directories ON PURPOSE so the two are told apart.

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack"
	"github.com/backstop-ai/backstop-core/pkg/packval"
	"gopkg.in/yaml.v3"
)

// zeroMatchHarnessInstall scaffolds the ISSUE-113 workspace, installs the PATCHED pack
// copy through the harness under test, and returns the workspace. The install call is
// exactly where ISSUE-158's defect lands, so a failure here is reported verbatim.
func zeroMatchHarnessInstall(t *testing.T) *e2eWorkspace {
	t.Helper()
	ws, err := newZeroMatchE2EWorkspace(t.TempDir())
	if err != nil {
		t.Fatalf("scaffolding the zero-match workspace: %v", err)
	}
	if err := ws.installZeroMatchSubstantivenessPack(repoRoot(t)); err != nil {
		t.Fatalf("installing the zero-match (patched) local pack: %v", err)
	}
	// The install path is READ FROM THE RESULT, never hardcoded: the manifest name is
	// `backstop/substantiveness`, so `pack add` materializes at
	// <ws.root>/.backstop/packs/backstop/substantiveness — TWO segments, not the one the
	// source directory's name suggests. Asserting installInfo up front means a future
	// harness change that stops recording it fails legibly instead of nil-dereferencing.
	if ws.installInfo == nil {
		t.Fatalf("the harness reported a successful install but recorded no *distribution.AddResult; " +
			"every assertion below resolves the installed tree through installInfo.InstalledPath")
	}
	if ws.installInfo.InstalledPath == "" {
		t.Fatalf("the harness recorded an AddResult with an empty InstalledPath: %#v", ws.installInfo)
	}
	return ws
}

// TestZeroMatchHarnessPack_InstallsAndPassesAllPackvalPhases (ISSUE-158 CLM-001) — the
// patched copy must be a REAL, INSTALLABLE, FULLY-VALIDATABLE pack. `pack add` runs the
// whole packval pipeline unconditionally on a scratch copy, so a patch that takes the
// pack's own fixtures out of its own rule's scope makes phase3-fixtures refuse the copy
// and the four ISSUE-113 E2E tests die at install, before the code under test is reached.
func TestZeroMatchHarnessPack_InstallsAndPassesAllPackvalPhases(t *testing.T) {
	requireAstGrepE2E(t)

	ws := zeroMatchHarnessInstall(t)

	result := packval.NewPipeline(ws.installInfo.InstalledPath, packval.PipelineOptions{Mode: "test"}).Run()
	for _, e := range result.Errors {
		t.Logf("packval error: phase=%q check=%q rule=%q claim=%q message=%q",
			e.Phase, e.Check, e.Rule, e.Claim, e.Message)
	}
	if result.Status != "pass" {
		t.Errorf("pack test on the INSTALLED patched copy at %s: status = %q, want %q (%d errors, logged above)",
			ws.installInfo.InstalledPath, result.Status, "pass", len(result.Errors))
	}

	// phase3-fixtures is asserted SPECIFICALLY. An overall pass that came from phase3
	// being SKIPPED reads as success while proving nothing — the exact vacuous shape this
	// cluster of work exists to prevent.
	var phase3 *packval.PhaseResult
	for i := range result.Phases {
		if result.Phases[i].Phase == "phase3-fixtures" {
			phase3 = &result.Phases[i]
		}
	}
	if phase3 == nil {
		t.Fatalf("pack test on %s produced no phase3-fixtures phase result at all; phases = %+v",
			ws.installInfo.InstalledPath, result.Phases)
	}
	if phase3.Status != "pass" {
		t.Errorf("phase3-fixtures status = %q, want %q (reason=%q)", phase3.Status, "pass", phase3.Reason)
	}
}

// TestZeroMatchHarnessGlob_IsManifestDerivedAndRootAnchored (ISSUE-158 CLM-002).
//
// WHY EQUALITY WITH THE MANIFEST AND NOT A LITERAL: a hardcoded fixture path in the
// harness is the SAME silent-drift class as the defect being fixed. If the pack's
// fixture layout ever moves, a derived scope moves with it; a hardcoded one goes stale
// and re-breaks `pack add` exactly the way the fictional `harness/fixtures/**/*.go`
// glob did.
func TestZeroMatchHarnessGlob_IsManifestDerivedAndRootAnchored(t *testing.T) {
	requireAstGrepE2E(t)

	ws := zeroMatchHarnessInstall(t)

	rulePath := filepath.Join(ws.installInfo.InstalledPath, "ast-grep", "rules", "referenced-symbol-go.yml")
	data, err := os.ReadFile(rulePath)
	if err != nil {
		t.Fatalf("reading the installed patched rule %s: %v", rulePath, err)
	}
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("the patched rule %s is not parseable YAML: %v\n%s", rulePath, err, data)
	}
	raw, ok := doc["files"]
	if !ok {
		t.Fatalf("the installed patched rule %s carries no top-level `files:` key — the harness's whole "+
			"mechanism is that scope restriction:\n%s", rulePath, data)
	}
	entries, ok := raw.([]any)
	if !ok || len(entries) == 0 {
		t.Fatalf("the installed patched rule %s has a `files:` key that is not a non-empty list: %#v", rulePath, raw)
	}
	got := make(map[string]bool, len(entries))
	for _, e := range entries {
		s, ok := e.(string)
		if !ok {
			t.Fatalf("`files:` entry %#v in %s is not a string", e, rulePath)
		}
		// ROOT-ANCHORED, NEVER `**/`-PREFIXED. This is the assertion that forbids the
		// wildcard-led variant STRUCTURALLY rather than by convention: that variant also
		// passes `pack test`, and is consumer-dark only by accident of ast-grep skipping
		// hidden directories.
		if strings.Contains(s, "**") {
			t.Errorf("`files:` entry %q contains a `**` segment; the scope must be ROOT-ANCHORED so its "+
				"darkness in the consumer tree is structural, not an accident of hidden-directory skipping", s)
		}
		if first, _, _ := strings.Cut(s, "/"); strings.Contains(first, "*") {
			t.Errorf("`files:` entry %q begins with a wildcard segment %q; the scope must be anchored at the "+
				"pack directory, which is the working directory packval runs the engine from", s, first)
		}
		got[s] = true
	}

	want := zeroMatchHarnessDeclaredFixturePaths(t, filepath.Join(ws.installInfo.InstalledPath, "pack.yml"))
	if len(want) == 0 {
		t.Fatalf("the installed pack.yml declares no fixtures for rule %q — the comparison below would be vacuous",
			"referenced-symbol-go")
	}
	// Compared as SETS: declaration order is not load-bearing, equality with the manifest is.
	for p := range want {
		if !got[p] {
			t.Errorf("`files:` is missing manifest-declared fixture path %q; scope = %v", p, keysOfZeroMatchSet(got))
		}
	}
	for p := range got {
		if !want[p] {
			t.Errorf("`files:` carries entry %q that the manifest does not declare as a fixture for rule %q; "+
				"the scope must be DERIVED, not authored", p, "referenced-symbol-go")
		}
	}
}

// zeroMatchHarnessDeclaredFixturePaths reads a pack.yml through the REAL manifest reader
// and returns the set of pack-relative fixture paths declared for referenced-symbol-go.
func zeroMatchHarnessDeclaredFixturePaths(t *testing.T, manifestPath string) map[string]bool {
	t.Helper()
	m, err := pack.ParseManifestFile(manifestPath)
	if err != nil {
		t.Fatalf("parsing manifest %s: %v", manifestPath, err)
	}
	out := map[string]bool{}
	for _, rule := range m.Content.Ruleset.Rules {
		if rule.ID != "referenced-symbol-go" {
			continue
		}
		for _, claim := range rule.Claims {
			for _, fx := range claim.Fixtures.Positive {
				out[fx.Path] = true
			}
			for _, fx := range claim.Fixtures.Negative {
				out[fx.Path] = true
			}
		}
	}
	return out
}

func keysOfZeroMatchSet(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	return out
}

// zeroMatchHarnessFakeRepoRoot builds a FAKE repo root holding a COPY of
// packs/substantiveness, so the derivation's refusal branches are reachable through the
// EXISTING signature: installZeroMatchSubstantivenessPack(repoRoot) resolves its source
// through substantivenessSourceDir(repoRoot) = <repoRoot>/packs/substantiveness. No new
// exported helper is needed, and the in-repo pack source is never mutated.
func zeroMatchHarnessFakeRepoRoot(t *testing.T) string {
	t.Helper()
	fakeRoot := t.TempDir()
	dst := filepath.Join(fakeRoot, "packs", "substantiveness")
	if err := copyPackSourceTree(substantivenessSourceDir(repoRoot(t)), dst); err != nil {
		t.Fatalf("copying the pack source into the fake repo root: %v", err)
	}
	return fakeRoot
}

// TestZeroMatchHarnessDerivation_RefusesLoudlyRatherThanPatchingBlind (ISSUE-158 CLM-002).
//
// WHY BOTH BRANCHES ARE PINNED: each is cheap to write and impossible to notice once
// broken, because a silently-wrong scope still installs and still looks green. An empty
// `files:` block, or a wildcard-led one, would silently un-anchor the whole mechanism.
func TestZeroMatchHarnessDerivation_RefusesLoudlyRatherThanPatchingBlind(t *testing.T) {
	requireAstGrepE2E(t)

	// ★ EACH REFUSAL'S OWN DISTINGUISHING PHRASE, stated exactly ONCE.
	//
	// THIS ASSERTION IS LOAD-BEARING, NOT BELT-AND-BRACES. "a non-nil error naming the
	// rule id and the manifest path" is ALSO true of a manifest PARSE error, so a test
	// that stopped there would pass while exercising a completely different code path.
	// Pinning the branch's OWN message is the only thing separating "the branch refused"
	// from "something upstream refused first".
	const emptyDerivationPhrase = "declares no fixture paths to anchor the zero-match scope on"
	const wildcardPathPhrase = "is not a clean pack-relative fixture path"

	cases := []struct {
		name   string
		phrase string
		// doctor rewrites the COPY's pack.yml to reach one refusal branch.
		doctor func(t *testing.T, manifest string)
		// expect describes what the doctored manifest must still look like once parsed,
		// so a doctoring recipe that stops the manifest parsing at all is caught here
		// rather than passing vacuously downstream.
		expect func(t *testing.T, paths map[string]bool)
	}{
		{
			name:   "empty derivation",
			phrase: emptyDerivationPhrase,
			doctor: func(t *testing.T, manifest string) {
				// ★ THE WHOLE `claims:` BLOCK GOES, NOT JUST `fixtures:`. validateFixtures
				// (pkg/pack/manifest.go) hard-requires at least one positive AND one negative
				// fixture per claim, so a claim with `fixtures:` removed makes
				// ParseManifestFile itself fail — and the refusal branch is never reached,
				// while the subtest still goes green off the parse error. There is no
				// rule-level "claims must be non-empty" check, so dropping the block parses
				// clean and derives zero paths.
				zeroMatchHarnessDropClaimsBlock(t, manifest)
			},
			expect: func(t *testing.T, paths map[string]bool) {
				if len(paths) != 0 {
					t.Fatalf("the doctored manifest still declares %d fixture path(s) for referenced-symbol-go; "+
						"the empty-derivation branch would not be reached: %v", len(paths), keysOfZeroMatchSet(paths))
				}
			},
		},
		{
			name:   "wildcard-led fixture path",
			phrase: wildcardPathPhrase,
			doctor: func(t *testing.T, manifest string) {
				zeroMatchHarnessWildcardOneFixturePath(t, manifest)
			},
			expect: func(t *testing.T, paths map[string]bool) {
				if len(paths) == 0 {
					t.Fatalf("the doctored manifest declares no fixture paths at all — this subtest would reach the " +
						"EMPTY-derivation branch instead of the wildcard branch it is written to pin")
				}
				wildcards := 0
				for p := range paths {
					if strings.Contains(p, "*") {
						wildcards++
					}
				}
				if wildcards == 0 {
					t.Fatalf("the doctored manifest carries no wildcard-led fixture path; paths = %v", keysOfZeroMatchSet(paths))
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fakeRoot := zeroMatchHarnessFakeRepoRoot(t)
			manifest := filepath.Join(fakeRoot, "packs", "substantiveness", "pack.yml")
			tc.doctor(t, manifest)
			// Prove the doctored input still PARSES and still reaches the intended branch.
			tc.expect(t, zeroMatchHarnessDeclaredFixturePaths(t, manifest))

			ws, err := newZeroMatchE2EWorkspace(t.TempDir())
			if err != nil {
				t.Fatalf("scaffolding the zero-match workspace: %v", err)
			}
			installErr := ws.installZeroMatchSubstantivenessPack(fakeRoot)
			if installErr == nil {
				t.Fatalf("the harness patched a %s manifest without refusing — an unanchored or empty scope "+
					"installs clean and looks green, which is exactly the trap this pins", tc.name)
			}
			msg := installErr.Error()
			if !strings.Contains(msg, tc.phrase) {
				t.Errorf("refusal for %s must carry its OWN distinguishing phrase %q (a manifest PARSE error also "+
					"names the rule id and the path, so anything weaker pins the wrong code path); got %q",
					tc.name, tc.phrase, msg)
			}
			if !strings.Contains(msg, "referenced-symbol-go") {
				t.Errorf("refusal for %s must name the rule id; got %q", tc.name, msg)
			}
			// THE RESOLVED MANIFEST PATH IS THE COPY'S, NOT THE SOURCE'S. The harness copies
			// the pack beside w.root and derives the scope from the COPY it is about to
			// patch — deriving from the source would leave the copy free to disagree with
			// the manifest that ships with it. The copy's location is found rather than
			// assumed, which also pins the "beside w.root, never inside it" invariant.
			copies, globErr := filepath.Glob(filepath.Join(filepath.Dir(ws.root), "zeromatch-pack-*"))
			if globErr != nil || len(copies) != 1 {
				t.Fatalf("expected exactly one zeromatch-pack-* copy beside %s; got %v (err %v)", ws.root, copies, globErr)
			}
			copyManifest := filepath.Join(copies[0], "pack.yml")
			if !strings.Contains(msg, copyManifest) {
				t.Errorf("refusal for %s must name the resolved manifest path %s (the doctored source was %s); got %q",
					tc.name, copyManifest, manifest, msg)
			}

			// NOTHING may be installed: the refusal is ordered BEFORE `pack add` runs.
			if ws.installed {
				t.Errorf("the workspace is marked installed after a refusal; got installInfo = %#v", ws.installInfo)
			}
			if _, statErr := os.Stat(filepath.Join(ws.root, ".backstop", "packs")); !os.IsNotExist(statErr) {
				t.Errorf("a refusal must leave the consumer workspace untouched, but %s exists (stat err = %v)",
					filepath.Join(ws.root, ".backstop", "packs"), statErr)
			}
		})
	}
}

// zeroMatchHarnessDropClaimsBlock removes the ENTIRE `claims:` block from the
// referenced-symbol-go rule of a COPY's pack.yml, leaving a manifest that parses clean
// and declares zero fixture paths for that rule.
func zeroMatchHarnessDropClaimsBlock(t *testing.T, manifestPath string) {
	t.Helper()
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest %s: %v", manifestPath, err)
	}
	text := string(data)
	const ruleMarker = "- id: referenced-symbol-go"
	ruleAt := strings.Index(text, ruleMarker)
	if ruleAt < 0 {
		t.Fatalf("manifest %s carries no %q rule declaration to doctor", manifestPath, "referenced-symbol-go")
	}
	claimsAt := strings.Index(text[ruleAt:], "\n        claims:")
	if claimsAt < 0 {
		t.Fatalf("manifest %s: the %q rule declares no `claims:` block at the expected indent",
			manifestPath, "referenced-symbol-go")
	}
	doctored := text[:ruleAt+claimsAt] + "\n"
	if err := os.WriteFile(manifestPath, []byte(doctored), 0o644); err != nil {
		t.Fatalf("write doctored manifest %s: %v", manifestPath, err)
	}
}

// zeroMatchHarnessWildcardOneFixturePath rewrites ONE declared fixture path of a COPY's
// pack.yml to a wildcard-led value. The replacement is DOUBLE-QUOTED because a bare
// leading `*` is a YAML alias indicator and would make the manifest unparseable — which
// would reach a parse error instead of the branch under test.
func zeroMatchHarnessWildcardOneFixturePath(t *testing.T, manifestPath string) {
	t.Helper()
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest %s: %v", manifestPath, err)
	}
	const declared = "- testdata/fixtures/rules/referenced-symbol-go/negative.go"
	text := string(data)
	if !strings.Contains(text, declared) {
		t.Fatalf("manifest %s no longer declares %q — locate the fixture path by reading the manifest, not by "+
			"assuming this literal", manifestPath, declared)
	}
	doctored := strings.Replace(text, declared, `- "**/negative.go"`, 1)
	if err := os.WriteFile(manifestPath, []byte(doctored), 0o644); err != nil {
		t.Fatalf("write doctored manifest %s: %v", manifestPath, err)
	}
}

// TestZeroMatchScope_AnchoringPredicateRejectsUnanchoredPaths (ISSUE-158 CLM-002) tables
// the predicate the derivation refuses on. The E2E refusal test above drives ONE of these
// shapes through the whole harness — which is the expensive, load-bearing proof — but the
// predicate has several arms and each of them is a way the scope could silently come
// un-anchored, so they are pinned directly and cheaply here.
func TestZeroMatchScope_AnchoringPredicateRejectsUnanchoredPaths(t *testing.T) {
	cases := []struct {
		path     string
		anchored bool
	}{
		{"testdata/fixtures/rules/referenced-symbol-go/negative.go", true},
		{"testdata/a.go", true},
		{"", false},
		{"   ", false},
		{"/abs/testdata/a.go", false},
		{"./testdata/a.go", false},
		{"../testdata/a.go", false},
		{".", false},
		{"..", false},
		// `**` ANYWHERE, not merely leading: a mid-path `**` still walks out of the pack's
		// own layout into whatever the invoking working directory happens to contain.
		{"**/negative.go", false},
		{"testdata/**/negative.go", false},
		{"*/negative.go", false},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			reason := zeroMatchUnanchoredReason(tc.path)
			if tc.anchored && reason != "" {
				t.Errorf("path %q must be usable as an anchored scope entry; refused with %q", tc.path, reason)
			}
			if !tc.anchored && reason == "" {
				t.Errorf("path %q must be refused as unanchored, but the predicate accepted it", tc.path)
			}
		})
	}
}

// TestZeroMatchScope_DerivationReadsTheManifest (ISSUE-158 CLM-002) pins the two
// remaining behaviours of the derivation that the harness-level tests cannot reach: it
// reports an unreadable manifest rather than silently deriving nothing, and it
// de-duplicates a path a manifest declares twice rather than emitting it twice.
func TestZeroMatchScope_DerivationReadsTheManifest(t *testing.T) {
	t.Run("unreadable manifest", func(t *testing.T) {
		missing := filepath.Join(t.TempDir(), "pack.yml")
		_, err := zeroMatchClassificationScope(missing)
		if err == nil {
			t.Fatalf("deriving from a nonexistent manifest %s must fail rather than yield an empty scope", missing)
		}
		if !strings.Contains(err.Error(), missing) {
			t.Errorf("the parse failure must name the manifest it could not read; got %q", err)
		}
	})

	t.Run("duplicate declared path", func(t *testing.T) {
		// The pack's negative fixture is re-declared as the positive one, so the rule
		// declares the SAME path twice. Emitting it twice would be harmless YAML and
		// useless noise; the derivation collapses it.
		dir := t.TempDir()
		src := filepath.Join(substantivenessSourceDir(repoRoot(t)), "pack.yml")
		data, err := os.ReadFile(src)
		if err != nil {
			t.Fatalf("read %s: %v", src, err)
		}
		doctored := strings.Replace(string(data),
			"- testdata/fixtures/rules/referenced-symbol-go/positive.go",
			"- testdata/fixtures/rules/referenced-symbol-go/negative.go", 1)
		manifest := filepath.Join(dir, "pack.yml")
		if err := os.WriteFile(manifest, []byte(doctored), 0o644); err != nil {
			t.Fatalf("write doctored manifest: %v", err)
		}

		scope, err := zeroMatchClassificationScope(manifest)
		if err != nil {
			t.Fatalf("deriving from the doctored manifest: %v", err)
		}
		if len(scope) != 1 {
			t.Fatalf("a path declared twice must be emitted once; got %v", scope)
		}
		if scope[0] != "testdata/fixtures/rules/referenced-symbol-go/negative.go" {
			t.Errorf("derived scope = %v, want the single declared path", scope)
		}
	})
}

// TestZeroMatchHarnessGlob_LeavesReferencedSymbolDarkEvenInHiddenDirs (ISSUE-158 CLM-003)
// — THE DISCRIMINATING TEST. ISSUE-113's meaning is that referenced-symbol-go yields ZERO
// findings anywhere in the consumer workspace, and that the darkness is STRUCTURAL.
func TestZeroMatchHarnessGlob_LeavesReferencedSymbolDarkEvenInHiddenDirs(t *testing.T) {
	requireAstGrepE2E(t)

	ws := zeroMatchHarnessInstall(t)
	sgconfig := filepath.Join(ws.installInfo.InstalledPath, "ast-grep", "sgconfig.yml")

	// `--no-ignore hidden` LOOKS GRATUITOUS AND IS NOT. Without it ast-grep skips the
	// hidden `.backstop/` tree, so the installed pack's own fixtures never enter the scan
	// and this test passes for a `**/`-prefixed glob too — accidentally satisfied rather
	// than discriminating. MEASURED 2026-08-17, real ast-grep 0.43.0:
	//   root-anchored (chosen)                 -> 0 referenced-symbol-go
	//   "**/testdata/fixtures/rules/.../*.go"  -> 2 referenced-symbol-go
	//   "**/fixtures/rules/**/*.go"            -> 2 referenced-symbol-go
	t.Run("hidden directories forced into the scan", func(t *testing.T) {
		findings := zeroMatchHarnessScan(t, ws.root, "scan", "--json", "--no-ignore", "hidden", "--config", sgconfig, ws.root)
		assertNoReferencedSymbolFindings(t, findings, "--no-ignore hidden")
	})

	// The everyday dispatch shape is pinned too, so a regression in either is visible.
	t.Run("default flags", func(t *testing.T) {
		findings := zeroMatchHarnessScan(t, ws.root, "scan", "--json", "--config", sgconfig, ws.root)
		assertNoReferencedSymbolFindings(t, findings, "default flags")
	})
}

// assertNoReferencedSymbolFindings scopes the assertion to referenced-symbol-go ONLY.
// Under `--no-ignore hidden` the installed pack's own negative.go legitimately yields one
// hollow-test-go finding; that rule is untouched by this lane, and a "zero findings of any
// rule" assertion would go red for a reason ISSUE-158 does not own.
func assertNoReferencedSymbolFindings(t *testing.T, findings []astGrepFinding, shape string) {
	t.Helper()
	var hits []astGrepFinding
	for _, f := range findings {
		if f.RuleID == "referenced-symbol-go" {
			hits = append(hits, f)
		}
	}
	if len(hits) != 0 {
		t.Errorf("the zero-match workspace yielded %d referenced-symbol-go finding(s) under %s — the patched scope "+
			"is not consumer-dark: %s", len(hits), shape, describeFindings(hits))
	}
}

// zeroMatchHarnessScan runs the real ast-grep binary from a chosen working directory.
//
// scanWithCombinedRuleset (substantiveness_fixture_polarity_test.go) CANNOT be reused
// here: its argv is fixed and carries neither `--no-ignore hidden` nor working-directory
// control, and widening it would change the shape its existing callers depend on. The
// tool name travels as DATA through the package's execCommand seam (root_test.go), never
// as an inline literal handed to exec.Command — backstop-self's no-baked-tool-exec.
func zeroMatchHarnessScan(t *testing.T, workDir string, args ...string) []astGrepFinding {
	t.Helper()
	cmd := execCommand(astGrepTool, args...)
	cmd.Dir = workDir
	var stderr []byte
	out, err := cmd.Output()
	if ee, ok := err.(*exec.ExitError); ok {
		stderr = ee.Stderr
	}
	var findings []astGrepFinding
	if jsonErr := json.Unmarshal(out, &findings); jsonErr != nil {
		t.Fatalf("ast-grep %v (dir=%s): unparseable json (%v); run err=%v stderr=%s stdout=%s",
			args, workDir, jsonErr, err, stderr, out)
	}
	return findings
}

// TestZeroMatchHarnessDocstring_DropsTheFalsifiedPhase3Premise (ISSUE-158 CLM-005) — a
// PROSE-DRIFT GUARD, not style policing.
//
// The harness's docstring claimed packval never runs these fixtures because
// `rule.File == ""` under a `rule_path:` declaration, citing ISSUE-092 as an open hole.
// ISSUE-092 CLOSED it: phase3 reads rule.RuleSourcePath(), DOES run the fixtures, and DOES
// notice the patch. The comment therefore asserted the opposite of the truth, and that is
// precisely why the fictional glob looked safe for as long as it did. An unfalsifiable
// comment about a validation hole is how a validation hole gets re-opened.
func TestZeroMatchHarnessDocstring_DropsTheFalsifiedPhase3Premise(t *testing.T) {
	requireAstGrepE2E(t)

	// Each falsified literal is stated exactly ONCE, in the slice the loop reads, so the
	// two spellings cannot drift into variants.
	falsified := []string{
		"packval never RUNS these fixtures at all",
		`rule.File == ""`,
	}

	path := filepath.Join(repoRoot(t), "cmd", "backstop", "gate_substantiveness_e2e.go")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading the harness source %s: %v", path, err)
	}
	for _, phrase := range falsified {
		if strings.Contains(string(data), phrase) {
			t.Errorf("%s still asserts the falsified ISSUE-092 premise %q — packval DOES run these fixtures now "+
				"(phase3 reads rule.RuleSourcePath()), so the comment tells a future reader the opposite of the truth",
				path, phrase)
		}
	}
}
