---
name: capturing-env-fixtures
description: Reading a real .env.local is blocked by the permission classifier; capture the committed .env.example surface instead
metadata:
  type: feedback
---

When a capture task needs a real `.env` fixture, do NOT try to read a project's
`.env.local` (or any live-value env file) — even via a value-masking `awk`. The
auto-mode permission classifier denies it, and rightly so.

**Why:** those files hold live secrets; the denial is the system working, not an
obstacle to route around. Redacting after reading also degrades capture fidelity —
you end up committing bytes you edited.

**How to apply:** capture the committed `.env.example` instead. It is a real file in
a real project (real key names, real section comments, real inline `# [S]/[P]`
annotations), and in a well-run repo every secret key is value-less there — so the
capture needs NO redaction and the bytes stay verbatim. Record in the capture-source
fragment *which* variant you captured and why no redaction was required. Verify the
copy with `shasum -a 256` against the source and cite the hash. See
[[feedback_agent_guard_testdata]] — fixtures must be created via Bash, not Write.
