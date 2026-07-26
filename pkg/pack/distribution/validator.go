package distribution

import (
	"fmt"

	"github.com/bmanson/backstop-core/pkg/packval"
)

// packvalCheckMode and packvalTestMode are the pipeline modes `backstop pack
// check` and `backstop pack test` run. They are named here so the two methods
// below differ in exactly one datum from the commands they mirror.
const (
	packvalCheckMode = "check"
	packvalTestMode  = "test"
)

// packvalFailStatus is the pipeline result status that means the pack was
// rejected. Any other status is an acceptance.
const packvalFailStatus = "fail"

// PackvalValidator is the production Validator: it validates a pack by running
// the same pkg/packval pipeline the pack check and pack test commands run.
//
// It is stateless — every call constructs a FRESH pipeline over the given
// directory, exactly as the commands do — so there is no second implementation of
// pack validation for the two paths to drift apart.
type PackvalValidator struct{}

// NewPackvalValidator constructs the production validator.
//
// It is the concrete implementation, NOT an internal default: nothing inside this
// package may call it to fill a dependency a caller failed to supply. Its only
// production caller is the assembly layer in cmd/backstop, which is what keeps a
// test double from ever being mistakable for production wiring.
func NewPackvalValidator() *PackvalValidator {
	return &PackvalValidator{}
}

// RunPackCheck validates the pack at packDir through the pipeline's check mode —
// the manifest and metadata phases, without fixture execution.
func (v *PackvalValidator) RunPackCheck(packDir string) error {
	return runPackvalPipeline(packDir, packvalCheckMode)
}

// RunPackTest validates the pack at packDir through the pipeline's test mode,
// which adds the fixture phase.
func (v *PackvalValidator) RunPackTest(packDir string) error {
	return runPackvalPipeline(packDir, packvalTestMode)
}

// runPackvalPipeline runs one pipeline over packDir in the given mode and renders
// a rejection as a *ValidationError.
//
// NO FIXTURE EXECUTOR IS SUPPLIED, matching what the commands do: packval's
// fixture phase substitutes its own default executor for a nil one, so this
// validator's fixture behavior is identical to the CLI's BY CONSTRUCTION rather
// than by coincidence. Passing an executor here would create exactly the second
// implementation REQ-003 forbids.
func runPackvalPipeline(packDir, mode string) error {
	result := packval.NewPipeline(packDir, packval.PipelineOptions{Mode: mode}).Run()
	if result.Status != packvalFailStatus {
		return nil
	}

	// The diagnostic names the pack directory, the failing phase, and the error
	// count. The PHASE is the load-bearing part: "validation failed" tells an
	// operator nothing about whether the manifest, the coherence rules, or the
	// fixtures rejected the pack, and those have entirely different fixes.
	return &ValidationError{Message: fmt.Sprintf(
		"pack validation (%s) of %s failed in %s: %d validation error(s)",
		mode, packDir, firstFailingPhase(result), len(result.Errors))}
}

// firstFailingPhase returns the name of the first phase the pipeline reported as
// failing. The pipeline stops at that phase and marks the rest skipped, so it is
// the phase that produced the verdict.
//
// It names the absence rather than returning an empty string, so a future result
// shape that reports errors without a failing phase reads as an anomaly instead
// of silently rendering a diagnostic with a hole in it.
func firstFailingPhase(result *packval.Result) string {
	for _, phase := range result.Phases {
		if phase.Status == packvalFailStatus {
			return phase.Phase
		}
	}

	return "no phase reported a failure"
}
