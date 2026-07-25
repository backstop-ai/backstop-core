---
title: "Stale Gate Binary Phantom Violations"
schema_version: issue/v1

issue:
  id: ISSUE-077
  title: "Stale Gate Binary Phantom Violations"
  type: technical-debt
  status: open
  created: "2026-07-25"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
---

# Stale Gate Binary Phantom Violations

## Problem

`backstop gate` invoked through the binary resolved from `PATH` executes STALE code, and the
resulting phantom violations get attributed to the user's diff — a false RED that looks like the
gate caught something real.

Bare `backstop` resolves to `/usr/local/bin/backstop`, which is only as fresh as the last manual
`go build -o /usr/local/bin/backstop`. Live state verified 2026-07-25 in this repo:

| Binary | Built | Freshness |
|---|---|---|
| `/usr/local/bin/backstop` (what bare `backstop` resolves to) | Jul 22 21:34 | 3 days stale |
| `bin/backstop` (Makefile target: `go build -o bin/backstop ./cmd/backstop/`) | Jul 25 16:35 | current |
| `./backstop` (repo root, an accidental extra copy) | Jul 18 | 7 days stale |

Three copies of the same binary a week apart, and only one of them correct at any given moment.
Nothing enforces that they converge, and nothing warns when they diverge.

### Failure mode (concrete, from ISSUE-062)

While implementing ISSUE-062, the implementer changed pack rule messages to drop the `func=`/
`symbol=` fields. The stale system binary's OLD message-parsing code read the NEW message format,
extracted empty func/symbol, and raised 3 phantom `noTarget` violations against a diff that had
introduced no such defect. At the same time, `go test ./cmd/backstop` compiled from source and
passed clean — because `go test` always builds from the working tree, it never touches the
installed binary.

Net effect: red gate, green test suite, and the disagreement points at a diff that is actually
correct. This is the vacuous-green failure inverted — a false RED that erodes trust in the gate
exactly as much as a false GREEN would, because the tool that exists to be the arbiter of
correctness is instead the thing lying. Documented as a "Known gap" in
`docs/CODEBASE-MAP.md` and captured in agent memory at
`.claude/agent-memory/implementer/project_pack_copies_and_stale_gate_binary.md`.

### Why it is a trap rather than a chore

An agent hitting this cannot self-heal. Overwriting `/usr/local/bin/backstop` is an out-of-repo
system path and the permission classifier correctly denies writes there. The best an agent can do
today is `go build -o <scratch>/backstop-new ./cmd/backstop`, verify clean there, and report to
the founder that the installed binary needs a manual rebuild/reinstall — which depends on a human
remembering to act on a report buried in a transcript.

CLAUDE.md already routes around the trap by specifying `./bin/backstop` explicitly, and the
gate-on-implement hook correctly invokes the relative path. But this is a convention living in a
markdown file and a hook, not something the tool enforces on itself — and it has already leaked:
plan artifacts have been authored with bare `backstop code check` as the verification command
(wrong command AND wrong binary in one string; see ISSUE-076), which is exactly the kind of drift
that happens when the correct invocation is tribal knowledge instead of load-bearing.

## Scope boundary

This is primarily a **local authoring / contributor on-ramp** problem, not a consumer-facing one.
A project that installs a released `backstop` binary has exactly one copy and never rebuilds it
from source, so binary-vs-source staleness structurally cannot arise for them — there is no
second, fresher copy to diverge from.

However, every contributor to backstop-core inherits this same dogfood loop: a new contributor's
very first `backstop gate` run after their very first change can hand them phantom violations, on
a project whose entire pitch is trustworthy verification. That contradiction alone justifies
fixing it once, in this repo, rather than deferring it as low-value.

The consumer-facing sibling of this defect class is **BUNDLE-020 (pack ↔ core version
compatibility)**, which covers the same shape of problem — a version boundary between a running
binary and the thing it parses — after the endpoints shift from binary-vs-source to
core-vs-pack. This issue and that bundle should cross-reference each other; do not fold
BUNDLE-020's scope into this issue or vice versa.

A related-but-distinct problem, also captured in
`.claude/agent-memory/implementer/project_pack_copies_and_stale_gate_binary.md`, is that a single
local pack can have 3-4 on-disk copies (tracked source, installed `.backstop/packs/`, per-test
`pkg/gate/testdata/*-pack/`) that must be kept in sync and relocked by hand. It is mentioned here
as related context only — it is a separate defect with its own fix surface (pack sync/relock
tooling) and is explicitly OUT of scope for this issue.

## Proposed resolution

Two complementary layers — the first removes the trap for the common case, the second makes any
remaining staleness loud instead of silent.

1. **Make the `PATH` entry a shim, not a binary.** Replace `/usr/local/bin/backstop` with a small
   script that locates the enclosing backstop-core checkout, rebuilds `bin/backstop` if it is
   older than the newest tracked `.go` file, then `exec`s the fresh binary with the original
   arguments. Bare `backstop` becomes correct by construction from any directory inside the repo.
   Also delete the stray Jul 18 `./backstop` copy at the repo root — it exists only to be picked
   up by accident (e.g. a shell that resolves `./backstop` before `PATH`) and serves no purpose
   the Makefile's `bin/backstop` doesn't already serve.
2. **Defense in depth — let the binary detect its own staleness.** At gate startup, compare the
   running executable's mtime (`os.Executable()` + `os.Stat`) against the newest `.go` file under
   the project root; if source is newer than the binary, exit 2 as a `*check.ConfigError` naming
   both timestamps and the rebuild command (`go build -o bin/backstop ./cmd/backstop/`). This
   covers every invocation that bypasses the shim — CI, scripts, agents that call a binary path
   directly — and converts what is currently a silent phantom-red into a loud, actionable config
   error instead of a violation the user has to debug from scratch.

The shim alone captures most of the value for the common local-dev case; the self-check is what
keeps the fix non-vacuous once any invocation path skips the shim.

## Acceptance criteria

- Bare `backstop` invoked from anywhere inside a backstop-core checkout runs code no older than
  the newest tracked `.go` file, with no manual rebuild step required by the developer.
- The stray stale `./backstop` copy at repo root is removed and not regenerated as a side effect
  of any build target.
- A `backstop` binary that IS stale relative to its own source tree (e.g. invoked directly by
  path, bypassing the shim) fails loudly with exit code 2 and a message naming both the binary's
  build time and the newest source file's mtime — never a silent phantom violation attributed to
  the user's diff.
- The ISSUE-062-style failure mode (stale binary misparsing a pack's updated message format into
  phantom violations, while `go test` passes clean) is not reproducible after the fix: forcing a
  binary older than a `.go` file change produces the exit-2 staleness error, not a gate violation.
- No behavior change for consumers running an installed release binary with no source tree
  present — the staleness check has nothing to compare against off a released install and must
  not misfire there (see Scope boundary).

## References

- `docs/CODEBASE-MAP.md` — "Known gap" heading (Pack lifecycle section) documenting the stale
  binary and the ISSUE-062 phantom-violation incident.
- `.claude/agent-memory/implementer/project_pack_copies_and_stale_gate_binary.md` — source memory
  for both this issue and the related-but-out-of-scope pack-copies problem.
- ISSUE-062 — the change (dropping `func=`/`symbol=` from pack rule messages) whose gate run
  surfaced this defect.
- ISSUE-076 (Plan Verification Commands Unresolvable) — a plan authored with bare
  `backstop code check`, compounding wrong-command with wrong-binary; evidence that the correct
  invocation needs to be enforced, not just documented.
- BUNDLE-020 (`bundles/BUNDLE-020-pack-core-version-compatibility.bundle.md`) — the consumer-facing
  sibling covering the same version-boundary defect class between core and packs.
- CLAUDE.md — already specifies `./bin/backstop` as the correct invocation; this issue makes that
  convention self-enforcing instead of a documentation-only workaround.
