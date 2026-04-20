---
title: "Pack Publishing Proxy — Native-Registry Fan-Out, Attestations, and Credential Management"
number: BUNDLE-002
created: "2026-04-09"
schema_version: bundle/v2

bundle:
  name: pack-publishing-proxy
  version: "0.1.0"
  created: "2026-04-09"
  category: feature

status:
  maturity: exploring

problem:
  summary: >
    Publishers want to ship SDK code alongside their pack's rules and
    scaffolds. SDK code lives in native registries (PyPI, npm, Go modules,
    crates) because consumers install via native toolchains. Today,
    publishers must manually publish to each registry, with no mechanical
    guarantee that the artifact passed backstop's validation gate before
    reaching consumers. A publishing proxy would let backstop validate and
    publish on their behalf, with signed attestations proving the artifact
    passed the gate. This bundle is extracted from BUNDLE-001 to allow the
    core pack lifecycle (declarative packs, git-native distribution,
    validation, composition, lightweight catalog) to proceed independently.

  user_story: >
    As a pack publisher shipping an SDK alongside my rules, I want to give
    backstop my registry credentials once and have it fan out validated SDK
    artifacts to native registries on my behalf, producing signed
    attestations that consumers can verify. As a consumer, I want to verify
    that the SDK I installed from npm/PyPI/crates came through backstop's
    gate, without trusting the native registry's security alone.

solution:
  approach: >
    Backstop acts as a publishing proxy (Buzzsprout model): authors grant
    scoped per-destination credentials, backstop runs the full pre-publish
    gate (validate + supply-chain + LLM-review + fixture-coverage +
    claim-coherence), fans out to native registries, and produces signed
    attestations stored in a transparency log. Consumers verify attestations
    locally via `backstop pack verify`. The proxy is an ecosystem-layer
    feature built on top of the core pack lifecycle (BUNDLE-001). It does
    not change how packs are authored, distributed via git, or composed —
    it adds a publication channel for SDK artifacts that live in native
    registries alongside the pack's git-native rules and scaffolds.
---

# Pack Publishing Proxy — Native-Registry Fan-Out, Attestations, and Credential Management

## Current Thinking

This bundle was extracted from BUNDLE-001 v0.5.0. The core pack lifecycle
(authoring, validation, git-native distribution, composition, lightweight
curated catalog) proceeds independently in BUNDLE-001. This bundle addresses
the ecosystem-layer features that require operational infrastructure beyond
git repos: publishing SDK artifacts to native registries, managing publisher
credentials, producing signed attestations, and propagating revocations
across registries backstop doesn't control.

### The publishing proxy model

Backstop is a publishing proxy, not a registry. The analogy is
Buzzsprout/Simplecast for podcasts: a fan-out gate to Spotify/Apple/Overcast,
not the thing listeners stream from. Authors grant scoped per-destination
credentials (sigstore identity, PyPI token, npm token, etc.); backstop
publishes SDKs to native registries on the author's behalf after the
pre-publish gate passes. Consumers always install from native registries
(`go get`, `npm install`, `pip install`, `cargo add`), never from backstop.

### Attestations as the consumer trust anchor

The pre-publish gate is invisible to consumers unless there is a
verification mechanism. Signed attestations are that mechanism: each
attestation is a signed statement binding (artifact hash, published
coordinates, backstop gate run, timestamp), stored in a transparency log.
Consumers verify via `backstop pack verify` or during `gate`; a valid
attestation matching the installed hash is the proof the artifact came
through backstop's gate. Without attestations, the pre-publish gate is
theater — "safe by default" has no mechanical grounding.

### Credential management is the hardest problem

Scoped per-registry tokens, encrypted at rest, rotatable, audited, never
logged, never in error messages. Any breach of backstop's credential store
is an ecosystem-wide supply chain attack. Whether credentials are
backstop-hosted, self-hosted by the publisher, or client-side only has
enormous implications for trust, operations, and attack surface.

## Draft Design Decisions

- **DD-1:** Backstop is a publishing proxy, not a registry. Authors
  grant scoped per-destination credentials (sigstore identity, PyPI
  token, npm token, etc.); backstop publishes SDKs to native registries
  on the author's behalf after the pre-publish gate passes. Model
  analogy: Buzzsprout/Simplecast for podcasts — a fan-out gate to
  Spotify/Apple/Overcast, not the thing listeners stream from.
  *Carried from BUNDLE-001 DD-22.*
- **DD-2:** Consumers never install from backstop. Install paths are
  always native — `go get`, `npm install`, `pip install`, `cargo add`.
  Backstop is not a distribution endpoint and never sits in the runtime
  or install path for SDK code. Bounds backstop's operational surface
  and legal exposure and sidesteps the "is backstop a package manager"
  trap. *Carried from BUNDLE-001 DD-23.*
- **DD-3:** The publisher owns the end-user support contract.
  Backstop's liability ends at "we validated what you gave us and
  pushed it where you told us to push it." Bug reports, support,
  versioning cadence, deprecation — all the publisher's responsibility.
  Backstop does not stand between publishers and their users.
  *Carried from BUNDLE-001 DD-24.*
- **DD-4:** Pre-publish is the single point of enforcement. The
  validate + supply-chain + LLM-review + fixture-coverage + claim-
  coherence gate runs at publish time. After the gate passes, the
  artifact lives in the native registry's trust model — backstop's job
  is done. Same gate machinery as BUNDLE-001 DD-3/DD-4/DD-7/DD-11/DD-12,
  scheduled at publish time. *Carried from BUNDLE-001 DD-25.*
- **DD-5:** Backstop produces signed attestations at publish time.
  Attestations are the mechanism that makes the pre-publish gate
  *verifiable at the consumer side*. Each attestation is a signed
  statement binding (artifact hash, published coordinates, backstop
  gate run, timestamp), stored in a transparency log. Consumers verify
  via `backstop pack verify` or during `gate`; a valid attestation
  matching the installed hash is the proof the artifact came through
  backstop's gate. Without attestations the pre-publish gate is
  invisible to consumers and "safe by default" has no mechanical
  grounding. *Carried from BUNDLE-001 DD-26.*
- **DD-6:** OQ-1 (signing story) strongly leans sigstore because the
  attestation model requires a transparency log, which sigstore ships.
  Leaving as an OQ only because the credential/identity model still
  needs thought (see OQ-1). Lean is strong enough to record here.
  *Carried from BUNDLE-001 DD-28.*

## Spec Seeds

These are provisional and will firm up as the open questions resolve.
Order is suggested implementation order, not commitment.

- **Credential Management and Scoped Token Storage** — storage model
  (backstop-hosted vs self-hosted vs client-side), encryption at rest,
  rotation, audit logging, per-registry scoping. The highest-risk
  component in this bundle.
- **Pre-Publish Gate Pipeline** — the same validation gate from
  BUNDLE-001, scheduled at publish time. Wiring `pack validate` +
  supply-chain subchecks + LLM review into a publish-time pipeline
  with pass/fail/retry semantics.
- **Native-Registry Fan-Out (Per-Ecosystem Adapters)** — per-ecosystem
  publish adapters for PyPI, npm, Go modules, crates. Each adapter
  handles the registry's specific auth, upload, and version semantics.
  Partial-publish compensation/retry strategy.
- **Attestation Production and Transparency Log** — attestation format
  (sigstore bundles vs in-toto vs SLSA), signing, transparency log
  storage and querying. The trust anchor for the entire proxy model.
- **Consumer-Side Attestation Verification (`backstop pack verify`)** —
  CLI command and gate integration for verifying installed SDKs against
  published attestations. Default-on vs opt-in behavior. Handling of
  SDKs with no attestation (pre-backstop or published outside proxy).
- **Yank/Revocation Propagation Across Native Registries** — per-
  ecosystem yank semantics, consumer-side detection of revoked
  artifacts, compensation for registries that don't support yank,
  interaction with BUNDLE-001's catalog revocation (DD-39).

## Open Questions

- **OQ-1: Credential storage model.** Backstop-hosted cloud service?
  Self-hosted pipeline authors run themselves? Client-side only with
  the author's machine doing the fan-out using locally-stored
  credentials? Each has very different trust, operational, and
  attack-surface implications. Probably the hardest question in this
  bundle. Options: (a) backstop-hosted with HSM-backed encryption —
  strongest trust anchor but backstop becomes a high-value target;
  (b) self-hosted — publishers run their own infra, backstop provides
  the pipeline but not the credential store; (c) client-side only —
  credentials never leave the author's machine, fan-out runs locally.
  Lean: unclear — each has deal-breaking tradeoffs.
  *Carried from BUNDLE-001 OQ-17.*

- **OQ-2: Attestation format and transport.** Sigstore bundles
  (cosign-style)? In-toto attestations? SLSA provenance? Stored where
  — backstop catalog only, git tag alongside source, native registry
  metadata slots (where available)? Format choice affects consumer
  verification UX and ecosystem interop. Lean: sigstore bundles
  (DD-6), but transport is open.
  *Carried from BUNDLE-001 OQ-18.*

- **OQ-3: Yank asymmetry across native registries.** Backstop can
  yank from its catalog by marking an attestation revoked, but the
  actual artifact lives in a native registry whose yank semantics vary
  wildly. Go modules are effectively immutable (proxy retention);
  PyPI allows yank but not delete; npm has `unpublish` with limits;
  Cargo allows yank. What signal reaches the consumer when backstop
  yanks but the native registry still serves the bits? Need to
  document per-ecosystem behavior and the consumer-side detection path.
  *Carried from BUNDLE-001 OQ-19.*

- **OQ-4: Consumer verification UX.** Is `backstop pack verify` run
  manually? Automatically on every `gate`? On CI only? On `pack
  install` only? What happens when an installed SDK has no attestation
  at all (pre-backstop or published outside the proxy) — hard error,
  warn, or allowed? The default matters a lot for adoption friction.
  Options: (a) default-on in `gate`, hard error on missing attestation;
  (b) default-on, warn on missing; (c) opt-in only. Lean: (b) —
  attestations without a verification mandate are theater (DD-5), but
  hard error on missing blocks adoption of pre-existing SDKs.
  *Carried from BUNDLE-001 OQ-20.*

- **OQ-5: Publisher self-hosting the gate.** Can a publisher run the
  pre-publish gate on their own infra and produce their own
  attestations, or must they go through backstop's hosted pipeline?
  Self-hosted preserves openness and avoids backstop being a single
  point of failure, but weakens the trust anchor (anyone can claim
  to have run the gate). Trade-off: decentralization vs trust
  strength. Options: (a) backstop-hosted only — strongest trust;
  (b) self-hosted with backstop-signed attestations (publisher runs
  gate, backstop counter-signs after verifying the run); (c) fully
  self-hosted with publisher-signed attestations (weaker trust).
  Lean: (b) — counter-signing preserves trust while allowing
  publisher infrastructure.
  *Carried from BUNDLE-001 OQ-21.*

- **OQ-6: Cross-registry publish consistency.** If a fan-out
  publishes successfully to PyPI but fails on npm, the pack is in an
  inconsistent state — some ecosystems have the new version, others
  don't. Transactional publish across heterogeneous registries is
  effectively impossible. Need a compensation/retry/acknowledgment
  story: roll back successful publishes, leave them and retry failed
  ones, or document partial-publish as an allowed state? Lean:
  retry failed + document partial-publish as allowed, because
  rollback is impossible on most registries.
  *Carried from BUNDLE-001 OQ-22.*

- **OQ-7: Attestation freshness and re-issuance.** An attestation is
  a snapshot of a gate run. If a CVE is later found in a transitive
  dep of the pack's rules or in a supply-chain scan rule unavailable
  at publish time, the attestation is still cryptographically valid
  but represents a stale judgment. Do attestations expire? Get
  re-issued on demand? Get invalidated by a separate "freshness feed"?
  Interacts with BUNDLE-001's known-bad list distribution (OQ-5) and
  revocation propagation (OQ-13). Lean: attestations don't expire
  cryptographically but carry a "validated-as-of" timestamp;
  consumers can set a maximum staleness threshold.
  *Carried from BUNDLE-001 OQ-23.*

## Sharp Edges / Risks

- **Credential handling is a juicy attack target.** Scoped per-registry
  tokens, encrypted at rest, rotatable, audited, never logged, never
  in error messages. Any breach of backstop's credential store is an
  ecosystem-wide supply chain attack. Must be treated as the single
  highest-risk component.
- **Cross-ecosystem inconsistency is a permanent state, not a bug.**
  Different native registries have different version semantics, yank
  support, caching behavior, and retention policies. The pack model
  has to treat "this SDK version is in ecosystem A but not yet in
  ecosystem B" as a legitimate state the lockfile and gate know how
  to represent.
- **Attestations without a verification mandate are theater.** If
  consumers never actually verify attestations (bad UX, or `gate`
  not enforcing them by default), the whole pre-publish gate loses
  its teeth. Verification must be as mechanical and default-on as
  the rest of the gate.

## Notes / Ideas

- This bundle depends on BUNDLE-001 for the core pack model, validation
  gate, and composition semantics. Nothing here changes how packs are
  authored or distributed via git — it adds a publication channel for
  SDK artifacts alongside the git-native pack.
- The credential storage question (OQ-1) may determine whether this
  bundle is viable as a backstop-hosted service or only as a self-hosted
  pipeline. That decision has major implications for the project's
  operational model.
- Family membership verification (BUNDLE-001 sharp edge) is stronger
  under the proxy model: attestations bind family members to the same
  publisher identity, so provenance is verifiable even if behavioral
  equivalence is not.

## Version History

- 0.1.0 (2026-04-09): Extracted from BUNDLE-001 v0.5.0. Publishing
  proxy, attestations, credential management, and native-registry
  fan-out moved here to let the core pack lifecycle proceed
  independently. 6 DDs carried from BUNDLE-001 (DD-22, DD-23, DD-24,
  DD-25, DD-26, DD-28 -> renumbered DD-1..DD-6). 7 OQs carried from
  BUNDLE-001 (OQ-17, OQ-18, OQ-19, OQ-20, OQ-21, OQ-22, OQ-23 ->
  renumbered OQ-1..OQ-7). 3 sharp edges carried over. 6 spec seeds
  identified. Maturity: exploring.

## References

- BUNDLE-001: Pack Distribution, Verification, and Review (parent —
  core pack lifecycle, git-native distribution, validation, composition,
  lightweight curated catalog)
- BUNDLE-001 DD-22, DD-25, DD-26, DD-28 (marked deferred, pointing here)
- BUNDLE-001 OQ-17, OQ-18, OQ-19, OQ-22, OQ-23 (resolved as deferred,
  pointing here)
- BUNDLE-001 DD-3/DD-4/DD-7/DD-11/DD-12 (gate machinery reused at
  publish time)
