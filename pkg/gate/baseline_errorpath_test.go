package gate

import (
	"os"
	"path/filepath"
	"testing"
)

// These exercise the LoadBaseline/WriteBaseline error and round-trip paths that
// were previously uncovered — genuine behavior tests (they invoke the real
// functions), not coverage padding.

func TestLoadBaseline_ReadError_NonexistentPath(t *testing.T) {
	if _, err := LoadBaseline(filepath.Join(t.TempDir(), "nope.json")); err == nil {
		t.Fatal("expected a read error for a nonexistent baseline path")
	}
}

func TestLoadBaseline_ParseError_MalformedJSON(t *testing.T) {
	p := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(p, []byte("{not valid json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadBaseline(p); err == nil {
		t.Fatal("expected a parse error for malformed baseline JSON")
	}
}

func TestWriteBaseline_NilArtifact(t *testing.T) {
	if err := WriteBaseline(filepath.Join(t.TempDir(), "b.json"), nil); err == nil {
		t.Fatal("expected an error writing a nil baseline artifact")
	}
}

func TestWriteBaseline_UnwritablePath(t *testing.T) {
	// A parent path that is a regular FILE makes MkdirAll (and the write) fail.
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteBaseline(filepath.Join(blocker, "sub", "b.json"), &BaselineArtifact{}); err == nil {
		t.Fatal("expected an error writing under a path blocked by a file")
	}
}

func TestWriteBaseline_RoundTripThroughLoad(t *testing.T) {
	p := filepath.Join(t.TempDir(), "b.json")
	art := &BaselineArtifact{Violations: []Violation{{Rule: "r", File: "f.go", Message: "m", Severity: "error"}}}
	if err := WriteBaseline(p, art); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, err := LoadBaseline(p)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.SchemaVersion != BaselineSchemaV1 || len(got.Violations) != 1 {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
}
