package backstopcore_test

import (
	"embed"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	backstopcore "github.com/backstop-ai/backstop-core"
)

// TestCLI_EmbedCohort_AllSchemasPresent verifies all artifact schemas from
// artifacts/*/v*/schema.json are accessible from the embedded FS. (CLM-006)
func TestCLI_EmbedCohort_AllSchemasPresent(t *testing.T) {
	schemas, err := backstopcore.ListSchemas()
	if err != nil {
		t.Fatalf("ListSchemas() error: %v", err)
	}
	if len(schemas) == 0 {
		t.Fatal("expected at least one schema in embedded FS, got 0")
	}

	for _, path := range schemas {
		_, err := fs.ReadFile(backstopcore.SchemaFS, path)
		if err != nil {
			t.Errorf("schema %s not readable from embedded FS: %v", path, err)
		}
	}
}

// TestCLI_EmbedCohort_SchemasParseAsJSON reads each embedded schema and
// verifies it parses as valid JSON. (CLM-007)
func TestCLI_EmbedCohort_SchemasParseAsJSON(t *testing.T) {
	schemas, err := backstopcore.ListSchemas()
	if err != nil {
		t.Fatalf("ListSchemas() error: %v", err)
	}

	for _, path := range schemas {
		data, err := fs.ReadFile(backstopcore.SchemaFS, path)
		if err != nil {
			t.Errorf("failed to read %s: %v", path, err)
			continue
		}
		if !json.Valid(data) {
			t.Errorf("schema %s is not valid JSON", path)
		}
	}
}

// TestCLI_EmbedCohort_BaseSchemaPresent verifies artifacts/base/schema.json
// is present in embedded FS. (CLM-008)
func TestCLI_EmbedCohort_BaseSchemaPresent(t *testing.T) {
	data, err := fs.ReadFile(backstopcore.SchemaFS, "artifacts/base/schema.json")
	if err != nil {
		t.Fatalf("base schema not found in embedded FS: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("base schema is empty")
	}
}

// TestCLI_Embed_UsesEmbedFS verifies SchemaFS is of type embed.FS
// (compile-time check via type assertion). (CLM-041)
func TestCLI_Embed_UsesEmbedFS(t *testing.T) {
	// Compile-time type assertion: SchemaFS must be embed.FS
	var _ embed.FS = backstopcore.SchemaFS
	// If this compiles, the claim is satisfied.
	// Also verify it implements fs.FS at runtime.
	var _ fs.FS = backstopcore.SchemaFS
}

// TestCLI_Embed_ListSchemaPaths verifies ListSchemas() returns all expected
// schema file paths. (CLM-042)
func TestCLI_Embed_ListSchemaPaths(t *testing.T) {
	schemas, err := backstopcore.ListSchemas()
	if err != nil {
		t.Fatalf("ListSchemas() error: %v", err)
	}

	// Must include base schema
	foundBase := false
	for _, s := range schemas {
		if s == "artifacts/base/schema.json" {
			foundBase = true
		}
		// Each path must match expected pattern
		if !strings.HasPrefix(s, "artifacts/") || !strings.HasSuffix(s, "schema.json") {
			t.Errorf("unexpected schema path format: %s", s)
		}
	}
	if !foundBase {
		t.Error("ListSchemas() did not include artifacts/base/schema.json")
	}

	// Must include at least one versioned schema (artifacts/*/v*/schema.json)
	foundVersioned := false
	for _, s := range schemas {
		if s != "artifacts/base/schema.json" && strings.Contains(s, "/v") {
			foundVersioned = true
			break
		}
	}
	if !foundVersioned {
		t.Error("ListSchemas() did not include any versioned schemas (artifacts/*/v*/schema.json)")
	}
}

// TestCLI_Embed_MatchesDiskSchemas compares embedded schema paths against
// actual disk files to ensure parity. (CLM-043)
func TestCLI_Embed_MatchesDiskSchemas(t *testing.T) {
	// Collect schemas from disk
	var diskSchemas []string
	root := "artifacts"
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Base(path) == "schema.json" {
			// Normalize to forward slashes for comparison
			diskSchemas = append(diskSchemas, filepath.ToSlash(path))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking disk artifacts: %v", err)
	}
	sort.Strings(diskSchemas)

	// Collect schemas from embed
	embeddedSchemas, err := backstopcore.ListSchemas()
	if err != nil {
		t.Fatalf("ListSchemas() error: %v", err)
	}
	sort.Strings(embeddedSchemas)

	// Compare
	if len(diskSchemas) != len(embeddedSchemas) {
		t.Fatalf("disk schemas (%d) != embedded schemas (%d)\ndisk: %v\nembed: %v",
			len(diskSchemas), len(embeddedSchemas), diskSchemas, embeddedSchemas)
	}

	for i := range diskSchemas {
		if diskSchemas[i] != embeddedSchemas[i] {
			t.Errorf("mismatch at index %d: disk=%s embedded=%s", i, diskSchemas[i], embeddedSchemas[i])
		}
	}
}
