---
title: "Adoption Telemetry"
number: BUNDLE-016
created: "2026-07-16"
schema_version: bundle/v2

bundle:
  name: adoption-telemetry
  version: "0.1.0"
  created: "2026-07-16"
  category: infrastructure

status:
  maturity: exploring

problem:
  summary: >
    Backstop has no adoption telemetry. If the first public release (DIR-001) ships
    without it, the growth curve starts not at zero but at "whenever we got around to
    it" — the first chapter of the adoption story is lost forever, because you cannot
    retroactively observe usage that was never recorded. Backstop is OSS flag-planting
    with capture via an agency: adoption numbers ARE the flag. The founder needs the
    ecosystem metrics a founder can brag about on a podcast — how many repos run
    backstop, how many packs exist — as evidence the strategy is working. The tension
    at the heart of this: backstop's brand is TRUST — deterministic enforcement for
    teams worried about slop. Telemetry built carelessly undercuts the exact trust the
    product sells, so anonymized, transparent, and respectful-by-design is a property
    of the claim itself, not a knob to be decided later.

  user_story: >
    As the founder, I want anonymized, transparent usage telemetry in place from the
    first public release, so that the adoption growth curve starts at day zero and I
    have real ecosystem numbers (repos using backstop, packs in existence) to point to
    as evidence the OSS flag-planting strategy is working — without ever making a
    trust-first, slop-averse audience feel surveilled by the tool that sells them
    determinism.
---

# Adoption Telemetry

## Current Thinking

This is a **seed** — a staked claim, not a design. The founder is reserving the problem
now because its defining constraint (day-one) cannot be satisfied retroactively; the
design work happens later, before or alongside DIR-001.

### Why this must exist from day one

Backstop's business model is OSS flag-planting with capture via an agency (see the
strategy memory). The flag is adoption: the number of repos running backstop and the
number of packs in the ecosystem are the evidence the strategy is working. Telemetry is
the only mechanism that makes those numbers real rather than anecdotal. And it is
uniquely time-sensitive: usage that happened before instrumentation existed is gone.
Ship the first release without telemetry and the adoption curve permanently begins at
"whenever we added it," not at launch. Retrofitting recovers the mechanism but never the
lost first chapter.

### The tension to hold honestly

Backstop sells TRUST — deterministic enforcement to teams who are worried about slop and
want a tool that is auditable and on their side. A telemetry system designed without care
is exactly the kind of thing that audience is allergic to, and it would undercut the
product's core promise. This is the real substance of the exploring phase: not "should we
have telemetry" (the claim answers yes) but "how do we collect adoption evidence in a way
that a trust-first, enforcement-selling product can stand behind without flinching." Every
open question below is downstream of that.

### Constraints (given — NOT open questions)

These are fixed by the claim. They frame the OQs; they are not up for debate here.

- **Day-one.** Telemetry lands with or before the first public release (couples to
  DIR-001). The pillar chain means there is genuine runway to design it right — this is a
  reason to start now, not a reason to rush a bad design in.
- **Anonymized + transparent, by design.** What is sent is documented exactly and is
  respectful by construction. This is non-negotiable and is a property of the claim, not
  something an OQ may trade away. OQs may decide the *mechanism* of anonymization, never
  whether to anonymize.
- **Thin-executor boundary untouched.** This is CLI infrastructure (how the binary
  reports usage), not check logic. It bakes in no language- or tool-specific knowledge and
  does not touch the packs-only enforcement spine. See CLAUDE.md first principle.

## Open Questions

These are genuinely open. The founder resolves them later — they are NOT pre-resolved
here, and resolving them is where the trust-vs-evidence tension gets adjudicated.

- **OQ-1 — Consent posture.** Opt-in, opt-out-with-first-run-notice, or build-flag? What
  do comparable dev tools do (Homebrew, Next.js, .NET, Deno all sit at different points on
  this spectrum) and which posture fits a trust-first *enforcement* tool specifically —
  where the audience's baseline suspicion is higher than for a general dev tool? Options:
  (a) opt-in (highest trust, lowest data — the growth curve undercounts); (b)
  opt-out with a clear, honest first-run notice (the Next.js/.NET lane); (c) compile-time
  build flag (distributors/enterprises can strip it entirely). No lean recorded — this is
  the load-bearing OQ and belongs to the founder.

- **OQ-2 — Collection scope.** Which events (gate runs? pack installs? `init`?), which
  dimensions (CLI version, OS/arch, pack names/versions?), and — stated as sharply as the
  inclusion list — what is categorically NEVER collected (repo names, file paths, source
  or findings content, project identity of any kind)? The never-list is as much a part of
  the design as the collect-list, and probably deserves to be written first.

- **OQ-3 — Anonymization mechanism.** Random per-install id, salted machine-derived hash,
  or no id at all (pure aggregate counts, no way to distinguish one install from another)?
  Each trades de-duplication accuracy (distinct-repo counts) against how much a wary user
  has to trust the anonymization claim. Plus retention: how long is anything kept? The
  *whether* of anonymization is a constraint above; only the *how* is open here.

- **OQ-4 — Ecosystem metrics source (possibly a different pillar).** "How many packs
  exist" may not be CLI telemetry at all — it could be registry-side observation
  (`pack add`/`install` hits, a GitHub topology scan of pack repos). Repo-count and
  pack-count may come from entirely different mechanisms. This OQ may reveal that part of
  the "ecosystem numbers" claim belongs to pack distribution / a future registry
  (BUNDLE-001 / BUNDLE-002) rather than to CLI telemetry, and should be split out if so.

- **OQ-5 — Transport and infra.** Where does it report, and what is the behavior when the
  network is unavailable, offline, or air-gapped (a silent no-op is a hard requirement — it
  must never block, slow, or error the CLI)? Is there a self-hosted / bring-your-own-endpoint
  story for enterprises who will not phone home to a vendor at all?

- **OQ-6 — Timing coupling to DIR-001.** Does telemetry land *inside* DIR-001's release
  scope, or as its own pre-release train that must merge before the first public tag?
  Bundling it into DIR-001 keeps "day-one" literally true but widens that directive's
  scope; a separate train keeps DIR-001 lean but adds an ordering dependency that could
  slip past the release.

## Version History

- 0.1.0 (2026-07-16): Initial seed at `exploring`. Staked the claim — anonymized,
  transparent adoption telemetry must land with or before the first public release
  (DIR-001), because the adoption story's first chapter cannot be recovered retroactively;
  plus ecosystem metrics (repo count, pack count) as flag-planting evidence. Recorded the
  trust-vs-evidence tension as the heart of the exploring phase, three given constraints
  (day-one, anonymized+transparent-by-design, thin-executor-untouched), and six genuinely
  open questions (consent posture, collection scope, anonymization mechanism, ecosystem
  metrics source, transport/infra, DIR-001 timing coupling). No OQs pre-resolved; no design
  decisions or spec seeds yet — those await founder-driven OQ resolution. Maturity stays
  `exploring`; promotion is founder-triggered.

## References

- **BUNDLE-003 (Onboarding Experience)** — `backstop init` is a likely telemetry event
  source (first-run) and shares the DIR-001 release coupling.
- **DIR-001 (Release Workflow)** — owns the first public release; day-one telemetry
  couples here (OQ-6). Currently the top backlog priority.
- **BUNDLE-001 / BUNDLE-002 (Pack distribution / registry)** — the likely home of
  ecosystem pack-count metrics if OQ-4 resolves toward registry-side observation.
- **Strategy memory: business model is agency-capture over OSS flag-planting** — the
  reason adoption numbers are load-bearing rather than vanity.
