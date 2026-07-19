---
title: "Stash — Secure Local Resource Store"
number: BUNDLE-018
created: "2026-07-19"
schema_version: bundle/v2

bundle:
  name: stash-local-resource-store
  version: "0.1.0"
  created: "2026-07-19"
  category: feature

status:
  maturity: exploring

problem:
  summary: >
    Developers have excellent CLOUD secret managers (GCP / AWS Secret Manager,
    HashiCorp Vault) but surprisingly poor LOCAL equivalents. Local development
    today is a mess of plaintext `.env` files, API keys copied between projects,
    secrets leaking into shell history, general-purpose password managers that
    are not optimized for developer workflows, and ad-hoc notes for the long tail
    of things a developer needs at hand — service links, endpoints, subscription
    renewal dates, a teammate's birthday. There is no local tool that is at once
    ENCRYPTED, as natural to reach for as the filesystem, and shaped for how
    developers and their agents actually work. Stash is that tool: an ENCRYPTED
    LOCAL RESOURCE STORE with an exceptional CLI experience and far stronger
    security guarantees than plaintext. Deep Backstop integration is intended,
    but Stash must stand on its own as a STANDALONE local developer tool first.

  user_story: >
    As a developer, I want a single encrypted local store I can reach for as
    naturally as the filesystem — `stash put`, `stash get`, `stash ls`,
    `stash search` — to hold everything from API keys and passwords to links,
    endpoints, certificates, and stray notes, so that I stop scattering secrets
    across plaintext `.env` files, shell history, and mismatched tools. For
    low-risk resources I want the value handed back instantly; for credentials I
    want the store to be able to ACT ON MY BEHALF (create the GitHub issue, call
    the API) without ever revealing the secret to the caller. As a Backstop user,
    I want my agents to be granted AUTHORITY to use a capability without ever
    receiving CUSTODY of the underlying credential — while Stash remains fully
    useful even if I never touch Backstop.

solution:
  approach: >
    Build an exceptional encrypted local resource store FIRST, as a standalone
    developer tool, with a clean seam toward deep Backstop integration — never
    the reverse. Model EVERYTHING as a RESOURCE — `{key, value, type, metadata,
    tags, policy}` — addressed by first-class hierarchical dot-namespaces
    (`credentials.github.work`, `links.mongodb.agent-memory`,
    `people.jason.birthday`). A thin CLI speaks over a local IPC channel (unix
    socket / named pipe) to a long-lived daemon (`stashd`) that OWNS everything
    security-sensitive — encrypted storage, encryption, authorization, IPC, audit
    logging, and future MCP-server duty; the CLI NEVER touches encrypted storage
    directly. Encryption is at rest, with the master key anchored in native OS
    facilities (macOS Keychain, Windows Credential Manager / DPAPI, Linux Secret
    Service). Per-resource POLICIES govern operations `{read, write, use, delete,
    metadata}`, built around the FUNDAMENTAL read-vs-use distinction: `read`
    transfers CUSTODY (the caller receives plaintext); `use` transfers AUTHORITY
    (the caller requests an operation and the resource is never revealed — Stash
    performs it on the caller's behalf). That `use` seam is the on-ramp to a
    secure EXECUTION BROKER and, via a CAPABILITY MODEL that hardcodes NO
    integration knowledge, to deep-but-deterministic Backstop integration where
    recipe packs contribute capabilities and agents only ask whether a capability
    is available. Guiding principles: store anything; reveal what is safe to
    reveal; execute on behalf of the caller when it is not; assume integrations
    evolve independently of the vault; be standalone-useful while enabling deep
    Backstop integration. This bundle is `exploring`: the founder-decided product
    shape below is recorded as Draft Design Decisions, and the genuinely open
    mechanism questions (caller identity, storage/encryption granularity,
    key custody, policy shape, `use`-in-V1 scope) are left OPEN for the founder
    to drive.
---

# Stash — Secure Local Resource Store

## Current Thinking

### The gap: cloud secret management is solved, local is not

The thesis is a contrast. Cloud secret management is a mature, well-served space —
GCP / AWS Secret Manager, HashiCorp Vault. The LOCAL developer equivalent is
conspicuously bad: plaintext `.env` files checked out beside source, API keys
copy-pasted between repos, secrets echoed into shell history, password managers
built for browser logins rather than developer workflows, and a long tail of
"where did I put that" — service links, endpoints, subscription renewal dates, a
teammate's birthday — living in scratch notes. Stash closes that gap with an
encrypted local store that feels like the filesystem and is safe like a vault.

### Standalone first, Backstop-integrated second — not the reverse

The load-bearing framing (DD-1, DD-10) is that Stash is valuable to a developer
who never touches Backstop. Deep Backstop integration is intended and designed
for (the capability model, DD-9), but the product is an exceptional local
resource store on its own merits. Building it "Backstop-first" would couple the
vault to a moving target and forfeit the standalone market. Everything here is
ordered accordingly: nail the store, keep the integration a clean seam.

### The read-vs-use distinction is the conceptual keystone

Most of what makes Stash more than "an encrypted key-value store" flows from one
distinction (DD-7): `read` transfers CUSTODY (the caller ends up holding the
plaintext); `use` transfers AUTHORITY (the caller asks for an operation that
NEEDS the resource, and the resource is never revealed — Stash performs the
operation itself and returns the result). Low-risk resources (notes, links) are
just `read`; credentials increasingly want `use`. This is the same shape as the
bounded-agency / DAG-of-DAGs thesis in the broader project — an agent should be
granted the AUTHORITY to accomplish something, not the CUSTODY of the secret that
accomplishes it. `use` is the seam that grows into the execution broker (DD-8)
and aligns naturally with MCP and tool-based agent interfaces.

### What is founder-DECIDED vs genuinely OPEN

The founder has already fixed the PRODUCT shape — the boundary and non-goals, the
resource model, the CLI philosophy, the daemon architecture, the encryption
anchor, the policy operation set, the read/use distinction, the broker direction,
the capability model, and the guiding principles. Those are recorded below as
Draft Design Decisions (DD-1..DD-10), not open questions. What remains genuinely
OPEN is the MECHANISM layer — chiefly CALLER IDENTITY over local IPC (without
which `use` cannot be enforced against a malicious local caller), the storage
engine and encryption granularity, master-key custody and unlock UX, the policy
model's declaration/inheritance shape and the V1 scope of `use`, namespace
semantics, the type system, the capability-model contract with Backstop, the IPC
protocol, the audit log, and onboarding/import. Those are the Open Questions
(OQ-1..OQ-11). Maturity stays `exploring`; the founder drives OQ resolution and
promotion.

## Draft Design Decisions

DD-1..DD-10 are FOUNDER-DECIDED positions from the founding brain-dump, recorded
as decided (not open). They fix the product's shape; the mechanism questions they
raise are captured as OQs, not pre-resolved here.

- **DD-1: Product boundary — a store, not a platform.** Stash is NOT a cloud
  product, NOT a password manager, NOT a personal-knowledge-management (PKM)
  system, and NOT an agent framework. Those categories may become CONSUMERS of
  Stash, never Stash itself. Explicit V1 NON-GOALS: cloud sync, embeddings,
  semantic search, vector DBs, PKM graphs, collaboration/multi-user, browser UI,
  mobile, and universal IAM. The discipline is deliberate: build an exceptional
  ENCRYPTED LOCAL RESOURCE STORE first; everything else is downstream of getting
  that right.

- **DD-2: Everything is a RESOURCE.** The single primitive is a resource —
  `{key, value, type, metadata, tags, policy}`. Types span the full developer
  long tail: API keys, passwords, notes, links, birthdays, subscription/renewal
  dates, certificates, endpoints, and arbitrary blobs. Hierarchical
  DOT-NAMESPACES are first-class, not a naming convention bolted on later:
  `credentials.github.work`, `links.mongodb.agent-memory`,
  `people.jason.birthday`, `projects.backstop.demo-url`. (The exact SEMANTICS of
  the hierarchy — inheritance, wildcards, subtree listing — are OQ-6; the type
  SYSTEM is OQ-7. The primitive itself is decided.)

- **DD-3: CLI philosophy — fast, memorable, discoverable, filesystem-shaped.**
  The CLI is the product surface and must feel as natural as a filesystem:
  filesystem-feeling verbs (`put` / `get` / `ls` / `search` / `delete`), shell
  completion as a first-class concern (not an afterthought), and a design where
  users should RARELY need to remember a full resource name — completion,
  search, and subtree navigation carry them. Fast, memorable, discoverable.

- **DD-4: Architecture — thin CLI, long-lived daemon, isolated storage.** The
  CLI talks over a local IPC channel (unix socket / named pipe) to a long-lived
  local daemon, `stashd`, which owns the encrypted storage. The CLI NEVER touches
  encrypted storage directly. The daemon owns storage, encryption, authorization,
  IPC, audit logging, and (future) MCP-server functionality. On macOS, `launchd`
  manages the daemon lifecycle — start at login, restart on failure. (The IPC
  protocol shape is OQ-9; how the daemon knows WHO is calling is the hard open
  question, OQ-4.)

- **DD-5: Encryption at rest, master key anchored in native OS facilities.**
  Resources are encrypted at rest. The master key leverages native OS key-custody
  facilities wherever possible: macOS Keychain, Windows Credential Manager /
  DPAPI, Linux Secret Service. (The storage engine and encryption GRANULARITY are
  OQ-2; key CUSTODY and unlock UX — including the headless / no-Secret-Service
  fallback and recovery story — are OQ-3.)

- **DD-6: Per-resource POLICIES over a fixed operation set.** Each resource
  carries a policy governing operations `{read, write, use, delete, metadata}`.
  The policy makes resource classes behave differently: low-risk resources (notes,
  links) retrieve normally; credentials get richer semantics. (How policies are
  DECLARED, attached, defaulted, and inherited — and whether `use` operations
  ship in V1 at all — are OQ-5. The operation SET is decided; its wiring is open.)

- **DD-7: The READ vs USE distinction is FUNDAMENTAL.** `read` transfers CUSTODY:
  the caller receives the plaintext value. `use` transfers AUTHORITY: the caller
  requests an operation that requires the resource, the resource is NEVER
  revealed, and Stash performs the operation on the caller's behalf. This
  distinction is expected to become increasingly important for AI agents, which
  should be granted the authority to accomplish a task, not custody of the secret
  that accomplishes it. This is the conceptual keystone the broker (DD-8) and the
  capability model (DD-9) are built on.

- **DD-8: Future direction — a secure EXECUTION BROKER.** The `use` seam grows
  into a broker: not "give me the GitHub token" but "create a GitHub issue."
  Stash retrieves the credential, authenticates, performs the operation, and
  returns the result — the credential never leaves the broker. This aligns
  naturally with MCP and tool-based agent interfaces. (Future direction, seam
  designed in V1; not necessarily V1-implemented — see OQ-5.)

- **DD-9: Backstop integration via a CAPABILITY MODEL — Stash hardcodes NO
  integration knowledge.** Stash exposes a capability model; recipe PACKS
  contribute capabilities; Backstop determines which capabilities are available
  for a given execution; agents only ask whether a desired capability is
  available. Stash owns credential binding, authorization, execution, and
  auditing. This keeps the integration logic DETERMINISTIC while Backstop evolves
  independently — the same "hardcode no ecosystem knowledge, packs carry the data"
  discipline backstop-core lives by. (The manifest/contract SHAPE and whether V1
  ships the capability surface or just leaves the seam clean is OQ-8.)

- **DD-10: Guiding principles.** Store anything; reveal what is safe to reveal;
  execute on behalf of the caller when it is not; assume integrations evolve
  independently of the vault; be standalone-useful while enabling deep Backstop
  integration. These are the tie-breakers for every open question below.

## Open Questions

Status index (numbers held stable across versions for traceability). NONE are
pre-resolved — each carries options and the author's non-binding LEAN for the
founder to react to. OQ-4 (caller identity) is the load-bearing one: without it,
`use` vs `read` cannot be enforced against a malicious local caller.

- OQ-1  Home & runtime — **OPEN**
- OQ-2  Storage engine + encryption granularity — **OPEN**
- OQ-3  Master-key custody & unlock UX — **OPEN**
- OQ-4  Caller identity & authorization (load-bearing) — **OPEN**
- OQ-5  Policy model shape + V1 scope of `use` — **OPEN**
- OQ-6  Namespace semantics — **OPEN**
- OQ-7  Type system — **OPEN**
- OQ-8  Capability-model contract with Backstop — **OPEN**
- OQ-9  IPC protocol shape + versioning — **OPEN** (possibly spec-time detail)
- OQ-10 Audit log — **OPEN**
- OQ-11 Onboarding / import — **OPEN**

### Open

- **OQ-1 — Home & runtime.** Where does Stash live and what is it built in?
  (a) Its OWN repo — consistent with the standalone-product posture (DD-1) and
  the project's packs-are-external pattern; (b) inside backstop-core. And
  separately, the implementation language/runtime: (i) Go, for consistency with
  core; (ii) whatever best fits a long-lived daemon with native OS-keychain
  integration. **Lean:** own repo (the standalone posture and packs-external
  precedent both point there); language genuinely open — Go buys consistency and
  a shared idiom, but the daemon's OS-keychain / IPC / broker needs may argue for
  whatever fits best. Founder decides.

- **OQ-2 — Storage engine + encryption granularity.** What backs the store and at
  what granularity is it encrypted? Engine: (a) SQLite (queryable, transactional,
  single-file); (b) flat encrypted files (simple, greppable-when-unlocked,
  per-resource blast radius); (c) append-only log (audit-friendly, natural
  history, compaction cost). Granularity: whole-store vs per-resource encryption.
  Key layering: an OS-keychain-wrapped DATA key (keychain holds a key-encryption
  key; a data key encrypts resources) vs the OS keychain directly holding the
  master. **Lean (weak):** SQLite with per-resource encryption and a
  keychain-wrapped data key — but this trades against the append-only log's
  native audit story (OQ-10) and wants deciding together with it.

- **OQ-3 — Master-key custody & unlock UX.** How is the master key held and how
  often must the user unlock? Sub-questions: biometric / Touch ID prompt
  FREQUENCY (per-op? per-session? timeout-based?); LOCK / timeout semantics; what
  happens when the keychain is UNAVAILABLE (headless / SSH / Linux without Secret
  Service — the developer-server case is not an edge case); and the RECOVERY /
  export story if the OS keychain is lost. **Lean:** session unlock with an
  idle-timeout re-lock for interactive use, plus an explicit non-keychain unlock
  path (passphrase-derived key) for headless — but the headless fallback and
  recovery need a real decision, not a default. Ties to OQ-4 (a headless CI
  caller is exactly where identity + custody collide).

- **OQ-4 — CALLER IDENTITY & AUTHORIZATION (load-bearing).** This is the hard one
  for the daemon model. Over local IPC, how does `stashd` know WHO is calling?
  Options: (a) unix peer credentials (SO_PEERCRED / getpeereid — uid/gid/pid of
  the connecting process); (b) per-caller TOKENS issued at enrollment; (c)
  process ATTESTATION (binary path / signature / cmdline of the peer pid). And
  how do per-resource policies (DD-6) BIND to principals — per-process?
  per-user? per-agent? Without a real answer, `use` vs `read` (DD-7) cannot be
  enforced against a malicious local caller — any process on the box could ask
  for custody. **Lean:** peer creds as the FLOOR (cheap, kernel-backed identity),
  layered with per-agent tokens for the capability/broker path where "which
  agent" matters more than "which uid" — but this is the decision most likely to
  reshape the whole daemon, so it wants the founder's full attention. Note the
  known weakness of pid-based attestation (pid reuse / TOCTOU) if that route is
  considered.

- **OQ-5 — Policy model shape + V1 scope of `use`.** Two coupled questions.
  DECLARATION: how are policies declared, attached, and defaulted — by TYPE
  (a credential type defaults to a masked/`use`-preferred policy), by NAMESPACE
  INHERITANCE (`credentials.*` inherits a stricter policy), explicit per-resource,
  or a layered combination? SCOPE: is any `use`-operation actually IN V1, or is V1
  `read`/`write`/`delete`/`metadata` with `use` designed as a clean SEAM (DD-7 /
  DD-8) to be filled later? **Lean:** namespace-inherited defaults keyed by type,
  overridable per-resource; and V1 ships the store + read/write/delete with `use`
  as a designed-but-thin seam (maybe one reference `use` operation to prove the
  shape) rather than a full broker — consistent with "store first" (DD-1). Founder
  sets the V1 line.

- **OQ-6 — Namespace semantics.** Are dot-namespaces (DD-2) a REAL hierarchy or a
  naming CONVENTION? Real hierarchy implies: policy/tag INHERITANCE down a
  subtree, WILDCARD operations (`stash get credentials.github.*`), and subtree
  behavior for `stash ls credentials.github`. Also: how are leaf-vs-namespace
  COLLISIONS handled (a value at `credentials.github` when
  `credentials.github.work` also exists)? **Lean:** real hierarchy — inheritance
  is what makes policy-by-namespace (OQ-5) and discoverable `ls`/completion (DD-3)
  pay off — with a defined collision rule (likely: a node is either a leaf or a
  namespace, not both). Founder decides how far the hierarchy semantics go in V1.

- **OQ-7 — Type system.** Are types (DD-2) an ENUMERATED built-in set or FREEFORM
  strings? And do types DRIVE behavior — default policies (OQ-5) and rendering
  (a birthday renders plainly; a credential masks on display)? Options:
  (a) closed enum with built-in behavior; (b) freeform strings, behavior purely
  from policy; (c) built-in types with sensible defaults PLUS freeform escape
  hatch. **Lean:** (c) — a curated set of built-in types that drive default
  policy + rendering, with freeform allowed for the long tail. Keeps the common
  case delightful (DD-3) without boxing the store in (DD-2 "store anything").

- **OQ-8 — Capability-model contract with Backstop.** What is the MANIFEST SHAPE
  by which recipe packs contribute capabilities (DD-9)? This ties directly to
  BUNDLE-015 (pack-scaffolding-recipes) and its adoption-record model — a
  capability is plausibly another pack-contributed, adoption-gated declarative
  artifact. SEQUENCING: does V1 ship the capability SURFACE, or just leave the
  seam clean (DD-9) and defer the contract until the store + `use` seam exist?
  **Lean:** leave the seam clean in V1 and design the contract against BUNDLE-015's
  pack/adoption substrate once `use` is real — building the Backstop contract
  before the standalone store is proven inverts DD-1. Founder sets the sequencing.

- **OQ-9 — IPC protocol shape + versioning.** What does the CLI↔`stashd` wire
  protocol look like — JSON-over-socket? length-prefixed framing? how is
  compatibility/versioning handled as the protocol evolves? **Lean:** likely a
  SPEC-TIME detail rather than a bundle-level fork — noted here so it is not lost,
  but not a promotion blocker. Flagging in case the founder considers it
  load-bearing (e.g. if the broker/MCP path constrains the protocol early).

- **OQ-10 — Audit log.** What is the audit log's SHAPE, storage, tamper-evidence,
  and V1 QUERYABILITY? Every `read`/`use`/`write` against a credential is a
  security-relevant event the daemon (DD-4) is positioned to record. Options
  range from a simple append-only file to a hash-chained tamper-evident log, with
  or without a `stash audit` query surface in V1. **Lean:** an append-only log in
  V1 (couples naturally with OQ-2 if the store itself is append-only), tamper-
  evidence and rich querying deferred unless the founder wants them early. Also
  relates to BUNDLE-017 (provenance/audit patterns) — worth reusing shape, not
  reinventing.

- **OQ-11 — Onboarding / import.** What is the V1 import story? `.env` import?
  migration from existing plaintext files or other password managers? Or is V1
  purely manual `put` with import deferred? **Lean:** a `.env` importer is the
  highest-leverage on-ramp for the target user (it directly attacks the plaintext-
  `.env` problem in the thesis) and is likely worth V1; broader migrations
  deferred. Founder sets V1 scope.

### Resolved / dissolved

None yet — this bundle is newly `exploring`; no OQ has been resolved. Resolutions
will move here with full rationale as the founder works through them.

### Non-forks (recorded, not open)

- **`stash` vs `git stash` muscle-memory.** The command name `stash` collides with
  `git stash` in developer shells. This is a conscious NAMING / UX check to make,
  not a fork that blocks design — flagged so it is not overlooked (e.g. confirm no
  aliasing surprise, consider the invoked binary name vs subcommand ergonomics).
- **`secrets` pack name (backstop-packs) is an UNRELATED concern.** backstop-packs
  already has a `secrets` pack — that is secret SCANNING (gitleaks → SARIF), a
  detection concern. Stash is secret STORAGE / brokering. Avoid conflating the two
  in naming, docs, or scope; they do not overlap.

## Spec Seeds

PROVISIONAL decomposition (bundle is `exploring`; seeds will firm up as OQs
resolve, and several depend directly on open questions). Listed in rough
implementation order.

- **Daemon + storage + encryption core (`stashd`)** — the long-lived local daemon
  that owns encrypted storage, encryption at rest, and the OS-keychain-anchored
  master key (DD-4, DD-5). The CLI never touches storage; this is where the
  storage engine and encryption granularity (OQ-2) and key custody / unlock UX
  (OQ-3) land. The foundation everything else sits on.

- **CLI + shell completion** — the filesystem-shaped command surface (`put` /
  `get` / `ls` / `search` / `delete`), first-class shell completion, and the
  IPC client half of the CLI↔daemon channel (DD-3, DD-4; protocol per OQ-9).
  Depends on the daemon seam existing.

- **Policy engine (read / write / use / delete / metadata)** — per-resource
  policy declaration, attachment, defaulting, and evaluation over the fixed
  operation set (DD-6), including the caller-identity binding that makes `use` vs
  `read` enforceable (DD-7). Shape and V1 `use`-scope are OQ-5; caller identity is
  OQ-4 (load-bearing — this seed cannot be sound without it).

- **Capability model + broker seam** — the `use`-operation execution path (DD-7,
  DD-8) and the capability contract by which recipe packs contribute capabilities
  and Backstop determines availability (DD-9). Whether V1 ships the surface or
  just a clean seam is OQ-8; ties to BUNDLE-015's pack/adoption substrate.

- **launchd / lifecycle packaging** — daemon lifecycle management (start at login,
  restart on failure) on macOS via `launchd`, and the equivalent packaging story
  on other platforms (DD-4). The "it's always there and just works" layer.

## Notes / Ideas

- **The read-vs-use distinction is the pitch.** `read` = custody, `use` =
  authority (DD-7). It is the cleanest one-line framing of why Stash is more than
  an encrypted key-value store, it is the on-ramp to the broker (DD-8), and it is
  the same shape as the project's bounded-agency / DAG-of-DAGs thesis — an agent
  gets AUTHORITY, not CUSTODY. Lead with it when this bundle is pitched or spec'd.
- **Caller identity (OQ-4) is the domino.** It is the one open question that can
  reshape the entire daemon, and without it the security story (`use` vs `read`)
  is unenforceable against a malicious local caller. Resolve it early — most other
  mechanism choices (policy binding, headless unlock, broker auth) hang off it.
- **Store first, integrate second (DD-1).** The recurring tie-breaker: when an OQ
  pits standalone polish against Backstop integration depth, DD-1 + DD-10 favor
  the standalone store. Backstop integration is a clean seam, not the foundation.
- **Reuse substrate, don't reinvent.** The capability model (OQ-8) plausibly rides
  BUNDLE-015's pack + adoption-record substrate; the audit log (OQ-10) should
  borrow shape from BUNDLE-017's provenance/audit patterns. "Integrate, don't
  build" applies to Stash's own internals too.

## References

- **BUNDLE-015 (pack-scaffolding-recipes)** — the pack recipe / adoption-record
  substrate the capability model (DD-9 / OQ-8) likely builds on: packs contribute
  declarative, adoption-gated artifacts; a Stash capability is plausibly one more.
- **BUNDLE-017 (recipe provenance ledger)** — the provenance / audit patterns the
  Stash audit log (OQ-10) should reuse rather than reinvent.
- **Bounded-agency / DAG-of-DAGs thesis (project memory)** — the read-vs-use
  distinction (DD-7) is the same shape: agents are granted authority to act, not
  custody of the secret. Stash is a concrete instance of the thesis.
- **backstop-packs `secrets` pack** — an UNRELATED concern (secret SCANNING via
  gitleaks → SARIF, not storage/brokering). Recorded to avoid naming/scope
  conflation (see Non-forks).
- **Cloud secret managers (GCP / AWS Secret Manager, HashiCorp Vault)** — the
  mature CLOUD prior art whose LOCAL equivalent this bundle fills. Reference for
  the policy/operation model and the secret-never-leaves broker pattern (DD-8).

## Version History

- 0.1.0 (2026-07-19): Initial bundle at `exploring`, authored from the founder's
  brain-dump. Ten founder-decided Draft Design Decisions (DD-1 product boundary /
  non-goals, DD-2 everything-is-a-resource + dot-namespaces, DD-3 filesystem-shaped
  CLI, DD-4 thin-CLI/long-lived-daemon architecture, DD-5 encryption-at-rest /
  OS-keychain anchor, DD-6 per-resource policy operation set, DD-7 the fundamental
  read-vs-use / custody-vs-authority distinction, DD-8 future execution broker,
  DD-9 capability-model Backstop integration hardcoding no integration knowledge,
  DD-10 guiding principles). Eleven OPEN mechanism questions (OQ-1 home/runtime,
  OQ-2 storage/encryption granularity, OQ-3 key custody/unlock UX, OQ-4 caller
  identity/authorization — load-bearing, OQ-5 policy shape + V1 `use` scope, OQ-6
  namespace semantics, OQ-7 type system, OQ-8 capability contract with Backstop,
  OQ-9 IPC protocol, OQ-10 audit log, OQ-11 onboarding/import), each with options
  and a non-binding author lean. Two non-forks recorded (`git stash` naming
  collision; the unrelated backstop-packs `secrets` scanning pack). Five
  provisional spec seeds. No OQ pre-resolved; no self-promotion — founder drives
  OQ resolution and promotion.
