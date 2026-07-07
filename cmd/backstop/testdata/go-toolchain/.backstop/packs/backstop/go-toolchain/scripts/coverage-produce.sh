#!/bin/sh
# go-toolchain coverage PRODUCER (ISSUE-045 option (ii)). The UN-SANDBOXED half of the
# coverage engine: the dispatch runs it via the runner (cwd = project root) with FULL
# toolchain + project access, IN PLACE of the plain engine command, so its `go test` /
# `go list` see the project. Its job is to run the Go coverage pass AND fold the two
# pieces of Go/toolchain knowledge the SANDBOXED parse-only convert
# (coverage-to-records.sh) cannot obtain itself — the module path and the package .go
# file list — into cover.out as PLAIN-TEXT comment lines. The convert then PARSES them
# (it runs no `go`). This producer/convert split is what keeps the executor language-
# blind AND the convert toolchain-free.
#
# Output contract: cover.out (the engine's declared stdout_artifact) — the raw Go
# `-coverprofile` with appended `#backstop-module <M>` and `#backstop-gofile
# <import-path>` lines. The producer's own stdout is noise; the dispatch reads the
# FILE. POSIX sh.

# Run the coverage pass. Tolerate a non-zero exit — a failing suite still yields a
# usable profile; the convert + gate decide the verdict, never this producer.
go test -coverprofile=cover.out ./... >/dev/null 2>&1 || true

# Nothing to enrich if the profile was not produced at all.
[ -f cover.out ] || exit 0

# Fold in the module path so the convert can strip the "<module>/" import-path prefix
# to yield repo-relative record paths (ISSUE-045 case-2, CLM-005).
module=$(go list -m 2>/dev/null | head -1)
if [ -n "$module" ]; then
  echo "#backstop-module $module" >> cover.out
fi

# Fold in every package's .go files (module-qualified, one per line) so the convert can
# emit a total:0 N/A record for a zero-statement file ABSENT from the profile whose
# package WAS measured (ISSUE-045 case-1, CLM-001). _test.go files are excluded by
# {{.GoFiles}} (test files are TestGoFiles), so only measurable source is listed.
go list -f '{{range .GoFiles}}{{$.ImportPath}}/{{.}}
{{end}}' ./... 2>/dev/null | while IFS= read -r gofile; do
  [ -n "$gofile" ] && echo "#backstop-gofile $gofile"
done >> cover.out

exit 0
