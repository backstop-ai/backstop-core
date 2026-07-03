#!/bin/sh
# typescript-toolchain build convert script (SPEC-040 fixture). Reads raw tool
# stdout on stdin and emits an empty (finding-free) SARIF document on stdout —
# a fixture stub proving the convert pipe is exercised through the pack, not the
# binary. A converter banner on stderr exercises the clean-stdout capture.
echo "typescript-toolchain build-to-sarif: normalizing output" >&2
printf '{"version":"2.1.0","runs":[{"results":[]}]}\n'
