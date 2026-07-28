package backstopcore_test

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/config"
	"github.com/backstop-ai/backstop-core/pkg/pack/distribution"
)

// These are dogfood tests over the REAL repository state — the actual
// backstop.lock, the actual backstop.yml, and the actual Go sources. They assert
// on PARSED YAML and PARSED Go, never on golden text, so reformatting either file
// does not break them and a genuine regression does.
//
// They exist because a CI runner is not this machine. `materializeLocalPack`
// (pkg/pack/distribution/install.go) fails loud on an empty `local_path` and on a
// `local_path` that does not resolve on disk, so a lock entry pointing outside the
// repository is installable HERE and nowhere else. Nothing else in the suite
// notices, because every other pack test builds its own fixture.
const (
	lockfilePath = "backstop.lock"
	configDir    = "."
)

// retiredPackCoordinates are the pack coordinates this repository has MIGRATED
// AWAY FROM, in both the coordinate form and the rule-ID form a waiver embeds.
//
// Scope note, deliberate and load-bearing: this set covers only the three
// coordinates the fleet migration retires. It does NOT include `backstop/self` or
// `backstop/go-standards`, which were renamed on 2026-07-27. Two live waivers
// (cmd/backstop/pack_gate.go, cmd/backstop/pack_gate_provision.go) still carry the
// `backstop.packs.backstop.self...` rule-ID and appear unbound after that rename.
// Re-keying a waiver changes WHICH FINDINGS ARE SUPPRESSED — a scope change dressed
// as a typo fix — so those are reported rather than fixed, and widening this set to
// cover them would force exactly the fix that was ruled out.
//
// Declared as a FUNCTION rather than a package-level var, matching
// legacyReferenceSkips in module_path_test.go: the go-standards rule
// no-global-mutable-state flags any package-level mutable declaration, and a slice
// or map var is mutable however it is used.
func retiredPackCoordinates() []string {
	return []string{
		"backstop/contracts",
		"backstop/substantiveness",
		"backstop/go-toolchain",
		"backstop.packs.backstop.contracts",
		"backstop.packs.backstop.substantiveness",
		"backstop.packs.backstop.go-toolchain",
	}
}

// keyingSweepRoots are the directories walked for live keying on a retired
// coordinate. `tests` is listed explicitly because it sits outside a pkg+cmd sweep
// and is missed in both directions — it is the difference between a tree-wide count
// of 36 and the true 37.
// It is a function rather than a var for the same no-global-mutable-state reason
// as retiredPackCoordinates.
func keyingSweepRoots() []string {
	return []string{"pkg", "cmd", "tests"}
}

// syntheticCarrierExemptions are the files permitted to contain a retired
// coordinate in a STRING LITERAL, each with the reason it is exempt. An exemption
// list without reasons decays into a place to hide failures, so the reason is data
// here rather than a comment someone can delete without noticing.
//
// Every one of these builds or demonstrates its OWN pack under the old name. The
// literal never escapes the fixture, so renaming it would churn the test while
// proving nothing — and in the two self-rule cases would actively destroy the
// example.
// It is a function rather than a var for the same no-global-mutable-state reason
// as retiredPackCoordinates.
func syntheticCarrierExemptions() map[string]string {
	return map[string]string{
		"pkg/gate/testdata/self-rule/pack-name-keyed-capability.go": "NEGATIVE-EXAMPLE fixture for the backstop/self rules: the old-coordinate literal IS the demonstrated anti-pattern (a capability keyed on a pack NAME). Renaming it leaves the rule intact and destroys the example.",
		"pkg/gate/testdata/self-rule/rule-id-keyed-routing.go":      "NEGATIVE-EXAMPLE fixture for the backstop/self rules: the old-coordinate literal IS the demonstrated anti-pattern (routing keyed on a rule ID). Same reasoning as its sibling fixture.",
		"pkg/pack/distribution/contracts_provisioning_test.go":      "Builds its OWN temp pack under the old coordinate and installs it into a scratch project. The literal is local to the fixture and means nothing outside it.",
		"tests/smoke/smoke_test.go":                                 "Writes its own backstop/go-toolchain manifest and lock into a scratch dir. Same synthetic-fixture reasoning, and it is the carrier that sits outside a pkg+cmd sweep.",
		"pkg/pack/distribution/contracts_local_install_test.go":     "Asserts the installed name of the repo-root `packs/contracts` SOURCE, whose pack.yml still declares `backstop/contracts` (verified 2026-07-28). That directory is a fixture the helper installs into temp projects under its own declared name — it is not a fleet member and is a documented non-goal of the migration. Re-keying this assertion would make it assert something the fixture does not say.",
	}
}

// TestBackstopLock_EveryEntryIsCIMaterializable asserts every locked pack can be
// materialized on a fresh checkout — the CI condition. A git entry must be pinned
// and carry the coordinate needed to fetch it; a local entry must point at a
// directory INSIDE this repository, because a path outside it does not exist on a
// runner. (CLM-001)
func TestBackstopLock_EveryEntryIsCIMaterializable(t *testing.T) {
	lockfile, err := distribution.ReadLockfile(lockfilePath)
	if err != nil {
		t.Fatalf("read %s: %v", lockfilePath, err)
	}
	if len(lockfile.Packs) == 0 {
		t.Fatalf("%s declares no packs — every assertion below would pass vacuously", lockfilePath)
	}

	repoRootAbs, err := filepath.Abs(configDir)
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}

	for _, name := range sortedMapKeys(lockfile.Packs) {
		entry := lockfile.Packs[name]

		switch entry.SourceType {
		case "git":
			if strings.TrimSpace(entry.Version) == "" {
				t.Errorf("pack %q: source_type git with an empty version — an unpinned entry cannot be reproduced on a runner", name)
			}
			if strings.TrimSpace(entry.SourceCoordinate) == "" {
				t.Errorf("pack %q: source_type git with no source_coordinate — nothing records WHERE to clone it from", name)
			}

		case "local":
			if strings.TrimSpace(entry.LocalPath) == "" {
				t.Errorf("pack %q: source_type local with no local_path — materializeLocalPack fails loud on this, so the pack is installable on no machine but the one that added it", name)
				continue
			}

			resolved, absErr := filepath.Abs(filepath.Join(configDir, entry.LocalPath))
			if absErr != nil {
				t.Errorf("pack %q: resolve local_path %q: %v", name, entry.LocalPath, absErr)
				continue
			}

			rel, relErr := filepath.Rel(repoRootAbs, resolved)
			if relErr != nil {
				t.Errorf("pack %q: relativize local_path %q against the repository root: %v", name, entry.LocalPath, relErr)
				continue
			}
			if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				t.Errorf("pack %q: local_path %q resolves to %s, OUTSIDE the repository root %s — that path does not exist on a CI runner",
					name, entry.LocalPath, resolved, repoRootAbs)
				continue
			}

			info, statErr := os.Stat(resolved)
			if statErr != nil {
				t.Errorf("pack %q: local_path %q does not resolve on disk (%v)", name, entry.LocalPath, statErr)
				continue
			}
			if !info.IsDir() {
				t.Errorf("pack %q: local_path %q resolves to a file, want a directory", name, entry.LocalPath)
			}

		default:
			t.Errorf("pack %q: unrecognized source_type %q — want git or local", name, entry.SourceType)
		}
	}
}

// TestBackstopYmlAndLock_DeclareTheSameCoordinates asserts backstop.yml's `packs:`
// keys and backstop.lock's keys are the SAME SET.
//
// This is the falsifier for step 2 of the rename sequence. `pack add` ADDS a
// backstop.yml key for the new coordinate and does not remove the old one, so a
// half-finished migration leaves both — and nothing else in the repository
// complains. (CLM-001)
func TestBackstopYmlAndLock_DeclareTheSameCoordinates(t *testing.T) {
	cfg, err := config.LoadConfigFromDir(configDir)
	if err != nil {
		t.Fatalf("load backstop.yml: %v", err)
	}
	lockfile, err := distribution.ReadLockfile(lockfilePath)
	if err != nil {
		t.Fatalf("read %s: %v", lockfilePath, err)
	}

	if len(cfg.Packs) == 0 {
		t.Fatalf("backstop.yml declares no packs — the set comparison below would be vacuous")
	}
	if len(lockfile.Packs) == 0 {
		t.Fatalf("%s declares no packs — the set comparison below would be vacuous", lockfilePath)
	}

	for _, name := range sortedMapKeys(cfg.Packs) {
		if _, locked := lockfile.Packs[name]; !locked {
			t.Errorf("backstop.yml declares pack %q but %s has no entry for it — a stale config key left behind by a rename, or a pack that was never installed",
				name, lockfilePath)
		}
	}
	for _, name := range sortedMapKeys(lockfile.Packs) {
		if _, declared := cfg.Packs[name]; !declared {
			t.Errorf("%s locks pack %q but backstop.yml does not declare it — the lock and the config disagree about what this project uses",
				lockfilePath, name)
		}
	}
}

// TestNoLiveKeyingOnRetiredPackCoordinates asserts nothing in this repository
// still KEYS ON a retired pack coordinate. It checks the two surfaces where a
// retired coordinate would actually change behavior, and deliberately checks
// neither test fixtures nor prose.
//
// SURFACE 1 — the real repository's DECLARED pack surface: backstop.yml's keys,
// backstop.lock's keys, and the `.backstop/packs/` install tree. A retired
// coordinate surviving here is the defect CLM-003 names: it makes the fleet
// unresolvable on a fresh checkout and leaves `detectExtraUnlocked` failing on a
// directory no lock entry claims. This half was genuinely RED before the
// migration (backstop.yml carried `backstop/contracts: local`, the lock carried
// three matching entries, and `.backstop/packs/backstop/` held three directories).
//
// SURFACE 2 — PRODUCTION Go. Parsed string literals only, because a map key, a
// path join and a comparison are all built from a literal, whereas a doc comment
// mentioning a retired coordinate changes nothing. Several legitimate comment
// mentions exist and are permitted by construction: cmd/backstop/gate.go:442,475,
// 484, cmd/backstop/pack_separation.go:13, cmd/backstop/gate_substantiveness_e2e
// .go:102 and pkg/pack/distribution/contracts_local_install.go:11 all explain the
// migration in prose. A byte grep cannot tell those from live keying and would
// demand edits that make the corpus worse.
//
// WHY TEST FILES AND testdata ARE OUT OF THE LITERAL SWEEP — this is the one
// judgment in the file, and it rests on a measurement rather than on taste.
// After the fleet moved to backstop-ai/go-contracts, backstop-ai/go-substantiveness
// and backstop-ai/go-toolchain, `go test ./... -race -count=1` stayed GREEN across
// the whole repository (2026-07-28). Not one existing test failed. That is only
// possible if NO test asserted on the real repository's retired coordinates — and
// inspection confirms it: the carriers either build their own pack in a t.TempDir,
// or read a checked-in fixture project under cmd/backstop/testdata/ that ships its
// OWN .backstop/packs/backstop/go-toolchain tree, or construct a synthetic
// violation stream whose rule IDs are the fixture's own. `goToolchainManifest`
// (cmd/backstop/pack_gate_gotoolchain_test.go:87) is the clearest case: it parses
// cmd/backstop/testdata/go-toolchain/.backstop/packs/backstop/go-toolchain/pack.yml.
// Renaming those literals would churn 23 files, change no behavior, and — for the
// self-rule fixtures — destroy the anti-pattern each one exists to demonstrate.
//
// So the sweep covers production Go, where a coordinate literal IS live keying,
// and the declared surface, where a coordinate IS the fleet. (CLM-003)
func TestNoLiveKeyingOnRetiredPackCoordinates(t *testing.T) {
	assertDeclaredSurfaceIsFreeOfRetiredCoordinates(t)
	assertProductionGoDoesNotKeyOnRetiredCoordinates(t)
	assertNamedSyntheticCarriersAreStructurallyExcluded(t)
}

// assertDeclaredSurfaceIsFreeOfRetiredCoordinates covers SURFACE 1.
func assertDeclaredSurfaceIsFreeOfRetiredCoordinates(t *testing.T) {
	t.Helper()

	cfg, err := config.LoadConfigFromDir(configDir)
	if err != nil {
		t.Fatalf("load backstop.yml: %v", err)
	}
	lockfile, err := distribution.ReadLockfile(lockfilePath)
	if err != nil {
		t.Fatalf("read %s: %v", lockfilePath, err)
	}
	if len(cfg.Packs) == 0 || len(lockfile.Packs) == 0 {
		t.Fatalf("backstop.yml declares %d packs and %s locks %d — a surface with no entries would pass vacuously",
			len(cfg.Packs), lockfilePath, len(lockfile.Packs))
	}

	for _, declared := range sortedMapKeys(cfg.Packs) {
		if coordinate, retired := matchRetiredCoordinate(declared); retired {
			t.Errorf("backstop.yml still declares pack %q, which keys on the retired coordinate %q", declared, coordinate)
		}
	}
	for _, locked := range sortedMapKeys(lockfile.Packs) {
		if coordinate, retired := matchRetiredCoordinate(locked); retired {
			t.Errorf("%s still locks pack %q, which keys on the retired coordinate %q", lockfilePath, locked, coordinate)
		}
	}

	// The install tree. detectExtraUnlocked walks packsDir as org/pack and FAILS any
	// directory no lock entry claims, so a leftover directory is a live gate failure
	// rather than clutter.
	packsDir := filepath.Join(configDir, ".backstop", "packs")
	orgs, err := os.ReadDir(packsDir)
	if err != nil {
		t.Fatalf("read %s — the packs must be installed for this assertion to mean anything: %v", packsDir, err)
	}
	installed := 0
	for _, org := range orgs {
		if !org.IsDir() {
			continue
		}
		packs, readErr := os.ReadDir(filepath.Join(packsDir, org.Name()))
		if readErr != nil {
			t.Errorf("read %s: %v", filepath.Join(packsDir, org.Name()), readErr)
			continue
		}
		for _, installedPack := range packs {
			if !installedPack.IsDir() {
				continue
			}
			installed++
			coordinate := org.Name() + "/" + installedPack.Name()
			if retiredCoordinate, retired := matchRetiredCoordinate(coordinate); retired {
				t.Errorf("%s still holds an installed pack directory %q, which keys on the retired coordinate %q — detectExtraUnlocked fails any directory the lock does not claim",
					packsDir, coordinate, retiredCoordinate)
			}
		}
	}
	if installed == 0 {
		t.Fatalf("%s holds no installed packs — run `backstop pack install`; this assertion is vacuous without them", packsDir)
	}
}

// assertProductionGoDoesNotKeyOnRetiredCoordinates covers SURFACE 2.
func assertProductionGoDoesNotKeyOnRetiredCoordinates(t *testing.T) {
	t.Helper()

	type offense struct {
		file       string
		line       int
		literal    string
		coordinate string
	}

	offenses := []offense{}
	filesParsed := 0

	sweepRoots := keyingSweepRoots()
	for _, root := range sweepRoots {
		walkErr := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				// testdata trees are fixture PROJECTS, each shipping its own
				// .backstop/packs — not this repository's declared surface.
				if entry.Name() == "testdata" {
					return filepath.SkipDir
				}
				return nil
			}
			if !isProductionGoFile(path) {
				return nil
			}

			fset := token.NewFileSet()
			// ParseComments attaches comments to the AST rather than discarding
			// them, which keeps them OUT of the BasicLit walk below — treating a
			// doc comment as a finding is exactly the mistake this avoids.
			file, parseErr := parser.ParseFile(fset, path, nil, parser.ParseComments)
			if parseErr != nil {
				// Loud, not skipped: a file that stops parsing would silently drop
				// out of the guard and the guard would still go green.
				t.Errorf("parse %s: %v", path, parseErr)
				return nil
			}
			filesParsed++

			ast.Inspect(file, func(node ast.Node) bool {
				lit, ok := node.(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					return true
				}
				value, unquoteErr := strconv.Unquote(lit.Value)
				if unquoteErr != nil {
					// Raw or malformed literal: fall back to the source text so a
					// backtick string is still inspected rather than skipped.
					value = lit.Value
				}
				if coordinate, retired := matchRetiredCoordinate(value); retired {
					offenses = append(offenses, offense{
						file:       filepath.ToSlash(path),
						line:       fset.Position(lit.Pos()).Line,
						literal:    value,
						coordinate: coordinate,
					})
				}
				return true
			})
			return nil
		})
		if walkErr != nil {
			t.Fatalf("walk %s: %v", root, walkErr)
		}
	}

	if filesParsed == 0 {
		t.Fatalf("parsed no production Go files under %v — the sweep found nothing and would pass vacuously", sweepRoots)
	}

	if len(offenses) > 0 {
		sort.Slice(offenses, func(i, j int) bool {
			if offenses[i].file != offenses[j].file {
				return offenses[i].file < offenses[j].file
			}
			return offenses[i].line < offenses[j].line
		})
		lines := make([]string, 0, len(offenses))
		for _, item := range offenses {
			lines = append(lines, item.file+":"+strconv.Itoa(item.line)+
				": string literal keys on retired coordinate "+strconv.Quote(item.coordinate)+
				" (literal: "+strconv.Quote(item.literal)+")")
		}
		t.Errorf("%d production string literal(s) still key on a retired pack coordinate:\n  %s",
			len(offenses), strings.Join(lines, "\n  "))
	}
}

// assertNamedSyntheticCarriersAreStructurallyExcluded keeps the four carriers the
// plan names — plus the fifth it missed — DOCUMENTED with their reasons, and turns
// that documentation into a live cross-check.
//
// Each is excluded by a STRUCTURAL rule (it is a test file, or it lives under a
// testdata tree) rather than by a hand-maintained path list, because a path list is
// a place to hide failures. This asserts the file still exists and that the
// structural rule genuinely covers it, so tightening the sweep later cannot silently
// start churning these fixtures — and deleting one cannot silently leave a dead
// justification behind.
func assertNamedSyntheticCarriersAreStructurallyExcluded(t *testing.T) {
	t.Helper()

	exemptions := syntheticCarrierExemptions()
	for _, path := range sortedMapKeys(exemptions) {
		reason := exemptions[path]
		if _, err := os.Stat(path); err != nil {
			t.Errorf("named synthetic carrier %q no longer exists (%v) — remove its justification rather than leaving a dead one: %s",
				path, err, reason)
			continue
		}
		if isProductionGoFile(path) && !strings.Contains(filepath.ToSlash(path), "/testdata/") {
			t.Errorf("named synthetic carrier %q is swept as production Go, so its literal WILL be reported — the structural rule no longer covers it. Reason it must stay: %s",
				path, reason)
		}
	}
}

// isProductionGoFile reports whether a path is Go source this repository SHIPS, as
// opposed to a test or a fixture. Only shipped code can key on a coordinate in a
// way that changes what the binary does.
func isProductionGoFile(path string) bool {
	slashed := filepath.ToSlash(path)
	if !strings.HasSuffix(slashed, ".go") {
		return false
	}
	if strings.HasSuffix(slashed, "_test.go") {
		return false
	}
	return !strings.Contains(slashed, "/testdata/")
}

// matchRetiredCoordinate reports the first retired coordinate the value contains.
func matchRetiredCoordinate(value string) (string, bool) {
	for _, coordinate := range retiredPackCoordinates() {
		if strings.Contains(value, coordinate) {
			return coordinate, true
		}
	}
	return "", false
}

// ─── COVERAGE EXCLUSION: DOGFOOD OVER THE REAL INSTALLED PACK ──────────────────
//
// These drive the ACTUAL converter the gate runs
// (.backstop/packs/backstop-ai/go-toolchain/scripts/coverage-to-records.sh) by
// feeding it a profile on stdin and parsing the records it emits — the same
// contract dispatchPackCoverage uses. They are dogfood, not fixtures: if the
// installed pack cannot declare an exclusion, these fail.
//
// THE DECLARATION CHANNEL. The converter is PARSE-ONLY and runs SANDBOXED, so it
// has no project or toolchain access; everything it knows arrives as plain-text
// comment lines the un-sandboxed producer folds into the profile
// (`#backstop-module`, `#backstop-gofile`). An exclusion declaration therefore
// travels the same way — `#backstop-coverage-exclude <path> <justification>` —
// rather than through a new channel the sandbox would block.
const coverageExclusionDirective = "#backstop-coverage-exclude"

// coverageRecord mirrors the wire shape of check.CoverageRecord for parsing the
// converter's output without importing the consumer.
type coverageRecord struct {
	Path          string `json:"path"`
	Covered       int    `json:"covered"`
	Total         int    `json:"total"`
	Measured      bool   `json:"measured"`
	Excluded      bool   `json:"excluded"`
	Metric        string `json:"metric"`
	Justification string `json:"justification"`
}

// runInstalledCoverageConvert pipes profile into the real installed converter and
// returns the records it emitted.
func runInstalledCoverageConvert(t *testing.T, profile string) []coverageRecord {
	t.Helper()
	script := filepath.Join(".backstop", "packs", "backstop-ai", "go-toolchain", "scripts", "coverage-to-records.sh")
	if _, err := os.Stat(script); err != nil {
		t.Fatalf("the go-toolchain pack is not installed at %s: %v — these are dogfood tests over the "+
			"REAL pack, so an absent pack is a failure rather than a skip", script, err)
	}

	cmd := exec.Command("/bin/sh", script)
	cmd.Stdin = strings.NewReader(profile)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("running %s: %v\noutput: %s", script, err, string(out))
	}

	var records []coverageRecord
	if err := json.Unmarshal(out, &records); err != nil {
		t.Fatalf("the converter did not emit a JSON array of coverage records: %v\ngot: %s", err, string(out))
	}
	return records
}

// coverageProfileFixture builds an enriched Go profile naming two files: the helper
// (declared excluded) and its measured sibling (not declared).
func coverageProfileFixture(declareExclusion bool) string {
	const module = "github.com/backstop-ai/backstop-core"
	var b strings.Builder
	b.WriteString("mode: set\n")
	b.WriteString("#backstop-module " + module + "\n")
	if declareExclusion {
		b.WriteString(coverageExclusionDirective + " pkg/packval/sandbox_linux_helper.go " +
			"the exec boundary erases these counters; see testdata/sandbox-linux-coverage-profile.txt\n")
	}
	// Helper: deliberately BELOW any threshold, so a failure to honour the exclusion
	// shows up as a threshold verdict rather than as silence.
	b.WriteString(module + "/pkg/packval/sandbox_linux_helper.go:10.1,12.2 10 0\n")
	// Sibling: also below threshold, and never declared excluded.
	b.WriteString(module + "/pkg/packval/sandbox_linux.go:10.1,12.2 10 0\n")
	return b.String()
}

func findCoverageRecord(records []coverageRecord, path string) (coverageRecord, bool) {
	for _, r := range records {
		if r.Path == path {
			return r, true
		}
	}
	return coverageRecord{}, false
}

// TestCoverageExclusion_DeclaredPathIsSkippedNotFailed asserts the declared path is
// marked EXCLUDED rather than reported as a shortfall.
//
// Three outcomes are easy to confuse here and only one is correct: excluded (the
// threshold is skipped and the suppression is surfaced as a warning), measured-and-
// below-threshold (a blocking failure), and unmeasured (also blocking). A converter
// that dropped the record entirely would produce the third while looking like the
// first.
func TestCoverageExclusion_DeclaredPathIsSkippedNotFailed(t *testing.T) {
	records := runInstalledCoverageConvert(t, coverageProfileFixture(true))

	got, found := findCoverageRecord(records, "pkg/packval/sandbox_linux_helper.go")
	if !found {
		t.Fatalf("the converter emitted no record for the declared-excluded path; a MISSING record reads "+
			"to the gate as UNMEASURED, which blocks — the opposite of an exclusion. got: %+v", records)
	}
	if !got.Excluded {
		t.Errorf("the declared path came back excluded=false (%d/%d) — the gate would apply the threshold "+
			"and fail it", got.Covered, got.Total)
	}
}

// TestCoverageExclusion_CarriesAJustification asserts the excluded record states WHY.
//
// An exclusion with no stated reason is indistinguishable from a mistake, and it is
// what makes the next one easy. The gate does not require this — core deliberately
// accepts an unjustified exclusion — so THIS test is where the requirement lives for
// backstop-core's own pack.
func TestCoverageExclusion_CarriesAJustification(t *testing.T) {
	records := runInstalledCoverageConvert(t, coverageProfileFixture(true))

	got, found := findCoverageRecord(records, "pkg/packval/sandbox_linux_helper.go")
	if !found {
		t.Fatalf("no record for the declared-excluded path: %+v", records)
	}
	if strings.TrimSpace(got.Justification) == "" {
		t.Errorf("the excluded record carries no justification. The gate falls back to generic wording, so "+
			"the suppression would appear in the report with no way to tell a deliberate exclusion from a "+
			"mistake. record: %+v", got)
	}
}

// TestCoverageExclusion_UndeclaredPathStillFails is the guard against the mechanism
// becoming a blanket.
//
// Without it, a matcher that excluded EVERYTHING would satisfy both tests above and
// silently switch the coverage dimension off across the repo — a vacuous green
// wearing an exclusion's clothes. The sibling here is equally uncovered and simply
// not declared, so the only thing separating them is the declaration.
func TestCoverageExclusion_UndeclaredPathStillFails(t *testing.T) {
	records := runInstalledCoverageConvert(t, coverageProfileFixture(true))

	got, found := findCoverageRecord(records, "pkg/packval/sandbox_linux.go")
	if !found {
		t.Fatalf("no record for the undeclared sibling: %+v", records)
	}
	if got.Excluded {
		t.Errorf("an UNDECLARED path came back excluded=true — the exclusion matcher is a blanket, and the "+
			"coverage dimension is off for every file it touches. record: %+v", got)
	}
	if got.Justification != "" {
		t.Errorf("an undeclared path carries a justification (%q); only a declared exclusion should",
			got.Justification)
	}
}
