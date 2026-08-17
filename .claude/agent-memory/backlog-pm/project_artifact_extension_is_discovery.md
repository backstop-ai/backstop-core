---
name: artifact-extension-is-discovery
description: A bare `.md` extension in an artifact dir makes the artifact INVISIBLE to backstop — absent from validate --all counts and unfindable by ID — while still reading as live work to a human
metadata:
  type: project
---

The `.issue.md` / `.bundle.md` / `.spec.md` / `.adr.md` / `.directive.md` suffix is
**load-bearing for discovery**, not a naming convention. An artifact saved as bare
`<ID>-<slug>.md` is invisible to the tooling.

Verified mechanically 2026-08-17 on `ISSUE-024`:
- `./bin/backstop artifact validate --issue ISSUE-024` → `Error: no issue artifact found with ID ISSUE-024`
- absent from `validate --all`'s 414-artifact / 158-issue counts
- yet 7,694 bytes of `status: open` technical-debt sitting in `issues/`

**Why:** this is the confirmed mechanism behind ISSUE-022/023/024 sitting untouched
from 2026-06-21 to 2026-08-17. An earlier sweep could only call the extension
"plausibly" related; the explicit-ID lookup settles it. The failure mode is **silent
absence, not a red** — the artifact is missing from every count, traceability join,
and validation pass, so no gate, no sweep, and no directive roster would ever surface
it. A human browsing `issues/` sees it and reads it as live work; that asymmetry is
what makes it dangerous.

**How to apply:**
- When an artifact seems inexplicably un-triaged, orphaned, or ancient, check its
  extension before theorizing about process gaps.
- Never conclude "no directive cites it" or "the corpus doesn't know about it" from
  counts alone — counts silently exclude these files.
- Detect the whole class in one shot:
  `find issues bundles specs adrs directives -maxdepth 1 -name "*.md" -not -name "*.issue.md" -not -name "*.bundle.md" -not -name "*.spec.md" -not -name "*.adr.md" -not -name "*.directive.md"`
  Worth running on any full sweep.
- Fixing it is a `git mv` with no content change — but it is **outside the slotting
  grant** (renaming an artifact) and needs to land on `main`, so escalate rather than
  do it. Rename BEFORE closing such an issue: closing it while invisible leaves no
  validated record.

Complements [[project_phantom_filed_issues]] (INBOX mention ≠ file exists) — this is
the inverse: the file exists but the tooling cannot see it. Both mean **`ls issues/`
and the tool's own view can disagree; check each independently.**
