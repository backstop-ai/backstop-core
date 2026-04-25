package gate

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"strings"
)

// ContractEntry represents a declared symbol in a spec's contracts section.
type ContractEntry struct {
	File      string // path to the file containing the symbol
	Name      string // symbol name
	Kind      string // "function", "type", "variable", "interface"
	Signature string // declared signature
}

// StepContractSignatureFunc returns a StepFunc that verifies spec contract
// declarations match actual code. Uses Go AST parsing.
func StepContractSignatureFunc(contracts []ContractEntry) StepFunc {
	return func(_ context.Context) StepResult {
		var violations []Violation

		// Group contracts by file to avoid parsing the same file multiple times.
		byFile := make(map[string][]ContractEntry)
		for _, c := range contracts {
			byFile[c.File] = append(byFile[c.File], c)
		}

		for file, entries := range byFile {
			// Skip non-Go files — the Go parser can only handle .go files.
			// Specs may reference non-Go files (JSON schemas, YAML configs,
			// shell scripts, markdown) in their contracts for documentation
			// purposes. These are not checkable via AST parsing.
			if !strings.HasSuffix(file, ".go") {
				continue
			}

			// Skip files that don't exist (may have been deleted/moved).
			if _, statErr := os.Stat(file); statErr != nil {
				continue
			}

			fset := token.NewFileSet()
			parsed, err := parser.ParseFile(fset, file, nil, parser.AllErrors)
			if err != nil {
				for _, e := range entries {
					violations = append(violations, Violation{
						Rule:     "contract_signature",
						File:     file,
						Message:  fmt.Sprintf("failed to parse file for symbol %s: %v", e.Name, err),
						Severity: "error",
					})
				}
				continue
			}

			for _, entry := range entries {
				var actualSig string
				var found bool

				switch entry.Kind {
				case "function":
					actualSig, found = findFunction(fset, parsed, entry.Name)
				case "type", "interface":
					actualSig, found = findType(parsed, entry.Name)
				case "variable":
					actualSig, found = findVariable(fset, parsed, entry.Name)
				case "constant":
					actualSig, found = findVariable(fset, parsed, entry.Name) // constants parsed like variables
				case "method":
					actualSig, found = findMethod(fset, parsed, entry.Name)
				default:
					violations = append(violations, Violation{
						Rule:     "contract_signature",
						File:     file,
						Message:  fmt.Sprintf("unknown contract kind %q for symbol %s", entry.Kind, entry.Name),
						Severity: "error",
					})
					continue
				}

				if !found {
					violations = append(violations, Violation{
						Rule:     "contract_signature",
						File:     file,
						Message:  fmt.Sprintf("symbol %s not found in %s", entry.Name, file),
						Severity: "error",
					})
					continue
				}

				if !signaturesMatch(entry.Signature, actualSig) {
					violations = append(violations, Violation{
						Rule:     "contract_signature",
						File:     file,
						Message:  fmt.Sprintf("symbol %s signature mismatch: expected %q, got %q", entry.Name, entry.Signature, actualSig),
						Severity: "error",
					})
				}
			}
		}

		status := "pass"
		if len(violations) > 0 {
			status = "fail"
		}
		if violations == nil {
			violations = []Violation{}
		}
		return StepResult{
			StepName:   StepContractSignature,
			Status:     status,
			Violations: violations,
		}
	}
}

// findFunction finds a function declaration by name and returns its signature.
func findFunction(fset *token.FileSet, file *ast.File, name string) (string, bool) {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != name {
			continue
		}
		// Skip methods (have a receiver)
		if fn.Recv != nil {
			continue
		}
		sig := formatFuncSignature(fset, fn)
		return sig, true
	}
	return "", false
}

// findMethod finds a method declaration (function with receiver) by name and returns its signature.
func findMethod(fset *token.FileSet, file *ast.File, name string) (string, bool) {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != name {
			continue
		}
		if fn.Recv == nil {
			continue
		}
		sig := formatMethodSignature(fset, fn)
		return sig, true
	}
	return "", false
}

// findType finds a type declaration by name and returns "type Name struct"
// or "type Name interface".
func findType(file *ast.File, name string) (string, bool) {
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, spec := range gen.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok || ts.Name.Name != name {
				continue
			}
			switch ts.Type.(type) {
			case *ast.StructType:
				return "type " + name + " struct", true
			case *ast.InterfaceType:
				return "type " + name + " interface", true
			default:
				return "type " + name, true
			}
		}
	}
	return "", false
}

// findVariable finds a var declaration by name and returns its signature.
func findVariable(fset *token.FileSet, file *ast.File, name string) (string, bool) {
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || (gen.Tok != token.VAR && gen.Tok != token.CONST) {
			continue
		}
		for _, spec := range gen.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for _, ident := range vs.Names {
				if ident.Name == name {
					prefix := "var"
					if gen.Tok == token.CONST {
						prefix = "const"
					}
					if vs.Type != nil {
						var buf bytes.Buffer
						if err := printer.Fprint(&buf, fset, vs.Type); err == nil {
							return prefix + " " + name + " " + buf.String(), true
						}
					}
					return prefix + " " + name, true
				}
			}
		}
	}
	return "", false
}

// formatFuncSignature formats a function declaration as its signature string.
func formatFuncSignature(fset *token.FileSet, fn *ast.FuncDecl) string {
	var buf bytes.Buffer
	buf.WriteString("func ")
	buf.WriteString(fn.Name.Name)

	// Print params
	buf.WriteString("(")
	if fn.Type.Params != nil {
		printFieldList(&buf, fset, fn.Type.Params.List)
	}
	buf.WriteString(")")

	// Print results
	if fn.Type.Results != nil && len(fn.Type.Results.List) > 0 {
		buf.WriteString(" ")
		if len(fn.Type.Results.List) == 1 && len(fn.Type.Results.List[0].Names) == 0 {
			var tbuf bytes.Buffer
			if err := printer.Fprint(&tbuf, fset, fn.Type.Results.List[0].Type); err == nil {
				buf.WriteString(tbuf.String())
			}
		} else {
			buf.WriteString("(")
			printFieldList(&buf, fset, fn.Type.Results.List)
			buf.WriteString(")")
		}
	}

	return buf.String()
}

// formatMethodSignature formats a method declaration as its signature string,
// including the receiver type.
func formatMethodSignature(fset *token.FileSet, fn *ast.FuncDecl) string {
	var buf bytes.Buffer
	buf.WriteString("func ")

	// Print receiver
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		buf.WriteString("(")
		printFieldList(&buf, fset, fn.Recv.List)
		buf.WriteString(") ")
	}

	buf.WriteString(fn.Name.Name)

	// Print params
	buf.WriteString("(")
	if fn.Type.Params != nil {
		printFieldList(&buf, fset, fn.Type.Params.List)
	}
	buf.WriteString(")")

	// Print results
	if fn.Type.Results != nil && len(fn.Type.Results.List) > 0 {
		buf.WriteString(" ")
		if len(fn.Type.Results.List) == 1 && len(fn.Type.Results.List[0].Names) == 0 {
			var tbuf bytes.Buffer
			if err := printer.Fprint(&tbuf, fset, fn.Type.Results.List[0].Type); err == nil {
				buf.WriteString(tbuf.String())
			}
		} else {
			buf.WriteString("(")
			printFieldList(&buf, fset, fn.Type.Results.List)
			buf.WriteString(")")
		}
	}

	return buf.String()
}

// printFieldList formats a field list (params or results) as a comma-separated string.
func printFieldList(buf *bytes.Buffer, fset *token.FileSet, fields []*ast.Field) {
	for i, field := range fields {
		if i > 0 {
			buf.WriteString(", ")
		}
		// Print names
		for j, name := range field.Names {
			if j > 0 {
				buf.WriteString(", ")
			}
			buf.WriteString(name.Name)
		}
		if len(field.Names) > 0 {
			buf.WriteString(" ")
		}
		// Print type
		var tbuf bytes.Buffer
		if err := printer.Fprint(&tbuf, fset, field.Type); err == nil {
			buf.WriteString(tbuf.String())
		}
	}
}

// signaturesMatch compares two signatures with normalized whitespace.
func signaturesMatch(expected, actual string) bool {
	return normalizeSignature(expected) == normalizeSignature(actual)
}

// normalizeSignature normalizes whitespace in a signature for comparison.
func normalizeSignature(sig string) string {
	// Collapse whitespace
	fields := strings.Fields(sig)
	return strings.Join(fields, " ")
}
