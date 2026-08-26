package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/config"
	"github.com/backstop-ai/backstop-core/pkg/gate"
	"github.com/backstop-ai/backstop-core/pkg/pack"
)

func TestTestNameMatcher_AllMatchEnumerationRemainsLanguageNeutral(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(issue188Root(t), "pkg", "gate", "step_testverify.go"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(data)
	matcherStart := strings.Index(source, "func (m TestNameMatcher) FindNames(line string) []string {")
	matcherEndMarker := "\nfunc (m TestNameMatcher) FindName(line string) (string, bool) {"
	collectorStart := strings.Index(source, "func collectTestFuncNamesScoped(")
	if matcherStart < 0 || collectorStart < 0 {
		t.Fatal("matcher or collector source start boundary absent")
	}
	matcherEnd := strings.Index(source[matcherStart:], matcherEndMarker)
	collectorEnd := strings.Index(source[collectorStart:], "\n\n// SPEC-037")
	if matcherEnd <= 0 || collectorEnd <= 0 {
		t.Fatal("matcher or collector source end boundary absent")
	}
	implementation := source[matcherStart:matcherStart+matcherEnd] + source[collectorStart:collectorStart+collectorEnd]
	lines := strings.Split(implementation, "\n")
	for i, line := range lines {
		if comment := strings.Index(line, "//"); comment >= 0 {
			lines[i] = line[:comment]
		}
	}
	body := strings.ToLower(strings.Join(lines, "\n"))
	for _, forbidden := range []string{"bash", "typescript", "semicolon", "describe(", "func test", `".go"`, `".ts"`, `";"`} {
		if strings.Contains(body, forbidden) {
			t.Errorf("matcher/collector implementation contains baked syntax token %q", forbidden)
		}
	}
}

func TestTestNameMatcher_AllMatchAPIAndNoBashMechanismInCore(t *testing.T) {
	m, err := gate.NewTestNameMatcher([]string{`item:([A-Za-z]+)`})
	if err != nil {
		t.Fatal(err)
	}
	got := m.FindNames("item:One item:Two")
	if len(got) != 2 || got[0] != "One" || got[1] != "Two" {
		t.Fatalf("FindNames=%v", got)
	}
	_, fixtureRoot := issue188Fixture(t)
	wantFixtureSuffix := "cmd/backstop/testdata/issue188-bash-reference-pack"
	if !strings.HasSuffix(filepath.ToSlash(fixtureRoot), wantFixtureSuffix) {
		t.Fatalf("fixture escaped testdata: %s", fixtureRoot)
	}
	root := issue188Root(t)
	cfg, err := config.LoadConfigFromPath(filepath.Join(root, "backstop.yml"))
	if err != nil {
		t.Fatal(err)
	}
	// Core may consume the published Bash mechanism by external coordinate. What this
	// fence forbids is a second Bash mechanism, an embedded manifest, or baked production
	// source. Registration therefore proves the intended boundary instead of violating it.
	if version := cfg.Packs["backstop-ai/bash-toolchain"]; version == "" {
		t.Error("production config must register the external backstop-ai/bash-toolchain pack")
	}
	for registered := range cfg.Packs {
		if strings.Contains(strings.ToLower(registered), "bash") && registered != "backstop-ai/bash-toolchain" {
			t.Errorf("production config registers an unexpected Bash pack: %s", registered)
		}
	}
	fixtureRoots := 0
	walkErr := filepath.WalkDir(root, func(path string, d os.DirEntry, entryErr error) error {
		if entryErr != nil {
			return entryErr
		}
		slashPath := filepath.ToSlash(path)
		if d.IsDir() {
			if filepath.Base(path) == "issue188-bash-reference-pack" {
				fixtureRoots++
				if path != fixtureRoot {
					t.Errorf("reference fixture duplicated outside its pinned root: %s", path)
				}
			}
			if path == filepath.Join(root, ".git") || path == filepath.Join(root, ".backstop") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.Contains(slashPath, "/testdata/") {
			return nil
		}
		if filepath.Base(path) == "pack.yml" {
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			manifest, parseErr := pack.ParseManifest(data)
			if parseErr != nil {
				return parseErr
			}
			contents := strings.ToLower(string(data))
			if strings.Contains(contents, "bash") {
				t.Errorf("production manifest contains a Bash mechanism: %s (%s/%s)", path, manifest.Name, manifest.Language)
			}
		}
		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			production := strings.ToLower(string(data))
			if strings.Contains(production, "issue188-bash-reference-pack") || strings.Contains(production, "backstop-ai/bash-toolchain") {
				t.Errorf("production Core source registers or embeds a Bash mechanism: %s", path)
			}
		}
		return nil
	})
	if walkErr != nil {
		t.Fatalf("walking production registration surfaces: %v", walkErr)
	}
	if fixtureRoots != 1 {
		t.Fatalf("reference fixture root count = %d, want 1", fixtureRoots)
	}
}
