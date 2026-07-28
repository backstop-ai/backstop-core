---
name: selfpack-absence-claim-tests
description: backstop/self absence claims — the pack is external+gitignored (a hard-fail test reds clean checkouts) and 5 of its 7 rule families are path-scoped away from new packages
metadata:
  type: project
---

When a spec carries a `kind: absence` claim like "backstop/self stays GREEN over
<new code>", audit the plan's test task against the pack's ACTUAL rules at
`../backstop-self-pack/rules/no-baked.yml` (installed copy:
`.backstop/packs/backstop/self`, gitignored).

**Why:** PLAN-SPEC-054's TASK-041 mandated a `cmd/backstop` test that "must FAIL
LOUDLY as capability-absent" if backstop/self is not installed. `.backstop/packs/`
is gitignored and self is sourced from the sibling `../backstop-self-pack` repo,
so that reds `go test ./...` / `make ci` on any clean checkout — and it
contradicts [[feedback_packs_always_external]] (absent pack content is NOT a
defect). The plan also said "follow the existing in-repo pattern for exercising
self-pack rules from a test"; no such pattern exists.

Empirically, in `no-baked.yml`:
- Family A (`no-baked-tool-exec`), B1 (`no-baked-tool-command`), B2
  (`no-baked-language-token`) have NO `paths:` block → they scan EVERY Go file,
  `_test.go` and `testdata/*.go` included.
- B3+ (neutral-spine, repo-layout, pack-name-keyed, rule-id-keyed, …) are
  `paths.include`-scoped to `pkg/gate/*.go`, `cmd/backstop/gate.go`,
  `cmd/backstop/pack_gate*.go`, `pkg/check/{manifest,parsers}.go`,
  `pkg/validate/plan.go`, `pkg/pack/engine/binding.go` + fixture globs, and all
  `exclude: *_test.go`. A NEW package is outside that list, so a "self-pack green
  over the new code" claim is near-vacuous for those families.
- B2's regex requires the quote-bracketed literal to be the WHOLE token:
  `"package.json"`, `"tsc"`, `"Cargo.toml"`, `".ts"` fire; `"a/b/package.json"`
  does not. So `filepath.Join(dir, "package.json")` in a TEST reds the gate —
  `backstop.yml` sets `pack_engines.sources."backstop/self"` to
  `applies-to: all-code, level: block`. Flag any plan whose FIXTURE corpus is
  named with B2 tokens (package.json / tsconfig.json / Cargo.toml / .ts).
- `.go` literals are B3-only (deliberately omitted from B2), so "a .go literal in
  a core test file is flagged" is FALSE outside the spine include list.

**How to apply:** require the absence-claim test to degrade the way the gate does
(capability_absent WARN / skip-with-signal), not hard-fail; and check the new
package is actually IN the rules' path scope before crediting the claim.
