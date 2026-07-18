#!/bin/sh
# ISSUE-068 fixture: the COVERAGE engine's convert. Reads the SAME shared suite
# run payload on stdin and emits one per-file coverage record — proving the single
# run's output is fanned into a DISTINCT convert producing a DISTINCT output type.
cat >/dev/null
printf '[{"path":"suite.ts","covered":9,"total":10,"measured":true,"excluded":false,"metric":"statement"}]\n'
