package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	hopPattern   = regexp.MustCompile(`/[A-Za-z0-9_/-]*#[A-Za-z0-9_-]+`)
	jlinkPattern = regexp.MustCompile(`JLINK-[0-9]{3}`)
	boundPattern = regexp.MustCompile(`BOUNDARY-[0-9]{3}`)
	ujTagPattern = regexp.MustCompile(`^@UJ-[0-9]{3}$`)
)

type QualityGate struct {
	Type        string
	Command     string
	MustPass    bool
	Description string
}

type Journey struct {
	GlobalKey            string
	CapabilityID         string
	Tag                  string
	Title                string
	JLinks               []string
	Hops                 []string
	HasGiven             bool
	HasWhen              bool
	HasThen              bool
	ExecutableTestTitle  string
	UsesDirectLoad       bool
	UsesGlobalNavigation bool
	Body                 string
	Boundaries           []string
}

type CapabilityArtifact struct {
	ID                  string
	Slug                string
	Title               string
	Status              string
	Strictness          string
	InfrastructureSpecs []string
	IntegrationSpec     string
	FeatureFile         string
	IDPattern           string
	AppOrigins          []string
	AuthStrategy        string
	SetupCommand        string
	TeardownCommand     string
	Gates               []QualityGate
	FeatureTitle        string
	Scenarios           []Journey
}

type CapabilityTree struct {
	Artifacts []CapabilityArtifact
}

func (tree CapabilityTree) Journeys() []Journey {
	var journeys []Journey
	for _, artifact := range tree.Artifacts {
		journeys = append(journeys, artifact.Scenarios...)
	}
	return journeys
}

type CapabilityMutation struct {
	Name                 string `yaml:"name"`
	OmitID               string `yaml:"omit_id"`
	ExtraID              string `yaml:"extra_id"`
	ExtraSlug            string `yaml:"extra_slug"`
	ExtraTitle           string `yaml:"extra_title"`
	OverrideID           string `yaml:"override_id"`
	OverrideTitle        string `yaml:"override_title"`
	OverrideStrictness   string `yaml:"override_strictness"`
	OverrideStatus       string `yaml:"override_status"`
	DropLastInfra        bool   `yaml:"drop_last_infra"`
	ExtraGate            string `yaml:"extra_gate"`
	DropDeployedGate     bool   `yaml:"drop_deployed_gate"`
	OverrideOrigin       string `yaml:"override_origin"`
	OverrideAuth         string `yaml:"override_auth"`
	OverrideIDPattern    string `yaml:"override_id_pattern"`
	OverrideFeatureFile  string `yaml:"override_feature_file"`
	ClearIntegrationSpec bool   `yaml:"clear_integration_spec"`
	OverrideBuiltGate    string `yaml:"override_built_gate"`
	ExpectedError        string `yaml:"expected_error"`
}

type ScenarioMutation struct {
	Name                   string `yaml:"name"`
	Capability             string `yaml:"capability"`
	OmitTag                string `yaml:"omit_tag"`
	ExtraTag               string `yaml:"extra_tag"`
	ExtraTitle             string `yaml:"extra_title"`
	OverrideFeatureTitle   string `yaml:"override_feature_title"`
	StripThenFrom          string `yaml:"strip_then_from"`
	LocalOnlyID            bool   `yaml:"local_only_id"`
	Tag                    string `yaml:"tag"`
	Inject                 string `yaml:"inject"`
	ReverseJLinks          bool   `yaml:"reverse_jlinks"`
	DropExecutableCoverage bool   `yaml:"drop_executable_coverage"`
	ReplaceIn              string `yaml:"replace_in"`
	Find                   string `yaml:"find"`
	Replace                string `yaml:"replace"`
	ExpectedError          string `yaml:"expected_error"`
}

type capabilityFile struct {
	Capability struct {
		ID         string `yaml:"id"`
		Title      string `yaml:"title"`
		Status     string `yaml:"status"`
		Strictness string `yaml:"strictness"`
	} `yaml:"capability"`
	InfrastructureSpecs []string `yaml:"infrastructure_specs"`
	IntegrationSpec     string   `yaml:"integration_spec"`
	Scenarios           struct {
		FeatureFile string `yaml:"feature_file"`
		IDPattern   string `yaml:"id_pattern"`
	} `yaml:"scenarios"`
	TestConfiguration struct {
		AppOrigins      []string `yaml:"app_origins"`
		AuthStrategy    string   `yaml:"auth_strategy"`
		SetupCommand    string   `yaml:"setup_command"`
		TeardownCommand string   `yaml:"teardown_command"`
	} `yaml:"test_configuration"`
	QualityGates []struct {
		Type        string `yaml:"type"`
		Command     string `yaml:"command"`
		MustPass    bool   `yaml:"must_pass"`
		Description string `yaml:"description"`
	} `yaml:"quality_gates"`
}

func ExpectedWebsiteCapabilities() []CapabilityArtifact {
	return []CapabilityArtifact{
		{ID: "CAP-004", Slug: "understand-backstop", Title: "Understand Backstop"},
		{ID: "CAP-005", Slug: "evaluate-fit", Title: "Evaluate Fit"},
		{ID: "CAP-006", Slug: "evaluate-guarantees", Title: "Evaluate Guarantees"},
		{ID: "CAP-007", Slug: "evaluate-compatibility", Title: "Evaluate Compatibility"},
		{ID: "CAP-008", Slug: "understand-system", Title: "Understand the System"},
		{ID: "CAP-009", Slug: "adopt-backstop", Title: "Adopt Backstop"},
		{ID: "CAP-010", Slug: "apply-backstop", Title: "Apply Backstop"},
		{ID: "CAP-011", Slug: "browse-pack-ecosystem", Title: "Browse the Pack Ecosystem"},
		{ID: "CAP-012", Slug: "extend-backstop", Title: "Extend Backstop"},
		{ID: "CAP-013", Slug: "inspect-evidence", Title: "Inspect the Evidence"},
		{ID: "CAP-014", Slug: "continue-beyond-backstop", Title: "Continue Beyond Backstop"},
	}
}

func ExpectedWebsiteJourneys() []Journey {
	return []Journey{
		{GlobalKey: "CAP-004/@UJ-001", CapabilityID: "CAP-004", Tag: "@UJ-001", Title: "Recognize the failure class and why Backstop exists", JLinks: []string{"JLINK-001"}, Hops: []string{"/#define-work", "/evaluate/#failure-fit"}},
		{GlobalKey: "CAP-004/@UJ-002", CapabilityID: "CAP-004", Tag: "@UJ-002", Title: "Distinguish what Backstop is from what it is not", JLinks: []string{"JLINK-002"}, Hops: []string{"/evaluate/#working-state", "/model/#operating-model"}},
		{GlobalKey: "CAP-005/@UJ-001", CapabilityID: "CAP-005", Tag: "@UJ-001", Title: "Confirm fit and continue to adoption", JLinks: []string{"JLINK-003", "JLINK-004"}, Hops: []string{"/use-cases/#choose-use-case", "/evaluate/#fit-decision", "/adopt/#install"}},
		{GlobalKey: "CAP-005/@UJ-002", CapabilityID: "CAP-005", Tag: "@UJ-002", Title: "Confirm no-fit and continue to boundary guidance", JLinks: []string{"JLINK-005"}, Hops: []string{"/model/#product-category", "/status/#adjacent-guidance"}},
		{GlobalKey: "CAP-006/@UJ-001", CapabilityID: "CAP-006", Tag: "@UJ-001", Title: "Distinguish a shipped mechanism from a guarantee", JLinks: []string{"JLINK-006"}, Hops: []string{"/model/#gates-and-policy", "/status/#supported-and-limited"}},
		{GlobalKey: "CAP-006/@UJ-002", CapabilityID: "CAP-006", Tag: "@UJ-002", Title: "Compare every public boundary state and its implication", JLinks: []string{"JLINK-007"}, Hops: []string{"/status/#boundary-states", "/model/#ownership-boundaries"}},
		{GlobalKey: "CAP-007/@UJ-001", CapabilityID: "CAP-007", Tag: "@UJ-001", Title: "Determine whether a named harness, model, or toolchain can operate Backstop", JLinks: []string{"JLINK-008"}, Hops: []string{"/model/#harness-integration", "/reference/#compatibility"}},
		{GlobalKey: "CAP-007/@UJ-002", CapabilityID: "CAP-007", Tag: "@UJ-002", Title: "Determine which lifecycle guarantees that compatibility does not preserve", JLinks: []string{"JLINK-009"}, Hops: []string{"/reference/#compatibility", "/status/#adjacent-guidance"}},
		{GlobalKey: "CAP-008/@UJ-001", CapabilityID: "CAP-008", Tag: "@UJ-001", Title: "Follow the artifact-to-plan-to-gate operating model", JLinks: []string{"JLINK-010"}, Hops: []string{"/model/#operating-model", "/reference/#artifact-schema-catalog"}},
		{GlobalKey: "CAP-008/@UJ-002", CapabilityID: "CAP-008", Tag: "@UJ-002", Title: "Inspect architecture and ownership boundaries", JLinks: []string{"JLINK-011"}, Hops: []string{"/model/#ownership-boundaries", "/status/#project-boundaries"}},
		{GlobalKey: "CAP-009/@UJ-002", CapabilityID: "CAP-009", Tag: "@UJ-002", Title: "Verify the configured repository's enforcement path", JLinks: []string{"JLINK-013", "JLINK-014"}, Hops: []string{"/adopt/#verify-enforcement", "/model/#enforcement-loop", "/reference/#gate"}},
		{GlobalKey: "CAP-010/@UJ-001", CapabilityID: "CAP-010", Tag: "@UJ-001", Title: "Select a concrete use case and its adoption action", JLinks: []string{"JLINK-015"}, Hops: []string{"/use-cases/#choose-use-case", "/adopt/#adoption-paths"}},
		{GlobalKey: "CAP-010/@UJ-002", CapabilityID: "CAP-010", Tag: "@UJ-002", Title: "Connect a use case to an applicable pack", JLinks: []string{"JLINK-016"}, Hops: []string{"/use-cases/#pack-backed-use-cases", "/packs/#choose-a-pack"}},
		{GlobalKey: "CAP-011/@UJ-001", CapabilityID: "CAP-011", Tag: "@UJ-001", Title: "Browse the generated installed-pack catalog", JLinks: []string{"JLINK-017"}, Hops: []string{"/packs/#installed-pack-catalog", "/reference/#pack-commands"}},
		{GlobalKey: "CAP-011/@UJ-002", CapabilityID: "CAP-011", Tag: "@UJ-002", Title: "Determine which pack addresses a problem and inspect its status", JLinks: []string{"JLINK-018"}, Hops: []string{"/packs/#choose-a-pack", "/status/#pack-direction"}},
		{GlobalKey: "CAP-012/@UJ-001", CapabilityID: "CAP-012", Tag: "@UJ-001", Title: "Decide whether a concern belongs in a pack and start authoring", JLinks: []string{"JLINK-019"}, Hops: []string{"/extend/#pack-or-not", "/reference/#pack-artifact"}},
		{GlobalKey: "CAP-012/@UJ-002", CapabilityID: "CAP-012", Tag: "@UJ-002", Title: "Continue from pack authoring to the contribution path", JLinks: []string{"JLINK-020"}, Hops: []string{"/extend/#author-a-pack", "/contributing/#contribution-paths"}},
		{GlobalKey: "CAP-013/@UJ-001", CapabilityID: "CAP-013", Tag: "@UJ-001", Title: "Trace an evaluation claim to its durable source", JLinks: []string{"JLINK-021"}, Hops: []string{"/model/#provenance-and-verification", "/reference/#source-traceability"}},
		{GlobalKey: "CAP-013/@UJ-002", CapabilityID: "CAP-013", Tag: "@UJ-002", Title: "Trace all generated product truth to authoritative sources", JLinks: []string{"JLINK-022", "JLINK-023"}, Hops: []string{"/packs/#installed-pack-catalog", "/reference/#cli-command-catalog", "/status/#release-history"}},
		{GlobalKey: "CAP-014/@UJ-001", CapabilityID: "CAP-014", Tag: "@UJ-001", Title: "Follow adjacent guidance beyond an intentional boundary", JLinks: []string{"JLINK-024"}, Hops: []string{"/status/#adjacent-guidance", "/contributing/#external-ownership"}},
		{GlobalKey: "CAP-014/@UJ-002", CapabilityID: "CAP-014", Tag: "@UJ-002", Title: "Confirm that adjacent guidance is not a Backstop guarantee", JLinks: []string{"JLINK-009"}, Hops: []string{"/reference/#compatibility", "/status/#adjacent-guidance"}},
	}
}

func LoadCapabilityTree(root string) (CapabilityTree, error) {
	var tree CapabilityTree
	for _, expected := range ExpectedWebsiteCapabilities() {
		dir := filepath.Join(root, "capabilities", expected.ID+"-"+expected.Slug)
		raw, err := os.ReadFile(filepath.Join(dir, "capability.yml"))
		if err != nil {
			return CapabilityTree{}, fmt.Errorf("%s: read capability.yml: %w", expected.ID, err)
		}
		var file capabilityFile
		if err := yaml.Unmarshal(raw, &file); err != nil {
			return CapabilityTree{}, fmt.Errorf("%s: parse capability.yml: %w", expected.ID, err)
		}
		featureName := file.Scenarios.FeatureFile
		if featureName == "" {
			featureName = "user-journeys.feature"
		}
		featureBytes, err := os.ReadFile(filepath.Join(dir, featureName))
		if err != nil {
			featureBytes, err = os.ReadFile(filepath.Join(dir, "user-journeys.feature"))
			if err != nil {
				return CapabilityTree{}, fmt.Errorf("%s: read user-journeys.feature: %w", expected.ID, err)
			}
		}
		featureTitle, scenarios := parseFeature(expected.ID, string(featureBytes))
		artifact := CapabilityArtifact{
			ID:                  file.Capability.ID,
			Slug:                expected.Slug,
			Title:               file.Capability.Title,
			Status:              file.Capability.Status,
			Strictness:          file.Capability.Strictness,
			InfrastructureSpecs: append([]string(nil), file.InfrastructureSpecs...),
			IntegrationSpec:     file.IntegrationSpec,
			FeatureFile:         file.Scenarios.FeatureFile,
			IDPattern:           file.Scenarios.IDPattern,
			AppOrigins:          append([]string(nil), file.TestConfiguration.AppOrigins...),
			AuthStrategy:        file.TestConfiguration.AuthStrategy,
			SetupCommand:        file.TestConfiguration.SetupCommand,
			TeardownCommand:     file.TestConfiguration.TeardownCommand,
			FeatureTitle:        featureTitle,
			Scenarios:           scenarios,
		}
		if artifact.Slug == "" {
			artifact.Slug = expected.Slug
		}
		for _, gate := range file.QualityGates {
			artifact.Gates = append(artifact.Gates, QualityGate{
				Type:        gate.Type,
				Command:     gate.Command,
				MustPass:    gate.MustPass,
				Description: gate.Description,
			})
		}
		tree.Artifacts = append(tree.Artifacts, artifact)
	}
	return tree, nil
}

func parseFeature(capabilityID, source string) (string, []Journey) {
	lines := strings.Split(source, "\n")
	featureTitle := ""
	var scenarios []Journey
	var current *Journey
	pendingTag := ""
	flush := func() {
		if current == nil {
			return
		}
		current.JLinks = jlinkPattern.FindAllString(current.Body, -1)
		current.Hops = hopPattern.FindAllString(current.Body, -1)
		current.Boundaries = boundPattern.FindAllString(current.Body, -1)
		current.UsesDirectLoad, current.UsesGlobalNavigation = detectNavigationCheats(current.Body)
		scenarios = append(scenarios, *current)
		current = nil
	}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "Feature:"):
			featureTitle = strings.TrimSpace(strings.TrimPrefix(trimmed, "Feature:"))
		case strings.HasPrefix(trimmed, "@"):
			flush()
			pendingTag = strings.Fields(trimmed)[0]
		case strings.HasPrefix(trimmed, "Scenario:"):
			if current != nil && pendingTag == "" {
				flush()
			}
			title := strings.TrimSpace(strings.TrimPrefix(trimmed, "Scenario:"))
			tag := pendingTag
			pendingTag = ""
			if tag == "" {
				tag = "@UJ-000"
			}
			globalKey := capabilityID + "/" + tag
			scenarioTitle := title
			if strings.HasPrefix(title, globalKey+" ") {
				scenarioTitle = strings.TrimPrefix(title, globalKey+" ")
			}
			current = &Journey{
				GlobalKey:           globalKey,
				CapabilityID:        capabilityID,
				Tag:                 tag,
				Title:               scenarioTitle,
				ExecutableTestTitle: title,
				Body:                title + "\n",
			}
		case current != nil && strings.HasPrefix(trimmed, "Given "):
			current.HasGiven = true
			current.Body += trimmed + "\n"
		case current != nil && strings.HasPrefix(trimmed, "When "):
			current.HasWhen = true
			current.Body += trimmed + "\n"
		case current != nil && strings.HasPrefix(trimmed, "Then "):
			current.HasThen = true
			current.Body += trimmed + "\n"
		case current != nil && strings.HasPrefix(trimmed, "And "):
			current.Body += trimmed + "\n"
		case current != nil && trimmed != "":
			current.Body += trimmed + "\n"
		}
	}
	flush()
	return featureTitle, scenarios
}

func detectNavigationCheats(body string) (directLoad bool, globalNav bool) {
	lower := strings.ToLower(body)
	directLoad = strings.Contains(lower, "navigate directly") || strings.Contains(lower, "direct load") || strings.Contains(lower, "direct-load")
	globalNav = strings.Contains(lower, "header navigation") || strings.Contains(lower, "site header") || strings.Contains(lower, "footer navigation") || strings.Contains(lower, "global nav")
	return directLoad, globalNav
}

func ValidateCapabilityArtifactMatrix(tree CapabilityTree) error {
	expected := ExpectedWebsiteCapabilities()
	seen := map[string]int{}
	for _, artifact := range tree.Artifacts {
		seen[artifact.ID]++
		if artifact.ID == "CAP-002" || artifact.ID == "CAP-003" {
			return fmt.Errorf("%s: reserved tombstone must not be reused", artifact.ID)
		}
	}
	if len(tree.Artifacts) != len(expected) {
		for _, want := range expected {
			if seen[want.ID] == 0 {
				return fmt.Errorf("%s: missing website capability", want.ID)
			}
		}
		for _, artifact := range tree.Artifacts {
			if !websiteCapabilityID(artifact.ID) {
				return fmt.Errorf("%s: unexpected website capability", artifact.ID)
			}
		}
		return fmt.Errorf("capability cardinality: got %d, want %d", len(tree.Artifacts), len(expected))
	}
	if _, err := websiteCapabilityCohortStatus(tree); err != nil {
		return err
	}
	for index, want := range expected {
		got := tree.Artifacts[index]
		if err := validateOneCapability(got, want); err != nil {
			return err
		}
	}
	return nil
}

func websiteCapabilityID(id string) bool {
	for _, expected := range ExpectedWebsiteCapabilities() {
		if expected.ID == id {
			return true
		}
	}
	return false
}

func validateOneCapability(got, want CapabilityArtifact) error {
	if got.ID != want.ID {
		return fmt.Errorf("%s: unexpected id at matrix position for %s", got.ID, want.ID)
	}
	if got.Slug != want.Slug {
		return fmt.Errorf("%s: slug %q, want %q", got.ID, got.Slug, want.Slug)
	}
	if got.Title != want.Title {
		return fmt.Errorf("%s: title %q, want %q", got.ID, got.Title, want.Title)
	}
	if got.Status != "draft" && got.Status != "verified" {
		return fmt.Errorf("%s: status %q, want draft until blocking gates pass", got.ID, got.Status)
	}
	if got.Strictness != "strict" {
		return fmt.Errorf("%s: strictness %q, want strict", got.ID, got.Strictness)
	}
	if strings.Join(got.InfrastructureSpecs, ",") != "SPEC-072,SPEC-073,SPEC-074,SPEC-075" {
		return fmt.Errorf("%s: infrastructure_specs must be SPEC-072, SPEC-073, SPEC-074, SPEC-075", got.ID)
	}
	if got.IntegrationSpec != "SPEC-076" {
		return fmt.Errorf("%s: integration_spec must be SPEC-076", got.ID)
	}
	if got.FeatureFile != "user-journeys.feature" {
		return fmt.Errorf("%s: scenarios.feature_file must be user-journeys.feature", got.ID)
	}
	if got.IDPattern != `^@UJ-[0-9]{3}$` {
		return fmt.Errorf("%s: scenarios.id_pattern must be ^@UJ-[0-9]{3}$", got.ID)
	}
	if len(got.AppOrigins) != 1 || got.AppOrigins[0] != "https://backstop.sh" {
		return fmt.Errorf("%s: app_origins must be exactly [https://backstop.sh]", got.ID)
	}
	if got.AuthStrategy != "none" {
		return fmt.Errorf("%s: auth_strategy must be none", got.ID)
	}
	if got.SetupCommand != "" || got.TeardownCommand != "" {
		return fmt.Errorf("%s: setup and teardown commands must be empty", got.ID)
	}
	built := fmt.Sprintf("./scripts/verify-website-capabilities.sh --capability %s", got.ID)
	deployed := fmt.Sprintf("./scripts/verify-website-capabilities.sh --capability %s --deployed-origin https://backstop.sh --commit \"$BACKSTOP_DEPLOY_COMMIT\" --run-id \"$BACKSTOP_DEPLOY_RUN_ID\"", got.ID)
	if len(got.Gates) != 2 {
		return fmt.Errorf("%s: must declare exactly two blocking acceptance gates", got.ID)
	}
	if got.Gates[0].Command != built || !got.Gates[0].MustPass {
		return fmt.Errorf("%s: built acceptance gate must be %q", got.ID, built)
	}
	if got.Gates[1].Command != deployed || !got.Gates[1].MustPass {
		return fmt.Errorf("%s: deployed acceptance gate must be %q", got.ID, deployed)
	}
	return nil
}

func ValidateScenarioAndCoverageMatrix(tree CapabilityTree) error {
	if err := ValidateCapabilityArtifactMatrix(tree); err != nil {
		return err
	}
	expected := ExpectedWebsiteJourneys()
	got := tree.Journeys()
	byKey := map[string]Journey{}
	for _, journey := range got {
		if !ujTagPattern.MatchString(journey.Tag) {
			return fmt.Errorf("%s: local tag %q is not a @UJ-NNN id", journey.GlobalKey, journey.Tag)
		}
		if _, exists := byKey[journey.GlobalKey]; exists {
			return fmt.Errorf("%s: duplicate global journey key", journey.GlobalKey)
		}
		byKey[journey.GlobalKey] = journey
	}
	for _, artifact := range tree.Artifacts {
		if artifact.FeatureTitle != artifact.Title {
			return fmt.Errorf("%s: feature title %q, want %q", artifact.ID, artifact.FeatureTitle, artifact.Title)
		}
	}
	if len(got) != len(expected) {
		for _, want := range expected {
			if _, ok := byKey[want.GlobalKey]; !ok {
				return fmt.Errorf("%s: missing scenario", want.GlobalKey)
			}
		}
		for key := range byKey {
			if !expectedJourneyKey(key) {
				return fmt.Errorf("%s: unexpected scenario", key)
			}
		}
		return fmt.Errorf("journey cardinality: got %d, want %d", len(got), len(expected))
	}
	for index, want := range expected {
		actual, ok := byKey[want.GlobalKey]
		if !ok {
			return fmt.Errorf("%s: missing scenario", want.GlobalKey)
		}
		if got[index].GlobalKey != want.GlobalKey {
			return fmt.Errorf("%s: out of matrix order, found %s", want.GlobalKey, got[index].GlobalKey)
		}
		if actual.Title != want.Title {
			return fmt.Errorf("%s: title %q, want %q", want.GlobalKey, actual.Title, want.Title)
		}
		if !actual.HasGiven || !actual.HasWhen || !actual.HasThen {
			return fmt.Errorf("%s: scenario must be executable Given/When/Then", want.GlobalKey)
		}
		if actual.UsesDirectLoad {
			return fmt.Errorf("%s: direct loads are prohibited", want.GlobalKey)
		}
		if actual.UsesGlobalNavigation {
			return fmt.Errorf("%s: global navigation is prohibited", want.GlobalKey)
		}
		if !strings.Contains(actual.ExecutableTestTitle, want.GlobalKey) {
			return fmt.Errorf("%s: missing executable browser coverage whose title contains the global key", want.GlobalKey)
		}
		if strings.Join(actual.JLinks, ",") != strings.Join(want.JLinks, ",") {
			return fmt.Errorf("%s: JLINK binding %v, want %v", want.GlobalKey, actual.JLinks, want.JLinks)
		}
		if strings.Join(actual.Hops, ",") != strings.Join(want.Hops, ",") {
			return fmt.Errorf("%s: hop sequence %v, want %v", want.GlobalKey, actual.Hops, want.Hops)
		}
		if want.GlobalKey == "CAP-014/@UJ-001" && !containsAll(actual.Boundaries, "BOUNDARY-005") {
			return fmt.Errorf("%s: must bind BOUNDARY-005 on the JLINK-024 continuation", want.GlobalKey)
		}
	}
	return nil
}

func expectedJourneyKey(key string) bool {
	for _, journey := range ExpectedWebsiteJourneys() {
		if journey.GlobalKey == key {
			return true
		}
	}
	return false
}

func containsAll(have []string, want string) bool {
	for _, item := range have {
		if item == want {
			return true
		}
	}
	return false
}

func cloneTree(tree CapabilityTree) CapabilityTree {
	cloned := CapabilityTree{Artifacts: make([]CapabilityArtifact, len(tree.Artifacts))}
	for i, artifact := range tree.Artifacts {
		copyArt := artifact
		copyArt.InfrastructureSpecs = append([]string(nil), artifact.InfrastructureSpecs...)
		copyArt.AppOrigins = append([]string(nil), artifact.AppOrigins...)
		copyArt.Gates = append([]QualityGate(nil), artifact.Gates...)
		copyArt.Scenarios = make([]Journey, len(artifact.Scenarios))
		for j, scenario := range artifact.Scenarios {
			copyScene := scenario
			copyScene.JLinks = append([]string(nil), scenario.JLinks...)
			copyScene.Hops = append([]string(nil), scenario.Hops...)
			copyScene.Boundaries = append([]string(nil), scenario.Boundaries...)
			copyArt.Scenarios[j] = copyScene
		}
		cloned.Artifacts[i] = copyArt
	}
	return cloned
}

func ApplyCapabilityMutation(tree CapabilityTree, mutation CapabilityMutation) CapabilityTree {
	cloned := cloneTree(tree)
	if mutation.OmitID != "" {
		filtered := cloned.Artifacts[:0]
		for _, artifact := range cloned.Artifacts {
			if artifact.ID != mutation.OmitID {
				filtered = append(filtered, artifact)
			}
		}
		cloned.Artifacts = filtered
		return cloned
	}
	if mutation.ExtraID != "" {
		cloned.Artifacts = append(cloned.Artifacts, CapabilityArtifact{
			ID:                  mutation.ExtraID,
			Slug:                mutation.ExtraSlug,
			Title:               mutation.ExtraTitle,
			Status:              "draft",
			Strictness:          "strict",
			InfrastructureSpecs: []string{"SPEC-072", "SPEC-073", "SPEC-074", "SPEC-075"},
			IntegrationSpec:     "SPEC-076",
			FeatureFile:         "user-journeys.feature",
			IDPattern:           `^@UJ-[0-9]{3}$`,
			AppOrigins:          []string{"https://backstop.sh"},
			AuthStrategy:        "none",
			Gates: []QualityGate{
				{Type: "acceptance", Command: "./scripts/verify-website-capabilities.sh --capability " + mutation.ExtraID, MustPass: true},
				{Type: "acceptance", Command: "./scripts/verify-website-capabilities.sh --capability " + mutation.ExtraID + " --deployed-origin https://backstop.sh --commit \"$BACKSTOP_DEPLOY_COMMIT\" --run-id \"$BACKSTOP_DEPLOY_RUN_ID\"", MustPass: true},
			},
		})
		return cloned
	}
	for i := range cloned.Artifacts {
		if cloned.Artifacts[i].ID != mutation.OverrideID {
			continue
		}
		if mutation.OverrideTitle != "" {
			cloned.Artifacts[i].Title = mutation.OverrideTitle
		}
		if mutation.OverrideStrictness != "" {
			cloned.Artifacts[i].Strictness = mutation.OverrideStrictness
		}
		if mutation.OverrideStatus != "" {
			cloned.Artifacts[i].Status = mutation.OverrideStatus
		}
		if mutation.DropLastInfra && len(cloned.Artifacts[i].InfrastructureSpecs) > 0 {
			cloned.Artifacts[i].InfrastructureSpecs = cloned.Artifacts[i].InfrastructureSpecs[:len(cloned.Artifacts[i].InfrastructureSpecs)-1]
		}
		if mutation.ExtraGate != "" {
			cloned.Artifacts[i].Gates = append(cloned.Artifacts[i].Gates, QualityGate{Type: "acceptance", Command: mutation.ExtraGate, MustPass: true})
		}
		if mutation.DropDeployedGate && len(cloned.Artifacts[i].Gates) > 1 {
			cloned.Artifacts[i].Gates = cloned.Artifacts[i].Gates[:1]
		}
		if mutation.OverrideOrigin != "" {
			cloned.Artifacts[i].AppOrigins = []string{mutation.OverrideOrigin}
		}
		if mutation.OverrideAuth != "" {
			cloned.Artifacts[i].AuthStrategy = mutation.OverrideAuth
		}
		if mutation.OverrideIDPattern != "" {
			cloned.Artifacts[i].IDPattern = mutation.OverrideIDPattern
		}
		if mutation.OverrideFeatureFile != "" {
			cloned.Artifacts[i].FeatureFile = mutation.OverrideFeatureFile
		}
		if mutation.ClearIntegrationSpec {
			cloned.Artifacts[i].IntegrationSpec = ""
		}
		if mutation.OverrideBuiltGate != "" && len(cloned.Artifacts[i].Gates) > 0 {
			cloned.Artifacts[i].Gates[0].Command = mutation.OverrideBuiltGate
		}
	}
	return cloned
}

func ApplyScenarioMutation(tree CapabilityTree, mutation ScenarioMutation) CapabilityTree {
	cloned := cloneTree(tree)
	for i := range cloned.Artifacts {
		if cloned.Artifacts[i].ID != mutation.Capability {
			continue
		}
		if mutation.OverrideFeatureTitle != "" {
			cloned.Artifacts[i].FeatureTitle = mutation.OverrideFeatureTitle
		}
		if mutation.OmitTag != "" {
			filtered := cloned.Artifacts[i].Scenarios[:0]
			for _, scenario := range cloned.Artifacts[i].Scenarios {
				if scenario.Tag != mutation.OmitTag {
					filtered = append(filtered, scenario)
				}
			}
			cloned.Artifacts[i].Scenarios = filtered
		}
		if mutation.ExtraTag != "" {
			cloned.Artifacts[i].Scenarios = append(cloned.Artifacts[i].Scenarios, Journey{
				GlobalKey:           cloned.Artifacts[i].ID + "/" + mutation.ExtraTag,
				CapabilityID:        cloned.Artifacts[i].ID,
				Tag:                 mutation.ExtraTag,
				Title:               mutation.ExtraTitle,
				ExecutableTestTitle: cloned.Artifacts[i].ID + "/" + mutation.ExtraTag + " " + mutation.ExtraTitle,
				HasGiven:            true,
				HasWhen:             true,
				HasThen:             true,
			})
		}
		if mutation.LocalOnlyID {
			for j := range cloned.Artifacts[i].Scenarios {
				if cloned.Artifacts[i].Scenarios[j].Tag == "@UJ-001" {
					cloned.Artifacts[i].Scenarios[j].ExecutableTestTitle = "@UJ-001 " + cloned.Artifacts[i].Scenarios[j].Title
				}
			}
		}
		target := mutation.Tag
		if target == "" {
			target = mutation.StripThenFrom
		}
		if target == "" {
			target = mutation.ReplaceIn
		}
		for j := range cloned.Artifacts[i].Scenarios {
			if target != "" && cloned.Artifacts[i].Scenarios[j].Tag != target {
				continue
			}
			if mutation.StripThenFrom != "" {
				cloned.Artifacts[i].Scenarios[j].HasThen = false
			}
			if mutation.Inject != "" {
				cloned.Artifacts[i].Scenarios[j].Body += mutation.Inject + "\n"
				cloned.Artifacts[i].Scenarios[j].UsesDirectLoad, cloned.Artifacts[i].Scenarios[j].UsesGlobalNavigation = detectNavigationCheats(cloned.Artifacts[i].Scenarios[j].Body)
			}
			if mutation.ReverseJLinks {
				cloned.Artifacts[i].Scenarios[j].JLinks = reverseStrings(cloned.Artifacts[i].Scenarios[j].JLinks)
				cloned.Artifacts[i].Scenarios[j].Hops = reverseStrings(cloned.Artifacts[i].Scenarios[j].Hops)
			}
			if mutation.DropExecutableCoverage {
				cloned.Artifacts[i].Scenarios[j].ExecutableTestTitle = cloned.Artifacts[i].Scenarios[j].Title
			}
			if mutation.Find != "" {
				cloned.Artifacts[i].Scenarios[j].Body = strings.ReplaceAll(cloned.Artifacts[i].Scenarios[j].Body, mutation.Find, mutation.Replace)
				cloned.Artifacts[i].Scenarios[j].JLinks = replaceAllStrings(cloned.Artifacts[i].Scenarios[j].JLinks, mutation.Find, mutation.Replace)
				cloned.Artifacts[i].Scenarios[j].Hops = replaceAllStrings(cloned.Artifacts[i].Scenarios[j].Hops, mutation.Find, mutation.Replace)
				cloned.Artifacts[i].Scenarios[j].Boundaries = replaceAllStrings(cloned.Artifacts[i].Scenarios[j].Boundaries, mutation.Find, mutation.Replace)
			}
		}
	}
	return cloned
}

func OmitJourneyLink(tree CapabilityTree, globalKey, link string) CapabilityTree {
	return mutateJourney(tree, globalKey, func(journey *Journey) {
		filtered := journey.JLinks[:0]
		for _, item := range journey.JLinks {
			if item != link {
				filtered = append(filtered, item)
			}
		}
		journey.JLinks = filtered
		journey.Body = strings.ReplaceAll(journey.Body, link, "")
	})
}

func WrongBindJourneyLink(tree CapabilityTree, globalKey, link, replacement string) CapabilityTree {
	return mutateJourney(tree, globalKey, func(journey *Journey) {
		journey.JLinks = replaceAllStrings(journey.JLinks, link, replacement)
		journey.Body = strings.ReplaceAll(journey.Body, link, replacement)
	})
}

func mutateJourney(tree CapabilityTree, globalKey string, edit func(*Journey)) CapabilityTree {
	cloned := cloneTree(tree)
	for i := range cloned.Artifacts {
		for j := range cloned.Artifacts[i].Scenarios {
			if cloned.Artifacts[i].Scenarios[j].GlobalKey == globalKey {
				edit(&cloned.Artifacts[i].Scenarios[j])
			}
		}
	}
	return cloned
}

func reverseStrings(values []string) []string {
	out := append([]string(nil), values...)
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

func replaceAllStrings(values []string, find, replace string) []string {
	out := make([]string, len(values))
	for i, value := range values {
		out[i] = strings.ReplaceAll(value, find, replace)
	}
	return out
}

func VerifyCapabilityArtifacts(root, capability string) error {
	tree, err := LoadCapabilityTree(root)
	if err != nil {
		return err
	}
	if capability != "" && !websiteCapabilityID(capability) {
		return fmt.Errorf("%s: not a website Seed 5 capability", capability)
	}
	if err := ValidateCapabilityArtifactMatrix(tree); err != nil {
		return err
	}
	if err := ValidateScenarioAndCoverageMatrix(tree); err != nil {
		return err
	}
	evidence, err := CollectPromotionEvidence(root)
	if err != nil {
		return err
	}
	if err := ValidatePromotion(tree, evidence); err != nil {
		return err
	}
	if capability != "" {
		found := false
		for _, artifact := range tree.Artifacts {
			if artifact.ID == capability {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("%s: missing website capability", capability)
		}
	}
	return nil
}
