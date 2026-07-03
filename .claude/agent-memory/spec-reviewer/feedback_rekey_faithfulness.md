---
name: rekey-faithfulness
description: When a spec re-keys an existing validation contract (e.g. layer→engine field requires/forbids), diff against the live code, not just the bundle prose
metadata:
  type: feedback
---

When a spec claims to "re-key" or "migrate" an existing validation contract onto a new
dimension, verify the re-key is faithful by diffing the spec's stated requires/forbids
against the actual source — do NOT trust the bundle's "e.g." enumeration as exhaustive.

**Why:** SPEC-031 (BUNDLE-010 Seed 2) re-keyed `validateLayerFields` from layer→engine.
The live layer-2 (semgrep) contract forbids `category`, `input_scope`, AND `validator`;
the spec's REQ-003 forbade only `input_scope`. Layer-1 (config-file) forbids four fields;
spec forbade only `rule_path`. The bundle REQ-003 used "e.g." and didn't enumerate forbids,
so reading bundle-only would have missed the silent narrowing. A planner implementing the
narrowed spec drops validation that exists today — a regression against backstop's own
fail-loud-on-misdeclaration ethos.

**How to apply:** For any spec that rewrites validateX/checkY, grep the current function
(here pkg/pack/validate_manifest.go `validateLayerFields`) and build a field-by-field
requires/forbids matrix. Flag any forbid present in code but absent from the spec's matrix
as a faithfulness gap unless the spec explicitly justifies the relaxation.
