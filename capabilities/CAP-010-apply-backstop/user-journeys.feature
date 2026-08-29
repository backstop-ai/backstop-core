Feature: Apply Backstop

  @UJ-001
  Scenario: CAP-010/@UJ-001 Select a concrete use case and its adoption action
    Given a visitor is at /use-cases/#choose-use-case
    When they follow the rendered JLINK-015 next-action link
    Then they arrive at /adopt/#adoption-paths and can select a concrete use case and its adoption action

  @UJ-002
  Scenario: CAP-010/@UJ-002 Connect a use case to an applicable pack
    Given a visitor is at /use-cases/#pack-backed-use-cases
    When they follow the rendered JLINK-016 next-action link
    Then they arrive at /packs/#choose-a-pack and can connect a use case to an applicable pack
