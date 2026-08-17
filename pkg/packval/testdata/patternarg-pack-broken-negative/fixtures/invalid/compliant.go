package patternargfixture

// THE BROKEN NEGATIVE. This file is DECLARED as the rule's negative fixture but it
// does not violate the rule at all — it never calls forbiddenCall, exactly like the
// positive fixture. Under BUNDLE-005 REQ-011 a negative fixture that does not trigger
// its rule is a phase-3 FAILURE, so this pack is the falsification substrate for
// CLM-007: it MUST turn phase 3 red once pattern-arg rules dispatch. Do not "fix" it.

func LookupCompliant(key string) string { return key }
