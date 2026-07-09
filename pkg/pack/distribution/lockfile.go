package distribution

import (
	"fmt"
	"os"
	"sort"

	"gopkg.in/yaml.v3"
)

// Lockfile represents the backstop.lock file containing pinned pack versions.
type Lockfile struct {
	Packs map[string]LockEntry `yaml:"packs"`
}

// LockEntry represents a single pack entry in the lockfile.
type LockEntry struct {
	Name        string  `yaml:"name"`
	Version     string  `yaml:"version,omitempty"`
	GitRef      *string `yaml:"git_ref"`
	ContentHash string  `yaml:"content_hash"`
	SourceType  string  `yaml:"source_type"`
	InstallDate string  `yaml:"install_date"`
	// LocalPath is the local-source pack's directory RELATIVE TO THE PROJECT ROOT,
	// recorded at add time so a later install can re-materialize the pack from a durable,
	// portable record. Empty for git-source packs. It is provenance only — it is NOT part
	// of ComputeContentHash (a source path is not pack content).
	LocalPath string `yaml:"local_path,omitempty"`
}

// ReadLockfile reads and parses a backstop.lock YAML file.
func ReadLockfile(path string) (*Lockfile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading lockfile %s: %w", path, err)
	}

	var lf Lockfile
	if err := yaml.Unmarshal(data, &lf); err != nil {
		return nil, fmt.Errorf("parsing lockfile %s: %w", path, err)
	}

	if lf.Packs == nil {
		lf.Packs = make(map[string]LockEntry)
	}

	return &lf, nil
}

// WriteLockfile writes a lockfile as YAML with sorted keys at all levels.
func WriteLockfile(path string, lockfile *Lockfile) error {
	// Build a sorted YAML document manually for deterministic output.
	doc := &yaml.Node{
		Kind: yaml.DocumentNode,
	}

	packsMapNode := &yaml.Node{
		Kind: yaml.MappingNode,
	}

	// Sort pack names for deterministic output.
	names := make([]string, 0, len(lockfile.Packs))
	for name := range lockfile.Packs {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		entry := lockfile.Packs[name]

		keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: name}
		valueNode := buildSortedLockEntryNode(entry)

		packsMapNode.Content = append(packsMapNode.Content, keyNode, valueNode)
	}

	rootMap := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "packs"},
			packsMapNode,
		},
	}

	doc.Content = append(doc.Content, rootMap)

	data, err := yaml.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshaling lockfile: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing lockfile %s: %w", path, err)
	}

	return nil
}

// buildSortedLockEntryNode creates a YAML mapping node with sorted keys for a LockEntry.
func buildSortedLockEntryNode(entry LockEntry) *yaml.Node {
	node := &yaml.Node{Kind: yaml.MappingNode}

	// Fields in alphabetical order for sorted YAML keys.
	addScalarPair(node, "content_hash", entry.ContentHash)

	if entry.GitRef == nil {
		node.Content = append(node.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "git_ref"},
			&yaml.Node{Kind: yaml.ScalarNode, Value: "null", Tag: "!!null"},
		)
	} else {
		addScalarPair(node, "git_ref", *entry.GitRef)
	}

	addScalarPair(node, "install_date", entry.InstallDate)

	// local_path is alphabetically between install_date and name; emit only when set
	// (omitempty), so pre-existing path-less local entries parse and round-trip cleanly.
	if entry.LocalPath != "" {
		addScalarPair(node, "local_path", entry.LocalPath)
	}

	addScalarPair(node, "name", entry.Name)
	addScalarPair(node, "source_type", entry.SourceType)

	if entry.Version != "" {
		addScalarPair(node, "version", entry.Version)
	} else {
		node.Content = append(node.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "version"},
			&yaml.Node{Kind: yaml.ScalarNode, Value: "null", Tag: "!!null"},
		)
	}

	return node
}

// addScalarPair adds a key-value scalar pair to a mapping node.
func addScalarPair(node *yaml.Node, key, value string) {
	node.Content = append(node.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: key},
		&yaml.Node{Kind: yaml.ScalarNode, Value: value},
	)
}
