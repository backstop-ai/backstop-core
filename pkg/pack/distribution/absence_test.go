package distribution_test

import (
	"io/fs"

	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strings"
	"testing"
)

// absence_test.go proves three STRUCTURAL facts about the distribution package's
// own sources, not runtime behaviors: the options structs carry no dependency,
// nothing inside the package invents a dependency a caller failed to supply, and
// no package-level command entry point survives.
//
// They are source scans rather than behavioral tests because each of the three
// defects is INVISIBLE at run time. A reintroduced GitCloner field breaks
// nothing until someone leaves it nil; a fallback constructor call makes every
// test pass; a surviving free function is simply never called by the tests that
// exist. Each is caught here or not at all.
//
// The scan reads the real .go files on disk so a NEW violation introduced later
// is caught too, not merely today's known set.

// distributionDependencyTypes are the interfaces a lifecycle command is
// assembled from. A field of one of these types on an options value is exactly
// the shape that let cmd/backstop build a command with a nil dependency.
func distributionDependencyTypes() []string {
	return []string{"GitCloner", "Validator", "VersionResolver", "Scanner", "RemediationGenerator"}
}

// distributionDependencyConstructors are the production implementations. Calling
// one from inside this package would be the internal defaulting DD-30 resolved
// against: it makes a test double indistinguishable from production wiring,
// because a missing dependency silently becomes a working one.
func distributionDependencyConstructors() []string {
	return []string{"NewExecGitCloner", "NewPackvalValidator", "NewTagVersionResolver"}
}

// distributionOptionsStructs are the four options types the lifecycle commands
// take. They are named explicitly so a RENAMED struct fails the scan's own
// completeness check rather than quietly dropping out of coverage.
func distributionOptionsStructs() []string {
	return []string{"AddOptions", "InstallOptions", "UpdateOptions", "UpgradeOptions"}
}

// distributionCommandEntryPoints are the package-level pipeline functions
// REQ-007 deletes. Each took an options value, so each remained callable with
// every dependency nil — the bypass around the fail-closed constructors.
func distributionCommandEntryPoints() []string {
	return []string{"Add", "Install", "Update", "Upgrade"}
}

// parseDistributionSources parses the package's NON-TEST sources. Test files are
// excluded deliberately: a test may legitimately construct a production cloner
// or validator, and it is the shipped code these claims are about.
//
// A parse failure or an empty file set FAILS. A structural guard that cannot
// read its subject must not report the subject clean.
func parseDistributionSources(t *testing.T) (*token.FileSet, map[string]*ast.File) {
	t.Helper()

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(info fs.FileInfo) bool {
		return !strings.HasSuffix(info.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatalf("parsing the distribution package sources: %v; this guard cannot run without them and must not pass silently", err)
	}

	pkg, ok := pkgs["distribution"]
	if !ok || len(pkg.Files) == 0 {
		t.Fatal("found no non-test sources for package distribution; the source scan is broken, not the package")
	}

	return fset, pkg.Files
}

// TestDistribution_OptionsStructsCarryNoDependencyFields asserts none of the
// four options structs declares a dependency-typed field (CLM-030).
//
// This scan is the ACTUAL enforcement, not the contract declarations: the
// contracts pack's signature compiler reduces any struct to `type AddOptions
// $$$` and never compares field lists, so a reintroduced GitCloner field would
// still satisfy the declared contract.
func TestDistribution_OptionsStructsCarryNoDependencyFields(t *testing.T) {
	fset, files := parseDistributionSources(t)

	dependency := map[string]bool{}
	for _, name := range distributionDependencyTypes() {
		dependency[name] = true
	}

	found := map[string]bool{}
	for _, file := range files {
		ast.Inspect(file, func(node ast.Node) bool {
			spec, ok := node.(*ast.TypeSpec)
			if !ok {
				return true
			}
			structType, ok := spec.Type.(*ast.StructType)
			if !ok {
				return true
			}
			if !containsString(distributionOptionsStructs(), spec.Name.Name) {
				return true
			}
			found[spec.Name.Name] = true

			for _, field := range structType.Fields.List {
				typeName := baseTypeName(field.Type)
				if !dependency[typeName] {
					continue
				}
				for _, fieldName := range field.Names {
					t.Errorf("%s declares a %s field %q at %s; a dependency on an options value can be omitted, which is the hole the constructors close",
						spec.Name.Name, typeName, fieldName.Name, fset.Position(fieldName.Pos()))
				}
			}
			return true
		})
	}

	for _, name := range distributionOptionsStructs() {
		if !found[name] {
			t.Errorf("did not find a declaration of %s; it was renamed or removed, and this scan silently stopped covering it", name)
		}
	}
}

// TestDistribution_NoInternalDependencyDefaults asserts nothing in the package's
// shipped sources calls a production dependency constructor (CLM-031).
//
// A single fallback anywhere reinstates the defect DD-30 resolved against: a
// caller that supplies nothing gets a working command anyway, so no test can
// tell production wiring from a double.
func TestDistribution_NoInternalDependencyDefaults(t *testing.T) {
	fset, files := parseDistributionSources(t)

	for path, file := range files {
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			ident, ok := call.Fun.(*ast.Ident)
			if !ok {
				return true
			}
			if containsString(distributionDependencyConstructors(), ident.Name) {
				t.Errorf("%s calls %s at %s; the package must never substitute a dependency a caller failed to supply",
					path, ident.Name, fset.Position(call.Pos()))
			}
			return true
		})
	}
}

// TestDistribution_NoPackageLevelCommandEntryPoints asserts no top-level Add,
// Install, Update, or Upgrade function survives (CLM-049).
//
// The grep is for the DECLARATIONS themselves. A surviving free function taking
// an options value would remain callable with no dependency at all and would
// reinstate exactly the hole REQ-005 and REQ-006 close, whether or not anything
// currently calls it.
func TestDistribution_NoPackageLevelCommandEntryPoints(t *testing.T) {
	fset, files := parseDistributionSources(t)

	var survivors []string
	for _, file := range files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil {
				continue
			}
			if containsString(distributionCommandEntryPoints(), fn.Name.Name) {
				survivors = append(survivors, fn.Name.Name+" at "+fset.Position(fn.Pos()).String())
			}
		}
	}

	if len(survivors) > 0 {
		sort.Strings(survivors)
		t.Errorf("package-level command entry points survive:\n  %s\neach is callable with every dependency nil, bypassing the constructors entirely",
			strings.Join(survivors, "\n  "))
	}
}

// baseTypeName reduces a field's type expression to the bare identifier a
// dependency would appear as, seeing through a pointer or a package qualifier so
// `*GitCloner` and `distribution.GitCloner` are not read as unrelated types.
func baseTypeName(expr ast.Expr) string {
	switch typed := expr.(type) {
	case *ast.Ident:
		return typed.Name
	case *ast.StarExpr:
		return baseTypeName(typed.X)
	case *ast.SelectorExpr:
		return typed.Sel.Name
	default:
		return ""
	}
}
