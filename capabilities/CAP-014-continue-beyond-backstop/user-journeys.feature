Feature: Continue Beyond Backstop

  @UJ-001
  Scenario: CAP-014/@UJ-001 Follow adjacent guidance beyond an intentional boundary
    Given a visitor is at /status/#adjacent-guidance inside BOUNDARY-005
    When they follow the one rendered JLINK-024 continuation that is also the BOUNDARY-005 adjacent-guidance link
    Then they arrive at /contributing/#external-ownership and can continue beyond an intentional Backstop boundary

  @UJ-002
  Scenario: CAP-014/@UJ-002 Confirm that adjacent guidance is not a Backstop guarantee
    Given a visitor is at /evaluate/#compatibility-limits
    When they follow the rendered JLINK-009 next-action link
    Then they arrive at /status/#adjacent-guidance and can confirm that adjacent guidance is not a Backstop guarantee
