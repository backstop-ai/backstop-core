package distribution_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack/distribution"
)

func TestProvenance_TracksAllFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "provenance.json")

	prov := &distribution.Provenance{
		Entries: []distribution.ProvenanceEntry{
			{
				ConfigFile: ".golangci.yml",
				SettingKey: "linters.enable.revive",
				SourcePack: "acme/go-http-standards",
				ValueHash:  "sha256:abc123",
			},
		},
	}

	if err := distribution.WriteProvenance(path, prov); err != nil {
		t.Fatalf("WriteProvenance: %v", err)
	}

	read, err := distribution.ReadProvenance(path)
	if err != nil {
		t.Fatalf("ReadProvenance: %v", err)
	}

	if len(read.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(read.Entries))
	}

	entry := read.Entries[0]
	if entry.ConfigFile != ".golangci.yml" {
		t.Errorf("ConfigFile = %q, want %q", entry.ConfigFile, ".golangci.yml")
	}
	if entry.SettingKey != "linters.enable.revive" {
		t.Errorf("SettingKey = %q, want %q", entry.SettingKey, "linters.enable.revive")
	}
	if entry.SourcePack != "acme/go-http-standards" {
		t.Errorf("SourcePack = %q, want %q", entry.SourcePack, "acme/go-http-standards")
	}
	if entry.ValueHash != "sha256:abc123" {
		t.Errorf("ValueHash = %q, want %q", entry.ValueHash, "sha256:abc123")
	}
}

func TestProvenance_CommittedToRepo(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".backstop", "pack-config-provenance.json")

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}

	prov := &distribution.Provenance{
		Entries: []distribution.ProvenanceEntry{
			{
				ConfigFile: ".golangci.yml",
				SettingKey: "linters.enable.revive",
				SourcePack: "acme/go-http-standards",
				ValueHash:  "sha256:abc123",
			},
		},
	}

	if err := distribution.WriteProvenance(path, prov); err != nil {
		t.Fatalf("WriteProvenance: %v", err)
	}

	// Verify file exists at the expected path.
	if _, err := os.Stat(path); err != nil {
		t.Errorf("provenance file not written to %s: %v", path, err)
	}
}

func TestReadProvenance_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "provenance.json")

	original := &distribution.Provenance{
		Entries: []distribution.ProvenanceEntry{
			{
				ConfigFile: ".eslintrc.json",
				SettingKey: "rules.no-console",
				SourcePack: "acme/js-standards",
				ValueHash:  "sha256:def456",
			},
			{
				ConfigFile: ".golangci.yml",
				SettingKey: "linters.enable.errcheck",
				SourcePack: "acme/go-standards",
				ValueHash:  "sha256:ghi789",
			},
		},
	}

	if err := distribution.WriteProvenance(path, original); err != nil {
		t.Fatalf("WriteProvenance: %v", err)
	}

	read, err := distribution.ReadProvenance(path)
	if err != nil {
		t.Fatalf("ReadProvenance: %v", err)
	}

	if len(read.Entries) != len(original.Entries) {
		t.Fatalf("Entries count = %d, want %d", len(read.Entries), len(original.Entries))
	}

	for i, entry := range read.Entries {
		orig := original.Entries[i]
		if entry.ConfigFile != orig.ConfigFile {
			t.Errorf("entry[%d].ConfigFile = %q, want %q", i, entry.ConfigFile, orig.ConfigFile)
		}
		if entry.SettingKey != orig.SettingKey {
			t.Errorf("entry[%d].SettingKey = %q, want %q", i, entry.SettingKey, orig.SettingKey)
		}
		if entry.SourcePack != orig.SourcePack {
			t.Errorf("entry[%d].SourcePack = %q, want %q", i, entry.SourcePack, orig.SourcePack)
		}
		if entry.ValueHash != orig.ValueHash {
			t.Errorf("entry[%d].ValueHash = %q, want %q", i, entry.ValueHash, orig.ValueHash)
		}
	}
}

func TestReadProvenance_MissingFile(t *testing.T) {
	prov, err := distribution.ReadProvenance("/nonexistent/provenance.json")
	if err != nil {
		t.Fatalf("ReadProvenance should not error on missing file: %v", err)
	}

	if prov == nil {
		t.Fatal("expected non-nil Provenance for missing file")
	}
	if len(prov.Entries) != 0 {
		t.Errorf("expected empty entries for missing file, got %d", len(prov.Entries))
	}
}

func TestWriteProvenance_JSONFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "provenance.json")

	prov := &distribution.Provenance{
		Entries: []distribution.ProvenanceEntry{
			{
				ConfigFile: ".golangci.yml",
				SettingKey: "linters.enable.revive",
				SourcePack: "acme/pack",
				ValueHash:  "sha256:abc",
			},
		},
	}

	if err := distribution.WriteProvenance(path, prov); err != nil {
		t.Fatalf("WriteProvenance: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading provenance: %v", err)
	}

	// Verify valid JSON.
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("invalid JSON: %v\ncontent: %s", err, string(data))
	}

	// Verify "entries" array exists.
	entries, ok := parsed["entries"]
	if !ok {
		t.Fatalf("missing 'entries' key in JSON:\n%s", string(data))
	}

	arr, ok := entries.([]interface{})
	if !ok {
		t.Fatalf("'entries' is not an array in JSON:\n%s", string(data))
	}

	if len(arr) != 1 {
		t.Errorf("entries array length = %d, want 1", len(arr))
	}
}
