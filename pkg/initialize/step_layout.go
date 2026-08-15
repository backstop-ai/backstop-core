package initialize

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// stepLayoutName is the report name for step 3.
const stepLayoutName = "layout"

// artifactDirectories are the SIX directories the full-SDLC profile scaffolds under
// the artifact root.
//
// SIX OF SEVEN, DELIBERATELY. The layout table covers seven artifact kinds, and
// `capabilities` is not among these six. That is a decision, not an omission: the six
// are exactly the work products of the two consumer tracks (issue -> plan and bundle
// -> spec -> plan -> implementation), whereas a capability artifact declares a named
// contract at the pack<->core wire seam, carries a directory-per-artifact shape unlike
// the flat file-per-artifact shape of these six, is authored by framework and pack
// authors, and is produced by NO step of a consuming project's onboarding. Pre-creating
// it would scaffold a directory no verb in the flow init just ran can fill.
var artifactDirectories = []string{ // nosemgrep: go.core.no-global-mutable-state — immutable directory list, never mutated
	"bundles",
	"specs",
	"plans",
	"issues",
	"adrs",
	"directives",
}

// stepLayout scaffolds the artifact-root-relative artifact layout (SPEC-069 REQ-004).
//
// It is carried by the `sdlc` capability: the full-SDLC profile creates exactly the
// six directories, and the pack-only profile creates NOTHING AT ALL — not even the
// artifact root, which it declares no `artifact_root` for and therefore has no layout
// to put there.
//
// Steps 2 (config) and 3 (layout) are SEPARATE even though both belong to the profile
// fork, because the pack-only profile writes a config and creates no directories.
// Folding them would make that profile express its difference as a conditional inside
// one step rather than as a step that is simply absent from the sequence.
//
// INIT NEVER CREATES ROOT-LEVEL `specs/`, `bundles/`, `plans/`, `issues/`, `adrs/` or
// `directives/` in a consumer repo. backstop-core's own root layout is a framework
// exception init does not produce and does not police.
//
// It ADDS ONLY WHAT IS MISSING and overwrites nothing. No file outside backstop's own
// surface is read to decide any of this.
func stepLayout(projectRoot string, capabilities map[Capability]bool) StepReport {
	if !capabilities[CapabilitySdlc] {
		return StepReport{
			Step:    stepLayoutName,
			Outcome: OutcomeSkipped,
			Detail:  "the pack-only profile scaffolds no artifact directories; run `backstop init` without --no-sdlc to get the full artifact layout",
		}
	}

	root := filepath.Join(projectRoot, artifactRootValue)
	created := []string{}

	for _, name := range artifactDirectories {
		target := filepath.Join(root, name)
		if directoryExists(target) {
			continue
		}
		if err := os.MkdirAll(target, 0o755); err != nil {
			return StepReport{
				Step:    stepLayoutName,
				Outcome: OutcomeBrokenPromise,
				Detail:  fmt.Sprintf("creating the artifact directory %s: %v", target, err),
			}
		}
		created = append(created, artifactRootValue+"/"+name+"/")
	}

	if len(created) == 0 {
		return StepReport{
			Step:    stepLayoutName,
			Outcome: OutcomeConverged,
			Detail: fmt.Sprintf("every artifact directory under %s/ was already present and was left untouched, contents included",
				artifactRootValue),
		}
	}

	return StepReport{
		Step:    stepLayoutName,
		Outcome: OutcomeDelivered,
		Detail:  fmt.Sprintf("created %s", strings.Join(created, ", ")),
	}
}
