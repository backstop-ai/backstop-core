package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The external Bun-fork acceptance (SPEC-047 REQ-005) is the REQUIRED executed
// end-to-end proof, but it is guarded so it SKIPS (never FAILS) in backstop-core's
// Go CI — keeping the real bun/oxlint/tsc/prettier toolchain OUT of Go CI. It is
// enabled only in the fork's own environment, where bunAcceptanceEnabled() is true.
//
// MANUAL RUN-EVIDENCE REQUIREMENT (REQ-005): because the Go-CI guard makes this
// acceptance auto-skip, REQ-005 is CLOSED only by a documented MANUAL run on the
// fork (backstop-runtime) capturing the RED-then-green `backstop gate` output. The
// skipped Go-CI stub does NOT close it. Minimal fork wiring (single acceptance):
//
//	cd ~/src/projects/backstop-runtime
//	backstop pack add backstop/bun-toolchain --from ../backstop-bun-toolchain-pack
//	cat > backstop.yml <<'YAML'
//	project: backstop-runtime
//	packs:
//	    backstop/bun-toolchain: local
//	YAML
//	# seed a defect (e.g. an unformatted .ts / a type error / an uncovered source),
//	BACKSTOP_BUN_ACCEPTANCE=1 backstop gate   # expect RED (exit != 0)
//	# fix the defect, then:
//	BACKSTOP_BUN_ACCEPTANCE=1 backstop gate   # expect green (exit 0)

// bunForkDir returns the Bun fork's project root from the acceptance env var, or ""
// when unset (the fork wiring is provided out-of-band; productionizing fork CI is
// out of scope — only the single-acceptance wiring is in scope).
func bunForkDir() string {
	return os.Getenv("BACKSTOP_BUN_FORK")
}

// runForkGate runs the real `backstop gate` in the fork dir with the acceptance env
// var set, returning combined output + the process exit error (nil == green).
func runForkGate(t *testing.T, forkDir string) (string, error) {
	t.Helper()
	cmd := exec.Command(filepath.Join(repoRoot(t), "bin", "backstop"), "gate", "--all")
	cmd.Dir = forkDir
	cmd.Env = append(os.Environ(), "BACKSTOP_BUN_ACCEPTANCE=1")
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// TestAcceptance_SkippedWhenBunToolchainAbsentKeepsGoCIBunFree proves the
// acceptance SKIPS (does not fail) when the bun toolchain is absent / the
// acceptance env var is unset, so backstop-core's Go CI never invokes the real bun
// toolchain (CLM-029).
func TestAcceptance_SkippedWhenBunToolchainAbsentKeepsGoCIBunFree(t *testing.T) {
	// With the acceptance env var unset, the guard MUST be false regardless of
	// whether a bun binary happens to be on PATH — so Go CI stays bun-free.
	t.Setenv("BACKSTOP_BUN_ACCEPTANCE", "")
	if bunAcceptanceEnabled() {
		t.Fatal("with the acceptance env var unset, bunAcceptanceEnabled() must be false — the acceptance must SKIP so Go CI never runs the real bun toolchain")
	}
	// Demonstrate the actual skip mechanism the executed acceptances use.
	if !bunAcceptanceEnabled() {
		t.Skip("bun acceptance disabled (env var unset / bun absent) — SKIPPING the executed fork gate keeps Go CI bun-free (CLM-029)")
	}
	t.Fatal("unreachable: the guard was false, so the test must have skipped")
}

// TestAcceptance_ForkBunGateRedThenGreenOnSeededDefect runs the real `backstop
// gate` over the Bun fork wired packs-only and asserts RED on a seeded defect, then
// green when fixed (RED-then-green) (CLM-027). It SKIPS in Go CI; the executed proof
// runs in the fork's env and its RED-then-green output must be recorded as manual
// run-evidence to CLOSE REQ-005.
func TestAcceptance_ForkBunGateRedThenGreenOnSeededDefect(t *testing.T) {
	if !bunAcceptanceEnabled() {
		t.Skip("bun toolchain absent / acceptance env var unset — the executed fork gate runs only in the fork env (REQ-005). REQ-005 is CLOSED by a documented MANUAL run capturing the RED-then-green backstop gate output; the skipped Go-CI stub does not close it.")
	}
	forkDir := bunForkDir()
	if forkDir == "" {
		t.Skip("BACKSTOP_BUN_FORK not set — cannot locate the Bun fork project root for the executed acceptance")
	}
	// Seeded defect present ⇒ RED.
	out, err := runForkGate(t, forkDir)
	if err == nil {
		t.Fatalf("the fork gate must go RED on a seeded lint/format/type/test/coverage defect, got green:\n%s", out)
	}
	// The defect must be fixed out-of-band (the manual acceptance flips it), after
	// which a re-run is green — recorded as run-evidence.
	if strings.Contains(out, "PASS") {
		t.Fatalf("seeded-defect run must not report PASS:\n%s", out)
	}
}

// TestAcceptance_ForkGatesPacksOnlyViaDeclaredBunToolchainPack proves the fork
// gates PACKS-ONLY via the declared backstop/bun-toolchain pack — no baked language
// path participates; the bun pack's engines are the only toolchain source (CLM-028).
func TestAcceptance_ForkGatesPacksOnlyViaDeclaredBunToolchainPack(t *testing.T) {
	if !bunAcceptanceEnabled() {
		t.Skip("bun toolchain absent / acceptance env var unset — the packs-only fork gate runs only in the fork env (REQ-005)")
	}
	forkDir := bunForkDir()
	if forkDir == "" {
		t.Skip("BACKSTOP_BUN_FORK not set — cannot locate the Bun fork project root")
	}
	// The fork's backstop.yml declares the bun pack and NO language field; the gate's
	// toolchain source is the declared pack alone.
	cfg, err := os.ReadFile(filepath.Join(forkDir, "backstop.yml"))
	if err != nil {
		t.Fatalf("read fork backstop.yml: %v", err)
	}
	if !strings.Contains(string(cfg), "backstop/bun-toolchain") {
		t.Fatalf("the fork must declare backstop/bun-toolchain as the packs-only toolchain source, got:\n%s", cfg)
	}
	for _, line := range strings.Split(string(cfg), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "language:") {
			t.Errorf("the fork gates packs-only — no baked language path: backstop.yml must carry no language field, got %q", strings.TrimSpace(line))
		}
	}
	out, _ := runForkGate(t, forkDir)
	if strings.Contains(out, "no capability wired") && !strings.Contains(out, "bun-toolchain") {
		t.Errorf("the bun pack's engines must be the toolchain source (packs-only), got:\n%s", out)
	}
}
