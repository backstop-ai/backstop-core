package distribution_test

import (
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack/distribution"
)

// The REQ-005 accessor suite. Every remote operation on an ALREADY-LOCKED pack must
// resolve its repository from the recorded source_coordinate rather than from the pack
// name — otherwise the moment REQ-003 keys the lock by manifest name, a divergent-name
// pack becomes uninstallable from its own lock.
//
// There is exactly ONE accessor and the URL builder is LAYERED on it. That matters
// because the two consumers need different things from the same decision:
// TagVersionResolver builds its own URL from a coordinate while the clone paths need a
// URL, so pack update resolves the coordinate twice in one invocation. Two independent
// resolutions would emit the fallback warning twice for one command, which trains
// operators to ignore it.

const (
	// The divergent pair this spec exists for: the pack is NAMED one thing and LIVES
	// somewhere else.
	divergentLockName   = "backstop/harness-toolchain"
	divergentCoordinate = "backstop-ai/backstop-harness-toolchain-pack"
)

func TestCoordinateForEntry_RecordedCoordinateReturnsNoWarning(t *testing.T) {
	entry := distribution.LockEntry{
		Name:             divergentLockName,
		SourceType:       "git",
		SourceCoordinate: divergentCoordinate,
	}

	coordinate, warning := distribution.CoordinateForEntry(divergentLockName, entry)

	if coordinate != divergentCoordinate {
		t.Errorf("coordinate = %q, want the RECORDED %q, not the pack name", coordinate, divergentCoordinate)
	}
	// Asserting the warning is EMPTY is the load-bearing half. An implementation that
	// warned unconditionally would satisfy a coordinate-only assertion while making the
	// fallback signal meaningless — a warning that fires always says nothing.
	if warning != "" {
		t.Errorf("warning = %q, want empty when the entry carries a coordinate; a signal that always fires is not a signal", warning)
	}
}

func TestCoordinateForEntry_MissingCoordinateFallsBackWithWarning(t *testing.T) {
	// The shape of EVERY lock entry written before this spec.
	entry := distribution.LockEntry{
		Name:       "acme/pack",
		SourceType: "git",
	}

	coordinate, warning := distribution.CoordinateForEntry("acme/pack", entry)

	if coordinate != "acme/pack" {
		t.Errorf("coordinate = %q, want the pack name %q as the compatibility fallback", coordinate, "acme/pack")
	}
	if warning == "" {
		t.Fatal("a fallback must never be silent — it is a compatibility path, not the primary one")
	}
	if !strings.Contains(warning, "acme/pack") {
		t.Errorf("warning %q does not name the pack; an operator with several packs cannot tell which one to fix", warning)
	}
	// NAME THE REMEDY. A warning that says only "no coordinate recorded" leaves the
	// operator with nothing to do about it.
	lower := strings.ToLower(warning)
	if !strings.Contains(lower, "re-add") && !strings.Contains(lower, "relock") && !strings.Contains(lower, "add") {
		t.Errorf("warning %q does not tell the operator how to fix it (re-add or relock the pack)", warning)
	}
}

// TestRemoteURLForEntry_BuildsURLFromTheSharedCoordinate is what makes it impossible for
// the resolver and the cloner to disagree about which repository a pack came from.
//
// THE DIVERGENT CASE IS THE ONLY ONE THAT CAN FAIL. If the lock key and the coordinate
// were the same string, a URL built from either would look correct, and the test would
// pass against an implementation that ignored the coordinate entirely.
func TestRemoteURLForEntry_BuildsURLFromTheSharedCoordinate(t *testing.T) {
	t.Run("recorded coordinate", func(t *testing.T) {
		entry := distribution.LockEntry{
			Name:             divergentLockName,
			SourceType:       "git",
			SourceCoordinate: divergentCoordinate,
		}

		coordinate, coordWarning := distribution.CoordinateForEntry(divergentLockName, entry)
		url, urlWarning := distribution.RemoteURLForEntry(divergentLockName, entry)

		if !strings.Contains(url, divergentCoordinate) {
			t.Errorf("url = %q, want it built from the COORDINATE %q", url, divergentCoordinate)
		}
		if strings.Contains(url, divergentLockName) {
			t.Errorf("url = %q names the lock key %q; the repository is the coordinate, not the pack name", url, divergentLockName)
		}
		// The URL must be the coordinate the shared accessor returned, not a second
		// resolution that happens to agree today.
		if !strings.Contains(url, coordinate) {
			t.Errorf("url = %q is not built from CoordinateForEntry's %q", url, coordinate)
		}
		if urlWarning != coordWarning {
			t.Errorf("RemoteURLForEntry warning = %q, want the SAME string CoordinateForEntry returned (%q) — layered, not duplicated", urlWarning, coordWarning)
		}
	})

	t.Run("fallback coordinate", func(t *testing.T) {
		entry := distribution.LockEntry{Name: "acme/pack", SourceType: "git"}

		coordinate, coordWarning := distribution.CoordinateForEntry("acme/pack", entry)
		url, urlWarning := distribution.RemoteURLForEntry("acme/pack", entry)

		if !strings.Contains(url, coordinate) {
			t.Errorf("url = %q is not built from the fallback coordinate %q", url, coordinate)
		}
		if urlWarning != coordWarning {
			t.Errorf("RemoteURLForEntry warning = %q, want the SAME string CoordinateForEntry returned (%q); two independent fallbacks double-warn for one invocation",
				urlWarning, coordWarning)
		}
		if urlWarning == "" {
			t.Error("the fallback warning must survive being layered through the URL builder")
		}
	})
}
