package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const deliveryInventorySchema = "backstop-core/seed4-delivery-inventory/v1"

type DeliveryInventory struct {
	SchemaVersion string          `yaml:"schema_version"`
	BaseCommit    string          `yaml:"base_commit"`
	Entries       []DeliveryEntry `yaml:"entries"`
}

type DeliveryEntry struct {
	Change  string `yaml:"change"`
	OldPath string `yaml:"old_path,omitempty"`
	Path    string `yaml:"path"`
	Role    string `yaml:"role"`
}

func allowedRoles() map[string]bool {
	return map[string]bool{
		"action-lock": true, "browser-verification": true, "build-dependency": true,
		"delivery-inventory": true, "deploy-stamp": true, "deploy-verifier": true,
		"include": true, "layout": true, "owner-asset-installer": true,
		"owner-release-import": true, "pack-declaration": true, "pack-lock": true,
		"page-wrapper": true, "rendered-contract-stamper": true, "retired-bootstrap": true,
		"release-evidence-verifier": true,
		"site-config":               true, "site-data": true, "stylesheet-composition": true,
		"structural-verifier": true, "test": true, "verification-entrypoint": true,
		"workflow": true,
	}
}

func prohibitedRoles() map[string]bool {
	return map[string]bool{
		"design-policy-validator": true, "engine": true, "fixture-corpus": true,
		"token-declaration": true, "visual-rule": true,
	}
}

func pageWrappers() map[string]bool {
	return map[string]bool{
		"docs/index.md": true, "docs/evaluate.md": true, "docs/model.md": true,
		"docs/adopt.md": true, "docs/use-cases.md": true, "docs/packs.md": true,
		"docs/extend.md": true, "docs/reference.md": true, "docs/status.md": true,
		"docs/contributing.md": true, "docs/getting-started.md": true,
		"docs/concepts.md": true, "docs/artifact-workflow.md": true,
		"docs/pack-authoring.md": true, "docs/cli-reference.md": true,
	}
}

func retiredBootstrap() map[string]bool {
	return map[string]bool{
		"docs/index.html":                     true,
		"docs/assets/css/backstop.css":        true,
		"docs/assets/css/backstop-tokens.css": true,
	}
}

func seed4LedgerPaths() map[string]bool {
	return map[string]bool{
		"specs/SPEC-074-derived-product-truth-pipeline.spec.md":         true,
		"plans/PLAN-SPEC-074-derived-product-truth-pipeline.plan.yml":   true,
		"specs/SPEC-075-static-public-site-design-system.spec.md":       true,
		"plans/PLAN-SPEC-075-static-public-site-design-system.plan.yml": true,
	}
}

func loadDeliveryInventory(path string) (DeliveryInventory, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return DeliveryInventory{}, fmt.Errorf("read inventory: %w", err)
	}
	var inventory DeliveryInventory
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&inventory); err != nil {
		return DeliveryInventory{}, fmt.Errorf("decode inventory: %w", err)
	}
	return inventory, nil
}

func validateDeliveryInventory(inventory DeliveryInventory) error {
	if inventory.SchemaVersion != deliveryInventorySchema {
		return fmt.Errorf("schema_version: expected %q, observed %q", deliveryInventorySchema, inventory.SchemaVersion)
	}
	if !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(inventory.BaseCommit) {
		return fmt.Errorf("base_commit: expected a full lowercase 40-hex commit, observed %q", inventory.BaseCommit)
	}
	if len(inventory.Entries) == 0 {
		return errors.New("entries: inventory must not be empty")
	}
	seen := map[string]bool{}
	for index, entry := range inventory.Entries {
		label := fmt.Sprintf("entries[%d] path=%q role=%q", index, entry.Path, entry.Role)
		if entry.Change != "A" && entry.Change != "M" && entry.Change != "D" && entry.Change != "R" {
			return fmt.Errorf("%s: invalid change %q", label, entry.Change)
		}
		if entry.Path == "" || filepath.IsAbs(entry.Path) || filepath.Clean(entry.Path) != entry.Path || strings.HasPrefix(entry.Path, "../") {
			return fmt.Errorf("%s: path must be a clean repository-relative path", label)
		}
		if seen[entry.Path] {
			return fmt.Errorf("%s: duplicate path", label)
		}
		seen[entry.Path] = true
		if prohibitedRoles()[entry.Role] || !allowedRoles()[entry.Role] {
			return fmt.Errorf("%s: prohibited or unknown role", label)
		}
		if entry.Change == "R" {
			if entry.OldPath == "" || entry.OldPath == entry.Path || seen[entry.OldPath] {
				return fmt.Errorf("%s: rename requires one distinct, unlisted old_path", label)
			}
			seen[entry.OldPath] = true
		} else if entry.OldPath != "" {
			return fmt.Errorf("%s: old_path is valid only for R", label)
		}
		if err := validatePathRole(entry); err != nil {
			return fmt.Errorf("%s: %w", label, err)
		}
	}
	return nil
}

func validatePathRole(entry DeliveryEntry) error {
	expected := expectedRole(entry.Path)
	if expected == "" {
		return errors.New("path is outside the closed Seed 4 matrix")
	}
	if expected != entry.Role {
		return fmt.Errorf("role/path mismatch: expected %q", expected)
	}
	if entry.Role == "retired-bootstrap" && entry.Change != "D" {
		return errors.New("retired-bootstrap is deletion-only")
	}
	return nil
}

func expectedRole(path string) string {
	switch {
	case path == ".backstop/seed4-delivery-inventory.yml":
		return "delivery-inventory"
	case path == ".backstop/website-pack-releases.yml":
		return "owner-release-import"
	case path == "backstop.yml":
		return "pack-declaration"
	case path == "backstop.lock":
		return "pack-lock"
	case path == "Gemfile" || path == "Gemfile.lock" || path == "package"+".json" || path == "package-lock"+".json":
		return "build-dependency"
	case path == "docs/_data/site-presentation.yml":
		return "site-data"
	case path == "docs/_config.yml" || path == "docs/CNAME":
		return "site-config"
	case strings.HasPrefix(path, "docs/_layouts/"):
		return "layout"
	case strings.HasPrefix(path, "docs/_includes/"):
		return "include"
	case pageWrappers()[path]:
		return "page-wrapper"
	case path == "docs/assets/css/site.css":
		return "stylesheet-composition"
	case retiredBootstrap()[path]:
		return "retired-bootstrap"
	case path == "playwright.config.ts" || path == "tests/public-site.spec.ts" || path == "tests/public-site-helpers.ts" || strings.HasPrefix(path, "tests/public-site/"):
		return "browser-verification"
	case strings.HasPrefix(path, "scripts/sitecheck/testdata/") || (strings.HasPrefix(path, "scripts/sitecheck/") && strings.HasSuffix(path, "_test.go")):
		return "test"
	case strings.HasPrefix(path, "scripts/sitecheck/"):
		return "structural-verifier"
	case path == "scripts/producttruth/generate.go":
		return "structural-verifier"
	case strings.HasPrefix(path, "scripts/render-public-site-contracts/") && strings.HasSuffix(path, "_test.go"):
		return "test"
	case strings.HasPrefix(path, "scripts/render-public-site-contracts/"):
		return "rendered-contract-stamper"
	case path == "scripts/verify-public-site.sh":
		return "verification-entrypoint"
	case path == "scripts/verify-public-product-model.sh":
		return "verification-entrypoint"
	case path == "scripts/tests/public-product-model/pages/discovery-evaluation-adoption-status.sh" ||
		path == "scripts/tests/public-product-model/pages/model-use-cases-packs.sh" ||
		path == "scripts/tests/public-product-model/pages/extend-reference-contributing.sh":
		return "test"
	case path == "scripts/verify-documentation-semantics-integration.sh":
		return "release-evidence-verifier"
	case path == "scripts/install-design-assets.sh":
		return "owner-asset-installer"
	case path == ".github/workflows/pages.yml" || path == ".github/workflows/site-verification.yml":
		return "workflow"
	case path == ".github/pages-actions.lock.yml":
		return "action-lock"
	case path == "scripts/stamp-pages-artifact.sh":
		return "deploy-stamp"
	case path == "scripts/verify-pages-deployment.sh":
		return "deploy-verifier"
	default:
		return ""
	}
}

func inventoryDiff(root, base string) ([]DeliveryEntry, error) {
	command := exec.Command("git", "-C", root, "diff", "--name-status", "--find-renames=100%", base+"...HEAD")
	output, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff from %s: %w", base, err)
	}
	var entries []DeliveryEntry
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		change := fields[0]
		if strings.HasPrefix(change, "R") && len(fields) == 3 {
			if seed4LedgerPaths()[fields[1]] || seed4LedgerPaths()[fields[2]] {
				return nil, fmt.Errorf("governed ledger path may not be renamed: %q", line)
			}
			entries = append(entries, DeliveryEntry{Change: "R", OldPath: fields[1], Path: fields[2]})
			continue
		}
		if len(fields) != 2 || (change != "A" && change != "M" && change != "D") {
			return nil, fmt.Errorf("unrecognized git diff row %q", line)
		}
		if seed4LedgerPaths()[fields[1]] {
			continue
		}
		entries = append(entries, DeliveryEntry{Change: change, Path: fields[1]})
	}
	return entries, nil
}

func validateInventoryMatchesDiff(root string, inventory DeliveryInventory) error {
	actual, err := inventoryDiff(root, inventory.BaseCommit)
	if err != nil {
		return err
	}
	want := make([]string, 0, len(inventory.Entries))
	got := make([]string, 0, len(actual))
	for _, entry := range inventory.Entries {
		want = append(want, diffIdentity(entry))
	}
	for _, entry := range actual {
		got = append(got, diffIdentity(entry))
	}
	sort.Strings(want)
	sort.Strings(got)
	if strings.Join(want, "\n") != strings.Join(got, "\n") {
		return fmt.Errorf("inventory differs from git diff\nexpected:\n%s\nobserved:\n%s", strings.Join(want, "\n"), strings.Join(got, "\n"))
	}
	return nil
}

func diffIdentity(entry DeliveryEntry) string {
	if entry.Change == "R" {
		return strings.Join([]string{entry.Change, entry.OldPath, entry.Path}, "\t")
	}
	return strings.Join([]string{entry.Change, entry.Path}, "\t")
}
