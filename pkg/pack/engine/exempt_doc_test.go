package engine

import (
	"os"
	"strings"
	"testing"
)

// TestExemptDoc_BindingCommentNamesNoEngine forbids a per-engine roster from ever
// being re-introduced into ExemptFromScopeFilter's doc comment.
//
// An enumeration of which engines declare which value is exactly what went stale:
// the comment claimed go-test declares the flag "false/unset" long after ISSUE-129
// made it true. De-enumerating without a guard would leave the failure mode exactly
// as it was — the next person to "helpfully" document the roster reopens it. This
// test makes the de-enumeration structural rather than a good intention.
func TestExemptDoc_BindingCommentNamesNoEngine(t *testing.T) {
	src, err := os.ReadFile("binding.go")
	if err != nil {
		t.Fatalf("read binding.go: %v", err)
	}
	lines := strings.Split(string(src), "\n")

	// Anchor on the yaml tag, which occurs EXACTLY ONCE in the file. Anchoring on
	// the bare field name would hit the comment's own opening word (Go doc
	// convention puts the field name first), walk backward from there, and collect
	// the PRECEDING field's doc comment instead — a block that names no engine, so
	// the negative assertion below would pass vacuously against the wrong target.
	const tag = `yaml:"exempt_from_scope_filter"`
	decl := -1
	for i, line := range lines {
		if strings.Contains(line, tag) {
			if decl >= 0 {
				t.Fatalf("anchor %s is no longer unique in binding.go (lines %d and %d)", tag, decl+1, i+1)
			}
			decl = i
		}
	}
	if decl < 0 {
		t.Fatalf("could not locate the ExemptFromScopeFilter declaration via %s", tag)
	}
	if strings.HasPrefix(strings.TrimSpace(lines[decl]), "//") {
		t.Fatalf("anchor landed on a comment line, not the declaration: %q", lines[decl])
	}

	// The target block is the contiguous run of //-prefixed lines immediately above
	// the declaration, terminated upward by the first non-comment line.
	start := decl
	for start > 0 && strings.HasPrefix(strings.TrimSpace(lines[start-1]), "//") {
		start--
	}
	block := strings.Join(lines[start:decl], "\n")

	if strings.TrimSpace(block) == "" {
		t.Fatalf("the ExemptFromScopeFilter doc comment is empty; de-enumerating must not mean deleting the semantics")
	}

	// Scope is EXACTLY this one block, and that is load-bearing rather than tidy:
	// binding.go legitimately names tools in OTHER doc comments (StrictSarif cites
	// the golangci-lint command-prefix sniff it replaced, PackageScoped cites the
	// `go test` sniff). A whole-file scan would be permanently red for reasons this
	// test has no business touching, and the obvious "fix" would gut those accurate,
	// deliberate comments.
	for _, name := range []string{
		"go-build", "go-test", "golangci", "go-coverage", "go-arch-lint",
		"semgrep", "ast-grep", "sandbox", "config-file",
	} {
		if strings.Contains(block, name) {
			t.Errorf("the ExemptFromScopeFilter doc comment names the engine %q. A per-engine roster in this "+
				"comment is what went stale (it claimed go-test declares false/unset after ISSUE-129 made it "+
				"true) and it will go stale again the moment a pack changes in a repository this comment cannot "+
				"see. State the SEMANTICS and point at the executable roster "+
				"(TestExemptAudit_EveryCommittedPackEngineHasAnIntentRow) instead.\nblock:\n%s", name, block)
		}
	}

	// Assert positively too, so the test cannot be satisfied by gutting the comment:
	// the decoupling from ScopeKind is the semantics it is required to state.
	if !strings.Contains(block, "ScopeKind") {
		t.Errorf("the ExemptFromScopeFilter doc comment no longer states its decoupling from ScopeKind:\n%s", block)
	}
}
