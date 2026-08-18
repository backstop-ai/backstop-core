package engine_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack"
	"github.com/backstop-ai/backstop-core/pkg/pack/distribution"
)

// contracts_installed_pack_polarity_test.go (PLAN-ISSUE-157, CLM-002).
//
// WHY THIS FILE EXISTS. The RELEASED `backstop-ai/go-contracts` mirror carried
// INVERTED positive/negative fixture polarity on its six pattern-arg signature
// rules: the CLEAN slot pointed at a file that fires the rule and the VIOLATING
// slot at one that does not. That makes the pack fail its own phase3-fixtures
// validation and therefore UNINSTALLABLE via `pack add`/`pack update`/`pack
// upgrade`. A verdict measured once in a terminal is not a durable pin, so this
// test asserts the property of the INSTALLED pack from core, the same way
// grep_installed_pack_test.go pins ISSUE-166's fix.
//
// It is in `engine_test` for the same reason that file is: it resolves the pack
// through the PRODUCTION readers (`pack.ParseManifestFile`,
// `distribution.ReadLockfile`), and `pkg/pack` imports `pkg/pack/engine`, so an
// in-package test importing them is an import cycle.
//
// It REUSES this package's existing `conventionRepoRoot`, `installedContractsPackName`
// and `semverGreater`; redeclaring any of them is a compile error.

// prePolarityFixContractsPackVersion is the version of the mirror that carried the
// INVERTED FIXTURE POLARITY. It is deliberately NOT `preFixContractsPackVersion`
// (1.2.0, the ISSUE-166 grep-filename bar): 1.3.0 already clears that bar while
// still carrying this defect, so binding this leg to it would read green for the
// wrong reason.
const prePolarityFixContractsPackVersion = "1.3.0"

// patternArgContractsEngine is the engine whose rules this test owns. The
// discriminator is the ENGINE, not "declares a pattern": `contract-absence`
// declares a pattern too, but it is a grep rule, is already correctly polarized,
// and is not part of this defect.
const patternArgContractsEngine = "ast-grep-contracts"

// patternArgSignatureRuleCount is how many `ast-grep-contracts` rules the pack
// declares. Asserting it stops a future rule addition (or removal) from silently
// shrinking the sweep to a vacuous pass.
const patternArgSignatureRuleCount = 6

// TestInstalledGoContractsPack_FixturePolarityIsCorrect (CLM-002) asserts the
// INSTALLED pack's fixture polarity is right, in three legs split by what each can
// prove where:
//   - LEG 1, THE LOCK, reads TRACKED data, so it holds in a fresh checkout where the
//     pack fleet is not installed — it is what keeps this test non-vacuous there.
//   - LEG 2, THE SLOTS, and LEG 3, THE POLARITY, read the INSTALLED tree, so they
//     follow the established guard: a load ERROR fails, a genuinely-absent fleet
//     skips with a `pack install` directive.
func TestInstalledGoContractsPack_FixturePolarityIsCorrect(t *testing.T) {
	root := conventionRepoRoot(t)

	// ── LEG 1: THE LOCK (tracked data — never skipped) ──────────────────────────
	lockPath := filepath.Join(root, "backstop.lock")
	lockfile, err := distribution.ReadLockfile(lockPath)
	if err != nil {
		// A tracked lockfile that will not read is a broken repo, never a skip.
		t.Fatalf("ReadLockfile(%s): %v", lockPath, err)
	}
	entry, ok := lockfile.Packs[installedContractsPackName]
	if !ok {
		t.Fatalf("backstop.lock has no entry for %s; core's contracts gate rides that pack",
			installedContractsPackName)
	}
	if !semverGreater(t, entry.Version, prePolarityFixContractsPackVersion) {
		t.Errorf("backstop.lock pins %s at %s, which is not greater than %s — the version "+
			"that carried the INVERTED fixture polarity. Core is still locked to a mirror "+
			"whose six pattern-arg signature rules point their CLEAN slot at a file that "+
			"fires the rule, which is why that pack fails its own phase3-fixtures validation",
			installedContractsPackName, entry.Version, prePolarityFixContractsPackVersion)
	}
	t.Logf("backstop.lock: %s version=%s source_type=%s",
		installedContractsPackName, entry.Version, entry.SourceType)

	// ── The installed tree, behind the established guard ────────────────────────
	packRoot := filepath.Join(root, ".backstop", "packs", installedContractsPackName)
	manifestPath := filepath.Join(packRoot, "pack.yml")
	manifest, err := pack.ParseManifestFile(manifestPath)
	if err != nil {
		// Distinguish "not installed" from "installed and broken". Only the former
		// skips; a manifest that exists and will not parse is a real failure.
		if _, statErr := os.Stat(manifestPath); statErr != nil {
			t.Skipf("%s is not installed — run `./bin/backstop pack install` (the pack fleet is not installed)",
				installedContractsPackName)
		}
		t.Fatalf("ParseManifestFile(%s): %v", manifestPath, err)
	}

	// THE INVOCATION COMES FROM THE PACK, NOT FROM THIS TEST. The tool, its base
	// arguments and its input flag are read out of the manifest's own engines: block,
	// so this test drives the rules the way the real dispatch does and bakes no tool
	// name into core. The language is inferred from the fixture's .go extension,
	// exactly as the pack's engines: comment says the dispatch relies on.
	spec, ok := manifest.Engines[patternArgContractsEngine]
	if !ok {
		t.Fatalf("installed %s declares no %s engine; the six signature rules ride it",
			installedContractsPackName, patternArgContractsEngine)
	}
	commandFields := strings.Fields(spec.Command)
	if len(commandFields) == 0 {
		t.Fatalf("installed %s engine %s declares an empty command", installedContractsPackName,
			patternArgContractsEngine)
	}
	if spec.InputFlag == "" {
		t.Fatalf("installed %s engine %s declares no input_flag; a pattern-arg engine feeds "+
			"its pattern through that flag", installedContractsPackName, patternArgContractsEngine)
	}
	invocation := patternArgInvocation{
		tool:     commandFields[0],
		baseArgs: commandFields[1:],
		flag:     spec.InputFlag,
	}

	// ── LEG 2: THE SLOTS ────────────────────────────────────────────────────────
	type polarityCase struct {
		ruleID   string
		pattern  string
		positive string
		negative string
	}
	var cases []polarityCase
	for _, rule := range manifest.Content.Ruleset.Rules {
		if rule.Engine != patternArgContractsEngine {
			// `contract-absence` lands here: a grep rule, already correctly
			// polarized, and not part of this defect. Excluded deliberately.
			continue
		}
		if rule.Pattern == "" {
			t.Errorf("installed %s rule %s declares engine %s but no pattern; a pattern-arg "+
				"rule with no pattern cannot be polarity-checked at all",
				installedContractsPackName, rule.ID, patternArgContractsEngine)
			continue
		}
		var positives, negatives []string
		for _, claim := range rule.Claims {
			for _, fixture := range claim.Fixtures.Positive {
				positives = append(positives, fixture.Path)
			}
			for _, fixture := range claim.Fixtures.Negative {
				negatives = append(negatives, fixture.Path)
			}
		}
		if len(positives) == 0 || len(negatives) == 0 {
			t.Errorf("installed %s rule %s declares %d positive and %d negative fixtures; "+
				"both slots must be populated or the rule's polarity is unverifiable",
				installedContractsPackName, rule.ID, len(positives), len(negatives))
			continue
		}
		for _, positive := range positives {
			for _, negative := range negatives {
				if positive == negative {
					t.Errorf("installed %s rule %s declares the SAME path %q as BOTH its "+
						"positive and its negative fixture; one file cannot be both the "+
						"clean case and the violating case",
						installedContractsPackName, rule.ID, positive)
				}
			}
			for _, negative := range negatives {
				cases = append(cases, polarityCase{
					ruleID:   rule.ID,
					pattern:  rule.Pattern,
					positive: positive,
					negative: negative,
				})
			}
		}
	}
	seen := map[string]bool{}
	for _, c := range cases {
		seen[c.ruleID] = true
	}
	if len(seen) != patternArgSignatureRuleCount {
		t.Fatalf("installed %s declares %d usable %s rules, want %d — the sweep this test "+
			"performs is defined by that set, so a change in it must be deliberate",
			installedContractsPackName, len(seen), patternArgContractsEngine,
			patternArgSignatureRuleCount)
	}

	// ── LEG 3: THE POLARITY, BY REAL MATCH COUNT ────────────────────────────────
	// BUNDLE-005 REQ-011: a POSITIVE fixture is the CLEAN case the rule must NOT
	// fire on; a NEGATIVE fixture is the VIOLATING case it MUST fire on.
	for _, c := range cases {
		positivePath := filepath.Join(packRoot, filepath.FromSlash(c.positive))
		negativePath := filepath.Join(packRoot, filepath.FromSlash(c.negative))

		positiveMatches := patternArgMatchCount(t, invocation, c.pattern, positivePath)
		negativeMatches := patternArgMatchCount(t, invocation, c.pattern, negativePath)

		if positiveMatches != 0 {
			t.Errorf("installed %s rule %s: POSITIVE (clean) fixture %s yields %d %s "+
				"matches for its own pattern %q, want 0 — the clean case must NOT fire the "+
				"rule. This is the inverted polarity that makes the pack fail its own "+
				"phase3-fixtures validation (negative fixture %s yields %d)",
				installedContractsPackName, c.ruleID, c.positive, positiveMatches,
				invocation.tool, c.pattern, c.negative, negativeMatches)
		}
		if negativeMatches < 1 {
			t.Errorf("installed %s rule %s: NEGATIVE (violating) fixture %s yields %d %s "+
				"matches for its own pattern %q, want at least 1 — the violating "+
				"case MUST fire the rule (positive fixture %s yields %d)",
				installedContractsPackName, c.ruleID, c.negative, negativeMatches,
				invocation.tool, c.pattern, c.positive, positiveMatches)
		}
		t.Logf("%s: pattern=%q positive %s=%d negative %s=%d",
			c.ruleID, c.pattern, c.positive, positiveMatches, c.negative, negativeMatches)
	}
}

// patternArgInvocation is one pattern-arg engine's PACK-DECLARED invocation: the
// tool, its base arguments and the flag its pattern rides on, all read from the
// manifest's engines: block. Core bakes no tool name of its own — the same reason
// the production dispatch reads these fields instead of hard-coding them.
type patternArgInvocation struct {
	tool     string
	baseArgs []string
	flag     string
}

// patternArgMatchCount runs the pack's REAL declared tool with the rule's own
// declared pattern over one fixture and returns the number of matches.
// `astGrepMatchCount` in contracts_go_rules_test.go lives in `package engine` and is
// not reachable from `engine_test`, so this local helper stands in; it needs no
// convert because the count, not the SARIF shape, is what this test asserts.
//
// Real tool required — a missing tool is a Fatalf, never a Skip, following the
// convention this package already sets.
//
// THE EXIT CODE IS NOT THE SIGNAL, STDOUT IS. `ast-grep run` exits 1 on a ZERO-match
// run while still writing a well-formed `[]` to stdout, and a zero-match run is the
// EXPECTED result for every clean fixture here. So the exit code is ignored (the same
// call `runEngineStdout` makes for findings engines) and the JSON is what must parse —
// unparseable stdout is the real failure, and it is reported loudly with stderr.
func patternArgMatchCount(t *testing.T, invocation patternArgInvocation, pattern, file string) int {
	t.Helper()
	if _, err := exec.LookPath(invocation.tool); err != nil {
		t.Fatalf("the pack-declared tool %q is required (no t.Skip): %v", invocation.tool, err)
	}
	if _, err := os.Stat(file); err != nil {
		t.Fatalf("declared fixture %s is not on disk in the installed pack: %v", file, err)
	}
	args := append(append([]string{}, invocation.baseArgs...), invocation.flag, pattern, file)
	cmd := exec.Command(invocation.tool, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	_ = cmd.Run() // exit 1 == zero matches; stdout is the contract.
	var matches []json.RawMessage
	if err := json.Unmarshal(stdout.Bytes(), &matches); err != nil {
		t.Fatalf("ast-grep JSON for pattern %q over %s is not an array: %v\nstdout: %s\nstderr: %s",
			pattern, file, err, stdout.String(), stderr.String())
	}
	return len(matches)
}
