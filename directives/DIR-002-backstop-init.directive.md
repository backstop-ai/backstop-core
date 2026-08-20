---
title: "backstop init Command"
number: DIR-002
created: "2026-04-19"
schema_version: directive/v1

directive:
  status: done
  completed: "2026-08-20"
  source:
    - "BUNDLE-003"
    - "ISSUE-122"
    - "ISSUE-124"
    - "ISSUE-134"
    - "ISSUE-139"
---

## Description

`backstop init` is the single command that takes a consuming project from
zero to first value. Per BUNDLE-003 (`onboarding-experience`, v0.10.0,
`defined`), init: `git init`-if-needed + install the omakase base (DD-11),
scaffold the `.backstop/`-rooted artifact layout (OQ-1), generate
`backstop.yml`, verify (not install) that the pack-declared toolchain
entrypoint runs and report an uninstalled/failing one as setup the consumer
still owes, naming the pack whose entrypoint could not run (REQ-011 v1.1.0),
seed a local gitignored baseline (OQ-3), wire CI by default via a recipe
pack (OQ-7), and capture the first gate result as observation ("here's what
we noticed"), not judgment (DD-3/DD-4) — all with zero manual config steps,
framework-blind throughout.

Init does **ZERO language/framework detection** — this is a HARD INVARIANT
(DD-13): detection, framework recognition, and CI-platform knowledge live in
packs as data, never in core. Languages enter a consumer project only via an
explicit `pack add` (REQ-010), never by inspecting the project's identity —
a language/framework name appearing in init's own code would itself be the
bug, and `backstop/self` enforces it. There is no "default language pack"
for init to wire; omakase is a fixed, framework-blind BASE bundle you
subtract capabilities from via flags (DD-2/OQ-2), not a per-language
default. Init also does not install a pack's devDependencies — the MVP is a
docs-only dependency-install guide living in the relevant pack's own
README, a pack-authoring convention, not a backstop-executed mechanism.

Target: under 2 minutes from install to first useful output.

Depends on DIR-001 (release workflow — users need the binary) and DIR-009 (pack smoke test — init wires packs, packs need to work).

**History (2026-08-12 correction, kept for record).** This directive's
Description originally read: "Scaffolds `.backstop/` directory structure,
creates `backstop.yml` and `backstop.lock` at root, detects language,
auto-installs dependencies (semgrep, golangci-lint, ruff), wires the default
language pack, runs the first gate, and captures the result as a baseline
presented as observation... not judgment." That framing — language
detection, dependency auto-install, a default language pack — is stale and
contradicted this directive's own source, BUNDLE-003. BUNDLE-003's OQ-5
dissolved the language-detection framing entirely (now DD-13, above).
Separately, "auto-installs dependencies (semgrep, golangci-lint, ruff)"
described what became REQ-032 (pack-declared project dependency
installation), which BUNDLE-003 **retired** in its 2026-08-12 v0.10.0 pass,
founder-ruled, in favor of the docs-only guide described above. The current
Description (top of this section) reflects the corrected, current state;
this paragraph is preserved only so a reader tracing history understands
what changed and why.

**Follow-on (2026-08-14): ISSUE-122.** `DiscoverArtifacts`
(`cmd/backstop/artifact_discover.go:47-49`) bakes two ecosystem-specific
directory literals — `vendor` (Go) and `node_modules` (Node) — directly into
core's artifact-discovery skip list, violating this directive's own DD-13
(zero-baked-language, established above). Homed here rather than a new
standing directive because DD-13 is already DIR-002's recorded hard
invariant and DIR-002's lane is already editing this exact file via
SPEC-068. The fix direction (not yet decided at plan time): source `vendor`/
`node_modules` from pack data the way `EngineBinding.StdoutArtifact`
sources generated/excluded paths today, rather than deleting the literals
outright — deletion alone would let discovery walk into dependency trees and
false-positive on artifact-shaped filenames there.

**Follow-on (2026-08-14): ISSUE-124.** Six residual hardcoded artifact
extension/directory literals survive SPEC-068's de-duplication of the
artifact-layout table — four `strings.HasSuffix(..., ".spec.md")` checks in
`pkg/gate/step_testverify.go` (120, 225, 520, 564), `pkg/validate/spec.go:751`
(`.spec.md` slug derivation), `pkg/validate/adr.go:159` (`.adr.md`),
`pkg/validate/bundle.go:275-278` (`.epic.bundle.md`/`.bundle.md`),
`pkg/validate/supports_resolution.go:252` (`.issue.md`), and
`cmd/backstop/gate_substantiveness_e2e.go:44` (a hardcoded `"specs"` dir in an
e2e harness). Each bypasses `pkg/artifact`'s `LayoutFor`/`Root.Dir` — the ONE
shared layout authority SPEC-068 established while closing six earlier copies
of the same table (the sixth, `pkg/validate/delivered_by.go`, was fixed as an
impl-reviewer finding during PLAN-SPEC-068's implementation; these residuals
were deliberately left out of that plan's scope, each needing its own
confirmation that it is a genuine artifact-type reference and not incidental
string matching).

Homed here because the failure mode is init's own deliverable: a project whose
artifact root or per-type directory naming differs from the historical default
— exactly what BUNDLE-003 OQ-1's `.backstop/`-rooted scaffolded layout produces —
drifts out from under production validation and gate discovery SILENTLY, visible
only as false-negative misses (an artifact file present on disk that the code
meant to see never discovers) rather than a loud failure. Same reasoning that
homed ISSUE-122 here: DIR-002 owns the BUNDLE-003 lane that established the
layout authority via SPEC-068, and this is that authority's unfinished adoption.

Note explicitly: `pkg/scaffold/scaffold.go`'s `FileExtension` entries are NOT in
this class — PLAN-SPEC-068's TASK-019 deliberately sanctions them as scaffold's
own local concern.

Also carry the issue's own scoping caveat: where a caller needs only the
extension and not the full directory, whoever plans this must confirm the shared
authority exposes that granularity before doing a mechanical string swap — if it
does not, closing that gap in the authority belongs to this fix too, since a
partial authority invites the same drift straight back in.

**Follow-on (2026-08-16): ISSUE-139.** The defective test,
`pkg/initialize/sourceset_scan_test.go`, is the sole mandated test for
SPEC-069 CLM-063, which asserts SPEC-069's own REQ-013 denylist claim ("this
spec's implementation changes no file under `pkg/gate`"). SPEC-069 declares
`source.bundle: BUNDLE-003` — this directive's own lane — and its scope
section names "any change to `pkg/gate`" as explicitly OUT of scope, which
is exactly the claim this test exists to enforce. The defect is in init's
own test of init's own claim, not in gate machinery: (1) the purity
assertion runs `git status --porcelain -- pkg/gate` against the shared
working tree and treats any dirt from any source as proof init leaked into
gate, with no way to attribute dirt to init versus a concurrent,
independently-scoped lane; and (2) the non-vacuity skip guard (which would
recognize `pkg/initialize` itself has no uncommitted changes and skip) sits
after the fatal check, so it is unreachable in exactly the steady state it
was written for. Both confirmed live in the current source
(`sourceset_scan_test.go:310-342`), not theoretical: `go test
./pkg/initialize/ -run TestInit_ChangesNoGatePackageFile` fails today,
citing `pkg/gate` paths belonging to two other lanes' in-scope work
(PLAN-ISSUE-118, PLAN-ISSUE-113 — both DIR-032 members, both `status:
draft`, both declaring `pkg/gate` in their own file scope). Note explicitly
this is NOT a DIR-032 "Gate Verdict Honesty" member despite being a
false-verdict bug — DIR-032's charter is a GATE STEP mis-reporting a verdict
it computed; this is a Go unit test in `pkg/initialize`, and reading
DIR-032's charter loosely enough to swallow it would swallow every test bug
in the repo. Second-order finding for whoever plans this: because the guard
is unreachable, CLM-063's first half currently has no steady state in which
it verifies anything — dirty tree false-fails, clean tree passes vacuously
— so the fix must restore a path where the assertion can actually verify,
not just silence the failure.

**Follow-on (2026-08-16): ISSUE-134 — and an explicit scope expansion,
founder-ruled.** `backstop doctor`'s toolchain-runs check
(`checkToolchainRuns`, `cmd/backstop/doctor_checks.go:227`) verifies
pack-declared test/build entrypoints via the shared `packEntrypointProber`,
but the prober selects bindings by `GateType == GateTypeTest ||
GateTypeBuild` only (`pack_entrypoint_prober.go:126`, doc comment:
"Selection is by STAGE, never by tool") — it never walks the separate
engine-registry path (`resolveEngineRegistry`/`dispatchPackEngines`,
`cmd/backstop/pack_gate.go`) that resolves findings-engine tools (semgrep,
ast-grep) at gate time. A project can run `backstop doctor`, see every
check pass — including "pack-declared test/build entrypoints execute" —
and still be missing a findings-engine tool from PATH entirely, with no
signal. The next `backstop gate` run hits exactly the failure mode
ISSUE-112 describes: a missing tool with no CrashGuard silently produces
empty SARIF, `pack_engines` passes, and downstream joins starve on the
empty finding set, producing misleading violations attributed to the wrong
cause — the same condition that produced bclabs-portal's 397 false
violations. Doctor's entire purpose is proactive pre-flight health
verification; this is a real gap in that purpose, not a residual
baked-literal cleanup item like ISSUE-122/ISSUE-124 above. Two constraints
recorded for whoever plans this (from the issue itself): `packEntrypointProber`
has exactly two callers — doctor's `checkToolchainRuns` and `backstop
init`'s own toolchain step (`init_toolchain.go`, SPEC-069) — so widening
`Probe`'s selection logic in place would silently change what `init`
verifies too; and doctor's existing pass/fail/refused report vocabulary was
designed for test/build entrypoints and doesn't map cleanly onto "this
findings-engine tool is absent," needing its own design pass.

Founder-approved home and framing, recorded here so a reader does not have
to reconstruct it (Brandon, 2026-08-16): this directive's remaining scope
was, until now, BUNDLE-003 delivery plus residual baked-literal cleanup
(ISSUE-122, ISSUE-124) and one red-test fix (ISSUE-139, itself already
noted below as "not residual cleanup"). ISSUE-134 is neither residual
cleanup nor a leftover from BUNDLE-003's own delivery — it is a genuinely
new defect class (doctor's diagnostic coverage of findings-engine tools)
surfaced after `backstop doctor` shipped under SPEC-070, itself sourced
from BUNDLE-003 (this directive's own first `source:` entry). The founder
ruled it belongs here rather than under DIR-032 (gate-verdict-honesty —
doctor is not a gate step, so it doesn't fit that directive's charter) or
as a new standalone directive (this directive's lane already owns the
doctor surface via BUNDLE-003/SPEC-070). This directive's scope is hereby
understood to include doctor's tool-detection diagnostic reliability going
forward as an ongoing concern, not merely BUNDLE-003 residue — stated
explicitly rather than left for a reader to infer from the source list
alone.

## Notes

- **Sequencing: ISSUE-122 must not be planned or implemented until SPEC-068
  has landed.** SPEC-068 ("Trustworthy Green Guards", currently in review,
  itself part of this directive's BUNDLE-003 lane) already rewrites the same
  switch statement ISSUE-122 targets — its REQ-007 changes `DiscoverArtifacts`
  to take an `artifact.Root`, deletes the private `artifactPatterns` map, and
  converts the line-48 skip list to a root-relative exclusion. Picking up
  ISSUE-122 first or concurrently would collide on the same three lines and
  re-derive a signature that is about to change out from under it. A future
  session should treat SPEC-068's implementation as a hard precondition for
  starting ISSUE-122's plan.
- **Sequencing: ISSUE-124 also depends on SPEC-068 having landed**, and
  should be planned together with — or immediately after — ISSUE-122, since
  both are residual baked-literal fixes against the same artifact-discovery/
  layout surface and would otherwise independently re-derive the same "what
  does the authority expose to a non-`pkg/artifact` caller" question. As of
  2026-08-14, SPEC-068 and PLAN-SPEC-068 both still carry `status: draft` on
  disk while a closeout pass is in flight — verify their real status before
  treating the precondition as met.
- **ISSUE-139 has no precondition and is red in the shared tree right now** —
  unlike ISSUE-122 and ISSUE-124, it does not gate on SPEC-068 landing. It
  touches only `pkg/initialize/sourceset_scan_test.go`, so it does not
  collide with SPEC-068's or ISSUE-122's artifact-discovery surface and can
  be planned independently and immediately; it is the one item in this
  directive's remaining scope that is not residual cleanup.
- **ISSUE-134 also has no SPEC-068 precondition and is not residual
  cleanup** — like ISSUE-139, it is a fresh defect discovered after
  BUNDLE-003 delivered `backstop doctor` (SPEC-070), not leftover
  baked-literal cleanup from BUNDLE-003's own SPEC-068 lane. It touches
  `cmd/backstop/pack_entrypoint_prober.go` and `doctor_checks.go`, a
  different surface than ISSUE-122/ISSUE-124's artifact-discovery/layout
  surface, so it does not collide with those and can be planned
  independently. Founder-ruled scope expansion, 2026-08-16 (see Description
  above): this directive's charter is no longer just BUNDLE-003 delivery
  plus residual cleanup — it now explicitly includes doctor's
  findings-engine tool-detection diagnostic coverage as an ongoing concern.
- **Closed 2026-08-20 — all sources terminal.** BUNDLE-003 reached
  `maturity: delivered` 2026-08-20 (v0.11.0); ISSUE-122, ISSUE-124,
  ISSUE-134, and ISSUE-139 are all `status: closed`. Three BUNDLE-003
  requirements carry explicit accounting of their own — none of it open
  work under this directive, all of it the bundle's own bookkeeping: REQ-022
  (version-skew capability-gap naming) is CARVED OUT / UNOWNED, deferred to
  BUNDLE-020 once its OQ-2/OQ-3 resolve; REQ-024 (doctor's stack-policy
  deviation check) is CARVED OUT / UNOWNED, deferred to ISSUE-121 under
  BUNDLE-004 pending a pack-manifest stack-policy surface that does not yet
  exist; REQ-033 (pack-only coverage floor) is OWNED-BUT-UNSATISFIED —
  SPEC-069 shipped a truthful report of the gap rather than the wiring
  mechanism, with absence claims proving no schema surface was invented to
  fake it. See BUNDLE-003's own "OWNERSHIP CORRECTION 2026-08-20 (v0.11.0)"
  and "DELIVERY NOTE 2026-08-20 (v0.11.0)" passages for the full accounting.
  With every source terminal, this directive is closed.
