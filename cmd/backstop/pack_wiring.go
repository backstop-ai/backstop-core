package main

import (
	"fmt"

	"github.com/backstop-ai/backstop-core/pkg/pack/distribution"
)

// pack_wiring.go is THE production assembly seam for the pack lifecycle (SPEC-055
// REQ-010). Every dependency every pack command runs on is constructed here, so
// "what does production actually wire?" has ONE answer instead of being spread
// across four Cobra files that each built a partial options literal — which is how
// `pack add` shipped with no cloner in it at all and nil-dereferenced on the first
// remote pack anyone tried (ISSUE-073).
//
// Assembly is where dependencies are chosen; the library layer supplies none of its
// own. A constructor inside pkg/pack/distribution that defaulted a missing cloner
// would make a test double indistinguishable from production wiring, which is the
// property these helpers exist to keep.
//
// An assembly failure is a CONFIGURATION error, not a violation: the operator's
// project is fine, the binary is mis-built. The callers surface it as exit 2.

// capabilityBundle is the bundle whose requirements track the two pack-upgrade
// capabilities this spec deliberately does not build. Naming it in the diagnostic is
// what makes the requirement numbers below resolvable.
const capabilityBundle = "BUNDLE-006"

// The capabilities `pack upgrade` declares but does not yet have, and the
// requirements that own them.
const (
	violationScanCapability = "the pack upgrade violation scan"
	violationScanReference  = capabilityBundle + " REQ-014"

	remediationCapability = "pack upgrade remediation bundle generation"
	remediationReference  = capabilityBundle + " REQ-018"
)

// newProductionAddCommand assembles `pack add` over the real cloner and the real
// validator.
//
// The validator is not optional here. A nil one used to skip pack check and pack
// test silently, which meant an invalid pack installed cleanly — the consumer
// learned about it from a broken gate rather than from the add that introduced it.
func newProductionAddCommand() (*distribution.AddCommand, error) {
	return distribution.NewAddCommand(distribution.NewExecGitCloner(), distribution.NewPackvalValidator())
}

// newProductionInstallCommand assembles `pack install` over the real cloner alone.
//
// Install takes NO validator on purpose: it is the hash-verified restore path, and
// what it restores was validated when it was added. Handing it one "for symmetry"
// would assert a re-validation it does not perform.
func newProductionInstallCommand() (*distribution.InstallCommand, error) {
	return distribution.NewInstallCommand(distribution.NewExecGitCloner())
}

// newProductionUpdateCommand assembles `pack update` over the real cloner, the real
// validator, and a resolver built on THE SAME cloner.
//
// Sharing the cloner is deliberate: resolution and the clone that follows it must
// reach the same repository through the same configuration, and two independently
// constructed cloners are two chances for them to diverge.
func newProductionUpdateCommand() (*distribution.UpdateCommand, error) {
	cloner := distribution.NewExecGitCloner()

	resolver, err := distribution.NewTagVersionResolver(cloner)
	if err != nil {
		return nil, fmt.Errorf("assembling the pack update version resolver: %w", err)
	}

	return distribution.NewUpdateCommand(cloner, distribution.NewPackvalValidator(), resolver)
}

// newProductionUpgradeCommand assembles `pack upgrade` over the real cloner and
// validator plus EXPLICIT implementations of the two capabilities that are declared
// but not yet built.
//
// They are explicit rather than nil because nil meant "skip", and skipping produced
// a successful major upgrade reporting zero baselined violations it had never
// scanned for. Failing loud is worse to use and honest; a vacuous green on the one
// operation where a consumer most needs to know what broke is not.
func newProductionUpgradeCommand() (*distribution.UpgradeCommand, error) {
	return distribution.NewUpgradeCommand(
		distribution.NewExecGitCloner(),
		distribution.NewPackvalValidator(),
		unavailableScanner{},
		unavailableRemediationGenerator{},
	)
}

// unavailableScanner stands in for the violation scan that BUNDLE-006 REQ-014 owns.
//
// It returns a typed error rather than an empty violation slice. An empty slice is
// indistinguishable from "scanned and found nothing", which is precisely the
// vacuous success this spec exists to close.
type unavailableScanner struct{}

// ScanViolations reports that the scan capability is not yet available.
func (s unavailableScanner) ScanViolations(projectDir, packDir string) ([]string, error) {
	return nil, &distribution.CapabilityUnavailableError{
		Capability: violationScanCapability,
		Reference:  violationScanReference,
	}
}

// unavailableRemediationGenerator stands in for the remediation bundle generation
// that BUNDLE-006 REQ-018 owns.
type unavailableRemediationGenerator struct{}

// GenerateBundle reports that the remediation capability is not yet available.
func (g unavailableRemediationGenerator) GenerateBundle(projectDir string, violations []string) (string, error) {
	return "", &distribution.CapabilityUnavailableError{
		Capability: remediationCapability,
		Reference:  remediationReference,
	}
}
