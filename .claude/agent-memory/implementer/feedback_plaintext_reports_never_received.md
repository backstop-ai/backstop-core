---
name: plaintext-reports-never-received
description: A final text message is NOT a message to a teammate — only SendMessage reaches them. Two complete phase-boundary reports were written as plain text and silently lost; the lead read "silence reads as stalled" and nearly closed the lane on its own evidence.
metadata:
  type: feedback
---

**The rule: if a teammate needs to read it, send it with SendMessage. Plain
text output is not a channel.**

Measured the hard way on PLAN-ISSUE-104 (2026-07-29). I wrote two full,
correct phase-boundary reports — red-proof, exit codes, provenance — as
ordinary final-turn text. The team lead received NONE of it. From their side
I looked idle mid-work three times in a row ("Silence reads as stalled",
"Final call... I will close the lane on my own evidence"). The lead ended up
re-running my control-vs-treatment and per-file gates themselves and
committing on their own evidence, because my report did not exist to them.

**Why:** in a team session the harness routes my text output to the user/parent
render path, not to teammate inboxes. The system prompt states this plainly
("Just writing a response in text is not visible to others on your team — you
MUST use the SendMessage tool"), and it is easy to read that as a formality
when your text output feels like it is going somewhere. It is not a formality.
The failure is SILENT in both directions: no error, no bounce, and the work
itself was fine — only the reporting evaporated.

**How to apply:**
- Every phase-boundary report, handoff, escalation, or blocked-on-you note goes
  through SendMessage to the named teammate. Do it BEFORE the final text.
- The final text message is for the human reading the transcript. Treat it as a
  DUPLICATE of what you sent, never as the delivery mechanism.
- If a teammate pings you asking for status you believe you already gave, that
  is this bug — do not just restate it in text a second time (I did, and lost
  it again). Switch channels immediately.
- Cost of the miss is not just latency: a lead with no report will verify and
  decide on their own evidence, which duplicates your work and can close a lane
  before your findings (here: the 1.96.0 not-a-version-artifact proof and the
  provenance quadruple-check) are on the record.
- SendMessage costs one call. Silence costs the whole report. Related:
  [[feedback_orchestration_sharp_edges]] (idle != done, from the other side).
