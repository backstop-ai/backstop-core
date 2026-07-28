package main

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"github.com/backstop-ai/backstop-core/pkg/check"
	"github.com/backstop-ai/backstop-core/pkg/pack"
)

// Engine provisioning SPLITS by who owns the tool (SPEC-034 REQ-008 / REQ-006).
//
//   - Layer-0 NATIVE toolchain (`go`, `golangci-lint`) is ASSUME-PRESENT: the
//     project owns its own compiler and linter, so backstop NEVER installs them.
//     A missing binary is a fail-loud *check.ConfigError (exit 2) NAMING the
//     tool — never a silent skip and never an auto-install attempt. These engines
//     carry a nil Provision record in the EngineBinding table (CLM-026/CLM-027).
//     This absent-tool fail-loud is NEW behavior the bridge adds; neither the
//     bespoke path nor the bare engine path emitted it before.
//
//   - Backstop-INTRODUCED engines (semgrep, ast-grep) carry a pinned Provision
//     record and stay auto-provisioned through the declared model (the lock /
//     VerifyLock path). Their absence on PATH is NOT a provisioning failure here:
//     they are provisioned per declaration, distinct from the assume-present
//     native toolchain (CLM-028). semgrep's findings path is unchanged (REQ-006);
//     only its provisioning is data-driven now.

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

// provisionEngines enforces the split-provisioning model over the engines the
// given packs reference (REQ-008). For every engine with a NIL Provision (the
// assume-present Layer-0 native toolchain), it verifies the tool binary resolves
// on PATH and fail-louds with a *check.ConfigError (exit 2) naming the tool when
// it does not. Engines with a non-nil Provision (semgrep/ast-grep) are skipped:
// they are pinned and auto-provisioned through the declared model, not
// assume-present. Tool names are deduped so a pack declaring both go-build and
// go-test checks `go` once.
func provisionEngines(packs []*pack.Manifest) error {
	resolve := resolveBinaryResolver()

	// Collect the assume-present tools to verify, deduped and ordered so a missing
	// tool fails loud deterministically.
	required := map[string]struct{}{}
	for _, manifest := range packs {
		seenEngine := map[string]struct{}{}
		for _, rule := range manifest.Content.Ruleset.Rules {
			if _, done := seenEngine[rule.Engine]; done {
				continue
			}
			seenEngine[rule.Engine] = struct{}{}

			binding, lookupErr := resolveEngineRegistry(manifest).Lookup(rule.Engine)
			if lookupErr != nil {
				return fmt.Errorf("pack %s: %w", manifest.NormalizedName, lookupErr)
			}
			// TRUST GATE (SPEC-035 REQ-003/CLM-030) — provisionEngines is the
			// EARLIEST resolveEngineRegistry caller that leads to running a tool, so
			// the allowlist check is routed HERE too: an un-allowlisted (or
			// version-divergent) provisioned tool fails loud with a *check.ConfigError
			// BEFORE provisioning, the natural chokepoint ahead of validate + dispatch.
			// It is the SAME engine.CheckToolAllowed the dispatch gate runs, via the
			// shared checkEngineToolAllowed (a nil-Provision binding is exempt here —
			// its on-PATH fail-loud below governs it, not the allowlist+lock pin).
			if gateErr := checkEngineToolAllowed(manifest, binding); gateErr != nil {
				return gateErr
			}
			// Non-nil Provision => backstop-introduced, pinned + auto-provisioned;
			// not subject to the assume-present fail-loud (CLM-028).
			if binding.Provision != nil {
				continue
			}
			tool := engineToolName(binding.Command)
			if tool == "" {
				// An assume-present engine with no command (the sandbox engine) ships
				// its own executable via the pack; nothing to resolve on PATH.
				continue
			}
			required[tool] = struct{}{}
		}
	}

	tools := make([]string, 0, len(required))
	for tool := range required {
		tools = append(tools, tool)
	}
	sort.Strings(tools)

	for _, tool := range tools {
		if _, err := resolve(tool); err != nil {
			return &check.ConfigError{Message: fmt.Sprintf(
				"required tool %q not found on PATH: it is an assume-present Layer-0 native tool the project must provide on PATH (backstop never auto-provisions it); add %q to PATH and retry",
				tool, tool,
			)}
		}
	}
	return nil
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
