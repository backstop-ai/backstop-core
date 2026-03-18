package artifact_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bmanson/backstop-core/pkg/artifact"
)

const validADR = `# ADR-0001: Backstop Is an Agent-First Discipline Framework

**Number:** ADR-0001
**Created:** 2026-03-17
**Status:** Accepted
**Deciders:** @bmanson
**Decisions:** D-052, D-024
**Schema-Version:** adr/v1

---

## Context

AI coding agents need discipline.

## Decision

We build backstop.

## Consequences

Everything changes.

## Alternatives Considered

Do nothing.

## References

None.
`

func TestParseFile_ValidADR(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ADR-0001-agent-first.adr.md")
	if err := os.WriteFile(path, []byte(validADR), 0644); err != nil {
		t.Fatal(err)
	}

	art, err := artifact.ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if art.Filename != "ADR-0001-agent-first.adr.md" {
		t.Errorf("Filename = %q, want %q", art.Filename, "ADR-0001-agent-first.adr.md")
	}
	if art.Title != "ADR-0001: Backstop Is an Agent-First Discipline Framework" {
		t.Errorf("Title = %q", art.Title)
	}
	if len(art.Metadata) != 6 {
		t.Errorf("len(Metadata) = %d, want 6", len(art.Metadata))
	}
	if art.Metadata["Number"] != "ADR-0001" {
		t.Errorf("Metadata[Number] = %q", art.Metadata["Number"])
	}
	if art.Metadata["Schema-Version"] != "adr/v1" {
		t.Errorf("Metadata[Schema-Version] = %q", art.Metadata["Schema-Version"])
	}
	if len(art.Sections) != 5 {
		t.Errorf("len(Sections) = %d, want 5; got %v", len(art.Sections), art.Sections)
	}
}

func TestParse_ExtractsTitle(t *testing.T) {
	art, err := artifact.Parse("# My Title\n---\n", "test.md")
	if err != nil {
		t.Fatal(err)
	}
	if art.Title != "My Title" {
		t.Errorf("Title = %q, want %q", art.Title, "My Title")
	}
}

func TestParse_ExtractsMetadata(t *testing.T) {
	input := `# Title

**Number:** ADR-0001
**Created:** 2026-03-17
**Status:** Accepted
**Schema-Version:** adr/v1

---
`
	art, err := artifact.Parse(input, "test.md")
	if err != nil {
		t.Fatal(err)
	}

	for _, key := range []string{"Number", "Created", "Status", "Schema-Version"} {
		if _, ok := art.Metadata[key]; !ok {
			t.Errorf("Metadata missing key %q", key)
		}
	}
	if art.Metadata["Number"] != "ADR-0001" {
		t.Errorf("Metadata[Number] = %q, want %q", art.Metadata["Number"], "ADR-0001")
	}
	if art.Metadata["Schema-Version"] != "adr/v1" {
		t.Errorf("Metadata[Schema-Version] = %q, want %q", art.Metadata["Schema-Version"], "adr/v1")
	}
}

func TestParse_ExtractsSections(t *testing.T) {
	input := "# Title\n---\n## Context\nSome text.\n## Decision\nMore text.\n"
	art, err := artifact.Parse(input, "test.md")
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"Context", "Decision"}
	if len(art.Sections) != len(want) {
		t.Fatalf("Sections = %v, want %v", art.Sections, want)
	}
	for i, s := range want {
		if art.Sections[i] != s {
			t.Errorf("Sections[%d] = %q, want %q", i, art.Sections[i], s)
		}
	}
}

func TestParse_NoMetadata(t *testing.T) {
	art, err := artifact.Parse("# Title\n---\n## Context\n", "test.md")
	if err != nil {
		t.Fatal(err)
	}
	if art.Metadata == nil {
		t.Error("Metadata is nil, want empty map")
	}
	if len(art.Metadata) != 0 {
		t.Errorf("len(Metadata) = %d, want 0", len(art.Metadata))
	}
}

func TestParse_StripsDirectory(t *testing.T) {
	dir := t.TempDir()
	deep := filepath.Join(dir, "some", "deep", "path")
	if err := os.MkdirAll(deep, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(deep, "ADR-0001-test.adr.md")
	if err := os.WriteFile(path, []byte("# Title\n---\n"), 0644); err != nil {
		t.Fatal(err)
	}

	art, err := artifact.ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if art.Filename != "ADR-0001-test.adr.md" {
		t.Errorf("Filename = %q, want %q", art.Filename, "ADR-0001-test.adr.md")
	}
}

func TestParse_NoSections(t *testing.T) {
	art, err := artifact.Parse("# Title\n**Key:** Val\n---\n", "test.md")
	if err != nil {
		t.Fatal(err)
	}
	if art.Sections == nil {
		t.Error("Sections is nil, want empty slice")
	}
	if len(art.Sections) != 0 {
		t.Errorf("len(Sections) = %d, want 0", len(art.Sections))
	}
}

func TestParse_BoldWithoutColon(t *testing.T) {
	input := "# Title\n**Note**\n**Number:** ADR-0001\n---\n"
	art, err := artifact.Parse(input, "test.md")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := art.Metadata["Note"]; ok {
		t.Error("Metadata contains 'Note' — bold without colon should be ignored")
	}
	if art.Metadata["Number"] != "ADR-0001" {
		t.Errorf("Metadata[Number] = %q, want %q", art.Metadata["Number"], "ADR-0001")
	}
}
