package distribution

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/backstop-ai/backstop-core/pkg/pack"
)

// scratchValidationDirPattern is the os.MkdirTemp pattern for the directory validation
// runs against. It is distinct from the clone temp dir so a leftover of either kind
// names which stage leaked it.
const scratchValidationDirPattern = "backstop-pack-validate-*"

// RunValidationOnScratchCopy runs pack check and pack test against a COPY of packDir,
// removes the copy on BOTH the success and the failure path, and reports any failure
// against sourceLabel rather than against the scratch directory (SPEC-056 REQ-008).
//
// WHY A COPY. pkg/packval MUTATES the directory it validates: phase 3 renders every
// tier:complete scaffold's declared sample_config into <packDir>/<scaffold.path>/ before
// running that scaffold's test command. All three of add, update and upgrade used to
// validate a tree in place and then copy and hash THAT tree, so for any pack declaring
// such a scaffold the hash recorded at add time could never be reproduced by a fresh
// clone of the same tag. The tree that reaches the install path and ComputeContentHash
// must be the pristine materialized tree that no validator has written to.
//
// WHY A LABEL AND NOT ONLY A DIRECTORY. runPackvalPipeline renders its diagnostic with
// the directory it was handed (validator.go:69-71). Reporting that directly would show
// an operator a temp path that no longer exists by the time they read it, so the failure
// is re-reported against the ORIGINAL source — the coordinate and tag for a remote pack,
// the local path for a local one.
//
// IT TAKES THE VALIDATOR AS A PARAMETER rather than reading a receiver, so all three
// commands share ONE implementation instead of three methods that drift apart.
func RunValidationOnScratchCopy(validator Validator, packDir, sourceLabel string) error {
	scratch, err := os.MkdirTemp("", scratchValidationDirPattern)
	if err != nil {
		return fmt.Errorf("creating validation scratch dir for %s: %w", sourceLabel, err)
	}
	// Removed on BOTH paths: a validation failure must not leak a temp tree either.
	defer func() { _ = os.RemoveAll(scratch) }()

	if copyErr := copyDirRecursive(packDir, scratch); copyErr != nil {
		return fmt.Errorf("preparing validation copy of %s: %w", sourceLabel, copyErr)
	}

	if checkErr := validator.RunPackCheck(scratch); checkErr != nil {
		return &ValidationError{Message: fmt.Sprintf("pack check for %s failed: %s",
			sourceLabel, scrubScratchPath(checkErr.Error(), scratch))}
	}
	if testErr := validator.RunPackTest(scratch); testErr != nil {
		return &ValidationError{Message: fmt.Sprintf("pack test for %s failed: %s",
			sourceLabel, scrubScratchPath(testErr.Error(), scratch))}
	}

	return nil
}

// scrubScratchPath removes the scratch directory from a validator's message.
//
// PREPENDING THE LABEL IS NOT ENOUGH. runPackvalPipeline embeds the directory it was
// handed inside its own text, so a wrapper that only prefixed the source would still
// hand the operator a dead path further along the same line. The scratch directory is
// replaced with the label so the message stays readable rather than developing a hole.
func scrubScratchPath(message, scratch string) string {
	if scratch == "" {
		return message
	}
	return strings.ReplaceAll(message, scratch, "the validation copy")
}

// MissingDependencyError is the fail-closed refusal a lifecycle command's
// constructor returns when a required dependency was explicitly passed as nil.
//
// A positional constructor already makes an OMITTED dependency a compile error;
// an explicitly written nil remains expressible, and this is what it becomes. It
// names BOTH the command being assembled and the dependency that was nil, so the
// diagnostic identifies the WIRING SITE rather than reporting a nil dereference
// from somewhere deep in an execution path.
type MissingDependencyError struct {
	Command    string
	Dependency string
}

func (e *MissingDependencyError) Error() string {
	return fmt.Sprintf("cannot assemble %s: its %s is nil; supply one at the wiring site", e.Command, e.Dependency)
}

// CapabilityUnavailableError is what a capability that is DECLARED but not yet
// BUILT returns instead of a vacuous success.
//
// Reference names the requirement tracking the gap, so the diagnostic points at
// the WORK rather than reading as a defect: an operator learns the capability is
// scheduled, not broken. Returning it — rather than an empty result — is what
// keeps an unbuilt capability from silently reporting "nothing to report".
//
// It is declared in this package, not in the assembly layer, because a command's
// Run must classify and propagate it and the CLI's error rendering keys a kind
// off it, while the production implementations that RETURN it live where the
// wiring lives.
type CapabilityUnavailableError struct {
	Capability string
	Reference  string
}

func (e *CapabilityUnavailableError) Error() string {
	return fmt.Sprintf("%s is declared but not yet available; it is tracked by %s", e.Capability, e.Reference)
}

// dependency names carried by *MissingDependencyError. They are the words an
// operator reads at a failed wiring site, so they are stated once here rather
// than spelled slightly differently at each of the ten guards below.
const (
	depGitCloner       = "git cloner"
	depValidator       = "validator"
	depVersionResolver = "version resolver"
	depScanner         = "scanner"
	depRemediation     = "remediation generator"
)

// AddCommand is `pack add`: it clones a pack's repository and validates the pack
// before anything is copied into the consumer project.
//
// The dependency fields are UNEXPORTED, so an AddCommand cannot be built by
// composite literal outside this package and NewAddCommand is the only assembly
// path. That is the structural half of REQ-006 — the constructor's nil guards
// only matter if there is no way around them.
type AddCommand struct {
	git       GitCloner
	validator Validator
}

// NewAddCommand assembles `pack add` over a cloner and a validator.
//
// Both are required because pack add clones AND validates. The cloner is
// required even though add's local-path branch never clones: NewExecGitCloner
// probes nothing at construction time, so a local-only consumer pays nothing for
// it, while a dependency that is "optional when a flag is set" is exactly the
// shape that let a nil reach a live code path in the first place.
func NewAddCommand(git GitCloner, validator Validator) (*AddCommand, error) {
	if git == nil {
		return nil, &MissingDependencyError{Command: "AddCommand", Dependency: depGitCloner}
	}
	if validator == nil {
		return nil, &MissingDependencyError{Command: "AddCommand", Dependency: depValidator}
	}

	return &AddCommand{git: git, validator: validator}, nil
}

// Run executes the pack add pipeline: resolve, clone, validate, install, merge
// config, record provenance, update manifest and lockfile.
//
// Validation is UNCONDITIONAL and completes before anything is copied into the
// consumer project. There is no longer a nil validator to skip it — the
// constructor makes that absence unrepresentable — so an invalid pack can no
// longer install cleanly.
func (c *AddCommand) Run(packRef string, opts AddOptions) (*AddResult, error) {
	isLocal := IsLocalPath(packRef)
	// packName is the pack's INSTALL IDENTITY and always comes from the MANIFEST
	// (SPEC-056 REQ-003). coordinate is the requested org/repository, recorded verbatim
	// and never used to build a path, a key, or an asset root.
	var packName, coordinate, version, packDir, sourceType string
	var gitRef *string
	var warnings []string
	// localRelPath holds the local pack's source dir RELATIVE TO THE PROJECT ROOT, recorded
	// in the lock so install can re-materialize it portably (empty for git-source packs).
	var localRelPath string

	if isLocal {
		local, err := resolveLocalPackSource(packRef, opts.ProjectDir)
		if err != nil {
			return nil, fmt.Errorf("resolving local pack %s: %w", packRef, err)
		}
		packName = local.packName
		packDir = local.packDir
		localRelPath = local.relPath
		sourceType = "local"

		if isPackInstalledAndCurrent(opts.ProjectDir, packName) {
			return &AddResult{PackName: packName, AlreadyCurrent: true}, nil
		}
	} else {
		// EXACTLY ONE effective version, resolved BEFORE any git subprocess runs
		// (REQ-001). Previously a bare org/name produced an empty version and the
		// pipeline cloned the ref "v", so the operator's diagnostic was a git error
		// about a nonexistent branch.
		resolvedCoordinate, effectiveVersion, resolveErr := ResolveEffectiveVersion(packRef, opts.Version)
		if resolveErr != nil {
			return nil, fmt.Errorf("pack add %s: %w", packRef, resolveErr)
		}
		coordinate = resolvedCoordinate
		sourceType = "git"

		// Clone the pack to a temporary directory.
		tmpDir, err := os.MkdirTemp("", "backstop-pack-clone-*")
		if err != nil {
			return nil, fmt.Errorf("creating temp dir: %w", err)
		}
		defer func() { _ = os.RemoveAll(tmpDir) }()

		gitURL := resolveGitURL(coordinate)
		if err := c.git.Clone(gitURL, versionTagPrefix+effectiveVersion, tmpDir); err != nil {
			return nil, fmt.Errorf("cloning pack %s: %w", coordinate, err)
		}
		packDir = tmpDir

		// THE IDENTITY GATE, before ANY consumer state is touched (REQ-002/REQ-003 and
		// REQ-007's ordering). It reads the cloned manifest and refuses a version that
		// disagrees with the tag or a name that cannot address a pack.
		identity, identityErr := ValidateRemoteIdentity(coordinate, effectiveVersion, tmpDir)
		if identityErr != nil {
			return nil, fmt.Errorf("pack add %s: %w", packRef, identityErr)
		}

		// The MANIFEST name is the identity from here on.
		packName = identity.InstallName
		version = identity.EffectiveVersion
		ref := identity.Tag
		gitRef = &ref

		// Divergence is a LOUD DIAGNOSTIC and never a refusal (REQ-006 / OQ-9 option
		// (b)). Requiring equality was rejected by name: name == coordinate is a fleet
		// CONVENTION, and a convention is something you notice breaking.
		if identity.Diverged {
			warnings = append(warnings, fmt.Sprintf(
				"pack %s declares the name %s in its manifest; installing it as %s at %s",
				coordinate, identity.ManifestName,
				identity.InstallName, filepath.Join(".backstop", "packs", identity.InstallName)))
		}

		// The already-current short-circuit MOVED here on purpose: it is keyed on the
		// INSTALL name, which for a git source is not knowable until the manifest is
		// read. Re-adding a current git pack therefore costs one shallow clone. Keying
		// it on the coordinate would reintroduce the coordinate-as-identity assumption
		// this spec removes — do not "optimize" it back.
		if isPackInstalledAndCurrent(opts.ProjectDir, packName) {
			return &AddResult{PackName: packName, AlreadyCurrent: true, Warnings: warnings}, nil
		}
	}

	// Validate against a SCRATCH COPY, BOTH branches. The local branch is not an
	// exception: for a local-path add the directory packval would mutate is the
	// OPERATOR'S OWN WORKING TREE (SPEC-056 REQ-008, spec Review Question 4).
	//
	// The label is what a failure quotes — the coordinate and tag for a git source, the
	// operator's own path for a local one.
	validationLabel := packDir
	if !isLocal {
		validationLabel = fmt.Sprintf("%s at tag %s", coordinate, versionTagPrefix+version)
	}
	if err := RunValidationOnScratchCopy(c.validator, packDir, validationLabel); err != nil {
		return nil, fmt.Errorf("pack add %s: %w", packRef, err)
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
		return nil, fmt.Errorf("tool_config conflicts:\n%s", describeConflicts(mergeResult.Conflicts))
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
		// Recorded VERBATIM for git sources and empty for local ones, whose source is
		// already recorded by LocalPath (REQ-004).
		SourceCoordinate: coordinate,
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
		PackName:         packName,
		Version:          version,
		ContentHash:      contentHash,
		InstalledPath:    installedPath,
		SourceCoordinate: coordinate,
		Warnings:         warnings,
	}, nil
}

// InstallCommand is `pack install`: the hash-verified RESTORE path that
// materializes what backstop.lock already records.
//
// Its only dependency is the cloner. Install does not re-validate — the packs it
// restores were validated when they were added, and their content hashes are
// what prove they are unchanged (DD-12).
type InstallCommand struct {
	git GitCloner
}

// NewInstallCommand assembles `pack install` over a cloner.
//
// There is deliberately no validator argument; adding one "for symmetry" would
// assert a re-validation install does not perform. The cloner is required even
// for the --cache path, which never clones, for the same reason add's is.
func NewInstallCommand(git GitCloner) (*InstallCommand, error) {
	if git == nil {
		return nil, &MissingDependencyError{Command: "InstallCommand", Dependency: depGitCloner}
	}

	return &InstallCommand{git: git}, nil
}

// Run restores the packs DECLARED in backstop.yml, reconciling them against
// backstop.lock (which supplies version/source/hash). The manifest is the source of
// truth for WHAT to install; a lock entry absent from the manifest is a stale entry
// (surfaced, not installed), and a manifest pack absent from the lock is surfaced too.
// Local packs are materialized by copying their recorded source into
// .backstop/packs/<name>/; git packs are cloned or read from cache. Content hashes are
// verified. Does NOT run validation or merge tool_config.
//
// There is no nil-cloner check: the cloner is present by construction, so the
// remaining failure modes are real git and hash failures, each carrying a
// diagnostic rather than the diagnostic-free exit the old guard produced.
func (c *InstallCommand) Run(opts InstallOptions) (*InstallResult, error) {
	lockPath := filepath.Join(opts.ProjectDir, "backstop.lock")
	lf, err := ReadLockfile(lockPath)
	if err != nil {
		return nil, fmt.Errorf("backstop.lock not found: %w", err)
	}

	result := &InstallResult{
		InstalledPacks: []string{},
		Warnings:       []string{},
	}

	// The DECLARED manifest is the source of truth for WHAT to install (Defect B). An
	// absent backstop.yml means nothing is declared: install NOTHING, no lf.Packs fallback.
	manifestPacks, manifestPresent, err := readManifestPacks(opts.ProjectDir)
	if err != nil {
		return nil, fmt.Errorf("resolving declared packs: %w", err)
	}
	if !manifestPresent {
		result.Warnings = append(result.Warnings,
			"no backstop.yml manifest found: nothing is declared, so nothing was installed")
		return result, nil
	}

	// Surface stale lock entries: present in the lock but NOT declared in the manifest
	// (e.g. a renamed slotly/go-standards). These are called out and NOT installed.
	staleNames := make([]string, 0, len(lf.Packs))
	for name := range lf.Packs {
		if _, declared := manifestPacks[name]; !declared {
			staleNames = append(staleNames, name)
		}
	}
	sort.Strings(staleNames)
	for _, name := range staleNames {
		result.Warnings = append(result.Warnings, fmt.Sprintf(
			"stale lock entry %q is not declared in backstop.yml; skipping it (run 'backstop pack remove' to clean it up)", name))
	}

	packsDir := filepath.Join(opts.ProjectDir, ".backstop", "packs")

	// Snapshot current state for atomic rollback.
	var snapshotDir string
	if info, statErr := os.Stat(packsDir); statErr == nil && info.IsDir() {
		snapshotDir, err = os.MkdirTemp("", "backstop-snapshot-*")
		if err != nil {
			return nil, fmt.Errorf("creating snapshot: %w", err)
		}
		defer func() { _ = os.RemoveAll(snapshotDir) }()
		if copyErr := copyDirRecursive(packsDir, snapshotDir); copyErr != nil {
			return nil, fmt.Errorf("snapshotting packs: %w", copyErr)
		}
	}

	// rollback best-effort restores the pre-install packs dir; its cleanup calls are
	// genuinely fire-and-forget (there is no recovery if undo itself fails).
	rollback := func() {
		_ = os.RemoveAll(packsDir)
		if snapshotDir != "" {
			_ = os.MkdirAll(filepath.Dir(packsDir), 0o755)
			_ = copyDirRecursive(snapshotDir, packsDir)
		}
	}

	// Install exactly what the manifest declares, in deterministic order.
	declaredNames := make([]string, 0, len(manifestPacks))
	for name := range manifestPacks {
		declaredNames = append(declaredNames, name)
	}
	sort.Strings(declaredNames)

	for _, name := range declaredNames {
		entry, inLock := lf.Packs[name]
		if !inLock {
			// A declared pack absent from the lock is surfaced, not silently skipped: we
			// have no version/source/hash to install it from.
			result.Warnings = append(result.Warnings, fmt.Sprintf(
				"declared pack %q is missing from backstop.lock; skipping it (run 'backstop pack add' or 'pack relock' to lock it)", name))
			continue
		}

		if entry.SourceType == "local" {
			if matErr := materializeLocalPack(opts, name, entry, packsDir); matErr != nil {
				rollback()
				return nil, matErr
			}
			result.InstalledPacks = append(result.InstalledPacks, name)
			continue
		}

		// Git packs: clone or read from cache.
		var sourceDir string
		if opts.CachePath != "" {
			sourceDir = filepath.Join(opts.CachePath, name)
			if _, statErr := os.Stat(sourceDir); statErr != nil {
				rollback()
				return nil, fmt.Errorf("pack %s not found in cache %s", name, opts.CachePath)
			}
		} else {
			tmpDir, mkErr := os.MkdirTemp("", "backstop-install-*")
			if mkErr != nil {
				rollback()
				return nil, mkErr
			}
			defer func() { _ = os.RemoveAll(tmpDir) }()

			// The repository comes from the RECORDED coordinate, never from the lock
			// key: after REQ-003 the key is the manifest name, so a divergent-name pack
			// would otherwise be uninstallable from its own lock.
			gitURL, coordWarning := RemoteURLForEntry(name, entry)
			if coordWarning != "" {
				result.Warnings = append(result.Warnings, coordWarning)
			}
			if cloneErr := c.git.Clone(gitURL, *entry.GitRef, tmpDir); cloneErr != nil {
				rollback()
				return nil, cloneErr
			}
			sourceDir = tmpDir
		}

		// Verify content hash.
		hash, hashErr := ComputeContentHash(sourceDir)
		if hashErr != nil {
			rollback()
			return nil, fmt.Errorf("computing hash for %s: %w", name, hashErr)
		}
		if hash != entry.ContentHash {
			rollback()
			return nil, fmt.Errorf("hash mismatch for pack %s: computed=%s locked=%s", name, hash, entry.ContentHash)
		}

		// Copy to .backstop/packs/.
		destDir := filepath.Join(packsDir, name)
		if mkErr := os.MkdirAll(filepath.Dir(destDir), 0o755); mkErr != nil {
			rollback()
			return nil, mkErr
		}
		if copyErr := copyDirRecursive(sourceDir, destDir); copyErr != nil {
			rollback()
			return nil, fmt.Errorf("copying pack %s: %w", name, copyErr)
		}

		result.InstalledPacks = append(result.InstalledPacks, name)
	}

	return result, nil
}

// UpdateCommand is `pack update`: it resolves the latest compatible version,
// clones it, validates it, and tamper-checks the installed copy.
//
// All three dependencies are required. The resolver in particular is no longer a
// field guarded at run time — an update with nothing to resolve versions with is
// not an update, so its absence is an assembly failure rather than a pipeline
// failure discovered after the operator has already invoked the command.
type UpdateCommand struct {
	git       GitCloner
	validator Validator
	resolver  VersionResolver
}

// NewUpdateCommand assembles `pack update` over a cloner, a validator, and a
// version resolver.
func NewUpdateCommand(git GitCloner, validator Validator, resolver VersionResolver) (*UpdateCommand, error) {
	if git == nil {
		return nil, &MissingDependencyError{Command: "UpdateCommand", Dependency: depGitCloner}
	}
	if validator == nil {
		return nil, &MissingDependencyError{Command: "UpdateCommand", Dependency: depValidator}
	}
	if resolver == nil {
		return nil, &MissingDependencyError{Command: "UpdateCommand", Dependency: depVersionResolver}
	}

	return &UpdateCommand{git: git, validator: validator, resolver: resolver}, nil
}

// Run executes the pack update pipeline: resolve the latest compatible version,
// clone it, validate it, tamper-detect the installed copy, and update the
// manifest and lockfile.
//
// The backstop.yml precheck runs BEFORE any clone, and the old runtime "version
// resolver required for update" check is gone — the constructor enforces it
// structurally, at the wiring site, rather than after the operator has already
// invoked the command.
func (c *UpdateCommand) Run(packName string, opts UpdateOptions) (*UpdateResult, error) {
	// Read current version from backstop.yml.
	currentVersion, isLocal, err := readPackVersion(opts.ProjectDir, packName)
	if err != nil {
		return nil, fmt.Errorf("reading the current version of %s: %w", packName, err)
	}

	// Local packs: no-op.
	if isLocal {
		return &UpdateResult{
			NoOp:    true,
			Message: fmt.Sprintf("pack %s is a local path pack; it updates when its source files change", packName),
		}, nil
	}

	// RESOLVE THE COORDINATE EXACTLY ONCE (REQ-005 / CLM-059). Update needs it at TWO
	// points — once for ls-remote below, once to build the clone URL further down — and
	// two independent resolutions would emit two identical fallback warnings for a single
	// invocation, which is the noise that teaches operators to ignore the signal.
	coordinate, coordWarning := CoordinateForEntry(packName, lockedEntryFor(opts.ProjectDir, packName))
	var warnings []string
	if coordWarning != "" {
		warnings = append(warnings, coordWarning)
	}

	resolved, err := c.resolver.ResolveLatestCompatible(coordinate, currentVersion)
	if err != nil {
		return nil, fmt.Errorf("resolving version: %w", err)
	}

	// Already at latest?
	if resolved == currentVersion {
		return &UpdateResult{
			OldVersion: currentVersion,
			NewVersion: currentVersion,
			NoOp:       true,
			Message:    fmt.Sprintf("pack %s is already at latest compatible version %s", packName, currentVersion),
			// A no-op update can still have fallen back on a coordinate; the
			// diagnostic must not be swallowed by the early return.
			Warnings: warnings,
		}, nil
	}

	// Major version bump? Refuse.
	if c.resolver.IsMajorBump(currentVersion, resolved) {
		return nil, fmt.Errorf("version %s is a major version bump from %s; use pack upgrade instead", resolved, currentVersion)
	}

	// Clone new version.
	tmpDir, err := os.MkdirTemp("", "backstop-update-*")
	if err != nil {
		return nil, fmt.Errorf("creating the temp dir to clone %s into: %w", packName, err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// The SAME coordinate resolved above — not a second RemoteURLForEntry call, which
	// would be the double-emission CLM-059 forbids.
	gitURL := resolveGitURL(coordinate)
	if err := c.git.Clone(gitURL, "v"+resolved, tmpDir); err != nil {
		return nil, fmt.Errorf("cloning pack %s at v%s: %w", coordinate, resolved, err)
	}

	// THE IDENTITY GATE, immediately after the clone and BEFORE tamper detection.
	// Ordering matters: DetectTamper compares the installed tree against the cloned one,
	// so running it first would report TAMPER on a version-drifted pack instead of the
	// cheaper, more specific version diagnostic — telling an operator their pack was
	// modified when the truth is the repository's manifest disagrees with its tag.
	updateIdentity, identityErr := ValidateRemoteIdentity(coordinate, resolved, tmpDir)
	if identityErr != nil {
		return nil, fmt.Errorf("pack update %s: %w", packName, identityErr)
	}
	if updateIdentity.Diverged {
		warnings = append(warnings, fmt.Sprintf(
			"pack %s declares the name %s in its manifest; it is installed as %s at %s",
			coordinate, updateIdentity.ManifestName,
			updateIdentity.InstallName, filepath.Join(".backstop", "packs", updateIdentity.InstallName)))
	}

	// Validate against a SCRATCH COPY. Beyond the hash, this is what keeps the tamper
	// comparison honest: DetectTamper below compares the installed tree against tmpDir,
	// and a contaminated tmpDir would surface the validator's own writes as ADDED files
	// — telling an operator their pack was modified when the tool modified the input.
	if err := RunValidationOnScratchCopy(c.validator, tmpDir,
		fmt.Sprintf("%s at tag v%s", packName, resolved)); err != nil {
		return nil, fmt.Errorf("pack update %s: %w", packName, err)
	}

	// Tamper detection.
	currentPackDir := filepath.Join(opts.ProjectDir, ".backstop", "packs", packName)
	if _, statErr := os.Stat(currentPackDir); statErr == nil {
		tamperResult, tamperErr := DetectTamper(currentPackDir, tmpDir)
		if tamperErr != nil {
			return nil, fmt.Errorf("tamper detection: %w", tamperErr)
		}

		if tamperResult.HasTamper && !opts.Acknowledge {
			var descriptions []string
			for _, change := range tamperResult.Changes {
				descriptions = append(descriptions, fmt.Sprintf("  %s: %s", change.Kind, change.Description))
			}
			return nil, fmt.Errorf("tamper detected in %s update:\n%s\nre-run with --acknowledge to proceed",
				packName, strings.Join(descriptions, "\n"))
		}
	}

	contentHash, err := replaceInstalledPack(opts.ProjectDir, updateIdentity.InstallName, tmpDir)
	if err != nil {
		return nil, fmt.Errorf("installing %s %s: %w", packName, resolved, err)
	}

	// Update backstop.yml.
	if err := updatePackVersion(opts.ProjectDir, updateIdentity.InstallName, resolved); err != nil {
		return nil, fmt.Errorf("recording %s %s in backstop.yml: %w", packName, resolved, err)
	}

	if err := recordGitPackInLock(opts.ProjectDir, updateIdentity.InstallName, resolved, contentHash,
		recordedCoordinateFor(opts.ProjectDir, packName)); err != nil {
		return nil, fmt.Errorf("recording %s %s in backstop.lock: %w", packName, resolved, err)
	}

	return &UpdateResult{
		OldVersion:  currentVersion,
		NewVersion:  resolved,
		ContentHash: contentHash,
		Warnings:    warnings,
	}, nil
}

// UpgradeCommand is `pack upgrade`: it crosses a major version, so beyond
// cloning and validating it scans the consumer codebase for violations the new
// major introduces and generates the remediation artifacts for them.
//
// All four dependencies are required. The scanner and remediation generator were
// previously skipped when nil, which turned an unwired upgrade into a silent
// partial success — the vacuous green this constructor exists to make impossible.
type UpgradeCommand struct {
	git         GitCloner
	validator   Validator
	scanner     Scanner
	remediation RemediationGenerator
}

// NewUpgradeCommand assembles `pack upgrade` over a cloner, a validator, a
// scanner, and a remediation generator.
func NewUpgradeCommand(git GitCloner, validator Validator, scanner Scanner, remediation RemediationGenerator) (*UpgradeCommand, error) {
	if git == nil {
		return nil, &MissingDependencyError{Command: "UpgradeCommand", Dependency: depGitCloner}
	}
	if validator == nil {
		return nil, &MissingDependencyError{Command: "UpgradeCommand", Dependency: depValidator}
	}
	if scanner == nil {
		return nil, &MissingDependencyError{Command: "UpgradeCommand", Dependency: depScanner}
	}
	if remediation == nil {
		return nil, &MissingDependencyError{Command: "UpgradeCommand", Dependency: depRemediation}
	}

	return &UpgradeCommand{git: git, validator: validator, scanner: scanner, remediation: remediation}, nil
}

// Run executes the pack upgrade pipeline: target a major version, validate it,
// scan the consumer codebase, generate remediation for what the scan found, then
// merge config and install.
//
// THE SCAN RUNS BEFORE THE TOOL-CONFIG MERGE, which is the first step that
// mutates consumer state. An upgrade whose scan capability is unavailable
// therefore fails with nothing written — no tool config, no provenance, no
// installed content, no manifest change, no lock change. Swapping the two back
// would leave a half-upgraded project behind on every such failure.
//
// Neither the scan nor the remediation generation is conditional on a dependency
// being present. Their old nil-skips produced a VACUOUS SUCCESS: an unwired
// upgrade crossed a major version and reported zero baselined violations.
func (c *UpgradeCommand) Run(packRef string, opts UpgradeOptions) (*UpgradeResult, error) {
	// Parse pack reference with explicit major version.
	packName, targetVersion := parsePackRef(packRef)

	// Read current version AND source type. The isLocal result used to be discarded
	// here, so upgrade cloned unconditionally.
	currentVersion, isLocal, err := readPackVersion(opts.ProjectDir, packName)
	if err != nil {
		return nil, fmt.Errorf("reading the current version of %s: %w", packName, err)
	}

	// ESTABLISH SOURCE TYPE BEFORE RESOLVING A COORDINATE (REQ-005). A local pack has no
	// repository at all, so asking for its coordinate is a category error that would
	// surface to the operator as a spurious fallback warning about a remote that does not
	// exist.
	if isLocal {
		return nil, fmt.Errorf("pack %s is a local path pack and has no repository to upgrade from; run 'backstop pack relock' to refresh it from its source", packName)
	}

	// The repository comes from the recorded coordinate, resolved once.
	coordinate, coordWarning := CoordinateForEntry(packName, lockedEntryFor(opts.ProjectDir, packName))
	var warnings []string
	if coordWarning != "" {
		warnings = append(warnings, coordWarning)
	}

	// Clone new version.
	tmpDir, err := os.MkdirTemp("", "backstop-upgrade-*")
	if err != nil {
		return nil, fmt.Errorf("creating the temp dir to clone %s into: %w", packName, err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	gitURL := resolveGitURL(coordinate)
	if err := c.git.Clone(gitURL, "v"+targetVersion, tmpDir); err != nil {
		return nil, fmt.Errorf("cloning pack %s at v%s: %w", coordinate, targetVersion, err)
	}

	// THE IDENTITY GATE, immediately after the clone and BEFORE the violation scan —
	// which already precedes the tool-config merge, upgrade's first consumer write.
	upgradeIdentity, identityErr := ValidateRemoteIdentity(coordinate, targetVersion, tmpDir)
	if identityErr != nil {
		return nil, fmt.Errorf("pack upgrade %s: %w", packRef, identityErr)
	}
	if upgradeIdentity.Diverged {
		warnings = append(warnings, fmt.Sprintf(
			"pack %s declares the name %s in its manifest; it is installed as %s at %s",
			coordinate, upgradeIdentity.ManifestName,
			upgradeIdentity.InstallName, filepath.Join(".backstop", "packs", upgradeIdentity.InstallName)))
	}

	// Validate against a SCRATCH COPY — the identical defect add and update carried.
	if err := RunValidationOnScratchCopy(c.validator, tmpDir,
		fmt.Sprintf("%s at tag v%s", packName, targetVersion)); err != nil {
		return nil, fmt.Errorf("pack upgrade %s: %w", packRef, err)
	}

	// Scan for violations the new major introduces — before any mutation.
	violations, err := c.scanner.ScanViolations(opts.ProjectDir, tmpDir)
	if err != nil {
		// An unavailable capability propagates UNCHANGED, so the caller can
		// recover its Capability and Reference and tell an operator what is
		// missing and what tracks it. Wrapping it into prose would leave only
		// the words "scanning violations" to act on.
		var unavailable *CapabilityUnavailableError
		if errors.As(err, &unavailable) {
			return nil, err
		}
		return nil, fmt.Errorf("scanning violations: %w", err)
	}

	// Generate a remediation bundle for what the scan found.
	var remediationBundle string
	if len(violations) > 0 {
		bundle, genErr := c.remediation.GenerateBundle(opts.ProjectDir, violations)
		if genErr != nil {
			// Rollback: don't update anything.
			return nil, fmt.Errorf("generating remediation bundle: %w", genErr)
		}
		remediationBundle = bundle
	}

	// Merge tool_config with conflict escalation. This is the first write into
	// the consumer project.
	backstopDir := filepath.Join(opts.ProjectDir, ".backstop")
	if err := os.MkdirAll(backstopDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating %s: %w", backstopDir, err)
	}

	provPath := filepath.Join(backstopDir, "pack-config-provenance.json")
	prov, err := ReadProvenance(provPath)
	if err != nil {
		return nil, fmt.Errorf("reading pack config provenance: %w", err)
	}

	mergeResult, err := MergeToolConfig(tmpDir, opts.ProjectDir, prov)
	if err != nil {
		return nil, fmt.Errorf("merging tool_config: %w", err)
	}

	if len(mergeResult.Conflicts) > 0 {
		return nil, fmt.Errorf("tool_config conflict during upgrade:\n%s", describeConflicts(mergeResult.Conflicts))
	}

	contentHash, err := replaceInstalledPack(opts.ProjectDir, upgradeIdentity.InstallName, tmpDir)
	if err != nil {
		return nil, fmt.Errorf("installing %s %s: %w", packName, targetVersion, err)
	}

	// Record provenance.
	for i := range mergeResult.Merged {
		mergeResult.Merged[i].SourcePack = upgradeIdentity.InstallName
	}
	prov.Entries = append(prov.Entries, mergeResult.Merged...)
	if err := WriteProvenance(provPath, prov); err != nil {
		return nil, fmt.Errorf("recording pack config provenance: %w", err)
	}

	// Update backstop.yml.
	if err := updatePackVersion(opts.ProjectDir, upgradeIdentity.InstallName, targetVersion); err != nil {
		return nil, fmt.Errorf("recording %s %s in backstop.yml: %w", packName, targetVersion, err)
	}

	if err := recordGitPackInLock(opts.ProjectDir, upgradeIdentity.InstallName, targetVersion, contentHash,
		recordedCoordinateFor(opts.ProjectDir, packName)); err != nil {
		return nil, fmt.Errorf("recording %s %s in backstop.lock: %w", packName, targetVersion, err)
	}

	return &UpgradeResult{
		OldVersion:          currentVersion,
		NewVersion:          targetVersion,
		ContentHash:         contentHash,
		RemediationBundle:   remediationBundle,
		BaselinedViolations: len(violations),
		Warnings:            warnings,
	}, nil
}

// localPackSource is what add resolves a local path reference down to.
type localPackSource struct {
	packName string
	packDir  string
	relPath  string
}

// resolveLocalPackSource reads a local path pack's manifest and computes the
// project-relative source path recorded in the lock.
//
// The recorded path is RELATIVE TO THE PROJECT ROOT (never absolute): backstop.lock is
// tracked/committed, so a machine-specific absolute path would be meaningless on CI,
// another checkout, or after a project move.
func resolveLocalPackSource(packRef, projectDir string) (*localPackSource, error) {
	absPath, err := filepath.Abs(packRef)
	if err != nil {
		return nil, fmt.Errorf("resolving local path: %w", err)
	}

	if _, statErr := os.Stat(filepath.Join(absPath, "pack.yml")); statErr != nil {
		return nil, fmt.Errorf("local path %s does not contain pack.yml", absPath)
	}

	if _, readErr := readPackManifest(absPath); readErr != nil {
		return nil, fmt.Errorf("reading local pack manifest: %w", readErr)
	}

	// Identity comes from the SAME reader the remote path uses (SPEC-056 CLM-038). The
	// ad-hoc untyped manifest decode that used to live here was a SECOND implementation
	// of "what is this pack called", and two readers that agree today drift tomorrow.
	//
	// ReadManifestIdentity tolerates a missing VERSION on purpose — testdata/local-pack
	// declares none and several local-path adds expect it to install. Version strictness
	// belongs to ValidateRemoteIdentity, which only the tag-cloning path calls.
	packName, _, identityErr := ReadManifestIdentity(absPath)
	if identityErr != nil {
		return nil, &IdentityError{
			Coordinate: absPath,
			Tag:        "local",
			Field:      "name",
			Problem:    identityErr.Error(),
		}
	}
	if nameErr := pack.ValidatePackName(packName); nameErr != nil {
		return nil, &IdentityError{
			Coordinate: absPath,
			Tag:        "local",
			Field:      "name",
			Problem:    nameErr.Error(),
		}
	}

	absProjectDir, absErr := filepath.Abs(projectDir)
	if absErr != nil {
		return nil, fmt.Errorf("resolving project dir: %w", absErr)
	}
	rel, relErr := filepath.Rel(absProjectDir, absPath)
	if relErr != nil {
		return nil, fmt.Errorf("computing relative local pack path: %w", relErr)
	}

	return &localPackSource{packName: packName, packDir: absPath, relPath: rel}, nil
}

// replaceInstalledPack swaps .backstop/packs/<name>/ for the freshly cloned copy
// and returns the installed content's hash. Update and upgrade both end this way.
func replaceInstalledPack(projectDir, packName, sourceDir string) (string, error) {
	installedPath := filepath.Join(projectDir, ".backstop", "packs", packName)
	if err := os.RemoveAll(installedPath); err != nil {
		return "", fmt.Errorf("removing the installed pack %s: %w", packName, err)
	}
	if err := os.MkdirAll(filepath.Dir(installedPath), 0o755); err != nil {
		return "", fmt.Errorf("creating the packs dir for %s: %w", packName, err)
	}
	if err := copyDirRecursive(sourceDir, installedPath); err != nil {
		return "", fmt.Errorf("copying %s into place: %w", packName, err)
	}

	return ComputeContentHash(installedPath)
}

// recordGitPackInLock writes the git pack's new version, ref, and hash into
// backstop.lock, creating the lockfile when it is missing.
// recordGitPackInLock rewrites a git pack's lock entry after an update or upgrade.
//
// sourceCoordinate IS A PARAMETER RATHER THAN SOMETHING THIS HELPER PRESERVES ON ITS OWN.
// The function REPLACES the whole LockEntry, so any field it does not explicitly carry is
// dropped — which is precisely how source_coordinate went missing on every rewrite before
// SPEC-056 REQ-004. Making the helper quietly read-modify-write would hide that class of
// bug rather than fix it: a caller with no coordinate to preserve now has to say so at the
// call site, in the open.
func recordGitPackInLock(projectDir, packName, version, contentHash, sourceCoordinate string) error {
	lockPath := filepath.Join(projectDir, "backstop.lock")
	lf, err := ReadLockfile(lockPath)
	if err != nil || lf == nil {
		// A missing or unreadable lock is expected here (the pack may have been
		// declared without one); fall back to a fresh lockfile.
		lf = &Lockfile{Packs: make(map[string]LockEntry)}
	}

	ref := "v" + version
	lf.Packs[packName] = LockEntry{
		Name:             packName,
		Version:          version,
		GitRef:           &ref,
		ContentHash:      contentHash,
		SourceType:       "git",
		InstallDate:      time.Now().UTC().Format(time.RFC3339),
		SourceCoordinate: sourceCoordinate,
	}

	return WriteLockfile(lockPath, lf)
}

// lockedEntryFor reads a pack's existing lock entry, or the zero entry when the lock is
// absent or unreadable. The zero entry carries no coordinate, which is exactly what
// CoordinateForEntry treats as "fall back and warn".
func lockedEntryFor(projectDir, packName string) LockEntry {
	lf, err := ReadLockfile(filepath.Join(projectDir, "backstop.lock"))
	if err != nil || lf == nil {
		return LockEntry{}
	}
	return lf.Packs[packName]
}

// recordedCoordinateFor reads the coordinate an existing lock entry already carries, so a
// rewrite carries it forward VERBATIM.
//
// It deliberately does NOT re-derive the coordinate from the pack name when the field is
// absent: that derivation is exactly what REQ-003 removes, and inventing a value here
// would write a guess into the lock as though it were recorded fact. An entry with no
// coordinate carries none forward; REQ-005's accessor is what decides, at RESOLUTION
// time, that the pack name is the compatibility fallback — and warns while doing it.
func recordedCoordinateFor(projectDir, packName string) string {
	lf, err := ReadLockfile(filepath.Join(projectDir, "backstop.lock"))
	if err != nil || lf == nil {
		return ""
	}
	return lf.Packs[packName].SourceCoordinate
}

// describeConflicts renders tool_config conflicts one per line, naming the file,
// the setting, and both values so an operator can see what disagrees.
func describeConflicts(conflicts []ConfigConflict) string {
	msgs := make([]string, 0, len(conflicts))
	for _, conflict := range conflicts {
		msgs = append(msgs, fmt.Sprintf("%s: %s (pack=%s, current=%s)",
			conflict.ConfigFile, conflict.SettingKey, conflict.PackValue, conflict.CurrentValue))
	}
	return strings.Join(msgs, "\n")
}
