#!/usr/bin/env bash
# Re-capture the committed semgrep SARIF fixtures from real tool output.
#
# The captured bytes are NEVER hand-edited. Provenance (tool version, command,
# date, sha256) lives in PROVENANCE.md beside the fixtures, never inside them —
# that is what keeps "captured" a true statement about these files.
#
# THE cd AND THE RELATIVE PATHS ARE LOAD-BEARING. semgrep derives the emitted
# ruleId from the --config path, so an ABSOLUTE config bakes the capturing
# machine's home directory into the committed bytes and makes the fixture
# unportable. Run from this directory with relative paths for both the config
# and the target.
set -euo pipefail
cd "$(dirname "$0")"

semgrep --config capture/rule-warning.yml --sarif capture/sample.go > descriptor-warning.sarif
semgrep --config capture/rule-error.yml   --sarif capture/sample.go > descriptor-error.sarif

# The pinned version backstop provisions (pkg/pack/engine/allowlist.go) is NOT
# the semgrep on PATH. uvx fails on this toolchain (ModuleNotFoundError:
# pkg_resources); this venv route works. THE PATH SCRUB IS LOAD-BEARING: with
# the ambient PATH the 1.96.0 CLI delegates to whatever newer semgrep-core it
# finds and emits semanticVersion 1.156.0 under a 1.96.0 filename.
#
#   uv venv -p 3.11 .venv196
#   uv pip install --python .venv196/bin/python semgrep==1.96.0 'setuptools==70.3.0'
#     (the setuptools PIN is required: an unpinned resolve picks 83.0.0, which no
#      longer ships pkg_resources, and the capture dies on ModuleNotFoundError)
#   PATH="$PWD/.venv196/bin:/usr/bin:/bin" .venv196/bin/semgrep \
#     --config capture/rule-warning.yml --sarif capture/sample.go \
#     > descriptor-warning-1.96.0.sarif
#
# See PROVENANCE.md for the recorded outcome of that capture.
