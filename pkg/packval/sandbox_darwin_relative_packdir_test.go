//go:build darwin

package packval

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ISSUE-216 (and the remaining substance of ISSUE-147): a RELATIVE pack directory
// reached the Seatbelt profile unresolved. `backstop pack test` with no path argument
// defaults packDir to ".", filepath.EvalSymlinks(".") stays ".", and the profile got
// `(subpath ".")`. Seatbelt matches absolute, kernel-resolved paths, so nothing in the
// pack was readable: /bin/sh exited 126 with "Operation not permitted", and phase 3
// reported a bare "layer3 positive failed" on a correct pack.
//
// None of these tests call t.Parallel: three of them change the process working
// directory (t.Chdir), which is process-global.

const relativePackDirMarker = "ISSUE-216-relative-packdir-marker"

// writeRelativePackDirFile writes data to root/rel with the given mode, creating
// parent directories.
func writeRelativePackDirFile(t *testing.T, root, rel, data string, mode os.FileMode) {
	t.Helper()
	full := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", rel, err)
	}
	if err := os.WriteFile(full, []byte(data), mode); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

// makeRelativePackDirPack builds a minimal pack under dir: a validator that prints
// its first argument and a positive fixture holding the marker.
func makeRelativePackDirPack(t *testing.T, dir string) {
	t.Helper()
	writeRelativePackDirFile(t, dir, "validators/check.sh", "#!/bin/sh\ncat \"$1\"\n", 0o755)
	writeRelativePackDirFile(t, dir, "fixtures/pos.txt", relativePackDirMarker+"\n", 0o644)
}

// resolvedAbsPackDir is the path CLM-002 says the sandbox must use for a pack dir:
// filepath.EvalSymlinks(filepath.Abs(dir)).
func resolvedAbsPackDir(t *testing.T, dir string) string {
	t.Helper()
	abs, err := filepath.Abs(dir)
	if err != nil {
		t.Fatalf("filepath.Abs(%q): %v", dir, err)
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		t.Fatalf("filepath.EvalSymlinks(%q): %v", abs, err)
	}
	return resolved
}

// TestSandboxDarwin_RelativePackDirValidatorRuns (CLM-001): a sandboxed run whose
// pack dir is relative can read and run files inside the pack.
func TestSandboxDarwin_RelativePackDirValidatorRuns(t *testing.T) {
	t.Run("dot pack dir from inside the pack", func(t *testing.T) {
		packDir := t.TempDir()
		makeRelativePackDirPack(t, packDir)
		t.Chdir(packDir)

		out, err := SandboxedRun("/bin/sh", []string{"validators/check.sh", "fixtures/pos.txt"}, ".")
		if err != nil {
			t.Fatalf("SandboxedRun with packDir \".\" failed — the Seatbelt profile cannot match a relative "+
				"subpath, so the validator inside the pack is unreadable (ISSUE-216): %v\noutput: %s", err, out)
		}
		if !strings.Contains(string(out), relativePackDirMarker) {
			t.Fatalf("validator ran but did not print the fixture's marker %q; output: %q", relativePackDirMarker, out)
		}
	})

	t.Run("relative subdirectory pack dir from the parent", func(t *testing.T) {
		parent := t.TempDir()
		makeRelativePackDirPack(t, filepath.Join(parent, "pack"))
		t.Chdir(parent)

		out, err := SandboxedRun("/bin/sh", []string{"validators/check.sh", "fixtures/pos.txt"}, "pack")
		if err != nil {
			t.Fatalf("SandboxedRun with relative packDir \"pack\" failed (ISSUE-216): %v\noutput: %s", err, out)
		}
		if !strings.Contains(string(out), relativePackDirMarker) {
			t.Fatalf("validator ran but did not print the fixture's marker %q; output: %q", relativePackDirMarker, out)
		}
	})
}

// TestSandboxDarwin_ProfileEmbedsAbsolutePackDir (CLM-002): the profile always embeds
// the absolute, symlink-resolved pack path, never the relative spelling it was given.
func TestSandboxDarwin_ProfileEmbedsAbsolutePackDir(t *testing.T) {
	cases := []struct {
		name    string
		packDir string
		setup   func(t *testing.T) string // returns the dir to chdir into
	}{
		{
			name:    "dot",
			packDir: ".",
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
		},
		{
			name:    "relative subdirectory",
			packDir: "pack",
			setup: func(t *testing.T) string {
				parent := t.TempDir()
				if err := os.MkdirAll(filepath.Join(parent, "pack"), 0o755); err != nil {
					t.Fatal(err)
				}
				return parent
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Chdir(tc.setup(t))
			want := resolvedAbsPackDir(t, tc.packDir)

			profile, err := darwinSandboxProfile(tc.packDir)
			if err != nil {
				t.Fatalf("darwinSandboxProfile(%q): %v", tc.packDir, err)
			}
			literal, err := seatbeltStringLiteral(want)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(profile, "(subpath "+literal+")") {
				t.Errorf("profile does not embed the absolute resolved pack dir %s:\n%s", literal, profile)
			}
			if strings.Contains(profile, "(subpath \""+tc.packDir+"\")") {
				t.Errorf("profile embeds the relative pack dir %q, which Seatbelt can never match:\n%s", tc.packDir, profile)
			}
		})
	}
}

// TestSandboxDarwin_InvocationDirMatchesResolvedPackDir (CLM-002): the sandbox-exec
// command's Dir and the helper request's Dir are the same resolved absolute path the
// profile embeds.
func TestSandboxDarwin_InvocationDirMatchesResolvedPackDir(t *testing.T) {
	t.Chdir(t.TempDir())
	want := resolvedAbsPackDir(t, ".")

	cmd, ackRead, err := newDarwinSandboxInvocation("/bin/true", nil, ".")
	if err != nil {
		t.Fatalf("newDarwinSandboxInvocation: %v", err)
	}
	t.Cleanup(func() {
		_ = ackRead.Close()
		for _, f := range cmd.ExtraFiles {
			_ = f.Close()
		}
	})

	if cmd.Dir != want {
		t.Errorf("sandbox-exec cmd.Dir = %q, want the resolved absolute pack dir %q", cmd.Dir, want)
	}

	prefix := sandboxHelperEnvVar + "="
	var encoded string
	for _, kv := range cmd.Env {
		if strings.HasPrefix(kv, prefix) {
			encoded = strings.TrimPrefix(kv, prefix)
		}
	}
	if encoded == "" {
		t.Fatalf("no %s entry in the sandbox-exec environment", sandboxHelperEnvVar)
	}
	var request sandboxHelperRequest
	if err := json.Unmarshal([]byte(encoded), &request); err != nil {
		t.Fatalf("decode helper request: %v", err)
	}
	if request.Dir != want {
		t.Errorf("helper request Dir = %q, want the resolved absolute pack dir %q", request.Dir, want)
	}
	if request.Dir != cmd.Dir {
		t.Errorf("helper request Dir %q and sandbox-exec cmd.Dir %q disagree", request.Dir, cmd.Dir)
	}
}

// TestSandboxDarwin_RunValidatorReportsRealExitCode (CLM-003): a failed validator run
// reports its real exit status, not a flattened 1, and puts the run error text in
// the output. The contract phase 3 relies on (nil error, Passed=false) is unchanged.
func TestSandboxDarwin_RunValidatorReportsRealExitCode(t *testing.T) {
	packDir := mustEvalSymlinks(t, t.TempDir())
	writeRelativePackDirFile(t, packDir, "validators/exit3.sh", "#!/bin/sh\necho validator-said-no\nexit 3\n", 0o755)
	writeRelativePackDirFile(t, packDir, "fixtures/pos.txt", relativePackDirMarker+"\n", 0o644)

	r, err := (&DefaultExecutor{}).RunValidator(packDir, "validators/exit3.sh", []string{"fixtures/pos.txt"})
	if err != nil {
		t.Fatalf("RunValidator must keep its nil-error contract on a failed run; got %v", err)
	}
	if r.Passed {
		t.Fatalf("a validator that exits 3 must report Passed=false; got %+v", r)
	}
	if r.ExitCode != 3 {
		t.Errorf("ExitCode = %d, want the validator's real exit status 3 (not a flattened 1); output: %q", r.ExitCode, r.Output)
	}
	if !strings.Contains(r.Output, "exit status 3") {
		t.Errorf("Output must carry the run error text (\"exit status 3\"); got %q", r.Output)
	}
	if !strings.Contains(r.Output, "validator-said-no") {
		t.Errorf("Output must still carry the validator's captured output; got %q", r.Output)
	}
}
