---
title: "Selfpack Family A (no-baked-tool-exec) Missing the *_test.go Exclusion Every Sibling Family Has"
schema_version: issue/v1

issue:
  id: ISSUE-096
  title: "Selfpack Family A (no-baked-tool-exec) Missing the *_test.go Exclusion Every Sibling Family Has"
  type: enhancement
  status: open
  created: "2026-07-28"

complexity:
  scope: isolated
  uncertainty: known
  risk: moderate
---

# Selfpack Family A (no-baked-tool-exec) Missing the *_test.go Exclusion Every Sibling Family Has

## Problem

`backstop-ai/backstop-self` v1.1.2 (installed at
`.backstop/packs/backstop-ai/backstop-self`, source
`/Users/bmanson/src/projects/backstop-self-pack`) ships six rule families under
`rules/no-baked.yml`. Family B2 (`no-baked-language-token`) carries a `*_test.go` path
exclusion, added in this same version; B3, B4, B5, and B6 all carry the identical
exclusion in their `paths.exclude` lists. Family A (`no-baked-tool-exec`,
`rules/no-baked.yml:6-29`) is now the **only** family without it — it excludes only
`tests/smoke/**` (lines 7-15).

The rationale that justified adding the exclusion to B2 applies identically to A. B2's
own comment (`rules/no-baked.yml:62-72`) states it: backstop-core is the module under
test, so its own harness/repo-meta tests must *name* manifest filenames and tool tokens
to assert on them — that's testing the subject, not baking routing into the shipped
binary. Family A's call-shape (a tool name passed as a string literal to
`exec.Command`/`exec.CommandContext`) is exactly the same category of self-testing
necessity: a test that builds the real backstop binary via `go build`, or drives a real
`bash`/`grep`/`semgrep` subprocess to prove an engine or hook behaves correctly, is
naming the tool under test, not shipping a baked routing decision.

## Evidence

Confirmed directly against the current tree (not inferred from the rule text):

**Already patched around, not fixed** — `cmd/backstop/integration_test.go`'s
`go build` call was rewritten during PLAN-ISSUE-020 Phase 4 (implementer-020-final,
2026-07-28) to route through the package's existing parametric `execCommand` helper
(`cmd/backstop/root_test.go:360`, `func execCommand(name string, args ...string)
*exec.Cmd { return exec.Command(name, args...) }`) specifically to stop tripping Family
A. That helper is a workaround for the rule's imprecision, not a fix to the rule.

**Dormant, unwaived, will block on next touch:**
- `cmd/backstop/pack_authoring_loop_test.go:18` — `exec.Command("go", "build", "-o",
  bin, ".")`, no waiver.
- `cmd/backstop/version_test.go:169` — `exec.Command("go", args...)` inside
  `buildBackstop`; currently carries an inline `// nosemgrep:
  backstop.packs.backstop-ai.backstop-self.rules.no-baked-tool-exec` suppression
  (added ISSUE-087 phase 2, 2026-07-27) rather than the pack-level fix. That comment is
  a semgrep-native suppression, not backstop's own `@waiver:` grammar (`pkg/waiver/`) —
  it silences the finding before backstop's waiver adjudication ever sees it, so this
  call site is currently un-auditable through the normal waiver-resolution gate step.

**Additional dormant sites found by a direct repo sweep while authoring this issue**
(`grep -rn` for `exec.Command`/`exec.CommandContext` with a string-literal first arg in
`*_test.go`, excluding `tests/smoke/**` and the allowed literals `git`/`gh`/`sh`/`/bin/sh`
/`sandbox-exec`), confirmed by reading each call site — none is inside a string
constant or comment, all are live invocations:
- `pkg/validate/spec_hook_test.go:26,45,63` — `exec.Command("bash", script)` ×3
  (`bash` is not in Family A's exception list; only `sh` is).
- `pkg/pack/engine/import_cycle_test.go:23` — `exec.Command("go", "list", "-deps",
  enginePkg)`.
- `pkg/pack/engine/contracts_grep_engine_test.go:160` — `exec.Command("grep", "-rn",
  "-e", pattern, target)`.
- `pkg/gate/self_rule_test.go:26` — `exec.CommandContext(context.Background(),
  "semgrep", "--config", rule, "--json", "--quiet", target)`.

None of these six additional sites has been touched recently enough to have tripped the
gate yet (`git log` shows their last touch predates this pack's Family A hardening), so
they are dormant in the same sense as the two named above — the next edit to any of
these six files goes blocking with no `*_test.go` exclusion to cover it, and (unlike
`version_test.go`) no ad-hoc suppression is already in place to mask it.

## Impact

Every one of these files is a legitimate self-test asserting on the real tool it names
(the Go toolchain, bash, grep, semgrep) — none is a routing/dispatch decision inside the
shipped binary. Without the exclusion, touching any of them forces one of two
non-durable workarounds already visible in the tree: rewrite the call through
`execCommand`-style indirection (fine locally, but a per-file dance every implementer
has to rediscover), or drop an inline `// nosemgrep:` comment that bypasses backstop's
own `@waiver:` adjudication path entirely (the `version_test.go` case) — a waiver that
never went through `pkg/waiver`'s audit trail. Per house rule, waivers are last resort
and the pack should learn from the scenario instead of accumulating scattered escapes
(`feedback_waivers_are_last_resort`, `feedback_dogfood_rules_as_packs`). Left as-is, this
is the same defect class B2 already fixed, just in the one family that didn't get the
fix.

## Fix (for the eventual plan — lives in the pack, not core)

Add a `*_test.go` path exclusion to Family A (`no-baked-tool-exec`,
`backstop-self-pack/rules/no-baked.yml:6-29`), mirroring B2's exclusion exactly (same
list position, same style of justifying comment). This is a **pack change** in
`backstop-ai/backstop-self` (`/Users/bmanson/src/projects/backstop-self-pack`), not a
core change:

1. Add `"*_test.go"` to Family A's `paths.exclude` alongside the existing
   `tests/smoke/**` entry.
2. Version bump (B2's equivalent change landed as 1.1.2; this should land as the next
   patch/minor, e.g. 1.1.3).
3. Re-run the pack's own fixture/validation suite (`pack test` / `pack check`) to
   confirm the exclusion doesn't blind Family A to a real violation shipped inside a
   test helper that *production* code imports — the same scoping care B2's own comment
   documents ("the binary's OWN routing is still fully covered — every non-test file
   remains in scope"). Family A's positive/negative fixtures
   (`fixtures/rules/valid/exec-variable.go`,
   `fixtures/rules/invalid/exec-baked-tool.go`) should gain a `*_test.go`-named
   counterpart pair the way B2 gained
   `fixtures/rules/valid/foreign-extension-in-test_test.go`.
4. Publish the tag, then `pack relock` in backstop-core (and any other consumer) to
   pick up the new version.
5. Once installed, the `version_test.go:169` inline `// nosemgrep:` comment and the
   `integration_test.go` `execCommand`-indirection workaround both become removable —
   confirm they still pass cleanly with the raw `exec.Command("go", ...)` form restored,
   as a falsifying check that the pack fix actually subsumes the workarounds it's meant
   to replace.

## References

- `backstop-self-pack/rules/no-baked.yml:6-29` (Family A, current), `:46-81` (Family B2,
  the precedent this issue asks Family A to match), `:83-115`, `:116-152`, `:153-200`,
  `:201-253` (B3-B6, all already carrying the `*_test.go` exclusion).
- `backstop-self-pack/pack.yml` (v1.1.2 manifest, Family A `no-baked-tool-exec` /
  Family B2 `no-baked-language-token` rule entries).
- ISSUE-087 TASK-016 — the B2 precedent: measured 16 rows, all in `*_test.go`, none a
  routing defect, on backstop-core 2026-07-27.
- Founder ratification, 2026-07-27 — "moving the exclusion into the pack is a
  manifestation of systemically adapting to new information," cited verbatim in both
  Family A's and Family B2's `tests/smoke/**` comments as the precedent for pack-level
  precision fixes over ad-hoc waivers.
- Discovered by implementer-020-final during PLAN-ISSUE-020 Phase 4 (2026-07-28) while
  fixing `cmd/backstop/integration_test.go`; ISSUE-095 and the ISSUE-020 lane are the
  discovery context, not the owning surface — this issue is standalone pack-precision
  work, not a duplicate of either.
