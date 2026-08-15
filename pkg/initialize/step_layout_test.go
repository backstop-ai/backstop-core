package initialize

import (
	"path/filepath"
	"testing"
)

// theSixArtifactDirectories are the six init scaffolds under `.backstop/`, returned
// fresh so every assertion below is about the same six.
func theSixArtifactDirectories() []string {
	return []string{
		".backstop/bundles",
		".backstop/specs",
		".backstop/plans",
		".backstop/issues",
		".backstop/adrs",
		".backstop/directives",
	}
}

// TestInit_FullSdlcScaffoldsAllSixArtifactDirectoriesUnderBackstop (SPEC-069 CLM-025).
func TestInit_FullSdlcScaffoldsAllSixArtifactDirectoriesUnderBackstop(t *testing.T) {
	root := t.TempDir()

	report := stepLayout(root, allCapabilities(t))
	if report.Outcome != OutcomeDelivered {
		t.Fatalf("layout step reported %v (%s), want OutcomeDelivered", report.Outcome, report.Detail)
	}

	for _, dir := range theSixArtifactDirectories() {
		if !exists(root, dir) {
			t.Fatalf("the full-SDLC layout is missing %s", dir)
		}
	}
}

// TestInit_PackOnlyScaffoldsNoArtifactDirectories (SPEC-069 CLM-027).
//
// The pack-only profile creates no artifact directory ANYWHERE. Steps 2 (config) and
// 3 (layout) are separate precisely so this profile expresses its difference as a step
// that is simply ABSENT rather than as a conditional buried inside one.
func TestInit_PackOnlyScaffoldsNoArtifactDirectories(t *testing.T) {
	root := t.TempDir()

	report := stepLayout(root, capabilitiesExcept(t, "sdlc"))
	if report.Outcome == OutcomeBrokenPromise {
		t.Fatalf("the layout step failed under the pack-only profile: %s", report.Detail)
	}

	for _, dir := range theSixArtifactDirectories() {
		if exists(root, dir) {
			t.Fatalf("the pack-only profile created %s; it scaffolds no artifact directories at all", dir)
		}
	}
	if exists(root, ".backstop") {
		t.Fatal("the pack-only profile created .backstop/; it has no artifact layout to root there")
	}
}

// TestInit_NeverCreatesRootLevelArtifactDirectories (SPEC-069 CLM-028, denylist).
//
// backstop-core's own ROOT-level layout is a framework exception init does not
// produce and does not police. A consumer repo gets `.backstop/`-rooted artifacts or
// none.
func TestInit_NeverCreatesRootLevelArtifactDirectories(t *testing.T) {
	rootLevel := []string{"specs", "bundles", "plans", "issues", "adrs", "directives"}

	for _, profile := range []struct {
		name         string
		capabilities map[Capability]bool
	}{
		{"full-sdlc", allCapabilities(t)},
		{"pack-only", capabilitiesExcept(t, "sdlc")},
	} {
		t.Run(profile.name, func(t *testing.T) {
			root := t.TempDir()
			stepLayout(root, profile.capabilities)

			for _, dir := range rootLevel {
				if exists(root, dir) {
					t.Fatalf("the %s profile created the ROOT-level directory %s/; init roots a consumer's artifacts under .backstop/ and never at the repository root",
						profile.name, dir)
				}
			}
		})
	}
}

// TestInit_ScaffoldsNoCapabilitiesDirectory (SPEC-069 CLM-104, denylist).
//
// SIX OF SEVEN, DELIBERATELY. SPEC-068's layout table covers seven artifact kinds;
// init scaffolds no `capabilities/` at either location. That is a DECISION the corpus
// records rather than an omission a later reader "fixes": a capability artifact
// declares a named contract at the pack<->core wire seam, has a directory-per-artifact
// shape unlike the other six, is authored by framework and pack authors, and is
// produced by no step of a consuming project's onboarding — so pre-creating it would
// scaffold a directory no verb in the flow init just ran can fill.
func TestInit_ScaffoldsNoCapabilitiesDirectory(t *testing.T) {
	for _, profile := range []struct {
		name         string
		capabilities map[Capability]bool
	}{
		{"full-sdlc", allCapabilities(t)},
		{"pack-only", capabilitiesExcept(t, "sdlc")},
	} {
		t.Run(profile.name, func(t *testing.T) {
			root := t.TempDir()
			stepLayout(root, profile.capabilities)

			for _, location := range []string{"capabilities", ".backstop/capabilities"} {
				if exists(root, location) {
					t.Fatalf("the %s profile created %s/; init scaffolds six of the seven artifact kinds, and the seventh is deliberately not one of them",
						profile.name, location)
				}
			}
		})
	}
}

// TestInit_CreatesOnlyTheMissingArtifactDirectories (SPEC-069 CLM-039).
//
// When some directories already exist, only the MISSING ones are created and the
// existing ones AND THEIR CONTENTS are untouched. Converge, never clobber.
func TestInit_CreatesOnlyTheMissingArtifactDirectories(t *testing.T) {
	root := t.TempDir()

	// Two of the six already exist and hold real artifacts.
	writeFile(t, root, ".backstop/specs/SPEC-001-existing.spec.md", "pre-existing spec content\n")
	writeFile(t, root, ".backstop/plans/PLAN-SPEC-001-existing.plan.yml", "pre-existing plan content\n")
	before := snapshotTree(t, filepath.Join(root, ".backstop"))

	report := stepLayout(root, allCapabilities(t))
	if report.Outcome != OutcomeDelivered {
		t.Fatalf("layout step reported %v (%s)", report.Outcome, report.Detail)
	}

	for _, dir := range theSixArtifactDirectories() {
		if !exists(root, dir) {
			t.Fatalf("the layout step left %s missing", dir)
		}
	}

	after := snapshotTree(t, filepath.Join(root, ".backstop"))
	for path, body := range before {
		got, survived := after[path]
		if !survived {
			t.Fatalf("the layout step removed the pre-existing artifact %s", path)
		}
		if got != body {
			t.Fatalf("the layout step rewrote the pre-existing artifact %s.\nbefore: %q\nafter:  %q", path, body, got)
		}
	}
	if len(after) != len(before) {
		t.Fatalf("the layout step wrote %d file(s) into an existing layout; it creates directories and nothing else", len(after)-len(before))
	}
}

// TestInit_LayoutConvergesOnASecondRun asserts the step reports converged when every
// directory is already present. It is additive: it is what makes the mandated
// second-run claim satisfiable at the runner level.
func TestInit_LayoutConvergesOnASecondRun(t *testing.T) {
	root := t.TempDir()

	if first := stepLayout(root, allCapabilities(t)); first.Outcome != OutcomeDelivered {
		t.Fatalf("first layout run reported %v, want OutcomeDelivered", first.Outcome)
	}
	second := stepLayout(root, allCapabilities(t))
	if second.Outcome != OutcomeConverged {
		t.Fatalf("second layout run reported %v (%s), want OutcomeConverged", second.Outcome, second.Detail)
	}
}
