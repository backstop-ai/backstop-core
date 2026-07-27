package distribution

import (
	"os"
	"path/filepath"
	"strings"

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
//
// It carries only the operator's knobs. The cloner and validator that used to
// live here now reach the pipeline through NewAddCommand, because a dependency
// on an options value can simply be omitted — which is how a nil validator once
// let an invalid pack install cleanly.
type AddOptions struct {
	ProjectDir string
	Version    string
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
	// SourceCoordinate is the org/repository the operator requested, recorded verbatim
	// alongside PackName so a caller can answer "what is this pack called here?" and
	// "where did it come from?" independently (SPEC-056 REQ-004). Empty for local
	// sources, whose origin is already the path the operator gave.
	SourceCoordinate string `json:"source_coordinate,omitempty"`
	// Warnings carries diagnostics the add produced without failing — today the
	// manifest-name-versus-coordinate divergence (SPEC-056 REQ-006). Divergence is LOUD
	// and never a refusal, so it needs somewhere to ride out of a SUCCESSFUL command.
	//
	// Warnings ride the RESULT rather than an injected writer: pkg/pack/distribution owns
	// no output stream and must not acquire a dependency in order to carry a string. The
	// check happens here; the rendering happens where every other CLI message is rendered.
	Warnings []string `json:"warnings,omitempty"`
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
