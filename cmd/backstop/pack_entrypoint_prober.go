package main

import (
	"context"
	"errors"
	"os/exec"
	"sort"

	"github.com/backstop-ai/backstop-core/pkg/check"
	"github.com/backstop-ai/backstop-core/pkg/pack"
	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
)

// pack_entrypoint_prober.go is THE ONE EXECUTION ROUTE for pack-declared test/build
// entrypoints, AND IT HAS TWO CALLERS.
//
// `backstop init`'s toolchain step reaches it through packToolchainProber
// (init_toolchain.go); `backstop doctor`'s toolchain-runs check consumes THIS type
// rather than building its own copy. Two independent copies of a pack-command
// EXECUTION path is exactly what SPEC-069 REQ-011 forbids, and they would drift on the
// one thing the two specs already disagree about — the runner method — so the sequence
// AND the method live here, once.
//
// It is NOT a cross-package extraction: both callers are `package main` in
// cmd/backstop, forced there because checkEngineToolAllowed and splitCommand are
// unexported. So this is a package-internal helper both callers already sit beside.
//
// THE SURFACE IS RAW AND REPORT-FREE, DELIBERATELY. It reports what RAN and what
// HAPPENED, and knows nothing about init's steps or doctor's check results: init's
// owed-setup-versus-verbatim classification and its exit-code mapping layer ON TOP of
// these values, and doctor's one-result-per-check rollup is the mirror-image layer on
// its own side. An init-flavored "shared" helper would be unusable by doctor and would
// be re-forked on sight.

// entrypointOutcome is the mechanically honest result of one declared entrypoint
// execution.
//
// It carries NO report vocabulary: "owed setup" and "verbatim" are init's words for
// these values, warn/fail/skip are doctor's.
type entrypointOutcome int

const (
	// entrypointRefused: the trusted-tool allowlist said no. The command was NEVER
	// split and NEVER run.
	entrypointRefused entrypointOutcome = iota // nosemgrep: go.core.no-global-mutable-state — immutable const, not mutable global
	// entrypointPassed: the command started and exited zero.
	entrypointPassed
	// entrypointUnstartable: the declared executable could not be STARTED at all.
	entrypointUnstartable
	// entrypointExitedNonZero: the command started and exited non-zero.
	entrypointExitedNonZero
)

// entrypointProbe is one selected binding's execution record.
type entrypointProbe struct {
	// PackName is the manifest name of the declaring pack.
	PackName string
	// EngineKey is the `engines:` map key, so attribution is stable across runs.
	EngineKey string
	// GateType is GateTypeTest or GateTypeBuild — the only two selected.
	GateType engine.GateType
	// Command is the pack's declared command string, VERBATIM.
	Command string
	// Outcome is the mechanically honest result.
	Outcome entrypointOutcome
	// ExitCode is meaningful ONLY for entrypointExitedNonZero.
	ExitCode int
	// Output is the COMBINED stdout+stderr the runner captured.
	Output []byte
	// Err is the refusal error or the start error; nil otherwise.
	Err error
}

// packEntrypointProber executes pack-declared test/build entrypoints.
//
// NO projectRoot FIELD OR PARAMETER, AND THAT IS NOT AN OMISSION. check.CommandRunner
// has no working-directory argument — the directory is a CONSTRUCTION-TIME property of
// check.ExecCommandRunner{Dir: …} — so each caller roots its own runner and hands it
// in. Introducing a second rooting mechanism here would be a second execution path in
// miniature.
type packEntrypointProber struct {
	Packs  []*pack.Manifest
	Runner check.CommandRunner
}

// Probe executes every declared test/build entrypoint EXACTLY ONCE, in deterministic
// order, and returns one entrypointProbe per selected binding.
//
// NO ERROR RETURN, BY DESIGN. An allowlist refusal is PER-BINDING state
// (entrypointRefused plus Err), not a whole-probe abort, because the two callers must
// dispose of it differently: init surfaces it as a CONFIG ERROR while doctor surfaces
// it as that check's failure. A shared `error` return would force one caller's
// disposition on the other, which is the init-flavored interface this extraction
// exists to avoid.
//
// ZERO SELECTED BINDINGS RETURNS AN EMPTY SLICE: ABSENCE IS THE CALLER'S TO CLASSIFY.
// Init reports capability-absent without failing; doctor warns. Probe itself never
// decides that.
//
// DETERMINISTIC ORDER IS PART OF THE SHARED CONTRACT: Packs are walked in SLICE order,
// and within each manifest the Engines are walked by SORTED KEY. Manifest.Engines is a
// MAP, so an unsorted walk would yield a different report on every run — and while init
// does not assert ordering, doctor's byte-identical-report claim is satisfiable only if
// this sorts. Do not "simplify" the sort away.
func (p *packEntrypointProber) Probe(ctx context.Context) []entrypointProbe {
	probes := []entrypointProbe{}

	for _, manifest := range p.Packs {
		if manifest == nil {
			continue
		}

		keys := make([]string, 0, len(manifest.Engines))
		for key := range manifest.Engines {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		for _, key := range keys {
			binding := manifest.Engines[key].Binding
			// SELECTION IS BY STAGE, NEVER BY TOOL. `test` and `build` are backstop's
			// own kill-chain vocabulary, so selecting on them names a STAGE of the
			// chain and introduces no tool or language noun.
			if binding.GateType != engine.GateTypeTest && binding.GateType != engine.GateTypeBuild {
				continue
			}

			probe := entrypointProbe{
				PackName:  manifest.NormalizedName,
				EngineKey: key,
				GateType:  binding.GateType,
				Command:   binding.Command,
			}

			// THE TRUST GATE SITS BEFORE THE SPLITTER AND BEFORE THE RUNNER. A command
			// whose tool the allowlist refuses is not executed at all — not split, not
			// handed to anything. This path executes arbitrary pack-supplied command
			// strings, so an unbound execution route here would be a hole in the
			// trusted-tool invariant, not a style preference.
			if gateErr := checkEngineToolAllowed(manifest, binding); gateErr != nil {
				probe.Outcome = entrypointRefused
				probe.Err = gateErr
				probes = append(probes, probe)
				continue
			}

			name, args := splitCommand(binding.Command)

			// ★★ THE RUNNER IS ENTERED THROUGH `Run`, NOT `RunStdout`, AND THIS IS THE
			// ONE DELIBERATE DIVERGENCE FROM runFindingsEngine.
			//
			// The two methods exist for OPPOSITE reasons and the shipped comments say
			// so: RunStdout exists so a tool's stderr banner cannot corrupt the SARIF
			// bytes on stdout, and Run exists for "the build/test executors whose
			// violation messages may legitimately include stderr" — which is exactly
			// the started-and-exited-non-zero case.
			//
			// A failing build or test entrypoint routinely writes its WHOLE diagnostic
			// to stderr. Binding RunStdout here would print an EMPTY "captured output
			// verbatim" for precisely the failures this path exists to surface, and
			// nothing else would change — which is why the divergence is invisible in
			// a diff review and why a test drives a stdout-EMPTY fixture to catch it.
			// Neither caller parses this output; both render it to a human, so the
			// contamination hazard RunStdout exists for does not apply while the
			// lost-diagnostic hazard does.
			//
			// THIS IS THE ONE PLACE THE METHOD IS CHOSEN FOR BOTH COMMANDS. If a future
			// reader wants doctor to capture differently, that is a spec-level
			// conversation to surface, never a private second runner.
			output, runErr := p.Runner.Run(ctx, name, args...)
			probe.Output = output

			// THE (b)/(c) SPLIT LIVES HERE, ONCE. Both callers demand the same one
			// mechanically honest signal — the executable could not be STARTED versus
			// it started and exited non-zero — so it is classified here and nowhere
			// else. Two copies of this two-line split is how two commands come to
			// disagree about the same failure.
			var exitErr *exec.ExitError
			switch {
			case runErr == nil:
				probe.Outcome = entrypointPassed
			case errors.As(runErr, &exitErr):
				probe.Outcome = entrypointExitedNonZero
				probe.ExitCode = exitErr.ExitCode()
			default:
				probe.Outcome = entrypointUnstartable
				probe.Err = runErr
			}

			probes = append(probes, probe)
		}
	}

	return probes
}
