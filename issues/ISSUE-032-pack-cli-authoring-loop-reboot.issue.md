---
title: "Pack-CLI authoring-loop reboot — modernize stale pack-authoring/validation/scaffold commands for the engine-pack model"
schema_version: issue/v1
number: "032"
delivered_by: PLAN-ISSUE-032

issue:
  id: ISSUE-032
  title: "Pack-CLI authoring-loop reboot — modernize stale pack-authoring/validation/scaffold commands for the engine-pack model"
  type: technical-debt
  status: closed
  created: "2026-06-28"
  closed: "2026-07-08"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
---

# Pack-CLI authoring-loop reboot — modernize stale pack-authoring/validation/scaffold commands for the engine-pack model

## Problem

Backstop's pack CONSUMPTION and INSTALL paths were modernized by BUNDLE-011's engine-dispatch
cutover: the gate runs declared engine commands, packs ship `pack.yml` with a `content.ruleset`
/ `engines:` shape, and `pack add` installs them correctly. The pack AUTHORING,
VALIDATION, and SCAFFOLD tooling was not modernized alongside this. The result is that every
CLI entrypoint for creating and checking packs is broken against the model it is supposed to
support.

The six concrete defects below were verified by reading the source and running
`./bin/backstop` against production packs.

### Defect A — `pack new` scaffolds dead artifacts

`backstop pack new --type rule` writes a `.standard.md` file under `standards/<language>/`
with `schema_version: standard/v1` (`pkg/pack/scaffold.go:82–83`). The native-standards
model is strategically retired (ISSUE-030, BUNDLE-011). `artifact validate` returns
"unrecognized schema_version prefix" for any `standard/v1` file; `LoadManifest` in
`pkg/check` has no live reader for `.standard.md` (the compiled-standards branch is
dead-fed). The artifact produced by `pack new --type rule` is born invalid, unread, and
unvalidated by any production code path.

`backstop pack new --type code` writes a `.recipe.md` under `recipes/<language>/<slug>/`
with `schema_version: recipe/v1`. This format is equally unrecognized by `artifact
validate` and unread by the gate.

Neither `--type rule` nor `--type code` emits a `pack.yml` — the only artifact format
the engine-pack model and `pack add` actually consume. There is no `--type engine`,
`--type toolchain`, or `--type mechanism` pack type available, despite those being the
three shapes of production pack in the tree today.

Source: `pkg/pack/scaffold.go:70–191` — `ScaffoldPack` routes to `scaffoldRulePack`
(`.standard.md` writer) and `scaffoldCodePack` (`.recipe.md` writer); no branch emits a
`pack.yml`. `ValidPackTypes` at `scaffold.go:13–16` registers only `"rule"` and `"code"`.

### Defect B — `pack check` / `pack test` reject production packs via stale phase logic

**`phase5-layer` rejects the `go-standards` pack.** `pkg/packval/phase5.go:22–23` checks
that every rule has `layer` in `[1, 2, 3]` (`"invalid layer"` if 0 or missing). The migrated
`go-standards` pack uses the engine model: rules declare `engine:` not `layer:`. The `layer`
field on its rules is 0 (unset). Every rule in `go-standards` fails `phase5-layer` with
`"invalid layer"`, rejecting a pack that the gate runs successfully on every CI push.

**`phase2-coherence` rejects the `go-toolchain` pack.** `pkg/packval/phase2.go:30–31`
errors `"rule has no claims"` for any rule without a `claims:` array. Engine packs like
`go-toolchain` run native toolchain steps (build, vet, test) that produce SARIF directly;
they have no claimable rule-level fixtures and carry no `claims:`. The coherence phase
treats claimless rules as a hard error, making it impossible to validate any
mechanism/toolchain-archetype pack.

**Combined:** `pack check` and `pack test` cannot validate either of the two production
packs currently installed in backstop-core. The commands designed to gate-keep packs
systematically reject the packs they are supposed to validate.

### Defect C — `pack check` / `pack test` always output JSON, ignoring `--format text`

`cmd/backstop/pack_check.go:16–22` and `cmd/backstop/pack_test_cmd.go:16–22` both contain
the same dead-branch bug:

```go
if format == "" {
    if jsonFlag != nil && *jsonFlag {
        format = "json"
    } else {
        format = "json"  // ← both branches assign "json"
    }
}
```

Both the true and false branches assign `format = "json"`. Running `pack check --format
text` has no effect when `format` enters the `RunE` as `""` (the flag's default). The
`--format` flag is wired and documented but non-functional: the output is always JSON.

### Defect D — `pack list` reports `RULES: 0` for packs with rules

`pkg/pack/distribution/list.go:36–39` defines `listManifest` with a top-level `Rules
[]interface{} \`yaml:"rules"\`` field. Modern packs nest rules under
`content.ruleset.rules:` (the engine-pack `pack.yml` shape). The `listManifest` decoder
finds nothing at the top-level `rules:` key, so `RuleCount` is always 0 regardless of how
many rules the pack declares.

`pack list` also shows `stale` or no-version for local packs whose lock entry was written
by `pack add` with a local path — the lock status computation does not handle the `local`
source type cleanly, surfacing misleading freshness signals.

### Defect E — `artifact new bundle` stamps `bundle/v1`, producing a filename that only `bundle/v2` accepts

`pkg/scaffold/scaffold.go:200–202` emits:

```
number: BUNDLE-NNN
schema_version: bundle/v1
```

The scaffold assigns a `BUNDLE-NNN` number (via the git-tag reservation mechanism), which
the filename builder incorporates into a `BUNDLE-NNN-<slug>.bundle.md` filename. The
`bundle/v1` schema's `filename_pattern` is `^[a-z0-9-]+(\\.epic)?\\.bundle\\.md$` — it
does not allow a `BUNDLE-NNN-` prefix. Only `bundle/v2`'s pattern
(`^(BUNDLE-[0-9]+-)?[a-z0-9-]+(\\.epic)?\\.bundle\\.md$`) accepts numbered filenames.

Every freshly-scaffolded bundle is born invalid against its own declared `schema_version`
and fails `artifact validate` until the author hand-bumps `schema_version: bundle/v1` →
`bundle/v2`. This was hit standing up BUNDLE-012.

### Defect F — no clean local-pack re-lock path

`pack add <path>` when the pack is already declared in `backstop.yml` returns an error
("already installed" or equivalent). `pack update` is a hard no-op for local packs — it
has no mechanism to re-resolve a local source and rewrite the lock `content_hash`. The
only way to regenerate a stale lock entry for a local pack is `pack remove` + `pack add`
— a manual two-step that must be discovered by trial and error. A future `pack install`
can silently mismatch the stale lock content without the author knowing.

## Impact

BUNDLE-012 (R6, OQ-5) needs to author `backstop/bun-toolchain` and was forced to plan
around hand-authoring from the `go-toolchain` template because the CLI scaffolder cannot
produce a valid pack. All six defects above compound: the authoring loop for every future
pack is broken at scaffold, broken at validate, and broken at re-lock. The "scaffold via
CLI" discipline (`feedback_scaffold_via_cli.md`) cannot be honored when the CLI produces
invalid artifacts.

## Solution

Fix the six defects:

**A — `pack new`:** emit a real `pack.yml` (with a `content.ruleset:` / `engines:`
skeleton); add `--type engine`, `--type toolchain`, and `--type mechanism` as first-class
scaffolding targets with appropriate starter shapes; stop writing `.standard.md` and
`.recipe.md` (that deletion is a pre-condition, owned jointly with ISSUE-030 which
removes the `scaffoldRulePack` function, but this issue owns adding the replacement
scaffolder for modern pack types). `ValidPackTypes` in `pkg/pack/scaffold.go` must be
updated to reflect the live types.

**B — `pack check` / `pack test`:** in `phase5-layer`, treat a zero/missing `layer`
field as exempt when the rule carries a non-empty `engine:` field (engine-model rules are
not layer-model rules); in `phase2-coherence`, exempt claimless rules when the pack's
declared archetype is `toolchain` or `mechanism` (or when the rule carries an `engine:`
and no `claims:` array is declared — toolchain commands produce SARIF directly, fixtures
are not applicable).

**C — `pack check` / `pack test` format bug:** in both `pack_check.go` and
`pack_test_cmd.go`, the `else` branch of the `if jsonFlag != nil && *jsonFlag` block must
assign `format = "text"` rather than `"json"`, so `--format text` actually takes effect.

**D — `pack list` rule count:** rewrite `listManifest` in
`pkg/pack/distribution/list.go` to decode `content.ruleset.rules:` for the engine-pack
shape (falling back to top-level `rules:` for any legacy shape). Fix the lock-status
display for local packs to show `current` rather than `stale` when the local path is
unchanged.

**E — `artifact new bundle` schema version:** change `pkg/scaffold/scaffold.go:202` to
emit `schema_version: bundle/v2` so freshly-scaffolded numbered bundles are valid against
the schema their filename requires.

**F — local-pack re-lock:** add a `pack relock <path>` subcommand (or make
`pack update` re-resolve a local source) that re-reads the local pack directory, recomputes
`content_hash`, and overwrites the lock entry in `backstop.lock`. A clean re-lock path
eliminates the manual `pack remove` + `pack add` workaround.

## Resolution

Rebooted the pack-CLI authoring loop for the engine-pack model: fixed packval to parse real
engine packs (string-enum EngineSpec), resolve-gated the check/test layer/claims exemptions
(no vacuous-green), replaced the dead .standard.md scaffolder with an engine-pack scaffolder,
fixed --format text / pack list counts / bundle-v2 / added pack relock, and proved it with a
real new→check→test→gate e2e. Fixed in 6923dba; the old pack-new spec SPEC-011 was deprecated
as superseded.

## References

- `pkg/pack/scaffold.go:13–16` — `ValidPackTypes` registry (`"rule"`, `"code"` only)
- `pkg/pack/scaffold.go:70–141` — `scaffoldRulePack` writes `.standard.md` with `schema_version: standard/v1`
- `pkg/pack/scaffold.go:143–191` — `scaffoldCodePack` writes `.recipe.md` with `schema_version: recipe/v1`
- `pkg/packval/phase5.go:22–23` — `"invalid layer"` error fires when `rule.Layer < 1` (zero/unset = invalid)
- `pkg/packval/phase2.go:30–31` — `"rule has no claims"` error fires for every claimless engine rule
- `cmd/backstop/pack_check.go:16–22` — dead-branch assigns `format = "json"` in both if/else arms
- `cmd/backstop/pack_test_cmd.go:16–22` — same dead-branch bug as `pack_check.go`
- `pkg/pack/distribution/list.go:36–39` — `listManifest.Rules` reads top-level `rules:`, misses `content.ruleset.rules:`
- `pkg/scaffold/scaffold.go:200–202` — `artifact new bundle` emits `schema_version: bundle/v1` with a numbered BUNDLE-NNN filename
- `artifacts/bundle/v1/schema.json:9` — `filename_pattern` rejects `BUNDLE-NNN-` prefix; only v2 accepts it
- `artifacts/bundle/v2/schema.json:9` — v2 `filename_pattern` accepts the numbered prefix the scaffolder emits
- ISSUE-030 — owns deletion of `scaffoldRulePack` + `ResolvePackNumber` (the `.standard.md` writer); this issue owns the replacement scaffolder for modern pack types
- BUNDLE-012 — Track-B companion; OQ-5 (bun-toolchain authoring) forced hand-authoring due to this broken scaffolder
- CLAUDE.md — "scaffold via CLI" discipline; "thin executor / packs-only" strategic direction
