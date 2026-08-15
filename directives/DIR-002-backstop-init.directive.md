---
title: "backstop init Command"
number: DIR-002
created: "2026-04-19"
schema_version: directive/v1

directive:
  status: active
  source:
    - "BUNDLE-003"
    - "ISSUE-122"
    - "ISSUE-124"
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
