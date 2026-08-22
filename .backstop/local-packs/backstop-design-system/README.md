# Backstop Design System

Backstop's design system is distributed as executable policy rather than a
reference document. The pack owns three things together:

- named visual tokens;
- deterministic recipes that scaffold the canonical Jekyll foundation and
  landing page;
- rules with fixtures that prevent the scaffolded guarantees from drifting.

## Recipes

`jekyll-foundation@1.0.0` creates the token source, product stylesheet, and
favicon under `docs/`.

`jekyll-landing-page@1.0.0` creates the Jekyll layout and landing page under
`docs/`.

Both recipes are create-only. They produce known-good starting points without
silently merging into consumer-owned files.

## Enforced decisions

- Product CSS consumes named colors from `backstop-tokens.css`.
- HTML does not bypass the system with inline styles.
- The canonical stylesheet retains visible keyboard focus and reduced-motion
  behavior.
- The canonical wordmark retains the `./b` shell mark and `.sh` domain.

Every rule carries a compliant fixture and a targeted violation fixture. Run
`backstop pack test .` to prove both sides of each claim.
