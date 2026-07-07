---
name: coverage-zerostmt-proxy-unsound
description: ISSUE-045 case-1 plan approach (gate-side directory proxy for zero-statement N/A) is unsound cross-language; blocked pending decision
metadata:
  type: project
---

ISSUE-045 case-1 (zero-statement file `pkg/pack/engine/fieldcontract.go` wrongly
flagged `coverage_unmeasured`) was planned as a LANGUAGE-NEUTRAL gate-side proxy in
`coveragePathsInScope`: "a no-record file whose DIRECTORY has >=1 record is
un-measurable-by-construction => N/A." Implementing exactly that **defeats the
existing bun anti-vacuous-green guard** `TestBunFixture_SeededUncoveredTsSourceRedsGateNotVacuousGreen`
(cmd/backstop): its seeded lcov measures `src/util.ts`, the changed `src/app.ts` is
genuinely untested (present, has statements, absent from lcov) in the SAME dir -> the
proxy silently passes it.

**Why:** `fieldcontract.go` (Go, zero-statement, measured sibling) and `src/app.ts`
(bun, untested, measured sibling) are STRUCTURALLY IDENTICAL in record-only data
(no own record, dir is measured). No language-neutral, record-only proxy can pass
one while firing the other. Go's -coverprofile emits total>0/covered=0 for an
untested-but-has-statements file, so in Go "no record in a measured pkg" really does
imply zero statements -- but lcov omits untested files entirely, so the invariant the
plan leaned on ("a measured package emits a block for every file with statements") is
GO-SPECIFIC, not universal.

**How to apply:** The thin-executor-correct fix is PRODUCER-SIDE, not gate-side: the
go-toolchain producer (which legitimately holds Go knowledge) should emit a `total:0`
N/A record for zero-statement .go files (e.g. `go list` the package's files, emit
total=0 for those absent from the profile). Then the gate's EXISTING `Total==0 => N/A`
guard handles fieldcontract.go with NO gate change, and the bun guard stays intact.
Alternative: elevate "every has-statements file must have a record" to a required
producer contract (all-files coverage) and treat the bun seeded-defect fixture as
non-conforming -- riskier (real bun projects without all:true would vacuous-green).
Do NOT re-add the gate-side directory proxy; do NOT weaken the bun guard by relocating
its file. Case-2 (root-basename collision) is unaffected and shipped. Relates to
[[project_coverage_producer_stream_bug]].

## UPDATE 2026-07-06: producer-side approach also blocked by the convert sandbox
Resolution 1 (producer emits total:0 N/A records; also the case-2 module-strip) was
implemented in the go-toolchain convert (`coverage-to-records.sh`) using `go list -m`
and `go list ./...`. **This does NOT work in the real gate:** the convert runs under
`packval.SandboxedRunStdout` — a deny-all macOS `sandbox-exec` profile (deny default,
deny file-write*, deny network*, file-read* ONLY for packDir + system dylib dirs, CWD =
packDir). Any `go list` there fails with `go.mod: operation not permitted` (project
reads denied; no module at packDir; go build-cache writes denied). Verified empirically
+ by a real `backstop gate` run where `pkg/pack/engine/fieldcontract.go` STILL fired
`coverage_unmeasured`. The convert is a PURE stdin->stdout transformer with NO toolchain
or project access — the plan assumed it could run go list, which is false.
**Consequence:** language-specific enrichment (module path for case-2 strip; package
file-list for case-1 N/A) must come from the UN-sandboxed ENGINE COMMAND
(`runner.RunStdout`, runs `go test`), not the convert. But the current dispatch execs
only the engine command's first whitespace-split token as-is (splitCommand has no quote
handling; no packRoot resolution), so a compound `sh -c '...'` or a pack producer-script
command is NOT runnable without a CORE dispatch change. Unit tests are GREEN but
MISLEADING — they feed records directly, bypassing the sandboxed dispatch (classic
[[project_pack_provisioning_integration_gap]]). Escalated for an architecture decision.
