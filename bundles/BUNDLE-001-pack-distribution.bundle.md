---
title: "Pack Distribution, Verification, and Review — Declarative Packs, Git-Tap Distribution, Mechanical Curation"
number: BUNDLE-001
created: "2026-04-08"
schema_version: bundle/v2

bundle:
  name: pack-distribution
  version: "1.1.0"
  created: "2026-04-08"
  updated: "2026-04-08"
  category: feature

status:
  maturity: exploring

problem:
  summary: >
    The current pack model is underspecified. `pack new` and `pack compile`
    exist, and SPEC-012 (the Go standards pack) is embedded via go:embed, but
    the model was built only deep enough to support that one pack. There is no
    distribution story, no third-party authoring story, no validation contract,
    no supply chain story, and no review story. Scaling the current model would
    repeat a known failure mode from the founder's day-job project: forcing
    pack authors to implement ~14 Go interfaces because packs are expected to
    live in-tree. That model is wrong for backstop. Packs should be declarative
    data authored by people who do not know (and do not need to know) that the
    CLI is written in Go. Distribution should be git-native, not registry-
    native. Validation should be mandatory and mechanical. Review of curated
    packs should be performed by an LLM that is itself a backstop artifact —
    because recommending human review for pack curation would contradict the
    project's own thesis. The supply chain risk surface should be designed out
    at the model level (no transitive trust, no network at load, mandatory
    lockfile, finite auditable graph) rather than patched on later.

  user_story: >
    As a pack author, I want to write a pack as plain files (manifest +
    .standard.md + fixtures) without learning Go, implementing interfaces, or
    contributing to backstop's monorepo. I push it to a git repo. Consumers
    add it with `backstop pack add <git-url>`, which validates it locally and
    pins it in their lockfile. As a pack consumer, I want strong defaults: I
    cannot accidentally load an unvalidated pack, I cannot inherit transitive
    dependencies I never approved, and I am told loudly when an update removes
    fixtures or downgrades severities. As a backstop maintainer running a
    lightweight curated catalog, I want LLM review with a versioned rubric to
    be the primary gate, with humans handling only the small fraction the LLM
    flags as ambiguous.

solution:
  approach: >
    Treat packs as declarative data distributed via git ("homebrew taps"),
    loaded through a single Go-side adapter that speaks pack-author terms.
    Make validation (claim-fixture-rule coherence, fixture coverage, semgrep
    --test pass) a hard precondition for loading any pack — including the
    embedded core pack, which is loaded through the same path as third-party
    packs to dogfood the loader. Design the supply chain surface to be finite:
    no transitive trust, no network at load, mandatory `backstop.lock`, tamper
    detection on update. Integrate existing tools (OSV-Scanner, sigstore,
    semgrep supply-chain rules, Trivy/Grype) as subchecks of `pack validate`
    and as steps in the gate kill chain rather than reinventing them. For
    curation, build an LLM reviewer that is itself a backstop artifact —
    versioned prompt + rubric + known-bad fixture corpus — so the reviewer of
    packs is mechanically verified the same way packs are. Packs are
    language-specific by construction: each pack targets exactly one
    language and ships that language's rules, scaffolds, SDK, contracts,
    test patterns, AST checks, and fixtures via a typed, extensible
    `content:` block (rules, scaffolds, sdk, contracts, test_patterns,
    ast_checks, rubrics, fixtures). Cross-language capabilities are
    expressed as a *family* of single-language packs coordinated by
    convention (shared publisher, shared name prefix, lockstep version
    cadence), not as a single multi-language artifact — backstop never
    models cross-language packs. A lightweight curated catalog provides
    discovery: a maintained, searchable index of vetted packs storing
    metadata only (name, git URL, language, capability tags, latest vetted
    ref, validation status). The catalog does not host binaries, proxy
    downloads, or hold publisher credentials — consumers clone from the
    publisher's git repo. Publishing proxy, signed attestations, credential
    management, and native-registry fan-out are deferred to BUNDLE-002.
---

# Pack Distribution, Verification, and Review

## Current Thinking

### The current state

`pack new` scaffolds a directory. `pack compile` turns `.standard.md` files
into semgrep rule YAML. SPEC-012 (12 requirements, 58 claims, 100% fixture
coverage) is embedded via go:embed and is the only real pack in existence.
There is no story for: external authors, distribution, lockfiles, signing,
update tamper detection, registry curation, supply chain scanning, rule
collision resolution, or non-semgrep rule engines.

### The 14-interface failure mode

The founder's day-job project (3rd attempt at "mechsuit") just shipped a
300+ file, 42k LOC CLI PR. A core pain point: pack authors had to implement
~14 Go interfaces because packs were assumed to live in the monorepo. That
assumption forced every pack author to become a Go contributor. Backstop is
explicitly rejecting that model. Packs are data; the loader adapts to packs;
error messages speak the author's vocabulary, not Go's.

### Distribution: git-native, not registry-native

Packs live in one of two places: alongside a consuming repo (in-repo), or
in a standalone git repo ("tap"). `backstop pack add <git-url>` clones into
a local cache, validates, and on success makes the pack available. No
central registry is required for v1. GitHub is the discovery layer.
Pinning is by git ref + content hash, recorded in `backstop.lock`. A
lightweight curated catalog adds gating and discovery on top — it does not
replace the git-native path.

### Validation: mandatory, mechanical, recursive

You cannot vibe your way to a standards pack. Every rule must declare the
claims it catches. Every claim must be substantiated by positive AND
negative fixtures that mechanically exercise the claim. `pack validate`
runs claim-fixture-rule coherence, semgrep `--test` at 100%, supply-chain
subchecks (OSV-Scanner, sigstore verification, Trivy/Grype), and refuses to
mark the pack loadable on any failure. The CLI refuses to load an
unvalidated pack. Pack repos dogfood backstop in their own CI to validate
themselves. The embedded core pack runs through the same loader and the
same validator as any third party — no special case.

### Supply chain: design it out, don't patch it on

Five structural choices remove most of the npm/dockerhub attack surface:

1. **No transitive trust.** If pack A references pack B, the consumer must
   add B explicitly. The dependency graph is finite, visible, and auditable.
2. **Network isolation at load.** Packs cannot fetch anything at validate,
   compile, or run time. All inputs are in the pack or in explicitly-added
   sibling packs.
3. **Mandatory lockfile.** `backstop.lock` pins every pack to a content
   hash from v1. `pack add` writes it, `pack update` diffs it, `gate`
   refuses to run on mismatch.
4. **Tamper detection on update.** `pack update` shows which rules changed,
   which fixtures were removed, which severities dropped. Fixture removal
   and severity downgrade are red-flag events requiring explicit consumer
   acknowledgment.
5. **Rule risk taxonomy.** Every rule declares a category (security/style/
   perf/correctness) and risk class. Security rules carry stricter
   requirements: mandatory bypass-attempt negative fixtures, mandatory
   author signature, stricter claim coverage thresholds.

Existing tools (OSV-Scanner, sigstore/cosign, semgrep supply-chain rules,
Trivy/Grype) run as subchecks of `pack validate` and as steps in the
existing 9-step gate kill chain. Supply chain violations are gate
violations, not a separate "scan" command.

### Two-tier ecosystem

Tier 1: ungated git taps. Anyone can publish; the consumer assumes the
risk; mandatory local validation is the only gate. Tier 2: a curated
catalog — a maintained, searchable index where every entry has passed the
validation gate and (when ready) the LLM reviewer. The catalog is not a
binary store and not a registry in the npm sense. Publication is "push to
your git repo and submit the URL to the catalog." The catalog answers
"what's out there and is it any good." The previous "final-approver
registry" framing is superseded by this lighter model.

### LLM as primary reviewer for the curated catalog

Humans are bad at pack review: fatigue, pattern blindness, inability to
hold whole pack + claims + fixtures in working memory, inconsistency across
reviewers. LLMs are better at exactly this shape of task: no fatigue,
single-pass coherence checks across rules/claims/fixtures, benchmarkable
against a known-bad corpus, cheap enough to run on every submission and
every update. Humans are escalation only — the ~1% the LLM flags as
ambiguous. Recommending human review for pack curation would contradict
backstop's own principles.

The reviewer is itself a backstop artifact: its prompt, rubric, and known-
bad fixture corpus live in a backstop repo with claims and fixtures
proving it catches the bad cases. Recursion: the reviewer of packs is
mechanically verified the same way packs are.

## Draft Design Decisions

- **DD-1:** Packs are declarative data, not Go code. No interfaces for
  authors to implement. The Go-side loader adapts to pack authors;
  errors speak in pack-author vocabulary. Rationale: rejects the
  14-interface model from the founder's day-job project.
- **DD-2:** Distribution model is git-native ("homebrew taps"), not
  npm-style. `backstop pack add <git-url>` clones, validates, caches,
  and pins. No central registry is required for v1. A lightweight curated
  catalog exists for discovery but is metadata-only (name, git URL,
  language, capability tags, latest vetted version, validation status).
  The catalog doesn't host binaries, doesn't proxy downloads, doesn't
  hold credentials. Consumers still clone from the publisher's repo. The
  catalog answers "what's out there and is it any good."
- **DD-3:** Validation is a hard precondition for loading. Unvalidated
  packs cannot be used. `pack validate` and `pack test` (semgrep --test)
  must pass at 100% before a pack is loadable.
- **DD-4:** Claim-fixture-rule mapping is enforced as a supply chain
  control. A rule that claims X but whose fixture does not exercise X is
  a validation failure. This catches subtly-malicious packs that scanners
  cannot.
- **DD-5:** The embedded Go standards pack (SPEC-012) becomes the "core
  tap" — always present in the binary, but loaded through the same loader
  path and validated by the same validator as any third-party pack. No
  special case. Proves the loader works for outsiders.
- **DD-6:** Two-tier ecosystem: ungated git taps (consumer-risk) and a
  curated catalog (a maintained, searchable index where every entry has
  passed the validation gate and the LLM reviewer). The catalog is not a
  binary store — it is metadata pointing at the publisher's git repo.
  Publication is "push to your git repo and submit the URL to the
  catalog." Safe by default, risky on explicit request. The earlier
  "final-approver registry" framing is superseded by this lighter model.
- **DD-7:** Supply chain risk is addressed by integrating existing tools
  (OSV-Scanner, sigstore/cosign, semgrep supply-chain rules, Trivy/Grype)
  as subchecks of `pack validate` and as gate kill chain steps — not by
  rebuilding them. Supply chain violations are gate violations.
- **DD-8:** No transitive trust. If pack A references pack B, the
  consumer must add B explicitly. Designs out the multi-layer dependency
  attack at the model level.
- **DD-9:** `backstop.lock` is mandatory from v1. Content-hash pinning of
  every pack. `pack add` writes it; `pack update` diffs it; `gate`
  refuses to run on mismatch.
- **DD-10:** `pack update` performs tamper detection. Fixture removal
  and severity downgrade are red-flag events requiring explicit consumer
  acknowledgment.
- **DD-11:** LLMs are the primary reviewers for the curated catalog;
  humans are escalation only. Recommending human review would contradict
  backstop's thesis. Reviewer is benchmarked against a known-bad corpus.
- **DD-12:** The reviewer is itself a backstop artifact — versioned
  prompt + rubric + known-bad fixture corpus, with claims and fixtures
  proving it catches bad packs. Recursive mechanical verification.
- **DD-13:** Packs are network-isolated at load. No fetches at validate,
  compile, or run time. The supply chain surface is finite and auditable.
- **DD-14:** Every rule declares a category (security/style/perf/
  correctness) and risk class in the manifest. Security rules carry
  stricter requirements: mandatory bypass-attempt negative fixtures,
  mandatory author signature, stricter claim coverage thresholds.
- **DD-15:** The 14-interface model is explicitly rejected. It made
  sense for in-tree Go packs; it is wrong for repo-native + git-tap
  distribution.
- **DD-16:** Pack content is typed and extensible. A pack declares a
  `content:` block enumerating which types it provides; the loader
  dispatches each type to its handler. Types: `rules` (semgrep patterns
  from `.standard.md` with claim-fixture-rule mapping and risk class),
  `scaffolds` (copy-once templates producing new files from parameters,
  consumer owns the output), `sdks` (language-keyed references to
  native-language modules, see DD-19), `contracts` (reusable public API
  signatures for the contract-signature gate step), `test_patterns`
  (per-language test substantiveness heuristics), `ast_checks`
  (declarative AST-level rules semgrep can't express cleanly, sandbox-
  gated — see OQ-3; note: `ast_checks` are a specific kind of layer 3
  custom validator per DD-43/DD-44, not a separate enforcement tier),
  `rubrics` (versioned LLM review rubrics for
  pack/spec/plan review), `fixtures` (shared fixtures referenced by
  rules/scaffolds/ast_checks). A minimal pack may ship only `rules`;
  a full pack any subset. *Note (0.8.0):* The pack archetype
  (enforcement vs code, see DD-46) is an additional manifest-level
  declaration alongside the content type list, not a replacement for
  it. Content types describe *what* the pack ships; the archetype
  describes the *validation contract* governing those contents.
- **DD-17 (revised in 0.4.0):** A pack is a language-specific
  enforcement unit. Each pack targets exactly one language and ships
  that language's rules, scaffolds, SDK, contracts, test patterns,
  AST checks, and fixtures. Cross-language capabilities are expressed
  as a *family* of single-language packs coordinated by convention:
  shared publisher, shared name prefix, lockstep version cadence
  (e.g., `acme/stripe-sheets-go@v1.2.0`,
  `acme/stripe-sheets-python@v1.2.0`,
  `acme/stripe-sheets-typescript@v1.2.0`). The consumer adds the
  family member matching their language. Backstop never models
  cross-language packs. The embedded Go standards pack (SPEC-012)
  is a language-specific pack for Go — no special case. *Rationale:*
  The earlier "pack as capability spanning languages" framing imposed
  per-language rule variants, multi-language SDK surface verification,
  and cross-language composition semantics on every pack to serve a
  minority of use cases. Language-specific packs collapse most of
  that complexity while preserving the composition model at the
  consumer level. Cross-language behavioral equivalence cannot be
  mechanically verified by backstop anyway, so modeling it in the
  manifest was false precision — the author is the only one who
  knows their Go and Python implementations are equivalent. Pushing
  that coordination out to naming convention puts the responsibility
  where the knowledge lives.
- **DD-18 (annotated in 0.8.0):** Recipes are not a pack content type.
  Advisory "how to do X correctly" content lives either in the prose/
  rationale of a `.standard.md` file (accompanying a rule) or as an
  enforceable rule plus a scaffold. Non-enforced advice is
  documentation, not a pack artifact. Closes a loose term from prior
  discussion. *Note:* What was previously discussed as "recipes" maps
  to the scaffold tiers in DD-47 (complete and skeleton scaffolds).
  The concept didn't disappear; it was refined into scaffolds with
  varying completeness levels. There is still no advisory tier.
- **DD-19 (revised in 0.4.0):** Scaffolds and SDKs are both scoped to
  the pack's single language. Scaffolds are copy-once templates
  backstop renders and writes. SDKs are native-language modules
  consumers import via their native toolchain; backstop tracks the
  reference in the lockfile and validates the pack's rules and
  fixtures against the SDK's public surface but does not distribute
  SDK code. The *publishing* of SDKs to native registries is deferred
  to BUNDLE-002. Because the pack itself is language-specific, scaffolds
  do not need a `language:` field and the `sdk:` field is singular
  (DD-20). *Note (0.8.0):* See DD-47 for the scaffold tier split
  (complete vs skeleton) and the different verification expectations
  for each tier.
- **DD-20 (revised in 0.4.0):** `sdk:` is a single optional entry in
  the manifest, not a list. Fields: `module` (canonical reference
  for the pack's language ecosystem), `version`, and `provides:`
  (the public surface the pack claims the SDK exposes). A pack ships
  at most one SDK. If a publisher wants to ship a capability with
  SDKs across multiple languages, they publish a family of
  single-language packs per DD-17, each with its own `sdk:` entry
  for its target ecosystem.
- **DD-21:** A pack's rules can reference its own SDK surface. Rules
  enforce correct usage of the SDK (e.g., "if you import this pack's
  SDK, call `pack.stripe.Subscribe`, not `stripe.Charge` directly").
  Rule + SDK + scaffold move as a single versioned unit. This is the
  mechanism that makes SDK usage *enforceable*, not merely *available*.
- **DD-22:** *Deferred — see BUNDLE-002.* Backstop as a publishing
  proxy (fan-out to native registries on the author's behalf behind a
  pre-publish gate). The core pack lifecycle does not require the proxy;
  packs distribute via git taps and SDK references are tracked in the
  lockfile. The proxy model, credential management, and native-registry
  fan-out are addressed in BUNDLE-002.
- **DD-23:** Consumers never install from backstop. Install paths are
  always native — `go get`, `npm install`, `pip install`, `cargo add`.
  Backstop is not a distribution endpoint and never sits in the runtime
  or install path for SDK code. Bounds backstop's operational surface
  and legal exposure and sidesteps the "is backstop a package manager"
  trap.
- **DD-24:** The publisher owns the end-user support contract.
  Backstop's liability ends at "we validated what you gave us and
  pushed it where you told us to push it." Bug reports, support,
  versioning cadence, deprecation — all the publisher's responsibility.
  Backstop does not stand between publishers and their users.
- **DD-25:** *Deferred — see BUNDLE-002.* Pre-publish as the single
  point of enforcement. Without the publishing proxy, there is no
  "pre-publish" gate in the proxy sense. Validation happens at
  pack-load time on the consumer side and at catalog-submission time.
  The pre-publish enforcement model is addressed in BUNDLE-002.
- **DD-26:** *Deferred — see BUNDLE-002.* Signed attestations at
  publish time. Attestations require the publishing proxy to produce
  them. Without the proxy, attestation production and consumer-side
  verification are not applicable. Addressed in BUNDLE-002.
- **DD-27:** The "curated catalog" is a curated index, not a binary
  store. It stores: pack name, git URL, language, capability tags,
  latest vetted git ref, validation status, listing date, revocation
  status. Metadata only; no binary hosting. The earlier "attestation
  index" framing is superseded — attestations are deferred to
  BUNDLE-002. The catalog's value is discovery and curation
  independent of the attestation model.
- **DD-28:** *Deferred — see BUNDLE-002.* Sigstore lean for signing.
  The signing story is meaningful only in the context of the publishing
  proxy and attestation model. Deferred to BUNDLE-002 where the
  credential/identity model is addressed.
- **DD-29 (re-resolved in 0.8.0):** OQ-2 (what is a "code pack") was
  previously resolved as "the term dissolves." Real-world prototyping
  with agents extracting packs from production codebases revealed that
  code packs are a distinct archetype with a distinct validation
  constraint. Re-resolved: code packs are a real archetype defined by
  a mandatory co-occurrence rule — if a pack declares `sdks` or
  `scaffolds`, it must also declare `rules` that enforce correct usage
  of that code. The distinction is not about content types (any pack
  can have any content type combination); it is a validation contract:
  code ships with enforcement, always. See DD-46 for the full two-
  archetype model.
- **DD-30:** Rule IDs are automatically namespaced by pack using
  `pack-name/rule-id` format with slash delimiters. On load, the loader
  prefixes every rule ID with the pack's canonical name (e.g., `GO-011`
  in pack `acme/go-style` becomes `acme/go-style/GO-011`). Rule ID
  collisions across packs are impossible by construction. Pack authors
  write unprefixed IDs; prefix is applied at load. Violation output shows
  the namespaced ID so the owning pack is always visible. Version is not
  embedded in the ID — it's tracked in the lockfile and backstop.yml.
  Matches industry convention: ESLint (`@typescript-eslint/no-unused-vars`),
  semgrep (`r/go.lang.security.audit.xss`), golangci-lint
  (`gocritic/hugeParam`).
- **DD-31:** Semantic overlap between packs is allowed, not a
  collision. Two different rules from two different packs catching
  the same bad pattern is redundant but not wrong — the code is
  still wrong, the consumer still fixes it. De-duplication is a
  presentation-layer concern (group by file+line, list which rules
  fired); the underlying model treats them as independent.
- **DD-32:** Contradictions are detected mechanically via fixture
  composition. At pack-load time, backstop runs the union of all
  loaded packs' positive fixtures through the composed ruleset. Any
  positive fixture that now fails is a hard error — one pack's
  known-good code is rejected by another pack's rule. Separately,
  it runs every negative fixture through the composed ruleset and
  verifies each still fails; a negative fixture that stops failing
  means one pack weakened another pack's enforcement, also a hard
  error. Fixtures are ground truth; no heuristics, no reasoning
  about pattern intersection.
- **DD-33:** Severity resolution is max-wins, override-required-to-
  downgrade. When the same offense is caught by multiple packs at
  different severities, the composed result is the highest severity
  present. Consumers cannot accidentally downgrade enforcement by
  adding a pack. Explicit downgrades live in `backstop.yml` under
  `rule_overrides:`, are recorded in the lockfile, and are surfaced
  in gate output so nothing is silent. Upgrades (raising severity
  beyond what any pack declares) are also allowed via the same
  mechanism.
- **DD-34:** Claim-level contradiction detection via the LLM
  reviewer. The same rubric-as-artifact reviewer from DD-11/DD-12
  performs semantic comparison of claim text across loaded packs
  and flags direct opposites (e.g., "wraps errors with %w" vs
  "errors are never wrapped"). Mechanical fixture composition
  (DD-32) catches contradictions that can be exercised; claim-level
  review catches contradictions that exist in principle even when
  no fixture happens to exercise them. Both run at pack-load time.
- **DD-35:** No automatic ordering or precedence between packs.
  "First wins" and "last wins" are both silent and therefore
  unacceptable per backstop's principles. Unresolvable
  contradictions require the consumer to either remove a pack,
  explicitly override the conflicting rule in `backstop.yml`, or
  fork one of the packs. Backstop will not pick a winner on the
  consumer's behalf.
- **DD-36:** Cross-language capability coordination is author
  responsibility, not backstop concern. When a publisher ships a
  family of language-specific packs for the same capability,
  behavioral equivalence across implementations is their
  responsibility to establish and communicate (naming, changelog,
  documentation, cross-pack CI). Backstop does not verify, enforce,
  or reason about cross-language equivalence, because it cannot do
  so mechanically. Attempting to model it would be false precision.
- **DD-37:** Single-language packs still address sprawl via
  composition at the consumer layer. The original motivation for
  cross-language packs was to prevent pack sprawl. Sprawl is still
  contained under the language-specific model because: (a) naming
  conventions keep families discoverable, (b) consumers load only
  what they need for their stack, (c) publishers still ship coherent
  families with shared versioning, (d) cross-pack composition
  semantics from DD-30..DD-35 handle the multi-pack case cleanly at
  the consumer level. The difference is that composition happens
  between packs in `backstop.yml`, not inside a single pack manifest.
- **DD-38:** Lightweight curated catalog for discovery. The catalog is
  a maintained, searchable index of vetted packs. Each entry contains:
  pack name, git URL, language, capability tags, latest vetted git ref,
  validation status, listing date, revocation status. The catalog
  doesn't host binaries, proxy downloads, or hold publisher credentials.
  Consumers clone from the publisher's git repo; the catalog only
  answers "what's out there and is it any good." Packs enter the
  catalog via a submission pipeline: publisher submits git URL ->
  backstop runs validate + test + (eventually) LLM review -> pass =
  listed, fail = rejected with feedback. Rationale: the full publishing
  proxy model (BUNDLE-002) is heavyweight and blocks the core pack
  lifecycle. A metadata-only catalog delivers discovery and curation
  value without requiring credential management, native-registry
  fan-out, or attestation infrastructure.
- **DD-39:** Catalog revocation is first-class. If a listed pack is
  later found to be bad (security issue, malicious, abandoned with
  known bugs), the catalog marks it revoked with a reason and
  timestamp. Consumers running `backstop gate` check the catalog's
  revocation list. No native-registry yank asymmetry to deal with —
  the catalog is the only index, and it's under backstop's control.
  Rationale: the yank-asymmetry problem (OQ-19 in the original
  framing) only exists when artifacts live in native registries
  backstop doesn't control. For git-tap packs indexed by the catalog,
  revocation is straightforward because backstop owns the index.
- **DD-40:** Periodic re-validation of catalog entries. The catalog
  points at git repos backstop doesn't control. A pack that was valid
  at listing time can rot (deleted repo, force-pushed breaking changes,
  new CVEs in dependencies). The catalog periodically re-validates
  listed packs against their pinned ref and flags entries that fail.
  Stale entries get a status downgrade, not silent removal. Rationale:
  a catalog that goes stale silently is worse than no catalog — it
  gives consumers false confidence. Active re-validation is the
  minimum integrity guarantee for an index pointing at third-party
  git repos.
- **DD-41:** Catalog listing requires integration of backstop's shared
  CI workflow. Backstop publishes a shared GitHub Actions workflow
  (`backstop-org/pack-ci/.github/workflows/validate.yml`). Packs
  seeking catalog listing must integrate this workflow — it runs
  `backstop pack validate`, `backstop pack test` (semgrep --test at
  100%), supply chain subchecks, and eventually the LLM reviewer on
  every push and PR. The catalog submission flow checks: (1) does the
  repo have the shared workflow referencing the canonical
  `backstop-org/pack-ci` repo (not a fork)? (2) is CI green on the
  submitted ref? Both must be true for listing. Ongoing: the catalog
  periodically checks CI status of listed packs; CI red triggers
  listing downgrade (see DD-40). Raw git taps without the workflow
  remain usable (tier 1, consumer-risk) but cannot be listed in the
  curated catalog. Rationale: continuous validation, not point-in-time.
  A pack proves it's valid on every commit, not just at submission.
  Uniform enforcement — every listed pack runs the same checks the same
  way. Version-locked to backstop — bumping the shared workflow version
  re-validates every pack against the new bar, enabling ecosystem-wide
  enforcement upgrades.
- **DD-42:** The shared CI workflow is the canonical definition of
  "validated." Local `backstop pack validate` and the shared workflow
  run the same checks, but the workflow's public CI logs are what the
  catalog trusts. This prevents "I ran it locally, trust me" and gives
  anyone (consumers, reviewers, the catalog's own re-validation) a
  public, auditable proof of validation status. Status badges backed by
  real CI, not self-applied labels.
- **DD-43:** Enforcement is layered: built-in rules -> custom
  declarative rules -> custom validators. Packs layer enforcement in
  three tiers ordered by trust and verification burden:
  - *Layer 1 — Built-in tool rules (highest trust).* Semgrep core
    rules, golangci-lint, `go vet`, language-native linters.
    Community-maintained, broadly tested, well-understood false
    positive rates. Packs reference these rules, they don't
    reimplement them.
  - *Layer 2 — Custom declarative rules (medium trust).* Pack-authored
    semgrep patterns compiled from `.standard.md`. Still declarative,
    still pattern-matching, still sandboxed. Subject to mandatory
    claim->fixture->rule validation (DD-3, DD-4). This is the bulk of
    what a pack ships.
  - *Layer 3 — Custom validators (lowest trust, highest scrutiny).*
    Shell scripts, AST walkers, cross-file checks — things layers 1
    and 2 genuinely cannot express. Subject to the strictest
    validation requirements. This is the escape hatch, not the default
    path.
  The ordering implies a rule: don't use layer 3 for something layer 1
  or 2 can handle. If semgrep can express it, write a semgrep rule. If
  a built-in linter already catches it, reference the linter rule. Only
  drop to custom validators when the first two layers genuinely can't
  do the job. Rationale: maximize use of battle-tested tooling, minimize
  custom surface area. Each layer has decreasing community trust and
  increasing per-pack verification burden.
- **DD-44:** Layer 3 custom validators are the legitimate domain for
  checks that require cross-file analysis, structural/filesystem checks,
  or checks on inputs that declarative tools can't process. Specific
  categories that legitimately need layer 3:
  - *Presence enforcement* — "every HTTP handler package must have a
    `middleware.go`." No single file tells you it's missing; you need
    to look at a directory and notice an absence. Semgrep can't find
    what isn't there.
  - *File naming and placement* — "`*_test.go` files must live next to
    their source, not in a `tests/` subdirectory." Filesystem structure
    check, not file content.
  - *Cross-file consistency* — "every public API function in `api.go`
    must have a corresponding test in `api_test.go`." Requires
    correlating two files.
  - *Import graph constraints* — "package `internal/core` must not
    import from `cmd/`." Requires traversing imports across files.
  - *Convention enforcement* — "every package must have a `doc.go` with
    a package comment." Presence + content across a directory.
  - *Binary file inspection* — "PNG assets must be under 500KB,"
    "protobuf files must compile." Semgrep doesn't do binary files.
  - *Encoding/format validation* — "YAML must parse without errors,"
    "JSON schema compliance." Semgrep pattern-matches text but doesn't
    validate structure.
  - *Semantic checks beyond pattern matching* — "cyclomatic complexity
    must be under 15." Requires actual parsing, not pattern matching.
  - *Content requiring external context* — "config values must be in
    the allowed enum from the schema." Semgrep sees text but doesn't
    know the schema.
  However, a single-file check that semgrep *can* express is NOT a
  valid layer 3 submission — it must be written as a layer 1 or 2 rule
  instead.
- **DD-45:** Layer 3 validators must declare their input scope, which
  affects trust and sandboxing requirements. Single-file validators
  (take one file path, return violations) are lower risk — bounded
  blast radius, parallelizable, deterministic. Multi-file validators
  (take a directory or file list, return violations) are higher risk —
  wider trust surface, must declare which files they read and why. Both
  are valid layer 3 submissions, but multi-file validators carry
  stricter verification requirements. Regardless of scope, no layer 3
  validator may write files, make network calls, or access environment
  variables — these are hard rejections.
- **DD-46:** Two pack archetypes — enforcement packs and code packs.
  An enforcement pack ships rules only: semgrep patterns, linter
  configs, custom validators, fixtures proving them. No code ships to
  the consumer's repo. A code pack ships SDKs, scaffolds, or both —
  and always ships rules that enforce correct usage of that code. The
  asymmetry is the key constraint: code packs always contain
  enforcement; enforcement packs never contain code. This is a
  mandatory co-occurrence rule enforced by `pack validate`: if the
  manifest declares `sdks` or `scaffolds`, it must also declare
  `rules` covering that code surface. A pack without rules that
  declares code content is a validation failure.
- **DD-47:** Scaffolds have two tiers — complete and skeleton — with
  different verification expectations.
  - *Complete scaffold:* Ships with full working implementation and
    full working unit tests. Consumer changes only configuration (API
    keys, endpoint URLs, project-specific values). Tests pass out of
    the box with only config changes. Pack fixtures validate that the
    rendered scaffold plus its tests are correct before the pack is
    loadable. If the tests don't pass with only config changes, the
    pack is broken.
  - *Skeleton scaffold:* Ships with implementation structure (file
    layout, function signatures, type definitions) containing TODOs,
    and test files with the right structure (right file names, right
    function signatures, right table-driven layout) but explicitly
    incomplete assertions. Consumer or their agent fills in the TODOs.
    Pack fixtures validate structural correctness — right files exist,
    test functions are present, assertions reference the right
    targets — but don't expect tests to pass. Tests pass after the
    consumer completes the implementation.
  Each scaffold declares its tier in the manifest so `pack validate`
  knows what "pass" means for that scaffold's fixtures. Both tiers
  ship with the rules that enforce correct usage of the scaffolded
  code (per DD-46).
- **DD-48:** All pack archetypes require mechanical proof via fixtures.
  `pack validate` enforces that every pack — enforcement or code —
  mechanically proves it does what it claims:
  - *Enforcement packs:* Every rule has positive and negative fixtures.
    Positive fixtures (known-good code) must not trigger the rule.
    Negative fixtures (known-bad code) must trigger the rule. 100%
    fixture coverage via semgrep --test or equivalent. Claims map to
    fixtures mechanically (DD-4).
  - *Code packs (SDK):* The SDK's own test suite must pass. Rules
    enforcing correct SDK usage must have fixtures proving they catch
    misuse. The `provides:` surface declared in the manifest must be
    verified.
  - *Code packs (complete scaffold):* The rendered scaffold's own
    tests must pass out of the box (config-only changes). Rules
    enforcing the scaffolded code patterns must have fixtures. Pack
    validation renders the scaffold with sample config and verifies
    tests pass.
  - *Code packs (skeleton scaffold):* The rendered skeleton must be
    structurally valid — declared files exist, test functions are
    present and well-formed, TODO markers are in the right places.
    Rules enforcing the skeleton's structural patterns must have
    fixtures. Tests are not expected to pass at render time.
  No pack is loadable without mechanical proof at its tier's expected
  completeness level. "Trust me, it works" is not a valid validation
  state for any archetype.
- **DD-49:** Pack items (scaffolds, SDKs, rules) are individually
  versioned within the pack. The pack itself carries a semver
  (`version: "1.2.0"`), and each item within the pack carries its own
  version. A pack at v1.2.0 might contain `stripe-webhook-handler@v2`
  and `event-router@v1`. Item versions track semantic changes
  independently — a scaffold can have a breaking change without bumping
  every other item in the pack. Both pack-level and item-level versions
  appear in the manifest. Pack-level version is what the lockfile pins
  by content hash. Item-level versions enable specs and plans to
  reference specific item versions for reproducibility.
- **DD-50:** Scaffolds declare `use_when` scenarios — a list of
  situations where the scaffold is the right choice. Each scaffold
  entry in the manifest includes a `use_when:` list of plain-language
  scenario descriptions. These serve two consumers: (1) the spec writer
  agent, which pattern-matches requirements against scenarios to choose
  the right scaffold, and (2) the LLM reviewer / `pack validate`, which
  checks whether the scaffold actually delivers what its `use_when`
  scenarios promise. Multiple matches across scaffolds are possible —
  the consuming agent evaluates specificity or escalates to the human.
  The `use_when` list also includes `assumes:` (preconditions the
  scaffold expects in the consumer's codebase) and `pairs_with:` (other
  pack items — rules, SDKs — that should be used alongside this
  scaffold).
- **DD-51:** Specs prescribe pack items by versioned coordinates. The
  spec schema gains a `prescribes:` field (or equivalent) on
  requirements and/or claims that references pack items using
  fully-qualified versioned coordinates:
  `pack-name@pack-version:item-name@item-version`. This formalizes the
  spec's opinion about which scaffolds, SDKs, and rules satisfy a given
  requirement. The spec reviewer can validate: does the prescribed
  scaffold's `use_when` match the requirement being addressed? Does the
  prescribed item exist in the referenced pack version?
- **DD-52:** Plans reference prescribed pack items from the spec. Plan
  tasks that involve rendering scaffolds or importing SDKs reference
  them using the same versioned coordinates the spec prescribed. The
  plan does not independently choose pack items or versions — it
  inherits from the spec. The implementer agent uses these references
  to know exactly what to render and at which version. This ensures the
  implementation matches what was specified, not whatever happens to be
  installed.
- **DD-53:** Gate verifies version consistency across the
  pack->spec->plan->implementation graph. The gate checks: (1) is the
  installed pack version compatible with what the spec prescribed?
  (2) did the implementation actually use the prescribed scaffold/SDK?
  (3) do the prescribed rules pass against the implementation? (4) does
  the lockfile-pinned pack version match the spec's references? Drift
  between any of these is a gate violation. Full traceability from "why
  this scaffold" to "prove it's there and it's the right version."
- **DD-54:** The pack manifest, spec schema, and plan schema form a
  connected prescription graph. Pack declares what's available (items,
  versions, `use_when` scenarios) -> spec prescribes what to use for
  each requirement (versioned coordinates) -> plan says when and how to
  render/import it (task-level references inheriting from spec) -> gate
  verifies it was actually used at the right version. This graph is the
  mechanism that makes pack content *traceable* through the entire
  backstop lifecycle, not just *available*.

- **DD-55:** Pack upgrades auto-generate remediation bundles. When a
  consumer runs `backstop pack upgrade`, the tool scans the codebase
  against the new pack version, captures all new violations, and creates
  a bundle scoping the remediation. This connects the pack lifecycle to
  the backstop artifact lifecycle — upgrades are work items, not
  surprises. The remediation bundle follows the normal workflow: specs,
  plans, agent implementation, gate verification. This is the mechanism
  that makes breaking pack changes (new rules, tighter enforcement) cheap
  to absorb — mechanical verification handles the remediation, so the
  incentive to defer upgrades weakens.
- **DD-56:** Enforcement pack semver model. Adding rules is a major
  version. Loosening rules is minor. Fixing false positives is patch.
  `backstop pack upgrade` uses semver to determine behavior: patch/minor
  auto-upgrade, major generates a remediation bundle and enforces new
  version on new code immediately while existing violations are
  baselined.
- **DD-57:** backstop.yml declares the current enforced pack version.
  One version per pack. All code validated against the current version.
  Spec-pinned versions are provenance/audit, not runtime enforcement.
  Upgrading is a conscious decision with mechanical support.

## Spec Seeds

These are provisional and will firm up as the open questions resolve.
Order is suggested implementation order, not commitment.

- **Pack Manifest & Authoring Contract** — the on-disk pack format,
  manifest schema, claim/fixture/rule relationships, rule risk taxonomy,
  author-facing error vocabulary. The artifact a third-party author
  reads to know what a pack is.
- **Pack Loader (Single Adapter Path)** — the Go-side loader that reads
  any pack (embedded core, local in-repo, cached git tap) through one
  code path. The thing that proves there is no special case for the
  core pack.
- **Pack Validate** — claim-fixture-rule coherence, semgrep --test gate,
  fixture coverage thresholds, rule-risk-class enforcement. Hard
  precondition for loadability.
- **Pack Distribution: Git Taps & Local Cache** — `pack add`, cache
  layout, content-hash computation, in-repo vs tap resolution.
- **Lockfile (`backstop.lock`)** — schema, write/diff/verify operations,
  integration with `gate`, multi-consumer / monorepo behavior.
- **Pack Update & Tamper Detection** — diff format, red-flag events
  (fixture removal, severity downgrade), explicit-acknowledgment flow.
- **Supply Chain Subchecks** — OSV-Scanner, sigstore verification,
  Trivy/Grype, semgrep supply-chain rules wired into `pack validate`
  and the gate kill chain.
- **Signing & Verification** — depends on OQ-1 (sigstore vs gpg vs
  minisign). Author signatures, verification at load, signature
  requirements per rule risk class.
- **LLM Reviewer Artifact** — versioned prompt, rubric, known-bad
  corpus, the backstop artifact that proves the reviewer catches the
  things it claims to catch.
- **Curated Catalog** — submission pipeline, validation gate at
  submission, LLM-first review, human escalation, revocation,
  re-validation, catalog schema, search/query interface.

## Open Questions

- **OQ-1: Signing story.** sigstore (keyless, OIDC-backed) vs gpg
  (entrenched but UX-hostile) vs minisign (simple, less ecosystem) vs
  other. Needs research into pack-author UX, CI integration, and key
  rotation. Lean: sigstore for the curated catalog path; OQ-open for
  ungated taps.

- **OQ-2: What is a "code pack"?** [RE-RESOLVED in 0.8.0 by DD-29/
  DD-46.] Previously resolved in 0.2.0 as "the term dissolves."
  Real-world prototyping with agents extracting packs from production
  codebases revealed code packs are a distinct archetype. Re-resolved:
  code packs are defined by a mandatory co-occurrence rule — if a pack
  declares `sdks` or `scaffolds`, it must also declare `rules`
  enforcing correct usage of that code. The distinction is a validation
  contract (code ships with enforcement, always), not a content type
  distinction. See DD-46 for the full two-archetype model.

- **OQ-3: Sandbox boundary — declarative-only forever?** Declarative-
  only is tractable to verify. Allowing Turing-complete code (Go
  plugins, WASM modules) inside packs makes the sandbox problem much
  harder. Question: stay declarative-only? For how long? What is the
  trigger that would make us reconsider? *Note (0.7.0):* DD-43, DD-44,
  and DD-45 partially address this — layer 3 custom validators are the
  answer to "what about non-declarative checks." They exist but are the
  escape hatch with the highest scrutiny. The remaining open question is
  the exact sandbox implementation (process isolation? seccomp? wasm?
  just convention + review?).

- **OQ-4: Lockfile format specifics.** Exact schema. Behavior in a
  monorepo with multiple consumers. Local-path packs vs git-ref packs
  vs embedded core. Hash algorithm. Lock-file-of-lockfiles for
  workspaces?

- **OQ-5: Known-bad list distribution channel.** The LLM reviewer's
  known-bad corpus and any catalog-side revocation list need to reach
  consumers. Shipped with the binary? Separate signed update channel?
  Ledger-based? How is freshness enforced?

- **OQ-6: Composition / collision model.** [RESOLVED in 0.3.0 by
  DD-30..DD-35.] Rule IDs are namespaced by pack so collisions are
  impossible by construction; semantic overlap is allowed and
  de-duplicated at presentation; contradictions are detected
  mechanically via fixture composition at pack-load and via
  claim-level LLM review; severity is max-wins with explicit
  overrides required to downgrade; backstop never picks a winner
  on the consumer's behalf.

- **OQ-7: Language-to-pack relationship.** Today `backstop.yml` has
  `language: go`. Does that field imply "automatically load the Go
  pack" or is pack selection always fully explicit via `packs: [...]`?
  Implicit loading is convenient but contradicts "no transitive trust"
  in spirit.

- **OQ-8: Pack-level vs rule-level versioning.** If one rule in a pack
  changes semantics, does the whole pack version bump (semver on the
  pack) or do rules carry their own versions? Changelog format?
  Interaction with the tamper-detection diff?

- **OQ-9: Fixture format.** Lean entirely on semgrep `--test` format
  (keeps authors in familiar tooling) or wrap it in a backstop format
  (lets us enforce claim mapping more strictly and accommodate non-
  semgrep engines)? Tradeoff between author ergonomics and enforcement
  reach.

- **OQ-10: In-repo pack vs external tap — semantic difference?** Or is
  it purely a "where did the bytes come from" difference, with
  identical loader behavior, identical validation, identical lockfile
  treatment? Lean: identical, but worth confirming.

- **OQ-11: Non-semgrep rules.** AST checks, contract signature checks,
  test substantiveness patterns, custom Go-side checkers. Do these
  live in packs too? If so, the rule format must accommodate multiple
  engines and the loader must dispatch by engine. If not, we end up
  with a parallel "checks-that-aren't-packs" concept that fragments
  the model.

- **OQ-12: Registry rubric timing.** The curated catalog is deferred,
  but the LLM-reviewer rubric needs to be designed early enough that
  `pack validate` enforces the same things the catalog would.
  Otherwise we get a two-tier validation gap. When does the rubric
  get written? Before or after the first non-core pack ships?

- **OQ-13: Revocation propagation.** When a pack is revoked in the
  curated catalog, how do consumers find out? Poll on `gate`? Push
  via a signed feed? Append-only ledger they can verify? How is a
  revoked pack's continued local presence handled — hard error, warn,
  configurable?

- **OQ-14: Per-language rule variants.** [RESOLVED in 0.4.0 by
  DD-17/DD-19.] Dead under the single-language-per-pack model. Each
  pack has rules in exactly one language; there are no variants to
  reconcile. A cross-language capability is a family of
  single-language packs, each with its own rules in its own language.

- **OQ-15: SDK version skew.** If a pack manifest says `sdk.go@v1.2.0`
  but the consumer's `go.mod` resolves to `@v1.0.0`, is that a
  pack-load failure, a warning, or ignored? How does the lockfile
  represent expected vs resolved version? Does `gate` refuse to run
  on skew, or only warn?

- **OQ-16: SDK surface verification (simplified in 0.4.0).** Under
  the language-specific pack model (DD-17), verification targets a
  single language per pack — one parser, one toolchain, one
  ecosystem. The cross-language complexity is gone. The remaining
  question is how backstop verifies the SDK's actual exported
  surface matches the `provides:` list in the manifest. Options:
  (a) static parse of the SDK module in the pack's one language;
  (b) build + introspect using that language's toolchain;
  (c) trust the manifest, verify only at publish time. Lean: (c)
  with publish-time introspection, because consumer-side
  introspection still adds toolchain burden even when scoped to one
  language.

- **OQ-17: Credential storage model.** [RESOLVED in 0.5.0 — deferred
  to BUNDLE-002.] No credentials are needed for the git-tap + catalog
  model. Publishers push to their own git repos; catalog submission
  does not require registry tokens. Credential management for
  native-registry fan-out is addressed in BUNDLE-002.

- **OQ-18: Attestation format and transport.** [RESOLVED in 0.5.0 —
  deferred to BUNDLE-002.] Attestations require the publishing proxy
  to produce them. Without the proxy, attestation format and transport
  are not applicable to the core pack lifecycle. Addressed in
  BUNDLE-002.

- **OQ-19: Yank asymmetry across native registries.** [RESOLVED in
  0.5.0 — not applicable.] Under the git-tap + catalog model, the
  catalog is the only index and is under backstop's control (DD-39).
  There are no native registries to yank from. The yank-asymmetry
  problem only arises under the publishing-proxy model (BUNDLE-002).

- **OQ-20: Consumer verification UX.** Is `backstop pack verify` run
  manually? Automatically on every `gate`? On CI only? On `pack
  install` only? What happens when an installed SDK has no attestation
  at all (pre-backstop or published outside the proxy) — hard error,
  warn, or allowed? The default matters a lot for adoption friction.

- **OQ-21: Publisher self-hosting the pre-publish gate.** Can a
  publisher run the pre-publish gate on their own infra and produce
  their own attestations, or must they go through backstop's hosted
  pipeline? Self-hosted preserves openness and avoids backstop being
  a single point of failure, but weakens the trust anchor (anyone
  can claim to have run the gate). Trade-off: decentralization vs
  trust strength.

- **OQ-22: Cross-registry publish consistency.** [RESOLVED in 0.5.0 —
  deferred to BUNDLE-002.] Cross-registry fan-out and the partial-
  publish consistency problem are only relevant under the publishing-
  proxy model. Addressed in BUNDLE-002.

- **OQ-23: Attestation freshness and re-issuance.** [RESOLVED in
  0.5.0 — deferred to BUNDLE-002.] Attestation freshness is only
  relevant when attestations exist. Addressed in BUNDLE-002.

- **OQ-24: Order-of-evaluation dependencies.** If a pack shipped an
  auto-fix (we currently don't plan to), the fix would need to run
  before another pack's rule evaluated against the result. Probably
  moot because backstop doesn't run auto-fix, but flag it so we
  don't sleepwalk into supporting fixes and break composition
  semantics.

- **OQ-25: Composition check performance.** Running N packs'
  fixtures through M packs' rules is O(N*M). For large pack counts
  and large fixture sets, this is expensive on every `pack add` /
  `pack update` / `gate`. Options: cache composition results keyed
  by the multiset of pack content hashes; only re-run on pack-set
  changes; incremental composition where only the delta is
  re-checked. Needs measurement before committing to an approach.

- **OQ-26: Negative-fixture weakening strictness.** Strict
  interpretation: every negative fixture from every pack must still
  fail under composition. Loose interpretation: a negative fixture
  is allowed to stop failing if another pack's rule catches the
  same offense via a different rule ID (enforcement preserved, just
  attributed differently). Strict is safer but forces pack authors
  to coordinate; loose is more permissive but needs machinery to
  confirm "the same offense is still caught by *something*." Lean
  strict for v1 because it needs no cross-rule semantic reasoning.

- **OQ-27: Catalog submission workflow.** What triggers submission?
  Publisher runs `backstop pack submit <git-url>`? Opens a PR to a
  catalog repo? Hits an API? How is the submitter's identity verified?
  What metadata does the submitter provide vs what backstop infers
  from the pack manifest? Options: (a) CLI command that posts to an
  API, (b) PR to a git-hosted catalog repo (Homebrew model), (c) both
  with the git repo as source of truth. Lean: (b) or (c) — a PR-based
  flow is transparent and auditable, and aligns with the git-native
  distribution model. *Note (0.6.0):* DD-41 partially answers this —
  the submission flow requires the shared CI workflow referencing
  `backstop-org/pack-ci` + green CI on the submitted ref. The remaining
  open parts are: what command triggers submission (`backstop pack
  submit`?), what metadata does the submitter provide vs what backstop
  infers from the pack manifest, and how is submitter identity verified.

- **OQ-28: Catalog hosting and availability.** Is the catalog a git
  repo itself (like Homebrew/homebrew-core), a hosted API, or both?
  A git repo is maximally transparent and forkable but harder to
  search. A hosted API is searchable but introduces an availability
  dependency. Both (git repo as source of truth, API as read layer)
  is probably right but adds operational surface. Lean: both, with
  the git repo as the canonical store and the API as a convenience
  layer.

- **OQ-29: Catalog staleness policy.** How often does periodic
  re-validation run? What's the grace period before a failing entry
  is downgraded? Does a downgraded entry get automatically relisted
  when the publisher fixes the issue, or must they resubmit? Options:
  (a) daily re-validation, 7-day grace, auto-relist on pass;
  (b) weekly re-validation, immediate downgrade, manual resubmit;
  (c) configurable per-entry. Lean: (a) — frequent enough to catch
  rot, lenient enough not to punish transient failures.

- **OQ-30: Non-GitHub publisher support.** The shared CI workflow
  (DD-41) is GitHub Actions-specific. Publishers on GitLab, Bitbucket,
  Codeberg, or self-hosted git are excluded from catalog listing unless
  equivalent shared pipelines are shipped for each platform. Options:
  (a) GitHub-only for v1, document the limitation, ship other platforms
  later; (b) define a platform-agnostic "validation attestation" format
  that any CI system can produce, and accept that instead of requiring a
  specific workflow integration; (c) offer a backstop-hosted validation
  service that publishers can call from any CI. Each trades off between
  simplicity, reach, and operational burden. Lean: (a) for v1 — GitHub
  dominates the OSS hosting space and shipping one platform well is
  better than shipping three poorly.

- **OQ-31: Mechanical acceptance/rejection heuristics for layer 3
  submissions.** The principle "try layers 1-2 first" is clear, but
  the mechanical filter for accepting or rejecting a layer 3 submission
  needs more work. The single-file vs multi-file distinction matters
  for sandboxing but is not sufficient as a graduation criterion —
  legitimate single-file layer 3 validators exist (binary inspection,
  format validation, semantic analysis). Questions: Can `pack validate`
  mechanically determine whether a check is expressible in semgrep?
  Probably not in general — that's undecidable. Is a self-declaration
  model sufficient ("I declare this can't be a semgrep rule because:
  binary input / structural check / semantic analysis")? Should the LLM
  reviewer evaluate layer 3 submissions specifically for "could this
  have been a semgrep rule?" What's the cost of a false rejection vs
  false acceptance? Lean: self-declaration with LLM reviewer
  cross-check, but needs more thought.

- **OQ-32: Scaffold rendering and validation mechanics.** Complete
  scaffolds must "pass out of the box with config-only changes." This
  implies pack validation needs to render the scaffold with sample/
  default config values and run the resulting tests. Questions: Where
  do sample config values live — in the scaffold's manifest entry, in
  a separate fixture config, or inferred? How does `pack validate`
  handle scaffolds that need real external services (Stripe API,
  database connections) — mock config? Docker-compose fixtures? Test
  doubles baked into the scaffold? How heavy is this validation step —
  does it run on every `pack validate` or only on `pack test`? Lean:
  sample config in the manifest, scaffold tests use test doubles (not
  real services), validation runs the full render+test cycle.

- **OQ-33: Item-level versioning granularity.** Is item-level
  versioning mandatory for all content types (rules, scaffolds, SDKs,
  contracts, test_patterns, ast_checks, rubrics), or only for scaffolds
  and SDKs? Rules change frequently and versioning each one individually
  may be high overhead for pack authors. Lean: mandatory for scaffolds
  and SDKs (consumer-facing, breaking changes matter), optional for
  rules (enforcement changes are covered by pack-level version + tamper
  detection from DD-10).

- **OQ-34: Spec prescription field design.** Where exactly does
  `prescribes:` live in the spec schema — on each requirement, on each
  claim, or as a top-level section mapping requirements to pack items?
  Each has different ergonomics. Per-requirement is most precise but
  verbose. Per-claim ties prescriptions to specific test scenarios.
  Top-level section is easier to maintain but loses the requirement-level
  connection. Needs prototyping with spec writer agents to see what
  feels right.

- **OQ-35: Version compatibility semantics.** When a spec prescribes
  `pack@1.2.0:scaffold@v2` but the installed pack is `@1.3.0` (still
  containing `scaffold@v2`), is that a gate pass or fail? Strict pin
  (exact match only) vs compatible range (semver-compatible is OK) vs
  item-level pin (pack version can float if the item version matches).
  Lean: item-level pin — the spec cares about the item version, the
  lockfile cares about the pack version. If the installed pack contains
  the right item at the right version, it passes regardless of
  pack-level version bump.

- **OQ-36: Backward compatibility of spec/plan schema changes.** Adding
  `prescribes:` to the spec schema and pack-item references to the plan
  schema is a schema evolution. Existing specs and plans don't have
  these fields. Are they required or optional? If required, all existing
  specs need updating. If optional, the gate can't enforce prescription
  consistency on older specs. Lean: optional for now, required when
  maturity allows.

## Sharp Edges / Risks

- **Catalog points at repos backstop doesn't control.** Force pushes,
  deleted repos, and tag mutation are all possible. Pinned refs +
  content hashing in the lockfile protect individual consumers; the
  catalog's periodic re-validation (DD-40) catches rot at the index
  level. But there is an inherent gap between a consumer's last
  `pack update` and the catalog's next re-validation run.
- **Pack-as-capability (DD-17) changes discoverability.** "Find a
  pack for capability X across language Y" is a different query
  shape than "find the Go pack." The catalog must index on both axes.
- **Runtime contradictions that don't appear in fixtures.** A pack
  author can't write fixtures for patterns they never anticipated.
  Composition passes at pack-load; real consumer code later
  triggers an unreported contradiction. Mitigation: when
  composition detects rule-coverage overlap between two packs on
  the consumer's *actual* codebase (not just fixtures), warn the
  pack pair is under-tested for this consumer.
- **Remediation loops.** Consumer fixes a file to satisfy pack A,
  now violates pack B. Composition check at pack-load prevents
  shipping the configuration, but the consumer still has to resolve
  it manually. The point is to fail fast at configuration time, not
  make fixes automatic.
- **Language-variant rules across capability packs.** A language
  pack and a capability pack can both target the same language with
  disagreeing rules. Same composition check catches it; no
  special-casing needed, but noted here because it will be the most
  common real-world collision.
- **Duplication across family members.** A publisher shipping a
  cross-language capability will duplicate manifest metadata, claim
  text, and scaffolding across their family of packs. This is
  accepted as the cost of the simpler model — forced unification
  would hold back each language's pack at the slowest common
  denominator. Drift between family members is acceptable and
  arguably healthy; the publisher owns keeping them coordinated via
  changelog and CI.
- **Family membership is convention, not enforcement.** There is no
  mechanical check that `acme/stripe-sheets-go@v1.2.0` and
  `acme/stripe-sheets-python@v1.2.0` actually belong to the same
  capability. Naming prefix and version cadence are the only
  signals. A malicious publisher could squat on a family name.
  Mitigation: the catalog's curation process can verify publisher
  identity consistency across family members at submission time.
- **GitHub-specific coupling.** The shared workflow (DD-41) ties the
  catalog to GitHub Actions. Worth the tradeoff for v1 (GitHub dominates
  OSS), but creates a platform lock-in risk. If backstop gains traction
  outside the GitHub ecosystem, this becomes a real gap.
- **Shared workflow versioning.** Bumping the workflow version may cause
  existing packs to fail if the new version is stricter. Need a
  migration path: pin to major version, deprecation warnings, grace
  period before downgrading listings.
- **Publisher can fork the workflow.** Mitigation: catalog checks that
  the `uses:` reference points at the canonical `backstop-org/pack-ci`
  repo. GitHub Actions syntax makes this verifiable by inspecting the
  workflow file.
- **CI-as-gate has latency.** Submission is not instant — publisher
  submits, CI runs, catalog lists when green. Acceptable for the
  workflow but worth setting expectations.
- **Layer 3 is the supply chain risk surface.** Layers 1 and 2 are
  declarative patterns — hard to weaponize, easy to audit. Layer 3 is
  executable code — the place where a malicious pack would hide its
  payload. The entire custom validator sandbox model is load-bearing
  for pack security. Every relaxation of layer 3 constraints widens
  the attack surface.
- **"Try layers 1-2 first" is hard to enforce mechanically.** A pack
  author can always claim "semgrep can't express this" for a check
  that semgrep actually can express. Without a way to mechanically
  verify the claim, enforcement relies on review (human or LLM). The
  self-declaration + reviewer model in OQ-31 is the best option
  identified so far, but it's softer enforcement than the rest of the
  pack model.
- **Complete scaffolds that need external services are hard to validate
  in CI.** A Stripe webhook scaffold's tests need either a Stripe
  test-mode key or mocked HTTP. The pack author must ship the test
  doubles as part of the scaffold — validation can't depend on external
  service availability. This is a constraint on what "complete" means:
  complete includes its own test isolation.
- **The boundary between "config change" and "implementation change" in
  complete scaffolds is fuzzy.** If the scaffold requires the consumer
  to implement a custom serializer or a project-specific middleware, is
  that still "config-only"? The pack manifest needs a clear definition
  of what the consumer is expected to change vs what they must leave
  alone. Otherwise "complete scaffold" becomes meaningless as a tier.
- **Item-level versioning adds authoring overhead.** Pack authors must
  now track versions per scaffold, per SDK, not just per pack. Tooling
  (`backstop pack bump`) should automate this, but the cognitive load
  is real. If we're not careful, version management becomes the thing
  that makes pack authoring painful — exactly the burden we're trying
  to avoid (DD-1).
- **Prescription coordinates create tight coupling between specs and
  pack versions.** If a pack reorganizes its items (renames, splits,
  merges scaffolds), every spec that prescribes those items breaks.
  Pack authors need to treat item names and versions as a public API
  with backward-compatibility obligations. This is a feature (stability)
  and a risk (rigidity).

## Notes / Ideas

- The integration gap pattern flagged in agent memory (implementations
  nail unit tests but miss wiring between packages) is exactly the
  failure mode this bundle is most exposed to. Loader / validator /
  scanner / lockfile / gate must be wired end-to-end with integration
  tests, not just unit tests per package.
- The "core tap" framing is the cheapest possible enforcement of
  "no special cases for the core pack" — if the core pack ever needs
  loader behavior third-party packs can't have, we have a design bug.
- The reviewer-as-backstop-artifact recursion is also the cleanest way
  to demonstrate backstop's thesis: even our pack curation is
  mechanically verified.

## Version History

- 0.1.0 (2026-04-08): Initial bundle. Captured problem (current pack
  model is underspecified, would impose 14-interface burden if scaled),
  vision (declarative packs, git-tap distribution, mandatory mechanical
  validation, LLM-primary curated review), 15 resolved design decisions
  carried in from prior discussion, 13 open questions, 10 spec seeds.
  Maturity: exploring. OQs intentionally not resolved in this pass —
  scope is deliberately broad to avoid premature narrowing.
- 0.2.0 (2026-04-08): Folded in a follow-up discussion. Added typed
  extensible pack `content:` block (rules, scaffolds, sdks, contracts,
  test_patterns, ast_checks, rubrics, fixtures) via DD-16. Reframed
  packs as capabilities rather than language buckets (DD-17), which
  makes the embedded Go pack just another capability pack. Dissolved
  the loose "code pack" term (DD-18, DD-29) and resolved OQ-2
  accordingly. Introduced the publishing-proxy model: backstop
  validates and fans SDKs out to native registries behind a
  pre-publish gate (DD-22), consumers always install natively
  (DD-23), publishers own their end-user support contract (DD-24),
  pre-publish is the single point of enforcement (DD-25). Added
  signed attestations as the consumer-side verification anchor
  (DD-26) and reframed the "curated registry" as a metadata-only
  catalog + attestation index (DD-27), simplifying DD-6. Strong lean
  toward sigstore noted (DD-28) while leaving OQ-1 open pending the
  credential/identity model. Clarified scaffold vs SDK ownership
  (DD-19), made `sdks` a list (DD-20), and established that pack
  rules can enforce SDK usage (DD-21). Added ten new open questions
  (OQ-14..OQ-23) covering per-language rule variants, SDK version
  skew, surface verification, credential storage, attestation
  format, cross-registry yank asymmetry, consumer verification UX,
  publisher self-hosting the gate, cross-registry publish
  consistency, and attestation freshness. Added a Sharp Edges /
  Risks section covering credential-store risk, permanent cross-
  ecosystem inconsistency, verification-mandate theater, and the
  discoverability shift implied by capability-framed packs.
  Maturity unchanged (exploring).
- 0.3.0 (2026-04-08): Resolved OQ-6 (composition / collision model).
  Added six new design decisions: rule IDs auto-namespaced by pack
  so collisions are impossible by construction (DD-30); semantic
  overlap allowed and de-duplicated at presentation (DD-31);
  contradictions detected mechanically via fixture composition at
  pack-load (DD-32); severity max-wins with explicit overrides
  required to downgrade (DD-33); claim-level contradiction
  detection via the LLM reviewer (DD-34); no automatic precedence —
  unresolvable contradictions force consumer action (DD-35). Added
  three new open questions: order-of-evaluation dependencies if
  auto-fix is ever supported (OQ-24), composition check
  performance scaling (OQ-25), strict vs loose negative-fixture
  weakening (OQ-26). Added three new sharp edges: runtime
  contradictions invisible to fixtures, remediation loops between
  packs, and language-variant rule collisions across capability
  packs. Maturity unchanged (exploring).
- 0.4.0 (2026-04-08): Major simplification — packs are now
  language-specific by construction. Cross-language capabilities are
  expressed as a family of single-language packs coordinated by
  convention (shared publisher, name prefix, lockstep version
  cadence), not as a single multi-language artifact. DD-17 revised
  from "pack as capability spanning languages" to "language-specific
  enforcement unit"; DD-19 revised so scaffolds and SDKs are both
  scoped to the pack's single language; DD-20 revised so `sdk:` is a
  singular optional entry, not a list. Added DD-36 (cross-language
  equivalence is author responsibility, not backstop concern — false
  precision to model) and DD-37 (sprawl is still addressed, just via
  composition at the consumer layer in `backstop.yml`). Resolved
  OQ-14 (per-language rule variants — dead under single-language
  packs). Simplified OQ-16 (SDK surface verification) to
  single-language scope, still open. Added two new sharp edges:
  duplication across family members (accepted cost) and family
  membership as convention rather than mechanical enforcement
  (mitigated by publisher-identity attestations). Maturity unchanged
  (exploring).
- 0.5.0 (2026-04-09): Scoped BUNDLE-001 to core pack lifecycle +
  lightweight catalog. Extracted publishing proxy, attestations,
  credentials, and native-registry fan-out to BUNDLE-002. DD-22,
  DD-25, DD-26, DD-28 marked deferred (pointing to BUNDLE-002).
  DD-38 (lightweight curated catalog), DD-39 (catalog revocation),
  DD-40 (periodic re-validation) added. DD-2 revised to include
  catalog framing; DD-6 revised to supersede "final-approver registry"
  with lighter catalog model; DD-19 noted SDK publishing deferred;
  DD-27 revised from "attestation index" to "curated index." OQ-17,
  OQ-18, OQ-19, OQ-22, OQ-23 resolved as deferred to BUNDLE-002.
  OQ-27 (catalog submission workflow), OQ-28 (catalog hosting), OQ-29
  (catalog staleness policy) added. Sharp edges revised: removed
  credential and cross-ecosystem risks (moved to BUNDLE-002), added
  catalog-points-at-uncontrolled-repos risk, revised family-membership
  mitigation from attestation-based to catalog-curation-based.
  Maturity unchanged (exploring).
- 0.6.0 (2026-04-08): Added shared CI workflow requirement for catalog
  listing (DD-41, DD-42). Added OQ-30 (non-GitHub publisher support).
  Added sharp edges for GitHub coupling, workflow versioning, workflow
  forking, CI latency. Partially addressed OQ-27 via DD-41. Maturity
  unchanged (exploring).
- 0.7.0 (2026-04-08): Added three-layer enforcement model (DD-43,
  DD-44, DD-45). Layer 1 = built-in tool rules, Layer 2 = custom
  declarative rules, Layer 3 = custom validators as escape hatch.
  Added OQ-31 (mechanical acceptance heuristics for layer 3). Two new
  sharp edges (layer 3 as supply chain risk, enforcement of "try 1-2
  first"). Annotated DD-16 re: ast_checks as layer 3. Annotated OQ-3
  as partially addressed. Maturity unchanged (exploring).
- 0.8.0 (2026-04-08): Re-resolved OQ-2 — code packs are a real
  archetype with a mandatory co-occurrence rule (code must ship with
  enforcement). Added DD-46 (two pack archetypes: enforcement packs
  and code packs), DD-47 (scaffold tiers: complete vs skeleton with
  different verification expectations), DD-48 (mechanical proof
  required for all archetypes at their tier's completeness level).
  Added OQ-32 (scaffold rendering and validation mechanics). Two new
  sharp edges (external service validation in complete scaffolds,
  config vs implementation boundary). Revised DD-29 (re-resolved with
  co-occurrence rule), annotated DD-18 (recipes map to scaffold tiers),
  annotated DD-16 (archetype is alongside content types), annotated
  DD-19 (pointer to DD-47 scaffold tier split). Motivated by real-world
  agent prototyping of pack extraction from production codebases.
  Maturity unchanged (exploring).
- 1.1.0 (2026-04-08): Refined DD-30 — specified `pack-name/rule-id`
  slash-delimited format for namespaced rule IDs. Added note that version
  is not embedded in ID (tracked in lockfile/backstop.yml). Added
  industry convention references (ESLint, semgrep, golangci-lint).
- 1.0.0 (2026-04-08): Added pack upgrade/remediation model (DD-55:
  upgrades auto-generate remediation bundles following normal backstop
  workflow), enforcement pack semver (DD-56: adding rules is major,
  loosening is minor, false-positive fixes are patch), single enforced
  version policy (DD-57: backstop.yml declares current version, spec-
  pinned versions are provenance not enforcement). Extracted from
  BUNDLE-004 design sessions. Maturity unchanged (exploring).
- 0.9.0 (2026-04-08): Added item-level versioning within packs (DD-49),
  scaffold `use_when` scenarios for agent-readable selection (DD-50),
  spec-level prescription of pack items by versioned coordinates
  (DD-51), plan-level references inheriting from specs (DD-52), gate
  verification of version consistency across the full graph (DD-53),
  and the connected prescription graph model (DD-54). Added OQ-33
  (versioning granularity), OQ-34 (prescription field design), OQ-35
  (version compatibility semantics), OQ-36 (schema backward
  compatibility). Two new sharp edges (authoring overhead, tight
  coupling). Motivated by real-world prototyping of spec writer agents
  selecting scaffolds from pack manifests. Maturity unchanged
  (exploring).

## References

- BUNDLE-002: Pack Publishing Proxy (deferred publishing proxy,
  attestations, credential management, native-registry fan-out)
- SPEC-012: Go standards pack (becomes the "core tap" under this model)
- cmd/backstop/pack_new.go, cmd/backstop/pack_compile.go (current
  scaffolding)
- ADR-0010: Verification Kill Chain (where supply chain subchecks slot in)
- ADR-0014: Runtime Integration (passive enforcement model)
- Agent memory: integration gap pattern (SPEC-008 lesson) — applies
  directly to the loader/validator/scanner/lockfile wiring risk
- External tools to integrate as subchecks: OSV-Scanner, sigstore/cosign,
  semgrep supply-chain rules, Trivy, Grype
