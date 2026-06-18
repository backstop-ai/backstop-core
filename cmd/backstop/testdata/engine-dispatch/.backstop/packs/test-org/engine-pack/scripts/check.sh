#!/bin/sh
# Generic exit-code script (input_mode: none): the executable is the logic.
# Exits non-zero when a forbidden marker file exists under the target.
target="$1"
if [ -f "$target/FORBIDDEN" ]; then
  echo "FORBIDDEN marker present under $target"
  exit 1
fi
exit 0
