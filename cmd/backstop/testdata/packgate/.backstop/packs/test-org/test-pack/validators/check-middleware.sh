#!/bin/sh
if [ -f "$1/middleware.go" ]; then
  echo "middleware present"
  exit 0
fi
echo "middleware.go missing"
exit 1

