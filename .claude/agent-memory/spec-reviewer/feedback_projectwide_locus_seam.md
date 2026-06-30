---
name: projectwide-locus-seam
description: REQ-004-style "re-express the build exemption" specs anchor to gate.go:1173, but its only caller is deleted upstream — the real locus is the engine path
metadata:
  type: feedback
---

When a BUNDLE-011 spec re-expresses the build-pass project-wide-scope exemption (`cv.Pass == check.CheckTypeBuild`, `checkViolationsToGate`, gate.go:1173) "as a declared property," verify the LOCUS against live call graph, not the bundle's line-number prose.

**Why:** On main, `checkViolationsToGate` is called ONLY from `realCodeChecker.CheckScoped` (gate.go:1122). SPEC-040 (Seed 2, the prerequisite) DELETES `realCodeChecker` + CheckScoped. So after the prerequisite lands, gate.go:1173 is ORPHANED dead code — "re-expressing it in place" / asserting "no `== CheckTypeBuild` remains in checkViolationsToGate" (SPEC-041 CLM-015) edits a function with no caller.

The real seam: `gate.Violation.ProjectWide` is what `filterViolations` (scope.go:194) exempts from diff-scope-filtering. On the ENGINE path (`dispatchPackEngines`), `binding.ScopeKind == ScopeKindProjectWide` is used ONLY for arg-shaping (pack_gate.go:411 — append `./...` vs changed-file target); it does NOT set `gate.Violation.ProjectWide`. So post-SPEC-040, engine-path build (`go-build`) violations get `ProjectWide=false` and a build break in an UNCHANGED file is scope-filtered AWAY — the exact under-broad regression the spec's own Sharp Edge 5 names. The declared-property mechanism the spec claims to "read" does not yet exist on the engine path; it must be BUILT (map `binding.ScopeKind` → `violation.ProjectWide`), and the spec doesn't name that as the work.

**How to apply:** For "re-express baked enum exemption X as declared property" REQs: (1) grep the only caller of the function holding the enum check; if an upstream prerequisite deletes that caller, the in-place "re-express" is wrong — it's a DELETE + a new bridge elsewhere. (2) Find where the target field (here ProjectWide) is actually CONSUMED (filterViolations) and where the engine path would need to SET it. (3) Confirm the declared property already flows end-to-end, or flag that the spec must build the missing bridge. Related: [[parser-locus-seam]] (same class — removal/re-expression claim naming the wrong call path).
