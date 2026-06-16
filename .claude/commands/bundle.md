---
description: Start or evolve a context bundle (dispatches bundle-author)
---
Work on the bundle described by: $ARGUMENTS

1. If creating a NEW bundle, **scaffold it via the CLI first** so it starts from a compliant
   template with a valid, non-colliding id:
   `./bin/backstop artifact new bundle --slug <kebab-slug>`
   Note the created path/id. (Evolving an existing bundle? Skip this — use the existing file.)
2. Dispatch the **bundle-author** agent to author INTO the pre-created file. Tell it NOT to
   run `backstop artifact new` and not to create a new file. The agent writes the artifact —
   do not hand-edit it yourself.

Invariants to hold:
- Bundles are the `bundle → spec → plan` track. A bundle starts at `exploring` with REAL open
  questions; the **user** drives OQ resolution and promotion. Do NOT pre-resolve OQs or
  promote maturity unless the user explicitly asks.

After the agent returns: run `./bin/backstop artifact validate`, confirm the bundle is clean,
and report the outstanding open questions. Do not promote it.
