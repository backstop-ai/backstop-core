package main

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/bmanson/backstop-core/pkg/pack"
	"github.com/bmanson/backstop-core/pkg/pack/engine"
)

// The Go lint pass runs golangci-lint as a CONFIG-FILE engine on golangci-lint
// v2's NATIVE SARIF (SPEC-034 REQ-005). The wiring is data — the `golangci`
// EngineBinding (pkg/pack/engine/binding.go) — dispatched by the existing
// runFindingsEngine: its stdout is captured via the clean-stdout RunStdout runner
// (NOT CombinedOutput, so a golangci stderr banner cannot corrupt the SARIF —
// CLM-016, Sharp Edge 4), it declares NO Convert (v2 emits SARIF directly), and
// it parses straight through parseSarif (check.ParsePackFindings). The invocation
// pins v2 SARIF: `golangci-lint run --output.sarif.path stdout` with no
// `golangci-lint version` probe and no v1/v2 flag branch (CLM-019). The
// pack-owned .golangci.yml is the optional config-file input (OQ-1: pack-defined).
//
// This file owns the lint engine's ONE behavioral guard the generic findings
// path does not provide: the strict-SARIF shape check (Sharp Edge 5). golangci v2
// emits SARIF, but a v1/too-old binary emits its own JSON, which the lenient
// SARIF parser would unmarshal into zero findings and read as a silent green. The
// guard rejects non-SARIF lint output as a fail-loud, engine-attributed error.

// requireLintSarifShape fail-louds when the native-SARIF lint engine's stdout is
// not a well-formed SARIF log (no `runs` array). It is the v1/too-old golangci
// guard (SPEC-034 REQ-005/CLM-019, Sharp Edge 5): without it, golangci v1 JSON
// (`{"Issues":[...]}`) unmarshals into the lenient SARIF parser as ZERO findings
// and the lint pass reads green on the wrong tool — vacuous green. The error
// names the pack and the engine so the failure is attributable, never silent.
//
// Empty stdout is NOT treated as malformed here: a genuinely clean lint run can
// emit an empty/whitespace document; the crash-vs-findings discipline for an
// unexpected empty result belongs to the runErr path, not the SARIF-shape check.
func requireLintSarifShape(manifest *pack.Manifest, binding engine.EngineBinding, stdout []byte) error {
	// The guard fires off the binding's DECLARED StrictSarif flag (REQ-006a/
	// CLM-023), NOT a "golangci-lint" command-prefix sniff: a pack declares which
	// of its engines emit strict native SARIF, so the shape guard rides the
	// declaration and no tool-name literal drives this control flow (Sharp Edge 5).
	if !binding.StrictSarif {
		return nil
	}
	trimmed := bytes.TrimSpace(stdout)
	if len(trimmed) == 0 {
		return nil
	}
	var probe struct {
		Runs *json.RawMessage `json:"runs"`
	}
	if err := json.Unmarshal(trimmed, &probe); err != nil || probe.Runs == nil {
		return fmt.Errorf(
			"pack %s engine %q: output is not golangci-lint v2 SARIF (no `runs` array) — a v1/too-old golangci-lint would silently read as zero findings; upgrade to v2 or fix the lint config",
			manifest.NormalizedName, binding.Command,
		)
	}
	return nil
}
