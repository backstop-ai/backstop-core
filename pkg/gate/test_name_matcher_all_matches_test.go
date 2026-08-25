package gate

import (
	"reflect"
	"testing"
)

func mustMatcher(t *testing.T, patterns ...string) TestNameMatcher {
	t.Helper()
	m, err := NewTestNameMatcher(patterns)
	if err != nil {
		t.Fatalf("NewTestNameMatcher: %v", err)
	}
	return m
}

func TestTestNameMatcher_FindNamesReturnsAllMatchesOnLine(t *testing.T) {
	m := mustMatcher(t, `item:([A-Za-z]+)`)
	if got, want := m.FindNames("item:First; item:Second"), []string{"First", "Second"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("FindNames = %v, want %v", got, want)
	}
}

func TestTestNameMatcher_FindNamesDeterministicAcrossPatternOverlap(t *testing.T) {
	m := mustMatcher(t, `prefix:([A-Za-z]+)`, `(prefix):[A-Za-z]+`, `prefix:([A-Za-z]+)`, `later:([A-Za-z]+)`)
	if got, want := m.FindNames("prefix:First later:Second"), []string{"First", "prefix", "Second"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("FindNames = %v, want %v", got, want)
	}
}

func TestTestNameMatcher_FindNamesRejectsEmptyOrMissingCapture(t *testing.T) {
	m := mustMatcher(t, `plain`, `optional:([A-Za-z]*)`, `name:([A-Za-z]+)`)
	if got, want := m.FindNames("plain optional: name:Kept"), []string{"Kept"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("FindNames = %v, want %v", got, want)
	}
}

func TestTestNameMatcher_FindNameCompatibilityUsesFirstEnumeratedName(t *testing.T) {
	m := mustMatcher(t, `item:([A-Za-z]+)`)
	if name, ok := m.FindName("item:First item:Second"); !ok || name != "First" {
		t.Fatalf("FindName = %q,%v", name, ok)
	}
	if name, ok := m.FindName("none"); ok || name != "" {
		t.Fatalf("no-match FindName = %q,%v", name, ok)
	}
}

func TestTestNameMatcher_GoAndTSSingleMatchBehaviorUnchanged(t *testing.T) {
	goMatcher := mustMatcher(t, `^\s*func\s+(Test\w+)\s*\(`)
	tsMatcher := mustMatcher(t,
		`test\(['"]([^'"]+)['"]`,
		`describe\(['"]([^'"]+)['"]`,
		`it\(['"]([^'"]+)['"]`,
	)
	if got := goMatcher.FindNames("func TestOne(t *testing.T) {"); !reflect.DeepEqual(got, []string{"TestOne"}) {
		t.Fatalf("Go names = %v", got)
	}
	for line, want := range map[string]string{
		"test('one', () => {":           "one",
		"describe('suite', () => {":     "suite",
		"it('does work', async () => {": "does work",
	} {
		if got := tsMatcher.FindNames(line); !reflect.DeepEqual(got, []string{want}) {
			t.Fatalf("TypeScript names for %q = %v, want [%s]", line, got, want)
		}
	}
}

func TestTestNameMatcher_AllMatchEnumerationPreservesLoudValidation(t *testing.T) {
	if _, err := NewTestNameMatcher([]string{"(["}); err == nil {
		t.Fatal("invalid regex was accepted")
	}
	m, err := NewTestNameMatcher(nil)
	if err != nil || m.HasPatterns() {
		t.Fatalf("empty matcher = %#v, %v", m, err)
	}
}
