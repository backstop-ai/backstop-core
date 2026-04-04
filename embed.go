package backstopcore

import (
	"embed"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

// SchemaFS embeds all artifact schemas from the artifacts/ directory.
// Each CLI binary version constitutes a schema cohort — a locked set
// of schemas that the binary validates against.
//
//go:embed all:artifacts
var SchemaFS embed.FS

// ListSchemas walks the embedded SchemaFS and returns sorted paths for
// all schema files matching artifacts/*/v*/schema.json and
// artifacts/base/schema.json.
func ListSchemas() ([]string, error) {
	var schemas []string
	err := fs.WalkDir(SchemaFS, "artifacts", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Base(path) != "schema.json" {
			return nil
		}

		// Normalize to forward slashes
		normalized := strings.ReplaceAll(path, string(filepath.Separator), "/")

		// Match artifacts/base/schema.json
		if normalized == "artifacts/base/schema.json" {
			schemas = append(schemas, normalized)
			return nil
		}

		// Match artifacts/*/v*/schema.json pattern
		parts := strings.Split(normalized, "/")
		if len(parts) >= 4 && parts[0] == "artifacts" && strings.HasPrefix(parts[2], "v") {
			schemas = append(schemas, normalized)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(schemas)
	return schemas, nil
}
