package gate

import (
	"strings"
	"testing"
)

// summaryRowFor returns the SUMMARY-TABLE row for a step — the FIRST line naming
// it. The later occurrence is the "<step> violations:" details header, and
// asserting on whole-output Contains would happily pass on text that landed in
// the wrong place.
func summaryRowFor(t *testing.T, out, stepName string) string {
	t.Helper()
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, stepName) {
			return line
		}
	}
	t.Fatalf("no summary row for %s in output:\n%s", stepName, out)
	return ""
}

// footerLine returns the "Total violations:" footer line.
func footerLine(t *testing.T, out string) string {
	t.Helper()
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "Total violations:") {
			return line
		}
	}
	t.Fatalf("no 'Total violations:' footer in output:\n%s", out)
	return ""
}

// detailLineContaining returns the violation-details line carrying the given
// message text.
func detailLineContaining(t *testing.T, out, message string) string {
	t.Helper()
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, message) && strings.Contains(line, "    - ") {
			return line
		}
	}
	t.Fatalf("no violation-details line carrying %q in output:\n%s", message, out)
	return ""
}

const (
	blockingMessage = "coverage 71.2% below threshold 80.0%"
	warningMessage  = "file excluded from coverage measurement"
)

// mixedSeverityResult is the ISSUE-100 motivating instance: ONE
// coverage_threshold step carrying one blocking shortfall and one non-blocking
// exclusion notice. Pre-fix its summary row read "(2 violations)" and its footer
// read "Total violations: 2" — one blocking problem reported as two.
func mixedSeverityResult() GateResult {
	return NewGateResultWithScope([]StepResult{
		{StepName: StepCoverageThreshold, Status: "fail", Violations: []Violation{
			{Rule: "coverage_threshold", File: "pkg/gate/step_coverage.go", Message: blockingMessage, Severity: "error"},
			{Rule: "coverage_exclusion", File: "pkg/gate/output.go", Message: warningMessage, Severity: "warning"},
		}},
	}, nil)
}

// TestSeverityTally_HumanStepLineSplitsBlockingAndWarnings (CLM-005): the
// per-step summary row reports the two classes SEPARATELY when both are present,
// so a reader is never told that one blocking problem is two.
func TestSeverityTally_HumanStepLineSplitsBlockingAndWarnings(t *testing.T) {
	out := FormatHuman(mixedSeverityResult(), true)
	row := summaryRowFor(t, out, StepCoverageThreshold)

	if !strings.Contains(row, "(1 blocking, 1 warnings)") {
		t.Errorf("summary row must split the classes as \"(1 blocking, 1 warnings)\", got %q", row)
	}
	if strings.Contains(row, "(2 violations)") {
		t.Errorf("summary row must NOT report the pre-fix inflated count \"(2 violations)\", got %q", row)
	}
}

// TestSeverityTally_HumanStepLineWarningsOnly (CLM-005): a step whose entries are
// ALL non-blocking reports them as warnings and says nothing about blocking. The
// status stays "warning" so the fixture is the real capability-absent shape.
func TestSeverityTally_HumanStepLineWarningsOnly(t *testing.T) {
	r := NewGateResultWithScope([]StepResult{
		{StepName: StepCoverageThreshold, Status: "warning", Violations: []Violation{
			{Rule: "coverage", Message: "coverage capability absent for go; declare or waive", Severity: "warning"},
		}},
	}, nil)
	out := FormatHuman(r, true)
	row := summaryRowFor(t, out, StepCoverageThreshold)

	if !strings.Contains(row, "(1 warnings)") {
		t.Errorf("a warning-only step's row must read \"(1 warnings)\", got %q", row)
	}
	if strings.Contains(row, "blocking") {
		t.Errorf("a warning-only step's row must not mention blocking, got %q", row)
	}
	if strings.Contains(row, "violations") {
		t.Errorf("a warning-only step's row must not call its entries violations, got %q", row)
	}
}

// TestSeverityTally_HumanUnchangedWhenNoWarnings (CLM-005, SE4): THE ZERO-CHURN
// CONTRACT. A blocking-only step and a warning-free run render BYTE-IDENTICALLY
// to the pre-fix output. Full-line equality, not substring — a stray appended
// "(+0 warnings)" must fail here. This guard PASSES pre-fix by design; its job is
// to stop the common case being quietly traded for a more uniform-looking format.
func TestSeverityTally_HumanUnchangedWhenNoWarnings(t *testing.T) {
	r := NewGateResultWithScope([]StepResult{
		{StepName: StepCoverageThreshold, Status: "fail", Violations: []Violation{
			{Rule: "coverage_threshold", File: "a.go", Message: "first", Severity: "error"},
			{Rule: "coverage_threshold", File: "b.go", Message: "second", Severity: "error"},
		}},
	}, nil)
	out := FormatHuman(r, true)

	const wantRow = "  coverage_threshold        fail  (2 violations)"
	if row := summaryRowFor(t, out, StepCoverageThreshold); row != wantRow {
		t.Errorf("blocking-only summary row must be byte-identical to the pre-fix render.\n got: %q\nwant: %q", row, wantRow)
	}

	const wantFooter = "  Total violations: 2"
	if footer := footerLine(t, out); footer != wantFooter {
		t.Errorf("a warning-free run's footer must be byte-identical to the pre-fix render.\n got: %q\nwant: %q", footer, wantFooter)
	}
}

// TestSeverityTally_HumanSummaryKeepsTotalViolationsPrefix (CLM-006, SE3): the
// footer's literal "  Total violations: " prefix is load-bearing —
// cmd/backstop/exit_surfacing_streams_test.go asserts on it as a stdout/stderr
// ORDERING probe. The warning count is APPENDED; the prefix is never rewritten.
func TestSeverityTally_HumanSummaryKeepsTotalViolationsPrefix(t *testing.T) {
	out := FormatHuman(mixedSeverityResult(), true)
	footer := footerLine(t, out)

	const prefix = "  Total violations: "
	if !strings.HasPrefix(footer, prefix) {
		t.Errorf("footer must still START WITH %q — cmd/backstop's stream-ordering probe depends on it. got %q", prefix, footer)
	}
	const want = "  Total violations: 1 (+1 warnings)"
	if footer != want {
		t.Errorf("footer = %q, want %q", footer, want)
	}
}

// TestSeverityTally_HumanDetailMarksWarningEntries (CLM-007): each non-blocking
// entry in the details block carries a per-line "[warning]" marker and blocking
// entries carry none, so a reader can map the tally's warning count onto the
// SPECIFIC listed findings instead of guessing which of them inflated the count.
func TestSeverityTally_HumanDetailMarksWarningEntries(t *testing.T) {
	out := FormatHuman(mixedSeverityResult(), true)

	warnLine := detailLineContaining(t, out, warningMessage)
	if !strings.Contains(warnLine, "[warning]") {
		t.Errorf("the warning entry's detail line must carry a [warning] marker, got %q", warnLine)
	}

	blockLine := detailLineContaining(t, out, blockingMessage)
	if strings.Contains(blockLine, "[warning]") {
		t.Errorf("a blocking entry's detail line must carry NO [warning] marker, got %q", blockLine)
	}
}
