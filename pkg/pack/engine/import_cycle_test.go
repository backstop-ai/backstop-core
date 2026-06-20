package engine_test

import (
	"os/exec"
	"strings"
	"testing"
)

// TestEngineBinding_NoImportCycle proves pkg/pack/engine is a leaf package
// importable by pkg/check, pkg/packval, and cmd/backstop without an import cycle
// (CLM-033 / REQ-013): the engine package must transitively import NONE of those
// three. Substantive: it resolves the engine package's real transitive import
// set via `go list -deps` and asserts the forbidden packages are absent —
// proving the leaf invariant structurally, not by a doc comment.
func TestEngineBinding_NoImportCycle(t *testing.T) {
	enginePkg := "github.com/bmanson/backstop-core/pkg/pack/engine"
	forbidden := []string{
		"github.com/bmanson/backstop-core/pkg/check",
		"github.com/bmanson/backstop-core/pkg/packval",
		"github.com/bmanson/backstop-core/cmd/backstop",
	}

	out, err := exec.Command("go", "list", "-deps", enginePkg).Output()
	if err != nil {
		t.Fatalf("go list -deps %s: %v", enginePkg, err)
	}
	deps := map[string]struct{}{}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		deps[strings.TrimSpace(line)] = struct{}{}
	}

	// Sanity: the dependency list is non-empty (the command really resolved deps),
	// so an empty result cannot vacuously "prove" the absence.
	if len(deps) == 0 {
		t.Fatal("go list -deps returned no dependencies; cannot verify the leaf invariant")
	}

	for _, f := range forbidden {
		if _, found := deps[f]; found {
			t.Errorf("pkg/pack/engine must be a leaf package but transitively imports %q; this would reintroduce the import cycle REQ-013 exists to prevent", f)
		}
	}
}
