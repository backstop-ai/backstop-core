package distribution

import (
	"fmt"
	"path/filepath"
)

// contracts_local_install.go (SPEC-038 REQ-013/Phase 7a) is the local-pack install
// helper the contracts provisioning + E2E tests drive. It installs the packs/contracts/
// SOURCE as a LOCAL pack via the REAL distribution.Add path (declared
// `backstop/contracts: local` in backstop.yml + a `local` lockfile entry) into a target
// project workspace, and returns the resolved install handle so a later gate run resolves
// the contracts pack from the INSTALLED (local) declaration — NEVER from testdata, with
// NO //go:embed and NO baked path (CLM-043/044/045). It REUSES the existing
// distribution.Add / VerifyLock primitives; it does NOT re-implement installation.

// ContractsPackSourceDir returns the in-repo installable contracts pack SOURCE at
// packs/contracts/ (location B in the PROVISIONING notes), relative to the repo root.
// This is the ORDINARY pack source a real gate run resolves from the installed local
// declaration — distinct from the engine/rule TDD testdata packs (location A).
func ContractsPackSourceDir(repoRoot string) string {
	return filepath.Join(repoRoot, "packs", "contracts")
}

// InstallContractsLocalPack installs the packs/contracts/ source as a LOCAL pack into
// projectDir via the REAL distribution.Add path. A nil Validator skips the stale pack
// check/test phases (the dogfood install does not gate on them) — the install path
// itself is REAL, not mocked. It returns the AddResult so callers can assert the
// installed path / content hash.
func InstallContractsLocalPack(repoRoot, projectDir string) (*AddResult, error) {
	return InstallContractsLocalPackWithValidator(repoRoot, projectDir, nil)
}

// InstallContractsLocalPackWithValidator is the validator-aware form: when validator is
// non-nil, the install runs the pack-check / pack-test phases over the contracts pack via
// the REAL Add path (the dogfood install of a contracts pack may gate on its own checks).
// A nil validator skips those phases. It exists so a caller can install the contracts pack
// AND assert the pack passes its own checks through the same real provisioning path.
func InstallContractsLocalPackWithValidator(repoRoot, projectDir string, validator Validator) (*AddResult, error) {
	res, err := Add(ContractsPackSourceDir(repoRoot), AddOptions{ProjectDir: projectDir, Validator: validator})
	if err != nil {
		return nil, fmt.Errorf("installing contracts local pack: %w", err)
	}
	return res, nil
}
