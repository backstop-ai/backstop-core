package initialize

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// stepGitName is the report name for step 1.
const stepGitName = "git"

// stepGit runs `git init` ONLY when the target directory contains no `.git`
// (SPEC-069 REQ-006).
//
// An existing repository is left ENTIRELY untouched: no re-init, no config write, no
// ref or HEAD mutation. `git init` over an existing repository is documented as safe,
// and that is exactly why the guard matters — a harmless re-init still rewrites the
// config file's mtime and would make init a command that edits a consumer's git state
// without being asked to.
//
// THE ONLY FACT THIS STEP INSPECTS IS THE PRESENCE OF `.git`. Nothing else in the
// project is read, which is what keeps init's decisions backstop-neutral.
func stepGit(projectRoot string) StepReport {
	if exists := directoryExists(filepath.Join(projectRoot, ".git")); exists {
		return StepReport{
			Step:    stepGitName,
			Outcome: OutcomeConverged,
			Detail:  "a git repository is already present; its HEAD, config and refs were left untouched",
		}
	}

	cmd := exec.CommandContext(context.Background(), "git", "init")
	cmd.Dir = projectRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		return StepReport{
			Step:    stepGitName,
			Outcome: OutcomeBrokenPromise,
			Detail: fmt.Sprintf("git init failed in %s: %v%s",
				projectRoot, err, formatCapturedOutput(output)),
		}
	}

	return StepReport{
		Step:    stepGitName,
		Outcome: OutcomeDelivered,
		Detail:  fmt.Sprintf("initialized a git repository at %s", projectRoot),
	}
}

// formatCapturedOutput renders a command's captured output for a report line,
// contributing nothing when there was none.
func formatCapturedOutput(output []byte) string {
	trimmed := strings.TrimSpace(string(output))
	if trimmed == "" {
		return ""
	}
	return "\n" + trimmed
}
