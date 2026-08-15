package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/check"
)

// doctor_harness_test.go is the SHARED machinery every doctor test file drives. It
// carries no mandated test name of its own.
//
// ★ THERE IS EXACTLY ONE TEMPLATE ROOT AND ONE STAGER. Every doctor corpus — the
// config-shaped projects, the toolchain matrix and outcome fixtures, the layout
// corpora — lives under cmd/backstop/testdata/doctor/projects/<name> and reaches a
// test only through stageDoctorProject. That is not tidiness: gate.FindUngatedArtifacts
// SKIPS `testdata` among its non-corpus exclusion trees, so a layout corpus read IN
// PLACE yields ZERO deviations for every case and the layout claims pass against
// nothing while reporting green. Staging into t.TempDir is what makes them falsifiable.

// doctorTemplateRoot is the ONE template root. No test may name a path beneath it
// directly.
const doctorTemplateRoot = "testdata/doctor/projects"

// stageDoctorProject copies a fixture project TEMPLATE into a fresh t.TempDir and
// returns the staged path.
//
// It t.Fatalf's on a missing template rather than skipping: a skip recreates exactly
// the vacuous green doctor exists to prevent.
func stageDoctorProject(t *testing.T, templateName string) string {
	t.Helper()

	source := filepath.Join(doctorTemplateRoot, templateName)
	if info, err := os.Stat(source); err != nil || !info.IsDir() {
		t.Fatalf("fixture template %q is missing at %s (err=%v)", templateName, source, err)
	}

	staged := t.TempDir()
	copyTree(t, source, staged)

	// RECURSIVELY, not just at the staged root. The no-config template carries
	// .gitkeep at its top level while layout-empty-root carries it at
	// .backstop/.gitkeep — a resolved artifact root that must EXIST and be EMPTY. A
	// top-level-only deletion leaves that file in place and the empty-root claim
	// asserts against the wrong condition.
	walkErr := filepath.Walk(staged, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && info.Name() == ".gitkeep" {
			return os.Remove(path)
		}
		return nil
	})
	if walkErr != nil {
		t.Fatalf("removing .gitkeep from staged %s: %v", templateName, walkErr)
	}

	// SYMLINK-RESOLVED, because every path a test compares against comes from a check
	// that resolved the working directory. On darwin t.TempDir is /var/folders/... while
	// os.Getwd inside it returns /private/var/folders/..., so an unresolved staged path
	// makes every path assertion fail for a reason that has nothing to do with doctor.
	resolved, resolveErr := filepath.EvalSymlinks(staged)
	if resolveErr != nil {
		t.Fatalf("resolving staged path %s: %v", staged, resolveErr)
	}
	return resolved
}

// gitInitProject makes dir a git work tree, and PROVES the staged directory was not
// already inside one.
//
// The trap it closes: git rev-parse walks UP, so a temp dir nested inside this
// repository's own work tree would report true and a test asserting the non-repo warn
// would inherit a repository it never asked for. t.TempDir is /var/folders/... on
// darwin, outside this repo — but that is asserted here rather than assumed.
func gitInitProject(t *testing.T, dir string) {
	t.Helper()

	if isGitWorkTree(dir) {
		t.Fatalf("staged project %s is ALREADY inside a git work tree; the non-repo cases cannot be trusted", dir)
	}

	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init in %s: %v\n%s", dir, err, out)
	}

	if !isGitWorkTree(dir) {
		t.Fatalf("git init in %s did not produce a work tree the detector recognizes", dir)
	}
}

// runDoctorInProject chdirs into dir, drives the REAL root command with args, and
// returns the captured output plus the TRUE integer exit code.
//
// args is the FULL root argument list, so a caller can put a flag in ROOT position
// ("--json", "doctor") as well as in sub-command position ("doctor", "--json").
func runDoctorInProject(t *testing.T, dir string, args ...string) (string, int) {
	t.Helper()

	original, wdErr := os.Getwd()
	if wdErr != nil {
		t.Fatalf("resolving working directory: %v", wdErr)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir to %s: %v", dir, err)
	}
	defer func() { _ = os.Chdir(original) }()

	root := NewRootCommand()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)
	err := root.Execute()

	// Mirror main.go: the root command silences cobra's own error printing, so
	// reportError is the SOLE printer of a failed command's diagnostic. The harness
	// reproduces that rather than dropping the message, because the loud-error claims
	// assert on text a real invocation would show.
	if err != nil {
		reportError(buf, err)
	}

	return buf.String(), doctorExitCode(err)
}

// doctorExitCode translates a command error into the process exit code, by exactly the
// rule reportError (main.go) applies: an *ExitCodeError carries its own Code, and an
// unclassified error is a config error.
//
// Every exit-matrix claim reads THIS integer. Nothing infers an exit code from text.
func doctorExitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *ExitCodeError
	if errors.As(err, &exitErr) {
		return exitErr.Code
	}
	return ExitConfigError
}

// doctorJSONPayload is the decoded --json payload, kept as maps so a test can assert
// KEY PRESENCE (a `remediation` key on a PASSING check, which is what the dropped
// omitempty buys) rather than only value equality.
type doctorJSONPayload struct {
	SchemaVersion string
	Checks        []map[string]interface{}
}

// decodeDoctorJSON unmarshals the --json payload out of stdout.
func decodeDoctorJSON(t *testing.T, stdout string) doctorJSONPayload {
	t.Helper()

	start := strings.Index(stdout, "{")
	if start < 0 {
		t.Fatalf("no JSON object in output:\n%s", stdout)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(stdout[start:]), &raw); err != nil {
		t.Fatalf("decoding doctor JSON: %v\noutput:\n%s", err, stdout)
	}

	payload := doctorJSONPayload{}
	if version, ok := raw["schema_version"].(string); ok {
		payload.SchemaVersion = version
	}
	entries, ok := raw["checks"].([]interface{})
	if !ok {
		t.Fatalf("doctor JSON carries no `checks` array:\n%s", stdout)
	}
	for _, entry := range entries {
		object, objectOK := entry.(map[string]interface{})
		if !objectOK {
			t.Fatalf("doctor JSON check entry is not an object: %#v", entry)
		}
		payload.Checks = append(payload.Checks, object)
	}
	return payload
}

// ids returns the check ids in report order.
func (p doctorJSONPayload) ids() []string {
	out := make([]string, 0, len(p.Checks))
	for _, entry := range p.Checks {
		id, _ := entry["id"].(string)
		out = append(out, id)
	}
	return out
}

// statuses maps check id to reported status.
func (p doctorJSONPayload) statuses() map[string]string {
	out := make(map[string]string, len(p.Checks))
	for _, entry := range p.Checks {
		id, _ := entry["id"].(string)
		status, _ := entry["status"].(string)
		out[id] = status
	}
	return out
}

// find returns the entry for id.
func (p doctorJSONPayload) find(t *testing.T, id string) map[string]interface{} {
	t.Helper()
	for _, entry := range p.Checks {
		if entryID, _ := entry["id"].(string); entryID == id {
			return entry
		}
	}
	t.Fatalf("doctor report carries no check %q; it reported %v", id, p.ids())
	return nil
}

// field returns one string field of the entry for id.
func (p doctorJSONPayload) field(t *testing.T, id, key string) string {
	t.Helper()
	value, _ := p.find(t, id)[key].(string)
	return value
}

// runDoctorJSON is the common case: run every check over a staged project and decode.
func runDoctorJSON(t *testing.T, dir string, extra ...string) (doctorJSONPayload, int) {
	t.Helper()
	args := append([]string{"doctor", "--json"}, extra...)
	stdout, code := runDoctorInProject(t, dir, args...)
	return decodeDoctorJSON(t, stdout), code
}

// markerFiles lists the *.marker files a fixture's entrypoints wrote, sorted.
//
// The gate_type matrix reads EXECUTION and NON-EXECUTION off this: a marker present is
// a real command having run, and a marker absent is a filesystem fact rather than a
// reading of the implementation's selection list.
func markerFiles(t *testing.T, dir string) []string {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading %s: %v", dir, err)
	}
	var found []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".marker") {
			found = append(found, entry.Name())
		}
	}
	sort.Strings(found)
	return found
}

// isGitWorkTree asks the SAME question checkGitRepository asks, through the same
// shipped exported detector, rather than shelling out to a second `git rev-parse`.
func isGitWorkTree(dir string) bool {
	return (&check.DefaultGitExecutor{Dir: dir}).IsGitRepo()
}
