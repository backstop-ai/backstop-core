---
name: clean-pack-addition-no-artifact
description: A clean pack addition with no defect and no ongoing work gets NO artifact — don't escalate it for a directive home; backstop.yml + backstop.lock + the commit + CI are the record
metadata:
  type: project
---

**A clean pack addition gets no artifact.** No directive, no issue, no
bundle. Founder ruling (Brandon, 2026-08-02), after I escalated
`backstop-ai/backstop-core-architecture` (adopted in one commit, `7211a59`)
as needing a home, then folded it into DIR-027 on a first ruling — which was
then **reversed and the fold-in removed**.

**Why:** DIR-027 is *Publication & Migration*; every thread there is an
in-progress migration (de-vendoring, renaming, local→remote). This pack never
needed migrating — it went in clean and already final, so there was no thread
to join. Worse, an inventory line would have been a fresh instance of the
exact staleness DIR-027's own "six packs, all `source_type: local`" error had
just demonstrated: **hand-maintained prose paraphrasing `backstop.lock`,
guaranteed to drift.** `backstop.yml` + `backstop.lock` + the adopting commit
+ CI proof are already a complete, self-verifying record. Artifacts are for
defects and ongoing work, not for bookkeeping a file that keeps itself.

**How to apply:** when a sweep finds a pack consumed with "zero citation
anywhere," that is **not automatically a gap**. Ask first: is there a defect,
or ongoing work attached? If no to both, it needs nothing — do not escalate a
home question. Escalate only if the pack arrived broken, or carries migration
/ rename / publication work.

**The generalization, which is the reusable part:** before proposing that
prose record a fact, ask whether a machine-maintained file already records it.
If it does, prose restating it is a drift liability, not documentation. This
is the same instinct behind [[project_phantom_filed_issues]] — the founder
prefers a gap that explains itself over a record that must be maintained.

**Live tension worth watching:** DIR-027 still contains a hand-written
seven-entry enumeration of `backstop.lock` (from the founder-approved
pack-count correction earlier the same day). By this ruling's own logic that
enumeration is itself drift-prone. It was approved before the ruling existed;
I have flagged it and not touched it. If it goes stale, this is why.

See [[feedback_packs_always_external]], [[project_corpus_note_supersedes]].
