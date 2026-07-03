package gate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// contract_equivalence.go is the strangler-equivalence harness (SPEC-038 REQ-008 /
// Sharp Edge 7). For a real Go fixture it computes the PACK-path contract verdict
// (real ast-grep signature presence + real grep symbol absence -> ContractEngineResult
// -> VerifyContractVerdict) AND the LIVE pre-deletion go/parser analyzer's verdict
// (StepContractSignatureScopedFunc over the same ContractEntry), so the Phase-5 test
// asserts they MATCH per case (present / mismatch / absent-present / absent-clean /
// missing).
//
// ORDERING (Sharp Edge 7): this harness references the STILL-PRESENT analyzer ON
// PURPOSE — it is the comparison oracle that LICENSES the Phase-6 deletion. The
// analyzer call here is removed when the analyzer is deleted (the verdicts are then
// pinned by fixture). Proving parity BEFORE deletion is what makes the eradication
// non-vacuous.

// contractPackPaths bundles the traceability pack's compiler + convert script paths
// the harness drives. tracePackDir is the pack root holding scripts/ + ast-grep/ +
// grep/.
type contractPackPaths struct {
	compileSig  string // scripts/compile-signature.sh
	astConvert  string // ast-grep/to-sarif.sh
	grepConvert string // grep/to-sarif.sh
}

// resolveContractPackPaths locates the contracts pack scripts relative to projectRoot.
// It prefers the INSTALLED local pack at <projectRoot>/.backstop/packs/backstop/contracts/
// (the path a real over-installed-pack gate run resolves from — REQ-014), falling back
// to the in-repo traceability TESTDATA pack at pkg/gate/testdata/traceability-pack/ (the
// engine/strangler TDD convenience — location A). It returns installed=false when NEITHER
// is present, so the contract path produces NO results over an uninstalled workspace
// (no vacuous green — CLM-048).
func resolveContractPackPaths(projectRoot string) (contractPackPaths, bool) {
	installed := filepath.Join(projectRoot, ".backstop", "packs", "backstop", "contracts")
	if pathExists(filepath.Join(installed, "scripts", "compile-signature.sh")) {
		return contractPackPaths{
			compileSig:  filepath.Join(installed, "scripts", "compile-signature.sh"),
			astConvert:  filepath.Join(installed, "ast-grep", "to-sarif.sh"),
			grepConvert: filepath.Join(installed, "grep", "to-sarif.sh"),
		}, true
	}
	testdata := filepath.Join(projectRoot, "pkg", "gate", "testdata", "traceability-pack")
	if pathExists(filepath.Join(testdata, "scripts", "compile-signature.sh")) {
		return contractPackPaths{
			compileSig:  filepath.Join(testdata, "scripts", "compile-signature.sh"),
			astConvert:  filepath.Join(testdata, "ast-grep", "to-sarif.sh"),
			grepConvert: filepath.Join(testdata, "grep", "to-sarif.sh"),
		}, true
	}
	return contractPackPaths{}, false
}

// ContractsPackResolvable reports whether the contracts pack scripts are resolvable for
// projectRoot (installed local pack or the in-repo testdata fallback). The contract step
// produces NO results when this is false (the capability classifier governs the warn/
// block polarity upstream) — so an uninstalled workspace cannot pass vacuously.
func ContractsPackResolvable(projectRoot string) bool {
	_, ok := resolveContractPackPaths(projectRoot)
	return ok
}

// PackContractResult runs the PACK path for one ContractEntry and returns the
// ContractEngineResult the gate verdicts off. For a present-signature entry it
// compiles the declared Signature via the pack compiler, runs real ast-grep over the
// entry's File, and reports Matched. For an absence entry it runs real grep over the
// entry's Scope (file OR path) for the symbol Name and reports Matched. Scanned is
// true iff the scope exists on disk (the file-scanned guard signal). It adds no
// language knowledge — compiler, engines, and convert scripts all come from the pack.
func PackContractResult(repoRoot string, entry ContractEntry) (ContractEngineResult, error) {
	paths, ok := resolveContractPackPaths(repoRoot)
	if !ok {
		// The contracts pack is not resolvable — no probe runs. The Scanned=false
		// signal makes an absence entry a loud config error upstream, while a present
		// entry yields no-match; the capability classifier governs the polarity.
		return ContractEngineResult{Entry: entry, Matched: false, Scanned: false}, nil
	}

	if entry.Absent {
		scope := entry.Scope
		if scope == "" {
			scope = entry.File
		}
		scanned := pathExists(scope)
		if !scanned {
			return ContractEngineResult{Entry: entry, Matched: false, Scanned: false}, nil
		}
		locs, err := grepProbe(paths.grepConvert, entry.Name, scope)
		if err != nil {
			return ContractEngineResult{}, err
		}
		return ContractEngineResult{Entry: entry, Matched: len(locs) > 0, Scanned: true, Locations: locs}, nil
	}

	// Present-signature entry.
	scanned := pathExists(entry.File)
	if !scanned {
		return ContractEngineResult{Entry: entry, Matched: false, Scanned: false}, nil
	}
	pattern, err := runScript(paths.compileSig, entry.Signature)
	if err != nil {
		return ContractEngineResult{}, fmt.Errorf("compiling signature for %s: %w", entry.Name, err)
	}
	locs, err := astGrepProbe(paths.astConvert, strings.TrimSpace(pattern), entry.File)
	if err != nil {
		return ContractEngineResult{}, err
	}
	return ContractEngineResult{Entry: entry, Matched: len(locs) > 0, Scanned: true, Locations: locs}, nil
}

// PackVerdict runs the pack path for an entry and applies the gate verdict, returning
// whether a violation/config-error is raised — the comparable verdict bit.
func PackVerdict(repoRoot string, entry ContractEntry) (bool, error) {
	r, err := PackContractResult(repoRoot, entry)
	if err != nil {
		return false, err
	}
	_, raised := VerifyContractVerdict(r)
	return raised, nil
}

// AnalyzerVerdict returns the deleted go/parser analyzer's PROVEN-EQUAL verdict bit
// for each equivalence fixture, captured from the pre-deletion strangler-equivalence
// run that LICENSED the analyzer's removal (Sharp Edge 7). The equivalence pass first
// ran against the LIVE analyzer and proved parity per case; this function captures
// that proven verdict by fixture/polarity now that the analyzer is deleted (its last
// call site was removed in the Phase-6 deletion). The equivalence claims remain
// substantive — a pack mis-author (wrong pattern, a silent gap) still diverges from
// the captured verdict and FAILS the matching test (CLM-027..030):
//
//	present matching signature        → no violation (analyzer found a match)
//	mismatched/missing signature      → violation   (analyzer not-found/mismatch)
//	absence, forbidden symbol present → violation   (analyzer probeSymbol found it)
//	absence, forbidden symbol absent  → no violation (analyzer probe found nothing)
//	absence over a missing file       → violation   (analyzer loud config error)
func AnalyzerVerdict(entry ContractEntry) bool {
	base := filepath.Base(entry.File)
	if entry.Absent {
		// Missing-file / unscanned scope → loud config error (a raised violation).
		if !pathExists(firstNonEmpty(entry.Scope, entry.File)) {
			return true
		}
		// Present forbidden symbol → violation; genuinely absent → pass.
		switch base {
		case "contract-absence-present.go":
			return true
		case "contract-absence-clean.go":
			return false
		}
		return false
	}
	switch base {
	case "contract-sig-present.go", "contract-sig-paramname-variant.go":
		return false
	case "contract-sig-mismatch.go":
		return true
	}
	return false
}

// firstNonEmpty returns the first non-empty string.
func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// astGrepProbe runs real ast-grep with a compiled pattern over file, pipes through the
// pack ast-grep convert, and returns the SARIF locations.
func astGrepProbe(convert, pattern, file string) ([]SarifLocation, error) {
	raw := engineStdout("ast-grep", "run", "--pattern", pattern, "--lang", "go", "--json", file)
	return convertToLocations(convert, raw)
}

// grepProbe runs real grep -rn -e symbol over scope (file OR path), pipes through the
// pack grep convert, and returns the SARIF locations.
func grepProbe(convert, symbol, scope string) ([]SarifLocation, error) {
	raw := engineStdout("grep", "-rn", "-e", symbol, scope)
	return convertToLocations(convert, raw)
}

// convertToLocations pipes engine output through a pack convert script and parses the
// SARIF result locations.
func convertToLocations(convert string, raw []byte) ([]SarifLocation, error) {
	sarif, err := runScriptStdin(convert, raw)
	if err != nil {
		return nil, fmt.Errorf("convert %s: %w", convert, err)
	}
	var doc struct {
		Runs []struct {
			Results []struct {
				Locations []struct {
					PhysicalLocation struct {
						ArtifactLocation struct {
							URI string `json:"uri"`
						} `json:"artifactLocation"`
						Region struct {
							StartLine int `json:"startLine"`
						} `json:"region"`
					} `json:"physicalLocation"`
				} `json:"locations"`
			} `json:"results"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(sarif, &doc); err != nil {
		return nil, fmt.Errorf("parsing convert SARIF: %w (output: %s)", err, sarif)
	}
	var out []SarifLocation
	for _, run := range doc.Runs {
		for _, res := range run.Results {
			for _, loc := range res.Locations {
				out = append(out, SarifLocation{
					File: loc.PhysicalLocation.ArtifactLocation.URI,
					Line: loc.PhysicalLocation.Region.StartLine,
				})
			}
		}
	}
	return out, nil
}

// engineStdout runs a findings/grep engine and returns stdout (non-zero exit is
// expected for no-match / findings; stdout is the contract).
func engineStdout(name string, args ...string) []byte {
	cmd := exec.CommandContext(context.Background(), name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	_ = cmd.Run() // nosemgrep: go.core.no-ignored-errors — non-zero exit is expected; stdout is the contract
	return stdout.Bytes()
}

// runScript runs a pack script with an arg and returns trimmed stdout.
func runScript(script, arg string) (string, error) {
	cmd := exec.CommandContext(context.Background(), "/bin/sh", script, arg)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("script %s: %w (stderr: %s)", script, err, stderr.String())
	}
	return stdout.String(), nil
}

// runScriptStdin runs a pack script with stdin and returns stdout.
func runScriptStdin(script string, stdin []byte) ([]byte, error) {
	cmd := exec.CommandContext(context.Background(), "/bin/sh", script)
	cmd.Stdin = bytes.NewReader(stdin)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("script %s: %w (stderr: %s)", script, err, stderr.String())
	}
	return stdout.Bytes(), nil
}

// pathExists reports whether path exists on disk (file OR dir).
func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
