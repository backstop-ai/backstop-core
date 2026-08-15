---
name: write-path-and-read-path-must-share-one-root
description: "Location tests stay green while NUMBERING silently breaks: `artifact new` resolved the root for the WRITE but handed the ID scan a project root, so a configured .backstop root restarted ids at 001. Any read+write pair must take the resolved value, typed so the wrong one is unrepresentable."
metadata:
  type: project
---

The id resolver's local-scan fallback counts existing artifacts to pick the next number. It took an
`IDOptions.ProjectRoot` STRING and built an `artifact.Root` literal inline, bypassing `ResolveRoot`,
while the WRITE path used the properly resolved root. Under a configured `.backstop` artifact root
the scan read a nonexistent project-root type directory, found nothing, and returned the first
number — colliding with an artifact that already existed.

**Why every location test stayed green:** the file's LOCATION was always correct. Only the NUMBER
was wrong. Assertions about "where did it land" cannot see a read/write divergence. The guard has to
plant an EXISTING artifact under the configured root and assert the NEW one continues past it.

**How to apply:**
- When one command both READS a corpus to derive something and WRITES into it, both must consume the
  same resolved value, and the resolution must happen BEFORE the first reader (here: move the root
  resolution above the id-resolution call).
- Prefer REPLACING the loose field over adding the resolved one beside it. The string field had
  exactly one use — the buggy line — so swapping it for a typed `artifact.Root` removed the second
  source of truth entirely instead of leaving both and inviting the same divergence again.
- A hand-constructed Root literal also silently violates the "Path is always absolute" guarantee the
  resolver establishes. Taking the struct as a parameter makes handing in a project root
  unrepresentable rather than merely discouraged.
- Falsify the regression test by reinstating the inline construction and confirming the restart
  appears; a numbering test that never observed the restart is not guarding anything.

See also [[a-resolver-cannot-count-its-own-alternatives]].
