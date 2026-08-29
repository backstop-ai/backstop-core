Feature: Evaluate Compatibility

  @UJ-001
  Scenario: CAP-007/@UJ-001 Determine whether a named harness, model, or toolchain can operate Backstop
    Given a visitor is at /evaluate/#compatibility
    When they follow the rendered JLINK-008 next-action link
    Then they arrive at /reference/#compatibility and can determine whether a named harness, model, or toolchain can operate Backstop

  @UJ-002
  Scenario: CAP-007/@UJ-002 Determine which lifecycle guarantees that compatibility does not preserve
    Given a visitor is at /evaluate/#compatibility-limits
    When they follow the rendered JLINK-009 next-action link
    Then they arrive at /status/#adjacent-guidance and can determine which lifecycle guarantees that compatibility does not preserve
