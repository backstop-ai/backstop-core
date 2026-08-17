package packval

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// ── CROSS-CUTTING GUARDS (ISSUE-160) ────────────────────────────────────────
// Both guards here cover ALL THREE fields this lane taught RunEngine to honor
// (Producer, CrashGuard, StrictSarif), which is why they live in their own file
// rather than in any one cycle's. Both reuse readGuardSource and windowFunc from
// executor_convert_test.go rather than re-declaring them.

// TestPackVal_EngineGateParity_ExecutorCarriesNoToolLiteral makes tool-blindness
// MECHANICAL (CLM-015). The strict-SARIF refusal in particular must not carry
// the gate's tool-naming wording: the gate's message is the exact text an
// implementer reads while writing that stage, and copying it would bake a tool
// name into the generic pack-validation dispatch.
//
// SCOPED TO THE WHOLE FILE, not to a function window: this is a FILE-LEVEL
// property, and a literal smuggled into a helper or a comment is exactly as much
// of a violation as one inside RunEngine.
//
// THIS STRENGTHENS — IT DOES NOT REPLACE —
// TestExecutor_ToolConfigResolvesViaBindingNotSwitch in engine_dispatch_test.go.
// That guard scans executor.go for three exact code patterns AND additionally
// scans phase3.go, a file this one does not read at all. This guard is BROADER
// on executor.go (more literals, case-insensitive, whole-file) and NARROWER in
// file coverage, so the two overlap without either subsuming the other. Deleting
// the older one as a redundant duplicate would silently drop phase3.go's
// tool-literal coverage.
//
// Measured against executor.go at HEAD 958b7b0 before this guard was written:
// zero occurrences of every listed literal. No pre-existing finding was trimmed
// out of the list to make it pass.
func TestPackVal_EngineGateParity_ExecutorCarriesNoToolLiteral(t *testing.T) {
	// Declared LOCALLY, not at package level: a package-level var is mutable
	// global state the go-standards pack blocks on sight, and nothing outside this
	// guard reads either value.
	//
	// THE BARE TOKEN `go` IS DELIBERATELY ABSENT from the list. It occurs as an
	// ordinary English word and inside identifiers throughout any Go source file,
	// so listing it would make this guard fail on contact and teach the next reader
	// to weaken the list rather than fix a violation. `golang` as a STANDALONE WORD
	// is the checkable form, and it is matched with word boundaries for that reason.
	toolLiterals := []string{
		"golangci",
		"semgrep",
		"ast-grep",
		"gofmt",
		"eslint",
		"oxlint",
		"tsc",
		"bun",
	}
	standaloneGolang := regexp.MustCompile(`(?i)\bgolang\b`)

	src := readGuardSource(t, filepath.Join("executor.go"))
	lower := strings.ToLower(src)

	const principle = "backstop is a THIN EXECUTOR with ZERO baked language/tool knowledge (CLAUDE.md's first " +
		"principle): every check and every toolchain comes from a PACK, and pkg/packval/executor.go is the GENERIC " +
		"pack-validation dispatch. A tool or language literal here is a defect to eradicate, not to waive. If this " +
		"fired while mirroring a gate-side message, describe the SHAPE and the CONSEQUENCE instead and name the " +
		"engine via binding.Command"

	for _, literal := range toolLiterals {
		if strings.Contains(lower, literal) {
			t.Fatalf("executor.go contains the baked tool literal %q: %s", literal, principle)
		}
	}
	if loc := standaloneGolang.FindString(src); loc != "" {
		t.Fatalf("executor.go contains the baked language literal %q: %s", loc, principle)
	}
}

// gateParityGuardSite is one honoring site the drift guard windows BY FUNCTION
// NAME, together with the field references that site must carry.
type gateParityGuardSite struct {
	name   string
	body   string
	tokens []string
}

// TestPackVal_EngineGateParity_NoDriftFromGateDispatch is the mechanical drift
// guard covering all three fields on both sides (CLM-016) — the Producer/
// CrashGuard/StrictSarif twin of this package's existing convert and
// stdout_artifact drift guards.
//
// After this lane, TWO dispatch paths independently honor these fields.
// Consolidating them into one authority is ISSUE-143's subject; it is open and
// unplanned, and this guard is what holds the line until it is picked up.
//
// IT IS A CONTENT SCAN, NOT A TREE-STATE CHECK: no git status, no git diff, no
// working-tree-cleanliness assertion and no line numbers. This is a shared tree
// with concurrent lanes, and a tree-state assertion blames whoever runs it.
//
// WINDOWING IS WHAT MAKES IT NON-VACUOUS, NOT STYLE. cmd/backstop/pack_gate.go
// contains TWO independent producer blocks — one in runCoverageEngine and one in
// runFindingsEngine — so a whole-file scan for binding.Producer COULD NOT FAIL
// even if runFindingsEngine's block were deleted outright. Window C exists for a
// different reason: runFindingsEngine holds only the CALL to
// requireLintSarifShape, while the binding.StrictSarif flag check itself lives in
// a second file.
//
// cmd/backstop/pack_gate.go and cmd/backstop/pack_gate_golint.go are READ-ONLY
// in this lane and are not in any task's file scope; the guard only reads them.
func TestPackVal_EngineGateParity_NoDriftFromGateDispatch(t *testing.T) {
	executorSrc := readGuardSource(t, filepath.Join("executor.go"))
	gateSrc := readGuardSource(t, filepath.Join("..", "..", "cmd", "backstop", "pack_gate.go"))
	golintSrc := readGuardSource(t, filepath.Join("..", "..", "cmd", "backstop", "pack_gate_golint.go"))

	sites := []gateParityGuardSite{
		{
			name:   "pkg/packval/executor.go RunEngine",
			body:   windowFunc(t, executorSrc, "func (d *DefaultExecutor) RunEngine("),
			tokens: []string{"binding.Producer", "binding.CrashGuard", "binding.StrictSarif"},
		},
		{
			name:   "cmd/backstop/pack_gate.go runFindingsEngine",
			body:   windowFunc(t, gateSrc, "func runFindingsEngine("),
			tokens: []string{"binding.Producer", "binding.CrashGuard", "requireLintSarifShape"},
		},
		{
			name:   "cmd/backstop/pack_gate_golint.go requireLintSarifShape",
			body:   windowFunc(t, golintSrc, "func requireLintSarifShape("),
			tokens: []string{"binding.StrictSarif"},
		},
	}

	const drift = "the two dispatch paths have DRIFTED. Both must honor the binding's declared producer " +
		"(invoking the resolved script in place of the command), crash guard (a non-zero exit with zero parseable " +
		"findings is a crash, not a clean pass) and strict-SARIF shape check (non-SARIF output from a declaring " +
		"binding fails loud instead of parsing to zero findings). A field honored on one path and dropped on the " +
		"other is a validator that lies about the very run the gate would refuse. Consolidating them into one " +
		"authority is ISSUE-143, which is open and unplanned; until it lands, this guard is what holds the line"

	for _, site := range sites {
		for _, tok := range site.tokens {
			if !strings.Contains(site.body, tok) {
				t.Fatalf("%s does not reference %s: %s", site.name, tok, drift)
			}
		}
	}
}
