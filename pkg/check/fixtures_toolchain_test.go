package check

// This file holds SARIF parser test fixtures as Go literals. The toolchain /
// eslint / tsc / regex-lines / declared-stack fixtures were removed with the
// in-process check engine and its parsers (ISSUE-018); only the SARIF fixture
// the surviving parser test consumes remains.

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
