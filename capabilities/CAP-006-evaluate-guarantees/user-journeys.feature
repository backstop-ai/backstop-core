Feature: Evaluate Guarantees

  @UJ-001
  Scenario: CAP-006/@UJ-001 Distinguish a shipped mechanism from a guarantee
    Given a visitor is at /model/#gates-and-policy
    When they follow the rendered JLINK-006 next-action link
    Then they arrive at /status/#supported-and-limited and can distinguish a shipped mechanism from a guarantee

  @UJ-002
  Scenario: CAP-006/@UJ-002 Compare every public boundary state and its implication
    Given a visitor is at /status/#boundary-states
    When they follow the rendered JLINK-007 next-action link
    Then they arrive at /model/#ownership-boundaries and can compare every public boundary state
