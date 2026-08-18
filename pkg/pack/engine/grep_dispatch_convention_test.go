// Package engine_test carries PLAN-ISSUE-166's source sweeps.
//
// ★ WHY AN EXTERNAL TEST PACKAGE (`engine_test`) RATHER THAN `engine`.
// TestGrepEngineDeclarations_ForceFilenameHeader must identify a pack manifest
// STRUCTURALLY — by whether the REAL reader parses it — rather than by the filename
// `pack.yml`, because a filename-keyed sweep provably misses a real convert-bearing
// grep declaration (`pkg/pack/engine/testdata/contracts-grep-engine.yml`). That
// reader is `pack.ParseManifestFile`, and `pkg/pack/manifest.go` imports
// `pkg/pack/engine` — so an IN-PACKAGE test importing it is rejected by the
// compiler with "import cycle not allowed in test" (verified 2026-08-18). The
// external test package is the only shape that can use the production reader; it
// lives in the same directory and the same file the plan names.
//
// The one consequence: `repoRoot` (declared in package `engine`) is not visible
// here, so this file declares `conventionRepoRoot`. That is NOT a silent duplicate
// of a helper this file could have reused — it is across a package boundary the
// compiler enforces — and the distinct name is deliberate so a reader is not misled
// into thinking one shadows the other.
package engine_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack"
)

// grep_dispatch_convention_test.go (PLAN-ISSUE-166, CLM-003/CLM-004/CLM-007):
// three SOURCE SWEEPS. They read repo files and invoke nothing. Their whole point
// is to make three conventions OPERABLE rather than commentary — a rule carried
// only by prose drifts, which is exactly how a third byte-identical copy of the
// convert script and four flag-less grep declarations accumulated unnoticed.

// conventionRepoRoot walks up from the test's working dir to the module root. See
// the package comment for why this cannot simply reuse package `engine`'s
// `repoRoot`.
func conventionRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for i := 0; i < 8; i++ {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	t.Fatal("could not locate module root (go.mod) from test working dir")
	return ""
}

// commandTool returns the first token of a declared `command:` string — the TOOL
// being invoked, as opposed to any occurrence of the word anywhere in the command.
// `ast-grep` and `semgrep` are therefore distinct tools, not grep, by construction.
func commandTool(command string) string {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

// commandHasFlag reports whether a declared command string passes a flag as its own
// token. Substring matching would let `-Hsomething` or a pattern containing "-I"
// satisfy the convention.
func commandHasFlag(command, flag string) bool {
	for _, f := range strings.Fields(command) {
		if f == flag {
			return true
		}
	}
	return false
}

// walkRepoFiles walks from root, skipping VCS metadata, installed pack output and
// vendored trees, and calls fn for each regular file.
//
// THE `.backstop/` EXCLUSION IS LOAD-BEARING TWICE. It drops the INSTALLED copy of
// `backstop-ai/go-contracts` (gitignored install output, fixed at its own source
// repo and pulled in with `pack update` — never edited in place) AND the two
// `verdict-probe` fixture packs, which live under a testdata-embedded
// `.backstop/packs/` path.
func walkRepoFiles(t *testing.T, root string, fn func(path string, d fs.DirEntry)) {
	t.Helper()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", ".backstop", "node_modules", "vendor":
				return fs.SkipDir
			}
			return nil
		}
		fn(path, d)
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s: %v", root, err)
	}
}

// grepDeclaration is one convert-bearing grep engine binding found by the sweep.
type grepDeclaration struct {
	rel     string // repo-relative manifest path
	engine  string // the engines: key
	command string
}

// TestGrepEngineDeclarations_ForceFilenameHeader (CLM-003) asserts that every
// repo-owned engine binding whose command is grep AND which declares a `convert:`
// forces BOTH the filename header (-H) and binary-file suppression (-I).
//
// Those two flags are what guarantee the convert's input is `file:line:text` on
// EVERY line — under GNU grep and BSD grep alike, for single-file, multi-file and
// directory targets, with no non-match informational line mixed in. They are
// asserted TOGETHER because they are added by the same edit for the same reason (a
// convert that must parse every stdout line); splitting them invites half the fix.
//
// ★ THE SWEEP IDENTIFIES MANIFESTS BY SHAPE, NOT BY FILENAME. A `pack.yml`-keyed
// walk is KNOWN INCOMPLETE: `pkg/pack/engine/testdata/contracts-grep-engine.yml` is
// a genuine convert-bearing grep declaration that such a walk cannot see. Parsing
// with the production reader also excludes `plans/`/`specs/` prose for free — those
// files contain `command: grep` inside narrative and illustrative YAML and must
// never be treated as edit targets — and it keeps working if a manifest moves.
func TestGrepEngineDeclarations_ForceFilenameHeader(t *testing.T) {
	root := conventionRepoRoot(t)

	var convertBearing []grepDeclaration
	var exempt []grepDeclaration
	manifestCount := 0

	walkRepoFiles(t, root, func(path string, d fs.DirEntry) {
		switch filepath.Ext(d.Name()) {
		case ".yml", ".yaml":
		default:
			return
		}
		manifest, err := pack.ParseManifestFile(path)
		if err != nil {
			return // not a pack manifest; the parse IS the filter.
		}
		if manifest.Name == "" || len(manifest.Engines) == 0 {
			return
		}
		manifestCount++
		rel, _ := filepath.Rel(root, path)

		names := make([]string, 0, len(manifest.Engines))
		for name := range manifest.Engines {
			names = append(names, name)
		}
		sort.Strings(names)

		for _, name := range names {
			spec := manifest.Engines[name]
			if commandTool(spec.Command) != "grep" {
				continue
			}
			decl := grepDeclaration{rel: rel, engine: name, command: spec.Command}
			if spec.Convert == "" {
				// EXEMPT, and the reason is STRUCTURAL: nothing ever parses a filename
				// out of this binding's stdout. If it later gains a `convert:`, it
				// leaves the exempt set automatically and the sweep demands both flags.
				exempt = append(exempt, decl)
				continue
			}
			convertBearing = append(convertBearing, decl)
		}
	})

	t.Logf("parsed %d pack manifests; %d convert-bearing grep bindings, %d exempt",
		manifestCount, len(convertBearing), len(exempt))

	// A sweep that silently matches nothing is the vacuous shape this whole lane is
	// about.
	if len(convertBearing) == 0 {
		t.Fatal("sweep found ZERO convert-bearing grep declarations — the walk or the " +
			"manifest predicate is broken, and this test would pass while checking nothing")
	}

	// At authoring there are exactly SIX. If the sweep finds a SEVENTH that is a
	// REAL FINDING: fix the declaration and say so. Never trim the expectation until
	// the count matches.
	const wantConvertBearing = 6
	if len(convertBearing) != wantConvertBearing {
		t.Errorf("found %d convert-bearing grep declarations, expected %d — if this is a "+
			"NEW declaration, give it -H -I; do not narrow the sweep",
			len(convertBearing), wantConvertBearing)
	}

	for _, decl := range convertBearing {
		hasH := commandHasFlag(decl.command, "-H")
		hasI := commandHasFlag(decl.command, "-I")
		t.Logf("convert-bearing: %s [%s] command=%q -H=%v -I=%v",
			decl.rel, decl.engine, decl.command, hasH, hasI)
		if !hasH {
			t.Errorf("%s [%s]: command %q must pass -H. GNU grep OMITS the filename when "+
				"the target is a single explicit file, and this binding's convert parses "+
				"<file>:<line>:<text>", decl.rel, decl.engine, decl.command)
		}
		if !hasI {
			t.Errorf("%s [%s]: command %q must pass -I. grep otherwise writes a non-match "+
				"\"Binary file <path> matches\" line to STDOUT that the convert must not be "+
				"asked to parse", decl.rel, decl.engine, decl.command)
		}
	}

	// ── THE EXEMPTIONS, ASSERTED RATHER THAN ASSUMED ────────────────────────────
	// Enumerated so the predicate cannot quietly widen. The assertion is the REASON
	// (no convert), not a hardcoded pass — a path list would rot the moment one of
	// these gained a convert.
	exemptPaths := map[string]bool{
		"cmd/backstop/testdata/hermetic-remote/scaffold-templating-pack/pack.yml": true,
		"cmd/backstop/testdata/hermetic-remote/scaffold-recipe-pack/pack.yml":     true,
	}
	seenExempt := map[string]bool{}
	for _, decl := range exempt {
		t.Logf("exempt (declares no convert): %s [%s] command=%q", decl.rel, decl.engine, decl.command)
		seenExempt[decl.rel] = true
		if !exemptPaths[decl.rel] {
			t.Errorf("%s [%s] is a grep binding with NO convert that this test does not "+
				"know about — confirm nothing parses its stdout, then enumerate it here",
				decl.rel, decl.engine)
		}
	}
	for p := range exemptPaths {
		if !seenExempt[p] {
			t.Errorf("expected exempt grep declaration at %s was not found — it either "+
				"moved or GAINED a convert, in which case it must now carry -H -I", p)
		}
	}
}

// stringLitArgs returns the string-literal arguments of a call expression, in
// order, ignoring non-literal arguments.
func stringLitArgs(call *ast.CallExpr) []string {
	var out []string
	for _, arg := range call.Args {
		lit, ok := arg.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			continue
		}
		unquoted, err := strconv.Unquote(lit.Value)
		if err != nil {
			continue
		}
		out = append(out, unquoted)
	}
	return out
}

// hasArg reports whether args contains an exact value.
func hasArg(args []string, want string) bool {
	for _, a := range args {
		if a == want {
			return true
		}
	}
	return false
}

// TestGrepTestHelpers_ForceFilenameHeader (CLM-004) asserts that every in-repo test
// helper shelling real grep for a contracts probe passes -H and -I too.
//
// These helpers bypass `pack.yml` ENTIRELY, which is the decisive fact against a
// manifest-only fix: -H added only to the declarations would leave four real-grep
// call sites still producing the broken 2-field shape on Linux. The assertion is on
// the SAME CALL EXPRESSION, not merely the same file, so an unrelated flag
// elsewhere in the file cannot satisfy it.
func TestGrepTestHelpers_ForceFilenameHeader(t *testing.T) {
	root := conventionRepoRoot(t)

	// EXCLUDED BY EXACT PATH: this lane's own two test files carry grep argv
	// fragments as test DATA (fixture bytes and sweep predicates), not as calls that
	// dispatch a contracts probe. Excluding them by exact path — rather than by a
	// pattern that might swallow a real helper — keeps the sweep's meaning intact.
	excluded := map[string]bool{
		"pkg/pack/engine/grep_dispatch_convention_test.go": true,
		"pkg/pack/engine/grep_convert_shape_test.go":       true,
	}

	type callSite struct {
		rel  string
		line int
		args []string
	}
	var sites []callSite

	fset := token.NewFileSet()
	walkRepoFiles(t, root, func(path string, d fs.DirEntry) {
		if !strings.HasSuffix(d.Name(), "_test.go") {
			return
		}
		rel, _ := filepath.Rel(root, path)
		if excluded[filepath.ToSlash(rel)] {
			return
		}
		src, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("reading %s: %v", rel, err)
			return
		}
		if !strings.Contains(string(src), `"grep", "-rn"`) {
			return
		}
		file, err := parser.ParseFile(fset, path, src, 0)
		if err != nil {
			t.Errorf("parsing %s: %v", rel, err)
			return
		}
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			args := stringLitArgs(call)
			// The call is a real-grep dispatch iff "grep" is IMMEDIATELY followed by
			// "-rn" among its string-literal arguments — the argv fragment the sweep
			// is defined over.
			for i := 0; i+1 < len(args); i++ {
				if args[i] == "grep" && args[i+1] == "-rn" {
					sites = append(sites, callSite{
						rel:  rel,
						line: fset.Position(call.Pos()).Line,
						args: args,
					})
					return true
				}
			}
			return true
		})
	})

	sort.Slice(sites, func(i, j int) bool {
		if sites[i].rel != sites[j].rel {
			return sites[i].rel < sites[j].rel
		}
		return sites[i].line < sites[j].line
	})

	// The COUNT is the exhaustiveness proof. A fifth helper added later must trip
	// this test rather than quietly inheriting the broken shape.
	const wantSites = 4
	if len(sites) == 0 {
		t.Fatal("sweep found ZERO real-grep test helpers — the walk is broken and this " +
			"test would pass while checking nothing")
	}
	if len(sites) != wantSites {
		t.Errorf("found %d real-grep test helper call sites, expected %d — a helper was "+
			"added or removed; give a new one -H -I rather than narrowing the sweep",
			len(sites), wantSites)
	}

	for _, s := range sites {
		hasH := hasArg(s.args, "-H")
		hasI := hasArg(s.args, "-I")
		t.Logf("real-grep helper: %s:%d args=%v -H=%v -I=%v", s.rel, s.line, s.args, hasH, hasI)
		if !hasH {
			t.Errorf("%s:%d shells real grep without -H; on Linux this call produces the "+
				"2-field shape the convert cannot parse", s.rel, s.line)
		}
		if !hasI {
			t.Errorf("%s:%d shells real grep without -I; a binary file anywhere in the "+
				"scope puts a non-match line on stdout", s.rel, s.line)
		}
	}
}

// TestThinExecutor_NoGrepInvocationInNonTestCode (CLM-007) asserts that non-test
// code under pkg/ and cmd/ declares NO grep invocation. Backstop is a THIN
// EXECUTOR with zero baked tool knowledge: the tool name must reach the runner only
// from pack DATA.
//
// ★ THIS TEST IS GREEN FROM BIRTH AND THAT IS THE POINT. Core carries no grep
// invocation today and this lane adds none, so passing it proves nothing on its
// own — it is a REGRESSION FENCE, not a RED. Its job is to fail the day someone
// "fixes" a future grep-shape bug INSIDE core (a -H injection, a filename-shape
// normalizer, a grep-aware SARIF fallback) instead of in a pack. Because it cannot
// go red on its own, PLAN-ISSUE-166 TASK-006 falsifies it deliberately: introduce a
// throwaway exec.Command("grep", "-rn") in a non-test file under pkg/, observe the
// RED, then remove it. An unfalsified always-green test is the vacuous assertion
// this repo forbids.
//
// ★ THE PREDICATE IS SCOPED TO INVOCATION, NOT TO THE WORD. The substring "grep"
// appears legitimately all over these packages — `ast-grep`, `semgrep`,
// `astGrepMatchCount`, doc comments on runFindingsEngine. A bare word-match would
// red instantly for the wrong reason and would then be "fixed" by weakening it. So
// the assertion is on the ARGV/COMMAND POSITION: the name argument of an
// exec.Command-shaped call, or the FIRST TOKEN of a command string literal.
// `ast-grep` and `semgrep` are distinct tool names and pass by construction.
func TestThinExecutor_NoGrepInvocationInNonTestCode(t *testing.T) {
	root := conventionRepoRoot(t)

	type violation struct {
		rel    string
		line   int
		detail string
	}
	var violations []violation
	var namePositions []violation
	rejected := map[string]int{}
	filesExamined := 0

	fset := token.NewFileSet()
	for _, top := range []string{"pkg", "cmd"} {
		walkRepoFiles(t, filepath.Join(root, top), func(path string, d fs.DirEntry) {
			if !strings.HasSuffix(d.Name(), ".go") || strings.HasSuffix(d.Name(), "_test.go") {
				return
			}
			// testdata trees are FIXTURE data, not core code.
			if strings.Contains(filepath.ToSlash(path), "/testdata/") {
				return
			}
			rel, _ := filepath.Rel(root, path)
			src, err := os.ReadFile(path)
			if err != nil {
				t.Errorf("reading %s: %v", rel, err)
				return
			}
			filesExamined++
			if !strings.Contains(string(src), "grep") {
				return
			}
			file, err := parser.ParseFile(fset, path, src, parser.ParseComments)
			if err != nil {
				t.Errorf("parsing %s: %v", rel, err)
				return
			}

			// (a) exec.Command-shaped calls: the NAME argument is the tool.
			ast.Inspect(file, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				if sel.Sel.Name != "Command" && sel.Sel.Name != "CommandContext" {
					return true
				}
				args := stringLitArgs(call)
				if len(args) == 0 {
					return true
				}
				// CommandContext's first STRING literal is still the name (the ctx
				// argument is not a literal and is skipped by stringLitArgs).
				if args[0] == "grep" {
					violations = append(violations, violation{
						rel:    rel,
						line:   fset.Position(call.Pos()).Line,
						detail: "exec.Command-shaped call naming \"grep\" as the tool",
					})
				} else {
					rejected[args[0]]++
				}
				return true
			})

			// (b) command string literals whose FIRST TOKEN is the tool.
			ast.Inspect(file, func(n ast.Node) bool {
				lit, ok := n.(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					return true
				}
				val, err := strconv.Unquote(lit.Value)
				if err != nil {
					return true
				}
				if !strings.Contains(val, "grep") {
					return true
				}
				tool := commandTool(val)
				if tool != "grep" {
					if tool != "" {
						rejected[tool]++
					}
					return true
				}
				// ★ A BARE TOOL NAME IS NOT AN INVOCATION. A single-token "grep" is a
				// NAME in a name position — a key in the trusted-tool allowlist, or the
				// `engines:` key a pack manifest declares — and naming an allowlisted
				// tool or an engine key is precisely what a thin executor legitimately
				// does: it never says HOW to run it. The command string, and therefore
				// every flag, still arrives from pack DATA. An INVOCATION is a command
				// literal with arguments ("grep -rn …") or an exec.Command-shaped call
				// naming grep, and both of those are caught (leg (a) above and the
				// multi-token case here).
				//
				// These occurrences are RECORDED rather than filtered silently, so a
				// reader can audit them instead of trusting this comment.
				if len(strings.Fields(val)) < 2 {
					namePositions = append(namePositions, violation{
						rel:    rel,
						line:   fset.Position(lit.Pos()).Line,
						detail: "bare tool NAME " + strconv.Quote(val) + " in a name position (not an invocation)",
					})
					return true
				}
				violations = append(violations, violation{
					rel:    rel,
					line:   fset.Position(lit.Pos()).Line,
					detail: "command string literal whose first token is \"grep\": " + val,
				})
				return true
			})
		})
	}

	t.Logf("examined %d non-test .go files under pkg/ and cmd/", filesExamined)
	if filesExamined == 0 {
		t.Fatal("examined ZERO files — the walk is broken and this fence checks nothing")
	}
	tools := make([]string, 0, len(rejected))
	for tool := range rejected {
		tools = append(tools, tool)
	}
	sort.Strings(tools)
	for _, tool := range tools {
		t.Logf("candidate rejected (%d occurrence(s)): first token %q is not grep", rejected[tool], tool)
	}
	for _, np := range namePositions {
		t.Logf("candidate rejected: %s:%d — %s", np.rel, np.line, np.detail)
	}

	for _, v := range violations {
		t.Errorf("%s:%d declares a grep invocation in NON-TEST core code (%s). Backstop "+
			"bakes no tool knowledge: grep reaches the runner only from pack DATA. If this "+
			"is a fix for a grep output-shape problem, it belongs in the PACK's command "+
			"declaration or convert script, not here", v.rel, v.line, v.detail)
	}
}
