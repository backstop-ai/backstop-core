package main

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack/distribution"
)

// pack_wiring_test.go proves the production assembly seam (SPEC-055 REQ-010): that
// cmd/backstop builds every pack lifecycle command's dependencies for real, that the
// two capabilities this spec does NOT build fail loud instead of vacuously, and that no
// command reaches the pipeline through a dependency-carrying options literal any more.

// TestProductionPackCommands_AssembleCompletely (CLM-061) — every production helper
// returns a usable command.
//
// BOTH halves are asserted on purpose. A helper returning (nil, nil) satisfies an
// error-only check and then nil-dereferences at first use, which is precisely the
// failure mode ISSUE-073 was: assembly that "succeeded" and blew up later.
func TestProductionPackCommands_AssembleCompletely(t *testing.T) {
	addCmd, err := newProductionAddCommand()
	if err != nil {
		t.Errorf("newProductionAddCommand returned an error: %v", err)
	}
	if addCmd == nil {
		t.Error("newProductionAddCommand returned a nil command; pack add would nil-dereference at its first clone")
	}

	installCmd, err := newProductionInstallCommand()
	if err != nil {
		t.Errorf("newProductionInstallCommand returned an error: %v", err)
	}
	if installCmd == nil {
		t.Error("newProductionInstallCommand returned a nil command; pack install would nil-dereference at its first clone")
	}

	updateCmd, err := newProductionUpdateCommand()
	if err != nil {
		t.Errorf("newProductionUpdateCommand returned an error: %v", err)
	}
	if updateCmd == nil {
		t.Error("newProductionUpdateCommand returned a nil command; pack update would nil-dereference at its first resolution")
	}

	upgradeCmd, err := newProductionUpgradeCommand()
	if err != nil {
		t.Errorf("newProductionUpgradeCommand returned an error: %v", err)
	}
	if upgradeCmd == nil {
		t.Error("newProductionUpgradeCommand returned a nil command; pack upgrade would nil-dereference at its first clone")
	}
}

// TestProductionUpgradeCapabilities_NameCapabilityAndReference (CLM-091) — the two
// production stubs standing in for capabilities this spec does not build return a typed
// *CapabilityUnavailableError naming BOTH the capability and the requirement tracking it.
//
// The REQUIREMENT REFERENCE is the assertion that matters. A bare "not implemented"
// reads to an operator as a defect in backstop; naming BUNDLE-006 REQ-014 / REQ-018
// points at scheduled work. And returning an error at all — rather than an empty
// violation slice — is what keeps `pack upgrade` from reporting a successful upgrade
// with zero baselined violations it never scanned for.
func TestProductionUpgradeCapabilities_NameCapabilityAndReference(t *testing.T) {
	violations, scanErr := unavailableScanner{}.ScanViolations(t.TempDir(), t.TempDir())
	if len(violations) != 0 {
		t.Errorf("the unavailable scanner must report no violations it did not scan for, got %d", len(violations))
	}
	assertCapabilityUnavailable(t, "violation scan", scanErr, "REQ-014")

	bundle, genErr := unavailableRemediationGenerator{}.GenerateBundle(t.TempDir(), []string{"some-violation"})
	if bundle != "" {
		t.Errorf("the unavailable remediation generator must name no bundle it did not generate, got %q", bundle)
	}
	assertCapabilityUnavailable(t, "remediation bundle generation", genErr, "REQ-018")
}

// assertCapabilityUnavailable requires err to be a *CapabilityUnavailableError whose
// rendered message names a capability and cites requirement.
func assertCapabilityUnavailable(t *testing.T, what string, err error, requirement string) {
	t.Helper()

	if err == nil {
		t.Fatalf("%s is not built by this spec; it must fail loud rather than return a vacuous success", what)
	}

	var unavailable *distribution.CapabilityUnavailableError
	if !errors.As(err, &unavailable) {
		t.Fatalf("%s must fail with *distribution.CapabilityUnavailableError so the command can classify it, got %T: %v", what, err, err)
	}

	if strings.TrimSpace(unavailable.Capability) == "" {
		t.Errorf("%s: the error names no capability; an operator cannot tell what is missing", what)
	}
	if !strings.Contains(unavailable.Reference, requirement) {
		t.Errorf("%s: the error must cite %s as the requirement tracking the gap, got reference %q", what, requirement, unavailable.Reference)
	}

	message := unavailable.Error()
	if !strings.Contains(message, unavailable.Capability) {
		t.Errorf("%s: the rendered message %q must name the capability %q", what, message, unavailable.Capability)
	}
	if !strings.Contains(message, requirement) {
		t.Errorf("%s: the rendered message %q must cite %s, or it reads as a defect rather than as scheduled work", what, message, requirement)
	}
	if !strings.Contains(message, bundleReference) {
		t.Errorf("%s: the rendered message %q must name %s, so the requirement number is resolvable", what, message, bundleReference)
	}
}

// bundleReference is the bundle whose requirements track the two unbuilt capabilities.
const bundleReference = "BUNDLE-006"

// isDependencyOptionsType reports whether name is one of the distribution options types
// whose dependency fields this spec removes.
func isDependencyOptionsType(name string) bool {
	switch name {
	case "AddOptions", "InstallOptions", "UpdateOptions", "UpgradeOptions":
		return true
	}
	return false
}

// isOptionsDependencyField reports whether name is one of the dependency fields those
// options types must no longer carry.
func isOptionsDependencyField(name string) bool {
	switch name {
	case "GitCloner", "Validator", "VersionResolver", "Scanner", "RemediationGenerator":
		return true
	}
	return false
}

// TestPackCommands_ConstructNoDependencyCarryingOptions (CLM-067, kind: absence) — no
// cmd/backstop production source builds a distribution options value carrying a
// dependency.
//
// It PARSES THE SOURCES ON DISK rather than checking a curated list of known-bad lines,
// because the property is about sites that do not exist yet: a command added next year
// that reintroduces `AddOptions{GitCloner: ...}` must fail this test without anyone
// remembering to extend it. Assembly happens in pack_wiring.go and nowhere else.
//
// It scans NON-TEST sources: the claim is about what a pack lifecycle COMMAND
// constructs, and a test that wires a double is not a command.
func TestPackCommands_ConstructNoDependencyCarryingOptions(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("reading the cmd/backstop sources: %v", err)
	}

	scanned := 0
	fset := token.NewFileSet()
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}

		file, parseErr := parser.ParseFile(fset, name, nil, 0)
		if parseErr != nil {
			t.Fatalf("parsing %s: %v", name, parseErr)
		}
		scanned++
		reportDependencyCarryingOptions(t, fset, file)
	}

	// A scan that walked nothing would pass silently and prove nothing.
	if scanned == 0 {
		t.Fatal("scanned no cmd/backstop production sources; the absence claim would be vacuous")
	}
}

// reportDependencyCarryingOptions reports every options literal in file that carries a
// dependency field, and every later assignment of a dependency field onto a variable
// bound to such a literal — the two ways a dependency can still reach an options value.
func reportDependencyCarryingOptions(t *testing.T, fset *token.FileSet, file *ast.File) {
	t.Helper()

	// Variables bound to an options literal, so `opts := distribution.AddOptions{}`
	// followed by `opts.GitCloner = c` is caught as well as the direct literal.
	optionsVars := map[string]bool{}

	ast.Inspect(file, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.CompositeLit:
			if !isDependencyOptionsLiteral(typed) {
				return true
			}
			for _, element := range typed.Elts {
				pair, ok := element.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				key, ok := pair.Key.(*ast.Ident)
				if !ok || !isOptionsDependencyField(key.Name) {
					continue
				}
				t.Errorf("%s: %s options literal carries dependency field %s; production dependencies are assembled in pack_wiring.go, never passed through an options value",
					fset.Position(pair.Pos()), optionsTypeName(typed), key.Name)
			}

		case *ast.AssignStmt:
			for index, rhs := range typed.Rhs {
				literal, ok := unwrapCompositeLit(rhs)
				if !ok || !isDependencyOptionsLiteral(literal) || index >= len(typed.Lhs) {
					continue
				}
				if target, ok := typed.Lhs[index].(*ast.Ident); ok {
					optionsVars[target.Name] = true
				}
			}
			for _, lhs := range typed.Lhs {
				selector, ok := lhs.(*ast.SelectorExpr)
				if !ok || !isOptionsDependencyField(selector.Sel.Name) {
					continue
				}
				base, ok := selector.X.(*ast.Ident)
				if !ok || !optionsVars[base.Name] {
					continue
				}
				t.Errorf("%s: %s.%s is assigned onto an options value; production dependencies are assembled in pack_wiring.go, never set on an options value",
					fset.Position(selector.Pos()), base.Name, selector.Sel.Name)
			}
		}
		return true
	})
}

// isDependencyOptionsLiteral reports whether literal constructs one of the four
// distribution options types.
func isDependencyOptionsLiteral(literal *ast.CompositeLit) bool {
	return isDependencyOptionsType(optionsTypeName(literal))
}

// optionsTypeName renders the literal's type name, matching both the qualified
// `distribution.AddOptions` form and a bare `AddOptions` should the package ever be
// dot-imported.
func optionsTypeName(literal *ast.CompositeLit) string {
	switch typed := literal.Type.(type) {
	case *ast.SelectorExpr:
		return typed.Sel.Name
	case *ast.Ident:
		return typed.Name
	}
	return ""
}

// unwrapCompositeLit returns the composite literal behind expr, seeing through the
// address-of a `&distribution.AddOptions{...}` would carry.
func unwrapCompositeLit(expr ast.Expr) (*ast.CompositeLit, bool) {
	if unary, ok := expr.(*ast.UnaryExpr); ok && unary.Op == token.AND {
		expr = unary.X
	}
	literal, ok := expr.(*ast.CompositeLit)
	return literal, ok
}
