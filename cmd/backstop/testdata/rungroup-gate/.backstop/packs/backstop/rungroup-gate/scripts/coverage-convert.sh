#!/bin/sh
# ISSUE-068 fixture COVERAGE convert: reads the SAME shared suite payload and emits
# one per-file coverage record — a DISTINCT output type from the same single run.
cat >/dev/null
printf '[{"path":"suite.ts","covered":9,"total":10,"measured":true,"excluded":false,"metric":"statement"}]\n'
