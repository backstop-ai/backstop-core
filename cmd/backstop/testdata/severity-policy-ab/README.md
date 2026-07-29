# severity-policy-ab — the ISSUE-105 A/B probe, as a tree

This fixture proves that a pack finding declaring SARIF severity `warning` is
non-blocking for a consumer that has written no per-dimension enforcement
config — the population ISSUE-105 measured, where a step's raw violation count
decided the verdict and the declared severity was never read.

`backstop.yml` deliberately declares NO `enforcement:` block. That absence IS
the fixture: the severity contract belongs to the finding, not to adopter
configuration, so the gate must reach the same verdict with and without a
policy entry. The e2e writes the entry in at runtime for the B run, and that one
block is the entire delta between the two runs.

The SARIF bytes in `descriptor-warning.sarif` and `descriptor-error.sarif` are
verbatim copies of `cmd/backstop/testdata/semgrep/fixtures/descriptor-{warning,
error}.sarif` — real captured semgrep output carrying the severity on the rule
descriptor with no result-level `level`. That directory's `PROVENANCE.md` is the
provenance of record for these copies, and the e2e asserts the copies are
byte-identical to it so they cannot drift into fabrication.
