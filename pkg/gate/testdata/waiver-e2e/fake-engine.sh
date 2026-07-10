#!/bin/sh
# Deterministic POSIX fake FINDINGS engine for the SPEC-049 waiver e2e. It writes
# SARIF findings at FIXED locations in src/app.go to the declared stdout_artifact
# (findings.sarif). WAIVER_E2E_SCENARIO selects which findings are emitted so a
# single fixture yields both clean terminal states:
#   waivable  -> only waivable-defect (src/app.go:5)   [suppressed by @waiver]
#   protected -> only protected-defect (src/app.go:10) [non-waivable -> gate ERROR]
#   (unset)   -> both
# stdout is human noise; the machine findings live in the artifact FILE.
artifact="findings.sarif"
scenario="${WAIVER_E2E_SCENARIO:-both}"

waivable='{"ruleId":"waivable-defect","level":"error","message":{"text":"waivable defect at src/app.go:5"},"locations":[{"physicalLocation":{"artifactLocation":{"uri":"src/app.go"},"region":{"startLine":5}}}]}'
protected='{"ruleId":"protected-defect","level":"error","message":{"text":"protected defect at src/app.go:10"},"locations":[{"physicalLocation":{"artifactLocation":{"uri":"src/app.go"},"region":{"startLine":10}}}]}'

case "$scenario" in
  waivable)  results="$waivable" ;;
  protected) results="$protected" ;;
  *)         results="$waivable,$protected" ;;
esac

cat > "$artifact" <<EOF
{"version":"2.1.0","\$schema":"https://json.schemastore.org/sarif-2.1.0.json","runs":[{"tool":{"driver":{"name":"waiver-e2e-fake"}},"results":[$results]}]}
EOF

echo "waiver-e2e-fake: findings written to $artifact (scenario=$scenario)"
exit 0
