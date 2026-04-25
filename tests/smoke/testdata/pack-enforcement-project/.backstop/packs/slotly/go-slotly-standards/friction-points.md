# Friction Points — Slotly Enforcement Pack Extraction (Round 3)

## Issues Found During Extraction

### 1. Semgrep pattern expressiveness limits for slotly-001

The token-save-safety rule (`slotly-001`) wants to catch "db.Save called on
a user struct that was previously loaded and decrypted." Semgrep can match
`$DB.Save($USER)` with a regex on the metavariable name, but it cannot
track dataflow: it cannot distinguish a freshly-constructed user (safe to
Save) from one that was loaded via GetUserByID (unsafe). The current rule
uses a name-based heuristic (`.*user.*`), which will false-positive on
`Save(&workspace)` passed through a variable named `currentUser` and
false-negative on a user struct named `record`.

**Severity:** Medium. The rule catches the most common violation pattern
from the actual Slotly codebase (variable is literally named `user`), but
a production pack would need taint-mode semgrep or a layer 3 validator for
full coverage.

### 2. Semgrep cross-statement ordering for slotly-002

The Slack signature rule (`slotly-002`) needs to enforce that
`verifier.Ensure()` is called BEFORE `json.Unmarshal`. Semgrep's
`pattern-not-inside` can express "Unmarshal without Ensure above it," but
Go's control flow (early returns, helper functions, middleware chains) makes
this fragile. The current rule covers the inline-verification pattern used
in Slotly's actual handlers but would miss middleware-based verification
where Ensure() is in a different function.

**Severity:** Medium. Slotly uses inline verification, so the rule works
for the source codebase. A generalized pack would need a layer 3 validator
or a semgrep inter-procedural rule.

### 3. Obfuscation bypass fixture for slotly-004 is aspirational

The `obfuscated-secret.go` fixture (bypass_attempt) stores a base64-encoded
token. The current semgrep rule (`slotly-004`) matches literal prefixes like
`xoxb-` and `sk-`. It will NOT catch the obfuscated fixture because the
literal value in source is `eG94Yi1...` (base64), not `xoxb-...`. This
means the fixture correctly demonstrates a bypass the rule misses.

**Severity:** Known limitation, documented. The fixture is honest about
what the rule cannot catch. A real security audit would add a layer 3
validator for entropy-based secret detection.

### 4. slotly-003 error-wrapping rule is noisy

The bare-error-return rule (`slotly-003`) will fire on every `return err`
regardless of context. In Go, it is idiomatic to return unwrapped errors
from simple one-liner helpers and from the top-level main function. The
rule's `risk_class: correctness` (not `security`) and `severity: WARNING`
reflect this, but a production pack would likely need an allowlist for
specific packages or function patterns.

**Severity:** Low. WARNING-level, correctness-class. Expected to need
tuning per consumer codebase.

### 5. Layer 3 validator uses grep — inherently heuristic

The `slack-handler-middleware-check.sh` validator uses `grep` to detect
the presence of `NewSecretsVerifier` and `/slack/` route strings. This is
a string-level presence check, not an AST check. It will false-positive if
the string appears in a comment and false-negative if verification is done
via a helper function imported from another package.

**Severity:** Low for an enforcement pack (presence checks are inherently
heuristic). The `category: presence` declaration makes this explicit.

### 6. Fixture Go files reference external packages

Fixtures for slotly-002 import `github.com/slack-go/slack` and fixtures for
slotly-003 import `golang.org/x/oauth2`. When `pack validate` Phase 3
creates a temp module for semgrep --test, the fixtures need to be
syntactically valid but semgrep does not resolve imports. For tool_config
fixtures (golangci-lint), the temp module WOULD need these dependencies in
its go.mod. The `slotly-tool-errcheck` fixtures avoid this by using only
stdlib imports.

**Severity:** Medium for tool_config validation. The semgrep rule fixtures
are fine (semgrep operates on syntax, not semantics). The tool_config
fixture for errcheck uses only stdlib and will work in a temp module. But
if we added a tool_config rule that needs third-party imports, the temp
module setup would need a `go get` step.

## Authoring Experience Notes

- The `bypass_attempt: true` requirement for security-class negative fixtures
  is a good forcing function. It made me think carefully about what each rule
  CANNOT catch, which is exactly the kind of honesty that makes packs
  trustworthy.

- The `category:` field being layer-3-only and `risk_class:` being universal
  was clear after reading BUNDLE-004. Previous round feedback resolved the
  confusion.

- Fixture lists (not single paths) are natural for negative fixtures where
  you want to show multiple violation modes. For positive fixtures, a single
  entry per claim is usually sufficient.
