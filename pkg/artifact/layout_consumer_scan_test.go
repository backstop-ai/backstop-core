package artifact

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// layout_consumer_scan_test.go is the AUTHORITY FENCE for the type→{Directory,
// Extension} table declared in layout.go (ISSUE-124).
//
// ★ WHY THIS IS THE FALSIFYING TEST AND NOT A BEHAVIORAL ONE. Every swap ISSUE-124
// mandates is behavior-preserving BY CONSTRUCTION: LayoutFor(KindSpec).Extension
// returns the string ".spec.md", so there is no input for which a consumer's private
// literal and the shared lookup differ. The defect IS the duplication, and duplication
// is observable at exactly one level — the source. A behavioral test would be green on
// both sides of the fix and calling it falsification would be a lie.
//
// ★ THE SCAN SET INCLUDES THREE ALREADY-CLEAN FILES ON PURPOSE. resolved_by.go,
// delivered_by.go and artifact_status.go were converted by SPEC-068. They are the
// regression fence AND the COMMENT-STRIPPING CANARY: all three carry the forbidden
// literals in DOC COMMENTS and none in code, so a scan that read raw bytes instead of
// parsed code would fire on them immediately. Comments are excluded STRUCTURALLY here —
// the file is parsed WITHOUT parser.ParseComments and only *ast.BasicLit nodes are
// examined, and a comment is not a BasicLit. If one of those three ever goes red, the
// scan is wrong; do NOT "fix" it by deleting the comments, which are correct.
//
// ★ MATCHING IS ASYMMETRIC AND THAT IS LOAD-BEARING. Extensions match by SUBSTRING
// because four of the occurrences are COMPOSED literals (".epic.bundle.md",
// "e2e.spec.md", "zeromatch.spec.md") that an equality test would miss entirely —
// leaving a green scan over unconverted code. Directories match by EQUALITY because
// Contains would fire on ordinary prose such as "resolving specs under %s".
//
// ★ THE DENYLIST IS DERIVED, NEVER TYPED. It is built by ranging Kinds() and reading
// LayoutFor. A hand-written list of seven extensions inside the test that forbids
// copies of the table would itself be the eighth copy.
//
// ★ `.epic` IS NOT FORBIDDEN. Per artifacts/bundle/v1/schema.json the bundle filename
// pattern is `^[a-z0-9-]+(\.epic)?\.bundle\.md$` — `.epic` is bundle-SCHEMA vocabulary
// that sits inside the stem, not a layout extension, so it is absent from the derived
// denylist. That is precisely why the denylist is derived: `".epic" + layout.Extension`
// composed at runtime is fine, while the literal ".epic.bundle.md" is not.
//
// Two things are deliberately OUT of scope. pkg/scaffold/scaffold.go's FileExtension is
// scaffold's own sanctioned local concern (SPEC-068 TASK-019). `_test.go` files are a
// different and much larger change; the boundary is mechanical and checkable — non-test
// Go files only. That makes cmd/backstop/gate_substantiveness_e2e.go IN scope (a test
// harness that is not named _test.go) while its own _test.go callers are OUT, which is
// the boundary working as specified rather than a defect.

// layoutScanSet is the repository-relative set of non-test Go files that must hold no
// private copy of the layout table.
//
// SHARP EDGE — THE SCAN BOUNDARY IS ITSELF A CLAIM. A denylist over a hand-typed file
// list silently empties when a file is renamed or moved, and an empty denylist passes
// forever. TestLayout_ConsumerScanBoundaryIsNonEmptyAndEveryScopedPathExists exists so
// that cannot happen.
func layoutScanSet() []string {
	return []string{
		// The six ISSUE-124 converts.
		filepath.Join("pkg", "gate", "step_testverify.go"),
		filepath.Join("pkg", "validate", "spec.go"),
		filepath.Join("pkg", "validate", "adr.go"),
		filepath.Join("pkg", "validate", "bundle.go"),
		filepath.Join("pkg", "validate", "supports_resolution.go"),
		filepath.Join("cmd", "backstop", "gate_substantiveness_e2e.go"),
		// The three SPEC-068 converts — regression fence and comment-stripping canary.
		filepath.Join("pkg", "validate", "resolved_by.go"),
		filepath.Join("pkg", "validate", "delivered_by.go"),
		filepath.Join("pkg", "gate", "artifact_status.go"),
	}
}

// layoutScanRoot walks up from the working directory to the directory holding go.mod.
func layoutScanRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("resolving working directory: %v", err)
	}
	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("walked to the filesystem root without finding go.mod")
		}
		dir = parent
	}
}

// scannedLiteral is one string literal with its REAL source line.
type scannedLiteral struct {
	line  int
	value string
}

// stringLiteralsIn returns every string literal in path, unquoted, each carrying the
// line it occupies in the FILE ON DISK.
//
// ★ REAL SOURCE POSITIONS, NOT POST-STRIP ONES. The established idiom in
// pkg/initialize/sourceset_scan_test.go re-prints the comment-free AST via go/printer
// and scans that string, which is fine for a yes/no claim but yields line numbers from
// the re-printed rendering. The gap is large enough to mislead — step_testverify.go's
// first spec-discovery guard is post-strip line 81 and real source line 120, and a
// developer handed "step_testverify.go:81" by a failure message reads the wrong
// function. So this walks the AST instead and reports fset.Position(lit.Pos()).Line.
//
// Comments are still excluded, structurally: parsing without parser.ParseComments drops
// them, and a comment is not a *ast.BasicLit regardless.
func stringLiteralsIn(t *testing.T, path string) []scannedLiteral {
	t.Helper()

	fset := token.NewFileSet()
	// Parsed WITHOUT parser.ParseComments — the zero mode is what drops the comments.
	parsed, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		t.Fatalf("parsing %s: %v", path, err)
	}

	literals := []scannedLiteral{}
	ast.Inspect(parsed, func(node ast.Node) bool {
		lit, isLit := node.(*ast.BasicLit)
		if !isLit || lit.Kind != token.STRING {
			return true
		}
		unquoted, unquoteErr := strconv.Unquote(lit.Value)
		if unquoteErr != nil {
			return true
		}
		literals = append(literals, scannedLiteral{line: fset.Position(lit.Pos()).Line, value: unquoted})
		return true
	})
	return literals
}

// TestLayout_NoConsumerCarriesAPrivateArtifactExtensionLiteral (ISSUE-124 CLM-001,
// CLM-007).
//
// THE FALSIFYING TEST. No file in the scan set may hold a string literal CONTAINING any
// kind's extension. Every consumer reads its extension from LayoutFor or decides kind
// through ClassifyFilename.
//
// Substring, not equality — see the header note. Every hit is reported rather than
// stopping at the first, because the point of the scan is the whole census.
func TestLayout_NoConsumerCarriesAPrivateArtifactExtensionLiteral(t *testing.T) {
	root := layoutScanRoot(t)

	hits := []string{}
	for _, relative := range layoutScanSet() {
		for _, literal := range stringLiteralsIn(t, filepath.Join(root, relative)) {
			for _, kind := range Kinds() {
				layout, ok := LayoutFor(kind)
				if !ok {
					t.Fatalf("LayoutFor(%q) reported ok=false for a kind Kinds() enumerated", kind)
				}
				if strings.Contains(literal.value, layout.Extension) {
					hits = append(hits, filepath.ToSlash(relative)+":"+itoa(literal.line)+
						" holds "+strconv.Quote(literal.value)+", which contains the "+string(kind)+
						" extension "+strconv.Quote(layout.Extension))
				}
			}
		}
	}

	if len(hits) != 0 {
		t.Fatalf("%d private artifact-extension literal(s) survive outside the shared layout table:\n  %s\n\nEach must read its extension from artifact.LayoutFor(kind).Extension, or decide the kind through artifact.ClassifyFilename when the question is \"is this file kind X\". A second copy of the table drifts from the first silently.",
			len(hits), strings.Join(hits, "\n  "))
	}
}

// TestLayout_NoConsumerCarriesAPrivateArtifactDirectoryLiteral (ISSUE-124 CLM-002).
//
// The directory half. EQUALITY, not Contains: an artifact directory name is a common
// English word, and Contains would fire on ordinary prose like "resolving specs under
// %s". Artifact-type directories are named through Root.Dir(kind), never joined by hand.
func TestLayout_NoConsumerCarriesAPrivateArtifactDirectoryLiteral(t *testing.T) {
	root := layoutScanRoot(t)

	hits := []string{}
	for _, relative := range layoutScanSet() {
		for _, literal := range stringLiteralsIn(t, filepath.Join(root, relative)) {
			for _, kind := range Kinds() {
				layout, ok := LayoutFor(kind)
				if !ok {
					t.Fatalf("LayoutFor(%q) reported ok=false for a kind Kinds() enumerated", kind)
				}
				if literal.value == layout.Directory {
					hits = append(hits, filepath.ToSlash(relative)+":"+itoa(literal.line)+
						" holds "+strconv.Quote(literal.value)+", which is the "+string(kind)+
						" directory name")
				}
			}
		}
	}

	if len(hits) != 0 {
		t.Fatalf("%d private artifact-directory literal(s) survive outside the shared layout table:\n  %s\n\nAn artifact type's directory is named by artifact.Root.Dir(kind), which is what keeps the artifact root in one place.",
			len(hits), strings.Join(hits, "\n  "))
	}
}

// TestLayout_ConsumerScanBoundaryIsNonEmptyAndEveryScopedPathExists (ISSUE-124 CLM-007).
//
// THE GUARD ON THE GUARD. Without this, a rename would empty the scan set and leave the
// two denylists above passing forever over nothing. Three properties: the set is
// non-empty, every path in it is really on disk (naming any that is not), and the
// DERIVED denylist is non-empty with no empty entry in it — an empty extension would
// make strings.Contains true for every literal, and an empty directory would match every
// bare "".
func TestLayout_ConsumerScanBoundaryIsNonEmptyAndEveryScopedPathExists(t *testing.T) {
	root := layoutScanRoot(t)

	set := layoutScanSet()
	if len(set) == 0 {
		t.Fatal("the consumer scan set is EMPTY; both denylists above would pass while checking no file at all")
	}

	missing := []string{}
	for _, relative := range set {
		if _, err := os.Stat(filepath.Join(root, relative)); err != nil {
			missing = append(missing, filepath.ToSlash(relative))
		}
	}
	if len(missing) != 0 {
		t.Fatalf("the consumer scan set names %d path(s) absent from disk: %s\n\nA scanned file that moved silently drops out of both denylists above. Update the set to the file's new home rather than removing the entry.",
			len(missing), strings.Join(missing, ", "))
	}

	kinds := Kinds()
	if len(kinds) == 0 {
		t.Fatal("Kinds() is EMPTY, so the derived denylist is empty and both scans above match nothing")
	}
	for _, kind := range kinds {
		layout, ok := LayoutFor(kind)
		if !ok {
			t.Fatalf("LayoutFor(%q) reported ok=false for a kind Kinds() enumerated", kind)
		}
		if layout.Extension == "" {
			t.Fatalf("kind %q has an EMPTY extension; strings.Contains would then be true for every literal and the extension scan would report the whole file", kind)
		}
		if layout.Directory == "" {
			t.Fatalf("kind %q has an EMPTY directory name; the directory scan would then match every empty string literal", kind)
		}
	}
}

// itoa renders a line number without pulling fmt into a file whose whole job is reading
// other files' literals.
func itoa(n int) string {
	return strconv.Itoa(n)
}
