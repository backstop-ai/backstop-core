#!/bin/sh
# Case (a): the declared entrypoint STARTS and exits ZERO.
#
# It echoes its argv so the argv-safety claim (CLM-108) can assert that shell
# metacharacters reached the process as LITERAL arguments -- unexpanded, unsplit, and
# never interpreted by a shell. The echo is on stdout on purpose; the stderr-only
# sibling is the one whose stdout must stay empty.
echo "argv: $*"
exit 0
