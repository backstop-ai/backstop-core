---
name: measure-fp-surface-before-designing-a-detector
description: Before planning a "flag every X that doesn't match a known set" check, MEASURE the false-positive surface over the real repo — the literal reading is often unusable and the filters that fix it are themselves plan content
metadata:
  type: project
---

When an issue asks for a detector phrased as "warn on every token/config/reference
that matches nothing installed", write the throwaway measurement script FIRST and
run it over the real tree before authoring any task.

**Why:** ISSUE-097 (unbound `@waiver:` tokens) asked literally for "name waivers whose
rule-ID prefix matches no installed pack namespace". Implemented literally that flagged
27 tokens of which 6 were real — a 1-in-4.5 ratio that would have been dismissed as
noise within a week, leaving a warning nobody reads: silent green wearing a costume,
the exact failure the issue existed to close. Three filters (exclude the config-declared
artifact root; reuse the EXISTING `excludeTestdataPaths`; require the id to have an
extractable `<org>/<pack>` prefix, i.e. >=3 slash segments) took 27 -> 6, and the 6 were
exactly the real ones. The measurement also surfaced 3 tokens the issue never named.

**How to apply:** a ~20-line python/bash sweep over the working tree, run twice — once
naive, once with candidate filters — costs minutes and converts the plan's central design
decision from taste into evidence. Put the measured before/after numbers in the plan's
`notes` so the reviewer and implementer can both check them, and derive each filter from
an EXISTING declared mechanism (config key, existing helper, structural property of an ID
scheme) rather than a new baked literal — a filter that needs a hardcoded directory list
is a zero-baked violation wearing a precision costume. Then give the implementer the
measured count as a checkpoint ("this run should report SIX; if it differs, STOP and
reconcile") so a drifted tree is caught before the fix erases the evidence.

Related: [[state-a-sweep-once]] (the canonical sweep line + a FINDING test that makes the
rule operable), [[verify-issue-premises]] (the issue's own numbers are claims, not facts).
