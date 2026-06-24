package gate

import (
	"fmt"

	"github.com/bmanson/backstop-core/pkg/config"
)

// TraceabilityDimension names one of the three traceability dimensions the
// polarity layer classifies. It is a string so it joins directly against an
// enforcement.toolchain entry's gate_type (the DECLARED discriminator).
type TraceabilityDimension string

// The three traceability dimensions (SPEC-036 REQ-001). A toolchain entry whose
// gate_type equals one of these strings DECLARES that dimension.
const (
	DimensionSubstantiveness TraceabilityDimension = "substantiveness"
	DimensionCoverage        TraceabilityDimension = "coverage"
	DimensionContracts       TraceabilityDimension = "contracts"
)

// knownDimensions is the exhaustive set of recognized dimension names. A
// non-empty gate_type that is NOT in this set is an unknown toolchain key — a
// class-1 BROKEN-DECLARED config defect (Sharp Edge 2), never a silent
// fall-through to UNDECLARED.
var knownDimensions = map[TraceabilityDimension]bool{
	DimensionSubstantiveness: true,
	DimensionCoverage:        true,
	DimensionContracts:       true,
}

// PolarityClass is the fail-loud class a dimension resolves to. The three
// fail-loud classes are mutually exclusive and exhaustive; ClassNone means the
// dimension is declared-and-working (or undeclared-but-present) and proceeds to
// its normal traceability step unchanged (SPEC-036 REQ-001).
type PolarityClass int

const (
	// ClassNone — not a fail-loud class; proceed to the normal analyzer step.
	ClassNone PolarityClass = 0
	// ClassBrokenDeclared (1) — declared dimension whose command errors, emits
	// unparseable output, or names an unknown toolchain key. Blocks (exit 2).
	ClassBrokenDeclared PolarityClass = 1
	// ClassCapabilityAbsent (2) — undeclared dimension with no capability wired
	// for the stack. Warns on the report surface and passes (exit 0), forever.
	ClassCapabilityAbsent PolarityClass = 2
	// ClassDeclaredIntentUnmet (3) — declared dimension whose required capability
	// is missing. A broken promise; blocks (exit 2).
	ClassDeclaredIntentUnmet PolarityClass = 3
)

// String renders a PolarityClass for diagnostics.
func (c PolarityClass) String() string {
	switch c {
	case ClassNone:
		return "none"
	case ClassBrokenDeclared:
		return "broken-declared"
	case ClassCapabilityAbsent:
		return "capability-absent"
	case ClassDeclaredIntentUnmet:
		return "declared-intent-unmet"
	default:
		return fmt.Sprintf("PolarityClass(%d)", int(c))
	}
}

// CapabilityState describes whether a dimension's capability exists for the
// project's stack and whether its declared command ran/parsed cleanly. It is
// computed by the gate wiring (from cfg.Language + baked-analyzer presence on
// the existing binary) and handed to the classifier, which contains NO
// language- or tool-specific branch of its own.
//
//   - Present: the capability exists for this project's stack at all.
//   - Working: the capability ran and parsed cleanly (only meaningful when
//     Present). Present && !Working is the broken-command / unparseable-output
//     condition.
//   - PackOrCommand: the exact pack or command the dimension needs or ran, for
//     fail-loud-and-useful messages.
//   - Detail: free-form detail (the observed failure / expected-vs-got hint).
type CapabilityState struct {
	Present       bool
	Working       bool
	PackOrCommand string
	Detail        string
}

// declaredDimension reports whether dim is DECLARED: cfg.Enforcement.Toolchain
// contains an entry whose gate_type EXACTLY equals dim's name. The join is
// exact (Sharp Edge 2) — a loose/substring match would degrade a broken promise
// into a vacuous green.
func declaredDimension(cfg *config.Config, dim TraceabilityDimension) bool {
	if cfg == nil {
		return false
	}
	for _, tp := range cfg.Enforcement.Toolchain {
		if tp.GateType == string(dim) {
			return true
		}
	}
	return false
}

// hasUnknownGateType reports whether any enforcement.toolchain entry declares a
// non-empty gate_type that matches NO recognized dimension. Such a malformed
// declaration is a class-1 BROKEN-DECLARED config defect (Sharp Edge 2): it must
// NOT be silently read as "undeclared" (which would fall through to a passing
// class 2 and hide the typo).
func hasUnknownGateType(cfg *config.Config) bool {
	if cfg == nil {
		return false
	}
	for _, tp := range cfg.Enforcement.Toolchain {
		if tp.GateType == "" {
			continue
		}
		if !knownDimensions[TraceabilityDimension(tp.GateType)] {
			return true
		}
	}
	return false
}

// ClassifyDimension classifies a single dimension into a PolarityClass by
// reading ONLY the declaration surface (cfg) and the supplied CapabilityState.
// It is language-agnostic: stack-specific capability knowledge lives in the
// wiring that computes cap, never in this function. The decision table (the
// complete allowlist, SPEC-036):
//
//	unknown gate_type anywhere           -> (1) BROKEN-DECLARED
//	declared + present + !working        -> (1) BROKEN-DECLARED
//	declared + !present                  -> (3) DECLARED-INTENT-UNMET
//	declared + present + working         -> none (proceed)
//	undeclared + !present                -> (2) CAPABILITY-ABSENT
//	undeclared + present                 -> none (proceed)
func ClassifyDimension(cfg *config.Config, dim TraceabilityDimension, cap CapabilityState) PolarityClass {
	// A malformed/unknown declared key is a config defect that blocks (Sharp
	// Edge 2) — checked first so it can never silently degrade to class 2.
	if hasUnknownGateType(cfg) {
		return ClassBrokenDeclared
	}

	declared := declaredDimension(cfg, dim)

	if declared {
		if !cap.Present {
			// Declared but the capability it needs is missing — broken promise.
			return ClassDeclaredIntentUnmet
		}
		if !cap.Working {
			// Declared, capability present, but command errored / output
			// unparseable — broken declaration.
			return ClassBrokenDeclared
		}
		// Declared and working — proceed to the normal analyzer step.
		return ClassNone
	}

	// Undeclared.
	if !cap.Present {
		// No declaration and no capability wired for the stack — un-adopted
		// capability; warn-and-pass forever.
		return ClassCapabilityAbsent
	}
	// Undeclared but a capability is present (e.g. the baked Go analyzer) —
	// proceed to the normal step.
	return ClassNone
}

// waivedDimension reports whether dim is waived: a toolchain entry whose
// gate_type EXACTLY equals dim's name has waived: true. The waive keys off the
// SAME exact gate_type join as declaration (Sharp Edge 2/3), so a waive can only
// ever target a real declared dimension.
func waivedDimension(cfg *config.Config, dim TraceabilityDimension) bool {
	if cfg == nil {
		return false
	}
	for _, tp := range cfg.Enforcement.Toolchain {
		if tp.GateType == string(dim) && tp.Waived {
			return true
		}
	}
	return false
}

// stackLabel returns the project's stack/language for fail-loud messages,
// defaulting to "unspecified" when cfg.Language is empty.
func stackLabel(cfg *config.Config) string {
	if cfg == nil || cfg.Language == "" {
		return "unspecified"
	}
	return cfg.Language
}

// capLabel returns the exact pack/command a dimension needs or ran, for
// fail-loud-and-useful messages, defaulting to a generic phrase when the wiring
// supplied none.
func capLabel(cap CapabilityState) string {
	if cap.PackOrCommand != "" {
		return cap.PackOrCommand
	}
	return "the required capability"
}

// PolarityStepResult converts a PolarityClass into a StepResult (SPEC-036
// REQ-002/003/004/005/006/007):
//
//   - Class 1 / Class 3 → Status "fail", ConfigErr true (block exit 2), with a
//     fail-loud-and-useful Violation. Class 1 carries expected-vs-got.
//   - Class 2 → non-failing "warning" status, ConfigErr false, a conspicuous
//     advisory Violation tagged Severity "warning". If the dimension is waived
//     (same exact gate_type join as classification) the advisory is suppressed
//     to a plain "pass".
//
// A waive can NEVER reach a class-1/class-3 result: the waive branch is only
// consulted for class 2 (CLM-023/024).
func PolarityStepResult(stepName string, dim TraceabilityDimension, class PolarityClass, cfg *config.Config, cap CapabilityState) StepResult {
	stack := stackLabel(cfg)
	pc := capLabel(cap)

	switch class {
	case ClassBrokenDeclared:
		// Class 1: broken declaration — command errored, output unparseable, or
		// an unknown toolchain key. Block (exit 2) with expected-vs-got detail.
		detail := cap.Detail
		if detail == "" {
			detail = "the declared command failed or its output was unparseable"
		}
		msg := fmt.Sprintf(
			"traceability dimension %q is DECLARED for the %s stack but its declared command %q is broken: %s. Expected the declared command/format to run and parse cleanly; got: %s. Fix the command or its declared format, or remove the declaration.",
			dim, stack, pc, detail, detail,
		)
		return StepResult{
			StepName:   stepName,
			Status:     "fail",
			ConfigErr:  true,
			Violations: []Violation{{Rule: string(dim) + "_broken_declared", Message: msg, Severity: "error"}},
		}

	case ClassDeclaredIntentUnmet:
		// Class 3: declared but the required capability is missing — a broken
		// promise. Block (exit 2).
		msg := fmt.Sprintf(
			"traceability dimension %q is DECLARED for the %s stack but its required capability (%s) is missing — a broken promise. Install/declare the capability that provides %q for the %s stack, or fix the declaration.",
			dim, stack, pc, dim, stack,
		)
		return StepResult{
			StepName:   stepName,
			Status:     "fail",
			ConfigErr:  true,
			Violations: []Violation{{Rule: string(dim) + "_declared_intent_unmet", Message: msg, Severity: "error"}},
		}

	case ClassCapabilityAbsent:
		// Class 2: undeclared, no capability wired for the stack. Waivable.
		// Warn-and-pass forever; never auto-promotes to blocking.
		if waivedDimension(cfg, dim) {
			// Waived: suppress the advisory to a plain pass.
			return StepResult{
				StepName:   stepName,
				Status:     "pass",
				Violations: []Violation{},
				Reason:     fmt.Sprintf("%s capability-absent advisory waived", dim),
			}
		}
		msg := fmt.Sprintf(
			"traceability dimension %q has no capability wired for the %s stack — adopt %s (e.g. a %s pack), then declare it via enforcement.toolchain gate_type: %s; or waive it (waived: true). This advisory is non-blocking (exit 0).",
			dim, stack, pc, stack, dim,
		)
		return StepResult{
			StepName:   stepName,
			Status:     "warning",
			ConfigErr:  false,
			Violations: []Violation{{Rule: string(dim) + "_capability_absent", Message: msg, Severity: "warning"}},
		}

	default:
		// ClassNone — proceed; not a fail-loud result. Return a plain pass so a
		// caller that maps every class through this function stays well-defined.
		return StepResult{
			StepName:   stepName,
			Status:     "pass",
			Violations: []Violation{},
		}
	}
}
