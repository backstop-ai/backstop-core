---
name: satisfy-nobaked-by-reading-the-manifest
description: When self-pack `no-baked-tool-exec` fires on a NEW test, do not hide the literal behind a helper — read the tool, base args and input flag out of the pack manifest the test already parses
metadata:
  type: feedback
---

A test that drives a pack's real engine will trip
`backstop-ai/backstop-self/…/no-baked-tool-exec` the moment it writes
`exec.Command("ast-grep", …)` — the rule matches a literal FIRST ARG at the
`exec.Command`/`exec.CommandContext` call site (allowlist: git, gh, sh,
/bin/sh, sandbox-exec) and, unlike families B1/B2, carries NO `*_test.go`
exclusion.

The cheap dodge is to pass the literal into a helper
(`runEngineStdout(t, "ast-grep", …)`) so the call site sees a variable — the
existing in-package convention, and it does silence the rule.

**Do the better thing when the test already parsed the manifest:** take the
invocation from the pack's own `engines:` block —

    spec := manifest.Engines["ast-grep-contracts"]   // Command, InputFlag
    fields := strings.Fields(spec.Command)           // tool, base args
    exec.Command(fields[0], append(baseArgs, spec.InputFlag, pattern, file)...)

**Why:** it satisfies the rule in SPIRIT (the tool name is pack-declared data,
which is the whole thin-executor thesis) and it strengthens the test — it now
exercises the declared command and input flag, so a pack that renames its tool
or drops `input_flag` fails here instead of silently diverging from what the
test hard-codes. Measured 2026-08-18 on PLAN-ISSUE-157: same match counts,
rule clean, no waiver.

**How to apply:** whenever a new test shells a language tool, ask whether the
manifest is already in hand. If yes, read the invocation from it. If not, the
helper-with-variable form is the fallback, never a waiver. Also drop
convenience flags the pack does not declare (`--lang go` was redundant —
ast-grep infers the language from the `.go` extension, which is exactly what
the pack's own dispatch relies on).
