package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type pagesActionsLock struct {
	SchemaVersion string            `yaml:"schema_version"`
	Actions       map[string]string `yaml:"actions"`
}

func requiredPagesActionCommits() map[string]string {
	return map[string]string{
		"actions/checkout":              "3d3c42e5aac5ba805825da76410c181273ba90b1",
		"ruby/setup-ruby":               "95ef2b042f9d7a56d8268cba8559e2842e2ad01b",
		"actions/setup-node":            "820762786026740c76f36085b0efc47a31fe5020",
		"actions/configure-pages":       "45bfe0192ca1faeb007ade9deae92b16b8254a0d",
		"actions/upload-pages-artifact": "fc324d3547104276b827a68afc52ff2a11cc49c9",
		"actions/deploy-pages":          "cd2ce8fcbc39b97be8ca5fce6e763baed58fa128",
	}
}

func VerifyPagesWorkflow(root string) []Finding {
	lockData, err := os.ReadFile(filepath.Join(root, ".github/pages-actions.lock.yml"))
	if err != nil {
		return []Finding{{Phase: "pages-workflow", Identity: "action lock", Expected: "readable", Observed: err.Error()}}
	}
	var lock pagesActionsLock
	decoder := yaml.NewDecoder(bytes.NewReader(lockData))
	decoder.KnownFields(true)
	if err := decoder.Decode(&lock); err != nil {
		return []Finding{{Phase: "pages-workflow", Identity: "action lock", Expected: "closed schema", Observed: err.Error()}}
	}
	var findings []Finding
	if lock.SchemaVersion != "backstop-core/pages-actions-lock/v1" {
		findings = append(findings, Finding{Phase: "pages-workflow", Identity: "action lock schema", Expected: "backstop-core/pages-actions-lock/v1", Observed: lock.SchemaVersion})
	}
	requiredActions := requiredPagesActionCommits()
	expected := make([]string, 0, len(requiredActions))
	for identity := range requiredActions {
		expected = append(expected, identity)
	}
	sort.Strings(expected)
	observed := make([]string, 0, len(lock.Actions))
	for identity, commit := range lock.Actions {
		observed = append(observed, identity)
		if !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(commit) {
			findings = append(findings, Finding{Phase: "pages-workflow", Identity: identity, Expected: "full lowercase 40-hex SHA", Observed: commit})
		}
		if want, ok := requiredActions[identity]; ok && commit != want {
			findings = append(findings, Finding{Phase: "pages-workflow", Identity: identity + " runtime-compatible pin", Expected: want, Observed: commit})
		}
	}
	sort.Strings(observed)
	if strings.Join(expected, "\n") != strings.Join(observed, "\n") {
		findings = append(findings, Finding{Phase: "pages-workflow", Identity: "action allowlist", Expected: strings.Join(expected, ","), Observed: strings.Join(observed, ",")})
	}

	pagesData, err := os.ReadFile(filepath.Join(root, ".github/workflows/pages.yml"))
	if err != nil {
		return append(findings, Finding{Phase: "pages-workflow", Identity: "deployment workflow", Expected: "readable", Observed: err.Error()})
	}
	siteData, err := os.ReadFile(filepath.Join(root, ".github/workflows/site-verification.yml"))
	if err != nil {
		return append(findings, Finding{Phase: "pages-workflow", Identity: "site verification workflow", Expected: "readable", Observed: err.Error()})
	}
	pagesWorkflow := string(pagesData)
	siteWorkflow := string(siteData)
	workflow := pagesWorkflow + "\n" + siteWorkflow

	usesPattern := regexp.MustCompile(`(?m)^\s*-?\s*uses:\s*([^@\s]+)@([^\s#]+)`)
	counts := map[string]int{}
	for _, match := range usesPattern.FindAllStringSubmatch(workflow, -1) {
		identity, commit := match[1], match[2]
		counts[identity]++
		want, ok := lock.Actions[identity]
		if !ok || commit != want {
			findings = append(findings, Finding{Phase: "pages-workflow", Identity: identity, Expected: want, Observed: commit})
		}
	}
	cardinality := map[string]int{
		"actions/checkout":              3,
		"ruby/setup-ruby":               2,
		"actions/setup-node":            2,
		"actions/configure-pages":       2,
		"actions/upload-pages-artifact": 1,
		"actions/deploy-pages":          1,
	}
	for identity, want := range cardinality {
		if counts[identity] != want {
			findings = append(findings, Finding{Phase: "pages-workflow", Identity: identity + " cardinality", Expected: fmt.Sprintf("%d", want), Observed: fmt.Sprintf("%d", counts[identity])})
		}
	}

	pagesRequired := []string{
		"branches: [main]", "workflow_dispatch:", "group: pages", "cancel-in-progress: false", "permissions: {}",
		"GOFLAGS: -mod=readonly", "fetch-depth: 0", "ruby-version: \"3.3.4\"", "pipx install semgrep==1.156.0",
		"BACKSTOP_SITE_OUTPUT: _site", "BACKSTOP_SITE_RETAIN: \"1\"", "./scripts/verify-public-site.sh",
		"artifact-id: ${{ steps.upload.outputs.artifact_id }}", "path: _site", "include-hidden-files: true",
		"pages: write", "id-token: write", "actions: read", "deployments: read", "needs: [build, deploy]",
		"./scripts/verify-pages-deployment.sh", "--artifact-id \"${{ needs.build.outputs.artifact-id }}\"",
	}
	for _, needle := range pagesRequired {
		if strings.Count(pagesWorkflow, needle) != 1 {
			findings = append(findings, Finding{Phase: "pages-workflow", Identity: needle, Expected: "exactly 1 in deployment workflow", Observed: fmt.Sprintf("%d", strings.Count(pagesWorkflow, needle))})
		}
	}
	if strings.Contains(pagesWorkflow, "uses: ./.github/workflows/site-verification.yml") {
		findings = append(findings, Finding{Phase: "pages-workflow", Identity: "reusable verification boundary", Expected: "absent from deployment workflow", Observed: "present"})
	}

	siteRequired := []string{
		"pull_request:", "contents: read", "pages: read", "GOFLAGS: -mod=readonly", "fetch-depth: 0", "ruby-version: \"3.3.4\"",
		"pipx install semgrep==1.156.0", "BACKSTOP_SITE_OUTPUT: _site", "BACKSTOP_SITE_RETAIN: \"0\"", "./scripts/verify-public-site.sh",
	}
	for _, needle := range siteRequired {
		if strings.Count(siteWorkflow, needle) != 1 {
			findings = append(findings, Finding{Phase: "pages-workflow", Identity: needle, Expected: "exactly 1 in site verification workflow", Observed: fmt.Sprintf("%d", strings.Count(siteWorkflow, needle))})
		}
	}
	for _, forbidden := range []string{"workflow_call:", "actions/upload-pages-artifact@", "actions/deploy-pages@", "pages: write", "id-token: write"} {
		if strings.Contains(siteWorkflow, forbidden) {
			findings = append(findings, Finding{Phase: "pages-workflow", Identity: "site verification isolation: " + forbidden, Expected: "absent", Observed: "present"})
		}
	}
	if regexp.MustCompile(`(?m)^\s*tags:`).MatchString(workflow) {
		findings = append(findings, Finding{Phase: "pages-workflow", Identity: "tag trigger", Expected: "absent", Observed: "present"})
	}
	if strings.Contains(workflow, "static_site_generator:") {
		findings = append(findings, Finding{Phase: "pages-workflow", Identity: "configure-pages generator injection", Expected: "absent for locked Jekyll build", Observed: "present"})
	}
	return findings
}
