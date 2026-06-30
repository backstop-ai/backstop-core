---
name: existing-test-coupling
description: Field/artifact-removal specs must name the EXISTING tests that assert the old behavior, not just the new claims that replace it
metadata:
  type: feedback
---

When a spec removes a struct field, function arm, or artifact, it must explicitly
account for EXISTING tests (in the very package it edits) that construct that field
or assert the old behavior — not only the symmetric coupling it happens to notice.

**Why:** SPEC-030 (packs-only / native-standards removal) was meticulous about the
`pkg/compile` test coupling (those tests read the deleted `STD-GO-001` source
standard, so the package must be deleted wholesale). But it completely missed that
`pkg/check/semgrep_executor_test.go` constructs `semgrepExecutor{manifestDir: ...}`
in 4 places and asserts `containsConfigFor(args, ".backstop/rules")` — i.e. the
existing test (1) won't compile after REQ-001 removes the field and (2) asserts the
exact standards-dir `--config` that REQ-004 deletes. The spec guarantees a green
`go test ./...` (REQ-005/CLM-021) but its new claims (CLM-001/002/003) only ADD
replacement tests; they never say the old `TestCodeCheck_SemgrepExecutor_RunsProjectAndPackConfigs`
must be deleted/rewritten. An implementer following the spec literally hits a red build.

**How to apply:** For any removal spec, grep the edited package's `*_test.go` for the
removed field/function/literal BEFORE approving. If existing tests reference it,
the spec must name them and say delete-or-rewrite. "We added new tests" is not the
same as "we handled the old tests that now contradict the change." Symmetric to
[[feedback_parser_locus_seam]] — name the right locus, including the test locus.

**Recurrence (2026-06-16):** A later SPEC-030 revision FIXED the `semgrep_executor_test.go`
coupling (added CLM-023 + Edit-1 delete/rewrite instructions) but still missed that
`pkg/check/check_test.go` constructs `Options{ManifestDir: dir}` (keyed literal) in 11
places across 10 test functions (`TestCodeCheck_RunWith_*`, `TestCodeCheck_FileFlag_RoutesByType`,
etc.). Removing `Options.ManifestDir` (REQ-002) breaks all 11 → red `go test ./pkg/check/`,
which is the spec's OWN `test_command`. Lesson reinforced: grep the WHOLE edited package
(every `*_test.go`), not just the one test file the spec already noticed. One named coupling
does not mean the author swept the package.

**Recurrence (2026-06-24, SPEC-039 dead-code prelude):** Two new wrinkles. (1) The
spec's `test_command` spanned TWO dirs (`./pkg/check/ ./cmd/backstop/`) but the
orphaned-test reconciliation named only `pkg/check/*`; `cmd/backstop/code_check_test.go`
feeds compiled/legacy `.manifest.json` fixtures through the deleted reader
(`TestCLI_TSDeclaredStack_SmokeEndToEnd`, `TestCodeCheck_LoadManifest_ConfigErrorPropagates...`,
`missingToolchainProject`) and was never named. Lesson: grep EVERY dir in the
`test_command`, not just the package the deletion lives in. (2) It also missed an
in-file SPEC-035 test (`TestCheckType_SemgrepRenamedToNeutralFindings`) that pins
`defaultManifest().RouteFile("notes.txt") == [findings]` — broken by the catch-all
deletion — because the author scanned for `.manifest.json` fixtures but not for tests
asserting the *default-manifest* behavior the deletion changes. Grep BOTH the deleted
symbol/fixture AND the surviving-API behavior the deletion alters.
