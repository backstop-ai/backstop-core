package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestWebsiteJourney_VerificationCleanupPasses(t *testing.T) {
	out := runWebsiteCapabilitiesScript(t, "scripts/tests/website-capabilities/verify-wrapper.sh")
	if !strings.Contains(out, "verify-wrapper: ok") {
		t.Fatalf("cleanup wrapper missing success evidence: %s", out)
	}
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
	root, wrapper := websiteJourneyWrapper(t)
	for _, total := range []string{"80.00", "80.01", "100.00"} {
		cmd := exec.Command(wrapper, "--accept-coverage", total)
		cmd.Dir = root
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("coverage %s must pass: %v\n%s", total, err, out)
		}
		if !strings.Contains(string(out), "accepted") {
			t.Fatalf("coverage %s: missing acceptance evidence: %s", total, out)
		}
	}
	cmd := exec.Command(wrapper, "--parse-coverage")
	cmd.Dir = root
	cmd.Stdin = strings.NewReader("ok\tpkg\t0.1s\tcoverage: 80.00% of statements")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("parse coverage: %v\n%s", err, out)
	}
	if got := strings.TrimSpace(string(out)); got != "80.00" {
		t.Fatalf("parse coverage = %q, want 80.00", got)
	}
}

func TestWebsiteJourney_VerifierRejectsCoverageFailureMatrix(t *testing.T) {
	root, wrapper := websiteJourneyWrapper(t)
	for _, test := range []struct {
		total string
		token string
	}{
		{"79.99", "79.99"},
		{"absent", "absent"},
		{"duplicate", "duplicate"},
		{"nonnumeric", "nonnumeric"},
	} {
		cmd := exec.Command(wrapper, "--accept-coverage", test.total)
		cmd.Dir = root
		out, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatalf("accepted coverage %s", test.total)
		}
		if !strings.Contains(string(out), test.token) {
			t.Fatalf("coverage %s: error %q does not name %q", test.total, out, test.token)
		}
	}
	for _, test := range []struct {
		input string
		want  string
	}{
		{"no coverage here", "absent"},
		{"coverage: 80.00% of statements\ncoverage: 81.00% of statements", "duplicate"},
	} {
		cmd := exec.Command(wrapper, "--parse-coverage")
		cmd.Dir = root
		cmd.Stdin = strings.NewReader(test.input)
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("parse %q: %v\n%s", test.input, err, out)
		}
		if got := strings.TrimSpace(string(out)); got != test.want {
			t.Fatalf("parse %q = %q, want %q", test.input, got, test.want)
		}
	}
}

func websiteJourneyWrapper(t *testing.T) (root, wrapper string) {
	t.Helper()
	absRoot, err := filepath.Abs(websiteJourneyRepoRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	return absRoot, filepath.Join(absRoot, "scripts", "verify-website-capabilities.sh")
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

func runWebsiteCapabilitiesScript(t *testing.T, relative string) string {
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
	return string(out)
}
