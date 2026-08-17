package rulepathfixture

// POSITIVE fixture: the rule must NOT fire here. The registry arrives by
// injection, so the forbidden globalRegistry access never appears.

type registry interface{ Get(string) string }

func LookupClean(reg registry, key string) string { return reg.Get(key) }
