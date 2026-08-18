---
name: doctored-fixture-preempted-by-validation
description: A plan test that doctors artifact/manifest DATA to reach a refusal branch is often pre-empted by upstream validation — and still goes green, pinning the wrong code path
metadata:
  type: project
---

When a plan's test doctors artifact DATA (a `pack.yml`, a spec, a lockfile) to drive a
fail-loud branch, run the doctored input through the REAL parser before believing the
branch is reachable. Upstream validation frequently rejects the doctored input first, so
the branch never executes — and the subtest STILL PASSES, because a parse error is also
"a non-nil error naming the rule id and the path."

**Why:** PLAN-ISSUE-158 (2026-08-17) prescribed "remove the `fixtures:` block" to force an
empty derivation. Measured: `validateFixtures` (`pkg/pack/manifest.go:672`, invoked from
`ParseManifestFile`'s rule loop at `:359`) hard-requires >=1 positive AND >=1 negative
fixture per claim, so `ParseManifestFile` failed before the refusal branch ran. The plan
parsed the manifest BEFORE checking emptiness, so the pin was vacuous while green. The
working recipe was to drop the whole `claims:` block — no rule-level claims-non-empty check
exists (the `len(Claims) == 0` at `:773` is for tool_configs). A wildcard-led fixture PATH,
by contrast, survives parse (paths are not shape-validated) and does reach its branch.

**How to apply:** For any "assert it refuses loudly" task, (1) run the doctored input
through the real parser/loader and confirm it parses, (2) confirm the derived value
actually hits the intended branch, and (3) demand the assertion match the branch's OWN
distinguishing message text — "non-nil error mentioning X" is satisfied by every upstream
error too. Also require the implementation task to guarantee the branch messages are
unique among errors reachable from that function. Relatedly, a plan prescribing a
hardcoded install path is usually wrong: `pack add` installs to
`.backstop/packs/<manifest name>`, and a manifest name like `backstop/substantiveness`
makes that TWO segments — derive it from `AddResult.InstalledPath`.

See [[census-through-real-parser]] for the same "run it, don't grep it" discipline applied
to counting, and [[new-guard-predicate-measure-existing-fixtures]].
