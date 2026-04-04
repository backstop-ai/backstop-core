package main

import (
	"embed"

	backstopcore "github.com/bmanson/backstop-core"
)

// SchemaFS re-exports the root-level embedded schema filesystem.
// The go:embed directive lives at the module root (embed.go) because
// go:embed cannot reference files above the package directory.
// This shim satisfies the SPEC-005 contract for cmd/backstop/embed.go.
var SchemaFS embed.FS = backstopcore.SchemaFS

// ListSchemas re-exports the root-level ListSchemas function.
// Returns paths of all embedded schema files for cohort introspection.
func ListSchemas() ([]string, error) {
	return backstopcore.ListSchemas()
}
