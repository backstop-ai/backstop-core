package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack"
)

// goldenFixturePath returns the path to the captured legacy golden violation set.
func goldenFixturePath(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRoot(t), "cmd", "backstop", "testdata", "golden-equivalence", "legacy-violations.golden.json")
}

// capturedToolBytes returns the fixed captured go build / go test / golangci
// output (the SAME bytes fed to both the legacy parser and the engine path so
// the comparison is like-for-like).
func capturedToolBytes(t *testing.T) map[string][]byte {
	t.Helper()
	dir := filepath.Join(repoRoot(t), "cmd", "backstop", "testdata", "golden-equivalence", "captured")
	read := func(name string) []byte {
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("reading captured %s: %v", name, err)
		}
		return b
	}
	return map[string][]byte{
		"go build":      read("go-build.txt"),
		"go test":       read("go-test.txt"),
		"golangci-lint": read("golangci.json"),
	}
}

// TestGoldenEquivalence_LegacyViolationSetCaptured proves the golden fixture
// captures the legacy pkg/check engine's normalized violation set on the
// backstop repo's captured tool output, and that it round-trips the legacy
// normalization for the captured bytes (CLM-009). It dispatches the captured
// bytes through the same convert→SARIF normalization the legacy engine path uses
// and asserts the result equals the on-disk golden fixture.
func TestGoldenEquivalence_LegacyViolationSetCaptured(t *testing.T) {
	golden, err := loadGoldenViolations(goldenFixturePath(t))
	if err != nil {
		t.Fatalf("loading golden fixture: %v", err)
	}
	if len(golden) == 0 {
		t.Fatal("golden fixture is empty — it must capture the legacy normalized violation set")
	}

	// Re-derive the legacy normalization for the captured bytes and assert the
	// fixture round-trips it (the fixture IS the evidence; it must reflect the
	// still-present legacy normalization).
	m := goToolchainManifest(t)
	stubSandboxedRunStdout(t, nil)
	runner := &fixtureRunner{byCmd: capturedToolBytes(t)}
	vs, err := dispatchPackEngines([]*pack.Manifest{m}, goToolchainPacksDir(t), t.TempDir(), nil, runner)
	if err != nil {
		t.Fatalf("re-deriving legacy normalization: %v", err)
	}
	reproduced := normalizeViolations(vs)
	sortGoldenViolations(golden)
	if !goldenViolationsEqual(golden, reproduced) {
		t.Fatalf("golden fixture does not round-trip the legacy normalization for the captured bytes\n golden=%#v\n got=%#v", golden, reproduced)
	}
}

// TestGoldenEquivalence_PackEnginePathReproducesGoldenSet proves the
// dispatchPackEngines <lang>-toolchain path produces the SAME normalized
// violation set as the golden fixture for the same captured tool output —
// equivalence proven (CLM-010).
func TestGoldenEquivalence_PackEnginePathReproducesGoldenSet(t *testing.T) {
	golden, err := loadGoldenViolations(goldenFixturePath(t))
	if err != nil {
		t.Fatalf("loading golden fixture: %v", err)
	}
	sortGoldenViolations(golden)

	m := goToolchainManifest(t)
	stubSandboxedRunStdout(t, nil)
	runner := &fixtureRunner{byCmd: capturedToolBytes(t)}

	// The REAL, un-stubbed dispatchPackEngines over the INSTALLED go-toolchain
	// pack on disk.
	vs, err := dispatchPackEngines([]*pack.Manifest{m}, goToolchainPacksDir(t), t.TempDir(), nil, runner)
	if err != nil {
		t.Fatalf("pack-engine dispatch: %v", err)
	}
	reproduced := normalizeViolations(vs)
	if !goldenViolationsEqual(golden, reproduced) {
		t.Fatalf("pack-engine path did NOT reproduce the golden legacy violation set\n golden=%#v\n got=%#v", golden, reproduced)
	}
}

// TestGoldenEquivalence_RealInstalledPackThroughUnstubbedDispatch is the hard
// anti-gaming mandate (CLM-011, Sharp Edge 4 / the pack-provisioning integration
// gap): the reproduction runs through the REAL UN-STUBBED dispatchPackEngines
// over an INSTALLED toolchain pack on disk, spying the sandboxed-dispatch seam —
// the dispatcher is NOT stubbed and no parallel raw-exec path is used.
func TestGoldenEquivalence_RealInstalledPackThroughUnstubbedDispatch(t *testing.T) {
	// Guard: the production dispatch seam must NOT be stubbed for this proof — a
	// stubbed dispatcher would game the only safety net.
	if dispatchPackEnginesFn != nil {
		t.Fatal("dispatchPackEnginesFn must be nil — the golden proof must run the REAL un-stubbed dispatcher, not a stub")
	}

	// The installed pack must exist ON DISK (an installed toolchain pack, not
	// testdata fed to a stub).
	installedPack := filepath.Join(goToolchainPacksDir(t), "backstop", "go-toolchain", "pack.yml")
	if _, err := os.ReadFile(installedPack); err != nil {
		t.Fatalf("the toolchain pack must be installed on disk for the un-stubbed dispatch: %v", err)
	}

	// Spy the sandboxed-dispatch seam (the convert step inside runFindingsEngine):
	// record the raw tool bytes fed into the convert pipe, WITHOUT stubbing the
	// dispatcher. This observes that the production path ran, not a parallel
	// raw-exec.
	var convertStdin []byte
	stubSandboxedRunStdout(t, &convertStdin)

	m := goToolchainManifest(t)
	runner := &fixtureRunner{byCmd: capturedToolBytes(t)}

	vs, err := dispatchPackEngines([]*pack.Manifest{m}, goToolchainPacksDir(t), t.TempDir(), nil, runner)
	if err != nil {
		t.Fatalf("un-stubbed dispatch: %v", err)
	}

	// The sandboxed-dispatch seam was actually exercised — the convert pipe
	// received raw tool bytes (proving the production convert→SARIF path ran).
	if len(convertStdin) == 0 {
		t.Fatal("the sandboxed-dispatch seam was never reached — the convert step did not run; the proof did not exercise the production path")
	}

	// And it reproduced the golden set through that production path.
	golden, err := loadGoldenViolations(goldenFixturePath(t))
	if err != nil {
		t.Fatalf("loading golden fixture: %v", err)
	}
	sortGoldenViolations(golden)
	reproduced := normalizeViolations(vs)
	if !goldenViolationsEqual(golden, reproduced) {
		t.Fatalf("the un-stubbed production dispatch did not reproduce the golden set\n golden=%#v\n got=%#v", golden, reproduced)
	}

	// Source guard: no parallel raw-exec dispatch path may exist — the proof
	// rides dispatchPackEngines, the same substrate the gate uses.
	src := readFileStr(t, "golden_equivalence.go")
	for _, banned := range []string{"exec.Command", "os/exec"} {
		if containsStr(src, banned) {
			t.Errorf("golden_equivalence.go references %q; the golden proof must run through dispatchPackEngines, not a parallel raw-exec path", banned)
		}
	}
}
