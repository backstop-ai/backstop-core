package main

import (
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/artifact"
	"github.com/backstop-ai/backstop-core/pkg/config"
	"github.com/backstop-ai/backstop-core/pkg/waiver"
)

// waiver_repo_unbound_test.go — ISSUE-097, the DURABLE REGRESSION GUARD and the only
// test in this lane that reads the real tree.
//
// It exists because the alternative is remembering. A pack rename silently unbinds every
// waiver keyed to the old namespace; nothing warns on install, nothing warns on gate, and
// `waiver list` cannot see a token no finding happens to land beside. This fails in the
// ordinary `go test` run instead, naming the exact tokens to re-key.

// repoRootForWaiverScan resolves the repository root from this test's own location: the
// package sits at <root>/cmd/backstop.
func repoRootForWaiverScan(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolving repository root: %v", err)
	}
	return root
}

// repoWaiverTokens harvests the real tree through the SAME production harvester the gate
// uses — never a reimplementation, which would be free to disagree with production and
// still go green.
func repoWaiverTokens(t *testing.T) []waiver.Waiver {
	t.Helper()
	root := repoRootForWaiverScan(t)

	declared := ""
	if cfg, err := config.LoadConfigFromPath(filepath.Join(root, "backstop.yml")); err == nil {
		declared = cfg.ArtifactRoot
	}
	resolved, err := artifact.ResolveRoot(root, declared)
	if err != nil {
		t.Fatalf("resolving artifact root: %v", err)
	}

	tokens := harvestProjectWaiverTokens(root, resolved)
	if len(tokens) == 0 {
		t.Fatal("the harvest returned NO tokens over the real repository. This tree carries many " +
			"well-formed waiver tokens, so zero means the harvest or its exclusions are broken — " +
			"most likely the artifact-root filter is testing Root.Path (which is the project root " +
			"here, since no artifact_root is declared) instead of the per-kind directories. A " +
			"vacuously-empty harvest would make both tests below pass while checking nothing")
	}
	return tokens
}

// TestRepo_CarriesNoUnboundWaiverTokens is THE REPO INVARIANT (CLM-001/002/003): the
// working tree carries zero waiver tokens whose <org>/<pack> namespace matches no
// backstop.lock entry.
//
// Namespaces come from backstop.lock, which is TRACKED, so this does not require
// .backstop/packs/ to be installed and stays green on a fresh clone and in CI. An
// unreadable lock is a FATAL here rather than a skip: a silent skip would rebuild the
// exact invisibility this issue exists to remove.
func TestRepo_CarriesNoUnboundWaiverTokens(t *testing.T) {
	root := repoRootForWaiverScan(t)

	namespaces := lockedPackNamespaces(root)
	if len(namespaces) == 0 {
		t.Fatalf("backstop.lock at %s yielded no pack namespaces; this check cannot run without "+
			"them, and skipping would rebuild the invisibility ISSUE-097 exists to remove",
			filepath.Join(root, "backstop.lock"))
	}

	diags := waiver.Unbound(repoWaiverTokens(t), namespaces)
	if len(diags) == 0 {
		return
	}

	var b strings.Builder
	for _, d := range diags {
		b.WriteString("\n  " + d.File + ":" + strconv.Itoa(d.Line) + "  " + d.RuleID)
	}
	t.Fatalf("%d waiver token(s) name a pack that backstop.lock does not record. Each one "+
		"suppresses NOTHING and will keep suppressing nothing until it is re-keyed to a "+
		"recorded pack namespace or removed. Known namespaces: %v%s",
		len(diags), namespaces, b.String())
}

// deadDottedSegment is the pre-rename dotted rule-id segment the 2026-07-27
// backstop/self -> backstop-ai/backstop-self rename left behind.
const deadDottedSegment = "backstop.packs.backstop.self.rules."

// TestRepo_CarriesNoStalePreRenameWaiverRuleIDs is THE OTHER HALF OF THE STRING, and the
// reason this file holds two tests rather than one.
//
// Unbound reads ONLY the <org>/<pack> coordinate half of a rule id. It is structurally
// blind to a token whose coordinate half was migrated while its trailing DOTTED segment
// was not — that token's candidate name IS a real lock entry, so Unbound emits nothing,
// the invariant above goes green, and the waiver is just as dead as before. No live
// finding catches it either: the rule produces none under any current dispatch shape
// (ISSUE-151). Nothing else in this lane covers it.
//
// SCOPED HONESTLY: this guards THIS rename's dead segment, not general well-formedness of
// dotted rule ids. A generic check would have to encode that the dotted segment mirrors
// the pack's on-disk path — a semgrep-specific convention other engines do not share, and
// baking it into core test code would be a tool assumption. The next rename reopens this
// blind spot with a new dead segment; that is a filed follow-on, not an oversight.
func TestRepo_CarriesNoStalePreRenameWaiverRuleIDs(t *testing.T) {
	// ASSERTED OVER HARVESTED TOKENS, NEVER RAW FILE BYTES. The needle appears in prose,
	// in Go string literals, and in this test's own source — none of which is a waiver
	// token and none of which suppresses anything. A raw-byte scan would flag them all,
	// including this file, and the test would be un-greenable.
	var stale []waiver.Waiver
	for _, tok := range repoWaiverTokens(t) {
		if strings.Contains(tok.RuleID, deadDottedSegment) {
			stale = append(stale, tok)
		}
	}
	if len(stale) == 0 {
		return
	}

	var b strings.Builder
	for _, tok := range stale {
		b.WriteString("\n  " + tok.File + ":" + strconv.Itoa(tok.Line) + "  " + tok.RuleID)
	}
	t.Fatalf("%d waiver token(s) still carry the dead pre-rename rule-id segment %q. NOTE THAT "+
		"THE COORDINATE HALF MAY ALREADY LOOK CORRECT — that is precisely the failure this "+
		"test exists to name, because the unbound check reads only the coordinate half and is "+
		"blind to a half-migrated id. Both halves must move together.%s",
		len(stale), deadDottedSegment, b.String())
}
