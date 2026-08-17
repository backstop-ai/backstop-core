package rulepathfixture

// NEGATIVE fixture: the rule MUST fire here — this reaches into the package-level
// globalRegistry singleton directly instead of taking it by injection.

func LookupViolating(key string) string { return globalRegistry.Get(key) }
