package main

import (
	"slices"
	"strings"
	"testing"

	backstopcore "github.com/backstop-ai/backstop-core"
)

// TestCLI_EmbedShim_ListSchemasMatchesRoot verifies the cmd/backstop
// ListSchemas re-export returns exactly the module root's schema paths.
func TestCLI_EmbedShim_ListSchemasMatchesRoot(t *testing.T) {
	got, err := ListSchemas()
	if err != nil {
		t.Fatalf("ListSchemas: %v", err)
	}
	want, err := backstopcore.ListSchemas()
	if err != nil {
		t.Fatalf("backstopcore.ListSchemas: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("ListSchemas returned no schema paths")
	}
	if !slices.Equal(got, want) {
		t.Fatalf("shim ListSchemas differs from the root:\ngot:  %v\nwant: %v", got, want)
	}
	for _, p := range got {
		if !strings.HasSuffix(p, "schema.json") {
			t.Errorf("ListSchemas returned a non-schema path %q", p)
		}
	}
}
