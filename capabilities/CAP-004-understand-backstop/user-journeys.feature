Feature: Understand Backstop

  @UJ-001
  Scenario: CAP-004/@UJ-001 Recognize the failure class and why Backstop exists
    Given a visitor is at /#define-work
    When they follow the rendered JLINK-001 next-action link
    Then they arrive at /evaluate/#failure-fit and can recognize the failure class Backstop exists to prevent

  @UJ-002
  Scenario: CAP-004/@UJ-002 Distinguish what Backstop is from what it is not
    Given a visitor is at /evaluate/#what-backstop-is
    When they follow the rendered JLINK-002 next-action link
    Then they arrive at /model/#operating-model and can distinguish what Backstop is from what it is not
