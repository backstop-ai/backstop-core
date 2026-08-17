---
name: path-include-inert-under-file-dispatch
description: semgrep paths.include/exclude containing any `/` is inert under explicit-file dispatch in EVERY spelling; semgrep's own suggested remedies do not work
metadata:
  type: project
---

A semgrep `paths.include`/`paths.exclude` pattern **containing a `/`** is unsatisfied
when semgrep is handed explicit FILE targets, in every spelling. Only a slash-free
(single-segment) pattern is honored under both directory and explicit-file dispatch.

Measured 2026-08-17, real semgrep 1.156.0, against `cmd/backstop/pack_gate.go` +
`pack_gate_provision.go` (explicit) vs `cmd/backstop` (directory):

- `cmd/backstop/pack_gate*.go` → 0 explicit / 2 directory
- `**/cmd/backstop/pack_gate*.go` → 0 / 2  ← semgrep's OWN suggested "unanchored" fix
- `/cmd/backstop/pack_gate*.go` → 0 / 2    ← semgrep's OWN suggested "anchored" fix
- `backstop/pack_gate*.go`, `*/*/pack_gate*.go` → 0 / 2
- `pack_gate*.go` (slash-free) → 2 / 2 ✅

**Why:** ISSUE-151 and DIR-032 item 20 both frame this as "directory-prefixed" globs.
That framing is too narrow — it is any `/`, and a tail-only or `**/`-prefixed pattern
is equally dark. semgrep 1.156 prints a deprecation warning recommending `**/` or `/`
anchoring; **both were measured dark**, so an implementer who trusts the tool's own
advice ships a no-op.

Two further facts that shape any fix:

- The **`exclude` side fails OPEN** — a slash-bearing exclude is ignored under
  explicit-file dispatch, so the rule fires on files the pack meant to exempt. The
  include side is the vacuous-green direction; the exclude side is a false RED.
- **Execution-based validation cannot catch it.** packval phase 3 already dispatches
  fixtures explicitly, but real packs bifurcate their include list into slash-bearing
  live-scope patterns plus slash-free "fixture hooks". Proven counterfactual on
  backstop-self's `no-structural-name-split-on-spine`: with hooks → 2 findings on its
  negative fixture; hooks removed → 0. The hook keeps phase 3 green while the live
  scope is dark. Only a **declaration-shape** check (phase 2, which `pack check` runs)
  sees it.

**How to apply:** when planning anything about pack path scoping, state the contract as
"contains a `/`" — the same rule `ciGlobScopingProblems`
(`cmd/backstop/ci_recipes_harness_test.go`) independently arrived at. Never bless the
`**/` or `/`-anchored spellings. Blast radius when last measured: 40 inert patterns /
12 rules / 2 installed external packs, zero in-repo packs — which is why
[[project_grandfathering_decides_the_fix_direction]] pushed the advisory to WARN, not
ERROR. See also [[feedback_verify_issue_premises]] and
[[project_sarif_suppression_measurement_layer]].

**Measurement trap:** zsh does not word-split unquoted expansions. Build semgrep targets
and `--config` flags as shell ARRAYS (`"${ARR[@]}"`) — a space-joined string becomes one
nonexistent-file target and yields a silent 0-finding run that mimics the defect exactly.
Also avoid `--quiet` with `--sarif`; it suppressed results here. Sanity check: a rule with
no `paths:` block at all always fires.
