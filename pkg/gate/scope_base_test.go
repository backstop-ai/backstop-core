package gate

import (
	"path/filepath"
	"strings"
	"testing"
)

// These cover the EXPLICIT diff base — `gate --base <rev>`.
//
// The flag exists because of one measured fact about CI: a GitHub Actions checkout
// is PRISTINE. Bare diff mode resolves its base as merge-base(HEAD, origin/main),
// and on a push to main that resolves to HEAD itself, so `git diff --name-only HEAD`
// returns nothing and there are no untracked files either. The result is not a wrong
// scope, it is an EMPTY one — and an empty scope passes every dimension. That is the
// vacuous green this whole flag exists to remove, so the tests below assert the
// CONTRAST (bare mode empty, base mode non-empty) rather than base mode alone.
//
// The git fixtures follow TestComputeGateScope_DiffAgainstRemoteBranch
// (scope_test.go) and reuse its runGit/writeFile helpers — there is one harness.

// initRepoWithBaseCommit creates a git repo on branch `main` carrying one commit,
// and returns the repo root and that commit's SHA.
func initRepoWithBaseCommit(t *testing.T) (string, string) {
	t.Helper()

	root := t.TempDir()
	runGit(t, root, "init")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "Test User")
	// Name the branch explicitly: the default branch name is a git CONFIG value and
	// differs across versions, so a test that assumed one would fail for a reason
	// that has nothing to do with base resolution.
	runGit(t, root, "checkout", "-b", "main")

	writeFile(t, filepath.Join(root, "base.go"), "package main\n")
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "commit A")

	return root, gitRevParse(t, root, "HEAD")
}

// gitRevParse returns the resolved SHA of a revision.
func gitRevParse(t *testing.T, root, rev string) string {
	t.Helper()

	out, err := gitOutput(root, "rev-parse", rev)
	if err != nil {
		t.Fatalf("git rev-parse %s: %v", rev, err)
	}
	return strings.TrimSpace(out)
}

// TestComputeGateScope_DiffAgainstExplicitBase asserts that with commits A then B,
// resolving against base=A yields exactly the files B changed. (CLM-005)
func TestComputeGateScope_DiffAgainstExplicitBase(t *testing.T) {
	root, baseSHA := initRepoWithBaseCommit(t)

	writeFile(t, filepath.Join(root, "changed.go"), "package main\nfunc changed() {}\n")
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "commit B")

	scope, err := ComputeGateScopeWithBase(root, GateScopeModeDiff, nil, baseSHA)
	if err != nil {
		t.Fatalf("ComputeGateScopeWithBase(base=A): %v", err)
	}

	if !scope.Contains("changed.go") {
		t.Errorf("scope must contain the file commit B added, got %#v", scope.Files)
	}
	// base.go was committed IN A, so it is not part of the diff SINCE A.
	if scope.Contains("base.go") {
		t.Errorf("scope must not contain a file that was already present at the base, got %#v", scope.Files)
	}
	if scope.Mode != GateScopeModeDiff {
		t.Errorf("scope mode = %q, want %q", scope.Mode, GateScopeModeDiff)
	}
}

// TestComputeGateScope_ExplicitBaseOnCleanTreeIsNonEmpty is THE CI CONDITION and
// the reason this flag exists.
//
// Both halves are asserted in ONE test so the CONTRAST is what fails if someone
// reintroduces a fallback: on a fully-committed tree with no origin/main ref, bare
// diff mode sees NOTHING while base mode sees commit B's files. A test that checked
// only the base-mode half would still pass if base mode quietly degraded into bare
// mode. (CLM-006)
func TestComputeGateScope_ExplicitBaseOnCleanTreeIsNonEmpty(t *testing.T) {
	root, baseSHA := initRepoWithBaseCommit(t)

	writeFile(t, filepath.Join(root, "changed.go"), "package main\nfunc changed() {}\n")
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "commit B")
	// Nothing uncommitted, and deliberately NO refs/remotes/origin/main — exactly
	// the shape actions/checkout produces.

	bare, err := ComputeGateScope(root, GateScopeModeDiff, nil)
	if err != nil {
		t.Fatalf("ComputeGateScope (bare diff): %v", err)
	}
	if len(bare.Files) != 0 {
		t.Fatalf("PRECONDITION FAILED: bare diff mode on a pristine tree must see NO files "+
			"(this is the CI vacuity being demonstrated), got %#v", bare.Files)
	}

	based, err := ComputeGateScopeWithBase(root, GateScopeModeDiff, nil, baseSHA)
	if err != nil {
		t.Fatalf("ComputeGateScopeWithBase(base=A): %v", err)
	}
	if len(based.Files) == 0 {
		t.Fatal("explicit base on a pristine tree must produce a NON-EMPTY scope — " +
			"an empty scope here is the vacuous green --base exists to prevent")
	}
	if !based.Contains("changed.go") {
		t.Errorf("explicit-base scope must contain commit B's file, got %#v", based.Files)
	}
}

// TestComputeGateScope_ExplicitBaseUsesMergeBaseNotTwoDot asserts the resolution is
// `git diff $(git merge-base HEAD <base>)` and NOT a two-dot diff against the ref.
//
// This is what makes ONE code path correct for BOTH CI events: for a pull_request
// the base sha is the fork point, and for a push the before-sha is already an
// ancestor of HEAD so the merge-base is the before-sha itself. Under a two-dot diff
// the side branch's file would appear in scope as a DELETION, which would put a file
// the branch never touched into the blocking set. (CLM-005)
func TestComputeGateScope_ExplicitBaseUsesMergeBaseNotTwoDot(t *testing.T) {
	root, forkPoint := initRepoWithBaseCommit(t)

	// A side branch off the fork point, carrying a file main never sees.
	runGit(t, root, "checkout", "-b", "side")
	writeFile(t, filepath.Join(root, "only_on_side.go"), "package main\n")
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "commit C (side)")
	sideSHA := gitRevParse(t, root, "HEAD")

	// main advances independently, so the two have genuinely diverged.
	runGit(t, root, "checkout", "main")
	writeFile(t, filepath.Join(root, "only_on_main.go"), "package main\n")
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "commit B (main)")

	scope, err := ComputeGateScopeWithBase(root, GateScopeModeDiff, nil, sideSHA)
	if err != nil {
		t.Fatalf("ComputeGateScopeWithBase(base=side): %v", err)
	}

	if !scope.Contains("only_on_main.go") {
		t.Errorf("scope must contain the file added since the merge-base, got %#v", scope.Files)
	}
	if scope.Contains("only_on_side.go") {
		t.Errorf("scope must NOT contain the side branch's file — that is the two-dot "+
			"symmetric difference, not the merge-base diff; got %#v", scope.Files)
	}

	// The resolved merge-base is the FORK POINT, not the side branch's tip.
	if scope.MergeBase != forkPoint {
		t.Errorf("resolved merge-base = %q, want the fork point %q", scope.MergeBase, forkPoint)
	}
	if scope.RequestedBase != sideSHA {
		t.Errorf("requested base = %q, want %q", scope.RequestedBase, sideSHA)
	}
}

// TestComputeGateScope_UnresolvableBaseIsHardError is THE ANTI-FALLBACK TEST and the
// most important one in this file.
//
// The existing fallbacks in resolveGateScopeDiff (to --all when not a git repo, to
// local changes when no remote exists) are PRECISELY why CI is silent today.
// Inheriting any of them here would defeat the flag, so an unresolvable base must
// produce an error AND no scope a caller could mistake for a valid one. (CLM-007)
func TestComputeGateScope_UnresolvableBaseIsHardError(t *testing.T) {
	root, _ := initRepoWithBaseCommit(t)

	writeFile(t, filepath.Join(root, "changed.go"), "package main\n")
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "commit B")

	const missingRef = "no-such-ref-anywhere"
	scope, err := ComputeGateScopeWithBase(root, GateScopeModeDiff, nil, missingRef)
	if err == nil {
		t.Fatalf("an unresolvable base must be a hard error, got scope %#v", scope)
	}
	if !strings.Contains(err.Error(), missingRef) {
		t.Errorf("the error must NAME the ref that could not resolve; got %q", err)
	}

	assertNoUsableScope(t, scope, "unresolvable base")
}

// TestComputeGateScope_BaseWithNoMergeBaseIsHardError covers the OTHER failure mode
// — the one that is easy to implement and easy to forget to prove.
//
// The rev RESOLVES, so a rev-parse check alone passes; it is merge-base that fails.
// A force-push or a grafted shallow clone produces exactly this shape in CI, and a
// guard shaped only around "does the rev exist" lets the dead sha through to a
// confusing exit 2. (CLM-007)
func TestComputeGateScope_BaseWithNoMergeBaseIsHardError(t *testing.T) {
	root, _ := initRepoWithBaseCommit(t)

	// An orphan branch is a root commit sharing NO ancestry with main.
	runGit(t, root, "checkout", "--orphan", "unrelated")
	writeFile(t, filepath.Join(root, "unrelated.go"), "package main\n")
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "unrelated root commit")
	orphanSHA := gitRevParse(t, root, "HEAD")

	runGit(t, root, "checkout", "main")
	headSHA := gitRevParse(t, root, "HEAD")

	scope, err := ComputeGateScopeWithBase(root, GateScopeModeDiff, nil, orphanSHA)
	if err == nil {
		t.Fatalf("a base sharing no merge-base with HEAD must be a hard error, got scope %#v", scope)
	}
	// Name BOTH revs: "no merge base" without saying between what is unactionable.
	for label, rev := range map[string]string{"the requested base": orphanSHA, "HEAD": headSHA} {
		if !strings.Contains(err.Error(), rev) {
			t.Errorf("the error must name %s (%s); got %q", label, rev, err)
		}
	}

	assertNoUsableScope(t, scope, "base with no merge-base")
}

// TestComputeGateScope_ExplicitBaseIncludesUntrackedFiles asserts parity with bare
// diff mode, which has included untracked files since ISSUE-004. A new file is the
// most common thing a change adds, and omitting it would let brand-new code through
// unchecked — the exact opposite of the flag's purpose. (CLM-005)
func TestComputeGateScope_ExplicitBaseIncludesUntrackedFiles(t *testing.T) {
	root, baseSHA := initRepoWithBaseCommit(t)

	writeFile(t, filepath.Join(root, "committed.go"), "package main\n")
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "commit B")

	writeFile(t, filepath.Join(root, "untracked.go"), "package main\n")

	scope, err := ComputeGateScopeWithBase(root, GateScopeModeDiff, nil, baseSHA)
	if err != nil {
		t.Fatalf("ComputeGateScopeWithBase: %v", err)
	}

	if !scope.Contains("untracked.go") {
		t.Errorf("explicit-base scope must include untracked files, got %#v", scope.Files)
	}
	if !scope.Contains("committed.go") {
		t.Errorf("explicit-base scope must still include committed changes, got %#v", scope.Files)
	}
}

// TestComputeGateScope_EmptyBaseKeepsExistingDiffBehavior is the DELEGATION GUARD
// for CLM-009: there must be exactly ONE implementation of "what changed".
//
// ComputeGateScope keeps its signature and delegates with an empty base, so passing
// an empty base must reproduce bare diff mode EXACTLY — same files, same warnings.
// If a second resolver is ever grown alongside the first, these two diverge.
func TestComputeGateScope_EmptyBaseKeepsExistingDiffBehavior(t *testing.T) {
	root, _ := initRepoWithBaseCommit(t)

	// The remote-tracking ref bare diff mode looks for, so this exercises the
	// remote-branch path rather than the no-remote fallback.
	runGit(t, root, "update-ref", "refs/remotes/origin/main", "HEAD")
	writeFile(t, filepath.Join(root, "base.go"), "package main\nfunc changed() {}\n")
	writeFile(t, filepath.Join(root, "brand_new.go"), "package main\n")

	viaWrapper, wrapperErr := ComputeGateScope(root, GateScopeModeDiff, nil)
	if wrapperErr != nil {
		t.Fatalf("ComputeGateScope: %v", wrapperErr)
	}
	viaEmptyBase, baseErr := ComputeGateScopeWithBase(root, GateScopeModeDiff, nil, "")
	if baseErr != nil {
		t.Fatalf("ComputeGateScopeWithBase(base=\"\"): %v", baseErr)
	}

	if len(viaWrapper.Files) == 0 {
		t.Fatal("PRECONDITION FAILED: the wrapper produced an empty scope, so the " +
			"equivalence below would hold vacuously")
	}
	if strings.Join(viaWrapper.Files, ",") != strings.Join(viaEmptyBase.Files, ",") {
		t.Errorf("empty base must reproduce bare diff mode exactly:\n  wrapper:    %#v\n  empty base: %#v",
			viaWrapper.Files, viaEmptyBase.Files)
	}
	if strings.Join(viaWrapper.Warnings, "|") != strings.Join(viaEmptyBase.Warnings, "|") {
		t.Errorf("empty base must reproduce bare diff mode's warnings exactly:\n  wrapper:    %#v\n  empty base: %#v",
			viaWrapper.Warnings, viaEmptyBase.Warnings)
	}
}

// assertNoUsableScope pins the three non-fallbacks a failed base resolution must
// never take: no silent --all, no HEAD diff, no empty-but-usable scope. Asserting
// only "err != nil" would pass against an implementation that returned an error
// AND a full-project scope the caller then ran with.
func assertNoUsableScope(t *testing.T, scope *GateScope, context string) {
	t.Helper()

	if scope == nil {
		return // Nothing usable was returned; that is the strongest possible outcome.
	}
	if scope.Mode == GateScopeModeAll {
		t.Errorf("%s: returned a scope in --all mode — a failed base resolution must never "+
			"silently widen to the whole project", context)
	}
	if len(scope.Files) > 0 {
		t.Errorf("%s: returned a non-empty file list %#v — a failed base resolution must not "+
			"fall back to a HEAD diff", context, scope.Files)
	}
	if scope.Mode != GateScopeModeAll && len(scope.Files) == 0 && !scope.Empty() {
		t.Errorf("%s: returned a scope that reports itself non-empty while carrying no files", context)
	}
}

// TestFormatHuman_ReportsExplicitBaseResolution covers the human-output half of the
// scope reporting: when a run used an EXPLICIT base, the report must name the mode,
// the requested base, the resolved merge-base and the in-scope file count.
//
// The CLI-level test for this lives in cmd/backstop, which is a different package
// and therefore does not exercise this formatter's branch. Without a test HERE the
// branch is uncovered — and, more to the point, an unreported base is how a CI run
// over zero files reads as an ordinary green.
func TestFormatHuman_ReportsExplicitBaseResolution(t *testing.T) {
	scope := newGateScope("", GateScopeModeDiff, []string{"changed.go"}, nil)
	scope.RequestedBase = "origin/main"
	scope.MergeBase = "abc1234def5678"

	result := NewGateResultWithScope([]StepResult{{StepName: "artifact_validation", Status: "pass"}}, scope)
	output := FormatHuman(result, true)

	for label, want := range map[string]string{
		"scope mode":          string(GateScopeModeDiff),
		"requested base":      "origin/main",
		"resolved merge-base": "abc1234def5678",
		"in-scope file count": "1",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("human output must report the %s (%q); output:\n%s", label, want, output)
		}
	}

	// A run WITHOUT an explicit base must not grow a base line naming nothing.
	plain := newGateScope("", GateScopeModeDiff, []string{"changed.go"}, nil)
	plainOutput := FormatHuman(NewGateResultWithScope([]StepResult{{StepName: "artifact_validation", Status: "pass"}}, plain), true)
	if strings.Contains(plainOutput, "requested base") {
		t.Errorf("a run with no explicit base must not report one; output:\n%s", plainOutput)
	}
}
