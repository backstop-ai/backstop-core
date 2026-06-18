package gate

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"
)

// TestGate_ContractSignature_MatchingSignaturePasses verifies pass when a
// declared contract function exists with matching signature.
func TestGate_ContractSignature_MatchingSignaturePasses(t *testing.T) {
	target, err := filepath.Abs("testdata/contract-target.go")
	if err != nil {
		t.Fatal(err)
	}

	contracts := []ContractEntry{
		{File: target, Name: "DoSomething", Kind: "function", Signature: "func DoSomething(ctx context.Context, name string) error"},
	}
	step := StepContractSignatureFunc(contracts)
	result := step(context.Background())

	if result.StepName != StepContractSignature {
		t.Errorf("expected step_name %q, got %q", StepContractSignature, result.StepName)
	}
	if result.Status != "pass" {
		t.Errorf("expected status %q, got %q; violations: %v", "pass", result.Status, result.Violations)
	}
}

// TestGate_ContractSignature_MissingFunctionFails verifies fail when a
// declared contract function is missing from the file.
func TestGate_ContractSignature_MissingFunctionFails(t *testing.T) {
	target, err := filepath.Abs("testdata/contract-target.go")
	if err != nil {
		t.Fatal(err)
	}

	contracts := []ContractEntry{
		{File: target, Name: "NonExistentFunc", Kind: "function", Signature: "func NonExistentFunc() error"},
	}
	step := StepContractSignatureFunc(contracts)
	result := step(context.Background())

	if result.Status != "fail" {
		t.Errorf("expected status %q, got %q", "fail", result.Status)
	}
	if len(result.Violations) == 0 {
		t.Error("expected at least one violation for missing function")
	}
}

// TestGate_ContractSignature_WrongSignatureFails verifies fail when a
// declared function exists but signature differs.
func TestGate_ContractSignature_WrongSignatureFails(t *testing.T) {
	target, err := filepath.Abs("testdata/contract-target.go")
	if err != nil {
		t.Fatal(err)
	}

	// DoSomething exists but we declare a wrong signature
	contracts := []ContractEntry{
		{File: target, Name: "DoSomething", Kind: "function", Signature: "func DoSomething(name string) string"},
	}
	step := StepContractSignatureFunc(contracts)
	result := step(context.Background())

	if result.Status != "fail" {
		t.Errorf("expected status %q, got %q", "fail", result.Status)
	}
	if len(result.Violations) == 0 {
		t.Error("expected at least one violation for wrong signature")
	}
}

// TestGate_ContractSignature_TypeAndInterfaceVerified verifies that contract
// types and interfaces are verified, not just functions.
func TestGate_ContractSignature_TypeAndInterfaceVerified(t *testing.T) {
	target, err := filepath.Abs("testdata/contract-target.go")
	if err != nil {
		t.Fatal(err)
	}

	contracts := []ContractEntry{
		{File: target, Name: "Widget", Kind: "type", Signature: "type Widget struct"},
		{File: target, Name: "Runner", Kind: "interface", Signature: "type Runner interface"},
	}
	step := StepContractSignatureFunc(contracts)
	result := step(context.Background())

	if result.Status != "pass" {
		t.Errorf("expected status %q, got %q; violations: %v", "pass", result.Status, result.Violations)
	}
}

// TestGate_ContractSignature_VariableVerified verifies that contract variable
// declarations are verified by the contract signature step.
func TestGate_ContractSignature_VariableVerified(t *testing.T) {
	target, err := filepath.Abs("testdata/contract-target.go")
	if err != nil {
		t.Fatal(err)
	}

	contracts := []ContractEntry{
		{File: target, Name: "DefaultTimeout", Kind: "variable", Signature: "var DefaultTimeout int"},
	}
	step := StepContractSignatureFunc(contracts)
	result := step(context.Background())

	if result.Status != "pass" {
		t.Errorf("expected status %q, got %q; violations: %v", "pass", result.Status, result.Violations)
	}
}

// TestGate_FindVariable_Found verifies that findVariable locates a var
// declaration and returns its signature.
func TestGate_FindVariable_Found(t *testing.T) {
	target, err := filepath.Abs("testdata/contract-target.go")
	if err != nil {
		t.Fatal(err)
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, target, nil, 0)
	if err != nil {
		t.Fatalf("parsing file: %v", err)
	}

	sig, ok := findVariable(fset, f, "DefaultTimeout")
	if !ok {
		t.Fatal("expected to find variable DefaultTimeout")
	}
	if sig != "var DefaultTimeout int" {
		t.Errorf("expected signature %q, got %q", "var DefaultTimeout int", sig)
	}
}

// TestGate_FindVariable_NotFound verifies that findVariable returns false
// for a variable name that does not exist.
func TestGate_FindVariable_NotFound(t *testing.T) {
	target, err := filepath.Abs("testdata/contract-target.go")
	if err != nil {
		t.Fatal(err)
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, target, nil, 0)
	if err != nil {
		t.Fatalf("parsing file: %v", err)
	}

	_, ok := findVariable(fset, f, "NonExistentVar")
	if ok {
		t.Error("expected findVariable to return false for missing variable")
	}
}

func TestGateSteps_FilterToChangedFiles_Contract(t *testing.T) {
	target, err := filepath.Abs("testdata/contract-target.go")
	if err != nil {
		t.Fatal(err)
	}
	result := StepContractSignatureScopedFunc([]ContractEntry{
		{File: target, Name: "DoSomething", Kind: "function", Signature: "func DoSomething(ctx context.Context, name string) error"},
		{File: "unchanged.go", Name: "Missing", Kind: "function", Signature: "func Missing()"},
	}, newGateScope("", GateScopeModeDiff, []string{target}, nil))(context.Background())
	if result.Status != "pass" || len(result.Violations) != 0 {
		t.Fatalf("expected contract step to ignore unchanged missing contract, got status=%s violations=%#v", result.Status, result.Violations)
	}
}

// TestGate_SplitReceiverQualifiedName parses every supported form of a method
// contract name: bare names, pointer- and value-receiver qualified names, and
// malformed inputs that must fall back to treating the whole string as a bare
// method name.
func TestGate_SplitReceiverQualifiedName(t *testing.T) {
	cases := []struct {
		name       string
		input      string
		wantRecv   string
		wantMethod string
	}{
		{name: "bare method name", input: "RouteFile", wantRecv: "", wantMethod: "RouteFile"},
		{name: "pointer receiver", input: "(*realCodeChecker).Check", wantRecv: "realCodeChecker", wantMethod: "Check"},
		{name: "value receiver", input: "(Widget).Run", wantRecv: "Widget", wantMethod: "Run"},
		{name: "value receiver with inner spaces trimmed", input: "( Widget ).Run", wantRecv: "Widget", wantMethod: "Run"},
		{name: "no dot after close paren falls back", input: "(*Type)method", wantRecv: "", wantMethod: "(*Type)method"},
		{name: "missing close paren falls back", input: "(*Type.method", wantRecv: "", wantMethod: "(*Type.method"},
		{name: "open paren with immediate dot yields empty recv", input: "().Run", wantRecv: "", wantMethod: "Run"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recv, method := splitReceiverQualifiedName(tc.input)
			if recv != tc.wantRecv {
				t.Errorf("recv: expected %q, got %q", tc.wantRecv, recv)
			}
			if method != tc.wantMethod {
				t.Errorf("method: expected %q, got %q", tc.wantMethod, method)
			}
		})
	}
}

// TestGate_ReceiverTypeName extracts the bare receiver type name from real
// receiver field lists parsed out of Go source, covering pointer receivers,
// value receivers, an empty receiver list, and a nil receiver.
func TestGate_ReceiverTypeName(t *testing.T) {
	src := `package fixture

type Widget struct{}

func (w *Widget) Pointer() {}
func (w Widget) Value() {}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "recv.go", src, 0)
	if err != nil {
		t.Fatalf("parsing source: %v", err)
	}

	recvByMethod := map[string]*ast.FieldList{}
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv == nil {
			continue
		}
		recvByMethod[fn.Name.Name] = fn.Recv
	}

	if got := receiverTypeName(recvByMethod["Pointer"]); got != "Widget" {
		t.Errorf("pointer receiver: expected %q, got %q", "Widget", got)
	}
	if got := receiverTypeName(recvByMethod["Value"]); got != "Widget" {
		t.Errorf("value receiver: expected %q, got %q", "Widget", got)
	}
	if got := receiverTypeName(nil); got != "" {
		t.Errorf("nil receiver: expected empty, got %q", got)
	}
	if got := receiverTypeName(&ast.FieldList{}); got != "" {
		t.Errorf("empty receiver list: expected empty, got %q", got)
	}
}

// TestGate_ContractSignature_ReceiverQualifiedDisambiguates verifies that a
// receiver-qualified method contract name selects the correct method when two
// types in the same file declare a same-named method.
func TestGate_ContractSignature_ReceiverQualifiedDisambiguates(t *testing.T) {
	target := filepath.Join(t.TempDir(), "ambiguous.go")
	if err := os.WriteFile(target, []byte(`package ambiguous

import "context"

type Alpha struct{}
type Beta struct{}

func (a *Alpha) Run(ctx context.Context) error { return nil }
func (b *Beta) Run(ctx context.Context) string { return "" }
`), 0o644); err != nil {
		t.Fatal(err)
	}

	pass := StepContractSignatureFunc([]ContractEntry{
		{File: target, Name: "(*Beta).Run", Kind: "method", Signature: "func (b *Beta) Run(ctx context.Context) string"},
	})(context.Background())
	if pass.Status != "pass" || len(pass.Violations) != 0 {
		t.Fatalf("expected receiver-qualified method to match Beta.Run, got status=%s violations=%#v", pass.Status, pass.Violations)
	}

	// Asking for Beta's name but Alpha's signature must fail — proving the
	// receiver qualifier really disambiguated rather than matching the first
	// same-named method.
	fail := StepContractSignatureFunc([]ContractEntry{
		{File: target, Name: "(*Beta).Run", Kind: "method", Signature: "func (b *Beta) Run(ctx context.Context) error"},
	})(context.Background())
	if fail.Status != "fail" {
		t.Fatalf("expected mismatched signature on Beta.Run to fail, got status=%s", fail.Status)
	}
}

func TestGate_ContractSignature_MethodVerified(t *testing.T) {
	target := filepath.Join(t.TempDir(), "method.go")
	if err := os.WriteFile(target, []byte(`package methodfixture

import "context"

type Widget struct{}

func (w *Widget) Run(ctx context.Context) error { return nil }
`), 0o644); err != nil {
		t.Fatal(err)
	}
	result := StepContractSignatureFunc([]ContractEntry{
		{File: target, Name: "Run", Kind: "method", Signature: "func (w *Widget) Run(ctx context.Context) error"},
	})(context.Background())
	if result.Status != "pass" || len(result.Violations) != 0 {
		t.Fatalf("expected method contract to pass, got status=%s violations=%#v", result.Status, result.Violations)
	}
}
