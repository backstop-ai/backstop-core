# Concepts

You've run `backstop init` and `backstop gate` and seen something happen. This document
explains why backstop is shaped the way it is, and defines the handful of words you'll keep
running into: **pack**, **engine**, **gate**, **dimension**, **baseline**, **waiver**,
**artifact**.

If you haven't run it yet, start with [getting-started.md](getting-started.md) and come back.

## The premise

Backstop is not a code quality scanner. It is a framework that assumes AI agents will
fabricate, skip steps, and ignore conventions unless mechanically constrained.

That assumption changes what the tool has to do. A linter's job is to find bugs in code that
a person wrote and mostly meant. Backstop's job is to make the *process* verifiable — to
answer questions like:

- Does the test that was promised actually exist, and did it pass?
- Does the code that claims to implement a requirement trace back to one?
- Is this "done" backed by evidence, or by an assertion?
- Is the green you're looking at real, or is it green because nothing ran?

The last one is the big one. **A false green is worse than a loud error.** A misleading error
at least stops you; a silent pass over a check that never executed lets bad work propagate and
accrue more work on top of it. Backstop is built to refuse rather than to guess — when it
can't tell whether a dimension is satisfied, it says so instead of passing.

## Packs: backstop ships zero checks

Here is the thing most people find surprising: **backstop core contains no checks at all.**
Not one lint rule, not one language-specific heuristic, not one baked-in tool invocation.

The checks live in **packs**. A pack is a versioned repository of rules, engine bindings,
fixtures, and (optionally) scaffolding recipes. It lives in its own repo, entirely outside
backstop core, and installs into your project like a dependency:

```bash
backstop pack add backstop-ai/go-standards@1.2.1
```

That command clones the pack at the tag, validates it, and installs it into
`.backstop/packs/`, which is **gitignored** — the same way `node_modules/` is. What you commit
is the declaration and the pin:

- **`backstop.yml`** — which packs you've adopted, at which versions, plus your enforcement
  policy.
- **`backstop.lock`** — the durable record: version, git ref, source coordinate, and a content
  hash of the installed tree.

Because the lock file is the durability boundary, an absent `.backstop/packs/` is not a
problem. `backstop pack install` reconstitutes it, and the content hash proves you got the
same bytes. This is why CI can be a clean checkout plus one install step.

A pack declares its checks in `pack.yml`. Rules name a rule file, a risk class, an engine, and
— importantly — **claims with fixtures**: positive examples the rule must accept and negative
examples it must reject. `backstop pack test` executes those fixtures, so a pack proves its
own rules actually fire before you trust them. A rule that catches nothing is a defect the
pack's own test suite finds.

## Thin executor: why it's not a linter

Backstop bakes in **zero** language and tool knowledge. It doesn't know what Go is, or
TypeScript, or Rust. It doesn't know that `go test` exists or what `eslint` does. It doesn't
detect your project's language — deliberately, because there is no "primary language" to
detect and guessing one is how a tool ends up with a favorite.

What backstop knows is a small, universal vocabulary:

- How to run a declared command from a **trusted-tool allowlist** — currently `semgrep` and
  `ast-grep` (the findings engines backstop introduces and pins itself) plus `grep` (a
  Layer-0 tool trusted at presence rather than a pinned version) — and how to provision the
  ones it introduced at a pinned version.
- How to read **SARIF**, the industry-standard findings format, as its universal wire format.
- How to scope, adjudicate, ratchet, and gate on the findings that come back.

Everything else — which files matter, what a violation is, which tool to run, what "coverage"
means for your stack — comes from a pack.

This is the actual differentiator, and it has three consequences worth internalizing:

**Any language works, and none is privileged.** A Go pack and a TypeScript pack are peers.
Neither is "native." Multi-language just means multiple packs. There is no plugin tier that
gets second-class treatment, because there is no first-class tier to be second to.

**You extend backstop without waiting on backstop.** If you need a rule that doesn't exist,
you write it in a pack — your own private pack in your own repo — and `pack add` it. No PR to
this project, no release cycle, no upstream negotiation. The extension point is the primary
mechanism, not an escape hatch bolted onto one.

**A baked-in check would be a bug, not a feature.** This project enforces the rule on itself:
one of the packs backstop-core installs is `backstop-ai/backstop-self`, whose job is to fail
the build if a language or tool name shows up in core's dispatch path. The framework gates
itself with the same machinery it gates you with.

## The gate

`backstop gate` is the verification command. Running it does roughly this:

1. **Resolve config and scope.** Bare `backstop gate` is *diff-scoped*: it looks at what
   changed against the merge base, plus untracked files. `--all` sweeps the whole project;
   `--file` takes explicit paths. Diff scope is the default because it's the loop you actually
   run dozens of times a day.
2. **Verify the lock.** Installed packs are hashed against `backstop.lock`. Tampered or
   drifted pack content is caught before any of its rules get a vote.
3. **Run the pack-declared engines.** Each engine binding in a `pack.yml` names a tool, an
   input mode (rule files, a config file, a pattern argument), and a **gate type**. Backstop
   builds the command, runs it, and parses SARIF back out.
4. **Run the backstop-native dimensions** over those findings and over your artifacts.
5. **Adjudicate waivers**, then **compare against the baseline**.
6. **Apply policy** and produce a verdict.

Exit codes: `0` pass, `1` violations, `2` configuration problem. That third one matters — a
broken setup is never reported as a pass.

### Dimensions

A **dimension** is one axis of verification, named in backstop's vocabulary rather than any
tool's. The ordered set:

| Dimension | What it answers |
|---|---|
| `pack_lock_verification` | Do installed packs match the lock? |
| `artifact_validation` | Do your artifacts conform to their schemas? |
| `pack_engines` | What did the pack-declared tools find? |
| `test_verification` | Do the promised tests exist — and did they pass? |
| `test_substantiveness` | Do those tests actually exercise the thing they claim to? |
| `coverage_threshold` | Is coverage at or above the floor? |
| `contract_signature` | Do declared contracts match the real signatures? |
| `artifact_status_drift` | Does artifact status match reality? |
| `requirement_traceability` | Does work trace back to a requirement? |
| `waiver_resolution` | Are waivers well-formed, unexpired, and legitimate? |
| `baseline_comparison` | Is this net-new, or pre-existing? |
| `ledger_integrity` | Is artifact numbering intact? |

The dimension vocabulary is what makes policy portable: you tune `coverage_threshold`, not
"the go-toolchain pack's coverage step." Swap the pack and the policy still means the same
thing.

Two of these are worth calling out because they're the ones a linter has no analogue for.
`test_verification` doesn't just check that a test *name* is present — it joins the mandated
test names against the actual test verdicts, so a test that exists and fails is a *critical*
violation, not a silent pass. And when no installed pack declares a test-running engine at
all, you get a loud "capability absent" advisory rather than a green checkmark over nothing.
`test_substantiveness` goes one step further: a test that calls nothing and asserts nothing
satisfies its name but not its purpose, and that's a distinct finding.

### The pack severity contract

Packs decide what blocks. A finding's SARIF `level` is the wire contract:

- **`warning`** — non-blocking. Surfaced loudly, never fails the gate.
- **`error`, or no level at all** — blocking. Fail-closed: an unlabeled finding is treated as
  serious.

This is "loud ≠ blocking" expressed on the wire. It's how a pack informs you about an
un-adopted capability without gating your build, while still failing hard on a real defect. It
is a supported thing for a pack author to *say*, not an accident of parsing.

### Policy

On top of the pack's severity, you set per-dimension policy in `backstop.yml`:

```yaml
enforcement:
  policy:
    coverage_threshold:
      level: block
      applies-to: new-code
    pack_engines:
      level: block
      applies-to: new-code
      sources:
        backstop-ai/backstop-self:
          level: block
          applies-to: all-code
```

Two orthogonal knobs:

- **`level`** — `off` (skip and report it as skipped), `warn` (surface, never fail), or
  `block` (fail). Default is `block`.
- **`applies-to`** — `new-code` grandfathers pre-existing findings against the baseline;
  `all-code` counts everything. An **absent** `applies-to` means `all-code` — the strict
  floor. A dimension is never silently grandfathered just because you forgot to say.

`sources` scopes an override to findings from one specific pack, which is how you can run most
packs on a forgiving ratchet while holding one pack to zero tolerance.

## Baseline and the ratchet

If adopting a tool means "fix 4,000 findings before your build goes green again," nobody
adopts it. The **baseline** is the answer.

A baseline is a snapshot of the findings that existed when you adopted backstop (or at some
later checkpoint). Under `applies-to: new-code`, a dimension compares current findings against
that snapshot and only counts what's *net-new*. Inherited debt stays visible but doesn't block
you.

This is a **ratchet**, not an amnesty. Two properties make it work:

**Touching a file forfeits its grandfathering.** The grandfather applies to code you leave
alone. Once a file enters scope — you edited it — its violations are yours. That's what makes
the debt actually drain instead of sitting there forever.

**The first run is observation, not failure.** Adopting backstop on an existing codebase
should read as "here's what we noticed," grouped and counted, exiting 0 — not as a wall of
red. You're capturing a starting position, not being graded on it.

The baseline lives at `.backstop/baseline.json`, gitignored by default so a solo or local-first
project gets the ratchet from day zero without needing a remote. Teams can graduate to a
CI-generated baseline, which is tamper-resistant and doesn't get stomped by concurrent local
runs.

## Waivers

Sometimes a finding is genuinely not a problem, or genuinely not fixable right now. The
**waiver** is the accountable escape hatch: a signed statement about *one finding*, not a
switch that silences a rule.

You write it as an inline token, in whatever comment syntax your language already has, at the
finding's location:

```go
// @waiver:go-core-no-init-functions:accepted-risk:2026-12-31 registers the driver, see ISSUE-042
```

The grammar is `@waiver:<rule-id>:<reason-code>:<expiry>[ note]`. Four reason codes, and only
four: `false-positive`, `accepted-risk`, `deferred`, `third-party`. Every waiver carries an
expiry date — there are no permanent waivers, only ones you haven't revisited yet. When it
expires, the finding comes back.

Backstop finds the token by byte-scanning the raw bytes of the finding's own reported line and
the line directly above it. It never parses your source and encodes no language's comment
syntax — which is the thin-executor principle showing up again, in the one subsystem where
it'd be most tempting to cheat.

Three things follow from the design:

- **A waiver is per-finding, per-rule, per-location.** You can't waive a rule globally, and you
  can't waive a category. Silencing a whole check is not an operation backstop offers.
- **`waiver_resolution` is itself a gate dimension.** Malformed tokens, expired waivers, and
  waivers pointing at rules that no longer fire are all findings.
- **Some things are non-waivable by policy.** The valve doesn't open on everything.

The intended posture: fixing beats waiving. If the *check* is wrong, fix the pack and bump its
version — a waiver is the interim measure while you do that, not the resolution.

## Artifacts, briefly

Everything above is about verifying code. Backstop also governs the work *before* the code —
the intent, the requirements, the plan — as **artifacts**: schema-validated markdown files
with structured frontmatter, living in your artifact root.

There are exactly two tracks:

- **`issue → plan`** — reactive work. A bug, tech debt, a policy violation. Issues never get
  specs; you plan directly from the issue.
- **`bundle → spec → plan → implementation`** — proactive work. A bundle explores a problem
  space and accumulates open questions and design decisions; a spec turns a piece of it into
  requirements, claims, and mandated test names; a plan sequences that into TDD tasks.

This is what closes the loop with the gate. A spec's **claims** name the tests that must
exist; `test_verification` checks that they do and that they passed. A spec's **contracts**
name signatures; `contract_signature` checks the code matches. `requirement_traceability`
checks the chain isn't broken. The artifacts aren't documentation that drifts — they're inputs
the gate reads.

You can also skip all of it. `backstop init --no-sdlc` gives you the pack-declared checks and
no artifact layout at all. The artifact workflow is a capability you adopt, not a tax you pay.

The full picture — every artifact type, the maturity levels, what promotion requires, how to
author them — is in [artifact-workflow.md](artifact-workflow.md).

## Putting it together

The pieces compose like this:

**Packs** declare what to check and which tools to run. **Engines** are those tool bindings;
backstop dispatches them and reads SARIF back. The **gate** organizes findings into
**dimensions**, adjudicates **waivers**, compares against the **baseline**, applies your
**policy**, and returns a verdict. **Artifacts** supply the promises — tests, contracts,
requirements — that several dimensions verify against.

Core supplies none of the knowledge and all of the mechanism. That split is the whole design.

## Where to go next

- **[getting-started.md](getting-started.md)** — the hands-on tutorial, if you skipped it.
- **[cli-reference.md](cli-reference.md)** — every command, flag, and exit code.
- **[pack-authoring.md](pack-authoring.md)** — write your own rules, engines, and recipes.
- **[artifact-workflow.md](artifact-workflow.md)** — the two tracks in full.
