---
title: "Dogfood a thin-executor enforcement rule that makes 'zero baked language knowledge' self-enforcing"
schema_version: issue/v1

issue:
  id: ISSUE-024
  title: "Dogfood a thin-executor enforcement rule that makes 'zero baked language knowledge' self-enforcing"
  type: technical-debt
  status: open
  created: "2026-06-21"

complexity:
  scope: cross-cutting
  uncertainty: exploratory
  risk: safe
---

# Dogfood a thin-executor enforcement rule that makes 'zero baked language knowledge' self-enforcing

## Problem

Backstop's first principle is "zero baked language/tool knowledge in the gate path" — the
executor is thin, packs declare engines, and the CLI binary speaks only SARIF. Every spec
and bundle in the eradication track (SPEC-034, SPEC-035, BUNDLE-010, BUNDLE-011) gestures
at this invariant, but the invariant is enforced only through code review, manual audit, and
ad hoc deletion-assertion tests. The moment a new hardcoded language check, tool invocation,
or Go-analysis import drifts back into the gate-execution path, the system has no mechanism
to go red.

**The rule must not live in the CLI binary.** Baking a "no-baked-knowledge" checker into
`cmd/backstop/` is itself the sin it is meant to catch — a language-aware check compiled
into the executor. The correct home is backstop's own self-dogfood pack
(`backstop/go-standards`, already declared in `backstop.yml` as `local`), or a dedicated
`backstop/thin-executor` pack that replaces or extends it. The rule is then executed by
the same engine dispatch that runs every other pack — the checker is not special.

**The engine must be ast-grep.** ast-grep is already wired (SPEC-035 REQ-004 establishes
`pattern-arg` input mode, which is the natural carrier for ad hoc structural patterns
without a rule-file on disk). ast-grep parses Go structurally; the forbidden patterns below
are syntactic, not semantic, so a query engine is the right tool. A semgrep rule would also
work but semgrep's `pattern` mode for Go is more verbose; ast-grep patterns are tighter.

**Forbidden patterns — scope limited to the gate-execution path.** The packages that must
remain thin are:

- `cmd/backstop/`
- `pkg/check/`
- `pkg/gate/`
- `pkg/pack/engine/`
- `pkg/packval/`

Within those packages (excluding the sanctioned-sites allowlist — see below), the following
patterns must not appear:

| Category | Examples | Signal |
|---|---|---|
| Hardcoded language-name literals | `"go"`, `"typescript"`, `"python"`, `"bash"` as string constants | Names a language; executor must not branch on language |
| Hardcoded tool-invocation literals | `"go test"`, `"go build"`, `"semgrep"`, `"golangci-lint"` | Bakes a specific tool; packs must declare this |
| Go-analysis imports | `go/parser`, `go/ast`, `go/types` | Go-specific parsing inside the executor; antithetical to thin-executor model |
| File-routing literals | `.go` suffix strings, `_test.go` routing expressions | Language-specific file dispatch in the executor |

**Sanctioned-site allowlist.** There are a small number of legitimate appearances of
language tokens in the gate-path packages — places where the executor correctly references
its own infrastructure rather than a target language:

- `pkg/gate/step_testverify.go` — the `go test ./...` call is the gate's own test runner
  invoked for test-verify; it is not routing by language.
- `pkg/packval/` — pack-validation harness that runs packs against fixture projects; the
  harness itself must invoke language toolchains as test infrastructure. Until ISSUE-019
  redesigns packval to route through generic engine dispatch, these are temporarily
  sanctioned.
- Any test file (`*_test.go`) at any location — test code may reference tool names and
  language literals for fixture and assertion purposes.

The allowlist is declared in the pack rule (e.g., a `paths.exclude` list in the ast-grep
rule YAML), not in the CLI binary.

**Property of the rule.** This is a forbidden-pattern rule with an allowlist. The correct
posture during eradication transition is LOUD, not blocking — `loud != blocking` is a
first-class backstop principle. The rule should emit findings (SARIF warnings) for every
violation so the eradication backlog is visible, but should not block CI until eradication
completes. Once eradication is done, the threshold can be tightened to blocking. The rule
functions as a living acceptance test: GREEN exactly when "zero baked language paths" holds
in the gate-execution packages.

**Why this cannot be built yet.** The rule depends on two downstream capabilities that are
not yet shipped:

1. **SPEC-035 `pattern-arg` input mode** — needed so the pack can supply an ast-grep
   pattern inline (via `rule.pattern`) without a rule-file on disk. Without `pattern-arg`,
   the rule either requires a checked-in `.yaml` rule file (possible but less clean) or
   abuses the existing input modes. Sequence AFTER SPEC-035.

2. **BUNDLE-009 absence semantics (OQ-7)** — the "must be absent" enforcement contract for
   detecting a forbidden pattern's presence-as-violation maps to the same mechanism that
   BUNDLE-009 OQ-7 is resolving: how absence is performed structurally. The thin-executor
   rule is a concrete use case that should inform OQ-7's resolution rather than pre-empt it.
   Sequence AFTER OQ-7 is resolved and its spec is written.

## Solution

Author the rule in backstop's own self-dogfood pack once the upstream capabilities exist.
Suggested sequencing:

1. SPEC-035 lands (`pattern-arg` input mode available).
2. BUNDLE-009 OQ-7 resolves and its spec ships (absence/forbidden-pattern semantics defined).
3. Author the rule in `backstop/go-standards` (or `backstop/thin-executor` if a dedicated
   pack is warranted by then):
   - One ast-grep rule per forbidden-pattern category (or a compound rule if ast-grep
     supports OR patterns at rule level).
   - `paths.include`: the five gate-execution packages listed above.
   - `paths.exclude`: test files (`*_test.go`) and the sanctioned-sites allowlist.
   - Severity: warning (loud, not blocking) during eradication; tighten to error after.
4. Add the pack to `backstop.yml` if not already present.
5. Run `./bin/backstop gate --all` to confirm findings surface on the current violations and
   green on the already-eradicated paths.
6. File a follow-on plan task to flip severity to blocking once the eradication backlog
   (ISSUE-018, ISSUE-019, ISSUE-022) clears.

## References

- `backstop.yml` — `backstop/go-standards: local` self-dogfood pack declaration
- `cmd/backstop/testdata/go-toolchain/.backstop/packs/backstop/go-toolchain/` — pack layout reference
- SPEC-035 (`specs/SPEC-035-pack-declared-engines-trusted-allowlist.spec.md`) — `pattern-arg` input mode (REQ-004); this issue sequences after SPEC-035 lands
- BUNDLE-009 (`bundles/BUNDLE-009-stack-aware-traceability.bundle.md`) — OQ-7 (absence/forbidden-pattern probe semantics); this issue is a concrete consumer of that resolution
- BUNDLE-010 / BUNDLE-011 — pack engine model and legacy code-check collapse; strategic context for the thin-executor invariant
- ISSUE-018 — eradicate baked-in code (the violations this rule will surface)
- ISSUE-019 — de-Go packval harness (sanctioned-site exemption until this lands)
- ISSUE-022 — go-shaped package scope selector (another gate-path baked-knowledge site)
- `CLAUDE.md` — "zero baked language/tool knowledge" first invariant and "dogfood rules as packs" enforcement philosophy
- `pkg/gate/step_testverify.go` — sanctioned-site: own test runner, not language-routing
- Eradication audit 2026-06-20 — items D, E, G in the audit map to violations this rule would catch
