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
	// SourceCoordinate is the git-source pack's requested `org/repository` reference,
	// recorded EXACTLY as the operator wrote it with only the `@version` suffix removed:
	// no case folding, no suffix stripping, no host-specific normalization (SPEC-056
	// REQ-004). Case-insensitivity is a GitHub property and packs may be hosted
	// anywhere, so normalizing here would bake in the host assumption DD-31 removed.
	//
	// It is empty for local-source packs, whose source is already recorded by LocalPath.
	// Like LocalPath it is PROVENANCE AND RESOLUTION INPUT — it is NOT part of
	// ComputeContentHash, because where a pack came from is not pack content and folding
	// it in would make every existing lock entry's hash unreproducible.
	//
	// It exists because REQ-003 keys the lock by MANIFEST name: without a recorded
	// coordinate, a pack whose name differs from its repository becomes uninstallable
	// from its own lock the moment that landed.
	SourceCoordinate string `yaml:"source_coordinate,omitempty"`
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

	// source_coordinate is alphabetically between name and source_type; emit only when
	// set, exactly as local_path is guarded above. The guard is LOAD-BEARING, not
	// cosmetic: this node is built by hand rather than marshalled from the struct, so
	// the `omitempty` tag alone does nothing here, and without it every pre-existing
	// entry would gain a blank source_coordinate key on its first rewrite — a diff in
	// every consumer's tracked backstop.lock for a field they have no value for.
	if entry.SourceCoordinate != "" {
		addScalarPair(node, "source_coordinate", entry.SourceCoordinate)
	}

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
