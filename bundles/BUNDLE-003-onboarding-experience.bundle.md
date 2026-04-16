---
title: "Onboarding Experience — Zero to First Value in Under 2 Minutes"
number: BUNDLE-003
created: "2026-04-09"
schema_version: bundle/v1

bundle:
  name: onboarding-experience
  version: "0.1.0"
  created: "2026-04-09"
  category: feature

status:
  maturity: exploring

problem:
  summary: >
    Today backstop has no onboarding path. Users must hand-create backstop.yml,
    manually set up .backstop/, figure out which pack to wire, and their first
    run produces a wall of violations with exit code 1. A tech lead evaluating
    the tool has ~30 minutes of patience. Every manual step and every hostile
    first impression is adoption friction that kills the eval. The objective is
    minimal time from downloading backstop to first value derived — under 2
    minutes from install to "oh, this caught something real in my code."

  user_story: >
    As a tech lead evaluating backstop for my team, I want to install the tool,
    run one command in my project, and see useful observations about my codebase
    within 2 minutes — without installing dependencies manually, without writing
    config files by hand, and without being told my entire codebase is broken.
    I want to feel like the tool understands my project, not that it's judging
    it.

solution:
  approach: >
    A single `backstop init` command that detects the project's language,
    installs required toolchain dependencies (semgrep, golangci-lint, ruff,
    etc.), scaffolds backstop.yml and .backstop/, wires the appropriate default
    language pack, runs a full check, and captures the results as a baseline —
    presented as passive observation ("here's what we noticed") rather than
    judgment ("427 violations found"). Post-init, default scope is diff-based:
    backstop only enforces on changed files, making day-one friction zero. The
    baseline exists so the gate knows inherited patterns from introduced ones.
    Over time the team chips away at the baseline; day one is zero blocking
    violations.
---

# Onboarding Experience

## Current Thinking

### The current state

There is no `backstop init`. Users must:
1. Somehow obtain the binary (no distributable yet)
2. Hand-create `backstop.yml` (two lines, but you have to know what)
3. Hand-create `.backstop/` directory structure
4. Figure out how the Go standards pack gets wired (it's embedded but implicit)
5. Run `backstop gate` — get hundreds of violations with no context
6. Give up

### The target experience

1. `brew install backstop` or `go install` — 10 seconds
2. `backstop init` — detects language, installs deps, scaffolds config, wires
   default pack, runs baseline — 30 seconds
3. Fix one thing, run `backstop check` on the changed file, see it go green
4. **That's first value. Under 2 minutes.**

Everything between "install" and "first green check" is friction that kills
adoption. The 9-step gate kill chain, artifacts, specs, plans — that's the
power user story. The tech lead doing a 30-minute eval never sees it.

### Key design principles

**Auto-install dependencies.** The tech lead should never have to
`pip install semgrep` or `brew install golangci-lint` themselves.
`backstop init` detects the language, identifies the required toolchain,
and installs what's missing. We already have `EnsureSemgrep` in the code
check pipeline (SPEC-008) — same pattern, applied at init time for the
full toolchain.

**Baseline as observation, not judgment.** If the first thing backstop says
is "427 violations found, exit code 1," the tech lead hears "your code
sucks." The reframe: init captures everything as a baseline and presents it
as "here's what we noticed" with a breakdown by category. Same data,
completely different emotional response. The tech lead thinks "cool, it
understands my codebase" instead of "great, another tool yelling at me."

**Diff-based scope by default post-init.** After init, `backstop check`
only looks at what you changed. The baseline exists so the gate knows what
was already there vs what you introduced. You're only accountable for new
patterns, not inherited ones. Progressive adoption — over time the team
chips away at the baseline, but day one is zero friction.

## Draft Design Decisions

- **DD-1:** `backstop init` is a single command that takes a project from
  zero to first value. It detects language, installs dependencies, scaffolds
  config, wires the default pack, runs a full check, and captures the
  baseline. No manual steps required between install and first useful output.

- **DD-2:** Dependency installation happens automatically during init.
  `backstop init` detects which tools are needed for the detected language
  (semgrep, golangci-lint, ruff, etc.) and installs any that are missing.
  Extends the existing `EnsureSemgrep` pattern from SPEC-008 to the full
  toolchain. Failures are reported clearly ("could not install golangci-lint:
  reason") rather than surfacing as cryptic errors on first check.

- **DD-3:** The first run's output is framed as a baseline observation, not
  a failure. Output uses language like "here's what we noticed" with
  violations grouped by category and count. Exit code is 0 (baseline
  captured), not 1 (violations found). The emotional framing matters as
  much as the data — the tool should feel like it understands the codebase,
  not that it's condemning it.

- **DD-4:** Post-init default scope is diff-based. `backstop check` without
  flags operates on changed files only (git diff). The baseline records what
  existed at init time so the gate can distinguish inherited patterns from
  newly introduced ones. Users are only accountable for new code from day
  one. `--all` is available for full-codebase checks when the team is ready.

- **DD-5:** Language detection drives default pack selection. `backstop init`
  inspects the project (file extensions, go.mod, package.json, pyproject.toml,
  Cargo.toml, etc.) and wires the appropriate default language pack
  automatically. Multi-language projects get multiple packs wired. The user
  can override or add packs later.

## Open Questions

- **OQ-1: Binary distribution.** How do users get backstop? `brew install`
  via a Homebrew tap? `go install`? Pre-built binaries on GitHub releases?
  Docker image? All of the above? Each has different first-install friction.
  `brew install` is lowest friction for macOS; `go install` assumes Go
  toolchain is present. Lean: Homebrew tap + GitHub releases + go install,
  in that priority order.

- **OQ-2: Dependency installation strategy.** How does backstop install
  dependencies? Homebrew for macOS tools? pip for semgrep? Direct binary
  download? What about environments where the user can't install system
  packages (CI containers, locked-down corporate machines)? The EnsureSemgrep
  pattern downloads a binary directly — does that scale to all tools?

- **OQ-3: Baseline storage format and location.** Where does the baseline
  live? `.backstop/baseline.json`? Committed to the repo (shared across the
  team) or local-only (per-developer)? If committed, how does it interact
  with branches? If local, how does a new team member get the baseline?
  The baseline is a gate input (step 7 in the kill chain) so its format
  matters for the gate's consumption.

- **OQ-4: Baseline granularity.** Is the baseline per-file, per-rule, or
  per-violation? Per-file is coarse (any change to a baselined file
  re-evaluates everything). Per-violation is precise but fragile (line
  numbers shift on any edit). Per-rule-per-file is a middle ground ("this
  file had 3 GO-011 violations at baseline time").

- **OQ-5: Progressive baseline reduction.** How does the team chip away at
  the baseline over time? Manual (`backstop baseline update`)? Automatic
  (if a file is touched and violations decrease, update the baseline)?
  Ratchet (baseline can only go down, never up)? Ratchet is the strongest
  model but needs careful UX.

- **OQ-6: Multi-language project init.** A project with Go backend and
  TypeScript frontend — does init wire both packs? Ask the user? Only wire
  the primary language? How does it detect "primary"?

- **OQ-7: Init in an existing backstop project.** What happens when someone
  runs `backstop init` in a project that already has `backstop.yml`? Error?
  Upgrade/re-detect? Offer to re-baseline? Must be non-destructive.

- **OQ-8: CI integration scaffolding.** Should `backstop init` also
  generate a `.github/workflows/backstop.yml` for CI? Or is that a separate
  `backstop ci init` command? The faster the team gets backstop into CI, the
  faster enforcement becomes real rather than optional.

## Spec Seeds

- **`backstop init` command** — language detection, config scaffolding,
  dependency installation, default pack wiring, baseline capture, output
  framing. The critical-path spec.
- **`backstop doctor`** — diagnose config, verify toolchain, check pack
  integrity, report what's missing or broken. The "help me fix my setup"
  command that init delegates to if something goes wrong.
- **Baseline capture and storage** — baseline format, storage location,
  gate integration (step 7), progressive reduction mechanics.
- **Dependency management** — generalized EnsureSemgrep pattern for the
  full toolchain. Detection, installation, version pinning, update path.
- **Binary distribution** — Homebrew tap, GitHub releases, go install,
  CI images. Making "step 1: install" as frictionless as possible.

## Notes / Ideas

- The emotional framing of first-run output is a product decision, not
  a technical one. It deserves design attention — the exact wording of
  the baseline summary will determine whether a tech lead feels welcomed
  or attacked. Consider user-testing the output with people outside the
  project.
- The baseline is gate step 7 (currently deferred). This bundle's baseline
  work directly unblocks that step in the kill chain.
- The `EnsureSemgrep` pattern in SPEC-008 already proves the dependency
  auto-install approach works for one tool. Generalizing it is the spec
  seed, not a new invention.
- Init should be idempotent or at minimum non-destructive. Running it
  twice should not break anything or lose config.

## Version History

- 0.1.0 (2026-04-09): Initial bundle. Captured problem (no onboarding
  path, tech lead has 30 minutes of patience), target experience (under 2
  minutes to first value), 5 design decisions (init command, auto-install
  deps, baseline as observation, diff-based default scope, language
  detection), 8 open questions, 5 spec seeds. Maturity: exploring.
  Motivated by thought exercise analyzing onboarding from a tech lead's
  perspective.

## References

- SPEC-008: Code check (EnsureSemgrep pattern, diff-based scope via
  ResolveScope)
- SPEC-010: Gate (step 7 baseline is currently deferred — this bundle's
  baseline work unblocks it)
- BUNDLE-001: Pack distribution (default pack wiring depends on packs
  being distributable)
