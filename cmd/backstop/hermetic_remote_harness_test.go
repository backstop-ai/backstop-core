package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The hermetic remote harness (SPEC-055 REQ-010). Every end-to-end remote claim —
// pack add / install / update / upgrade driven through the BUILT binary — needs a
// repository that behaves like a pack's GitHub origin without any network. This file
// is helper source only: it declares no `func Test…` and is the substrate the remote
// e2e suites, and later BUNDLE-006 REQ-042's parity suite, EXTEND rather than rebuild.
//
// The mechanism is a `url.<file://repo>.insteadOf = https://github.com/<org>/<pack>.git`
// rewrite installed via GIT_CONFIG_GLOBAL. Production code runs UNCHANGED — including
// distribution's hardcoded resolveGitURL construction (add.go:315) — and git itself
// redirects the clone to a local repository.
//
// LOAD-BEARING: the redirect only works because the clone INHERITS the ambient
// environment (SPEC-055 CLM-005 and its sharp edge). t.Setenv mutates the TEST
// process; a runner that hands the child a scrubbed env sends it to GitHub instead,
// and the suite silently becomes network-dependent without anything turning red.
// hermeticGitEnv is therefore the env any runner must give the child, and
// assertPackURLRedirected PROVES the redirect reached a child process rather than
// trusting that it did.
//
// Binaries come from the existing buildBackstopBinary (pack_authoring_loop_test.go:15)
// so the tests drive the exact command wiring shipped to consumers. There is no second
// binary builder here on purpose.

// hermeticRemote is a local git repository standing in for a pack's remote origin.
type hermeticRemote struct {
	// Path is the repository's working tree on disk.
	Path string
	// FileURL is Path as a file:// URL — the redirect target.
	FileURL string
	// Tags are the tags created, in the order requested.
	Tags []string
}

// newHermeticRemote copies a pack fixture SOURCE tree into a fresh temp directory,
// turns it into a real git repository, and creates the requested tags (v-prefixed, as
// the distribution pipelines construct them: `Clone(url, "v"+version, dst)`).
//
// Each tag gets its own commit that rewrites pack.yml's version to that tag, so two
// tags never name identical content — the newer-compatible-version case needs the
// resolved tag to be materially different from the installed one, and a repository
// whose tags all point at one commit cannot falsify that.
func newHermeticRemote(t *testing.T, packSourceDir string, tags ...string) *hermeticRemote {
	t.Helper()
	requireGit(t)

	if _, err := os.Stat(filepath.Join(packSourceDir, ".git")); err == nil {
		t.Fatalf("pack source %s already carries a repository; the fixture sources are plain trees and this harness is what makes them repositories", packSourceDir)
	}

	repo := filepath.Join(t.TempDir(), "remote")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatalf("creating hermetic remote dir: %v", err)
	}
	copyTree(t, packSourceDir, repo)

	mustGit(t, repo, "init")
	mustGit(t, repo, "add", "-A")
	mustGit(t, repo, "commit", "-m", "import pack source")

	created := make([]string, 0, len(tags))
	for _, tag := range tags {
		if setPackVersion(t, repo, strings.TrimPrefix(tag, "v")) {
			mustGit(t, repo, "add", "-A")
			mustGit(t, repo, "commit", "-m", "set pack version "+tag)
		}
		mustGit(t, repo, "tag", tag)
		created = append(created, tag)
	}

	return &hermeticRemote{Path: repo, FileURL: fileURL(repo), Tags: created}
}

// redirectPackURL points the production URL for org/name at repoPath for the rest of
// the test, by writing a temporary gitconfig and exporting GIT_CONFIG_GLOBAL. It
// returns the gitconfig path.
//
// org and name MUST be the pair the tested pack ref resolves to: the rewrite is keyed
// on the exact URL productionPackURL builds. A mismatched pair does not fail — it
// misses, and the clone quietly reaches the network.
func redirectPackURL(t *testing.T, org, name, repoPath string) string {
	t.Helper()
	requireGit(t)

	cfgPath := filepath.Join(t.TempDir(), "gitconfig")
	// The identity block matters because this config REPLACES the developer's global
	// one for the test's lifetime; without it a child git that needs an identity fails
	// for a reason that has nothing to do with what is under test.
	body := strings.Join([]string{
		"[user]",
		"\tname = " + hermeticIdentityName,
		"\temail = " + hermeticIdentityEmail,
		"[url \"" + fileURL(repoPath) + "\"]",
		"\tinsteadOf = " + productionPackURL(org, name),
		"",
	}, "\n")
	if err := os.WriteFile(cfgPath, []byte(body), 0o644); err != nil {
		t.Fatalf("writing hermetic gitconfig: %v", err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", cfgPath)
	return cfgPath
}

// assertPackURLRedirected proves the redirect reaches a CHILD process and resolves to
// the intended repository, by asking a child git to list the production URL's tags and
// requiring the hermetic remote's tags back.
//
// Success alone is already dispositive — the org/name pair a hermetic test uses cannot
// exist upstream — but matching the tag set also catches a redirect that landed on the
// wrong local repository. Call this before asserting on lifecycle behavior; otherwise a
// green suite is indistinguishable from one silently talking to GitHub.
func assertPackURLRedirected(t *testing.T, org, name string, remote *hermeticRemote) {
	t.Helper()

	url := productionPackURL(org, name)
	out, err := gitOutput(t, t.TempDir(), hermeticGitEnv(t), "ls-remote", "--tags", url)
	if err != nil {
		t.Fatalf("the insteadOf redirect did not reach the child process: listing %s failed: %v\n%s", url, err, out)
	}

	seen := map[string]bool{}
	for _, line := range strings.Split(out, "\n") {
		idx := strings.Index(line, refTagsPrefix)
		if idx < 0 {
			continue
		}
		seen[strings.TrimSuffix(line[idx+len(refTagsPrefix):], "^{}")] = true
	}
	for _, tag := range remote.Tags {
		if !seen[tag] {
			t.Fatalf("%s redirected somewhere other than %s: tag %q missing from the listing\n%s", url, remote.Path, tag, out)
		}
	}
}

// hermeticGitEnv is the environment a child process must be given so the insteadOf
// redirect survives the process boundary. Any runner that executes the built binary
// under a redirect uses this — never a hand-built or filtered env.
//
// It fails rather than defaulting when GIT_CONFIG_GLOBAL is unset: a missing redirect
// is a network call, not a degraded local run.
func hermeticGitEnv(t *testing.T) []string {
	t.Helper()
	cfgPath := os.Getenv("GIT_CONFIG_GLOBAL")
	if cfgPath == "" {
		t.Fatal("GIT_CONFIG_GLOBAL is unset: call redirectPackURL before running the binary, or the child clones from the network")
	}
	// os.Environ already carries it (t.Setenv), but re-appending makes the value
	// explicit and survives a caller that pared the inherited environment down.
	return append(os.Environ(), "GIT_CONFIG_GLOBAL="+cfgPath)
}

// newConsumerProject returns a temp project root carrying a backstop.yml the pack
// lifecycle commands can read and write.
func newConsumerProject(t *testing.T) string {
	t.Helper()
	proj := t.TempDir()
	if err := os.WriteFile(filepath.Join(proj, "backstop.yml"), []byte("project: hermetic-consumer\npacks: {}\n"), 0o644); err != nil {
		t.Fatalf("writing consumer backstop.yml: %v", err)
	}
	return proj
}

// productionPackURL mirrors distribution's resolveGitURL (add.go:315). The redirect is
// keyed on this exact string; if resolveGitURL's construction changes, this must change
// with it or every hermetic remote test starts reaching the network.
func productionPackURL(org, name string) string {
	return "https://github.com/" + org + "/" + name + ".git"
}

const (
	hermeticIdentityName  = "backstop hermetic harness"
	hermeticIdentityEmail = "hermetic@backstop.test"
	refTagsPrefix         = "refs/tags/"
)

// packVersionLine matches the manifest's top-level version line.
var packVersionLine = regexp.MustCompile(`(?m)^version:[^\n]*$`)

// setPackVersion rewrites the repository manifest's version and reports whether the
// file actually changed (so the caller only commits when there is something to commit).
func setPackVersion(t *testing.T, repoPath, version string) bool {
	t.Helper()
	manifest := filepath.Join(repoPath, "pack.yml")
	data, err := os.ReadFile(manifest)
	if os.IsNotExist(err) {
		return false
	}
	if err != nil {
		t.Fatalf("reading %s: %v", manifest, err)
	}
	if !packVersionLine.Match(data) {
		return false
	}
	updated := packVersionLine.ReplaceAll(data, []byte("version: "+version))
	if string(updated) == string(data) {
		return false
	}
	if writeErr := os.WriteFile(manifest, updated, 0o644); writeErr != nil {
		t.Fatalf("writing %s: %v", manifest, writeErr)
	}
	return true
}

// fileURL renders a filesystem path as the file:// URL git rewrites to.
func fileURL(path string) string {
	return "file://" + path
}

// requireGit skips rather than fails when git is absent: this harness drives the real
// binary, and an environment without it cannot honestly run these claims.
func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available on PATH; the hermetic remote harness drives the real git binary")
	}
}

// gitOutput runs git in dir with an explicit environment, returning combined output.
func gitOutput(t *testing.T, dir string, env []string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// mustGit runs a repository-construction git command, failing the test on error.
// It supplies an author/committer identity through the environment so construction
// works even when GIT_CONFIG_GLOBAL already points at a redirect config.
func mustGit(t *testing.T, repoPath string, args ...string) string {
	t.Helper()
	env := append(os.Environ(),
		"GIT_AUTHOR_NAME="+hermeticIdentityName,
		"GIT_AUTHOR_EMAIL="+hermeticIdentityEmail,
		"GIT_COMMITTER_NAME="+hermeticIdentityName,
		"GIT_COMMITTER_EMAIL="+hermeticIdentityEmail,
	)
	out, err := gitOutput(t, repoPath, env, args...)
	if err != nil {
		t.Fatalf("git %s in %s: %v\n%s", strings.Join(args, " "), repoPath, err, out)
	}
	return out
}
