Feature: Understand the System

  @UJ-001
  Scenario: CAP-008/@UJ-001 Follow the artifact-to-plan-to-gate operating model
    Given a visitor is at /model/#operating-model
    When they follow the rendered JLINK-010 next-action link
    Then they arrive at /reference/#artifact-schema-catalog and can follow the artifact-to-plan-to-gate operating model

  @UJ-002
  Scenario: CAP-008/@UJ-002 Inspect architecture and ownership boundaries
    Given a visitor is at /model/#ownership-boundaries
    When they follow the rendered JLINK-011 next-action link
    Then they arrive at /status/#project-boundaries and can inspect architecture and ownership boundaries
