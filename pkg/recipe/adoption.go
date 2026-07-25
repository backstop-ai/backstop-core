package recipe

import (
	"errors"
	"fmt"
	"os"
	"sort"

	"gopkg.in/yaml.v3"
)

// AdoptionRecordName is the TRACKED project-root record of which recipes a
// project has adopted — the recipe analog of backstop.lock. It is committed, not
// gitignored: it is the durable answer to "which recipe, at which version, is
// adopted here", and the applier reads it to tell its OWN previous output apart
// from a file the consumer wrote.
const AdoptionRecordName = "backstop-recipes.lock"

// AdoptionEntry is one adopted recipe: the thin triple {recipe ref, @version,
// adopted} and NOTHING else (REQ-005).
//
// The thinness is the contract, not an oversight. Per-op and per-region
// provenance, content hashes, and forensic replay are the RICH ledger BUNDLE-017
// owns; adding "just one more" field here couples this capability to a
// downstream bundle and starts a second, divergent ledger. Recipe is the
// VERSION-INDEPENDENT ref (<pack>:<recipe>) so re-applying at a new pin updates
// the entry in place rather than accumulating one row per version — Version
// carries the pin that was actually applied.
type AdoptionEntry struct {
	Recipe  string `yaml:"recipe"`
	Version string `yaml:"version"`
	Adopted string `yaml:"adopted"`
}

// AdoptionRecord is the whole tracked record: adopted recipes keyed by their
// version-independent ref.
type AdoptionRecord struct {
	Recipes map[string]AdoptionEntry `yaml:"recipes"`
}

// ReadAdoptions reads the tracked adoption record.
//
// A MISSING file yields an EMPTY record and no error (the ReadLockfile shape):
// "this project has adopted nothing yet" is a normal state, not a failure, so a
// first apply never has to special-case the record's absence. A file that exists
// but does not parse IS an error — silently treating corruption as "nothing
// adopted" would make the applier mistake its own previous output for consumer
// files.
func ReadAdoptions(path string) (*AdoptionRecord, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &AdoptionRecord{Recipes: make(map[string]AdoptionEntry)}, nil
		}
		return nil, fmt.Errorf("read adoption record %q: %w", path, err)
	}

	var record AdoptionRecord
	if err := yaml.Unmarshal(data, &record); err != nil {
		return nil, fmt.Errorf("parse adoption record %q: %w", path, err)
	}
	if record.Recipes == nil {
		record.Recipes = make(map[string]AdoptionEntry)
	}

	return &record, nil
}

// WriteAdoptions writes the adoption record with SORTED keys, so the tracked file
// is byte-deterministic across runs (the property WriteLockfile has, and the
// reason neither record produces a spurious diff on every apply).
func WriteAdoptions(path string, record *AdoptionRecord) error {
	if record == nil {
		return errors.New("write adoption record: no record was supplied")
	}

	refs := make([]string, 0, len(record.Recipes))
	for ref := range record.Recipes {
		refs = append(refs, ref)
	}
	sort.Strings(refs)

	entries := &yaml.Node{Kind: yaml.MappingNode}
	for _, ref := range refs {
		entries.Content = append(entries.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: ref},
			adoptionEntryNode(record.Recipes[ref]),
		)
	}

	document := &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "recipes"},
			entries,
		},
	}}}

	data, err := yaml.Marshal(document)
	if err != nil {
		return fmt.Errorf("marshal adoption record %q: %w", path, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write adoption record %q: %w", path, err)
	}

	return nil
}

// adoptionEntryNode renders one entry with its three fields in a fixed
// alphabetical order, so the whole document's shape is decided by the data alone.
func adoptionEntryNode(entry AdoptionEntry) *yaml.Node {
	node := &yaml.Node{Kind: yaml.MappingNode}
	for _, pair := range [][2]string{
		{"adopted", entry.Adopted},
		{"recipe", entry.Recipe},
		{"version", entry.Version},
	} {
		node.Content = append(node.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: pair[0]},
			&yaml.Node{Kind: yaml.ScalarNode, Value: pair[1]},
		)
	}

	return node
}
