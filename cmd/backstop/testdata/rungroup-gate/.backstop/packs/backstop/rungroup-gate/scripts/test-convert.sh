#!/bin/sh
# ISSUE-068 fixture TEST convert: reads the shared suite payload on stdin and emits
# an EMPTY SARIF result set (zero findings) so the pack_engines step is a clean pass.
cat >/dev/null
printf '{"version":"2.1.0","runs":[{"results":[]}]}\n'
