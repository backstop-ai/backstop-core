#!/bin/sh
# Deterministic self-targeting fake FINDINGS engine for the SPEC-048 real-runner
# installed-pack e2e (REQ-003). It faithfully models the self-targeting-toolchain
# I/O contract that made both dispatch defects invisible:
#
#   (i)   SELF-TARGETS: it scans its OWN working directory (the run's cwd =
#         projectRoot) for the seeded marker in src/app.ts. It never expects — and
#         MUST NOT depend on — a bolted-on projectRoot path argument.
#   (ii)  ARG-SENSITIVE (the DEFECT-1 trap): if ANY argument is appended (a stray
#         scan target), it scans NOTHING and emits NO finding — exactly as
#         `tsc --noEmit <dir>` silently typechecks nothing. So a bolted-on
#         projectRoot would vacuous-green even the SEEDED variant.
#   (iii) writes its real machine-readable findings (SARIF) to the declared
#         stdout_artifact FILE (findings.sarif in cwd), and prints only human
#         summary NOISE to stdout (the DEFECT-2 trap: a stdout-fed convert sees no
#         findings).
#
# It also records its argument count to argc.txt so the e2e can PROVE, on the real
# dispatch path, that no scan target was appended (self-target).

# Record argc for the e2e's self-target assertion (DEFECT-1 proof).
printf '%s' "$#" > argc.txt

artifact="findings.sarif"

emit_empty() {
  cat > "$artifact" <<'SARIF'
{"version":"2.1.0","$schema":"https://json.schemastore.org/sarif-2.1.0.json","runs":[{"tool":{"driver":{"name":"fake-engine"}},"results":[]}]}
SARIF
}

emit_finding() {
  cat > "$artifact" <<'SARIF'
{"version":"2.1.0","$schema":"https://json.schemastore.org/sarif-2.1.0.json","runs":[{"tool":{"driver":{"name":"fake-engine"}},"results":[{"ruleId":"fake-seeded-defect","level":"error","message":{"text":"seeded defect: SEEDED_DEFECT marker present in src/app.ts"},"locations":[{"physicalLocation":{"artifactLocation":{"uri":"src/app.ts"},"region":{"startLine":1}}}]}]}]}
SARIF
}

# Human-summary NOISE to stdout — deliberately NOT SARIF (no results array, no
# <result>): the misleading summary a stdout-fed convert would read as zero
# findings and green a failing run (DEFECT-2).
echo "fake-engine: scan complete; machine-readable findings written to findings.sarif"

# DEFECT-1 trap: a stray path arg suppresses the finding (scans nothing).
if [ "$#" -ne 0 ]; then
  emit_empty
  exit 0
fi

# Self-target: scan the cwd's own source for the seeded marker.
if [ -f src/app.ts ] && grep -q 'SEEDED_DEFECT' src/app.ts; then
  emit_finding
  exit 1
fi

emit_empty
exit 0
