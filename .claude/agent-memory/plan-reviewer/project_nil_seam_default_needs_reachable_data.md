---
name: nil-seam-default-needs-reachable-data
description: A plan's "nil seam defaults to the real path" trick only works if the library can REACH the data the real gate consumes — check the production gate's actual inputs before accepting it
metadata:
  type: project
---

When a plan says "seam X defaults to the REAL implementation when nil, so tests
that leave it nil drive production code", verify the defaulting package can
actually obtain every input the real gate needs. Trace the PRODUCTION call site
and read its argument sources.

**Why:** PLAN-SPEC-054 planned `pkg/recipe`'s `TransformDispatch` to default to an
in-package `EngineTransformDispatch` calling
`engine.CheckToolAllowed(TrustedToolAllowlist(), tool, lockedVersion)`. But the
real gate (`cmd/backstop/pack_gate.go:812 checkEngineToolAllowed`) sources both
args from `binding.Provision.Tool` / `binding.Provision.Version` — i.e. the PACK
manifest's `engines:` block. `Apply(resolved *ResolvedRecipe, opts ApplyOptions)`
receives neither a `pack.Manifest` nor a lock handle, and the declared
`Op`/`RecipeManifest` contracts had no engine/tool field. The default was
unimplementable without baking a tool name (REQ-006 violation) or inventing
undeclared contract surface — and the claim testing the rejection (CLM-025) had
no declaring surface either.

**How to apply:** For every nil-seam default a plan promises, ask (a) where does
the real gate get each argument, and (b) is that value reachable from the
defaulting function's declared signature + the artifact it's handed? If not, the
seam belongs at the layer that HAS the data (usually `cmd/backstop`), and the
plan must retarget the claim's test accordingly. Related:
[[project_shared_cache_seam_wiring]], [[project_reconciliation_swap_enable_wiring]].

Two companions checked the same review:
- `waiver.Adjudicate` (pkg/waiver/adjudicate.go:86) is LINE-windowed — the
  association window is exactly `{Finding.Line, Finding.Line-1}`. Any plan that
  synthesizes a whole-FILE finding (e.g. a recipe-output divergence) must PIN
  which line the finding carries, or "consumer adds a covering waiver" is an
  unspecified contract the implementer will co-invent with its own test.
- `pkg/check` has no backstop-internal imports, so `pkg/<new> -> pkg/check` for
  `*check.ConfigError` is genuinely cycle-free (that part of the plan was right).
