# Pack Authoring

Backstop ships zero checks. Every rule that ever fires in your gate came from a pack you
installed — so the moment you want a check that doesn't exist yet, you write a pack. That is
not an escape hatch; it is the primary extension mechanism, and it needs no PR to backstop
core and no release cycle.

This guide walks through writing your first pack: an ordinary findings pack that runs a
static-analysis tool over your code and reports violations. If you haven't read
[concepts.md](concepts.md) yet, the "Packs" and "Thin executor" sections there are the *why*;
this is the *how*.

## What a pack is

A pack is a **git repository** with a `pack.yml` manifest at its root. The manifest declares:

- **Engines** — how a tool gets invoked and what stage of the gate its output belongs to.
- **Rules** — the actual checks, each naming an engine, a rule file, and a risk class.
- **Claims and fixtures** — for every rule, code samples it *must* flag and samples it *must
  not*. `backstop pack test` executes them, so a pack proves its own rules work before anyone
  trusts them.

Consumers install it with `backstop pack add <org>/<pack>@<version>`, which clones the repo at
the matching git tag into gitignored `.backstop/packs/`, validates it, and records it in
`backstop.yml` and `backstop.lock`.

A pack's own tests are the deliverable as much as its rules are. A rule with no fixture
proving it fires is a rule nobody should install.

## Scaffold a pack

```bash
backstop pack new --type engine --language go --slug my-checks
cd my-checks
backstop pack check .
```

`--type` is one of `engine` (your own validator script *is* the logic), `mechanism` (wrap an
existing tool), or `toolchain` (bundle a language's native build/test/lint passes). All three
scaffold the same valid skeleton — the type only changes the description blurb and hints at
what you're meant to grow into.

`--slug` must match `^[a-z][a-z0-9]*(-[a-z0-9]+)*$`, 2–64 characters. `--language` is a
required *documentation* field; backstop never uses it to reject or route anything.

What you get:

```
my-checks/
  pack.yml
  validators/my-checks.sh
  fixtures/valid/example.txt
  fixtures/invalid/example.txt
```

The scaffolded pack is genuinely green: `pack check` and `pack test` both pass out of the box,
with no external tool to install. Its sample rule discriminates for real — the invalid fixture
carries a marker string, the valid one doesn't.

> **The scaffold enforces nothing at gate time, by design.** Its sample rule declares no
> `input_scope`, so at gate time the validator is handed the project *root directory* as its
> only argument and skips it. That keeps a fresh pack from failing anyone's build before you've
> written a real check — but don't mistake the green for coverage. Replace the sample.

## The manifest

Here's a real, installed, working pack — `backstop-ai/cobra-cli-standards`, trimmed to one
rule. This is the shape most first packs want: semgrep rules over source files.

```yaml
name: backstop-ai/cobra-cli-standards
version: "0.2.1"
language: go
archetype: enforcement
description: "Enforcement standards for cobra-based Go CLIs."

content:
  ruleset:
    version: "0.1.0"
    rules:
      - id: cli-core-exit-code-discipline
        standard: "os.Exit only in main() — commands return errors; the root owns exit codes"
        rule_path: rules/core/cli-core.yml
        risk_class: correctness
        engine: semgrep
        claims:
          - id: clm-005
            text: "os.Exit outside main() is rejected"
            fixtures:
              positive:
                - fixtures/rules/valid/cli-005-exit-in-main.go
              negative:
                - fixtures/rules/invalid/cli-005-exit-in-helper.go
```

Field by field:

| Field | Notes |
| --- | --- |
| `name` | `<org>/<pack>`. **This is the install identity** — the directory, the `backstop.yml` key, and the lock key all read the name the manifest declares, not the coordinate someone typed. |
| `version` | Strict semver. Must match the git tag it's published under (see [Publishing](#publishing)). |
| `language` | Required, documentation only. `any` and `neutral` are honest values for language-agnostic packs. |
| `archetype` | `enforcement`, `code`, or `recipes`. Rules packs are `enforcement`. |
| `content.ruleset.rules[].risk_class` | One of `security`, `correctness`, `style`, `perf`. Required per rule. |
| `content.ruleset.rules[].engine` | Which engine runs this rule — a base engine name, or a key from your own `engines:` block. |
| `content.ruleset.rules[].rule_path` | Pack-relative path to the tool's own rule file. Must exist. |
| `content.ruleset.rules[].claims` | What the rule promises, with the fixtures that prove it. |

Note `engine: semgrep` with no `engines:` block anywhere. That's a **base engine** — backstop
embeds a small neutral substrate declaring four generic engines (`semgrep`, `ast-grep`,
`sandbox`, `config-file`) so an ordinary rules pack doesn't have to restate how to invoke
semgrep. Name one of those and you're done.

### Declaring your own engine

You need an `engines:` block when the base engines don't describe your tool — a different
invocation, a different gate stage, or a converter that normalizes non-SARIF output.

```yaml
engines:
  ast-grep-substantiveness:
    command: ast-grep scan --json
    input_mode: config-file
    input_flag: --config
    scope_kind: file-args
    category: opinion
    gate_type: substantiveness
    convert: ast-grep/to-sarif.sh
    provision:
      tool: ast-grep
      version: 0.43.0
```

The load-bearing fields:

- **`command`** — the tool and its flags. Backstop appends the targets itself; never hardcode
  a path or a target here.
- **`input_mode`** — `rule-flags` (one `--config` per rule file, semgrep's shape),
  `config-file` (one config for the whole run), `pattern-arg` (the pattern is an argument),
  or `none` (the executable *is* the logic).
- **`scope_kind`** — `file-args` (run against the scoped files) or `project-wide` (run once
  against `project_target`).
- **`gate_type`** — which stage of the kill chain this engine feeds: `lint`, `build`, `test`,
  `findings`, `coverage`, `substantiveness`, or `contracts`. Backstop routes on *this*, never
  on a pack name. An unrecognized value fails loudly.
- **`convert`** — pack-relative script that turns the tool's native output into SARIF. Omit it
  if the tool emits SARIF already.
- **`provision`** — declare this only for a tool backstop should download and pin for you.

> **Name your engine key distinctly.** Pack bindings *override* the embedded base engines by
> key. A pack that redefines the bare `semgrep` key silently changes engine resolution for
> every other pack installed in the same consumer project — a defect that shows up in someone
> else's repo, not in your pack's tests. `backstop-ai/ci-workflows` names its binding
> `semgrep-ci` for exactly this reason.

### Which tools you can name

Backstop provisions and pins the tools it introduces on the user's behalf. That trust floor
applies only to bindings carrying a `provision:` block, and it currently covers:

- `semgrep` — pinned at `1.156.0`
- `ast-grep` — pinned at `0.43.0`
- `grep` — pinned as "present" (a POSIX tool whose version backstop doesn't introduce)

Declaring `provision:` for anything outside that list is an exit-2 config error, and so is
pinning an allowlisted tool to a version other than the one above.

Any *other* tool — `go`, `golangci-lint`, your team's internal linter — is fair game to name in
a `command:`, but backstop won't install it. It must already be on `PATH` in the consumer's
environment, or the gate exits 2. That's the honest tradeoff: no unpinned downloads, no silent
version drift.

## Rules, claims, and fixtures

Every claim carries fixtures, and the polarity is the part people get backwards:

- **`positive`** — the rule must **not** fire. This is compliant code. A positive fixture that
  triggers the rule is a false positive, and `pack test` fails.
- **`negative`** — the rule **must** fire. This is the violation the rule exists to catch. A
  negative fixture that doesn't trigger fails too, with a hint that you may be shipping an
  untestable claim.

The installed packs put them under `fixtures/rules/valid/` and `fixtures/rules/invalid/` — a
convention, not a requirement; only the declared paths matter.

**Security rules carry an extra burden.** If `risk_class: security`, phase 6 requires at least
one fixture marked `bypass_attempt: true` — a deliberate attempt to sneak past the rule — and
forbids security claims from sharing fixtures with each other:

```yaml
        risk_class: security
        claims:
          - id: clm-001
            text: "Secret-named string flags are rejected"
            fixtures:
              positive:
                - fixtures/rules/valid/cli-001-prompted-secret.go
              negative:
                - fixtures/rules/invalid/cli-001-secret-flag.go
                - path: fixtures/rules/invalid/cli-001-bypass-shorthand-flag.go
                  bypass_attempt: true
```

Write fixtures from real code you've actually seen fail, not from imagination. A fixture that
was never observed to break is a fixture that doesn't falsify anything.

## ⚠ Sharp edge: slash-bearing path patterns are inert

**This one has bitten every pack in the ecosystem, including backstop's own.** Read it before
you write a `paths:` block.

Semgrep's `paths.include` / `paths.exclude` behave differently depending on how the tool is
invoked. Handed a **directory** to walk, a pattern like `cmd/backstop/*.go` matches fine.
Handed an **explicit list of files** — which is what `backstop gate` does on every run, since
diff scoping means backstop already knows exactly which files changed — the same pattern
matches **nothing**.

Measured against real semgrep 1.156.0, with the same two explicit file targets:

```yaml
include: "cmd/backstop/pack_gate*.go"      # → 0 findings
include: "**/cmd/backstop/pack_gate*.go"   # → 0 findings
include: "/cmd/backstop/pack_gate*.go"     # → 0 findings
include: "pack_gate*.go"                   # → 2 findings  ✓
```

Only the slash-free spelling survives. The consequences differ by key, and both are bad:

- **A slash-bearing `include` makes the rule dark.** It never runs on the everyday gate. Green,
  silently, forever — the vacuous-green class.
- **A slash-bearing `exclude` fails open.** The exemption doesn't apply, so the rule fires on
  files you explicitly meant to exempt, and reads to your users as a false RED they thought
  they'd already suppressed.

> **Do not apply semgrep's own deprecation remedy.** Semgrep prints a warning on these patterns
> recommending exactly the `**/`-prefixed and `/`-anchored rewrites. Both were measured dark
> under explicit-file dispatch and change nothing. The measurement above is the authority, not
> the tool's warning text.

The fix is to restate the scope with a **slash-free, single-segment** pattern — `"*_test.go"`,
`"handler*.go"` — which is honored under both dispatch shapes. Where no slash-free spelling
preserves the directory scope you wanted, your real choice is a wider blast radius or no path
scoping at all. That's a genuine authoring judgement call, which is why backstop warns rather
than errors.

`pack check` and `pack test` surface this for you as a non-blocking advisory at phase 2:

```
WARN [phase2-coherence/path-scope-dispatch] semgrep rule "cli.core.exit-code-discipline"
declares a paths.exclude pattern containing a "/": "**/*_test.go". A slash-bearing path
pattern is unsatisfied under the gate's explicit-file dispatch in EVERY spelling, so this
exclusion FAILS OPEN — the rule fires on files the pack explicitly meant to exempt, which
reads as a false RED the author believes they already suppressed.
```

There's a second advisory, `path-scope-fixture-mask`, for the nastiest version of this: your
rule's real patterns are dark, but a broad "hook" pattern still matches your own fixtures — so
`pack test` stays green while the rule does nothing in production. If you see it, your test
suite is lying to you.

## Validate: `pack check`

```bash
backstop pack check ./my-checks
```

Runs the manifest and metadata phases — everything except fixture execution. Fast, no tool
required, no engine invoked. Run it constantly while editing the manifest.

```
status: pass
- phase1-structural: pass
- phase2-coherence: pass
- phase4-archetype: pass
- phase5-layer: pass
- phase6-risk-class: pass
```

What each phase is looking for:

| Phase | Checks |
| --- | --- |
| 1 — structural | Required fields, semver version, valid archetype and risk classes, every referenced rule file / fixture / validator actually exists. |
| 2 — coherence | Unique rule IDs, cross-references resolve, path-scope advisories. |
| 4 — archetype | Archetype-appropriate content (an `enforcement` pack must not ship scaffolds; a `recipes` pack must declare recipes). |
| 5 — layer | Engine resolution, layer/category consistency, validator presence. |
| 6 — risk class | Security-rule obligations: bypass fixtures, no shared fixtures. |

A failure in an earlier phase skips the later ones, so fix from the top down.

## Test: `pack test`

```bash
backstop pack test ./my-checks
```

Everything `check` does, **plus phase 3** — which actually runs your engine against every
declared fixture and asserts the polarity. This is the step that proves your rules fire.

```
status: pass
- phase1-structural: pass
- phase2-coherence: pass
- phase3-fixtures: pass
- phase4-archetype: pass
- phase5-layer: pass
- phase6-risk-class: pass
```

Phase 3 runs your engine sandboxed, so it needs the tool available (or provisionable) and your
fixtures on disk.

> **macOS: pass an absolute path.** On darwin, `backstop pack test ./relative/path` can fail
> every fixture with `sandboxed run (stdout) failed: exit status 71` — the sandbox profile
> rejects a relative pack directory. The same command with an absolute path passes. Use
> `backstop pack test "$(pwd)"` from inside the pack until this is fixed.

## What a finding actually does

Your rule fires. Does the consumer's build fail? **You decide, on the wire**, via SARIF
severity:

- **`level: warning`** — non-blocking. Reported loudly, never fails the gate.
- **`level: error`, or no level at all** — blocking. Fail-closed: an unlabeled finding is
  treated as serious, so a pack can't disable enforcement by declaring nothing.

This is "loud ≠ blocking" expressed as a contract. Use `warning` deliberately, for a capability
the consumer hasn't adopted yet or a signal you want visible without gating. Use `error` for
defects and broken promises.

One mechanical detail worth knowing, because it has bitten before: **semgrep declares severity
only on the rule descriptor**, never on individual results. Backstop reads the result's own
`level` first, then falls back to the producing rule's
`tool.driver.rules[].defaultConfiguration.level`. So `severity: WARNING` in your semgrep rule
does reach the gate correctly — but if you write your own `convert` script, emit the level in
one of those two places or every finding will block.

## Iterate against a real project

The fastest loop is to install your in-progress pack as a **local** pack in a project you
actually care about:

```bash
cd ~/src/my-project
backstop pack add ../my-checks     # a filesystem path, not org/name
backstop gate --all
```

The path must start with `./`, `../`, or `/` — a bare `my-checks` is read as a remote
`org/pack` coordinate, not a directory.

A local install records `source_type: local` and a project-relative path in `backstop.lock`.
After you edit the pack in place, refresh its lock entry:

```bash
backstop pack relock ../my-checks
```

Note the asymmetry: `relock` takes a filesystem **path**, while `remove` / `update` / `upgrade`
take a pack **name**. Guessing wrong errors loudly rather than silently, but it's a known rough
edge.

Watch for false positives on real code the way you'd watch for missed detections. A rule that
fires on legitimate patterns will get waived, and a waived rule enforces nothing.

## Publishing

Packs distribute as ordinary git repositories. There's no registry to submit to and nothing to
publish anywhere:

1. Push the pack to its own repo — `<org>/<pack-name>` on GitHub.
2. Bump `version:` in `pack.yml`.
3. Tag the commit `v<version>` — the tag must be `v` plus the exact manifest version.
4. Push `main` **before** the tag, so the tag is reachable from the default branch.

```bash
git commit -am "release 0.2.0"
git push origin main
git tag v0.2.0 && git push origin v0.2.0
```

Consumers then install it:

```bash
backstop pack add my-org/my-checks@0.2.0
```

**The tag is the version.** `pack add` clones at the tag and then runs an identity gate that
refuses the install if the cloned `pack.yml`'s version doesn't equal the tag, if the manifest
is missing or unparseable, or if its name is unusable. A version bump with no matching tag is
simply not installable; a tag whose manifest disagrees is refused before any consumer state is
touched.

The manifest's `name:` is the install identity, not the coordinate the user typed. If you clone
`my-org/checks` and its manifest says `name: my-org/my-checks`, the install succeeds under the
manifest name and reports the divergence loudly. Keep them in sync anyway — the divergence is a
diagnostic you don't want your users reading.

Private packs work identically. A pack in a private repo your team can clone needs no special
treatment; the "publish" step is just `git push`.

## Beyond a first pack

Three capabilities worth knowing exist, none of which a first findings pack needs:

**`producer:` scripts.** A findings engine may declare a pack-relative script that runs
*un-sandboxed* in place of the command's tool, letting the pack own something core deliberately
doesn't. `backstop-ai/go-toolchain` uses one because `go build` writes its located diagnostics
to stderr while backstop captures stdout by design — so the pack folds stderr into stdout
itself rather than backstop growing Go-specific knowledge.

**Non-findings gate types.** `gate_type: coverage` routes to a separate coverage-records
channel instead of the SARIF findings channel; `substantiveness` and `contracts` feed their own
dedicated gate steps. These are how a pack teaches backstop what "coverage" means for a stack
it knows nothing about.

**`recipes:`.** A `recipes` archetype pack materializes parameterized starter files into a
consumer repo via `backstop recipe apply <pack>:<recipe>@<version> --param k=v`.
`backstop-ai/ci-workflows` ships four CI gate-workflow recipes plus the semgrep rules that keep
what they scaffold from being silently undone. Recipes write files; they never call
provisioning APIs.

## Where to go next

- **[cli-reference.md](cli-reference.md)** — every `pack` subcommand and flag in full.
- **[concepts.md](concepts.md)** — why packs are external, and what the gate does with your
  findings.
- **[getting-started.md](getting-started.md)** — the consumer-side tutorial, if you want to see
  a pack from the other end.
