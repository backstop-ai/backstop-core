---
description: Advance a bundle's maturity (dispatches bundle-author)
---
Promote the bundle: $ARGUMENTS to the next maturity (`exploring → defined → ready`).

Promotion is a deliberate, user-initiated step — only run this when the user asks for it.

1. Dispatch the **bundle-author** agent to advance the maturity. It must satisfy the
   maturity gates (required sections + `requirements[]` + `solution.*` fields) from content
   ALREADY in the bundle — do not invent scope to clear a gate. Keep `version` compliant
   (`updated` required once version > 0.1.0).
2. Run `./bin/backstop artifact validate` and confirm the bundle is clean at the new maturity.

Report the new maturity + validation result.
