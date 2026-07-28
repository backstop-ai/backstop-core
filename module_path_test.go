package backstopcore_test

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const (
	// legacyModulePath is the pre-rename module path. No live Go source, Go
	// fixture, or script may reference it once the rename lands.
	//
	// ⚠ This is the ONE place in the live tree that must keep the legacy string.
	// The ISSUE-087 sweep (a repo-wide sed over .go/.go.txt/.sh/go.mod) rewrote
	// this very literal, which silently collapsed the two constants into the
	// same value and made the walk below search for the canonical path — a guard
	// that can never fail. Any future mechanical path rewrite must exclude this
	// file, or restore this literal afterward and re-confirm the guard still
	// distinguishes the two paths.
	legacyModulePath = "github.com/bmanson/backstop-core"
	// canonicalModulePath is the post-rename module path this repo declares.
	canonicalModulePath = "github.com/backstop-ai/backstop-core"
)

// repoRoot is the directory the root-package test binary runs in. Go runs a
// package's tests with the working directory set to the package directory, and
// this package IS the repo root, so "." is the repo root. Every skip below is
// matched against a path RELATIVE to this root.
const repoRoot = "."

// guardSourceFile is this test's own path relative to repoRoot. See the
// self-skip in TestModulePath_NoLegacyReferencesInLiveTree for why it exists.
const guardSourceFile = "module_path_test.go"

// legacyReferenceSkips names the directories the live-tree walk does not enter.
//
// Every entry is matched against the path RELATIVE TO THE REPO ROOT, never
// against a bare directory basename. That is not stylistic precision — a
// basename match silently swallows tracked files that ARE in the rename's
// scope, and the misses are invisible to every other check in this plan
// because the downstream re-greps filter with --include="*.go".
//
// Verified 2026-07-27, the two concrete traps:
//
//	.backstop  — there are 19 directories named .backstop in this repo. Exactly
//	one of them, the repo-root one, is off limits. Another,
//	cmd/backstop/testdata/go-toolchain/.backstop/packs/backstop/go-toolchain/
//	scripts/coverage-to-records.sh, is a TRACKED TESTDATA FIXTURE carrying the
//	legacy path, and it is squarely in the rename's scope. This anchored skip is
//	that fixture's ONLY falsifier. Do NOT "simplify" this into a basename match.
//
//	plans/specs/issues/bundles/directives/adrs — the repo-root ones are
//	historical artifact records, but nested fixture trees with the same names
//	exist under pkg/validate/testdata/, cmd/backstop/testdata/ and
//	pkg/gate/testdata/. Those are fixtures, not history, and a basename match
//	would exclude them from the guard for the same reason.
//
// Why each root-level entry is skipped:
//
//	.git, dist, bin, node_modules
//	  VCS internals and build output — not source, and regenerated at will.
//
//	plans, specs, issues, bundles, directives, adrs
//	  Historical records of what the module path WAS. CLAUDE.md forbids
//	  hand-editing artifacts, and ISSUE-087/DIR-001 legitimately name BOTH
//	  paths because they describe this rename.
//
//	.backstop
//	  Installed EXTERNAL pack content — gitignored, and not this repo's to
//	  edit. Its packs/backstop/go-toolchain/scripts/coverage-to-records.sh:21
//	  carries the legacy path in a comment. Editing it in place is NON-DURABLE:
//	  invisible to git, never reaching the pack's source repo, and overwritten
//	  by the next `pack install`. So the sweep correctly leaves it alone, which
//	  means the legacy path survives there permanently — and without this skip
//	  the guard is unsatisfiable, RED no matter how complete the sweep is, with
//	  both escapes wrong (edit installed pack content, or delete the guard).
//	  The pack-side comment is a filed follow-on in the backstop/go-toolchain
//	  SOURCE repo. This skip is NOT justified by tamper detection: VerifyLock
//	  skips every source_type:local entry (pkg/pack/distribution/verify.go:47-49)
//	  and go-toolchain is local, so pack_lock_verification never inspects it.
func legacyReferenceSkips() map[string]struct{} {
	return map[string]struct{}{
		".git":         {},
		"dist":         {},
		"bin":          {},
		"node_modules": {},
		"plans":        {},
		"specs":        {},
		"issues":       {},
		"bundles":      {},
		"directives":   {},
		"adrs":         {},
		".backstop":    {},
	}
}

// isSweptFile reports whether a file is in the module-rename sweep's scope:
// Go sources, Go fixture text, shell scripts, and the module file itself.
func isSweptFile(name string) bool {
	if name == "go.mod" { // nosemgrep: backstop.packs.backstop.self.rules.no-baked-language-token — repo-meta test naming this repo's OWN module file, not baked routing in the binary
		return true
	}
	if strings.HasSuffix(name, ".go.txt") {
		return true
	}
	ext := filepath.Ext(name)
	return ext == ".go" || ext == ".sh"
}

// TestModulePath_GoModDeclaresCanonicalModule asserts go.mod's parsed module
// directive is the canonical path. Asserting on the parsed directive rather
// than a substring of the file keeps a stray comment from passing vacuously.
// (CLM-001)
func TestModulePath_GoModDeclaresCanonicalModule(t *testing.T) {
	// nosemgrep: backstop.packs.backstop.self.rules.no-baked-language-token — repo-meta test asserting on this repo's OWN module file; naming it IS the claim, not baked routing in the binary
	data, err := os.ReadFile(filepath.Join(repoRoot, "go.mod"))
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}

	declared := ""
	for _, line := range strings.Split(string(data), "\n") {
		if idx := strings.Index(line, "//"); idx >= 0 {
			line = line[:idx]
		}
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "module" {
			declared = fields[1]
			break
		}
	}

	if declared == "" {
		t.Fatal("go.mod declares no module directive")
	}
	if declared != canonicalModulePath {
		t.Errorf("go.mod module directive = %q, want %q", declared, canonicalModulePath)
	}
}

// TestModulePath_NoLegacyReferencesInLiveTree walks the live tree and asserts
// no Go source, Go fixture, or script still references the legacy module path.
// Every offender is reported, not just the first — the full list is what proves
// the sweep was complete. (CLM-002)
func TestModulePath_NoLegacyReferencesInLiveTree(t *testing.T) {
	skips := legacyReferenceSkips()
	needle := []byte(legacyModulePath)
	offenders := []string{}

	walkErr := filepath.WalkDir(repoRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk %s: %w", path, err)
		}

		rel, relErr := filepath.Rel(repoRoot, path)
		if relErr != nil {
			return fmt.Errorf("relativize %s: %w", path, relErr)
		}
		rel = filepath.ToSlash(rel)

		if d.IsDir() {
			// Root-anchored: only a top-level directory matching a skip name is
			// excluded. A nested directory with the same basename is walked.
			if _, skip := skips[rel]; skip {
				return filepath.SkipDir
			}
			return nil
		}

		// This guard's own source is the ONE live .go file that must contain the
		// legacy path — it declares it as the constant being searched for. Without
		// this anchored self-skip the guard reports itself forever and can never
		// go green, no matter how complete the sweep is. Anchored to the exact
		// repo-root-relative path, so a same-named file elsewhere is still walked.
		if rel == guardSourceFile {
			return nil
		}

		if !isSweptFile(d.Name()) {
			return nil
		}

		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("read %s: %w", path, readErr)
		}
		if bytes.Contains(content, needle) {
			offenders = append(offenders, rel)
		}
		return nil
	})
	if walkErr != nil {
		t.Fatalf("walking repo root: %v", walkErr)
	}

	if len(offenders) > 0 {
		sort.Strings(offenders)
		t.Errorf("%d file(s) still reference the legacy module path %q:\n  %s",
			len(offenders), legacyModulePath, strings.Join(offenders, "\n  "))
	}
}

// TestModulePath_AgentDefinitionSnippetsImportCanonicalModule asserts the eight
// agent-definition files' fenced Go validator snippets import the canonical
// module path.
//
// These files are .md, so the live-tree walk skips them by extension. Without
// this test the agent-snippet update has NO falsifier at all, and a stale
// snippet ships silently — breaking the planner's own validator the next time
// anyone runs it.
//
// The list is FIXED rather than a glob: a glob that matches nothing passes
// vacuously, and the point is that all eight are covered. The per-file import
// set is NOT uniform — the two planner files carry two imports, the other six
// add pkg/schema. Verified 2026-07-27. A file that renamed artifact and
// validate but left pkg/schema stale must still FAIL; that is the specific miss
// this test exists to catch. (CLM-004)
func TestModulePath_AgentDefinitionSnippetsImportCanonicalModule(t *testing.T) {
	agentFiles := []struct {
		path     string
		packages []string
	}{
		{".claude/agents/planner.md", []string{"pkg/artifact", "pkg/validate"}},
		{".github/agents/planner.agent.md", []string{"pkg/artifact", "pkg/validate"}},
		{".claude/agents/plan-reviewer.md", []string{"pkg/artifact", "pkg/schema", "pkg/validate"}},
		{".claude/agents/spec-author.md", []string{"pkg/artifact", "pkg/schema", "pkg/validate"}},
		{".claude/agents/spec-reviewer.md", []string{"pkg/artifact", "pkg/schema", "pkg/validate"}},
		{".github/agents/plan-reviewer.agent.md", []string{"pkg/artifact", "pkg/schema", "pkg/validate"}},
		{".github/agents/spec-author.agent.md", []string{"pkg/artifact", "pkg/schema", "pkg/validate"}},
		{".github/agents/spec-reviewer.agent.md", []string{"pkg/artifact", "pkg/schema", "pkg/validate"}},
	}

	for _, agentFile := range agentFiles {
		// Fail loudly on a missing file rather than skipping it — a renamed or
		// removed agent definition must not silently drop out of the guard.
		data, err := os.ReadFile(filepath.Join(repoRoot, agentFile.path))
		if err != nil {
			t.Errorf("read %s: %v", agentFile.path, err)
			continue
		}
		content := string(data)

		if n := strings.Count(content, legacyModulePath); n > 0 {
			t.Errorf("%s: %d reference(s) to the legacy module path %q",
				agentFile.path, n, legacyModulePath)
		}

		// Assert on content, not on line position — line numbers drift.
		for _, pkg := range agentFile.packages {
			want := canonicalModulePath + "/" + pkg
			if n := strings.Count(content, want); n != 1 {
				t.Errorf("%s: found %d occurrence(s) of import %q, want exactly 1",
					agentFile.path, n, want)
			}
		}
	}
}
