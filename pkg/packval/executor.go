package packval

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/backstop-ai/backstop-core/pkg/check"
	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
)

// FixtureExecutor runs a pack's declared fixtures against their resolved engines.
// The findings path is GENERIC: RunEngine takes a resolved engine.EngineBinding
// (pack DATA) and the fixture targets — there is no tool-named method and no baked
// tool switch. RunValidator/RunScaffoldTest remain the sandbox/scaffold seams.
type FixtureExecutor interface {
	// RunEngine dispatches a resolved engine binding (command from pack DATA) at the
	// given targets. It is the single generic replacement for the retired tool-named
	// fixture methods (ISSUE-019).
	RunEngine(packDir string, binding engine.EngineBinding, targets []string) (ExecutionResult, error)
	RunValidator(packDir, validator string, fixturePaths []string) (ExecutionResult, error)
	RunScaffoldTest(packDir, scaffoldPath, testCommand string) (ExecutionResult, error)
}

type ExecutionResult struct {
	Passed      bool     `json:"passed"`
	Output      string   `json:"output,omitempty"`
	ExitCode    int      `json:"exit_code"`
	Diagnostics []string `json:"diagnostics,omitempty"`
}

type DefaultExecutor struct{}

// buildEngineArgv constructs the command + args from the engine binding DATA: the
// binding.Command supplies the executable and its baked-in flags, the InputFlag (when
// set) is injected once, and the resolved targets (rule/config file + fixture) are
// appended. No tool name is ever a literal here — it comes entirely from the binding.
func buildEngineArgv(binding engine.EngineBinding, targets []string) (string, []string) {
	fields := strings.Fields(binding.Command)
	if len(fields) == 0 {
		return "", nil
	}
	name := fields[0]
	args := append([]string{}, fields[1:]...)
	if binding.InputFlag != "" {
		args = append(args, binding.InputFlag)
	}
	args = append(args, targets...)
	return name, args
}

// RunEngine runs the resolved engine binding at the targets and reports whether the
// engine FIRED (produced findings). The trust floor is enforced FIRST: a provisioned
// tool that is not on the trusted-tool allowlist (or not pinned to its allowlisted
// version) is NEVER executed — engine.CheckToolAllowed fail-louds before the command
// is handed to the runner (SPEC-035 REQ-002). "Fired" is decided from the engine's
// SARIF output — the universal contract backstop speaks — so the signal is
// tool/language-blind: a positive fixture that the rule matches yields >=1 finding
// (Passed=true), a clean negative yields zero (Passed=false).
func (d *DefaultExecutor) RunEngine(packDir string, binding engine.EngineBinding, targets []string) (ExecutionResult, error) {
	if binding.Provision != nil {
		if err := engine.CheckToolAllowed(
			engine.TrustedToolAllowlist(),
			binding.Provision.Tool,
			binding.Provision.Version,
		); err != nil {
			return ExecutionResult{}, fmt.Errorf("engine %q failed the trusted-tool allowlist gate: %w", binding.Command, err)
		}
	}
	name, args := buildEngineArgv(binding, targets)
	if name == "" {
		return ExecutionResult{}, fmt.Errorf("engine binding has no command to run")
	}
	// PRODUCER vs plain-command split (ISSUE-160), mirroring the FINDINGS-path
	// semantics cmd/backstop's runFindingsEngine already carries. binding.Producer
	// is an optional pack-relative script run IN PLACE of the tool: a pack declares
	// one when its tool splits its real output across streams a stdout-only capture
	// cannot both see, and the PACK owns that merge because the pack is what knows
	// its tool's stream behavior. RunEngine previously ignored the field entirely,
	// so a pack declaring a producer silently got the plain command — and an empty
	// or wrong-shaped payload parses to ZERO findings with NO error, which is a
	// POSITIVE phase-3 fixture's silent clean pass over a run that invoked the
	// wrong program.
	//
	// THE ASYMMETRY WITH THE COVERAGE PATH IS DELIBERATE. A coverage producer is
	// invoked BARE because it shapes its own whole invocation. A findings producer
	// must NOT be: the fixture TARGETS are the entire point of a phase-3 run, so a
	// bare call would drop the rule/config file and the fixture path and every
	// fixture would come back not-fired. The producer therefore REPLACES THE
	// INVOKED NAME ONLY and receives THE SAME ARGS the plain command would have.
	//
	// UN-SANDBOXED BY CONSTRUCTION: this exec.Command is already un-sandboxed (only
	// the convert step below goes through SandboxedRunStdout), so the swap inherits
	// the same property the gate gets from calling its runner directly. Routing the
	// producer through the sandbox would be a divergence dressed up as caution.
	invoked := name
	// What the never-started refusal below NAMES. On the plain branch that is the
	// declared command (wording unchanged); on the producer branch it is the
	// resolved script path, because the producer is what failed to start and naming
	// the command would misdirect the reader to a tool that was never reached.
	neverStartedSubject := binding.Command
	if binding.Producer != "" {
		// Resolved under packDir with the SAME filepath.Join+os.Stat pattern the
		// convert block below uses. A declared-but-missing producer is a fail-loud
		// broken-pack error naming the declared value AND the resolved path — never
		// a silent fall-back to the plain command, which would make a mis-typed
		// declaration look like it worked.
		producerPath := filepath.Join(packDir, filepath.FromSlash(binding.Producer))
		if info, statErr := os.Stat(producerPath); statErr != nil || info.IsDir() {
			return ExecutionResult{Passed: false, ExitCode: 1},
				fmt.Errorf("engine %q: declared producer %q is missing or not a file (%s)", binding.Command, binding.Producer, producerPath)
		}
		invoked = producerPath
		neverStartedSubject = producerPath
	}
	cmd := exec.Command(invoked, args...)
	cmd.Dir = packDir
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	// A findings engine legitimately exits non-zero WHEN it reports findings, so the
	// exit code is not the contract — the SARIF on stdout is. A run whose PROCESS
	// NEVER STARTED, however, is a broken run and not a finding-free pass — fail loud
	// so a missing engine never reads as a clean negative (vacuous green).
	// check.NeverStarted is the single authority for that class, and it is a CLASS,
	// not `runErr != nil`: binding.Command is pack DATA that may carry a path
	// separator, and a path-ful command that cannot be exec'd never reaches LookPath,
	// so it reports an *fs.PathError rather than the *exec.Error this check once
	// matched alone (ISSUE-140).
	runErr := cmd.Run()
	if check.NeverStarted(runErr) {
		return ExecutionResult{Passed: false, Output: stdout.String(), ExitCode: 1},
			fmt.Errorf("engine %q never started: the process could not be executed — this is a broken run, not a finding-free pass; ensure the command is present and executable: %w", neverStartedSubject, runErr)
	}
	// Apply the binding's declared convert BEFORE parsing (ISSUE-141). packval's
	// executor previously handed raw stdout to the parser, so a pack whose engine
	// emits non-SARIF output and ships a reshaper died at the parse on output the
	// convert would have made parseable.
	//
	// ORDER: strictly after the never-started refusal above. A process that never
	// started produced no bytes, so running a convert over its empty stdout would
	// manufacture a convert-step failure to describe an engine that never ran.
	//
	// NO IMPORT: SandboxedRunStdout is same-package. cmd/backstop's convert reaches
	// it through resolveSandboxedRunStdout, whose production value IS this function
	// — the gate side has always called INTO this package for its convert, so both
	// paths carry the same sandboxing guarantee rather than two approximations.
	payload := stdout.Bytes()
	// SELECT the payload BEFORE the convert (ISSUE-144). A binding that declares a
	// stdout_artifact writes its real machine-readable output to THAT FILE (relative
	// to the run's working dir, which here is packDir — cmd.Dir above) and prints only
	// a human summary, or nothing, to stdout. Reading stdout in that case throws the
	// engine's actual output away, and parseSarif reads empty stdout as ZERO findings
	// with NO error — a Passed=false/nil-error verdict that IS the success condition
	// for a negative fixture. A clean pass over a run whose output was never read.
	//
	// The base is packDir, not a project root: cmd/backstop's runFindingsEngine joins
	// against projectRoot because THAT is its run's working dir. Same rule, different
	// value.
	//
	// ORDER: strictly after the never-started refusal (a process that never ran
	// produced no artifact, so blaming the missing file would misattribute the
	// failure) and strictly before the convert (which must reshape the SELECTED
	// bytes). A declared-but-unproduced artifact is a fail-loud broken run, never a
	// silent fall-back to stdout — that fall-back is this defect re-expressed as
	// something that looks fixed.
	if binding.StdoutArtifact != "" {
		artifactPath := filepath.Join(packDir, filepath.FromSlash(binding.StdoutArtifact))
		body, readErr := os.ReadFile(artifactPath)
		if readErr != nil {
			return ExecutionResult{Passed: false, Output: stdout.String(), ExitCode: 1},
				fmt.Errorf("engine %q: declared stdout_artifact %q not produced (%s): %w", binding.Command, binding.StdoutArtifact, artifactPath, readErr)
		}
		payload = body
	}
	// STRICT-SARIF SHAPE GUARD (ISSUE-160), mirroring the MECHANISM and the GATING
	// of the gate's own guard — never its wording. A binding declares StrictSarif
	// when its engine emits native SARIF; without this guard an engine that emitted
	// some OTHER valid JSON instead got the lenient parse, which unmarshals any JSON
	// object into the SARIF log struct, finds no runs, and returns ZERO findings with
	// NO error. That is a POSITIVE phase-3 fixture's silent clean pass over output
	// the parser could not read.
	//
	// FOUR THINGS ARE LOAD-BEARING:
	//   - binding.StrictSarif keeps the guard OPT-IN; an engine that does not promise
	//     SARIF keeps the deliberate lenient read.
	//   - binding.Convert == "" is the gating. Non-SARIF input is PRECISELY what a
	//     convert exists to reshape, so guarding a convert-declaring binding's
	//     pre-convert payload would fail loud on a correctly-authored pack.
	//   - THE SUBJECT IS payload, NOT stdout: the stdout_artifact selection above has
	//     already chosen the artifact's bytes when one was declared, and reading raw
	//     stdout here would silently re-open that defect on this path.
	//   - the empty-payload early return is DELIBERATE. A genuinely clean run may emit
	//     nothing; the emptiness-is-suspicious discipline belongs to the crash guard
	//     and the never-started refusal, not to a SHAPE check.
	//
	// THE MESSAGE NAMES NO TOOL AND NO LANGUAGE. The gate's twin names a specific
	// linter because it lives in a tool-specific file; this is the generic
	// pack-validation dispatch, where a baked tool literal violates the
	// zero-baked-language first principle. It describes the SHAPE and the
	// CONSEQUENCE, and names the engine via the binding's own declared command.
	if binding.StrictSarif && binding.Convert == "" {
		if trimmed := bytes.TrimSpace(payload); len(trimmed) > 0 {
			var probe struct {
				Runs *json.RawMessage `json:"runs"`
			}
			if err := json.Unmarshal(trimmed, &probe); err != nil || probe.Runs == nil {
				return ExecutionResult{Passed: false, Output: stdout.String(), ExitCode: 1},
					fmt.Errorf("engine %q declares strict SARIF but its output is not a SARIF log (no `runs` array) — a lenient parse would read this as zero findings", binding.Command)
			}
		}
	}
	if binding.Convert != "" {
		convertPath := filepath.Join(packDir, filepath.FromSlash(binding.Convert))
		if info, statErr := os.Stat(convertPath); statErr != nil || info.IsDir() {
			return ExecutionResult{Passed: false, Output: stdout.String(), ExitCode: 1},
				fmt.Errorf("engine %q declares a convert script that is missing or not a file: %s", binding.Command, convertPath)
		}
		converted, convErr := SandboxedRunStdout(convertPath, nil, packDir, payload)
		if convErr != nil {
			return ExecutionResult{Passed: false, Output: stdout.String(), ExitCode: 1},
				fmt.Errorf("engine %q: convert step (%s) failed: %w", binding.Command, binding.Convert, convErr)
		}
		payload = converted
	}
	findings, parseErr := check.ParsePackFindings(payload)
	if parseErr != nil {
		return ExecutionResult{Passed: false, Output: stdout.String(), ExitCode: 1},
			fmt.Errorf("engine %q produced no parseable SARIF: %w", binding.Command, parseErr)
	}
	// CRASH-vs-FINDINGS GUARD (ISSUE-160), mirroring the gate's own. For a binding
	// that declares CrashGuard — a native build/test engine — a non-zero run that
	// yields ZERO parseable findings is a tool/infra crash, not a finding-free pass.
	// Without this, such a run returned Passed=false with a NIL error: precisely the
	// verdict a POSITIVE phase-3 fixture accepts as a silent clean pass, so a crashed
	// engine validated the pack over a run that produced nothing at all.
	//
	// ALL THREE CONJUNCTS ARE LOAD-BEARING:
	//   - binding.CrashGuard keeps the guard OPT-IN. Every rule-fed findings engine
	//     leaves it false, and its exit code is not its contract — the SARIF is.
	//   - runErr != nil restricts it to a non-zero exit. A CLEAN exit with zero
	//     findings is an ordinary positive fixture passing (the clean example
	//     produced no finding) and must stay error-free.
	//   - len(findings) == 0 is what makes it a CRASH rather than a report: an engine
	//     that exits non-zero BECAUSE it emitted findings is working correctly.
	//
	// POSITION: by here the never-started refusal has already returned for a process
	// that never ran, so runErr is a STARTED process's exit status — which is what
	// makes the word "crashed" accurate rather than a misdescription of a binary that
	// never executed at all.
	if binding.CrashGuard && runErr != nil && len(findings) == 0 {
		return ExecutionResult{Passed: false, Output: stdout.String(), ExitCode: 1},
			fmt.Errorf("engine %q crashed: non-zero exit with no parseable findings: %w", binding.Command, runErr)
	}
	return ExecutionResult{Passed: len(findings) > 0, Output: stdout.String(), ExitCode: 0}, nil
}

func (d *DefaultExecutor) RunValidator(packDir, validator string, fixturePaths []string) (ExecutionResult, error) {
	out, err := SandboxedRun(validator, fixturePaths, packDir)
	if err != nil {
		return ExecutionResult{Passed: false, Output: string(out), ExitCode: 1}, nil
	}
	return ExecutionResult{Passed: true, Output: string(out), ExitCode: 0}, nil
}

func (d *DefaultExecutor) RunScaffoldTest(packDir, scaffoldPath, testCommand string) (ExecutionResult, error) {
	cmd := exec.Command("sh", "-c", testCommand)
	cmd.Dir = packDir + "/" + scaffoldPath
	out, err := cmd.CombinedOutput()
	return resultFromCmd(out, err), nil
}

type MockExecutor struct {
	EngineFn       func(packDir string, binding engine.EngineBinding, targets []string) (ExecutionResult, error)
	ValidatorFn    func(packDir, validator string, fixturePaths []string) (ExecutionResult, error)
	ScaffoldTestFn func(packDir, scaffoldPath, testCommand string) (ExecutionResult, error)
}

func (m *MockExecutor) RunEngine(packDir string, binding engine.EngineBinding, targets []string) (ExecutionResult, error) {
	if m.EngineFn != nil {
		return m.EngineFn(packDir, binding, targets)
	}
	return ExecutionResult{Passed: true, ExitCode: 0}, nil
}
func (m *MockExecutor) RunValidator(packDir, validator string, fixturePaths []string) (ExecutionResult, error) {
	if m.ValidatorFn != nil {
		return m.ValidatorFn(packDir, validator, fixturePaths)
	}
	return ExecutionResult{Passed: true, ExitCode: 0}, nil
}
func (m *MockExecutor) RunScaffoldTest(packDir, scaffoldPath, testCommand string) (ExecutionResult, error) {
	if m.ScaffoldTestFn != nil {
		return m.ScaffoldTestFn(packDir, scaffoldPath, testCommand)
	}
	return ExecutionResult{Passed: true, ExitCode: 0}, nil
}

func resultFromCmd(out []byte, err error) ExecutionResult {
	r := ExecutionResult{Output: string(out), ExitCode: 0, Passed: true}
	if err == nil {
		return r
	}
	r.Passed = false
	if exitErr, ok := err.(*exec.ExitError); ok {
		r.ExitCode = exitErr.ExitCode()
	} else {
		r.ExitCode = 1
	}
	return r
}
