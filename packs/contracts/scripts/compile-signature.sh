#!/bin/sh
# contract->ast-grep-pattern COMPILER (SPEC-038 REQ-002/CLM-006). This is a
# PACK-RELATIVE script — the backstop binary NEVER compiles or renders a
# signature; it passes the human-readable Signature here and reads back an
# ast-grep pattern. The compiler turns a declared Go signature such as
#   func RouteFile(path string, mode int) (string, error)
# into a STRUCTURAL ast-grep pattern that is param-NAME/whitespace-insensitive
# (CLM-007) while preserving the function name and the return clause:
#   func RouteFile($$$PARAMS) (string, error)
# The parameter list is replaced with the ast-grep multi-metavariable $$$PARAMS
# (matches any params, so a param-name/whitespace variant still matches), and the
# return clause is preserved verbatim (so a return-type change does NOT match).
# The signature is read from $1 (or stdin if $1 is empty). ast-grep knows nothing
# of Go signatures; backstop knows nothing of ast-grep patterns — this script is
# the only place the mapping lives.
sig="$1"
if [ -z "$sig" ]; then
  sig=$(cat)
fi

# Drop a leading "func " if present, then split "<name>(<params>)<rest>".
# name = up to the first '('. rest = everything after the matching first ')'
# of the parameter list (the return clause, possibly empty).
work=${sig#func }
work=$(printf '%s' "$work" | sed 's/^[[:space:]]*//')

name=${work%%(*}
name=$(printf '%s' "$name" | sed 's/[[:space:]]*$//')

# rest after first ")": params live between first "(" and first ")".
afterparen=${work#*\)}
afterparen=$(printf '%s' "$afterparen" | sed 's/^[[:space:]]*//')

if [ -n "$afterparen" ]; then
  printf 'func %s($$$PARAMS) %s' "$name" "$afterparen"
else
  printf 'func %s($$$PARAMS)' "$name"
fi
