---
title: "Onboarding Experience — Zero to First Value"
number: BUNDLE-003
created: "2026-04-09"
schema_version: bundle/v2

bundle:
  name: onboarding-experience
  version: "0.9.0"
  created: "2026-04-09"
  updated: "2026-08-12"
  category: feature

status:
  maturity: defined

problem:
  summary: >
    Backstop has no onboarding path. A consumer must hand-create backstop.yml,
    figure out the artifact-directory layout, know which packs to add, and their
    first run either errors on missing SDLC directories or produces a wall of
    violations. Two profiles onboarded by hand (a full-SDLC greenfield TypeScript
    repo and a pack-only consumer) both reached a clean gate PASS — but only after
    a human absorbed a ranked list of sharp edges the tool gave no help with.
    Every one of those sharp edges is adoption friction. The objective is minimal
    time from having the binary to first value derived: install, one command,
    "this caught something real in my code" — without hand-writing config,
    guessing the layout, or being told the whole codebase is broken.

  user_story: >
    As a tech lead evaluating backstop for my team, I want to run one command in
    my project and see useful observations about my codebase in minutes — without
    hand-writing config, without guessing where artifacts go, and without the tool
    hard-erroring on directories I never created. I want the tool to feel like it
    understands my project, not like it is judging it.

solution:
  approach: >
    A single `backstop init` command that takes a consuming project from zero to
    first value, scaffolded for one of two validated profiles: full-SDLC greenfield
    (adopting backstop's artifact pipeline) or pack-only (adopting packs without the
    SDLC artifacts). Init installs an opinionated base bundle (omakase — bare `init`
    is the full bundle you subtract from via flags) and does ZERO language/framework
    detection; languages enter only via explicit `pack add`. It writes the
    profile-correct backstop.yml, scaffolds the `.backstop/`-rooted artifact layout,
    emits a canonical .gitignore, seeds a local baseline, verifies the toolchain
    actually runs, wires CI by default via a recipe pack, and captures the first gate
    result as a baseline framed as observation ("here's what we noticed"), not
    judgment. Post-init, default scope is diff-based — you are accountable for new
    patterns, not inherited ones. The init algorithm is specified by transcribing the
    two validated hand-onboarding checklists, not by inventing a flow; each ranked
    sharp edge becomes either an init guardrail or a `backstop doctor` diagnostic.
    Detection, framework recognition, and CI-platform knowledge live in packs as data,
    never in core (a HARD INVARIANT).
    Binary + system-toolchain acquisition and baseline generation are owned elsewhere
    (DIR-001, BUNDLE-007) and are out of scope here.

requirements:
  - id: REQ-001
    version: "1.0.0"
    text: >
      `backstop init` must take a consuming project from "the binary is present" to
      first useful output in a single command, with no manual step required between
      the two (DD-1).
  - id: REQ-002
    version: "1.0.0"
    text: >
      Bare `backstop init` must install the full opinionated base bundle without
      prompting for any input, and must expose subtraction flags (`--only <cap>` /
      `--no-<cap>`) as the only way to narrow that default set. Because it never
      prompts, init must run identically in an interactive terminal and in a headless
      or CI environment (OQ-2, DD-2).
  - id: REQ-003
    version: "1.0.0"
    text: >
      Init must generate the `backstop.yml` correct for the selected profile without
      the consumer hand-writing policy. The full-SDLC greenfield profile gets a minimal
      config (`project:` plus the artifact pipeline enabled). The pack-only profile must
      additionally set `enforcement.policy` `level: off` for every SDLC dimension —
      `test_verification`, `coverage_threshold`, `contract_signature`,
      `test_substantiveness`, `artifact_status_drift` — because those dimensions
      hard-error on a missing `specs/` directory rather than skipping (DD-2).
  - id: REQ-004
    version: "1.0.0"
    text: >
      Init must scaffold the `.backstop/`-rooted artifact layout in consumer repos,
      leaving only `backstop.yml` / `backstop.lock` and `.backstop/` visible at the
      repo root. backstop-core's own root-level artifact layout is a recognized
      framework exception and must not be treated as a layout violation (OQ-1).
  - id: REQ-005
    version: "1.0.0"
    text: >
      Init must emit one canonical `.gitignore` covering the full set the validated
      onboarding produced — `.backstop/packs/`, `.backstop/baseline.json`,
      `.backstop/pack-config-provenance.json`, plus every path the installed packs
      declare as generated output — so that ignore contents no longer diverge between
      onboarded repos (DD-7).
  - id: REQ-006
    version: "1.0.0"
    text: >
      Init must run `git init` only when the target directory has no `.git`, and must
      leave an existing repository's git state untouched (DD-11).
  - id: REQ-007
    version: "1.0.0"
    text: >
      Re-running init must converge and never clobber: it detects only
      backstop-neutral facts (presence of `.git`, of `backstop.yml`, of the artifact
      directories), adds only what is missing, overwrites no existing consumer file,
      and reports its findings purely in backstop terms (OQ-6, DD-11, DD-14).
  - id: REQ-008
    version: "1.0.0"
    text: >
      Init must perform ZERO language, framework, or CI-platform detection. No
      language, framework, or CI-platform name may appear in core CLI code on the init
      path; that knowledge lives in packs as data and reaches init only through pack
      manifests and recipes. The `backstop/self` pack enforces this (DD-13, OQ-5, OQ-6).
  - id: REQ-009
    version: "1.0.0"
    text: >
      Init must apply pack-supplied recipes through a single generic mechanism — copy
      a template to the path the recipe itself declares — and must not interpret,
      rewrite, or language-specialize template content. Every language-specific
      artifact init produces (a first source file, toolchain config) must originate in
      a pack recipe, never in core; init adds the backstop layer only and never
      reimplements what an ecosystem scaffolder (`create-next-app`, `cargo new`, …)
      already does (DD-12, DD-7, DD-14).
  - id: REQ-010
    version: "1.0.0"
    text: >
      Languages must enter a consumer project only via an explicit `pack add`;
      multi-language projects are simply multiple packs. Init must have no concept of
      a "primary language" and must never select packs on a project's inspected
      identity (OQ-5, DD-11).
  - id: REQ-011
    version: "1.0.0"
    text: >
      Init must verify the toolchain actually RUNS by executing the pack-declared
      test/compile entrypoint once and confirming success, rather than inferring
      health from package-manager configuration or exit code. Evidence: pnpm 11.11
      exited nonzero with `ERR_PNPM_IGNORED_BUILDS` while vitest ran fine — trusting
      the package manager would have produced a false init failure (DD-6).
  - id: REQ-012
    version: "1.0.0"
    text: >
      Init must seed a gitignored local baseline at `.backstop/baseline.json` so a
      solo or remoteless consumer gets the ratchet from day zero, and that local
      baseline must be superseded without migration by the CI-generated baseline
      BUNDLE-007 owns once a team adopts it (OQ-3, DD-10).
  - id: REQ-013
    version: "1.0.0"
    text: >
      The `baseline_comparison` message emitted on a remoteless repository must be
      self-consistent — it must not simultaneously claim a baseline is required and
      that none can exist (OQ-3).
  - id: REQ-014
    version: "1.0.0"
    text: >
      Init must capture the first gate run as the baseline and present it as
      observation, not failure: findings grouped by category with counts, phrased as
      what was noticed, and an exit code of 0 for "baseline captured" rather than 1
      for "violations found" (DD-3).
  - id: REQ-015
    version: "1.0.0"
    text: >
      After init, an unflagged gate run must be diff-scoped — evaluating changed files
      against the seeded baseline so inherited patterns are separated from introduced
      ones — with full-codebase evaluation available on demand (DD-4).
  - id: REQ-016
    version: "1.0.0"
    text: >
      Init must wire CI by default by applying the selected variant from a CI recipe
      pack that holds per-platform templates as data (`--ci github` as the default
      variant, `--no-ci` to opt out). No `ci` verb may be added: the generated job
      chains the existing platform-agnostic commands (`pack install` → `baseline pull`
      → `gate`), which is what gives local and CI runs identical semantics (OQ-7,
      DD-13).
  - id: REQ-017
    version: "1.0.0"
    text: >
      When no CI recipe pack is installed, the CI step of init must fail LOUDLY with
      guidance naming the pack to add, while every other init step still completes.
      There is no baked per-platform fallback, so silent success is unrepresentable
      (OQ-7).
  - id: REQ-018
    version: "1.0.0"
    text: >
      Init must install its omakase base through portable git-ref pack references, so
      the `backstop.lock` it commits contains no machine-specific paths. Local-path
      packs a consumer adds afterward are restored on the same machine from a
      gitignored local-provenance cache, and promoting one to shareable stays an
      explicit git-ref step (OQ-4).
  - id: REQ-019
    version: "1.0.0"
    text: >
      Init's happy path must execute the transcribed hand-onboarding sequence
      (scaffold via recipe → write profile-correct `backstop.yml` → create artifact
      dirs → write `.gitignore` → add each pack → run the gate), and the acceptance
      bar is the outcome that sequence already achieved by hand: on a fresh repo, init
      followed by `backstop gate` reaches PASS with zero violations, for both the
      full-SDLC greenfield and the pack-only profile (DD-8, DD-2).
  - id: REQ-020
    version: "1.0.0"
    text: >
      `backstop doctor` must exist as the diagnostic command for a setup that is off,
      with one check per ranked sharp edge from the hand-onboarding write-ups, and
      init must name it in the guidance it prints whenever a step it cannot complete
      is diagnosable (DD-8 corollary, DD-9).
  - id: REQ-021
    version: "1.1.0"
    text: >
      `backstop version` must stamp the build commit and build date, so a stale binary is
      identifiable on sight. Reporting a bare `dev` as the VERSION STRING of a non-release
      build is CORRECT and must be preserved, not eliminated: `cmd/backstop/version.go`
      deliberately refuses to report anything release-shaped for a locally built binary —
      it rejects `(devel)`, any `+` build metadata (so `v0.11.0+dirty` is a tag plus
      uncommitted changes, not a release), and Go pseudo-versions — precisely so a local
      build cannot pretend to be an official release. What must become identifiable is the
      BUILD, not the release: `dev` plus a commit and a build date distinguishes a fresh
      dev build from a stale one, which is the diagnosability the dogfood incident actually
      needed (DD-9 as corrected 2026-08-11).
    versions:
      - version: "1.0.0"
        text: >
          `backstop version` must stamp the build commit and build date, and must never
          report a bare `dev`, so a stale binary is identifiable on sight (DD-9).
      - version: "1.1.0"
        text: >
          `backstop version` must stamp the build commit and build date, so a stale binary is
          identifiable on sight. Reporting a bare `dev` as the VERSION STRING of a non-release
          build is CORRECT and must be preserved, not eliminated: `cmd/backstop/version.go`
          deliberately refuses to report anything release-shaped for a locally built binary —
          it rejects `(devel)`, any `+` build metadata (so `v0.11.0+dirty` is a tag plus
          uncommitted changes, not a release), and Go pseudo-versions — precisely so a local
          build cannot pretend to be an official release. What must become identifiable is the
          BUILD, not the release: `dev` plus a commit and a build date distinguishes a fresh
          dev build from a stale one, which is the diagnosability the dogfood incident actually
          needed (DD-9 as corrected 2026-08-11).
        correction: >
          CORRECTION (2026-08-11, v0.8.1, founder-ruled). The v1.0.0 clause "must never report a
          bare `dev`" was WRONG and is withdrawn: it contradicted a deliberate anti-spoofing
          design already shipped in `cmd/backstop/version.go`, whose whole purpose is that a
          non-release build reports `dev` rather than something that looks like a release.
          Founder ruling, verbatim: "it can be dev for a local build if that's standard
          practice." Verified against the code and the running binary the same day: `./bin/backstop
          version` reports `backstop version dev`, and `resolveVersion` returns `dev` for every
          input that is not an injected goreleaser stamp or a strictly released module version.
          Nothing is lost by the withdrawal — the underlying stale-binary concern is carried by
          the surviving build-commit/build-date clause above, by REQ-022 (which names the stale
          binary as the cause instead of surfacing the downstream engine error — the exact
          dogfood failure mode), and by REQ-028 (which stamps producing-binary identity onto every
          result so a stale green can be quarantined). No NEW mechanism was ordered by the ruling
          and none is invented here.
  - id: REQ-022
    version: "1.0.0"
    text: >
      `pack add` and `gate` must compare the binary's capability against the features
      the pack manifest declares it requires, and on skew must fail with a diagnostic
      naming the binary as older than the pack requires — instead of surfacing the
      downstream engine error (e.g. `declared stdout_artifact ... not produced`) that
      gives no hint the binary is the cause (DD-9).
  - id: REQ-023
    version: "1.0.0"
    text: >
      `backstop doctor` must expose the REQ-011 toolchain-execution check as a
      standalone, re-runnable diagnostic with actionable remediation, so a consumer
      can diagnose a broken toolchain without re-running init (DD-6, doctor seed).
  - id: REQ-024
    version: "1.0.0"
    text: >
      `backstop doctor` must check the installed runtime/toolchain version against the
      stack policy declared by the installed packs and warn on deviation. The policy
      values are pack data — no runtime version may be baked into core. Evidence: the
      dogfood machine ran only non-LTS Node against a documented Node-LTS stack
      decision and nothing warned (doctor seed, DD-13).
  - id: REQ-025
    version: "1.0.0"
    text: >
      `backstop doctor` must validate a consumer repo's artifact layout against the
      canonical `.backstop/`-rooted layout and report each deviation with the path it
      expected (OQ-1, doctor seed).
  - id: REQ-026
    version: "1.0.0"
    text: >
      The schema cohort a binary reports must be derived from the CONTENT of its
      embedded schemas, not from their names or count, so that any change to an
      embedded schema's bytes changes the cohort identifier. Evidence (verified
      2026-08-11): `computeCohortID` (`cmd/backstop/root.go`) builds the cohort string
      from schema file PATHS alone — `11-schemas[adr/v1,...,spec/v1]` — so the in-place
      revision of `bundle/v2` that BUNDLE-014 shipped (adding the `Draft`-prefixed
      section names, `text:` in place of `statement:`, per-REQ `version:`, and
      `bundle.updated`) left the reported cohort byte-identical. A name-derived cohort
      cannot detect the skew that produced the false green in REQ-027, which is why
      content-derivation is a requirement and not an implementation detail (DD-9).
  - id: REQ-027
    version: "1.0.0"
    text: >
      `backstop artifact validate` and `backstop gate` must assert the validating
      binary's schema cohort against each artifact's declared `schema_version` and MUST
      REFUSE to report green when they cannot prove the cohort covers the schema
      features the artifact uses. A validator too old — or too new — for an artifact's
      schema must fail loudly; "cannot determine" is a failure, never a pass. Evidence:
      BUNDLE-001 in bclabs-portal was authored, promoted to `defined`, and revised five
      times (0.2.0 → 0.2.5) across multiple sessions with `artifact validate` reporting
      GREEN throughout, while violating the then-current `bundle/v2` schema in 41
      places; the nonconformance surfaced instantly under a current binary. The only
      schema cross-check performed today is `schema_version` PREFIX vs artifact type
      (`pkg/validate/{bundle,spec,issue,adr}.go`) — nothing compares schema revisions.
      This guard belongs in core's own validate/gate rather than any single harness,
      because the skew bit in a different toolchain than the one that caught it (DD-9).
  - id: REQ-028
    version: "1.0.0"
    text: >
      Every `gate` and `artifact validate` result, in both human and `--json` form, must
      record the version and schema cohort of the binary that produced it, so a stored
      or forwarded result can be quarantined rather than projected as truth when it was
      produced by a validator that could not have caught the violation. Evidence: gate
      results already stamp `GitSHA` and `GeneratedAt` (`cmd/backstop/gate.go`) but carry
      no binary identity, so a recorded green is indistinguishable from a green a stale
      validator could not have withheld (DD-9).
  - id: REQ-029
    version: "1.0.0"
    text: >
      Artifact discovery must resolve the artifact root from configuration and apply
      that one resolution uniformly across gate status resolution, validation reference
      resolution, and scaffolding — so a consuming repo can hold the whole artifact
      chain under `.backstop/` and be discovered, while backstop-core keeps the root
      layout (the OQ-1 framework exception) by configuring the repo root. Evidence
      (verified 2026-08-11, unchanged since the 2026-07-17 write-up): three independent
      hardcodings of the root layout — `pkg/gate/artifact_status.go` joins projectRoot
      with `specs`/`bundles`/`issues`/`directives`/`plans`; `pkg/validate/resolved_by.go`
      maps BUNDLE→`bundles`, SPEC→`specs`, ISSUE→`issues`, DIR→`directives`;
      `pkg/scaffold/scaffold.go` scaffolds into root `specs`/`issues`/`directives`/
      `bundles`. No `artifact_root` key exists in the `backstop.yml` schema or loader.
      REQ-004 is unimplementable — and actively harmful — until this lands (OQ-1).
  - id: REQ-030
    version: "1.0.0"
    text: >
      Artifact discovery must name the artifact root it actually scanned in gate output,
      and must fail loudly when a configured artifact root does not exist on disk rather
      than walking absent directories and reporting a passing dimension. An artifact root
      that EXISTS but is empty remains a legitimate pass — the validated greenfield
      profile reached gate PASS with empty artifact dirs and that outcome must not
      regress. Evidence: backstop-runtime already placed `.backstop/bundles/` while its
      `specs/` stayed at root; because the binary only discovers `projectRoot/bundles`,
      those bundles are almost certainly not gated and nothing says so — the same
      false-green family as REQ-027 (OQ-1).
  - id: REQ-031
    version: "1.0.0"
    text: >
      Init must never scaffold an artifact layout the installed binary's discovery
      cannot resolve. Until REQ-029 ships, init must place consuming-repo artifacts at
      the root layout discovery can see and state that the `.backstop/`-rooted layout is
      not yet supported; REQ-004 is gated on REQ-029 and must not be implemented ahead of
      it. Rationale: a repo that adopts the intended layout early gets its whole artifact
      chain SILENTLY UNDISCOVERED, which is strictly worse than the divergence REQ-004
      exists to end (OQ-1, REQ-029).
  - id: REQ-032
    version: "1.0.0"
    text: >
      Init must materialize and install the project-level dependencies the installed
      packs' engines invoke but do not vendor, taking both the dependency set and the
      install invocation from pack recipe data, and must do so before the REQ-011
      toolchain-execution check runs. Neither dependency names nor package-manager
      invocations may appear in core (DD-13). Evidence: step 1 of the pack-only
      hand-onboarding was installing `eslint @eslint/js typescript-eslint typescript
      vitest @vitest/coverage-v8` by hand — the pack ships rules and config, not
      packages — and without that step REQ-011's verification cannot pass. Distinct from
      DIR-001's scope: these are project-scoped dependencies a pack declares, not the
      host binary or system runtime (DD-10, DD-12).
  - id: REQ-033
    version: "1.0.0"
    text: >
      The pack-only profile must not silently forfeit coverage enforcement. Because
      REQ-003 sets `coverage_threshold` to `level: off` for a consumer that has no specs
      to source thresholds from, init's pack-only profile must instead wire a
      spec-independent coverage floor (a global and/or per-glob minimum under
      `enforcement`) over the coverage records the installed packs already produce.
      Evidence: `ts-coverage` emits per-file coverage records, but `coverage_threshold`
      sources its thresholds from SDLC specs, so the pack-only consumer ends up holding
      coverage data with no "fail under N%" knob. The knob itself is a core
      enforcement-policy capability this bundle CONSUMES and does not design (see Out of
      Scope / Dependencies) (DD-2).
---

# Onboarding Experience

## Current Thinking

### Provenance: dogfood-grounded

This bundle began (0.1.0, April 2026) as a thought exercise — analyzing onboarding
from a tech lead's perspective. It is now **dogfood-grounded**: the requirements for
`backstop init` and `backstop doctor` derive from onboarding two real repos by hand,
one write-up per profile (see References). Those two write-ups are the empirical
requirements corpus. Both currently live as untracked field notes in other repos, so
the load-bearing substance is lifted into the design decisions below (especially DD-8)
rather than left to survive by citation alone.

### Grounding pass, 2026-08-11 — requirements re-checked against the real dogfood evidence

The 25 requirements authored on 2026-08-10 were derived from this bundle's own resolved OQs
and DDs. On 2026-08-11 they were re-checked line by line against the two hand-onboarding
source documents themselves, rather than against this bundle's summary of them:

- `~/src/projects/backstop-packs/BACKSTOP-INIT-REQUIREMENTS.md` — the **pack-only** profile.
- `~/src/projects/backstop-packs/BACKSTOP-INIT-DOGFOOD-FULL-SDLC.md` — the **full-SDLC
  greenfield** profile (bclabs-portal, 2026-07-12, with a second version-skew instance
  appended 2026-07-16 and the artifact-layout founder ruling appended 2026-07-17).

Note the second document is the same corpus the References section lists as
`bclabs-portal/docs/dogfood/init-sharp-edges.md`; it now lives in `backstop-packs/`
alongside its pack-only sibling. Both remain untracked field notes — this bundle is still
their durable home.

Fifteen concrete findings were checked. Ten were already covered by REQ-001..025. Two were
ruled out of scope with reasons recorded below (semgrep rule-id readability; `artifact new`
number reservation). Eight new requirements (REQ-026..033) closed the remaining gaps, and the
code state behind each was verified against HEAD on 2026-08-11 rather than trusted from notes
written in July — `computeCohortID` is still path-derived, all three artifact-discovery
hardcodings are still present, no `artifact_root` config key exists, and gate results still
carry no producing-binary identity. The two serious ones (schema-cohort false green,
artifact-root silent undiscovery) are generalized into DD-15.

One finding was RETIRED by its own successor evidence: the pack-only document's recommendation
that init write `onlyBuiltDependencies: [esbuild]` into `pnpm-workspace.yaml` was superseded
three weeks later when pnpm 11.11 ignored that key entirely and the failure proved cosmetic.
That reversal is precisely the argument for REQ-011 (verify the toolchain RUNS) — and writing
a package-manager-specific key from core would violate DD-13 regardless.

### The current state

There is no `backstop init`. A consumer must:
1. Hand-create `backstop.yml` — and know the profile-correct contents (a pack-only
   consumer must set `enforcement.policy` `level: off` for every SDLC dimension or the
   gate hard-errors on a missing `specs/` directory).
2. Decide an artifact-directory layout with no canonical answer (live repos already
   diverge — see OQ-1).
3. Know which packs to add and add each one.
4. Write a `.gitignore` that matches what the packs and gate actually emit.
5. Run `backstop gate` — and debug whatever it surfaces with no diagnostic help.

Both profiles reached gate PASS by hand, which proves the flow exists; each step above
is a sharp edge the tool should absorb.

### The target experience

1. Obtain the binary — owned by DIR-001, out of scope here.
2. `backstop init` — installs the omakase base (prompt-free; subtract via flags),
   writes profile-correct config, scaffolds the `.backstop/`-rooted layout, emits the
   canonical `.gitignore`, seeds a local baseline, verifies the toolchain runs, and
   wires CI by default. Zero language/framework detection — languages arrive via
   explicit `pack add`.
3. Fix one thing, run a diff-scoped check, watch it go green. That is first value.

### Key principles (product thesis)

- **Baseline as observation, not judgment.** If the first thing backstop says is "427
  violations, exit 1," the tech lead hears "your code sucks." Init captures the first
  run as a baseline and presents it as "here's what we noticed," grouped by category —
  same data, opposite emotional response.
- **Diff-scoped by default post-init.** After init, an unflagged check looks only at
  changed files. The baseline records what existed at init so the gate separates
  inherited patterns from introduced ones. You are accountable for new code from day one.
- **Two profiles, not one generic flow.** The single most important empirical finding:
  onboarding is not one-size-fits-all. Full-SDLC and pack-only consumers need
  materially different config, and picking the wrong one silently produces the
  hard-error experience (see DD-2).

## Draft Requirements

REQ-001 through REQ-033 are carried in the frontmatter `requirements` block. REQ-001..025 are
each derived from an already-resolved OQ, an existing DD, or a spec seed in this bundle.
REQ-026..033 were added in 0.8.0 by the grounding pass described under Current Thinking —
each is a finding from the two hand-onboarding write-ups that none of the first 25 covered.

They partition cleanly across **three** spec seeds:

| Seed | Requirements |
|---|---|
| `backstop init` | REQ-001 – REQ-019, REQ-031, REQ-032, REQ-033 |
| `backstop doctor` | REQ-020 – REQ-025 |
| Trustworthy-green guards (core) | REQ-026 – REQ-030 |

The third seed is new in 0.8.0. It exists because REQ-026..030 are changes to core's
validate / gate / discovery paths that init and doctor both DEPEND on but neither OWNS —
folding them into either seed would have made that seed's scope dishonest. See Spec Seeds.

### `backstop init` (REQ-001 – REQ-019)

- **Shape of the command** (REQ-001, REQ-002): one command from binary to first value with no
  manual step in between (DD-1); omakase base installed prompt-free with subtract-via-flags,
  which is also what makes init headless/CI-safe (OQ-2).
- **Profile correctness** (REQ-003): the headline two-profile fork — full-SDLC greenfield gets
  a minimal config; pack-only additionally gets `enforcement.policy` `level: off` on all five
  SDLC dimensions, because they hard-error rather than skip on a missing `specs/` (DD-2).
- **What init writes** (REQ-004, REQ-005): the `.backstop/`-rooted consumer layout with
  backstop-core's root layout as the explicit framework exception (OQ-1), and one canonical
  `.gitignore` that ends the divergence observed across onboarded repos (DD-7).
- **Thin-executor boundary** (REQ-008, REQ-009, REQ-010): zero language/framework/CI-platform
  detection, no such literal in core init code, recipes applied by a generic
  copy-template-to-declared-path mechanism, and languages entering only via explicit
  `pack add` (DD-13, DD-12, OQ-5). REQ-009 is also where DD-14 lands: init adds the backstop
  layer, ecosystem scaffolders own the project.
- **Idempotency** (REQ-006, REQ-007): `git init` only when there is no `.git`; re-init
  converges, never clobbers, and stays framework-blind (DD-11, OQ-6).
- **Ground truth over configuration** (REQ-011): init executes the pack-declared toolchain
  entrypoint once rather than trusting a package manager's exit code (DD-6).
- **Baseline and scope** (REQ-012, REQ-013, REQ-014, REQ-015): seed a gitignored local
  baseline day-zero (OQ-3), fix the self-contradictory remoteless message (OQ-3), frame the
  first run as observation with exit 0 (DD-3), and make post-init default scope diff-based
  (DD-4).
- **CI** (REQ-016, REQ-017): wired by default via a CI recipe pack whose templates are data,
  no `ci` verb (the job chains `pack install` → `baseline pull` → `gate`), and a loud but
  non-blocking failure when no CI pack is present (OQ-7).
- **Lock portability** (REQ-018): init's own installs use portable git-refs so the committed
  lock carries no machine-specific paths; the gitignored local-provenance cache covers
  locally-added packs (OQ-4).
- **Acceptance** (REQ-019): the transcribed hand-onboarding sequence is the happy path, and
  the bar is the outcome it already produced by hand — init then `backstop gate` reaching PASS
  with zero violations on a fresh repo, for both profiles (DD-8).
- **Layout sequencing guard** (REQ-031, added 0.8.0): init must not scaffold a layout
  discovery cannot resolve. REQ-004 is GATED on REQ-029; until that lands, init keeps
  consuming-repo artifacts at root and says so. Without this, REQ-004 as written would ship
  the exact silent-undiscovery failure the write-up warned against (DD-15).
- **Pack-declared project dependencies** (REQ-032, added 0.8.0): init installs the devDeps a
  pack's engines invoke but do not vendor, from pack recipe data, before REQ-011 runs. This
  closes a hole BETWEEN two existing requirements — REQ-009 writes the toolchain config and
  REQ-011 executes the toolchain, and nothing in between installed anything.
- **Pack-only coverage floor** (REQ-033, added 0.8.0): REQ-003 turns `coverage_threshold`
  off for the pack-only profile, which would otherwise leave that consumer holding coverage
  records with no threshold to fail against. Init wires a spec-independent floor instead.

### `backstop doctor` (REQ-020 – REQ-025)

- **The command** (REQ-020): one check per ranked sharp edge; init points at it rather than
  absorbing diagnosis (DD-8 corollary).
- **Version-skew diagnosability** (REQ-021, REQ-022): a build-identity stamp (commit + build
  date) alongside the version string, and a binary-vs-pack capability comparison in `pack add`
  and `gate` that names the stale binary instead of surfacing the downstream engine error. This
  was the highest-pain sharp edge in the 2026-07-12 dogfood (DD-9).
  **CORRECTION (2026-08-11, v0.8.1, founder-ruled):** the phrase this bullet previously used —
  "a real version stamp instead of bare `dev`" — is withdrawn. Bare `dev` IS the real answer for
  a non-release build. See the REQ-021 v1.1.0 correction and DD-9 mechanic (a).
- **Standalone diagnostics** (REQ-023, REQ-024, REQ-025): re-run the toolchain-execution check
  outside init (DD-6), check runtime version against pack-declared stack policy (the
  unenforced Node-LTS observation), and validate the artifact layout against the canonical
  `.backstop/` root (OQ-1).

### Trustworthy-green guards (REQ-026 – REQ-030) — new in 0.8.0

The doctor requirements above make version skew **diagnosable**. These make it **unable to
certify**. The distinction is the whole point of DD-15: a misleading error stops you, but a
false green propagates — promotions, hand-offs, and derived specs and plans accrue on a
foundation that was never actually validated.

- **Cohort integrity** (REQ-026, REQ-027, REQ-028): the reported cohort must be derived from
  schema CONTENT (today it is derived from schema paths, so an in-place `bundle/v2` revision
  is invisible); `validate` and `gate` must refuse green when they cannot prove their cohort
  covers the artifact's schema; and every result must carry the producing binary's identity so
  a stale green can be quarantined downstream. REQ-021/REQ-022 sit under `doctor` because they
  are diagnostics a human runs; these three are guards that run unasked, on every validation.
- **Artifact-root resolution** (REQ-029, REQ-030): discovery resolves the artifact root from
  config through ONE resolver shared by gate, validate, and scaffold; discovery names the root
  it scanned and fails loudly on a configured root that is absent — while an existing-but-empty
  root stays a pass, preserving the validated greenfield outcome.

### Deliberately NOT requirements here

- Baseline generation mechanics, binary distribution, and system-toolchain acquisition —
  owned by BUNDLE-007 and DIR-001 (DD-10). REQ-012 and REQ-013 are init's *consumption* of
  the baseline, not its machinery.
- The pack-recipe capability itself, the concrete CI recipe pack, and the gitignored
  local-provenance lock-schema change — consumed by REQ-009/REQ-016/REQ-018 but designed
  elsewhere (see Out of Scope / Dependencies). The recipe capability remains a BLOCKING
  dependency for the init spec.
- The cascading `coverage_threshold` misdiagnosis when tests fail — recorded under
  Observations as a pack-side messaging concern, not init scope.
- Registry-era pack auto-detection — explicitly dissolved by OQ-5.
- The `enforcement.coverage.min_pct` knob ITSELF — a core enforcement-policy capability.
  REQ-033 requires init to WIRE a spec-independent coverage floor for the pack-only profile;
  designing the knob belongs to enforcement policy (see Out of Scope / Dependencies).
- Semgrep's path-derived rule IDs (e.g.
  `backstop/typescript-standards/backstop.packs...rules.security.ts.security.no-eval`) — a
  core-side output/waiver-token readability cleanup, explicitly logged as cosmetic and
  non-blocking in the pack-only write-up. Real, but it is gate-output ergonomics, not
  onboarding; it needs its own issue.
- `artifact new` number reservation (no gap-fill, no `--number`, stray reservation tags) — a
  `pkg/scaffold` CLI defect, not init behavior. Recorded under Observations with its verified
  root cause so it can be filed separately. REQ-020's "one check per ranked sharp edge" would
  sweep a doctor-side orphan-tag check in by implication; whether to make that explicit is a
  founder call, not one this pass took.

## Draft Design Decisions

- **DD-1: `backstop init` is the single zero-to-first-value command.** It installs the
  opinionated base bundle (omakase; DD-2 / OQ-2), writes profile-correct config, scaffolds
  the `.backstop/`-rooted artifact layout (OQ-1), emits the canonical `.gitignore`, seeds a
  local baseline (OQ-3), verifies the toolchain runs, and wires CI by default (OQ-7). It
  does ZERO language/framework detection (DD-11 / DD-13). No manual steps between having the
  binary and first useful output. Owned downstream by DIR-002.

- **DD-2 (headline): Init scaffolds for one of TWO validated profiles.** The **full-SDLC
  greenfield** profile (repo adopting backstop's artifact pipeline: artifact dirs + a
  minimal `backstop.yml` carrying `project:` + `language:`) reaches gate PASS on clean
  defaults with zero policy boilerplate. The **pack-only** profile (a consumer adopting
  packs but NOT the SDLC artifacts) must instead set `enforcement.policy` `level: off`
  for every SDLC dimension (`test_verification`, `coverage_threshold`,
  `contract_signature`, `test_substantiveness`, `artifact_status_drift`), because those
  dimensions hard-error on a missing `specs/` directory rather than skipping. Init must
  generate the correct config for the chosen profile so neither consumer hand-writes
  policy. Evidence: the full-SDLC dogfood of bclabs-portal (empty repo → gate PASS, 10
  passed / 2 skipped / 0 violations, 2026-07-12) and the pack-only profile captured in
  `backstop-packs/BACKSTOP-INIT-REQUIREMENTS.md`. This asymmetry is a genuine init design
  fork; earlier versions of this bundle imagined only one generic flow.

- **DD-3: The first run is framed as baseline observation, not failure.** Output uses
  "here's what we noticed," grouped by category and count; exit code is 0 (baseline
  captured), not 1 (violations found). The emotional framing is a product decision that
  matters as much as the data.

- **DD-4: Post-init default scope is diff-based.** An unflagged check operates on changed
  files only; the baseline records what existed at init so the gate distinguishes
  inherited from introduced patterns. Full-codebase checks are available on demand.

- **DD-5 (RETRACTED — superseded by OQ-5 / DD-11):** ~~Language detection drives default
  pack selection.~~ Init does ZERO language detection. Languages enter only via explicit
  `pack add`; multi-language = multiple packs. Retained here as a record of the reversal:
  detection was a baked-language assumption (a thin-executor violation), and removing it
  dissolves the multi-language question entirely (no "primary language" to pick). See the
  OQ-5 resolution and the hard invariant DD-13.

- **DD-6: Init verifies the toolchain actually RUNS.** It executes the language's
  test/compile entrypoint once (e.g. vitest / tsc) and confirms success rather than
  trusting package-manager configuration, whose semantics shift between majors. Evidence:
  pnpm 11.11 ignored `onlyBuiltDependencies` and exited nonzero with
  `ERR_PNPM_IGNORED_BUILDS`, which turned out cosmetic (vitest ran fine). Trusting the
  package-manager exit code would have produced a false init failure; executing the
  toolchain is ground truth.

- **DD-7: Init emits one canonical `.gitignore` and scaffolds at least one source file.**
  The canonical ignore list is the superset that worked in the dogfood: `.backstop/packs/`,
  `.backstop/baseline.json`, `.backstop/pack-config-provenance.json`,
  `.backstop/ts-coverage/`, `.backstop/ts-test-results.json`, `node_modules/`,
  `coverage/`. At least one source file is scaffolded because `tsc` on an empty repo REDs
  with TS18003 (No inputs found). Evidence: `.gitignore` diverged across onboarded repos
  (one ignored provenance + baseline; another ignored only `packs/`); a single canonical
  list removes the divergence.

- **DD-8: The init algorithm is specified BY transcribing the validated hand-onboarding
  checklist, not by re-inventing a flow.** The two hand write-ups are the empirical
  requirements corpus (one document per profile — see References); the greenfield doc's
  "Manual steps performed" list IS init's happy-path step sequence. Captured here so it
  survives the source docs being deleted, the full-SDLC greenfield sequence is:
  1. Scaffold a minimal project with ≥1 source file + one test + minimal config
     (toolchain devDeps, `tsconfig.json`, workspace config) and install dependencies.
  2. Write a minimal `backstop.yml` (`project:` + `language:` for greenfield; the
     pack-only profile adds the `enforcement.policy` `level: off` lines per DD-2).
  3. Create the artifact dirs (layout per OQ-1).
  4. Write the canonical `.gitignore` (DD-7).
  5. `backstop pack add <ref>` for each pack (toolchain, standards, secrets, contracts,
     substantiveness).
  6. `backstop gate` → iterate to PASS.
  Corollary: each ranked sharp edge from the write-ups is either a `doctor` check or an
  init guardrail — init automates the happy path, `doctor` diagnoses the deviations.
  Rationale: the flow is already validated by reaching gate PASS on a real repo by hand;
  transcribing "what the human did" is lower-risk than designing a new sequence.

- **DD-9: Init and `doctor` make CLI/pack version skew diagnosable.** Two mechanics:
  (a) `backstop version` stamps the build commit and date — **CORRECTED 2026-08-11 (v0.8.1,
  founder-ruled): the original trailing clause "never a bare `dev`" is WITHDRAWN.** It
  contradicted the shipped anti-spoofing design in `cmd/backstop/version.go`, which reports
  `dev` for any non-release build on purpose so a local build cannot masquerade as a release.
  Founder ruling: "it can be dev for a local build if that's standard practice." The mechanic
  survives as build IDENTITY (commit + date) making a stale dev build distinguishable from a
  fresh one — not as a prohibition on the string `dev`; (b)
  `pack add` and `gate` compare the binary's capability against the pack manifest's
  required features and fail loudly with "your backstop is older than this pack requires"
  instead of surfacing a downstream engine error. Evidence: the highest-pain sharp edge in
  the 2026-07-12 dogfood — a stale binary reporting bare `dev` predated the pack
  producer/convert split, and the gate reported `declared stdout_artifact ... not produced`
  quoting the plain engine command, giving no hint the binary was simply too old. This
  consumed most of the debugging time. Feeds the `backstop doctor` spec seed.

- **DD-10: Adjacent scope is owned elsewhere and is out of scope here (settled).**
  Baseline mechanics are owned by **BUNDLE-007** — resolved as: baseline is a CI-generated
  post-merge artifact, cached locally at `.backstop/baseline.json` (gitignored), pulled by
  the gate, ratchet-only (can only go down), never committed or hand-edited. Binary
  distribution (Homebrew tap, GitHub releases, cross-platform builds) and system-toolchain
  acquisition are owned by **DIR-001** (Release Workflow), now the top backlog priority.
  This bundle references those boundaries and does not re-litigate them; init consumes their
  outputs (a binary is present; a baseline can be generated), it does not implement them.

- **DD-11: Greenfield init = `git init` (only if not already a repo) + backstop base.**
  Init's job is to add the backstop layer, not the project. It runs `git init` only when
  there is no `.git`, then installs the omakase base. Zero language/framework detection —
  the project's identity is never inspected (emergent from OQ-5 / OQ-6).

- **DD-12: Packs may carry SCAFFOLDING RECIPES.** A recipe is a template plus a
  self-declared target path. `pack add` can bootstrap a repo for its language via a recipe,
  and init applies recipes through a GENERIC copy-template-to-declared-path mechanism —
  never language- or platform-aware code. (Depends on the pack-recipe capability, which
  does not yet exist — see Out of Scope / Dependencies.)

- **DD-13: HARD INVARIANT — init's thin-executor boundary.** Detection, framework
  recognition, and CI-platform knowledge live in PACKS as data, NEVER in core/CLI. Core
  dispatches; packs know. A language, framework, or platform name appearing in core CLI code
  IS the bug; `backstop/self` enforces it. This is the invariant that makes every other init
  resolution safe — omakase, recipes, and CI templates are all data supplied by packs.

- **DD-14: backstop COMPOSES with ecosystem scaffolders — it does not own project
  scaffolding.** `create-next-app`, `rails new`, `cargo new`, etc. produce the project;
  `backstop init` adds the backstop layer on top (converge-not-clobber, DD/OQ-6). Init never
  reimplements what an ecosystem scaffolder already does.

- **DD-15 (added 0.8.0): A false green is a strictly worse failure than a loud error, and
  onboarding is where both are manufactured.** DD-9 treats version skew as a DEBUGGING tax —
  the 2026-07-12 instance produced a misleading error that cost most of a day. The 2026-07-16
  instance produced something else: `artifact validate` reported GREEN across five revisions
  and a promotion to `defined` on a bundle that violated its own schema in 41 places. A
  misleading error at least stops you; a false green lets nonconformant governed state
  propagate and accrue derived work on an invalid foundation. Two distinct mechanisms in this
  bundle's evidence produce that class of failure, and both surface during onboarding, because
  onboarding is exactly when a consumer's binary, packs, and layout are least likely to agree:
  (a) **cohort skew** — the validator's embedded schemas silently predate the artifact's
  (REQ-026, REQ-027, REQ-028); (b) **silent undiscovery** — the gate walks an artifact root the
  consumer isn't using and reports a passing dimension over nothing (REQ-029, REQ-030).
  Rationale for making this a decision rather than a note: it establishes the acceptance
  posture for all five requirements — the correct behavior on "I cannot tell" is REFUSE, not
  warn and not pass. That inverts the default this codebase otherwise favors
  (loud-≠-blocking / warn-on-un-adopted-capability), and the inversion is deliberate: an
  un-adopted capability is a missing benefit, whereas an unverifiable green is an active lie
  about work that was already accepted on its strength. It also constrains init directly —
  REQ-031 forbids init from scaffolding the layout (b) would silently swallow, even though
  REQ-004 calls for that layout, until REQ-029 makes it discoverable.

## Open Questions

All seven open questions were driven to resolution by the founder in a 2026-07-13
working session (recorded in 0.6.0 below). None remain open. They are kept here — marked
RESOLVED with the decision and rationale — rather than deleted, so the reasoning survives.
The founder ruled promotion to `defined` on 2026-08-12 (see Version History 0.9.0).

- **OQ-1 (load-bearing) — RESOLVED: Canonical artifact-dir layout.** Decision: `.backstop/`
  is the artifact root for consumer repos; **backstop-core keeps the root layout as the
  framework exception** (it IS the framework — its artifacts are primary content, not
  governance overhead). Init scaffolds and validates the `.backstop/`-rooted layout for
  consumers. Rationale: one canonical consumer layout ends the live-repo divergence
  (backstop-core root vs backstop-runtime `.backstop/` vs stale test projects) and keeps the
  consumer root clean (only `backstop.yml`/`.lock` + `.backstop/` visible). This deliberately
  re-adopts the previously-retracted `.backstop/` convention, now with the framework
  exception made explicit rather than implicit.

- **OQ-2 — RESOLVED: How the init profile / capability set is chosen.** Decision: **omakase
  default** — bare `backstop init` installs the full opinionated bundle, PROMPT-FREE; you
  SUBTRACT capabilities via flags (`--only X` / `--no-X`, Cargo-features style: a default
  feature set you strip from). Rationale: an opinionated default is the fastest path to value
  and removes wrong-default guessing; subtraction is explicit and scriptable. Because it is
  prompt-free, init is headless/CI-safe by default — the interactive-vs-CI tension dissolves
  entirely.

- **OQ-3 — RESOLVED: baseline on a remoteless repo.** Decision: init **seeds a GITIGNORED
  LOCAL baseline** so solo/local-first users get the ratchet from day zero; the CI-generated
  baseline (BUNDLE-007) is the later team upgrade (tamper-resistance, no concurrent
  stomping). Also fix the self-contradictory remoteless `baseline_comparison` message.
  Rationale: local-first users should not need a remote to get the ratchet; CI baseline is an
  upgrade, not a prerequisite. Reflects back to BUNDLE-007 / DIR-003 as a new day-zero-local
  baseline mode (see Out of Scope / Dependencies).

- **OQ-4 — RESOLVED: local-pack restore from a fresh clone.** Decision: the committed
  `backstop.lock` records **only portable git-ref packs**; a GITIGNORED local provenance
  cache records local-path pack sources so `pack install` restores them **on the same
  machine**. Promoting a local pack to shareable is an explicit git-ref step. Rationale: keeps
  the committed lock portable (no machine-specific paths leak into shared history) while a
  single machine can still restore local packs; sharing is a deliberate act. The gitignored
  local-provenance lock mechanism is a pack-CLI / lock-schema change (see Out of Scope /
  Dependencies).

- **OQ-5 — RESOLVED (DISSOLVED): multi-language init.** Decision: init does **ZERO language
  detection**. Languages enter only via explicit `pack add` (which may optionally scaffold
  via a recipe); multi-language = multiple packs. Registry-era auto-detect is out of
  scope / future. Rationale: language detection was a baked-language assumption (a
  thin-executor violation); removing it dissolves the question — there is no "primary
  language" to pick. Supersedes the former DD-5; codified as DD-11 / DD-13.

- **OQ-6 — RESOLVED: re-init / idempotency.** Decision: **converge, never clobber,
  FRAMEWORK-BLIND.** Init detects only backstop-neutral facts (is there a `.git`? a
  `backstop.yml`? artifact dirs?), adds only what is missing, reports only in backstop terms,
  and NEVER interprets the repo's language/framework identity. Rationale: idempotent and
  non-destructive re-runs, and staying framework-blind keeps init inside the thin-executor
  boundary (DD-13). Codified as DD-11 / DD-14.

- **OQ-7 — RESOLVED: consumer CI-workflow scaffolding.** Decision: CI is wired **by default**
  (omakase), delivered as a **CI RECIPE PACK** holding per-platform templates as DATA; init
  applies the selected variant via the generic recipe mechanism (`--ci github` default,
  `--no-ci` opts out). **No `ci` verb** — the CI job chains existing agnostic commands
  (`pack install` → `baseline pull` → `gate`), which also gives local/CI parity. If no CI
  pack is installed, the CI STEP fails LOUDLY with actionable guidance (no baked fallback
  exists — the architecture makes silent success impossible) while the rest of init succeeds.
  Rationale: CI-platform knowledge is data in a pack, never baked into core (DD-13);
  default-on makes enforcement real fast; the loud-but-non-blocking failure honors
  loud-≠-blocking. Depends on the recipe capability + a concrete CI recipe pack (see Out of
  Scope / Dependencies).

### Questions closed earlier (recorded, not open)

- **Baseline storage / granularity / progressive reduction** — RESOLVED via BUNDLE-007;
  captured in DD-10. Not reopened here.
- **Binary distribution** and **dependency/toolchain installation strategy** — MOVED to
  DIR-001 (Release Workflow), the owner of "how users get the binary + system toolchain."
  Captured in DD-10. Not litigated here.

## Spec Seeds

Three non-overlapping seeds, in suggested implementation order: **trustworthy-green guards →
`backstop init` → `backstop doctor`**. Baseline, binary distribution, and toolchain
acquisition are deliberately NOT seeds here — they belong to BUNDLE-007 and DIR-001 (DD-10).

- **Trustworthy-green guards (core)** — new in 0.8.0, and FIRST in order. Covers: a
  content-derived schema cohort (REQ-026), cohort assertion with refuse-to-green in
  `artifact validate` and `gate` (REQ-027), producing-binary provenance on every result
  (REQ-028), config-resolved artifact root shared by gate / validate / scaffold (REQ-029), and
  scanned-root reporting with loud failure on an absent configured root (REQ-030). Rationale
  for sequencing it first: REQ-004 (`.backstop/` layout) is unimplementable without REQ-029,
  and every acceptance claim the other two seeds make is only as trustworthy as the validator
  asserting it (DD-15). Touches `cmd/backstop/root.go`, `pkg/validate/`, `pkg/gate/
  artifact_status.go`, `pkg/scaffold/`, and the `backstop.yml` schema — no init code.

- **`backstop init` command** — the critical-path spec. Covers: omakase base install with
  subtract-via-flags (DD-2 / OQ-2), `git init`-if-needed + converge-not-clobber re-init
  (DD-11 / DD-14 / OQ-6), `.backstop/`-rooted artifact layout scaffolding (OQ-1),
  profile-correct `backstop.yml` generation, canonical `.gitignore` emission + scaffold ≥1
  source file (DD-7), verify the toolchain runs (DD-6), local baseline seeding (OQ-3),
  default CI wiring via recipe (OQ-7), and first-gate baseline capture with observation
  framing (DD-3 / DD-4). Zero language/framework detection (DD-13). Specified by transcribing
  the greenfield hand-onboarding checklist (DD-8). **Blocked on** the pack-recipe capability
  (init consumes recipes but cannot be built until it exists — see Out of Scope /
  Dependencies).

- **`backstop doctor`** — the "help me fix my setup" diagnostic init delegates to when
  something is off. Covers: binary version stamp present (commit+date, not bare `dev`) and
  binary-vs-pack capability skew (DD-9); toolchain actually executes (DD-6); Node version
  vs stack policy (Node LTS); artifact-layout validation once OQ-1 lands. Each check is the
  diagnosis of a ranked sharp edge from the write-ups (DD-8 corollary).

## Notes / Ideas

### Out of Scope / Dependencies

The 0.6.0 OQ resolutions lean on capabilities that this bundle consumes but does NOT design
here. These are recorded so spec authoring does not accidentally absorb them:

- **Pack-recipe capability (BLOCKING DEPENDENCY)** — how packs declare and ship scaffolding
  + CI templates (template + self-declared target path; generic copy-to-path apply). Init
  consumes recipes but **cannot be built until this exists**. Likely its own bundle.
- **CI recipe pack** — a concrete pack deliverable in backstop-packs holding
  github / gitlab / bitbucket / jenkins templates as data. Depends on the recipe capability.
- **Gitignored local-provenance lock mechanism (OQ-4)** — a pack-CLI / lock-schema change so
  local-path pack sources restore on the same machine while the committed lock stays portable.
- **Local-first baseline seeding (OQ-3)** — a new day-zero-local baseline mode; reflects back
  to BUNDLE-007 / DIR-003 (the baseline subsystem).
- **Pack registry + pack-declared `detect:` field** — registry-era auto-detection; deferred,
  relates to pack distribution (BUNDLE-001 / BUNDLE-002). Explicitly NOT how languages enter
  in this bundle (OQ-5 dissolved detection).
- **Bundle→spec promotion gate check** — an orthogonal workflow-integrity hole: a spec whose
  parent bundle is not promoted should be a violation but currently is not enforced (this
  bundle's own legacy SPEC-020..029 were auto-generated against an unpromoted parent). Belongs
  to gate/workflow hardening, not init.

### Observations and evidence

- The emotional framing of first-run output (DD-3) is a product decision, not a technical
  one. The exact wording of the baseline summary determines whether a tech lead feels
  welcomed or attacked; it deserves design attention and possibly user-testing outside the
  project.
- **Cascading coverage error masks the real failure (pack-side, first-run polish).** When
  a test FAILS, `coverage_threshold` fails with a misleading secondary error — `declared
  stdout_artifact ".backstop/ts-coverage/backstop-summary.json" not produced` — because
  vitest emits no coverage summary on a failed run. The real failure is caught loudly in
  `pack_engines`, but the coverage step points at the pack producer rather than saying
  "tests failed, so coverage was not computed." Same misdiagnosis family as DD-9. A
  pack-side messaging concern, not strictly init, but it degrades the clean first-run
  experience init aims to deliver. (2026-07-12 verification pass.)
- **Node LTS is unenforced.** The dogfood machine had only non-LTS Node (23 + 25) despite a
  documented "Node LTS" stack decision, and nothing warned. Feeds the `doctor` seed as a
  node-version-vs-stack-policy check.
- **Positive — enforcement is real, not vacuous green.** On the live bclabs-portal repo, an
  injected `eval()` tripped the standards no-eval rule, a type error tripped tsc TS2322, a
  hardcoded AWS key tripped no-hardcoded-credentials, and a failing test tripped vitest —
  all drove the gate RED; a clean repo goes GREEN. First validation of the "real-project
  e2e / RED-when-it-should" bar in an external consumer.
- **Positive — waiver-hint renders in the wild.** The waiver-hint wiring
  (`↳ to waive: @waiver:...`) renders correctly in a live external consumer; first
  confirmation of that requirement outside the framework repo.
- **Observation — `ledger_integrity: skipped (ledger not implemented)`.** Core already
  reserves the concept a consumer is about to implement.
- **`artifact new` number reservation — out of scope here, but root-caused (2026-08-11).**
  The write-up reports that `artifact new spec` cannot gap-fill a number, that git tags are
  the undocumented reservation mechanism, and that an orphan tag got reused while a stray tag
  one higher was created. Reading `pkg/scaffold/idresolver.go` explains the observed symptom
  exactly: `GitTagResolver.Resolve` takes max+1 over `backstop/<type>/*` tags, CREATES the
  annotated tag, and only THEN pushes it — and a push failure that is not a `TagConflictError`
  (a remoteless repo, for instance) returns `FallbackError`, at which point `ResolveID` falls
  through to `LocalScanResolver`, which numbers from FILES on disk. So the run leaves behind
  the locally-created tag at max+1 while returning a lower, file-derived number. That is the
  "produced SPEC-007, stray-tagged 008" pair. It is worth noting this fires hardest on exactly
  the repo shape init targets — freshly created and remoteless. Still a `pkg/scaffold` defect
  rather than an init requirement, so it is recorded here for a separate issue rather than
  requirement-ized.
- **Semgrep rule-id readability — out of scope here.** The pack-only write-up logs it as
  cosmetic, core-side, and non-blocking; it degrades waiver tokens and gate output generally,
  not onboarding specifically. Recorded so it is not lost, not adopted as init scope.
- **Go-forward.** Derive the `backstop init` and `backstop doctor` specs directly from the
  two per-profile hand-onboarding checklists (DD-8), NOT by re-inventing the flow — after
  OQ-1 and OQ-2 are resolved and the bundle is promoted. Spec authoring is a
  transcription-and-hardening exercise against a validated flow.

## Version History

- 0.1.0 (2026-04-09): Initial bundle. Problem (no onboarding path), target experience,
  5 design decisions (init command, auto-install deps, baseline as observation, diff-based
  default scope, language detection), 8 open questions, 5 spec seeds. Maturity: exploring.
  A thought exercise analyzing onboarding from a tech lead's perspective.
- 0.2.0 (2026-04-19): Added DD-6..9 mandating `.backstop/` as the consuming-repo artifact
  root. Resolved baseline OQs via BUNDLE-007.
- 0.3.0 (2026-07-12): Evidence intake from the first real greenfield onboarding
  (bclabs-portal: empty repo → gate PASS). Added the two-profile decision, version-skew
  diagnosability, verify-the-toolchain-runs, canonical `.gitignore` + scaffold source file;
  added OQs for profile selection, artifact layout, remoteless baseline, local-pack restore.
- 0.4.0 (2026-07-12): Provenance reframe — established the two hand-onboarding write-ups as
  the empirical requirements corpus; added the "specify init by transcription" decision.
- 0.5.0 (2026-07-13): **From-scratch rewrite to current discipline.** Purged the legacy
  auto-generated SPEC-020..029 and their 9 plans (never committed — untracked cruft from a
  2026-05-30 auto-dispatch experiment run against this then-unpromoted bundle) and reset to
  a clean `exploring` foundation from which init will be properly re-speced in order
  (resolve OQs → promote → spec). Kept the dogfood gold: two-profile model (now the headline
  DD-2), version-skew diagnosability (DD-9), verify-the-toolchain-runs (DD-6), canonical
  `.gitignore` + scaffold source file (DD-7), init-by-transcription (DD-8), and the
  requirements-corpus framing. **Retracted the `.backstop/`-root mandate** (old DD-6/7/8):
  the dogfood used the ROOT convention and passed, and live repos diverge, so the layout is
  reopened as OQ-1 (the load-bearing open question) — root vs `.backstop/` vs detect-both.
  **Moved** binary distribution and dependency/toolchain installation OUT to DIR-001 (their
  owner); dropped the corresponding OQs and the binary/dependency spec seeds. **Recorded**
  the baseline OQs as resolved-via-BUNDLE-007 rather than open. Distilled the DD set to the
  ten dogfood-grounded decisions and the two spec seeds (init, doctor). Maturity unchanged:
  exploring — no self-promotion, no pre-resolved OQs.
- 0.6.0 (2026-07-13): **All 7 OQs resolved** in a founder-driven working session; each
  converted open → RESOLVED with decision + rationale (none remain open). Decisions: OQ-1
  `.backstop/` root for consumers, root for backstop-core (framework exception); OQ-2 omakase
  prompt-free default with subtract-via-flags (headless-safe, tension dissolved); OQ-3 seed a
  gitignored local baseline day-zero (CI baseline is the team upgrade); OQ-4 committed lock
  holds only portable git-refs + a gitignored local-provenance cache for local packs; OQ-5
  DISSOLVED — zero language detection, languages enter via `pack add`; OQ-6 converge /
  never-clobber / framework-blind re-init; OQ-7 CI wired by default via a CI recipe pack, no
  `ci` verb, loud failure if the pack is absent. Added emergent DD-11 (`git init` + backstop
  base, zero detection), DD-12 (pack scaffolding recipes via a generic copy-to-path apply),
  DD-13 (HARD INVARIANT — detection / framework / CI-platform knowledge lives in packs as
  data, never in core; `backstop/self` enforces), DD-14 (composes with ecosystem scaffolders,
  does not own project scaffolding). RETRACTED DD-5 (language detection) as superseded.
  Revised DD-1, the solution approach, and the init spec seed to drop language detection and
  reflect omakase + recipes + local baseline + default CI. Added an "Out of Scope /
  Dependencies" note recording the pack-recipe capability (a BLOCKING dependency init can't be
  built without), the CI recipe pack, the local-provenance lock change, local-first baseline
  seeding, registry-era detection, and the orthogonal bundle→spec promotion gate hole.
  Maturity unchanged: exploring — promotion is founder-triggered separately.
- 0.7.0 (2026-08-10): **Draft Requirements authored** — 25 formal requirements
  (REQ-001..025) added to frontmatter plus a matching `## Draft Requirements` section,
  each derived from an already-resolved OQ, an existing DD, or a spec seed; no new scope
  introduced. Partitioned non-overlappingly across the two seeds: REQ-001..019 to
  `backstop init` (command shape and omakase, two-profile config, `.backstop/` layout,
  canonical `.gitignore`, thin-executor boundary and generic recipe apply, `git init`
  -if-needed and converge-not-clobber, toolchain-runs verification, local baseline +
  observation framing + diff-scoped default, default CI via recipe pack with loud absent-pack
  failure, portable-lock installs, and the transcribed-sequence acceptance bar), REQ-020..025
  to `backstop doctor` (the command itself, version stamp, binary-vs-pack capability skew,
  standalone toolchain check, pack-declared stack-policy version check, layout validation).
  Recorded what is deliberately NOT a requirement here (baseline mechanics, binary
  distribution, the recipe capability and CI recipe pack, the local-provenance lock change,
  the pack-side cascading coverage message, registry-era detection). **One tension resolved
  and flagged:** DD-7's literal `.gitignore` list contains TypeScript-specific paths, which
  DD-13 forbids in core, so REQ-005 states the backstop-owned entries literally and defers
  the rest to what installed packs declare as generated output. Maturity unchanged:
  exploring — promotion remains founder-triggered.

- 0.8.0 (2026-08-11): **Grounding pass — requirements reconciled against the two
  hand-onboarding source documents themselves**, rather than against this bundle's summary of
  them. Fifteen concrete findings checked: ten already covered by REQ-001..025, two ruled out
  of scope with reasons recorded (semgrep rule-id readability — cosmetic core output
  ergonomics; `artifact new` number reservation — a `pkg/scaffold` defect, root-caused under
  Observations), one retired by its own successor evidence (the `pnpm-workspace.yaml`
  `onlyBuiltDependencies` recommendation, superseded when pnpm 11.11 ignored the key and the
  failure proved cosmetic — which is the argument FOR REQ-011). **Eight new requirements
  (REQ-026..033)**, each with its code state re-verified against HEAD on 2026-08-11 rather
  than trusted from July notes. The two serious gaps: (1) **schema-cohort false green** —
  REQ-026 content-derived cohort (`computeCohortID` is still path-derived, so BUNDLE-014's
  in-place `bundle/v2` revision was invisible), REQ-027 refuse-to-green when cohort coverage
  is unprovable (the only schema cross-check today is prefix-vs-artifact-type), REQ-028
  producing-binary provenance on every result (gate stamps `GitSHA`/`GeneratedAt` but no
  binary identity); (2) **artifact-root silent undiscovery** — REQ-029 config-resolved
  artifact root through one resolver (all three hardcodings confirmed present; no
  `artifact_root` key exists), REQ-030 scanned-root reporting with loud failure on an absent
  configured root while existing-but-empty stays a pass. Plus REQ-031 (init must not scaffold
  a layout discovery cannot see — REQ-004 is GATED on REQ-029), REQ-032 (install
  pack-declared project devDeps from recipe data before REQ-011 runs — a hole BETWEEN REQ-009
  and REQ-011), REQ-033 (pack-only coverage floor, since REQ-003 turns `coverage_threshold`
  off). Added **DD-15** generalizing both false-green mechanisms and setting the acceptance
  posture — on "I cannot tell", REFUSE rather than warn — a deliberate inversion of the
  loud-≠-blocking default, justified because an unverifiable green is an active lie about
  already-accepted work. Added a **third spec seed**, "Trustworthy-green guards (core)"
  (REQ-026..030), sequenced FIRST; the seeds now partition as init REQ-001..019 + 031..033,
  doctor REQ-020..025, guards REQ-026..030. Maturity unchanged: exploring — this pass changed
  requirements content only; promotion remains founder-triggered, and the prior structural
  assessment is untouched.
- 0.8.1 (2026-08-11): **Single-requirement correction — REQ-021 → v1.1.0, founder-ruled.** The
  clause "must never report a bare `dev`" was found to contradict shipped, deliberate behavior:
  `cmd/backstop/version.go` reports `dev` for ANY non-release build on purpose (rejecting
  `(devel)`, `+` build metadata, and Go pseudo-versions) so a local build cannot masquerade as an
  official release — verified against the code and against `./bin/backstop version`, which prints
  `backstop version dev`. Founder ruling: "it can be dev for a local build if that's standard
  practice" — the CODE is right and the requirement text was wrong. REQ-021 keeps its
  build-commit/build-date clause (build IDENTITY is what separates a stale dev build from a fresh
  one) and withdraws the prohibition; the prior text is preserved under `versions:` with a dated
  `correction`. Matching dated corrections applied to DD-9 mechanic (a) and the Current Thinking
  version-skew bullet, which both repeated the withdrawn phrasing. **Nothing is lost:** the
  2026-07-12 dogfood concern (a weeks-old binary reporting only `dev`, costing significant
  debugging time) is carried by REQ-022 — which names the stale binary as the cause instead of
  surfacing the downstream engine error, the exact observed failure mode — and by REQ-028, which
  stamps producing-binary identity onto every result. No new mechanism was invented beyond the
  one-line ruling. No requirement added or retired; no OQ resolved; maturity unchanged: exploring.
- 0.9.0 (2026-08-12): **Promoted to `defined` — founder-ruled.** No content authored in this
  pass: every structural requirement for `defined` was already in place from the prior passes —
  33 requirements (REQ-001..033) across two authoring passes (0.7.0's 25 plus 0.8.0's 8
  dogfood-grounded additions, with 0.8.1's REQ-021 correction), all 7 original open questions
  resolved since 2026-07-13, and populated Draft Requirements / Draft Design Decisions (DD-1..15) /
  Spec Seeds / Version History sections plus `solution.approach`. Promotion-readiness was verified
  independently before the ruling by flipping maturity on a throwaway copy and running
  `artifact validate` clean. This entry records the maturity change, the `bundle.updated` bump, and
  a one-line correction to the Open Questions preamble, which still asserted maturity would stay
  `exploring`. Unblocks `/spec` against the three seeds, in order: trustworthy-green guards →
  `backstop init` → `backstop doctor`. Context: DIR-002 sits at BACKLOG position 1, and its
  blocking dependency (BUNDLE-015 REQ-018, the CI recipe pack) was delivered via SPEC-067.

## References

### Requirements corpus (hand-onboarding write-ups — one per profile)

These are the empirical requirements corpus for the `backstop init` and `backstop doctor`
specs, not incidental links. Both are currently UNTRACKED field notes living in OTHER repos;
treat them as transient. Their durable home is this bundle — DD-8 and the sharp-edge
decisions/OQs/notes above lift the load-bearing substance in so it survives the docs being
deleted.

- **`backstop-packs/BACKSTOP-INIT-DOGFOOD-FULL-SDLC.md`** — the **full-SDLC greenfield**
  profile hand-onboarding (empty repo → `backstop gate` PASS, 2026-07-12; version-skew
  instance #2 appended 2026-07-16; artifact-layout founder ruling appended 2026-07-17). Its
  "Manual steps performed" list is transcribed into DD-8 as init's happy-path sequence; its
  ranked sharp edges are the source of DD-9, DD-15, and REQ-026..031. Previously cited here
  as `bclabs-portal/docs/dogfood/init-sharp-edges.md` — same corpus, relocated alongside its
  pack-only sibling (path corrected 2026-08-11).
- **`backstop-packs/BACKSTOP-INIT-REQUIREMENTS.md`** — the **pack-only** profile
  hand-onboarding (consumer adopting packs without SDLC artifacts). Source of the
  `enforcement.policy` `level: off` requirement in DD-2, and of REQ-032 / REQ-033.

### Related artifacts

- DIR-001: Release Workflow — owns binary distribution + system-toolchain acquisition +
  post-merge baseline generation (out of scope here; DD-10).
- DIR-002: `backstop init` command — the directive this bundle sources.
- BUNDLE-007: Baseline — owns baseline mechanics (CI-generated post-merge, gitignored local
  cache, ratchet-only); resolves this bundle's baseline questions (DD-10).
- BUNDLE-001: Pack distribution — default pack wiring depends on packs being distributable.
- SPEC-008: Code check (diff-based scope via ResolveScope).
- SPEC-010: Gate (baseline step consumes BUNDLE-007's output).
