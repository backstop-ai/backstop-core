package config_test

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/config"
)

// TestConfig_LanguageFieldRemoved (CLM-011): config.Config has NO Language field —
// a struct guard asserts the field is gone so any cfg.Language reader fails to
// compile (the compile-time regression guard). The single-language field is wrong
// for a polyglot repo and is fully retired (SPEC-046 REQ-003).
func TestConfig_LanguageFieldRemoved(t *testing.T) {
	if _, ok := reflect.TypeOf(config.Config{}).FieldByName("Language"); ok {
		t.Error("config.Config must have NO Language field — the single-language field is retired (CLM-011)")
	}
}

// TestConfig_LanguageKeyIgnoredCleanlyFieldRemoved (CLM-012, MANDATED): a backstop.yml
// that still carries a `language:` key parses cleanly with NO error and the key is
// ignored — the field is fully GONE, not rejected. Proves the retirement is inert,
// not a strict-mode rejection that would break older configs in the wild.
func TestConfig_LanguageKeyIgnoredCleanlyFieldRemoved(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "backstop.yml")
	if err := os.WriteFile(path, []byte("project: legacy\nlanguage: go\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.LoadConfigFromPath(path)
	if err != nil {
		t.Fatalf("a backstop.yml carrying a stray `language:` key must parse cleanly (the field is gone, not rejected), got: %v", err)
	}
	if cfg == nil || cfg.Project != "legacy" {
		t.Fatalf("the config must load with the language key ignored, got %+v", cfg)
	}
}

// TestConfig_NoTestAssertsConfigLanguage (CLM-025): the COMPLETENESS CONTRACT for the
// language-reader sweep — NO _test.go in cmd/backstop, pkg/config, pkg/gate, OR
// pkg/check reads config.Config.Language. The look-alikes check.Options.Language and
// pack.Manifest.Language are SEPARATE fenced fields (receivers like opts/o/m/manifest)
// and are explicitly NOT flagged; the guard trips ONLY on config.Config.Language.
func TestConfig_NoTestAssertsConfigLanguage(t *testing.T) {
	// config.Config.Language reads: a `*Cfg`-named receiver's `.Language`, or an
	// explicit `config.Config.Language`. (Options/Manifest receivers — opts, o, m,
	// manifest, bind — do not end in "cfg"/"Cfg", so the look-alikes are fenced.)
	readRe := regexp.MustCompile(`\b[A-Za-z0-9_]*[Cc]fg\.Language\b|\bconfig\.Config\.Language\b`)
	// A config.Config composite literal that sets the Language field.
	litRe := regexp.MustCompile(`config\.Config\{[^{}]*Language\s*:`)

	pkgs := []string{
		filepath.Join("..", "..", "cmd", "backstop"),
		filepath.Join("..", "..", "pkg", "config"),
		filepath.Join("..", "..", "pkg", "gate"),
		filepath.Join("..", "..", "pkg", "check"),
	}
	for _, dir := range pkgs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("reading %s: %v", dir, err)
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), "_test.go") {
				continue
			}
			// Skip this guard's own file: it names config.Config.Language patterns in
			// its regexes/comments to assert their absence (not a live reader).
			if e.Name() == "config_language_removal_test.go" {
				continue
			}
			b, err := os.ReadFile(filepath.Join(dir, e.Name()))
			if err != nil {
				t.Fatalf("reading %s: %v", e.Name(), err)
			}
			stripped := stripLineComments(string(b))
			if readRe.MatchString(stripped) || litRe.MatchString(stripped) {
				t.Errorf("%s/%s reads config.Config.Language — the language-keyed tests must be updated, not preserved; the look-alikes check.Options.Language / pack.Manifest.Language survive and are fenced (CLM-025)", dir, e.Name())
			}
		}
	}
}

// stripLineComments removes `//` line comments so the source guard scans only code,
// not prose that legitimately discusses the retired field.
func stripLineComments(src string) string {
	lines := strings.Split(src, "\n")
	for i, ln := range lines {
		if idx := strings.Index(ln, "//"); idx >= 0 {
			lines[i] = ln[:idx]
		}
	}
	return strings.Join(lines, "\n")
}
