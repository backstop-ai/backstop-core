package distribution_test

// Real-repository suite for the production VersionResolver (SPEC-055 CLM-029).
//
// This is the test that keeps the resolver from being mock-only. Everything it
// touches is local: the repository is built by the hermetic harness, and the
// production https:// URL reaches it through git's own url.insteadOf rewrite, so
// the resolver still asks for the URL production constructs. Nothing here reaches
// the network.

import (
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack/distribution"
)

// TestTagVersionResolver_ResolvesAgainstRealTaggedRepository proves resolution
// drives a REAL remote tag listing and selects the expected tag (CLM-029).
//
// The repository carries the shapes a stub cannot produce faithfully: an
// ANNOTATED tag, whose listing emits an extra peeled "^{}" entry, and a
// prerelease tag. It also carries a higher-major tag, so the same-major rule is
// exercised against real output rather than a hand-built slice.
//
// Resolution goes through the package's own URL construction — the test supplies
// a pack NAME, and the redirect is installed for exactly the URL that
// construction yields — so a resolver that accepted a URL directly, or that built
// a different one, finds no repository and fails here.
func TestTagVersionResolver_ResolvesAgainstRealTaggedRepository(t *testing.T) {
	packName := "hermetic/version-resolution"

	repo := newTaggedRepo(t, taggedRepoSpec{Revisions: []repoRevision{
		{Message: "release 1.0.0", Files: packTreeFiles(packName, "1.0.0"), Tags: []repoTag{{Name: "v1.0.0"}}},
		{Message: "release 1.2.0", Files: packTreeFiles(packName, "1.2.0"), Tags: []repoTag{{Name: "v1.2.0", Annotated: true}}},
		{Message: "release 1.10.0", Files: packTreeFiles(packName, "1.10.0"), Tags: []repoTag{{Name: "v1.10.0"}}},
		{Message: "release 2.0.0", Files: packTreeFiles(packName, "2.0.0"), Tags: []repoTag{{Name: "v2.0.0"}}},
		{Message: "prerelease and a moving pointer", Files: packTreeFiles(packName, "1.11.0"), Tags: []repoTag{
			{Name: "v1.11.0-rc1", Annotated: true},
			{Name: "latest"},
		}},
	}})

	// Prove the raw listing genuinely carries the shapes the assertion below
	// depends on. Without this, "the resolver ignored the peeled entry" would pass
	// over output that never contained one.
	rawRefs := lsRemoteTagRefs(t, repo.URL)
	assertRefsContain(t, rawRefs, "^{}", "a peeled ref (the annotated tags produce one)")
	assertRefsContain(t, rawRefs, "v2.0.0", "the higher-major tag the same-major rule must decline")
	assertRefsContain(t, rawRefs, "v1.11.0-rc1", "the prerelease tag the strict filter must ignore")

	productionURL := "https://github.com/" + packName + ".git"
	withGitConfigRedirect(t, productionURL, repo.URL)

	resolver, err := distribution.NewTagVersionResolver(distribution.NewExecGitCloner())
	if err != nil {
		t.Fatalf("assembling a resolver over the production cloner: %v", err)
	}

	resolved, err := resolver.ResolveLatestCompatible(packName, "1.0.0")
	if err != nil {
		t.Fatalf("resolving %s from 1.0.0 against %s: %v", packName, repo.URL, err)
	}

	// 1.10.0 is the highest same-major RELEASE. Picking 2.0.0 means the major
	// boundary was crossed; 1.11.0-rc1 means the prerelease filter is absent;
	// 1.2.0 means tags were compared as strings.
	if resolved != "1.10.0" {
		t.Errorf("resolved %q against the real repository, want 1.10.0", resolved)
	}
}

// assertRefsContain fails the test unless some raw ref line carries substring,
// naming what that shape is FOR so a fixture drift reads as a missing precondition
// rather than as an unexplained assertion.
func assertRefsContain(t *testing.T, refs []string, substring, purpose string) {
	t.Helper()

	for _, ref := range refs {
		if strings.Contains(ref, substring) {
			return
		}
	}

	t.Fatalf("the raw tag listing carries no %q — %s is missing, so the resolver assertion would pass vacuously:\n%s",
		substring, purpose, strings.Join(refs, "\n"))
}
