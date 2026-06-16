---
description: Author a spec from a bundle seed, then review it
---
Author a spec from: $ARGUMENTS

1. Confirm the **source bundle** — specs are NEVER standalone; they derive from a bundle.
   If no source bundle is identified, stop and ask.
2. **Scaffold the spec file(s) via the CLI FIRST** so each starts from a compliant template
   with a valid, non-colliding id:
   `./bin/backstop artifact new spec --slug <kebab-slug>`
   Authoring several specs at once (e.g. one per bundle spec-seed)? **Reserve ALL ids serially
   first**, before dispatching any agent — `artifact new` auto-assigns the next id, so parallel
   agents would race and collide. Note each created path/id.
3. Dispatch the **spec-author** agent (one per spec; parallel is fine) to author INTO its
   pre-created file. Tell it NOT to run `backstop artifact new` and not to create a new file.
   The agent writes the artifact — do not hand-edit.
4. Run `./bin/backstop artifact validate`; if a spec has violations, hand them back to its
   spec-author to fix until clean.
5. Dispatch the **spec-reviewer** agent (one per spec) to review against the source bundle
   (coverage gaps, ambiguities, missing claims). Report findings.

Report the validated spec(s) + reviewer findings. Do not start planning until the user says go.
