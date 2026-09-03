Feature: Inspect the Evidence

  @UJ-001
  Scenario: CAP-013/@UJ-001 Trace an evaluation claim to its durable source
    Given a visitor is at /model/#provenance-and-verification
    When they follow the rendered JLINK-021 next-action link
    Then they arrive at /reference/#source-traceability and can trace an evaluation claim to its durable source

  @UJ-002
  Scenario: CAP-013/@UJ-002 Trace all generated product truth to authoritative sources
    Given a visitor is at /packs/#installed-pack-catalog
    When they follow the rendered JLINK-022 next-action link to /reference/#cli-command-catalog
    And they follow the rendered JLINK-023 next-action link
    Then they arrive at /status/#release-history and can trace generated product truth to authoritative sources
