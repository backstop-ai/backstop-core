package check

// Tool-output fixtures for semgrep executor unit tests. These capture the real
// output shapes of semgrep so every executor test parses canned bytes rather
// than invoking a live tool.
//
// SPEC-034 cutover: the golangci-lint / go build / go test fixtures were removed
// with the bespoke executors they fed — the Go toolchain output shapes are now
// captured under cmd/backstop/testdata/go-toolchain/fixtures and exercised by the
// go-toolchain pack engine tests. Only the semgrep fixtures remain here.

// fixtureSemgrepFindings is semgrep `--json` output with three results. The
// first carries a pack-namespaced check_id ("slotly/go-standards/no-panic")
// that must be preserved verbatim for source-pack attribution; the others use
// plain rule IDs. Severities cover both ERROR and WARNING.
const fixtureSemgrepFindings = `{
  "results": [
    {
      "check_id": "slotly/go-standards/no-panic",
      "path": "pkg/server/handler.go",
      "start": {"line": 31, "col": 3},
      "end": {"line": 31, "col": 40},
      "extra": {
        "message": "panic() is disallowed; return an error instead",
        "severity": "ERROR"
      }
    },
    {
      "check_id": "no-fmt-println",
      "path": "cmd/app/main.go",
      "start": {"line": 12, "col": 2},
      "end": {"line": 12, "col": 25},
      "extra": {
        "message": "use the structured logger instead of fmt.Println",
        "severity": "WARNING"
      }
    },
    {
      "check_id": "slotly/go-standards/require-context",
      "path": "pkg/server/client.go",
      "start": {"line": 8, "col": 1},
      "end": {"line": 8, "col": 30},
      "extra": {
        "message": "exported funcs doing IO must accept a context.Context",
        "severity": "ERROR"
      }
    }
  ],
  "errors": [],
  "paths": {"scanned": ["pkg/server/handler.go", "cmd/app/main.go", "pkg/server/client.go"]}
}`

// fixtureSemgrepClean is semgrep `--json` output with no findings.
const fixtureSemgrepClean = `{"results": [], "errors": [], "paths": {"scanned": []}}`

// fixtureSemgrepFindingsWithPreamble reproduces the live ISSUE-006 failure:
// real semgrep emits non-JSON banner/progress bytes (UTF-8 multibyte
// punctuation — the exact 'â'-class byte from the observed
// `invalid character 'â' looking for beginning of value` error) BEFORE the
// JSON document. The JSON payload itself is byte-identical to
// fixtureSemgrepFindings, so a correct extraction yields the same three
// findings. CLM-001 asserts this parses cleanly.
const fixtureSemgrepFindingsWithPreamble = "┌──── semgrep ────┐\n" +
	"Scanning 3 files with 12 rules… ✔\nâ progress: 100%\n" +
	fixtureSemgrepFindings
