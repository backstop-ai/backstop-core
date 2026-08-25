package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/artifact"
	"github.com/backstop-ai/backstop-core/pkg/gate"
	"github.com/backstop-ai/backstop-core/pkg/pack"
	"github.com/backstop-ai/backstop-core/pkg/pack/distribution"
	"github.com/backstop-ai/backstop-core/pkg/packval"
	"gopkg.in/yaml.v3"
)

const issue188Commit = "ed221e896fab2209bbbdcece0e96d28e22e27002"
const issue188Tree = "0fee4b18605c74090b2271cf312a753ec614c432"

var issue188FixtureHashes = map[string]string{
	"pack.yml":                 "1fe143f9413a11a57951b9dee5e57216ca44ff33c5179bfd73fca27abfbd7a20",
	"scripts/test-produce.sh":  "670d155fe83732bfb5b7a508ffb11cc75d78557b550cebe6df6f3d2325433999",
	"scripts/test-to-sarif.sh": "046ad294dfa6a643a66d2f0a38fc6cebbe39e73e59fcbf9d1fc777ff56b43912",
}

var issue188FormerlyHidden = []string{
	"verify_claim_type_evidence_matrix_negative",
	"verify_compatibility_claim_rejects_guarantee_equivalence",
	"verify_consequential_claim_rejects_incomplete_record",
	"verify_product_boundary_rejects_invalid_claim_linkage",
	"verify_product_boundary_rejects_invalid_structured_fields",
	"verify_product_boundary_state_matrix_negative",
	"verify_required_architecture_view_inventory",
	"verify_required_architecture_view_rejects_missing_or_invalid_view",
}

type issue188SpecFrontmatter struct {
	Claims []struct {
		Tests []string `yaml:"tests"`
	} `yaml:"claims"`
}

func issue188Root(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(out))
}

func issue188Fixture(t *testing.T) (*pack.Manifest, string) {
	t.Helper()
	root := filepath.Join(issue188Root(t), "cmd", "backstop", "testdata", "issue188-bash-reference-pack")
	for rel, want := range issue188FixtureHashes {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatal(err)
		}
		if got := fmt.Sprintf("%x", sha256.Sum256(data)); got != want {
			t.Fatalf("fixture %s digest %s, want %s", rel, got, want)
		}
	}
	manifest, err := pack.ParseManifestFile(filepath.Join(root, "pack.yml"))
	if err != nil {
		t.Fatal(err)
	}
	return manifest, root
}

func stageIssue188Corpus(t *testing.T) string {
	t.Helper()
	root := issue188Root(t)
	tree, err := exec.Command("git", "-C", root, "rev-parse", issue188Commit+"^{tree}").Output()
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(tree)) != issue188Tree {
		t.Fatal("pinned tree mismatch")
	}
	out, err := exec.Command(
		"git", "-C", root, "ls-tree", "-r", "--name-only", issue188Commit,
		"scripts/tests/public-product-model",
		"scripts/verify-public-product-model.sh",
		"specs/SPEC-072-public-product-model.spec.md",
		"plans/PLAN-SPEC-072-public-product-model.plan.yml",
	).Output()
	if err != nil {
		t.Fatal(err)
	}
	destination := t.TempDir()
	for _, rel := range strings.Fields(string(out)) {
		data, showErr := exec.Command("git", "-C", root, "show", issue188Commit+":"+rel).Output()
		if showErr != nil {
			t.Fatal(showErr)
		}
		path := filepath.Join(destination, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return destination
}

func issue188AllReferences(t *testing.T, root string) []string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, "specs", "SPEC-072-public-product-model.spec.md"))
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(data), "---", 3)
	if len(parts) != 3 {
		t.Fatal("SPEC-072 frontmatter is malformed")
	}
	var frontmatter issue188SpecFrontmatter
	if err := yaml.Unmarshal([]byte(parts[1]), &frontmatter); err != nil {
		t.Fatal(err)
	}
	var references []string
	for _, claim := range frontmatter.Claims {
		references = append(references, claim.Tests...)
	}
	return references
}

func issue188Discovered(t *testing.T, root string, legacy bool) map[string]string {
	t.Helper()
	manifest, _ := issue188Fixture(t)
	classifier := gate.NewSourceClassifier(nil, manifest.Classification.Test)
	matcher, err := gate.NewTestNameMatcher(manifest.TestNamePatterns)
	if err != nil {
		t.Fatal(err)
	}
	legacyPatterns := make([]*regexp.Regexp, 0, len(manifest.TestNamePatterns))
	for _, pattern := range manifest.TestNamePatterns {
		compiled, compileErr := regexp.Compile(pattern)
		if compileErr != nil {
			t.Fatal(compileErr)
		}
		legacyPatterns = append(legacyPatterns, compiled)
	}
	found := map[string]string{}
	walkErr := filepath.WalkDir(filepath.Join(root, "scripts"), func(path string, entry os.DirEntry, entryErr error) error {
		if entryErr != nil {
			return entryErr
		}
		if entry.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		if !classifier.IsTestFile(rel) {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for _, line := range strings.Split(string(data), "\n") {
			if legacy {
				for _, pattern := range legacyPatterns {
					submatch := pattern.FindStringSubmatch(line)
					if len(submatch) > 1 && submatch[1] != "" {
						found[submatch[1]] = path
						break
					}
				}
				continue
			}
			for _, name := range matcher.FindNames(line) {
				found[name] = path
			}
		}
		return nil
	})
	if walkErr != nil {
		t.Fatal(walkErr)
	}
	return found
}

func issue188Distinct(values []string) map[string]bool {
	distinct := make(map[string]bool, len(values))
	for _, value := range values {
		distinct[value] = true
	}
	return distinct
}

func issue188Snapshot(t *testing.T, root string) map[string][32]byte {
	t.Helper()
	snapshot := map[string][32]byte{}
	walkErr := filepath.WalkDir(root, func(path string, entry os.DirEntry, entryErr error) error {
		if entryErr != nil {
			return entryErr
		}
		if entry.IsDir() {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		snapshot[filepath.ToSlash(rel)] = sha256.Sum256(data)
		return nil
	})
	if walkErr != nil {
		t.Fatal(walkErr)
	}
	return snapshot
}

func installIssue188Fixture(t *testing.T, root string) {
	t.Helper()
	manifest, fixtureRoot := issue188Fixture(t)
	installedRoot := filepath.Join(root, ".backstop", "packs", filepath.FromSlash(manifest.Name))
	walkErr := filepath.WalkDir(fixtureRoot, func(path string, entry os.DirEntry, entryErr error) error {
		if entryErr != nil {
			return entryErr
		}
		rel, relErr := filepath.Rel(fixtureRoot, path)
		if relErr != nil {
			return relErr
		}
		target := filepath.Join(installedRoot, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		mode := os.FileMode(0o644)
		if strings.HasPrefix(filepath.ToSlash(rel), "scripts/") {
			mode = 0o755
		}
		return os.WriteFile(target, data, mode)
	})
	if walkErr != nil {
		t.Fatal(walkErr)
	}
	configText := "project: issue188-fixture\npacks:\n  " + manifest.Name + ": local\n"
	if err := os.WriteFile(filepath.Join(root, "backstop.yml"), []byte(configText), 0o644); err != nil {
		t.Fatal(err)
	}
	contentHash, err := distribution.ComputeContentHash(installedRoot)
	if err != nil {
		t.Fatal(err)
	}
	lock := &distribution.Lockfile{Packs: map[string]distribution.LockEntry{
		manifest.Name: {
			Name:        manifest.Name,
			ContentHash: contentHash,
			SourceType:  "local",
			InstallDate: "2026-08-25T00:00:00Z",
			LocalPath:   fixtureRoot,
		},
	}}
	if err := distribution.WriteLockfile(filepath.Join(root, "backstop.lock"), lock); err != nil {
		t.Fatal(err)
	}
}

func TestGate_SPEC072DiscoversAll44BashFunctionsAcrossSameLineDeclarations(t *testing.T) {
	root := stageIssue188Corpus(t)
	references := issue188AllReferences(t, root)
	if len(references) != 47 {
		t.Fatalf("SPEC-072 references = %d, want 47", len(references))
	}
	distinctReferences := issue188Distinct(references)
	if len(distinctReferences) != 44 {
		t.Fatalf("SPEC-072 distinct references = %d, want 44", len(distinctReferences))
	}
	control := issue188Discovered(t, root, true)
	corrected := issue188Discovered(t, root, false)
	if len(control) != 36 || len(corrected) != 44 {
		t.Fatalf("discovery control=%d corrected=%d, want 36/44", len(control), len(corrected))
	}
	for _, name := range issue188FormerlyHidden {
		if _, ok := control[name]; ok {
			t.Errorf("legacy first-match consumer unexpectedly found %s", name)
		}
		if _, ok := corrected[name]; !ok {
			t.Errorf("corrected consumer is missing %s", name)
		}
	}
	manifest, _ := issue188Fixture(t)
	matcher, err := gate.NewTestNameMatcher(manifest.TestNamePatterns)
	if err != nil {
		t.Fatal(err)
	}
	classifier := gate.NewSourceClassifier(nil, manifest.Classification.Test)
	mandated := make([]gate.MandatedTest, 0, len(references))
	for _, name := range references {
		mandated = append(mandated, gate.MandatedTest{FuncName: name})
	}
	resolved := gate.ResolveMandatedTestPaths(mandated, root, classifier, matcher)
	resolvedByName := make(map[string]string, len(resolved))
	for _, test := range resolved {
		if test.FilePath == "" {
			t.Errorf("mandated reference %s did not resolve", test.FuncName)
		}
		resolvedByName[test.FuncName] = filepath.ToSlash(test.FilePath)
	}
	for _, hidden := range issue188FormerlyHidden {
		if _, referenced := distinctReferences[hidden]; !referenced {
			t.Errorf("formerly hidden declaration %s is not an actual SPEC-072 reference", hidden)
		}
		path, ok := resolvedByName[hidden]
		if !ok || !strings.Contains(path, "scripts/tests/public-product-model/") {
			t.Errorf("formerly hidden %s resolved to unexpected path %s", hidden, path)
		}
	}
}

func TestGate_SPEC072TerminalDiscoveryAndDriftClearAfterAllMatchEnumeration(t *testing.T) {
	root := stageIssue188Corpus(t)
	before := issue188Snapshot(t, root)
	specPath := filepath.Join(root, "specs", "SPEC-072-public-product-model.spec.md")
	planPath := filepath.Join(root, "plans", "PLAN-SPEC-072-public-product-model.plan.yml")
	mutations := map[string][2]string{
		specPath: {"status: draft", "status: implemented"},
		planPath: {"status: draft", "status: completed"},
	}
	for path, pair := range mutations {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Count(string(data), pair[0]) != 1 {
			t.Fatalf("expected exactly one %q token in %s", pair[0], path)
		}
		changed := strings.Replace(string(data), pair[0], pair[1], 1)
		if err := os.WriteFile(path, []byte(changed), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	after := issue188Snapshot(t, root)
	allPaths := make(map[string]struct{}, len(before)+len(after))
	for path := range before {
		allPaths[path] = struct{}{}
	}
	for path := range after {
		allPaths[path] = struct{}{}
	}
	var changed []string
	var added []string
	var deleted []string
	for path := range allPaths {
		beforeDigest, existedBefore := before[path]
		afterDigest, existsAfter := after[path]
		switch {
		case !existedBefore:
			added = append(added, path)
		case !existsAfter:
			deleted = append(deleted, path)
		case beforeDigest != afterDigest:
			changed = append(changed, path)
		}
	}
	sort.Strings(changed)
	sort.Strings(added)
	sort.Strings(deleted)
	wantChanged := []string{
		"plans/PLAN-SPEC-072-public-product-model.plan.yml",
		"specs/SPEC-072-public-product-model.spec.md",
	}
	if len(added) != 0 || len(deleted) != 0 || strings.Join(changed, "\n") != strings.Join(wantChanged, "\n") {
		t.Fatalf("fixture changes: modified=%v added=%v deleted=%v; want only modified=%v", changed, added, deleted, wantChanged)
	}

	installIssue188Fixture(t, root)
	sandboxRunner, err := packval.NewSandboxRunner(packval.SandboxModeExternal)
	if err != nil {
		t.Fatal(err)
	}
	steps := buildGateStepsWithSandbox(root, artifact.Root{Path: root}, sandboxRunner)
	result, _ := gate.New(gate.WithSteps(steps)).Run(context.Background())
	results := map[string]gate.StepResult{}
	for _, step := range result.Steps {
		results[step.StepName] = step
	}
	for _, name := range []string{"pack_lock_verification", "pack_engines", gate.StepTestVerification, gate.StepArtifactStatusDrift} {
		step, ok := results[name]
		if !ok {
			t.Errorf("assembled gate omitted %s", name)
			continue
		}
		if step.Status != "pass" || len(step.Violations) != 0 {
			t.Errorf("assembled %s = status %s violations %+v", name, step.Status, step.Violations)
		}
	}
}
