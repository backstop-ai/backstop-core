---
title: "Pack Command Execution Governance"
number: BUNDLE-021
created: "2026-07-26"
schema_version: bundle/v2

bundle:
  name: pack-command-execution-governance
  version: "0.1.0"
  created: "2026-07-26"
  category: infrastructure

status:
  maturity: exploring

problem:
  summary: >
    NOTHING coherent governs the commands a pack asks backstop to run — and the parts that DO
    work were built for other reasons and never designed as a posture. Verified against the tree
    2026-07-26. There are three distinct execution surfaces with three different trust postures
    and no unifying model. (1) `engine.TrustedToolAllowlist()` (`pkg/pack/engine/allowlist.go:17`)
    gates ONLY bindings carrying a non-nil `Provision` — the tools backstop DOWNLOADS and
    version-pins on the user's behalf (`pack_gate.go:813`, `manifest.go:547`,
    `packval/executor.go:63`, `pack_gate_provision.go:85`, `recipe_apply.go:137` all guard on
    `Provision`). That is a real supply-chain control over what backstop itself installs, and it
    is legitimate — but it does not reach commands invoking software already on the machine. Its
    doc comment claims otherwise ("A tool ABSENT from this map may not be run by any
    pack-declared command"); correcting that claim and its dead entries is ISSUE-082, filed
    separately. (2) Pack-shipped CONVERT scripts run through `packval.SandboxedRunStdout`
    (`pkg/packval/sandbox.go:93`) on both the findings path (`pack_gate.go:693`) and the coverage
    path (`pack_gate.go:472`), under a deny-default, deny-network, deny-file-write macOS profile
    whose reads are scoped to the pack directory plus system dylib paths
    (`darwinSandboxProfile`, `sandbox.go:36`). That is the only real trust boundary in the
    system today. (3) Everything else runs with FULL AMBIENT PERMISSIONS: pack-declared engine
    commands go `splitCommand` → `check.ExecCommandRunner.RunStdout` → `exec.CommandContext`
    (`pack_gate.go:591,648`; `pkg/check/runner.go:52`), the recipe transform dispatch does the
    same (`cmd/backstop/recipe_apply.go:213`), and — the sharpest case — a pack-shipped coverage
    PRODUCER script is executed directly from the pack directory, unsandboxed, with cwd set to
    the project root (`pack_gate.go:428`), by deliberate design (ISSUE-045). One meaningful
    mitigation exists and is undocumented as a guarantee: `splitCommand` (`pack_gate.go:887`)
    uses `strings.Fields`, so there is NO SHELL — no pipes, redirects, globs, or quoting. A pack
    can run one program with plain arguments, not a shell one-liner. And on Linux none of the
    sandboxing exists at all: `sandbox.go:76,112` return `sandbox unavailable on linux in this
    build` (ISSUE-020, now a launch blocker), so the one real boundary is absent exactly where
    CI runs. The underlying question this bundle owns is not "which tool names are allowed" —
    that framing was never going to be the mechanism — but what posture SHOULD govern
    pack-declared execution, given that installing a pack means agreeing to run its code.

  user_story: >
    As a consumer installing a third-party pack from a GitHub coordinate into a repository I am
    responsible for — and as the maintainer of a tool whose entire value proposition is that it
    tells the truth about a codebase — I want the trust posture around pack-declared command
    execution to be a DELIBERATE, STATED position rather than the accidental residue of three
    features built for other reasons, so that I can answer "what can a pack I install actually
    do to this machine?" with something more precise than "anything a program on your PATH can
    do, plus run its own scripts," and so that whatever the answer turns out to be, it is a
    posture I chose rather than one I inherited. Concretely: the doc comment that currently
    claims a non-negotiable security gate must either become true or be replaced by an honest
    statement of what IS guaranteed — and if the honest answer is "installing a pack is like
    installing an npm package with a postinstall hook," that must be said out loud, in the
    documentation and at the moment of install, rather than left for a consumer to discover by
    reading `pack_gate.go`.

solution:
  approach: >
    UNDECIDED — nothing is chosen and no OQ is pre-resolved. This bundle exists to hold the
    question, not to answer it. Six genuine forks are recorded below with no leans; the founder
    drives resolution and promotion. What IS settled going in are three standing invariants,
    recorded as DD-1..DD-3, which CONSTRAIN every candidate answer without selecting any of
    them: the governance mechanism must be generic over pack-DECLARED data with zero baked
    tool/language knowledge (thin executor — an allowlist of tool NAMES is precisely the shape
    this invariant is suspicious of, which is part of why the question is open at all); packs
    stay external and the declaration must travel with the pack while `backstop.lock` remains
    the durable record; and the failure posture is adjudicated per-case by loud-≠-blocking
    rather than set uniformly. The shape of the decision space, as raised: publisher trust
    (signing/provenance) versus behavioral declaration (a pack states what its command NEEDS and
    core derives a profile) versus extending the sandbox to engine commands versus locating the
    judgment in the consumer's own `backstop.yml`. These are not mutually exclusive and may
    compose. Two hard grounding facts constrain any answer. FIRST, the sandbox cannot simply be
    widened: SPEC-054 recorded (spec line 958) that `packval.SandboxedRun*`'s deny-all-writes
    profile makes a recipe `transform` — whose entire purpose is to write a consumer file —
    structurally impossible, and the same logic exempts the coverage producer, which must run
    the project's real toolchain from the project root. Extending the boundary means designing a
    graduated profile, not flipping a switch. SECOND, there is no publisher-identity substrate
    to build on: `resolveGitURL` (`pkg/pack/distribution/add.go:315`) hardcodes
    `https://github.com/<name>.git`, the `GitCloner` interface has no production implementation
    at all (ISSUE-073 — remote install currently nil-panics), and the file named `provenance.go`
    tracks tool_config CONTRIBUTION attribution, not publisher provenance. Whatever lands must
    also survive the Linux hole (ISSUE-020) — a posture that depends on a boundary that does not
    exist on the platform CI runs on is not a posture.
---

# Pack Command Execution Governance

## Current Thinking

### The three execution surfaces — enumerated and verified 2026-07-26

Every claim below was checked against the tree on 2026-07-26. The point of the enumeration is
that the surfaces do not share a model: each was built for its own reason, and the aggregate
posture is a residue rather than a design.

| Surface | Runner | Trust control | Where |
|---|---|---|---|
| Provisioned engine tool (semgrep, ast-grep) | `check.ExecCommandRunner` | Allowlist + lock-pin | `pack_gate.go:591,648` (gate at `:816`) |
| Non-provisioned engine command (go, bun, npx, grep…) | `check.ExecCommandRunner` | **none** | `pack_gate.go:443,591,648` |
| Pack-shipped **convert** script | `packval.SandboxedRunStdout` | **sandbox** (deny-default) | `pack_gate.go:472` (coverage), `:693` (findings) |
| Pack-shipped **producer** script | `check.ExecCommandRunner` | **none** — deliberate | `pack_gate.go:428` |
| Pack-shipped sandbox **validator** | `packval.SandboxedRun` | **sandbox** (deny-default) | `pkg/packval/executor.go` |
| Recipe **transform** dispatch | `check.ExecCommandRunner` | Allowlist (provisioned only) | `cmd/backstop/recipe_apply.go:213` |

Read that table as one sentence: **pack-shipped scripts are sometimes sandboxed and sometimes
not, and the split is not by risk — it is by whether the script needs to write.**

### What the tool allowlist ACTUALLY covers

`engine.TrustedToolAllowlist()` (`pkg/pack/engine/allowlist.go:17`) returns eight
`{tool → pinned version}` entries: `semgrep 1.96.0`, `ast-grep 0.43.0`, and presence-only `"*"`
pins for `grep`, `rg`, `oxlint`, `bun`, `tsc`, `prettier`. `CheckToolAllowed` (`:66`) rejects a
tool absent from the map, or present but not matching the caller's lock-resolved version.

Every one of the five call sites gates on the binding carrying a **non-nil `Provision`**:

- `cmd/backstop/pack_gate.go:812-815` — `checkEngineToolAllowed`, `if binding.Provision == nil { return nil }`
- `pkg/pack/manifest.go:547` — `validateEngine`, `if binding.Provision != nil { … }`
- `pkg/packval/executor.go:63` — `RunEngine`, same guard
- `cmd/backstop/pack_gate_provision.go:85` — same guard on the provisioning path
- `cmd/backstop/recipe_apply.go:137` — reuses `checkEngineToolAllowed` before building the transform dispatch

A `provision:` block is what backstop DOWNLOADS and version-pins on the user's behalf. So the
allowlist's real subject is *"which tools will backstop itself fetch and install, and at what
pinned version"* — **a supply-chain control, and a legitimate one.** Read that way it is not
broken; it is doing a narrower job well.

What it does NOT do is reach commands that invoke software already on the machine. A
non-provisioned binding — `go build ./...`, `npx --no-install eslint`, a pack's own script —
never touches the check. The doc comment at `allowlist.go:7-9` claims the opposite: *"A tool
ABSENT from this map may not be run by any pack-declared command, no matter what a pack
declares — this is the non-negotiable security gate."* That is false as written.

**Correcting the comment and the dead entries is ISSUE-082, filed 2026-07-26, and is NOT this
bundle's scope.** ISSUE-082 also establishes the reachability fact that makes the gap concrete:
five of the eight entries (`rg`, `oxlint`, `bun`, `tsc`, `prettier`) have zero `provision:`
blocks anywhere in either pack repo, so nothing can ever reach `CheckToolAllowed` with those
tool names. Notably `typescript-toolchain` — the pack those four entries were ADDED for —
invokes everything through `npx --no-install` and declares no provision block at all. This
bundle owns the governance question underneath that cleanup: once the comment is honest, what
should be true instead?

### What the sandbox ACTUALLY covers

`darwinSandboxProfile` (`pkg/packval/sandbox.go:36`) builds:

```
(version 1)(import "bsd.sb")(deny default)(allow process*)(allow file-read* <scoped subpaths>)(deny network*)(deny file-write*)
```

The read subpaths are the symlink-resolved pack directory plus the system/Homebrew dylib paths a
dynamically-linked interpreter needs at dyld load. **No project path is readable, nothing is
writable, and there is no network.** Two entry points use it: `SandboxedRun` (`:62`,
CombinedOutput — the exit-code sandbox validator) and `SandboxedRunStdout` (`:93`, clean stdout
— the convert step, wired at `pack_gate.go:472` for coverage and `:693` for findings, through
the `resolveSandboxedRunStdout` test seam at `:64`).

This is the real trust boundary in the system today, and it is a tight one. It is also narrow by
construction: it applies to scripts whose job is to PARSE bytes handed to them on stdin.

### What NOTHING covers

Pack-declared engine **commands**: `splitCommand(binding.Command)` → `check.ExecCommandRunner`
→ `exec.CommandContext`, with `Dir` set to the project root. `pkg/check/runner.go:38,53` even
carry `nosemgrep: go.lang.security.audit.dangerous-exec-command` suppressions with the
justification *"declared engine command from a verified pack manifest (not user input)"* —
where "verified" means content-hash-verified against the lock, i.e. proof the bytes are
UNTAMPERED, not proof they are benign.

And the sharpest case, which is the one worth sitting with: **`pack_gate.go:428` executes a
pack-shipped producer SCRIPT directly, unsandboxed, from the pack directory with cwd = the
project root.** This is not an oversight — the comment at `:409-414` records it as a deliberate
ISSUE-045 decision, because the producer's job is to run the project's real toolchain (`go
test` / `go list`) and fold the results into a payload, which is impossible under a profile that
denies project reads, writes, and network. The split it documents is explicit: *"the producer
runs UN-SANDBOXED … the convert below runs SANDBOXED (parse only)."*

So the honest statement of today's boundary is: **arbitrary pack-shipped executable code runs
with the user's full ambient permissions whenever it needs to do anything real, and is sandboxed
only when it is confined to pure parsing.** The sandbox is not the trust boundary for packs; it
is the trust boundary for the ONE step that could afford to have one.

### The one real mitigation — there is no shell

`splitCommand` (`cmd/backstop/pack_gate.go:887`) is:

```go
fields := strings.Fields(command)
```

No `sh -c`, no shell interpolation anywhere on the path. So a pack-declared command is *one
program name plus whitespace-separated plain arguments* — no pipes, no redirects, no globbing,
no command substitution, no `;`/`&&` chaining. `buildEngineArgv` (`pkg/packval/executor.go:41`)
and the transform dispatch (`recipe_apply.go:213`) tokenize the same way.

This materially narrows the blast radius of a malicious `command:` string, and it is currently
an **implementation detail** — a tokenization convenience that happens to be doing real security
work, with no test asserting it as an invariant and no documentation stating it as a guarantee.
That is OQ-6.

### On Linux, none of it exists

`pkg/packval/sandbox.go:76` and `:112` both return
`errors.New("sandbox unavailable on linux in this build")` — a hard error, not a silent
passthrough, which is the correct failure direction. Tracked as **ISSUE-020** (`risk: critical`,
open since 2026-06-21), now a launch blocker.

Cited here as an **interaction, not as this bundle's scope**, and the interaction is
load-bearing in both directions. Any posture that leans on the sandbox is leaning on something
that does not exist on the platform CI runs on. Conversely, whatever ISSUE-020 builds for Linux
(seccomp? bubblewrap? landlock? a container?) will be a SECOND profile implementation, and if
this bundle has by then defined what a profile must express, the Linux work implements a stated
contract instead of re-deriving one. Sequencing between the two is a real question and is
deliberately not answered here.

### The functional frame — installing a pack IS agreeing to run its code

This is the frame the whole question has to be argued inside, and it is why "enumerate the
allowed tool names" was never going to be the mechanism.

A pack's entire purpose is to bring a toolchain. `backstop.yml` declares packs; `pack add`
copies them into gitignored `.backstop/packs/`; the gate runs what they declare. That is the
same bargain as an npm `postinstall`, a GitHub Action pinned to a tag, or a `pre-commit` hook
repo — **installing it is consenting to execute it.** That is inherent to the architecture, not
a defect in it, and no amount of name-matching changes it: a pack that wants to run something
hostile does not need an exotic tool, it needs `bash` — or, today, just a producer script.

Two things follow. First, a tool-NAME allowlist is a weak primitive for this job even in
principle; it constrains the noun and says nothing about the verb. Second, and this is the part
worth being uncomfortable about: **the current implementation is arguably fine as an engineering
matter and indefensible as a communication matter.** The code does roughly what an npm install
does. The doc comment claims a "non-negotiable security gate." The gap between those two is the
actual defect, and it is possible that the entire correct outcome of this bundle is an honest
statement plus an install-time disclosure — with no new enforcement at all. That outcome must
stay genuinely on the table through OQ resolution rather than being crowded out by the more
interesting mechanisms.

### Prior art and adjacency, checked

- **BUNDLE-001 OQ-3** ("Sandbox boundary — declarative-only forever?", `exploring`, raised
  2026-04-08) is the nearest existing question, and it is a DIFFERENT one: it asks whether packs
  may contain Turing-complete checks at all, and notes the sandbox IMPLEMENTATION choice
  (process isolation / seccomp / wasm / convention) as unresolved. It predates the packs-only
  architecture and does not reach engine-command execution. Adjacent, not overlapping — but
  whoever resolves OQ-3 here should read it, because "just don't allow arbitrary code" was
  considered once already.
- **BUNDLE-005** (`ready`) covers sandboxing at `pack check` / `pack test` time — validating a
  pack's own fixtures — which is authoring-time verification, not consumer-time execution
  governance.
- **ADR-0017** carries a registry-as-publisher model (D-096/D-097: backstop runs gates,
  publishes under backstop-controlled scopes, is the authoritative source for content hashes)
  and names signing infrastructure as required-but-unbuilt. That is the closest thing to prior
  art for OQ-1, and it presumes a registry this project does not have.
- **`DetectTamper`** (`pkg/pack/distribution/tamper.go:44`) already detects four
  adversarial-UPDATE categories — fixture removal, severity downgrade, risk-class change, rule
  removal — between two versions of a pack. That is a real, shipping, behavior-oriented
  integrity check, and it is prior art for OQ-2's shape: it reasons about what a pack DOES, not
  what it is named. It says nothing about command execution.

### What is decided vs open

**DECIDED:** nothing about the mechanism. Only the three standing invariants recorded as
DD-1..DD-3, which constrain every candidate without selecting one.

**OPEN:** all six questions below. None has a lean recorded, by design. Maturity stays
`exploring`; promotion is the founder's call.

## Draft Design Decisions

These are **INVARIANTS INHERITED FROM STANDING PROJECT LAW**, recorded so that every candidate
answer is measured against them. **None of them answers any open question.** DD-3 in particular
supplies the principle that will ADJUDICATE OQ-5 — it is not OQ-5's answer.

- **DD-1: The mechanism is generic over pack-DECLARED data — zero baked tool/language
  knowledge.** (Standing law: thin executor.) Whatever governs execution must be a generic
  operation over data a pack declares, never a table of known tools, a special case per
  language, or hardcoded knowledge of which pack may run what. This invariant is unusually sharp
  here, because **the artifact under examination is itself a hardcoded list of tool names.**
  `TrustedToolAllowlist` survives the `backstop/self` rule only via an explicit in-code
  suppression (`allowlist.go`, `nosemgrep: no-baked-language-token` on the `tsc` entry) plus a
  comment arguing the key is *"a tool-name lookup datum … NOT a baked routing/command literal —
  it never sources a command."* That argument is coherent for a supply-chain pin. It is worth
  re-examining honestly if the allowlist is ever proposed as the GOVERNANCE mechanism rather
  than the provisioning pin — a name-matching governance table would be much harder to defend
  under DD-1, and ISSUE-082 already records that the suppressed dogfood rule was substantively
  correct.

- **DD-2: Packs stay external; a declaration travels WITH the pack; the lock is the durable
  record.** (Standing law: packs always external.) If the answer involves a pack DECLARING
  something (OQ-2's shape), that declaration is authored in pack-owned content and, if it must
  survive re-materialization of gitignored `.backstop/packs/`, recorded in `backstop.lock`. It
  may never live in a core-side registry of external packs — that recreates the vendoring model
  the architecture exists to avoid. Note the existing durability substrate and its limit:
  `LockEntry` carries `content_hash` (`ComputeContentHash`, `distribution/hash.go:17`), which
  proves the installed bytes are UNTAMPERED and says nothing about whether they are safe.

- **DD-3: The failure posture is adjudicated by loud-≠-blocking, per case — not set uniformly.**
  (Standing law: block defects and broken promises; warn-with-guidance for un-adopted
  capability; the enemy is silent/vacuous green, not passing.) A pack doing something it
  declared it would not do is a broken promise and is block-shaped. A pack that has simply not
  yet adopted a new declaration is un-adopted capability and is warn-shaped. This DD forbids a
  uniform answer and fixes the principle; sorting the concrete cases is OQ-5 and is unresolved.

## Open Questions

Six genuine forks. **No leans are recorded — deliberately.** Options are stated with the
strongest case for and against each, because the founder resolves these and a lean would bias
the record. They are not mutually exclusive; several could compose into one posture.

- **OQ-1 — IS PUBLISHER TRUST THE RIGHT PRIMITIVE?** Rather than enumerating tools or commands,
  trust the SOURCE: signing, provenance attestation, or a consumer-held trusted-publisher list,
  with the pack's identity rather than its contents carrying the weight. This is how the
  ecosystem generally answers "you're about to run someone's code" (npm provenance, sigstore,
  Actions pinned to a SHA), and it composes cleanly with the functional frame above — if
  installing a pack is inherently consenting to execute it, then WHO you're consenting to is the
  decision that actually matters. **The load-bearing sub-question: what establishes publisher
  identity, given packs are plain git repos?** Today there is nothing to build on, verified
  2026-07-26: `resolveGitURL` (`add.go:315`) hardcodes `https://github.com/<name>.git`, so a
  publisher IS a GitHub org string with no verification; `GitCloner` (`add.go:14`) has **no
  production implementation whatsoever** and `pack add` passes it nil (ISSUE-073 — remote
  installs currently panic), so the remote path this question governs is not merely unguarded,
  it does not run; and the file named `provenance.go` is about tool_config CONTRIBUTION
  attribution, not publisher provenance. ADR-0017's registry-as-publisher model (D-096/D-097)
  presumes registry and signing infrastructure that does not exist. So: is publisher identity a
  GitHub org (free, weak, and already implied by the coordinate)? a signature over the pack
  content (real, but needs key distribution and a revocation story)? or a curated list that
  someone must maintain? And is this answerable at all before DIR-026 / SPEC-055 make remote
  consumption real — or does the fact that remote install is being BUILT RIGHT NOW make this the
  moment to decide, before there is an installed base to migrate?

- **OQ-2 — SHOULD PACKS DECLARE BEHAVIOR INSTEAD OF NAMES?** A pack states what its command
  NEEDS — reads project files, writes files, network access, subprocess spawning — and core
  derives a sandbox profile from the declaration rather than matching a string. This is the same
  shape as **BUNDLE-020's OQ-1 resolution** (v0.2.0, 2026-07-26): describe the CONTRACT, do not
  enumerate the implementations; a version number, like a tool name, is only ever a proxy for
  the thing that actually matters. It also fits DD-1 far more comfortably than a name list, and
  it has in-repo precedent in `engine.FieldContract{Requires, Forbids}`
  (`pkg/pack/engine/fieldcontract.go`) — a pack declaring named requirements that core
  validates. **The load-bearing sub-question: what happens when a declaration is WRONG, or
  LIES?** Two failure modes with different answers. An HONEST declaration that is merely
  incomplete (the author forgot the tool writes a cache file) is a usability problem — the
  command fails under a too-tight profile, and the fix is a good error message naming the denied
  operation. A DISHONEST declaration is the security case, and the only thing that makes a
  declaration more than documentation is that the derived profile is ENFORCED — which routes
  straight into OQ-3 (the sandbox must actually be able to express and enforce the derived
  profile) and dies on Linux today (ISSUE-020). Note the honest counter: a declaration that is
  enforced is valuable; a declaration that is merely recorded is a comment with extra steps.
  Also unresolved: is `DetectTamper`'s posture (`tamper.go:44`) — flagging adversarial CHANGES
  between versions — a cheaper approximation, i.e. does a declaration only need to be checked
  when it CHANGES?

- **OQ-3 — SHOULD THE SANDBOX EXTEND TO ENGINE COMMANDS, NOT JUST CONVERT SCRIPTS?** Is the
  current split principled or incidental? **The case for principled:** convert scripts parse
  bytes on stdin and legitimately need nothing else, so deny-everything costs them nothing;
  engine commands invoke user-installed toolchains that legitimately read the whole project and
  write build output, so a sandbox around them would have to be permissive enough to be nearly
  meaningless. **The case for incidental:** the coverage PRODUCER (`pack_gate.go:428`) is a
  pack-SHIPPED script — the same trust category as the convert script sitting three functions
  away — and it runs completely unsandboxed. The line was drawn at "does it need to write,"
  which is a capability question, not a trust question. **And extending is NOT a simple
  widening — this is the hard constraint.** SPEC-054 (spec line 958; plan lines 2198, 2339)
  recorded that `packval.SandboxedRun*`'s deny-all-writes profile makes a recipe `transform` —
  which by definition writes a consumer file — *structurally impossible*, and explicitly
  declined to route the transform dispatch through it, adding that *"relaxing the profile to fit
  would be a security regression."* So the real question is whether a GRADUATED profile is worth
  designing (project-read + scoped-write + no-network for engine commands; pack-read-only for
  converts), what a "scoped write" even means for `go build` writing into a module cache outside
  the project, and whether that graduation is affordable given it must be implemented twice —
  once for `sandbox-exec` and once for whatever ISSUE-020 lands on Linux.

- **OQ-4 — WHERE DOES THE CONSUMER'S OWN JUDGMENT ENTER?** Should `backstop.yml` carry a
  consumer-owned trust declaration — per-pack or global — so that the party ACCEPTING the risk
  makes the call, rather than core's authors making it for everyone? **For:** the consumer is
  the only one who knows whether a pack came from their own org or a stranger, and core cannot
  know that; it is also the only option that scales without core maintaining a curated list.
  **Against, and this is the strong objection: a setting that everyone reflexively enables is
  not a control.** If the first thing every consumer must do to make the gate run is set
  `trust: true`, the field is a speed bump that teaches people to click through security
  prompts — and this project's whole thesis is that a check everyone learns to bypass is worse
  than no check. Sub-questions: is there a formulation that resists reflexive enabling (per-pack
  and specific, rather than global and boolean — e.g. the consumer acknowledges a specific
  content hash, so the acknowledgment expires on every pack update)? Does the answer differ for
  a first-party org pack versus a third-party one, and can core even distinguish them? And how
  does this interact with OQ-1 — is a trusted-publisher list simply the consumer-owned
  declaration, stated once instead of per-pack?

- **OQ-5 — WHAT IS THE FAILURE POSTURE, AND DOES IT DIFFER BY MOMENT?** Block, warn, or prompt —
  and is the answer the same at a pack's FIRST INSTALL as on every subsequent gate run? DD-3
  fixes the principle (loud ≠ blocking) but not the sorting. The moments are genuinely
  different: **install** is interactive, happens once, and is the natural place for a disclosure
  or a consent prompt — but `pack add` also runs in CI and in automation, where a prompt is a
  hang; **every gate run** is non-interactive by definition, high-frequency, and is where a
  warning goes stale fastest. A third possibility is that the posture belongs at neither, but at
  the moment content CHANGES — `pack update` / `relock` — which is where `DetectTamper` already
  lives. Note the standing rule cuts specifically against the tempting answer here: making
  execution governance BLOCK on first install would make backstop unusable on day one for every
  existing pack, while making it merely warn on every run produces a message everyone learns to
  ignore. Also: is a governance failure a CONFIG error (exit 2, the `pack_lock_verification`
  posture) or a VIOLATION (exit 1)? And what is the retroactive default for the packs that exist
  today and declare nothing — which, as with BUNDLE-020's OQ-6, is 100% of them?

- **OQ-6 — DOES THE SHELL-FREE EXECUTION PROPERTY NEED TO BE A STATED GUARANTEE?**
  `strings.Fields` tokenization (`pack_gate.go:887`, `packval/executor.go:41`,
  `recipe_apply.go:213`) is currently an implementation detail doing real security work: no
  shell means no pipes, redirects, globs, chaining, or command substitution from a pack-declared
  `command:` string. **Should it be an INVARIANT with a test that fails when someone introduces
  `sh -c`?** For: it is the single most valuable property the current implementation actually
  has, it is cheap to assert, and today nothing stops a future contributor from "fixing" a
  pack's inability to pipe by reaching for a shell — which would silently delete the guarantee.
  Against: three separate tokenization sites means an invariant has three places to be violated,
  a unit test on `splitCommand` proves nothing about a future fourth exec path, and stating a
  guarantee narrower than consumers will assume may be worse than stating none. **The
  load-bearing sub-question: does anything legitimately NEED shell semantics?** A pack wanting
  `tool | filter > out.json` today must ship a script — which routes into the producer/convert
  distinction OQ-3 governs, and which is arguably the correct answer (a script is a reviewable
  file with a content hash; a shell string in a manifest is neither). If that IS the answer, it
  should be written down as the intended escape hatch rather than left as an accident of
  tokenization. Related and unverified: whether `splitCommand`'s existing `@waiver:` on the
  `no-structural-name-split-on-spine` rule (`pack_gate.go:888`) is the right place to anchor
  such an invariant, or whether it should be a dogfooded rule in `backstop-self-pack` that goes
  RED on a shell invocation anywhere in the exec path.

## Notes / Ideas

- **The strongest possible outcome might be "write it down."** Every mechanism in OQ-1..OQ-4 has
  real cost, and the actual defect today is a false claim in a doc comment, not an exploited
  hole. An honest statement — *"installing a pack means running its code, exactly like an npm
  postinstall; here is what IS guaranteed: content-hash integrity, no shell, version-pinned
  provisioning; here is what is NOT"* — plus a disclosure at install time, might be the whole
  correct answer. It would cost days rather than weeks, would not be wrong later if a mechanism
  lands on top of it, and is the only option that ships before the Tier-1 blockers. This is
  recorded as a note rather than as a lean because it is the option most likely to be crowded
  out by the more interesting ones.

- **The producer script is the case that most resists the current framing.** Everything else can
  be argued as "a pack invokes a tool the user already installed." `pack_gate.go:428` runs a
  file the PACK SHIPPED, unsandboxed, from the project root. If a mental model has to be picked
  for what a pack can do, that line is the one that sets the ceiling — and it sits three
  functions away from a convert script that is denied even the ability to read the project.

- **"Verified pack manifest" means untampered, not benign.** The `nosemgrep` justification at
  `pkg/check/runner.go:38,53` reads *"declared engine command from a verified pack manifest (not
  user input)."* The verification in question is `ComputeContentHash` against `backstop.lock` —
  proof the bytes have not changed since install, which is a real and useful property, and which
  is orthogonal to whether the bytes were ever safe. Any argument in this bundle that leans on
  "verified" should say which of the two it means.

- **A name-based governance mechanism would be the most defensible thing to reject early.**
  DD-1 already makes it awkward; ISSUE-082 shows the existing list has drifted into five dead
  entries within one release cycle of being written; and the functional frame shows the noun was
  never the risky part. If OQ-1/OQ-2 both stall, the fallback should not silently be "extend the
  allowlist."

- **Sequencing against ISSUE-020 cuts both ways.** If this bundle resolves toward "declare
  behavior, derive a profile" (OQ-2 + OQ-3), it hands the Linux sandbox work a SPEC — a list of
  capabilities a profile must be able to express — instead of leaving it to re-derive one. If it
  resolves toward publisher trust or disclosure, ISSUE-020 is untouched by it. Worth knowing
  which world we're in before ISSUE-020 is planned, without blocking ISSUE-020 on this.

- **Spec seeds are deliberately absent.** Nothing is decided, and the decomposition of the work
  depends entirely on which of OQ-1..OQ-4 the answer lands in — publisher trust, behavioral
  declaration, sandbox extension, and consumer declaration produce four structurally different
  work breakdowns with almost no overlap. Writing seeds now would fabricate a shape the bundle
  has not earned.

## References

- **ISSUE-082** — *Tool Allowlist Unreachable Entries* (filed 2026-07-26, `technical-debt`,
  `scope: isolated`). The mechanical cleanup: five dead entries, a false doc comment, and the
  correctly-suppressed dogfood rule. **This bundle is the governance question underneath it, and
  does not duplicate its scope.** ISSUE-082 can and should proceed independently — it makes a
  false claim honest, which is strictly good regardless of what this bundle decides.
- **ISSUE-020** — *Linux sandbox is a hard error* (2026-06-21, `risk: critical`, open). Launch
  blocker. A hard interaction: the boundary this bundle reasons about does not exist on the
  platform CI runs on.
- **ISSUE-073** — *pack add nil GitCloner panic*. Remote pack installation has no production
  implementation; every real install to date is `source_type: local`. Directly constrains OQ-1's
  publisher-identity question.
- **ISSUE-045** — the decision record for the un-sandboxed producer / sandboxed convert split
  (`pack_gate.go:409-414`).
- **BUNDLE-020** — *Pack Core Version Compatibility* (`exploring`, v0.2.0). Its **DD-6**
  explicitly excludes `TrustedToolAllowlist` as a TRUST rather than compatibility boundary and
  names this bundle as the correct home — which is what surfaced the question. Its **OQ-1
  resolution (DD-4)** — declare named contracts, not versions; describe the contract rather than
  enumerating implementations — is the direct methodological precedent for OQ-2 here, including
  its conditional guard (DD-4d) against a declaration vocabulary becoming a back door for baked
  tool knowledge. Not a dependency in either direction.
- **ISSUE-062** — the originating incident for the compatibility thread that produced BUNDLE-020
  and, transitively, this bundle.
- **BUNDLE-001 OQ-3** — the 2026-04-08 "declarative-only forever?" question, including the
  still-open sandbox-implementation choice. Adjacent prior art; predates packs-only.
- **ADR-0017** — registry-as-publisher model (D-096/D-097) and the unbuilt signing
  infrastructure it presumes. Prior art for OQ-1.
- **SPEC-054** (line 958) / **PLAN-SPEC-054** (lines 2198, 2339) — the finding that
  `SandboxedRun*`'s deny-all-writes profile makes a recipe transform structurally impossible.
  The hard constraint on OQ-3.
- **DIR-026 / SPEC-055** — production remote dependency assembly. Tier-1 launch blocker; the
  work that makes OQ-1's remote-publisher question real rather than hypothetical.

### Tier

**Tier-2 by the founder's launch razor — NOT a launch blocker.** Recorded explicitly so triage
does not over-prioritize it on the strength of the word "security."

The three Tier-1 blockers are: **recipes** (SPEC-054), **remote pack consumption** (DIR-026 /
SPEC-055), and **Linux/CI viability** (ISSUE-020). This bundle is downstream of all three in
priority and interacts with two of them (remote consumption creates the third-party-publisher
population OQ-1 governs; the Linux work implements whatever profile OQ-3 would define). The
argument for Tier-2 rather than Tier-3 is that the honest-disclosure outcome noted above is
cheap and that DIR-026 is being built right now — deciding after there is an installed base of
remote packs is more expensive than deciding before.

## Version History

- **0.1.0** (2026-07-26): Initial bundle at `exploring`. Created from **BUNDLE-020's DD-6**,
  which excluded `TrustedToolAllowlist` from the compatibility question as a TRUST rather than
  compatibility boundary and named this bundle as its home.

  Problem framing established against a full verification sweep of the tree on 2026-07-26 —
  every line reference in this bundle was checked, not inherited. Findings recorded: the tool
  allowlist gates ONLY non-nil-`Provision` bindings across all five call sites
  (`pack_gate.go:813`, `manifest.go:547`, `packval/executor.go:63`,
  `pack_gate_provision.go:85`, `recipe_apply.go:137`), making it a supply-chain control over
  what backstop itself downloads rather than the universal gate its doc comment
  (`allowlist.go:7-9`) claims; the macOS sandbox (`sandbox.go:36,62,93`) is the only real trust
  boundary and covers only pack-shipped convert scripts and sandbox validators; pack-declared
  engine commands, the recipe transform dispatch, and — **surfaced during verification and not
  previously part of the framing** — the pack-shipped coverage PRODUCER script
  (`pack_gate.go:428`, unsandboxed by deliberate ISSUE-045 design, cwd = project root) all run
  with full ambient permissions; and the one real mitigation is that `splitCommand`'s
  `strings.Fields` tokenization (`pack_gate.go:887`) means there is no shell.

  Six open questions raised with **no leans recorded, by design**: publisher trust as the
  primitive (OQ-1), behavioral declaration over name-matching (OQ-2, methodologically parallel
  to BUNDLE-020's DD-4), whether the sandbox should extend to engine commands given SPEC-054's
  finding that deny-all-writes makes a transform structurally impossible (OQ-3), where the
  consumer's own judgment enters (OQ-4), the failure posture and whether it differs by moment
  (OQ-5), and whether shell-free execution should become a stated, tested invariant (OQ-6).

  Three standing invariants recorded as DD-1..DD-3 (thin executor / packs external / loud ≠
  blocking) as CONSTRAINTS on candidate answers, explicitly not as resolutions. DD-1 is noted as
  unusually sharp here because the artifact under examination is itself a hardcoded list of tool
  names surviving `backstop/self` only via an in-code suppression.

  Spec seeds deliberately omitted — the work breakdown differs structurally across OQ-1..OQ-4,
  so writing seeds now would fabricate a shape the bundle has not earned. No `requirements[]`
  array; no promotion attempted. Tier-2 recorded explicitly so triage does not over-prioritize
  on the word "security."

  `bundle.category` set to `infrastructure` (the scaffold default was `feature`), matching
  sibling BUNDLE-020 — this governs a trust boundary in the pack substrate, not a user-facing
  feature.
