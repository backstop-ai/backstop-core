package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/artifact"
	"github.com/backstop-ai/backstop-core/pkg/waiver"
)

// waiver_unbound_test.go — ISSUE-097, the PRODUCTION half: the whole-tree harvest, its
// two precision filters, the lock as the namespace authority, and the wiring that keeps
// the check from being a dark gate.
//
// Fixture PROJECTS only. The one test that measures the real repository lives in
// waiver_repo_unbound_test.go.

// unboundFixtureRuleID keys a pack namespace no fixture lock records.
const unboundFixtureRuleID = "ghost-org/ghost-pack/ghost.rules.some-rule"

// writeFixtureFile writes content at a project-relative path, creating parents.
func writeFixtureFile(t *testing.T, root, rel, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("creating %s: %v", filepath.Dir(full), err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("writing %s: %v", rel, err)
	}
}

// tokenSource returns a Go source file body carrying one unbound waiver token. The
// marker is assembled at runtime so this test file does not itself plant a token the
// repository-wide harvest would pick up.
func tokenSource() string {
	return "package fixture\n\n// @" + "waiver:" + unboundFixtureRuleID + ":deferred:2999-01-01 fixture token\nfunc F() {}\n"
}

// harvestedRuleIDs reduces a harvest to the files it read tokens from.
func harvestedFiles(tokens []waiver.Waiver) []string {
	out := make([]string, 0, len(tokens))
	for _, tok := range tokens {
		out = append(out, filepath.ToSlash(tok.File))
	}
	return out
}

// TestUnboundWaiverHarvest_ExcludesArtifactRootAndTestdata is CLM-011, the precision
// claim — the one that decides whether this feature survives contact with the repo.
// Without F1 and F2 the whole-tree scan reports 26 candidates in this repository of
// which 5 are real, and a warning at that signal ratio is silent green in a costume.
//
// THE UNCONFIGURED LEG IS THE LOAD-BEARING HALF. This repository declares no
// artifact_root, so ResolveRoot returns the PROJECT ROOT marked Configured:false. An
// implementation that excluded everything under Root.Path would therefore exclude the
// entire tree and report nothing, forever — and a configured-root-only fixture would go
// green over exactly that bug.
func TestUnboundWaiverHarvest_ExcludesArtifactRootAndTestdata(t *testing.T) {
	cases := []struct {
		name     string
		declared string
	}{
		{"unconfigured artifact_root — the shape this repository actually has", ""},
		{"configured artifact_root", "docs"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			prefix := ""
			if tc.declared != "" {
				prefix = tc.declared + "/"
			}

			// The SAME token in four places. Two must survive the filters, two must not.
			writeFixtureFile(t, root, "src/app.go", tokenSource())
			writeFixtureFile(t, root, prefix+"issues/ISSUE-001-example.issue.md", "# quoted\n\n"+tokenSource())
			writeFixtureFile(t, root, "pkg/thing/testdata/sample.go", tokenSource())
			writeFixtureFile(t, root, "pkg/thing/mytestdata/sample.go", tokenSource())
			if tc.declared != "" {
				// ResolveRoot refuses a configured root that is absent from disk.
				if err := os.MkdirAll(filepath.Join(root, tc.declared), 0o755); err != nil {
					t.Fatalf("creating declared root: %v", err)
				}
			}

			resolved, err := artifact.ResolveRoot(root, tc.declared)
			if err != nil {
				t.Fatalf("resolving artifact root: %v", err)
			}
			if resolved.Configured != (tc.declared != "") {
				t.Fatalf("premise broken: Configured = %v for declared %q", resolved.Configured, tc.declared)
			}

			tokens := harvestProjectWaiverTokens(root, resolved)
			files := harvestedFiles(tokens)

			if len(tokens) != 2 {
				t.Fatalf("expected exactly 2 harvested tokens (the plain source file and the "+
					"mytestdata one), got %d from %v", len(tokens), files)
			}

			var sawPlain, sawLookalike bool
			for _, f := range files {
				switch f {
				case "src/app.go":
					sawPlain = true
				case "pkg/thing/mytestdata/sample.go":
					sawLookalike = true
				default:
					t.Errorf("unexpected harvested file %q", f)
				}
			}
			if !sawPlain {
				// This is the assertion that makes the test capable of failing against an
				// implementation that excludes Root.Path.
				if tc.declared == "" {
					t.Error("a plain source file at the project root was NOT harvested under an " +
						"UNCONFIGURED artifact_root; the exclusion is testing Root.Path rather than " +
						"the per-kind Root.Dir(kind) directories, which in this repo means the check " +
						"reports nothing, forever")
				} else {
					t.Error("a plain source file was not harvested")
				}
			}
			if !sawLookalike {
				t.Error("a directory named `mytestdata` was excluded; the testdata filter must be an " +
					"exact directory-SEGMENT match, not a substring")
			}
			for _, f := range files {
				if strings.Contains(f, "issues/") {
					t.Errorf("a token quoted inside artifact prose (%s) was harvested; an issue that "+
						"QUOTES a broken token is documentation, not configuration", f)
				}
				if strings.Contains(f, "/testdata/") {
					t.Errorf("a token under a testdata directory (%s) was harvested", f)
				}
			}
		})
	}
}

// TestUnboundWaiverNamespaces_ReadFromLockNotInstalledPacks is CLM-009's fresh-clone
// claim. buildWaiverPolicy reads namespaces from .backstop/packs/, which is GITIGNORED:
// keying this check to it would declare every pack-namespaced waiver in the repository
// unbound on any tree where packs are simply not installed. backstop.lock is TRACKED and
// is the project's durable record of pack identity.
func TestUnboundWaiverNamespaces_ReadFromLockNotInstalledPacks(t *testing.T) {
	root := t.TempDir()
	writeFixtureFile(t, root, "backstop.lock", `packs:
  backstop-ai/backstop-self:
    name: backstop-ai/backstop-self
    version: 1.1.3
    source_type: local
    install_date: "2026-08-16T00:00:00Z"
`)
	// Deliberately NO .backstop/packs directory at all — the fresh-clone shape.
	if _, err := os.Stat(filepath.Join(root, ".backstop", "packs")); !os.IsNotExist(err) {
		t.Fatalf("premise broken: the fixture must have no installed packs")
	}

	namespaces := lockedPackNamespaces(root)
	if len(namespaces) != 1 || namespaces[0] != "backstop-ai/backstop-self" {
		t.Fatalf("namespaces must come from backstop.lock's entry names, got %v", namespaces)
	}

	bound := []waiver.Waiver{{RuleID: "backstop-ai/backstop-self/some.rules.a-rule", File: "src/app.go", Line: 3}}
	if diags := waiver.Unbound(bound, namespaces); len(diags) != 0 {
		t.Errorf("a token keyed to a LOCKED pack must not be flagged on a tree with no installed "+
			"packs, got %d diagnostics (%#v)", len(diags), diags)
	}

	unbound := []waiver.Waiver{{RuleID: unboundFixtureRuleID, File: "src/app.go", Line: 3}}
	if diags := waiver.Unbound(unbound, namespaces); len(diags) != 1 {
		t.Errorf("premise broken: a token keyed to an unrecorded pack must still be flagged, got %d", len(diags))
	}

	// THE NEGATIVE LEG (SE3): with backstop.lock ABSENT the namespace set is empty and
	// the scan yields ZERO diagnostics rather than flagging every token in the tree.
	empty := t.TempDir()
	if ns := lockedPackNamespaces(empty); len(ns) != 0 {
		t.Errorf("a missing backstop.lock must yield an EMPTY namespace slice, got %v", ns)
	}
	if diags := waiver.Unbound(unbound, lockedPackNamespaces(empty)); len(diags) != 0 {
		t.Errorf("with no lock the scan must stay SILENT, got %d diagnostics; \"no namespaces "+
			"known\" is not \"no pack is legitimate\"", len(diags))
	}
}

// unboundFixtureProject copies the waiver-e2e fixture into a temp dir and plants one
// unbound token in a source file the fixture's lock cannot bind.
func unboundFixtureProject(t *testing.T) string {
	t.Helper()
	temp := t.TempDir()
	copyTree(t, waiverE2EFixtureRoot(t), temp)
	t.Setenv("WAIVER_E2E_SCENARIO", "waivable")
	writeFixtureFile(t, temp, "src/orphan.go", tokenSource())
	return temp
}

// runGateScoped drives the REAL shipped runGate in dir. With no files it runs the full
// `--all` sweep; with files it runs the narrow explicit-file scope. It exists beside
// runGateInDir because the scope-independence claim needs BOTH shapes over one project,
// and runGateInDir hardcodes --all.
func runGateScoped(t *testing.T, dir string, files ...string) (string, error) {
	t.Helper()
	orig, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(orig) }()

	cmd := newGateCommand(new(bool))
	cmd.Flags().Bool("json", false, "")
	if len(files) == 0 {
		if err := cmd.Flags().Set("all", "true"); err != nil {
			t.Fatalf("set --all: %v", err)
		}
	} else {
		for _, f := range files {
			if err := cmd.Flags().Set("file", f); err != nil {
				t.Fatalf("set --file: %v", err)
			}
		}
	}
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	err := runGate(cmd, files)
	return buf.String(), err
}

// TestGate_UnboundWaiverScanWiredAtConstructionSite is the anti-dark-gate test
// (CLM-009). SPEC-049 shipped a fully-built waiver subsystem that suppressed nothing for
// want of one call at the construction site; a unit test of the harvest function alone
// cannot detect an unwired Option, so this drives the REAL shipped gate.
func TestGate_UnboundWaiverScanWiredAtConstructionSite(t *testing.T) {
	out, _ := runGateScoped(t, unboundFixtureProject(t))

	if !strings.Contains(out, unboundFixtureRuleID) {
		t.Fatalf("the shipped gate must name the unbound token; the Option is defined but not "+
			"called at the construction site. Output:\n%s", out)
	}
	if !strings.Contains(out, "unbound") {
		t.Errorf("the waiver_resolution report must carry an `unbound` clause, got:\n%s", out)
	}
}

// TestGate_UnboundWaiverScan_IdenticalAcrossScopes is the claim ISSUE-097's 2026-08-16
// entry demands: the fix must surface "regardless of gate scope, not just on a full
// sweep", because this project's day-to-day loop is the diff-scoped gate.
//
// The narrow scope deliberately EXCLUDES the file carrying the token — a scope that
// happens to contain it would prove nothing.
func TestGate_UnboundWaiverScan_IdenticalAcrossScopes(t *testing.T) {
	project := unboundFixtureProject(t)

	allOut, _ := runGateScoped(t, project)
	narrowOut, _ := runGateScoped(t, project, "src/app.go")

	allClause := unboundClause(t, allOut)
	narrowClause := unboundClause(t, narrowOut)

	if allClause != narrowClause {
		t.Fatalf("the unbound clause must be IDENTICAL across scopes.\n  --all:  %q\n  narrow: %q\n"+
			"a scope-dependent clause means the harvest read the gate run's ACTIVE scope instead of "+
			"computing its own whole-tree file list", allClause, narrowClause)
	}
	if !strings.Contains(narrowClause, unboundFixtureRuleID) {
		t.Errorf("the narrow-scope run must still name the token in a file OUTSIDE its scope, got %q",
			narrowClause)
	}
}

// unboundClause extracts the `N unbound (...)` clause from a gate's human output.
func unboundClause(t *testing.T, out string) string {
	t.Helper()
	for _, line := range strings.Split(out, "\n") {
		idx := strings.Index(line, " unbound (")
		if idx < 0 {
			continue
		}
		start := strings.LastIndex(line[:idx], " ")
		end := strings.Index(line[idx:], ")")
		if start < 0 || end < 0 {
			continue
		}
		return strings.TrimSpace(line[start : idx+end+1])
	}
	t.Fatalf("no unbound clause found in gate output:\n%s", out)
	return ""
}

// TestWaiverList_NamesUnboundTokens is CLM-010. The section is ALWAYS LABELED, present
// even at count ZERO, matching the Active / Expiring / Unused sections — a section that
// disappears when empty is how this class of rot hid in the first place.
func TestWaiverList_NamesUnboundTokens(t *testing.T) {
	// The always-labeled half, asserted where a zero count is constructible.
	if out := formatWaiverList(waiver.Result{}, waiverCLINow()); !strings.Contains(strings.ToLower(out), "unbound") {
		t.Errorf("the unbound section must be labeled even at count ZERO, got:\n%s", out)
	}

	// The naming half, over a real fixture project driven through the shipped handler.
	project := unboundFixtureProject(t)
	orig, _ := os.Getwd()
	if err := os.Chdir(project); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(orig) }()

	cmd := newWaiverCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	if err := runWaiverList(cmd, nil); err != nil {
		t.Fatalf("runWaiverList: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, unboundFixtureRuleID) {
		t.Fatalf("`waiver list` must name the unbound token's rule-id, got:\n%s", out)
	}
	if !strings.Contains(out, "src/orphan.go") {
		t.Errorf("`waiver list` must name the token's file so the reader can navigate to the line "+
			"they must edit, got:\n%s", out)
	}
}
