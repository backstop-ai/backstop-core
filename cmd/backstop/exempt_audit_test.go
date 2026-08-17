package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
	"github.com/backstop-ai/backstop-core/pkg/packval"
)

// exemptIntent is one engine's recorded scope-filter intent. The table below IS
// ISSUE-136's audit, made executable: every engine in the committed corpus must
// carry a row, so a newly added engine cannot join the corpus unaudited.
type exemptIntent struct {
	// scopeKind is what the manifest declares; a row disagreeing with the manifest
	// fails, so the audit cannot describe a pack that no longer exists.
	scopeKind engine.ScopeKind
	// declared is whether the manifest WRITES exempt_from_scope_filter at all —
	// the tri-state an absent key and an explicit false would otherwise collapse.
	declared bool
	// value is the effective ExemptFromScopeFilter the binding resolves to.
	value bool
	// why is the grounding. It must cite the reason the decision is what it is —
	// a shipped behaviour, a by-construction argument, or a named follow-on — so a
	// future reader can CHECK the reasoning rather than trust the row.
	why string
}

const (
	goToolchainMirror = "cmd/backstop/testdata/go-toolchain/.backstop/packs/backstop/go-toolchain/pack.yml"

	// whyFileArgs is the by-construction argument shared by every file-args engine:
	// it is invoked with the scoped files as arguments, so every finding it can
	// produce is already ON a file in scope and the diff-scope filter is a no-op.
	// No decision is owed, and the advisory correctly never asks for one.
	whyFileArgs = "file-args engine: invoked with the scoped files as arguments, so every finding it can " +
		"produce is already on a file in scope and the diff-scope filter is a no-op by construction — no decision is owed"
)

// exemptAuditCorpus is every COMMITTED pack source in this repo, keyed
// "<pack source path>|<engine name>".
//
// The corpus is deliberately narrow: packs/*/pack.yml (globbed, so deleting a
// stale pack directory shrinks the corpus rather than breaking this test) plus the
// ONE committed mirror of a released external pack. The rest of testdata/ is
// synthetic fixtures, not real engine declarations, and sweeping it in would drown
// the audit in noise.
var exemptAuditCorpus = map[string]exemptIntent{ // nosemgrep: go.core.no-global-mutable-state — immutable audit table, never mutated
	"packs/base-engines/pack.yml|semgrep":  {scopeKind: engine.ScopeKindFileArgs, declared: false, value: false, why: whyFileArgs},
	"packs/base-engines/pack.yml|ast-grep": {scopeKind: engine.ScopeKindFileArgs, declared: false, value: false, why: whyFileArgs},
	"packs/base-engines/pack.yml|sandbox":  {scopeKind: engine.ScopeKindFileArgs, declared: false, value: false, why: whyFileArgs},
	"packs/base-engines/pack.yml|config-file": {
		scopeKind: engine.ScopeKindProjectWide,
		declared:  true,
		value:     false,
		why: "config-file is the GENERIC native-linter shape (a tool running its own built-in rules project-wide " +
			"from one config file) that golangci instantiates, so it inherits golangci's decision: ISSUE-070 " +
			"(\"Gate Diffscope Leaks Projectwide Lint\", closed 2026-07-27) established that project-wide lint " +
			"findings on files the change never touched MUST be filtered out, or the gate reds on pre-existing " +
			"debt. Non-exempt is that shipped behaviour, recorded explicitly rather than defaulted",
	},

	"packs/contracts/pack.yml|grep":                          {scopeKind: engine.ScopeKindFileArgs, declared: false, value: false, why: whyFileArgs},
	"packs/contracts/pack.yml|ast-grep-contracts":            {scopeKind: engine.ScopeKindFileArgs, declared: false, value: false, why: whyFileArgs},
	"packs/substantiveness/pack.yml|ast-grep-substantiveness": {scopeKind: engine.ScopeKindFileArgs, declared: false, value: false, why: whyFileArgs},

	goToolchainMirror + "|go-build": {
		scopeKind: engine.ScopeKindProjectWide,
		declared:  true,
		value:     true,
		why: "a compile error on an unchanged file is still a broken build for the change that exposed it, so " +
			"build violations must survive diff-scope filtering. Declared explicitly since SPEC-041",
	},
	goToolchainMirror + "|go-test": {
		scopeKind: engine.ScopeKindProjectWide,
		declared:  true,
		value:     true,
		why: "ISSUE-129 inverted this: a test failure in an unchanged file caused by a change elsewhere is exactly " +
			"the violation a diff-scoped gate was silently discarding, so test violations are exempt from the filter",
	},
	goToolchainMirror + "|golangci": {
		scopeKind: engine.ScopeKindProjectWide,
		declared:  false,
		value:     false,
		why: "PENDING an explicit declaration in the EXTERNAL go-toolchain pack. The audit's finding is that " +
			"non-exempt is CORRECT and load-bearing — it is ISSUE-070's delivered behaviour, and flipping it " +
			"would regress a shipped fix — but the decision is recorded nowhere, so `pack check` now emits a " +
			"standing exempt-scope-decision advisory for it. Recording it is a pack-manifest edit + version bump " +
			"+ tag + relock + fixture-mirror sync in another repository, filed as its own defect per ISSUE-136 " +
			"Direction item 2; PLAN-ISSUE-136 deliberately does not absorb it",
	},
	goToolchainMirror + "|go-coverage": {
		scopeKind: engine.ScopeKindProjectWide,
		declared:  false,
		value:     false,
		why: "PENDING an explicit declaration in the EXTERNAL go-toolchain pack, same follow-on as golangci. The " +
			"audit's finding is that the decision does not even reach the filter: gate_type: coverage routes this " +
			"engine to the coverage-records channel (dispatchPackCoverage -> ParsePackCoverage), not the SARIF " +
			"findings channel that filterViolations guards, so either value is inert today",
	},
}

// TestExemptAudit_EveryCommittedPackEngineHasAnIntentRow asserts, in BOTH
// directions, that the audit table and the committed corpus agree exactly: an
// engine with no row fails (the ratchet — a newly added engine cannot join
// unaudited), and a row naming an engine that no longer exists fails (the record
// cannot outlive its subject).
func TestExemptAudit_EveryCommittedPackEngineHasAnIntentRow(t *testing.T) {
	root := repoRoot(t)

	manifests, err := filepath.Glob(filepath.Join(root, "packs", "*", "pack.yml"))
	if err != nil {
		t.Fatalf("glob committed packs: %v", err)
	}
	if len(manifests) == 0 {
		t.Fatalf("corpus is empty: no packs/*/pack.yml found under %s", root)
	}
	manifests = append(manifests, filepath.Join(root, filepath.FromSlash(goToolchainMirror)))

	seen := map[string]bool{}
	for _, path := range manifests {
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			t.Fatalf("relativise %s: %v", path, relErr)
		}
		key := filepath.ToSlash(rel)

		pack, parseErr := packval.ParseManifest(path)
		if parseErr != nil {
			t.Fatalf("ParseManifest(%s): %v", key, parseErr)
		}

		// ExemptDecisionPending is the SINGLE authority for "a decision is owed
		// here" (pkg/packval). The census calls it rather than re-deriving the rule,
		// so the advisory and the audit can never disagree about the same manifest.
		pendingHere := map[string]bool{}
		for _, name := range packval.ExemptDecisionPending(pack) {
			pendingHere[name] = true
		}

		for name, binding := range pack.Engines {
			rowKey := key + "|" + name
			seen[rowKey] = true

			row, ok := exemptAuditCorpus[rowKey]
			if !ok {
				t.Errorf("UNAUDITED ENGINE %q: it declares scope_kind=%v exempt=%v but has no intent row. "+
					"Add one to exemptAuditCorpus stating whether it declares exempt_from_scope_filter, the "+
					"intended value, and WHY — an engine may not join the committed corpus unaudited.",
					rowKey, binding.ScopeKind, binding.ExemptFromScopeFilter)
				continue
			}
			if row.scopeKind != binding.ScopeKind {
				t.Errorf("%s: row says scope_kind=%v, manifest says %v", rowKey, row.scopeKind, binding.ScopeKind)
			}
			if row.value != binding.ExemptFromScopeFilter {
				t.Errorf("%s: row says exempt value=%v, manifest resolves %v", rowKey, row.value, binding.ExemptFromScopeFilter)
			}
			// Key presence is the tri-state, and it is what the row's `declared`
			// field claims. A project-wide engine is pending EXACTLY when it does not
			// declare the key, so the two must be inverses.
			if binding.ScopeKind == engine.ScopeKindProjectWide {
				if row.declared == pendingHere[name] {
					t.Errorf("%s: row says declared=%v but ExemptDecisionPending reports pending=%v; "+
						"a project-wide engine is pending exactly when it does NOT declare the key",
						rowKey, row.declared, pendingHere[name])
				}
			} else if row.declared {
				t.Errorf("%s: row claims the key is declared, but no decision is owed for a file-args engine", rowKey)
			}
			if strings.TrimSpace(row.why) == "" {
				t.Errorf("%s: row carries no grounding", rowKey)
			}
		}
	}

	for rowKey := range exemptAuditCorpus {
		if !seen[rowKey] {
			t.Errorf("STALE INTENT ROW %q: the audit names an engine the committed corpus no longer declares. "+
				"The record cannot outlive its subject — delete the row.", rowKey)
		}
	}
}

// TestExemptAudit_PackCheckReportsAdvisoryAndExitsZero drives the REAL pack check
// command and locks the exit code: an advisory reports and does NOT block.
//
// This is a GREEN-ON-ARRIVAL pin, not a red-first test. CLM-003's genuine red-first
// falsification is TestExemptDecision_AdvisoryNeverFailsTheRun (pkg/packval),
// observed red before the implementation existed. What this adds is END-TO-END
// reach — the real CLI command and its real exit code, which a package-level test
// on FinalizeStatus cannot cover.
func TestExemptAudit_PackCheckReportsAdvisoryAndExitsZero(t *testing.T) {
	packDir := filepath.Join(repoRoot(t), "cmd", "backstop", "testdata", "exempt-decision-pack")

	out, err := executeCommand(NewRootCommand(), "pack", "check", packDir)
	if err != nil {
		t.Fatalf("pack check returned an error (non-zero exit) on a pack whose ONLY finding is an advisory: %v\noutput:\n%s", err, out)
	}
	if !strings.Contains(out, "status: pass") {
		t.Errorf("reported status is not pass:\n%s", out)
	}
	if !strings.Contains(out, "exempt-scope-decision") {
		t.Errorf("output carries no exempt-scope-decision advisory:\n%s", out)
	}
	if !strings.Contains(out, "mylint") {
		t.Errorf("advisory does not name the fixture's project-wide engine (mylint):\n%s", out)
	}
	// The advisory must render with its phase segment; an empty Phase renders the
	// literal "WARN [/exempt-scope-decision]".
	if !strings.Contains(out, "WARN [phase2-coherence/exempt-scope-decision]") {
		t.Errorf("advisory did not render with its phase segment:\n%s", out)
	}
}
