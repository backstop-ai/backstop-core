Feature: Extend Backstop

  @UJ-001
  Scenario: CAP-012/@UJ-001 Decide whether a concern belongs in a pack and start authoring
    Given a visitor is at /extend/#pack-or-not
    When they follow the rendered JLINK-019 next-action link
    Then they arrive at /reference/#pack-artifact and can decide whether a concern belongs in a pack

  @UJ-002
  Scenario: CAP-012/@UJ-002 Continue from pack authoring to the contribution path
    Given a visitor is at /extend/#author-a-pack
    When they follow the rendered JLINK-020 next-action link
    Then they arrive at /contributing/#contribution-paths and can continue from pack authoring to the contribution path
