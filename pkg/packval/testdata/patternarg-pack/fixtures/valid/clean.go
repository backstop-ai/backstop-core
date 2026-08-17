package patternargfixture

// POSITIVE fixture: the rule must NOT fire here. Nothing in this file calls
// forbiddenCall, so the inline pattern has nothing to match.

func LookupClean(key string) string { return key }
