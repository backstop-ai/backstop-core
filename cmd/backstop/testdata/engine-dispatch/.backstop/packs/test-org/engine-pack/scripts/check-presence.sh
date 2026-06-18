#!/bin/sh
# Exit-code sandbox validator: fails (non-zero) when the marker file is absent.
target="$1"
if [ -f "$target/MARKER" ]; then
  exit 0
fi
echo "MARKER file missing under $target"
exit 1
