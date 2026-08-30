Feature: Browse the Pack Ecosystem

  @UJ-001
  Scenario: CAP-011/@UJ-001 Browse the generated installed-pack catalog
    Given a visitor is at /packs/#installed-pack-catalog
    When they follow the rendered JLINK-017 next-action link
    Then they arrive at /reference/#pack-commands and can browse the generated installed-pack catalog

  @UJ-002
  Scenario: CAP-011/@UJ-002 Determine which pack addresses a problem and inspect its status
    Given a visitor is at /packs/#choose-a-pack
    When they follow the rendered JLINK-018 next-action link
    Then they arrive at /status/#pack-direction and can determine which pack addresses a problem
