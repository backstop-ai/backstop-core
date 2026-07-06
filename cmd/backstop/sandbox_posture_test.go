package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/baseengines"
	"github.com/bmanson/backstop-core/pkg/pack"
)

// TestEndState_NoToolNameUsedAsDispatchDiscriminator is the SPEC-035 REQ-006 /
// CLM-026 end-state guard: after the isNativeSarifLintEngine / isNativeGoTestEngine
// retirements, NO tool-name literal ("golangci-lint" / "go test") drives Go
// control flow in the cmd/backstop or pkg/pack dispatch source — the strict-SARIF
// and package-scope behaviors ride DECLARED binding flags (StrictSarif /
// PackageScoped), so backstop dispatches on declarations, not tool names. The
// scan strips line comments (which legitimately NAME the retired sniffs while
// explaining them) and ignores plain `Command:` data literals (the built-in
// bindings' commands ARE "go test"/"golangci-lint" as data); it fails only on a
// tool-name literal used in a COMPARISON/sniff (HasPrefix/HasSuffix/Contains/==),
// i.e. a tool name driving a branch. The legacy pkg/check standards-routing
// `hasSemgrepSignal` is explicitly OUT of scope (delegated, not this dispatch path).
func TestEndState_NoToolNameUsedAsDispatchDiscriminator(t *testing.T) {
	toolNames := []string{`"golangci-lint"`, `"go test"`}
	sniffTokens := []string{"HasPrefix", "HasSuffix", "Contains", "== ", "!= "}

	roots := []string{".", filepath.Join("..", "..", "pkg", "pack")}
	for _, root := range roots {
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return err
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			raw, rerr := os.ReadFile(path)
			if rerr != nil {
				t.Fatalf("reading %s: %v", path, rerr)
			}
			for i, line := range strings.Split(string(raw), "\n") {
				code := line
				if idx := strings.Index(code, "//"); idx >= 0 {
					code = code[:idx] // strip the line comment
				}
				for _, tn := range toolNames {
					if !strings.Contains(code, tn) {
						continue
					}
					for _, sniff := range sniffTokens {
						if strings.Contains(code, sniff) {
							t.Errorf("%s:%d uses tool-name literal %s in a dispatch discriminator (%q); dispatch must key off declared binding flags (StrictSarif/PackageScoped), not tool names:\n  %s",
								path, i+1, tn, sniff, strings.TrimSpace(line))
						}
					}
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walking %s: %v", root, err)
		}
	}
}

// convertBindingManifest builds an in-memory manifest dispatching one ast-grep
// rule against the engine-dispatch fixture pack. ast-grep's real EngineBinding
// declares a Convert script (ast-grep/to-sarif.sh) and a Provision (allowlisted
// + pinned), so it exercises BOTH the main-tool allowlist gate and the
// sandbox-trusted convert step in one dispatch.
func convertBindingManifest() *pack.Manifest {
	return &pack.Manifest{
		NormalizedName: "test-org/engine-pack",
		Content: pack.Content{Ruleset: pack.Ruleset{Rules: []pack.Rule{
			{ID: "ast-grep-proof", Engine: "ast-grep", RulePath: "ast-grep/proof-rule.yml", Standard: "x"},
		}}},
	}
}

// TestConvert_ScriptIsSandboxTrustedNotToolAllowlisted is the SPEC-035 REQ-008 /
// CLM-033 posture pin: the pack `convert` script is arbitrary pack code that
// rests its trust on the SANDBOX, and is deliberately EXEMPT from the trusted-tool
// allowlist (which gates only the engine's main TOOL command). The convert step
// runs through the sandbox seam (resolveSandboxedRunStdout), never through
// CheckToolAllowed — so a convert script's interpreter is never allowlist-checked.
// Substantive: dispatches the real ast-grep convert pipe and proves (a) the main
// tool is the ONLY thing gated by checkEngineToolAllowed, and (b) the convert
// still runs via the sandbox seam and yields the converted SARIF finding.
func TestConvert_ScriptIsSandboxTrustedNotToolAllowlisted(t *testing.T) {
	// The ast-grep binding gates its MAIN tool (ast-grep) through the allowlist;
	// the convert script is NOT a second allowlisted tool. checkEngineToolAllowed
	// keys solely off Provision.Tool.
	bind := baseengines.Registry()["ast-grep"]
	if bind.Convert == "" {
		t.Fatal("fixture invariant: the ast-grep built-in must declare a Convert script")
	}
	if gateErr := checkEngineToolAllowed(convertBindingManifest(), bind); gateErr != nil {
		t.Fatalf("the main tool (ast-grep) is allowlisted+pinned; the allowlist gate must pass and must NOT consult the convert script: %v", gateErr)
	}

	// The convert step runs through the sandbox seam (sandbox-trusted), producing
	// SARIF — it is never subjected to a tool-allowlist check of its own.
	var gotStdin []byte
	stubSandboxedRunStdout(t, &gotStdin)
	runner := &fixtureRunner{byCmd: map[string][]byte{"ast-grep scan": []byte(astGrepJSONStdout())}}

	violations, err := dispatchPackEngines([]*pack.Manifest{convertBindingManifest()}, engineDispatchPacksDir(t), t.TempDir(), nil, runner)
	if err != nil {
		t.Fatalf("dispatch with sandbox-trusted convert must succeed: %v", err)
	}
	if len(gotStdin) == 0 {
		t.Error("the convert script must have run through the sandbox seam (received engine stdout on stdin); it is sandbox-trusted, not allowlist-gated")
	}
	if len(violations) == 0 {
		t.Error("the sandbox-trusted convert pipe must yield the converted SARIF finding")
	}
}

// TestValidator_IsSandboxTrustedNotToolAllowlisted is the SPEC-035 REQ-008 /
// CLM-034 posture pin (the validator analog of CLM-033): the sandbox engine's
// `validator` script is arbitrary pack code that rests its trust on the SANDBOX
// and is EXEMPT from the trusted-tool allowlist — the sandbox binding carries no
// Provision/tool, so checkEngineToolAllowed is a no-op for it, yet the validator
// still runs through the sandbox seam. Substantive: dispatches a real sandbox
// validator rule under an active allowlist and asserts (a) no allowlist rejection
// and (b) the validator ran through the sandbox seam.
func TestValidator_IsSandboxTrustedNotToolAllowlisted(t *testing.T) {
	// The sandbox built-in declares no Provision, so the allowlist gate is a
	// no-op — the validator is trusted via the sandbox, not the tool-allowlist.
	sandboxBind := baseengines.Registry()["sandbox"]
	if sandboxBind.Provision != nil {
		t.Fatal("fixture invariant: the sandbox built-in must carry no Provision (its validator is sandbox-trusted, not allowlisted)")
	}
	if gateErr := checkEngineToolAllowed(&pack.Manifest{NormalizedName: "acme/sb"}, sandboxBind); gateErr != nil {
		t.Fatalf("a sandbox binding has no tool to allowlist; the gate must be a no-op, got: %v", gateErr)
	}

	projectRoot := t.TempDir()
	packDir := filepath.Join(projectRoot, ".backstop", "packs")
	packRoot := filepath.Join(packDir, "acme", "sb")
	if err := os.MkdirAll(packRoot, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(packRoot, "v.sh"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write validator: %v", err)
	}
	m := &pack.Manifest{
		NormalizedName: "acme/sb",
		Content: pack.Content{Ruleset: pack.Ruleset{Rules: []pack.Rule{
			{ID: "sb", Engine: "sandbox", Validator: "v.sh", InputScope: "multi-file", Category: "presence"},
		}}},
	}

	called := false
	orig := sandboxedRun
	sandboxedRun = func(string, []string, string) ([]byte, error) { called = true; return nil, nil }
	t.Cleanup(func() { sandboxedRun = orig })

	if _, err := dispatchPackEngines([]*pack.Manifest{m}, packDir, projectRoot, nil, emptySarifRunner{}); err != nil {
		t.Fatalf("the sandbox validator carries no tool and must NOT be allowlist-gated, got: %v", err)
	}
	if !called {
		t.Fatal("the sandbox validator must still run through the sandbox seam — it is sandbox-trusted, exempt from the tool-allowlist")
	}
}

// TestSandbox_PlatformUnavailableSurfacedNotSilent is the SPEC-035 REQ-008 /
// CLM-035 posture pin: when the sandbox is unavailable on a platform (the sandbox
// is macOS sandbox-exec; a Linux build returns "sandbox unavailable"), that
// residual platform-conditional trust boundary is surfaced LOUDLY as a dispatch
// error — never silently bypassed (which would run the convert UNSANDBOXED or
// drop the finding to a silent green). Substantive: injects a sandbox seam that
// returns the platform-unavailable error and asserts the dispatch fails loud,
// naming the failure, rather than passing.
func TestSandbox_PlatformUnavailableSurfacedNotSilent(t *testing.T) {
	orig := sandboxedRunStdout
	sandboxedRunStdout = func(cmd string, args []string, packDir string, stdin []byte) ([]byte, error) {
		return nil, errors.New("sandbox unavailable on linux in this build")
	}
	t.Cleanup(func() { sandboxedRunStdout = orig })

	runner := &fixtureRunner{byCmd: map[string][]byte{"ast-grep scan": []byte(astGrepJSONStdout())}}
	_, err := dispatchPackEngines([]*pack.Manifest{convertBindingManifest()}, engineDispatchPacksDir(t), t.TempDir(), nil, runner)
	if err == nil {
		t.Fatal("an unavailable sandbox must surface a LOUD dispatch error, not a silent pass (running convert unsandboxed or dropping to green is the bug)")
	}
	if !strings.Contains(err.Error(), "sandbox unavailable") && !strings.Contains(err.Error(), "convert") {
		t.Errorf("the surfaced error must name the sandbox/convert failure, got: %v", err)
	}
}
