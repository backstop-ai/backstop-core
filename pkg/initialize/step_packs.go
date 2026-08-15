package initialize

import (
	"fmt"
	"strings"

	"github.com/backstop-ai/backstop-core/pkg/check"
	"github.com/backstop-ai/backstop-core/pkg/pack/distribution"
)

// stepPacksName is the report name for step 4.
const stepPacksName = "packs"

// stepPacks installs exactly the pack refs the consumer named (SPEC-069 REQ-010,
// REQ-018).
//
// PACKS ENTER A CONSUMER PROJECT ONLY THROUGH AN EXPLICIT CONSUMER ACT. Init holds no
// pack roster and no pack-name literal, has no concept of a "primary language", and
// never selects a pack from anything it finds on disk — which is why this function
// reads `refs` and nothing else about the project.
//
// EVERY REF IS CLASSIFIED BEFORE ANY INSTALL RUNS. A local filesystem path is REFUSED
// as a config error, and the refusal has to precede the first install or a portable
// ref supplied ahead of an unportable one leaves the lock half-written.
//
// The classification calls the SHIPPED distribution.IsLocalPath and defines no
// predicate of its own. Two definitions of "is this a local path" drifting apart is
// not hypothetical: a ref init classified remote and the add path classified local
// produces exactly the machine-specific `local_path` lock entry REQ-018 exists to
// prevent, and it fails nowhere near init.
//
// With no `--pack` the step is a REPORTED no-op — zero packs installed, plus a line
// naming `backstop pack add`. A silent skip is what a baked roster would be needed to
// avoid, and there is no roster: there is nothing for init to install that a consumer
// did not name, so the report says so.
func stepPacks(projectRoot string, refs []string, installer PackInstaller) (StepReport, error) {
	for _, ref := range refs {
		if !distribution.IsLocalPath(ref) {
			continue
		}
		return StepReport{}, &check.ConfigError{Message: fmt.Sprintf(
			"--pack %q is a local filesystem path. Init installs packs through PORTABLE git-ref references only, because a local source records a machine-specific local_path in backstop.lock and that lock is committed — the next machine to clone this repository could not resolve it. "+
				"Supply the pack as an org/repository reference, or install it after init with `backstop pack add %s`",
			ref, ref)}
	}

	if len(refs) == 0 {
		return StepReport{
			Step:    stepPacksName,
			Outcome: OutcomeSkipped,
			Detail:  "no --pack was supplied, so no pack was installed. Add one at any time with `backstop pack add <org>/<pack>@<version>`",
		}, nil
	}

	installed := make([]string, 0, len(refs))
	for _, ref := range refs {
		if err := installer.Install(projectRoot, ref); err != nil {
			return StepReport{
				Step:    stepPacksName,
				Outcome: OutcomeBrokenPromise,
				Detail: fmt.Sprintf("installing pack %s failed: %v%s",
					ref, err, alreadyInstalledSuffix(installed)),
			}, nil
		}
		installed = append(installed, ref)
	}

	return StepReport{
		Step:    stepPacksName,
		Outcome: OutcomeDelivered,
		Detail:  fmt.Sprintf("installed %s", strings.Join(installed, ", ")),
	}, nil
}

// alreadyInstalledSuffix names the packs that DID install before the failure.
//
// A failure report that named only the pack that broke would leave the consumer
// guessing about the state of their project; naming what landed is what makes the
// next command they run an informed one.
func alreadyInstalledSuffix(installed []string) string {
	if len(installed) == 0 {
		return ""
	}
	return fmt.Sprintf(" (%s installed successfully before this)", strings.Join(installed, ", "))
}
