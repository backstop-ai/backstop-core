#!/bin/sh
# Real grep-output -> SARIF converter shipped by the traceability pack (SPEC-038
# REQ-005/CLM-019). Reads `grep -rn -H -I -e <pattern> <targets>` output on stdin —
# lines of the form "<file>:<line>:<matched text>" — and emits SARIF 2.1.0 on
# stdout. Each grep match line becomes one SARIF result with a physicalLocation
# (artifactLocation.uri = file, region.startLine = the 1-indexed line grep
# reports). The forbidden token's presence IS the finding; the gate inverts a
# present match to an absence VIOLATION (the polarity lives gate-side, not here).
# A stderr banner exercises clean-stdout capture. Backstop ships no grep knowledge
# — this script is pack DATA the pack author wrote.
#
# THE INPUT CONTRACT (ISSUE-166): every stdin line MUST carry the filename, which
# is why the pack's grep command declares -H and -I.
#   -H  GNU grep OMITS the filename when the target is exactly ONE explicit file,
#       emitting a 2-field "<line>:<text>". The old `NF >= 3` guard dropped every
#       such line SILENTLY at exit 0, so a single-file absence probe reported zero
#       matches on Linux whatever the file actually contained — a vacuous green in
#       a security-relevant check.
#   -I  grep otherwise writes a NON-match "Binary file <path> matches" line to
#       STDOUT, interleaved with real matches, whenever the scope holds a binary
#       file. That line has no "<file>:<line>:" shape and is not a finding.
#
# UNPARSEABLE INPUT IS REFUSED, NEVER DROPPED. A line that is not
# "<file>:<digits>:<text>" produces a stderr diagnostic naming it and a NON-ZERO
# exit. It is deliberately NOT parsed heuristically: "6:42: text" is genuinely
# ambiguous between a filename-less match and a file named "6", so a "robust"
# 2-field parser FABRICATES a finding at a file that does not exist — trading a
# silent false negative for a silent false positive in the same probe.
#
# ZERO MATCHES REMAINS A CLEAN PASS: empty stdin exits 0 with a well-formed SARIF
# document carrying an EMPTY results array. That is the ORDINARY case for an
# absence probe, and the refusal above is per-LINE, only for lines that exist.
#
# Do NOT "simplify" the refusal back into a drop.
echo "grep to-sarif: transforming matches" >&2

# Buffer stdin so the input can be VALIDATED before any of it is converted: a
# refusal must emit no partial SARIF. The only mechanisms used here are the shell's
# own >&2 and awk's stdout — no /dev/stderr or /dev/null open, which the convert
# sandbox denies.
input=$(cat)

# VALIDATE. Print the first line that is not "<file>:<digits>:" and stop. Wholly
# empty lines are skipped: grep never emits one as a match, and the trailing
# newline of a well-formed stream would otherwise be read as a violation.
bad=$(printf '%s\n' "$input" | awk '
  /^$/ { next }
  /^[^:]*:[0-9]+:/ { next }
  { print; exit }
')
if [ -n "$bad" ]; then
  echo "grep to-sarif: REFUSING unparseable grep output line: $bad" >&2
  echo "grep to-sarif: expected <file>:<line>:<text> on every input line." >&2
  echo "grep to-sarif: the pack's grep command must pass -H -I (-H: grep omits the filename for a single explicit file target; -I: grep writes a non-match 'Binary file <path> matches' line to stdout)." >&2
  exit 1
fi

# Emit one JSON object per grep match line (newline-delimited), then `jq -s`
# slurps the stream into an array and wraps it as SARIF. Only the first two colons
# are structural ("file:line:"); the match text may itself contain colons, so the
# text is recovered by stripping that prefix rather than by re-joining fields.
printf '%s\n' "$input" | awk -F: '
  /^[^:]*:[0-9]+:/ {
    file=$1; line=$2;
    text=$0;
    sub(/^[^:]*:[^:]*:/, "", text);
    gsub(/\\/, "\\\\", text); gsub(/"/, "\\\"", text); gsub(/\t/, " ", text);
    printf "{\"file\":\"%s\",\"line\":%s,\"text\":\"%s\"}\n", file, line, text;
  }
' | jq -s '{
  version: "2.1.0",
  runs: [
    {
      results: [ .[] | {
        ruleId: "contract-absence",
        level: "error",
        message: { text: ("forbidden symbol present: " + .text) },
        locations: [ {
          physicalLocation: {
            artifactLocation: { uri: .file },
            region: { startLine: (.line | tonumber) }
          }
        } ]
      } ]
    }
  ]
}'
