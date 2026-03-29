package validate_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bmanson/backstop-core/pkg/artifact"
	"github.com/bmanson/backstop-core/pkg/schema"
	"github.com/bmanson/backstop-core/pkg/validate"
)

// repoRoot walks up from the current working directory to find go.mod.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find repo root (no go.mod found)")
		}
		dir = parent
	}
}

func TestADR_ExistingADRs(t *testing.T) {
	root := repoRoot(t)
	adrsDir := filepath.Join(root, "adrs")
	artifactsRoot := filepath.Join(root, "artifacts")

	entries, err := filepath.Glob(filepath.Join(adrsDir, "*.adr.md"))
	if err != nil {
		t.Fatal(err)
	}

	if len(entries) != 18 {
		t.Fatalf("expected 18 ADR files, found %d", len(entries))
	}

	for _, path := range entries {
		filename := filepath.Base(path)
		t.Run(filename, func(t *testing.T) {
			// Parse
			art, err := artifact.ParseFile(path)
			if err != nil {
				t.Fatalf("ParseFile: %v", err)
			}

			// Resolve schema path
			schemaRelPath, err := schema.ResolveSchemaPath(art)
			if err != nil {
				t.Fatalf("ResolveSchemaPath: %v", err)
			}

			// Load schema
			schemaFullPath := filepath.Join(root, schemaRelPath)
			sch, err := schema.LoadArtifactSchema(schemaFullPath, artifactsRoot)
			if err != nil {
				t.Fatalf("LoadArtifactSchema: %v", err)
			}

			// Validate
			result := validate.ADR(art, sch)
			if !result.Pass() {
				for _, v := range result.Violations {
					t.Errorf("  [%s] %s: %s", v.Severity, v.Rule, v.Message)
				}
			}
		})
	}
}
