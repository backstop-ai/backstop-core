package main

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// copyTree recursively copies src into dst, preserving executable bits.
func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(src, path)
		if relErr != nil {
			return relErr
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		info, infoErr := d.Info()
		if infoErr != nil {
			return infoErr
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		return os.WriteFile(target, data, info.Mode().Perm())
	})
	if err != nil {
		t.Fatalf("copying fixture tree: %v", err)
	}
}

// runGateInDir chdirs into dir and drives the REAL shipped runGate over the full
// project sweep, returning its captured human output and error.
func runGateInDir(t *testing.T, dir string) (string, error) {
	t.Helper()
	orig, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(orig) }()

	cmd := newGateCommand(new(bool))
	// The --json flag lives on the root command in production; register it locally
	// so runGate's GetBool("json") resolves to human output.
	cmd.Flags().Bool("json", false, "")
	if err := cmd.Flags().Set("all", "true"); err != nil {
		t.Fatalf("set --all: %v", err)
	}
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	err := runGate(cmd, nil)
	return buf.String(), err
}

// TestGateCLI_Waiver_SuppressesOverInstalledPack proves the SHIPPED `backstop
// gate` construction path suppresses a code-located pack_engines finding via a
// co-located @waiver and terminates in the distinct PASS·N-waivers state — proving
// WithWaiver is actually CALLED and fed, not merely defined (CLM-070). RED against
// a construction site that never calls WithWaiver (the dark gate): the waivable
// finding would stand and fail the gate.
func TestGateCLI_Waiver_SuppressesOverInstalledPack(t *testing.T) {
	temp := t.TempDir()
	copyTree(t, waiverE2EFixtureRoot(t), temp)
	t.Setenv("WAIVER_E2E_SCENARIO", "waivable")

	out, err := runGateInDir(t, temp)
	if err != nil {
		t.Fatalf("the shipped gate must GREEN when the sole finding is waived; got err=%v\noutput:\n%s", err, out)
	}
	if !strings.Contains(out, "PASS · 1 waivers") {
		t.Fatalf("the shipped gate must render the distinct PASS·N-waivers state; output:\n%s", out)
	}
}

// TestGateCLI_Waiver_SelfRuleErrorsOverInstalledPack proves the shipped path
// errors when a @waiver targets the packs-declared non-waivable rule — proving
// buildWaiverPolicy's extracted set reaches the real reconciliation (CLM-071). RED
// against the dark gate: without WithWaiver there is no waiver adjudication and no
// "declared non-waivable" gate error.
func TestGateCLI_Waiver_SelfRuleErrorsOverInstalledPack(t *testing.T) {
	temp := t.TempDir()
	copyTree(t, waiverE2EFixtureRoot(t), temp)
	t.Setenv("WAIVER_E2E_SCENARIO", "protected")

	out, err := runGateInDir(t, temp)
	if err == nil {
		t.Fatalf("the shipped gate must FAIL on a @waiver targeting a non-waivable rule; output:\n%s", out)
	}
	if !strings.Contains(out, "declared non-waivable") {
		t.Fatalf("the shipped gate must raise a non-waivable gate ERROR via the extracted Policy; output:\n%s", out)
	}
}
