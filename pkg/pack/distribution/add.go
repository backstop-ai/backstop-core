package distribution

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// GitCloner abstracts git clone operations for testability.
type GitCloner interface {
	Clone(url, version, destDir string) error
	ListTags(url string) ([]string, error)
}

// Validator abstracts pack check and pack test operations for testability.
type Validator interface {
	RunPackCheck(packDir string) error
	RunPackTest(packDir string) error
}

// GitError represents a git operation error.
type GitError struct {
	Message string
}

func (e *GitError) Error() string { return e.Message }

// ValidationError represents a validation failure.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

// AddOptions configures the pack add command.
type AddOptions struct {
	ProjectDir string
	Version    string
	GitCloner  GitCloner
	Validator  Validator
}

// AddResult holds the result of a successful pack add operation.
type AddResult struct {
	PackName      string `json:"pack_name"`
	Version       string `json:"version"`
	ContentHash   string `json:"content_hash"`
	InstalledPath string `json:"installed_path"`
}

// backstopYml represents backstop.yml preserving all fields during read-modify-write.
// Packs is a map of ref → version, matching pkg/config.Packs format.
type backstopYml struct {
	Project  string            `yaml:"project,omitempty"`
	Language string            `yaml:"language,omitempty"`
	Packs    map[string]string `yaml:"packs"`
	// Catch-all for fields we don't explicitly model.
	Extra map[string]interface{} `yaml:",inline"`
}

// Add implements the pack add pipeline: resolve, clone, validate, install,
// merge config, record provenance, update manifest and lockfile.
func Add(packRef string, opts AddOptions) (*AddResult, error) {
	isLocal := isLocalPath(packRef)
	var packName, version, packDir, sourceType string
	var gitRef *string

	if isLocal {
		// Local path pack.
		absPath, err := filepath.Abs(packRef)
		if err != nil {
			return nil, fmt.Errorf("resolving local path: %w", err)
		}

		if _, statErr := os.Stat(filepath.Join(absPath, "pack.yml")); statErr != nil {
			return nil, fmt.Errorf("local path %s does not contain pack.yml", absPath)
		}

		manifest, readErr := readPackManifest(absPath)
		if readErr != nil {
			return nil, fmt.Errorf("reading local pack manifest: %w", readErr)
		}

		// Extract name from manifest.
		nameData, _ := os.ReadFile(filepath.Join(absPath, "pack.yml"))
		var nameMap map[string]interface{}
		if err := yaml.Unmarshal(nameData, &nameMap); err == nil {
			if n, ok := nameMap["name"].(string); ok {
				packName = n
			}
		}
		if packName == "" {
			return nil, fmt.Errorf("local pack manifest missing name")
		}

		_ = manifest
		packDir = absPath
		sourceType = "local"
	} else {
		// Git pack: parse org/pack-name@version.
		packName, version = parsePackRef(packRef)
		if opts.Version != "" {
			version = opts.Version
		}
		sourceType = "git"
		ref := "v" + version
		gitRef = &ref
	}

	// Check if already installed.
	if isPackInstalled(opts.ProjectDir, packName) {
		return nil, fmt.Errorf("pack %s is already installed; use pack update or pack upgrade instead", packName)
	}

	if !isLocal {
		// Clone the pack to a temporary directory.
		tmpDir, err := os.MkdirTemp("", "backstop-pack-clone-*")
		if err != nil {
			return nil, fmt.Errorf("creating temp dir: %w", err)
		}
		defer os.RemoveAll(tmpDir)

		gitURL := resolveGitURL(packName)
		if err := opts.GitCloner.Clone(gitURL, "v"+version, tmpDir); err != nil {
			return nil, err
		}
		packDir = tmpDir
	}

	// Validate: run pack check and pack test.
	if opts.Validator != nil {
		if err := opts.Validator.RunPackCheck(packDir); err != nil {
			return nil, err
		}
		if err := opts.Validator.RunPackTest(packDir); err != nil {
			return nil, err
		}
	}

	// Copy pack to .backstop/packs/org/pack-name/ (both git and local).
	installedPath := filepath.Join(opts.ProjectDir, ".backstop", "packs", packName)
	if err := os.MkdirAll(filepath.Dir(installedPath), 0o755); err != nil {
		return nil, fmt.Errorf("creating packs dir: %w", err)
	}

	if err := copyDirRecursive(packDir, installedPath); err != nil {
		return nil, fmt.Errorf("copying pack: %w", err)
	}

	// Compute content hash from the installed copy.
	contentHash, err := ComputeContentHash(installedPath)
	if err != nil {
		os.RemoveAll(installedPath)
		return nil, fmt.Errorf("computing content hash: %w", err)
	}

	// Snapshot files for transactional rollback.
	ymlPath := filepath.Join(opts.ProjectDir, "backstop.yml")
	lockPath := filepath.Join(opts.ProjectDir, "backstop.lock")
	backstopDir := filepath.Join(opts.ProjectDir, ".backstop")
	if err := os.MkdirAll(backstopDir, 0o755); err != nil {
		return nil, err
	}
	provPath := filepath.Join(backstopDir, "pack-config-provenance.json")

	ymlSnap, _ := os.ReadFile(ymlPath)
	lockSnap, _ := os.ReadFile(lockPath)
	provSnap, _ := os.ReadFile(provPath)

	rollback := func() {
		os.RemoveAll(installedPath)
		if ymlSnap != nil {
			os.WriteFile(ymlPath, ymlSnap, 0o644)
		}
		if lockSnap != nil {
			os.WriteFile(lockPath, lockSnap, 0o644)
		} else {
			os.Remove(lockPath)
		}
		if provSnap != nil {
			os.WriteFile(provPath, provSnap, 0o644)
		} else {
			os.Remove(provPath)
		}
	}

	// Merge tool_config.
	prov, err := ReadProvenance(provPath)
	if err != nil {
		rollback()
		return nil, err
	}

	mergeResult, err := MergeToolConfig(packDir, opts.ProjectDir, prov)
	if err != nil {
		rollback()
		return nil, fmt.Errorf("merging tool_config: %w", err)
	}

	if len(mergeResult.Conflicts) > 0 {
		rollback()
		var msgs []string
		for _, c := range mergeResult.Conflicts {
			msgs = append(msgs, fmt.Sprintf("%s: %s (pack=%s, current=%s)", c.ConfigFile, c.SettingKey, c.PackValue, c.CurrentValue))
		}
		return nil, fmt.Errorf("tool_config conflicts:\n%s", strings.Join(msgs, "\n"))
	}

	// Record provenance.
	for i := range mergeResult.Merged {
		mergeResult.Merged[i].SourcePack = packName
	}
	prov.Entries = append(prov.Entries, mergeResult.Merged...)
	if err := WriteProvenance(provPath, prov); err != nil {
		rollback()
		return nil, err
	}

	// Update backstop.yml.
	if err := updateBackstopYml(opts.ProjectDir, packName, version, isLocal, packDir); err != nil {
		rollback()
		return nil, err
	}

	// Update backstop.lock.
	lf, _ := ReadLockfile(lockPath)
	if lf == nil {
		lf = &Lockfile{Packs: make(map[string]LockEntry)}
	}

	lf.Packs[packName] = LockEntry{
		Name:        packName,
		Version:     version,
		GitRef:      gitRef,
		ContentHash: contentHash,
		SourceType:  sourceType,
		InstallDate: time.Now().UTC().Format(time.RFC3339),
	}

	if err := WriteLockfile(lockPath, lf); err != nil {
		rollback()
		return nil, err
	}

	// Ensure .backstop/packs/ in .gitignore.
	if err := ensureGitignore(opts.ProjectDir); err != nil {
		return nil, err
	}

	return &AddResult{
		PackName:      packName,
		Version:       version,
		ContentHash:   contentHash,
		InstalledPath: installedPath,
	}, nil
}

// parsePackRef parses "org/pack-name@version" into name and version.
func parsePackRef(ref string) (string, string) {
	parts := strings.SplitN(ref, "@", 2)
	name := parts[0]
	version := ""
	if len(parts) > 1 {
		version = parts[1]
	}
	return name, version
}

// isLocalPath determines if a pack reference is a local filesystem path.
func isLocalPath(ref string) bool {
	if strings.HasPrefix(ref, "/") || strings.HasPrefix(ref, "./") || strings.HasPrefix(ref, "../") {
		return true
	}
	// Check if it's an absolute path on any OS.
	if filepath.IsAbs(ref) {
		return true
	}
	return false
}

// resolveGitURL constructs a git URL from an org/pack-name reference.
func resolveGitURL(packName string) string {
	return "https://github.com/" + packName + ".git"
}

// isPackInstalled checks if a pack is already listed in backstop.yml.
func isPackInstalled(projectDir, packName string) bool {
	data, err := os.ReadFile(filepath.Join(projectDir, "backstop.yml"))
	if err != nil {
		return false
	}

	var yml backstopYml
	if err := yaml.Unmarshal(data, &yml); err != nil {
		return false
	}

	_, exists := yml.Packs[packName]
	return exists
}

// updateBackstopYml adds a pack entry to backstop.yml.
func updateBackstopYml(projectDir, packName, version string, isLocal bool, localPath string) error {
	path := filepath.Join(projectDir, "backstop.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var yml backstopYml
	if err := yaml.Unmarshal(data, &yml); err != nil {
		return err
	}

	if yml.Packs == nil {
		yml.Packs = make(map[string]string)
	}

	if isLocal {
		yml.Packs[packName] = "local"
	} else {
		yml.Packs[packName] = version
	}

	out, err := yaml.Marshal(&yml)
	if err != nil {
		return err
	}

	return os.WriteFile(path, out, 0o644)
}

// ensureGitignore ensures .backstop/packs/ is listed in .gitignore.
func ensureGitignore(projectDir string) error {
	gitignorePath := filepath.Join(projectDir, ".gitignore")
	entry := ".backstop/packs/"

	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		if os.IsNotExist(err) {
			return os.WriteFile(gitignorePath, []byte(entry+"\n"), 0o644)
		}
		return err
	}

	content := string(data)
	if strings.Contains(content, entry) {
		return nil
	}

	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += entry + "\n"

	return os.WriteFile(gitignorePath, []byte(content), 0o644)
}

// copyDirRecursive recursively copies a directory.
func copyDirRecursive(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, relErr := filepath.Rel(src, path)
		if relErr != nil {
			return relErr
		}

		target := filepath.Join(dst, rel)

		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}

		return os.WriteFile(target, data, info.Mode())
	})
}
