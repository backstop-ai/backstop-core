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
	// Absent inverts the assertion: when true, the entry asserts the named
	// symbol must NOT exist in File. It passes iff the symbol is genuinely
	// absent from the named file and fails if the symbol reappears (a deletion
	// regression guard). An absent entry ignores Signature (mutually exclusive
	// intent — a deleted symbol has no signature to match). Unlike a present
	// entry, an absent entry on a missing or non-Go file is a loud config error,
	// not a silent skip — a silent pass would be vacuous green for an absence
	// assertion.
	Absent bool
}

// StepContractSignatureFunc returns a StepFunc that verifies spec contract
// declarations match actual code. Uses Go AST parsing.
func StepContractSignatureFunc(contracts []ContractEntry) StepFunc {
	return StepContractSignatureScopedFunc(contracts, nil)
}

// StepContractSignatureScopedFunc verifies contract files in scope.
func StepContractSignatureScopedFunc(contracts []ContractEntry, scope *GateScope) StepFunc {
	return func(_ context.Context) StepResult {
		var violations []Violation

		// Group contracts by file to avoid parsing the same file multiple times.
		byFile := make(map[string][]ContractEntry)
		for _, c := range contracts {
			if scope != nil && scope.Mode != GateScopeModeAll && !scope.Contains(c.File) {
				continue
			}
			byFile[c.File] = append(byFile[c.File], c)
		}

		for file, entries := range byFile {
			// Split each file's entries by intent. The file-level non-Go and
			// missing-file SKIPS below short-circuit only the present-only
			// entries (a deleted file cannot host a present symbol, so skipping
			// is acceptable). Absence assertions are entry-aware: an absent
			// entry on a missing or non-Go file is a LOUD config error, not a
			// silent skip, because "assert X absent from a file that isn't there
			// / can't be probed" would pass without probing anything — vacuous
			// green for an absence assertion.
			var presentEntries, absentEntries []ContractEntry
			for _, e := range entries {
				if e.Absent {
					absentEntries = append(absentEntries, e)
				} else {
					presentEntries = append(presentEntries, e)
				}
			}

			isGoFile := strings.HasSuffix(file, ".go")
			_, statErr := os.Stat(file)
			fileExists := statErr == nil

			// Absence assertions: loud config errors for non-Go / missing files
			// (cannot AST-probe), evaluated BEFORE any parse.
			for _, entry := range absentEntries {
				if !isGoFile {
					violations = append(violations, Violation{
						Rule:     "contract_signature",
						File:     file,
						Message:  fmt.Sprintf("absence assertion for symbol %s targets non-Go file %s: absence is an AST/source probe and only .go files are checkable", entry.Name, file),
						Severity: "error",
					})
					continue
				}
				if !fileExists {
					violations = append(violations, Violation{
						Rule:     "contract_signature",
						File:     file,
						Message:  fmt.Sprintf("absence assertion for symbol %s targets missing file %s: cannot probe absence in a file that does not exist", entry.Name, file),
						Severity: "error",
					})
					continue
				}
				if !isValidContractIdentifier(entry.Name) {
					violations = append(violations, Violation{
						Rule:     "contract_signature",
						File:     file,
						Message:  fmt.Sprintf("absence contract must name a real Go identifier, got %q: a descriptive slug is not AST-scannable", entry.Name),
						Severity: "error",
					})
					continue
				}
			}

			// Present-only entries keep today's exact skip behavior for non-Go
			// and missing files (unchanged present path).
			if !isGoFile || !fileExists {
				presentEntries = nil
			}

			// Nothing left to parse for this file.
			probeAbsent := absentEntriesNeedingProbe(absentEntries, isGoFile, fileExists)
			if len(presentEntries) == 0 && len(probeAbsent) == 0 {
				continue
			}

			fset := token.NewFileSet()
			parsed, err := parser.ParseFile(fset, file, nil, parser.AllErrors)
			if err != nil {
				for _, e := range presentEntries {
					violations = append(violations, Violation{
						Rule:     "contract_signature",
						File:     file,
						Message:  fmt.Sprintf("failed to parse file for symbol %s: %v", e.Name, err),
						Severity: "error",
					})
				}
				for _, e := range probeAbsent {
					violations = append(violations, Violation{
						Rule:     "contract_signature",
						File:     file,
						Message:  fmt.Sprintf("failed to parse file for absence assertion on symbol %s: %v", e.Name, err),
						Severity: "error",
					})
				}
				continue
			}

			// Absence probes: found==true → FAIL (regression caught);
			// found==false → PASS (deletion held, no violation).
			for _, entry := range probeAbsent {
				if !knownContractKind(entry.Kind) {
					violations = append(violations, Violation{
						Rule:     "contract_signature",
						File:     file,
						Message:  fmt.Sprintf("unknown contract kind %q for symbol %s", entry.Kind, entry.Name),
						Severity: "error",
					})
					continue
				}
				_, found := probeSymbol(fset, parsed, entry)
				if found {
					violations = append(violations, Violation{
						Rule:     "contract_signature",
						File:     file,
						Message:  fmt.Sprintf("symbol %s expected absent but present in %s (deletion regression)", entry.Name, file),
						Severity: "error",
					})
				}
			}

			for _, entry := range presentEntries {
				if !knownContractKind(entry.Kind) {
					violations = append(violations, Violation{
						Rule:     "contract_signature",
						File:     file,
						Message:  fmt.Sprintf("unknown contract kind %q for symbol %s", entry.Kind, entry.Name),
						Severity: "error",
					})
					continue
				}
				actualSig, found := probeSymbol(fset, parsed, entry)

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

// knownContractKind reports whether kind is a recognized contract symbol kind
// that probeSymbol can dispatch on. An unrecognized kind is a config error on
// both the present and absent paths.
func knownContractKind(kind string) bool {
	switch kind {
	case "function", "type", "interface", "variable", "constant", "method":
		return true
	default:
		return false
	}
}

// probeSymbol dispatches to the right find* probe for the entry's kind and
// returns the rendered signature (present path) and whether the symbol was
// found in the parsed file. An unknown kind returns ("", false); the caller is
// responsible for emitting the unknown-kind violation.
func probeSymbol(fset *token.FileSet, parsed *ast.File, entry ContractEntry) (string, bool) {
	switch entry.Kind {
	case "function":
		return findFunction(fset, parsed, entry.Name)
	case "type", "interface":
		return findType(parsed, entry.Name)
	case "variable":
		return findVariable(fset, parsed, entry.Name)
	case "constant":
		return findVariable(fset, parsed, entry.Name) // constants parsed like variables
	case "method":
		return findMethod(fset, parsed, entry.Name)
	default:
		return "", false
	}
}

// absentEntriesNeedingProbe returns the absence entries that survived the
// loud-config-error gates (Go file, exists on disk, valid identifier) and so
// must be AST-probed. Entries that already produced a config-error violation are
// excluded so they are not double-reported.
func absentEntriesNeedingProbe(absentEntries []ContractEntry, isGoFile, fileExists bool) []ContractEntry {
	if !isGoFile || !fileExists {
		return nil
	}
	var probe []ContractEntry
	for _, e := range absentEntries {
		if !isValidContractIdentifier(e.Name) {
			continue
		}
		probe = append(probe, e)
	}
	return probe
}

// isValidContractIdentifier reports whether name is usable as an absence-contract
// symbol name. The bare name must be a valid Go identifier so the AST scanner can
// probe for it; for the receiver-qualified method form "(*Type).method" or
// "(Type).method" the method component is validated (the receiver type is a
// disambiguator, not the probed identifier). A descriptive slug such as
// "bespoke-toolchain-tests" is rejected — it is not AST-scannable.
func isValidContractIdentifier(name string) bool {
	_, method := splitReceiverQualifiedName(name)
	return token.IsIdentifier(method)
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
	// A method contract name may be the bare method name (e.g. "RouteFile") or
	// the receiver-qualified form "(*Type).method" / "(Type).method" used to
	// disambiguate a method from a same-named function or another type's method.
	wantRecv, wantMethod := splitReceiverQualifiedName(name)

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != wantMethod {
			continue
		}
		if fn.Recv == nil {
			continue
		}
		if wantRecv != "" && receiverTypeName(fn.Recv) != wantRecv {
			continue
		}
		sig := formatMethodSignature(fset, fn)
		return sig, true
	}
	return "", false
}

// splitReceiverQualifiedName parses a method contract name. For the
// receiver-qualified form "(*Type).method" or "(Type).method" it returns the
// receiver type ("Type", pointer star stripped) and the method name. For a bare
// method name it returns an empty receiver and the name unchanged.
func splitReceiverQualifiedName(name string) (recv, method string) {
	if !strings.HasPrefix(name, "(") {
		return "", name
	}
	close := strings.Index(name, ")")
	if close < 0 || !strings.HasPrefix(name[close:], ").") {
		return "", name
	}
	recv = strings.TrimSpace(name[1:close])
	recv = strings.TrimPrefix(recv, "*")
	method = name[close+2:]
	return recv, method
}

// receiverTypeName returns the bare type name of a method receiver, stripping a
// leading pointer star (e.g. "*realCodeChecker" -> "realCodeChecker").
func receiverTypeName(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}
	expr := recv.List[0].Type
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name
	}
	return ""
}

// findType finds a type declaration by name and returns "type Name struct",
// "type Name interface", or — for a defined type over a concrete underlying
// type (e.g. "type InputMode string", "type Registry map[string]EngineBinding")
// — "type Name <underlying>" so a spec contract can pin the underlying type. A
// struct/interface body is summarized as the bare keyword (the spec convention),
// while a named/primitive underlying type is rendered so the contract signature
// matches what the spec author declared rather than collapsing to "type Name".
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
				if underlying := underlyingTypeString(ts.Type); underlying != "" {
					return "type " + name + " " + underlying, true
				}
				return "type " + name, true
			}
		}
	}
	return "", false
}

// underlyingTypeString renders the underlying type expression of a defined type
// (the right-hand side of a `type Name = expr`-free `type Name expr`) into the
// source-level string a spec contract would declare, e.g. "string",
// "map[string]EngineBinding", "[]byte", "*Provision", "pkg.Type". It returns ""
// for an expression shape it cannot render, in which case findType falls back to
// the bare "type Name" form.
func underlyingTypeString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		if x := underlyingTypeString(t.X); x != "" {
			return x + "." + t.Sel.Name
		}
		return ""
	case *ast.StarExpr:
		if x := underlyingTypeString(t.X); x != "" {
			return "*" + x
		}
		return ""
	case *ast.ArrayType:
		elem := underlyingTypeString(t.Elt)
		if elem == "" {
			return ""
		}
		if t.Len == nil {
			return "[]" + elem
		}
		return ""
	case *ast.MapType:
		k := underlyingTypeString(t.Key)
		v := underlyingTypeString(t.Value)
		if k == "" || v == "" {
			return ""
		}
		return "map[" + k + "]" + v
	default:
		return ""
	}
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
