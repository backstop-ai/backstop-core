package distribution_test

// Real-git test harness for the production remote-dependency suites (SPEC-055).
//
// This file holds NO test functions — it is the substrate the ExecGitCloner and
// TagVersionResolver suites drive. Every repository it builds is LOCAL: nothing
// here reaches a remote host, and the production https:// URL is redirected at a
// local repository by git's own url.insteadOf rewrite (withGitConfigRedirect)
// rather than by a rewriting seam in production code.
//
// The `git` and `/bin/sh` literals below are backstop's own VCS/shell infra,
// which backstop/self Family A allows explicitly. No other tool name may appear
// as a literal here.

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// fakeGitLoudMarker is printed on stderr by the fakeGitFailsLoudly stand-in. A
// test that must prove git was NEVER invoked (the option-like argument guards)
// asserts this marker is absent: without a marker, "no subprocess ran" and "the
// subprocess also rejected the argument" are indistinguishable.
const fakeGitLoudMarker = "FAKE-GIT-WAS-INVOKED"

// fakeGitHangSeconds is far longer than any bounded git invocation under test, so
// a cloner that fails to bound its subprocess wedges rather than passing.
const fakeGitHangSeconds = 600

// fakeGitBehavior selects what the stand-in git executable does when invoked.
type fakeGitBehavior int

const (
	// fakeGitHangs never returns on its own; only a timeout kill ends it.
	fakeGitHangs fakeGitBehavior = iota
	// fakeGitFailsLoudly exits non-zero after printing fakeGitLoudMarker on stderr.
	fakeGitFailsLoudly
)

// repoTag describes one tag created at a revision.
//
// Annotated matters: `git ls-remote --tags` emits an extra peeled
// "<sha> refs/tags/<name>^{}" line for an ANNOTATED tag and none for a
// lightweight one. A repository whose tags are all lightweight makes the
// peeled-ref filter untestable — the filter assertion would pass over output that
// never contained a peeled entry to begin with.
type repoTag struct {
	Name      string
	Annotated bool
}

// repoRevision is one commit in a harness repository: the files it writes over the
// working tree, and the tags (if any) created at it. Revisions after the last
// tagged one are what make "checked out the requested TAG, not HEAD" falsifiable —
// a repository where the tag is HEAD cannot fail that assertion.
type repoRevision struct {
	Message string
	Files   map[string]string
	Tags    []repoTag
}

// taggedRepoSpec describes a repository as an ordered sequence of revisions.
type taggedRepoSpec struct {
	Revisions []repoRevision
}

// taggedRepo is a materialized local repository plus the file:// URL that reaches it.
type taggedRepo struct {
	Path string
	URL  string
}

// newTaggedRepo builds a real git repository in t.TempDir() by invoking the real
// git binary once per revision: init, write the revision's files, commit, tag.
func newTaggedRepo(t *testing.T, spec taggedRepoSpec) taggedRepo {
	t.Helper()
	gitBinary := gitBinaryOrSkip(t)

	if len(spec.Revisions) == 0 {
		t.Fatal("a harness repository needs at least one revision; an empty repository has no tag to check out")
	}

	dir := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create the harness repository directory: %v", err)
	}

	runGit(t, gitBinary, dir, "-c", "init.defaultBranch=main", "init")

	for index, revision := range spec.Revisions {
		writeRepoFiles(t, dir, revision.Files)

		message := revision.Message
		if message == "" {
			message = fmt.Sprintf("harness revision %d", index+1)
		}

		runGit(t, gitBinary, dir, "add", "-A")
		runGit(t, gitBinary, dir, "commit", "--allow-empty", "-m", message)

		for _, tag := range revision.Tags {
			if tag.Annotated {
				runGit(t, gitBinary, dir, "tag", "-a", tag.Name, "-m", "annotated "+tag.Name)
				continue
			}
			runGit(t, gitBinary, dir, "tag", tag.Name)
		}
	}

	return taggedRepo{Path: dir, URL: "file://" + filepath.ToSlash(dir)}
}

// packTreeFiles returns the small pack tree a harness revision commits. The version
// is written into BOTH pack.yml and a VERSION marker so a test can tell a tagged
// revision from a later one by file CONTENT rather than by commit identity.
func packTreeFiles(name, version string) map[string]string {
	return map[string]string{
		"pack.yml":  fmt.Sprintf("name: %s\nversion: %s\ndescription: harness fixture pack\n", name, version),
		"VERSION":   version + "\n",
		"README.md": fmt.Sprintf("# %s\n\nHarness fixture pack at version %s.\n", name, version),
	}
}

// withGitConfigRedirect points GIT_CONFIG_GLOBAL at a temporary gitconfig carrying
// a `url.<to>.insteadOf = <from>` rewrite for the calling test's lifetime, and
// returns the config path.
//
// This is how a production https:// URL is aimed at a LOCAL repository without any
// rewriting seam in production code: git itself substitutes the URL, so the code
// under test still asks for the real one. It only works if the code under test
// INHERITS the ambient environment — which is precisely what it makes testable.
//
// The rewrite is read back through git before returning, so a malformed config or
// an ignored GIT_CONFIG_GLOBAL fails here instead of silently letting a later test
// reach the network.
func withGitConfigRedirect(t *testing.T, from, to string) string {
	t.Helper()
	gitBinary := gitBinaryOrSkip(t)

	configPath := filepath.Join(t.TempDir(), "gitconfig")
	contents := fmt.Sprintf("[url %q]\n\tinsteadOf = %s\n", to, from)
	if err := os.WriteFile(configPath, []byte(contents), 0o600); err != nil {
		t.Fatalf("write the redirecting gitconfig: %v", err)
	}

	t.Setenv("GIT_CONFIG_GLOBAL", configPath)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")

	// Read the rewrite back with the AMBIENT environment (no override), so this
	// asserts the environment variable just set is genuinely honored.
	verify := exec.Command(gitBinary, "config", "--global", "--get", "url."+to+".insteadOf")
	verify.Dir = t.TempDir()
	output, err := verify.CombinedOutput()
	if err != nil {
		t.Fatalf("read back the url.insteadOf rewrite from %s: %v\n%s", configPath, err, output)
	}
	if got := strings.TrimSpace(string(output)); got != from {
		t.Fatalf("the gitconfig at %s rewrites %q, want %q — the redirect would silently miss", configPath, got, from)
	}

	return configPath
}

// lsRemoteTagRefs returns the RAW `git ls-remote --tags` lines for a repository.
//
// It exists so a test can prove the underlying output actually contains peeled
// "^{}" entries before asserting the production listing dropped them — without
// that proof, a peeled-ref filter assertion passes vacuously.
func lsRemoteTagRefs(t *testing.T, url string) []string {
	t.Helper()
	gitBinary := gitBinaryOrSkip(t)

	output := runGit(t, gitBinary, t.TempDir(), "ls-remote", "--tags", url)

	lines := make([]string, 0)
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			lines = append(lines, trimmed)
		}
	}

	return lines
}

// newFakeGitBinary writes a tiny POSIX-shell stand-in for git into a temp dir and
// returns its path, for the diagnostics that need a git which HANGS or which fails
// LOUDLY. Point ExecGitCloner.GitBinary at it — the test process's PATH is never
// mutated, which is exactly why that field exists.
func newFakeGitBinary(t *testing.T, behavior fakeGitBehavior) string {
	t.Helper()
	skipUnlessPOSIX(t)

	var script string
	switch behavior {
	case fakeGitHangs:
		script = fmt.Sprintf("#!/bin/sh\nexec sleep %d\n", fakeGitHangSeconds)
	case fakeGitFailsLoudly:
		script = fmt.Sprintf("#!/bin/sh\necho %s >&2\nexit 3\n", fakeGitLoudMarker)
	default:
		t.Fatalf("unknown fake git behavior %d", behavior)
		return ""
	}

	path := filepath.Join(t.TempDir(), "git")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write the fake git binary: %v", err)
	}

	return path
}

// missingGitBinaryPath returns a path that does not exist, for pointing
// ExecGitCloner.GitBinary at an ABSENT executable. The test process's PATH stays
// untouched, so a parallel test cannot be affected by the missing-binary case.
func missingGitBinaryPath(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "git-that-is-not-installed")
	if _, err := os.Stat(path); err == nil {
		t.Fatalf("the supposedly missing git binary %s exists; the diagnostic under test would not fire", path)
	}

	return path
}

// gitBinaryOrSkip resolves the real git executable, skipping the calling test with
// a clear message when git is unavailable — so the suite degrades honestly rather
// than failing opaquely deep inside a subprocess error.
func gitBinaryOrSkip(t *testing.T) string {
	t.Helper()
	skipUnlessPOSIX(t)

	path, err := exec.LookPath("git")
	if err != nil {
		t.Skipf("the real-git harness needs a git binary on PATH; none found: %v", err)
	}

	return path
}

// skipUnlessPOSIX skips tests that depend on POSIX-shell stand-ins and file:// URLs.
func skipUnlessPOSIX(t *testing.T) {
	t.Helper()

	if runtime.GOOS == "windows" {
		t.Skip("the real-git harness writes POSIX-shell stand-ins and file:// URLs; it does not run on Windows")
	}
}

// runGit invokes the real git binary in dir and returns its combined output,
// failing the test on any non-zero exit.
//
// The invocation is INSULATED from ambient git configuration (including any
// url.insteadOf rewrite a test installed via withGitConfigRedirect), so harness
// repositories are built identically regardless of what the test has arranged for
// the code under test.
func runGit(t *testing.T, gitBinary, dir string, args ...string) string {
	t.Helper()

	cmd := exec.Command(gitBinary, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), hermeticGitEnv()...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s in %s: %v\n%s", strings.Join(args, " "), dir, err, output)
	}

	return string(output)
}

// hermeticGitEnv is the environment overlay that makes a harness git invocation
// independent of the developer's machine: no global or system config, a fixed
// identity so commits and annotated tags succeed, and no interactive prompting.
func hermeticGitEnv() []string {
	return []string{
		"GIT_CONFIG_GLOBAL=" + os.DevNull,
		"GIT_CONFIG_SYSTEM=" + os.DevNull,
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_TERMINAL_PROMPT=0",
		"GIT_AUTHOR_NAME=backstop harness",
		"GIT_AUTHOR_EMAIL=harness@backstop.invalid",
		"GIT_COMMITTER_NAME=backstop harness",
		"GIT_COMMITTER_EMAIL=harness@backstop.invalid",
	}
}

// writeRepoFiles writes a revision's files into the repository working tree,
// creating parent directories and overwriting whatever the previous revision left.
func writeRepoFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()

	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		target := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatalf("create the directory for %s: %v", name, err)
		}
		if err := os.WriteFile(target, []byte(files[name]), 0o644); err != nil {
			t.Fatalf("write %s into the harness repository: %v", name, err)
		}
	}
}
