---
title: Packs
layout: default
permalink: /pack/guide/
hero_question: "Packs: a guide"
hero_lede: "Use a pack when a rule can be checked deterministically. Leave genuine judgment calls to the agent or reviewer."
---

## 1. Pack or not {#pack-or-not}

Use a pack when a rule can be checked deterministically. Leave genuine judgment calls to the agent or reviewer.

A pack can apply everywhere or to one repository. Make it as specific as the standard requires.

[What is a pack?](/pack/) is the noun. [Published packs](/pack/examples/#choose-a-pack) if a maintained pack already owns the standard.

## 2. Author a pack {#author-a-pack}

Author the pack in its own repository. Do not vendor it into core.

### Scaffold the pack

<pre><code>backstop pack new --type engine --language go --slug my-standard</code></pre>

`--type` is `engine`, `mechanism`, or `toolchain`. The scaffold writes a valid `pack.yml`, a sample sandbox validator, and a positive/negative fixture pair that can pass check, test, and the gate.

### Choose a pack type

All three types emit the same valid engine-pack skeleton. The type is the growth path, not a different file format.

- **engine** — the validator executable is the enforcement logic. Choose this when the check is your own script or binary and does not wrap a language toolchain.
- **mechanism** — wrap one native tool or command as an engine. Choose this when the rule is "run this tool and convert its output."
- **toolchain** — bundle a language's native build, test, and lint passes as engines. Choose this when the pack owns a stack's verification suite, like `go-toolchain` or `typescript-toolchain`.

### Understand the layout

The pack is a directory. `pack.yml` sits at the root. Every path in the manifest (`convert`, `validator`, fixture files, `rule_path`) is relative to that directory. A missing file fails `pack check`.

`pack new` writes:

```
my-standard/
  pack.yml
  validators/my-standard.sh
  fixtures/valid/example.txt
  fixtures/invalid/example.txt
```

Replace the sample in place. A native-tool pack keeps the convert script next to the engine it serves:

```
pack.yml
grep/to-sarif.sh
ast-grep/to-sarif.sh
testdata/fixtures/
```

A toolchain pack often keeps convert and produce scripts under `scripts/`. Rule configs declared by `rule_path` also live in the pack. There is no second copy in core.

### Add the engine

An engine executes the rule. It is a runner. Semgrep, ast-grep, a shell script, and a native toolchain linter are examples, not the list. Packs are designed to support pretty much anything; that is why converters exist. The pack declares the engine. Many rules can share one engine.

The scaffold writes a sandbox engine: a shell validator, no external tool.

```yaml
engines:
  my-standard-engine:
    command: ""
    input_mode: none
    scope_kind: file-args
    gate_type: findings
```

Replace that sample with the real runner. If the engine is a linter, the pack supplies the custom rules (`pattern`, `rule_path`).

```yaml
engines:
  ast-grep-contracts:
    command: ast-grep run --json
    input_mode: pattern-arg
    convert: ast-grep/to-sarif.sh
    provision:
      tool: ast-grep
      version: 0.43.0
```

If the engine invokes a tool, pin it. Tool pins are trust declarations, not installers.

### Add conversion where needed

When the engine runs a native tool, the pack owns the conversion. `convert` is a pack-relative script. It reads the tool's stdout and emits SARIF. Core does not parse `grep`, `go test`, or `ast-grep` output.

```yaml
engines:
  grep:
    command: grep -rn -H -I
    input_mode: pattern-arg
    convert: grep/to-sarif.sh
    provision:
      tool: grep
      version: "*"
```

The script lives in the pack: `grep/to-sarif.sh`, `ast-grep/to-sarif.sh`, `scripts/build-to-sarif.sh`. Unparseable input must fail the convert, not be dropped. Most converters emit SARIF. A coverage engine converts to coverage records instead.

If the tool already emits SARIF, omit `convert`.

The sample sandbox validator has no convert: the validator is the logic.

<div class="pack-model">
<dl>
<dt>Claim</dt>
<dd>The definition. What must be true.</dd>
<dt>Rule</dt>
<dd>The implementation. How that requirement is checked.</dd>
<dt>Engine</dt>
<dd>The execution mechanism. The deterministic machinery that runs the rule.</dd>
<dt>Fixtures</dt>
<dd>Proof of the implementation. Known-positive and known-negative cases that demonstrate the rule behaves correctly.</dd>
</dl>
<p>The claim is the definition; the rule is its implementation; the engine executes that implementation. Fixtures exercise the rule against known cases to prove the implementation behaves correctly.</p>
</div>

### Declare exit status

<!-- backstop-claim: CLAIM-038 -->
A findings engine's output, not its exit status, is its contract: a tool may exit non-zero because it found something, and any non-empty output is parsed as findings whatever the status. The exit status decides one case, an empty output.

Each engine has a set of exit statuses that mean "ran and found nothing". The set is `[0]` unless the engine declares `success_exit_codes`. When the tool's raw output is empty and its exit status is not in the set, the engine has failed. Backstop reports it as a configuration error (exit 2), never as a clean pass. Raw output means the `stdout_artifact` file when one is declared, otherwise stdout, and it is judged before any `convert:` runs. A convert therefore cannot turn a crashed tool's empty output into an empty SARIF result.

Some tools exit non-zero with empty output on a clean run. `grep` exits 1 when nothing matches. Declare that status, even when the engine has a convert:

```yaml
engines:
  grep:
    command: grep -rn -H -I
    # grep exits 1 with empty output when nothing matches, a clean result
    success_exit_codes: [0, 1]
```

The value must be a non-empty list of distinct integers from 0 to 255 that includes 0. Omitting the key, or setting it to null, gives the default `[0]`. A run that fails without any exit status is never accepted.

An engine with `crash_guard: true` uses the same set: a listed status with no findings is clean, and any other status with no findings is a crash. Without a declared set its behaviour is unchanged.

`command:` runs with the project root as its working directory. It is never resolved against the pack directory, so `command: sh tool.sh` with `tool.sh` in the pack cannot find the script at gate time. Ship such a script as `producer: tool.sh` instead; `producer:` is resolved against the pack directory. `backstop pack check` and `backstop pack test` fail a `command:` token that names a pack file (check `command-pack-relative-file`), except on engines that declare `producer:`.

**Behavior change (next minor release):** findings engines that exit non-zero with empty output now fail, with a message naming the status and the `success_exit_codes` line to declare when that status is a legitimate no-findings result.
<!-- /backstop-claim -->

### Define claims

A claim is the definition of what must be true. Declare it on the rule with an `id`, `text`, and both fixture polarities. There is no separate claims list.

### Implement rules

A rule implements the deterministic check for that claim. It names how a violation is reported. `risk_class` is `security`, `correctness`, `style`, or `perf`.

```yaml
content:
  ruleset:
    version: 0.1.0
    rules:
      - id: my-standard-sample
        engine: my-standard-engine
        validator: validators/my-standard.sh
        risk_class: correctness
        claims:
          - id: my-standard-clm-001
            text: "Sample enforced property — replace with your own claim."
            fixtures:
              positive:
                - fixtures/valid/example.txt
              negative:
                - fixtures/invalid/example.txt
```

The sample validator exits 0 on a clean file. It prints a message naming the target and exits non-zero when it fires. Replace the marker check `BACKSTOP-SAMPLE-VIOLATION` with the real detection.

### Add fixtures

Fixtures are proof of the implementation. Every claim needs both polarities. Positive is known-good: the engine must stay silent. Negative is known-bad: the engine must fire.

The sample positive file is clean. The sample negative file carries `BACKSTOP-SAMPLE-VIOLATION`. Rewrite both files when you replace the sample rule. Iteration belongs inside the pack rather than in every consuming repository.

### Scope the rules

Set `input_scope` to `single-file` or `multi-file` when the rule should inspect consumer code. Without one, the sample rule only demonstrates pack check/test behavior.

### Run checks and tests

`pack check` validates the manifest. It does not run fixtures.

<pre><code>backstop pack check ./my-standard</code></pre>

`pack test` runs the engine against the declared fixtures and fails if the pair does not discriminate.

<pre><code>backstop pack test ./my-standard</code></pre>

### Try the pack in a repository

Install the local pack in a consumer repository and run that repository's gate.

<pre><code>backstop pack add ./my-standard</code></pre>

### Participate

Backstop is not accepting outside contributions now. That is planned.

<!-- backstop-journey-link: JLINK-020 -->
[How to participate](/contributing/#contribution-paths)

## 3. Publish a pack {#publish-a-pack}

To publish a pack today:

1. Make sure the pack passes its local checks/tests.
2. `pack.yml` must declare a valid semantic version.
3. Create a Git tag matching that version (for example, pack version `1.2.0` → tag `v1.2.0`).
4. The tag must point to the exact commit being published.

That tagged Git commit is the immutable version consumers install and pin.

Some Backstop-maintained packs use stronger CI publishing pipelines today, including owner acceptance, version/tag verification, and immutable release checks. These are prototypes, not the canonical publishing model. Shared CI guidance and reusable publishing tooling are coming soon.
