package validate_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ISSUE-018 Section F deletion-assertion tests. These pin the ABSENCE of the
// dead native-standards validator cluster so it cannot be silently
// reintroduced (the SPEC-034-style deletion guard). They are red while the
// symbols still exist and go green only after the Section F deletion lands.

// nonTestGoSources returns the contents of every non-test .go file in the
// current package directory (tests run with CWD == the package dir), keyed by
// file name, so the assertions scan production source only.
func nonTestGoSources(t *testing.T) map[string]string {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("reading package dir: %v", err)
	}
	out := map[string]string{}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		b, rerr := os.ReadFile(name)
		if rerr != nil {
			t.Fatalf("reading %s: %v", name, rerr)
		}
		out[name] = string(b)
	}
	return out
}

// TestNativeStandardsValidator_Removed proves the validate.Standard
// native-standards validator function is gone from every non-test source under
// pkg/validate. CLM-001.
func TestNativeStandardsValidator_Removed(t *testing.T) {
	for name, src := range nonTestGoSources(t) {
		if strings.Contains(src, "func Standard(") {
			t.Errorf("%s still defines `func Standard(`; the native-standards validator must be deleted", name)
		}
	}
	if _, err := os.Stat("standard.go"); err == nil {
		t.Error("pkg/validate/standard.go still exists; the native-standards validator file must be deleted")
	}
}

// TestConfig_StandardsDirs_Removed proves the Config struct no longer carries a
// StandardsDirs field / standards_dirs yaml tag. CLM-002. It scans the config
// package source via a relative path from this package directory.
func TestConfig_StandardsDirs_Removed(t *testing.T) {
	configSrc := readSiblingPackageSource(t, "config", "config.go")
	if strings.Contains(configSrc, "StandardsDirs") {
		t.Error("pkg/config/config.go still declares the StandardsDirs field; it must be removed (zero production readers)")
	}
	if strings.Contains(configSrc, "standards_dirs") {
		t.Error("pkg/config/config.go still declares the standards_dirs yaml tag; it must be removed")
	}
}

// TestPlan_FileCategory_NoStandardMd proves the fileCategory artifactExts list
// in pkg/validate/plan.go no longer contains ".standard.md" while the
// fileCategory function itself is preserved. CLM-003 / CLM-011.
func TestPlan_FileCategory_NoStandardMd(t *testing.T) {
	srcs := nonTestGoSources(t)
	planSrc, ok := srcs["plan.go"]
	if !ok {
		t.Fatal("pkg/validate/plan.go not found; fileCategory must remain in this file")
	}
	if !strings.Contains(planSrc, "func fileCategory(") {
		t.Error("fileCategory function was removed; only the .standard.md ext entry should be dropped, not the function")
	}
	if strings.Contains(planSrc, `".standard.md"`) {
		t.Error("plan.go fileCategory artifactExts still lists \".standard.md\"; that entry must be removed")
	}
}

// readSiblingPackageSource reads a source file from a sibling package directory
// (../<pkg>/<file>), so a pkg/validate test can assert on pkg/config source.
func readSiblingPackageSource(t *testing.T, pkg, file string) string {
	t.Helper()
	p := filepath.Join("..", pkg, file)
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("reading %s: %v", p, err)
	}
	return string(b)
}
