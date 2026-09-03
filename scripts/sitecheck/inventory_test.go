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
	if err := validateInventoryMatchesDiff(root, inventory); err != nil {
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
		"docs/_data/entity-reference.yml":                   "site-data",
		"docs/plan.md":                                      "entity-page",
		"docs/issue.md":                                     "entity-page",
		"docs/spec.md":                                      "entity-page",
		"docs/bundle.md":                                    "entity-page",
		"docs/pack.md":                                      "entity-page",
		"docs/directive.md":                                 "entity-page",
		"docs/adr.md":                                       "entity-page",
		"docs/capability.md":                                "entity-page",
		"scripts/entityref/main.go":                         "entity-generator",
		"scripts/entityref/main_test.go":                    "test",
		"scripts/verify-public-site.sh":                     "verification-entrypoint", "scripts/install-design-assets.sh": "owner-asset-installer",
		"scripts/verify-public-product-model.sh":                                           "verification-entrypoint",
		"scripts/tests/public-product-model/pages/discovery-evaluation-adoption-status.sh": "test",
		"scripts/tests/public-product-model/pages/extend-reference-contributing.sh":        "test",
		"scripts/tests/public-product-model/pages/model-use-cases-packs.sh":                "test",
		"scripts/verify-documentation-semantics-integration.sh":                            "release-evidence-verifier",
		".github/workflows/pages.yml":                                                      "workflow", ".github/workflows/site-verification.yml": "workflow",
		".github/pages-actions.lock.yml":  "action-lock",
		"scripts/stamp-pages-artifact.sh": "deploy-stamp", "scripts/verify-pages-deployment.sh": "deploy-verifier",
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

func TestDeliveryInventory_ISSUE197EntityReferencePathsAreClosedRows(t *testing.T) {
	entityPaths := []string{
		"docs/plan.md", "docs/issue.md", "docs/spec.md", "docs/bundle.md",
		"docs/pack.md", "docs/directive.md", "docs/adr.md", "docs/capability.md",
	}
	for _, path := range entityPaths {
		if got := expectedRole(path); got != "entity-page" {
			t.Fatalf("expectedRole(%q) = %q, want entity-page", path, got)
		}
	}
	if got := expectedRole("scripts/entityref/main.go"); got != "entity-generator" {
		t.Fatalf("expectedRole(entityref main) = %q", got)
	}
	if got := expectedRole("scripts/entityref/main_test.go"); got != "test" {
		t.Fatalf("expectedRole(entityref test) = %q", got)
	}
	if got := expectedRole("docs/_data/entity-reference.yml"); got != "site-data" {
		t.Fatalf("expectedRole(entity overlay) = %q", got)
	}
	if !allowedRoles()["entity-page"] || !allowedRoles()["entity-generator"] {
		t.Fatal("allowedRoles must include entity-page and entity-generator")
	}
	if prohibitedRoles()["entity-page"] || prohibitedRoles()["entity-generator"] {
		t.Fatal("entity roles must not be prohibited")
	}
	if allowedRoles()["governance-artifact"] {
		t.Fatal("allowedRoles() must not include governance-artifact")
	}
	for _, path := range []string{"docs/unlisted.md", "docs/plan-notes.md", "docs/nested/plan.md", "scripts/entityref/README.md", "scripts/entityrefextra/main.go"} {
		if got := expectedRole(path); got != "" {
			t.Fatalf("expectedRole(%q) = %q, want empty", path, got)
		}
	}

	root := filepath.Clean(filepath.Join("..", ".."))
	inventory, err := loadDeliveryInventory(filepath.Join(root, ".backstop", "seed4-delivery-inventory.yml"))
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, entry := range inventory.Entries {
		counts[entry.Path]++
	}
	for _, path := range append([]string{"docs/_data/entity-reference.yml", "scripts/entityref/main.go", "scripts/entityref/main_test.go"}, entityPaths...) {
		if counts[path] != 1 {
			t.Fatalf("inventory path %q count = %d, want 1", path, counts[path])
		}
	}
	if err := validateDeliveryInventory(inventory); err != nil {
		t.Fatal(err)
	}
	omitted := inventory
	omitted.Entries = append([]DeliveryEntry(nil), inventory.Entries...)
	omitted.Entries = omitted.Entries[:len(omitted.Entries)-1]
	if err := validateInventoryMatchesDiff(root, omitted); err == nil || !strings.Contains(err.Error(), "inventory differs") {
		t.Fatalf("omitting a row did not fail: %v", err)
	}
}

func TestDeliveryInventory_ISSUE198PackFleetPathsAreClosedRows(t *testing.T) {
	packFleetPaths := []string{
		"docs/pack/examples.md",
		"docs/pack/guide.md",
		"docs/_includes/generated/published-pack-catalog.md",
	}
	for _, path := range packFleetPaths {
		if got := expectedRole(path); got != wantRoleForPackFleet(path) {
			t.Fatalf("expectedRole(%q) = %q, want %q", path, got, wantRoleForPackFleet(path))
		}
	}
	for _, path := range []string{"docs/packs.md", "docs/extend.md", "docs/pack-authoring.md"} {
		if got := expectedRole(path); got != "page-wrapper" {
			t.Fatalf("expectedRole(%q) = %q, want page-wrapper for deletion matrix", path, got)
		}
	}
	for _, path := range []string{
		"docs/_data/published-pack-inventory.yml",
		"docs/_data/content-topology.yml",
		"issues/ISSUE-198-pack-fleet.issue.md",
		"plans/PLAN-ISSUE-198-extend-visitor-page.plan.yml",
		"docs/pack/README.md",
		"docs/unlisted.md",
	} {
		if got := expectedRole(path); got != "" {
			t.Fatalf("expectedRole(%q) = %q, want empty (outside Seed 4 matrix)", path, got)
		}
	}

	root := filepath.Clean(filepath.Join("..", ".."))
	inventory, err := loadDeliveryInventory(filepath.Join(root, ".backstop", "seed4-delivery-inventory.yml"))
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, entry := range inventory.Entries {
		counts[entry.Path]++
	}
	for _, path := range packFleetPaths {
		if counts[path] != 1 {
			t.Fatalf("inventory path %q count = %d, want 1", path, counts[path])
		}
	}
	for _, path := range []string{"docs/extend.md", "docs/packs.md", "docs/pack-authoring.md", "docs/_includes/generated/installed-pack-catalog.md"} {
		if counts[path] != 1 {
			t.Fatalf("inventory deletion path %q count = %d, want 1", path, counts[path])
		}
	}
	if err := validateDeliveryInventory(inventory); err != nil {
		t.Fatal(err)
	}
	omitted := inventory
	omitted.Entries = append([]DeliveryEntry(nil), inventory.Entries...)
	omitted.Entries = omitted.Entries[:len(omitted.Entries)-1]
	if err := validateInventoryMatchesDiff(root, omitted); err == nil || !strings.Contains(err.Error(), "inventory differs") {
		t.Fatalf("omitting a pack fleet row did not fail: %v", err)
	}
}

func wantRoleForPackFleet(path string) string {
	if strings.HasPrefix(path, "docs/_includes/") {
		return "include"
	}
	return "page-wrapper"
}

func TestDeliveryInventory_ISSUE190GovernanceArtifactsAreClosedRows(t *testing.T) {
	outside := []string{
		"issues/ISSUE-190-restore-canonical-homepage-direction.issue.md",
		"plans/PLAN-ISSUE-190-restore-canonical-homepage-direction.plan.yml",
		"issues/ISSUE-191-unrelated.issue.md",
		"plans/PLAN-ISSUE-191-unrelated.plan.yml",
		"issues/nested/ISSUE-190-restore-canonical-homepage-direction.issue.md",
	}
	assertOutsideSeed4Matrix(t, outside)
	assertInventoryOmitsPaths(t, outside[:2])
}

func TestDeliveryInventory_ISSUE191CursorAgentEnvironmentPathsAreClosedRows(t *testing.T) {
	cursorPaths := []string{".cursor/Dockerfile", ".cursor/environment.json", ".cursor/install.sh"}
	assertOutsideSeed4Matrix(t, cursorPaths)
	if allowedRoles()["agent-environment"] {
		t.Fatal("allowedRoles() must not include agent-environment")
	}
	if allowedRoles()["governance-artifact"] {
		t.Fatal("allowedRoles() must not include governance-artifact")
	}
	for _, path := range []string{".cursorrules", "docs/.cursor/Dockerfile", "docs/secret-policy.yml"} {
		if got := expectedRole(path); got != "" {
			t.Fatalf("closed-matrix near-miss %q admitted as %q", path, got)
		}
	}
	assertInventoryOmitsPaths(t, cursorPaths)
}

func TestDeliveryInventory_ISSUE191GovernanceArtifactsAreClosedRows(t *testing.T) {
	outside := []string{
		"issues/ISSUE-191-cursor-env-files-outside-seed4-matrix.issue.md",
		"plans/PLAN-ISSUE-191-cursor-env-files-outside-seed4-matrix.plan.yml",
		"issues/ISSUE-191-unrelated.issue.md",
		"plans/PLAN-ISSUE-191-unrelated.plan.yml",
	}
	assertOutsideSeed4Matrix(t, outside)
	assertInventoryOmitsPaths(t, outside[:2])
}

func TestSiteCheck_DeliveryInventoryIgnoresPathsOutsideSeed4Matrix(t *testing.T) {
	assertOutsideSeed4Matrix(t, []string{
		".cursor/Dockerfile",
		".github/workflows/ci.yml",
		"scripts/websitejourney/main.go",
		"issues/ISSUE-190-restore-canonical-homepage-direction.issue.md",
		"plans/PLAN-ISSUE-191-cursor-env-files-outside-seed4-matrix.plan.yml",
	})
	root, base := seed4InventoryGitFixture(t, func(root string) {
		mustWrite(t, filepath.Join(root, "scripts", "sitecheck", "inventory.go"), "package main\n")
		mustWrite(t, filepath.Join(root, "scripts", "websitejourney", "extra.go"), "package main\n")
		mustWrite(t, filepath.Join(root, ".cursor", "Dockerfile"), "FROM scratch\n")
		mustWrite(t, filepath.Join(root, "specs", "SPEC-075-static-public-site-design-system.spec.md"), "status: implemented\n")
	})
	entries, err := inventoryDiff(root, base)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Change != "A" || entries[0].Path != "scripts/sitecheck/inventory.go" {
		t.Fatalf("filtered diff = %#v, want only scripts/sitecheck/inventory.go", entries)
	}
	inventory := DeliveryInventory{
		SchemaVersion: deliveryInventorySchema,
		BaseCommit:    base,
		Entries: []DeliveryEntry{
			{Change: "A", Path: "scripts/sitecheck/inventory.go", Role: "structural-verifier"},
		},
	}
	if err := validateInventoryMatchesDiff(root, inventory); err != nil {
		t.Fatal(err)
	}
	omitted := inventory
	omitted.Entries = nil
	if err := validateInventoryMatchesDiff(root, omitted); err == nil || !strings.Contains(err.Error(), "inventory differs") {
		t.Fatalf("omitting a Seed 4 path did not fail: %v", err)
	}
	extra := inventory
	extra.Entries = append(append([]DeliveryEntry(nil), inventory.Entries...), DeliveryEntry{
		Change: "A", Path: "scripts/websitejourney/extra.go", Role: "structural-verifier",
	})
	if err := validateDeliveryInventory(extra); err == nil || !strings.Contains(err.Error(), "outside the closed") {
		t.Fatalf("out-of-matrix inventory row was accepted: %v", err)
	}
}

func assertOutsideSeed4Matrix(t *testing.T, paths []string) {
	t.Helper()
	for _, path := range paths {
		if got := expectedRole(path); got != "" {
			t.Fatalf("expectedRole(%q) = %q, want empty (outside Seed 4 matrix)", path, got)
		}
	}
}

func assertInventoryOmitsPaths(t *testing.T, paths []string) {
	t.Helper()
	root := filepath.Clean(filepath.Join("..", ".."))
	inventory, err := loadDeliveryInventory(filepath.Join(root, ".backstop", "seed4-delivery-inventory.yml"))
	if err != nil {
		t.Fatal(err)
	}
	listed := map[string]bool{}
	for _, entry := range inventory.Entries {
		listed[entry.Path] = true
	}
	for _, path := range paths {
		if listed[path] {
			t.Fatalf("inventory still classifies out-of-matrix path %q", path)
		}
	}
}

func TestDeliveryInventory_GitDiffAndMatch(t *testing.T) {
	root, base := seed4InventoryGitFixture(t, func(root string) {
		mustWrite(t, filepath.Join(root, "scripts", "sitecheck", "inventory.go"), "package main\n")
		mustWrite(t, filepath.Join(root, "scripts", "websitejourney", "extra.go"), "package main\n")
		if err := os.Rename(filepath.Join(root, "old"), filepath.Join(root, "new")); err != nil {
			t.Fatal(err)
		}
		mustWrite(t, filepath.Join(root, "specs", "SPEC-075-static-public-site-design-system.spec.md"), "status: implemented\n")
	})
	entries, err := inventoryDiff(root, base)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || diffIdentity(entries[0]) != "A\tscripts/sitecheck/inventory.go" {
		t.Fatalf("entries = %#v, want only the Seed 4 sitecheck path", entries)
	}
	inventory := DeliveryInventory{SchemaVersion: deliveryInventorySchema, BaseCommit: base, Entries: entries}
	if err := validateInventoryMatchesDiff(root, inventory); err != nil {
		t.Fatal(err)
	}
	inventory.Entries = nil
	if err := validateInventoryMatchesDiff(root, inventory); err == nil || !strings.Contains(err.Error(), "inventory differs") {
		t.Fatalf("mismatch error = %v", err)
	}
	if _, err := inventoryDiff(root, strings.Repeat("0", 40)); err == nil {
		t.Fatal("unknown base unexpectedly passed")
	}

	ledgerRoot := t.TempDir()
	runGit := gitRunner(t, ledgerRoot)
	runGit("init", "-q")
	runGit("config", "user.name", "Sitecheck Test")
	runGit("config", "user.email", "sitecheck@example.invalid")
	mustWrite(t, filepath.Join(ledgerRoot, "specs", "SPEC-075-static-public-site-design-system.spec.md"), "status: ready-for-implementation\n")
	runGit("add", "specs/SPEC-075-static-public-site-design-system.spec.md")
	runGit("commit", "-qm", "base")
	ledgerBase := runGit("rev-parse", "HEAD")
	if err := os.Rename(
		filepath.Join(ledgerRoot, "specs", "SPEC-075-static-public-site-design-system.spec.md"),
		filepath.Join(ledgerRoot, "specs", "renamed.spec.md"),
	); err != nil {
		t.Fatal(err)
	}
	runGit("add", "-A")
	runGit("commit", "-qm", "rename ledger")
	if _, err := inventoryDiff(ledgerRoot, ledgerBase); err == nil || !strings.Contains(err.Error(), "may not be renamed") {
		t.Fatalf("ledger rename error = %v", err)
	}
}

func seed4InventoryGitFixture(t *testing.T, mutate func(root string)) (root, base string) {
	t.Helper()
	root = t.TempDir()
	runGit := gitRunner(t, root)
	runGit("init", "-q")
	runGit("config", "user.name", "Sitecheck Test")
	runGit("config", "user.email", "sitecheck@example.invalid")
	mustWrite(t, filepath.Join(root, "specs", "SPEC-075-static-public-site-design-system.spec.md"), "status: ready-for-implementation\n")
	mustWrite(t, filepath.Join(root, "old"), "same\n")
	runGit("add", "specs/SPEC-075-static-public-site-design-system.spec.md", "old")
	runGit("commit", "-qm", "base")
	base = runGit("rev-parse", "HEAD")
	mutate(root)
	runGit("add", "-A")
	runGit("commit", "-qm", "change")
	return root, base
}

func gitRunner(t *testing.T, root string) func(args ...string) string {
	t.Helper()
	return func(args ...string) string {
		t.Helper()
		command := exec.Command("git", append([]string{"-C", root}, args...)...)
		output, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, output)
		}
		return strings.TrimSpace(string(output))
	}
}

func mustWrite(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}
