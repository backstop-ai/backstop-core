package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/backstop-ai/backstop-core/pkg/check"
	"github.com/backstop-ai/backstop-core/pkg/initialize"
	"github.com/backstop-ai/backstop-core/pkg/pack"
)

// packToolchainProber is init's concrete initialize.ToolchainProber (SPEC-069 REQ-011).
//
// IT IS A THIN ADAPTER OVER THE SHARED packEntrypointProber AND DOES NOTHING
// MECHANICAL OF ITS OWN — no second selection, no second allowlist call, no exec. The
// three execution steps (checkEngineToolAllowed -> splitCommand -> the shared
// check.CommandRunner) live in pack_entrypoint_prober.go and nowhere else, so there is
// no second execution path to audit and no shell anywhere.
//
// EVERYTHING THIS FILE ADDS IS REPORTING, AND ALL OF IT IS INIT-SPECIFIC. Doctor's
// rollup over the same raw outcomes is the mirror-image layer on its own side and
// never lands here.
//
// THE LOCATION IS FORCED: checkEngineToolAllowed and splitCommand are unexported in
// package main, so a pkg/initialize implementation would need a second copy of both —
// the exact second execution path REQ-011 forbids.
type packToolchainProber struct {
	// Packs is the manifest corpus to probe.
	//
	// WHEN IT IS NIL, Probe loads the corpus INSTALLED AT PROBE TIME from projectRoot,
	// and that is the PRODUCTION case rather than a fallback. Init's pack step installs
	// packs during the SAME run, so a corpus captured when the command was constructed
	// would be the empty set the project had BEFORE init ran — the toolchain step would
	// then report capability-absent for a project that had just installed a pack
	// declaring an entrypoint. A non-nil value is an explicit corpus a caller supplies.
	Packs []*pack.Manifest
	// Runner is rooted at the project root by its constructor, because
	// check.CommandRunner takes no per-call directory.
	Runner check.CommandRunner
}

// Probe executes every installed pack's declared test/build entrypoint once and
// renders each outcome in init's own vocabulary.
//
// THE ONLY ERROR IT RETURNS IS A CONFIG ERROR. An allowlist refusal is neither a
// toolchain pass nor a toolchain fail: the command was never run, so there is no
// verdict to report, and reporting one either way would be init inventing a result for
// something that did not happen.
func (p *packToolchainProber) Probe(projectRoot string) ([]initialize.StepReport, error) {
	packs := p.Packs
	if packs == nil {
		loaded, err := loadInstalledPacks(projectRoot)
		if err != nil {
			return nil, &check.ConfigError{Message: fmt.Sprintf(
				"reading the installed packs to find their declared test and build entrypoints: %v", err)}
		}
		packs = loaded
	}

	shared := &packEntrypointProber{Packs: packs, Runner: p.Runner}
	probes := shared.Probe(context.Background())

	// CAPABILITY-ABSENT, NOT A FAILURE. When no installed pack declares a test or build
	// engine there is simply nothing to execute. An un-adopted capability is a missing
	// benefit, never a broken promise, so init reports it and does not fail the run.
	if len(probes) == 0 {
		return []initialize.StepReport{{
			Step:    initialize.StepToolchain,
			Outcome: initialize.OutcomeCapabilityAbsent,
			Detail: "no installed pack declares a test or build entrypoint, so there was nothing to run. " +
				"That is not a failure — it means your packs do not yet describe how to build or test this project",
		}}, nil
	}

	reports := make([]initialize.StepReport, 0, len(probes))
	for _, probe := range probes {
		if probe.Outcome == entrypointRefused {
			// Surfaced as the returned ERROR, so it maps to a config error rather than
			// to a toolchain verdict. The refusal already names the tool and the pack.
			return nil, &check.ConfigError{Message: fmt.Sprintf(
				"pack %s declares the %s entrypoint %q, which backstop will not run: %v",
				probe.PackName, probe.GateType, probe.Command, probe.Err)}
		}
		reports = append(reports, toolchainReport(probe, projectRoot))
	}
	return reports, nil
}

// toolchainReport renders ONE probe in init's vocabulary.
//
// ★ THE TWO FAILURE OUTCOMES ARE SPLIT AND THEIR LABELS DIFFER, AND THAT SPLIT IS THE
// POINT. Treating any non-zero exit as owed setup would commit exactly the exit-code
// cause-inference REQ-011's first sentence forbids, and would re-enact the
// misdiagnosis that is the requirement's own evidence.
//
//	could not be STARTED  -> a SETUP step the consumer still owes. It names the pack
//	                         whose entrypoint could not run and points at THAT PACK's
//	                         own documented install steps. It invents no install command
//	                         and installs nothing.
//	started, exited != 0  -> the exit code plus the captured output VERBATIM, attributed
//	                         to the pack and the command and to NOTHING ELSE. NO cause
//	                         is claimed: init must not call this owed setup, must not
//	                         name dependencies or a package manager, and must not
//	                         attribute it to a wrapper — because it cannot tell.
//
// WHAT INIT CANNOT DISTINGUISH, STATED RATHER THAN PROMISED: a pack declares ONE
// command per engine, so if the declared entrypoint is a wrapper invocation, the exit
// status init reads IS the wrapper's. Core cannot see past it and must not pretend to.
// The temptations to "fix" that — wrapper-aware parsing of the captured output, a
// second probing command, a package-manager-shaped retry — all bake ecosystem knowledge
// into core, and the durable guard against them is this POSTURE rather than any scan:
// the non-zero case REPORTS and does not DIAGNOSE.
func toolchainReport(probe entrypointProbe, projectRoot string) initialize.StepReport {
	switch probe.Outcome {
	case entrypointPassed:
		return initialize.StepReport{
			Step:    initialize.StepToolchain,
			Outcome: initialize.OutcomeDelivered,
			Detail: fmt.Sprintf("pack %s: the declared %s entrypoint %q ran in %s and exited 0",
				probe.PackName, probe.GateType, probe.Command, projectRoot),
		}

	case entrypointUnstartable:
		return initialize.StepReport{
			Step:    initialize.StepToolchain,
			Outcome: initialize.OutcomeBrokenPromise,
			Detail: fmt.Sprintf("pack %s: the declared %s entrypoint %q could not be STARTED in %s (%v). "+
				"This is setup you still owe: follow that pack's own documented install steps. Backstop installs nothing on your behalf and has no install command of its own to offer",
				probe.PackName, probe.GateType, probe.Command, projectRoot, probe.Err),
		}

	default:
		return initialize.StepReport{
			Step:    initialize.StepToolchain,
			Outcome: initialize.OutcomeBrokenPromise,
			Detail: fmt.Sprintf("pack %s: the declared %s entrypoint %q ran in %s and exited %d.%s",
				probe.PackName, probe.GateType, probe.Command, projectRoot, probe.ExitCode,
				capturedEntrypointOutput(probe.Output)),
		}
	}
}

// capturedEntrypointOutput renders an entrypoint's captured output VERBATIM.
//
// It is the COMBINED stdout+stderr the shared prober captured. Nothing is parsed,
// summarized or interpreted: the point of the verbatim rule is that the consumer reads
// what their own tool said, not init's paraphrase of it.
func capturedEntrypointOutput(output []byte) string {
	body := strings.TrimRight(string(output), "\n")
	if body == "" {
		return " It produced no output."
	}
	return "\nIts output was:\n" + body
}
