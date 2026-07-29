# Captured semgrep SARIF fixtures — provenance

Every `.sarif` file in this directory is UNMODIFIED tool output. Nothing was
pretty-printed, trimmed, or annotated after capture. Provenance lives here,
beside the fixtures, never inside them — that is what keeps "captured" a true
statement about those files.

Captured 2026-07-28 by PLAN-ISSUE-104 (severity resolution from the SARIF rule
descriptor). The property these fixtures exist to prove: **real semgrep does not
put `level` on the result.** It states the rule's severity on the rule
descriptor, `runs[].tool.driver.rules[].defaultConfiguration.level`, joined to
the result by `ruleId`. A capture here that carried a result-level `level` would
mean the rule or the command drifted, and the premise of the whole lane would
need re-measuring.

## Capture inputs (committed, so the captures are reproducible)

| File | Role |
| --- | --- |
| `capture/rule-warning.yml` | one semgrep rule, `severity: WARNING` |
| `capture/rule-error.yml` | the same rule at `severity: ERROR` |
| `capture/sample.go` | the trivial file the rule matches, EXACTLY ONCE |
| `capture.sh` | the exact commands, re-runnable |

`capture.sh` `cd`s to this directory and passes RELATIVE paths for both the
config and the target. That is load-bearing, not tidiness: semgrep derives the
emitted `ruleId` from the `--config` path, so an absolute config bakes the
capturing machine's home directory into the committed bytes. With the relative
config the id is the portable `capture.capture-sample-panic` — the `capture.`
namespace comes from the rule file's directory, and contains no path fragment
from this machine.

`capture/sample.go` is deliberately minimal so the rule fires on precisely one
line. `packFindingsAtLevel` (cmd/backstop/pack_severity_contract_test.go)
hard-fatals unless the dispatch yields exactly one violation, so a two-match
capture would red the contract test for a reason unrelated to severity.

## Committed fixtures

### descriptor-warning.sarif
- tool: `Semgrep OSS`, `semanticVersion` **1.156.0** (read from the fixture's own
  `runs[0].tool.driver.semanticVersion`, not from what the shell reported)
- command: `semgrep --config capture/rule-warning.yml --sarif capture/sample.go`
  run from this directory
- date: 2026-07-28
- sha256: `292022583cd81b17902e351216e5c10618233830171e2c560276f964560f8ff7`
- shape: 1 result; NO `level` key on the result; descriptor
  `defaultConfiguration.level = "warning"`; `ruleId` = `capture.capture-sample-panic`

### descriptor-error.sarif
- tool: `Semgrep OSS`, `semanticVersion` **1.156.0**
- command: `semgrep --config capture/rule-error.yml --sarif capture/sample.go`
  run from this directory
- date: 2026-07-28
- sha256: `6378740d228272f2a704c65303aa3bceaeb0f11935a150fc4d589c1c399c6cf3`
- shape: 1 result; NO `level` key on the result; descriptor
  `defaultConfiguration.level = "error"`; `ruleId` = `capture.capture-sample-panic`

### descriptor-warning-1.96.0.sarif
- tool: `Semgrep OSS`, `semanticVersion` **1.96.0** — the version backstop
  actually provisions (`pkg/pack/engine/allowlist.go`), which is NOT the 1.156.0
  on PATH. Captured to confirm the fix is correct against the version that runs
  in the gate.
- command:
  `PATH="$VENV/bin:/usr/bin:/bin" $VENV/bin/semgrep --config capture/rule-warning.yml --sarif capture/sample.go`
  run from this directory, where `$VENV` is the pinned venv built below
- date: 2026-07-28
- sha256: `688e923a99a4b4b649d79b7526cd5841f20b3a843357921e2ccce7694a10fd57`
- shape: **identical to 1.156.0** — 1 result, NO `level` key on the result,
  descriptor `defaultConfiguration.level = "warning"`, same result key set
  (`fingerprints, locations, message, properties, ruleId`) and same descriptor
  key set. The defect is therefore not a version artifact.

## Reproducing the 1.96.0 capture — the recipe, and what failed

The PATH scrub is load-bearing, not tidiness. Invoked with the ambient PATH the
1.96.0 CLI delegates to whatever newer `semgrep-core` it finds and emits
`semanticVersion: 1.156.0` — a capture that LOOKS pinned, is labelled 1.96.0 in
its filename, and was produced by a different engine. Always assert the captured
`semanticVersion` equals the version you claim; this one was asserted before it
was committed.

Working recipe (executed 2026-07-28):

```
uv venv -p 3.11 .venv196
uv pip install --python .venv196/bin/python semgrep==1.96.0 'setuptools==70.3.0'
PATH="$PWD/.venv196/bin:/usr/bin:/bin" .venv196/bin/semgrep \
  --config capture/rule-warning.yml --sarif capture/sample.go \
  > descriptor-warning-1.96.0.sarif
```

**The setuptools PIN is required, and an unpinned `setuptools` is not enough.**
The first attempt installed `semgrep==1.96.0 setuptools`, which resolved
setuptools **83.0.0** — a version that no longer ships `pkg_resources` — and the
capture died verbatim with:

```
  File ".../site-packages/opentelemetry/instrumentation/dependencies.py", line 4, in <module>
    from pkg_resources import (
ModuleNotFoundError: No module named 'pkg_resources'
```

Downgrading to `setuptools==70.3.0` restored `pkg_resources` (with a
`DeprecationWarning`) and the capture then succeeded on the first try. `uvx` was
reported as failing this same way during planning; the Docker route needs a
daemon that was not running.

## What is deliberately NOT captured here

**The result-level direction.** A real golangci-lint SARIF carrying result-level
severities is ALREADY committed at
`cmd/backstop/testdata/go-toolchain/fixtures/golangci-v2.sarif` (SPEC-034): two
results with `level` (`error` and `warning`) under a driver with only a `name`
and no `rules` array. That is exactly the non-regression shape the descriptor
fallback must not disturb — a producer that states severity on the result and
has no descriptor to fall back to. Capturing a second one would prove nothing
the tree does not already prove.

**The conflict shape** — a result carrying its own `level` while its descriptor
declares a DIFFERENT one. No producer emits it: semgrep omits the result level
entirely, golangci emits no descriptor, and real output never contradicts
itself. Precedence can therefore only be pinned on a constructed input, which
lives in `pkg/check/parsers_test.go`, doc-commented as spec-derived and naming
SARIF §3.27.10. It is not, and must never become, a file in this directory.

**The both-absent floor** — no `level` anywhere. `sarifSampleJSON`
(`pkg/check/fixtures_toolchain_test.go`) already carries EXAMPLE003: no result
level, in a log with no driver rules.
