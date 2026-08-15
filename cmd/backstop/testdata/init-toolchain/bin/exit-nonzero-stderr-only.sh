#!/bin/sh
# ★ THE CAPTURE-METHOD TRIPWIRE (SPEC-069 CLM-122 / Sharp Edge 15).
#
# ITS STDOUT MUST STAY EMPTY. This script writes its ENTIRE diagnostic to stderr and
# NOTHING AT ALL to stdout. A fixture that also writes to stdout passes under BOTH
# check.CommandRunner.Run (combined) and RunStdout (stdout only), so the claim goes
# vacuous while looking exactly like it is working -- and the whole reason CLM-122
# exists is that the one-line difference between those two methods is invisible in a
# diff review. Do not add an echo here.
#
# A failing build or test entrypoint routinely diagnoses only on stderr, so this is
# the realistic case, not the exotic one.
echo "entrypoint failed: dependency resolution error (diagnostic on stderr only)" >&2
exit 7
