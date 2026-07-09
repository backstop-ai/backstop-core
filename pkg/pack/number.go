package pack

import (
	"fmt"
	"regexp"
)

var languagePattern = regexp.MustCompile(`^[a-z]+$`) // nosemgrep: go.core.no-global-mutable-state — compile-once immutable regexp, never reassigned

// ValidateLanguage checks that the language string matches ^[a-z]+$.
func ValidateLanguage(language string) error {
	if language == "" {
		return fmt.Errorf("language is required")
	}
	if !languagePattern.MatchString(language) {
		return fmt.Errorf("language must match ^[a-z]+$, got %q", language)
	}
	return nil
}

// ResolvePackNumber is DELETED (ISSUE-032 Defect A / ISSUE-030 fold). It scanned
// `standards/<language>/` for `STD-<LANG>-NNN` files to auto-number a native-standards
// artifact — a concept the packs-only decision retired. Engine packs are named by slug
// (local/<slug>), not numbered. See the lineage tombstone in scaffold.go and the
// deletion assertion in deletion_assertion_test.go.
