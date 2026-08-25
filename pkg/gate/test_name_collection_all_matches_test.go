package gate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTestVerify_CollectsAndResolvesLaterSameLineName(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "tests", "sample.fixture")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("case:First; case:Later; case:First\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	classifier := NewSourceClassifier(nil, []string{"tests/**/*.fixture"})
	matcher := mustMatcher(t, `case:([A-Za-z]+)`)
	found := collectTestFuncNamesScoped(root, nil, classifier, matcher)
	if len(found) != 2 || found["First"] != path || found["Later"] != path {
		t.Fatalf("found = %#v, want both deduplicated names at %s", found, path)
	}
	resolved := ResolveMandatedTestPaths([]MandatedTest{{FuncName: "Later"}}, root, classifier, matcher)
	if len(resolved) != 1 || resolved[0].FilePath != path {
		t.Fatalf("resolved = %#v", resolved)
	}
	inScope := newGateScope(root, GateScopeModeFile, []string{"tests/sample.fixture"}, nil)
	if scoped := collectTestFuncNamesScoped(root, inScope, classifier, matcher); len(scoped) != 2 || scoped["Later"] != path {
		t.Fatalf("in-scope names = %#v, want both names", scoped)
	}
	scope := newGateScope(root, GateScopeModeFile, []string{"tests/other.fixture"}, nil)
	if scoped := collectTestFuncNamesScoped(root, scope, classifier, matcher); len(scoped) != 0 {
		t.Fatalf("out-of-scope names = %#v", scoped)
	}
}
