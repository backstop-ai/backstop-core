package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

const (
	testDeployCommit = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	testDeployRunID  = "12345"
)

type deployedIdentityMutationsFile struct {
	InvalidDeployments []DeployedIdentityMutation `yaml:"invalid_deployments"`
}

func loadDeployedIdentityMutations(t *testing.T) deployedIdentityMutationsFile {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "deployed-identity-mutations.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var document deployedIdentityMutationsFile
	if err := yaml.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	if len(document.InvalidDeployments) < 8 {
		t.Fatal("deployed-identity-mutations.yml must cover origin, redirect, marker, and cheat cases")
	}
	return document
}

func TestWebsiteJourney_DeployedSiteAllJourneysPass(t *testing.T) {
	built, m := mustAcceptedBuiltTree(t)
	stamped := t.TempDir()
	if err := copyBuiltTree(built, stamped); err != nil {
		t.Fatal(err)
	}
	if err := StampDeployedIdentity(stamped, testDeployCommit, testDeployRunID); err != nil {
		t.Fatal(err)
	}
	if err := TraverseDeployedSite(m, DeployedRequest{
		Origin: CanonicalDeployedOrigin,
		Commit: testDeployCommit,
		RunID:  testDeployRunID,
		Fetch:  FixtureDeployedFetcher(stamped, CanonicalDeployedOrigin, ""),
	}); err != nil {
		t.Fatal(err)
	}
	document := loadDeployedIdentityMutations(t)
	for _, mutation := range document.InvalidDeployments {
		t.Run(mutation.Name, func(t *testing.T) {
			copyDir := t.TempDir()
			if err := copyBuiltTree(built, copyDir); err != nil {
				t.Fatal(err)
			}
			req, err := ApplyDeployedIdentityMutation(copyDir, testDeployCommit, testDeployRunID, mutation)
			if err != nil {
				t.Fatal(err)
			}
			err = TraverseDeployedSite(m, req)
			if err == nil {
				t.Fatal("accepted invalid deployed identity")
			}
			if !strings.Contains(err.Error(), mutation.ExpectedError) {
				t.Fatalf("error %q does not name %q", err, mutation.ExpectedError)
			}
			if strings.Contains(strings.ToLower(err.Error()), "rolled back") {
				t.Fatalf("must not claim rollback: %v", err)
			}
		})
	}
}

func TestWebsiteJourney_DeployedFetcherRejectsNonCanonicalOrigin(t *testing.T) {
	status, body, err := DefaultDeployedFetcher()("http://example.invalid/")
	if err == nil || status != 0 || body != "" {
		t.Fatalf("non-canonical fetch: status=%d body=%q err=%v", status, body, err)
	}
	if !strings.Contains(err.Error(), CanonicalDeployedOrigin) {
		t.Fatalf("error %q does not name %s", err, CanonicalDeployedOrigin)
	}
}

func TestWebsiteJourney_DeployedSiteRejectsMalformedIdentity(t *testing.T) {
	_, m := mustAcceptedBuiltTree(t)
	err := TraverseDeployedSite(m, DeployedRequest{
		Origin: CanonicalDeployedOrigin,
		Commit: "not-a-sha",
		RunID:  "0",
		Fetch:  FixtureDeployedFetcher(t.TempDir(), CanonicalDeployedOrigin, ""),
	})
	if err == nil || !strings.Contains(err.Error(), "commit/run identity") {
		t.Fatalf("malformed identity: %v", err)
	}
}

func TestWebsiteJourney_StampRejectsMalformedAndDuplicateMarkers(t *testing.T) {
	built, _ := mustAcceptedBuiltTree(t)
	if err := StampDeployedIdentity(built, "not-a-sha", testDeployRunID); err == nil {
		t.Fatal("accepted malformed commit")
	}
	if err := StampDeployedIdentity(built, testDeployCommit, testDeployRunID); err != nil {
		t.Fatal(err)
	}
	if err := StampDeployedIdentity(built, testDeployCommit, testDeployRunID); err == nil || !strings.Contains(err.Error(), "already present") {
		t.Fatalf("duplicate stamp: %v", err)
	}
}

func TestWebsiteJourney_FixtureFetcherCoversRouteNormalization(t *testing.T) {
	built, _ := mustAcceptedBuiltTree(t)
	fetch := FixtureDeployedFetcher(built, CanonicalDeployedOrigin, "")
	if _, _, err := fetch("https://example.invalid/"); err == nil {
		t.Fatal("accepted non-canonical fixture URL")
	}
	status, body, err := fetch(CanonicalDeployedOrigin)
	if err != nil || status != 200 || body == "" {
		t.Fatalf("origin root: status=%d err=%v", status, err)
	}
	status, _, err = fetch(CanonicalDeployedOrigin + "/status")
	if err != nil || status != 200 {
		t.Fatalf("slash normalization: status=%d err=%v", status, err)
	}
	if err := os.Remove(BuiltRoutePath(built, "/reference/")); err != nil {
		t.Fatal(err)
	}
	status, _, err = fetch(CanonicalDeployedOrigin + "/reference/")
	if err != nil || status != 404 {
		t.Fatalf("missing file: status=%d err=%v", status, err)
	}
}

func TestWebsiteJourney_DeployedHTTPFetcherHonorsStatusAndRedirectBan(t *testing.T) {
	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.Path == "/status/" {
				return &http.Response{
					StatusCode: http.StatusFound,
					Header:     http.Header{"Location": []string{CanonicalDeployedOrigin + "/"}},
					Body:       io.NopCloser(strings.NewReader("")),
					Request:    req,
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("ok")),
				Request:    req,
			}, nil
		}),
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return fmt.Errorf("deployed: redirect following is prohibited")
		},
	}
	fetch := DeployedFetcherWithClient(client)
	status, body, err := fetch(CanonicalDeployedOrigin + "/")
	if err != nil || status != http.StatusOK || body != "ok" {
		t.Fatalf("canonical GET: status=%d body=%q err=%v", status, body, err)
	}
	_, _, err = fetch(CanonicalDeployedOrigin + "/status/")
	if err == nil || !strings.Contains(err.Error(), "redirect") {
		t.Fatalf("redirect: %v", err)
	}
}

func TestWebsiteJourney_WorkflowWiringPasses(t *testing.T) {
	root := websiteJourneyRepoRoot(t)
	absRoot, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(filepath.Join(absRoot, "scripts", "tests", "website-capabilities", "workflow-wiring.sh"))
	cmd.Dir = absRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("workflow wiring: %v\n%s", err, out)
	}
}
