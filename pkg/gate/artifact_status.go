package gate

// ISSUE-042 Phase 1 (CLM-001/002/003/013/014): the shared, EXPORTED artifact-status
// resolver substrate. It extends the currently spec-shaped frontmatter parsing to also
// parse issues/ plans/ directives/ for `status` + mandated tests, adds a TYPE-AWARE
// three-class status classifier, and builds the cross-artifact issue->plan linkage. It
// is deliberately independent of the drift step so ISSUE-043's delivered_by /
// close-friction validator can reuse the SAME status+mandated-test resolution (CLM-013).
//
// It REUSES ExtractMandatedTests / MandatedTest for spec claim->tests discovery (do NOT
// reimplement mandated-test extraction). Issues carry the SAME claims[].tests shape, so
// the resolver applies the identical claim->tests walk to them. Plans and directives
// declare no own mandated tests — their records carry status only. This code introduces
// NO baked language noun: it reads artifact frontmatter exclusively.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ArtifactKind names one of the four backstop artifact types the resolver walks.
type ArtifactKind string

const (
	KindIssue     ArtifactKind = "issue"
	KindSpec      ArtifactKind = "spec"
	KindPlan      ArtifactKind = "plan"
	KindDirective ArtifactKind = "directive"
)

// StatusClass is the TYPE-AWARE lifecycle class a declared status resolves to. It is
// SEPARATE from isTerminalSpecStatus (which folds ALL terminals together and is left
// untouched, ISSUE-031). Critically it carves out ClassSuccessTerminal — the class
// isTerminalSpecStatus never names — so the drift step can hold success-terminal artifacts
// to their mandated-test existence while still excluding retired ones.
type StatusClass int

const (
	// ClassNonTerminal — live/in-flight status. The drift step's WARN direction (a
	// delivered-looking non-terminal with all tests present) applies here; it NEVER blocks.
	ClassNonTerminal StatusClass = iota
	// ClassSuccessTerminal — the artifact declares delivery complete (issue closed / spec
	// implemented / directive done / plan completed). The drift step's BLOCK direction
	// (any mandated test absent) applies here.
	ClassSuccessTerminal
	// ClassRetiredTerminal — replaced/canceled/deprecated. No delivery obligation; the
	// drift step EXCLUDES it, preserving ISSUE-031's retired-exclusion.
	ClassRetiredTerminal
)

// String renders a StatusClass for diagnostics and violation messages.
func (c StatusClass) String() string {
	switch c {
	case ClassNonTerminal:
		return "non-terminal"
	case ClassSuccessTerminal:
		return "success-terminal"
	case ClassRetiredTerminal:
		return "retired-terminal"
	default:
		return "unknown-status-class"
	}
}

// ClassifyArtifactStatus maps a declared status to exactly one of {non-terminal,
// success-terminal, retired-terminal} PER artifact TYPE (CLM-002). The success-terminal
// status differs by type (issue closed / spec implemented / directive done / plan
// completed); the retired set (replaced/canceled/obsoleted, plus spec-only deprecated) is
// uniform. An UNRECOGNIZED status defaults to non-terminal — the safe direction, since the
// WARN direction never blocks; a typo can never manufacture a spurious success-terminal
// BLOCK.
//
// NB: there is NO spec `active` — `active` is a DIRECTIVE status. The spec success-terminal
// is `implemented`.
func ClassifyArtifactStatus(kind ArtifactKind, status string) StatusClass {
	// Retired terminals are uniform across types (deprecated exists only for specs, and
	// obsoleted (ISSUE-048) is a delivered-then-removed retired terminal for issue/spec/
	// plan — classifying either retired for any type is harmless; the enum simply never
	// occurs elsewhere). Retired MUST return here, BEFORE the success-terminal switch.
	switch status {
	case "replaced", "canceled", "deprecated", "obsoleted":
		return ClassRetiredTerminal
	}
	// Success terminals are TYPE-specific.
	switch kind {
	case KindIssue:
		if status == "closed" {
			return ClassSuccessTerminal
		}
	case KindSpec:
		if status == "implemented" {
			return ClassSuccessTerminal
		}
	case KindDirective:
		if status == "done" {
			return ClassSuccessTerminal
		}
	case KindPlan:
		if status == "completed" {
			return ClassSuccessTerminal
		}
	}
	// Everything else (live statuses + unrecognized) is non-terminal.
	return ClassNonTerminal
}

// ArtifactStatusRecord is one resolved artifact: its id, kind, declared status, the
// TYPE-aware class that status resolves to, and its mandated tests (empty for plans /
// directives). SpecID is populated for plans only — the id (SPEC-NNN or ISSUE-NNN) the
// plan implements, driving the issue->plan linkage.
type ArtifactStatusRecord struct {
	ID            string
	Kind          ArtifactKind
	Status        string
	Class         StatusClass
	Path          string
	MandatedTests []MandatedTest
	SpecID        string
}

// ArtifactStatusResolution is the resolver output: every resolved record plus the
// issue->plan index (plans keyed by their spec_id) so a consumer can ask which plan
// delivers a given issue (CLM-014).
type ArtifactStatusResolution struct {
	Records       []ArtifactStatusRecord
	plansBySpecID map[string]ArtifactStatusRecord
}

// BackingPlan returns the plan record whose spec_id == the given artifact id (the
// PLAN-ISSUE-NNN / PLAN-SPEC-NNN convention), and false when no plan backs it. ISSUE-043's
// delivered_by / close-friction validator reuses this directly; it is useful here to
// resolve a closed issue's backing plan.
func (r *ArtifactStatusResolution) BackingPlan(artifactID string) (ArtifactStatusRecord, bool) {
	if r == nil || r.plansBySpecID == nil {
		return ArtifactStatusRecord{}, false
	}
	p, ok := r.plansBySpecID[artifactID]
	return p, ok
}

// ResolveArtifactStatus walks projectRoot's issues/ specs/ plans/ directives/ directories
// and returns one record per artifact carrying (id, kind, declared status, class,
// mandated tests), plus the issue->plan linkage. Spec mandated tests are surfaced via the
// reused ExtractMandatedTests machinery; issues use the identical claim->tests walk;
// plans/directives carry status only. A missing directory is not an error (a repo need
// not have all four); unparseable files are skipped.
func ResolveArtifactStatus(projectRoot string) (*ArtifactStatusResolution, error) {
	res := &ArtifactStatusResolution{plansBySpecID: map[string]ArtifactStatusRecord{}}

	// Specs: reuse ExtractMandatedTests for the claim->tests machinery (it groups tests
	// per spec by SpecID). Terminal (retired) specs are dropped by ExtractMandatedTests,
	// which is fine — retired specs are excluded from drift anyway; their records still
	// get status/class below.
	specDir := filepath.Join(projectRoot, "specs")
	specTests := map[string][]MandatedTest{}
	if mts, err := ExtractMandatedTests(specDir); err == nil {
		for _, mt := range mts {
			specTests[mt.SpecID] = append(specTests[mt.SpecID], mt)
		}
	}
	if err := walkArtifactDir(specDir, ".spec.md", func(path string, fm *artifactFrontmatter) {
		id := fm.Number
		res.Records = append(res.Records, ArtifactStatusRecord{
			ID:            id,
			Kind:          KindSpec,
			Status:        fm.Status,
			Class:         ClassifyArtifactStatus(KindSpec, fm.Status),
			Path:          path,
			MandatedTests: specTests[id],
		})
	}); err != nil {
		return nil, fmt.Errorf("resolving specs under %s: %w", specDir, err)
	}

	// Issues: parse issue frontmatter (issue.id, issue.status) + the SAME claims[].tests
	// walk, building MandatedTest records (reusing the MandatedTest type). Unlike specs,
	// issues are resolved regardless of status so closed issues keep their mandated tests
	// — the closed+absent BLOCK case depends on it.
	if err := walkArtifactDir(filepath.Join(projectRoot, "issues"), ".issue.md", func(path string, fm *artifactFrontmatter) {
		id := fm.Issue.ID
		res.Records = append(res.Records, ArtifactStatusRecord{
			ID:            id,
			Kind:          KindIssue,
			Status:        fm.Issue.Status,
			Class:         ClassifyArtifactStatus(KindIssue, fm.Issue.Status),
			Path:          path,
			MandatedTests: claimsToMandatedTests(fm.Claims, id, path),
		})
	}); err != nil {
		return nil, fmt.Errorf("resolving issues: %w", err)
	}

	// Directives: status only (directive.status); no mandated tests.
	if err := walkArtifactDir(filepath.Join(projectRoot, "directives"), ".directive.md", func(path string, fm *artifactFrontmatter) {
		res.Records = append(res.Records, ArtifactStatusRecord{
			ID:     fm.Number,
			Kind:   KindDirective,
			Status: fm.Directive.Status,
			Class:  ClassifyArtifactStatus(KindDirective, fm.Directive.Status),
			Path:   path,
		})
	}); err != nil {
		return nil, fmt.Errorf("resolving directives: %w", err)
	}

	// Plans: whole-file YAML (.plan.yml). status + plan_id + spec_id, plus task test_names
	// as MandatedTests (ISSUE-048). Index each plan by its spec_id for the issue->plan
	// linkage.
	if err := walkPlanDir(filepath.Join(projectRoot, "plans"), func(path string, fm *planFrontmatter) {
		rec := ArtifactStatusRecord{
			ID:            fm.PlanID,
			Kind:          KindPlan,
			Status:        fm.Status,
			Class:         ClassifyArtifactStatus(KindPlan, fm.Status),
			Path:          path,
			SpecID:        fm.SpecID,
			MandatedTests: planTaskMandatedTests(fm, path),
		}
		res.Records = append(res.Records, rec)
		if fm.SpecID != "" {
			res.plansBySpecID[fm.SpecID] = rec
		}
	}); err != nil {
		return nil, fmt.Errorf("resolving plans: %w", err)
	}

	return res, nil
}

// claimsToMandatedTests applies the claim->tests walk to issue claims, building
// MandatedTest records that reuse the shared type (the SpecFile carries the issue path
// and SpecID the issue id, so a drift violation can attribute back to the artifact).
func claimsToMandatedTests(claims []claimBlock, artifactID, path string) []MandatedTest {
	var out []MandatedTest
	for _, c := range claims {
		for _, name := range c.Tests {
			out = append(out, MandatedTest{
				FuncName: name,
				SpecFile: path,
				SpecID:   artifactID,
				ClaimID:  c.ID,
			})
		}
	}
	return out
}

// artifactFrontmatter is the union of the fields the resolver reads across the fenced
// (.md) artifact types. yaml ignores keys a given file does not carry, so one struct
// serves issues (issue.*), specs (number/status/claims), and directives (directive.*).
type artifactFrontmatter struct {
	Number string `yaml:"number"`
	Status string `yaml:"status"`
	Issue  struct {
		ID     string `yaml:"id"`
		Status string `yaml:"status"`
	} `yaml:"issue"`
	Directive struct {
		Status string `yaml:"status"`
	} `yaml:"directive"`
	Claims []claimBlock `yaml:"claims"`
}

// claimBlock is the minimal claim shape the resolver reads: the claim id and its mandated
// test names. It mirrors the spec/issue schema claim shape (id + tests).
type claimBlock struct {
	ID    string   `yaml:"id"`
	Tests []string `yaml:"tests"`
}

// planFrontmatter is the whole-file plan YAML shape the resolver reads (plans are pure
// .plan.yml, not fenced markdown). It also reads phases[].tasks[].test_names (ISSUE-048):
// the mandated test names a completed plan delivered, so a `completed` plan is held to the
// SAME success-terminal absent-test BLOCK as issues/specs. yaml ignores tasks that carry
// no test_names, so plans authored before this field carry no MandatedTests (unchanged).
type planFrontmatter struct {
	PlanID string          `yaml:"plan_id"`
	SpecID string          `yaml:"spec_id"`
	Status string          `yaml:"status"`
	Phases []planPhaseNode `yaml:"phases"`
}

// planPhaseNode / planTaskNode are the minimal phase/task shape the resolver reads to
// surface task test_names. They deliberately read ONLY id + test_names — validate.Plan
// owns full plan-structure validation.
type planPhaseNode struct {
	Tasks []planTaskNode `yaml:"tasks"`
}

type planTaskNode struct {
	ID        string   `yaml:"id"`
	TestNames []string `yaml:"test_names"`
}

// planTaskMandatedTests flattens a plan's phases[].tasks[].test_names into MandatedTest
// records (reusing the shared type). SpecFile carries the plan path and SpecID the plan id
// so a drift violation attributes back to the plan; ClaimID carries the delivering task id.
func planTaskMandatedTests(fm *planFrontmatter, path string) []MandatedTest {
	var out []MandatedTest
	for _, phase := range fm.Phases {
		for _, task := range phase.Tasks {
			for _, name := range task.TestNames {
				out = append(out, MandatedTest{
					FuncName: name,
					SpecFile: path,
					SpecID:   fm.PlanID,
					ClaimID:  task.ID,
				})
			}
		}
	}
	return out
}

// walkArtifactDir reads every fenced-frontmatter artifact file with the given suffix in
// dir, unmarshals its frontmatter, and invokes visit. A missing dir is a no-op (not an
// error); unparseable files are skipped.
func walkArtifactDir(dir, suffix string, visit func(path string, fm *artifactFrontmatter)) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading artifact dir %s: %w", dir, err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), suffix) {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		raw, rErr := readFencedFrontmatter(path)
		if rErr != nil {
			continue
		}
		var fm artifactFrontmatter
		if yaml.Unmarshal(raw, &fm) != nil {
			continue
		}
		visit(path, &fm)
	}
	return nil
}

// walkPlanDir reads every .plan.yml in dir as whole-file YAML and invokes visit.
func walkPlanDir(dir string, visit func(path string, fm *planFrontmatter)) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading plan dir %s: %w", dir, err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".plan.yml") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		raw, rErr := os.ReadFile(path)
		if rErr != nil {
			continue
		}
		var fm planFrontmatter
		if yaml.Unmarshal(raw, &fm) != nil {
			continue
		}
		visit(path, &fm)
	}
	return nil
}

// readFencedFrontmatter returns the YAML frontmatter bytes between the opening and closing
// `---` fences of a markdown artifact file. It reads the whole file (no open/close handle to
// leak or leave unchecked) and returns os.ErrNotExist when the file lacks an opening fence,
// so the caller skips it. It returns raw bytes so the caller unmarshals into the type it
// needs.
func readFencedFrontmatter(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return nil, os.ErrNotExist
	}
	var fm []string
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "---" {
			break
		}
		fm = append(fm, line)
	}
	return []byte(strings.Join(fm, "\n")), nil
}
