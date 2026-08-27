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
	lockPath := filepath.Join(root, ".github/pages-actions.lock.yml")
	lockData, err := os.ReadFile(lockPath)
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

	workflowData, err := os.ReadFile(filepath.Join(root, ".github/workflows/pages.yml"))
	if err != nil {
		return append(findings, Finding{Phase: "pages-workflow", Identity: "workflow", Expected: "readable", Observed: err.Error()})
	}
	workflow := string(workflowData)
	usesPattern := regexp.MustCompile(`(?m)^\s*-?\s*uses:\s*([^@\s]+)@([^\s#]+)`)
	uses := usesPattern.FindAllStringSubmatch(workflow, -1)
	counts := map[string]int{}
	for _, match := range uses {
		identity, commit := match[1], match[2]
		counts[identity]++
		want, ok := lock.Actions[identity]
		if !ok || commit != want {
			findings = append(findings, Finding{Phase: "pages-workflow", Identity: identity, Expected: want, Observed: commit})
		}
	}
	for _, identity := range expected {
		want := 1
		if identity == "actions/checkout" {
			want = 2
		}
		if counts[identity] != want {
			findings = append(findings, Finding{Phase: "pages-workflow", Identity: identity + " cardinality", Expected: fmt.Sprintf("%d", want), Observed: fmt.Sprintf("%d", counts[identity])})
		}
	}
	requiredText := []string{
		"branches: [main]", "workflow_dispatch:", "group: pages", "cancel-in-progress: false",
		"permissions: {}", "contents: read", "pages: read", "pages: write", "id-token: write", "actions: read", "deployments: read",
		"gh api \"repos/${GITHUB_REPOSITORY}/pages\" --jq .build_type", "[ \"$mode\" != \"workflow\" ]",
		"fetch-depth: 0", "ruby-version: \"3.3.4\"", "GOFLAGS: -mod=readonly", "pipx install semgrep==1.156.0", "BACKSTOP_SITE_OUTPUT: _site",
		"BACKSTOP_SITE_RETAIN: \"1\"", "BACKSTOP_SITE_COMMIT: ${{ github.sha }}",
		"path: _site", "include-hidden-files: true", "needs: [build, deploy]",
		"./scripts/verify-pages-deployment.sh", "--artifact-id \"${{ needs.build.outputs.artifact-id }}\"",
	}
	for _, needle := range requiredText {
		want := 1
		if needle == "contents: read" || needle == "pages: read" {
			want = 2
		}
		if strings.Count(workflow, needle) != want {
			findings = append(findings, Finding{Phase: "pages-workflow", Identity: needle, Expected: fmt.Sprintf("exactly %d", want), Observed: fmt.Sprintf("%d", strings.Count(workflow, needle))})
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
