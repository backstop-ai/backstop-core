package distribution_test

// Behavior suite for the production GitCloner (SPEC-055 REQ-001).
//
// Every clone here drives the REAL git binary against a LOCAL repository built by
// the hermetic harness (hermetic_git_test.go). Nothing reaches a remote host: the
// production https:// URL that CLM-005 exercises is redirected at a local
// repository by git's own url.insteadOf rewrite, not by a seam in production code.
//
// The `git` and `/bin/sh` literals below are backstop's own VCS/shell infra, which
// backstop/self Family A allows explicitly.

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/bmanson/backstop-core/pkg/pack/distribution"
)

// promptGuardMarker is written to stderr by the prompt-requiring git stand-in when
// it observes that terminal prompting was NOT disabled. Its presence in a
// diagnostic is the proof that the cloner left git free to prompt.
const promptGuardMarker = "GIT-WOULD-HAVE-PROMPTED"

// redirectedProductionURL is the https URL CLM-005 asks the cloner to clone. It is
// never contacted — withGitConfigRedirect rewrites it to a local repository — but
// it must be shaped like the one resolveGitURL builds, because the whole point is
// that production code runs unchanged.
const redirectedProductionURL = "https://github.com/backstop-ai/harness-pack.git"

// cloneTimeout bounds every real-git invocation in this suite. It is generous
// enough that a local clone never trips it and short enough that a wedged
// subprocess fails the run instead of hanging CI.
const cloneTimeout = 60 * time.Second

func TestExecGitCloner_Clone_MaterializesTaggedRepo(t *testing.T) {
	repo := newTaggedRepo(t, taggedRepoSpec{
		Revisions: []repoRevision{
			{
				Files: packTreeFiles("harness-pack", "1.0.0"),
				Tags:  []repoTag{{Name: "v1.0.0", Annotated: true}},
			},
		},
	})

	destDir := newCloneDestination(t)
	cloner := &distribution.ExecGitCloner{Timeout: cloneTimeout}

	if err := cloner.Clone(repo.URL, "v1.0.0", destDir); err != nil {
		t.Fatalf("Clone(%s, v1.0.0) returned %v; want the pack tree materialized", repo.URL, err)
	}

	// Assert CONTENT, not mere non-emptiness: a clone that produced an empty or
	// partial tree would satisfy a "directory exists" check.
	if got := readClonedFile(t, destDir, "VERSION"); got != "1.0.0\n" {
		t.Errorf("VERSION in the cloned tree is %q; want %q", got, "1.0.0\n")
	}
	if got := readClonedFile(t, destDir, "pack.yml"); !strings.Contains(got, "version: 1.0.0") {
		t.Errorf("pack.yml in the cloned tree is %q; want it to carry version: 1.0.0", got)
	}
}

func TestExecGitCloner_Clone_ChecksOutRequestedTagNotHead(t *testing.T) {
	// The repository has a commit AFTER the tag, and the two revisions differ in
	// file CONTENT. A repository whose tag is HEAD cannot falsify this.
	repo := newTaggedRepo(t, taggedRepoSpec{
		Revisions: []repoRevision{
			{
				Files: packTreeFiles("harness-pack", "1.0.0"),
				Tags:  []repoTag{{Name: "v1.0.0", Annotated: true}},
			},
			{
				Files: packTreeFiles("harness-pack", "2.0.0-unreleased"),
			},
		},
	})

	destDir := newCloneDestination(t)
	cloner := &distribution.ExecGitCloner{Timeout: cloneTimeout}

	if err := cloner.Clone(repo.URL, "v1.0.0", destDir); err != nil {
		t.Fatalf("Clone(%s, v1.0.0) returned %v; want the tagged revision", repo.URL, err)
	}

	got := readClonedFile(t, destDir, "VERSION")
	if got == "2.0.0-unreleased\n" {
		t.Fatalf("the cloned tree carries the HEAD revision (%q); want the revision tagged v1.0.0", got)
	}
	if got != "1.0.0\n" {
		t.Fatalf("VERSION in the cloned tree is %q; want %q", got, "1.0.0\n")
	}
}

func TestExecGitCloner_ListTags_ReturnsBareTagNamesWithoutPeeledRefs(t *testing.T) {
	repo := newTaggedRepo(t, taggedRepoSpec{
		Revisions: []repoRevision{
			{
				Files: packTreeFiles("harness-pack", "1.0.0"),
				Tags:  []repoTag{{Name: "v1.0.0", Annotated: true}},
			},
			{
				Files: packTreeFiles("harness-pack", "1.1.0"),
				Tags:  []repoTag{{Name: "v1.1.0", Annotated: true}},
			},
			{
				Files: packTreeFiles("harness-pack", "2.0.0"),
				Tags:  []repoTag{{Name: "v2.0.0"}},
			},
		},
	})

	// Prove the UNDERLYING output genuinely carries peeled entries. Without this,
	// asserting "no entry ends in ^{}" passes over output that never had one.
	rawRefs := lsRemoteTagRefs(t, repo.URL)
	peeled := 0
	for _, ref := range rawRefs {
		if strings.HasSuffix(ref, "^{}") {
			peeled++
		}
	}
	if peeled == 0 {
		t.Fatalf("the harness repository produced no peeled ^{} refs (%v); the filter under test would pass vacuously", rawRefs)
	}

	cloner := &distribution.ExecGitCloner{Timeout: cloneTimeout}
	tags, err := cloner.ListTags(repo.URL)
	if err != nil {
		t.Fatalf("ListTags(%s) returned %v; want the repository's tags", repo.URL, err)
	}

	for _, tag := range tags {
		if strings.HasSuffix(tag, "^{}") {
			t.Errorf("ListTags returned peeled ref %q; peeled entries must be excluded", tag)
		}
		if strings.HasPrefix(tag, "refs/") {
			t.Errorf("ListTags returned %q; want a bare tag name, not a full ref path", tag)
		}
	}

	// A filter that dropped EVERYTHING would satisfy the assertions above.
	for _, want := range []string{"v1.0.0", "v1.1.0", "v2.0.0"} {
		if !containsString(tags, want) {
			t.Errorf("ListTags returned %v; want it to contain %q", tags, want)
		}
	}
}

func TestExecGitCloner_Clone_RunsGitNonInteractively(t *testing.T) {
	// The stand-in blocks exactly as a credential or host-key prompt would, UNLESS
	// the caller disabled terminal prompting. A cloner that leaves prompting
	// enabled therefore does not return until its own timeout kills the process —
	// which this test detects as a timeout diagnostic rather than a hang.
	standIn := newPromptRequiringGitStandIn(t)

	destDir := newCloneDestination(t)
	cloner := &distribution.ExecGitCloner{GitBinary: standIn, Timeout: 3 * time.Second}

	err := cloner.Clone("file:///harness/would-prompt", "v1.0.0", destDir)
	if err != nil {
		if strings.Contains(err.Error(), promptGuardMarker) {
			t.Fatalf("git was invoked with terminal prompting still enabled: %v", err)
		}
		t.Fatalf("Clone against a prompt-requiring source returned %v; want it to complete non-interactively", err)
	}
}

func TestExecGitCloner_Clone_HonorsAmbientGitConfigRedirect(t *testing.T) {
	// The load-bearing hermeticity test: production code asks for the REAL https
	// URL and git itself substitutes a local repository. It only works because the
	// cloner INHERITS the ambient environment instead of clearing it.
	repo := newTaggedRepo(t, taggedRepoSpec{
		Revisions: []repoRevision{
			{
				Files: packTreeFiles("harness-pack", "1.0.0"),
				Tags:  []repoTag{{Name: "v1.0.0", Annotated: true}},
			},
		},
	})
	withGitConfigRedirect(t, redirectedProductionURL, repo.URL)

	destDir := newCloneDestination(t)
	cloner := &distribution.ExecGitCloner{Timeout: cloneTimeout}

	if err := cloner.Clone(redirectedProductionURL, "v1.0.0", destDir); err != nil {
		t.Fatalf("Clone(%s, v1.0.0) returned %v; the ambient url.insteadOf rewrite should have redirected it to %s", redirectedProductionURL, err, repo.URL)
	}

	if got := readClonedFile(t, destDir, "VERSION"); got != "1.0.0\n" {
		t.Errorf("VERSION in the redirected clone is %q; want %q", got, "1.0.0\n")
	}
}

func TestExecGitCloner_Clone_RejectsOptionLikeURL(t *testing.T) {
	// GitBinary points at a stand-in that announces itself loudly on stderr, so
	// "the guard rejected it" and "git also happened to reject it" are
	// distinguishable.
	loudGit := newFakeGitBinary(t, fakeGitFailsLoudly)
	cloner := &distribution.ExecGitCloner{GitBinary: loudGit, Timeout: cloneTimeout}

	err := cloner.Clone("--upload-pack=touch /tmp/pwned", "v1.0.0", newCloneDestination(t))

	assertGitErrorWithoutSubprocess(t, err, "--upload-pack=touch /tmp/pwned")
}

func TestExecGitCloner_Clone_RejectsOptionLikeRef(t *testing.T) {
	loudGit := newFakeGitBinary(t, fakeGitFailsLoudly)
	cloner := &distribution.ExecGitCloner{GitBinary: loudGit, Timeout: cloneTimeout}

	err := cloner.Clone("file:///harness/repo", "--upload-pack=touch /tmp/pwned", newCloneDestination(t))

	assertGitErrorWithoutSubprocess(t, err, "--upload-pack=touch /tmp/pwned")
}

func TestExecGitCloner_ListTags_RejectsOptionLikeURL(t *testing.T) {
	loudGit := newFakeGitBinary(t, fakeGitFailsLoudly)
	cloner := &distribution.ExecGitCloner{GitBinary: loudGit, Timeout: cloneTimeout}

	tags, err := cloner.ListTags("--upload-pack=touch /tmp/pwned")

	if len(tags) != 0 {
		t.Errorf("ListTags returned %v for a rejected URL; want no tags", tags)
	}
	assertGitErrorWithoutSubprocess(t, err, "--upload-pack=touch /tmp/pwned")
}

func TestNewExecGitCloner_DoesNotProbeForGitBinary(t *testing.T) {
	// Construction must succeed where git is absent, because `pack install --cache`
	// assembles its commands in airgapped environments that never invoke git. A
	// missing binary is a Clone/ListTags-time diagnostic, not an assembly failure.
	withoutGitOnPath(t)

	cloner := distribution.NewExecGitCloner()

	if cloner == nil {
		t.Fatal("NewExecGitCloner returned nil where git is absent; construction must not probe for the binary")
	}
	if cloner.Timeout <= 0 {
		t.Errorf("NewExecGitCloner produced Timeout %v; want a positive default so an invocation is always bounded", cloner.Timeout)
	}
	if cloner.GitBinary != "" {
		t.Errorf("NewExecGitCloner resolved GitBinary to %q; want it empty so the executable is resolved at invocation time, not at construction", cloner.GitBinary)
	}
}

func TestExecGitCloner_Clone_StripsRootGitDirectory(t *testing.T) {
	// The fixture carries dot-git-PREFIXED files (.gitignore, and a nested one) so a
	// strip implemented as a tree walk over names containing ".git" is caught here
	// rather than silently corrupting authored pack content. A nested directory
	// literally named ".git" cannot be committed to a git repository at all, so it
	// is not fixture-representable; the implementation removes exactly one path.
	files := packTreeFiles("harness-pack", "1.0.0")
	files[".gitignore"] = "node_modules/\n"
	files["rules/.gitignore"] = "*.tmp\n"
	files["rules/core.yml"] = "rules: []\n"

	repo := newTaggedRepo(t, taggedRepoSpec{
		Revisions: []repoRevision{
			{Files: files, Tags: []repoTag{{Name: "v1.0.0", Annotated: true}}},
		},
	})

	// Prove the SOURCE has a root .git: a fixture that never had one makes the
	// "destination has none" assertion vacuous.
	if info, err := os.Stat(filepath.Join(repo.Path, ".git")); err != nil || !info.IsDir() {
		t.Fatalf("the harness repository at %s has no root .git directory (%v); the strip under test would pass vacuously", repo.Path, err)
	}

	destDir := newCloneDestination(t)
	cloner := &distribution.ExecGitCloner{Timeout: cloneTimeout}

	if err := cloner.Clone(repo.URL, "v1.0.0", destDir); err != nil {
		t.Fatalf("Clone(%s, v1.0.0) returned %v; want a stripped pack tree", repo.URL, err)
	}

	if _, err := os.Stat(filepath.Join(destDir, ".git")); !os.IsNotExist(err) {
		t.Errorf("the cloned tree still carries a root .git directory (stat error %v); Clone must strip it before returning", err)
	}

	// A strip that removed the whole tree would satisfy the check above on its own.
	for name, want := range map[string]string{
		"VERSION":          "1.0.0\n",
		".gitignore":       "node_modules/\n",
		"rules/.gitignore": "*.tmp\n",
		"rules/core.yml":   "rules: []\n",
	} {
		if got := readClonedFile(t, destDir, name); got != want {
			t.Errorf("%s in the stripped tree is %q; want %q — the strip must remove only the root .git", name, got, want)
		}
	}
}

func TestExecGitCloner_Clone_RepeatedClonesHashIdentically(t *testing.T) {
	// The property the strip exists to buy. Two clones of one tag differ only in
	// their .git (reflog timestamps, object layout), so without the strip these
	// hashes cannot agree — which is what would make `pack add` and `pack install`
	// disagree about the same pack.
	repo := newTaggedRepo(t, taggedRepoSpec{
		Revisions: []repoRevision{
			{
				Files: packTreeFiles("harness-pack", "1.0.0"),
				Tags:  []repoTag{{Name: "v1.0.0", Annotated: true}},
			},
		},
	})

	cloner := &distribution.ExecGitCloner{Timeout: cloneTimeout}

	firstDir := newCloneDestination(t)
	if err := cloner.Clone(repo.URL, "v1.0.0", firstDir); err != nil {
		t.Fatalf("the first Clone returned %v", err)
	}
	secondDir := newCloneDestination(t)
	if err := cloner.Clone(repo.URL, "v1.0.0", secondDir); err != nil {
		t.Fatalf("the second Clone returned %v", err)
	}

	// Both hashes are COMPUTED. A hardcoded constant would pin the fixture content
	// and break on any unrelated fixture edit.
	firstHash, err := distribution.ComputeContentHash(firstDir)
	if err != nil {
		t.Fatalf("hash the first clone: %v", err)
	}
	secondHash, err := distribution.ComputeContentHash(secondDir)
	if err != nil {
		t.Fatalf("hash the second clone: %v", err)
	}

	if firstHash != secondHash {
		t.Errorf("two clones of v1.0.0 hashed to %s and %s; they must agree, or a pack's recorded content hash is unreproducible", firstHash, secondHash)
	}
	if firstHash == "" {
		t.Error("the computed content hash is empty; two empty trees would agree vacuously")
	}
}

// newCloneDestination returns an existing empty directory for a clone, mirroring
// the Add/Install/Update/Upgrade pipelines, which hand Clone an os.MkdirTemp path.
func newCloneDestination(t *testing.T) string {
	t.Helper()

	dir := filepath.Join(t.TempDir(), "dest")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create the clone destination: %v", err)
	}

	return dir
}

// readClonedFile reads one file out of a cloned tree, failing the calling test when
// it is absent — so a missing file reports the path rather than an empty string
// that a content comparison would report as a mismatch.
func readClonedFile(t *testing.T, destDir, name string) string {
	t.Helper()

	contents, err := os.ReadFile(filepath.Join(destDir, filepath.FromSlash(name)))
	if err != nil {
		t.Fatalf("read %s out of the cloned tree at %s: %v", name, destDir, err)
	}

	return string(contents)
}

// newPromptRequiringGitStandIn writes a git stand-in that BLOCKS when terminal
// prompting is enabled and exits cleanly when it is disabled — the hermetic
// equivalent of a repository that would ask for credentials or a host key.
//
// It lives here rather than in the shared harness because it is the only stand-in
// whose behavior is conditioned on the environment the code under test sets.
func newPromptRequiringGitStandIn(t *testing.T) string {
	t.Helper()
	skipUnlessPOSIX(t)

	script := "#!/bin/sh\n" +
		"if [ \"$GIT_TERMINAL_PROMPT\" = \"0\" ]; then\n" +
		"  exit 0\n" +
		"fi\n" +
		"echo " + promptGuardMarker + " >&2\n" +
		"exec sleep " + strconv.Itoa(fakeGitHangSeconds) + "\n"

	path := filepath.Join(t.TempDir(), "git")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write the prompt-requiring git stand-in: %v", err)
	}

	return path
}

// withoutGitOnPath empties the process PATH for the calling test's lifetime and
// proves git is genuinely unresolvable, so "construction does not probe" is
// asserted against an environment where a probe would actually fail.
func withoutGitOnPath(t *testing.T) {
	t.Helper()

	t.Setenv("PATH", t.TempDir())

	if path, err := exec.LookPath("git"); err == nil {
		t.Fatalf("git is still resolvable at %s after emptying PATH; a construction-time probe would succeed and the assertion would be vacuous", path)
	}
}

// assertGitErrorWithoutSubprocess asserts a guard rejected an option-like argument
// as a typed *GitError naming it, and that NO git subprocess ran — the loud
// stand-in's marker is absent from the diagnostic.
func assertGitErrorWithoutSubprocess(t *testing.T, err error, argument string) {
	t.Helper()

	if err == nil {
		t.Fatalf("an argument beginning with \"-\" (%q) was accepted; want it rejected before git is invoked", argument)
	}

	var gitErr *distribution.GitError
	if !errors.As(err, &gitErr) {
		t.Fatalf("rejection returned %T (%v); want *distribution.GitError", err, err)
	}
	if strings.Contains(err.Error(), fakeGitLoudMarker) {
		t.Fatalf("git was invoked before the guard rejected %q: %v", argument, err)
	}
	if !strings.Contains(err.Error(), argument) {
		t.Errorf("the diagnostic %q does not name the rejected argument %q", err.Error(), argument)
	}
}

// containsString reports whether values holds want.
func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}

	return false
}
