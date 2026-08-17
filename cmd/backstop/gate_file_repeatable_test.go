package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ISSUE-093 — `--file` is REPEATABLE and ACCUMULATES. At HEAD the flag is
// registered with pflag's StringVar, which OVERWRITES rather than accumulates, so
// `--file a --file b` silently scoped ONE file while the summary line was the only
// tell. Making it repeatable forces an explicit decision about EMPTY entries,
// which is settled here as a config error rather than left as a new silent hole.
//
// The flag is a StringARRAYVar, never a StringSliceVar: StringSliceVar splits on
// commas, so a path containing one would be silently shredded into two
// nonexistent paths — trading the old bug for a subtler new one.

// gateFlagsFor parses argv against a fresh gate command and returns the command
// so a test can read the parsed flag values and positional args.
func gateFlagsFor(t *testing.T, argv ...string) (fileFlags, args []string) {
	t.Helper()
	cmd := newGateCommand(new(bool))
	if err := cmd.ParseFlags(argv); err != nil {
		t.Fatalf("parse %v: %v", argv, err)
	}
	values, err := cmd.Flags().GetStringArray("file")
	if err != nil {
		t.Fatalf("read --file as a string array: %v", err)
	}
	return values, cmd.Flags().Args()
}

// runGateInScratchProject runs runGate against a throwaway project so a missing
// guard falls through to a scope computation over an EMPTY tree rather than over
// the real repository. The distinction matters: at HEAD an empty `--file` value
// silently becomes a diff-scoped sweep, and this harness makes that outcome cheap
// to observe instead of a multi-minute real gate run.
func runGateInScratchProject(t *testing.T, args []string, argv ...string) error {
	t.Helper()
	projectRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectRoot, "backstop.yml"), []byte("project: gate-file-repeatable\nlanguage: go\n"), 0o644); err != nil {
		t.Fatalf("write backstop.yml: %v", err)
	}
	orig, _ := os.Getwd()
	if err := os.Chdir(projectRoot); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })

	cmd := newGateCommand(new(bool))
	if err := cmd.ParseFlags(argv); err != nil {
		t.Fatalf("parse %v: %v", argv, err)
	}
	return runGate(cmd, args)
}

// TestGate_FileFlagIsRepeatableAndAccumulates is THE SECONDARY DEFECT (CLM-009).
// At HEAD only the LAST occurrence survives, because pflag's StringVar overwrites.
// The assertion is on the exact ordered slice, not on length: a two-element slice
// holding the same value twice would pass a length check while still having
// dropped an occurrence.
func TestGate_FileFlagIsRepeatableAndAccumulates(t *testing.T) {
	fileFlags, _ := gateFlagsFor(t, "--file", "a.go", "--file", "b.go")
	want := []string{"a.go", "b.go"}
	if len(fileFlags) != len(want) {
		t.Fatalf("--file must ACCUMULATE across occurrences; want %v, got %v", want, fileFlags)
	}
	for i := range want {
		if fileFlags[i] != want[i] {
			t.Fatalf("--file values must survive IN ORDER; want %v, got %v", want, fileFlags)
		}
	}

	// A comma in a path must NOT split the value: that is the StringSliceVar
	// failure mode, a silent corruption replacing a silent drop.
	commaFlags, _ := gateFlagsFor(t, "--file", "pkg/a,b/c.go")
	if len(commaFlags) != 1 || commaFlags[0] != "pkg/a,b/c.go" {
		t.Errorf("a path containing a comma must survive as ONE value (StringArrayVar, not StringSliceVar); got %v", commaFlags)
	}
}

// TestGate_RepeatedFileFlagAndPositionalArgsBothAccumulate is the NON-REGRESSION
// half (CLM-010): the shape that already worked — one flag plus trailing
// positional paths — keeps working, and MIXING the two forms accumulates both, in
// flags-then-args order. The assertion is on the COMPOSED list runGate builds,
// not just on the flag values, because that composition is what reaches the scope.
func TestGate_RepeatedFileFlagAndPositionalArgsBothAccumulate(t *testing.T) {
	fileFlags, args := gateFlagsFor(t, "--file", "a.go", "--file", "b.go", "c.go", "d.go")

	composed := append(append([]string{}, fileFlags...), args...)
	want := []string{"a.go", "b.go", "c.go", "d.go"}
	if strings.Join(composed, " ") != strings.Join(want, " ") {
		t.Fatalf("flag values and trailing positionals must BOTH reach the scope, flags first; want %v, got %v (flags=%v args=%v)", want, composed, fileFlags, args)
	}
}

// TestGate_EmptyFileValueIsConfigError settles DEFECT-3 (CLM-011). At HEAD
// runGate keys file-mode on `fileValue != ""`, so `--file ""` falls THROUGH to a
// diff-scoped sweep: the operator asked for one file and got the whole changed
// set, with only the summary line as the tell. Repeatability forces the question,
// so it is answered here — an empty entry is a config error, never a silent
// fall-through and never a quiet drop.
//
// The assertion is POSITIVE about which failure occurred: "some error happened"
// would not distinguish the fix from an unrelated failure.
func TestGate_EmptyFileValueIsConfigError(t *testing.T) {
	for _, tc := range []struct {
		name string
		argv []string
	}{
		{"the only entry is empty", []string{"--file", ""}},
		{"one bad entry among good ones", []string{"--file", "a.go", "--file", ""}},
		{"a whitespace-only entry", []string{"--file", "   "}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := runGateInScratchProject(t, nil, tc.argv...)
			var exitErr *ExitCodeError
			if !errors.As(err, &exitErr) {
				t.Fatalf("an empty --file value must be a config error, not a silent diff-scoped sweep; got %T %v", err, err)
			}
			if exitErr.Code != ExitConfigError {
				t.Fatalf("expected exit %d (config error), got %d: %s", ExitConfigError, exitErr.Code, exitErr.Message)
			}
			if !strings.Contains(exitErr.Message, "--file") {
				t.Errorf("the message must name the offending flag; got %q", exitErr.Message)
			}
			if !strings.Contains(strings.ToLower(exitErr.Message), "empty") {
				t.Errorf("the message must name the EMPTY value as the problem; got %q", exitErr.Message)
			}
			// A dropped-empty implementation would silently enter diff scope and
			// report a changed-file sweep instead of refusing.
			if strings.Contains(exitErr.Message, "changed files") {
				t.Errorf("the run entered DIFF SCOPE instead of refusing: %q", exitErr.Message)
			}
		})
	}
}

// TestGate_RepeatableFileStillMutuallyExclusiveWithAllAndBase pins that the
// flag-type change did not invert or drop either exclusion (CLM-012). The two
// guards that reference the file flag move from a non-empty STRING test to a
// non-empty LIST test; their MESSAGES are asserted verbatim so the rewrite cannot
// quietly reword a contract other callers read.
func TestGate_RepeatableFileStillMutuallyExclusiveWithAllAndBase(t *testing.T) {
	for _, tc := range []struct {
		name    string
		argv    []string
		message string
	}{
		{"--all with --file", []string{"--all", "--file", "a.go"}, "--all and --file are mutually exclusive"},
		{"--base with --file", []string{"--base", "HEAD", "--file", "a.go"}, "--base and --file are mutually exclusive"},
		{"--all with a REPEATED --file", []string{"--all", "--file", "a.go", "--file", "b.go"}, "--all and --file are mutually exclusive"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := runGateInScratchProject(t, nil, tc.argv...)
			var exitErr *ExitCodeError
			if !errors.As(err, &exitErr) {
				t.Fatalf("expected ExitCodeError, got %T %v", err, err)
			}
			if exitErr.Code != ExitConfigError {
				t.Fatalf("expected exit %d, got %d: %s", ExitConfigError, exitErr.Code, exitErr.Message)
			}
			if !strings.Contains(exitErr.Message, tc.message) {
				t.Errorf("guard message must be UNCHANGED by the flag-type rewrite; want %q in %q", tc.message, exitErr.Message)
			}
		})
	}
}
