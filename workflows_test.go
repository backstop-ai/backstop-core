package backstopcore_test

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
	yaml "gopkg.in/yaml.v3"
)

// These are the REPO's tests about the REPO's own release automation. They
// assert on PARSED workflow structure rather than on golden text, so that
// reformatting a workflow does not break them and deleting a job does.
const (
	releaseWorkflowFile      = ".github/workflows/release.yml"
	tagIntegrityWorkflowFile = ".github/workflows/tag-integrity.yml"
	ciWorkflowFile           = ".github/workflows/ci.yml"
	// probeWorkflowFile is the temporary Landlock/bubblewrap probe from
	// PLAN-ISSUE-020 TASK-001. Its VALUES are durable in
	// pkg/packval/testdata/ubuntu-runner-probe.txt; the workflow that produced
	// them is scaffolding, and TestCIWorkflow_NoProbeWorkflowRemains is what
	// keeps it from becoming permanent.
	probeWorkflowFile = ".github/workflows/linux-sandbox-probe.yml"
)

// selfGateInvocation is the command neither workflow may run. Self-gating is a
// founder-approved scope cut (ISSUE-087): it needs a macOS runner plus a fleet
// lock migration. The absence is pinned by a test so that adding it later is a
// conscious act. (CLM-017)
const selfGateInvocation = "backstop gate"

// workflowStringList decodes a YAML field that GitHub Actions accepts as either
// a bare scalar or a sequence — `needs: build` and `needs: [a, b]` are both
// legal and mean different things to a test that only handles one of them.
type workflowStringList []string

func (l *workflowStringList) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		var single string
		if err := node.Decode(&single); err != nil {
			return fmt.Errorf("decode scalar: %w", err)
		}
		*l = workflowStringList{single}
		return nil
	case yaml.SequenceNode:
		var many []string
		if err := node.Decode(&many); err != nil {
			return fmt.Errorf("decode sequence: %w", err)
		}
		*l = many
		return nil
	default:
		return fmt.Errorf("want a scalar or a sequence, got YAML kind %d", node.Kind)
	}
}

// workflowPermissions decodes the `permissions:` key, which GitHub Actions
// accepts in TWO shapes: a mapping of scope→level, and the bare scalars
// `read-all` / `write-all`.
//
// Decoding it as a bare map[string]string would ERROR on the scalar form rather
// than report it, and decoding it as `any` would silently yield nothing an
// assertion could read — either way a permissions assertion becomes a mystery.
// So both shapes are captured, and the assertions below fail loudly NAMING the
// shape they found. Same spirit as workflowStringList.
type workflowPermissions struct {
	// Scopes is populated for the mapping form.
	Scopes map[string]string
	// Scalar is populated for the `read-all` / `write-all` form.
	Scalar string
}

func (p *workflowPermissions) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		var scalar string
		if err := node.Decode(&scalar); err != nil {
			return fmt.Errorf("decode scalar permissions: %w", err)
		}
		p.Scalar = scalar
		return nil
	case yaml.MappingNode:
		scopes := map[string]string{}
		if err := node.Decode(&scopes); err != nil {
			return fmt.Errorf("decode permissions mapping: %w", err)
		}
		p.Scopes = scopes
		return nil
	default:
		return fmt.Errorf("permissions: want a scope mapping or the scalars read-all/write-all, got YAML kind %d", node.Kind)
	}
}

type workflowStep struct {
	Name string            `yaml:"name"`
	Uses string            `yaml:"uses"`
	Run  string            `yaml:"run"`
	With map[string]any    `yaml:"with"`
	Env  map[string]string `yaml:"env"`
	// ID is what a later step's `if:` expression addresses a step by
	// (`steps.<id>.conclusion`). Decoded so the guard on the post-gate
	// confirmation step can be checked against the step it actually names —
	// see TestCIWorkflow_GateJobProvesTheBaselineLandedAfterTheGate.
	ID string `yaml:"id"`
	// If and ContinueOnError are what a "blocking" job stops blocking through.
	// They are decoded so their ABSENCE can be asserted rather than assumed —
	// see TestCIWorkflow_PackInstallFailureFailsTheJob.
	If              string `yaml:"if"`
	ContinueOnError bool   `yaml:"continue-on-error"`
}

type workflowJob struct {
	Name        string              `yaml:"name"`
	Needs       workflowStringList  `yaml:"needs"`
	Permissions workflowPermissions `yaml:"permissions"`
	// Env is decoded so a JOB-level binding can be asserted ABSENT. A token
	// bound here reaches every step in the job, including the ones that pipe
	// `curl | sh` and run `pipx install` — see
	// TestCIWorkflow_GateJobAuthenticatesTheSelfHealingBaselinePull.
	Env   map[string]string `yaml:"env"`
	Steps []workflowStep    `yaml:"steps"`
}

type workflowFile struct {
	Name        string                 `yaml:"name"`
	Permissions workflowPermissions    `yaml:"permissions"`
	Jobs        map[string]workflowJob `yaml:"jobs"`
}

// readWorkflowSource reads a workflow and FAILS when it is missing or empty.
// Every assertion below is built on this: an absence assertion over a file that
// does not exist is a tautology, so a missing file must error the test rather
// than be skipped or swallowed.
func readWorkflowSource(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if strings.TrimSpace(string(data)) == "" {
		t.Fatalf("%s exists but is empty", path)
	}
	return string(data)
}

func loadWorkflow(t *testing.T, path string) workflowFile {
	t.Helper()

	var parsed workflowFile
	if err := yaml.Unmarshal([]byte(readWorkflowSource(t, path)), &parsed); err != nil {
		t.Fatalf("%s: parse YAML: %v", path, err)
	}
	if len(parsed.Jobs) == 0 {
		t.Fatalf("%s: declares no jobs", path)
	}
	return parsed
}

// workflowTriggers returns the mapping under the workflow's trigger key.
//
// The trigger key is the literal `on`, which YAML 1.1 resolves to the BOOLEAN
// true rather than the string "on". gopkg.in/yaml.v3 follows the YAML 1.2 core
// schema and keeps it a string — but that is a property of this parser, not of
// the file, and a parser change would silently turn every trigger assertion
// into a lookup miss. So look up both spellings, and fail loudly naming the
// keys actually present when neither is there.
func workflowTriggers(t *testing.T, path string) map[string]any {
	t.Helper()

	var top map[any]any
	if err := yaml.Unmarshal([]byte(readWorkflowSource(t, path)), &top); err != nil {
		t.Fatalf("%s: parse YAML: %v", path, err)
	}

	for _, key := range []any{"on", true} {
		raw, present := top[key]
		if !present {
			continue
		}
		mapping, ok := asMapping(raw)
		if !ok {
			t.Fatalf("%s: trigger key %v decoded as %T, want a mapping", path, key, raw)
		}
		return mapping
	}

	present := make([]string, 0, len(top))
	for key := range top {
		present = append(present, fmt.Sprintf("%v", key))
	}
	sort.Strings(present)
	t.Fatalf("%s: no trigger key found — looked for the string %q and the boolean true; top-level keys present: %v",
		path, "on", present)
	return nil
}

// asMapping normalizes the two shapes yaml.v3 produces for a nested mapping.
func asMapping(raw any) (map[string]any, bool) {
	switch typed := raw.(type) {
	case map[string]any:
		return typed, true
	case map[any]any:
		normalized := make(map[string]any, len(typed))
		for key, value := range typed {
			normalized[fmt.Sprintf("%v", key)] = value
		}
		return normalized, true
	default:
		return nil, false
	}
}

// asStrings normalizes a YAML scalar-or-sequence into a string slice.
func asStrings(raw any) []string {
	switch typed := raw.(type) {
	case string:
		return []string{typed}
	case []any:
		values := make([]string, 0, len(typed))
		for _, item := range typed {
			values = append(values, fmt.Sprintf("%v", item))
		}
		return values
	default:
		return nil
	}
}

func sortedMapKeys[V any](mapping map[string]V) []string {
	keys := make([]string, 0, len(mapping))
	for key := range mapping {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// allWorkflowSteps flattens every step in every job.
func allWorkflowSteps(workflow workflowFile) []workflowStep {
	steps := []workflowStep{}
	for _, jobID := range sortedMapKeys(workflow.Jobs) {
		steps = append(steps, workflow.Jobs[jobID].Steps...)
	}
	return steps
}

// leadingCommentBlock returns the comment block at the very top of a file — the
// consecutive `#` lines before any YAML content. Comments are discarded by the
// parser, so the header's text is the only way to assert on it.
func leadingCommentBlock(source string) string {
	lines := []string{}
	for _, line := range strings.Split(source, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			lines = append(lines, line)
			continue
		}
		if strings.TrimSpace(line) == "" && len(lines) == 0 {
			continue
		}
		break
	}
	return strings.Join(lines, "\n")
}

// TestReleaseWorkflow_TriggersOnlyOnSemverTagPush asserts release.yml fires on
// a `v*` tag push and on nothing else. A release workflow that also fires on
// every main push or on pull requests publishes garbage. (CLM-013)
func TestReleaseWorkflow_TriggersOnlyOnSemverTagPush(t *testing.T) {
	triggers := workflowTriggers(t, releaseWorkflowFile)

	if got := sortedMapKeys(triggers); len(got) != 1 || got[0] != "push" {
		t.Errorf("%s triggers on %v, want exactly [push] — any other trigger releases from an untagged ref",
			releaseWorkflowFile, got)
	}

	push, ok := asMapping(triggers["push"])
	if !ok {
		t.Fatalf("%s: `push` trigger is %T, want a mapping declaring tags", releaseWorkflowFile, triggers["push"])
	}

	tags := asStrings(push["tags"])
	if len(tags) == 0 {
		t.Fatalf("%s: `push` trigger declares no tags filter — it would fire on every branch push", releaseWorkflowFile)
	}
	if !containsString(tags, "v*") {
		t.Errorf("%s: push tags = %v, want it to include %q", releaseWorkflowFile, tags, "v*")
	}

	for _, forbidden := range []string{"branches", "branches-ignore"} {
		if _, present := push[forbidden]; present {
			t.Errorf("%s: push trigger declares %q — a release workflow must not fire on branch pushes",
				releaseWorkflowFile, forbidden)
		}
	}
	if _, present := triggers["pull_request"]; present {
		t.Errorf("%s: declares a pull_request trigger — a release workflow must not fire on pull requests",
			releaseWorkflowFile)
	}

	workflow := loadWorkflow(t, releaseWorkflowFile)
	wantPermissions := map[string]string{
		"contents": "write", // create the GitHub Release
		"actions":  "read",  // query ci.yml's conclusion for the tagged SHA
	}
	if workflow.Permissions.Scalar != "" {
		t.Fatalf("%s: workflow permissions decoded as the scalar %q, want a scope mapping",
			releaseWorkflowFile, workflow.Permissions.Scalar)
	}
	for scope, want := range wantPermissions {
		if got := workflow.Permissions.Scopes[scope]; got != want {
			t.Errorf("%s: permissions[%q] = %q, want %q", releaseWorkflowFile, scope, got, want)
		}
	}
}

// TestReleaseWorkflow_RequiresTagIntegrityAndGreenCI asserts the goreleaser job
// cannot run until BOTH the tag-integrity job and the green-CI job succeed.
// `needs` is the enforcement mechanism: a green-CI job that exists but is not
// in `needs` gates nothing. (CLM-014)
func TestReleaseWorkflow_RequiresTagIntegrityAndGreenCI(t *testing.T) {
	workflow := loadWorkflow(t, releaseWorkflowFile)

	publisherID, publisher := findJob(t, workflow, "runs goreleaser", func(job workflowJob) bool {
		return anyStep(job, func(step workflowStep) bool {
			return strings.Contains(step.Uses, "goreleaser")
		})
	})

	needs := []string(publisher.Needs)
	if len(needs) < 2 {
		t.Fatalf("%s: job %q needs %v — want it to need BOTH a tag-integrity job and a green-CI job",
			releaseWorkflowFile, publisherID, needs)
	}

	neededIntegrity := false
	neededGreenCI := false
	for _, needed := range needs {
		job, declared := workflow.Jobs[needed]
		if !declared {
			t.Errorf("%s: job %q needs %q, which is not a declared job (jobs: %v)",
				releaseWorkflowFile, publisherID, needed, sortedMapKeys(workflow.Jobs))
			continue
		}
		if anyStep(job, stepAssertsAncestry) {
			neededIntegrity = true
		}
		if anyStep(job, stepQueriesCIStatusForThisCommit) {
			neededGreenCI = true
		}
	}

	if !neededIntegrity {
		t.Errorf("%s: none of job %q's needs (%v) asserts tag integrity — no job among them runs %q",
			releaseWorkflowFile, publisherID, needs, "git merge-base --is-ancestor")
	}
	if !neededGreenCI {
		t.Errorf("%s: none of job %q's needs (%v) queries CI's conclusion for the tagged commit — "+
			"a step must consult CI status for GITHUB_SHA via `gh run list` or `gh api` rather than assume it",
			releaseWorkflowFile, publisherID, needs)
	}
}

// TestTagIntegrityWorkflow_ChecksSemverAndAncestryOfMain asserts the tag is
// both well-formed and reachable from main, and that the checkout has the
// history the ancestry query needs to answer with. (CLM-015)
func TestTagIntegrityWorkflow_ChecksSemverAndAncestryOfMain(t *testing.T) {
	workflow := loadWorkflow(t, tagIntegrityWorkflowFile)

	steps := allWorkflowSteps(workflow)
	if len(steps) == 0 {
		t.Fatalf("%s: declares no steps", tagIntegrityWorkflowFile)
	}

	if !anyStepIn(steps, func(step workflowStep) bool {
		return strings.Contains(step.Run, `[0-9]+\.[0-9]+\.[0-9]+`)
	}) {
		t.Errorf("%s: no step matches the pushed tag against a semver pattern — a malformed tag would release",
			tagIntegrityWorkflowFile)
	}

	if !anyStepIn(steps, stepAssertsAncestry) {
		t.Errorf("%s: no step runs `git merge-base --is-ancestor` against origin/main — a tag off a side branch would release",
			tagIntegrityWorkflowFile)
	}

	checkouts := 0
	for _, step := range steps {
		if !strings.Contains(step.Uses, "actions/checkout") {
			continue
		}
		checkouts++
		depth, present := step.With["fetch-depth"]
		if !present {
			t.Errorf("%s: checkout step %q sets no fetch-depth — the default shallow clone cannot answer the ancestry query",
				tagIntegrityWorkflowFile, step.Name)
			continue
		}
		if got := fmt.Sprintf("%v", depth); got != "0" {
			t.Errorf("%s: checkout step %q sets fetch-depth %s, want 0 (full history)",
				tagIntegrityWorkflowFile, step.Name, got)
		}
	}
	if checkouts == 0 {
		t.Errorf("%s: no actions/checkout step — the ancestry check has no repository to query",
			tagIntegrityWorkflowFile)
	}
}

// TestTagIntegrityWorkflow_CarriesTemporarySuccessorHeader asserts the header
// names this workflow as temporary and names its successors. This is the one
// legitimately textual assertion in the file: the header's WORDING is the
// deliverable, and YAML comments never reach the parser. (CLM-016)
func TestTagIntegrityWorkflow_CarriesTemporarySuccessorHeader(t *testing.T) {
	source := readWorkflowSource(t, tagIntegrityWorkflowFile)

	header := leadingCommentBlock(source)
	if header == "" {
		t.Fatalf("%s: no leading comment block — the temporary-successor header is the deliverable", tagIntegrityWorkflowFile)
	}
	lowerHeader := strings.ToLower(header)

	// Identifying tokens rather than full sentences, so rewording the header
	// does not break the test but dropping a successor does.
	for _, token := range []string{"temporary", "req-018", "go-distribution"} {
		if !strings.Contains(lowerHeader, token) {
			t.Errorf("%s: header does not name %q; header:\n%s", tagIntegrityWorkflowFile, token, header)
		}
	}

	// The one NEGATIVE token assertion. ISSUE-086 was narrowed on 2026-07-27 to
	// the packless-baseline defect and no longer owns the hand-baked-CI concern
	// (ISSUE-020 does). Presence assertions cannot catch a WRONG citation — they
	// only check that expected tokens are there — and this header is shipped
	// source that outlives the plan, read by people who will chase whatever
	// issue number it names.
	const wrongCitation = "ISSUE-086"
	if strings.Contains(source, wrongCitation) {
		t.Errorf("%s: cites %s, which no longer owns the hand-baked-CI concern — cite ISSUE-020 or nothing",
			tagIntegrityWorkflowFile, wrongCitation)
	}
}

// TestReleaseWorkflows_DoNotSelfGate asserts both workflows exist and neither
// invokes `backstop gate`.
//
// The existence assertions are load-bearing, not decoration: an absence-only
// assertion over files that do not exist passes trivially, which would make
// this the one test here that starts green. (CLM-017)
func TestReleaseWorkflows_DoNotSelfGate(t *testing.T) {
	paths := []string{releaseWorkflowFile, tagIntegrityWorkflowFile}

	// FIRST: both files exist and are non-empty. readWorkflowSource fails the
	// test on a missing file rather than skipping it.
	for _, path := range paths {
		if source := readWorkflowSource(t, path); source == "" {
			t.Fatalf("%s: read back empty", path)
		}
	}

	for _, path := range paths {
		workflow := loadWorkflow(t, path)
		steps := allWorkflowSteps(workflow)
		if len(steps) == 0 {
			t.Fatalf("%s: declares no steps — the absence check below would be vacuous", path)
		}
		for _, step := range steps {
			for field, value := range map[string]string{"run": step.Run, "uses": step.Uses} {
				if strings.Contains(value, selfGateInvocation) {
					t.Errorf("%s: step %q %s invokes %q — self-gating is a deliberate scope cut; read release.yml's header before adding it",
						path, step.Name, field, selfGateInvocation)
				}
			}
		}

		// Steps are where an invocation lives, but scan the whole file too, so
		// a `backstop gate` smuggled into a workflow-level field is caught.
		// Comments are stripped: the release.yml header must be free to NAME
		// the command it is explaining the absence of.
		for lineNo, line := range strings.Split(readWorkflowSource(t, path), "\n") {
			if idx := strings.Index(line, "#"); idx >= 0 {
				line = line[:idx]
			}
			if strings.Contains(line, selfGateInvocation) {
				t.Errorf("%s:%d: invokes %q outside a step", path, lineNo+1, selfGateInvocation)
			}
		}
	}

	// The absence must read as a DECISION. release.yml's header names the
	// fast-follow: a macOS runner (Linux's sandbox is a hard no-op per
	// ISSUE-020) plus a fleet lock migration.
	header := leadingCommentBlock(readWorkflowSource(t, releaseWorkflowFile))
	if header == "" {
		t.Fatalf("%s: no leading comment block — the absence of self-gating must be recorded as a decision", releaseWorkflowFile)
	}
	lowerHeader := strings.ToLower(header)
	for _, token := range []string{selfGateInvocation, "macos", "issue-020"} {
		if !strings.Contains(lowerHeader, strings.ToLower(token)) {
			t.Errorf("%s: header does not name %q, so the missing self-gate reads as an oversight; header:\n%s",
				releaseWorkflowFile, token, header)
		}
	}
}

// stepAssertsAncestry reports whether a step runs the tag-is-on-main check.
func stepAssertsAncestry(step workflowStep) bool {
	return strings.Contains(step.Run, "merge-base --is-ancestor") && strings.Contains(step.Run, "origin/main")
}

// stepQueriesCIStatusForThisCommit reports whether a step consults CI's
// conclusion for the tagged commit rather than assuming it. ci.yml triggers on
// main pushes and pull requests, never on tags, so the query must be by commit
// SHA.
func stepQueriesCIStatusForThisCommit(step workflowStep) bool {
	if !strings.Contains(step.Run, "GITHUB_SHA") {
		return false
	}
	return strings.Contains(step.Run, "gh run list") || strings.Contains(step.Run, "gh api")
}

func anyStep(job workflowJob, predicate func(workflowStep) bool) bool {
	return anyStepIn(job.Steps, predicate)
}

func anyStepIn(steps []workflowStep, predicate func(workflowStep) bool) bool {
	for _, step := range steps {
		if predicate(step) {
			return true
		}
	}
	return false
}

// findJob returns the single job matching predicate, failing when zero or more
// than one match — an ambiguous match would let an assertion land on the wrong
// job and pass for the wrong reason.
func findJob(t *testing.T, workflow workflowFile, description string, predicate func(workflowJob) bool) (string, workflowJob) {
	t.Helper()

	matches := []string{}
	for _, jobID := range sortedMapKeys(workflow.Jobs) {
		if predicate(workflow.Jobs[jobID]) {
			matches = append(matches, jobID)
		}
	}
	if len(matches) != 1 {
		t.Fatalf("found %d jobs that %s (%v), want exactly 1; jobs declared: %v",
			len(matches), description, matches, sortedMapKeys(workflow.Jobs))
	}
	return matches[0], workflow.Jobs[matches[0]]
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

// goreleaserEnvRefPattern matches a `{{ .Env.NAME }}` template reference in the
// goreleaser config, tolerating the optional inner spacing goreleaser allows.
var goreleaserEnvRefPattern = regexp.MustCompile(`{{\s*\.Env\.([A-Za-z_][A-Za-z0-9_]*)\s*}}`)

// TestReleaseWorkflow_ExportsEveryEnvVarGoreleaserTemplatesReference is the
// falsifier for a CROSS-FILE drift that neither file can catch alone:
// .goreleaser.yml renders `{{ .Env.X }}` from the PROCESS environment, so a
// variable it templates but the workflow never exports is simply empty at render
// time. Nothing fails locally — `goreleaser check` validates config shape, not
// the runner's environment — and the first real tagged run is where it surfaces.
//
// That is exactly how HOMEBREW_TAP_TOKEN was missed: the brews block templated
// it while the goreleaser step exported only GITHUB_TOKEN.
//
// The required set is DERIVED from .goreleaser.yml rather than hardcoded here,
// so adding a new `{{ .Env.* }}` reference without exporting it fails this test
// automatically instead of needing someone to remember to extend it.
func TestReleaseWorkflow_ExportsEveryEnvVarGoreleaserTemplatesReference(t *testing.T) {
	configSource, err := os.ReadFile(goreleaserConfigFile)
	if err != nil {
		t.Fatalf("read %s: %v", goreleaserConfigFile, err)
	}

	required := map[string]bool{}
	for _, match := range goreleaserEnvRefPattern.FindAllStringSubmatch(string(configSource), -1) {
		required[match[1]] = true
	}
	if len(required) == 0 {
		t.Fatalf("%s templates no {{ .Env.* }} references — this test has nothing to prove and the pattern has probably drifted", goreleaserConfigFile)
	}

	workflow := loadWorkflow(t, releaseWorkflowFile)
	_, publisher := findJob(t, workflow, "runs goreleaser", func(job workflowJob) bool {
		for _, step := range job.Steps {
			if strings.Contains(step.Uses, "goreleaser") {
				return true
			}
		}
		return false
	})

	exported := map[string]string{}
	for _, step := range publisher.Steps {
		if !strings.Contains(step.Uses, "goreleaser") {
			continue
		}
		for name, value := range step.Env {
			exported[name] = value
		}
	}

	for name := range required {
		value, present := exported[name]
		if !present {
			t.Errorf("%s templates {{ .Env.%s }} but the goreleaser step in %s does not export it — it renders empty on the first real tagged run", goreleaserConfigFile, name, releaseWorkflowFile)
			continue
		}
		if strings.TrimSpace(value) == "" {
			t.Errorf("goreleaser step exports %s with an empty value", name)
		}
	}
}

// ─── ci.yml: the blocking gate job (PLAN-ISSUE-020 Phase 4) ─────────────────────
//
// These assert that backstop-core's own CI DOGFOODS `backstop gate` rather than
// hand-rolling lint/test/coverage beside it. ISSUE-020's whole premise is that the
// Linux sandbox defect stayed invisible because this repo's CI never called the
// gate — so a workflow that runs the tools directly is not a weaker version of the
// same check, it is the gap.
//
// Everything below reads PARSED YAML. A test that breaks on reindentation teaches
// the next person to delete it.

// stepScript returns a step's run script with COMMENT LINES REMOVED.
//
// Every assertion in this section matches on script text, and a shell comment is
// not an invocation. This is not hypothetical tidiness: the baseline job's
// `baseline generate` step carries the comment "equivalent to `./backstop gate
// --all --json`", which a raw strings.Contains reads as a second job running the
// gate with --all. That made findJob see two blocking jobs and made the
// scope-flag assertion fail against a comment.
//
// The load-bearing comments this plan REQUIRES in ci.yml — the base-derivation
// rationale, the golangci-lint pin explanation, the "install nothing for the
// sandbox" note — all name the very strings these tests forbid. Without this
// helper, writing the documentation the plan mandates would fail the tests that
// mandate it.
func stepScript(step workflowStep) string {
	kept := []string{}
	for _, line := range strings.Split(step.Run, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}

// stepMentions reports whether a step's executable content — its decommented run
// script or any of its env values — contains needle. Env matters because a
// workflow expression is idiomatically bound to an env var rather than
// interpolated into the script body.
func stepMentions(step workflowStep, needle string) bool {
	if strings.Contains(stepScript(step), needle) {
		return true
	}
	for _, value := range step.Env {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}

// ciBlockingJob returns the job that invokes `backstop gate`.
//
// It is found by BEHAVIOUR rather than by the job id, so renaming the job does not
// silently skip every assertion in this section — the failure would be "no job runs
// backstop gate", which is the thing worth knowing.
func ciBlockingJob(t *testing.T) workflowJob {
	t.Helper()
	workflow := loadWorkflow(t, ciWorkflowFile)
	_, job := findJob(t, workflow, "runs `backstop gate`", func(candidate workflowJob) bool {
		return anyStep(candidate, func(step workflowStep) bool {
			return strings.Contains(stepScript(step), selfGateInvocation)
		})
	})
	return job
}

// stepIndex returns the position of the first step whose run script contains
// needle, or -1. Order assertions need positions, not booleans.
func stepIndex(job workflowJob, needle string) int {
	for i, step := range job.Steps {
		if strings.Contains(stepScript(step), needle) {
			return i
		}
	}
	return -1
}

// stepIndexes returns EVERY position whose decommented script contains needle.
//
// It exists beside stepIndex rather than replacing it, and what it is FOR changed
// under ISSUE-099. ci.yml used to invoke `backstop gate` TWICE — a diagnostic
// --json capture and the blocking run — so stepIndex's first-match answer named
// the wrong step and an ordering fence anchored on it was escapable by inserting a
// step between the two. `--json-out` collapsed those into ONE invocation, so the
// first match IS the blocking step today.
//
// stepIndexes stays, and is now MORE load-bearing rather than less: the exact-count
// assertions it enables (`len(gateSteps) != 1`) are the mechanism that catches a
// re-introduced second invocation, which is the regression ISSUE-099 exists to
// prevent. Six existing call sites across three tests depend on stepIndex's
// first-match contract, so it is left alone.
func stepIndexes(job workflowJob, needle string) []int {
	found := []int{}
	for i, step := range job.Steps {
		if strings.Contains(stepScript(step), needle) {
			found = append(found, i)
		}
	}
	return found
}

// lastStepIndex returns the position of the LAST step whose decommented script
// contains needle, or -1.
func lastStepIndex(job workflowJob, needle string) int {
	found := stepIndexes(job, needle)
	if len(found) == 0 {
		return -1
	}
	return found[len(found)-1]
}

// TestCIWorkflow_BlockingJobRunsBackstopGate is the flip itself: the job builds the
// binary, installs the packs, and THEN runs the gate with an explicit `--base`.
//
// The order is the assertion that matters. `pack install` after the gate would run
// every dimension against an empty `.backstop/packs/`, which reports
// capability_absent and passes — a green build that checked nothing. (CLM-018)
func TestCIWorkflow_BlockingJobRunsBackstopGate(t *testing.T) {
	job := ciBlockingJob(t)

	build := stepIndex(job, "go build")
	install := stepIndex(job, "pack install")
	gate := stepIndex(job, selfGateInvocation)

	if build < 0 {
		t.Errorf("%s: the blocking job never builds the backstop binary", ciWorkflowFile)
	}
	if install < 0 {
		t.Fatalf("%s: the blocking job never runs `pack install`; every gate dimension would report "+
			"capability_absent against an empty .backstop/packs/ and the job would pass having checked nothing",
			ciWorkflowFile)
	}
	if gate < 0 {
		t.Fatalf("%s: the blocking job never runs %q", ciWorkflowFile, selfGateInvocation)
	}
	if install > gate {
		t.Errorf("%s: `pack install` is step %d and the gate is step %d — the gate must run AFTER the packs "+
			"it consumes are materialized", ciWorkflowFile, install, gate)
	}
	if build > install {
		t.Errorf("%s: the binary is built at step %d but used at step %d", ciWorkflowFile, build, install)
	}

	if !anyStep(job, func(step workflowStep) bool {
		return strings.Contains(stepScript(step), selfGateInvocation) && strings.Contains(stepScript(step), "--base")
	}) {
		t.Errorf("%s: the gate is invoked without `--base`. A CI checkout is pristine, so bare diff mode "+
			"resolves merge-base HEAD origin/main to HEAD on a push and finds nothing to check — the vacuous "+
			"green this job exists to prevent", ciWorkflowFile)
	}
}

// TestCIWorkflow_GateRunsOnceAndEmitsTheJSONReportInTheBlockingStep is ISSUE-099's
// shipped shape: ONE gate invocation, both surfaces. (CLM-007, CLM-009)
//
// ci.yml used to run the entire kill chain TWICE per push — a diagnostic
// `gate --json > gate-report.json || echo ...` capture followed by the blocking
// run — because the CLI rendered EITHER the human table OR the JSON document to
// stdout and nothing wrote JSON to a file. On run 32151610956 that duplicated
// ~10.5min of pack_engines and ~10.2min of coverage_threshold. `--json-out` writes
// the envelope to a file while the table still renders to the log, so one step now
// does both.
//
// The exact-count leg is the point: a second invocation is precisely the regression
// this lane exists to prevent, and it must fail LOUDLY rather than silently
// re-anchor an ordering fence elsewhere in this file.
func TestCIWorkflow_GateRunsOnceAndEmitsTheJSONReportInTheBlockingStep(t *testing.T) {
	job := ciBlockingJob(t)

	gateSteps := stepIndexes(job, selfGateInvocation)
	if len(gateSteps) != 1 {
		t.Fatalf("%s: %d gate-job steps invoke %q (%v), want EXACTLY 1. Since ISSUE-099 the blocking step emits "+
			"the JSON report itself with --json-out; a second invocation re-runs the whole kill chain, which is "+
			"the ~20-minutes-per-push cost this lane removed",
			ciWorkflowFile, len(gateSteps), selfGateInvocation, stepNames(job, gateSteps))
	}
	gate := job.Steps[gateSteps[0]]
	script := stepScript(gate)

	if !strings.Contains(script, "--json-out gate-report.json") {
		t.Errorf("%s: the gate step %q does not carry `--json-out gate-report.json`. Without it the JSON envelope "+
			"is never written and the `Upload the gate JSON report` step below publishes nothing:\n%s",
			ciWorkflowFile, gate.Name, script)
	}

	// THE RETIRED CAPTURE'S TWO SIGNATURES. A `>` redirect of the gate into a file
	// or an `|| echo` swallow would mean the diagnostic capture came back — either
	// as a second step or folded into this one, where the `|| echo` would also stop
	// the job blocking on the verdict.
	if strings.Contains(script, ">") {
		t.Errorf("%s: the gate step %q redirects with `>`. The envelope is written by --json-out, not by shell "+
			"redirection — a redirect captures stdout and would take the human table with it:\n%s",
			ciWorkflowFile, gate.Name, script)
	}
	if strings.Contains(script, "|| echo") {
		t.Errorf("%s: the gate step %q ends in `|| echo`. THIS is the blocking invocation and its exit is the "+
			"job's; `|| echo` reports instead of gating:\n%s", ciWorkflowFile, gate.Name, script)
	}

	// THE BLOCKING CONTRACT, restated at the step this claim owns rather than left
	// to another test's map iteration.
	if gate.If != "" {
		t.Errorf("%s: the gate step %q declares if: %q. A conditional blocking step is a job that can pass "+
			"without gating", ciWorkflowFile, gate.Name, gate.If)
	}
	if strings.Contains(script, "|| true") {
		t.Errorf("%s: the gate step %q swallows its exit with `|| true`:\n%s", ciWorkflowFile, gate.Name, script)
	}

	// CLM-009: THE FLAG NAME. `--json-file` contains `--file` as a substring and
	// would trip TestCIWorkflow_BlockingJobNeverUsesAllOrFileScope, whose ban is a
	// standing founder ruling — the fix for that failure would be to weaken a
	// shipped, correct test. `--json-out` does not collide.
	if strings.Contains(script, "--file") {
		t.Errorf("%s: the gate step %q carries `--file` in its decommented script. If this is a `--json-file` "+
			"spelling of the report flag, RENAME THE FLAG to --json-out: "+
			"TestCIWorkflow_BlockingJobNeverUsesAllOrFileScope substring-matches `--file` over this step and its "+
			"ban is not negotiable here:\n%s", ciWorkflowFile, gate.Name, script)
	}

	// The retirement must not quietly drop the artifact it exists to preserve.
	uploadFound := false
	for _, step := range job.Steps {
		if !strings.Contains(step.Uses, "upload-artifact") {
			continue
		}
		path, _ := step.With["path"].(string)
		if !strings.Contains(path, "gate-report.json") {
			continue
		}
		uploadFound = true
		if !strings.Contains(path, "cover.out") {
			t.Errorf("%s: the upload step %q publishes gate-report.json but not cover.out. The gate JSON carries "+
				"per-FILE coverage totals only; the raw profile is the sole surface that can say WHICH FUNCTIONS "+
				"carry zero counters:\npath: %s", ciWorkflowFile, step.Name, path)
		}
		if !strings.Contains(step.If, "always()") {
			t.Errorf("%s: the upload step %q has if: %q, which must include always(). The run worth diagnosing is "+
				"the one that FAILED, and --json-out writes the file BEFORE the process decides its exit "+
				"precisely so that upload has something to publish", ciWorkflowFile, step.Name, step.If)
		}
	}
	if !uploadFound {
		t.Errorf("%s: no step uploads gate-report.json as an artifact. Collapsing the two gate invocations must "+
			"not drop the report the surviving one now writes", ciWorkflowFile)
	}
}

// TestCIWorkflow_BlockingJobHasNoRawToolInvocations is what makes the flip REAL
// rather than additive. If the hand-baked Lint / Test / coverage-threshold steps
// survive beside the gate, CI is still not dogfooding it — it is running the tools
// twice and blocking on the copy that is not the product.
//
// This is also the test that fails the day someone re-adds a "quick lint step".
// (CLM-018)
func TestCIWorkflow_BlockingJobHasNoRawToolInvocations(t *testing.T) {
	job := ciBlockingJob(t)

	for invocation, why := range map[string]string{
		"golangci-lint run": "the lint dimension belongs to the go-toolchain pack, reached through the gate",
		"go test":           "the test and coverage dimensions belong to the pack's go-test engine",
		"go tool cover":     "a hand-rolled coverage threshold is a second, weaker copy of coverage_threshold",
	} {
		for i, step := range job.Steps {
			if strings.Contains(stepScript(step), invocation) {
				t.Errorf("%s: blocking-job step %d (%q) runs %q — %s", ciWorkflowFile, i, step.Name, invocation, why)
			}
		}
	}
}

// TestCIWorkflow_BlockingJobNeverUsesAllOrFileScope pins the two scope flags out of
// the blocking invocation.
//
// ISSUE-091 and ISSUE-093 are BOTH FIXED — `--all` now dispatches its own
// explicit file list and agrees with a diff-scoped run over the same files
// (CLM-008), and `--file` no longer crashes on an unclaimed file or drops
// repeated occurrences. Neither original bug independently justifies the ban
// anymore (see ISSUE-152 / ISSUE-156, the follow-on issues that asked whether
// it should lift). FOUNDER RULING (2026-08-17): the ban STAYS, but on a
// DIFFERENT and durable basis — CI's blocking job is a latency- and
// scope-sensitive path, and diff scope with an explicit base is the
// deliberately narrow, fast, predictable shape that job wants regardless of
// whether `--all`/`--file` are correct. This is a standing policy choice, not
// a placeholder pending further decision — ISSUE-152 and ISSUE-156 are closed
// with this ruling recorded, and this is not open for re-litigation absent a
// new argument.
// Diff scope with an explicit base is the shape CI's blocking job actually uses
// today, and it stays that way by design. (CLM-020)
func TestCIWorkflow_BlockingJobNeverUsesAllOrFileScope(t *testing.T) {
	job := ciBlockingJob(t)

	for _, step := range job.Steps {
		if !strings.Contains(stepScript(step), selfGateInvocation) {
			continue
		}
		for flag, issue := range map[string]string{
			"--all":  "both original defects are fixed, but the blocking job stays diff-scoped by deliberate policy (ISSUE-152)",
			"--file": "both original defects are fixed, but the blocking job stays diff-scoped by deliberate policy (ISSUE-156)",
		} {
			if strings.Contains(stepScript(step), flag) {
				t.Errorf("%s: the gate invocation carries %s (%s):\n%s", ciWorkflowFile, flag, issue, stepScript(step))
			}
		}
	}
}

// TestCIWorkflow_DerivesDiffBasePerEvent asserts the base comes from the event
// payload, and from the RIGHT field for each event — the two triggers carry it in
// different places. (CLM-019)
func TestCIWorkflow_DerivesDiffBasePerEvent(t *testing.T) {
	job := ciBlockingJob(t)

	// Matched against EXECUTABLE content — decommented script plus env values —
	// rather than the raw file. A workflow that only mentioned these expressions in
	// a comment explaining the derivation would otherwise pass while deriving
	// nothing.
	for expression, event := range map[string]string{
		"github.event.pull_request.base.sha": "pull_request",
		"github.event.before":                "push",
	} {
		if !anyStep(job, func(step workflowStep) bool { return stepMentions(step, expression) }) {
			t.Errorf("%s: no blocking-job step derives the diff base from %s for the %s event",
				ciWorkflowFile, expression, event)
		}
	}

	if stepIndex(job, "BASE") < 0 {
		t.Errorf("%s: the blocking job has no step that resolves a BASE value", ciWorkflowFile)
	}
}

// TestCIWorkflow_BaseGuardChecksResolvabilityNotZeroSha is shaped around
// RESOLVABILITY on purpose, and the reason is a failure mode a zero-SHA string
// comparison does not catch.
//
// On a force-push, `github.event.before` names a commit that no longer exists in
// the fetched history. It is NON-ZERO, so a guard that only compares against the
// all-zeros SHA does not fire; the dead sha flows through to the gate, and the job
// exits 2 with a confusing cause on a perfectly ordinary force-push.
//
// One `git rev-parse --verify` covers all three ways the derived value goes bad:
// empty (missing payload field), all-zeros (branch creation, which includes the
// very first push of this history to a new remote), and unreachable. (CLM-019)
func TestCIWorkflow_BaseGuardChecksResolvabilityNotZeroSha(t *testing.T) {
	job := ciBlockingJob(t)

	if !anyStep(job, func(step workflowStep) bool {
		return strings.Contains(stepScript(step), "git rev-parse --verify")
	}) {
		t.Fatalf("%s: no step verifies the DERIVED base resolves. A guard that only string-compares against "+
			"the all-zeros SHA misses the force-push shape entirely, where github.event.before is non-zero "+
			"AND unreachable", ciWorkflowFile)
	}
}

// TestCIWorkflow_TriggersOnPushAndPullRequest — both events, per the founder
// ruling. Proving only one proves half of the base-derivation claim, since the two
// paths read different payload fields. (CLM-019)
func TestCIWorkflow_TriggersOnPushAndPullRequest(t *testing.T) {
	triggers := workflowTriggers(t, ciWorkflowFile)

	for _, want := range []string{"push", "pull_request"} {
		if _, present := triggers[want]; !present {
			t.Errorf("%s: no %q trigger; declared triggers are %v", ciWorkflowFile, want, sortedMapKeys(triggers))
		}
	}
}

// TestCIWorkflow_CheckoutFetchesFullHistory — `fetch-depth: 0` on the blocking
// job's checkout. Without it the PR base sha is not in the local history, so the
// base guard cannot resolve it and every run falls back or exits 2. (CLM-019)
func TestCIWorkflow_CheckoutFetchesFullHistory(t *testing.T) {
	job := ciBlockingJob(t)

	checkouts := 0
	for _, step := range job.Steps {
		if !strings.HasPrefix(step.Uses, "actions/checkout@") {
			continue
		}
		checkouts++
		depth, present := step.With["fetch-depth"]
		if !present {
			t.Errorf("%s: the blocking job's checkout declares no fetch-depth; the default shallow clone "+
				"does not contain the base commit", ciWorkflowFile)
			continue
		}
		if fmt.Sprintf("%v", depth) != "0" {
			t.Errorf("%s: checkout fetch-depth is %v, want 0 (full history)", ciWorkflowFile, depth)
		}
	}
	if checkouts == 0 {
		t.Errorf("%s: the blocking job never checks out the repository", ciWorkflowFile)
	}
}

// TestCIWorkflow_InstallsProvisionedToolsAtAllowlistPins reads the pins from
// pkg/pack/engine at TEST TIME rather than restating them.
//
// That is the whole point: a hardcoded "1.96.0" here would let a pin bump in
// allowlist.go leave CI silently behind, installing a version the dispatch gate
// then refuses — which surfaces as an exit-2 config error, not as a version
// mismatch. (CLM-021)
func TestCIWorkflow_InstallsProvisionedToolsAtAllowlistPins(t *testing.T) {
	job := ciBlockingJob(t)
	allowlist := engine.TrustedToolAllowlist()

	// Only the backstop-INTRODUCED tools are installed by CI. The "*" entries are
	// Layer-0/runtime tools backstop does not provision, so requiring an install
	// step for them would assert something the allowlist does not claim.
	for _, tool := range []string{"semgrep", "ast-grep"} {
		pin, present := allowlist[tool]
		if !present {
			t.Fatalf("pkg/pack/engine no longer pins %q; this test's premise is gone and it must be rewritten "+
				"rather than deleted", tool)
		}
		if pin == "*" {
			t.Fatalf("%q is pinned to %q (presence-only); it is no longer an auto-provisioned tool", tool, pin)
		}
		if !anyStep(job, func(step workflowStep) bool {
			return strings.Contains(stepScript(step), tool) && strings.Contains(stepScript(step), pin)
		}) {
			t.Errorf("%s: no blocking-job step installs %s at the allowlist pin %s — the dispatch gate will "+
				"refuse whatever version the runner happens to have", ciWorkflowFile, tool, pin)
		}
	}
}

// TestCIWorkflow_PinsGolangciLintToPackCompatibleMajor is the falsifier the
// allowlist CANNOT provide: golangci-lint is not on it and never will be, because
// nil-Provision bindings return early from checkEngineToolAllowed.
//
// Three assertions, and the middle one is the load-bearing one. The go-toolchain
// pack declares `golangci-lint run --output.sarif.path stdout --show-stats=false`,
// which is v2-ONLY syntax — a v1 binary fails it with an unknown-flag error and the
// lint dimension breaks. go.mod's tool directive pins v1.64.8 for unrelated
// reasons, so installing through `go tool` installs exactly the wrong major.
//
// The pin is asserted rather than derived because the pack lives in gitignored
// .backstop/packs/: a test that read it would be unrunnable on a fresh checkout,
// before `pack install` has run. (CLM-022)
func TestCIWorkflow_PinsGolangciLintToPackCompatibleMajor(t *testing.T) {
	job := ciBlockingJob(t)

	installer := -1
	for i, step := range job.Steps {
		if strings.Contains(stepScript(step), "golangci-lint") {
			installer = i
			break
		}
	}
	if installer < 0 {
		t.Fatalf("%s: no blocking-job step installs a golangci-lint binary. provisionEngines resolves the "+
			"executable BY NAME on PATH, so the lint dimension is unavailable without one", ciWorkflowFile)
	}
	install := job.Steps[installer]

	if !regexp.MustCompile(`v2\.\d+\.\d+`).MatchString(stepScript(install)) {
		t.Errorf("%s: the golangci-lint install pins no v2.x version:\n%s\nThe go-toolchain pack declares "+
			"`--output.sarif.path`, which is v2-only syntax; a v1 pin breaks the lint dimension with an "+
			"unknown-flag error", ciWorkflowFile, stepScript(install))
	}
	if strings.Contains(stepScript(install), "go tool") {
		t.Errorf("%s: golangci-lint is installed via `go tool`, which resolves go.mod's tool directive "+
			"(v1.64.8) — the wrong major, and not a binary on PATH:\n%s", ciWorkflowFile, stepScript(install))
	}
}

// TestCIWorkflow_InstallsNoSandboxTooling is a real falsifier, not a formality.
//
// Landlock and seccomp are KERNEL APIs: there is nothing to install. A reappearing
// `apt-get install bubblewrap` means someone is reviving the mechanism the founder
// ruled out on 2026-07-28 — and pkg/packval/testdata/ubuntu-runner-probe.txt shows
// installing it does not even help. On stock ubuntu-24.04 the probe measured
// apparmor_restrict_unprivileged_userns=1 and all seven bwrap invocations failing,
// INCLUDING both positive controls, with bubblewrap 0.9.0 present at /usr/bin/bwrap.
//
// The sandbox's actual runner requirement is a kernel property, and it is asserted
// where it can be: TASK-013's execution tests fail loudly on a host without
// Landlock, and they run inside the gate. A workflow-level kernel probe would be a
// second, weaker copy of that. (CLM-023)
func TestCIWorkflow_InstallsNoSandboxTooling(t *testing.T) {
	workflow := loadWorkflow(t, ciWorkflowFile)

	for _, step := range allWorkflowSteps(workflow) {
		for _, forbidden := range []string{"bubblewrap", "bwrap", "install -y bwrap"} {
			if strings.Contains(stepScript(step), forbidden) {
				t.Errorf("%s: step %q references %q. The mechanism is Landlock + seccomp — kernel APIs with "+
					"nothing to install — and the probe measured bubblewrap present-and-broken on the runner:\n%s",
					ciWorkflowFile, step.Name, forbidden, stepScript(step))
			}
		}
	}
}

// TestCIWorkflow_GoVersionComesFromGoMod — setup-go reads `go-version-file: go.mod`
// so the toolchain has ONE authority. A second hardcoded version drifts silently.
func TestCIWorkflow_GoVersionComesFromGoMod(t *testing.T) {
	workflow := loadWorkflow(t, ciWorkflowFile)

	setups := 0
	for _, step := range allWorkflowSteps(workflow) {
		if !strings.HasPrefix(step.Uses, "actions/setup-go@") {
			continue
		}
		setups++
		if got := fmt.Sprintf("%v", step.With["go-version-file"]); got != "go.mod" {
			t.Errorf("%s: setup-go go-version-file is %q, want \"go.mod\"", ciWorkflowFile, got)
		}
		if raw, present := step.With["go-version"]; present {
			t.Errorf("%s: setup-go hardcodes go-version %v beside the go.mod file reference — two authorities "+
				"for one toolchain", ciWorkflowFile, raw)
		}
	}
	if setups == 0 {
		t.Errorf("%s: no setup-go step", ciWorkflowFile)
	}
}

// TestCIWorkflow_PackInstallFailureFailsTheJob — the blocking job's load-bearing
// steps must have no escape hatch. A `continue-on-error` on `pack install` produces
// the exact silent failure this plan exists to remove: the packs are absent, every
// dimension reports capability_absent, and the job goes green. (CLM-018)
func TestCIWorkflow_PackInstallFailureFailsTheJob(t *testing.T) {
	job := ciBlockingJob(t)

	for i, step := range job.Steps {
		loadBearing := strings.Contains(stepScript(step), "pack install") || strings.Contains(stepScript(step), selfGateInvocation)
		if !loadBearing {
			continue
		}
		if step.ContinueOnError {
			t.Errorf("%s: step %d (%q) sets continue-on-error, so its failure does not block", ciWorkflowFile, i, step.Name)
		}
		if step.If != "" {
			t.Errorf("%s: step %d (%q) is conditional on %q, so it can be skipped entirely",
				ciWorkflowFile, i, step.Name, step.If)
		}
		if strings.Contains(stepScript(step), "|| true") {
			t.Errorf("%s: step %d (%q) swallows its exit code with `|| true`:\n%s",
				ciWorkflowFile, i, step.Name, stepScript(step))
		}
	}
}

// TestCIWorkflow_BaselineJobInstallsPacksBeforeGenerating is coupled to this
// plan's fleet migration and would be arbitrary without it.
//
// TASK-003 converted the locked packs from local to git sources. VerifyLock SKIPS
// local entries but HARD-FAILS git entries whose directories are absent
// (pkg/pack/distribution/verify.go:46-60), and runBaselineGenerate has no install
// step of its own (cmd/backstop/baseline.go:50-75) — so after the migration this
// job goes from ONE missing_pack failure to SIX, published straight into the
// artifact main's ratchet compares against. Adding `pack install` is what stops
// this plan from making ISSUE-086 worse. (CLM-029)
func TestCIWorkflow_BaselineJobInstallsPacksBeforeGenerating(t *testing.T) {
	workflow := loadWorkflow(t, ciWorkflowFile)

	_, job := findJob(t, workflow, "generates the baseline", func(candidate workflowJob) bool {
		return anyStep(candidate, func(step workflowStep) bool {
			return strings.Contains(stepScript(step), "baseline generate")
		})
	})

	install := stepIndex(job, "pack install")
	generate := stepIndex(job, "baseline generate")

	if install < 0 {
		t.Fatalf("%s: the baseline job never runs `pack install`; after the local-to-git fleet migration its "+
			"published artifact would carry six missing_pack failures instead of one", ciWorkflowFile)
	}
	if install > generate {
		t.Errorf("%s: `pack install` is step %d but `baseline generate` is step %d — the packs must exist "+
			"before the baseline that records them is generated", ciWorkflowFile, install, generate)
	}
}

// TestCIWorkflow_NoProbeWorkflowRemains — the TASK-001 probe was scaffolding. Its
// values live durably in pkg/packval/testdata/ubuntu-runner-probe.txt; the workflow
// that produced them does not stay.
func TestCIWorkflow_NoProbeWorkflowRemains(t *testing.T) {
	if _, err := os.Stat(probeWorkflowFile); err == nil {
		t.Errorf("%s still exists. It was a temporary probe (PLAN-ISSUE-020 TASK-001); its measured values "+
			"are durable in pkg/packval/testdata/ubuntu-runner-probe.txt", probeWorkflowFile)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat %s: %v", probeWorkflowFile, err)
	}
}

// TestCIWorkflow_LeavesNoUngitignoredDroppings is the guard against CI gating its
// own build output.
//
// The gate's diff scope INCLUDES UNTRACKED FILES, and several steps run before it,
// so anything a step writes into the workspace lands in the set of files the gate
// then blocks on. This is not hypothetical: run 30386673582 reported 11 in-scope
// files, of which ast-grep.zip (the engine-tools download) and gate-report.json (the
// gate's own --json-out write) were STEP OUTPUT rather than source. cover.out was already
// gitignored, which is exactly why it never appeared — and is the precedent this
// test generalises.
//
// The assertion is deliberately about GITIGNORE rather than about cleanup: a step
// that removes its own output is one `set -e` away from not running, whereas an
// ignored path cannot enter diff scope at all.
func TestCIWorkflow_LeavesNoUngitignoredDroppings(t *testing.T) {
	ignored, err := os.ReadFile(".gitignore")
	if err != nil {
		t.Fatalf("reading .gitignore: %v", err)
	}
	patterns := map[string]bool{}
	for _, line := range strings.Split(string(ignored), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			patterns[line] = true
		}
	}

	// Artifacts the blocking job's own steps write into the workspace. Each MUST be
	// gitignored or the gate blocks on a file CI created seconds earlier. cover.out's
	// pattern is root-anchored (/cover.out, ISSUE-fixed alongside the testdata/init-
	// toolchain/bin/ and testdata/go-toolchain-coverage/ fixture collision a bare
	// `cover.out` pattern was also swallowing) — the anchor still covers the repo-root
	// file CI produces, so the check below matches the real, current pattern string.
	for artifact, producedBy := range map[string]string{
		"ast-grep.zip":     "the provisioned-engine-tools download",
		"gate-report.json": "the blocking gate step's own --json-out write",
		"/cover.out":       "the go-toolchain pack's coverage producer",
	} {
		if !patterns[artifact] {
			t.Errorf(".gitignore does not cover %q, written by %s. Diff scope includes untracked files, so "+
				"the gate would block on output its own job produced — measured on run 30386673582, where "+
				"two such files were in scope", artifact, producedBy)
		}
	}
}

// TestCIWorkflow_BaselineJobInstallsTheSameToolsAsGate is the STRUCTURAL fix for how
// the baseline job broke, rather than a fix for the breakage itself.
//
// The gate job installed its Layer-0 analyzers and provisioned engine tools; the baseline
// job did not — and nothing noticed, because the baseline job had NEVER SUCCESSFULLY
// RUN. Its first execution (main, 30398137055) failed at provisionEngines, which fails
// loud when a Layer-0 tool is missing from PATH. Both jobs run a gate, so both need
// the same toolchain, and asserting parity is what stops them drifting apart the next
// time a tool is added to one of them.
func TestCIWorkflow_BaselineJobInstallsTheSameToolsAsGate(t *testing.T) {
	workflow := loadWorkflow(t, ciWorkflowFile)
	gateJob, ok := workflow.Jobs["gate"]
	if !ok {
		t.Fatalf("%s has no gate job", ciWorkflowFile)
	}
	baselineJob, ok := workflow.Jobs["baseline"]
	if !ok {
		t.Fatalf("%s has no baseline job", ciWorkflowFile)
	}

	// Every tool the gate job puts on PATH must also be on the baseline job's PATH.
	for _, tool := range []string{"golangci-lint", "go-arch-lint", "semgrep", "ast-grep"} {
		inGate := anyStep(gateJob, func(s workflowStep) bool { return strings.Contains(stepScript(s), tool) })
		inBaseline := anyStep(baselineJob, func(s workflowStep) bool { return strings.Contains(stepScript(s), tool) })
		if inGate && !inBaseline {
			t.Errorf("%s: the gate job installs %q and the baseline job does not. `baseline generate` runs a "+
				"FULL-SCOPE gate, and provisionEngines exits 2 when a Layer-0 tool is missing from PATH — "+
				"this is exactly how run 30398137055 died", ciWorkflowFile, tool)
		}
	}
}

// ─── ci.yml: authorizing the self-healing baseline pull (ISSUE-176) ─────────────
//
// `backstop gate` already self-heals a missing `.backstop/baseline.json`:
// resolveBaselineCache → refreshBaselineFromRemote → runBaselinePull runs
// unconditionally before the gate executes. On CI it has never worked, and it
// degrades SILENTLY to a nil baseline. The gap is authorization, not code:
// ci.yml binds no token (so ensureGitHubAuth's `gh auth status` shell-out fails
// first) and grants no `actions: read` (so the pull's `gh api
// repos/.../actions/runs` and `.../actions/artifacts` calls would 403 even with
// one). These four tests pin the wiring that closes it.

// baselinePullInvocation is the command the post-gate confirmation step shells
// in its failure branch — and the command that must appear EXACTLY ONCE in the
// gate job, because a second one anywhere would create the file the
// confirmation step checks for. (CLM-003)
const baselinePullInvocation = "baseline pull"

// committedBaselinePath is the file the self-healing pull lands, and the only
// observable this repo has for whether the pull worked: ApplyPolicy overwrites
// baseline_comparison's reason unconditionally under the dogfood policy, so
// neither the printed table nor --json can confirm or refute it.
const committedBaselinePath = ".backstop/baseline.json"

// stepShellsGitHubCLI reports whether a gate-job step can shell `gh`, and so
// needs the token bound in its OWN env:.
//
// The set is DERIVED rather than listed by index. It has TWO members since
// ISSUE-099 collapsed ci.yml to a single gate invocation: the one `backstop gate`
// step (which runs resolveBaselineCache), and the confirmation step's failure
// branch, which runs a bare `baseline pull` that is useless unauthenticated.
//
// The derived-not-listed choice is MORE load-bearing at two members than it was at
// three, not less: a two-member set is small enough to look listable by index, and
// listing it by index is exactly how it silently half-wires the next time a step
// moves.
func stepShellsGitHubCLI(step workflowStep) bool {
	script := stepScript(step)
	return strings.Contains(script, selfGateInvocation) || strings.Contains(script, baselinePullInvocation)
}

// gitHubTokenBinding returns the value and the name of whichever GitHub-token
// variable an env block declares, or empty strings when it declares neither.
func gitHubTokenBinding(env map[string]string) (value string, name string) {
	for _, candidate := range []string{"GH_TOKEN", "GITHUB_TOKEN"} {
		if bound, present := env[candidate]; present {
			return bound, candidate
		}
	}
	return "", ""
}

// secretsReferencePattern matches a `secrets.NAME` expression anywhere in a
// workflow's text.
var secretsReferencePattern = regexp.MustCompile(`secrets\.([A-Za-z_][A-Za-z0-9_-]*)`)

// stepConclusionPattern extracts the step id a `steps.<id>.conclusion`
// expression addresses.
var stepConclusionPattern = regexp.MustCompile(`steps\.([A-Za-z0-9_-]+)\.conclusion`)

// stepNames renders a set of step indices as index+name pairs, so a failure
// names the offending steps rather than making the reader count.
func stepNames(job workflowJob, indices []int) []string {
	named := make([]string, 0, len(indices))
	for _, i := range indices {
		named = append(named, fmt.Sprintf("%d (%q)", i, job.Steps[i].Name))
	}
	return named
}

// TestCIWorkflow_GateJobAuthenticatesTheSelfHealingBaselinePull asserts the
// token reaches every gate-job step that can shell `gh`, STEP-LEVEL, and reaches
// nothing else. (CLM-001)
//
// The negative legs are what make "step-level, not job-level" a claim rather
// than prose. A job-level `env:` is the obvious spelling and passes every
// positive assertion here — while handing the token to all ten steps, including
// the two that pipe `curl | sh` and run `pipx install` and the one that clones
// eight remote pack repositories. None of those needs it.
func TestCIWorkflow_GateJobAuthenticatesTheSelfHealingBaselinePull(t *testing.T) {
	job := ciBlockingJob(t)

	needsToken := []int{}
	for i, step := range job.Steps {
		if stepShellsGitHubCLI(step) {
			needsToken = append(needsToken, i)
		}
	}
	if len(needsToken) == 0 {
		t.Fatalf("%s: no gate-job step shells `gh` — none invokes %q or %q. Every assertion below would be "+
			"vacuous over an empty set", ciWorkflowFile, selfGateInvocation, baselinePullInvocation)
	}
	inDerivedSet := map[int]bool{}
	for _, i := range needsToken {
		inDerivedSet[i] = true
	}

	// POSITIVE LEG: every derived member carries the binding, in its OWN env:.
	for _, i := range needsToken {
		step := job.Steps[i]
		value, name := gitHubTokenBinding(step.Env)
		if name == "" {
			t.Errorf("%s: step %d (%q) shells `gh` but declares no GH_TOKEN/GITHUB_TOKEN in its own env:. "+
				"ensureGitHubAuth runs `gh auth status` BEFORE the pull reaches the GitHub API, so an "+
				"unauthenticated step fails there and the self-healing baseline pull degrades silently to a nil "+
				"baseline (ISSUE-176). Steps that need it: %v",
				ciWorkflowFile, i, step.Name, stepNames(job, needsToken))
			continue
		}
		if !strings.Contains(value, "secrets.GITHUB_TOKEN") && !strings.Contains(value, "github.token") {
			t.Errorf("%s: step %d (%q) binds %s to %q — want the AUTO-GENERATED ${{ secrets.GITHUB_TOKEN }} "+
				"(equivalently ${{ github.token }}): repository-scoped, minted per run, expiring with the job. "+
				"Never a hand-rolled PAT", ciWorkflowFile, i, step.Name, name, value)
		}
	}

	// NEGATIVE LEG (a): no step OUTSIDE the derived set carries the token.
	strays := []int{}
	for i, step := range job.Steps {
		if inDerivedSet[i] {
			continue
		}
		if _, name := gitHubTokenBinding(step.Env); name != "" {
			strays = append(strays, i)
		}
	}
	if len(strays) > 0 {
		t.Errorf("%s: these gate-job steps bind a GitHub token but never shell `gh`: %v. The exposure is scoped "+
			"at the STEP deliberately — the steps that execute the most third-party code in this job (the "+
			"`curl | sh` installer, the `pipx install`, the eight-repo pack clone) must not hold it",
			ciWorkflowFile, stepNames(job, strays))
	}

	// NEGATIVE LEG (b): the JOB binds neither name. An absent job env: satisfies
	// this trivially and correctly.
	if _, name := gitHubTokenBinding(job.Env); name != "" {
		t.Errorf("%s: the gate JOB declares %s in its job-level env:, which hands the token to ALL of its steps. "+
			"Bind it step-level on the %d steps that shell `gh` instead: %v",
			ciWorkflowFile, name, len(needsToken), stepNames(job, needsToken))
	}

	// LEAST-PRIVILEGE FENCE: the auto-token and nothing else. The scan surface is
	// the union of every step's env: and with: values, the job's own env:, and
	// every step's RAW run script — raw, because a `secrets.` expression
	// interpolated straight into a script body is the smuggling route that
	// matters most, and a comment naming one is worth flagging anyway.
	type scannedText struct {
		where string
		text  string
	}
	surfaces := []scannedText{}
	for _, name := range sortedMapKeys(job.Env) {
		surfaces = append(surfaces, scannedText{fmt.Sprintf("the job's env: %s", name), job.Env[name]})
	}
	for i, step := range job.Steps {
		for _, name := range sortedMapKeys(step.Env) {
			surfaces = append(surfaces, scannedText{fmt.Sprintf("step %d (%q) env: %s", i, step.Name, name), step.Env[name]})
		}
		for _, name := range sortedMapKeys(step.With) {
			surfaces = append(surfaces, scannedText{
				fmt.Sprintf("step %d (%q) with: %s", i, step.Name, name),
				fmt.Sprintf("%v", step.With[name]),
			})
		}
		surfaces = append(surfaces, scannedText{fmt.Sprintf("step %d (%q) run script", i, step.Name), step.Run})
	}
	for _, surface := range surfaces {
		for _, match := range secretsReferencePattern.FindAllStringSubmatch(surface.text, -1) {
			if match[1] == "GITHUB_TOKEN" {
				continue
			}
			t.Errorf("%s: %s references secrets.%s. The gate job is authorized by the auto-generated "+
				"GITHUB_TOKEN alone — a hand-rolled PAT is a wider, longer-lived credential and is fenced out here "+
				"so substituting one later is a conscious act that fails a test", ciWorkflowFile, surface.where, match[1])
		}
	}
}

// TestCIWorkflow_GateJobGrantsActionsReadWithoutWideningTheWorkflow asserts the
// permission widening is real AND narrow: `actions: read` on the gate job, and
// the workflow-level block untouched. (CLM-002)
func TestCIWorkflow_GateJobGrantsActionsReadWithoutWideningTheWorkflow(t *testing.T) {
	workflow := loadWorkflow(t, ciWorkflowFile)
	job := ciBlockingJob(t)

	if job.Permissions.Scalar != "" {
		t.Fatalf("%s: the gate job's permissions decoded as the scalar %q. Want the mapping "+
			"{contents: read, actions: read} — a scalar grants vastly more than the pull needs",
			ciWorkflowFile, job.Permissions.Scalar)
	}
	scopes := job.Permissions.Scopes
	if len(scopes) == 0 {
		t.Fatalf("%s: the gate job declares no job-level permissions: block. Without `actions: read` the "+
			"self-healing pull's `gh api repos/.../actions/runs` and `.../actions/artifacts` calls 403 and the "+
			"gate degrades silently to a nil baseline (ISSUE-176)", ciWorkflowFile)
	}

	if got := scopes["contents"]; got != "read" {
		t.Errorf("%s: the gate job's permissions[contents] = %q, want %q. THIS LEG IS NOT REDUNDANT with the "+
			"workflow-level block: a JOB-level permissions block REPLACES the workflow-level one outright and "+
			"every scope it does not list becomes `none`, so omitting contents:read revokes it and "+
			"actions/checkout fails as a mystery", ciWorkflowFile, got, "read")
	}
	if got := scopes["actions"]; got != "read" {
		t.Errorf("%s: the gate job's permissions[actions] = %q, want %q — the narrowest scope that lets the "+
			"pull's `gh api repos/.../actions/runs` and `.../actions/artifacts` calls answer",
			ciWorkflowFile, got, "read")
	}
	for _, scope := range sortedMapKeys(scopes) {
		level := scopes[scope]
		if scope != "contents" && scope != "actions" {
			t.Errorf("%s: the gate job grants an extra scope %q: %q — want exactly {contents: read, actions: read}",
				ciWorkflowFile, scope, level)
		}
		if strings.Contains(level, "write") {
			t.Errorf("%s: the gate job grants %q: %q — the pull only ENUMERATES and DOWNLOADS this repository's "+
				"own runs and artifacts; it writes nothing", ciWorkflowFile, scope, level)
		}
	}

	// THE BLAST-RADIUS FENCE. `actions: read` at the workflow level would widen
	// the baseline job and every future job for no reason, and that is exactly
	// the "simpler" edit this assertion exists to fail.
	if workflow.Permissions.Scalar != "" {
		t.Fatalf("%s: the WORKFLOW-level permissions decoded as the scalar %q, want the mapping {contents: read}",
			ciWorkflowFile, workflow.Permissions.Scalar)
	}
	workflowScopes := workflow.Permissions.Scopes
	if len(workflowScopes) != 1 || workflowScopes["contents"] != "read" {
		t.Errorf("%s: workflow-level permissions = %v, want exactly {contents: read}. The widening belongs to the "+
			"gate job and to no other", ciWorkflowFile, workflowScopes)
	}
	if level, present := workflowScopes["actions"]; present {
		t.Errorf("%s: workflow-level permissions grant actions: %q. Scope it at the JOB — a workflow-level grant "+
			"reaches the baseline job and every future job, none of which pulls a baseline", ciWorkflowFile, level)
	}
}

// TestCIWorkflow_GateJobProvesTheBaselineLandedAfterTheGate asserts the wiring
// carries its own in-band falsifier: a guarded, post-gate step that observes
// whether the pull landed a file. (CLM-003(ii))
func TestCIWorkflow_GateJobProvesTheBaselineLandedAfterTheGate(t *testing.T) {
	job := ciBlockingJob(t)

	gateSteps := stepIndexes(job, selfGateInvocation)
	if len(gateSteps) != 1 {
		t.Fatalf("%s: %d gate-job steps invoke %q (%v), want exactly 1 — the blocking run, which now emits the "+
			"JSON report itself via --json-out (ISSUE-099). The ordering leg below anchors on the LAST of them, "+
			"so a second would silently re-anchor it",
			ciWorkflowFile, len(gateSteps), selfGateInvocation, stepNames(job, gateSteps))
	}
	lastGate := lastStepIndex(job, selfGateInvocation)

	confirmations := stepIndexes(job, committedBaselinePath)
	if len(confirmations) != 1 {
		t.Fatalf("%s: %d gate-job steps reference %s in their decommented script (%v), want exactly 1 — the "+
			"post-gate confirmation step. ApplyPolicy overwrites baseline_comparison's Reason unconditionally "+
			"under this repo's dogfood policy, so that step is the ONLY surface that can say whether the "+
			"self-healing pull landed a file (ISSUE-176)",
			ciWorkflowFile, len(confirmations), committedBaselinePath, stepNames(job, confirmations))
	}
	confirmIndex := confirmations[0]
	confirm := job.Steps[confirmIndex]
	script := stepScript(confirm)

	if strings.Contains(script, "|| true") {
		t.Errorf("%s: the confirmation step %q swallows its exit code with `|| true`:\n%s",
			ciWorkflowFile, confirm.Name, script)
	}
	if strings.Contains(script, selfGateInvocation) {
		t.Errorf("%s: the confirmation step %q carries %q in its DECOMMENTED script. It declares an `if:`, and "+
			"TestCIWorkflow_PackInstallFailureFailsTheJob fails any step whose script contains that literal and "+
			"is conditional — keep the explanation in a comment, which stepScript strips",
			ciWorkflowFile, confirm.Name, selfGateInvocation)
	}

	// THE GUARD, asserted precisely rather than as "declares if: always()".
	if !strings.Contains(confirm.If, "always()") {
		t.Errorf("%s: the confirmation step %q has if: %q, which must include always() — the run worth "+
			"diagnosing is the one whose gate FAILED", ciWorkflowFile, confirm.Name, confirm.If)
	}
	if !strings.Contains(confirm.If, "!= 'skipped'") {
		t.Errorf("%s: the confirmation step %q has if: %q, which must exclude the case where the gate step was "+
			"SKIPPED. This job can die at tool install, at build or at pack install; under a bare always() the "+
			"step then finds no baseline (nothing had a chance to fetch one) and blames ISSUE-176 for a failure "+
			"three steps upstream — the exact misattribution burden ISSUE-176 exists to remove",
			ciWorkflowFile, confirm.Name, confirm.If)
	}
	references := stepConclusionPattern.FindAllStringSubmatch(confirm.If, -1)
	if len(references) != 1 {
		t.Fatalf("%s: the confirmation step's if: %q references %d `steps.<id>.conclusion` expressions, want "+
			"exactly 1 — the blocking gate step's", ciWorkflowFile, confirm.If, len(references))
	}
	referencedID := references[0][1]
	if got := job.Steps[lastGate].ID; got != referencedID {
		t.Errorf("%s: the confirmation step's guard names steps.%s.conclusion, but the blocking gate step "+
			"(step %d, %q) declares id %q. A guard naming an id no step declares does NOT fail safe — it goes "+
			"INERT. GitHub resolves the missing context property to null; `null != 'skipped'` compares mismatched "+
			"types so both are coerced to number (null → 0, 'skipped' → NaN); every comparison involving NaN is "+
			"false, so the `!=` is TRUE and the condition collapses to `always() && true`. The step then runs on "+
			"EVERY run, including the early-death runs this guard exists to suppress",
			ciWorkflowFile, referencedID, lastGate, job.Steps[lastGate].Name, got)
	}

	if confirmIndex <= lastGate {
		t.Errorf("%s: the confirmation step is step %d and the LAST gate invocation is step %d. A check placed "+
			"before the gate proves nothing — the pull that creates the file runs INSIDE the gate, in "+
			"resolveBaselineCache", ciWorkflowFile, confirmIndex, lastGate)
	}
}

// TestCIWorkflow_TheOnlyBaselinePullFollowsEveryGateInvocation is the
// load-bearing falsifier, and it is deliberately INDEX-FREE in its primary leg.
// (CLM-003(i))
func TestCIWorkflow_TheOnlyBaselinePullFollowsEveryGateInvocation(t *testing.T) {
	job := ciBlockingJob(t)

	pulls := stepIndexes(job, baselinePullInvocation)
	if len(pulls) != 1 {
		t.Fatalf("%s: %d gate-job steps invoke %q (%v), want EXACTLY 1 — the post-gate confirmation step's "+
			"failure branch. A second `baseline pull` ANYWHERE in this job would create the very file the "+
			"confirmation step checks for, making the proof self-fulfilling. Uniqueness rather than ordering stays "+
			"the primary fence because it holds regardless of how many gate steps there are: a forging pull "+
			"ANYWHERE in this job creates that file, whereas an ordering fence only constrains where it sits",
			ciWorkflowFile, len(pulls), baselinePullInvocation, stepNames(job, pulls))
	}

	gateSteps := stepIndexes(job, selfGateInvocation)
	if len(gateSteps) != 1 {
		t.Fatalf("%s: %d gate-job steps invoke %q (%v), want exactly 1 since ISSUE-099 collapsed the diagnostic "+
			"capture into the blocking run — a second would re-anchor the ordering leg below",
			ciWorkflowFile, len(gateSteps), selfGateInvocation, stepNames(job, gateSteps))
	}
	lastGate := lastStepIndex(job, selfGateInvocation)

	if pulls[0] <= lastGate {
		t.Errorf("%s: the only `%s` is step %d, at or before the LAST gate invocation (step %d). It must follow "+
			"EVERY gate invocation, or it creates the file the confirmation checks for",
			ciWorkflowFile, baselinePullInvocation, pulls[0], lastGate)
	}
}
