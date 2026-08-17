---
name: inert-decoy-fixtures-vacuous
description: A "filter still discriminates" guard is vacuous when its decoy fixtures are inert — the widened-filter mutation is invisible because the extractor skips unparseable input silently
metadata:
  type: project
---

A guard of the shape "over a directory holding a real X and several non-X decoys, assert
the extractor sees X and nothing else" is VACUOUS unless every decoy would actually
PRODUCE OUTPUT if the filter widened. Read the extractor's body past the filename filter:
if it does `parse(...); if err != nil { continue }` and then requires substantive
frontmatter (a status, a test command, a contracts block), then a bare `README.md` /
`notes.txt` / content-less decoy is skipped SILENTLY on the second gate, and the mutation
that widens the first gate changes nothing observable.

**Why:** PLAN-ISSUE-124 (2026-08-16) designed its only defense against its own ★ sharp
edge (`HasSuffix(x, "")` is always true when `LayoutFor` returns a zero `KindLayout`) as
exactly this shape — five files, one substantive spec, four decoys with UNSPECIFIED
content. All four target extractors in `pkg/gate/step_testverify.go` skip unparseable
files and then require frontmatter; the mutation would have stayed green. A count-shaped
extractor was worse than vacuous: the real fixture was `implemented` (non-terminal), so
`CountTerminalSpecs` returned 0 correct AND 0 mutated.

**How to apply:** when a plan mandates a discrimination guard, for EACH extractor under
test name the specific decoy property that makes the widened filter observable — a decoy
carrying a terminal status for a terminal-COUNT extractor, a decoy carrying
`mandated_tests` for a mandated-test extractor, a decoy carrying `contracts.provides` for
a contracts extractor. Decoy NAMES are not enough; decoy CONTENT is the load-bearing part
and plans routinely specify only the names. Also push the capacity-to-go-red proof into
the phase that AUTHORS the guard — a plan that defers all mutation testing to a final
verification task only discovers the guard is hollow after the change has landed.

Related: [[shortcircuit-dependent-guard]] (an earlier predicate short-circuits so the
guard is green at HEAD with the defect live), [[new-guard-predicate-measure-existing-fixtures]]
(run the real engine over real fixtures instead of reasoning about precision),
[[e2e-fixture-already-loud-at-head]].
