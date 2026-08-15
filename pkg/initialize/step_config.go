package initialize

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/backstop-ai/backstop-core/pkg/config"
	"gopkg.in/yaml.v3"
)

// stepConfigName is the report name for step 2.
const stepConfigName = "config"

// artifactRootValue is the project-relative directory the full-SDLC profile declares
// as its artifact root, and it is the SAME directory REQ-004's layout step creates.
// One constant, so the two cannot drift into a config that points at a layout that
// does not exist.
const artifactRootValue = ".backstop"

// sdlcDimensions are the five gate dimensions the pack-only profile must turn OFF.
//
// They are named here because they are the five that HARD-ERROR on a missing `specs/`
// directory rather than skipping — a pack-only consumer has no specs/ and never will,
// so leaving them enforced would make every gate run fail for a layout that profile
// deliberately does not create. They are backstop's own dimension vocabulary, not tool
// or language names.
var sdlcDimensions = []string{ // nosemgrep: go.core.no-global-mutable-state — immutable dimension list, never mutated
	"test_verification",
	"coverage_threshold",
	"contract_signature",
	"test_substantiveness",
	"artifact_status_drift",
}

// stepConfig writes the profile-correct backstop.yml (SPEC-069 REQ-003, REQ-033).
//
// IT IS UNCONDITIONAL — config generation is not a capability. An init that does not
// write the config produces nothing a consumer can use.
//
// THE PROFILE IS DERIVED FROM THE RESOLVED CAPABILITY SET AND NEVER FROM THE PROJECT:
// `sdlc` present is the full-SDLC greenfield profile, `sdlc` subtracted is the
// pack-only profile. Reading the project to pick one would be detection.
//
// An existing backstop.yml is preserved BYTE-FOR-BYTE and reported as already
// present. The consumer's policy decisions are theirs.
func stepConfig(projectRoot string, capabilities map[Capability]bool) StepReport {
	configPath := filepath.Join(projectRoot, "backstop.yml")
	if pathExists(configPath) {
		return StepReport{
			Step:    stepConfigName,
			Outcome: OutcomeConverged,
			Detail:  "backstop.yml is already present and was left byte-for-byte unchanged",
		}
	}

	fullSdlc := capabilities[CapabilitySdlc]

	generated, err := generateConfig(projectRoot, fullSdlc)
	if err != nil {
		return StepReport{
			Step:    stepConfigName,
			Outcome: OutcomeBrokenPromise,
			Detail:  fmt.Sprintf("generating backstop.yml: %v", err),
		}
	}
	if writeErr := os.WriteFile(configPath, generated, 0o644); writeErr != nil {
		return StepReport{
			Step:    stepConfigName,
			Outcome: OutcomeBrokenPromise,
			Detail:  fmt.Sprintf("writing backstop.yml: %v", writeErr),
		}
	}

	if fullSdlc {
		return StepReport{
			Step:    stepConfigName,
			Outcome: OutcomeDelivered,
			Detail: fmt.Sprintf("wrote backstop.yml for the full-SDLC profile, declaring %s as the artifact root",
				artifactRootValue),
		}
	}

	// THE PACK-ONLY GAP, REPORTED RATHER THAN WIRED (REQ-033). Turning
	// coverage_threshold off forfeits coverage enforcement for a consumer whose packs
	// still emit coverage records. The spec-independent floor that would replace it
	// DOES NOT EXIST: the shipped schema declares `enforcement` with
	// additionalProperties:false, so a floor key written here would be rejected at
	// load — and adding it to the schema would be init designing an enforcement-policy
	// surface. So the forfeiture is stated and the run is NOT failed: an un-adopted
	// capability is a missing benefit, not a broken promise.
	return StepReport{
		Step:    stepConfigName,
		Outcome: OutcomeDelivered,
		Detail: fmt.Sprintf(
			"wrote backstop.yml for the pack-only profile, turning off the five gate dimensions that hard-error without an artifact layout (%s). "+
				"GAP: turning off coverage_threshold forfeits coverage enforcement even though your packs may still emit coverage records, and the "+
				"spec-independent coverage floor that would replace it does not exist yet — there is no configuration key to set, so nothing is being left undone here",
			joinDimensions()),
	}
}

// generateConfig marshals the profile's config through the SHIPPED config types.
//
// Marshalling config.Config rather than hand-assembling YAML text is what makes
// "every key the generated config carries is one the shipped schema declares" true BY
// CONSTRUCTION: an undeclared key is unrepresentable, because there is no field to put
// it in. That is the durable form of REQ-033's tripwire, and it is why the coverage
// floor cannot be quietly added here without also editing pkg/config and the schema.
func generateConfig(projectRoot string, fullSdlc bool) ([]byte, error) {
	name, err := projectName(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("naming the project for its generated config: %w", err)
	}

	generated := config.Config{Project: name}

	if fullSdlc {
		// The greenfield profile declares where its artifacts live and sets NO
		// dimension to off. Nothing else is written: the consumer's enforcement policy
		// is theirs to choose, and a generated default would be init making that
		// choice for them.
		generated.ArtifactRoot = artifactRootValue
	} else {
		policy := make(map[string]config.DimensionPolicy, len(sdlcDimensions))
		for _, dimension := range sdlcDimensions {
			policy[dimension] = config.DimensionPolicy{Level: "off"}
		}
		generated.Enforcement.Policy = policy
	}

	return yaml.Marshal(generated)
}

// projectName is the target directory's BASENAME and is derived from nothing else.
//
// No file in the project is read to name it. A manifest-derived name would be
// detection — the one thing init does not do — and it would make init's output depend
// on which ecosystem's files happen to be lying around.
func projectName(projectRoot string) (string, error) {
	absolute, err := filepath.Abs(projectRoot)
	if err != nil {
		return "", fmt.Errorf("resolving the project directory: %w", err)
	}
	name := filepath.Base(absolute)
	if name == "" || name == string(filepath.Separator) || name == "." {
		return "", fmt.Errorf("the project directory %q has no usable basename to name the project after", projectRoot)
	}
	return name, nil
}

// joinDimensions renders the five dimension names for a report line.
func joinDimensions() string {
	joined := ""
	for i, dimension := range sdlcDimensions {
		if i > 0 {
			joined += ", "
		}
		joined += dimension
	}
	return joined
}
