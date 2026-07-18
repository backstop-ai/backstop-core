package main

// sharedRunCache memoizes an engine run's post-stdout_artifact PAYLOAD bytes by the
// OPAQUE declared run_group key (ISSUE-068 Option C). Two DISTINCT engines that
// declare the SAME non-empty run_group (e.g. a test engine and a coverage engine
// pointed at one superset command) SHARE ONE run: the first engine to dispatch runs
// the command/producer and stores its payload here; a later engine in the same group
// REUSES that payload and skips re-running the command, then fans it into its OWN
// convert. Core dedupes SOLELY by this declared key — it NEVER inspects, parses, or
// normalizes the commands (thin-executor / DD-3).
//
// The cache is per-GATE: gate.go constructs ONE instance and threads it through the
// pack_engines (writer) and coverage (reader) dispatch call-sites via the WithCache
// entry points, so a run_group-shared suite runs ONCE across the whole gate. Every
// OTHER caller (code check, substantiveness, contracts, and every existing test)
// gets a FRESH throwaway cache from the plain dispatchPackEngines/dispatchPackCoverage
// wrappers, preserving today's unchanged two-run behavior. It is a per-invocation
// value passed by pointer — never package-global mutable state.
type sharedRunCache struct {
	byKey map[string][]byte
}

// newSharedRunCache returns an empty cache ready to memoize run payloads.
func newSharedRunCache() *sharedRunCache {
	return &sharedRunCache{byKey: map[string][]byte{}}
}

// get returns the memoized payload for a NON-EMPTY run_group key, or (nil, false)
// when the key is empty (the safe default: no dedup) or not yet cached. An empty key
// NEVER hits — an engine that declares no run_group always runs its own command. The
// key is NAMESPACED by pack identity (ISSUE-068): two DIFFERENT packs that happen to
// declare the same run_group string must NOT share a run, so the pack's normalized
// name qualifies the opaque declared key.
func (c *sharedRunCache) get(namespace, runGroup string) ([]byte, bool) {
	if c == nil || runGroup == "" {
		return nil, false
	}
	payload, ok := c.byKey[runCacheKey(namespace, runGroup)]
	return payload, ok
}

// put memoizes payload under a NON-EMPTY run_group key, NAMESPACED by pack identity
// (ISSUE-068). An empty run_group is a no-op, so an engine with no declared run_group
// never pollutes the cache and never dedupes.
func (c *sharedRunCache) put(namespace, runGroup string, payload []byte) {
	if c == nil || runGroup == "" {
		return
	}
	c.byKey[runCacheKey(namespace, runGroup)] = payload
}

// runCacheKey qualifies the OPAQUE declared run_group with the owning pack's identity
// so the memoized run is scoped PER PACK (ISSUE-068 cross-pack collision fix). The NUL
// separator cannot appear in a pack's NormalizedName (validated to slash-joined
// [A-Za-z0-9-] parts) nor in a YAML-scalar run_group, so the two segments can never
// alias across a boundary. Core still NEVER inspects what the run_group MEANS
// (thin-executor / DD-3) — it only prefixes the pack namespace.
func runCacheKey(namespace, runGroup string) string {
	return namespace + "\x00" + runGroup
}
