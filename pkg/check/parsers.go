package check

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Parser translates raw tool output into violations for a target CheckType.
// The named-format registry binds each format string to one of these.
type Parser func(out []byte, target CheckType) ([]Violation, error)

// formatParsers maps each named output format to its Parser. Only the neutral
// "sarif" format survives: the in-process check engine that consumed the
// eslint-json/tsc/regex-lines parsers (via the toolchain-executor registry) was
// deleted by ISSUE-018. The gate's LIVE findings path (ParsePackFindings) and
// the Go toolchain (normalized to SARIF by the go-toolchain pack convert scripts)
// both parse through "sarif".
var formatParsers map[string]Parser = map[string]Parser{ // nosemgrep: go.core.no-global-mutable-state — immutable parser registry, never reassigned
	"sarif": parseSarif,
}

// lookupParser resolves a named output format to a Parser, or returns an error
// for an unknown name. Fail-loud: a declared toolchain whose format is not
// known is a config error, never a silent skip (REQ-006 / CLM-004).
func lookupParser(format string) (Parser, error) {
	parser, ok := formatParsers[format]
	if !ok {
		return nil, &ConfigError{Message: fmt.Sprintf("unknown output format %q: must be \"sarif\"", format)}
	}
	return parser, nil
}

// ParsePackFindings parses a findings engine's normalized output for the pack
// engine dispatch path (SPEC-031 REQ-005/REQ-006/CLM-019/CLM-036). It resolves
// the parser exclusively through lookupParser("sarif") — the dispatch path owns
// no engine enumeration and never references golangci-json/eslint-json. The
// returned violations are stamped with CheckTypeFindings, the pack-findings pass.
// A non-SARIF input fails loud via parseSarif's JSON rejection.
func ParsePackFindings(out []byte) ([]Violation, error) {
	parser, err := lookupParser("sarif")
	if err != nil {
		return nil, fmt.Errorf("resolving sarif parser for pack findings: %w", err)
	}
	return parser(out, CheckTypeFindings)
}

// sarifLog is the subset of a SARIF 2.1.0 log the sarif parser consumes.
type sarifLog struct {
	Runs []struct {
		Results []struct {
			RuleID  string `json:"ruleId"`
			Level   string `json:"level"`
			Message struct {
				Text string `json:"text"`
			} `json:"message"`
			Locations []struct {
				PhysicalLocation struct {
					ArtifactLocation struct {
						URI string `json:"uri"`
					} `json:"artifactLocation"`
					Region struct {
						StartLine int `json:"startLine"`
						Snippet   struct {
							Text string `json:"text"`
						} `json:"snippet"`
					} `json:"region"`
				} `json:"physicalLocation"`
			} `json:"locations"`
			// PartialFingerprints is the SARIF stable, content-derived fingerprint
			// (semgrep emits these). When present it is the line-INDEPENDENT identity
			// the baseline keys on, so multiple same-rule findings in one file stay
			// distinct and a finding survives unrelated line shifts. Engines that omit
			// it fall back to the snippet text, then to the coarse message identity.
			PartialFingerprints map[string]string `json:"partialFingerprints"`
			// Suppressions is the SARIF suppression list (ISSUE-017). A result with a
			// non-empty suppressions array is INACTIVE per the SARIF spec — e.g.
			// semgrep, in --sarif mode, emits `// nosemgrep`-suppressed findings as
			// results carrying `suppressions: [{kind:"inSource"}]` rather than
			// dropping them. parseSarif must not count a suppressed result as a live
			// violation, or an inline-justified finding reads as a false failure.
			Suppressions []struct {
				Kind string `json:"kind"`
			} `json:"suppressions"`
		} `json:"results"`
	} `json:"runs"`
}

// parseSarif parses a SARIF log into violations: File from
// locations[0].physicalLocation.artifactLocation.uri, Line from
// region.startLine, Message from message.text, Rule from ruleId, Severity from
// level (error/warning, default error when level absent).
func parseSarif(out []byte, target CheckType) ([]Violation, error) {
	trimmed := bytes.TrimSpace(out)
	if len(trimmed) == 0 {
		return nil, nil
	}
	var log sarifLog
	if err := json.Unmarshal(trimmed, &log); err != nil {
		return nil, fmt.Errorf("parsing SARIF output: %w", err)
	}
	var violations []Violation
	for _, run := range log.Runs {
		for _, r := range run.Results {
			// A SARIF result carrying suppressions is inactive (ISSUE-017): skip it so
			// an inline-justified `// nosemgrep` finding is not counted as a violation.
			if len(r.Suppressions) > 0 {
				continue
			}
			file, line, snippet := "", 0, ""
			if len(r.Locations) > 0 {
				pl := r.Locations[0].PhysicalLocation
				file = pl.ArtifactLocation.URI
				line = pl.Region.StartLine
				snippet = strings.TrimSpace(pl.Region.Snippet.Text)
			}
			violations = append(violations, Violation{
				Pass:        target,
				File:        file,
				Line:        line,
				Message:     r.Message.Text,
				Severity:    sarifSeverity(r.Level),
				Rule:        r.RuleID,
				Fingerprint: sarifFingerprint(r.PartialFingerprints, snippet),
			})
		}
	}
	return violations, nil
}

// sarifFingerprint derives a content-based, line-INDEPENDENT identity for a SARIF
// result: the partialFingerprints (deterministically ordered) when the engine emits
// them, else the region snippet text. Returns "" when the engine provides neither,
// in which case the baseline falls back to its coarse message-level identity. Reads
// no source and knows no language — it consumes only what the SARIF carries.
func sarifFingerprint(partial map[string]string, snippet string) string {
	if len(partial) > 0 {
		keys := make([]string, 0, len(partial))
		for k := range partial {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, k := range keys {
			parts = append(parts, k+"="+partial[k])
		}
		return strings.Join(parts, ";")
	}
	return snippet
}

// sarifSeverity maps a SARIF level to the check severity vocabulary. SARIF
// "warning" maps to warning; everything else (including absent/error/note)
// defaults to error so an absent level is treated as a finding.
func sarifSeverity(level string) string {
	if level == "warning" {
		return "warning"
	}
	return "error"
}
