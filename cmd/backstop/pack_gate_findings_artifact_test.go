package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack"
	"github.com/bmanson/backstop-core/pkg/pack/engine"
)

// SPEC-048 REQ-002 — the stdout_artifact payload-selection matrix over the REAL
// runFindingsEngine (plus the REQ-004a runCoverageEngine %w mirror). These pin the
// DEFECT-2 fix: when the binding declares a stdout_artifact, the FILE's bytes (not
// the engine's noise stdout) are the payload fed to BOTH the convert and the
// strict-SARIF shape guard; a declared-but-missing artifact fail-louds; an empty
// StdoutArtifact keeps stdout (backward compatible). The two engines' payload
// blocks stay behaviorally mirrored (both %w-wrapped, projectRoot-relative).

// artifactSarifWithFinding is a valid SARIF log carrying one finding — the real
// machine-readable output an engine writes to its stdout_artifact FILE.
func artifactSarifWithFinding() string {
	return `{"version":"2.1.0","runs":[{"tool":{"driver":{"name":"fake"}},"results":[{"ruleId":"seeded-defect",` +
		`"level":"error","message":{"text":"seeded defect in the artifact file"},` +
		`"locations":[{"physicalLocation":{"artifactLocation":{"uri":"src/app.ts"},"region":{"startLine":1}}}]}]}]}`
}

// noiseStdout is human-summary NOISE with no `runs` array — the misleading stdout a
// stdout-fed convert/shape-guard would read as zero findings (DEFECT-2).
func noiseStdout() string {
	return `{"summary":"scan complete; findings written to the artifact file","errors":0}`
}

// writeConvertEcho writes a pack-relative convert script that echoes its stdin
// verbatim, so the payload the dispatch feeds the convert surfaces unchanged.
func writeConvertEcho(t *testing.T, packRoot string) string {
	t.Helper()
	writeFileStr(t, filepath.Join(packRoot, "scripts", "convert.sh"), "#!/bin/sh\ncat\n")
	return "scripts/convert.sh"
}

// TestRunFindingsEngine_StdoutArtifactFileFeedsConvertNotStdout proves the DEFECT-2
// fix on the convert path (CLM-005): with StdoutArtifact set and the file PRESENT,
// the FILE's bytes (not the noise stdout) are fed to the convert — a findings-bearing
// artifact over a finding-free stdout yields the file's findings.
func TestRunFindingsEngine_StdoutArtifactFileFeedsConvertNotStdout(t *testing.T) {
	projectRoot := t.TempDir()
	packRoot := t.TempDir()
	convert := writeConvertEcho(t, packRoot)

	var convertStdin []byte
	stubSandboxedRunStdout(t, &convertStdin)

	artifactBody := artifactSarifWithFinding()
	runner := &artifactWritingRunner{
		dir:          projectRoot,
		artifactName: "findings.sarif",
		artifactBody: artifactBody,
		stdoutNoise:  noiseStdout(),
	}
	binding := engine.EngineBinding{
		Command:        "fake run",
		InputMode:      engine.InputModeNone,
		ScopeKind:      engine.ScopeKindProjectWide,
		Convert:        convert,
		StdoutArtifact: "findings.sarif",
	}

	violations, err := runFindingsEngine(&pack.Manifest{NormalizedName: "test-org/artifact"}, packRoot, projectRoot, nil, binding, nil, runner, newSharedRunCache())
	if err != nil {
		t.Fatalf("runFindingsEngine: %v", err)
	}
	if string(convertStdin) != artifactBody {
		t.Errorf("DEFECT-2: convert must be fed the stdout_artifact FILE, not stdout.\n got: %q\nwant: %q", string(convertStdin), artifactBody)
	}
	if len(violations) != 1 {
		t.Fatalf("the artifact FILE's one finding must survive to the consumer, got %d: %#v", len(violations), violations)
	}
	if !strings.Contains(violations[0].Message, "seeded defect in the artifact file") {
		t.Errorf("the surviving finding must come from the artifact FILE, got %#v", violations[0])
	}
}

// TestRunFindingsEngine_StdoutArtifactFileFeedsStrictSarifShapeGuard proves payload
// selection applies to the shape-guard path too (CLM-006): with StdoutArtifact set,
// the file PRESENT, and NO convert declared, the FILE's bytes (not the noise stdout)
// are what requireLintSarifShape validates.
func TestRunFindingsEngine_StdoutArtifactFileFeedsStrictSarifShapeGuard(t *testing.T) {
	projectRoot := t.TempDir()
	packRoot := t.TempDir()

	artifactBody := artifactSarifWithFinding()
	runner := &artifactWritingRunner{
		dir:          projectRoot,
		artifactName: "findings.sarif",
		artifactBody: artifactBody,
		stdoutNoise:  noiseStdout(), // non-SARIF: fails the shape guard if fed
	}
	binding := engine.EngineBinding{
		Command:        "fake run",
		InputMode:      engine.InputModeNone,
		ScopeKind:      engine.ScopeKindProjectWide,
		StrictSarif:    true, // no convert: the shape guard runs over the payload
		StdoutArtifact: "findings.sarif",
	}

	violations, err := runFindingsEngine(&pack.Manifest{NormalizedName: "test-org/artifact"}, packRoot, projectRoot, nil, binding, nil, runner, newSharedRunCache())
	if err != nil {
		t.Fatalf("the shape guard must validate the SARIF artifact FILE (not the noise stdout), got err: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("the artifact FILE's finding must parse through the shape-guard path, got %d: %#v", len(violations), violations)
	}
	if !strings.Contains(violations[0].Message, "seeded defect in the artifact file") {
		t.Errorf("the finding must come from the artifact FILE, got %#v", violations[0])
	}
}

// TestRunFindingsEngine_StdoutArtifactMissingFailsLoud proves the no-silent-fallback
// guard (CLM-007): with StdoutArtifact set but the file MISSING, runFindingsEngine
// fails loud — an error naming the pack, the engine command, the declared artifact,
// and the resolved path — and does NOT fall back to reading stdout.
func TestRunFindingsEngine_StdoutArtifactMissingFailsLoud(t *testing.T) {
	projectRoot := t.TempDir()
	packRoot := t.TempDir()

	// emptySarifRunner returns a clean SARIF on stdout and writes NO artifact file —
	// so a silent stdout fallback would read green; the fix must fail loud instead.
	binding := engine.EngineBinding{
		Command:        "fake run",
		InputMode:      engine.InputModeNone,
		ScopeKind:      engine.ScopeKindProjectWide,
		StdoutArtifact: "missing.sarif",
	}

	_, err := runFindingsEngine(&pack.Manifest{NormalizedName: "test-org/artifact"}, packRoot, projectRoot, nil, binding, nil, emptySarifRunner{}, newSharedRunCache())
	if err == nil {
		t.Fatal("a declared-but-missing stdout_artifact must fail loud, not silently fall back to stdout (DEFECT-2)")
	}
	msg := err.Error()
	for _, want := range []string{"test-org/artifact", "fake run", "missing.sarif", filepath.Join(projectRoot, "missing.sarif")} {
		if !strings.Contains(msg, want) {
			t.Errorf("the fail-loud error must name %q; got: %v", want, msg)
		}
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the fail-loud error must wrap the underlying read error (%%w) so it is unwrappable; got: %v", err)
	}
}

// TestRunFindingsEngine_NoStdoutArtifactUsesStdoutPayload proves backward
// compatibility (CLM-008): with StdoutArtifact EMPTY, the payload remains the
// engine's stdout unchanged — the default findings path is undisturbed.
func TestRunFindingsEngine_NoStdoutArtifactUsesStdoutPayload(t *testing.T) {
	projectRoot := t.TempDir()
	packRoot := t.TempDir()

	runner := &capturingRunner{out: []byte(artifactSarifWithFinding())}
	binding := engine.EngineBinding{
		Command:        "fake run",
		InputMode:      engine.InputModeNone,
		ScopeKind:      engine.ScopeKindProjectWide,
		StdoutArtifact: "", // no artifact: keep stdout as the payload
	}

	violations, err := runFindingsEngine(&pack.Manifest{NormalizedName: "test-org/artifact"}, packRoot, projectRoot, nil, binding, nil, runner, newSharedRunCache())
	if err != nil {
		t.Fatalf("runFindingsEngine: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("with no stdout_artifact, the engine's stdout is the payload; its one finding must parse, got %d: %#v", len(violations), violations)
	}
	if !strings.Contains(violations[0].Message, "seeded defect in the artifact file") {
		t.Errorf("the finding must come from the stdout payload, got %#v", violations[0])
	}
}

// TestRunCoverageEngine_StdoutArtifactMissingWrapsReadError proves the REQ-004a
// mirror (CLM-012): runCoverageEngine's stdout_artifact-missing branch wraps the
// underlying read error with %w, so errors.Is / errors.Unwrap reaches the
// os.ReadFile error — matching the findings-path fail-loud branch. RED pre-fix (%v
// breaks the chain).
func TestRunCoverageEngine_StdoutArtifactMissingWrapsReadError(t *testing.T) {
	projectRoot := t.TempDir()
	packRoot := t.TempDir()

	binding := engine.EngineBinding{
		Command:        "cov run",
		InputMode:      engine.InputModeNone,
		ScopeKind:      engine.ScopeKindProjectWide,
		Convert:        "scripts/convert.sh",
		StdoutArtifact: "missing-cover.out",
	}

	// emptySarifRunner writes no artifact file, so the declared stdout_artifact is
	// missing and the fail-loud branch fires before the convert.
	_, err := runCoverageEngine(&pack.Manifest{NormalizedName: "test-org/coverage"}, packRoot, projectRoot, binding, nil, emptySarifRunner{}, newSharedRunCache())
	if err == nil {
		t.Fatal("a declared-but-missing coverage stdout_artifact must fail loud")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the missing-artifact error must wrap the os.ReadFile error with %%w (errors.Is reaches os.ErrNotExist); got: %v", err)
	}
}
