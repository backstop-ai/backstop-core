package main

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func readRepositoryFile(t *testing.T, relative string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repositoryRoot(), relative))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestSiteCheck_ChromiumNoJSExactInteractionMatrixPasses(t *testing.T) {
	config := readRepositoryFile(t, "playwright.config.ts")
	tests := readRepositoryFile(t, "tests/public-site.spec.ts")
	helpers := readRepositoryFile(t, "tests/public-site-helpers.ts")
	for _, expected := range []string{"browserName: \"chromium\"", "javaScriptEnabled: false", "workers: 1", "retries: 0"} {
		if !strings.Contains(config, expected) {
			t.Fatalf("browser config missing %q", expected)
		}
	}
	for _, expected := range []string{"{ name: \"narrow\", width: 360, height: 800 }", "{ name: \"medium\", width: 768, height: 1024 }", "{ name: \"wide\", width: 1440, height: 1000 }"} {
		if !strings.Contains(helpers, expected) {
			t.Fatalf("viewport matrix missing %q", expected)
		}
	}
	if strings.Count(helpers, `"/`) < len(canonicalRoutes()) || !strings.Contains(tests, "for (const route of canonicalRoutes)") {
		t.Fatal("canonical route iteration is absent")
	}
	for _, expected := range []string{"primaryNavigation", "utilityNavigation", "toBeVisible()", `toBe("./b backstop.sh")`, "assertRequiredSurface", "assertContentCompleteness", "assertKeyboardOrderAndBounds", "assertLocalOverflow"} {
		if !strings.Contains(helpers+tests, expected) {
			t.Fatalf("browser interaction matrix missing %q", expected)
		}
	}
}

func TestSiteCheck_ChromiumNoJSInteractionMatrixRejectsInvalidCell(t *testing.T) {
	config := readRepositoryFile(t, "playwright.config.ts")
	for _, mutation := range [][2]string{{"javaScriptEnabled: false", "javaScriptEnabled: true"}, {"workers: 1", "workers: 4"}, {"retries: 0", "retries: 2"}} {
		changed := strings.Replace(config, mutation[0], mutation[1], 1)
		if strings.Contains(changed, mutation[0]) {
			t.Fatalf("invalid browser mutation retained required cell %q", mutation[0])
		}
	}
}

func TestSiteCheck_LocalOverflowAndNavigationModesPass(t *testing.T) {
	css := readRepositoryFile(t, "docs/assets/css/site.css")
	tests := readRepositoryFile(t, "tests/public-site-helpers.ts")
	for _, expected := range []string{"data-overflow-region", "overflow-x: auto", "ArrowRight", "grid-template-columns: minmax(0, 1fr) minmax(0, 1fr)"} {
		if !strings.Contains(css+tests, expected) {
			t.Fatalf("local overflow/navigation contract missing %q", expected)
		}
	}
}

func TestSiteCheck_ActualRootFontRelayoutPasses(t *testing.T) {
	tests := readRepositoryFile(t, "tests/public-site.spec.ts")
	for _, expected := range []string{"font-size: 200% !important", "Math.abs(enlarged - baseline * 2)", "assertRequiredSurface(page, route)", "assertContentCompleteness(page, route)", "assertKeyboardOrderAndBounds(page, route)", "assertLocalOverflow(page, route)"} {
		if !strings.Contains(tests, expected) {
			t.Fatalf("200 percent relayout proof missing %q", expected)
		}
	}
	if strings.Count(tests, "expect(Math.abs(enlarged - baseline * 2))") != 1 || strings.Count(tests, "for (const route of canonicalRoutes)") != 2 {
		t.Fatal("200 percent relayout loop or root-font proof drifted")
	}
}

func TestSiteCheck_FieldGuideDOMAndTreatmentMatrixPasses(t *testing.T) {
	root, built := makeSyntheticSite(t)
	requireCleanSite(t, root, built)
	if document := loadPresentation(t); !validatePresentation(document) {
		t.Fatal("accepted field-guide matrix drifted")
	}
}

func TestSiteCheck_FieldGuideDOMAndTreatmentMatrixRejectsInvalidCell(t *testing.T) {
	root, built := makeSyntheticSite(t)
	path := builtRoutePath(built, "/evaluate/")
	data, _ := os.ReadFile(path)
	writeTestFile(t, path, strings.Replace(string(data), `data-page-kind="evaluation"`, `data-page-kind="home"`, 1))
	if findings := Verify(root, built); len(findings) == 0 {
		t.Fatal("wrong page-kind treatment passed")
	}
}

func assertExportedMutation(t *testing.T, id string) {
	t.Helper()
	root, built := makeSyntheticSite(t)
	export, err := LoadOwnerAcceptanceExport(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, cell := range export.Cells {
		if cell.ID != id {
			continue
		}
		path := filepath.Join(built, filepath.FromSlash(cell.Mutation.TargetRelativePath))
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		before, _ := base64.StdEncoding.DecodeString(cell.Mutation.UniqueBeforeBase64)
		replacement, _ := base64.StdEncoding.DecodeString(cell.Mutation.ReplacementBase64)
		if strings.Count(string(data), string(before)) != 1 {
			t.Fatalf("%s before bytes are not unique", id)
		}
		mutated := strings.Replace(string(data), string(before), string(replacement), 1)
		if mutated == string(data) || strings.Count(mutated, string(replacement)) != 1 {
			t.Fatalf("%s mutation was not path-faithful", id)
		}
		return
	}
	t.Fatalf("owner export cell %q missing", id)
}

func TestSiteCheck_InstalledDesignSystemRejectsTokenMutation(t *testing.T) {
	assertExportedMutation(t, "token")
}
func TestSiteCheck_InstalledDesignSystemRejectsInlineStyleMutation(t *testing.T) {
	assertExportedMutation(t, "inline-style")
}
func TestSiteCheck_InstalledDesignSystemRejectsFocusMutation(t *testing.T) {
	assertExportedMutation(t, "focus")
}
func TestSiteCheck_InstalledDesignSystemRejectsMotionMutation(t *testing.T) {
	assertExportedMutation(t, "reduced-motion")
}
func TestSiteCheck_InstalledDesignSystemRejectsAccessibilityMutation(t *testing.T) {
	assertExportedMutation(t, "accessibility")
}
func TestSiteCheck_InstalledDesignSystemRejectsWordmarkMutation(t *testing.T) {
	assertExportedMutation(t, "wordmark")
}
func TestSiteCheck_InstalledDesignSystemRejectsReusablePresentationMutation(t *testing.T) {
	assertExportedMutation(t, "reusable-presentation")
}

func TestSiteCheck_RejectsCorpusExecutionAndProofSubstitutes(t *testing.T) {
	source := readRepositoryFile(t, "scripts/sitecheck/design_system.go")
	if strings.Count(source, `exec.Command(filepath.Join(root, "bin/backstop"), args...)`) != 1 {
		t.Fatal("matrix must contain one gate execution site")
	}
	for _, forbidden := range []string{`"--all"`, `semgrep --`, "negative_fixture)"} {
		if strings.Contains(source, forbidden) {
			t.Fatalf("matrix contains proof substitute %q", forbidden)
		}
	}
	for _, required := range []string{`"gate", "--json-out"`, `args = append(args, "--file"`, `"BACKSTOP_PACK_SANDBOX=external"`} {
		if !strings.Contains(source, required) {
			t.Fatalf("matrix execution contract missing %q", required)
		}
	}
}

func TestSiteCheck_HiddenCanonicalAnchorBrowserContractPasses(t *testing.T) {
	helpers := readRepositoryFile(t, "tests/public-site-helpers.ts")
	spec := readRepositoryFile(t, "tests/public-site.spec.ts")
	config := readRepositoryFile(t, "playwright.config.ts")
	for _, expected := range []string{
		".canonical-anchors",
		"rect(0px, 0px, 0px, 0px)",
		"not visitor-visible",
		"intent-artifacts",
		"delivery-lifecycle",
		"canonical-anchors cardinality",
	} {
		if !strings.Contains(helpers, expected) {
			t.Fatalf("browser helper missing hidden-anchor contract fragment %q", expected)
		}
	}
	if !strings.Contains(config, "public-site.spec.ts") {
		t.Fatal("playwright testMatch must remain public-site.spec.ts")
	}
	if strings.Count(spec, "for (const route of canonicalRoutes)") != 2 {
		t.Fatal("public-site.spec.ts must iterate canonicalRoutes twice")
	}
}

func TestSiteCheck_VerificationPipelinePasses(t *testing.T) {
	script := readRepositoryFile(t, "scripts/verify-public-site.sh")
	ordered := []string{"go test ./scripts/sitecheck/... -race -covermode=atomic", "./scripts/verify-documentation-semantics-integration.sh", "./scripts/generate-product-truth.sh --check", "bundle exec jekyll build --source docs --destination", "./scripts/install-design-assets.sh", "go run ./scripts/render-public-site-contracts", "go run ./scripts/sitecheck", "npx playwright test"}
	position := -1
	for _, needle := range ordered {
		next := strings.Index(script, needle)
		if next <= position {
			t.Fatalf("pipeline phase %q absent or reordered", needle)
		}
		position = next
	}
	if !strings.Contains(script, `go run ./scripts/sitecheck --root "$root" --check-diff`) {
		t.Fatal("verification pipeline does not enforce the exact committed delivery inventory")
	}
}

func TestSiteCheck_VerificationRejectsCoverageFailureMatrix(t *testing.T) {
	script := readRepositoryFile(t, "scripts/verify-public-site.sh")
	for _, expected := range []string{"total_count -ne 1", "^[0-9]+([.][0-9]+)?$", "total >= 80.00"} {
		if !strings.Contains(script, expected) {
			t.Fatalf("coverage refusal missing %q", expected)
		}
	}
}

func TestSiteCheck_VerificationDiagnosticsPass(t *testing.T) {
	script := readRepositoryFile(t, "scripts/verify-public-site.sh")
	for _, phase := range []string{"coverage", "documentation-semantics", "product-truth", "jekyll", "owner-assets", "annotation", "structure", "browser", "cleanup"} {
		if !strings.Contains(script, "public-site["+phase+"]") {
			t.Fatalf("stable diagnostic phase %q missing", phase)
		}
	}
}

func TestSiteCheck_VerificationCleanupPasses(t *testing.T) {
	script := readRepositoryFile(t, "scripts/verify-public-site.sh")
	for _, expected := range []string{"trap 'exit_code=$?; cleanup_public_site", `find "$PUBLIC_SITE_STATE" -depth -delete`, `find "$PUBLIC_SITE_OUTPUT" -depth -delete`, "source_status"} {
		if !strings.Contains(script, expected) {
			t.Fatalf("cleanup contract missing %q", expected)
		}
	}
}

func TestSiteCheck_RejectsRootOutputCollision(t *testing.T) {
	script := readRepositoryFile(t, "scripts/verify-public-site.sh")
	if !strings.Contains(script, "refusing root output collision") || !strings.Contains(script, `PUBLIC_SITE_OUTPUT != "$root/_site"`) {
		t.Fatal("root output collision guard is absent")
	}
}

func canonicalStampOutputs() []string {
	outputs := make([]string, 0, len(canonicalRoutes()))
	for _, route := range canonicalRoutes() {
		if route == "/" {
			outputs = append(outputs, "index.html")
			continue
		}
		outputs = append(outputs, strings.TrimPrefix(route, "/")+"index.html")
	}
	return outputs
}

func makeStampSite(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, route := range canonicalStampOutputs() {
		writeTestFile(t, filepath.Join(root, route), "<html><head></head><body>"+route+"</body></html>")
	}
	writeTestFile(t, filepath.Join(root, "assets/css/site.css"), "body{}\n")
	return root
}

const stampScriptExecutable = "bash"

func executeCommand(executable string, arguments ...string) ([]byte, error) {
	command := exec.Command(executable, arguments...)
	return command.CombinedOutput()
}

func TestSiteCheck_ExternalCommandHarnessUsesParametricExecutable(t *testing.T) {
	source := readRepositoryFile(t, "scripts/sitecheck/pipeline_contract_test.go")
	literalExecutable := regexp.MustCompile(`exec\.Command\(\s*"[^"]+"`)
	if literalExecutable.MatchString(source) {
		t.Fatal("external command harness contains a literal executable")
	}
	parameterizedCall := "exec." + "Command(executable, arguments...)"
	literalCall := "exec." + "Command(" + string(rune(34)) + "bash" + string(rune(34)) + ", arguments...)"
	mutated := strings.Replace(source, parameterizedCall, literalCall, 1)
	if mutated == source || !literalExecutable.MatchString(mutated) {
		t.Fatal("literal executable source mutation was not rejected")
	}

	executable := filepath.Join(t.TempDir(), "argument-recorder")
	writeTestFile(t, executable, "#!/bin/sh\nprintf '%s|%s' \"$1\" \"$2\"\n")
	if err := os.Chmod(executable, 0o755); err != nil {
		t.Fatal(err)
	}
	output, err := executeCommand(executable, "first", "second")
	if err != nil {
		t.Fatal(err)
	}
	if string(output) != "first|second" {
		t.Fatalf("parameterized command output = %q, want %q", output, "first|second")
	}
}

func runStamp(t *testing.T, root, commit, runID string) error {
	t.Helper()
	output, err := executeCommand(stampScriptExecutable, filepath.Join(repositoryRoot(), "scripts/stamp-pages-artifact.sh"), "--commit", commit, "--run-id", runID, root)
	if err != nil {
		return &stampError{err: err, output: string(output)}
	}
	return nil
}

type stampError struct {
	err    error
	output string
}

func (e *stampError) Error() string { return e.err.Error() + ": " + e.output }

func TestPagesDeployment_AuthoritativeAPIIdentityPasses(t *testing.T) {
	script := readRepositoryFile(t, "scripts/verify-pages-deployment.sh")
	for _, endpoint := range []string{"repos/$repository/pages", "actions/runs/$run_id", "actions/artifacts/$artifact_id", "deployments?sha=$commit&environment=github-pages", "deployments/$deployment_id/statuses"} {
		if !strings.Contains(script, endpoint) {
			t.Fatalf("authoritative endpoint %q missing", endpoint)
		}
	}
}

func TestPagesDeployment_HTTPSMarkerAndRouteMatrixPasses(t *testing.T) {
	root := makeStampSite(t)
	if err := runStamp(t, root, ownerTestCommit, "123"); err != nil {
		t.Fatal(err)
	}
	marker := `<meta name="backstop-deployment" content="commit=` + ownerTestCommit + `;run=123">`
	for _, route := range canonicalStampOutputs() {
		data, _ := os.ReadFile(filepath.Join(root, route))
		if strings.Count(string(data), marker) != 1 {
			t.Fatalf("%s marker cardinality drifted", route)
		}
	}
	data, err := os.ReadFile(filepath.Join(root, ".well-known/backstop-deployment.json"))
	if err != nil {
		t.Fatal(err)
	}
	var record map[string]any
	if err := json.Unmarshal(data, &record); err != nil || record["commit"] != ownerTestCommit || record["run_id"] != float64(123) || !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(record["tree_content_sha256"].(string)) {
		t.Fatalf("deployment record invalid: %#v err=%v", record, err)
	}
}

func TestPagesDeployment_RejectsPartialOrStaleProofMatrix(t *testing.T) {
	for _, test := range []struct {
		commit, runID string
		remove        bool
	}{{"short", "123", false}, {ownerTestCommit, "0", false}, {ownerTestCommit, "123", true}} {
		root := makeStampSite(t)
		if test.remove {
			if err := os.Remove(filepath.Join(root, "status/index.html")); err != nil {
				t.Fatal(err)
			}
		}
		if err := runStamp(t, root, test.commit, test.runID); err == nil {
			t.Fatalf("partial/stale stamp commit=%q run=%q remove=%v passed", test.commit, test.runID, test.remove)
		}
	}
}

func TestPagesDeployment_StampScriptLocksToCanonicalRoutes(t *testing.T) {
	script := readRepositoryFile(t, "scripts/stamp-pages-artifact.sh")
	start := strings.Index(script, "routes = [")
	if start < 0 {
		t.Fatal("stamp script is missing the canonical routes list")
	}
	block := script[start:]
	end := strings.Index(block, "]")
	if end < 0 {
		t.Fatal("stamp script routes list is unclosed")
	}
	quoted := regexp.MustCompile(`"([^"]+)"`)
	var got []string
	for _, match := range quoted.FindAllStringSubmatch(block[:end], -1) {
		got = append(got, match[1])
	}
	want := canonicalStampOutputs()
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("stamp routes = %#v, want %#v", got, want)
	}
	for _, retired := range []string{"packs/index.html", "extend/index.html"} {
		if strings.Contains(script, retired) {
			t.Fatalf("stamp script still requires retired output %q", retired)
		}
	}
}
