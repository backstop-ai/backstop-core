---
name: fetch-the-artifact-the-fix-would-pull
description: For a CI-wiring fix whose premise is "landing file X resolves N failures", download X yourself and hash it against the local copy — that turns the prediction into a measurement
metadata:
  type: project
---

When a plan's premise is "wire up the fetch and the missing file will resolve these
failures", the premise is a claim about a file nobody has opened. Fetch it and check it
against the specific assertions the failing tests make.

**Why:** PLAN-ISSUE-176 (2026-08-18). ISSUE-176 argued that authorizing `backstop gate`'s
self-healing baseline pull in CI would fix three ratchet tests. Downloading the artifact the
pull actually selects (`gh api .../actions/runs?branch=main&status=success` → newest run →
`.../artifacts` → `.../zip`) showed it carried ZERO neutral-spine findings for all three
asserted sites — and its sha256 was **byte-identical** to this darwin checkout's
`.backstop/baseline.json`. That reframed the whole diagnosis: the darwin-green /
Linux-red split was not a platform behavior difference at all, it was the SAME FILE, present
here and absent there (self-healed onto disk by some earlier local gate run). The premise
became measured instead of argued, and the plan could state flatly that a green local suite
is zero evidence for the CI claim.

**How to apply:** any issue-sourced lane whose fix is "make the fetch work". Before writing
phases: resolve the artifact the way the production code resolves it, download it, and check
its CONTENT against what the failing tests assert — not just that it exists. If it hashes
equal to a local file, say so; that pair of hashes is the strongest sentence in the plan.
Then check the resolver for filters it decodes but never applies (`resolveLatestSuccessfulMainRun`
decodes each run's `Name` and never filters on it — latent, since all 20 rows were `CI`) and
file that separately rather than absorbing it. See [[project_platform_gated_defect_verification_ceiling]]
for the complementary case, where no local measurement can substitute.
