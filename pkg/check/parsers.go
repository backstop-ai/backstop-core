package check

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Parser translates raw tool output into violations for a target CheckType.
// It mirrors the signature of the existing parse* funcs (e.g. parseGolangciJSON)
// so the named-format registry can wrap them without behavioral drift.
type Parser func(out []byte, target CheckType) ([]Violation, error)

// formatParsers maps each named output format to its Parser. The go-tool
// formats wrap the existing parse* funcs in check.go verbatim so the landed
// ISSUE-002 Go parsers have no behavioral drift; eslint-json/tsc/sarif/
// regex-lines are new pure functions with no tool invocation.
var formatParsers = map[string]Parser{
	"golangci-json": func(out []byte, target CheckType) ([]Violation, error) {
		return retargetViolations(parseGolangciJSON(out))(target)
	},
	"go-build": func(out []byte, target CheckType) ([]Violation, error) {
		return retarget(parseGoBuildErrors(out), target), nil
	},
	"go-test": func(out []byte, target CheckType) ([]Violation, error) {
		return retarget(parseGoTestFailures(out), target), nil
	},
	"eslint-json": parseESLintJSON,
	"tsc":         parseTscOutput,
	"sarif":       parseSarif,
	"regex-lines": parseRegexLines,
}

// lookupParser resolves a named output format to a Parser, or returns an error
// for an unknown name. Fail-loud: a declared toolchain whose format is not
// known is a config error, never a silent skip (REQ-006 / CLM-004).
func lookupParser(format string) (Parser, error) {
	parser, ok := formatParsers[format]
	if !ok {
		return nil, &ConfigError{Message: fmt.Sprintf("unknown output format %q: must be one of golangci-json, go-build, go-test, eslint-json, tsc, sarif, regex-lines", format)}
	}
	return parser, nil
}

// retarget stamps each violation's Pass with the target check type. The
// generic command executor binds a format to a pass at execution time, so the
// parser's violations must carry the pass they were produced for.
func retarget(violations []Violation, target CheckType) []Violation {
	for i := range violations {
		violations[i].Pass = target
	}
	return violations
}

// retargetViolations adapts a (violations, error) pair from a wrapped parser
// into a target-stamping closure, preserving the error.
func retargetViolations(violations []Violation, err error) func(CheckType) ([]Violation, error) {
	return func(target CheckType) ([]Violation, error) {
		if err != nil {
			return nil, err
		}
		return retarget(violations, target), nil
	}
}

// eslintFile is one entry of eslint's JSON array output.
type eslintFile struct {
	FilePath string `json:"filePath"`
	Messages []struct {
		RuleID   string `json:"ruleId"`
		Severity int    `json:"severity"`
		Message  string `json:"message"`
		Line     int    `json:"line"`
	} `json:"messages"`
}

// parseESLintJSON parses eslint `--format json` output (an array of
// {filePath, messages[{ruleId, severity 1|2, message, line}]}) into violations.
// severity 2 maps to error, 1 to warning; File=filePath, Rule=ruleId.
func parseESLintJSON(out []byte, target CheckType) ([]Violation, error) {
	trimmed := bytes.TrimSpace(out)
	if len(trimmed) == 0 {
		return nil, nil
	}
	var files []eslintFile
	if err := json.Unmarshal(trimmed, &files); err != nil {
		return nil, err
	}
	var violations []Violation
	for _, f := range files {
		for _, m := range f.Messages {
			violations = append(violations, Violation{
				Pass:     target,
				File:     f.FilePath,
				Line:     m.Line,
				Message:  m.Message,
				Severity: eslintSeverity(m.Severity),
				Rule:     m.RuleID,
			})
		}
	}
	return violations, nil
}

// eslintSeverity maps eslint's numeric severity (2=error, 1=warning) to the
// check severity vocabulary. Unknown values default to error (fail-loud).
func eslintSeverity(sev int) string {
	if sev == 1 {
		return "warning"
	}
	return "error"
}

// tscLineRe matches a tsc --noEmit diagnostic line:
// `file(line,col): error TSxxxx: message` (severity keyword error|warning).
var tscLineRe = regexp.MustCompile(`^(.+?)\((\d+),(\d+)\):\s*(error|warning)\s+(TS\d+):\s*(.+)$`)

// parseTscOutput parses tsc --noEmit output lines into violations. Lines that
// do not match the diagnostic shape (summary lines, blanks) are ignored.
// Rule=TSxxxx, Severity from the error/warning keyword.
func parseTscOutput(out []byte, target CheckType) ([]Violation, error) {
	var violations []Violation
	for _, raw := range strings.Split(string(out), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		m := tscLineRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		lineNo, err := strconv.Atoi(m[2])
		if err != nil {
			continue
		}
		violations = append(violations, Violation{
			Pass:     target,
			File:     m[1],
			Line:     lineNo,
			Message:  m[6],
			Severity: m[4],
			Rule:     m[5],
		})
	}
	return violations, nil
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
					} `json:"region"`
				} `json:"physicalLocation"`
			} `json:"locations"`
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
		return nil, err
	}
	var violations []Violation
	for _, run := range log.Runs {
		for _, r := range run.Results {
			file, line := "", 0
			if len(r.Locations) > 0 {
				pl := r.Locations[0].PhysicalLocation
				file = pl.ArtifactLocation.URI
				line = pl.Region.StartLine
			}
			violations = append(violations, Violation{
				Pass:     target,
				File:     file,
				Line:     line,
				Message:  r.Message.Text,
				Severity: sarifSeverity(r.Level),
				Rule:     r.RuleID,
			})
		}
	}
	return violations, nil
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

// defaultRegexLinesPattern is the default regex-lines pattern with named groups
// file/line/col/message. A declared toolchain may override this in a future
// extension; today the generic format uses this default shape
// (`file:line:col message`).
var defaultRegexLinesPattern = regexp.MustCompile(`^(?P<file>[^:\s]+):(?P<line>\d+):(?P<col>\d+)\s+(?P<message>.+)$`)

// parseRegexLines parses generic tool output line-by-line using the default
// named-group pattern (file/line/col/message), defaulting severity to error.
// Non-matching lines yield nothing. The rule group is optional; when present it
// populates Rule.
func parseRegexLines(out []byte, target CheckType) ([]Violation, error) {
	return parseRegexLinesWith(out, target, defaultRegexLinesPattern)
}

// parseRegexLinesWith is parseRegexLines with an explicit compiled pattern, so a
// declared toolchain can supply a custom named-group pattern. Groups file/line/
// col/message are recognized; an optional rule group populates Rule.
func parseRegexLinesWith(out []byte, target CheckType, pattern *regexp.Regexp) ([]Violation, error) {
	names := pattern.SubexpNames()
	var violations []Violation
	for _, raw := range strings.Split(string(out), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		m := pattern.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		v := Violation{Pass: target, Severity: "error"}
		for i, name := range names {
			if i == 0 || name == "" || i >= len(m) {
				continue
			}
			switch name {
			case "file":
				v.File = m[i]
			case "line":
				if n, err := strconv.Atoi(m[i]); err == nil {
					v.Line = n
				}
			case "message":
				v.Message = m[i]
			case "rule":
				v.Rule = m[i]
			}
		}
		violations = append(violations, v)
	}
	return violations, nil
}
