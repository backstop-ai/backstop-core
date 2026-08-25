package check

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestSandboxAuthorization_SanitizerPreservesUnrelatedEnvironment(t *testing.T) {
	t.Setenv("BACKSTOP_PACK_SANDBOX", "ambient")
	tests := []struct {
		name        string
		environment []string
		remove      []string
		want        []string
	}{
		{
			name:        "duplicates empty values and similar prefixes",
			environment: []string{"FIRST=one", "BACKSTOP_PACK_SANDBOX=external", "KEEP=first", "BACKSTOP_PACK_SANDBOX=", "BACKSTOP_PACK_SANDBOX_EXTRA=keep", "KEEP=second"},
			remove:      []string{"BACKSTOP_PACK_SANDBOX"},
			want:        []string{"FIRST=one", "KEEP=first", "BACKSTOP_PACK_SANDBOX_EXTRA=keep", "KEEP=second"},
		},
		{
			name:        "multiple requested names",
			environment: []string{"ONE=1", "TWO=2", "THREE=3", "ONE=again"},
			remove:      []string{"ONE", "THREE"},
			want:        []string{"TWO=2"},
		},
		{
			name:        "no matches still returns fresh result",
			environment: []string{"KEEP=one", "KEEP=two"},
			remove:      []string{"ABSENT"},
			want:        []string{"KEEP=one", "KEEP=two"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			before := append([]string(nil), test.environment...)
			got := WithoutEnvironment(test.environment, test.remove...)
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("WithoutEnvironment() = %#v, want %#v", got, test.want)
			}
			if !reflect.DeepEqual(test.environment, before) {
				t.Fatalf("input mutated: %#v, want %#v", test.environment, before)
			}
			if len(got) > 0 && len(test.environment) > 0 {
				got[0] = "MUTATED=result"
				if !reflect.DeepEqual(test.environment, before) {
					t.Fatalf("result aliases input: %#v", test.environment)
				}
			}
		})
	}

	if got := os.Getenv("BACKSTOP_PACK_SANDBOX"); got != "ambient" {
		t.Fatalf("ambient environment mutated: %q", got)
	}
}

func TestSandboxAuthorization_NoCheckToPackvalDependencyCycle(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture is POSIX-only")
	}

	runner := &ExecCommandRunner{Env: []string{"RUNNER_SENTINEL=explicit"}}
	combined, err := runner.Run(context.Background(), "/bin/sh", "-c", `printf '%s' "$RUNNER_SENTINEL"; printf '%s' "$RUNNER_SENTINEL" >&2`)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := string(combined); got != "explicitexplicit" {
		t.Fatalf("Run output = %q, want explicit environment on stdout and stderr", got)
	}

	stdout, err := runner.RunStdout(context.Background(), "/bin/sh", "-c", `printf '%s' "$RUNNER_SENTINEL"; printf ignored >&2`)
	if err != nil {
		t.Fatalf("RunStdout: %v", err)
	}
	if got := string(stdout); got != "explicit" {
		t.Fatalf("RunStdout output = %q, want explicit", got)
	}

	source, err := os.ReadFile(filepath.Join("runner.go"))
	if err != nil {
		t.Fatalf("read runner.go: %v", err)
	}
	if strings.Contains(string(source), "pkg/packval") {
		t.Fatal("pkg/check must not import pkg/packval")
	}
}

func TestRunner_EnvironmentAndDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture is POSIX-only")
	}
	t.Setenv("RUNNER_INHERITED", "parent")

	dir := t.TempDir()
	resolvedDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("resolve temp directory: %v", err)
	}
	runner := &ExecCommandRunner{Dir: dir, Env: []string{"RUNNER_EXPLICIT=child"}}
	for name, run := range map[string]func(context.Context, string, ...string) ([]byte, error){
		"Run":       runner.Run,
		"RunStdout": runner.RunStdout,
	} {
		t.Run(name, func(t *testing.T) {
			out, err := run(context.Background(), "/bin/sh", "-c", `printf '%s|%s|%s' "$PWD" "$RUNNER_EXPLICIT" "$RUNNER_INHERITED"`)
			if err != nil {
				t.Fatalf("run: %v", err)
			}
			if got, want := string(out), resolvedDir+"|child|"; got != want {
				t.Fatalf("output = %q, want %q", got, want)
			}
		})
	}

	inherited := &ExecCommandRunner{}
	out, err := inherited.RunStdout(context.Background(), "/bin/sh", "-c", `printf '%s' "$RUNNER_INHERITED"`)
	if err != nil {
		t.Fatalf("inherited RunStdout: %v", err)
	}
	if got := string(out); got != "parent" {
		t.Fatalf("nil Env inherited value = %q, want parent", got)
	}
}
