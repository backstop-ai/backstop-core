---
title: "SPEC-049: Waiver Subsystem — Accountable, Engine-Neutral Suppression"
number: SPEC-049
created: "2026-07-09"
status: draft
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: >
    Implement the backstop waiver subsystem from BUNDLE-013 (all 15
    requirements, one coupled spec). A waiver is a backstop-native inline DSL
    token `@waiver:<rule-id>:<reason-code>:<expiry>` (plus optional trailing
    note/issue-ref) that a human or runtime agent writes inside whatever comment
    their language already uses, at the source location of a finding. Backstop
    adjudicates by LINE-SCANNING the finding's own SARIF-reported location for
    the token — it reads raw bytes at a location the engine already reported and
    NEVER parses source or encodes any language's comment syntax (the
    load-bearing zero-baked property). Identity is per-finding and location-based
    (no stored fingerprint, no file/rule blanket). Every waiver carries a
    mandatory expiry defaulted by reason-code; on expiry the shield lifts and the
    finding re-fires, with a loud pre-expiry warning. A declared non-waivable
    tier (pack-manifest/policy-declared, shipping with backstop/self rules and
    critical-severity secrets) makes waiving a cardinal invariant a gate error.
    Malformed tokens are themselves gate findings. Waivers apply to code-located
    findings only; structural dimensions keep their existing accountable
    resolution paths. Seed D replaces the pkg/gate/step_deferred.go
    waiver-resolution stub with real adjudication, adds the distinct
    pass-with-waivers terminal state and reporting, the ISSUE-050 ratchet
    interaction, the pre-filled-token affordance, the read-only
    `backstop waiver list` CLI, and the step-9 audit data-feed. Work is phased
    A (DSL + adjudication core) -> B (lifecycle/expiry) -> C (non-waivable tier)
    -> D (gate step-8 integration, reporting, ratchet, CLI); D integrates the
    rest. Core adjudication lives in pkg/waiver; gate integration in pkg/gate;
    CLI in cmd/backstop.
  subject: pkg/waiver

verification:
  level: integration
  test_command: go test ./pkg/waiver/... ./pkg/gate/... ./cmd/backstop/... -race -coverprofile=cover.out
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: >
      Backstop must define a waiver DSL token of the form
      `@waiver:<rule-id>:<reason-code>:<expiry>` with an optional trailing
      free-text note and/or issue reference. `reason-code` is a CLOSED enum with
      exactly four members: `false-positive`, `accepted-risk`, `deferred`,
      `third-party`. No other reason-code value is valid. `expiry` is an
      ISO-8601 date (YYYY-MM-DD). The grammar must parse and validate
      deterministically: `ParseToken` returns a populated Waiver for a
      well-formed token and a non-nil error for any malformed token.
    supports: waiver-subsystem:REQ-001
    follows: STD-GO-001:GO-010
  - id: REQ-002
    text: >
      Waiver adjudication must be inline and engine-neutral. Engines emit
      findings normally (no report-everything pack contract is imposed), and
      backstop suppresses a finding by LINE-SCANNING the finding's own reported
      SARIF location for a matching `@waiver` token. Adjudication must remain
      zero-baked: `Adjudicate` receives the findings plus a `LineReader` that
      yields the RAW bytes of a requested source line, and MUST NOT receive a
      language identifier, MUST NOT invoke any comment parser, and MUST NOT
      encode any language's comment syntax. Because `@waiver` is a token foreign
      to every engine, the finding is emitted regardless of engine; backstop does
      100% of the suppressing.
    supports: waiver-subsystem:REQ-002
    follows: STD-GO-001:GO-010
  - id: REQ-003
    text: >
      Waiver identity must be per-finding and location-based, not a stored
      content fingerprint. A waiver applies ONLY to the specific finding at the
      token's associated location. There is NO file-blanket and NO rule-blanket
      scope: a second finding of the same rule at a different location in the
      same file is NOT suppressed by a token co-located with the first finding.
      A debt-heavy file that is touched is discharged finding-by-finding.
    supports: waiver-subsystem:REQ-003
    follows: STD-GO-001:GO-010
  - id: REQ-004
    text: >
      Every waiver must carry a mandatory expiry; no waiver is permanent. Each
      reason-code sets a DEFAULT duration used when authoring guidance is
      generated: `false-positive` long-lived (~1yr), and `accepted-risk`,
      `deferred`, and `third-party` short-lived (~90d). Durations are tunable
      configuration, not contract, but every reason-code MUST resolve to a
      default duration. An ACTIVE (unexpired) waiver suppresses its finding; on
      expiry the waiver becomes inactive and the finding re-fires under normal
      enforcement. The gate must emit a loud heads-up BEFORE a waiver expires
      (the pre-expiry warning is the grace period).
    supports: waiver-subsystem:REQ-004
    follows: STD-GO-001:GO-010
  - id: REQ-005
    text: >
      The gate must detect and warn on unused/dangling waivers — a `@waiver`
      token whose associated location no longer has a matching live finding. An
      unused waiver is surfaced as a WARNING (not a block and not silent); a
      waiver with a matching live finding at its location is NOT flagged unused.
    supports: waiver-subsystem:REQ-005
    follows: STD-GO-001:GO-010
  - id: REQ-006
    text: >
      Backstop must support a DECLARED non-waivable tier: a rule or severity
      self-declares itself un-waivable in its pack manifest / enforcement policy
      — declared, NOT a core-hardcoded list. The waivability decision is made by
      a `Policy` supplied to adjudication; core code contains no baked list of
      protected rules. The shipped configuration places backstop/self rules and
      critical-severity secrets in the non-waivable set. A `@waiver` token
      targeting a rule the Policy reports non-waivable is a gate ERROR, not a
      suppression. A rule NOT in the declared non-waivable set remains waivable.
    supports: waiver-subsystem:REQ-006
    follows: STD-GO-001:GO-010
  - id: REQ-007
    text: >
      Malformed `@waiver` tokens must themselves be gate findings. A token with
      too few fields (bad structure), an unknown reason-code, a missing expiry,
      or an invalid expiry format each produces a gate finding at the token's
      location. The waiver grammar is enforced, not best-effort. A well-formed
      token is NOT flagged as malformed.
    supports: waiver-subsystem:REQ-007
    follows: STD-GO-001:GO-010
  - id: REQ-008
    text: >
      Token-to-finding location matching semantics must be defined precisely.
      Line association: a token associates with a finding if it appears TRAILING
      on the finding's own start line OR on the line IMMEDIATELY ABOVE the
      finding's start line. A token two or more lines above the finding does NOT
      associate. Multi-line findings: a token on the line immediately above the
      finding's start line associates with the whole finding region. False-match
      avoidance: because backstop cannot parse the language, a token only
      suppresses when its `<rule-id>` EXACTLY equals the finding's own rule-id;
      a `@waiver`-looking string whose rule-id differs from the finding's rule-id
      (e.g. an incidental occurrence inside a string literal) does NOT suppress
      the finding. Residual risk when a fully-valid token naming the exact rule
      appears inside a string literal is accepted and documented (Sharp Edges).
    supports: waiver-subsystem:REQ-008
    follows: STD-GO-001:GO-010
  - id: REQ-009
    text: >
      `rule-id` in the token must be a stable, ergonomic identifier for the rule
      being waived. When a pack renames a rule, the stale rule-id matches no live
      finding and the waiver surfaces as unused/dangling (per REQ-005) rather
      than silently waiving a DIFFERENT rule. A waiver whose rule-id matches the
      live finding's rule-id suppresses it; a waiver never suppresses a
      co-located finding whose rule-id differs from the token's rule-id.
    supports: waiver-subsystem:REQ-009
    follows: STD-GO-001:GO-010
  - id: REQ-010
    text: >
      Waivers apply to CODE-LOCATED findings only, and the classification must be
      EXHAUSTIVE over the gate's current live dimensions so no dimension is
      accidentally mis-scoped. The WAIVABLE (code-located) dimensions are exactly
      `pack_engines` (pack code-rule findings, including the flagship
      non-waivable `backstop/self` rules — these findings flow through
      `pack_engines`, NOT the removed `code_check` dimension) and
      `test_substantiveness` (source-located per the substantiveness routing). The
      NON-WAIVABLE dimensions are `artifact_status_drift`, `contract_signature`,
      `test_verification`, and `artifact_validation`; they may NOT be suppressed
      by a `@waiver` token and keep their existing accountable resolution paths
      (retire / replace / resolved-by / obsoleted). The file-level
      `coverage_threshold` dimension is locationless and uses a DEFINED
      annotation convention rather than a per-line source token (see REQ below /
      Implementation): a `@waiver:coverage_threshold:<reason>:<expiry>` token on
      the FILE's first line waives that file's coverage finding.
    supports: waiver-subsystem:REQ-010
    follows: STD-GO-001:GO-010
  - id: REQ-011
    text: >
      The core CLI is READ-ONLY with respect to waivers: `backstop waiver list`
      must report active, expiring-soon, and unused/dangling waivers. Core
      backstop must NOT write or insert `@waiver` tokens — inserting a comment
      requires language comment-syntax, which is baked-language knowledge — so no
      code path in core authors tokens; authoring and re-certification belong to
      the human or the runtime agent.
    supports: waiver-subsystem:REQ-011
    follows: STD-GO-001:GO-010
  - id: REQ-012
    text: >
      Gate reporting must never be silent about waivers. A run that passes
      BECAUSE of active waivers is a DISTINCT terminal state (`PASS · N waivers`)
      visually distinguishable from a clean pass; a run with zero waivers must
      NOT render as the waiver state. An active-waiver summary is ALWAYS shown;
      the actionable subset (expiring-soon and unused) is surfaced inline on
      every run; full per-waiver detail is available on demand. (Surfacing
      engine-native VISIBLE suppressions on a separate line is a Notes/Ideas
      follow-on per bundle DD-6, NOT a v1 requirement — see Sharp Edges.)
    supports: waiver-subsystem:REQ-012
    follows: STD-GO-001:GO-010
  - id: REQ-013
    text: >
      Waivers must interact correctly with the ISSUE-050 file-level ratchet. A
      valid ACTIVE waiver satisfies the ratchet for its finding (the accountable
      "or waived" branch). An EXPIRED waiver does NOT satisfy the ratchet (the
      live finding stands and demands action). An UNUSED waiver satisfies
      nothing. Baseline generation (a machine snapshot) and waiver authoring (a
      human decision) remain DISTINCT operations — baseline generation never
      authors waivers.
    supports: waiver-subsystem:REQ-013
    follows: STD-GO-001:GO-010
  - id: REQ-014
    text: >
      When the gate blocks on a WAIVABLE finding, its output must hand the author
      a pre-filled, neutral `@waiver:<rule>:<reason>:<expiry>` token for that
      specific finding (carrying the finding's own rule-id and a
      reason-code-defaulted expiry), so acknowledging is at most as much friction
      as writing an engine-native `//nolint`. The gate must NOT emit a pre-filled
      token for a NON-WAIVABLE finding (it cannot be waived) nor for a
      structural/non-code finding (outside the waivable surface).
    supports: waiver-subsystem:REQ-014
    follows: STD-GO-001:GO-010
  - id: REQ-015
    text: >
      The active-waiver set must be exposed to the gate's step-9 audit/ledger
      surface via a data-feed accessor, so that "what are we deliberately
      ignoring, why, and until when" is a first-class auditable question. Because
      the existing `GateResult`/`Violation` types carry no reason-code or expiry,
      the waiver reconciliation pass must persist the active `[]waiver.Waiver`
      (each carrying at least rule-id, reason-code, and expiry) onto a NEW carrier
      field on `GateResult` (e.g. `ActiveWaivers []waiver.Waiver`), and
      `ActiveWaiverFeed` reads that field. Step 9 itself is unbuilt; this
      requirement is the data-feed CONTRACT, not step-9 implementation.
    supports: waiver-subsystem:REQ-015
    follows: STD-GO-001:GO-010
  - id: REQ-016
    text: >
      The waiver reconciliation must be ENABLED AND FED at the shipped
      `backstop gate` construction site, not merely defined at the pkg/gate
      level, or shipped `backstop gate` stays dark and suppresses nothing. This
      requires BOTH seams that the baseline analog has (not just the pkg/gate
      swap): (a) a `WithWaiver(...)` gate Option in pkg/gate/gate.go that sets a
      new `g.waiverEnabled = true` and attaches the runtime inputs, mirroring
      `WithBaseline`; the Run-loop swap is guarded `if g.waiverEnabled &&
      result.StepName == StepWaiverResolution`, mirroring the baseline guard; and
      (b) `cmd/backstop/gate.go` must CALL `WithWaiver` at the same construction
      site where it calls `gate.WithBaseline`, constructing the runtime inputs the
      pass consumes: a `LineReader` over the active scope, the current time, and
      the production `Policy`. The production `Policy` MUST be built by EXTRACTING
      the declared non-waivable sets from the INSTALLED pack manifests — the
      CLM-027 "declared, not hardcoded" mechanism realized IN PRODUCTION, not just
      the shipped-config data used to construct the pkg/waiver `Policy` type. The
      shipped `backstop gate` path must therefore actually suppress a code-located
      finding via a `@waiver` and error on a `backstop/self` non-waivable waiver
      end-to-end over the installed packs.
    supports: waiver-subsystem:REQ-011
    follows: STD-GO-001:GO-010
  - id: REQ-017
    text: >
      Waiver reconciliation MUST run BEFORE the baseline/ratchet evaluation in the
      shipped step list so the accumulated violation set is ALREADY
      waiver-subtracted when `baseline_comparison` captures its NewViolations.
      Because `baseline_comparison` reads the accumulated set at its loop position,
      an active waiver only satisfies the ISSUE-050 ratchet (REQ-013 / CLM-055) if
      `StepWaiverResolution` is ordered BEFORE `StepBaselineComparison` in the
      assembled `cmd/backstop/gate.go` step list. Ordering waiver after baseline
      (the current deferred-stub order) would let an active-waived finding still
      count as a new violation against the ratchet, defeating REQ-013. The shipped
      step order must place waiver resolution ahead of baseline comparison.
    supports: waiver-subsystem:REQ-013
    follows: STD-GO-001:GO-010

claims:
  # REQ-001 — DSL grammar + closed reason-code enum
  - id: CLM-001
    requirement: REQ-001
    text: A well-formed token `@waiver:<rule>:<reason>:<expiry>` parses into a populated Waiver
    tests:
      - TestWaiver_ParseToken_WellFormedParses
  - id: CLM-002
    requirement: REQ-001
    text: A token with an optional trailing free-text note parses and captures the note
    tests:
      - TestWaiver_ParseToken_OptionalNoteCaptured
  - id: CLM-003
    requirement: REQ-001
    text: A token with an optional issue reference parses and captures the issue-ref
    tests:
      - TestWaiver_ParseToken_OptionalIssueRefCaptured
  - id: CLM-004
    requirement: REQ-001
    text: reason-code `false-positive` is accepted by the grammar
    tests:
      - TestWaiver_ParseToken_ReasonFalsePositiveValid
  - id: CLM-005
    requirement: REQ-001
    text: reason-code `accepted-risk` is accepted by the grammar
    tests:
      - TestWaiver_ParseToken_ReasonAcceptedRiskValid
  - id: CLM-006
    requirement: REQ-001
    text: reason-code `deferred` is accepted by the grammar
    tests:
      - TestWaiver_ParseToken_ReasonDeferredValid
  - id: CLM-007
    requirement: REQ-001
    text: reason-code `third-party` is accepted by the grammar
    tests:
      - TestWaiver_ParseToken_ReasonThirdPartyValid
  - id: CLM-008
    requirement: REQ-001
    text: An unknown reason-code (outside the closed enum) is rejected with a parse error
    tests:
      - TestWaiver_ParseToken_UnknownReasonRejected

  # REQ-002 — zero-baked engine-neutral line-scan adjudication
  - id: CLM-009
    requirement: REQ-002
    text: An engine-emitted finding is suppressed when a matching token sits at the finding's reported location
    tests:
      - TestWaiver_Adjudicate_SuppressesAtReportedLocation
  - id: CLM-010
    requirement: REQ-002
    text: Adjudication suppresses regardless of the surrounding comment prefix, reading only raw line bytes via LineReader
    tests:
      - TestWaiver_Adjudicate_CommentSyntaxAgnostic
  - id: CLM-011
    requirement: REQ-002
    kind: absence
    text: Adjudicate takes no language identifier and invokes no comment/source parser (zero-baked)
    tests:
      - TestWaiver_Adjudicate_NoLanguageParserInvoked
  - id: CLM-067
    requirement: REQ-002
    subject: pkg/gate
    text: The waiver reconciliation pass REMOVES suppressed findings from the already-accumulated pack_engines violation set (the subtraction actually mutates the accumulated results, else it suppresses nothing)
    tests:
      - TestGateWaiver_Suppress_MutatesAccumulatedPackEngines

  # REQ-003 — per-finding location identity, no blanket scope
  - id: CLM-012
    requirement: REQ-003
    text: A waiver suppresses exactly the finding co-located with its token
    tests:
      - TestWaiver_Identity_SuppressesColocatedFinding
  - id: CLM-013
    requirement: REQ-003
    text: A same-rule finding at a different location in the same file is NOT suppressed (no file-blanket)
    tests:
      - TestWaiver_Identity_NoFileBlanket
  - id: CLM-014
    requirement: REQ-003
    text: With two same-rule findings, waiving one leaves the other firing (no rule-blanket)
    tests:
      - TestWaiver_Identity_NoRuleBlanket

  # REQ-004 — mandatory expiry, reason-code default durations, re-fire, pre-expiry warning
  - id: CLM-015
    requirement: REQ-004
    text: reason-code `false-positive` resolves to a long-lived default duration
    tests:
      - TestWaiver_DefaultDuration_FalsePositiveLongLived
  - id: CLM-016
    requirement: REQ-004
    text: reason-code `accepted-risk` resolves to a short-lived default duration
    tests:
      - TestWaiver_DefaultDuration_AcceptedRiskShortLived
  - id: CLM-017
    requirement: REQ-004
    text: reason-code `deferred` resolves to a short-lived default duration
    tests:
      - TestWaiver_DefaultDuration_DeferredShortLived
  - id: CLM-018
    requirement: REQ-004
    text: reason-code `third-party` resolves to a short-lived default duration
    tests:
      - TestWaiver_DefaultDuration_ThirdPartyShortLived
  - id: CLM-019
    requirement: REQ-004
    text: An active (unexpired) waiver suppresses its finding
    tests:
      - TestWaiver_Expiry_ActiveWaiverSuppresses
  - id: CLM-020
    requirement: REQ-004
    text: An expired waiver does NOT suppress — the finding re-fires under normal enforcement
    tests:
      - TestWaiver_Expiry_ExpiredWaiverRefires
  - id: CLM-021
    requirement: REQ-004
    text: A loud pre-expiry warning is emitted within the grace window before a waiver expires
    tests:
      - TestWaiver_Expiry_PreExpiryWarningEmitted

  # REQ-005 — unused/dangling detection
  - id: CLM-022
    requirement: REQ-005
    text: A waiver with no live finding at its location is surfaced as an unused/dangling warning
    tests:
      - TestWaiver_Unused_DanglingSurfacedAsWarning
  - id: CLM-023
    requirement: REQ-005
    text: A waiver with a matching live finding at its location is NOT flagged unused
    tests:
      - TestWaiver_Unused_UsedWaiverNotFlagged

  # REQ-006 — declared non-waivable tier (matrix: waivable vs each protected class)
  - id: CLM-024
    requirement: REQ-006
    text: A waiver on a normal code rule (not in the non-waivable set) suppresses the finding
    tests:
      - TestWaiver_NonWaivable_NormalRuleWaivable
  - id: CLM-025
    requirement: REQ-006
    text: A waiver targeting a backstop/self rule is a gate ERROR, not a suppression
    tests:
      - TestWaiver_NonWaivable_SelfRuleIsError
  - id: CLM-026
    requirement: REQ-006
    text: A waiver targeting a critical-severity secret is a gate ERROR, not a suppression
    tests:
      - TestWaiver_NonWaivable_CriticalSecretIsError
  - id: CLM-027
    requirement: REQ-006
    text: The non-waivable set is supplied by a declared Policy, with no core-hardcoded rule list
    tests:
      - TestWaiver_NonWaivable_PolicyDeclaredNotHardcoded
  - id: CLM-028
    requirement: REQ-006
    text: A rule not present in the declared non-waivable set remains waivable
    tests:
      - TestWaiver_NonWaivable_UndeclaredRuleRemainsWaivable

  # REQ-007 — malformed token is itself a gate finding (matrix of malformations)
  - id: CLM-029
    requirement: REQ-007
    text: A token with too few fields (bad structure) is reported as a gate finding
    tests:
      - TestWaiver_Malformed_BadStructureIsFinding
  - id: CLM-030
    requirement: REQ-007
    text: A token with an unknown reason-code is reported as a gate finding
    tests:
      - TestWaiver_Malformed_UnknownReasonIsFinding
  - id: CLM-031
    requirement: REQ-007
    text: A token with a missing expiry is reported as a gate finding
    tests:
      - TestWaiver_Malformed_MissingExpiryIsFinding
  - id: CLM-032
    requirement: REQ-007
    text: A token with an invalid expiry format is reported as a gate finding
    tests:
      - TestWaiver_Malformed_InvalidExpiryIsFinding
  - id: CLM-033
    requirement: REQ-007
    text: A well-formed token is NOT reported as malformed
    tests:
      - TestWaiver_Malformed_WellFormedNotFlagged

  # REQ-008 — token-to-finding location matching semantics
  - id: CLM-034
    requirement: REQ-008
    text: A token trailing on the finding's own start line associates and suppresses
    tests:
      - TestWaiver_Match_SameLineTrailingSuppresses
  - id: CLM-035
    requirement: REQ-008
    text: A token on the line immediately above the finding associates and suppresses
    tests:
      - TestWaiver_Match_LineAboveSuppresses
  - id: CLM-036
    requirement: REQ-008
    text: A token two or more lines above the finding does NOT associate (surfaces as unused)
    tests:
      - TestWaiver_Match_TwoLinesAboveDoesNotAssociate
  - id: CLM-037
    requirement: REQ-008
    text: For a multi-line finding, a token above the finding's start line associates with the whole region
    tests:
      - TestWaiver_Match_MultiLineFindingAssociates
  - id: CLM-066
    requirement: REQ-008
    text: For a multi-line finding, a token TRAILING on the finding's start line associates and suppresses
    tests:
      - TestWaiver_Match_MultiLineTrailingStartLineAssociates
  - id: CLM-038
    requirement: REQ-008
    text: A token whose rule-id differs from the finding's rule-id does NOT suppress (false-match avoidance)
    tests:
      - TestWaiver_Match_MismatchedRuleIdNoSuppress

  # REQ-009 — stable rule-id + rename behavior
  - id: CLM-039
    requirement: REQ-009
    text: A waiver whose rule-id matches the live finding's rule-id suppresses it
    tests:
      - TestWaiver_RuleId_MatchingRuleSuppresses
  - id: CLM-040
    requirement: REQ-009
    text: After a pack renames a rule, the stale rule-id matches no live finding and surfaces as unused/dangling
    tests:
      - TestWaiver_RuleId_RenamedRuleSurfacesUnused

  # REQ-010 — waivable-surface scope boundary (EXHAUSTIVE matrix over live dimensions)
  - id: CLM-041
    requirement: REQ-010
    subject: pkg/gate
    text: A pack_engines finding (the LIVE pack code-rule dimension) is waivable
    tests:
      - TestGateWaiver_Scope_PackEnginesWaivable
  - id: CLM-064
    requirement: REQ-010
    subject: pkg/gate
    text: A test_substantiveness finding (source-located) is waivable
    tests:
      - TestGateWaiver_Scope_SubstantivenessWaivable
  - id: CLM-042
    requirement: REQ-010
    subject: pkg/gate
    text: An artifact_status_drift finding is NOT waivable by a @waiver token
    tests:
      - TestGateWaiver_Scope_StatusDriftNotWaivable
  - id: CLM-043
    requirement: REQ-010
    subject: pkg/gate
    text: A contract_signature finding is NOT waivable by a @waiver token
    tests:
      - TestGateWaiver_Scope_ContractSignatureNotWaivable
  - id: CLM-044
    requirement: REQ-010
    subject: pkg/gate
    text: A test_verification finding is NOT waivable by a @waiver token
    tests:
      - TestGateWaiver_Scope_TestVerificationNotWaivable
  - id: CLM-065
    requirement: REQ-010
    subject: pkg/gate
    text: An artifact_validation finding is NOT waivable by a @waiver token
    tests:
      - TestGateWaiver_Scope_ArtifactValidationNotWaivable
  - id: CLM-045
    requirement: REQ-010
    subject: pkg/gate
    text: A locationless coverage_threshold finding is waived by a @waiver:coverage_threshold token on the file's FIRST line (annotation convention), not a per-line source token
    tests:
      - TestGateWaiver_Scope_CoverageFirstLineAnnotation

  # REQ-011 — read-only CLI
  - id: CLM-046
    requirement: REQ-011
    subject: cmd/backstop
    text: "`backstop waiver list` reports active waivers"
    tests:
      - TestWaiverList_ReportsActive
  - id: CLM-047
    requirement: REQ-011
    subject: cmd/backstop
    text: "`backstop waiver list` reports expiring-soon waivers"
    tests:
      - TestWaiverList_ReportsExpiringSoon
  - id: CLM-048
    requirement: REQ-011
    subject: cmd/backstop
    text: "`backstop waiver list` reports unused/dangling waivers"
    tests:
      - TestWaiverList_ReportsUnused
  - id: CLM-049
    requirement: REQ-011
    kind: absence
    text: No core code path writes or inserts a @waiver token (authoring belongs to human/agent)
    tests:
      - TestWaiver_ReadOnly_NoTokenAuthoringInCore

  # REQ-012 — reporting never silent
  - id: CLM-050
    requirement: REQ-012
    subject: pkg/gate
    text: A run passing because of active waivers renders the distinct `PASS · N waivers` terminal state
    tests:
      - TestGateWaiver_Report_DistinctPassWithWaiversState
  - id: CLM-051
    requirement: REQ-012
    subject: pkg/gate
    text: A clean run with zero waivers does NOT render as the waiver state
    tests:
      - TestGateWaiver_Report_CleanRunNotWaiverState
  - id: CLM-052
    requirement: REQ-012
    subject: pkg/gate
    text: The active-waiver summary is always shown on a run with active waivers
    tests:
      - TestGateWaiver_Report_SummaryAlwaysShown
  - id: CLM-053
    requirement: REQ-012
    subject: pkg/gate
    text: The actionable subset (expiring-soon and unused) is surfaced inline every run
    tests:
      - TestGateWaiver_Report_ActionableSubsetInline

  # REQ-013 — ISSUE-050 ratchet interaction (matrix: active / expired / unused)
  - id: CLM-055
    requirement: REQ-013
    subject: pkg/gate
    text: A valid ACTIVE waiver satisfies the file-level ratchet for its finding
    tests:
      - TestGateWaiver_Ratchet_ActiveSatisfies
  - id: CLM-056
    requirement: REQ-013
    subject: pkg/gate
    text: An EXPIRED waiver does NOT satisfy the ratchet (live finding demands action)
    tests:
      - TestGateWaiver_Ratchet_ExpiredDoesNotSatisfy
  - id: CLM-057
    requirement: REQ-013
    subject: pkg/gate
    text: An UNUSED waiver satisfies nothing in the ratchet
    tests:
      - TestGateWaiver_Ratchet_UnusedSatisfiesNothing
  - id: CLM-058
    requirement: REQ-013
    subject: pkg/gate
    text: Baseline generation does not author waivers — the two operations stay distinct
    tests:
      - TestGateWaiver_Ratchet_BaselineDoesNotAuthorWaivers

  # REQ-014 — pre-filled neutral token on a blocked waivable finding
  - id: CLM-059
    requirement: REQ-014
    subject: pkg/gate
    text: Blocking on a waivable finding emits a pre-filled @waiver token carrying that finding's rule-id and defaulted expiry
    tests:
      - TestGateWaiver_Prefill_WaivableFindingEmitsToken
  - id: CLM-060
    requirement: REQ-014
    subject: pkg/gate
    text: A non-waivable finding does NOT get a pre-filled waiver token
    tests:
      - TestGateWaiver_Prefill_NonWaivableFindingNoToken
  - id: CLM-061
    requirement: REQ-014
    subject: pkg/gate
    text: A structural/non-code finding does NOT get a pre-filled waiver token
    tests:
      - TestGateWaiver_Prefill_StructuralFindingNoToken

  # REQ-015 — step-9 audit data-feed contract
  - id: CLM-062
    requirement: REQ-015
    subject: pkg/gate
    text: The active-waiver set is exposed to the step-9 audit surface via the data-feed accessor
    tests:
      - TestGateWaiver_AuditFeed_ActiveSetExposed
  - id: CLM-063
    requirement: REQ-015
    subject: pkg/gate
    text: Each audit-feed entry carries the rule-id, reason-code, and expiry of its waiver
    tests:
      - TestGateWaiver_AuditFeed_EntryCarriesRuleReasonExpiry

  # REQ-016 — production construction-site wiring + real-CLI-path proof
  - id: CLM-068
    requirement: REQ-016
    subject: pkg/gate
    text: WithWaiver sets waiverEnabled and the Run-loop swaps StepWaiverResolution for computeWaiverResult only when enabled
    tests:
      - TestGateWaiver_Wiring_WithWaiverEnablesReconciliation
  - id: CLM-069
    requirement: REQ-016
    subject: cmd/backstop
    text: The production Policy is built by EXTRACTING the declared non-waivable sets from the installed pack manifests, not from a hardcoded list
    tests:
      - TestWaiverPolicy_ExtractedFromInstalledPackManifests
  - id: CLM-070
    requirement: REQ-016
    subject: cmd/backstop
    text: The SHIPPED `backstop gate` construction path (not gate.New inside package gate) suppresses a code-located finding via a @waiver over the committed installed-pack fixture
    tests:
      - TestGateCLI_Waiver_SuppressesOverInstalledPack
  - id: CLM-071
    requirement: REQ-016
    subject: cmd/backstop
    text: The SHIPPED `backstop gate` path errors when a @waiver targets a backstop/self non-waivable rule over the installed-pack fixture
    tests:
      - TestGateCLI_Waiver_SelfRuleErrorsOverInstalledPack

  # REQ-017 — waiver-before-baseline ordering in the shipped pipeline
  - id: CLM-072
    requirement: REQ-017
    subject: cmd/backstop
    text: The assembled step list orders StepWaiverResolution BEFORE StepBaselineComparison
    tests:
      - TestGateCLI_StepOrder_WaiverBeforeBaseline
  - id: CLM-073
    requirement: REQ-017
    subject: cmd/backstop
    text: Against the REAL pipeline order, an active waiver subtracts its finding before baseline captures NewViolations, so the finding does not count against the ratchet
    tests:
      - TestGateCLI_Ratchet_ActiveWaiverSubtractedBeforeBaseline

contracts:
  - file: pkg/waiver/waiver.go
    provides:
      - name: ReasonCode
        kind: type
        signature: "type ReasonCode string"
        notes: "Closed enum: false-positive, accepted-risk, deferred, third-party"
      - name: Waiver
        kind: type
        signature: "type Waiver struct { RuleID string; Reason ReasonCode; Expiry time.Time; Note string; IssueRef string; File string; Line int }"
        notes: "One parsed inline waiver, located per-finding"
      - name: ParseToken
        kind: function
        signature: "func ParseToken(raw string, loc Location) (Waiver, error)"
        notes: "Grammar parse + validation; non-nil error for any malformed token (REQ-001/REQ-007)"
      - name: DefaultDuration
        kind: function
        signature: "func DefaultDuration(rc ReasonCode) (time.Duration, bool)"
        notes: "Reason-code default expiry duration; ok=false for an unknown reason-code"
      - name: Expired
        kind: method
        signature: "func (w Waiver) Expired(now time.Time) bool"
        notes: "True when now is at or past the waiver's expiry"
  - file: pkg/waiver/adjudicate.go
    provides:
      - name: Finding
        kind: type
        signature: "type Finding struct { RuleID string; File string; Line int; EndLine int; Severity string }"
        notes: "A code-located engine finding, at its SARIF-reported location"
      - name: Location
        kind: type
        signature: "type Location struct { File string; Line int }"
        notes: "A file+line coordinate for a token"
      - name: LineReader
        kind: type
        signature: "type LineReader func(file string, line int) (string, bool)"
        notes: "Yields RAW bytes of one source line; the ONLY source access adjudication gets — no language identifier, no comment parser (REQ-002)"
      - name: Policy
        kind: interface
        signature: "type Policy interface { Waivable(ruleID string, severity string) bool }"
        notes: "Declared non-waivable tier; supplied by pack manifest/enforcement policy, never core-hardcoded (REQ-006)"
      - name: Diagnostic
        kind: type
        signature: "type Diagnostic struct { RuleID string; File string; Line int; Message string; Kind string }"
        notes: "A waiver-produced gate finding; Kind is malformed or non-waivable"
      - name: Result
        kind: type
        signature: "type Result struct { Suppressed []Finding; Active []Waiver; Expiring []Waiver; Expired []Waiver; Unused []Waiver; Malformed []Diagnostic; NonWaivable []Diagnostic }"
        notes: "Adjudication outcome consumed by the gate step and reporting"
      - name: Adjudicate
        kind: function
        signature: "func Adjudicate(findings []Finding, read LineReader, policy Policy, now time.Time) Result"
        notes: "Zero-baked line-scan adjudication; receives no language identifier (REQ-002/REQ-003/REQ-008)"
    consumes:
      - source: pkg/waiver
        name: Waiver
        kind: type
      - source: pkg/waiver
        name: ParseToken
        kind: function
  - file: pkg/gate/step_waiver.go
    provides:
      - name: StepWaiverResolutionScopedFunc
        kind: function
        signature: "func StepWaiverResolutionScopedFunc(scope *GateScope) StepFunc"
        notes: "The registered waiver_resolution step function — a placeholder that emits the skipped/pending StepResult, exactly mirroring StepBaselineComparisonScopedFunc. When waivers are enabled the Run loop REPLACES its output with computeWaiverResult (see gate.go), the same way the loop already replaces the baseline step with computeBaselineResult. The loop GAINS a waiver reconciliation pass; it is NOT unchanged."
      - name: StepWaiverResolutionFunc
        kind: function
        signature: "func StepWaiverResolutionFunc() StepFunc"
        notes: "Unscoped constructor delegating to the scoped form"
      - name: WithWaiver
        kind: function
        signature: "func WithWaiver(read waiver.LineReader, policy waiver.Policy, now time.Time) Option"
        notes: "Construction-site enablement seam mirroring WithBaseline (gate.go:51-53): sets a new g.waiverEnabled = true and attaches the runtime inputs (LineReader, Policy, now) that computeWaiverResult consumes. The Run-loop swap is guarded `if g.waiverEnabled && result.StepName == StepWaiverResolution` (mirror gate.go:139). Without this seam AND the cmd/backstop/gate.go call, shipped `backstop gate` stays dark and suppresses nothing (REQ-016)."
      - name: computeWaiverResult
        kind: method
        signature: "func (g *Gate) computeWaiverResult(accumulated []StepResult, read waiver.LineReader, policy waiver.Policy, now time.Time) StepResult"
        notes: "Run-loop reconciliation pass MODELED ON computeBaselineResult (gate.go:163). Sees ALL accumulated StepResults; recasts the waivable-surface preceding-step Violations (pack_engines, test_substantiveness) into []waiver.Finding, calls the pure pkg/waiver.Adjudicate, REMOVES the Suppressed findings from the accumulated results' Violation slices (the actual subtraction — CLM-067), persists the active []waiver.Waiver onto GateResult.ActiveWaivers (S1 carrier), and returns the waiver_resolution StepResult (distinct pass-with-waivers state, summary, actionable subset)."
      - name: ActiveWaiverFeed
        kind: function
        signature: "func ActiveWaiverFeed(res GateResult) []waiver.Waiver"
        notes: "Step-9 audit data-feed accessor (REQ-015); reads the new GateResult.ActiveWaivers carrier field populated by computeWaiverResult (GateResult/Violation carry no reason-code/expiry, so a dedicated carrier is required)."
    consumes:
      - source: pkg/waiver
        name: Adjudicate
        kind: function
      - source: pkg/waiver
        name: Result
        kind: type
      - source: pkg/waiver
        name: Finding
        kind: type
      - source: pkg/waiver
        name: LineReader
        kind: type
      - source: pkg/waiver
        name: Policy
        kind: interface
      - source: pkg/waiver
        name: Waiver
        kind: type
  - file: cmd/backstop/waiver.go
    provides:
      - name: waiverCmd
        kind: variable
        signature: "var waiverCmd *cobra.Command"
        notes: "Read-only `backstop waiver` parent command"
      - name: waiverListCmd
        kind: variable
        signature: "var waiverListCmd *cobra.Command"
        notes: "`backstop waiver list` — active / expiring-soon / unused"
      - name: runWaiverList
        kind: function
        signature: "func runWaiverList(cmd *cobra.Command, args []string) error"
        notes: "Cobra RunE handler; read-only, never writes tokens (REQ-011)"
    consumes:
      - source: pkg/waiver
        name: Waiver
        kind: type
      - source: pkg/gate
        name: ActiveWaiverFeed
        kind: function
  - file: cmd/backstop/gate.go
    provides:
      - name: buildWaiverPolicy
        kind: function
        signature: "func buildWaiverPolicy(projectRoot string) (waiver.Policy, error)"
        notes: "Builds the PRODUCTION Policy by EXTRACTING the declared non-waivable sets from the INSTALLED pack manifests — the CLM-027 declared-not-hardcoded mechanism realized in production (REQ-016). backstop/self rules and critical-severity secrets ship in the non-waivable set."
      - name: buildWaiverLineReader
        kind: function
        signature: "func buildWaiverLineReader(projectRoot string, scope *gate.GateScope) waiver.LineReader"
        notes: "Constructs the LineReader over the active scope that computeWaiverResult consumes; yields RAW bytes of a requested source line, no language knowledge (REQ-016)."
    consumes:
      - source: pkg/gate
        name: WithWaiver
        kind: function
      - source: pkg/gate
        name: StepWaiverResolutionScopedFunc
        kind: function
      - source: pkg/gate
        name: StepBaselineComparisonScopedFunc
        kind: function
      - source: pkg/waiver
        name: Policy
        kind: interface
      - source: pkg/waiver
        name: LineReader
        kind: type
---

# SPEC-049: Waiver Subsystem — Accountable, Engine-Neutral Suppression

## Overview

Backstop has no accountable, gate-visible way to say "I acknowledge this finding
and am choosing not to fix it right now." Today's only two escape valves are
both bad: engine-native suppressions (`//nolint`, `nosemgrep`,
`#gitleaks:allow`) are engine-specific and vanish inside the engine before
backstop's SARIF ever sees them, and silent baseline grandfathering hides
pre-existing debt with no justification and no expiry. This spec implements the
BUNDLE-013 waiver subsystem: a first-class, engine-neutral, tracked, justified,
expiring, gate-visible "chosen not to fix."

A waiver is a backstop-native inline DSL token
`@waiver:<rule-id>:<reason-code>:<expiry>` (plus optional trailing note/issue-ref)
that a human or the runtime agent writes inside whatever comment their language
already uses, at the finding's source location. Backstop adjudicates by
**line-scanning the finding's own SARIF-reported location** for the token.
Because `@waiver` is a token foreign to every engine, engines emit findings
normally and backstop does 100% of the suppressing — no report-everything pack
contract, and it stays zero-baked: backstop reads raw bytes at a location the
engine already reported and never parses source or learns any language's comment
syntax. This is the load-bearing property of the whole subsystem.

This is ONE coupled spec covering all 15 BUNDLE-013 requirements. It is phased by
the bundle's four spec seeds and must be implemented A -> B -> C -> D:

- **Seed A — DSL + zero-baked line-scan adjudication (foundation).** Token
  grammar and reason-code enum, per-finding location identity, the engine-neutral
  line-scan, token-to-finding matching semantics, stable rule-id + rename
  behavior, and malformed-token-as-gate-finding. Covers REQ-001, REQ-002,
  REQ-003, REQ-007, REQ-008, REQ-009.
- **Seed B — Lifecycle & expiry.** Mandatory expiry, reason-code default
  durations, re-fire on expiry, the loud pre-expiry warning, and unused/dangling
  detection. Covers REQ-004, REQ-005.
- **Seed C — Declared non-waivable tier.** Pack-manifest/policy-declared
  un-waivable rules/severities, the shipped backstop/self + critical-secrets set,
  and gate-error-on-waiving-a-protected-rule. Covers REQ-006.
- **Seed D — Gate step-8 integration, reporting, ratchet & CLI.** Adds a
  waiver-resolution RECONCILIATION PASS to the gate Run loop (modeled on the
  existing baseline pass) that adjudicates and subtracts suppressed findings from
  the accumulated results, replacing the `pkg/gate/step_deferred.go` skipped
  stub; the distinct pass-with-waivers terminal state and reporting; the
  ISSUE-050 ratchet interaction; the pre-filled neutral token; the read-only
  `backstop waiver list` CLI; the exhaustive code-located-only scope boundary
  over the LIVE gate dimensions; and the step-9 audit feed via a new
  `GateResult` carrier field. Covers REQ-010, REQ-011, REQ-012, REQ-013, REQ-014,
  REQ-015.

## Requirements

Requirements REQ-001 through REQ-015 are defined in frontmatter and trace to
BUNDLE-013 REQ-001..015 via `supports`. Each requirement has at least one claim;
claims are defined in frontmatter.

### Waiver token grammar

A waiver token has the shape:

```
@waiver:<rule-id>:<reason-code>:<expiry>[ <note-and/or-issue-ref>]
```

| Field | Required | Constraint |
|-------|----------|------------|
| `rule-id` | yes | Stable ergonomic identifier of the rule being waived; must EXACTLY equal the finding's own rule-id to suppress |
| `reason-code` | yes | Closed enum: `false-positive` \| `accepted-risk` \| `deferred` \| `third-party` — no other value valid |
| `expiry` | yes | ISO-8601 date (`YYYY-MM-DD`); mandatory, no permanent waivers |
| note / issue-ref | no | Optional trailing free text and/or issue reference |

Any deviation (too few fields, unknown reason-code, missing or invalid expiry)
is itself a gate finding (REQ-007).

### Reason-code default durations

Every reason-code resolves to a default expiry duration (used to pre-fill the
authoring token, REQ-014). Durations are tunable configuration, not contract.

| reason-code | default duration | lifetime class |
|-------------|------------------|----------------|
| `false-positive` | ~1yr | long-lived |
| `accepted-risk` | ~90d | short-lived |
| `deferred` | ~90d | short-lived |
| `third-party` | ~90d | short-lived |

### Waivable surface (REQ-010 scope boundary)

Waivers apply to **code-located findings only**. The table below is the complete,
EXHAUSTIVE matrix over the gate's current LIVE dimensions — it must match the
REQ-010 claims exactly. Note `code_check` is GONE; pack code-rule findings
(including the flagship non-waivable `backstop/self`) flow through `pack_engines`.

| Live gate dimension | Class | Waivable by `@waiver`? | Accountable path if not |
|---------------------|-------|------------------------|-------------------------|
| `pack_engines` | code-located | YES | — |
| `test_substantiveness` | code-located (source-located per substantiveness routing) | YES | — |
| `artifact_status_drift` | structural | NO | retire / replace / resolved-by / obsoleted |
| `contract_signature` | structural | NO | retire / replace / resolved-by / obsoleted |
| `test_verification` | structural | NO | retire / replace / resolved-by / obsoleted |
| `artifact_validation` | structural | NO | fix the artifact / retire |
| `coverage_threshold` | file-level / locationless | NO (per-line token) | annotation convention: `@waiver:coverage_threshold:<reason>:<expiry>` on the file's FIRST line |

### Non-waivable tier (REQ-006)

Waivability is decided by a declared `Policy`, never a core-hardcoded list.

| Rule class | Waivable? | Outcome of a `@waiver` targeting it |
|------------|-----------|-------------------------------------|
| Normal code rule, not in the declared set | YES | Suppressed (if active) |
| `backstop/self` rule (shipped non-waivable) | NO | Gate ERROR |
| Critical-severity secret (shipped non-waivable) | NO | Gate ERROR |
| Rule not present in the declared non-waivable set | YES | Suppressed (if active) |

### Ratchet interaction (REQ-013)

| Waiver state | Satisfies ISSUE-050 file-level ratchet? |
|--------------|------------------------------------------|
| ACTIVE (valid, unexpired) | YES — the accountable "or waived" branch |
| EXPIRED | NO — live finding stands, ratchet demands action |
| UNUSED (no live finding) | NO — satisfies nothing |

## Implementation

### Package layout

- `pkg/waiver` — the DSL grammar (`ParseToken`), the zero-baked line-scan
  adjudicator (`Adjudicate`), lifecycle/expiry (`DefaultDuration`, `Expired`),
  and the declared non-waivable `Policy` interface. Core logic; no language or
  comment-syntax knowledge.
- `pkg/gate` — `step_waiver.go` replaces the `step_deferred.go`
  waiver-resolution stub, wires adjudication into gate step 8, enforces the
  code-located-only scope boundary, produces the distinct pass-with-waivers
  reporting, feeds the ratchet, emits the pre-filled token, and exposes the
  step-9 audit feed.
- `cmd/backstop` — the read-only `backstop waiver list` command.

### Adjudication pipeline (the mechanical steps a planner maps tasks to)

`Adjudicate(findings, read, policy, now)` runs these passes; each is
independently testable:

1. **Token harvest (zero-baked line-scan).** For each finding, read the RAW
   bytes of the finding's association window via `LineReader` — the finding's
   own start line (trailing token) and the line immediately above it — and scan
   those bytes for a literal `@waiver:` occurrence. No comment parser, no
   language identifier (REQ-002, REQ-008).
2. **Grammar parse (`ParseToken`).** Parse each harvested occurrence. A malformed
   token becomes a `Diagnostic{Kind: "malformed"}` (REQ-001, REQ-007).
3. **Rule-id match.** A parsed token suppresses ONLY when its `rule-id` exactly
   equals the finding's rule-id (false-match avoidance / rename safety — REQ-008,
   REQ-009). Location association is per-finding; no file/rule blanket (REQ-003).
4. **Non-waivable check.** If `policy.Waivable(ruleID, severity)` is false, the
   token does not suppress and becomes a `Diagnostic{Kind: "non-waivable"}` gate
   error (REQ-006).
5. **Lifecycle classification.** For a waivable, matched token: if
   `w.Expired(now)` the finding re-fires (NOT suppressed) and the waiver is
   recorded Expired; otherwise it is Active and its finding is Suppressed. Active
   waivers within the grace window are also recorded Expiring for the pre-expiry
   warning (REQ-004).
6. **Unused/dangling detection.** Any `@waiver` token whose associated location
   has no matching live finding is recorded Unused (REQ-005, REQ-009 rename).

The `Result` carries `Suppressed`, `Active`, `Expiring`, `Expired`, `Unused`,
`Malformed`, and `NonWaivable`, which the gate step consumes for suppression,
reporting, ratchet, and the audit feed.

### Gate step 8 integration — a run-loop reconciliation pass (Seed D)

A `StepFunc` is `func(ctx) StepResult` (pkg/gate/result.go): it returns only its
OWN result and CANNOT reach back into preceding steps to remove their
violations. But waiver suppression must delete waived findings from the
ALREADY-accumulated `pack_engines` / `test_substantiveness` violations. The gate
already solves exactly this shape for baselines: `computeBaselineResult(accumulated
[]StepResult) StepResult` (pkg/gate/gate.go:163) is a Run-loop reconciliation pass
that sees ALL accumulated results, and the Run loop swaps the baseline step's
placeholder output for it (`if g.baselineEnabled && result.StepName ==
StepBaselineComparison { result = g.computeBaselineResult(results) }`).
`StepWaiverResolution` is likewise already a recognized meta/deferred step
(pkg/gate/policy.go, baseline.go, gate.go accumulator skip-list).

Waiver suppression is specified the SAME way — the Run loop GAINS a waiver
reconciliation pass (it is not "unchanged"):

1. `StepWaiverResolutionScopedFunc` stays registered as the placeholder waiver
   step (mirrors `StepBaselineComparisonScopedFunc`), emitting a skipped/pending
   `StepResult` when waivers are disabled.
2. When waivers are enabled, the Run loop replaces that step's output with
   `g.computeWaiverResult(results, read, policy, now)`, mirroring the baseline
   swap.
3. `computeWaiverResult` collects ONLY the waivable-surface findings from the
   accumulated results — `pack_engines` and `test_substantiveness` (REQ-010);
   `artifact_status_drift`, `contract_signature`, `test_verification`, and
   `artifact_validation` are excluded, and `coverage_threshold` is handled by the
   first-line annotation convention. It recasts those `Violation`s into
   `[]waiver.Finding`.
4. It calls the pure `pkg/waiver.Adjudicate(findings, read, policy, now)`.
5. It **removes** the `Suppressed` findings from the accumulated results' own
   `Violations` slices — the actual subtraction that makes suppression real
   (CLM-067) — and adds `Malformed` and `NonWaivable` diagnostics as gate
   findings (a `@waiver` on a `pack_engines` `backstop/self` rule is thus a gate
   ERROR — CLM-025).
6. It persists the active `[]waiver.Waiver` onto the new `GateResult.ActiveWaivers`
   carrier field (needed because `GateResult`/`Violation` carry no
   reason-code/expiry — S1), feeding both the step-9 audit surface via
   `ActiveWaiverFeed` (REQ-015) and the ratchet (Active satisfies;
   Expired/Unused do not — REQ-013).
7. It returns the `waiver_resolution` `StepResult` carrying the distinct
   `PASS · N waivers` terminal state, the always-on active-waiver summary, and
   the inline actionable subset (Expiring + Unused) (REQ-012).

The pre-filled neutral token (REQ-014) is emitted for a blocked WAIVABLE finding
(`pack_engines` / `test_substantiveness`) only, never for a non-waivable or
structural finding.

### Coverage annotation convention (REQ-010)

`coverage_threshold` findings are file-level and locationless — there is no
single source line to co-locate a token with (the OQ-4 parked fallback). The
convention keeps the same DSL with a defined location rule for locationless
findings: a `@waiver:coverage_threshold:<reason>:<expiry>` token on the FILE's
FIRST line waives that file's coverage finding. This is the only dimension that
uses the first-line rule; all code-located dimensions use per-finding
association (REQ-008).

### Shipped construction site — enable AND feed the pass (Seed D, REQ-016)

The pkg/gate reconciliation pass is inert until the shipped `backstop gate`
construction site turns it on and feeds it — exactly like baseline, which has TWO
seams, not one:

1. **pkg/gate swap (already specified):** `computeWaiverResult`, mirroring
   `computeBaselineResult`, swapped in the Run loop.
2. **Construction-site enablement + feed (this requirement):**
   - `WithWaiver(read, policy, now)` in pkg/gate sets `g.waiverEnabled = true` and
     attaches the runtime inputs, mirroring `WithBaseline` (gate.go:51-53). The
     Run-loop swap is guarded `if g.waiverEnabled && result.StepName ==
     StepWaiverResolution` (mirror of gate.go:139).
   - `cmd/backstop/gate.go` calls `WithWaiver` at the same construction site where
     it calls `gate.WithBaseline` (~:114), constructing: a `LineReader` over the
     active scope (`buildWaiverLineReader`), the current time, and the production
     `Policy` (`buildWaiverPolicy`).
   - `buildWaiverPolicy` builds the production `Policy` by **extracting the
     declared non-waivable sets from the INSTALLED pack manifests** — the CLM-027
     "declared, not hardcoded" mechanism realized in production. The shipped
     configuration places `backstop/self` rules and critical-severity secrets in
     the non-waivable set by virtue of those manifests, not a core list.

Without seam 2, every unit test and the pkg/gate e2e can be green while shipped
`backstop gate` suppresses nothing — so REQ-016's claims drive the REAL CLI
construction path (CLM-070/071) over the committed installed-pack fixture.

### Shipped step order — waiver before baseline (Seed D, REQ-017)

`baseline_comparison` captures its NewViolations from the accumulated violation
set AT ITS LOOP POSITION. In today's deferred step list baseline (`:707`)
precedes waiver (`:708`), so baseline would capture NewViolations BEFORE waiver
subtraction and an ACTIVE waiver would NOT satisfy the ratchet (REQ-013 /
CLM-055) in the real pipeline — even though synthetic-StepResult ratchet tests
pass in isolation. The shipped `cmd/backstop/gate.go` step list must therefore
order `StepWaiverResolution` BEFORE `StepBaselineComparison`, so the accumulated
set is already waiver-subtracted when baseline captures NewViolations. CLM-072
asserts the order; CLM-073 proves the ratchet interaction against the real
pipeline order, not synthetic results.

### CLI (Seed D)

`backstop waiver list` is read-only. It runs adjudication over the current scope
and prints active, expiring-soon, and unused/dangling waivers. It never writes or
inserts a token (REQ-011) — authoring a comment requires language comment-syntax,
which is baked-language knowledge that belongs to the human or runtime agent.

## Verification

Verification is defined in frontmatter: integration level, 80% coverage
threshold, targeting `pkg/waiver`, `pkg/gate`, and `cmd/backstop`. Integration
level is chosen because the subsystem is cross-package — core adjudication in
`pkg/waiver`, gate step-8 wiring and reporting in `pkg/gate`, and the shipped
construction site + CLI in `cmd/backstop` — and the load-bearing behavior is the
wiring, not any single unit. The `test_command` deliberately does NOT apply a
`-run` filter: a filter would restrict which tests execute while
`-coverprofile` still measures the WHOLE package, making the 80% threshold
meaningless (unmeetable) for `pkg/gate`. Running the full suites of the three
target packages measures coverage against all their tests, so the threshold is
meaningful for the waiver-owned code inside them. Claims are defined in
frontmatter; every requirement has at least one claim, and the REQ-006, REQ-008,
REQ-010, and REQ-013 matrices are covered cell by cell. The production wiring
(REQ-016) and ordering (REQ-017) claims are proven against the REAL
`backstop gate` construction path over the committed installed-pack fixture —
not against synthetic in-package `gate.New` StepResults — so a green unit +
pkg/gate e2e cannot coexist with a shipped `backstop gate` that suppresses
nothing.

## Sharp Edges

- **Zero-baked boundary: adjudication is a byte-scan of the SARIF-reported line,
  NOT comment parsing.** The single most common way to get this wrong is to
  tokenize the language (or run a comment lexer) to "find the comment that holds
  the waiver." That bakes language knowledge and violates the cardinal
  invariant. The design is enforceably zero-baked: `Adjudicate` is handed only a
  `LineReader func(file, line) (string, bool)` yielding RAW bytes and a set of
  findings with their reported locations — it receives NO language identifier and
  invokes NO parser. CLM-011 is an absence claim asserting exactly this. Any
  implementation that adds a language/comment parameter to the adjudication path
  is a defect, not a feature.

- **Token-location matching is deliberately narrow, and the false-match residual
  is real.** Association is only same-line-trailing OR the line immediately
  above the finding's start line — a token two lines up does NOT associate
  (CLM-036), preventing a distant waiver from silently covering unrelated
  findings. Because backstop cannot tell a comment from a string literal without
  parsing, the primary false-match defense is that a token only suppresses when
  its `rule-id` EXACTLY equals the finding's own rule-id (CLM-038); an incidental
  `@waiver:...` inside a string that names a DIFFERENT rule is inert. The
  residual — a fully-valid token naming the EXACT firing rule that happens to sit
  inside a string literal on the association line — is accepted, not solved,
  precisely because solving it would require language parsing. This is a
  conscious trade, not an oversight.

- **Non-waivable enforceability is only as strong as the engine's
  suppression-reporting — do not over-promise.** The non-waivable guarantee is
  airtight for semgrep-based `backstop/self` findings because semgrep EMITS the
  finding in SARIF, so backstop sees the `@waiver` attempt and can raise the gate
  error. It does NOT extend to engine-native allows: a golangci `//nolint` or a
  `#gitleaks:allow` suppresses inside the engine and is invisible to backstop, so
  backstop cannot police it. Per DD-12 this is out of scope by design — a visible
  `gitleaks:allow` is an explicit, source-visible human choice to expose,
  categorically different from silent debt. The spec claims cover the enforceable
  channel (backstop's own `@waiver`); it must not claim coverage it cannot
  deliver. Relatedly, SURFACING engine-native visible suppressions (e.g.
  semgrep's SARIF `suppressions`) on a separate gate line is deliberately NOT a
  v1 requirement: it is a Notes/Ideas follow-on per bundle DD-6, and the current
  `Finding` type has no SARIF-`suppressions` ingestion, so declaring it in v1
  would be undeclared plumbing inconsistent with DD-6. REQ-012 v1 is exactly the
  distinct pass-with-waivers state + always-on summary + inline actionable subset.

- **Grammar enforcement: a malformed `@waiver` is itself a gate finding, never a
  silent no-op.** If a bad reason-code, missing/invalid expiry, or mangled
  structure were simply ignored, an author could believe they had waived a
  finding while the token did nothing — silent non-suppression is exactly the
  failure mode the subsystem exists to kill. REQ-007 / CLM-029..032 make every
  malformation a `Diagnostic{Kind: "malformed"}` gate finding.

- **Expired suppression must re-fire immediately, not carry a grace after
  expiry.** The grace period is the loud pre-expiry WARNING (CLM-021). At the
  moment of expiry the shield lifts and the finding re-fires under normal
  enforcement (CLM-020). An implementation that keeps suppressing for a window
  AFTER expiry recreates the silent-debt problem.

- **Target the LIVE `pack_engines` dimension, not the dead `code_check` one — or
  suppress nothing.** `StepCodeCheckScopedFunc` is GONE; pack code-rule findings
  (including the flagship non-waivable `backstop/self`) are emitted under
  `StepName: "pack_engines"`. If the reconciliation pass collects findings from a
  `code_check` dimension, the waivable surface is EMPTY — vacuous green — AND the
  `backstop/self`-gate-ERROR path (CLM-025) never fires. The waivable surface is
  exactly `pack_engines` + `test_substantiveness`; feed the pass only those.

- **The waivable-surface boundary is load-bearing and easy to over-apply.** It is
  tempting to let `@waiver` suppress anything the gate reports. It must not:
  `artifact_status_drift`, `contract_signature`, `test_verification`, and
  `artifact_validation` are structural dimensions with their own accountable
  lifecycles (retire / resolved-by / obsoleted) and are NOT waivable
  (CLM-042..044, CLM-065). Coverage is file-level with no single source location,
  so it uses the first-line annotation convention, not a per-line source token
  (CLM-045). The classification is exhaustive over the live dimensions so an
  implementer cannot accidentally mis-scope `test_substantiveness`.

- **Suppression is a reconciliation pass, not a `StepFunc` return.** A `StepFunc`
  (`func(ctx) StepResult`) returns only its OWN result and cannot mutate
  preceding steps' violations — but suppression must DELETE waived findings from
  the already-accumulated `pack_engines` / `test_substantiveness` violations.
  Model it on `computeBaselineResult` (the existing baseline reconciliation pass
  that sees all accumulated results): a `computeWaiverResult` invoked in the Run
  loop that mutates the accumulated results. CLM-067 guards that the subtraction
  actually happens — a pass that computes a waiver result but never removes the
  finding from `pack_engines` suppresses nothing and is a silent no-op.

- **Baseline generation and waiver authoring are distinct operations.** The
  ISSUE-050 ratchet is the bridge that converts silent baseline-grandfathered
  debt into loud, expiring waivers at the moment a file is touched — but the
  baseline snapshot (a machine operation) must never author a `@waiver` (a human
  decision). CLM-058 guards this: baseline generation writes no tokens.

- **Core is read-only; token authoring would leak language knowledge.** Writing a
  `@waiver` requires knowing the target language's comment syntax. That is baked
  knowledge core must never hold, so `backstop waiver list` and every core path
  are read-only (CLM-049 absence). Authoring belongs to the human or the runtime
  agent, which may legitimately hold language knowledge.

- **The pkg/gate pass is inert until the shipped construction site enables AND
  feeds it — two seams, not one.** Baseline has both a `computeBaselineResult`
  swap AND a `WithBaseline` construction call; a spec that defines only the swap
  ships a dark gate. `WithWaiver` must set `g.waiverEnabled` and the Run loop must
  guard the swap on it, AND `cmd/backstop/gate.go` must call `WithWaiver` with a
  real `LineReader`, `now`, and the production `Policy`. If the `Policy` is
  anything other than the sets EXTRACTED from the installed pack manifests
  (CLM-069), the "declared, not hardcoded" property (CLM-027) is true in the type
  but false in production, and `backstop/self` non-waivable enforcement never
  fires through the shipped path. The REQ-016 tests drive the real CLI
  construction path over the committed installed-pack fixture precisely so this
  cannot pass vacuously.

- **Waiver must run BEFORE baseline, or an active waiver silently fails the
  ratchet.** `baseline_comparison` reads the accumulated violation set at its loop
  position. If waiver resolution is ordered after baseline (the current
  deferred-stub order), an active-waived finding is still present when baseline
  captures NewViolations, so it counts as a new violation against the ISSUE-050
  ratchet — defeating REQ-013/CLM-055 in the real pipeline even while synthetic
  ratchet tests pass. Order `StepWaiverResolution` ahead of
  `StepBaselineComparison` in the shipped step list (CLM-072) and prove the
  interaction against the real order (CLM-073).

## Review Questions

1. Does `Adjudicate` receive ONLY findings, a raw-bytes `LineReader`, a `Policy`,
   and `now` — with no language identifier, no file-extension switch, and no
   comment lexer anywhere on the suppression path? (Zero-baked boundary.)

2. Is location association strictly same-line-trailing OR the single line
   immediately above the finding's start line, with a token two-or-more lines
   above proven inert? (REQ-008 / CLM-036.)

3. Does suppression require an EXACT rule-id match, so that a `@waiver` naming a
   different rule — or a stale rule-id after a pack rename — surfaces as
   unused/dangling rather than silently waiving something? (REQ-008/REQ-009.)

4. Is the non-waivable decision sourced entirely from the supplied `Policy`, with
   no hardcoded list of `backstop/self` rules or secret rule-ids in core? And is
   a `@waiver` on a Policy-declared non-waivable rule raised as a gate ERROR,
   not swallowed? (REQ-006.)

5. Does an expired waiver re-fire its finding at the instant of expiry (with the
   grace delivered earlier as the pre-expiry warning), rather than continuing to
   suppress?

6. Is the waivable surface restricted to the LIVE code-located dimensions
   (`pack_engines`, `test_substantiveness`), with `artifact_status_drift`,
   `contract_signature`, `test_verification`, and `artifact_validation` provably
   non-waivable and coverage handled by the first-line annotation convention?

7. Does a pass that depends on active waivers render as the distinct
   `PASS · N waivers` state (never as a clean pass), with an always-on summary and
   inline actionable subset (expiring/unused)?

8. Does the ratchet treat ACTIVE / EXPIRED / UNUSED waivers as satisfy /
   not-satisfy / not-satisfy respectively, and does baseline generation author no
   tokens?

9. Does `cmd/backstop/gate.go` actually CALL `WithWaiver` (not just define the
   pkg/gate swap), constructing a real `LineReader`, `now`, and a production
   `Policy` EXTRACTED from the installed pack manifests — and does a test drive the
   real `backstop gate` construction path (not in-package `gate.New`) to prove a
   `@waiver` suppresses and a `backstop/self` waiver errors over the installed-pack
   fixture? (REQ-016.)

10. Is `StepWaiverResolution` ordered BEFORE `StepBaselineComparison` in the
    shipped step list, and is the ratchet interaction proven against that real
    order so an active waiver is subtracted before baseline captures
    NewViolations? (REQ-017.)

## References

- BUNDLE-013 (waiver-subsystem): source bundle; settled model in DD-1..DD-12,
  requirements REQ-001..015, four spec seeds.
- ISSUE-050 (strict file-level ratchet): the driver — touching a file forces
  every finding to be fixed or waived; this subsystem is the accountable
  "or waived" valve (REQ-013).
- SPEC-010 REQ-006 (gate step 8, waiver resolution): the reserved, stubbed step
  (`pkg/gate/step_deferred.go` `StepWaiverResolutionScopedFunc`) this spec
  replaces with real adjudication.
- SPEC-019 (baseline): reserved the waiver-aware baseline integration seam this
  spec fills; baseline generation and waiver authoring stay distinct (REQ-013).
- DIR-003 / BUNDLE-007 (baseline): the CI-generated baseline whose silent
  grandfathering this subsystem replaces with accountable waivers.
- ISSUE-046 (exported `NormalizePath`): the reported-location path substrate the
  line-scan adjudication relies on.
- `pkg/gate/gate.go` `WithBaseline` / `computeBaselineResult` / Run-loop swap: the
  TWO-seam construction precedent the waiver wiring mirrors (REQ-016).
- `cmd/backstop/gate.go` construction site (`gate.WithBaseline` call ~:114) and
  assembled step list (baseline ~:707 precedes waiver ~:708): the production
  surface REQ-016 extends and REQ-017 reorders.
