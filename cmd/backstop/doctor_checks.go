package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/backstop-ai/backstop-core/pkg/artifact"
	"github.com/backstop-ai/backstop-core/pkg/check"
	"github.com/backstop-ai/backstop-core/pkg/config"
	"github.com/backstop-ai/backstop-core/pkg/gate"
	"github.com/backstop-ai/backstop-core/pkg/pack"
	"github.com/backstop-ai/backstop-core/pkg/schema"
)

// doctor_checks.go holds the check FUNCTIONS. It calls NO loader.
//
// THE SPLIT IS LOAD-BEARING, NOT ORGANIZATIONAL. doctor.go gathers — it is the one file
// that calls config.DiscoverConfigPath, config.LoadConfig and loadInstalledPacks, each
// exactly once, before the registry runs. This file reads the DATA that gathering
// produced. A loader call appearing here is the second abort path REQ-003 exists to
// forbid, and a source scan asserts its absence because it would read as ordinary
// defensive error handling in review.
//
// ONE CONDITION, ONE OWNER. A check whose input could not be gathered is SKIPPED with the
// OWNING check named, never failed a second time — otherwise a single absent backstop.yml
// would produce four failures and the report would say four things are broken when one is.

// doctorContext is gathered ONCE and passed to every check. No check gathers its own
// input.
//
// ConfigPathErr, ConfigErr and PacksErr are carried as DATA rather than raised, which is
// the mechanism behind REQ-003: every load failure reaches the checks as a condition to
// REPORT, so no check can turn one into a refusal to start. SearchDir is the working
// directory discovery started from, which checkConfigPresent names in its failure;
// ConfigPath is empty exactly when ConfigPathErr is non-nil, which is the SKIP signal
// checkConfigLoads and checkPacksInstalled read.
type doctorContext struct {
	ProjectRoot   string
	SearchDir     string
	ConfigPath    string
	ConfigPathErr error
	Config        *config.Config
	ConfigErr     error
	Packs         []*pack.Manifest
	PacksErr      error
	Runner        check.CommandRunner
}

// checkConfigPresent owns ONE condition: no backstop.yml is discoverable.
func checkConfigPresent(ctx doctorContext) doctorResult {
	result := doctorResult{ID: doctorCheckConfigPresent, Title: "backstop.yml is discoverable"}

	if ctx.ConfigPathErr != nil {
		result.Status = doctorStatusFail
		result.Message = fmt.Sprintf("no backstop.yml was found searching up from %s: %v", ctx.SearchDir, ctx.ConfigPathErr)
		result.Remediation = "create a backstop.yml at the project root, or run `backstop init` to write one"
		return result
	}

	result.Status = doctorStatusPass
	result.Message = fmt.Sprintf("found at %s", ctx.ConfigPath)
	return result
}

// checkConfigLoads owns ONE condition: the discovered backstop.yml does not load or
// validate.
//
// It reports the LOADER'S OWN error text — the very error runGate converts into its
// exit-2 — and is SKIPPED naming config-present when nothing was discovered to load, so
// the absent-config condition is reported exactly once.
func checkConfigLoads(ctx doctorContext) doctorResult {
	result := doctorResult{ID: doctorCheckConfigLoads, Title: "backstop.yml loads and validates"}

	if ctx.ConfigPathErr != nil {
		result.Status = doctorStatusSkipped
		result.Message = fmt.Sprintf("no backstop.yml was discovered; %s owns that condition", doctorCheckConfigPresent)
		return result
	}
	if ctx.ConfigErr != nil {
		result.Status = doctorStatusFail
		result.Message = fmt.Sprintf("%s does not load: %v", ctx.ConfigPath, ctx.ConfigErr)
		result.Remediation = "fix the reported error in backstop.yml; this is the same failure that makes `backstop gate` refuse to run"
		return result
	}

	result.Status = doctorStatusPass
	result.Message = fmt.Sprintf("%s loads and validates", ctx.ConfigPath)
	return result
}

// checkGitRepository owns ONE condition: the project root is not inside a git work tree.
//
// IT WARNS AND NEVER FAILS, which is a decision rather than an oversight. The gate itself
// already treats a missing work tree as a loud fallback to a full sweep rather than a
// refusal (pkg/gate/scope.go), so a non-repo is DEGRADED CAPABILITY, not a broken
// promise. Failing here would exit 1 in every non-repo directory, including deliberate
// ones; saying nothing would hide the id-reservation loss.
//
// Detection goes through the existing exported detector rather than a fourth
// `git rev-parse` shell-out of doctor's own.
func checkGitRepository(ctx doctorContext) doctorResult {
	result := doctorResult{ID: doctorCheckGitRepository, Title: "the project root is a git work tree"}

	if (&check.DefaultGitExecutor{Dir: ctx.ProjectRoot}).IsGitRepo() {
		result.Status = doctorStatusPass
		result.Message = fmt.Sprintf("%s is inside a git work tree", ctx.ProjectRoot)
		return result
	}

	result.Status = doctorStatusWarn
	result.Message = fmt.Sprintf(
		"%s is not inside a git work tree: the gate's diff scope falls back to a full sweep, and `artifact new` cannot reserve an id through a tag",
		ctx.ProjectRoot)
	result.Remediation = "run `git init` if this project should be versioned; backstop still runs without it, with the degraded behaviour named above"
	return result
}

// checkPacksInstalled owns ONE condition: a declared pack is missing or unparseable, or
// none is declared.
//
// THE THREE OUTCOMES FOLLOW THE SHAPE loadInstalledPacks ALREADY HAS: a config declaring
// no packs returns an EMPTY slice with a NIL error, while a declared-but-missing pack or
// an unparseable pack.yml returns an error. So empty-with-nil-error is the WARN case
// (backstop enforces nothing — un-adopted capability, loud but not blocking) and the
// error is the FAIL case.
//
// It is SKIPPED when the config is absent or unloadable, because loadInstalledPacks loads
// backstop.yml ITSELF and would otherwise surface a config problem under a pack heading.
func checkPacksInstalled(ctx doctorContext) doctorResult {
	result := doctorResult{ID: doctorCheckPacksInstalled, Title: "declared packs are installed"}

	if ctx.ConfigPathErr != nil {
		result.Status = doctorStatusSkipped
		result.Message = fmt.Sprintf("no backstop.yml was discovered; %s owns that condition", doctorCheckConfigPresent)
		return result
	}
	if ctx.ConfigErr != nil {
		result.Status = doctorStatusSkipped
		result.Message = fmt.Sprintf("backstop.yml does not load; %s owns that condition", doctorCheckConfigLoads)
		return result
	}
	if ctx.PacksErr != nil {
		result.Status = doctorStatusFail
		result.Message = fmt.Sprintf("the declared pack set could not be loaded: %v", ctx.PacksErr)
		result.Remediation = "run `backstop pack install` to install the packs backstop.yml declares"
		return result
	}
	if len(ctx.Packs) == 0 {
		result.Status = doctorStatusWarn
		result.Message = "backstop.yml declares no packs, so backstop enforces nothing"
		result.Remediation = "add a pack with `backstop pack add <org>/<pack>@<version>`; every check backstop runs comes from a pack"
		return result
	}

	result.Status = doctorStatusPass
	result.Message = fmt.Sprintf("%d declared pack(s) are present and parseable", len(ctx.Packs))
	return result
}

// checkBuildIdentity SURFACES the running binary's identity. It compares nothing.
//
// THE STALE-BINARY CASE IS THE POINT: a weeks-old binary reporting bare `dev` is the
// highest-ranked sharp edge from the hand-onboarding write-ups, whose skew was
// misdiagnosed as a pack producer error. Making it VISIBLE is the whole job.
//
// IT READS THE ONE RESOLUTION every output surface reads, so `backstop doctor` and
// `backstop version` cannot describe the same binary differently, and it performs NO pack
// read at all — capability-set comparison is BUNDLE-020's mechanism via SPEC-068, and the
// absence of a pack read here is what keeps that separation falsifiable rather than
// asserted.
func checkBuildIdentity(ctx doctorContext) doctorResult {
	// ctx is deliberately unread: this check performs NO pack read and no comparison, which
	// is the property a source scan asserts. It keeps the parameter because every registry
	// entry shares one Run signature.
	_ = ctx

	cohort, cohortErr := schema.ComputeCohort(SchemaFS)
	return describeBuildIdentity(effectiveBuildIdentity(), cohort.ID, cohortErr)
}

// describeBuildIdentity renders one resolved identity as a result.
//
// It is split out as a PURE function of its inputs, mirroring resolveBuildIdentity's own
// shape, so the stamped and unstamped cases are both reachable without building a binary
// per case. checkBuildIdentity supplies the real values.
func describeBuildIdentity(identity BuildIdentity, cohortID string, cohortErr error) doctorResult {
	result := doctorResult{ID: doctorCheckBuildIdentity, Title: "the running binary's build identity"}

	cohort := cohortID
	if cohortErr != nil {
		cohort = fmt.Sprintf("%s (%v)", unknownBuildField, cohortErr)
	}
	result.Message = fmt.Sprintf("version %s, commit %s, built %s, schema cohort %s",
		identity.Version, identity.Commit, identity.BuildDate, cohort)

	// ABSENT IS A WARN, NEVER A FAIL: the binary still runs, so blocking would be wrong.
	// "Absent" is any part of the identity no source supplied — a bare `dev` version is
	// exactly the stale-binary shape, and an unknown commit or date is the same skew seen
	// from a different field.
	var absent []string
	if identity.Version == "" || identity.Version == "dev" {
		absent = append(absent, "version")
	}
	if identity.Commit == "" || identity.Commit == unknownBuildField {
		absent = append(absent, "commit")
	}
	if identity.BuildDate == "" || identity.BuildDate == unknownBuildField {
		absent = append(absent, "build date")
	}
	if cohortErr != nil {
		absent = append(absent, "schema cohort")
	}

	if len(absent) == 0 {
		result.Status = doctorStatusPass
		return result
	}

	result.Status = doctorStatusWarn
	result.Message += fmt.Sprintf(" — no build identity for: %s", strings.Join(absent, ", "))
	result.Remediation = "this binary carries no release stamp, so it cannot be told apart from a stale one; install a released build if you did not build it yourself just now"
	return result
}

// checkToolchainRuns verifies the toolchain by EXECUTING it, once per pack-declared
// entrypoint, and never infers toolchain health from a file on disk.
//
// ★ IT CONSUMES THE SHARED packEntrypointProber AND OWNS NOTHING MECHANICAL. Selection by
// declared gate_type, the trust gate before the splitter and the runner, the command
// splitting, the deterministic order, the started-versus-exited-nonzero split, and the
// choice of runner method are ALL the prober's. `backstop init`'s toolchain step reaches
// the same helper, so there is ONE execution route to audit rather than two that drift —
// and the runner method is exactly where they had already begun to: the prober binds
// check.CommandRunner.Run (COMBINED stdout+stderr), so a failing build's diagnostic —
// which is entirely on stderr — survives into the verbatim report.
//
// WHAT DOCTOR OWNS IS REPORT VOCABULARY: the skip-first branch, the rollup to ONE result,
// the outcome→status mapping (including its own disposition of a refusal, which init
// surfaces as a config error instead — which is why Probe returns no error and leaves
// disposition to its caller), the empty-slice→warn decision, and the remediation text.
func checkToolchainRuns(ctx doctorContext) doctorResult {
	result := doctorResult{ID: doctorCheckToolchainRuns, Title: "pack-declared test/build entrypoints execute"}

	// SKIP FIRST, BEFORE THE PROBER IS EVEN CONSTRUCTED. An ungathered pack set must
	// never reach the prober and must never read as "no entrypoint declared" — that
	// outcome requires a pack set that WAS gathered and declares none.
	//
	// ★ ALL THREE CONDITIONS, NOT JUST PacksErr. gatherDoctorContext only calls
	// loadInstalledPacks when the config was BOTH discovered and loaded, so an absent or
	// unloadable backstop.yml leaves Packs nil with PacksErr ALSO nil — indistinguishable
	// from a gathered-and-empty pack set unless the config errors are read here too. A
	// PacksErr-only predicate reports "no installed pack declares a test or build
	// entrypoint" on a project whose packs were never looked at, which is exactly the
	// ungathered-set-as-outcome-(d) reading REQ-006 forbids.
	if ctx.ConfigPathErr != nil || ctx.ConfigErr != nil || ctx.PacksErr != nil {
		result.Status = doctorStatusSkipped
		result.Message = fmt.Sprintf("the installed pack set could not be gathered; %s owns that condition", doctorCheckPacksInstalled)
		return result
	}

	probes := (&packEntrypointProber{Packs: ctx.Packs, Runner: ctx.Runner}).Probe(context.Background())

	// Outcome (d): a gathered pack set declaring no toolchain entrypoint. The prober
	// deliberately does not decide absence — this is the caller's call, and doctor's
	// answer is WARN, never a silent pass.
	if len(probes) == 0 {
		result.Status = doctorStatusWarn
		result.Message = "no installed pack declares a test or build entrypoint, so nothing was executed"
		result.Remediation = "install a pack that declares a `gate_type: test` or `gate_type: build` engine; backstop executes only what packs declare"
		return result
	}

	// ONE RESULT, whose message enumerates every entrypoint separately and whose status
	// is the WORST outcome among them.
	var lines []string
	var remediations []string
	status := doctorStatusPass
	for _, probe := range probes {
		line, probeStatus, remediation := describeEntrypointProbe(probe)
		lines = append(lines, line)
		if remediation != "" {
			remediations = append(remediations, remediation)
		}
		status = worseDoctorStatus(status, probeStatus)
	}

	result.Status = status
	result.Message = fmt.Sprintf("%d declared entrypoint(s):\n%s", len(probes), strings.Join(lines, "\n"))
	result.Remediation = strings.Join(remediations, "\n")
	return result
}

// describeEntrypointProbe renders ONE probe as a report line, its status, and its
// remediation.
//
// The classification is READ OFF THE PROBE, never re-derived: doctor attempts no
// inspection of the underlying error and no second reading of an exit code.
func describeEntrypointProbe(probe entrypointProbe) (string, string, string) {
	attribution := fmt.Sprintf("%s (%s, gate_type %s): `%s`", probe.PackName, probe.EngineKey, probe.GateType, probe.Command)

	switch probe.Outcome {
	case entrypointPassed:
		return fmt.Sprintf("  pass  %s", attribution), doctorStatusPass, ""

	case entrypointUnstartable:
		// SETUP THE CONSUMER STILL OWES. Doctor points at THAT PACK's own documented
		// install steps and installs nothing itself — it has no idea what the pack's
		// toolchain is, and inventing an install command would be exactly the baked
		// tool knowledge backstop refuses to carry.
		return fmt.Sprintf("  fail  %s — the declared executable could not be started: %v", attribution, probe.Err),
			doctorStatusFail,
			fmt.Sprintf("setup the consumer still owes: install the toolchain pack %s documents, then re-run `backstop doctor --check %s`",
				probe.PackName, doctorCheckToolchainRuns)

	case entrypointExitedNonZero:
		// VERBATIM, AND WITH NO INFERRED CAUSE. The pnpm ERR_PNPM_IGNORED_BUILDS
		// evidence is a nonzero exit whose obvious reading was wrong, so this is never
		// reclassified as owed setup even when it looks like a missing dependency.
		return fmt.Sprintf("  fail  %s — exit %d\n%s", attribution, probe.ExitCode, entrypointOutputVerbatim(probe.Output)),
			doctorStatusFail,
			""

	case entrypointRefused:
		// Doctor's OWN disposition: a check FAILURE. Init surfaces the same value as a
		// CONFIG ERROR, and that asymmetry is deliberate — do not "fix" it in either
		// direction.
		return fmt.Sprintf("  fail  %s — refused before it ran: %v", attribution, probe.Err),
			doctorStatusFail,
			fmt.Sprintf("pack %s declares a provisioned tool that is not on the trusted-tool allowlist; backstop will not run it", probe.PackName)
	}

	return fmt.Sprintf("  fail  %s — unrecognized outcome", attribution), doctorStatusFail, ""
}

// entrypointOutputVerbatim returns the captured output UNALTERED, minus the trailing
// newline the join would double.
//
// IT DOES NOT INDENT, AND THAT IS THE POINT. REQ-006 requires a nonzero exit's output
// reported VERBATIM, and the Message this feeds is what the --json payload carries — so
// reformatting here would put doctor's own whitespace inside the bytes a consumer reads
// as the tool's output. Presentation belongs to the HUMAN renderer, which indents
// continuation lines at print time and leaves the stored message exact.
//
// An empty capture is reported in words rather than as a blank block, because a blank
// block reads as "the report forgot to print it" and cannot be told apart from a capture
// that was never made.
func entrypointOutputVerbatim(output []byte) string {
	text := strings.TrimRight(string(output), "\n")
	if text == "" {
		return "(the command produced no output)"
	}
	return text
}

// checkArtifactLayout reports each artifact-shaped file that is not DIRECTLY in the
// directory the resolution expects for its own kind.
//
// ★ IT CONSUMES THE SHARED RESOLUTION OUTRIGHT AND HOLDS NO LAYOUT KNOWLEDGE OF ITS OWN.
// No artifact directory name, no suffix or filename pattern, no corpus walk, no exclusion
// list — all four come from artifact.ResolveRoot and gate.FindUngatedArtifacts. Three
// independent hardcodings of the root layout are the defect the shared resolution removes,
// and a single literal here would make this the fourth. That is also what makes "what
// doctor calls a deviation" and "what the gate calls ungated" ONE predicate by
// construction rather than two implementations that agree today.
//
// ★ THE PREDICATE IS PER KIND, NEVER ROOT CONTAINMENT, and the difference is not
// stylistic. When no root is configured the resolved root IS the project root, which
// contains everything — so a containment test reports NOTHING on a repo keeping artifacts
// under `.backstop/`, which is precisely the shape this check exists for. The reduced form
// passes everywhere and detects the one thing it was built for nowhere, and no in-repo run
// reveals it because backstop-core is clean under both readings.
//
// A DEVIATION IS REPORTED, NEVER REPAIRED, AND NEVER WIDENS DISCOVERY: naming an ungated
// spec is not an invitation to scan it, move it, or count it.
func checkArtifactLayout(ctx doctorContext) doctorResult {
	result := doctorResult{ID: doctorCheckArtifactLayout, Title: "artifacts sit where the resolved root expects them"}

	// A nil or failed config yields the EMPTY declaration, which resolves to the project
	// root — and must still produce a REPORT rather than an error, because no doctor path
	// may abort.
	declared := ""
	if ctx.Config != nil {
		declared = ctx.Config.ArtifactRoot
	}

	root, rootErr := artifact.ResolveRoot(ctx.ProjectRoot, declared)
	if rootErr != nil {
		// THE TWO TYPED FAILURES ARE DISTINGUISHED BY TYPE, never by string match: they
		// mean genuinely different things, and a caller parsing messages could not tell
		// them apart.
		result.Status = doctorStatusFail
		var missing *artifact.RootMissingError
		if errors.As(rootErr, &missing) {
			result.Message = fmt.Sprintf("the configured artifact_root %q does not exist at %s, so nothing was scanned", missing.Declared, missing.Path)
			result.Remediation = fmt.Sprintf("create %s, or change artifact_root in backstop.yml to the directory that holds this project's artifacts", missing.Path)
			return result
		}
		var invalid *artifact.RootInvalidError
		if errors.As(rootErr, &invalid) {
			result.Message = fmt.Sprintf("the artifact_root declaration %q is invalid: %s", invalid.Declared, invalid.Reason)
			result.Remediation = "declare artifact_root as a project-relative directory inside the project root"
			return result
		}
		result.Message = fmt.Sprintf("the artifact root could not be resolved: %v", rootErr)
		result.Remediation = "fix the artifact_root declaration in backstop.yml"
		return result
	}

	deviations, findErr := gate.FindUngatedArtifacts(ctx.ProjectRoot, root)
	if findErr != nil {
		result.Status = doctorStatusFail
		result.Message = fmt.Sprintf("the artifact corpus under %s could not be read: %v", root.Path, findErr)
		result.Remediation = "make the project root readable, then re-run this check"
		return result
	}

	if len(deviations) == 0 {
		result.Status = doctorStatusPass
		result.Message = fmt.Sprintf("every artifact sits in the directory expected for its kind under %s", root.Path)
		return result
	}

	// A DEVIATION IS LOUD, NOT BLOCKING. It mirrors how the gate itself reports the same
	// finding — the shared conversion stamps these `warning` — and it is the honest
	// severity for a condition whose two remedies are both legitimate consumer choices.
	var lines []string
	for _, deviation := range deviations {
		lines = append(lines, fmt.Sprintf("  %s is not directly in %s", deviation.Path, deviation.ExpectedDir))
	}
	result.Status = doctorStatusWarn
	result.Message = fmt.Sprintf("%d artifact(s) are not gated from where they sit under %s:\n%s",
		len(deviations), root.Path, strings.Join(lines, "\n"))
	// BOTH REMEDIES, ASSERTING NEITHER AS CORRECT: which one is right is the consumer's
	// layout decision, and the shared resolution is policy-FREE — it has no notion that
	// any directory is canonical, so an opinion here would be doctor's own.
	result.Remediation = "either move each file to the expected path above, or declare the artifact_root in backstop.yml that makes its current location the expected one"
	return result
}

// worseDoctorStatus returns the worse of two statuses, ordered fail > warn > pass.
func worseDoctorStatus(current, candidate string) string {
	rank := map[string]int{doctorStatusPass: 0, doctorStatusWarn: 1, doctorStatusFail: 2}
	if rank[candidate] > rank[current] {
		return candidate
	}
	return current
}
