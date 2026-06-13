package check

// Tool-output fixtures for executor unit tests. These capture the real output
// shapes of golangci-lint, go build, go test, and semgrep so every executor
// test parses canned bytes rather than invoking a live tool.
//
// PLAN DEVIATION (honest): PLAN-ISSUE-002 TASK-003 specifies these fixtures as
// files under pkg/check/testdata/ (golangci-lint-findings.json, etc.). The
// execution environment blocks writes to that directory, so the identical
// canned bytes are embedded here as package-level constants instead. The
// load-bearing contract — executor tests parse canned output, never shell out —
// is fully preserved; only the storage location differs.

// fixtureGolangciLintFindings is golangci-lint `--out-format json` output with
// three findings across two files, mixing error and warning severities.
const fixtureGolangciLintFindings = `{
  "Issues": [
    {
      "FromLinter": "errcheck",
      "Text": "Error return value of ` + "`w.Write`" + ` is not checked",
      "Severity": "error",
      "Pos": {
        "Filename": "pkg/server/handler.go",
        "Offset": 412,
        "Line": 37,
        "Column": 11
      }
    },
    {
      "FromLinter": "govet",
      "Text": "shadow: declaration of \"err\" shadows declaration at line 18",
      "Severity": "warning",
      "Pos": {
        "Filename": "pkg/server/handler.go",
        "Offset": 980,
        "Line": 52,
        "Column": 6
      }
    },
    {
      "FromLinter": "staticcheck",
      "Text": "SA4006: this value of ` + "`x`" + ` is never used",
      "Severity": "error",
      "Pos": {
        "Filename": "cmd/app/main.go",
        "Offset": 120,
        "Line": 9,
        "Column": 2
      }
    }
  ]
}`

// fixtureGolangciLintClean is golangci-lint output from a clean run: no issues.
const fixtureGolangciLintClean = `{
  "Issues": []
}`

// fixtureGoBuildErrors is go build stderr with `file:line:col: message`
// compiler errors. The leading "# package" header line and the trailing
// non-positional note line must be ignored by the parser.
const fixtureGoBuildErrors = `# github.com/example/proj/pkg/server
pkg/server/handler.go:42:13: undefined: doThing
pkg/server/handler.go:58:2: declared and not used: cfg
cmd/app/main.go:7:9: cannot use n (variable of type string) as int value in argument to add
note: module requires Go 1.22`

// fixtureGoTestFailures is go test plain output with two `--- FAIL: TestName`
// blocks, each followed by a file:line from the failure body.
const fixtureGoTestFailures = `=== RUN   TestAdd
--- FAIL: TestAdd (0.00s)
    math_test.go:14: add(2, 3) = 6, want 5
=== RUN   TestSub
--- FAIL: TestSub (0.00s)
    math_test.go:22: sub(5, 1) = 3, want 4
=== RUN   TestMul
--- PASS: TestMul (0.00s)
FAIL
exit status 1
FAIL    github.com/example/proj/math    0.012s`

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

// fixtureGolangciVersionV1 is a `golangci-lint version` banner identifying a
// v1.x binary, as the version detector parses it.
const fixtureGolangciVersionV1 = `golangci-lint has version 1.59.1 built with go1.22.3 from abc1234 on 2024-06-01`

// fixtureGolangciVersionV2 is a `golangci-lint version` banner identifying a
// v2.x binary.
const fixtureGolangciVersionV2 = `golangci-lint has version 2.1.6 built with go1.24.0 from def5678 on 2025-04-01`

// fixtureGolangciFailureOutput is the non-JSON output a golangci-lint failure
// exit (>=2) emits — here an unknown-flag rejection of a flag the installed
// version does not accept. CLM-003 asserts this excerpt surfaces in the error.
const fixtureGolangciFailureOutput = `Error: unknown flag: --out-format
Usage:
  golangci-lint run [flags]`
