package backstopcore_test

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	yaml "gopkg.in/yaml.v3"
)

// These are the REPO's tests about the REPO's own release automation. They
// assert on PARSED workflow structure rather than on golden text, so that
// reformatting a workflow does not break them and deleting a job does.
const (
	releaseWorkflowFile      = ".github/workflows/release.yml"
	tagIntegrityWorkflowFile = ".github/workflows/tag-integrity.yml"
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

type workflowStep struct {
	Name string         `yaml:"name"`
	Uses string         `yaml:"uses"`
	Run  string         `yaml:"run"`
	With map[string]any `yaml:"with"`
}

type workflowJob struct {
	Name  string             `yaml:"name"`
	Needs workflowStringList `yaml:"needs"`
	Steps []workflowStep     `yaml:"steps"`
}

type workflowFile struct {
	Name        string                 `yaml:"name"`
	Permissions map[string]string      `yaml:"permissions"`
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
	for scope, want := range wantPermissions {
		if got := workflow.Permissions[scope]; got != want {
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
