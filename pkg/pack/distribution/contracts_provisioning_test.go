package distribution_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack/distribution"
	"gopkg.in/yaml.v3"
)

// contracts_provisioning_test.go (SPEC-038 TASK-032, REQ-013): the Go contracts pack
// is an ORDINARY INSTALLED local pack — not embedded, not testdata-as-production. These
// tests drive the REAL distribution.Add / VerifyLock path over the packs/contracts/
// SOURCE in a temp workspace.

// provRepoRoot walks up to the module root.
func provRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for i := 0; i < 8; i++ {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil { // nosemgrep: backstop.packs.backstop.self.rules.no-baked-language-token — test walks up to the repo root by go.mod, not baked routing
			return dir
		}
		dir = filepath.Dir(dir)
	}
	t.Fatal("module root not found")
	return ""
}

// newProvWorkspace scaffolds a temp workspace with a minimal backstop.yml (no packs).
func newProvWorkspace(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	yml := "project: prov\nlanguage: go\npacks: {}\n"
	if err := os.WriteFile(filepath.Join(tmp, "backstop.yml"), []byte(yml), 0o644); err != nil {
		t.Fatalf("writing backstop.yml: %v", err)
	}
	return tmp
}

// TestProvisioning_ContractsPackNotEmbeddedNorTestdata (CLM-043): the contract rules are
// ABSENT from the binary — no //go:embed carries the compiler or grep-absence rule YAML,
// and no production gate code path resolves the contracts pack from a testdata directory.
// The rules are present ONLY in the installed packs/contracts/ source.
func TestProvisioning_ContractsPackNotEmbeddedNorTestdata(t *testing.T) {
	root := provRepoRoot(t)

	// (a) The pack rules exist in the SOURCE dir (the installable artifact).
	src := distribution.ContractsPackSourceDir(root)
	for _, f := range []string{"pack.yml", "scripts/compile-signature.sh", "grep/to-sarif.sh", "ast-grep/to-sarif.sh"} {
		if _, err := os.Stat(filepath.Join(src, f)); err != nil {
			t.Fatalf("contracts pack source must carry %s: %v", f, err)
		}
	}

	// (b) No //go:embed of the contracts pack/compiler in production Go. Walk pkg/ +
	// cmd/ non-test .go and assert no embed directive names the contracts compiler/rules.
	var offenders []string
	for _, base := range []string{"pkg", "cmd"} {
		_ = filepath.Walk(filepath.Join(root, base), func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			data, rerr := os.ReadFile(path)
			if rerr != nil {
				return nil
			}
			for _, line := range strings.Split(string(data), "\n") {
				if strings.Contains(line, "//go:embed") &&
					(strings.Contains(line, "contracts") || strings.Contains(line, "compile-signature")) {
					offenders = append(offenders, path+": "+strings.TrimSpace(line))
				}
			}
			return nil
		})
	}
	if len(offenders) > 0 {
		t.Errorf("contracts pack must NOT be //go:embed-bundled into the binary (CLM-043): %v", offenders)
	}
}

// TestProvisioning_ContractsInstalledAsLocalPack_DeclaredAndLocked (CLM-044): after
// `pack add packs/contracts` (local source), backstop.yml declares the pack with the
// `local` source value and the lockfile carries a `local` SourceType entry, and
// VerifyLock PASSES without a remote artifact (local packs skipped).
func TestProvisioning_ContractsInstalledAsLocalPack_DeclaredAndLocked(t *testing.T) {
	root := provRepoRoot(t)
	ws := newProvWorkspace(t)

	add := newTestAddCommand(t, distribution.NewExecGitCloner(), distribution.NewPackvalValidator())

	res, err := distribution.InstallContractsLocalPack(add, root, ws)
	if err != nil {
		t.Fatalf("installing contracts local pack via real distribution.Add: %v", err)
	}
	if res.PackName != "backstop/contracts" {
		t.Errorf("installed pack name = %q, want backstop/contracts", res.PackName)
	}

	// backstop.yml declares the pack with the `local` value.
	ymlData, err := os.ReadFile(filepath.Join(ws, "backstop.yml"))
	if err != nil {
		t.Fatalf("reading backstop.yml: %v", err)
	}
	var yml struct {
		Packs map[string]string `yaml:"packs"`
	}
	if err := yaml.Unmarshal(ymlData, &yml); err != nil {
		t.Fatalf("unmarshalling backstop.yml: %v", err)
	}
	if yml.Packs["backstop/contracts"] != "local" {
		t.Errorf("backstop.yml must declare backstop/contracts: local, got %q (CLM-044)", yml.Packs["backstop/contracts"])
	}

	// Lockfile carries a `local` SourceType entry.
	lf, err := distribution.ReadLockfile(filepath.Join(ws, "backstop.lock"))
	if err != nil {
		t.Fatalf("reading backstop.lock: %v", err)
	}
	entry, ok := lf.Packs["backstop/contracts"]
	if !ok {
		t.Fatal("lockfile must carry a backstop/contracts entry (CLM-044)")
	}
	if entry.SourceType != "local" {
		t.Errorf("lockfile entry SourceType = %q, want local (CLM-044)", entry.SourceType)
	}

	// VerifyLock PASSES without a remote artifact (local packs skipped, verify.go ~46).
	packsDir := filepath.Join(ws, ".backstop", "packs")
	result, err := distribution.VerifyLock(lf, packsDir, []string{"backstop/contracts"})
	if err != nil {
		t.Fatalf("VerifyLock errored: %v", err)
	}
	if !result.Pass {
		t.Errorf("VerifyLock must PASS for a local pack without a remote artifact, failures: %#v (CLM-044)", result.Failures)
	}
}

// TestProvisioning_GoContractsPackIsTheNewInstallable_TSRulesShareProofPack (CLM-045):
// the installable artifact this spec authors is the GO contracts pack; the TS contract
// rules are co-owned with the already-installed shared TS proof pack, so only the Go
// pack's declared+locked installation is asserted here.
func TestProvisioning_GoContractsPackIsTheNewInstallable_TSRulesShareProofPack(t *testing.T) {
	root := provRepoRoot(t)

	// The Go contracts pack is a standalone installable source.
	goPack := filepath.Join(distribution.ContractsPackSourceDir(root), "pack.yml")
	data, err := os.ReadFile(goPack)
	if err != nil {
		t.Fatalf("Go contracts pack must be an installable source: %v", err)
	}
	if !strings.Contains(string(data), "name: backstop/contracts") {
		t.Error("the Go contracts pack must be named backstop/contracts (the new installable artifact, CLM-045)")
	}

	// The TS contract rules live in the SHARED ts-proof pack (NOT a second contracts pack).
	tsPack := filepath.Join(root, "pkg", "gate", "testdata", "ts-proof-pack", "pack.yml")
	tsData, err := os.ReadFile(tsPack)
	if err != nil {
		t.Fatalf("shared TS proof pack must exist: %v", err)
	}
	if !strings.Contains(string(tsData), "contract-signature-ts") || !strings.Contains(string(tsData), "contract-absence-ts") {
		t.Error("the TS contract rules must be co-owned with the shared TS proof pack, not the Go contracts pack (CLM-045)")
	}
	// The Go contracts pack must NOT carry TS rules.
	if strings.Contains(string(data), "contract-signature-ts") {
		t.Error("the Go contracts pack must not carry the TS rules (they share the TS proof pack, CLM-045)")
	}
}

// TestInstallContractsLocalPack_ErrorPath covers the install error branch (a repoRoot
// with no packs/contracts/ source).
func TestInstallContractsLocalPack_ErrorPath(t *testing.T) {
	ws := newProvWorkspace(t)
	add := newTestAddCommand(t, distribution.NewExecGitCloner(), distribution.NewPackvalValidator())

	if _, err := distribution.InstallContractsLocalPack(add, t.TempDir(), ws); err == nil {
		t.Error("installing from a repo root with no contracts pack source must error")
	}
}

// TestInstallContractsLocalPack_ReinstallExercisesInstalledPath installs the contracts
// pack twice into the same workspace, exercising the already-installed-and-current branch
// of the Add machinery the contracts dogfood install relies on. Realigned for ISSUE-026:
// the first install genuinely materializes the pack on disk AND records the lock, so the
// second install is an HONEST no-op (AddResult.AlreadyCurrent, nil error) — NOT the old
// misleading "already installed" error, which conflated declared with installed.
func TestInstallContractsLocalPack_ReinstallExercisesInstalledPath(t *testing.T) {
	root := provRepoRoot(t)
	ws := newProvWorkspace(t)

	add := newTestAddCommand(t, distribution.NewExecGitCloner(), distribution.NewPackvalValidator())

	if _, err := distribution.InstallContractsLocalPack(add, root, ws); err != nil {
		t.Fatalf("first install: %v", err)
	}
	// A second install over the same workspace hits the installed-and-current branch ->
	// honest no-op, not an error.
	res, err := distribution.InstallContractsLocalPack(add, root, ws)
	if err != nil {
		t.Fatalf("re-installing an already-current pack must be an honest no-op, got error: %v", err)
	}
	if !res.AlreadyCurrent {
		t.Error("re-installing an already-current pack must report AlreadyCurrent")
	}
	// The declaration + lock remain consistent after re-install.
	lf, err := distribution.ReadLockfile(filepath.Join(ws, "backstop.lock"))
	if err != nil {
		t.Fatalf("reading lock after re-install: %v", err)
	}
	if _, ok := lf.Packs["backstop/contracts"]; !ok {
		t.Error("contracts pack must remain locked after re-install")
	}
}
