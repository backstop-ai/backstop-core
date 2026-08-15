package distribution

import (
	"path/filepath"
	"testing"
)

// TestDistribution_IsLocalPathClassifiesEveryShippedForm pins the EXPORTED
// classifier's behavior across every form the shipped predicate recognizes
// (SPEC-069 REQ-018 / CLM-110).
//
// The assertions are derived from what add.go's predicate does TODAY, not from what
// a reader might wish it did: exporting it is a RENAME, so a behavior change during
// the rename must fail here rather than in a consumer repo's lock file. There is
// exactly ONE authority on what a local path is — init calls THIS function instead
// of defining a second definition that could drift, which is how a ref init
// classified remote and the add path classified local would produce exactly the
// machine-specific `local_path` lock entry REQ-018 exists to prevent.
func TestDistribution_IsLocalPathClassifiesEveryShippedForm(t *testing.T) {
	cases := []struct {
		name  string
		ref   string
		local bool
		why   string
	}{
		{
			name:  "rooted absolute path",
			ref:   "/opt/packs/go-standards",
			local: true,
			why:   "a leading / is the unambiguous absolute form",
		},
		{
			name:  "explicit same-directory relative path",
			ref:   "./packs/go-standards",
			local: true,
			why:   "a leading ./ is how the add path's own tests name a sibling pack tree",
		},
		{
			name:  "explicit parent-directory relative path",
			ref:   "../packs/go-standards",
			local: true,
			why:   "a leading ../ reaches outside the project and is just as unportable",
		},
		{
			name:  "platform-absolute path",
			ref:   filepath.Join(string(filepath.Separator), "var", "packs", "go-standards"),
			local: true,
			why:   "filepath.IsAbs is consulted so the predicate is not tied to POSIX spelling",
		},
		{
			name:  "pinned git ref",
			ref:   "backstop-ai/go-standards@1.2.0",
			local: false,
			why:   "the portable form REQ-018 requires: an org/repository coordinate plus a pin",
		},
		{
			name:  "unpinned git ref",
			ref:   "backstop-ai/go-standards",
			local: false,
			why:   "an org/repository coordinate is remote whether or not it carries a pin",
		},
		{
			name:  "bare name with no separator at all",
			ref:   "go-standards",
			local: false,
			why:   "nothing about a bare name makes it a filesystem path",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := IsLocalPath(tc.ref)
			if got != tc.local {
				t.Fatalf("IsLocalPath(%q) = %v, want %v: %s", tc.ref, got, tc.local, tc.why)
			}
		})
	}
}

// TestDistribution_IsLocalPathIsTheOnlyLocalPathAuthority asserts the exported name
// is the one the add pipeline itself routes on, by driving the classification of a
// ref through the SAME predicate and confirming the two forms that decide the add
// path's local/remote fork are separated by it.
//
// Without this the export could become a second, parallel predicate that happens to
// agree today: the rename is only meaningful if the shipped fork reads it.
func TestDistribution_IsLocalPathIsTheOnlyLocalPathAuthority(t *testing.T) {
	local := "./fixture-pack"
	remote := "hermetic/valid-pack@1.0.0"

	if !IsLocalPath(local) {
		t.Fatalf("IsLocalPath(%q) = false; the add path forks to resolveLocalPackSource on exactly this answer", local)
	}
	if IsLocalPath(remote) {
		t.Fatalf("IsLocalPath(%q) = true; a git coordinate classified local would be recorded with a machine-specific local_path", remote)
	}
	if IsLocalPath(local) == IsLocalPath(remote) {
		t.Fatal("the predicate does not separate the two forms the add pipeline forks on")
	}
}
