package patternargfixture

// NEGATIVE fixture: the rule MUST fire here — it calls forbiddenCall directly,
// which is exactly what the rule's inline pattern matches.

func LookupViolating(key string) string { return forbiddenCall(key) }
