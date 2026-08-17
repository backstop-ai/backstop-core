package main

import (
	"os"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/artifact"
)

// gate_substantiveness_layout_test.go pins the cmd/backstop half of ISSUE-124: both E2E
// workspace builders name their spec directory and their spec fixture through the shared
// artifact layout table rather than through private "specs" / ".spec.md" literals.
//
// ★ GREEN BEFORE AND AFTER THE SWAP, BY DESIGN — the same framing as the pkg/gate and
// pkg/validate pins. It falsifies a BOTCHED swap, not the duplication; the source scan in
// pkg/artifact/layout_consumer_scan_test.go is what falsifies the duplication.
//
// It lives in its own file rather than folded into the pkg/gate pin because it is the
// only pin in `package main`, and because the two builders are ISSUE-113 siblings whose
// exact shape four existing test files depend on — a reviewer needs to see this without
// reading a cross-package file.
//
// gate_substantiveness_e2e.go is a test HARNESS that happens not to be named _test.go,
// which is why it is IN the scan set's scope while its own _test.go callers are OUT. That
// is the ISSUE-124 boundary (non-test Go files only) working as specified.

// TestSubstantivenessE2E_WorkspaceSpecPathsComeFromSharedLayout (ISSUE-124 CLM-002).
//
// Both builders are driven for real, and both assertions derive from the shared table, so
// a private literal that drifts from it goes red here.
//
// ★ THE EXPECTED DIRECTORY IS RESOLVED VIA artifact.ResolveRoot, NOT BUILT AS A Root
// STRUCT LITERAL. PLAN-SPEC-068's notes record a Root-literal misuse that cost that lane
// a round: Root.Path is absolute-and-cleaned BY GUARANTEE of the resolver, and a
// hand-built literal skips the absolutization the production path performs — so the two
// sides can disagree for a reason that has nothing to do with the layout table.
//
// ★ AND THE FIXTURE FILENAME IS DISCOVERED BY READING THE DIRECTORY, never asserted
// against a name this test typed. A test that hardcoded "e2e.spec.md" would be an
// eighth private copy of the very table it is checking.
func TestSubstantivenessE2E_WorkspaceSpecPathsComeFromSharedLayout(t *testing.T) {
	specLayout, ok := artifact.LayoutFor(artifact.KindSpec)
	if !ok {
		t.Fatal("artifact.LayoutFor(KindSpec) reported ok=false; the shared spec layout is unavailable")
	}

	builders := map[string]func(string) (*e2eWorkspace, error){
		"newE2EWorkspace":          newE2EWorkspace,
		"newZeroMatchE2EWorkspace": newZeroMatchE2EWorkspace,
	}

	for name, build := range builders {
		t.Run(name, func(t *testing.T) {
			tmp := t.TempDir()

			workspace, err := build(tmp)
			if err != nil {
				t.Fatalf("%s(%s): %v", name, tmp, err)
			}

			root, resolveErr := artifact.ResolveRoot(tmp, "")
			if resolveErr != nil {
				t.Fatalf("artifact.ResolveRoot(%s, \"\"): %v", tmp, resolveErr)
			}
			wantDir := root.Dir(artifact.KindSpec)
			if workspace.specDir != wantDir {
				t.Fatalf("%s built its spec dir at %s, want %s.\nAn artifact type's directory is named by artifact.Root.Dir(kind); a hand-joined literal drifts from the shared table silently.",
					name, workspace.specDir, wantDir)
			}

			entries, readErr := os.ReadDir(workspace.specDir)
			if readErr != nil {
				t.Fatalf("reading %s's spec dir %s: %v", name, workspace.specDir, readErr)
			}
			specFiles := []string{}
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				if strings.HasSuffix(entry.Name(), specLayout.Extension) {
					specFiles = append(specFiles, entry.Name())
				}
			}
			if len(specFiles) != 1 {
				found := []string{}
				for _, entry := range entries {
					found = append(found, entry.Name())
				}
				t.Fatalf("%s's spec dir holds %d file(s) ending in the shared spec extension %q, want exactly 1. Directory contents: %v.\nThe fixture spec's name must be composed from artifact.LayoutFor(KindSpec).Extension — a fixture written with a private literal would still be discovered by the harness today and would drift the moment the table changed.",
					name, len(specFiles), specLayout.Extension, found)
			}
		})
	}
}
