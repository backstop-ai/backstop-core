#!/bin/sh
grep -qx ISSUE188_FIXTURE_PASS || exit 65
printf '{"version":"2.1.0","runs":[{"results":[]}]}\n'
