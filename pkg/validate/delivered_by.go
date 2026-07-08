package validate

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/bmanson/backstop-core/pkg/artifact"
)

// deliveredByRe is the shape a top-level `delivered_by` pointer must take: the
// PLAN-ISSUE-NNN that delivered a closed issue (ISSUE-043). This is the same
// compiled-regex-singleton idiom every validator in this package uses
// (issue.go, plan.go, spec.go): a *regexp.Regexp is immutable after init and
// safe for concurrent use, so it is not the mutable-global hazard the rule
// targets — analogous to the const carve-out the rule already documents.
var deliveredByRe = regexp.MustCompile(`^PLAN-ISSUE-[0-9]{3}$`) // nosemgrep: go.core.no-global-mutable-state — immutable compiled-regex singleton, package idiom

// resolvePlansDir locates the plans/ directory that backs an issue, anchored on
// the ISSUE's OWN source path — NEVER the ambient working directory. Production:
// <root>/issues/ISSUE-NNN.issue.md → <root>/plans. This keeps the delivered_by
// trace a pure function of its inputs: under BACKSTOP_CONFIG the CWD diverges
// from the project root, so a CWD-relative resolution would produce false
// RED/GREEN on a rule every closed issue hits (CLM-012).
//
// It returns the resolved plans dir and whether it is usable. A directory-less
// SourcePath (Parse handed a base-only synthetic filename, so filepath.Dir ==
// ".") or a missing plans/ sibling yields (_, false) — the caller MUST treat
// that as a fail-loud error, never a silent trace-satisfied pass (CLM-011).
//
// Filesystem access is static reads only (filepath.Dir/Join, os.Stat); this
// helper never shells out — pkg/validate has no os/exec and keeps it that way
// (liveness is ISSUE-042's gate job, CLM-009).
func resolvePlansDir(art *artifact.ParsedArtifact) (string, bool) {
	if art.SourcePath == "" {
		return "", false
	}
	dir := filepath.Dir(art.SourcePath)
	if dir == "." || dir == "" {
		// Base-only / directory-less path — cannot anchor a sibling plans/ dir.
		return "", false
	}
	plansDir := filepath.Join(dir, "..", "plans")
	info, err := os.Stat(plansDir)
	if err != nil || !info.IsDir() {
		return "", false
	}
	return plansDir, true
}

// validateDeliveredBy runs the static conditions that let a closed issue satisfy
// traceability by tracing to a delivered backing plan (ISSUE-043). It returns a
// (possibly empty) slice of fail-loud violations; an empty result means the
// trace is satisfied. Every failure mode has a DISTINCT rule name and error
// severity — a bad delivered_by NEVER silently satisfies the close.
//
// The issue→plan linkage is RE-DERIVED here (scan plans, index by spec_id ==
// issue-id) on top of pkg/validate's OWN plan parsing (artifact.ParseFile +
// validate.Plan). It does NOT import pkg/gate: pkg/gate consumes validators, so
// importing it here would invert the layering (CLM-009). The clean-plan check
// REUSES validate.Plan rather than reimplementing plan validation (CLM-005).
func validateDeliveredBy(art *artifact.ParsedArtifact, deliveredBy, issueID string) []Violation {
	// 1. Format check — before any filesystem lookup (CLM-002).
	if !deliveredByRe.MatchString(deliveredBy) {
		return []Violation{{
			Rule:     "issue/delivered-by-malformed",
			File:     art.Filename,
			Message:  fmt.Sprintf("delivered_by '%s' must match PLAN-ISSUE-NNN", deliveredBy),
			Severity: "error",
		}}
	}

	// 2. Resolve the plans/ dir relative to the issue's own path (CLM-011/012).
	plansDir, ok := resolvePlansDir(art)
	if !ok {
		return []Violation{{
			Rule:     "issue/delivered-by-plans-unresolvable",
			File:     art.Filename,
			Message:  fmt.Sprintf("delivered_by '%s' cannot be verified: no plans/ directory resolves from the issue's source path", deliveredBy),
			Severity: "error",
		}}
	}

	// 3. Locate the named plan file (CLM-003). Glob only errors on a malformed
	//    pattern; deliveredBy is regex-validated above (no glob metacharacters),
	//    so an error here is treated the same as "no such plan file".
	matches, err := filepath.Glob(filepath.Join(plansDir, deliveredBy+"-*.plan.yml"))
	if err != nil || len(matches) == 0 {
		return []Violation{{
			Rule:     "issue/delivered-by-plan-not-found",
			File:     art.Filename,
			Message:  fmt.Sprintf("delivered_by '%s' names no plan file (%s-*.plan.yml) under %s", deliveredBy, deliveredBy, plansDir),
			Severity: "error",
		}}
	}

	// 4. Parse the backing plan and validate it clean via the REUSED Plan
	//    validator (CLM-005). A parse error or any plan violation is plan-invalid.
	planArt, err := artifact.ParseFile(matches[0])
	if err != nil {
		return []Violation{{
			Rule:     "issue/delivered-by-plan-invalid",
			File:     art.Filename,
			Message:  fmt.Sprintf("delivered_by '%s' backing plan %s failed to parse: %v", deliveredBy, filepath.Base(matches[0]), err),
			Severity: "error",
		}}
	}
	if planRes := Plan(planArt, nil); len(planRes.Violations) > 0 {
		return []Violation{{
			Rule:     "issue/delivered-by-plan-invalid",
			File:     art.Filename,
			Message:  fmt.Sprintf("delivered_by '%s' backing plan %s does not itself validate clean (%d violation(s))", deliveredBy, filepath.Base(matches[0]), len(planRes.Violations)),
			Severity: "error",
		}}
	}

	// 5. Static-trust the plan's terminal state (CLM-004). Only `completed` — the
	//    plan success-terminal — is a delivered success; draft/ready/implementing
	//    AND the retired terminals replaced/canceled all reject. The validator
	//    trusts this terminal because ISSUE-042 enforces terminal-success ⟹
	//    mandated-tests-exist at the gate; liveness is not re-checked here.
	if status := getFrontmatterString(planArt, "status"); status != "completed" {
		return []Violation{{
			Rule:     "issue/delivered-by-plan-not-completed",
			File:     art.Filename,
			Message:  fmt.Sprintf("delivered_by '%s' backing plan status is '%s', not 'completed' (only a completed plan is a delivered trace)", deliveredBy, status),
			Severity: "error",
		}}
	}

	// 6. Back-match the linkage: the plan must actually back THIS issue (CLM-006).
	if specID := getFrontmatterString(planArt, "spec_id"); specID != issueID {
		return []Violation{{
			Rule:     "issue/delivered-by-spec-mismatch",
			File:     art.Filename,
			Message:  fmt.Sprintf("delivered_by '%s' backing plan spec_id '%s' does not match closing issue '%s'", deliveredBy, specID, issueID),
			Severity: "error",
		}}
	}

	return nil
}
