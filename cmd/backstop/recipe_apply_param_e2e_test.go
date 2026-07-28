package main

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/check"
	"github.com/backstop-ai/backstop-core/pkg/recipe"
)

// ISSUE-081 Gap 2 — OPERATOR-SUPPLIED PARAMS, driven through the SHIPPED command.
//
// Before `--param`, a recipe declaring a `required: true` param with NO
// `default:` was UNREACHABLE through the CLI: direct mode substituted from the
// recipe's declared defaults alone, so the apply died on an unresolvable
// placeholder no invocation could satisfy.
//
// Every test here goes through NewRootCommand(). A unit test of the key=value
// parser alone would leave the flag-to-ApplyOptions thread unproven, which is the
// integration gap this repo keeps rediscovering — a parser can be perfect while
// the value never reaches Apply.
const (
	recipeParamE2EProject = "recipe-param-e2e"
	recipeParamE2EPack    = "demo-org/param-pack"
	recipeParamE2EID      = "adopt"
)

// suppliedParamValue is deliberately unlike any placeholder text, so a site that
// still carries the literal `{{ … }}` is obvious in a failure message.
const suppliedParamValue = "supplied-app-name"

// separatorParamValue carries BOTH separators that a careless flag registration
// would eat: a comma (pflag's StringSlice comma-splits, which is why the flag must
// be a StringArray) and an `=` (the key=value split must take the FIRST `=` only).
const separatorParamValue = "a,b=c"

// stageRecipeParamE2EProject copies the committed param fixture project — its
// installed pack under .backstop/packs included — into a fresh temp root, so a run
// mutates the copy and never the tracked fixture.
//
// It is a LOCAL stager rather than a parameterization of stageRecipeE2EProject:
// that helper hardcodes recipeE2EProject and lives in a file outside this task's
// scope, so widening its signature would force edits in every existing caller.
func stageRecipeParamE2EProject(t *testing.T) string {
	t.Helper()
	src, err := filepath.Abs(filepath.Join("testdata", recipeParamE2EProject))
	if err != nil {
		t.Fatalf("resolve fixture project %q: %v", recipeParamE2EProject, err)
	}
	dst := t.TempDir()
	copyTree(t, src, dst)
	return dst
}

// paramRun is one invocation of the shipped command with SEPARATED streams.
// runRecipeApplyCLI merges stdout and stderr into one buffer, which would make
// any assertion about WHICH stream a diagnostic landed on unfalsifiable.
type paramRun struct {
	stdout string
	stderr string
	err    error
}

// runRecipeApplyWithParams drives the ASSEMBLED root command from inside
// projectRoot, appending one repeated `--param` for each supplied argument.
func runRecipeApplyWithParams(t *testing.T, projectRoot string, ref string, params ...string) paramRun {
	t.Helper()
	t.Chdir(projectRoot)

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	root := NewRootCommand()
	root.SetOut(stdout)
	root.SetErr(stderr)

	args := []string{"recipe", "apply", ref}
	for _, param := range params {
		args = append(args, "--param", param)
	}
	root.SetArgs(args)
	err := root.Execute()

	return paramRun{stdout: stdout.String(), stderr: stderr.String(), err: err}
}

// requiredNoDefaultParam names the recipe's required param that declares no
// default — the one ONLY an operator can supply. Deriving it from the parsed
// manifest keeps the fixture's own declaration authoritative, and the exactly-one
// guard fails loud if the fixture drifts into proving something else.
func requiredNoDefaultParam(t *testing.T, manifest *recipe.RecipeManifest) string {
	t.Helper()

	var found []string
	for _, spec := range manifest.Params {
		if spec.Required && spec.Default == "" {
			found = append(found, spec.Name)
		}
	}
	if len(found) != 1 {
		t.Fatalf("fixture defect: want EXACTLY ONE required-no-default param (the case the CLI could not reach), got %v", found)
	}

	return found[0]
}

// defaultedParam names a param the recipe declares with a default, so a test can
// show a declared name is accepted without relying on the required one.
func defaultedParam(t *testing.T, manifest *recipe.RecipeManifest) string {
	t.Helper()

	for _, spec := range manifest.Params {
		if !spec.Required && spec.Default != "" {
			return spec.Name
		}
	}
	t.Fatal("fixture defect: the recipe declares no defaulted param")

	return ""
}

// projectTree lists every file beneath root, so a test can prove a REJECTED
// invocation wrote nothing at all rather than merely missing one expected path.
func projectTree(t *testing.T, root string) []string {
	t.Helper()

	var paths []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		paths = append(paths, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatalf("walk project root %q: %v", root, err)
	}
	sort.Strings(paths)

	return paths
}

// expectedCreateSite renders the create op's DECLARED target and DECLARED payload
// through the same param scope the applier owes them — the recipe's declared
// defaults plus whatever the operator supplied. Building the expectation from
// parsed DATA rather than a hardcoded path is what keeps it from agreeing with a
// broken applier by coincidence.
func expectedCreateSite(t *testing.T, manifest *recipe.RecipeManifest, recipeDir string, supplied map[string]string) (string, string) {
	t.Helper()

	params := declaredParamDefaults(manifest)
	for name, value := range supplied {
		params[name] = value
	}

	createOp := manifest.Ops[0]
	target, err := recipe.Substitute(createOp.Target, params)
	if err != nil {
		t.Fatalf("render the declared target %q against the effective params: %v", createOp.Target, err)
	}

	rawPayload, err := os.ReadFile(filepath.Join(recipeDir, filepath.FromSlash(createOp.Payload)))
	if err != nil {
		t.Fatalf("read the declared payload %q: %v", createOp.Payload, err)
	}
	content, err := recipe.Substitute(string(rawPayload), params)
	if err != nil {
		t.Fatalf("render the declared payload against the effective params: %v", err)
	}

	return target, content
}

// assertConfigError checks the rejection carries the exit-2 shape (main.go maps a
// *check.ConfigError to ExitConfigError) and quotes the offending argument. Every
// rejection in this file is PRE-APPLY: nothing was applied, so the INVOCATION is
// what must change.
func assertConfigError(t *testing.T, run paramRun, offending string) {
	t.Helper()

	if run.err == nil {
		t.Fatalf("the invocation must fail loud, got nil\nstdout:\n%s\nstderr:\n%s", run.stdout, run.stderr)
	}
	var configErr *check.ConfigError
	if !errors.As(run.err, &configErr) {
		t.Errorf("error must be a *check.ConfigError (exit 2 — nothing was applied), got %T: %v", run.err, run.err)
	}
	if !strings.Contains(run.err.Error(), offending) {
		t.Errorf("error must quote the offending argument %q so the operator can see what to fix, got: %v", offending, run.err)
	}
}

// TestRecipeApply_CLI_ParamFlagSuppliesRequiredNoDefaultParam proves CLM-007: a
// repeatable `--param key=value` threads into ApplyOptions.Params, so a
// `required: true` param declaring no `default:` resolves through the SHIPPED
// CLI. The without-the-flag subtest is what proves the flag is load-bearing
// rather than decorative.
func TestRecipeApply_CLI_ParamFlagSuppliesRequiredNoDefaultParam(t *testing.T) {
	t.Run("the supplied value resolves the required param", func(t *testing.T) {
		projectRoot := stageRecipeParamE2EProject(t)
		ref, manifest, recipeDir := stagedRecipe(t, projectRoot, recipeParamE2EPack, recipeParamE2EID)
		required := requiredNoDefaultParam(t, manifest)

		// Both the SUPPLIED and the DEFAULTED param must be load-bearing in the
		// resulting path; a test exercising only the defaulted one would pass
		// against a CLI that ignored --param entirely.
		declaredTarget := manifest.Ops[0].Target
		if !strings.Contains(declaredTarget, "{{ "+required+" }}") {
			t.Fatalf("fixture defect: the declared target %q does not depend on the required param %q", declaredTarget, required)
		}
		defaulted := defaultedParam(t, manifest)
		if !strings.Contains(declaredTarget, "{{ "+defaulted+" }}") {
			t.Fatalf("fixture defect: the declared target %q does not depend on the defaulted param %q", declaredTarget, defaulted)
		}

		wantTarget, wantContent := expectedCreateSite(t, manifest, recipeDir, map[string]string{required: suppliedParamValue})

		run := runRecipeApplyWithParams(t, projectRoot, ref, required+"="+suppliedParamValue)
		if run.err != nil {
			t.Fatalf("backstop recipe apply %s --param %s=%s failed: %v\nstdout:\n%s\nstderr:\n%s",
				ref, required, suppliedParamValue, run.err, run.stdout, run.stderr)
		}

		got, readErr := os.ReadFile(filepath.Join(projectRoot, filepath.FromSlash(wantTarget)))
		if readErr != nil {
			t.Fatalf("the file did not land at the substituted target %q: %v\nstdout:\n%s", wantTarget, readErr, run.stdout)
		}
		if string(got) != wantContent {
			t.Errorf("materialized content =\n%q\nwant the substituted payload\n%q", string(got), wantContent)
		}
		// The supplied value must reach the BYTES, not only the path.
		if !strings.Contains(string(got), suppliedParamValue) {
			t.Errorf("the written bytes carry no supplied value %q: %s", suppliedParamValue, got)
		}
	})

	t.Run("the same invocation without the flag fails", func(t *testing.T) {
		projectRoot := stageRecipeParamE2EProject(t)
		ref, _, _ := stagedRecipe(t, projectRoot, recipeParamE2EPack, recipeParamE2EID)

		run := runRecipeApplyWithParams(t, projectRoot, ref)
		if run.err == nil {
			t.Fatalf("a required param declaring no default cannot be resolved without --param, so this must fail\nstdout:\n%s", run.stdout)
		}
	})
}

// TestRecipeApply_CLI_MalformedParamFlagFailsLoud proves the first half of
// CLM-008: a `--param` with no `=`, or with an empty key, is a fail-loud config
// error naming the offending argument — never silently ignored.
func TestRecipeApply_CLI_MalformedParamFlagFailsLoud(t *testing.T) {
	cases := []struct {
		name  string
		param string
	}{
		{name: "no separator at all", param: "app_name"},
		{name: "an empty key", param: "=value"},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			projectRoot := stageRecipeParamE2EProject(t)
			ref, _, _ := stagedRecipe(t, projectRoot, recipeParamE2EPack, recipeParamE2EID)
			before := projectTree(t, projectRoot)

			run := runRecipeApplyWithParams(t, projectRoot, ref, testCase.param)
			assertConfigError(t, run, testCase.param)

			if after := projectTree(t, projectRoot); !reflect.DeepEqual(after, before) {
				t.Errorf("a rejected invocation wrote beneath the project root:\ngot  %v\nwant %v", after, before)
			}
		})
	}
}

// TestRecipeApply_CLI_DuplicateParamKeyFailsLoud proves the second half of
// CLM-008: a repeated key is refused rather than silently last-wins. A map write
// keeps the last value without saying so; saying it out loud costs the operator
// nothing and prevents a value they supplied from vanishing.
//
// The key used is one the recipe DECLARES, so the undeclared-name check cannot be
// what rejects this — only the duplicate check can.
func TestRecipeApply_CLI_DuplicateParamKeyFailsLoud(t *testing.T) {
	projectRoot := stageRecipeParamE2EProject(t)
	ref, manifest, _ := stagedRecipe(t, projectRoot, recipeParamE2EPack, recipeParamE2EID)
	required := requiredNoDefaultParam(t, manifest)
	before := projectTree(t, projectRoot)

	run := runRecipeApplyWithParams(t, projectRoot, ref, required+"=first", required+"=second")
	assertConfigError(t, run, required)

	// Not silently last-wins: nothing was applied at all.
	if after := projectTree(t, projectRoot); !reflect.DeepEqual(after, before) {
		t.Errorf("a duplicate key was applied last-wins instead of refused:\ngot  %v\nwant %v", after, before)
	}
}

// TestRecipeApply_CLI_UndeclaredParamFailsLoud proves CLM-009: a `--param` naming
// a param the recipe does not declare is refused pre-apply.
//
// Without this check a typo (`--param app_nmae=x`) is silently dropped and
// resurfaces as an unresolvable-placeholder failure the operator cannot attribute
// — the SAME undiagnosable shape ISSUE-081 was filed about. So the message must
// not read like one, which the assertion below pins.
func TestRecipeApply_CLI_UndeclaredParamFailsLoud(t *testing.T) {
	const undeclared = "not_a_param"

	t.Run("an undeclared name is refused", func(t *testing.T) {
		projectRoot := stageRecipeParamE2EProject(t)
		ref, manifest, _ := stagedRecipe(t, projectRoot, recipeParamE2EPack, recipeParamE2EID)
		required := requiredNoDefaultParam(t, manifest)
		before := projectTree(t, projectRoot)

		// The required param IS supplied, so the undeclared name is the only
		// defect left in the invocation.
		run := runRecipeApplyWithParams(t, projectRoot, ref,
			required+"="+suppliedParamValue,
			undeclared+"=whatever",
		)
		assertConfigError(t, run, undeclared)

		if strings.Contains(run.err.Error(), "unresolvable placeholder") {
			t.Errorf("the rejection reads as an unresolvable-placeholder failure, which is the undiagnosable shape this check exists to replace: %v", run.err)
		}
		if after := projectTree(t, projectRoot); !reflect.DeepEqual(after, before) {
			t.Errorf("a rejected invocation wrote beneath the project root:\ngot  %v\nwant %v", after, before)
		}
	})

	t.Run("the recipe's declared params are still accepted", func(t *testing.T) {
		projectRoot := stageRecipeParamE2EProject(t)
		ref, manifest, _ := stagedRecipe(t, projectRoot, recipeParamE2EPack, recipeParamE2EID)
		required := requiredNoDefaultParam(t, manifest)
		defaulted := defaultedParam(t, manifest)

		// Both declared names, including an explicit override of a param that
		// carries a default — so the check cannot pass by rejecting everything.
		run := runRecipeApplyWithParams(t, projectRoot, ref,
			required+"="+suppliedParamValue,
			defaulted+"=settings",
		)
		if run.err != nil {
			t.Fatalf("every supplied name is DECLARED by the recipe and must be accepted: %v\nstderr:\n%s", run.err, run.stderr)
		}
	})
}

// TestRecipeApply_CLI_ParamValueWithSeparatorsSurvivesIntact proves CLM-010 — the
// StringArray-not-StringSlice guard.
//
// pflag's StringSlice comma-splits its values, so registering the flag that way
// would silently deliver `a` and `b=c` as two params. Splitting key=value on the
// LAST `=` would mangle the rest. This asserts on the WRITTEN path and bytes
// rather than on an internal map, so it stays an end-to-end claim about what the
// operator actually gets.
func TestRecipeApply_CLI_ParamValueWithSeparatorsSurvivesIntact(t *testing.T) {
	projectRoot := stageRecipeParamE2EProject(t)
	ref, manifest, recipeDir := stagedRecipe(t, projectRoot, recipeParamE2EPack, recipeParamE2EID)
	required := requiredNoDefaultParam(t, manifest)

	if !strings.Contains(separatorParamValue, ",") || !strings.Contains(separatorParamValue, "=") {
		t.Fatalf("the guard value %q must carry BOTH separators or it proves nothing", separatorParamValue)
	}
	wantTarget, wantContent := expectedCreateSite(t, manifest, recipeDir, map[string]string{required: separatorParamValue})

	run := runRecipeApplyWithParams(t, projectRoot, ref, required+"="+separatorParamValue)
	if run.err != nil {
		t.Fatalf("a value carrying `,` and `=` must survive intact: %v\nstdout:\n%s\nstderr:\n%s", run.err, run.stdout, run.stderr)
	}

	got, readErr := os.ReadFile(filepath.Join(projectRoot, filepath.FromSlash(wantTarget)))
	if readErr != nil {
		t.Fatalf("the file did not land at the substituted target %q: %v\nstdout:\n%s", wantTarget, readErr, run.stdout)
	}
	if string(got) != wantContent {
		t.Errorf("materialized content =\n%q\nwant the substituted payload\n%q", string(got), wantContent)
	}
	if !strings.Contains(string(got), separatorParamValue) {
		t.Errorf("the written bytes carry %q rather than the whole supplied value %q — the value was split", got, separatorParamValue)
	}
}
