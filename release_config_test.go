package backstopcore_test

import (
	"fmt"
	"os"
	"path"
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// goreleaserConfigFile is the release config this repo ships at its root. These
// are the REPO's tests about the REPO's own release configuration — they assert
// on parsed structure, never on raw text, so reindentation cannot break them.
const goreleaserConfigFile = ".goreleaser.yml"

// releaseBuildTarget is the package the released binary is built from.
const releaseBuildTarget = "./cmd/backstop"

// versionSymbol is the ldflags symbol path the release build must inject into.
// cmd/backstop is package main and declares `var version` there, so the path is
// main.version. If this and the variable in cmd/backstop ever diverge, every
// release silently ships "dev".
const versionSymbol = "main.version"

// Ratified 2026-07-27. The `brews:` block takes the REPOSITORY, and the
// repository is `homebrew-tap` — Homebrew strips the `homebrew-` prefix only
// when resolving a user-facing `brew tap backstop-ai/tap`. Asserting the
// shorthand `tap` would assert a repository that does not exist. A block
// pointing at the wrong tap fails silently: it publishes a formula into a
// namespace nobody is watching.
const (
	tapOwner = "backstop-ai"
	tapRepo  = "homebrew-tap"
)

// handWrittenFormulaKeys are the keys whose presence in a `brews:` entry means
// someone overrode goreleaser's formula generator. goreleaser writes the
// `install` and `test` stanzas itself, so finding them IN THE CONFIG is the
// signal, not their absence from the published formula. Enumerating them is
// what makes "no hand-written formula body" mechanically falsifiable.
func handWrittenFormulaKeys() []string {
	return []string{
		"formula",
		"template",
		"custom_block",
		"install",
		"test",
		"post_install",
		"caveats",
		"service",
	}
}

// stringList accepts a YAML scalar or a sequence of scalars. goreleaser allows
// `ldflags` in both shapes, and a test that only understood one would fail on a
// config that is perfectly valid.
type stringList []string

func (s *stringList) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		var single string
		if err := node.Decode(&single); err != nil {
			return fmt.Errorf("decode scalar: %w", err)
		}
		*s = stringList{single}
	case yaml.SequenceNode:
		var many []string
		if err := node.Decode(&many); err != nil {
			return fmt.Errorf("decode sequence: %w", err)
		}
		*s = many
	default:
		return fmt.Errorf("expected a scalar or a sequence, got YAML node kind %d", node.Kind)
	}
	return nil
}

// goreleaserBuildIgnore is one pruned goos/goarch pair.
type goreleaserBuildIgnore struct {
	Goos   string `yaml:"goos"`
	Goarch string `yaml:"goarch"`
}

type goreleaserBuild struct {
	Main    string                  `yaml:"main"`
	Dir     string                  `yaml:"dir"`
	Binary  string                  `yaml:"binary"`
	Goos    []string                `yaml:"goos"`
	Goarch  []string                `yaml:"goarch"`
	Ldflags stringList              `yaml:"ldflags"`
	Ignore  []goreleaserBuildIgnore `yaml:"ignore"`
}

type goreleaserChecksum struct {
	NameTemplate string `yaml:"name_template"`
}

type goreleaserRepository struct {
	Owner string `yaml:"owner"`
	Name  string `yaml:"name"`
}

// goreleaserConfig is the subset of the config these tests make claims about.
// Unlisted keys are ignored by design — this is a shape assertion, not a
// schema, and goreleaser itself is the authority on the full schema.
type goreleaserConfig struct {
	Version  int                    `yaml:"version"`
	Builds   []goreleaserBuild      `yaml:"builds"`
	Archives []map[string]yaml.Node `yaml:"archives"`
	Checksum goreleaserChecksum     `yaml:"checksum"`
	Release  map[string]yaml.Node   `yaml:"release"`
	Brews    []map[string]yaml.Node `yaml:"brews"`
}

// loadGoreleaserConfig parses the config twice: once into the typed shape above
// and once into a raw top-level key map. The raw map is what distinguishes "the
// key is absent" from "the key is present and empty" — a distinction the typed
// decode collapses, and one that matters for `release:`, whose defaults are
// entirely acceptable.
func loadGoreleaserConfig(t *testing.T) (goreleaserConfig, map[string]yaml.Node) {
	t.Helper()

	data, err := os.ReadFile(goreleaserConfigFile)
	if err != nil {
		t.Fatalf("read %s: %v", goreleaserConfigFile, err)
	}

	var cfg goreleaserConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("parse %s: %v", goreleaserConfigFile, err)
	}

	top := map[string]yaml.Node{}
	if err := yaml.Unmarshal(data, &top); err != nil {
		t.Fatalf("parse %s top-level keys: %v", goreleaserConfigFile, err)
	}

	return cfg, top
}

// singleBuild returns the config's one and only build entry. More than one
// build entry means the four-platform claim below is being made about a config
// that also builds something else.
func singleBuild(t *testing.T, cfg goreleaserConfig) goreleaserBuild {
	t.Helper()

	if len(cfg.Builds) != 1 {
		t.Fatalf("builds: got %d entries, want exactly 1", len(cfg.Builds))
	}
	return cfg.Builds[0]
}

// TestGoreleaserConfig_DeclaresFourTargetPlatforms asserts the release builds
// ./cmd/backstop for EXACTLY the four wanted pairs.
//
// The assertions are on SETS and are EXACT. A config that also builds
// windows/386 would satisfy a "contains" assertion and ship binaries nobody
// asked for, so equality is the only assertion that carries the claim. goos and
// goarch must be declared explicitly: goreleaser defaults them to a wider set
// than this project wants, so an omitted key is a silent widening. (CLM-010)
func TestGoreleaserConfig_DeclaresFourTargetPlatforms(t *testing.T) {
	cfg, _ := loadGoreleaserConfig(t)
	build := singleBuild(t, cfg)

	// `main` is resolved relative to `dir` when `dir` is set, so the effective
	// target is the join of the two.
	effectiveMain := path.Clean(path.Join(build.Dir, build.Main))
	if wantMain := path.Clean(releaseBuildTarget); effectiveMain != wantMain {
		t.Errorf("build main = %q (dir %q + main %q), want %q",
			effectiveMain, build.Dir, build.Main, wantMain)
	}

	if got, want := sortedCopy(build.Goos), []string{"darwin", "linux"}; !equalStrings(got, want) {
		t.Errorf("build goos = %v, want exactly %v", got, want)
	}
	if got, want := sortedCopy(build.Goarch), []string{"amd64", "arm64"}; !equalStrings(got, want) {
		t.Errorf("build goarch = %v, want exactly %v", got, want)
	}

	wantPairs := []string{
		"darwin/amd64",
		"darwin/arm64",
		"linux/amd64",
		"linux/arm64",
	}
	if got := buildPairs(build); !equalStrings(got, wantPairs) {
		t.Errorf("build produces pairs %v, want exactly %v (ignore: %+v)",
			got, wantPairs, build.Ignore)
	}
}

// buildPairs is the goos × goarch cross product with the `ignore:` entries
// removed — the set of platforms the build actually produces.
func buildPairs(build goreleaserBuild) []string {
	ignored := map[string]struct{}{}
	for _, entry := range build.Ignore {
		ignored[entry.Goos+"/"+entry.Goarch] = struct{}{}
	}

	pairs := []string{}
	for _, goos := range build.Goos {
		for _, goarch := range build.Goarch {
			pair := goos + "/" + goarch
			if _, skip := ignored[pair]; skip {
				continue
			}
			pairs = append(pairs, pair)
		}
	}
	sort.Strings(pairs)
	return pairs
}

func sortedCopy(values []string) []string {
	out := append([]string{}, values...)
	sort.Strings(out)
	return out
}

func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

// TestGoreleaserConfig_InjectsVersionIntoMainVersionSymbol asserts the build
// injects the release version into the main.version symbol.
//
// This is the join between version resolution and the release config: the
// assertion is on the SYMBOL PATH, not merely on the presence of some -X flag,
// because an -X against the wrong symbol injects nothing and the binary reports
// "dev" from a release build. (CLM-011)
func TestGoreleaserConfig_InjectsVersionIntoMainVersionSymbol(t *testing.T) {
	cfg, _ := loadGoreleaserConfig(t)
	build := singleBuild(t, cfg)

	if len(build.Ldflags) == 0 {
		t.Fatal("build declares no ldflags; the release version is never injected")
	}

	want := "-X " + versionSymbol + "={{.Version}}"
	joined := strings.Join(build.Ldflags, " ")
	// Template whitespace is a style choice goreleaser accepts either way, so
	// normalize it rather than pinning one spelling.
	normalized := strings.NewReplacer("{{ .Version }}", "{{.Version}}", "{{ .Version}}", "{{.Version}}", "{{.Version }}", "{{.Version}}").Replace(joined)
	if !strings.Contains(normalized, want) {
		t.Errorf("build ldflags %q do not inject %q", joined, want)
	}
}

// TestGoreleaserConfig_ProducesArchivesChecksumsAndRelease asserts one
// `goreleaser release` invocation yields archives, a checksums file and a
// GitHub Release.
//
// The top-level `version: 2` key is asserted here too: goreleaser v2 requires
// it, and its absence surfaces as a confusing runtime error rather than a parse
// error. (CLM-011)
func TestGoreleaserConfig_ProducesArchivesChecksumsAndRelease(t *testing.T) {
	cfg, top := loadGoreleaserConfig(t)

	if cfg.Version != 2 {
		t.Errorf("top-level version = %d, want 2 (the goreleaser v2 config schema)", cfg.Version)
	}

	if len(cfg.Archives) == 0 {
		t.Error("archives: no entries; the release publishes bare binaries with no archives")
	}

	if cfg.Checksum.NameTemplate == "" {
		t.Error("checksum: no name_template declared")
	}

	if _, present := top["release"]; !present {
		t.Fatal("release: section absent; goreleaser publishes no GitHub Release")
	}

	if disable, present := cfg.Release["disable"]; present && disable.Value == "true" {
		t.Error("release.disable is true; the GitHub Release is switched off")
	}
}

// TestGoreleaserConfig_DeclaresHomebrewTapTarget asserts the `brews:` block
// targets the ratified tap by OWNER and NAME both, and carries no hand-written
// formula body.
//
// Homebrew ships at launch (founder reversal 2026-07-27), so this asserts the
// block's PRESENCE — an earlier draft asserted its absence, back when the tap
// was a deferral, and leaving that in place would block the thing that was
// asked for.
//
// There is deliberately no `go install` assertion here: that path needs no
// config at all. It works because the module was renamed and version resolution
// falls back to build info. A `go install` section in this file would mean
// someone misunderstood which mechanism carries it. (CLM-019)
func TestGoreleaserConfig_DeclaresHomebrewTapTarget(t *testing.T) {
	cfg, _ := loadGoreleaserConfig(t)

	if len(cfg.Brews) == 0 {
		t.Fatal("brews: no entries; goreleaser publishes no Homebrew formula")
	}

	// EVERY entry is checked, not just the first: a second entry pointing at
	// some other tap would publish into a namespace nobody is watching, and
	// that failure is silent.
	for i, entry := range cfg.Brews {
		repoNode, ok := entry["repository"]
		if !ok {
			t.Errorf("brews[%d]: no repository block; the tap is unnamed", i)
			continue
		}

		var repository goreleaserRepository
		if err := repoNode.Decode(&repository); err != nil {
			t.Errorf("brews[%d]: decode repository: %v", i, err)
			continue
		}

		if repository.Owner != tapOwner {
			t.Errorf("brews[%d]: repository owner = %q, want %q", i, repository.Owner, tapOwner)
		}
		if repository.Name != tapRepo {
			t.Errorf("brews[%d]: repository name = %q, want %q — the tap REPOSITORY, "+
				"not the `brew tap` shorthand", i, repository.Name, tapRepo)
		}

		for _, key := range handWrittenFormulaKeys() {
			if _, present := entry[key]; present {
				t.Errorf("brews[%d]: declares %q — the formula must be goreleaser-GENERATED, "+
					"and this key is how a hand-authored formula gets smuggled in", i, key)
			}
		}
	}
}
