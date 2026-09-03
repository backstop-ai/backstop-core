Feature: Adopt Backstop

  @UJ-002
  Scenario: CAP-009/@UJ-002 Verify the configured repository's enforcement path
    Given a visitor is at /adopt/#verify-enforcement
    When they follow the rendered JLINK-013 next-action link to /model/#enforcement-loop
    And they follow the rendered JLINK-014 next-action link
    Then they arrive at /reference/#gate and can verify the configured repository's enforcement path
