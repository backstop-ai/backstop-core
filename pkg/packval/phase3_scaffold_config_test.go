package packval_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/packval"
)

// TestRunFixtures_CompleteScaffoldWritesSampleConfigIntoPackDir characterizes the
// mutation SPEC-056 REQ-008 exists to CONTAIN (CLM-080): phase 3 renders every
// tier:complete scaffold's declared sample_config entries into
// <packDir>/<scaffold.path>/<relPath> before running that scaffold's test command.
//
// THIS IS A CHARACTERIZATION TEST, NOT A REGRESSION TEST. The behavior it pins is
// CORRECT for packval — rendering the sample config is how a complete scaffold's test
// command gets something to run against — and WRONG for anything that hashes the
// directory afterward. Nothing in this spec changes the rendering. What changes is that
// no command hands packval a directory it is about to copy and hash: add, update and
// upgrade all move validation onto a scratch copy.
//
// WHAT THIS TEST IS FOR. If phase 3's rendering ever moves or stops, REQ-008's whole
// scratch-copy seam becomes decorative — and CLM-081 through CLM-090 would keep passing,
// because a seam that contains a mutation which no longer happens passes trivially. This
// test is the thing that reds instead, naming the reason directly.
//
// NO SUBPROCESS IS SPAWNED. The MockExecutor's ScaffoldTestFn returns Passed, and the
// rendering at phase3.go:145-169 runs strictly BEFORE the RunScaffoldTest call at :173,
// so the assertion holds without any shell.
func TestRunFixtures_CompleteScaffoldWritesSampleConfigIntoPackDir(t *testing.T) {
	const (
		scaffoldDir     = "scaffold"
		sampleConfigRel = "rendered-settings.yml"
		sampleContent   = "marker: rendered by packval phase 3\n"
	)

	packDir := t.TempDir()
	// The scaffold's path must exist: RunScaffoldTest uses it as the subprocess working
	// directory, and phase 1 stats it. The sample_config TARGET must not — an authored
	// file there would make a rendered one indistinguishable from it.
	if err := os.MkdirAll(filepath.Join(packDir, scaffoldDir), 0o755); err != nil {
		t.Fatalf("creating the scaffold directory: %v", err)
	}
	target := filepath.Join(packDir, scaffoldDir, sampleConfigRel)
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("the sample_config target must be absent before the run, or this test cannot attribute the file to packval (stat error: %v)", err)
	}

	pack := &packval.PackManifest{
		Name:      "acme/scaffold-config-pack",
		Version:   "1.0.0",
		Language:  "neutral",
		Archetype: "code",
		Content: packval.Content{
			Scaffolds: []packval.Scaffold{{
				ID:           "config-scaffold",
				Tier:         "complete",
				Path:         scaffoldDir,
				TestCommand:  ":",
				SampleConfig: map[string]string{sampleConfigRel: sampleContent},
			}},
		},
	}

	executor := &packval.MockExecutor{
		ScaffoldTestFn: func(_, _, _ string) (packval.ExecutionResult, error) {
			return packval.ExecutionResult{Passed: true, ExitCode: 0}, nil
		},
	}

	res := packval.RunFixtures(pack, packDir, executor)
	if res == nil {
		t.Fatal("RunFixtures returned nil")
	}
	if res.Status != "pass" {
		t.Fatalf("phase 3 status = %q, want pass; errors: %+v", res.Status, res.Errors)
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("phase 3 did not render the sample_config entry to %s: %v — if the rendering moved, REQ-008's scratch-copy seam no longer contains anything and CLM-081..090 pass vacuously",
			target, err)
	}
	if string(data) != sampleContent {
		t.Errorf("rendered %s = %q, want the declared contents %q", target, string(data), sampleContent)
	}
}
