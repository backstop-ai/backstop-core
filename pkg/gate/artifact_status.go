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

	"github.com/backstop-ai/backstop-core/pkg/artifact"
	"gopkg.in/yaml.v3"
)

// ArtifactKind names one of the four backstop artifact types the resolver walks.
type ArtifactKind string

const (
	KindIssue     ArtifactKind = "issue"
	KindSpec      ArtifactKind = "spec"
	KindPlan      ArtifactKind = "plan"
	KindDirective ArtifactKind = "directive"
	KindBundle    ArtifactKind = "bundle" // nosemgrep: go.core.no-global-mutable-state — immutable const, not mutable global
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
	case KindBundle:
		if status == "delivered" {
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
	BundleName    string
	BundleReqs    []BundleReqVersion
}

// BundleReqVersion is the current declared version of one bundle requirement.
type BundleReqVersion struct {
	ReqID          string
	CurrentVersion string
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

// ResolveArtifactStatus walks the RESOLVED ARTIFACT ROOT's issues/ specs/ plans/
// directives/ bundles/ directories and returns one record per artifact carrying (id,
// kind, declared status, class, mandated tests), plus the issue->plan linkage. Spec
// mandated tests are surfaced via the reused ExtractMandatedTests machinery; issues use
// the identical claim->tests walk; plans/directives carry status only. A missing
// directory is not an error (a repo need not have all five); unparseable files are
// skipped.
//
// artifactRoot is the root artifact.ResolveRoot produced, NOT the project root. For an
// unconfigured project the two are the same directory, which is why every caller that
// predates SPEC-068 keeps working unchanged. Every type directory name and extension
// comes from the SHARED artifact layout table; this walker carries no private copy.
//
// THE MISSING-DIRECTORY TOLERANCE in walkArtifactDir / walkPlanDir / walkBundleDir is
// LOAD-BEARING and must not be "hardened": it is what keeps an existing-but-empty
// artifact root a pass. The loud failure for a configured root that is ABSENT lives one
// layer up, at the ROOT, where REQ-008 puts it.
//
// The walk is deliberately NON-RECURSIVE. That is the calibration point the REQ-008
// ungated-artifact scan is defined against — making it recursive would silently empty
// that scan's finding set.
func ResolveArtifactStatus(artifactRoot string) (*ArtifactStatusResolution, error) {
	res := &ArtifactStatusResolution{plansBySpecID: map[string]ArtifactStatusRecord{}}
	root := artifact.Root{Path: artifactRoot}

	// Specs: reuse ExtractMandatedTests for the claim->tests machinery (it groups tests
	// per spec by SpecID). Terminal (retired) specs are dropped by ExtractMandatedTests,
	// which is fine — retired specs are excluded from drift anyway; their records still
	// get status/class below.
	specDir := root.Dir(artifact.KindSpec)
	specTests := map[string][]MandatedTest{}
	if mts, err := ExtractMandatedTests(specDir); err == nil {
		for _, mt := range mts {
			specTests[mt.SpecID] = append(specTests[mt.SpecID], mt)
		}
	}
	if err := walkArtifactDir(specDir, extensionFor(artifact.KindSpec), func(path string, fm *artifactFrontmatter) {
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

	// Bundles: status is nested under status.maturity, and joins use bundle.name.
	if err := walkBundleDir(root.Dir(artifact.KindBundle), func(path string, fm *bundleFrontmatter) {
		res.Records = append(res.Records, ArtifactStatusRecord{
			ID:         fm.Number,
			Kind:       KindBundle,
			Status:     fm.Status.Maturity,
			Class:      ClassifyArtifactStatus(KindBundle, fm.Status.Maturity),
			Path:       path,
			BundleName: fm.Bundle.Name,
			BundleReqs: bundleReqs(fm),
		})
	}); err != nil {
		return nil, fmt.Errorf("resolving bundles: %w", err)
	}

	// Issues: parse issue frontmatter (issue.id, issue.status) + the SAME claims[].tests
	// walk, building MandatedTest records (reusing the MandatedTest type). Unlike specs,
	// issues are resolved regardless of status so closed issues keep their mandated tests
	// — the closed+absent BLOCK case depends on it.
	if err := walkArtifactDir(root.Dir(artifact.KindIssue), extensionFor(artifact.KindIssue), func(path string, fm *artifactFrontmatter) {
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
	if err := walkArtifactDir(root.Dir(artifact.KindDirective), extensionFor(artifact.KindDirective), func(path string, fm *artifactFrontmatter) {
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
	if err := walkPlanDir(root.Dir(artifact.KindPlan), func(path string, fm *planFrontmatter) {
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

// bundleFrontmatter is the minimal bundle shape needed by traceability: bundle.name,
// nested status.maturity, and current requirement ids/versions.
type bundleFrontmatter struct {
	Number string `yaml:"number"`
	Bundle struct {
		Name string `yaml:"name"`
	} `yaml:"bundle"`
	Status struct {
		Maturity string `yaml:"maturity"`
	} `yaml:"status"`
	Requirements []bundleReqNode `yaml:"requirements"`
}

type bundleReqNode struct {
	ID      string `yaml:"id"`
	Version string `yaml:"version"`
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

func bundleReqs(fm *bundleFrontmatter) []BundleReqVersion {
	var out []BundleReqVersion
	for _, req := range fm.Requirements {
		if req.ID == "" {
			continue
		}
		out = append(out, BundleReqVersion{ReqID: req.ID, CurrentVersion: req.Version})
	}
	return out
}

// unknownKindExtension is what extensionFor yields for a kind the shared layout table
// does not cover. It is deliberately unmatchable: an empty string would make every
// strings.HasSuffix test true, turning a missing table entry into a walk over every file
// in the directory rather than none.
const unknownKindExtension = "\x00no-such-kind"

// extensionFor reads one kind's file extension from the SHARED layout table, so no
// extension string in this file is a private copy.
func extensionFor(kind artifact.Kind) string {
	layout, known := artifact.LayoutFor(kind)
	if !known {
		return unknownKindExtension
	}
	return layout.Extension
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
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), extensionFor(artifact.KindPlan)) {
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

// walkBundleDir reads every fenced bundle artifact in dir and invokes visit. Bundles need
// a dedicated frontmatter struct because their status is nested under status.maturity.
func walkBundleDir(dir string, visit func(path string, fm *bundleFrontmatter)) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading bundle dir %s: %w", dir, err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), extensionFor(artifact.KindBundle)) {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		raw, rErr := readFencedFrontmatter(path)
		if rErr != nil {
			continue
		}
		var fm bundleFrontmatter
		if yaml.Unmarshal(raw, &fm) != nil {
			continue
		}
		visit(path, &fm)
	}
	return nil
}

// UngatedArtifact is one artifact-shaped file the status walk cannot reach: it is not
// DIRECTLY in the type directory its own filename declares.
//
// ExpectedDir is where discovery WOULD have found it, and it is what makes the report
// actionable rather than merely accusatory.
type UngatedArtifact struct {
	Path        string
	Kind        artifact.Kind
	ExpectedDir string
	Root        string
}

// Message is the human-readable report for one ungated artifact.
//
// UNGATED IS NOT UNDISCOVERED, and this wording is load-bearing. CLI discovery walks
// the resolved artifact root RECURSIVELY, so a file nested below its own type directory
// is discovered and schema-validated while still being ungated by the NON-RECURSIVE
// status walk. Every file in this set is ungated; only a file discovery genuinely
// cannot reach is undiscovered, and this message never claims that.
func (u UngatedArtifact) Message() string {
	return fmt.Sprintf(
		"artifact-shaped file %s is UNGATED: the artifact status walk reads %s directly and this file is not there",
		u.Path, u.ExpectedDir)
}

// FindUngatedArtifacts walks projectRoot and reports every artifact-shaped file that is
// not DIRECTLY inside the type directory its filename declares.
//
// THE PREDICATE IS PER KIND, NOT ROOT CONTAINMENT. A containment test passes almost
// every case in this family and fails the motivating one: a project that configures no
// artifact root and keeps bundles at .backstop/bundles/ has those files INSIDE the
// default root, because the project root contains itself. Per-kind is what surfaces
// them. The function is named FindUngatedArtifacts precisely so the name cannot
// re-suggest containment.
//
// THE EXCLUSION SET IS ENUMERATED, NOT INHERITED. The five shared non-corpus names come
// from artifact.NonCorpusDirNames, and installed pack trees under .backstop/packs are
// excluded on top — but `.backstop` ITSELF IS WALKED, unlike in CLI discovery. Phrasing
// this as "whatever discovery skips" would make the clause report NOTHING in the
// unconfigured case, which is the case it exists for.
//
// It is CALIBRATED against the non-recursive status walk above. Making that walk
// recursive would silently empty this function's finding set.
func FindUngatedArtifacts(projectRoot string, root artifact.Root) ([]UngatedArtifact, error) {
	// Absolutize on entry. Root.Path is already absolute by ResolveRoot's guarantee,
	// and comparing an absolute expected directory against a relative walk path yields
	// either zero findings or one per artifact — both of which look like a working
	// implementation.
	absProject, err := filepath.Abs(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("resolving project root %q: %w", projectRoot, err)
	}
	// AND resolve symlinks, on BOTH sides of every comparison below. Absolutizing alone
	// is not enough: os.Getwd returns a symlink-RESOLVED path, so a caller that passes
	// "." (which runGate does whenever config-path discovery fails) produces walk paths
	// under /private/var while a Root resolved from the same directory's /var form
	// produces expected directories under /var. Nothing matches, and the result is one
	// finding per artifact — which looks exactly like a working implementation.
	absProject = resolveSymlinks(absProject)
	root.Path = resolveSymlinks(root.Path)

	nonCorpus := make(map[string]bool)
	for _, name := range artifact.NonCorpusDirNames() {
		nonCorpus[name] = true
	}

	var found []UngatedArtifact
	walkErr := filepath.Walk(absProject, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			base := filepath.Base(path)
			if nonCorpus[base] {
				return filepath.SkipDir
			}
			// Installed pack trees are never the consumer's corpus, and keying on the
			// PARENT name covers both layouts with one rule.
			if base == "packs" && filepath.Base(filepath.Dir(path)) == ".backstop" {
				return filepath.SkipDir
			}
			return nil
		}

		kind, ok := artifact.ClassifyFilename(info.Name())
		if !ok {
			return nil
		}
		expected := root.Dir(kind)
		if filepath.Dir(path) == expected {
			return nil
		}
		found = append(found, UngatedArtifact{
			Path:        path,
			Kind:        kind,
			ExpectedDir: expected,
			Root:        root.Path,
		})
		return nil
	})
	if walkErr != nil {
		return nil, fmt.Errorf("walking project root %s: %w", absProject, walkErr)
	}

	return found, nil
}

// resolveSymlinks returns path with symlinks resolved, or path unchanged when it cannot
// be resolved (it may not exist yet, and a non-existent path is not an error here — the
// walk below reports that in its own terms).
func resolveSymlinks(path string) string {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return path
	}
	return resolved
}

// UngatedFindingsToViolations converts ungated-artifact findings into gate violations,
// marking EVERY one ProjectWide.
//
// The conversion lives here, in ONE place, so the marking cannot be forgotten at a call
// site. It is load-bearing rather than cosmetic: filterViolations keeps a violation only
// when it is ProjectWide or its File is in the diff scope, and an ungated finding names
// a file that is BY DEFINITION not in the diff — so without the marking a bare
// `backstop gate` would silently drop exactly the findings this scan exists to surface.
func UngatedFindingsToViolations(found []UngatedArtifact) []Violation {
	out := make([]Violation, 0, len(found))
	for _, f := range found {
		out = append(out, Violation{
			Rule:        StepArtifactValidation,
			File:        f.Path,
			Message:     f.Message(),
			Severity:    "warning",
			ProjectWide: true,
		})
	}
	return out
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
