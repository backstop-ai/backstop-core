package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ISSUE-093 END TO END, THROUGH THE REAL CLI. Phases 2 and 3 fix two halves of one
// command surface independently and each is pinned at the unit level; what is
// unproven until here is that the COMPOSED path behaves. These drive the root
// command (runGateCommand), not the internal helpers.
//
// The fixture pack is SYNTHETIC (`acme/toolchain`) and is shaped like a real
// toolchain pack in the ways that matter: it declares `classification:` globs, and
// its engine declares package_scoped + crash_guard + a project-wide
// project_target. Its engine runs through a PRODUCER script that emits SARIF
// echoing the arguments it received, so a test can observe WHICH package targets
// reached the engine — and whether the engine ran at all.

// issue093Project materializes a fixture project whose pack claims `**/*.acme`
// files only, alongside a NON-claimed file in a directory holding no claimed
// source at all. It returns the project root.
func issue093Project(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	write := func(rel, body string) {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatalf("mkdir for %s: %v", rel, err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}

	write("backstop.yml", "project: issue093-e2e\npacks:\n  acme/toolchain: \"1.0.0\"\n")
	write("backstop.lock", `packs:
  acme/toolchain:
    name: acme/toolchain
    version: 1.0.0
    source_type: local
    local_path: .backstop/packs/acme/toolchain
    install_date: "2026-08-17T00:00:00Z"
`)
	// The artifact corpus must exist or the run exits 2 on a config error that
	// would masquerade as this test's own assertion failing.
	if err := os.MkdirAll(filepath.Join(root, "specs"), 0o755); err != nil {
		t.Fatalf("mkdir specs: %v", err)
	}

	// A CLAIMED file, in its own directory.
	write("pkg/widget/widget.acme", "widget source\n")
	// An UNCLAIMED file, in a directory holding nothing the pack owns — the
	// issue's own reproduction shape.
	write("workflows/deploy.yml", "name: deploy\n")

	write(".backstop/packs/acme/toolchain/pack.yml", `name: acme/toolchain
version: 1.0.0
language: go
archetype: enforcement
description: Synthetic toolchain-shaped pack for the ISSUE-093 end-to-end reproductions.
classification:
  source:
    - "**/*.acme"
  test:
    - "**/*_spec.acme"
engines:
  acme-test:
    command: go test
    producer: scripts/echo-args.sh
    input_mode: none
    scope_kind: project-wide
    project_target: "./..."
    package_scoped: true
    crash_guard: true
    category: mechanism
    gate_type: test
    # Mirrors the real go-toolchain test binding. It is also what keeps the
    # producer's "I ran, with these targets" result VISIBLE: without it the
    # diff/file scope filter drops any finding whose file is outside the scope
    # under test, and this test's whole observability channel would vanish exactly
    # when the scope is the unclaimed file.
    exempt_from_scope_filter: true
content:
  ruleset:
    version: 1.0.0
    rules:
      - id: acme-test-rule
        standard: inline standard text
        risk_class: correctness
        engine: acme-test
        category: mechanism
`)
	// The producer MODELS A REAL PACKAGE-SCOPED TOOLCHAIN PASS rather than merely
	// echoing. Two behaviors matter, and both are what a real tool does:
	//
	//  1. Handed a package target whose directory holds NONE of the pack's source,
	//     it exits non-zero having printed NOTHING. That is precisely the input the
	//     CrashGuard branch turns into "crashed: non-zero exit with no parseable
	//     findings" — so the crash this issue is about genuinely REPRODUCES here
	//     rather than being an assertion that can never fail.
	//  2. Otherwise it emits SARIF whose single result message is the verbatim
	//     argument list, making "which package targets reached the engine"
	//     observable through the real CLI with no runner spy.
	script := filepath.Join(root, ".backstop", "packs", "acme", "toolchain", "scripts", "echo-args.sh")
	if err := os.MkdirAll(filepath.Dir(script), 0o755); err != nil {
		t.Fatalf("mkdir scripts: %v", err)
	}
	body := `#!/bin/sh
for a in "$@"; do
  case "$a" in
    ./*)
      # No source the pack owns under this target: nothing to do, and no output
      # to report it with. A real toolchain fails exactly this way.
      ls "$a"/*.acme >/dev/null 2>&1 || exit 1
      ;;
  esac
done
printf '{"version":"2.1.0","runs":[{"results":[{"ruleId":"acme-invoked","level":"warning","message":{"text":"ENGINE-RAN-WITH: %s"},"locations":[{"physicalLocation":{"artifactLocation":{"uri":"pkg/widget/widget.acme"},"region":{"startLine":1}}}]}]}]}' "$*"
`
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatalf("write producer: %v", err)
	}
	return root
}

// crashSignature is the verbatim wording the CrashGuard branch emits. ISSUE-093's
// whole subject is that a legitimate no-op reached this message.
const crashSignature = "crashed: non-zero exit with no parseable findings"

// TestIssue093_NonGoDirectoryFileDoesNotCrashGate is the issue's PRIMARY
// reproduction through the real CLI (CLM-014): a file-mode scope over a file in a
// directory the dispatching pack owns nothing in. At HEAD the engine is handed
// that directory, finds nothing to do, exits non-zero, and CrashGuard reports the
// no-op as an engine crash.
//
// The assertion is on the ABSENCE OF THE CRASH SPECIFICALLY, not merely on exit
// 0: a future change that suppressed all findings would reach exit 0 while
// leaving the defect in place.
func TestIssue093_NonGoDirectoryFileDoesNotCrashGate(t *testing.T) {
	root := issue093Project(t)

	stdout, stderr, exitCode := runGateCommand(t, root, "--file", "workflows/deploy.yml")
	combined := stdout + stderr

	if strings.Contains(combined, crashSignature) {
		t.Errorf("ISSUE-093 REPRODUCED: a scope the pack claims nothing in still reported an engine crash:\n%s", combined)
	}
	if exitCode != ExitPass {
		t.Errorf("expected exit %d, got %d:\n%s", ExitPass, exitCode, combined)
	}
	// The skip must be LOUD. A silent skip is a smaller lie than a crash, but the
	// operator still reads PASS unable to tell whether the engine ran.
	if !strings.Contains(combined, "was NOT dispatched") {
		t.Errorf("the skip must be REPORTED, not silent — no skip advisory surfaced:\n%s", combined)
	}
	// And the engine genuinely must not have run.
	if strings.Contains(combined, "ENGINE-RAN-WITH") {
		t.Errorf("the engine was DISPATCHED for a scope its pack claims nothing in:\n%s", combined)
	}
}

// TestIssue093_MultipleFileFlagsAllReachTheScope is the COMPOSED property
// (CLM-014): repeatability (CLM-009) and per-file claim-gating (CLM-001) holding
// at the same time. At HEAD this invocation scopes ONE file — the last --file
// wins — and then crashes on it. Both halves must be gone.
func TestIssue093_MultipleFileFlagsAllReachTheScope(t *testing.T) {
	root := issue093Project(t)

	stdout, stderr, exitCode := runGateCommand(t, root,
		"--file", "pkg/widget/widget.acme", "--file", "workflows/deploy.yml")
	combined := stdout + stderr

	// BOTH paths reached the scope. At HEAD the reported count is 1.
	if !strings.Contains(combined, "2 explicit files") {
		t.Errorf("both --file occurrences must reach the scope; the run reported otherwise:\n%s", combined)
	}
	if strings.Contains(combined, crashSignature) {
		t.Errorf("the unclaimed half of the scope still crashed the engine:\n%s", combined)
	}
	if exitCode != ExitPass {
		t.Errorf("expected exit %d, got %d:\n%s", ExitPass, exitCode, combined)
	}

	// The CLAIMED file contributed its package; the UNCLAIMED one contributed
	// nothing. This is the per-file claim gate operating inside a multi-file scope.
	if !strings.Contains(combined, "ENGINE-RAN-WITH") {
		t.Fatalf("the engine must run for the CLAIMED half of the scope:\n%s", combined)
	}
	if !strings.Contains(combined, "./pkg/widget") {
		t.Errorf("the claimed file's package ./pkg/widget must reach the engine:\n%s", combined)
	}
	if strings.Contains(combined, "./workflows") {
		t.Errorf("the UNCLAIMED file contributed a package target ./workflows — the claim gate is per-file and must have dropped it:\n%s", combined)
	}
}
