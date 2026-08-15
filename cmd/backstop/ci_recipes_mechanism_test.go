package main

// SPEC-067 REQ-006 / REQ-007 / REQ-008 — substitution completeness, the
// required-param refusal, and the three-way zero-baked-platform-knowledge proof.

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/check"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// TestCIRecipes_PayloadsCarryNoActionsExpressionAndOnlyDeclaredParams proves
// CLM-044, in both halves. No payload contains a `${{` expression — the
// substituter reads every `{{ ... }}` span as a param NAME and hard-errors on an
// undeclared one, so an expression-bearing payload would force the
// pass-through-param workaround REQ-006 exists to avoid. And every `{{ ... }}`
// span that DOES appear names a param that recipe declares, which is what makes
// the apply resolvable.
func TestCIRecipes_PayloadsCarryNoActionsExpressionAndOnlyDeclaredParams(t *testing.T) {
	for _, recipeID := range ciAllRecipeIDs() {
		manifest := ciRecipeManifest(t, recipeID)

		declared := map[string]bool{}
		names := []string{}
		for _, spec := range manifest.Params {
			declared[spec.Name] = true
			names = append(names, spec.Name)
		}
		sort.Strings(names)

		for _, op := range manifest.Ops {
			if op.Payload == "" {
				continue
			}
			payload := string(ciPayloadBytes(t, recipeID, op.Payload))

			if strings.Contains(payload, "${{") {
				t.Errorf("recipe %q's payload %s carries a `${{` expression; the substituter would read it as a param name and hard-error", recipeID, op.Payload)
			}
			for _, span := range ciPlaceholderSpans(payload) {
				if !declared[span] {
					t.Errorf("recipe %q's payload %s references the placeholder %q, which the recipe does not declare (it declares %v)", recipeID, op.Payload, span, names)
				}
			}
		}
	}
}

// ciPlaceholderSpans returns the trimmed inner text of every `{{ ... }}` span.
func ciPlaceholderSpans(payload string) []string {
	spans := []string{}
	rest := payload
	for {
		open := strings.Index(rest, "{{")
		if open < 0 {
			return spans
		}
		rest = rest[open+2:]
		closing := strings.Index(rest, "}}")
		if closing < 0 {
			return spans
		}
		spans = append(spans, strings.TrimSpace(rest[:closing]))
		rest = rest[closing+2:]
	}
}

// TestCIRecipes_ApplyWithoutRequiredVersionParamFailsExitOneBeforeWriting proves
// CLM-049. `backstop_version` is required with NO default, so an apply that omits
// it must fail LOUD before writing anything.
//
// THE EXIT CODE IS THE POINT. `effectiveParams` deliberately leaves a
// required-with-no-default param ABSENT from the substitution scope, so the
// failure surfaces inside `Substitute` as an ordinary op failure that
// `recipe apply` returns as &ExitCodeError{Code: ExitViolations} — EXIT 1, and
// specifically NOT the exit-2 *check.ConfigError shape reserved for malformed,
// duplicate or undeclared --param input. A test asserting exit 2 would be "fixed"
// by changing core, which this spec forbids.
func TestCIRecipes_ApplyWithoutRequiredVersionParamFailsExitOneBeforeWriting(t *testing.T) {
	const recipeID = ciRecipeGitHubActions

	manifest := ciRecipeManifest(t, recipeID)
	target := manifest.Ops[0].Target

	required := false
	for _, spec := range manifest.Params {
		if spec.Name == "backstop_version" {
			required = spec.Required && spec.Default == ""
		}
	}
	if !required {
		t.Fatalf("recipe %q no longer declares backstop_version as required-with-no-default; a defaulted version silently pins whatever release was current when the recipe was authored", recipeID)
	}

	root := ciStageConsumer(t)
	out, err := ciApply(t, root, recipeID, manifest.Version)
	if err == nil {
		t.Fatalf("the apply succeeded with no backstop_version supplied\noutput:\n%s", out)
	}

	var configErr *check.ConfigError
	if errors.As(err, &configErr) {
		t.Fatalf("the failure is a *check.ConfigError (exit 2): %v\nan unresolvable required param is an OP failure through the normal apply path, not a config refusal", configErr)
	}
	code, message := ciExitCodeOf(t, err)
	if code != ExitViolations {
		t.Errorf("exit code = %d, want %d (violations)", code, ExitViolations)
	}
	if !strings.Contains(message, "backstop_version") {
		t.Errorf("the failure message does not name the unresolved param backstop_version; got:\n%s", message)
	}

	if _, statErr := os.Stat(filepath.Join(root, filepath.FromSlash(target))); statErr == nil {
		t.Errorf("the declared target %s exists after a failed apply; the failure did not precede the write", target)
	}
}

// TestCIRecipes_CoreProductionSourceCarriesNoPlatformLiteral proves CLM-050
// (absence) — the strongest claim in SPEC-067 and the one with the one
// legitimate exception.
//
// ★ THIS TEST IS EXPECTED TO PASS THE FIRST TIME IT RUNS, and that green is
// CORRECT rather than vacuous: it is an absence claim over core source this spec
// does not touch. Measured 2026-08-11 against exactly this predicate: zero hits
// on half (i), zero on half (ii). A RED here is a NEW finding — a platform
// literal has entered core production source — and is reported as such. It is
// NEVER repaired by dropping the `github` token, by making that token match
// case-insensitively, by widening the exemption to CI-platform-shaped literals
// generally, or by editing cmd/backstop/baseline.go. All four are named
// forbidden in SPEC-067's Sharp Edge 5.
//
// THE EXEMPTION HAS TWO SPELLINGS AND A SCAN THAT KNOWS ONLY ONE IS WRONG IN A
// WAY THAT LOOKS RIGHT. `cmd/backstop/baseline.go:171` writes the same module
// path REGEX-ESCAPED inside a regexp.MustCompile pattern, so the characters
// after `github` are `\.com`, not `.com/`. Both spellings denote the same module
// path and both are exempt (spec 1.0.3, founder ruling 2026-08-11).
//
// The token is matched CASE-SENSITIVELY, which is a deliberate narrowing: the
// capitalized mentions in baseline.go (a "GitHub Actions" comment,
// ensureGitHubAuth, two error strings) are that feature's own vocabulary, were
// never what this claim measured, and neither pass nor fail it. Whether core
// should carry that knowledge at all is a separate architectural question, filed
// as its own issue rather than smuggled in here.
func TestCIRecipes_CoreProductionSourceCarriesNoPlatformLiteral(t *testing.T) {
	repoRoot := ciRepoRoot(t)
	literals := []string{
		"gitlab", "bitbucket", "jenkins", "jenkinsfile",
		".github/workflows", ".gitlab-ci", "bitbucket-pipelines",
	}

	literalHits := []string{}
	tokenHits := []string{}
	scanned := 0

	for _, subject := range []string{"pkg/recipe", "cmd/backstop"} {
		root := filepath.Join(repoRoot, filepath.FromSlash(subject))
		walkErr := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			scanned++
			source := string(data)
			lowered := strings.ToLower(source)
			rel, relErr := filepath.Rel(repoRoot, path)
			if relErr != nil {
				return relErr
			}
			relSlash := filepath.ToSlash(rel)

			// (i) the literal set, matched CASE-INSENSITIVELY.
			for _, literal := range literals {
				if strings.Contains(lowered, literal) {
					literalHits = append(literalHits, relSlash+" contains "+literal)
				}
			}

			// (ii) every occurrence of the lowercase token `github`, matched
			// CASE-SENSITIVELY, must open a module-path reference in one of its
			// TWO exempt spellings.
			for index := 0; ; {
				found := strings.Index(source[index:], "github")
				if found < 0 {
					break
				}
				at := index + found
				rest := source[at+len("github"):]
				if !strings.HasPrefix(rest, ".com/") && !strings.HasPrefix(rest, `\.com`) {
					line := strings.Count(source[:at], "\n") + 1
					tokenHits = append(tokenHits, relSlash+":"+itoa(line)+" — "+ciExcerpt(source, at))
				}
				index = at + len("github")
			}
			return nil
		})
		if walkErr != nil {
			t.Fatalf("walk %s: %v", subject, walkErr)
		}
	}

	if scanned == 0 {
		t.Fatalf("no non-test Go file was scanned; this absence claim would pass vacuously")
	}
	sort.Strings(literalHits)
	sort.Strings(tokenHits)

	if len(literalHits) != 0 {
		t.Errorf("core production source carries %d CI-platform literal(s):\n%s", len(literalHits), strings.Join(literalHits, "\n"))
	}
	if len(tokenHits) != 0 {
		t.Errorf("core production source carries %d occurrence(s) of the lowercase token `github` outside a module-path reference (neither `github.com/` nor `github\\.com`):\n%s",
			len(tokenHits), strings.Join(tokenHits, "\n"))
	}
}

// ciExcerpt quotes the neighbourhood of a hit so a failure is locatable without
// opening the file.
func ciExcerpt(source string, at int) string {
	start := at - 30
	if start < 0 {
		start = 0
	}
	end := at + 40
	if end > len(source) {
		end = len(source)
	}
	return strings.ReplaceAll(source[start:end], "\n", "\\n")
}

// itoa is a local decimal formatter, kept here so the scan above reports line
// numbers without pulling strconv into a file whose subject is string scanning.
func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	digits := []byte{}
	for value > 0 {
		digits = append([]byte{byte('0' + value%10)}, digits...)
		value /= 10
	}
	return string(digits)
}

// TestCIRecipes_AllFourPlatformsApplyThroughOneUnchangedInvocation proves
// CLM-051, the positive half of the thin-executor proof: four materially
// different CI platforms flow through ONE unchanged command, the reference
// argument being the only input that differs.
//
// The SHAPE is asserted structurally as well as by the four successes, because
// the claim is about the invocation and not merely the outcome: identical flag
// set, identical param set, one differing argument.
func TestCIRecipes_AllFourPlatformsApplyThroughOneUnchangedInvocation(t *testing.T) {
	root := ciStageConsumer(t)
	params := []string{"backstop_version=" + ciProbeVersion}

	shapes := map[string]bool{}
	for _, recipeID := range ciAllRecipeIDs() {
		manifest := ciRecipeManifest(t, recipeID)

		out, err := ciApply(t, root, recipeID, manifest.Version, params...)
		if err != nil {
			t.Fatalf("applying %q through the shared invocation failed: %v\noutput:\n%s", recipeID, err, out)
		}
		if _, statErr := os.Stat(filepath.Join(root, filepath.FromSlash(manifest.Ops[0].Target))); statErr != nil {
			t.Errorf("applying %q reported success but wrote no %s: %v", recipeID, manifest.Ops[0].Target, statErr)
		}

		// The invocation with its ONE varying input removed. Every platform must
		// collapse to the same string, which is what "one unchanged invocation"
		// means mechanically.
		shapes[strings.Join(append([]string{"recipe", "apply", "<ref>"}, ciFlagPairs(params)...), " ")] = true

		// And every recipe must declare the SAME param names, so the shared
		// invocation is not shared by accident.
		if got := ciParamNames(t, recipeID); strings.Join(got, ",") != strings.Join(ciParamNames(t, ciRecipeGitHubActions), ",") {
			t.Errorf("recipe %q declares params %v, which differ from %q's %v; the invocation is then not platform-independent",
				recipeID, got, ciRecipeGitHubActions, ciParamNames(t, ciRecipeGitHubActions))
		}
	}

	if len(shapes) != 1 {
		t.Errorf("the four applies used %d distinct invocation shapes, want exactly 1: %v", len(shapes), shapes)
	}
}

// ciFlagPairs renders the --param flags the invocation carries.
func ciFlagPairs(params []string) []string {
	rendered := []string{}
	for _, param := range params {
		rendered = append(rendered, "--param", param)
	}
	return rendered
}

// ciParamNames returns one recipe's declared param names, sorted.
func ciParamNames(t *testing.T, recipeID string) []string {
	t.Helper()

	names := []string{}
	for _, spec := range ciRecipeManifest(t, recipeID).Params {
		names = append(names, spec.Name)
	}
	sort.Strings(names)
	return names
}

// TestCIRecipes_RegisteredCommandSurfaceUnchanged proves CLM-052 (absence): this
// spec adds no core command, subcommand or flag. It is an ANTI-REGRESSION PIN and
// depends on no pack, so it is expected green from the moment it is written.
func TestCIRecipes_RegisteredCommandSurfaceUnchanged(t *testing.T) {
	root := NewRootCommand()
	// cobra registers `help` and `completion` LAZILY, on the first Execute. A
	// bare NewRootCommand().Commands() therefore reports eight commands while
	// `backstop --help` prints ten — and the claim is about the surface an
	// operator sees, so the two defaults are materialized here rather than
	// dropped from the expected set.
	root.InitDefaultHelpCmd()
	root.InitDefaultCompletionCmd()

	registered := []string{}
	for _, cmd := range root.Commands() {
		registered = append(registered, cmd.Name())
	}
	sort.Strings(registered)

	// `init` is SPEC-069's, not this spec's. This pin asserts that THE CI-RECIPES SPEC
	// added no core surface, and it does that by enumerating the whole set — so every
	// LATER spec that legitimately adds a command has to come here and say so, by name.
	// That is the pin working, not the pin being wrong: an unexplained addition still
	// fails, and the list stays an honest record of who added what.
	want := []string{
		"artifact", "baseline", "commands", "completion", "gate", "help",
		"init", "pack", "recipe", "version", "waiver",
	}
	if strings.Join(registered, ",") != strings.Join(want, ",") {
		t.Errorf("the registered top-level command set is %v, want exactly %v", registered, want)
	}

	for _, cmd := range root.Commands() {
		if cmd.Name() != "recipe" {
			continue
		}
		subcommands := []string{}
		for _, sub := range cmd.Commands() {
			subcommands = append(subcommands, sub.Name())
		}
		sort.Strings(subcommands)
		if strings.Join(subcommands, ",") != "apply" {
			t.Errorf("the `recipe` namespace registers %v, want exactly [apply]", subcommands)
		}
	}

	banned := []string{"ci", "workflow", "github", "gitlab", "bitbucket", "jenkins"}
	ciWalkCommands(root, func(cmd *cobra.Command) {
		for _, token := range banned {
			if ciNamesToken(cmd.Name(), token) {
				t.Errorf("command %q names the platform token %q; this spec adds no platform-named surface", cmd.CommandPath(), token)
			}
		}
		cmd.Flags().VisitAll(func(flag *pflag.Flag) {
			for _, token := range banned {
				if ciSanctionedTokenFlag(cmd.CommandPath(), flag.Name, token) {
					continue
				}
				if ciNamesToken(flag.Name, token) {
					t.Errorf("command %q declares flag --%s, which names the platform token %q", cmd.CommandPath(), flag.Name, token)
				}
			}
		})
	})
}

// ciNamesToken reports whether an identifier NAMES a token.
//
// The two-armed test is deliberate. A short token like `ci` must be matched as a
// whole hyphen- or underscore-delimited SEGMENT: as a bare substring it fires on
// `recipe`, which is the shipped SPEC-054 namespace and has nothing to do with
// CI — a false positive that would make this claim un-passable without deleting
// a real command. The long platform names carry no such collision, so a
// substring match is both safe and stricter for them.
func ciNamesToken(identifier string, token string) bool {
	lowered := strings.ToLower(identifier)
	for _, segment := range strings.FieldsFunc(lowered, func(r rune) bool {
		return r == '-' || r == '_' || r == '.'
	}) {
		if segment == token {
			return true
		}
	}
	return len(token) > 3 && strings.Contains(lowered, token)
}

// ciWalkCommands visits every command in the assembled tree.
func ciWalkCommands(cmd *cobra.Command, visit func(*cobra.Command)) {
	visit(cmd)
	for _, sub := range cmd.Commands() {
		ciWalkCommands(sub, visit)
	}
}

// ciSanctionedTokenFlag reports whether a flag naming a banned token is one a LATER
// spec explicitly mandates.
//
// EXACTLY ONE ENTRY, AND THE DISTINCTION IS PRINCIPLED RATHER THAN AN EXEMPTION OF
// CONVENIENCE. `github`, `gitlab`, `bitbucket`, `jenkins` and `workflow` are PLATFORM
// names — core holding one is the bake this pin exists to prevent, and none is
// sanctioned here or anywhere. `ci` is different in kind: on `backstop init` it names
// backstop's OWN step, the one governed solely by that flag's presence, and its VALUE
// is a whole pinned <pack>:<recipe>@<version> ref that core never inspects. SPEC-069
// REQ-016 mandates that flag by name.
//
// The check is keyed on the exact command path AND the exact flag name, so the
// exemption cannot spread: a --ci flag on any other command, or a --github flag on
// init, still fails.
func ciSanctionedTokenFlag(commandPath, flagName, token string) bool {
	return token == "ci" && commandPath == "backstop init" && flagName == "ci"
}

// TestCIRecipes_TheSanctionedFlagExemptionIsExactlyOnePair keeps the exemption above
// from becoming a hole.
//
// An exemption inside an anti-regression pin is the classic way a pin quietly stops
// pinning: it is added narrowly for one real reason and then widens, one plausible
// case at a time, until the ban it guards is decorative. So the narrowness is asserted
// here rather than asserted in a comment — the exemption admits EXACTLY the one pair a
// later spec mandates, and every near-miss around it is still refused.
func TestCIRecipes_TheSanctionedFlagExemptionIsExactlyOnePair(t *testing.T) {
	if !ciSanctionedTokenFlag("backstop init", "ci", "ci") {
		t.Fatal("the one flag SPEC-069 REQ-016 mandates is not sanctioned, so `backstop init --ci` cannot exist")
	}

	refused := []struct {
		commandPath, flagName, token, why string
	}{
		{"backstop init", "github", "github", "a PLATFORM name on init is the exact bake this pin exists to prevent"},
		{"backstop init", "gitlab", "gitlab", "likewise"},
		{"backstop init", "jenkins", "jenkins", "likewise"},
		{"backstop init", "workflow", "workflow", "likewise"},
		{"backstop recipe apply", "ci", "ci", "the exemption is keyed on init ALONE; a --ci elsewhere is a second entry point"},
		{"backstop gate", "ci", "ci", "likewise"},
		{"backstop init ci", "ci", "ci", "a nested `ci` VERB under init is not the flag that was sanctioned"},
	}
	for _, tc := range refused {
		if ciSanctionedTokenFlag(tc.commandPath, tc.flagName, tc.token) {
			t.Fatalf("the exemption admits %q --%s (token %q), which it must not: %s",
				tc.commandPath, tc.flagName, tc.token, tc.why)
		}
	}
}
