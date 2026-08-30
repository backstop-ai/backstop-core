package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestWebsiteJourney_VerificationCleanupPasses(t *testing.T) {
	runWebsiteCapabilitiesScript(t, "scripts/tests/website-capabilities/verify-wrapper.sh")
}

func TestWebsiteJourney_RemainsIntegrationConsumer(t *testing.T) {
	root := websiteJourneyRepoRoot(t)
	absRoot, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(filepath.Join(absRoot, "scripts", "verify-website-capabilities.sh"), "--assert-consumer")
	cmd.Dir = absRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("integration consumer: %v\n%s", err, out)
	}
	if strings.Contains(string(out), "sitecheck") {
		t.Fatal(string(out))
	}
}

func TestWebsiteJourney_VerifierAcceptsCoverageAtThreshold(t *testing.T) {
	runWebsiteCapabilitiesScript(t, "scripts/tests/website-capabilities/coverage-cleanup-mutations.sh")
}

func TestWebsiteJourney_VerifierRejectsCoverageFailureMatrix(t *testing.T) {
	runWebsiteCapabilitiesScript(t, "scripts/tests/website-capabilities/coverage-cleanup-mutations.sh")
}

func TestWebsiteJourney_WrapperPropagatesGovernedDependencyFailure(t *testing.T) {
	root := websiteJourneyRepoRoot(t)
	absRoot, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(filepath.Join(absRoot, "scripts", "verify-website-capabilities.sh"))
	cmd.Dir = absRoot
	cmd.Env = append(os.Environ(), "BACKSTOP_WEBSITE_SELF_TEST=1", "BACKSTOP_WEBSITE_PREREQ_FAIL=verify-public-product-model")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("wrapper accepted a failed governed dependency")
	}
	text := string(out)
	if !strings.Contains(text, "verify-public-product-model") || !strings.Contains(text, "before traversal") {
		t.Fatalf("dependency failure: %s", text)
	}
}

func runWebsiteCapabilitiesScript(t *testing.T, relative string) {
	t.Helper()
	root := websiteJourneyRepoRoot(t)
	path := filepath.Join(root, relative)
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(abs)
	cmd.Dir = root
	if absRoot, err := filepath.Abs(root); err == nil {
		cmd.Dir = absRoot
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s: %v\n%s", relative, err, out)
	}
}
