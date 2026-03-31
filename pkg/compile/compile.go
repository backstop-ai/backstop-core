package compile

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmanson/backstop-core/pkg/artifact"
	"github.com/bmanson/backstop-core/pkg/schema"
	"github.com/bmanson/backstop-core/pkg/validate"
)

// Compile parses a standard artifact and emits compiler outputs.
func Compile(standardPath string, opts CompileOptions) (*CompileResult, error) {
	art, err := artifact.ParseFile(standardPath)
	if err != nil {
		return nil, fmt.Errorf("parse standard: %w", err)
	}

	sch, err := loadSchemaForCompile(art, opts)
	if err != nil {
		return nil, err
	}

	validation := validate.Standard(art, sch)
	if !validation.Pass() {
		var b strings.Builder
		b.WriteString("standard validation failed:")
		for _, v := range validation.Violations {
			b.WriteString("\n- ")
			b.WriteString(v.Rule)
			if v.Message != "" {
				b.WriteString(": ")
				b.WriteString(v.Message)
			}
		}
		return nil, errors.New(b.String())
	}

	number := art.Metadata["number"]
	status := art.Metadata["status"]
	language := frontmatterString(art.Frontmatter, "language")
	scope := frontmatterString(art.Frontmatter, "scope")
	supersededBy := frontmatterString(art.Frontmatter, "superseded_by")

	rules, err := parseRules(art)
	if err != nil {
		return nil, err
	}

	manifest := &EnforcementManifest{
		Standard:     number,
		Language:     language,
		CompiledFrom: filepath.Base(standardPath),
		Rules:        make([]ManifestRule, 0, len(rules)),
	}

	semgrepRules := make([]SemgrepRule, 0)
	nativeChecks := make([]NativeCheck, 0)
	warnings := make([]string, 0)

	for _, rule := range rules {
		if rule.IsAdvisory() {
			continue
		}

		manifestRule := ManifestRule{
			ID:             rule.ID,
			Name:           rule.Name,
			Severity:       rule.Severity,
			ComplianceTier: rule.ComplianceTier,
		}

		switch rule.Strategy() {
		case "pattern", "regex":
			langs := resolveRuleLanguages(scope, language, rule)
			if len(langs) == 0 {
				return nil, fmt.Errorf("rule %s requires languages for universal %s strategy", rule.ID, rule.Strategy())
			}
			semgrepRules = append(semgrepRules, EmitSemgrepRule(rule, langs))
			manifestRule.Enforcement = "semgrep"
		case "metric":
			if len(rule.Languages) == 0 && language != "" {
				rule.Languages = []string{language}
			}
			nativeChecks = append(nativeChecks, EmitNativeCheck(rule))
			manifestRule.Enforcement = "native"
		case "delegated":
			manifestRule.Enforcement = "delegated"
			manifestRule.DelegatedTo = &DelegatedTarget{
				Tool: detectionString(rule.Detection, "enforced_by"),
				Rule: detectionString(rule.Detection, "rule"),
			}
		default:
			return nil, fmt.Errorf("unsupported strategy %q for rule %s", rule.Strategy(), rule.ID)
		}

		manifest.Rules = append(manifest.Rules, manifestRule)
	}

	if status == "deprecated" {
		warning := fmt.Sprintf("standard %s is deprecated", number)
		if supersededBy != "" {
			warning += "; superseded by " + supersededBy
		}
		warnings = append(warnings, warning)
	}

	outDir := opts.EffectiveOutputDir()
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, fmt.Errorf("create output dir: %w", err)
	}

	outputPaths := make([]string, 0, 3)
	manifestPath := filepath.Join(outDir, fmt.Sprintf("%s.manifest.json", number))

	if len(semgrepRules) > 0 {
		manifest.SemgrepConfig = fmt.Sprintf("%s.semgrep.yml", number)
	}
	if len(nativeChecks) > 0 {
		manifest.NativeChecksFile = fmt.Sprintf("%s.native.json", number)
	}

	if err := writeManifest(manifest, manifestPath); err != nil {
		return nil, fmt.Errorf("write manifest: %w", err)
	}
	outputPaths = append(outputPaths, manifestPath)

	if len(semgrepRules) > 0 {
		semgrepPath := filepath.Join(outDir, manifest.SemgrepConfig)
		if err := WriteSemgrepFile(semgrepRules, semgrepPath); err != nil {
			return nil, fmt.Errorf("write semgrep: %w", err)
		}
		outputPaths = append(outputPaths, semgrepPath)
	}

	if len(nativeChecks) > 0 {
		nativePath := filepath.Join(outDir, manifest.NativeChecksFile)
		if err := WriteNativeChecksFile(nativeChecks, nativePath); err != nil {
			return nil, fmt.Errorf("write native checks: %w", err)
		}
		outputPaths = append(outputPaths, nativePath)
	}

	return &CompileResult{
		Manifest:     manifest,
		SemgrepRules: semgrepRules,
		NativeChecks: nativeChecks,
		Warnings:     warnings,
		OutputPaths:  outputPaths,
	}, nil
}

func loadSchemaForCompile(art *artifact.ParsedArtifact, opts CompileOptions) (*schema.Schema, error) {
	if opts.SchemaSource != nil {
		schemaVersion := art.Metadata["schema_version"]
		parts := strings.SplitN(schemaVersion, "/", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return nil, fmt.Errorf("invalid schema_version %q", schemaVersion)
		}
		sch, err := opts.SchemaSource.LoadSchema(parts[0], parts[1])
		if err != nil {
			return nil, fmt.Errorf("load schema via source: %w", err)
		}
		return sch, nil
	}

	schemaPath, err := schema.ResolveSchemaPath(art)
	if err != nil {
		return nil, fmt.Errorf("resolve schema path: %w", err)
	}
	sch, err := schema.LoadArtifactSchema(schemaPath, "artifacts")
	if err != nil {
		return nil, fmt.Errorf("load schema: %w", err)
	}
	return sch, nil
}

func parseRules(art *artifact.ParsedArtifact) ([]Rule, error) {
	rawRules, ok := art.Frontmatter["rules"]
	if !ok {
		return nil, fmt.Errorf("rules missing from frontmatter")
	}

	ruleItems, ok := rawRules.([]interface{})
	if !ok {
		return nil, fmt.Errorf("rules must be an array")
	}

	seen := make(map[string]struct{}, len(ruleItems))
	rules := make([]Rule, 0, len(ruleItems))
	for i, item := range ruleItems {
		ruleMap, ok := item.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("rules[%d] must be an object", i)
		}

		rule := Rule{
			ID:             mapString(ruleMap, "id"),
			Name:           mapString(ruleMap, "name"),
			Category:       mapString(ruleMap, "category"),
			Severity:       mapString(ruleMap, "severity"),
			Description:    mapString(ruleMap, "description"),
			ComplianceTier: mapString(ruleMap, "compliance_tier"),
			Fix:            mapString(ruleMap, "fix"),
			Detection:      mapStringMap(ruleMap, "detection"),
			Languages:      mapStrings(ruleMap, "languages"),
		}

		if len(rule.Languages) == 0 && rule.Detection != nil {
			rule.Languages = mapStrings(rule.Detection, "languages")
		}

		if _, exists := seen[rule.ID]; exists {
			return nil, fmt.Errorf("duplicate rule id %q", rule.ID)
		}
		seen[rule.ID] = struct{}{}

		rules = append(rules, rule)
	}

	return rules, nil
}

func resolveRuleLanguages(scope, language string, rule Rule) []string {
	if scope == "universal" {
		return rule.Languages
	}
	if language == "" {
		return nil
	}
	return []string{language}
}

func frontmatterString(frontmatter map[string]interface{}, key string) string {
	if frontmatter == nil {
		return ""
	}
	if val, ok := frontmatter[key].(string); ok {
		return val
	}
	return ""
}

func mapString(input map[string]interface{}, key string) string {
	if input == nil {
		return ""
	}
	if val, ok := input[key].(string); ok {
		return val
	}
	return ""
}

func mapStringMap(input map[string]interface{}, key string) map[string]interface{} {
	if input == nil {
		return nil
	}
	if val, ok := input[key].(map[string]interface{}); ok {
		return val
	}
	return nil
}

func mapStrings(input map[string]interface{}, key string) []string {
	if input == nil {
		return nil
	}
	raw, ok := input[key]
	if !ok {
		return nil
	}
	arr, ok := raw.([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func detectionString(detection map[string]interface{}, key string) string {
	if detection == nil {
		return ""
	}
	if val, ok := detection[key].(string); ok {
		return val
	}
	return ""
}

type manifestJSON struct {
	Standard         string             `json:"standard"`
	Language         string             `json:"language"`
	CompiledFrom     string             `json:"compiled_from"`
	SemgrepConfig    string             `json:"semgrep_config,omitempty"`
	NativeChecksFile string             `json:"native_checks_file,omitempty"`
	Rules            []manifestRuleJSON `json:"rules"`
}

type manifestRuleJSON struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Severity       string         `json:"severity"`
	ComplianceTier string         `json:"compliance_tier"`
	Enforcement    string         `json:"enforcement"`
	DelegatedTo    *delegatedJSON `json:"delegated_to,omitempty"`
}

type delegatedJSON struct {
	Tool string `json:"tool"`
	Rule string `json:"rule"`
}

func writeManifest(manifest *EnforcementManifest, path string) error {
	rules := make([]manifestRuleJSON, 0, len(manifest.Rules))
	for _, r := range manifest.Rules {
		var delegated *delegatedJSON
		if r.DelegatedTo != nil {
			delegated = &delegatedJSON{Tool: r.DelegatedTo.Tool, Rule: r.DelegatedTo.Rule}
		}
		rules = append(rules, manifestRuleJSON{
			ID:             r.ID,
			Name:           r.Name,
			Severity:       r.Severity,
			ComplianceTier: r.EffectiveTier(),
			Enforcement:    r.Enforcement,
			DelegatedTo:    delegated,
		})
	}

	data, err := json.MarshalIndent(manifestJSON{
		Standard:         manifest.Standard,
		Language:         manifest.Language,
		CompiledFrom:     manifest.CompiledFrom,
		SemgrepConfig:    manifest.SemgrepConfig,
		NativeChecksFile: manifest.NativeChecksFile,
		Rules:            rules,
	}, "", "  ")
	if err != nil {
		return err
	}

	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}
