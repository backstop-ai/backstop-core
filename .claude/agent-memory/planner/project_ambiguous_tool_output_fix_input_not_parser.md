---
name: ambiguous-tool-output-fix-input-not-parser
description: When a defect is "the parser assumed one output shape", measure whether the two shapes are DISTINGUISHABLE before planning a robust-parser fix — usually they are not, and the fix belongs at the invocation
metadata:
  type: project
---

Learned authoring PLAN-ISSUE-166 (GNU grep omits the filename for a single explicit file
target; the pack's `to-sarif.sh` awk guard `NF >= 3` silently dropped the 2-field lines,
so a security-relevant absence probe reported zero matches on Linux CI).

**An issue that offers "make the parser handle both shapes" is offering a candidate, not a
conclusion. Measure the ambiguity first.** Feed the parser a line that is legal under BOTH
shapes. Here `6:42: text` is line 6 whose match text starts `42:`, and is also
indistinguishable from file `6`, line 42 — the shipped script emits a SARIF finding at a
file named `6` that does not exist. A "robust" parser trades a silent false NEGATIVE for a
silent false POSITIVE in the same probe. That measurement, not taste, is what selects
fixing the INPUT (`grep -H`) over fixing the consumer.

**Prove the input-side flag is idempotent, not just correct.** Run the flagged and
unflagged invocation over every target shape that works TODAY (single file, multi-file,
directory) and assert byte-identity for the working ones. Without that, adding the flag is
an unmeasured behavior change across every call site. `-H` is documented identically in BSD
grep 2.6.0-FreeBSD and GNU grep — check both man pages when the defect is a platform
divergence.

**`grep` on PATH may not be the `grep` the gate runs.** An interactive shell in this
environment defines a `grep` FUNCTION wrapping `ugrep`; `exec.Command("grep", …)` resolves
`/usr/bin/grep`. Hand-measure with the absolute path or the "confirmed" reading describes a
tool nothing dispatches.

**Keep the silent-drop half even after fixing the input.** The guard was wrong about the
shape AND wrong about what to do with input it cannot parse. Converting the drop into a
LOUD refusal (nonzero exit + stderr naming the line) is what makes the convention operable —
the next pack author who writes the obvious `command: grep -rn` goes red immediately instead
of shipping the same silent false negative. This is the [[state-a-sweep-once]] pattern
applied to a shell script.

**The byte-identical-copies question resolves to BEHAVIOR, not text.** Three in-repo copies
plus one released external copy were kept identical by hope; a fourth sibling
(`ts-proof-pack`) carried the same defect and the issue never noticed it. Byte-identity
cannot cover deliberately-divergent siblings and cannot cross a repo boundary, so the
structural guard is a DISCOVERY-driven test asserting the property over every copy it finds.
Say so explicitly and defer the cross-repo mirror-sync problem as its own issue.

**`packs/<name>/` in core is NOT what the gate runs.** For contracts, the gate reads the
installed `backstop-ai/go-contracts` (extracted repo at
`/Users/bmanson/src/projects/backstop-go-contracts-pack`, lock-recorded, git source) — the
in-repo `packs/contracts` source is only `pack add`ed by tests and has already diverged on
name and version. A pack-data fix needs BOTH, and the released one needs bump + tag + push +
`pack update` (see [[planning-a-pack-data-fix]]). `reference_pack_topology` in the
project auto-memory still describes go-contracts as un-extracted — it is stale.
