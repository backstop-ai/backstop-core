package compile

import (
	"strings"

	"github.com/bmanson/backstop-core/pkg/schema"
)

// Rule represents a single enforcement rule extracted from standard frontmatter.
type Rule struct {
	ID             string
	Name           string
	Category       string
	Severity       string
	Description    string
	Detection      map[string]interface{}
	ComplianceTier string
	Languages      []string
	Fix            string
}

// Strategy returns the detection strategy for this rule.
func (r Rule) Strategy() string {
	if r.Detection == nil {
		return ""
	}

	strategy, ok := r.Detection["strategy"].(string)
	if !ok {
		return ""
	}

	return strategy
}

// IsAdvisory returns true when the rule has a note but no enforceable detection fields.
func (r Rule) IsAdvisory() bool {
	if r.Detection == nil {
		return false
	}

	if _, hasNote := r.Detection["note"]; !hasNote {
		return false
	}

	enforceableFields := []string{"semgrep", "metric", "pattern", "enforced_by"}
	for _, field := range enforceableFields {
		val, exists := r.Detection[field]
		if exists && hasMeaningfulValue(val) {
			return false
		}
	}

	return true
}

func hasMeaningfulValue(v interface{}) bool {
	if v == nil {
		return false
	}
	switch val := v.(type) {
	case string:
		return strings.TrimSpace(val) != ""
	case []interface{}:
		return len(val) > 0
	case map[string]interface{}:
		return len(val) > 0
	case bool:
		return val
	case int:
		return val != 0
	case float64:
		return val != 0
	default:
		return true
	}
}

// CompileOptions holds compiler configuration.
type CompileOptions struct {
	OutputDir    string
	SchemaSource SchemaSource
}

// EffectiveOutputDir returns the configured output directory or the default.
func (o CompileOptions) EffectiveOutputDir() string {
	if o.OutputDir == "" {
		return ".backstop/rules/"
	}

	return o.OutputDir
}

// CompileResult contains compiler outputs.
type CompileResult struct {
	Manifest     *EnforcementManifest
	SemgrepRules []SemgrepRule
	NativeChecks []NativeCheck
	Warnings     []string
	OutputPaths  []string
}

// EnforcementManifest defines emitted manifest metadata.
type EnforcementManifest struct {
	Standard         string
	Language         string
	CompiledFrom     string
	SemgrepConfig    string
	NativeChecksFile string
	Rules            []ManifestRule
}

// ManifestRule describes a rule entry in the enforcement manifest.
type ManifestRule struct {
	ID             string
	Name           string
	Severity       string
	ComplianceTier string
	Enforcement    string
	DelegatedTo    *DelegatedTarget
}

// EffectiveTier returns the configured compliance tier or the default.
func (r ManifestRule) EffectiveTier() string {
	if r.ComplianceTier == "" {
		return "baseline"
	}

	return r.ComplianceTier
}

// DelegatedTarget specifies delegated enforcement details.
type DelegatedTarget struct {
	Tool string
	Rule string
}

// SemgrepRule defines a generated Semgrep rule.
type SemgrepRule struct {
	ID           string
	Message      string
	Severity     string
	Languages    []string
	Pattern      string
	PatternRegex string
}

// NativeCheck defines a generated native checker rule.
type NativeCheck struct {
	ID        string      `json:"id"`
	Message   string      `json:"message"`
	Severity  string      `json:"severity"`
	Language  string      `json:"language"`
	Metric    string      `json:"metric"`
	Operator  string      `json:"operator"`
	Threshold interface{} `json:"threshold"`
	Exclude   []string    `json:"exclude,omitempty"`
}

// SchemaSource loads schemas used by the compiler.
type SchemaSource interface {
	LoadSchema(artifactType, version string) (*schema.Schema, error)
}

// MapSeverity maps canonical severity values to Semgrep severity strings.
func MapSeverity(severity string) string {
	switch strings.ToLower(severity) {
	case "error":
		return "ERROR"
	case "warning":
		return "WARNING"
	case "info":
		return "INFO"
	default:
		return strings.ToUpper(severity)
	}
}
