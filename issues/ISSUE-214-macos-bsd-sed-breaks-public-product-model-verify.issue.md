---
title: "macOS BSD sed breaks public product model verify"
schema_version: issue/v1

issue:
  id: ISSUE-214
  title: "macOS BSD sed breaks public product model verify"
  type: bug
  status: open
  created: "2026-10-12"

complexity:
  scope: contained
  uncertainty: known
  risk: safe
---

# macOS BSD sed breaks public product model verify

## Problem

`scripts/tests/public-product-model/pages/extend-reference-contributing.sh` (line 282, inside
`verify_reference_paper_ink_chrome`) edits a scratch copy in place with GNU-only syntax:

```
sed -i 's/{% if paper_kinds contains page_kind %}//g; s/{% endif %}//g' "$tmp/layout.html"
```

GNU sed accepts a bare `-i`. BSD sed (macOS) requires an argument to `-i` (the backup suffix), so it
consumes the script as the suffix and then parses the file path as the script, failing with an error
of the form `sed: 1: "<path>": invalid command code f` (the exact text depends on the path; with a
shorter path it reports an unterminated substitute). The script runs under `set -euo pipefail`, so
`scripts/verify-public-product-model.sh` exits 1 immediately after printing "CLAIM-005 layout PASS".

Consequence in this repo on macOS: the `backstop-ai/bash-toolchain` `bash-test` engine runs this
verifier, so it fails during `backstop gate`. The gate then never reaches lint, tests or coverage on
that machine; and per ISSUE-209 (one engine error discards the rest of the dispatch) the failure
masks the other engines' evidence rather than appearing alongside it. Linux CI (`ubuntu-latest`)
uses GNU sed, so the defect is invisible there.

### Reproduction

Scratch only, no network:

```
mkdir -p /tmp/sed-repro && cd /tmp/sed-repro
printf '{%% if paper_kinds contains page_kind %%}x{%% endif %%}\n' > layout.html
sed -i 's/{% endif %}//g' layout.html; echo "rc=$?"   # macOS: sed error, rc=1; GNU: rc=0
```

Full path: on macOS run `scripts/verify-public-product-model.sh`; it stops after "CLAIM-005 layout PASS".

### Other GNU-only usages

Grepped `scripts/` and `.github/` for `sed -i`/`-r`, `grep -P`, `date -d`/`--date`, `stat -c`,
`readlink -f`, `xargs -r`, `find -printf`, `realpath -s`, `mktemp --tmpdir`, `-regextype`. The only
hit is the single `sed -i` at `extend-reference-contributing.sh:282`. No other GNU-only usages found.

## Impact

- The gate cannot run to completion on macOS developer machines in this repo; the failure is a
  portability bug in a test script, not a product finding, but it blocks all later gate steps there.
- Because CI is Linux-only, nothing exercises the script on BSD userland, so the defect persists
  unnoticed.

## Root cause

Non-portable in-place edit (`sed -i` with no suffix) in a verifier script that is otherwise
portable.

## Direction

(Direction, not a plan.)

1. Replace the in-place edit with a portable form: write to a temp file and `mv` it over the
   original (`sed '...' "$tmp/layout.html" > "$tmp/layout.new" && mv "$tmp/layout.new" "$tmp/layout.html"`),
   or `sed -i.bak '...' file && rm -f file.bak`, which both GNU and BSD accept.
2. Optionally add a macOS leg (`macos-latest`) to CI that runs `scripts/verify-public-product-model.sh`
   so BSD-userland regressions fail visibly.
3. Keep the fix to that one line; the mutation the test makes (strip the two Liquid guards) must be
   unchanged so the "unguarded-tokens mutation" check keeps its meaning.

## Notes

- Related, not the same defect: `scripts/verify-documentation-semantics-integration.sh` creates
  `mktemp -d "$ROOT/tmp/documentation-semantics.XXXXXX"` (line 311). The reported "never cleaned up,
  no trap" is NOT confirmed on this branch: line 312 immediately sets
  `trap 'rm -rf "$scratch"' EXIT HUP INT TERM` (present since commit 8fb3a05), and `tmp/` here holds
  no `documentation-semantics.*` directories. Leftovers, if observed, would come from a path the trap
  cannot cover (SIGKILL or a hard timeout, or runs of an older script revision extracted by
  `git archive` at line 276). That is a separate, unreproduced observation, so it is deliberately
  kept out of this issue; file a sibling only once a leak is reproduced against the current script.
  Folding an unreproduced claim into this bug would blur a fully verified one-line fix.
- The same file's other `mktemp -d` scratch directories (system temp, not under `tmp/`) are removed
  only on specific pass/fail branches, so a mid-function failure (such as this sed error) leaks them.
  Minor; a single `trap` in the script would cover it and may be done in the same change.
- ISSUE-205 (`gate --all` scans gitignored artifacts) is not present in this branch's `issues/`
  directory; it exists only in commit a9ee787 on the `review-ledger` branch. Cite it as such until
  that branch merges.
- ISSUE-209 (engine dispatch error discards findings) is on this branch.
