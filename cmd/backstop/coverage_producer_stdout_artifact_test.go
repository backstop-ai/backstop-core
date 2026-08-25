package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack"
	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
)

// artifactWritingRunner simulates a toolchain coverage engine like
// `go test -coverprofile=cover.out ./...`: its real output (the profile/records)
// is written to a declared FILE in the working dir, while its STDOUT carries only
// noise (a test SUMMARY and, on failing packages, test-failure lines that look
// like "foo_test.go:42:"). It is the faithful model of the SPEC-042 producer
// anomaly: the convert must be fed the FILE, never this stdout.
type artifactWritingRunner struct {
	dir          string // where the artifact file is written (the runner's Dir)
	artifactName string // the declared stdout_artifact filename
	artifactBody string // the real engine output written to the file
	stdoutNoise  string // the misleading stdout (summary + failure lines)
}

func (r *artifactWritingRunner) Run(context.Context, string, ...string) ([]byte, error) {
	return nil, nil
}

func (r *artifactWritingRunner) RunStdout(_ context.Context, _ string, _ ...string) ([]byte, error) {
	// Write the real output to the declared artifact file, exactly as
	// `go test -coverprofile=<file>` does — to the runner's working dir.
	_ = os.WriteFile(filepath.Join(r.dir, r.artifactName), []byte(r.artifactBody), 0o644)
	// Return only the misleading stdout.
	return []byte(r.stdoutNoise), nil
}

// coverageStdoutArtifactManifest builds a manifest with a coverage engine that
// declares a stdout_artifact (the file its real output lands in) plus a convert
// that echoes its stdin, so the test can observe WHICH bytes the producer fed the
// convert.
func coverageStdoutArtifactManifest(engineName, artifactName string) *pack.Manifest {
	binding := engineBindingForGateType(engine.GateTypeCoverage)
	binding.StdoutArtifact = artifactName
	return &pack.Manifest{
		NormalizedName: "test-org/coverage-routing",
		Engines: map[string]pack.EngineSpec{
			engineName: {Binding: binding},
		},
		Content: pack.Content{Ruleset: pack.Ruleset{Rules: []pack.Rule{
			{ID: engineName + "-rule", Engine: engineName, Standard: "x"},
		}}},
	}
}

// TestCoverageProducer_FeedsConvertDeclaredArtifactNotCommandStdout proves the
// coverage producer pipes the DECLARED stdout_artifact FILE (the engine's real
// output) into the convert — NOT the command's stdout, which for a
// `-coverprofile`-style engine is only the test summary/failure noise.
//
// This reproduces the SPEC-042 producer bug: before the fix, the producer fed the
// command's stdout (noise) to the convert, so a changed source file got NO
// coverage record and the gate falsely red-flagged it as "no coverage
// measurement". The fixture's artifact body carries a real record for
// terminal.go; the stdout noise carries a misleading "<file>_test.go:NN:" line
// that the convert would mis-read as a bogus record if fed stdout.
func TestCoverageProducer_FeedsConvertDeclaredArtifactNotCommandStdout(t *testing.T) {
	var convertStdin []byte
	sandboxRunner := directConvertSandboxRunner(&convertStdin)

	// The convert echoes whatever it receives on stdin as the normalized records.
	// (The artifact body is already valid coverage-records JSON.)
	packsDir := coverageRoutingEchoPacksDir(t)

	artifactBody := `[{"path":"pkg/validate/terminal.go","covered":40,"total":42,"measured":true,"excluded":false,"metric":"statement"}]`
	// Misleading stdout: a test SUMMARY plus a failure line that LOOKS like a
	// profile-bearing ".go:" line — the exact noise that made the old path emit a
	// bogus record and drop the real one.
	stdoutNoise := "ok  \tpkg/validate\t(cached)\tcoverage: 96.2% of statements\n" +
		"--- FAIL: TestFoo\n    smoke_test.go:384: boom\n"

	projectRoot := t.TempDir()
	runner := &artifactWritingRunner{
		dir:          projectRoot,
		artifactName: "cover.out",
		artifactBody: artifactBody,
		stdoutNoise:  stdoutNoise,
	}
	manifest := coverageStdoutArtifactManifest("cov-engine", "cover.out")

	result, err := dispatchPackCoverageWithEvidence([]*pack.Manifest{manifest}, packsDir, projectRoot, nil, runner, sandboxRunner)
	if err != nil {
		t.Fatalf("dispatchPackCoverage: %v", err)
	}

	// The convert must have been fed the ARTIFACT FILE body, not the stdout noise.
	if string(convertStdin) != artifactBody {
		t.Errorf("convert was fed the wrong stream.\n got: %q\nwant: %q (the declared stdout_artifact file)", string(convertStdin), artifactBody)
	}

	// The real record for the changed source file must survive to the consumer.
	found := false
	for _, r := range result.Records {
		if r.Path == "pkg/validate/terminal.go" {
			found = true
			if r.Total != 42 || r.Covered != 40 {
				t.Errorf("terminal.go record mangled: %#v", r)
			}
		}
		if r.Path == "smoke_test.go" {
			t.Errorf("a bogus record from stdout noise leaked through: %#v", r)
		}
	}
	if !found {
		t.Fatalf("the producer must surface a coverage record for the changed file from the declared artifact; got %#v", result.Records)
	}
}

// coverageRoutingEchoPacksDir returns a scratch .backstop/packs dir whose convert
// script ECHOES its stdin verbatim (so the test observes exactly which bytes the
// producer fed the convert).
func coverageRoutingEchoPacksDir(t *testing.T) string {
	t.Helper()
	packsDir := t.TempDir()
	packRoot := filepath.Join(packsDir, "test-org", "coverage-routing")
	mkDirAll(t, filepath.Join(packRoot, "scripts"))
	// Echo stdin straight through as the normalized records output.
	writeFileStr(t, filepath.Join(packRoot, "scripts", "convert.sh"), "#!/bin/sh\ncat\n")
	return packsDir
}
