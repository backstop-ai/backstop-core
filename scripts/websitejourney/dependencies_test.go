package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type dependencyVerdictMutationsFile struct {
	InvalidVerdicts  []DependencyVerdictMutation `yaml:"invalid_verdicts"`
	CapabilityBreaks []DependencyVerdictMutation `yaml:"capability_breaks"`
}

func loadDependencyVerdictMutations(t *testing.T) dependencyVerdictMutationsFile {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "dependency-verdict-mutations.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var document dependencyVerdictMutationsFile
	if err := yaml.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	if len(document.InvalidVerdicts) == 0 || len(document.CapabilityBreaks) == 0 {
		t.Fatal("dependency-verdict-mutations.yml must declare invalid_verdicts and capability_breaks")
	}
	return document
}

func TestWebsiteJourney_ExactDependencyVerdictMatrixPasses(t *testing.T) {
	m := mustLoadWebsiteCapabilityMap(t)
	tree := mustLoadCapabilityTree(t)
	result, err := EvaluatePrerequisites(m, tree, FreshZeroPrerequisiteRunner())
	if err != nil {
		t.Fatal(err)
	}

	want := ExpectedPrerequisites()
	if len(result.Verdicts) != len(want) {
		t.Fatalf("verdict cardinality: got %d, want %d", len(result.Verdicts), len(want))
	}
	for index, expected := range want {
		got := result.Verdicts[index]
		if got.ID != expected.ID || got.Spec != expected.Spec || got.Command != expected.Command {
			t.Fatalf("verdicts[%d] = %s %s %s, want %s %s %s", index, got.ID, got.Spec, got.Command, expected.ID, expected.Spec, expected.Command)
		}
		if got.ExitCode != 0 || !got.Fresh || got.Skipped || got.Synthetic || got.Engine != "" {
			t.Fatalf("%s: want attributed fresh zero exit, got %+v", got.ID, got)
		}
	}

	wantCaps := ExpectedWebsiteCapabilities()
	if len(result.DependentCapabilities) != len(wantCaps) {
		t.Fatalf("dependent capabilities: got %d, want %d", len(result.DependentCapabilities), len(wantCaps))
	}
	for index, expected := range wantCaps {
		if result.DependentCapabilities[index] != expected.ID {
			t.Fatalf("dependents[%d] = %s, want %s", index, result.DependentCapabilities[index], expected.ID)
		}
	}
	if !result.TraversalAllowed {
		t.Fatal("successful attributed verdicts must allow traversal")
	}
	if result.TraversalRan {
		t.Fatal("prerequisite evaluation must not start traversal")
	}
}

func TestWebsiteJourney_RejectsInvalidDependencyVerdictMatrix(t *testing.T) {
	document := loadDependencyVerdictMutations(t)
	base := mustLoadWebsiteCapabilityMap(t)
	tree := mustLoadCapabilityTree(t)
	for _, mutation := range document.InvalidVerdicts {
		t.Run(mutation.Name, func(t *testing.T) {
			var err error
			if mutation.RequestPrimitive != "" {
				err = RefuseUngovernedGenericPrimitive(mutation.RequestPrimitive)
			} else {
				_, err = EvaluatePrerequisites(ApplyDependencyVerdictMutation(base, mutation), tree, prerequisiteRunnerFor(mutation))
			}
			if err == nil {
				t.Fatal("accepted invalid dependency verdict matrix")
			}
			if !strings.Contains(err.Error(), mutation.ExpectedError) {
				t.Fatalf("error %q does not name %q", err, mutation.ExpectedError)
			}
			if mutation.RequestPrimitive == "" && strings.Contains(err.Error(), "traversal ran") {
				t.Fatalf("invalid verdict must fail before traversal: %v", err)
			}
		})
	}
}

func TestWebsiteJourney_EveryDependencyVerdictRemovalBreaksCapabilities(t *testing.T) {
	document := loadDependencyVerdictMutations(t)
	base := mustLoadWebsiteCapabilityMap(t)
	tree := mustLoadCapabilityTree(t)
	if len(document.CapabilityBreaks) != 4 {
		t.Fatalf("capability_breaks = %d, want one removal per upstream verdict", len(document.CapabilityBreaks))
	}
	for _, mutation := range document.CapabilityBreaks {
		t.Run(mutation.OmitID, func(t *testing.T) {
			result, err := EvaluatePrerequisites(ApplyDependencyVerdictMutation(base, mutation), tree, FreshZeroPrerequisiteRunner())
			if err == nil {
				t.Fatalf("omitting %s still allowed capabilities to proceed", mutation.OmitID)
			}
			if !strings.Contains(err.Error(), mutation.OmitID) {
				t.Fatalf("error %q does not name omitted verdict %s", err, mutation.OmitID)
			}
			if result.TraversalAllowed || result.TraversalRan {
				t.Fatalf("%s: traversal must not run after a missing verdict", mutation.OmitID)
			}
			for _, capability := range ExpectedWebsiteCapabilities() {
				if !strings.Contains(err.Error(), capability.ID) {
					t.Fatalf("omitting %s did not break %s before traversal: %v", mutation.OmitID, capability.ID, err)
				}
			}
		})
	}
}

func TestWebsiteJourney_RefusesUngovernedGenericPrimitive(t *testing.T) {
	err := RefuseUngovernedGenericPrimitive("generic-capability-runtime")
	if err == nil {
		t.Fatal("Seed 5 accepted an ungoverned generic capability primitive")
	}
	if !strings.Contains(err.Error(), "generic-capability-runtime") {
		t.Fatalf("error %q does not name the requested primitive", err)
	}
	named := false
	for _, prerequisite := range ExpectedPrerequisites() {
		if strings.Contains(err.Error(), prerequisite.ID) {
			named = true
			break
		}
	}
	if !named {
		t.Fatalf("error %q does not name a separately governed dependency", err)
	}
	if strings.Contains(strings.ToLower(err.Error()), "implemented in seed 5") {
		t.Fatalf("refusal expanded Seed 5 instead of stopping at a named dependency: %v", err)
	}
}

func TestWebsiteJourney_CLIAndExecPrerequisiteRunner(t *testing.T) {
	root := websiteJourneyRepoRoot(t)
	if code := run([]string{"-bogus"}, io.Discard, io.Discard); code != 2 {
		t.Fatalf("unknown flag exit = %d, want 2", code)
	}
	if code := run([]string{"-root", root}, io.Discard, io.Discard); code != 0 {
		t.Fatalf("artifact-only CLI exit = %d", code)
	}
	if code := run([]string{"-root", root, "-capability", "CAP-004"}, io.Discard, io.Discard); code != 0 {
		t.Fatalf("capability CLI exit = %d", code)
	}
	if code := run([]string{"-root", root}, failingWriter{}, io.Discard); code != 1 {
		t.Fatalf("stdout failure exit = %d, want 1", code)
	}
	if code := run([]string{"-root", root, "-capability", "CAP-004"}, failingWriter{}, io.Discard); code != 1 {
		t.Fatalf("capability stdout failure exit = %d, want 1", code)
	}
	if code := run([]string{"-root", t.TempDir()}, io.Discard, io.Discard); code != 1 {
		t.Fatalf("missing root exit = %d, want 1", code)
	}

	previous := newPrerequisiteRunner
	t.Cleanup(func() { newPrerequisiteRunner = previous })
	newPrerequisiteRunner = func(string) CommandRunner { return FreshZeroPrerequisiteRunner() }
	if code := run([]string{"-root", root, "-prerequisites"}, io.Discard, io.Discard); code != 0 {
		t.Fatalf("prerequisites CLI exit = %d", code)
	}
	newPrerequisiteRunner = func(string) CommandRunner {
		return func(CommandRequest) (CommandResult, error) {
			return CommandResult{ExitCode: 1, Fresh: true}, nil
		}
	}
	if code := run([]string{"-root", root, "-prerequisites"}, io.Discard, io.Discard); code != 1 {
		t.Fatalf("failed prerequisites CLI exit = %d, want 1", code)
	}

	stubs := t.TempDir()
	for _, prerequisite := range ExpectedPrerequisites() {
		path := filepath.Join(stubs, prerequisite.Command)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	zero, err := ExecPrerequisiteRunner(stubs)(CommandRequest(ExpectedPrerequisites()[0]))
	if err != nil || zero.ExitCode != 0 || !zero.Fresh {
		t.Fatalf("exec zero = %+v err=%v", zero, err)
	}
	failPath := filepath.Join(stubs, ExpectedPrerequisites()[1].Command)
	if err := os.WriteFile(failPath, []byte("#!/bin/sh\nexit 3\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	nonzero, err := ExecPrerequisiteRunner(stubs)(CommandRequest(ExpectedPrerequisites()[1]))
	if err != nil || nonzero.ExitCode != 3 {
		t.Fatalf("exec nonzero = %+v err=%v", nonzero, err)
	}
	engine, err := ExecPrerequisiteRunner(stubs)(CommandRequest{Command: "jekyll"})
	if err != nil || engine.Engine != "owner-engine" {
		t.Fatalf("disallowed command = %+v err=%v", engine, err)
	}
	if _, err := ExecPrerequisiteRunner(t.TempDir())(CommandRequest(ExpectedPrerequisites()[0])); err == nil {
		t.Fatal("missing public entrypoint must surface an exec error")
	}

	m := mustLoadWebsiteCapabilityMap(t)
	tree := mustLoadCapabilityTree(t)
	if _, err := EvaluatePrerequisites(m, tree, nil); err == nil {
		t.Fatal("nil runner must fail before traversal")
	}
	if _, err := EvaluatePrerequisites(m, tree, func(CommandRequest) (CommandResult, error) {
		return CommandResult{}, fmt.Errorf("runner exploded")
	}); err == nil || !strings.Contains(err.Error(), "runner exploded") {
		t.Fatalf("runner error = %v", err)
	}
	reordered := cloneMap(m)
	reordered.Prerequisites[0], reordered.Prerequisites[1] = reordered.Prerequisites[1], reordered.Prerequisites[0]
	if _, err := EvaluatePrerequisites(reordered, tree, FreshZeroPrerequisiteRunner()); err == nil {
		t.Fatal("reordered prerequisites must fail")
	}
	if err := writeCLI(&failingWriter{}, "x"); err == nil {
		t.Fatal("writeCLI must surface writer failure")
	}
	if writeCLIError(&failingWriter{}, fmt.Errorf("boom")) != 1 {
		t.Fatal("writeCLIError must return 1 when stderr write fails")
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, fmt.Errorf("write failed")
}

func prerequisiteRunnerFor(mutation DependencyVerdictMutation) CommandRunner {
	return func(req CommandRequest) (CommandResult, error) {
		result := CommandResult{ExitCode: 0, Fresh: true}
		if req.ID != mutation.OverrideID {
			return result, nil
		}
		if mutation.Skipped {
			result.Skipped = true
		}
		if mutation.Stale {
			result.Fresh = false
		}
		if mutation.Synthetic {
			result.Synthetic = true
		}
		if mutation.ExitCode != 0 {
			result.ExitCode = mutation.ExitCode
		}
		if mutation.Engine != "" {
			result.Engine = mutation.Engine
		}
		return result, nil
	}
}
