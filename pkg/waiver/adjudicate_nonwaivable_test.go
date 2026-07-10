package waiver

import "testing"

// hasNonWaivableAt reports whether Result.NonWaivable carries a diagnostic at
// file+line with Kind "non-waivable".
func hasNonWaivableAt(res Result, file string, line int) bool {
	for _, d := range res.NonWaivable {
		if d.File == file && d.Line == line && d.Kind == DiagnosticNonWaivable {
			return true
		}
	}
	return false
}

// TestWaiver_NonWaivable_NormalRuleWaivable proves a waiver on a normal code
// rule (not in the non-waivable set) suppresses the finding (CLM-024).
func TestWaiver_NonWaivable_NormalRuleWaivable(t *testing.T) {
	policy := NewDeclaredPolicy([]string{"backstop/self/no-baked-language"}, []string{"critical"})
	findings := []Finding{{RuleID: "normal-rule", File: "f", Line: 5, Severity: "warning"}}
	read := lineReaderFrom("f", map[int]string{5: "x // @waiver:normal-rule:deferred:2999-01-01"})
	res := Adjudicate(findings, read, policy, fixedNow)
	if !suppressed(res, "f", 5, "normal-rule") {
		t.Fatal("a normal (undeclared) rule must remain waivable")
	}
}

// TestWaiver_NonWaivable_SelfRuleIsError proves a waiver targeting a
// backstop/self rule is a gate ERROR, not a suppression (CLM-025).
func TestWaiver_NonWaivable_SelfRuleIsError(t *testing.T) {
	policy := NewDeclaredPolicy([]string{"backstop/self/no-baked-language"}, []string{"critical"})
	findings := []Finding{{RuleID: "backstop/self/no-baked-language", File: "f", Line: 5, Severity: "error"}}
	read := lineReaderFrom("f", map[int]string{5: "x // @waiver:backstop/self/no-baked-language:accepted-risk:2999-01-01"})
	res := Adjudicate(findings, read, policy, fixedNow)
	if suppressed(res, "f", 5, "backstop/self/no-baked-language") {
		t.Fatal("a backstop/self rule must NOT be suppressible")
	}
	if !hasNonWaivableAt(res, "f", 5) {
		t.Fatalf("waiving a non-waivable rule must produce a non-waivable diagnostic; NonWaivable=%+v", res.NonWaivable)
	}
}

// TestWaiver_NonWaivable_CriticalSecretIsError proves a waiver targeting a
// critical-severity secret is a gate ERROR, not a suppression (CLM-026).
func TestWaiver_NonWaivable_CriticalSecretIsError(t *testing.T) {
	policy := NewDeclaredPolicy(nil, []string{"critical"})
	findings := []Finding{{RuleID: "secrets/aws-key", File: "f", Line: 8, Severity: "critical"}}
	read := lineReaderFrom("f", map[int]string{8: "key := \"...\" // @waiver:secrets/aws-key:accepted-risk:2999-01-01"})
	res := Adjudicate(findings, read, policy, fixedNow)
	if suppressed(res, "f", 8, "secrets/aws-key") {
		t.Fatal("a critical-severity secret must NOT be suppressible")
	}
	if !hasNonWaivableAt(res, "f", 8) {
		t.Fatal("waiving a critical-severity secret must produce a non-waivable diagnostic")
	}
}

// TestWaiver_NonWaivable_UndeclaredRuleRemainsWaivable proves a rule NOT present
// in the declared non-waivable set remains waivable (CLM-028).
func TestWaiver_NonWaivable_UndeclaredRuleRemainsWaivable(t *testing.T) {
	policy := NewDeclaredPolicy([]string{"backstop/self/no-baked-language"}, []string{"critical"})
	findings := []Finding{{RuleID: "go-standards/line-length", File: "f", Line: 3, Severity: "warning"}}
	read := lineReaderFrom("f", map[int]string{3: "x // @waiver:go-standards/line-length:deferred:2999-01-01"})
	res := Adjudicate(findings, read, policy, fixedNow)
	if !suppressed(res, "f", 3, "go-standards/line-length") {
		t.Fatal("a rule outside the declared non-waivable set must remain waivable")
	}
	if hasNonWaivableAt(res, "f", 3) {
		t.Fatal("an undeclared rule must not produce a non-waivable diagnostic")
	}
}
