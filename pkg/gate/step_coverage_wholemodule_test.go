package gate

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// fakeWholeModuleRunner returns a fixed whole-module `go test ./...` output for
// the whole-module command and records how many times it was invoked, so tests
// can assert the suite is consulted once.
type fakeWholeModuleRunner struct {
	out   []byte
	err   error
	calls int
}

func (r *fakeWholeModuleRunner) Run(_ context.Context, _ string, _ ...string) ([]byte, error) {
	r.calls++
	return r.out, r.err
}

func writeGoMod(t *testing.T, dir, module string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module "+module+"\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestPackageLabelFromLine(t *testing.T) {
	const mod = "github.com/bmanson/backstop-core"
	cases := []struct {
		line string
		want string
	}{
		{"ok  \tgithub.com/bmanson/backstop-core/pkg/gate\t1.9s\tcoverage: 88.2% of statements", "./pkg/gate"},
		{"ok  \tgithub.com/bmanson/backstop-core\t0.5s\tcoverage: 91.0% of statements", "."},
		{"ok  \tgithub.com/bmanson/backstop-core/cmd/backstop\t9s\tcoverage: 87.1% of statements", "./cmd/backstop"},
		{"ok  \tgithub.com/other/module/pkg/x\t0.5s\tcoverage: 80.0% of statements", ""},
	}
	for _, tc := range cases {
		if got := packageLabelFromLine(tc.line, mod); got != tc.want {
			t.Errorf("packageLabelFromLine(%q) = %q, want %q", tc.line, got, tc.want)
		}
	}
}

// TestWholeModulePackageCoverage_GreenPath proves the per-package coverage map
// is derived from a SINGLE whole-module run (calls == 1) and keys each changed
// package's self-coverage by its target label — the same numbers a dedicated
// per-package run would report.
func TestWholeModulePackageCoverage_GreenPath(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir, "github.com/bmanson/backstop-core")
	output := "ok  \tgithub.com/bmanson/backstop-core/pkg/gate\t1.9s\tcoverage: 88.2% of statements\n" +
		"ok  \tgithub.com/bmanson/backstop-core/cmd/backstop\t9.0s\tcoverage: 87.1% of statements\n" +
		"?   \tgithub.com/bmanson/backstop-core/pkg/empty\t[no test files]\n"
	runner := &fakeWholeModuleRunner{out: []byte(output)}
	scope := &GateScope{ProjectRoot: dir}

	cov, ok := wholeModulePackageCoverage(context.Background(), runner, scope)
	if !ok {
		t.Fatal("expected coverage map on green path")
	}
	if runner.calls != 1 {
		t.Errorf("whole-module run consulted %d times, want 1", runner.calls)
	}
	if got := cov["./pkg/gate"]; got != 88.2 {
		t.Errorf("./pkg/gate coverage = %v, want 88.2", got)
	}
	if got := cov["./cmd/backstop"]; got != 87.1 {
		t.Errorf("./cmd/backstop coverage = %v, want 87.1", got)
	}
	if _, present := cov["./pkg/empty"]; present {
		t.Error("a [no test files] package must be absent from the map (forces per-package fallback)")
	}
}

// TestWholeModulePackageCoverage_RedPathFallsBack proves a failing/erroring
// whole-module run yields (nil,false) so callers fall back to dedicated
// per-package runs, preserving prior behavior exactly on the red path.
func TestWholeModulePackageCoverage_RedPathFallsBack(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir, "github.com/bmanson/backstop-core")
	runner := &fakeWholeModuleRunner{out: []byte("FAIL\tgithub.com/bmanson/backstop-core/pkg/gate\n"), err: os.ErrClosed}
	scope := &GateScope{ProjectRoot: dir}

	if cov, ok := wholeModulePackageCoverage(context.Background(), runner, scope); ok || cov != nil {
		t.Errorf("red path must return (nil,false), got (%v,%v)", cov, ok)
	}
}
