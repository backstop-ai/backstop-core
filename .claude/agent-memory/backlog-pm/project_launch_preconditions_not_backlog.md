---
name: launch-preconditions-are-not-backlog-items
description: A clean backlog is NOT launch sign-off — check repo settings/secrets/tags too; and re-verify them immediately before reporting, because a readiness snapshot decays in minutes (v0.1.0 shipped 2026-07-29T00:36Z)
metadata:
  type: project
---

Two lessons from the v0.1.0 sign-off, one about scope and one about freshness.

**1. Check physical preconditions, not just the corpus.** On 2026-07-28 every
artifact said go — all four tiered blockers delivered, CI green, zero open items
arguing against v0.1.0 — while the only real risks sat outside the artifact
system entirely: `HOMEBREW_TAP_TOKEN` was unset, and the tap repo was private.
The backlog only tracks work someone filed; credentials, repo visibility, DNS,
org settings and never-executed workflows are load-bearing at launch and are
nobody's issue.

**2. A launch-readiness snapshot decays in MINUTES — re-verify before reporting.**
Both blockers I raised were resolved by the founder *while I was writing the
report*: token set 00:34:10Z, v0.1.0 tagged and released 00:36:08Z (run
`30411560553`, success, 4 platform archives + checksums, formula pushed as
`backstop.rb` to the tap ROOT — goreleaser's default when no `directory:` is set;
don't assume `Formula/`). My checks were correct when run and obsolete ~50
minutes later. Launch day is the highest-velocity window there is, and the
founder acts on findings faster than a sweep completes.

**How to apply:** for any "are we ready to ship/tag/launch" question, run the
cheap external checks alongside the corpus read — `gh secret list`,
`gh repo view <tap> --json isPrivate`, `git tag -l "v*"`, `gh release list`, and
grep the release config for unconditional `.Env`/secret references. Then **re-run
the time-sensitive ones immediately before you send**, and timestamp findings
("as of HH:MMZ") so a stale one reads as stale rather than wrong. Report them as
their own class — "physical preconditions, not backlog items" — so a clean
backlog is never read as a green light. Also verify the corpus can *answer* the
question: DIR-001 (blocker #4) was `queued` with ISSUE-087 `open` and its plan
`draft` despite full delivery, and the gate's own
`artifact_status_drift_advisory` caught it — so **run `gate --all` and read the
advisory step** rather than trusting `status:` fields.

**Residual worth remembering:** `.goreleaser.yml` still references
`{{ .Env.HOMEBREW_TAP_TOKEN }}` unconditionally with no `skip_upload`, so a
future token rotation/expiry fails the release *after* binaries publish. Offered,
not yet routed. Related: [[launch-tiering]], [[mechanism-vs-ecosystem-gap]],
[[corpus-conventions-note-supersedes]].
