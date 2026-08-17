package rulepathfixture

// THE BROKEN NEGATIVE. This file is DECLARED as the rule's negative fixture but it
// does not violate the rule at all — it takes the registry by injection exactly as
// the positive fixture does. Under BUNDLE-005 REQ-011 a negative fixture that does
// not trigger its rule is a phase-3 FAILURE, so this pack is the falsification
// substrate for CLM-003: it MUST turn `pack test` red. Do not "fix" it.

type registry interface{ Get(string) string }

func LookupCompliant(reg registry, key string) string { return reg.Get(key) }
