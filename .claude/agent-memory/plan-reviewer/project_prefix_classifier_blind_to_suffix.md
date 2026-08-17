---
name: prefix-classifier-blind-to-suffix
description: A plan that re-keys a compound id and cites its own new prefix-keyed check as the safety net is almost always wrong — the check reads only the prefix
metadata:
  type: project
---

When a plan (a) introduces a classifier that keys on the FIRST N segments of a
compound identifier and (b) also performs a rename of that identifier, verify the
claim "my new check catches a partial rename." It usually cannot.

**Why:** PLAN-ISSUE-097 built `Unbound(tokens, namespaces)` extracting
`segments[0]+"/"+segments[1]` from a waiver rule-id and comparing to `backstop.lock`
entry names. TASK-011 then re-keyed five tokens whose ids carry the org/pack name
TWICE — once as the coordinate prefix, once inside the dotted semgrep rule id
(`backstop-ai/backstop-self/backstop.packs.backstop-ai.backstop-self.rules.<id>`).
The plan wrote: "Changing only the first half ... would leave the token just as
unbound; TASK-010 will catch it." False both ways — a half-migrated token's first
two segments ARE the locked pack name, so `Unbound` emits nothing and the repo
invariant test goes GREEN over a token that still suppresses nothing. The repo
already contained that exact half-migrated shape at
`cmd/backstop/bun_ratchet_flip_test.go:128`. Compounded by a sharp edge saying the
rule is dormant, so no live finding can catch it either.

**How to apply:** For any rename task, write out the post-rename string and run the
plan's OWN specified classifier algorithm against the *most likely wrong* variant by
hand. If the classifier goes green on it, the plan's safety-net claim is a blocker.
Also: grep the repo for the half-migrated shape — the mistake is usually already
present somewhere, which both proves the risk and gives you the citation.

Related: [[project_repurposed_test_claim_text_drift]],
[[project_verified_enumeration_do_not_rederive]].
