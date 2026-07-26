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

// InstallContractsLocalPack installs the packs/contracts/ source as a LOCAL pack
// into projectDir through the REAL pack add path, and returns the AddResult so
// callers can assert the installed path and content hash.
//
// It RECEIVES the assembled command rather than building one. A library-layer
// helper that constructed its own validator would be the internal defaulting
// REQ-005 forbids, and it would leave a test double indistinguishable from
// production wiring — the caller's choice of dependencies is the only thing that
// distinguishes them, so the caller has to make it.
//
// Validation is unconditional: whatever validator the supplied command carries
// runs over the contracts pack. There is no variant that skips it.
func InstallContractsLocalPack(add *AddCommand, repoRoot, projectDir string) (*AddResult, error) {
	res, err := add.Run(ContractsPackSourceDir(repoRoot), AddOptions{ProjectDir: projectDir})
	if err != nil {
		return nil, fmt.Errorf("installing contracts local pack: %w", err)
	}
	return res, nil
}
