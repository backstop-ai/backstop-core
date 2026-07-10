// Package waiver implements the backstop waiver subsystem (SPEC-049): a
// backstop-native, engine-neutral, accountable "chosen not to fix" valve.
//
// A waiver is an inline DSL token
//
//	@waiver:<rule-id>:<reason-code>:<expiry>[ <note-and/or-issue-ref>]
//
// that a human or runtime agent writes inside whatever comment their language
// already uses, at a finding's source location. The load-bearing property of
// the whole subsystem is that adjudication is ZERO-BAKED: it byte-scans the raw
// bytes of a finding's own SARIF-reported line for the token and NEVER parses
// source or encodes any language's comment syntax. No language identifier and
// no comment lexer ever enter this package's suppression path.
package waiver

import (
	"fmt"
	"strings"
	"time"
)

// ReasonCode is the CLOSED enum of waiver reason codes (REQ-001). Exactly four
// members are valid; any other value is a malformed token.
type ReasonCode string

const (
	// ReasonFalsePositive marks a finding the author asserts is not a real
	// problem. It carries the long-lived default duration.
	ReasonFalsePositive ReasonCode = "false-positive"
	// ReasonAcceptedRisk marks a real finding the author knowingly accepts.
	ReasonAcceptedRisk ReasonCode = "accepted-risk"
	// ReasonDeferred marks a finding the author intends to fix later.
	ReasonDeferred ReasonCode = "deferred"
	// ReasonThirdParty marks a finding in third-party code out of the author's
	// direct control.
	ReasonThirdParty ReasonCode = "third-party"
)

// validReason reports whether rc is one of the four closed-enum members.
func validReason(rc ReasonCode) bool {
	switch rc {
	case ReasonFalsePositive, ReasonAcceptedRisk, ReasonDeferred, ReasonThirdParty:
		return true
	}
	return false
}

// expiryLayout is the ISO-8601 date layout every expiry must match.
const expiryLayout = "2006-01-02"

// waiverMarker is the literal marker adjudication byte-scans for. It is a token
// foreign to every engine, so engines emit findings normally and backstop does
// 100% of the suppressing.
const waiverMarker = "@waiver:"

// Waiver is one parsed inline waiver, located per-finding.
type Waiver struct {
	RuleID   string
	Reason   ReasonCode
	Expiry   time.Time
	Note     string
	IssueRef string
	File     string
	Line     int
}

// Default reason-code durations. false-positive is long-lived (~1yr); the other
// three are short-lived (~90d). Durations are tunable configuration, not
// contract, but every reason-code MUST resolve to a default (REQ-004).
const (
	longLivedDuration  = 365 * 24 * time.Hour
	shortLivedDuration = 90 * 24 * time.Hour
)

// DefaultDuration returns the default expiry duration for a reason-code, used to
// pre-fill an authoring token (REQ-014). ok is false for an unknown reason-code.
func DefaultDuration(rc ReasonCode) (time.Duration, bool) {
	switch rc {
	case ReasonFalsePositive:
		return longLivedDuration, true
	case ReasonAcceptedRisk, ReasonDeferred, ReasonThirdParty:
		return shortLivedDuration, true
	}
	return 0, false
}

// Expired reports whether the waiver is expired at now — true when now is AT or
// past the waiver's expiry. Expiry lifts the shield IMMEDIATELY: the grace is
// delivered earlier as the pre-expiry warning, not as a post-expiry window
// (Sharp Edge 5).
func (w Waiver) Expired(now time.Time) bool {
	return !now.Before(w.Expiry)
}

// issueRefLooksLike reports whether a whitespace-delimited word looks like an
// issue reference: a #-number (e.g. #123) or a KEY-number (e.g. ISSUE-050).
func issueRefLooksLike(word string) bool {
	if len(word) >= 2 && word[0] == '#' {
		for _, r := range word[1:] {
			if r < '0' || r > '9' {
				return false
			}
		}
		return true
	}
	dash := strings.IndexByte(word, '-')
	if dash <= 0 || dash == len(word)-1 {
		return false
	}
	key, num := word[:dash], word[dash+1:]
	for _, r := range key {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	for _, r := range num {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// ParseToken parses a single harvested token occurrence into a Waiver, or
// returns a non-nil error for any malformed token (REQ-001 / REQ-007). raw is
// the raw line bytes containing the token; loc is the token's file+line. The
// grammar is deterministic: the token core is exactly rule-id:reason:expiry,
// with an optional trailing note and/or issue reference after the first space.
//
// rule-ids are kebab-case and pack-namespaced with '/', never ':' — so the core
// splits unambiguously on ':'.
func ParseToken(raw string, loc Location) (Waiver, error) {
	idx := strings.Index(raw, waiverMarker)
	if idx < 0 {
		return Waiver{}, fmt.Errorf("waiver: no %s token in %q", waiverMarker, raw)
	}
	rest := raw[idx+len(waiverMarker):]

	// Split the core (rule:reason:expiry) from the optional trailing text at the
	// first run of whitespace.
	core := rest
	trailing := ""
	if sp := strings.IndexAny(rest, " \t"); sp >= 0 {
		core = rest[:sp]
		trailing = strings.TrimSpace(rest[sp+1:])
	}

	parts := strings.Split(core, ":")
	if len(parts) != 3 {
		return Waiver{}, fmt.Errorf("waiver: malformed structure %q: want rule-id:reason-code:expiry", core)
	}
	ruleID, reasonStr, expiryStr := parts[0], parts[1], parts[2]
	if ruleID == "" {
		return Waiver{}, fmt.Errorf("waiver: missing rule-id in %q", core)
	}
	reason := ReasonCode(reasonStr)
	if !validReason(reason) {
		return Waiver{}, fmt.Errorf("waiver: unknown reason-code %q", reasonStr)
	}
	if expiryStr == "" {
		return Waiver{}, fmt.Errorf("waiver: missing expiry in %q", core)
	}
	expiry, err := time.Parse(expiryLayout, expiryStr)
	if err != nil {
		return Waiver{}, fmt.Errorf("waiver: invalid expiry %q: want YYYY-MM-DD", expiryStr)
	}

	w := Waiver{
		RuleID: ruleID,
		Reason: reason,
		Expiry: expiry,
		File:   loc.File,
		Line:   loc.Line,
	}
	if trailing != "" {
		w.Note, w.IssueRef = splitTrailing(trailing)
	}
	return w, nil
}

// splitTrailing separates the trailing text into a free-text note and an
// optional issue reference. The first word that looks like an issue reference
// becomes IssueRef; the remaining words (joined) become the Note.
func splitTrailing(trailing string) (note string, issueRef string) {
	words := strings.Fields(trailing)
	kept := make([]string, 0, len(words))
	for _, w := range words {
		if issueRef == "" && issueRefLooksLike(w) {
			issueRef = w
			continue
		}
		kept = append(kept, w)
	}
	return strings.Join(kept, " "), issueRef
}
