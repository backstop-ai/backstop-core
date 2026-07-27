package recipe

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// The catcher for spec/code signature drift on pkg/recipe's declared contract
// surface (ISSUE-080 CLM-011).
//
// WHY A TEST AND NOT THE GATE: the contracts dimension compiles every `kind: type`
// entry to the ast-grep pattern `type <Name> $$$` — an EXISTENCE check. A stale
// `type WaiverReader func(rule string, file string) (covered bool)` and a correct
// `type WaiverReader func(rule string, file string) DivergenceVerdict` compile to
// the identical pattern and BOTH match the shipped declaration, so no gate
// dimension can go red on the difference. Without this file, CLM-011 would be a
// claim nothing could falsify — a green that means nothing.

// spec054Path locates the spec whose declared contracts this file holds the
// code to. Tests run with the package directory as the working directory.
func spec054Path() string {
	return filepath.Join("..", "..", "specs", "SPEC-054-recipe-apply-and-manifest.spec.md")
}

// contractedFile is the source file whose declared type contracts this guard
// covers. It is deliberately ONE file rather than all of pkg/recipe: the spec
// writes its signature strings WITHOUT struct tags while go/printer emits them, so
// the tag-bearing types in manifest.go and adoption.go cannot be compared until
// someone decides whether the spec's signature convention includes tags. That is a
// convention call, not something this test should settle by quietly stripping them.
const contractedFile = "pkg/recipe/apply.go"

// contractedTypes are the type entries compared: EVERY `kind: type` entry SPEC-054
// declares for contractedFile, not merely the ones ISSUE-080 changed.
//
// The scope is that file's whole surface deliberately. A guard covering only the
// three types one change happened to touch would leave the other four drifting
// silently — the gate cannot see them either, since it compiles every type entry to
// the existence-only pattern — and it would decay into a record of one past edit
// rather than a standing check. All seven were verified congruent when this widened,
// so it costs nothing today and catches the next edit wherever on the file it lands.
//
// ADDING a type entry to the spec for this file? Add it here too. The list is
// asserted to be COMPLETE below, so forgetting is a failure rather than a silent
// hole.
func contractedTypes() []string {
	return []string{
		"ApplyMode",
		"ApplyOptions",
		"TransformDispatch",
		"WaiverReader",
		"DivergenceVerdict",
		"ApplyResult",
		"PreservedDivergence",
	}
}

// specFrontmatter is the slice of SPEC-054's YAML frontmatter this file reads. The
// frontmatter is parsed as YAML rather than regexed out of the whole document, so
// a signature quoted in the spec's PROSE can never be mistaken for the contract.
type specFrontmatter struct {
	Contracts []struct {
		File     string `yaml:"file"`
		Provides []struct {
			Name      string `yaml:"name"`
			Kind      string `yaml:"kind"`
			Signature string `yaml:"signature"`
		} `yaml:"provides"`
	} `yaml:"contracts"`
}

// TestContract_SPEC054_DeclaredSignaturesMatchShippedTypes asserts SPEC-054's
// declared signature strings are TEXTUALLY equal to the type declarations parsed
// out of the shipped source (CLM-011).
//
// A missing spec file, an unreadable one, or an ABSENT contract entry is a
// FAILURE, never a skip. A skip here would be exactly the vacuous green this test
// exists to remove — and an absent entry is how a forgotten DivergenceVerdict
// declaration would otherwise slip through.
func TestContract_SPEC054_DeclaredSignaturesMatchShippedTypes(t *testing.T) {
	declared := declaredSignatures(t)
	shipped := shippedTypeDeclarations(t, "apply.go")

	// COMPLETENESS: every type entry the spec declares for this file must be in the
	// compared set. Without this, a type added to the spec later would sit outside
	// the guard and nothing would say so — the guard would silently narrow over
	// time, which is exactly how a standing check rots into a one-off.
	compared := make(map[string]bool, len(contractedTypes()))
	for _, name := range contractedTypes() {
		compared[name] = true
	}
	for name := range declared {
		if !compared[name] {
			t.Errorf("SPEC-054 declares a type contract for %q that this test does not compare; add it to contractedTypes() or the entry drifts unguarded", name)
		}
	}

	for _, name := range contractedTypes() {
		declaredSignature, present := declared[name]
		if !present {
			t.Errorf("SPEC-054 declares no contract entry for %q; the spec must describe every type on this surface, or a new one ships undocumented", name)
			continue
		}
		shippedDeclaration, found := shipped[name]
		if !found {
			t.Errorf("apply.go declares no type %q, but SPEC-054 contracts it as %q", name, declaredSignature)
			continue
		}

		want := canonicalSignature(declaredSignature)
		got := canonicalSignature(shippedDeclaration)
		if want != got {
			t.Errorf("SPEC-054 and the shipped source disagree about %s.\n  spec:    %s\n  shipped: %s\nThe spec is the contract every future reader and agent trusts; a stale signature misleads them and no gate dimension can catch it.", name, want, got)
		}
	}
}

// declaredSignatures reads SPEC-054's frontmatter and returns every contracted
// type's declared signature string, keyed by name.
func declaredSignatures(t *testing.T) map[string]string {
	t.Helper()

	path := spec054Path()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read SPEC-054 at %q: %v — the spec is the other half of this comparison, so its absence is a failure, not a reason to skip", path, err)
	}

	var parsed specFrontmatter
	if err := yaml.Unmarshal([]byte(frontmatter(t, string(raw))), &parsed); err != nil {
		t.Fatalf("parse SPEC-054 frontmatter as YAML: %v", err)
	}

	signatures := make(map[string]string)
	for _, contract := range parsed.Contracts {
		if contract.File != contractedFile {
			continue
		}
		for _, provided := range contract.Provides {
			if provided.Kind != "type" {
				continue
			}
			signatures[provided.Name] = provided.Signature
		}
	}
	if len(signatures) == 0 {
		t.Fatalf("SPEC-054's frontmatter declares no type contracts for %q; the parse found nothing to compare against", contractedFile)
	}

	return signatures
}

// frontmatter returns the document's YAML frontmatter — the text between the first
// two `---` fences.
func frontmatter(t *testing.T, document string) string {
	t.Helper()

	lines := strings.Split(document, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		t.Fatalf("SPEC-054 does not open with a `---` frontmatter fence")
	}

	for index := 1; index < len(lines); index++ {
		if strings.TrimSpace(lines[index]) == "---" {
			return strings.Join(lines[1:index], "\n")
		}
	}

	t.Fatalf("SPEC-054's frontmatter fence is never closed")
	return ""
}

// shippedTypeDeclarations parses a source file in this package and renders each
// type declaration back to source text, keyed by type name.
//
// It parses WITHOUT parser.ParseComments so go/printer cannot emit doc or field
// comments into the rendered form — a comparison against prose would fail for
// reasons that have nothing to do with drift.
//
// It renders the ENCLOSING *ast.GenDecl rather than the *ast.TypeSpec, because the
// TypeSpec alone renders as "WaiverReader func(…)" — WITHOUT the leading `type`
// keyword the spec's signature string carries. Resolving that by loosening the
// comparison to a substring or suffix check was considered and rejected: a
// substring check is exactly the lossy form that would let a real signature change
// slip through, which is the failure this whole file exists to prevent.
func shippedTypeDeclarations(t *testing.T, filename string) map[string]string {
	t.Helper()

	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, filename, nil, 0)
	if err != nil {
		t.Fatalf("parse %q: %v", filename, err)
	}

	declarations := make(map[string]string)
	for _, decl := range parsed.Decls {
		generic, isGeneric := decl.(*ast.GenDecl)
		if !isGeneric || generic.Tok != token.TYPE {
			continue
		}
		for _, spec := range generic.Specs {
			typeSpec, isType := spec.(*ast.TypeSpec)
			if !isType {
				continue
			}
			declarations[typeSpec.Name.Name] = renderTypeDeclaration(t, fset, generic, typeSpec)
		}
	}

	return declarations
}

// renderTypeDeclaration renders one type declaration WITH its `type` keyword. A
// solitary declaration is printed whole; a member of a grouped `type ( … )` block
// is printed as its spec with the keyword prefixed, since printing the group would
// drag in its siblings.
func renderTypeDeclaration(t *testing.T, fset *token.FileSet, generic *ast.GenDecl, typeSpec *ast.TypeSpec) string {
	t.Helper()

	if len(generic.Specs) == 1 {
		return renderNode(t, fset, generic)
	}

	return "type " + renderNode(t, fset, typeSpec)
}

// renderNode prints one AST node back to source text.
func renderNode(t *testing.T, fset *token.FileSet, node ast.Node) string {
	t.Helper()

	var buffer bytes.Buffer
	if err := printer.Fprint(&buffer, fset, node); err != nil {
		t.Fatalf("render declaration: %v", err)
	}

	return buffer.String()
}

// canonicalSignature reduces a signature to the form both sides are compared in:
// the spec writes struct fields separated by "; " on one line, the printer writes
// them one per line, and only that difference is normalized away. Field names,
// field ORDER and every type are preserved, so a genuine change still fails.
func canonicalSignature(signature string) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(signature, ";", " ")), " ")
}
