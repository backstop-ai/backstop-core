---
description: Start or evolve a context bundle (dispatches bundle-author)
---
Dispatch the **bundle-author** agent to work on the bundle described by: $ARGUMENTS

Invariants to hold:
- Bundles are the `bundle → spec → plan` track. A bundle starts at `exploring` with REAL
  open questions; the **user** drives OQ resolution and promotion. Do NOT pre-resolve OQs
  or promote maturity unless the user explicitly asks.
- The agent writes the artifact — do not hand-edit it yourself.

After the agent returns: run `./bin/backstop artifact validate`, confirm the bundle is clean,
and report the open questions still outstanding. Do not promote it.
