package check

import (
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// gitleaksShapedLog returns a SARIF log whose structure is mirrored from
// gitleaks 8.30.1 output (dir mode), secret redacted. Field for field it
// reproduces what the real producer emits: the schemastore $schema, a driver
// with semanticVersion/informationUri and a rule descriptor carrying only id +
// shortDescription (no defaultConfiguration), and results with no `level`,
// all-empty dir-mode partialFingerprints and a `properties` bag. Only the
// generic-api-key descriptor of the real 222 is kept. The properties bag of
// both results is substituted verbatim from `props` (ISSUE-208).
func gitleaksShapedLog(props string) string {
	return `{
  "$schema": "https://json.schemastore.org/sarif-2.1.0.json",
  "version": "2.1.0",
  "runs": [
    {
      "tool": {
        "driver": {
          "name": "gitleaks",
          "semanticVersion": "v8.0.0",
          "informationUri": "https://github.com/gitleaks/gitleaks",
          "rules": [
            {
              "id": "generic-api-key",
              "shortDescription": {
                "text": "Detected a Generic API Key, potentially exposing access to various services and sensitive operations."
              }
            }
          ]
        }
      },
      "results": [
        {
          "message": { "text": "generic-api-key has detected secret for file src/config.go." },
          "ruleId": "generic-api-key",
          "locations": [
            {
              "physicalLocation": {
                "artifactLocation": { "uri": "src/config.go" },
                "region": {
                  "startLine": 3,
                  "startColumn": 8,
                  "endLine": 3,
                  "endColumn": 50,
                  "snippet": { "text": "REDACTED" }
                }
              }
            }
          ],
          "partialFingerprints": {
            "commitSha": "",
            "email": "",
            "author": "",
            "date": "",
            "commitMessage": ""
          },
          "properties": ` + props + `
        },
        {
          "message": { "text": "generic-api-key has detected secret for file pkg/app/settings.go." },
          "ruleId": "generic-api-key",
          "locations": [
            {
              "physicalLocation": {
                "artifactLocation": { "uri": "pkg/app/settings.go" },
                "region": {
                  "startLine": 12,
                  "startColumn": 5,
                  "endLine": 12,
                  "endColumn": 47,
                  "snippet": { "text": "REDACTED" }
                }
              }
            }
          ],
          "partialFingerprints": {
            "commitSha": "",
            "email": "",
            "author": "",
            "date": "",
            "commitMessage": ""
          },
          "properties": ` + props + `
        }
      ]
    }
  ]
}`
}

// singleResultLog returns a minimal one-result SARIF log whose result carries
// `propsField` verbatim as an extra member (e.g. `, "properties": {...}`), or
// no properties member at all when propsField is "".
func singleResultLog(propsField string) string {
	return `{
  "version": "2.1.0",
  "runs": [
    {
      "results": [
        {
          "ruleId": "probe-rule",
          "level": "warning",
          "message": { "text": "probe message" },
          "locations": [
            {
              "physicalLocation": {
                "artifactLocation": { "uri": "probe.go" },
                "region": { "startLine": 7 }
              }
            }
          ]` + propsField + `
        }
      ]
    }
  ]
}`
}

// TestParseSarif_PropertyBagArrayTagsSurfacesViolation pins ISSUE-208 CLM-001:
// a gitleaks-shaped log whose results carry the SARIF 2.1.0 §3.8.2 reserved
// `properties.tags` ARRAY parses with no error and surfaces every result as a
// Violation with concrete, unchanged fields. Before the fix the array made
// json.Unmarshal fail and parseSarif rejected the whole log.
func TestParseSarif_PropertyBagArrayTagsSurfacesViolation(t *testing.T) {
	// ISSUE-210 amendment: the all-empty dir-mode partialFingerprints are
	// degenerate and both snippets are "REDACTED", so each fingerprint is the
	// position hash of its own start line and column, never the old collapsed
	// constant below.
	const collapsedFingerprint = "author=;commitMessage=;commitSha=;date=;email="
	positionFingerprint := func(line, column int) string {
		sum := sha256.Sum256([]byte("position:" + strconv.Itoa(line) + ":" + strconv.Itoa(column)))
		return "sha256:" + hex.EncodeToString(sum[:])
	}
	type wantViolation struct {
		file        string
		line        int
		message     string
		fingerprint string
	}
	wants := []wantViolation{
		{file: "src/config.go", line: 3, message: "generic-api-key has detected secret for file src/config.go.", fingerprint: positionFingerprint(3, 8)},
		{file: "pkg/app/settings.go", line: 12, message: "generic-api-key has detected secret for file pkg/app/settings.go.", fingerprint: positionFingerprint(12, 5)},
	}
	rows := []struct {
		name  string
		props string
	}{
		{name: "primary empty tags array", props: `{"tags": []}`},
		{name: "extra non-empty tags array", props: `{"tags": ["secret", "api-key"]}`},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			violations, err := parseSarif([]byte(gitleaksShapedLog(row.props)), CheckTypeFindings)
			if err != nil {
				t.Fatalf("parseSarif: %v (an array-valued property must not reject the log)", err)
			}
			if len(violations) != len(wants) {
				t.Fatalf("got %d violations, want %d", len(violations), len(wants))
			}
			for i, want := range wants {
				v := violations[i]
				if v.File != want.file {
					t.Errorf("violation %d File = %q, want %q", i, v.File, want.file)
				}
				if v.Line != want.line {
					t.Errorf("violation %d Line = %d, want %d", i, v.Line, want.line)
				}
				if v.Rule != "generic-api-key" {
					t.Errorf("violation %d Rule = %q, want generic-api-key", i, v.Rule)
				}
				if v.Message != want.message {
					t.Errorf("violation %d Message = %q, want %q", i, v.Message, want.message)
				}
				if v.Severity != "error" {
					t.Errorf("violation %d Severity = %q, want error (fail-closed default: no result or descriptor level)", i, v.Severity)
				}
				if v.Fingerprint != want.fingerprint {
					t.Errorf("violation %d Fingerprint = %q, want %q (position hash: degenerate partialFingerprints, redacted snippet)", i, v.Fingerprint, want.fingerprint)
				}
				if v.Fingerprint == collapsedFingerprint {
					t.Errorf("violation %d Fingerprint is the collapsed constant %q", i, collapsedFingerprint)
				}
				if strings.Contains(v.Fingerprint, "REDACTED") {
					t.Errorf("violation %d Fingerprint %q contains the redaction marker", i, v.Fingerprint)
				}
				if v.Pass != CheckTypeFindings {
					t.Errorf("violation %d Pass = %q, want %q", i, v.Pass, CheckTypeFindings)
				}
				if v.Properties != nil {
					t.Errorf("violation %d Properties = %#v, want nil (the non-string tags entry is ignored)", i, v.Properties)
				}
			}
			if violations[0].Fingerprint == violations[1].Fingerprint {
				t.Errorf("both results share Fingerprint %q, want distinct per-position fingerprints", violations[0].Fingerprint)
			}
		})
	}
}

// TestParseSarif_PropertyBagStringsPreservedAlongsideNonString pins ISSUE-208
// CLM-002: string-valued property entries (including the substantiveness join
// keys func/symbol/substantiveness_role and a JSON-escaped value) are preserved
// in their decoded form next to array/object/number/bool/null entries on the
// same result, and no non-string entry leaks into Properties in any form.
func TestParseSarif_PropertyBagStringsPreservedAlongsideNonString(t *testing.T) {
	in := singleResultLog(`,
          "properties": {
            "func": "spaced test name",
            "symbol": "readmodel",
            "substantiveness_role": "hollow",
            "quoted": "say \"hi\" to café",
            "tags": ["a", "b"],
            "meta": {"nested": "value", "n": 1},
            "rank": 3,
            "ok": true,
            "gone": null
          }`)
	violations, err := parseSarif([]byte(in), CheckTypeFindings)
	if err != nil {
		t.Fatalf("parseSarif: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("got %d violations, want 1", len(violations))
	}
	want := map[string]string{
		"func":                 "spaced test name",
		"symbol":               "readmodel",
		"substantiveness_role": "hollow",
		"quoted":               "say \"hi\" to café",
	}
	if got := violations[0].Properties; !reflect.DeepEqual(got, want) {
		t.Errorf("Properties = %#v, want exactly %#v", got, want)
	}
}

// TestParseSarif_PropertyBagNonStringValuesIgnored pins ISSUE-208 CLM-002 per
// JSON kind: each non-string value is dropped while a string sentinel beside it
// is kept. The null row pins the deliberate behavior change that a null-valued
// entry is dropped (the old map[string]string decode stored it as "").
func TestParseSarif_PropertyBagNonStringValuesIgnored(t *testing.T) {
	rows := []struct {
		name  string
		value string
	}{
		{name: "array", value: `["x", "y"]`},
		{name: "object", value: `{"k": "v"}`},
		{name: "integer", value: `42`},
		{name: "float", value: `3.5`},
		{name: "true", value: `true`},
		{name: "false", value: `false`},
		{name: "null", value: `null`},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			in := singleResultLog(`, "properties": {"sentinel": "kept", "probed": ` + row.value + `}`)
			violations, err := parseSarif([]byte(in), CheckTypeFindings)
			if err != nil {
				t.Fatalf("parseSarif: %v", err)
			}
			if len(violations) != 1 {
				t.Fatalf("got %d violations, want 1", len(violations))
			}
			props := violations[0].Properties
			if got, ok := props["sentinel"]; !ok || got != "kept" {
				t.Errorf("Properties[sentinel] = %q (present=%v), want \"kept\"", got, ok)
			}
			if got, ok := props["probed"]; ok {
				t.Errorf("probed key present with %q, want it absent (non-string %s value)", got, row.name)
			}
			if len(props) != 1 {
				t.Errorf("Properties = %#v, want only the sentinel", props)
			}
		})
	}
}

// TestParseSarif_PropertyBagNilMapWhenNoStringEntries pins ISSUE-208 CLM-002:
// when a result keeps no string entry — properties absent, the null literal, an
// empty object, or only non-string values — Properties is strictly nil and the
// log parses without error.
func TestParseSarif_PropertyBagNilMapWhenNoStringEntries(t *testing.T) {
	rows := []struct {
		name       string
		propsField string
	}{
		{name: "absent", propsField: ``},
		{name: "properties null", propsField: `, "properties": null`},
		{name: "empty object", propsField: `, "properties": {}`},
		{name: "only non-string values", propsField: `, "properties": {"tags": [], "rank": 1, "ok": false, "gone": null, "meta": {}}`},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			violations, err := parseSarif([]byte(singleResultLog(row.propsField)), CheckTypeFindings)
			if err != nil {
				t.Fatalf("parseSarif: %v", err)
			}
			if len(violations) != 1 {
				t.Fatalf("got %d violations, want 1", len(violations))
			}
			v := violations[0]
			if v.Properties != nil {
				t.Errorf("Properties = %#v (len %d), want strict nil", v.Properties, len(v.Properties))
			}
			if v.Rule != "probe-rule" || v.File != "probe.go" || v.Line != 7 || v.Severity != "warning" {
				t.Errorf("existing fields changed: Rule=%q File=%q Line=%d Severity=%q", v.Rule, v.File, v.Line, v.Severity)
			}
		})
	}
}

// TestParseSarif_PropertyBagMalformedStillFails pins ISSUE-208 CLM-003: the
// lenient property bag must not be implemented by tolerating decode errors.
// Genuinely malformed SARIF — truncated JSON, a wrong top-level type, a
// wrong-typed non-property field, or a `properties` that is not an object —
// still fails loud with "parsing SARIF output" and yields no violations.
func TestParseSarif_PropertyBagMalformedStillFails(t *testing.T) {
	valid := singleResultLog(`, "properties": {"func": "x"}`)
	truncated := valid[:strings.Index(valid, `"probe message"`)]
	rows := []struct {
		name string
		in   string
	}{
		{name: "truncated mid-result", in: truncated},
		{name: "top-level array", in: `[{"version": "2.1.0", "runs": []}]`},
		{name: "top-level string", in: `"not a sarif log"`},
		{name: "string startLine", in: strings.Replace(singleResultLog(``), `"startLine": 7`, `"startLine": "7"`, 1)},
		{name: "properties is array", in: singleResultLog(`, "properties": ["func", "x"]`)},
		{name: "properties is string", in: singleResultLog(`, "properties": "func=x"`)},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			violations, err := parseSarif([]byte(row.in), CheckTypeFindings)
			if err == nil {
				t.Fatalf("parseSarif returned nil error and %d violations, want a parse failure", len(violations))
			}
			if !strings.Contains(err.Error(), "parsing SARIF output") {
				t.Errorf("error = %q, want it to contain \"parsing SARIF output\"", err)
			}
			if violations != nil {
				t.Errorf("violations = %#v, want nil on malformed input", violations)
			}
		})
	}
}
