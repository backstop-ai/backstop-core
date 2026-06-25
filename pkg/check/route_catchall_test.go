package check

import (
	"os"
	"strings"
	"testing"
)

// SPEC-039 REQ-007 deletion-assertion + routing + behavior-preserving tests for
// the non-Go semgrep catch-all in routeFileDefaults. RED now (the default arm
// still exists), GREEN only after the TASK-007 catch-all deletion.

// catchallNonTestGoSources returns the contents of every non-test .go file in
// the current package directory, keyed by file name, for the source scan.
func catchallNonTestGoSources(t *testing.T) map[string]string {
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

// TestRouteFileDefaults_NonGoCatchAll_Removed is the REQ-007 deletion-assertion:
// no non-test .go source under pkg/check contains a routeFileDefaults `default`
// arm returning []CheckType{CheckTypeFindings}. CLM-001.
func TestRouteFileDefaults_NonGoCatchAll_Removed(t *testing.T) {
	// The deleted catch-all body. Its presence anywhere in production source means
	// the non-Go findings catch-all survives. (The .go/.ts/.tsx matched case
	// returns a four-element slice that includes CheckTypeFindings but never this
	// single-element literal, so this scan is specific to the deleted arm.)
	deletedArm := "return []CheckType{CheckTypeFindings}"
	for name, src := range catchallNonTestGoSources(t) {
		if strings.Contains(src, deletedArm) {
			t.Errorf("%s still contains %q — the non-Go catch-all (routeFileDefaults default arm) must be deleted", name, deletedArm)
		}
	}
}

// TestRouteFileDefaults_NonGoFileRoutesToNothing pins CLM-002 (and the surviving
// assertion for CLM-007): a non-.go/.ts/.tsx file routes via RouteFile on the
// default manifest to the EMPTY slice — no findings, no catch-all.
func TestRouteFileDefaults_NonGoFileRoutesToNothing(t *testing.T) {
	m := defaultManifest()
	for _, f := range []string{"README.md", "config.yml", "notes.txt"} {
		if got := m.RouteFile(f); len(got) != 0 {
			t.Errorf("%s routed to %v, want EMPTY slice (catch-all removed)", f, got)
		}
	}
}

// TestRouteFileDefaults_GoFileStillRoutesAllPasses pins CLM-003: a .go file
// still routes to {lint, build, test, findings} on the default manifest.
func TestRouteFileDefaults_GoFileStillRoutesAllPasses(t *testing.T) {
	m := defaultManifest()
	checks := m.RouteFile("pkg/foo/bar.go")
	want := []CheckType{CheckTypeLint, CheckTypeBuild, CheckTypeTest, CheckTypeFindings}
	if len(checks) != len(want) {
		t.Fatalf(".go routed to %v, want %v", checks, want)
	}
	for i, ct := range want {
		if checks[i] != ct {
			t.Errorf("checks[%d] = %v, want %v", i, checks[i], ct)
		}
	}
}

// TestRouteFileDefaults_TsFilesStillRouteAllPasses pins CLM-004: .ts and .tsx
// files still route to {lint, build, test, findings} on the default manifest
// (the deletion touches only the default branch, not the matched cases).
func TestRouteFileDefaults_TsFilesStillRouteAllPasses(t *testing.T) {
	m := defaultManifest()
	want := []CheckType{CheckTypeLint, CheckTypeBuild, CheckTypeTest, CheckTypeFindings}
	for _, path := range []string{"src/app.ts", "src/widget.tsx"} {
		checks := m.RouteFile(path)
		if len(checks) != len(want) {
			t.Fatalf("%s routed to %v, want %v", path, checks, want)
		}
		for i, ct := range want {
			if checks[i] != ct {
				t.Errorf("%s checks[%d] = %v, want %v", path, i, checks[i], ct)
			}
		}
	}
}

// TestCodeCheck_FindingsHasNoCheckExecutor pins CLM-006: the executor map built
// for code-check contains lint/build/test only, never findings —
// substantiating that the deleted non-Go catch-all could not have run anything
// (a routed findings pass had no executor, recorded as a Skipped no-op; findings
// runs on the pack engine, not through any pkg/check executor).
//
// The check is exercised over a stack that actually builds the native passes —
// the TypeScript built-in toolchain (eslint/tsc + a declared test command) — so
// the "lint/build/test present, findings absent" assertion is substantive. (The
// Go path bakes NO native commands — they come from the go-toolchain pack — so
// it builds an empty map and would not exercise the present-vs-absent contrast.)
func TestCodeCheck_FindingsHasNoCheckExecutor(t *testing.T) {
	runner := &fakeRunner{}
	cfg := loadConfigFromYAML(t, tsBackstopYML)
	execs := buildExecutorsForConfig(Options{Language: "typescript", Config: cfg}, runner)
	if _, ok := execs[CheckTypeFindings]; ok {
		t.Error("executor map contains CheckTypeFindings; the registry must build lint/build/test only (findings runs on the pack engine)")
	}
	for _, ct := range []CheckType{CheckTypeLint, CheckTypeBuild, CheckTypeTest} {
		if _, ok := execs[ct]; !ok {
			t.Errorf("executor map missing %v; the native passes must build for a built-in stack", ct)
		}
	}
}

// TestCheckType_NeutralFindings_UnknownExtRoutesEmpty pins CLM-007: the
// SPEC-035-style neutral-findings rename stays pinned on surviving sites (.go
// still routes findings) while an unknown extension now routes EMPTY.
func TestCheckType_NeutralFindings_UnknownExtRoutesEmpty(t *testing.T) {
	m := defaultManifest()
	// Surviving site: .go still routes the neutral findings pass.
	goRoutes := m.RouteFile("main.go")
	sawFindings := false
	for _, ct := range goRoutes {
		if ct == CheckTypeFindings {
			sawFindings = true
		}
	}
	if !sawFindings {
		t.Errorf(".go routing = %v, want to include CheckTypeFindings (neutral rename pinned)", goRoutes)
	}
	// The neutral string survives.
	if got := CheckTypeFindings.String(); got != "findings" {
		t.Errorf("CheckTypeFindings.String() = %q, want \"findings\"", got)
	}
	// Unknown extension now routes EMPTY (catch-all removed).
	if got := m.RouteFile("notes.txt"); len(got) != 0 {
		t.Errorf("unknown-ext routing = %v, want EMPTY (no catch-all)", got)
	}
}
