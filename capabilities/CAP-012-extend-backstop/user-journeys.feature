Feature: Extend Backstop

  @UJ-002
  Scenario: CAP-012/@UJ-002 Continue from pack authoring to the contribution path
    Given a visitor is at /pack/guide/#author-a-pack
    When they follow the rendered JLINK-020 next-action link
    Then they arrive at /contributing/#contribution-paths and can continue from pack authoring to the contribution path
