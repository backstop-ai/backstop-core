package engineerrorfixture

// POSITIVE fixture. The rule that would judge it is structurally invalid, so the
// engine never evaluates it — see rules/broken.yml for what semgrep actually does.

type registry interface{ Get(string) string }

func LookupClean(reg registry, key string) string { return reg.Get(key) }
