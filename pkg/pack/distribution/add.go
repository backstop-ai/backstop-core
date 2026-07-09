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
	// AlreadyCurrent is true when the pack was already genuinely installed (materialized
	// on disk AND recorded in the lock), so Add was an honest no-op rather than a
	// re-install. It lets the CLI print a clear "already installed and up to date" message
	// instead of a misleading error or a silent zero-output exit.
	AlreadyCurrent bool `json:"already_current,omitempty"`
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
	// localRelPath holds the local pack's source dir RELATIVE TO THE PROJECT ROOT, recorded
	// in the lock so install can re-materialize it portably (empty for git-source packs).
	var localRelPath string

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
		nameData := readFileOrNil(filepath.Join(absPath, "pack.yml"))
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

		// Record the source path RELATIVE TO THE PROJECT ROOT (never absolute): backstop.lock
		// is tracked/committed, so a machine-specific absolute path would be meaningless on CI,
		// another checkout, or after a project move. A relative path is portable.
		absProjectDir, absErr := filepath.Abs(opts.ProjectDir)
		if absErr != nil {
			return nil, fmt.Errorf("resolving project dir: %w", absErr)
		}
		rel, relErr := filepath.Rel(absProjectDir, absPath)
		if relErr != nil {
			return nil, fmt.Errorf("computing relative local pack path: %w", relErr)
		}
		localRelPath = rel
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

	// Distinguish DECLARED from INSTALLED. Manifest membership in backstop.yml is NOT
	// enough: a pack is genuinely installed-and-current only when it is materialized on
	// disk AND recorded in the lock. When it IS current, honestly report a no-op; when it
	// is declared-but-absent (or the lock diverged), fall through to the materialize/lock
	// pipeline below and actually install it.
	if isPackInstalledAndCurrent(opts.ProjectDir, packName) {
		return &AddResult{PackName: packName, AlreadyCurrent: true}, nil
	}

	if !isLocal {
		// Clone the pack to a temporary directory.
		tmpDir, err := os.MkdirTemp("", "backstop-pack-clone-*")
		if err != nil {
			return nil, fmt.Errorf("creating temp dir: %w", err)
		}
		defer func() { _ = os.RemoveAll(tmpDir) }()

		gitURL := resolveGitURL(packName)
		if err := opts.GitCloner.Clone(gitURL, "v"+version, tmpDir); err != nil {
			return nil, fmt.Errorf("cloning pack %s: %w", packName, err)
		}
		packDir = tmpDir
	}

	// Validate: run pack check and pack test.
	if opts.Validator != nil {
		if err := opts.Validator.RunPackCheck(packDir); err != nil {
			return nil, fmt.Errorf("pack check for %s: %w", packName, err)
		}
		if err := opts.Validator.RunPackTest(packDir); err != nil {
			return nil, fmt.Errorf("pack test for %s: %w", packName, err)
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
		_ = os.RemoveAll(installedPath)
		return nil, fmt.Errorf("computing content hash: %w", err)
	}

	// Snapshot files for transactional rollback.
	ymlPath := filepath.Join(opts.ProjectDir, "backstop.yml")
	lockPath := filepath.Join(opts.ProjectDir, "backstop.lock")
	backstopDir := filepath.Join(opts.ProjectDir, ".backstop")
	if err := os.MkdirAll(backstopDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating .backstop dir: %w", err)
	}
	provPath := filepath.Join(backstopDir, "pack-config-provenance.json")

	ymlSnap := readFileOrNil(ymlPath)
	lockSnap := readFileOrNil(lockPath)
	provSnap := readFileOrNil(provPath)

	// rollback best-effort restores the snapshotted files; its cleanup calls are
	// genuinely fire-and-forget (there is no recovery if undo itself fails).
	rollback := func() {
		_ = os.RemoveAll(installedPath)
		if ymlSnap != nil {
			_ = os.WriteFile(ymlPath, ymlSnap, 0o644)
		}
		if lockSnap != nil {
			_ = os.WriteFile(lockPath, lockSnap, 0o644)
		} else {
			_ = os.Remove(lockPath)
		}
		if provSnap != nil {
			_ = os.WriteFile(provPath, provSnap, 0o644)
		} else {
			_ = os.Remove(provPath)
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

	// Update backstop.lock. A missing/unreadable lock is expected on a first add — fall
	// back to a fresh lockfile rather than ignoring the error silently.
	lf, lockErr := ReadLockfile(lockPath)
	if lockErr != nil || lf == nil {
		lf = &Lockfile{Packs: make(map[string]LockEntry)}
	}

	lf.Packs[packName] = LockEntry{
		Name:        packName,
		Version:     version,
		GitRef:      gitRef,
		ContentHash: contentHash,
		SourceType:  sourceType,
		InstallDate: time.Now().UTC().Format(time.RFC3339),
		LocalPath:   localRelPath,
	}

	if err := WriteLockfile(lockPath, lf); err != nil {
		rollback()
		return nil, err
	}

	// Ensure .backstop/packs/ in .gitignore.
	if err := ensureGitignore(opts.ProjectDir); err != nil {
		return nil, fmt.Errorf("updating .gitignore: %w", err)
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

// isPackInstalledAndCurrent reports whether a pack is genuinely installed and current:
// its artifact is MATERIALIZED on disk (.backstop/packs/<name>/ exists AND is non-empty)
// AND backstop.lock holds an entry for that name. Manifest membership in backstop.yml
// alone does NOT count — a declared-but-absent pack, an empty materialized dir, or one
// whose lock entry is missing/diverged is NOT installed-and-current and must be
// (re)installed. Read/stat errors are treated explicitly as "not current" so Add proceeds
// to install rather than silently swallowing the condition.
func isPackInstalledAndCurrent(projectDir, packName string) bool {
	packDir := filepath.Join(projectDir, ".backstop", "packs", packName)
	entries, err := os.ReadDir(packDir)
	if err != nil || len(entries) == 0 {
		// Missing or empty materialized dir ⇒ not installed on disk.
		return false
	}

	lf, err := ReadLockfile(filepath.Join(projectDir, "backstop.lock"))
	if err != nil || lf == nil {
		// Missing/unreadable lock ⇒ not current.
		return false
	}

	_, ok := lf.Packs[packName]
	return ok
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

// readFileOrNil reads a file, returning nil when it cannot be read (e.g. it does not
// exist yet). It handles the read error explicitly — the caller treats "no bytes" as
// "no prior file to snapshot" — rather than silently discarding an error inline.
func readFileOrNil(path string) []byte {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return data
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
