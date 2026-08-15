#!/bin/sh
# grep-output -> SARIF converter for the acceptance fixture pack (SPEC-069).
#
# Reads `grep -rn -e <pattern> <targets>` output on stdin -- lines of the form
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
awk '
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
    printf "{\"ruleId\":\"acceptance-marker\",\"level\":\"error\",";
    printf "\"message\":{\"text\":\"acceptance marker present: %s\"},", text;
    printf "\"locations\":[{\"physicalLocation\":{\"artifactLocation\":{\"uri\":\"%s\"},", file;
    printf "\"region\":{\"startLine\":%s}}}]}", line;
  }
  END {
    printf "]}]}\n";
  }
'
