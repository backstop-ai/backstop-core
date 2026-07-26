package distribution

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// defaultGitBinary is the executable ExecGitCloner invokes when GitBinary is empty.
// git is backstop's own distribution channel (BUNDLE-006 DD-1), which is why it is
// the one tool name allowed to appear here as a literal. No other tool, language,
// or platform name may.
const defaultGitBinary = "git"

// defaultGitTimeout bounds one git invocation when Timeout is unset. It is long
// enough for a large pack over a slow link and short enough that a wedged remote
// fails a build instead of hanging it forever.
const defaultGitTimeout = 5 * time.Minute

// gitDirectoryName is the repository metadata directory a clone leaves behind and
// that Clone strips before returning.
const gitDirectoryName = ".git"

// tagRefPrefix and peeledRefSuffix are the ref shapes remote tag listing emits. An
// annotated tag produces a second, peeled "<name>^{}" entry for the object the tag
// points at; both refer to the same tag, so the peeled one is dropped.
const (
	tagRefPrefix    = "refs/tags/"
	peeledRefSuffix = "^{}"
)

// optionPrefix is the leading character git reads as the start of an option. A URL
// or ref beginning with it is rejected before git is invoked, so a pack coordinate
// can never smuggle in a git option.
const optionPrefix = "-"

// ExecGitCloner is the production GitCloner: it clones and lists tags by running
// the real git binary.
//
// GitBinary is the executable to invoke; empty means "git" resolved from PATH at
// invocation time. It exists so the missing-binary diagnostic is testable without
// mutating the calling process's PATH. Timeout bounds a single invocation so a hung
// remote cannot wedge a build; zero means the package default.
//
// Neither field is a dependency: this type IS the concrete production
// implementation, and the no-internal-defaults rule forbids distribution
// synthesizing a GitCloner where a caller supplied none — not a concrete
// implementation carrying its own tunables.
type ExecGitCloner struct {
	GitBinary string
	Timeout   time.Duration
}

// NewExecGitCloner constructs the production cloner with the package's default
// timeout.
//
// It deliberately does NOT probe for the git binary. Command assembly must succeed
// in an airgapped environment where git is absent — `pack install --cache` never
// clones — so a missing binary surfaces at Clone or ListTags time as a typed
// GitError instead of failing construction.
func NewExecGitCloner() *ExecGitCloner {
	return &ExecGitCloner{Timeout: defaultGitTimeout}
}

// Clone materializes the repository at url, at exactly the ref named by version,
// into destDir. The callers that use it (add, install, update, upgrade) create
// destDir with os.MkdirTemp, so it is empty and git accepts it.
//
// The clone is shallow and single-ref: only the requested tag is fetched.
//
// After the clone completes and BEFORE returning, the ROOT .git directory is
// stripped, so destDir holds authored pack content only by the time any caller sees
// it. That is what makes two clones of one tag hash identically — they otherwise
// differ in reflog timestamps and object layout — and therefore what makes a pack's
// recorded content hash reproducible across machines.
//
// The strip removes exactly ONE path. It does not walk the tree: a pack may
// legitimately ship .gitignore files or a fixture tree, and removing those would
// corrupt authored content. It also covers only the source this cloner creates —
// a local-path pack is never cloned and still hashes whatever is on disk.
func (c *ExecGitCloner) Clone(url, version, destDir string) error {
	switch {
	case isOptionLike(url):
		return optionLikeArgumentError("clone", "repository URL", url)
	case isOptionLike(version):
		return optionLikeArgumentError("clone", "ref", version)
	}

	// The ref rides a glued --branch=<ref> rather than a positional after "--",
	// because git clone takes no positional ref; gluing it to the flag is what makes
	// it unreadable as an option. url and destDir do go after "--".
	args := []string{
		"clone",
		"--depth=1",
		"--single-branch",
		"--branch=" + version,
		"--",
		url,
		destDir,
	}

	if _, err := c.run("clone", url, version, args); err != nil {
		return err
	}

	return stripRootGitDirectory(destDir, url, version)
}

// ListTags returns the bare tag names the repository at url publishes.
//
// Peeled "^{}" entries are excluded HERE, where the ref shape is known, rather than
// downstream: a consumer filtering on a version pattern would hide the duplicate
// today and let it through the moment that pattern relaxes.
//
// A failing listing returns an error, never an empty slice — an empty list would
// surface to an operator as "already at the latest version", which would be a lie.
func (c *ExecGitCloner) ListTags(url string) ([]string, error) {
	if isOptionLike(url) {
		return nil, optionLikeArgumentError("ls-remote", "repository URL", url)
	}

	output, err := c.run("ls-remote", url, "", []string{"ls-remote", "--tags", "--", url})
	if err != nil {
		// The wrap states the CONSEQUENCE the git-level diagnostic cannot: this is
		// the listing a version resolution reads, so a caller that swallowed it
		// would report "already at the latest version" instead of a failure.
		return nil, fmt.Errorf("cannot determine the available versions of %s: %w", url, err)
	}

	tags := make([]string, 0)
	for _, line := range strings.Split(output, "\n") {
		_, ref, found := strings.Cut(strings.TrimSpace(line), "\t")
		if !found {
			continue
		}

		ref = strings.TrimSpace(ref)
		if !strings.HasPrefix(ref, tagRefPrefix) {
			continue
		}

		name := strings.TrimPrefix(ref, tagRefPrefix)
		if strings.HasSuffix(name, peeledRefSuffix) {
			continue
		}

		tags = append(tags, name)
	}

	return tags, nil
}

// run invokes git with args, returning its stdout. Every failure comes back as a
// *GitError naming the operation, the repository, the ref where one applies, and
// whatever git wrote to stderr.
func (c *ExecGitCloner) run(operation, url, ref string, args []string) (string, error) {
	timeout := c.invocationTimeout()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	command := exec.CommandContext(ctx, c.binary(), args...)
	command.Env = append(os.Environ(), nonInteractiveGitEnv(os.Environ())...)

	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	err := command.Run()
	target := describeTarget(url, ref)

	switch {
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		return "", &GitError{Message: fmt.Sprintf(
			"git %s of %s timed out after %s: %s",
			operation, target, timeout, gitOutput(&stderr))}
	case errors.Is(err, exec.ErrNotFound), errors.Is(err, fs.ErrNotExist):
		return "", &GitError{Message: fmt.Sprintf(
			"git %s of %s could not run the git executable %q: %v",
			operation, target, c.binary(), err)}
	case err != nil:
		return "", &GitError{Message: fmt.Sprintf(
			"git %s of %s failed: %v: %s",
			operation, target, err, gitOutput(&stderr))}
	}

	return stdout.String(), nil
}

// binary resolves the git executable to invoke.
func (c *ExecGitCloner) binary() string {
	if c.GitBinary != "" {
		return c.GitBinary
	}

	return defaultGitBinary
}

// invocationTimeout resolves the bound for one git invocation, so a zero-valued
// ExecGitCloner is still bounded rather than able to hang forever.
func (c *ExecGitCloner) invocationTimeout() time.Duration {
	if c.Timeout > 0 {
		return c.Timeout
	}

	return defaultGitTimeout
}

// nonInteractiveGitEnv is the environment overlay that keeps git from blocking on a
// credential or host-key prompt. It is APPENDED to the ambient environment, never
// substituted for it.
//
// The inheritance is LOAD-BEARING BEHAVIOR, not incidental: a consumer's configured
// credential helper and any url.insteadOf redirection live in the ambient
// environment, and the hermetic test suite proves the production URL is redirected
// at a local repository through exactly that mechanism. A future "hygiene" change
// that clears the environment silently converts every hermetic test into a network
// call.
//
// For the same reason the batch-mode transport default is applied only when the
// ambient environment does not already set one: overriding a consumer's own
// transport configuration would be a different kind of clobbering.
func nonInteractiveGitEnv(ambient []string) []string {
	overlay := []string{"GIT_TERMINAL_PROMPT=0"}

	if !environmentDefines(ambient, "GIT_SSH_COMMAND") {
		overlay = append(overlay, "GIT_SSH_COMMAND=ssh -o BatchMode=yes")
	}

	return overlay
}

// environmentDefines reports whether environment carries an assignment for name.
func environmentDefines(environment []string, name string) bool {
	prefix := name + "="
	for _, entry := range environment {
		if strings.HasPrefix(entry, prefix) {
			return true
		}
	}

	return false
}

// stripRootGitDirectory removes the repository metadata directory a clone leaves at
// the root of destDir.
//
// A failed strip is an error, not a warning: handing back a partially stripped tree
// would let a caller record a content hash nothing can reproduce.
func stripRootGitDirectory(destDir, url, ref string) error {
	if err := os.RemoveAll(filepath.Join(destDir, gitDirectoryName)); err != nil {
		return &GitError{Message: fmt.Sprintf(
			"git clone of %s materialized %s but its %s directory could not be removed (%v); refusing to hand back a partially stripped tree",
			describeTarget(url, ref), destDir, gitDirectoryName, err)}
	}

	return nil
}

// isOptionLike reports whether git would read value as an option rather than as the
// repository or ref it was supposed to be.
func isOptionLike(value string) bool {
	return strings.HasPrefix(value, optionPrefix)
}

// optionLikeArgumentError is the refusal returned for such a value, BEFORE any
// subprocess runs. role names what the value was supposed to be, so the diagnostic
// says which coordinate is malformed.
func optionLikeArgumentError(operation, role, value string) error {
	return &GitError{Message: fmt.Sprintf(
		"git %s refused: the %s %q begins with %q, which git would read as an option; git was not invoked",
		operation, role, value, optionPrefix)}
}

// describeTarget renders the repository, and the ref where one applies, for a
// diagnostic. Naming both is what lets an operator tell WHICH pack at WHICH version
// failed.
func describeTarget(url, ref string) string {
	if ref == "" {
		return fmt.Sprintf("repository %s", url)
	}

	return fmt.Sprintf("repository %s at ref %s", url, ref)
}

// gitOutput renders captured git output for a diagnostic, naming its absence rather
// than trailing off into empty space.
func gitOutput(captured *bytes.Buffer) string {
	trimmed := strings.TrimSpace(captured.String())
	if trimmed == "" {
		return "git wrote nothing to stderr"
	}

	return trimmed
}
