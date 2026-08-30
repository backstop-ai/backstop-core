package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type adoptionExecutionMutationsFile struct {
	InvalidAdoptions []AdoptionExecutionMutation `yaml:"invalid_adoptions"`
}

func loadAdoptionExecutionMutations(t *testing.T) adoptionExecutionMutationsFile {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "adoption-execution-mutations.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var document adoptionExecutionMutationsFile
	if err := yaml.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	if len(document.InvalidAdoptions) < 12 {
		t.Fatal("adoption-execution-mutations.yml must cover missing, changed, shell, mocked, copied, and preinstalled cases")
	}
	return document
}

func mustOwnerAdoption(t *testing.T) []OwnerAdoptionInstruction {
	t.Helper()
	instructions, err := LoadOwnerAdoptionInstructions(websiteJourneyRepoRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(instructionIDs(instructions), ",") != strings.Join(ExpectedAdoptionInstructionIDs(), ",") {
		t.Fatalf("owner adoption IDs = %v", instructionIDs(instructions))
	}
	return instructions
}

func mustAdoptionBuiltTree(t *testing.T, instructions []OwnerAdoptionInstruction) (string, WebsiteCapabilityMap) {
	t.Helper()
	built, m := mustAcceptedBuiltTree(t)
	if err := WriteRenderedAdoptionBlocks(built, instructions); err != nil {
		t.Fatal(err)
	}
	return built, m
}

func TestWebsiteJourney_AdoptionCommandsProduceWorkingGate(t *testing.T) {
	instructions := mustOwnerAdoption(t)
	built, m := mustAdoptionBuiltTree(t, instructions)
	if err := TraverseBuiltJourneys(built, m); err != nil {
		t.Fatal(err)
	}
	disposable := t.TempDir()
	if err := CreateDisposableAdoptionRepo(disposable); err != nil {
		t.Fatal(err)
	}
	receipt, err := ExecuteAdoption(AdoptionRequest{
		BuiltRoot:        built,
		DisposableRoot:   disposable,
		BuiltIdentity:    "built:accepted",
		DeployedIdentity: "deployed:none",
		Instructions:     instructions,
	}, DirectAdoptionRunner())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(receipt.InstructionIDs, ",") != "ADOPT-INSTALL,ADOPT-CONFIGURE,ADOPT-ENFORCE" {
		t.Fatalf("receipt IDs = %v", receipt.InstructionIDs)
	}
	if len(receipt.Digests) != 3 || len(receipt.ExitCodes) != 3 || receipt.ExitCodes[2] != 0 {
		t.Fatalf("receipt digests/exits = %v %v", receipt.Digests, receipt.ExitCodes)
	}
	if receipt.BuiltIdentity != "built:accepted" || receipt.DeployedIdentity != "deployed:none" {
		t.Fatalf("receipt identity = %s %s", receipt.BuiltIdentity, receipt.DeployedIdentity)
	}
	if _, err := os.Stat(receipt.BinaryPath); err != nil {
		t.Fatalf("installed binary: %v", err)
	}
	if _, err := os.Stat(receipt.ConfigPath); err != nil {
		t.Fatalf("configuration: %v", err)
	}
	if receipt.BinaryPath != filepath.Join(disposable, ".backstop-bin", "backstop") {
		t.Fatalf("binary path = %s", receipt.BinaryPath)
	}
}

func TestWebsiteJourney_RejectsInvalidAdoptionExecutionMatrix(t *testing.T) {
	document := loadAdoptionExecutionMutations(t)
	instructions := mustOwnerAdoption(t)
	built, _ := mustAdoptionBuiltTree(t, instructions)
	for _, mutation := range document.InvalidAdoptions {
		t.Run(mutation.Name, func(t *testing.T) {
			mutated, opts := ApplyAdoptionMutation(instructions, mutation)
			disposable := t.TempDir()
			if err := CreateDisposableAdoptionRepo(disposable); err != nil {
				t.Fatal(err)
			}
			if mutation.Preinstalled {
				if err := os.MkdirAll(filepath.Join(disposable, ".backstop-bin"), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(disposable, ".backstop-bin", "backstop"), []byte("preinstalled"), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			_, err := ExecuteAdoption(AdoptionRequest{
				BuiltRoot:        built,
				DisposableRoot:   disposable,
				BuiltIdentity:    "built:accepted",
				DeployedIdentity: "deployed:none",
				Instructions:     mutated,
				Options:          opts,
			}, DirectAdoptionRunner())
			if err == nil {
				t.Fatal("accepted invalid adoption execution")
			}
			if !strings.Contains(err.Error(), "CAP-009") || !strings.Contains(err.Error(), mutation.ExpectedError) {
				t.Fatalf("error %q does not name CAP-009 and %q", err, mutation.ExpectedError)
			}
		})
	}
}

func instructionIDs(instructions []OwnerAdoptionInstruction) []string {
	ids := make([]string, 0, len(instructions))
	for _, instruction := range instructions {
		ids = append(ids, instruction.ID)
	}
	return ids
}
