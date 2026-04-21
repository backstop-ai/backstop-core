#!/bin/sh
if grep -R "fmt.Printf" "$1" >/dev/null 2>&1; then
  echo "fmt.Printf usage detected"
  exit 1
fi
echo "no printf usage"
exit 0

