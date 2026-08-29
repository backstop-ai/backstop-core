package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func validInventory() DeliveryInventory {
	return DeliveryInventory{
		SchemaVersion: deliveryInventorySchema,
		BaseCommit:    "89f4138aa97e7ba1e8d7c67595cfaee1caefa797",
		Entries: []DeliveryEntry{
			{Change: "A", Path: ".backstop/seed4-delivery-inventory.yml", Role: "delivery-inventory"},
			{Change: "A", Path: "scripts/sitecheck/inventory.go", Role: "structural-verifier"},
			{Change: "A", Path: "scripts/sitecheck/inventory_test.go", Role: "test"},
			{Change: "A", Path: "scripts/sitecheck/testdata/delivery-inventory-mutations.yml", Role: "test"},
		},
	}
}

func TestSiteCheck_Seed4DeliveryInventoryPasses(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	inventory, err := loadDeliveryInventory(filepath.Join(root, ".backstop", "seed4-delivery-inventory.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := validateDeliveryInventory(inventory); err != nil {
		t.Fatal(err)
	}
}

func TestSiteCheck_Seed4DeliveryInventoryRejectsInvalidMatrix(t *testing.T) {
	tests := []struct {
		name string
		edit func(*DeliveryInventory)
		want string
	}{
		{"short base", func(i *DeliveryInventory) { i.BaseCommit = "ca2ffc2" }, "base_commit"},
		{"duplicate path", func(i *DeliveryInventory) { i.Entries = append(i.Entries, i.Entries[0]) }, "duplicate path"},
		{"invalid change", func(i *DeliveryInventory) { i.Entries[0].Change = "C" }, "invalid change"},
		{"unknown role", func(i *DeliveryInventory) { i.Entries[0].Role = "mystery" }, "prohibited or unknown role"},
		{"role mismatch", func(i *DeliveryInventory) { i.Entries[0].Role = "layout" }, "role/path mismatch"},
		{"unlisted path", func(i *DeliveryInventory) { i.Entries[0].Path = "docs/secret-policy.yml" }, "outside the closed"},
		{"retired file modified", func(i *DeliveryInventory) {
			i.Entries[0] = DeliveryEntry{Change: "M", Path: "docs/assets/css/backstop-tokens.css", Role: "retired-bootstrap"}
		}, "deletion-only"},
		{"rename missing old path", func(i *DeliveryInventory) { i.Entries[0].Change = "R" }, "rename requires"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			inventory := validInventory()
			test.edit(&inventory)
			err := validateDeliveryInventory(inventory)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected error containing %q, got %v", test.want, err)
			}
		})
	}
}

func TestSiteCheck_RejectsMechanicalLocalPolicySurfacesAndProtectedBytes(t *testing.T) {
	for _, role := range []string{"visual-rule", "engine", "fixture-corpus", "token-declaration", "design-policy-validator"} {
		inventory := validInventory()
		inventory.Entries[0].Role = role
		if err := validateDeliveryInventory(inventory); err == nil || !strings.Contains(err.Error(), "prohibited") {
			t.Fatalf("role %q was not rejected: %v", role, err)
		}
	}
	inventory := validInventory()
	inventory.Entries[0] = DeliveryEntry{Change: "A", Path: ".backstop/packs/backstop-ai/backstop-design-system/rules/no-inline-styles.yml", Role: "test"}
	if err := validateDeliveryInventory(inventory); err == nil || !strings.Contains(err.Error(), "outside the closed") {
		t.Fatalf("installed owner bytes were not rejected: %v", err)
	}
}

func TestDeliveryInventory_LoadRejectsMissingAndUnknownFields(t *testing.T) {
	if _, err := loadDeliveryInventory(filepath.Join(t.TempDir(), "missing.yml")); err == nil || !strings.Contains(err.Error(), "read inventory") {
		t.Fatalf("missing inventory error = %v", err)
	}
	path := filepath.Join(t.TempDir(), "invalid.yml")
	if err := os.WriteFile(path, []byte("schema_version: x\nunknown: true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadDeliveryInventory(path); err == nil || !strings.Contains(err.Error(), "field unknown") {
		t.Fatalf("unknown field error = %v", err)
	}
}

func TestDeliveryInventory_RejectsRemainingShapeMutations(t *testing.T) {
	tests := []struct {
		name string
		edit func(*DeliveryInventory)
		want string
	}{
		{"schema", func(i *DeliveryInventory) { i.SchemaVersion = "wrong" }, "schema_version"},
		{"empty", func(i *DeliveryInventory) { i.Entries = nil }, "must not be empty"},
		{"absolute", func(i *DeliveryInventory) { i.Entries[0].Path = "/tmp/file" }, "repository-relative"},
		{"unclean", func(i *DeliveryInventory) { i.Entries[0].Path = "docs/../file" }, "repository-relative"},
		{"old path on add", func(i *DeliveryInventory) { i.Entries[0].OldPath = "old" }, "only for R"},
		{"duplicate rename old path", func(i *DeliveryInventory) {
			i.Entries[1] = DeliveryEntry{Change: "R", OldPath: i.Entries[0].Path, Path: "scripts/sitecheck/main.go", Role: "structural-verifier"}
		}, "unlisted old_path"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			inventory := validInventory()
			test.edit(&inventory)
			err := validateDeliveryInventory(inventory)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q, got %v", test.want, err)
			}
		})
	}
}

func TestDeliveryInventory_PathRoleMatrix(t *testing.T) {
	tests := map[string]string{
		"Gemfile": "build-dependency", "package.json": "build-dependency",
		".backstop/website-pack-releases.yml": "owner-release-import", "backstop.yml": "pack-declaration", "backstop.lock": "pack-lock",
		"docs/_data/site-presentation.yml": "site-data", "docs/_config.yml": "site-config",
		"docs/_layouts/default.html": "layout", "docs/_includes/site-header.html": "include",
		"docs/index.md": "page-wrapper", "docs/assets/css/site.css": "stylesheet-composition",
		"docs/assets/css/backstop.css": "stylesheet-composition", "docs/assets/css/backstop-tokens.css": "retired-bootstrap",
		"docs/index.html": "public-homepage", "playwright.config.ts": "browser-verification",
		"tests/public-site/site.spec.js": "browser-verification", "scripts/sitecheck/check.go": "structural-verifier",
		"scripts/producttruth/generate.go": "structural-verifier",
		"scripts/sitecheck/check_test.go":  "test", "scripts/sitecheck/testdata/case.yml": "test",
		"scripts/render-public-site-contracts/main.go":      "rendered-contract-stamper",
		"scripts/render-public-site-contracts/main_test.go": "test",
		"scripts/verify-public-site.sh":                     "verification-entrypoint", "scripts/install-design-assets.sh": "owner-asset-installer",
		"scripts/verify-public-product-model.sh":                                           "verification-entrypoint",
		"scripts/tests/public-product-model/pages/discovery-evaluation-adoption-status.sh": "test",
		"scripts/tests/public-product-model/pages/extend-reference-contributing.sh":        "test",
		"scripts/tests/public-product-model/pages/model-use-cases-packs.sh":                "test",
		"scripts/verify-documentation-semantics-integration.sh":                            "release-evidence-verifier",
		".github/workflows/pages.yml":                                                      "workflow", ".github/workflows/site-verification.yml": "workflow",
		".github/pages-actions.lock.yml":  "action-lock",
		"scripts/stamp-pages-artifact.sh": "deploy-stamp", "scripts/verify-pages-deployment.sh": "deploy-verifier",
		".cursor/Dockerfile":       "agent-environment",
		".cursor/environment.json": "agent-environment",
		".cursor/install.sh":       "agent-environment",
	}
	for path, role := range tests {
		if got := expectedRole(path); got != role {
			t.Errorf("expectedRole(%q) = %q, want %q", path, got, role)
		}
	}
	if got := expectedRole("docs/unlisted.md"); got != "" {
		t.Fatalf("unlisted role = %q", got)
	}
}

func TestDeliveryInventory_ISSUE190GovernanceArtifactsAreClosedRows(t *testing.T) {
	const issuePath = "issues/ISSUE-190-restore-canonical-homepage-direction.issue.md"
	const planPath = "plans/PLAN-ISSUE-190-restore-canonical-homepage-direction.plan.yml"
	for _, path := range []string{issuePath, planPath} {
		if got := expectedRole(path); got != "governance-artifact" {
			t.Fatalf("expectedRole(%q) = %q, want governance-artifact", path, got)
		}
	}
	for _, path := range []string{
		"issues/ISSUE-191-unrelated.issue.md",
		"plans/PLAN-ISSUE-191-unrelated.plan.yml",
		"issues/nested/ISSUE-190-restore-canonical-homepage-direction.issue.md",
	} {
		if got := expectedRole(path); got != "" {
			t.Fatalf("unrelated governance path %q admitted as %q", path, got)
		}
	}

	root := filepath.Clean(filepath.Join("..", ".."))
	inventory, err := loadDeliveryInventory(filepath.Join(root, ".backstop", "seed4-delivery-inventory.yml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, requiredPath := range []string{issuePath, planPath} {
		matched := 0
		for _, entry := range inventory.Entries {
			if entry.Path == requiredPath && entry.Change == "A" && entry.Role == "governance-artifact" {
				matched++
			}
		}
		if matched != 1 {
			t.Fatalf("required inventory row %q cardinality = %d, want 1", requiredPath, matched)
		}
		omitted := inventory
		omitted.Entries = append([]DeliveryEntry(nil), inventory.Entries...)
		for index, entry := range omitted.Entries {
			if entry.Path == requiredPath {
				omitted.Entries = append(omitted.Entries[:index], omitted.Entries[index+1:]...)
				break
			}
		}
		if err := validateInventoryMatchesDiff(root, omitted); err == nil || !strings.Contains(err.Error(), "inventory differs") {
			t.Fatalf("omitting %q did not fail exact diff matching: %v", requiredPath, err)
		}
	}
}

func TestDeliveryInventory_ISSUE191CursorAgentEnvironmentPathsAreClosedRows(t *testing.T) {
	cursorPaths := []string{".cursor/Dockerfile", ".cursor/environment.json", ".cursor/install.sh"}
	for _, path := range cursorPaths {
		if got := expectedRole(path); got != "agent-environment" {
			t.Fatalf("expectedRole(%q) = %q, want agent-environment", path, got)
		}
	}
	if !allowedRoles()["agent-environment"] {
		t.Fatal("allowedRoles() missing agent-environment")
	}
	if prohibitedRoles()["agent-environment"] {
		t.Fatal("prohibitedRoles() unexpectedly contains agent-environment")
	}
	for _, path := range []string{".cursorrules", "docs/.cursor/Dockerfile", "docs/secret-policy.yml"} {
		if got := expectedRole(path); got != "" {
			t.Fatalf("closed-matrix near-miss %q admitted as %q", path, got)
		}
	}

	root := filepath.Clean(filepath.Join("..", ".."))
	inventory, err := loadDeliveryInventory(filepath.Join(root, ".backstop", "seed4-delivery-inventory.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := validateDeliveryInventory(inventory); err != nil {
		t.Fatalf("validateDeliveryInventory: %v", err)
	}
	for _, requiredPath := range cursorPaths {
		matched := 0
		for _, entry := range inventory.Entries {
			if entry.Path == requiredPath && entry.Change == "A" && entry.Role == "agent-environment" {
				matched++
			}
		}
		if matched != 1 {
			t.Fatalf("required inventory row %q cardinality = %d, want 1", requiredPath, matched)
		}
	}
}

func TestDeliveryInventory_ISSUE191GovernanceArtifactsAreClosedRows(t *testing.T) {
	const issuePath = "issues/ISSUE-191-cursor-env-files-outside-seed4-matrix.issue.md"
	const planPath = "plans/PLAN-ISSUE-191-cursor-env-files-outside-seed4-matrix.plan.yml"
	for _, path := range []string{issuePath, planPath} {
		if got := expectedRole(path); got != "governance-artifact" {
			t.Fatalf("expectedRole(%q) = %q, want governance-artifact", path, got)
		}
	}
	for _, path := range []string{
		"issues/ISSUE-191-unrelated.issue.md",
		"plans/PLAN-ISSUE-191-unrelated.plan.yml",
	} {
		if got := expectedRole(path); got != "" {
			t.Fatalf("unrelated governance path %q admitted as %q", path, got)
		}
	}

	root := filepath.Clean(filepath.Join("..", ".."))
	inventory, err := loadDeliveryInventory(filepath.Join(root, ".backstop", "seed4-delivery-inventory.yml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, requiredPath := range []string{issuePath, planPath} {
		matched := 0
		for _, entry := range inventory.Entries {
			if entry.Path == requiredPath && entry.Change == "A" && entry.Role == "governance-artifact" {
				matched++
			}
		}
		if matched != 1 {
			t.Fatalf("required inventory row %q cardinality = %d, want 1", requiredPath, matched)
		}
		omitted := inventory
		omitted.Entries = append([]DeliveryEntry(nil), inventory.Entries...)
		for index, entry := range omitted.Entries {
			if entry.Path == requiredPath {
				omitted.Entries = append(omitted.Entries[:index], omitted.Entries[index+1:]...)
				break
			}
		}
		if err := validateInventoryMatchesDiff(root, omitted); err == nil || !strings.Contains(err.Error(), "inventory differs") {
			t.Fatalf("omitting %q did not fail exact diff matching: %v", requiredPath, err)
		}
	}
}

func TestDeliveryInventory_GitDiffAndMatch(t *testing.T) {
	root := t.TempDir()
	runGit := func(args ...string) string {
		t.Helper()
		command := exec.Command("git", append([]string{"-C", root}, args...)...)
		output, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, output)
		}
		return strings.TrimSpace(string(output))
	}
	runGit("init", "-q")
	runGit("config", "user.name", "Sitecheck Test")
	runGit("config", "user.email", "sitecheck@example.invalid")
	if err := os.MkdirAll(filepath.Join(root, "specs"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "specs/SPEC-075-static-public-site-design-system.spec.md"), []byte("status: ready-for-implementation\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "old"), []byte("same\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit("add", "old")
	runGit("commit", "-qm", "base")
	base := runGit("rev-parse", "HEAD")
	if err := os.Rename(filepath.Join(root, "old"), filepath.Join(root, "new")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "added"), []byte("new\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "specs/SPEC-075-static-public-site-design-system.spec.md"), []byte("status: implemented\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit("add", "-A")
	runGit("commit", "-qm", "change")
	entries, err := inventoryDiff(root, base)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || diffIdentity(entries[0]) == "" || diffIdentity(entries[1]) == "" {
		t.Fatalf("entries = %#v", entries)
	}
	inventory := DeliveryInventory{SchemaVersion: deliveryInventorySchema, BaseCommit: base, Entries: entries}
	if err := validateInventoryMatchesDiff(root, inventory); err != nil {
		t.Fatal(err)
	}
	inventory.Entries = inventory.Entries[:1]
	if err := validateInventoryMatchesDiff(root, inventory); err == nil || !strings.Contains(err.Error(), "inventory differs") {
		t.Fatalf("mismatch error = %v", err)
	}
	if _, err := inventoryDiff(root, strings.Repeat("0", 40)); err == nil {
		t.Fatal("unknown base unexpectedly passed")
	}
}
