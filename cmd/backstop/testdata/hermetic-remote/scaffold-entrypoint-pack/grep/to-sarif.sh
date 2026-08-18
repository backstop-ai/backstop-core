#!/bin/sh
# grep-output -> SARIF converter for the DD-7 evidence fixture pack (SPEC-069).
#
# Reads `grep -rn -H -I -e <pattern> <targets>` output on stdin -- lines of the form
# "<file>:<line>:<matched text>" -- and emits SARIF 2.1.0 on stdout. Each match
# becomes one SARIF result with a physicalLocation whose artifactLocation.uri is the
# file and whose region.startLine is the 1-indexed line grep reported.
#
# It is derived from packs/contracts/grep/to-sarif.sh, with two deliberate
# differences: the ruleId is THIS pack's rule id, and the JSON is assembled with awk
# alone so the fixture depends on nothing beyond the POSIX text tools the sandboxed
# convert step already has. No marker literal appears here -- the pattern is the
# pack manifest's DATA and this script never needs to know it.
#
# Zero matches is the ordinary case and must produce a well-formed SARIF document
# with an EMPTY results array, never empty stdout: an empty payload would fail the
# parse and read as a broken pack rather than as a clean run.
#
# THE INPUT CONTRACT (ISSUE-166): every stdin line MUST carry the filename, which is
# why this pack's grep command declares -H and -I.
#   -H  GNU grep OMITS the filename when the target is exactly ONE explicit file,
#       emitting a 2-field "<line>:<text>". The old guard dropped every such line
#       SILENTLY at exit 0 -- a single-file probe then reported zero matches on
#       Linux whatever the file actually contained.
#   -I  grep otherwise writes a NON-match "Binary file <path> matches" line to
#       STDOUT, interleaved with real matches, whenever the scope holds a binary
#       file. That line is not a finding and has no "<file>:<line>:" shape.
#
# UNPARSEABLE INPUT IS REFUSED, NEVER DROPPED: a stderr diagnostic naming the line
# plus a NON-ZERO exit. It is deliberately not parsed heuristically -- "6:42: text"
# is ambiguous between a filename-less match and a file named "6", so a "robust"
# parser fabricates a finding at a file that does not exist. Do NOT "simplify" the
# refusal back into a drop.

# Buffer stdin so the input is VALIDATED before any of it is converted: a refusal
# must emit no partial SARIF. The only mechanisms used are the shell's own >&2 and
# awk's stdout -- no /dev/stderr or /dev/null open, which the convert sandbox denies,
# and no jq dependency (this fixture is deliberately POSIX-tools-only).
input=$(cat)

# VALIDATE. Print the first line that is not "<file>:<digits>:" and stop. Wholly
# empty lines are skipped: grep never emits one as a match, and the trailing newline
# of a well-formed stream would otherwise read as a violation.
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

printf '%s\n' "$input" | awk '
  BEGIN {
    printf "{\"version\":\"2.1.0\",\"runs\":[{\"results\":[";
    first = 1;
  }
  # Only the first two colons are structural ("file:line:"); the matched text may
  # itself contain colons, so the text is recovered by stripping that prefix rather
  # than by re-joining split fields.
  /^[^:]*:[0-9]+:/ {
    file = $0; sub(/:[0-9]+:.*$/, "", file);
    rest = $0; sub(/^[^:]*:/, "", rest);
    line = rest; sub(/:.*$/, "", line);
    text = rest; sub(/^[0-9]+:/, "", text);
    gsub(/\\/, "\\\\", text); gsub(/"/, "\\\"", text); gsub(/\t/, " ", text);
    gsub(/\\/, "\\\\", file); gsub(/"/, "\\\"", file);
    if (!first) printf ",";
    first = 0;
    printf "{\"ruleId\":\"scaffolded-source-present\",\"level\":\"error\",";
    printf "\"message\":{\"text\":\"scaffolded source marker present: %s\"},", text;
    printf "\"locations\":[{\"physicalLocation\":{\"artifactLocation\":{\"uri\":\"%s\"},", file;
    printf "\"region\":{\"startLine\":%s}}}]}", line;
  }
  END {
    printf "]}]}\n";
  }
'
