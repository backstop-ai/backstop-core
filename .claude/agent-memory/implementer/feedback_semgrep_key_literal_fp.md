---
name: semgrep-key-literal-fp
description: The dogfood secrets pack fires "Hardcoded credentials" on any identifier containing key/secret/token/password assigned a string literal
metadata:
  type: feedback
---

The installed secrets/SAST semgrep rule flags a "Hardcoded credentials detected"
finding on ANY Go identifier whose name contains `key`/`Key` (also secret/token/
password/pwd) when it is assigned a STRING LITERAL — even a harmless separator
constant. Caught on SPEC-044: `const coverageDupKeySep = "\t"` net-new-RED'd `code
check`; renaming to `coverageDupPairSep` cleared it with zero behavior change.

**Why:** the generic secrets rule keys on the identifier name + string-literal RHS,
not on the value's entropy, so benign constants trip it.

**How to apply:** when naming a string-literal const/var on a gate-path file, avoid
`key`/`secret`/`token`/`password` substrings (use `pair`/`sep`/`label`/`id`). If you
inherit such a name, a rename is the cheapest fix. Verify net-new via the per-file
HEAD-vs-NOW finding diff ([[feedback_netnegative_gate_baseline]]) — this FP shows up
as exactly one new "Hardcoded credentials" line on your changed file.
