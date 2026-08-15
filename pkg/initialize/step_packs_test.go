package initialize

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

// TestInit_WithoutPackFlagInstallsNothingAndSaysSo (SPEC-069 CLM-049).
//
// Bare init installs ZERO packs and SAYS SO, naming `backstop pack add` as the way to
// add them. A SILENT skip fails this claim: it is what keeps a pack roster out of core
// — there is nothing for init to install that a consumer did not name, and the report
// has to say that rather than leave the consumer wondering.
func TestInit_WithoutPackFlagInstallsNothingAndSaysSo(t *testing.T) {
	installer, _, _, _, _ := defaultFakes()

	report, err := stepPacks("/does/not/matter", nil, installer)
	if err != nil {
		t.Fatalf("bare init errored in the packs step: %v", err)
	}

	if len(installer.refs) != 0 {
		t.Fatalf("bare init installed %v; with no --pack it must install nothing at all", installer.refs)
	}
	if report.Outcome == OutcomeBrokenPromise {
		t.Fatalf("bare init failed the packs step: %s", report.Detail)
	}
	if !strings.Contains(report.Detail, "backstop pack add") {
		t.Fatalf("the packs report does not name `backstop pack add` as the way to add packs.\ngot: %s", report.Detail)
	}
	if strings.TrimSpace(report.Detail) == "" {
		t.Fatal("the packs step skipped silently; a silent skip leaves the consumer with no packs and no idea why")
	}
}

// TestInit_InstallsExactlyTheSuppliedPackRefsInOrder (SPEC-069 CLM-050).
//
// `--pack` is repeatable; exactly the refs supplied, in the ORDER supplied, and no
// others. Order is asserted rather than set-membership because pack installation is
// sequential and a later pack's config merge can depend on an earlier one's.
func TestInit_InstallsExactlyTheSuppliedPackRefsInOrder(t *testing.T) {
	installer, _, _, _, _ := defaultFakes()
	supplied := []string{"acme/first@1.0.0", "acme/second@2.1.0", "other/third@0.4.2"}

	report, err := stepPacks("/project", supplied, installer)
	if err != nil {
		t.Fatalf("packs step errored: %v", err)
	}
	if report.Outcome != OutcomeDelivered {
		t.Fatalf("packs step reported %v (%s), want OutcomeDelivered", report.Outcome, report.Detail)
	}

	if len(installer.refs) != len(supplied) {
		t.Fatalf("installed %v, want exactly %v", installer.refs, supplied)
	}
	for i := range supplied {
		if installer.refs[i] != supplied[i] {
			t.Fatalf("install %d was %q, want %q; the refs must be installed in the order the consumer supplied them",
				i, installer.refs[i], supplied[i])
		}
	}
}

// TestInit_PackSelectionIsUnaffectedByProjectContents (SPEC-069 CLM-052).
//
// Two fixture projects with DIFFERENT on-disk contents and the SAME `--pack` arguments
// install the identical pack set. Nothing on disk influences pack selection: init has
// no concept of a "primary language" and holds no pack roster.
func TestInit_PackSelectionIsUnaffectedByProjectContents(t *testing.T) {
	supplied := []string{"acme/first@1.0.0", "acme/second@2.1.0"}

	// The two fixture trees differ in the filename AND the content a detecting
	// implementation might key on, and share only their --pack arguments. They live in
	// testdata, so the marker names put no literal in the init source set the denylist
	// claims scan. The packs step writes nothing, so reading them in place is safe.
	projectA := filepath.Join("testdata", "pack-selection-a")
	projectB := filepath.Join("testdata", "pack-selection-b")
	if !exists(projectA, "marker-a.txt") || !exists(projectB, "marker-b.txt") {
		t.Fatal("the two pack-selection fixture projects are missing; without differing contents the claim cannot falsify anything")
	}

	installerA, _, _, _, _ := defaultFakes()
	if _, err := stepPacks(projectA, supplied, installerA); err != nil {
		t.Fatalf("packs step errored over project A: %v", err)
	}
	installerB, _, _, _, _ := defaultFakes()
	if _, err := stepPacks(projectB, supplied, installerB); err != nil {
		t.Fatalf("packs step errored over project B: %v", err)
	}

	if len(installerA.refs) != len(installerB.refs) {
		t.Fatalf("project A installed %v and project B installed %v; the on-disk contents changed the pack set",
			installerA.refs, installerB.refs)
	}
	for i := range installerA.refs {
		if installerA.refs[i] != installerB.refs[i] {
			t.Fatalf("install %d differed between the two projects: %q vs %q", i, installerA.refs[i], installerB.refs[i])
		}
	}
}

// TestInit_PacksStepRefusesALocalPathBeforeInstallingAnything is the pkg/initialize
// half of REQ-018's refusal. It is not a mandated name — CLM-087's mandated test
// asserts the CONFIG-ERROR EXIT and lives in cmd/backstop — but the ORDER is asserted
// here, where the step is: the refusal must happen BEFORE any install runs, or a
// portable ref installed ahead of an unportable one leaves the lock half-written.
func TestInit_PacksStepRefusesALocalPathBeforeInstallingAnything(t *testing.T) {
	installer, _, _, _, _ := defaultFakes()

	// The portable ref comes FIRST, so an implementation that classified as it went
	// would already have installed it by the time it reached the local path.
	_, err := stepPacks("/project", []string{"acme/portable@1.0.0", "./local-pack"}, installer)
	if err == nil {
		t.Fatal("a local filesystem --pack value was accepted; it produces a machine-specific local_path lock entry")
	}
	if len(installer.refs) != 0 {
		t.Fatalf("installs ran before the refusal: %v. Every ref is classified BEFORE any install, so a refused run leaves nothing half-installed",
			installer.refs)
	}
	message := err.Error()
	if !strings.Contains(message, "./local-pack") {
		t.Fatalf("the refusal does not quote the offending ref.\ngot: %s", message)
	}
	if !strings.Contains(message, "backstop pack add") {
		t.Fatalf("the refusal does not point at `backstop pack add` after init.\ngot: %s", message)
	}
}

// TestInit_PacksStepSurfacesAnInstallFailureAsABrokenPromise asserts the step reports
// a failed install as a promise init did not keep — the consumer ASKED for that pack.
// Additive: it is what makes the exit-code matrix's packs row reachable.
func TestInit_PacksStepSurfacesAnInstallFailureAsABrokenPromise(t *testing.T) {
	installer, _, _, _, _ := defaultFakes()
	installer.failures["acme/broken@1.0.0"] = errors.New("the remote refused the clone")

	report, err := stepPacks("/project", []string{"acme/broken@1.0.0"}, installer)
	if err != nil {
		t.Fatalf("an install failure must be a reported step outcome, not a config error: %v", err)
	}
	if report.Outcome != OutcomeBrokenPromise {
		t.Fatalf("a failed install reported %v, want OutcomeBrokenPromise", report.Outcome)
	}
	if !strings.Contains(report.Detail, "the remote refused the clone") {
		t.Fatalf("the report does not surface the underlying error.\ngot: %s", report.Detail)
	}
}
