Feature: Pack Gate Enforcement
  As a developer using backstop
  I want to install an enforcement pack and have the gate enforce its rules
  So that my code meets the pack's standards without manual configuration

  Background:
    Given the backstop CLI is built and available at bin/backstop
    And a test project exists with Go source files
    And a valid enforcement pack exists in a git-accessible location

  @UJ-001
  Scenario: Install pack and gate enforces its rules
    Given I am in the test project directory
    And no packs are installed
    When I run `backstop pack add <pack-url>`
    Then the command exits with code 0
    And .backstop/packs/ contains the installed pack
    And backstop.yml lists the pack with its version
    And backstop.lock contains the pack with a content hash
    When I run `backstop gate`
    Then the gate output includes violations with the pack's namespaced rule IDs
    And the violations reference the pack's rules (pack-name/rule-id format)

  @UJ-002
  Scenario: Gate verifies lock integrity
    Given I have a pack installed with a valid lockfile
    When I modify a file inside .backstop/packs/<pack-name>/
    And I run `backstop gate`
    Then the gate fails with a lock verification error
    And the error identifies the tampered pack by name
    And the error shows the expected vs actual content hash

  @UJ-003
  Scenario: Gate fails on missing pack
    Given backstop.yml lists a pack
    But .backstop/packs/ does not contain it
    When I run `backstop gate`
    Then the gate fails with a missing pack error
    And the error identifies the missing pack

  @UJ-004
  Scenario: Remove pack and gate no longer enforces its rules
    Given I have a pack installed and the gate enforces its rules
    When I run `backstop pack remove <pack-name>`
    Then the command exits with code 0
    And .backstop/packs/ no longer contains the pack
    And backstop.yml no longer lists the pack
    And backstop.lock no longer contains the pack
    When I run `backstop gate`
    Then the gate output does not include the pack's rule IDs

  @UJ-005
  Scenario: Pack tool_config is applied during gate
    Given I have a pack installed that declares tool_config for golangci-lint
    When I run `backstop gate`
    Then the code check step uses the pack's tool configuration
    And violations from the pack's configured linter rules appear in gate output

  @UJ-006
  Scenario: Multiple packs compose without conflict
    Given I have two enforcement packs installed with non-overlapping rules
    When I run `backstop gate`
    Then violations from both packs appear in gate output
    And each violation's rule ID is namespaced to its source pack
