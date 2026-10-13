package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/gate"
	"github.com/backstop-ai/backstop-core/pkg/pack"
)

// sarif_degenerate_identity_e2e_test.go (ISSUE-210) drives gitleaks-shaped SARIF
// (all-empty dir-mode partialFingerprints) through the REAL findings dispatch
// (dispatchPackEngines -> runFindingsEngine -> real SARIF parse -> the
// pack_gate.go bridge RegionHash = Fingerprint) and pins that same-rule secrets
// in one file keep distinct baseline identities, that no raw snippet reaches the
// baseline or gate JSON, and that baseline comparison neither grandfathers a new
// secret nor hides a fixed one. The tool executable is stood in by
// degenerateIdentityRunner (the standard cmd/backstop dispatch-e2e pattern); the
// dispatch, parse and bridge under test all execute for real. All snippet
// strings are synthetic and non-secret.

const (
	degenerateIdentityAlpha = "synthetic-snippet-alpha"
	degenerateIdentityBravo = "synthetic-snippet-bravo"
	degenerateIdentityFile  = "src/config.go"
)

// degenerateIdentitySecret is one gitleaks-shaped finding the runner emits.
type degenerateIdentitySecret struct {
	line, column int
	snippet      string
}

// degenerateIdentityRunner returns a fixed gitleaks-shaped SARIF log for any
// invocation: rule generic-api-key, no level, all-empty dir-mode
// partialFingerprints.
type degenerateIdentityRunner struct {
	log []byte
}

func (r *degenerateIdentityRunner) Run(_ context.Context, _ string, _ ...string) ([]byte, error) {
	return r.log, nil
}

func (r *degenerateIdentityRunner) RunStdout(_ context.Context, _ string, _ ...string) ([]byte, error) {
	return r.log, nil
}

// degenerateIdentityLog marshals the secrets into a gitleaks-shaped SARIF log
// with every result in degenerateIdentityFile.
func degenerateIdentityLog(t *testing.T, secrets ...degenerateIdentitySecret) []byte {
	t.Helper()
	results := make([]map[string]any, 0, len(secrets))
	for _, s := range secrets {
		results = append(results, map[string]any{
			"message": map[string]any{"text": "generic-api-key has detected secret for file " + degenerateIdentityFile + "."},
			"ruleId":  "generic-api-key",
			"locations": []any{map[string]any{
				"physicalLocation": map[string]any{
					"artifactLocation": map[string]any{"uri": degenerateIdentityFile},
					"region": map[string]any{
						"startLine":   s.line,
						"startColumn": s.column,
						"snippet":     map[string]any{"text": s.snippet},
					},
				},
			}},
			"partialFingerprints": map[string]string{
				"commitSha": "", "email": "", "author": "", "date": "", "commitMessage": "",
			},
		})
	}
	out, err := json.Marshal(map[string]any{
		"version": "2.1.0",
		"runs":    []any{map[string]any{"tool": map[string]any{"driver": map[string]any{"name": "gitleaks"}}, "results": results}},
	})
	if err != nil {
		t.Fatalf("marshal degenerate identity fixture: %v", err)
	}
	return out
}

// degenerateIdentityDispatch runs the real findings dispatch (nil scope) over a
// runner emitting the given secrets and requires one violation per secret.
func degenerateIdentityDispatch(t *testing.T, secrets ...degenerateIdentitySecret) []gate.Violation {
	t.Helper()
	runner := &degenerateIdentityRunner{log: degenerateIdentityLog(t, secrets...)}
	violations, err := dispatchPackEngines(
		[]*pack.Manifest{semgrepDispatchManifest()},
		engineDispatchPacksDir(t),
		t.TempDir(),
		nil,
		runner,
	)
	if err != nil {
		t.Fatalf("dispatchPackEngines: %v", err)
	}
	if len(violations) != len(secrets) {
		t.Fatalf("got %d violations through real dispatch, want %d: %#v", len(violations), len(secrets), violations)
	}
	return violations
}

// degenerateIdentitySnippetHash is the independent oracle for the snippet hash.
func degenerateIdentitySnippetHash(snippet string) string {
	sum := sha256.Sum256([]byte("snippet:" + snippet))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// degenerateIdentityPositionHash is the independent oracle for the position hash.
func degenerateIdentityPositionHash(line, column int) string {
	sum := sha256.Sum256([]byte("position:" + strconv.Itoa(line) + ":" + strconv.Itoa(column)))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// degenerateIdentityBaseline builds a baseline artifact from dispatched violations.
func degenerateIdentityBaseline(violations []gate.Violation) *gate.BaselineArtifact {
	enriched := make([]gate.Violation, 0, len(violations))
	for _, v := range violations {
		enriched = append(enriched, gate.EnrichViolationIdentity(v))
	}
	return &gate.BaselineArtifact{SchemaVersion: gate.BaselineSchemaV1, Violations: enriched}
}

// TestSarifDegenerateIdentity_DistinctIdentityThroughDispatch pins ISSUE-210
// CLM-002 and CLM-003 through the real dispatch: two same-rule secrets in one
// file (informative snippets, or both redacted at different positions) get
// distinct RegionHash and distinct IdentityHash.
func TestSarifDegenerateIdentity_DistinctIdentityThroughDispatch(t *testing.T) {
	rows := []struct {
		name    string
		secrets []degenerateIdentitySecret
		want    []string
	}{
		{
			name:    "informative snippets",
			secrets: []degenerateIdentitySecret{{3, 8, degenerateIdentityAlpha}, {9, 8, degenerateIdentityBravo}},
			want:    []string{degenerateIdentitySnippetHash(degenerateIdentityAlpha), degenerateIdentitySnippetHash(degenerateIdentityBravo)},
		},
		{
			name:    "redacted at different positions",
			secrets: []degenerateIdentitySecret{{3, 8, "REDACTED"}, {9, 8, "REDACTED"}},
			want:    []string{degenerateIdentityPositionHash(3, 8), degenerateIdentityPositionHash(9, 8)},
		},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			violations := degenerateIdentityDispatch(t, row.secrets...)
			for i, v := range violations {
				if v.RegionHash != row.want[i] {
					t.Errorf("violation %d RegionHash = %q, want %q", i, v.RegionHash, row.want[i])
				}
			}
			if violations[0].RegionHash == violations[1].RegionHash {
				t.Errorf("both secrets share RegionHash %q", violations[0].RegionHash)
			}
			a := gate.EnrichViolationIdentity(violations[0])
			b := gate.EnrichViolationIdentity(violations[1])
			if a.IdentityHash == b.IdentityHash {
				t.Errorf("both secrets share IdentityHash %s (Identity %q): same-rule secrets in one file collapse", a.IdentityHash, a.Identity)
			}
		})
	}
}

// TestSarifDegenerateIdentity_NoRawSnippetPersisted pins ISSUE-210 CLM-004: on
// the degenerate path no raw snippet reaches the baseline file WriteBaseline
// writes or the gate JSON FormatJSON produces, and every persisted region_hash
// is a sha256: hash.
func TestSarifDegenerateIdentity_NoRawSnippetPersisted(t *testing.T) {
	violations := degenerateIdentityDispatch(t,
		degenerateIdentitySecret{3, 8, degenerateIdentityAlpha},
		degenerateIdentitySecret{9, 8, degenerateIdentityBravo},
	)
	baselinePath := filepath.Join(t.TempDir(), "baseline.json")
	if err := gate.WriteBaseline(baselinePath, degenerateIdentityBaseline(violations)); err != nil {
		t.Fatalf("WriteBaseline: %v", err)
	}
	baselineBytes, err := os.ReadFile(baselinePath)
	if err != nil {
		t.Fatalf("read baseline: %v", err)
	}
	enriched := degenerateIdentityBaseline(violations).Violations
	result := gate.NewGateResult([]gate.StepResult{{
		StepName:      "pack_engines",
		Status:        "fail",
		Violations:    append([]gate.Violation(nil), enriched...),
		NewViolations: append([]gate.Violation(nil), enriched...),
	}})
	gateJSON, err := gate.FormatJSON(result)
	if err != nil {
		t.Fatalf("FormatJSON: %v", err)
	}
	for name, data := range map[string][]byte{"baseline": baselineBytes, "gate JSON": gateJSON} {
		for _, raw := range []string{degenerateIdentityAlpha, degenerateIdentityBravo} {
			if bytes.Contains(data, []byte(raw)) {
				t.Errorf("%s bytes contain raw snippet %q", name, raw)
			}
		}
	}
	var decoded gate.BaselineArtifact
	if err := json.Unmarshal(baselineBytes, &decoded); err != nil {
		t.Fatalf("decode baseline: %v", err)
	}
	if len(decoded.Violations) != 2 {
		t.Fatalf("decoded baseline holds %d violations, want 2", len(decoded.Violations))
	}
	for i, v := range decoded.Violations {
		if !strings.HasPrefix(v.RegionHash, "sha256:") {
			t.Errorf("baseline violation %d region_hash = %q, want a sha256: hash", i, v.RegionHash)
		}
	}
}

// TestSarifDegenerateIdentity_SecondSecretReportedOnNilScope pins ISSUE-210
// CLM-005: a baseline holding one degenerate-fingerprint secret does not
// grandfather a second same-rule secret in the same file on a nil-scope
// compare, and dropping one of two baselined secrets reports it as fixed.
func TestSarifDegenerateIdentity_SecondSecretReportedOnNilScope(t *testing.T) {
	rows := []struct {
		name      string
		baseline  []degenerateIdentitySecret
		current   []degenerateIdentitySecret
		wantNew   []string
		wantFixed []string
	}{
		{
			name:     "informative snippet B is new",
			baseline: []degenerateIdentitySecret{{3, 8, degenerateIdentityAlpha}},
			current:  []degenerateIdentitySecret{{3, 8, degenerateIdentityAlpha}, {9, 8, degenerateIdentityBravo}},
			wantNew:  []string{degenerateIdentitySnippetHash(degenerateIdentityBravo)},
		},
		{
			name:     "redacted B is new",
			baseline: []degenerateIdentitySecret{{3, 8, "REDACTED"}},
			current:  []degenerateIdentitySecret{{3, 8, "REDACTED"}, {9, 8, "REDACTED"}},
			wantNew:  []string{degenerateIdentityPositionHash(9, 8)},
		},
		{
			name:      "dropping B reports it fixed",
			baseline:  []degenerateIdentitySecret{{3, 8, degenerateIdentityAlpha}, {9, 8, degenerateIdentityBravo}},
			current:   []degenerateIdentitySecret{{3, 8, degenerateIdentityAlpha}},
			wantFixed: []string{degenerateIdentitySnippetHash(degenerateIdentityBravo)},
		},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			baseline := degenerateIdentityBaseline(degenerateIdentityDispatch(t, row.baseline...))
			current := degenerateIdentityDispatch(t, row.current...)
			comparison := gate.CompareBaseline(current, baseline, gate.BaselineCompareOptions{Scope: nil})
			assertDegenerateIdentityRegions(t, "NewViolations", comparison.NewViolations, row.wantNew)
			assertDegenerateIdentityRegions(t, "FixedViolations", comparison.FixedViolations, row.wantFixed)
		})
	}
}

// assertDegenerateIdentityRegions requires got to hold exactly the violations
// whose RegionHash values are want, in order.
func assertDegenerateIdentityRegions(t *testing.T, label string, got []gate.Violation, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: got %d, want %d: %#v", label, len(got), len(want), got)
	}
	for i, v := range got {
		if v.RegionHash != want[i] {
			t.Errorf("%s[%d] RegionHash = %q, want %q", label, i, v.RegionHash, want[i])
		}
		if v.File != degenerateIdentityFile {
			t.Errorf("%s[%d] File = %q, want %q", label, i, v.File, degenerateIdentityFile)
		}
	}
}
