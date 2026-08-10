---
name: zsh-pipestatus-is-one-indexed
description: In zsh `${PIPESTATUS[0]}` is ALWAYS EMPTY (zsh arrays are 1-indexed, and the array is $pipestatus) — a reported "exit code" from that idiom is no evidence at all
metadata:
  type: feedback
---

The shell here is zsh. `cmd | tail; echo "EXIT=${PIPESTATUS[0]}"` prints `EXIT=` with
NOTHING after it — zsh spells the array `$pipestatus` and indexes from 1, so the bash
idiom silently yields empty rather than erroring. The safe forms: `$pipestatus[1]` in
zsh, or simply do not pipe — redirect to a file, capture `$?` on its own line, then
grep the file.

**Why:** the reporting protocol demands TRUE exit codes with no pipe-masking, and this
idiom looks exactly like compliance while carrying no information. I emitted two
`EXIT=` lines with empty values before noticing, and had to re-run everything unpiped.
A reader skimming the report would have read a blank as a pass.

**How to apply:** never report an exit code obtained through `${PIPESTATUS[0]}`. When a
command's output needs filtering AND its status matters, write
`cmd > /tmp/out.txt 2>&1; echo "EXIT=$?"` then inspect /tmp/out.txt — this also keeps
the full output available when the summary line turns out to be the wrong slice.
Related: [[feedback_verify_with_gate_not_llm]].
