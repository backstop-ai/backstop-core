---
description: Author a spec from a bundle seed, then review it
---
Author a spec from: $ARGUMENTS

1. Confirm the **source bundle** — specs are NEVER standalone; they derive from a bundle.
   If no source bundle is identified, stop and ask.
2. Dispatch the **spec-author** agent to author/evolve the spec. The agent writes the
   artifact — do not hand-edit.
3. Run `./bin/backstop artifact validate`; if the spec has violations, hand them back to
   the spec-author to fix until clean.
4. Dispatch the **spec-reviewer** agent to review the spec against its source bundle
   (coverage gaps, ambiguities, missing claims). Report its findings.

Report the validated spec + reviewer findings. Do not start planning until the user says go.
