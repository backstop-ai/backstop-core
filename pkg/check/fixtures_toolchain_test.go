package check

import "os"

// writeFileForTest writes content to path with 0o644 perms. Tests use it to
// materialize config fixtures into a t.TempDir() at runtime, the codebase's
// dominant pattern (the test process may write to temp dirs even though the
// implementer agent cannot create committed testdata/ files in this
// environment).
func writeFileForTest(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}

// This file holds toolchain/parser test fixtures as Go literals.
//
// NOTE (environment constraint): the implementer agent in this environment is
// not permitted to create files under testdata/. TASK-001 explicitly sanctions
// fixture literals in a *_test.go helper as the alternative to testdata files
// ("or as fixture literals in a new pkg/check/fixtures_toolchain_test.go
// helper"). All parser/registry/TS fixtures therefore live here as constants,
// keeping the tests fully hermetic with no live tool invocation.

// sarifSampleJSON is a SARIF 2.1.0 log with runs[].results[] carrying ruleId,
// level (error AND warning, plus one with level absent that must default to
// error), message.text, and locations[0].physicalLocation.artifactLocation.uri
// + region.startLine. Exercises the sarif format parser (CLM-005).
const sarifSampleJSON = `{
  "version": "2.1.0",
  "runs": [
    {
      "tool": { "driver": { "name": "example-analyzer" } },
      "results": [
        {
          "ruleId": "EXAMPLE001",
          "level": "error",
          "message": { "text": "undefined symbol referenced" },
          "locations": [
            {
              "physicalLocation": {
                "artifactLocation": { "uri": "src/alpha.rs" },
                "region": { "startLine": 42 }
              }
            }
          ]
        },
        {
          "ruleId": "EXAMPLE002",
          "level": "warning",
          "message": { "text": "unused variable" },
          "locations": [
            {
              "physicalLocation": {
                "artifactLocation": { "uri": "src/beta.rs" },
                "region": { "startLine": 7 }
              }
            }
          ]
        },
        {
          "ruleId": "EXAMPLE003",
          "message": { "text": "level absent should default to error" },
          "locations": [
            {
              "physicalLocation": {
                "artifactLocation": { "uri": "src/gamma.rs" },
                "region": { "startLine": 3 }
              }
            }
          ]
        }
      ]
    }
  ]
}`

// eslintSampleJSON is an eslint JSON array of {filePath, messages:[{ruleId,
// severity (1 AND 2), message, line}]}. The second file has an empty messages
// array (a clean file). Exercises the eslint-json format (CLM-003, CLM-005).
const eslintSampleJSON = `[
  {
    "filePath": "/repo/src/app.ts",
    "messages": [
      {
        "ruleId": "no-undef",
        "severity": 2,
        "message": "'foo' is not defined.",
        "line": 12
      },
      {
        "ruleId": "no-unused-vars",
        "severity": 1,
        "message": "'bar' is assigned a value but never used.",
        "line": 18
      }
    ]
  },
  {
    "filePath": "/repo/src/util.ts",
    "messages": []
  }
]`

// tscSampleTxt is tsc --noEmit stderr of the shape
// `file(line,col): error TSxxxx: message`, including a multi-line / non-matching
// line to prove robustness. Exercises the tsc format (CLM-003).
const tscSampleTxt = `src/app.ts(12,7): error TS2304: Cannot find name 'foo'.
src/util.ts(3,1): warning TS6133: 'bar' is declared but its value is never read.
Found 2 errors in 2 files.
src/widget.ts(40,15): error TS2322: Type 'number' is not assignable to type 'string'.`

// regexLinesSampleTxt is generic `file:line:col message` output, including a
// line the default pattern should match and a line it should not. Exercises the
// regex-lines configurable parser (CLM-005).
const regexLinesSampleTxt = `src/lib.rs:10:5 borrow of moved value
this line has no position and must not match
src/main.rs:3:1 unused import`

// declaredStackBackstopYML is a declared-stack config (language: rust) with
// enforcement.toolchain.{lint,build,test} {command, format: regex-lines,
// extensions} — the declared-stack-as-data shape (CLM-002).
const declaredStackBackstopYML = `project: rust-example
language: rust
enforcement:
  toolchain:
    lint:
      command: "cargo clippy --message-format short"
      format: regex-lines
      extensions: [".rs"]
    build:
      command: "cargo build"
      format: regex-lines
      extensions: [".rs"]
    test:
      command: "cargo test"
      format: regex-lines
      extensions: [".rs"]
`

// tsBackstopYML is a typescript config WITH enforcement.test_command declared
// (the explicit TS test command — CLM-004 positive case).
const tsBackstopYML = `project: ts-example
language: typescript
enforcement:
  test_command: "vitest run"
  toolchain:
    test:
      command: "vitest run"
      format: regex-lines
      test_dependency_command: "vitest related"
`

// tsNoTestCommandBackstopYML is a typescript config WITHOUT
// enforcement.test_command — the config-error fixture (CLM-004 negative case).
const tsNoTestCommandBackstopYML = `project: ts-example
language: typescript
`
