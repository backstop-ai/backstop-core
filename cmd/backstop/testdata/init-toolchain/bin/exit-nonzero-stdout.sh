#!/bin/sh
# Case (c) with a STDOUT diagnostic: the entrypoint STARTS and exits non-zero, and
# says why on stdout. It is the ordinary shape, and it is precisely the shape that
# CANNOT distinguish Run from RunStdout -- which is why the stderr-only sibling
# exists.
echo "entrypoint failed: 3 checks did not pass"
exit 3
