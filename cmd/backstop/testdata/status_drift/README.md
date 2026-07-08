# status_drift wiring/e2e fixtures (ISSUE-042 TASK-007)

The wiring tests in cmd/backstop/status_drift_gate_test.go build each scenario
in an isolated t.TempDir() at runtime (the same pattern as gate_buildsteps_test.go)
rather than as static files here — one temp project per exit-code scenario keeps a
blocking closed-issue fixture from confounding a delivered-but-open exit-0 assertion.

Each temp project carries:
- a backstop.yml + an installed minimal `backstop/drift-toolchain` pack whose ONLY
  job is to declare classification.test globs (**/*_test.go) + test_name_patterns
  (func Test...), so the full-sweep existence check (collectTestFuncNames /
  ResolveMandatedTestPaths) can discover the fixture's present tests. It declares a
  single no-op grep rule solely so the manifest's `content` block is non-empty; the
  drift step never dispatches engines.
- issues/ specs/ (etc.) fixtures with known status/test states.
- for present-test scenarios, a src/*_test.go carrying the mandated func name so
  existence resolves PRESENT.

Scenario matrix (one temp project each):
- WiredIntoBuildGateSteps  — no-pack project; the step appears in buildGateSteps.
- FullSweep                — closed issue + absent test, file OUT of the diff scope.
- DeliveredOpen            — open issue + PRESENT test -> warning, exit 0.
- SuccessTerminalAbsent    — closed issue + absent test -> blocks (exit non-zero).
- PresentButFailing        — closed issue + PRESENT test -> drift emits NOTHING
                             (a failing-but-present test is pack_engines' job).
- RetiredTerminal          — replaced issue -> no violation.
- PolicyEntry              — the backstop.yml block+new-code entry parses and
                             grandfathers a baselined finding while a new one blocks.
