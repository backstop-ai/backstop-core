package waiver

import (
	"strings"
	"time"
)

// Finding is a code-located engine finding, at its SARIF-reported location.
type Finding struct {
	RuleID   string
	File     string
	Line     int
	EndLine  int
	Severity string
}

// Location is a file+line coordinate for a token.
type Location struct {
	File string
	Line int
}

// LineReader yields the RAW bytes of one source line (1-indexed). It is the
// ONLY source access adjudication is given — there is no language identifier and
// no comment parser anywhere on the suppression path (REQ-002, the zero-baked
// property). ok is false when the line does not exist.
type LineReader func(file string, line int) (string, bool)

// Policy decides the declared non-waivable tier (REQ-006): a rule or severity a
// pack manifest / enforcement policy declares un-waivable. It is supplied to
// adjudication, never core-hardcoded. A nil Policy treats every rule as waivable.
type Policy interface {
	Waivable(ruleID string, severity string) bool
}

// Diagnostic kinds — a waiver-produced gate finding is either a malformed token
// or a token targeting a declared non-waivable rule.
const (
	DiagnosticMalformed   = "malformed"
	DiagnosticNonWaivable = "non-waivable"
)

// Diagnostic is a waiver-produced gate finding.
type Diagnostic struct {
	RuleID  string
	File    string
	Line    int
	Message string
	Kind    string
}

// Result is the adjudication outcome consumed by the gate step and reporting.
type Result struct {
	Suppressed  []Finding
	Active      []Waiver
	Expiring    []Waiver
	Expired     []Waiver
	Unused      []Waiver
	Malformed   []Diagnostic
	NonWaivable []Diagnostic
	// Unbound carries tokens whose pack namespace matches no pack the project's lock
	// records (ISSUE-097). ADJUDICATE NEVER POPULATES IT: it is produced by the
	// tree-driven Unbound scan and attached by the caller. Reading a finding-driven
	// Result and concluding the unbound class is covered is the precise misconception
	// this field exists to correct.
	Unbound []Diagnostic
}

// graceWindow is the pre-expiry warning window: an active waiver whose expiry is
// within this window of now is surfaced as Expiring (REQ-004). It defaults to 30
// days, matching the backstop.yml waiver_warning_days default.
const graceWindow = 30 * 24 * time.Hour

// tokenState accumulates one unique harvested token and the findings whose
// association windows it appears in.
type tokenState struct {
	pt       parsedToken
	findings []Finding
}

// Adjudicate performs zero-baked line-scan adjudication (REQ-002/003/008). It
// receives the findings, a raw-bytes LineReader, a declared Policy, and now —
// and NO language identifier. For each finding it byte-scans its association
// window (the finding's own start line, trailing, and the single line
// immediately above) for literal @waiver: occurrences, parses each, and
// suppresses ONLY on an EXACT rule-id match with per-finding location identity
// (no file/rule blanket). Malformed tokens become Diagnostics.
//
// Pipeline: token harvest -> grammar parse (malformed) -> exact rule-id match ->
// [non-waivable policy check, Phase C] -> lifecycle classification (active
// suppresses / expired re-fires / expiring warns) -> unused/dangling detection.
func Adjudicate(findings []Finding, read LineReader, policy Policy, now time.Time) Result {
	res := Result{}
	if read == nil {
		return res
	}

	// Harvest every unique token across all findings' association windows,
	// recording which findings each token is associated with. Insertion order is
	// stable (findings order, window order, column order) for deterministic output.
	tokens := map[string]*tokenState{}
	var order []string
	for _, f := range findings {
		for _, tok := range harvestWindow(f, read) {
			key := diagKey(tok.file, tok.line, tok.col)
			st, ok := tokens[key]
			if !ok {
				st = &tokenState{pt: tok}
				tokens[key] = st
				order = append(order, key)
			}
			st.findings = append(st.findings, f)
		}
	}

	suppressedFindings := map[string]bool{}
	for _, key := range order {
		st := tokens[key]
		tok := st.pt
		if tok.err != nil {
			res.Malformed = append(res.Malformed, Diagnostic{
				RuleID:  associatedRuleID(st),
				File:    tok.file,
				Line:    tok.line,
				Message: tok.err.Error(),
				Kind:    DiagnosticMalformed,
			})
			continue
		}

		// Exact rule-id match: the token's rule-id must equal an associated
		// finding's rule-id. A token matching nothing is unused/dangling.
		matched := matchingFindings(st)
		if len(matched) == 0 {
			res.Unused = append(res.Unused, tok.waiver)
			continue
		}

		// Non-waivable check (REQ-006): a token targeting a Policy-declared
		// non-waivable rule/severity does NOT suppress and is a gate ERROR. The
		// decision is Policy-supplied, never core-hardcoded (CLM-027).
		if policy != nil && !policy.Waivable(tok.waiver.RuleID, matched[0].Severity) {
			res.NonWaivable = append(res.NonWaivable, Diagnostic{
				RuleID:  tok.waiver.RuleID,
				File:    tok.file,
				Line:    tok.line,
				Message: "rule " + tok.waiver.RuleID + " is declared non-waivable and cannot be suppressed by a @waiver token",
				Kind:    DiagnosticNonWaivable,
			})
			continue
		}

		// Lifecycle classification per matched finding. Expiry lifts the shield
		// immediately (Sharp Edge 5).
		if tok.waiver.Expired(now) {
			res.Expired = append(res.Expired, tok.waiver)
			continue
		}
		res.Active = append(res.Active, tok.waiver)
		if tok.waiver.Expiry.Sub(now) <= graceWindow {
			res.Expiring = append(res.Expiring, tok.waiver)
		}
		for _, f := range matched {
			fk := diagKey(f.File, f.Line, 0) + ":" + f.RuleID
			if suppressedFindings[fk] {
				continue
			}
			suppressedFindings[fk] = true
			res.Suppressed = append(res.Suppressed, f)
		}
	}
	return res
}

// matchingFindings returns the findings a parsed token exactly matches by
// rule-id among the findings it is associated with (per-finding location
// identity; no file/rule blanket).
func matchingFindings(st *tokenState) []Finding {
	var out []Finding
	for _, f := range st.findings {
		if f.RuleID == st.pt.waiver.RuleID {
			out = append(out, f)
		}
	}
	return out
}

// associatedRuleID returns a representative rule-id for a malformed token's
// diagnostic (the first associated finding's rule-id, if any).
func associatedRuleID(st *tokenState) string {
	if len(st.findings) > 0 {
		return st.findings[0].RuleID
	}
	return ""
}

// parsedToken is one occurrence scanned from a window line: either a parsed
// waiver or a parse error, plus its coordinates.
type parsedToken struct {
	waiver Waiver
	err    error
	file   string
	line   int
	col    int
}

// harvestWindow reads a finding's association window (start line trailing, and
// the single line immediately above) via the LineReader and returns every
// @waiver token occurrence parsed from those raw bytes. It NEVER reads a line
// outside the window and NEVER parses the source — it scans raw bytes only.
func harvestWindow(f Finding, read LineReader) []parsedToken {
	var out []parsedToken
	for _, line := range windowLines(f) {
		text, ok := read(f.File, line)
		if !ok {
			continue
		}
		for _, col := range scanOccurrences(text) {
			w, err := ParseToken(text[col:], Location{File: f.File, Line: line})
			out = append(out, parsedToken{waiver: w, err: err, file: f.File, line: line, col: col})
		}
	}
	return out
}

// windowLines returns the association window for a finding: the line
// immediately above its start line, and the start line itself (trailing). A
// multi-line finding associates via its START line only (REQ-008) — EndLine
// does not widen the window.
func windowLines(f Finding) []int {
	lines := []int{f.Line}
	if f.Line > 1 {
		lines = append(lines, f.Line-1)
	}
	return lines
}

// scanOccurrences returns the byte offsets of every literal @waiver: occurrence
// in a line's raw bytes — the zero-baked byte scan (no comment lexer).
func scanOccurrences(text string) []int {
	var cols []int
	off := 0
	for {
		i := strings.Index(text[off:], waiverMarker)
		if i < 0 {
			return cols
		}
		cols = append(cols, off+i)
		off += i + len(waiverMarker)
	}
}

// diagKey uniquely identifies a token occurrence for diagnostic de-duplication
// across overlapping finding windows.
func diagKey(file string, line, col int) string {
	return file + ":" + itoa(line) + ":" + itoa(col)
}

// itoa is a tiny non-allocating-ish int formatter to avoid importing strconv
// solely for keys.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
