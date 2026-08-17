package engineerrorfixture

// NEGATIVE fixture. It genuinely reaches into the global singleton, so a WORKING
// rule would fire here. The rule is broken, so nothing fires — which is exactly
// how a broken engine run surfaces to phase 3.

func LookupViolating(key string) string { return globalRegistry.Get(key) }
