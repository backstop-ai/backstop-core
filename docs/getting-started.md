# Getting started

Five minutes, one command, one real finding in your own code.

Backstop ships with zero baked-in knowledge of any language or tool. It enforces nothing until you
add a **pack** — a versioned bundle of checks that lives in its own repo and declares which tools to
run. This walkthrough adds one published pack to a small Go project and watches the gate catch
something real.

## Before you start

You need:

- The `backstop` binary — grab a platform build from the
  [releases page](https://github.com/backstop-ai/backstop-core/releases), or build from source with
  `make build` (drops it at `./bin/backstop`).
- Whatever toolchain the pack you add expects on your `PATH`. The Go pack used below shells out to
  `go` and `golangci-lint`. Backstop auto-provisions the tools it introduces itself (semgrep,
  ast-grep) at pinned versions, but it will not install your language's toolchain for you — and it
  tells you plainly when something is missing rather than quietly skipping the check.

## A project to point it at

Any project works. For this walkthrough, a two-file Go module — a map-backed inventory store with
one honest bug planted in it:

```go
// inventory.go
func (s *Store) DumpTo(path string) error {
	f, _ := os.Create(path)
	defer f.Close()
	for sku := range s.items {
		f.WriteString(sku + "\n")
	}
	return nil
}
```

Three swallowed errors, no compiler complaint, tests green. This is exactly the shape of code an
agent produces at 2am and a reviewer waves through.

## One command

`backstop init` sets up config, layout, `.gitignore`, and packs in a single prompt-free invocation.
`--no-sdlc` keeps the footprint minimal — no artifact directories, just pack-declared checks — and
`--pack` installs a pack in the same pass:

```console
$ backstop init --no-sdlc --pack backstop-ai/go-toolchain@1.8.0
backstop init — /private/tmp/inv-onecmd
  git        delivered          initialized a git repository at /private/tmp/inv-onecmd
  config     delivered          wrote backstop.yml for the pack-only profile, turning off the five gate dimensions that hard-error without an artifact layout (test_verification, coverage_threshold, contract_signature, test_substantiveness, artifact_status_drift). GAP: turning off coverage_threshold forfeits coverage enforcement even though your packs may still emit coverage records, and the spec-independent coverage floor that would replace it does not exist yet — there is no configuration key to set, so nothing is being left undone here
  packs      delivered          installed backstop-ai/go-toolchain@1.8.0
  gitignore  delivered          appended to .gitignore, leaving every pre-existing line untouched: .backstop/baseline.json, .backstop/pack-config-provenance.json, cover.out. Pack-derived entries come from each engine's declared stdout_artifact, which names only what that engine writes for the gate to read. Anything else a toolchain leaves on disk — dependency directories, native build output, local caches — is not covered and stays yours to ignore; backstop does not guess at it.
  scaffold   skipped            no source file was scaffolded, because --scaffold was not supplied. To scaffold one later, run `backstop recipe apply <pack>:<recipe>@<version>` with a scaffold recipe an installed pack declares. Not every pack ecosystem ships one, so a skipped scaffold is not an error
  toolchain  delivered          pack backstop-ai/go-toolchain: the declared build entrypoint "go build" ran in /private/tmp/inv-onecmd and exited 0
  toolchain  delivered          pack backstop-ai/go-toolchain: the declared test entrypoint "go test -coverprofile=cover.out" ran in /private/tmp/inv-onecmd and exited 0
  baseline   capability absent  no local baseline was seeded: the seeding machinery does not exist yet and is owned by ISSUE-056. Nothing is broken and nothing is owed — the gate runs without it, you simply do not get a local ratchet until that lands
  ci         skipped            no CI was wired, because --ci was not supplied. To wire it later, run `backstop recipe apply <pack>:<recipe>@<version>` with a CI recipe an installed pack declares
  observe    delivered          ran the gate once and noticed 2 finding(s), grouped by dimension — pack_lock_verification: 0, artifact_validation: 0, pack_engines: 2, [every other dimension: 0]. These were already in your project; init does not treat them as something it broke

[followed by the same tally as a per-dimension table]
```

Two things worth noticing in that wall of text, because they are the whole personality of the tool:

- **Every step reports.** `delivered`, `skipped`, `capability absent` — with the reason. Nothing is
  silently omitted, and a step that couldn't run says so instead of passing. Even a known gap in
  backstop itself (baseline seeding) is named with its tracking issue rather than glossed over.
- **The 2 findings are observation, not failure.** `init` exits 0. Pre-existing violations in a
  project you just started governing are never treated as something init broke. You will not be
  told your whole codebase is on fire on day one.

Prefer to do it in two steps, or adding a pack to a project that's already initialized? Same result:

```bash
backstop init --no-sdlc
backstop pack add backstop-ai/go-toolchain@1.8.0
```

```console
Added backstop-ai/go-toolchain@1.8.0 (hash: 7fbbe158a0964e236bda5261d6008af29715841450742b89bedc64e077331cf3)
```

The pack installs into gitignored `.backstop/packs/` — the same way `node_modules` works. What gets
committed is `backstop.yml` (which packs, which versions) and `backstop.lock` (the content-hashed
record). Anyone who clones your repo runs `backstop pack install` and gets the same checks at the
same versions, verified against the recorded hash.

The entire config backstop wrote for you:

```yaml
project: inv-onecmd
packs:
    backstop-ai/go-toolchain: 1.8.0
enforcement:
    policy:
        artifact_status_drift:
            level: "off"
        contract_signature:
            level: "off"
        coverage_threshold:
            level: "off"
        test_substantiveness:
            level: "off"
        test_verification:
            level: "off"
```

## See what it caught

```console
$ backstop gate
Gate Results
────────────────────────────────────────────────────────────
backstop version dev
schema cohort: 49f08ac21d140c28440ee7998af00e3079bdc73ab6de5e7d21655a55eb8e0093
artifact root: /private/tmp/inv-onecmd (configured: false)
────────────────────────────────────────────────────────────
Gate running against 7 changed files (use --all for full sweep).
────────────────────────────────────────────────────────────
  pack_lock_verification    pass  (1ms)
  artifact_validation       pass  (2ms)
  pack_engines              fail  (1308ms)  (2 violations)
  test_verification         skipped  (disabled by enforcement policy (level: off))
  test_substantiveness      skipped  (disabled by enforcement policy (level: off))
  coverage_threshold        skipped  (disabled by enforcement policy (level: off))
  contract_signature        skipped  (disabled by enforcement policy (level: off))
  artifact_status_drift     skipped  (2ms)  (disabled by enforcement policy (level: off))
  artifact_status_drift_advisory pass
  requirement_traceability  pass
  requirement_traceability_advisory pass
  waiver_resolution         pass  (clean — no active waivers)
  baseline_comparison       skipped  (superseded by per-dimension enforcement policy)
  ledger_integrity          skipped  (ledger not implemented)

  pack_engines violations:
    - [backstop-ai/go-toolchain/errcheck] Error return value of `f.Close` is not checked (inventory.go)
      ↳ to waive: @waiver:backstop-ai/go-toolchain/errcheck:accepted-risk:2026-11-17
    - [backstop-ai/go-toolchain/errcheck] Error return value of `f.WriteString` is not checked (inventory.go)
      ↳ to waive: @waiver:backstop-ai/go-toolchain/errcheck:accepted-risk:2026-11-17
────────────────────────────────────────────────────────────
  Steps: 6 passed, 1 failed, 7 skipped, 0 warned
  Total violations: 2

FAIL
```

Exit code 1. Read that output closely:

- **Every violation names the pack that owns it.** `backstop-ai/go-toolchain/errcheck` — not an
  anonymous "lint error". You know which pack to fix, bump, or drop.
- **Bare `gate` is diff-scoped** (7 changed files here), so it's fast enough to run on every save.
  `backstop gate --all` does the full sweep.
- **The skipped dimensions are the ones `--no-sdlc` turned off.** They say so, with the reason. A
  check that isn't running always tells you it isn't running — the enemy is silent green, not
  failure.
- **Every finding ships its own escape hatch.** That `@waiver:` line is a real, dated token you paste
  above the offending line to accept the risk explicitly, in code, with an expiry. Waivers are a last
  resort, not a config flag — and `waiver_resolution` is itself a gate dimension, so a stale one gets
  caught.

Failing tests come through the same channel, attributed the same way — the pack declares `go test`
as an engine, so a broken assertion shows up as a `backstop-ai/go-toolchain/go-test` violation right
alongside the lint findings.

## Fix it, watch it go green

Handle the errors for real:

```go
func (s *Store) DumpTo(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	for sku := range s.items {
		if _, err := f.WriteString(sku + "\n"); err != nil {
			_ = f.Close()
			return err
		}
	}
	return f.Close()
}
```

```console
$ backstop gate
...
  pack_engines              pass  (2203ms)
...
────────────────────────────────────────────────────────────
  Steps: 7 passed, 0 failed, 7 skipped, 0 warned
  Total violations: 0

PASS
```

Exit 0. That's the loop: write, gate, fix, gate. Wire `backstop gate` into your pre-commit hook, your
CI job, and your agent's task-completion check, and the loop runs itself.

## When something looks wrong

`backstop doctor` diagnoses setup problems in one pass — config discovery, git, installed packs, the
running binary's build identity, whether your packs' declared entrypoints actually execute, and
whether the tools they need resolve on `PATH`:

```console
$ backstop doctor
backstop doctor
  config-present   pass     backstop.yml is discoverable — found at /private/tmp/inv-onecmd/backstop.yml
  config-loads     pass     backstop.yml loads and validates — /private/tmp/inv-onecmd/backstop.yml loads and validates
  git-repository   pass     the project root is a git work tree — /private/tmp/inv-onecmd is inside a git work tree
  packs-installed  pass     declared packs are installed — 1 declared pack(s) are present and parseable
  build-identity   warn     the running binary's build identity — version dev, commit 22c75748881d26196ca9d1094459879f1187b67d-dirty, built 2026-08-19T22:11:10Z, schema cohort 49f08ac21d140c28440ee7998af00e3079bdc73ab6de5e7d21655a55eb8e0093 — no build identity for: version
                            this binary carries no release stamp, so it cannot be told apart from a stale one; install a released build if you did not build it yourself just now
  toolchain-runs   pass     pack-declared test/build entrypoints execute — 2 declared entrypoint(s):
                              pass  backstop-ai/go-toolchain (go-build, gate_type build): `go build`
                              pass  backstop-ai/go-toolchain (go-test, gate_type test): `go test -coverprofile=cover.out`
  engine-tools-present pass     pack-declared findings-engine tools resolve on PATH — 2 required engine tool(s):
                              pass  go (backstop-ai/go-toolchain, engine go-build)
                              pass  golangci-lint (backstop-ai/go-toolchain, engine golangci)
  artifact-layout  pass     artifacts sit where the resolved root expects them — every artifact sits in the directory expected for its kind under /private/tmp/inv-onecmd
```

Each check reports `pass`, `warn`, `fail`, or `skipped` — and a skip names the check that owns its
condition, so a chain of unmet prerequisites never reads as an unrelated failure.

(The `build-identity` warn above is honest: this run used a locally built `dev` binary rather than a
release. On a downloaded release build it passes.)

## What backstop did *not* do

Worth stating plainly, because it's the design:

- It did not guess your language. Go support came entirely from `backstop-ai/go-toolchain` — a pack,
  in its own repo, at a pinned version. There is no built-in Go path in the binary. Adding a new
  language means adding a pack, never patching backstop.
- It did not make you hand-write config. The fifteen-line `backstop.yml` above is the whole thing,
  and `init`/`pack add` maintained it for you.
- It did not declare your codebase broken. It found two real errors in one function and stopped.

## Next steps

- [Concepts](concepts.md) — why the gate is shaped this way, what packs and dimensions actually are,
  and what "an AI agent discipline framework" means in practice.
- [CLI reference](cli-reference.md) — every command and flag: `gate`, `pack`, `waiver`, `baseline`,
  `artifact`, `recipe`, `doctor`.
- [Pack authoring](pack-authoring.md) — write your own checks. Your team's conventions become a
  versioned pack you install like any other.
- [Artifact workflow](artifact-workflow.md) — drop `--no-sdlc` and adopt the full track: `issue →
  plan` for reactive work, `bundle → spec → plan` for features, with the gate verifying that what
  shipped matches what was promised.
