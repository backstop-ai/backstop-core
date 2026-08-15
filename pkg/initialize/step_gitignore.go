package initialize

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/backstop-ai/backstop-core/pkg/config"
	"github.com/backstop-ai/backstop-core/pkg/pack"
)

// stepGitignoreName is the report name for step 5.
const stepGitignoreName = "gitignore"

// backstopOwnedIgnores are the THREE literals core states, and they are backstop's own
// paths — the installed-pack tree, the gitignored local baseline, and the pack-config
// provenance record. No language, framework or tool path joins them: everything else
// in the entry set is pack-DECLARED.
var backstopOwnedIgnores = []string{ // nosemgrep: go.core.no-global-mutable-state — immutable literal list, never mutated
	".backstop/packs/",
	".backstop/baseline.json",
	".backstop/pack-config-provenance.json",
}

// gitignoreResidueNotice is the accepted residue, stated in words (REQ-005).
//
// It is a SENTENCE rather than a guess because the alternative is init enumerating
// paths for toolchains it knows nothing about. Closing the residue properly needs a
// new pack-manifest field — a declaration a pack author makes — and that is another
// bundle's surface, not something init should paper over.
const gitignoreResidueNotice = "Pack-derived entries come from each engine's declared stdout_artifact, which names only what that engine writes for the gate to read. " +
	"Anything else a toolchain leaves on disk — dependency directories, native build output, local caches — is not covered and stays yours to ignore; backstop does not guess at it."

// stepGitignore emits the canonical `.gitignore` (SPEC-069 REQ-005).
//
// THE ENTRY SET IS THE THREE CORE LITERALS PLUS, FOR EACH INSTALLED PACK, EVERY
// ENGINE'S DECLARED `stdout_artifact`. Nothing is derived, guessed or defaulted, and
// an engine declaring none contributes nothing.
//
// THIS STEP RUNS AFTER THE PACK STEP, AND THE ORDER IS LOAD-BEARING RATHER THAN TIDY.
// Because the entry set is a function of the INSTALLED packs, emitting the file first
// is unsatisfiable: a `--pack` run on a fresh repo would write a `.gitignore` missing
// every pack-derived entry, which is precisely the cross-repo ignore divergence this
// requirement exists to end.
//
// THE WRITE IS APPEND-ONLY against an existing file — read, skip entries already
// present, append the rest, never rewrite — the same posture the shipped pack-add path
// already takes.
func stepGitignore(projectRoot string, packs []*pack.Manifest) StepReport {
	entries := append([]string{}, backstopOwnedIgnores...)
	entries = append(entries, declaredStdoutArtifacts(packs)...)

	gitignorePath := filepath.Join(projectRoot, ".gitignore")
	existing, readErr := os.ReadFile(gitignorePath)
	if readErr != nil && !os.IsNotExist(readErr) {
		return StepReport{
			Step:    stepGitignoreName,
			Outcome: OutcomeBrokenPromise,
			Detail:  fmt.Sprintf("reading %s: %v", gitignorePath, readErr),
		}
	}

	present := map[string]bool{}
	for _, line := range strings.Split(string(existing), "\n") {
		present[strings.TrimSpace(line)] = true
	}

	missing := make([]string, 0, len(entries))
	for _, entry := range entries {
		if present[entry] {
			continue
		}
		present[entry] = true
		missing = append(missing, entry)
	}

	if len(missing) == 0 {
		return StepReport{
			Step:    stepGitignoreName,
			Outcome: OutcomeConverged,
			Detail:  ".gitignore already carries every backstop-owned and pack-declared entry. " + gitignoreResidueNotice,
		}
	}

	// APPEND-ONLY. The existing bytes are carried through untouched and the new
	// entries follow, separated by a newline only when the file did not already end
	// with one — so a consumer's last line is never joined to backstop's first.
	var builder strings.Builder
	builder.Write(existing)
	if len(existing) > 0 && !strings.HasSuffix(string(existing), "\n") {
		builder.WriteString("\n")
	}
	builder.WriteString(strings.Join(missing, "\n"))
	builder.WriteString("\n")

	if writeErr := os.WriteFile(gitignorePath, []byte(builder.String()), 0o644); writeErr != nil {
		return StepReport{
			Step:    stepGitignoreName,
			Outcome: OutcomeBrokenPromise,
			Detail:  fmt.Sprintf("writing %s: %v", gitignorePath, writeErr),
		}
	}

	action := "wrote .gitignore with"
	if len(existing) > 0 {
		action = "appended to .gitignore, leaving every pre-existing line untouched:"
	}
	return StepReport{
		Step:    stepGitignoreName,
		Outcome: OutcomeDelivered,
		Detail:  fmt.Sprintf("%s %s. %s", action, strings.Join(missing, ", "), gitignoreResidueNotice),
	}
}

// declaredStdoutArtifacts collects every installed pack engine's declared
// `stdout_artifact`, deduplicated and sorted.
//
// SORTED because Manifest.Engines is a MAP: an unsorted walk would emit a different
// .gitignore on every run, and a file whose byte content changes without its meaning
// changing is a diff a consumer has to review for nothing.
//
// The field is read straight off the parsed manifest. Init contributes no path of its
// own here — an engine that declares nothing contributes nothing, and inventing a
// plausible output path for it would be core holding knowledge about a tool it has
// never heard of.
func declaredStdoutArtifacts(packs []*pack.Manifest) []string {
	seen := map[string]bool{}
	declared := []string{}

	for _, manifest := range packs {
		if manifest == nil {
			continue
		}
		for _, spec := range manifest.Engines {
			artifact := strings.TrimSpace(spec.Binding.StdoutArtifact)
			if artifact == "" || seen[artifact] {
				continue
			}
			seen[artifact] = true
			declared = append(declared, artifact)
		}
	}

	sort.Strings(declared)
	return declared
}

// installedManifests reads the manifests of the packs the project DECLARES, as they
// stand at the moment it is called.
//
// WHY IT IS READ AT STEP TIME RATHER THAN INJECTED. The pack step installs packs during
// the SAME run, so a corpus captured before Run started would be the empty set the
// project had before init touched it — and the gitignore entry set is a function of the
// packs that are installed NOW.
//
// WHY IT LIVES HERE RATHER THAN BEHIND A SEAM. The PackInstaller seam installs; it does
// not enumerate. Widening it to also report manifests would give the seam two
// responsibilities and would make every test double answer a question it has no
// business answering. This reads DECLARED DATA through the shipped config loader and
// the shipped manifest parser, and decides nothing.
//
// IT IS DELIBERATELY TOLERANT. A project with no config, no packs directory, or a pack
// that failed to install yields whatever could be read and no error: the gitignore step
// must work in a project with zero packs, and a pack that failed to install has already
// been reported LOUDLY by the pack step. Failing here as well would attribute that
// failure to the wrong step.
func installedManifests(projectRoot string) []*pack.Manifest {
	cfg, err := config.LoadConfigFromPath(filepath.Join(projectRoot, "backstop.yml"))
	if err != nil || cfg == nil || len(cfg.Packs) == 0 {
		return nil
	}

	names := make([]string, 0, len(cfg.Packs))
	for name := range cfg.Packs {
		names = append(names, name)
	}
	// SORTED because Config.Packs is a map: an unsorted walk would order the
	// pack-derived gitignore entries differently on every run.
	sort.Strings(names)

	packsDir := filepath.Join(projectRoot, ".backstop", "packs")
	manifests := make([]*pack.Manifest, 0, len(names))
	for _, name := range names {
		manifest, parseErr := pack.ParseManifestFile(filepath.Join(packsDir, filepath.FromSlash(name), "pack.yml"))
		if parseErr != nil {
			continue
		}
		manifests = append(manifests, manifest)
	}
	return manifests
}
