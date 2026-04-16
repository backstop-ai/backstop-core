package distribution

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

// Provenance tracks tool_config settings contributed by packs.
type Provenance struct {
	Entries []ProvenanceEntry `json:"entries"`
}

// ProvenanceEntry records a single tool_config setting contributed by a pack.
type ProvenanceEntry struct {
	ConfigFile string `json:"config_file"`
	SettingKey  string `json:"setting_key"`
	SourcePack string `json:"source_pack"`
	ValueHash  string `json:"value_hash"`
}

// ReadProvenance reads a provenance JSON file. Returns an empty Provenance
// (not an error) if the file does not exist, supporting first-time use.
func ReadProvenance(path string) (*Provenance, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Provenance{Entries: []ProvenanceEntry{}}, nil
		}
		return nil, fmt.Errorf("reading provenance %s: %w", path, err)
	}

	var prov Provenance
	if err := json.Unmarshal(data, &prov); err != nil {
		return nil, fmt.Errorf("parsing provenance %s: %w", path, err)
	}

	if prov.Entries == nil {
		prov.Entries = []ProvenanceEntry{}
	}

	return &prov, nil
}

// WriteProvenance writes a provenance file as JSON with an "entries" array.
func WriteProvenance(path string, prov *Provenance) error {
	if prov.Entries == nil {
		prov.Entries = []ProvenanceEntry{}
	}

	data, err := json.MarshalIndent(prov, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling provenance: %w", err)
	}

	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("writing provenance %s: %w", path, err)
	}

	return nil
}
