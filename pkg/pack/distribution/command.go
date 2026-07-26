package distribution

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

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
	isLocal := isLocalPath(packRef)
	var packName, version, packDir, sourceType string
	var gitRef *string
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
		if err := c.git.Clone(gitURL, "v"+version, tmpDir); err != nil {
			return nil, fmt.Errorf("cloning pack %s: %w", packName, err)
		}
		packDir = tmpDir
	}

	// Validate: run pack check and then pack test, both BEFORE any copy.
	if err := c.validator.RunPackCheck(packDir); err != nil {
		return nil, fmt.Errorf("pack check for %s: %w", packName, err)
	}
	if err := c.validator.RunPackTest(packDir); err != nil {
		return nil, fmt.Errorf("pack test for %s: %w", packName, err)
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

			gitURL := resolveGitURL(name)
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

	resolved, err := c.resolver.ResolveLatestCompatible(packName, currentVersion)
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

	gitURL := resolveGitURL(packName)
	if err := c.git.Clone(gitURL, "v"+resolved, tmpDir); err != nil {
		return nil, fmt.Errorf("cloning pack %s at v%s: %w", packName, resolved, err)
	}

	// Validate.
	if err := c.validator.RunPackCheck(tmpDir); err != nil {
		return nil, fmt.Errorf("pack check for %s: %w", packName, err)
	}
	if err := c.validator.RunPackTest(tmpDir); err != nil {
		return nil, fmt.Errorf("pack test for %s: %w", packName, err)
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

	contentHash, err := replaceInstalledPack(opts.ProjectDir, packName, tmpDir)
	if err != nil {
		return nil, fmt.Errorf("installing %s %s: %w", packName, resolved, err)
	}

	// Update backstop.yml.
	if err := updatePackVersion(opts.ProjectDir, packName, resolved); err != nil {
		return nil, fmt.Errorf("recording %s %s in backstop.yml: %w", packName, resolved, err)
	}

	if err := recordGitPackInLock(opts.ProjectDir, packName, resolved, contentHash); err != nil {
		return nil, fmt.Errorf("recording %s %s in backstop.lock: %w", packName, resolved, err)
	}

	return &UpdateResult{
		OldVersion:  currentVersion,
		NewVersion:  resolved,
		ContentHash: contentHash,
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

	// Read current version.
	currentVersion, _, err := readPackVersion(opts.ProjectDir, packName)
	if err != nil {
		return nil, fmt.Errorf("reading the current version of %s: %w", packName, err)
	}

	// Clone new version.
	tmpDir, err := os.MkdirTemp("", "backstop-upgrade-*")
	if err != nil {
		return nil, fmt.Errorf("creating the temp dir to clone %s into: %w", packName, err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	gitURL := resolveGitURL(packName)
	if err := c.git.Clone(gitURL, "v"+targetVersion, tmpDir); err != nil {
		return nil, fmt.Errorf("cloning pack %s at v%s: %w", packName, targetVersion, err)
	}

	// Validate.
	if err := c.validator.RunPackCheck(tmpDir); err != nil {
		return nil, fmt.Errorf("pack check for %s: %w", packName, err)
	}
	if err := c.validator.RunPackTest(tmpDir); err != nil {
		return nil, fmt.Errorf("pack test for %s: %w", packName, err)
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

	contentHash, err := replaceInstalledPack(opts.ProjectDir, packName, tmpDir)
	if err != nil {
		return nil, fmt.Errorf("installing %s %s: %w", packName, targetVersion, err)
	}

	// Record provenance.
	for i := range mergeResult.Merged {
		mergeResult.Merged[i].SourcePack = packName
	}
	prov.Entries = append(prov.Entries, mergeResult.Merged...)
	if err := WriteProvenance(provPath, prov); err != nil {
		return nil, fmt.Errorf("recording pack config provenance: %w", err)
	}

	// Update backstop.yml.
	if err := updatePackVersion(opts.ProjectDir, packName, targetVersion); err != nil {
		return nil, fmt.Errorf("recording %s %s in backstop.yml: %w", packName, targetVersion, err)
	}

	if err := recordGitPackInLock(opts.ProjectDir, packName, targetVersion, contentHash); err != nil {
		return nil, fmt.Errorf("recording %s %s in backstop.lock: %w", packName, targetVersion, err)
	}

	return &UpgradeResult{
		OldVersion:          currentVersion,
		NewVersion:          targetVersion,
		ContentHash:         contentHash,
		RemediationBundle:   remediationBundle,
		BaselinedViolations: len(violations),
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

	// The name is read separately from the manifest model, which carries only the
	// tool_config the merge needs. A pack.yml that parses but declares no name is a
	// pack that cannot be addressed, so it is rejected rather than installed nameless.
	packName := ""
	var nameFields map[string]interface{}
	if err := yaml.Unmarshal(readFileOrNil(filepath.Join(absPath, "pack.yml")), &nameFields); err == nil {
		if declared, ok := nameFields["name"].(string); ok {
			packName = declared
		}
	}
	if packName == "" {
		return nil, fmt.Errorf("local pack manifest missing name")
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
func recordGitPackInLock(projectDir, packName, version, contentHash string) error {
	lockPath := filepath.Join(projectDir, "backstop.lock")
	lf, err := ReadLockfile(lockPath)
	if err != nil || lf == nil {
		// A missing or unreadable lock is expected here (the pack may have been
		// declared without one); fall back to a fresh lockfile.
		lf = &Lockfile{Packs: make(map[string]LockEntry)}
	}

	ref := "v" + version
	lf.Packs[packName] = LockEntry{
		Name:        packName,
		Version:     version,
		GitRef:      &ref,
		ContentHash: contentHash,
		SourceType:  "git",
		InstallDate: time.Now().UTC().Format(time.RFC3339),
	}

	return WriteLockfile(lockPath, lf)
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
