package main

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"github.com/backstop-ai/backstop-core/pkg/check"
	"github.com/backstop-ai/backstop-core/pkg/pack"
	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
)

// Engine provisioning PRESENCE-CHECKS EVERY declared engine tool. Backstop
// installs nothing, ever — so a tool that is not already on PATH cannot run, and
// an engine that cannot run scanned nothing.
//
// The Provision record governs TRUST, never installation (ISSUE-112). It is a
// trusted-tool allowlist entry plus a version pin: it says backstop is willing to
// run this tool at this version, and it is checked against
// engine.TrustedToolAllowlist before anything else. It has never fetched,
// installed, or otherwise guaranteed a binary.
//
// That distinction was previously mis-stated as a SPLIT (SPEC-034 REQ-008 /
// CLM-026/027/028), under which:
//
//   - nil-Provision Layer-0 tools (`go`, `golangci-lint`) were "assume-present"
//     and fail-louded when absent, but
//   - pinned tools (semgrep, ast-grep) were "pinned + auto-provisioned" and were
//     SKIPPED entirely — neither installed nor probed.
//
// Since nothing performs the "auto-provisioning" that exemption assumed, a pinned
// engine whose binary was missing flowed straight into dispatch, produced empty
// output, parsed as zero findings, and reported a clean pass. ISSUE-112 records
// that reaching a real CI gate. BOTH kinds are now probed by the same check and
// both fail loud with a *check.ConfigError (exit 2) NAMING the tool; only the
// refusal MESSAGE differs, because the two have different remedies to offer.
//
// What did NOT change: backstop still installs nothing on either branch, the
// allowlist trust gate still fires ahead of any presence probe, and the findings
// path of a tool that DOES run is untouched (SPEC-034 REQ-006).

// binaryResolver is a test seam: nil in production (resolveBinaryResolver falls
// back to exec.LookPath), overridden by tests to simulate a tool's presence or
// absence without depending on the host PATH. Declared WITHOUT an initializer so
// it holds no package-level mutable default; the real resolver is resolved lazily
// at the call site (same shape as the sandboxedRun/engineRegistry seams).
var binaryResolver func(name string) (string, error)

// resolveBinaryResolver returns the injected resolver seam or the concrete
// exec.LookPath.
func resolveBinaryResolver() func(name string) (string, error) {
	if binaryResolver != nil {
		return binaryResolver
	}
	return exec.LookPath
}

// requiredTool is one engine tool the provisioning walk must find on PATH, with
// the attribution its refusal message needs. A bare tool name cannot say WHO
// declared it or under what pin, and a refusal that cannot answer "declared by
// what?" sends the reader hunting through every installed pack.
type requiredTool struct {
	// name is the PROBED argv[0] of the declaring binding's command — what exec
	// will actually resolve, and therefore what the user must put on PATH.
	name string
	// pack and engine name the declaration site.
	pack   string
	engine string
	// provision is the binding's trust record, or nil for an assume-present
	// Layer-0 tool. It selects which refusal message the tool gets.
	provision *engine.Provision
}

// collectRequiredEngineTools is THE ONE SITE that decides WHICH tools a pack set
// requires on PATH: it walks each manifest's rules, resolves each rule's engine
// through the registry, runs the trust gate, extracts argv[0], dedupes, and returns
// the result in sorted probed-name order.
//
// ★ IT HAS TWO CONSUMERS WITH DIFFERENT DISPOSITIONS, AND THAT IS WHY IT EXISTS
// SEPARATELY FROM provisionEngines. The gate FAILS FAST on the first absence (exit
// 2); `backstop doctor`'s engine-tools check ENUMERATES every one and reports each.
// A diagnostic that stopped at the first missing tool would be a worse gate rather
// than a better diagnostic. So the COLLECTION is shared and the DISPOSITION is each
// caller's — the same shape, and the same reasoning, as packEntrypointProber's two
// callers. Before ISSUE-134 doctor had no consumer of this walk at all, which is how
// doctor reported an all-green exit 0 on a project the gate refused with exit 2.
//
// It probes argv[0] of the DECLARED COMMAND (engineToolName), never Provision.Tool.
// argv[0] is what exec resolves; Provision.Tool is the trust key, and the two
// legitimately differ — ast-grep ships `sg` as a second entry point, so a pack may
// pin `ast-grep` and invoke `sg`. Do not unify them: the pinned name appears in the
// MESSAGE for attribution, while the probed name is the one that has to exist.
//
// ★ THE WALK RESOLVES ENGINES THROUGH RULES, NOT THROUGH manifest.Engines, AND THAT
// IS A CORRECTNESS PROPERTY RATHER THAN AN IMPLEMENTATION DETAIL. An engine no rule
// binds is never dispatched, so its tool's absence cannot produce a vacuous green,
// and probing it would refuse a run that was never going to invoke it — a live
// fixture depends on that. Widening this to manifest.Engines looks more thorough and
// makes BOTH consumers wrong: the gate would refuse, and doctor would fail on a
// project the gate passes, which is the cries-wolf failure a diagnostic never
// recovers from. Widening WHICH packs are walked is a caller's decision; this
// function's rule-driven walk is not.
//
// Tool names are deduped so a pack declaring both go-build and go-test probes `go`
// once, and sorted so a missing tool fails loud deterministically — which is also
// what makes doctor's per-tool report lines stable across runs.
func collectRequiredEngineTools(packs []*pack.Manifest) ([]requiredTool, error) {
	required := map[string]requiredTool{}
	for _, manifest := range packs {
		seenEngine := map[string]struct{}{}
		for _, rule := range manifest.Content.Ruleset.Rules {
			if _, done := seenEngine[rule.Engine]; done {
				continue
			}
			seenEngine[rule.Engine] = struct{}{}

			binding, lookupErr := resolveEngineRegistry(manifest).Lookup(rule.Engine)
			if lookupErr != nil {
				return nil, fmt.Errorf("pack %s: %w", manifest.NormalizedName, lookupErr)
			}
			// TRUST GATE (SPEC-035 REQ-003/CLM-030) — this collection is the EARLIEST
			// resolveEngineRegistry walk that leads to running a tool, so the allowlist
			// check is routed HERE too: an un-allowlisted (or version-divergent)
			// provisioned tool fails loud with a *check.ConfigError BEFORE any presence
			// probe, the natural chokepoint ahead of validate + dispatch. It is the SAME
			// engine.CheckToolAllowed the dispatch gate runs, via the shared
			// checkEngineToolAllowed (a nil-Provision binding returns early there — the
			// on-PATH fail-loud in the CALLER governs it, not the allowlist+lock pin).
			// Keep it AHEAD of the probe: a tool backstop refuses to trust should be
			// reported as untrusted, not as missing.
			if gateErr := checkEngineToolAllowed(manifest, binding); gateErr != nil {
				return nil, gateErr
			}
			tool := engineToolName(binding.Command)
			if tool == "" {
				// An engine with no command (the sandbox engine) ships its own
				// executable via the pack; nothing to resolve on PATH.
				continue
			}
			if _, dup := required[tool]; dup {
				continue
			}
			required[tool] = requiredTool{
				name:      tool,
				pack:      manifest.NormalizedName,
				engine:    rule.Engine,
				provision: binding.Provision,
			}
		}
	}

	names := make([]string, 0, len(required))
	for name := range required {
		names = append(names, name)
	}
	sort.Strings(names)

	tools := make([]requiredTool, 0, len(names))
	for _, name := range names {
		tools = append(tools, required[name])
	}
	return tools, nil
}

// provisionEngines verifies that every engine tool the given packs reference
// resolves on PATH, and fail-louds with a *check.ConfigError (exit 2) naming the
// tool when one does not. It installs nothing.
//
// SELECTION IS NOT ITS OWN: collectRequiredEngineTools decides which tools are
// required and in what order, and doctor's engine-tools check consumes that same
// authority. What lives HERE is the gate's DISPOSITION — return on the FIRST absent
// name in sorted order, as a *check.ConfigError.
//
// BOTH kinds of binding are probed — assume-present Layer-0 tools (nil Provision)
// and pinned ones alike (ISSUE-112). A pinned Provision is an allowlist trust
// entry, not an installer, so exempting pinned bindings from this probe let an
// engine with no binary report a clean, finding-free scan. Only the refusal
// message differs between the two; see the file header.
func provisionEngines(packs []*pack.Manifest) error {
	tools, err := collectRequiredEngineTools(packs)
	if err != nil {
		return err
	}

	resolve := resolveBinaryResolver()
	for _, tool := range tools {
		if _, resolveErr := resolve(tool.name); resolveErr != nil {
			return &check.ConfigError{Message: absentToolMessage(tool)}
		}
	}
	return nil
}

// absentToolMessage renders the refusal for a tool that did not resolve on PATH.
// The two branches differ because they offer different remedies, not because one
// is more forgiving: neither installs anything.
func absentToolMessage(rt requiredTool) string {
	if rt.provision == nil {
		return fmt.Sprintf(
			"required tool %q not found on PATH: it is an assume-present Layer-0 native tool the project must provide on PATH (backstop never auto-provisions it); add %q to PATH and retry",
			rt.name, rt.name,
		)
	}
	// A pinned tool names BOTH the probed argv[0] — the binary that must actually
	// exist — and the Provision tool+version it rode in on. When the two are equal
	// this reads naturally; when they diverge (a pack pinning `ast-grep` but
	// invoking `sg`) the reader learns the missing binary AND the pin behind it.
	return fmt.Sprintf(
		"required tool %q not found on PATH: it is argv[0] of the command engine %q of pack %s declares, pinned in the trusted-tool allowlist as tool %q version %q — backstop does not install pack-declared tools (the pin is a trust entry, not an installer); add %q to PATH and retry",
		rt.name, rt.engine, rt.pack, rt.provision.Tool, rt.provision.Version, rt.name,
	)
}

// engineToolName extracts the executable name from a binding Command
// ("golangci-lint run ..." -> "golangci-lint", "go build" -> "go"). An empty
// command yields "" (the sandbox engine ships its own executable).
func engineToolName(command string) string {
	// @waiver:backstop/self/backstop.packs.backstop.self.rules.no-structural-name-split-on-spine:false-positive:2027-07-17 legitimate executable-name extraction from a command line (argv[0] is a whitespace-free token by shell semantics), not name-from-message extraction (ISSUE-062)
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}
