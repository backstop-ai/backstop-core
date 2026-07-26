package distribution_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack/distribution"
)

// contracts_local_install_test.go covers the REQ-013 redesign: the contracts
// local-install helper RECEIVES an assembled command instead of assembling one.
//
// A library-layer helper that built its own validator would be exactly the
// internal defaulting REQ-005 forbids, and would make the helper's dependencies
// indistinguishable from production wiring. Receiving the command is what keeps
// the assembly decision at the wiring site.

// recordingContractsValidator DELEGATES to the production validator and records
// which phases ran.
//
// Recording alone would prove only that the phases were invoked; delegating means
// the contracts pack must genuinely pass them. Both halves are needed: an
// invocation count is satisfied by a validator that accepts everything, and a
// passing install is satisfied by a validator that never ran.
type recordingContractsValidator struct {
	inner   *distribution.PackvalValidator
	checked bool
	tested  bool
}

func (v *recordingContractsValidator) RunPackCheck(packDir string) error {
	v.checked = true
	return v.inner.RunPackCheck(packDir)
}

func (v *recordingContractsValidator) RunPackTest(packDir string) error {
	v.tested = true
	return v.inner.RunPackTest(packDir)
}

// TestInstallContractsLocalPack_InstallsWithSuppliedCommand (CLM-092) asserts the
// helper installs the packs/contracts SOURCE through a caller-supplied assembled
// command, with validation running, and returns the real AddResult.
//
// It asserts the installed path and the content hash rather than only the error,
// because a helper that returned a zero-valued result would satisfy an
// error-only check while having installed nothing.
func TestInstallContractsLocalPack_InstallsWithSuppliedCommand(t *testing.T) {
	root := provRepoRoot(t)
	ws := newProvWorkspace(t)

	add := newTestAddCommand(t, distribution.NewExecGitCloner(), distribution.NewPackvalValidator())

	res, err := distribution.InstallContractsLocalPack(add, root, ws)
	if err != nil {
		t.Fatalf("installing the contracts pack through a supplied command: %v", err)
	}

	if res.PackName != "backstop/contracts" {
		t.Errorf("installed pack name = %q, want backstop/contracts", res.PackName)
	}
	if res.InstalledPath == "" {
		t.Error("InstalledPath is empty; the helper returned a result that records no installation")
	} else if _, statErr := os.Stat(res.InstalledPath); statErr != nil {
		t.Errorf("InstalledPath %q does not exist on disk: %v", res.InstalledPath, statErr)
	}
	if !contentDigest.MatchString(res.ContentHash) {
		t.Errorf("ContentHash = %q, want a 64-character hex digest; an empty or truncated hash means nothing was hashed", res.ContentHash)
	}
}

// TestInstallContractsLocalPack_ContractsPackPassesUnconditionalValidation
// (CLM-093) asserts that the packs/contracts source PASSES pack check and pack
// test, so the dogfood install stays green now that validation is unconditional.
//
// This claim exists because the redesign REMOVES the nil-validator skip that was
// hiding whether the pack passes. It is asserted twice over: directly against the
// source, and through the install path with a validator that records both phases
// actually ran.
func TestInstallContractsLocalPack_ContractsPackPassesUnconditionalValidation(t *testing.T) {
	root := provRepoRoot(t)
	source := distribution.ContractsPackSourceDir(root)

	production := distribution.NewPackvalValidator()
	if err := production.RunPackCheck(source); err != nil {
		t.Errorf("the contracts pack source must pass pack check: %v", err)
	}
	if err := production.RunPackTest(source); err != nil {
		t.Errorf("the contracts pack source must pass pack test: %v", err)
	}

	recorder := &recordingContractsValidator{inner: distribution.NewPackvalValidator()}
	add := newTestAddCommand(t, distribution.NewExecGitCloner(), recorder)

	if _, err := distribution.InstallContractsLocalPack(add, root, newProvWorkspace(t)); err != nil {
		t.Fatalf("installing the contracts pack with validation running: %v", err)
	}
	if !recorder.checked {
		t.Error("the install did not run pack check; validation is meant to be unconditional")
	}
	if !recorder.tested {
		t.Error("the install did not run pack test; validation is meant to be unconditional")
	}
}

// nilValidatorSkipContract matches a doc contract promising that an absent
// validator skips the validation phases. REQ-008 makes validation unconditional,
// so any surviving statement of that promise is a lie in the documentation even
// once the code implementing it is gone.
var nilValidatorSkipContract = regexp.MustCompile(`(?i)nil\s+validator\s+skips`)

// contentDigest is the shape AddResult.ContentHash carries: the bare digest. The
// `sha256:` prefix belongs to the LOCK entry, not to the result.
var contentDigest = regexp.MustCompile(`^[0-9a-f]{64}$`)

// TestDistribution_NoValidatorAwareInstallVariant (CLM-094) asserts the package
// declares NO validator-aware install variant and NO nil-validator-skip contract.
//
// It scans the package's real non-test sources rather than asserting against a
// known list, so a variant reintroduced later is caught too.
func TestDistribution_NoValidatorAwareInstallVariant(t *testing.T) {
	dir := filepath.Join(provRepoRoot(t), "pkg", "pack", "distribution")

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading the distribution package sources at %s: %v", dir, err)
	}

	var variants, skipContracts []string
	scanned := 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		source, readErr := os.ReadFile(filepath.Join(dir, name))
		if readErr != nil {
			t.Fatalf("reading package source %s: %v", name, readErr)
		}
		scanned++
		if strings.Contains(string(source), "InstallContractsLocalPackWithValidator") {
			variants = append(variants, name)
		}
		if nilValidatorSkipContract.Match(source) {
			skipContracts = append(skipContracts, name)
		}
	}
	if scanned == 0 {
		t.Fatalf("scanned no package sources in %s; the scan is broken and would pass vacuously", dir)
	}

	if len(variants) > 0 {
		t.Errorf("InstallContractsLocalPackWithValidator still declared in %v; under unconditional validation the distinction the variant expressed no longer exists", variants)
	}
	if len(skipContracts) > 0 {
		t.Errorf("a nil-validator-skip doc contract survives in %v; validation is unconditional, so the promise is no longer true", skipContracts)
	}
}
