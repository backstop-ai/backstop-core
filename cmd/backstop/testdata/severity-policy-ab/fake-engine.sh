#!/bin/sh
# Deterministic POSIX fake FINDINGS engine for the ISSUE-105 severity A/B probe.
# It COPIES one of two CAPTURED real-semgrep SARIF files to the declared
# stdout_artifact (findings.sarif). SEVERITY_AB_LEVEL selects which:
#   warning -> descriptor-warning.sarif  (declared NON-BLOCKING by contract)
#   error   -> descriptor-error.sarif    (declared blocking)
#   (unset) -> warning
# The bytes are UNMODIFIED semgrep output — severity on the RULE DESCRIPTOR, no
# result-level `level` — copied verbatim from
# cmd/backstop/testdata/semgrep/fixtures/ (see that directory's PROVENANCE.md).
# It must never echo SARIF it constructs inline: a fabricated fixture would not
# exercise the ISSUE-104 descriptor-resolution hop this probe rides through.
artifact="findings.sarif"
level="${SEVERITY_AB_LEVEL:-warning}"

case "$level" in
  error)   source_sarif="descriptor-error.sarif" ;;
  warning) source_sarif="descriptor-warning.sarif" ;;
  *)
    echo "severity-ab-fake: unknown SEVERITY_AB_LEVEL=$level (want warning|error)" >&2
    exit 2
    ;;
esac

cat "$source_sarif" > "$artifact"

echo "severity-ab-fake: findings written to $artifact (level=$level)"
exit 0
