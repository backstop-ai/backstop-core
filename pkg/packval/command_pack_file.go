package packval

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// commandPackFileCheckName is the phase2 check for a command: token naming a
// pack file (ISSUE-207 D-6).
const commandPackFileCheckName = "command-pack-relative-file"

// commandPackFileFinding is one producer-less engine command: token that resolves
// to a regular file under the pack dir.
type commandPackFileFinding struct {
	Engine string
	Token  string
}

// commandPackFileFindings returns, sorted by engine then token, every command:
// token of a producer-less engine that names a regular file under packDir. The
// gate runs command: with the PROJECT root as cwd, so such a token is broken at
// gate time by construction. An engine declaring producer: is skipped entirely:
// core never invokes its command: tool, and its tokens are arguments the producer
// interprets (they may legitimately exist in both the pack and the project).
// Skipped tokens: flags ("-..."), absolute paths, tokens escaping packDir after
// Clean, and anything that is not a regular file (directories, globs, absent
// paths).
func commandPackFileFindings(pack *PackManifest, packDir string) []commandPackFileFinding {
	if pack == nil {
		return nil
	}
	names := make([]string, 0, len(pack.Engines))
	for name := range pack.Engines {
		names = append(names, name)
	}
	sort.Strings(names)
	var out []commandPackFileFinding
	for _, name := range names {
		binding := pack.Engines[name]
		if binding.Producer != "" {
			continue
		}
		tokens := strings.Fields(binding.Command)
		sort.Strings(tokens)
		seen := map[string]bool{}
		for _, token := range tokens {
			if seen[token] || !cpfNamesPackFile(packDir, token) {
				continue
			}
			seen[token] = true
			out = append(out, commandPackFileFinding{Engine: name, Token: token})
		}
	}
	return out
}

// cpfNamesPackFile reports whether token resolves to a regular file under packDir.
func cpfNamesPackFile(packDir, token string) bool {
	if token == "" || strings.HasPrefix(token, "-") || filepath.IsAbs(token) {
		return false
	}
	rel := filepath.Clean(filepath.FromSlash(token))
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	info, err := os.Stat(filepath.Join(packDir, rel))
	return err == nil && info.Mode().IsRegular()
}
