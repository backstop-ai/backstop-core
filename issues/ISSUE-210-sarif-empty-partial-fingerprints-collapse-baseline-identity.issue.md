---
title: "SARIF empty partialFingerprints collapse baseline identity"
schema_version: issue/v1

issue:
  id: ISSUE-210
  title: "SARIF empty partialFingerprints collapse baseline identity"
  type: bug
  status: open
  created: "2026-10-12"

complexity:
  scope: contained
  uncertainty: known
  risk: critical
---

# SARIF empty partialFingerprints collapse baseline identity

## Problem

`sarifFingerprint` (`pkg/check/parsers.go:187-202`) derives a result's baseline fingerprint from
`partialFingerprints` whenever the map has any keys (`if len(partial) > 0`, `:188`). It never
checks that the values are non-empty. A producer that emits the keys with empty values therefore
yields a constant fingerprint, and the region-snippet fallback (`return snippet`, `:201`) is never
reached.

gitleaks in `dir` mode does this: every result carries
`partialFingerprints: {commitSha, email, author, date, commitMessage}` with every value `""`
(there is no commit in a directory scan). For every such result the function returns the same
string, `author=;commitMessage=;commitSha=;date=;email=`, regardless of the secret's location or
content.

That string becomes the finding's `RegionHash`: `cmd/backstop/pack_gate.go:994`
(`RegionHash: v.Fingerprint`). Baseline identity is built in `EnrichViolationIdentity`
(`pkg/gate/baseline.go:205-224`) as `Rule|File|region`; `Line` is excluded from identity by
design (so a finding survives line drift), and the message-hash fallback only applies when
`RegionHash` is empty (`:217-219`), which it no longer is. So identity degrades to `Rule|File`.

## Impact

Two distinct secrets that trigger the same rule in the same file get the same identity hash.
A scratch probe confirmed this (parse two gitleaks-shaped results, enrich both, compare
`IdentityHash`: equal for same rule + same file, different for different files). Consequences:

- **Fail-open for new secrets.** After one secret in a file is baselined, a NEW same-rule secret
  added to that file has an identical identity and is classified as already grandfathered
  (`CompareBaseline`, `pkg/gate/baseline.go:115`, matching at `:~126-150`). On a full sweep (`--all`) or nil-scope run
  (the CI full sweep; `applies-to: new-code` policy) it passes green.
- **Mitigated only on diff-scoped runs.** When the file is in an explicit diff/file scope, the
  ISSUE-050 touched-file revocation (`baseline.go:135-148`) revokes the grandfather and the finding
  is reported. That does not cover the full sweep, and it relies on the file being in the diff.
- **Waivers are unaffected.** Waiver matching keys on file + line + rule
  (`pkg/gate/step_waiver.go:113-131`, `:184`), not on `RegionHash`, so a waiver does not widen
  to sibling secrets.
- **Different files stay distinct** (identity includes `File`), so the collapse is per-file.
- **Fixes are invisible.** Two baselined secrets in one file collapse to one baseline entry, so
  fixing one of them never appears in `FixedViolations`; the ratchet cannot show progress.

This is a fail-open path in a secrets-detection dimension, so treat it as a blocker.

## Latency and ordering with ISSUE-208

This is latent today. ISSUE-208 (gitleaks SARIF `properties.tags` is an array, which the
`map[string]string` properties decode rejects, failing the whole parse) means no gitleaks result
currently reaches `sarifFingerprint`. Fixing ISSUE-208 exposes this defect: the first gitleaks run
that parses successfully produces collapsed identities. **ISSUE-210 must land with or before
ISSUE-208 ships**, otherwise the ISSUE-208 fix turns a fail-closed parse error into a fail-open
baseline.

The defect is general to any SARIF producer that emits `partialFingerprints` keys with empty
values, not only gitleaks; gitleaks is the observed case.

## Direction

(Direction, not a plan.)

1. **Fall back only when ALL `partialFingerprints` values are empty.** In `sarifFingerprint`, keep
   today's ordered `k=v;` output byte-identical whenever at least one value is non-empty (do NOT
   filter empty entries per key): per-key filtering would re-key existing baselines for any
   producer that emits mixed empty and non-empty values. Only the fully degenerate case (map has
   keys, every trimmed value empty) takes the new path.
2. **Never persist a raw snippet or match.** `EnrichViolationIdentity` stores the region
   unhashed (`v.RegionHash = region`, `Identity = identity + "|" + region`,
   `pkg/gate/baseline.go:~215-221`), and both fields are serialized (`identity` and
   `region_hash`, `pkg/gate/result.go:130-132`), written to `.backstop/baseline.json` by
   `WriteBaseline` (which enriches every violation, `baseline.go:99-101`), and printed in the gate
   JSON. gitleaks `snippet.text` is the plaintext secret, so a plain snippet fallback would write
   secrets into a tracked file and CI output. Any snippet- or match-derived component must be
   HASHED before it enters `RegionHash`/`Identity`.
3. **With `--redact` the snippet and message discriminate nothing.** Snippets are all `REDACTED`,
   and the message (`<rule> has detected secret for file <f>.`) is identical per file, so neither
   the snippet nor the message-hash fallback (`baseline.go:217-219`) separates two secrets. The
   remaining per-secret discriminators for gitleaks are the start line and column, which identity
   excludes by design. The plan must openly choose between (a) a line/column-derived component
   used only for this degenerate case (re-keys on line drift, so a baselined secret that moves
   would read as new), and (b) a documented residual collapse for redacted, otherwise
   indistinguishable results. Neither is decided here; do not claim the collapse is fixed under
   option (b). Without `--redact` a hashed snippet does separate them.
4. **Tests** (gitleaks-shaped results):
   - all-empty `partialFingerprints` with differing snippets: distinct `IdentityHash`;
   - all-empty with identical redacted snippets: assert the chosen outcome from item 3;
   - a MIXED-value `partialFingerprints` fixture: fingerprint pinned byte-identical to today's
     `k=v;` form (existing `TestParsePackFindings_FingerprintFromSarif` and
     `TestSarifFingerprint_StableAcrossScopeInvocations` stay green);
   - no raw snippet text appears in a baseline written by `WriteBaseline` nor in the serialized
     gate JSON;
   - a baseline comparison showing a second same-rule secret in a baselined file is reported on a
     full-sweep (nil-scope) run.
5. **Conflict with ISSUE-208.** The in-progress ISSUE-208 test
   (`pkg/check/parsers_property_bag_test.go`, mandated by PLAN-ISSUE-208) asserts
   `Fingerprint == "author=;commitMessage=;commitSha=;date=;email="` (`wantFingerprint`, `:131`,
   checked at `:174-175`), which pins the collapsed value. Since this issue lands with or before
   ISSUE-208, its plan must amend PLAN-ISSUE-208's mandated fingerprint assertion, or ISSUE-208
   must drop that assertion before shipping, so the two do not conflict.
6. Sequence the fix before or with ISSUE-208.

## Complexity estimate

Contained, known, critical risk. The code change is small (one function plus a hash helper), but
the risk is fail-open secrets handling and identity stability: any change to the non-degenerate
fingerprint form re-keys existing baselines, and a raw secret reaching the baseline file is a
leak. The open choice in item 3 is a design decision, not research.

## Notes

- Repro: parse a SARIF log with two results of the same `ruleId` in one file, each carrying
  `partialFingerprints` with all-empty values and different `region.snippet.text`; call
  `EnrichViolationIdentity` on both `check.Violation`s mapped through `RegionHash: v.Fingerprint`
  and compare `IdentityHash`. Then write a baseline containing one and run `CompareBaseline`
  with a nil scope over both.
