package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type CommandRequest struct {
	ID      string
	Spec    string
	Command string
}

type CommandResult struct {
	ExitCode  int
	Fresh     bool
	Skipped   bool
	Synthetic bool
	Engine    string
}

type CommandRunner func(CommandRequest) (CommandResult, error)

type DependencyVerdict struct {
	ID        string
	Spec      string
	Command   string
	ExitCode  int
	Fresh     bool
	Skipped   bool
	Synthetic bool
	Engine    string
}

type PrerequisiteRun struct {
	Verdicts              []DependencyVerdict
	DependentCapabilities []string
	TraversalAllowed      bool
	TraversalRan          bool
}

type DependencyVerdictMutation struct {
	Name             string `yaml:"name"`
	OmitID           string `yaml:"omit_id"`
	OverrideID       string `yaml:"override_id"`
	Skipped          bool   `yaml:"skipped"`
	Stale            bool   `yaml:"stale"`
	Synthetic        bool   `yaml:"synthetic"`
	ExitCode         int    `yaml:"exit_code"`
	Command          string `yaml:"command"`
	Engine           string `yaml:"engine"`
	RequestPrimitive string `yaml:"request_primitive"`
	ExpectedError    string `yaml:"expected_error"`
}

func ExpectedPrerequisites() []MapPrerequisite {
	return []MapPrerequisite{
		{ID: "seed1-product-model", Spec: "SPEC-072", Command: "./scripts/verify-public-product-model.sh"},
		{ID: "seed2-documentation-semantics", Spec: "SPEC-073", Command: "./scripts/verify-documentation-semantics-integration.sh"},
		{ID: "seed3-product-truth", Spec: "SPEC-074", Command: "./scripts/verify-product-truth.sh"},
		{ID: "seed4-public-site", Spec: "SPEC-075", Command: "./scripts/verify-public-site.sh"},
	}
}

func dependentCapabilityIDs() []string {
	artifacts := ExpectedWebsiteCapabilities()
	ids := make([]string, 0, len(artifacts))
	for _, artifact := range artifacts {
		ids = append(ids, artifact.ID)
	}
	return ids
}

func FreshZeroPrerequisiteRunner() CommandRunner {
	return func(CommandRequest) (CommandResult, error) {
		return CommandResult{ExitCode: 0, Fresh: true}, nil
	}
}

func ExecPrerequisiteRunner(root string) CommandRunner {
	return func(req CommandRequest) (CommandResult, error) {
		if !allowedPrerequisiteCommand(req.Command) {
			return CommandResult{Engine: "owner-engine"}, nil
		}
		cmd := exec.Command(req.Command)
		cmd.Dir = root
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err == nil {
			return CommandResult{ExitCode: 0, Fresh: true}, nil
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			return CommandResult{ExitCode: exitErr.ExitCode(), Fresh: true}, nil
		}
		return CommandResult{}, err
	}
}

func allowedPrerequisiteCommand(command string) bool {
	for _, prerequisite := range ExpectedPrerequisites() {
		if prerequisite.Command == command {
			return true
		}
	}
	return false
}

func EvaluatePrerequisites(m WebsiteCapabilityMap, _ CapabilityTree, runner CommandRunner) (PrerequisiteRun, error) {
	dependents := dependentCapabilityIDs()
	blocked := blockedBeforeTraversal(dependents)
	want := ExpectedPrerequisites()
	if len(m.Prerequisites) != len(want) {
		missing := missingPrerequisiteID(m, want)
		return blocked, failBeforeTraversal(missing, dependents, "missing")
	}
	verdicts := make([]DependencyVerdict, 0, len(want))
	for index, expected := range want {
		got := m.Prerequisites[index]
		if got.ID != expected.ID {
			return blocked, failBeforeTraversal(expected.ID, dependents, "missing")
		}
		if got.Spec != expected.Spec || got.Command != expected.Command {
			return blocked, failBeforeTraversal(got.ID, dependents, "wrong-command")
		}
		if runner == nil {
			return blocked, failBeforeTraversal(got.ID, dependents, "missing")
		}
		result, err := runner(CommandRequest{ID: got.ID, Spec: got.Spec, Command: got.Command})
		if err != nil {
			return blocked, failBeforeTraversal(got.ID, dependents, err.Error())
		}
		verdict := DependencyVerdict{
			ID:        got.ID,
			Spec:      got.Spec,
			Command:   got.Command,
			ExitCode:  result.ExitCode,
			Fresh:     result.Fresh,
			Skipped:   result.Skipped,
			Synthetic: result.Synthetic,
			Engine:    result.Engine,
		}
		if verdict.Engine != "" {
			return blocked, failBeforeTraversal(verdict.ID, dependents, "direct-engine "+verdict.Engine)
		}
		if verdict.Skipped {
			return blocked, failBeforeTraversal(verdict.ID, dependents, "skipped")
		}
		if verdict.Synthetic {
			return blocked, failBeforeTraversal(verdict.ID, dependents, "synthetic")
		}
		if !verdict.Fresh {
			return blocked, failBeforeTraversal(verdict.ID, dependents, "stale")
		}
		if verdict.ExitCode != 0 {
			return blocked, failBeforeTraversal(verdict.ID, dependents, fmt.Sprintf("nonzero exit %d", verdict.ExitCode))
		}
		verdicts = append(verdicts, verdict)
	}
	return PrerequisiteRun{
		Verdicts:              verdicts,
		DependentCapabilities: dependents,
		TraversalAllowed:      true,
		TraversalRan:          false,
	}, nil
}

func RefuseUngovernedGenericPrimitive(primitive string) error {
	return fmt.Errorf("%s: ungoverned generic primitive; separately governed by seed4-public-site (SPEC-075)", primitive)
}

func ApplyDependencyVerdictMutation(m WebsiteCapabilityMap, mutation DependencyVerdictMutation) WebsiteCapabilityMap {
	cloned := cloneMap(m)
	if mutation.OmitID != "" {
		filtered := cloned.Prerequisites[:0]
		for _, prerequisite := range cloned.Prerequisites {
			if prerequisite.ID != mutation.OmitID {
				filtered = append(filtered, prerequisite)
			}
		}
		cloned.Prerequisites = filtered
	}
	if mutation.Command != "" && mutation.OverrideID != "" {
		for i := range cloned.Prerequisites {
			if cloned.Prerequisites[i].ID == mutation.OverrideID {
				cloned.Prerequisites[i].Command = mutation.Command
			}
		}
	}
	return cloned
}

func missingPrerequisiteID(m WebsiteCapabilityMap, want []MapPrerequisite) string {
	have := map[string]bool{}
	for _, prerequisite := range m.Prerequisites {
		have[prerequisite.ID] = true
	}
	for _, expected := range want {
		if !have[expected.ID] {
			return expected.ID
		}
	}
	return "prerequisites"
}

func blockedBeforeTraversal(dependents []string) PrerequisiteRun {
	return PrerequisiteRun{
		DependentCapabilities: dependents,
		TraversalAllowed:      false,
		TraversalRan:          false,
	}
}

func failBeforeTraversal(verdictID string, dependents []string, reason string) error {
	return fmt.Errorf("%s: %s; failed before traversal; blocked capabilities: %s", verdictID, reason, strings.Join(dependents, ","))
}
