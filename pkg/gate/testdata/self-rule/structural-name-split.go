package fixtures

import "strings"

// Structural name extraction on the neutral spine — must be rejected by
// no-structural-name-split-on-spine (matched via the *structural-name-split* include
// hook). This is the tokenValue defect (ISSUE-062): it slices an identifier out of a
// message at the first whitespace, assuming the value is a single whitespace-free token
// — correct for a Go func name, WRONG for a spaced/quoted test description.
func tokenValue(s, key string) string {
	idx := strings.Index(s, key)
	if idx < 0 {
		return ""
	}
	rest := s[idx+len(key):]
	if end := strings.IndexAny(rest, " \t\n"); end >= 0 {
		return rest[:end]
	}
	return rest
}

// A second shape of the same defect: split the message on whitespace and take the first
// field as the name.
func firstToken(message string) string {
	return strings.Fields(message)[0]
}
