Feature: Evaluate Fit

  @UJ-001
  Scenario: CAP-005/@UJ-001 Confirm fit and continue to adoption
    Given a visitor is at /use-cases/#choose-use-case
    When they follow the rendered JLINK-003 next-action link to /evaluate/#fit-decision
    And they follow the rendered JLINK-004 next-action link
    Then they arrive at /adopt/#install and can continue from a confirmed fit to adoption

  @UJ-002
  Scenario: CAP-005/@UJ-002 Confirm no-fit and continue to boundary guidance
    Given a visitor is at /model/#product-category
    When they follow the rendered JLINK-005 next-action link
    Then they arrive at /status/#adjacent-guidance and can continue from a confirmed no-fit
