package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack"
)

// rungroupPacksDir returns the .backstop/packs dir holding the ISSUE-068 rungroup
// fixture pack (a test engine + coverage engine sharing one declared run_group).
func rungroupPacksDir(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRoot(t), "cmd", "backstop", "testdata", "rungroup", ".backstop", "packs")
}

// rungroupManifest loads the ISSUE-068 rungroup fixture pack manifest, proving the
// pack.yml parses (its run_group declarations round-trip through ParseManifest AND
// pass validateRunGroups coherence).
func rungroupManifest(t *testing.T) *pack.Manifest {
	t.Helper()
	m, err := pack.ParseManifestFile(filepath.Join(rungroupPacksDir(t),
		"backstop", "rungroup", "pack.yml"))
	if err != nil {
		t.Fatalf("rungroup fixture pack must parse (run_group + coherence): %v", err)
	}
	return m
}

// setEngineRun mutates one engine binding's Command + RunGroup in-memory so a
// dispatch-level test can vary the dedup inputs without re-authoring the on-disk
// (coherent) fixture. Dedup keying is a dispatch concern independent of parse-time
// coherence, so mutating here is faithful.
func setEngineRun(t *testing.T, m *pack.Manifest, engineName, command, runGroup string) {
	t.Helper()
	spec, ok := m.Engines[engineName]
	if !ok {
		t.Fatalf("engine %q not in fixture manifest; have %v", engineName, m.Engines)
	}
	spec.Binding.Command = command
	spec.Binding.RunGroup = runGroup
	m.Engines[engineName] = spec
}

// countCalls returns how many runner invocations had the given command name.
func countCalls(runner *fixtureRunner, name string) int {
	n := 0
	for _, c := range runner.calls {
		if c.name == name {
			n++
		}
	}
	return n
}

// TestSharedRunCache_SharedKeyRunsCommandOnce proves two DISTINCT engines (test +
// coverage) declaring the SAME run_group cause the underlying command to execute
// ONCE across the shared run-cache (ISSUE-068 CLM-004/CLM-005). The pack_engines
// (findings) dispatch is the WRITER; the coverage dispatch is the READER and reuses
// the memoized payload rather than re-running the command.
func TestSharedRunCache_SharedKeyRunsCommandOnce(t *testing.T) {
	if dispatchPackEnginesFn != nil {
		t.Fatal("dispatchPackEnginesFn must be nil — this test must run the REAL cache-aware dispatch, not a stub")
	}
	m := rungroupManifest(t)
	stubSandboxedRunStdout(t, nil)
	runner := &fixtureRunner{byCmd: map[string][]byte{"suite run": []byte("SHARED-PAYLOAD-TOKEN")}}
	packsDir := rungroupPacksDir(t)
	projectRoot := t.TempDir()

	cache := newSharedRunCache()

	// WRITER: the findings/test dispatch runs first (gate order). The coverage rule
	// is stripped from the findings path exactly as the real gate does.
	enginePacks := excludeDedicatedStepRules([]*pack.Manifest{m})
	if _, err := dispatchPackEnginesWithCache(cache, enginePacks, packsDir, projectRoot, nil, runner); err != nil {
		t.Fatalf("findings dispatch (writer): %v", err)
	}
	// READER: the coverage dispatch reuses the memoized payload.
	if _, err := dispatchPackCoverageWithCache(cache, []*pack.Manifest{m}, packsDir, projectRoot, nil, runner); err != nil {
		t.Fatalf("coverage dispatch (reader): %v", err)
	}

	if got := countCalls(runner, "suite"); got != 1 {
		t.Fatalf("shared run_group must run the command ONCE across both steps, got %d invocations: %#v", got, runner.calls)
	}
}

// TestSharedRunCache_BothConvertsReceiveSharedOutput proves the single run's payload
// is fanned into BOTH engines' OWN converts (ISSUE-068 CLM-004/CLM-006): the test
// convert extracts findings and the coverage convert extracts coverage records, each
// from the SAME memoized payload.
func TestSharedRunCache_BothConvertsReceiveSharedOutput(t *testing.T) {
	m := rungroupManifest(t)

	// Record every stdin the convert step received, then run the real convert script.
	orig := sandboxedRunStdout
	var convertStdins [][]byte
	sandboxedRunStdout = func(cmd string, args []string, packDir string, stdin []byte) ([]byte, error) {
		convertStdins = append(convertStdins, append([]byte(nil), stdin...))
		return runConvertScriptDirect(cmd, stdin)
	}
	t.Cleanup(func() { sandboxedRunStdout = orig })

	runner := &fixtureRunner{byCmd: map[string][]byte{"suite run": []byte("SHARED-PAYLOAD-TOKEN")}}
	packsDir := rungroupPacksDir(t)
	projectRoot := t.TempDir()

	cache := newSharedRunCache()
	enginePacks := excludeDedicatedStepRules([]*pack.Manifest{m})
	findings, err := dispatchPackEnginesWithCache(cache, enginePacks, packsDir, projectRoot, nil, runner)
	if err != nil {
		t.Fatalf("findings dispatch: %v", err)
	}
	records, err := dispatchPackCoverageWithCache(cache, []*pack.Manifest{m}, packsDir, projectRoot, nil, runner)
	if err != nil {
		t.Fatalf("coverage dispatch: %v", err)
	}

	// The test convert produced SARIF findings; the coverage convert produced records.
	if len(findings) != 1 {
		t.Fatalf("test convert must yield one finding from the shared payload, got %d: %#v", len(findings), findings)
	}
	if len(records) != 1 {
		t.Fatalf("coverage convert must yield one record from the shared payload, got %d: %#v", len(records), records)
	}

	// BOTH converts received the identical shared payload (the one run's output).
	if len(convertStdins) != 2 {
		t.Fatalf("expected exactly two convert invocations (one per engine), got %d", len(convertStdins))
	}
	for i, in := range convertStdins {
		if string(in) != "SHARED-PAYLOAD-TOKEN" {
			t.Errorf("convert %d received %q, want the shared run payload %q", i, string(in), "SHARED-PAYLOAD-TOKEN")
		}
	}
}

// TestSharedRunCache_AbsentKeyRunsTwice proves the SAFE DEFAULT (ISSUE-068
// CLM-005/CLM-008): with NO run_group declared, the command runs TWICE — once for
// the findings dispatch and once for the coverage dispatch — the unchanged two-run
// behavior that keeps separate-build toolchains from regressing.
func TestSharedRunCache_AbsentKeyRunsTwice(t *testing.T) {
	m := rungroupManifest(t)
	// Clear the declared key on both engines (safe-default variation).
	setEngineRun(t, m, "suite-test", "suite run --coverage", "")
	setEngineRun(t, m, "suite-coverage", "suite run --coverage", "")

	stubSandboxedRunStdout(t, nil)
	runner := &fixtureRunner{byCmd: map[string][]byte{"suite run": []byte("PAYLOAD")}}
	packsDir := rungroupPacksDir(t)
	projectRoot := t.TempDir()

	cache := newSharedRunCache()
	enginePacks := excludeDedicatedStepRules([]*pack.Manifest{m})
	if _, err := dispatchPackEnginesWithCache(cache, enginePacks, packsDir, projectRoot, nil, runner); err != nil {
		t.Fatalf("findings dispatch: %v", err)
	}
	if _, err := dispatchPackCoverageWithCache(cache, []*pack.Manifest{m}, packsDir, projectRoot, nil, runner); err != nil {
		t.Fatalf("coverage dispatch: %v", err)
	}

	if got := countCalls(runner, "suite"); got != 2 {
		t.Fatalf("absent run_group must run the command TWICE (unchanged default), got %d: %#v", got, runner.calls)
	}
}

// TestSharedRunCache_CrossPackSameKeyRunsEachOnce proves the run-cache key is
// NAMESPACED by pack identity (ISSUE-068 cross-pack collision fix): two DIFFERENT
// packs that each declare the SAME run_group string must NOT share a run — each pack's
// command runs ONCE. Before the fix the cache keyed on the raw run_group alone, so the
// second pack silently reused the first pack's memoized payload (its command NEVER ran)
// — a cross-pack collision. Each pack lives under its OWN NormalizedName so packRoot (=
// packDir/NormalizedName) resolves to real on-disk convert scripts; the fixture is
// copied into both namespaces in a temp packDir.
func TestSharedRunCache_CrossPackSameKeyRunsEachOnce(t *testing.T) {
	if dispatchPackEnginesFn != nil {
		t.Fatal("dispatchPackEnginesFn must be nil — this test must run the REAL cache-aware dispatch, not a stub")
	}
	srcPack := filepath.Join(rungroupPacksDir(t), "backstop", "rungroup")

	// Materialize a temp packDir holding the SAME fixture pack under two DISTINCT
	// namespaces so each manifest resolves its own real convert scripts on disk.
	packsDir := t.TempDir()
	packAlpha := filepath.Join(packsDir, "alpha-org", "rungroup")
	packBeta := filepath.Join(packsDir, "beta-org", "rungroup")
	for _, dst := range []string{packAlpha, packBeta} {
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.CopyFS(dst, os.DirFS(srcPack)); err != nil {
			t.Fatalf("copy fixture pack to %s: %v", dst, err)
		}
	}

	mAlpha, err := pack.ParseManifestFile(filepath.Join(packAlpha, "pack.yml"))
	if err != nil {
		t.Fatalf("alpha pack must parse: %v", err)
	}
	mBeta, err := pack.ParseManifestFile(filepath.Join(packBeta, "pack.yml"))
	if err != nil {
		t.Fatalf("beta pack must parse: %v", err)
	}
	// Give the two packs DISTINCT identities so packRoot resolves to their own copies.
	mAlpha.NormalizedName = "alpha-org/rungroup"
	mBeta.NormalizedName = "beta-org/rungroup"
	// Each declares the SAME run_group string "shared" on its findings (test) engine but
	// a DIFFERENT command, so a collision would surface as the second command not running.
	setEngineRun(t, mAlpha, "suite-test", "alpha run", "shared")
	setEngineRun(t, mBeta, "suite-test", "beta run", "shared")

	stubSandboxedRunStdout(t, nil)
	runner := &fixtureRunner{byCmd: map[string][]byte{
		"alpha run": []byte("ALPHA-PAYLOAD"),
		"beta run":  []byte("BETA-PAYLOAD"),
	}}
	projectRoot := t.TempDir()

	cache := newSharedRunCache()
	enginePacks := excludeDedicatedStepRules([]*pack.Manifest{mAlpha, mBeta})
	if _, err := dispatchPackEnginesWithCache(cache, enginePacks, packsDir, projectRoot, nil, runner); err != nil {
		t.Fatalf("findings dispatch over two packs: %v", err)
	}

	if got := countCalls(runner, "alpha"); got != 1 {
		t.Errorf("alpha pack command must run exactly once, got %d: %#v", got, runner.calls)
	}
	if got := countCalls(runner, "beta"); got != 1 {
		t.Errorf("beta pack shares the SAME run_group string but is a DIFFERENT pack — its command must still run once (no cross-pack reuse), got %d: %#v", got, runner.calls)
	}
}

// TestSharedRunCache_DedupKeysOnDeclaredFieldNotCommand proves core dedupes SOLELY
// by the opaque declared RunGroup field and NEVER inspects command strings
// (thin-executor / DD-3, ISSUE-068 CLM-006/CLM-008):
//   - two engines with DIFFERENT commands but the SAME declared key still dedupe (the
//     reader's command never runs — the memoized payload is keyed on the field), and
//   - two engines with IDENTICAL commands but NO declared key do NOT dedupe (both run).
func TestSharedRunCache_DedupKeysOnDeclaredFieldNotCommand(t *testing.T) {
	packsDir := rungroupPacksDir(t)

	t.Run("different commands, same key => dedupe", func(t *testing.T) {
		m := rungroupManifest(t)
		setEngineRun(t, m, "suite-test", "alpha run", "shared")
		setEngineRun(t, m, "suite-coverage", "beta run", "shared")

		stubSandboxedRunStdout(t, nil)
		runner := &fixtureRunner{byCmd: map[string][]byte{
			"alpha run": []byte("ALPHA-PAYLOAD"),
			"beta run":  []byte("BETA-PAYLOAD"),
		}}
		projectRoot := t.TempDir()

		cache := newSharedRunCache()
		enginePacks := excludeDedicatedStepRules([]*pack.Manifest{m})
		if _, err := dispatchPackEnginesWithCache(cache, enginePacks, packsDir, projectRoot, nil, runner); err != nil {
			t.Fatalf("findings dispatch (writer): %v", err)
		}
		if _, err := dispatchPackCoverageWithCache(cache, []*pack.Manifest{m}, packsDir, projectRoot, nil, runner); err != nil {
			t.Fatalf("coverage dispatch (reader): %v", err)
		}

		if got := countCalls(runner, "alpha"); got != 1 {
			t.Errorf("writer command must run once, got %d: %#v", got, runner.calls)
		}
		if got := countCalls(runner, "beta"); got != 0 {
			t.Errorf("reader command must NEVER run when the declared key hits the cache — dedup keys on the field, not the command; got %d beta calls: %#v", got, runner.calls)
		}
	})

	t.Run("identical commands, no key => two runs", func(t *testing.T) {
		m := rungroupManifest(t)
		setEngineRun(t, m, "suite-test", "gamma run", "")
		setEngineRun(t, m, "suite-coverage", "gamma run", "")

		stubSandboxedRunStdout(t, nil)
		runner := &fixtureRunner{byCmd: map[string][]byte{"gamma run": []byte("GAMMA-PAYLOAD")}}
		projectRoot := t.TempDir()

		cache := newSharedRunCache()
		enginePacks := excludeDedicatedStepRules([]*pack.Manifest{m})
		if _, err := dispatchPackEnginesWithCache(cache, enginePacks, packsDir, projectRoot, nil, runner); err != nil {
			t.Fatalf("findings dispatch: %v", err)
		}
		if _, err := dispatchPackCoverageWithCache(cache, []*pack.Manifest{m}, packsDir, projectRoot, nil, runner); err != nil {
			t.Fatalf("coverage dispatch: %v", err)
		}

		if got := countCalls(runner, "gamma"); got != 2 {
			t.Errorf("identical commands with NO declared key must NOT dedupe (core never sniffs commands), want 2 runs got %d: %#v", got, runner.calls)
		}
	})
}
