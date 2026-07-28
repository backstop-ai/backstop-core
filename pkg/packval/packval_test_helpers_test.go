package packval_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/packval"
)

func writeFile(t *testing.T, root, rel, data string) {
	t.Helper()
	full := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

func baseManifest() *packval.PackManifest {
	return &packval.PackManifest{
		Name:      "acme/example",
		Version:   "1.0.0",
		Language:  "go",
		Archetype: "code",
		Content: packval.Content{
			Ruleset: packval.Ruleset{
				Rules: []packval.Rule{
					{
						ID:         "R1",
						Engine:     "semgrep",
						File:       "rules/r1.yml",
						RiskClass:  "security",
						Layer:      3,
						Category:   "presence",
						InputScope: "single-file",
						Validator:  "validators/v.sh",
						Claims: []packval.Claim{
							{
								ID: "C1",
								Fixtures: packval.Fixtures{
									Positive: []packval.FixtureRef{{Path: "fixtures/p.go", BypassAttempt: true}},
									Negative: []packval.FixtureRef{{Path: "fixtures/n.go"}},
								},
							},
						},
					},
				},
			},
		},
	}
}

func makePackDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, dir, "rules/r1.yml", "rules:\n  - id: R1\n    pattern: x\n")
	writeFile(t, dir, "fixtures/p.go", "package p")
	writeFile(t, dir, "fixtures/n.go", "package p")
	writeFile(t, dir, "validators/v.sh", "#!/bin/sh\nexit 0\n")
	writeFile(t, dir, "pack.yml", strings.TrimSpace(`
name: acme/example
version: 1.0.0
language: go
archetype: code
content:
  ruleset:
    rules:
      - id: R1
        engine: semgrep
        file: rules/r1.yml
        risk_class: security
        layer: 3
        category: presence
        input_scope: single-file
        validator: validators/v.sh
        claims:
          - id: C1
            fixtures:
              positive:
                - path: fixtures/p.go
                  bypass_attempt: true
              negative:
                - fixtures/n.go
`))
	return dir
}
