package gate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	DefaultBaselineTTL = 15 * time.Minute
	BaselineSchemaV1   = "baseline/v1"
)

type BaselineArtifact struct {
	SchemaVersion   string                    `json:"schema_version"`
	GeneratedAt     string                    `json:"generated_at,omitempty"`
	GitSHA          string                    `json:"git_sha,omitempty"`
	BackstopVersion string                    `json:"backstop_version,omitempty"`
	StepCounts      map[string]int            `json:"steps,omitempty"`
	StepRuleCounts  map[string]map[string]int `json:"step_rule_counts,omitempty"`
	Violations      []Violation               `json:"violations"`
}

func NewBaselineArtifactFromSteps(steps []StepResult, generatedAt, gitSHA, backstopVersion string) BaselineArtifact {
	artifact := BaselineArtifact{
		SchemaVersion:   BaselineSchemaV1,
		GeneratedAt:     strings.TrimSpace(generatedAt),
		GitSHA:          strings.TrimSpace(gitSHA),
		BackstopVersion: strings.TrimSpace(backstopVersion),
		StepCounts:      map[string]int{},
		StepRuleCounts:  map[string]map[string]int{},
		Violations:      []Violation{},
	}
	for _, step := range steps {
		if step.StepName == StepBaselineComparison || step.StepName == StepWaiverResolution || step.StepName == StepLedgerIntegrity {
			continue
		}
		artifact.StepCounts[step.StepName] = len(step.Violations)
		ruleCounts := map[string]int{}
		for _, violation := range step.Violations {
			enriched := EnrichViolationIdentity(violation)
			artifact.Violations = append(artifact.Violations, enriched)
			ruleCounts[enriched.Rule]++
		}
		artifact.StepRuleCounts[step.StepName] = ruleCounts
	}
	return artifact
}

type BaselineComparison struct {
	NewViolations    []Violation
	FixedViolations  []Violation
	SeededViolations []Violation
}

type BaselineCompareOptions struct {
	Scope                     *GateScope
	AllowRuleSetChangeSeeding bool
	ChangedFiles              map[string]struct{}
}

func LoadBaseline(path string) (*BaselineArtifact, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read baseline %s: %w", path, err)
	}
	var artifact BaselineArtifact
	if err := json.Unmarshal(data, &artifact); err != nil {
		return nil, fmt.Errorf("parse baseline %s: %w", path, err)
	}
	if artifact.SchemaVersion == "" {
		artifact.SchemaVersion = BaselineSchemaV1
	}
	for i := range artifact.Violations {
		artifact.Violations[i] = EnrichViolationIdentity(artifact.Violations[i])
	}
	if artifact.Violations == nil {
		artifact.Violations = []Violation{}
	}
	return &artifact, nil
}

func WriteBaseline(path string, artifact *BaselineArtifact) error {
	if artifact == nil {
		return fmt.Errorf("baseline artifact is nil")
	}
	copyArtifact := *artifact
	if copyArtifact.SchemaVersion == "" {
		copyArtifact.SchemaVersion = BaselineSchemaV1
	}
	if copyArtifact.Violations == nil {
		copyArtifact.Violations = []Violation{}
	}
	for i := range copyArtifact.Violations {
		copyArtifact.Violations[i] = EnrichViolationIdentity(copyArtifact.Violations[i])
	}
	data, err := json.MarshalIndent(copyArtifact, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal baseline: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create baseline dir %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write baseline %s: %w", path, err)
	}
	return nil
}

func CompareBaseline(current []Violation, baseline *BaselineArtifact, options BaselineCompareOptions) BaselineComparison {
	if baseline == nil {
		return BaselineComparison{NewViolations: []Violation{}, FixedViolations: []Violation{}, SeededViolations: []Violation{}}
	}
	currentInScope := current
	if options.Scope != nil && options.Scope.Mode != GateScopeModeAll {
		currentInScope = filterViolations(options.Scope, current)
	}
	baselineSet := map[string]Violation{}
	for _, v := range baseline.Violations {
		enriched := EnrichViolationIdentity(v)
		baselineSet[enriched.IdentityHash] = enriched
	}
	currentSet := map[string]Violation{}
	newViolations := []Violation{}
	seededViolations := []Violation{}
	for _, v := range currentInScope {
		enriched := EnrichViolationIdentity(v)
		currentSet[enriched.IdentityHash] = enriched
		if _, ok := baselineSet[enriched.IdentityHash]; ok {
			// Strict file-level ratchet (ISSUE-050): baseline grandfathers a finding
			// ONLY while nobody touches its file. The moment the file enters an explicit
			// diff/file scope, its grandfather is REVOKED and the finding falls through
			// to newViolations — every pre-existing finding in a touched file must be
			// resolved (fixed or waived), not merely kept net-new-clean. Untouched files
			// (per-finding scope.Contains == false), --all mode, and nil scope keep
			// grandfathering unchanged (scopeTouches). Pass the RAW finding File:
			// scope.Contains self-normalizes both sides against the real ProjectRoot,
			// whereas enriched.File was normalized with projectRoot=="".
			if !scopeTouches(options.Scope, v.File) {
				continue
			}
			// touched: grandfather revoked → fall through to newViolations.
		}
		if options.AllowRuleSetChangeSeeding && options.Scope != nil && options.Scope.Mode == GateScopeModeAll && isExistingCodeViolation(enriched, options.ChangedFiles) {
			seededViolations = append(seededViolations, enriched)
			continue
		}
		newViolations = append(newViolations, enriched)
	}
	fixedViolations := []Violation{}
	for key, old := range baselineSet {
		if _, ok := currentSet[key]; ok {
			continue
		}
		if options.Scope != nil && options.Scope.Mode != GateScopeModeAll && (old.File == "" || !options.Scope.Contains(old.File)) {
			continue
		}
		fixedViolations = append(fixedViolations, old)
	}
	if newViolations == nil {
		newViolations = []Violation{}
	}
	if fixedViolations == nil {
		fixedViolations = []Violation{}
	}
	if seededViolations == nil {
		seededViolations = []Violation{}
	}
	return BaselineComparison{NewViolations: newViolations, FixedViolations: fixedViolations, SeededViolations: seededViolations}
}

// scopeTouches reports whether file was explicitly touched by an author under a
// diff/file-scoped gate run — the single definition of "touched" for the strict
// file-level ratchet (ISSUE-050). It is the guard that keeps revocation honest:
// nil scope → false (grandfather kept); GateScopeModeAll excluded (scope.Contains
// is universally true in --all, so revoking there would blanket-revoke the whole
// repo and break gradual baseline generation); and revocation is keyed per-finding
// on scope.Contains(file), so a project-wide dimension's findings on UNtouched
// files stay grandfathered.
func scopeTouches(scope *GateScope, file string) bool {
	return scope != nil &&
		(scope.Mode == GateScopeModeDiff || scope.Mode == GateScopeModeFile) &&
		scope.Contains(file)
}

func isExistingCodeViolation(v Violation, changedFiles map[string]struct{}) bool {
	if len(changedFiles) == 0 {
		return true
	}
	file := strings.TrimSpace(v.File)
	if file == "" {
		return false
	}
	_, changed := changedFiles[file]
	return !changed
}

func EnrichViolationIdentity(v Violation) Violation {
	// Canonicalize File BEFORE it enters identity so a finding has ONE identity
	// regardless of the textual path form the SARIF engine emitted by invocation
	// scope (full-scope directory-walk "./pkg/x.go" vs diff-scope explicit
	// "pkg/x.go"). projectRoot=="" applies only the idempotent
	// Clean+ToSlash+strip-"./" subset of the SINGLE NormalizePath helper (never a
	// second implementation). This chokepoint runs on baseline write/load and every
	// current violation in CompareBaseline, so all dimensions get one canonical
	// identity (ISSUE-046). Write it back to File too, so the raw-path consumers
	// (isExistingCodeViolation, CompareBaseline's scope.Contains(old.File)) key on
	// the same canonical form.
	v.File = NormalizePath("", v.File)
	identity := strings.TrimSpace(v.Rule) + "|" + strings.TrimSpace(v.File)
	region := strings.TrimSpace(v.RegionHash)
	if region == "" {
		region = hashString(strings.TrimSpace(v.Message) + "|" + strings.TrimSpace(v.Severity) + "|" + strings.TrimSpace(v.SourcePack))
	}
	v.Identity = identity + "|" + region
	v.RegionHash = region
	v.IdentityHash = hashString(v.Identity)
	return v
}

func hashString(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}
