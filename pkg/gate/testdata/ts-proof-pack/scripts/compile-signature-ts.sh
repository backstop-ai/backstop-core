#!/bin/sh
# TS contract->ast-grep-pattern COMPILER (SPEC-038 REQ-002/CLM-006, TS form). Turns
# a declared TS signature such as
#   function routeFile(path: string, mode: number): string
# into a STRUCTURAL ast-grep pattern that is param-NAME/whitespace-insensitive
# (CLM-007) while preserving the function name and the return type:
#   function routeFile($$$PARAMS): string { $$$BODY }
# (TypeScript function declarations need a body node for ast-grep's pattern parser
# to bind to the declaration; $$$BODY matches any body.) The binary never compiles
# a signature — this pack script is the only place the mapping lives. The leading
# `export` modifier is intentionally dropped so an exported or non-exported decl
# both match. Signature read from $1 or stdin.
sig="$1"
if [ -z "$sig" ]; then sig=$(cat); fi
work=${sig#export }
work=${work#function }
work=$(printf '%s' "$work" | sed 's/^[[:space:]]*//')
name=${work%%(*}
name=$(printf '%s' "$name" | sed 's/[[:space:]]*$//')
after=${work#*\)}
after=$(printf '%s' "$after" | sed 's/^[[:space:]]*//')
if [ -n "$after" ]; then
  printf 'function %s($$$PARAMS)%s { $$$BODY }' "$name" "$after"
else
  printf 'function %s($$$PARAMS) { $$$BODY }' "$name"
fi
