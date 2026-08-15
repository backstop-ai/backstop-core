package artifact

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestResolveRoot_AbsentDeclarationResolvesToProjectRootUnconfigured pins CLM-031:
// an absent artifact_root resolves to the project root marked UNCONFIGURED. This is
// the framework exception that lets backstop-core keep its repo-root layout without
// configuring anything, so it must not error.
func TestResolveRoot_AbsentDeclarationResolvesToProjectRootUnconfigured(t *testing.T) {
	dir := t.TempDir()

	root, err := ResolveRoot(dir, "")
	if err != nil {
		t.Fatalf("ResolveRoot(%q, \"\") returned error %v, want nil", dir, err)
	}
	wantPath, err := filepath.Abs(dir)
	if err != nil {
		t.Fatalf("filepath.Abs(%q): %v", dir, err)
	}
	if root.Path != filepath.Clean(wantPath) {
		t.Errorf("root.Path = %q, want %q", root.Path, filepath.Clean(wantPath))
	}
	if root.Configured {
		t.Error("root.Configured = true, want false — an absent declaration is unconfigured")
	}
	if root.Declared != "" {
		t.Errorf("root.Declared = %q, want empty", root.Declared)
	}
}

// TestResolveRoot_ConfiguredExistingDirectoryResolvesConfigured pins CLM-032.
func TestResolveRoot_ConfiguredExistingDirectoryResolvesConfigured(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".backstop"), 0o755); err != nil {
		t.Fatalf("creating .backstop: %v", err)
	}

	root, err := ResolveRoot(dir, ".backstop")
	if err != nil {
		t.Fatalf("ResolveRoot(%q, \".backstop\") returned error %v, want nil", dir, err)
	}
	abs, err := filepath.Abs(filepath.Join(dir, ".backstop"))
	if err != nil {
		t.Fatalf("filepath.Abs: %v", err)
	}
	if root.Path != filepath.Clean(abs) {
		t.Errorf("root.Path = %q, want %q", root.Path, filepath.Clean(abs))
	}
	if !root.Configured {
		t.Error("root.Configured = false, want true")
	}
	if root.Declared != ".backstop" {
		t.Errorf("root.Declared = %q, want \".backstop\"", root.Declared)
	}
}

// TestResolveRoot_ConfiguredAbsentRootReturnsTypedErrorNotFallback pins CLM-033. The
// SILENT FALLBACK is the bug this claim exists to forbid, so asserting only that an
// error came back would leave it undetected — the returned Root must NOT be the
// project root.
func TestResolveRoot_ConfiguredAbsentRootReturnsTypedErrorNotFallback(t *testing.T) {
	dir := t.TempDir()

	root, err := ResolveRoot(dir, ".backstop")
	if err == nil {
		t.Fatalf("ResolveRoot on an absent configured root returned nil error and root %+v, want a typed error", root)
	}

	var missing *RootMissingError
	if !errors.As(err, &missing) {
		t.Fatalf("error %v (%T) is not a *RootMissingError", err, err)
	}
	if missing.Declared != ".backstop" {
		t.Errorf("RootMissingError.Declared = %q, want \".backstop\"", missing.Declared)
	}
	if missing.Path == "" {
		t.Error("RootMissingError.Path is empty; the diagnostic must name the path that was absent")
	}

	abs, absErr := filepath.Abs(dir)
	if absErr != nil {
		t.Fatalf("filepath.Abs: %v", absErr)
	}
	if root.Path == filepath.Clean(abs) {
		t.Errorf("root.Path fell back to the project root %q; a silent fallback is exactly what this claim forbids", root.Path)
	}
}

// TestResolveRoot_ConfiguredEmptyDirectoryResolves pins CLM-034 and pairs with
// CLM-033 above. An EXISTING-but-empty root is a legitimate state — it is what
// `backstop init` produces — and tightening "empty is suspicious" into "empty is a
// failure" breaks the init seed's acceptance bar.
func TestResolveRoot_ConfiguredEmptyDirectoryResolves(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".backstop"), 0o755); err != nil {
		t.Fatalf("creating .backstop: %v", err)
	}

	root, err := ResolveRoot(dir, ".backstop")
	if err != nil {
		t.Fatalf("ResolveRoot on an existing EMPTY root returned error %v, want nil", err)
	}
	if !root.Configured {
		t.Error("root.Configured = false, want true")
	}
	abs, absErr := filepath.Abs(filepath.Join(dir, ".backstop"))
	if absErr != nil {
		t.Fatalf("filepath.Abs: %v", absErr)
	}
	if root.Path != filepath.Clean(abs) {
		t.Errorf("root.Path = %q, want %q", root.Path, filepath.Clean(abs))
	}
}

// TestResolveRoot_ConfiguredRootThatIsAFileReturnsTypedError pins CLM-035. The type
// is *RootInvalidError, NOT *RootMissingError, and the distinction is load-bearing
// across plans: SPEC-070's doctor branches on exactly these two types —
// *RootMissingError reports the DECLARED value, *RootInvalidError reports the REASON.
// RootMissingError carries no Reason field, so routing not-a-directory there would
// leave the actual reason unnameable.
func TestResolveRoot_ConfiguredRootThatIsAFileReturnsTypedError(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".backstop"), []byte("not a directory\n"), 0o644); err != nil {
		t.Fatalf("writing .backstop as a file: %v", err)
	}

	_, err := ResolveRoot(dir, ".backstop")
	if err == nil {
		t.Fatal("ResolveRoot on a root that is a FILE returned nil error, want a typed error")
	}

	var invalid *RootInvalidError
	if !errors.As(err, &invalid) {
		t.Fatalf("error %v (%T) is not a *RootInvalidError", err, err)
	}
	if invalid.Reason == "" {
		t.Fatal("RootInvalidError.Reason is empty; the reason must be nameable by a caller")
	}
	if !strings.Contains(invalid.Reason, "directory") {
		t.Errorf("RootInvalidError.Reason = %q, want it to name not-a-directory", invalid.Reason)
	}

	var missing *RootMissingError
	if errors.As(err, &missing) {
		t.Error("a not-a-directory root resolved to *RootMissingError; it must be *RootInvalidError so the reason stays nameable")
	}
}

// TestResolveRoot_AbsoluteDeclaredRootRejected pins CLM-036. Note the deliberate
// asymmetry the contract records: the DECLARED value is project-relative by rule
// while the RESOLVED Path is absolute by guarantee — two different strings.
func TestResolveRoot_AbsoluteDeclaredRootRejected(t *testing.T) {
	dir := t.TempDir()
	absolute := filepath.Join(t.TempDir(), "elsewhere")
	if err := os.MkdirAll(absolute, 0o755); err != nil {
		t.Fatalf("creating elsewhere: %v", err)
	}

	_, err := ResolveRoot(dir, absolute)
	if err == nil {
		t.Fatalf("ResolveRoot accepted an ABSOLUTE declared root %q, want rejection", absolute)
	}
	var invalid *RootInvalidError
	if !errors.As(err, &invalid) {
		t.Fatalf("error %v (%T) is not a *RootInvalidError", err, err)
	}
	if invalid.Reason == "" {
		t.Error("RootInvalidError.Reason is empty for an absolute declared root")
	}
	if invalid.Declared != absolute {
		t.Errorf("RootInvalidError.Declared = %q, want %q", invalid.Declared, absolute)
	}
}

// TestResolveRoot_EscapingDeclaredRootRejected pins CLM-037. Both the plain "../"
// form and the sneakier "sub/../../" form are rejected — the comparison is a
// filepath.Rel check, not a string-prefix test, so a sibling directory sharing a
// name prefix is not conflated with an escape.
func TestResolveRoot_EscapingDeclaredRootRejected(t *testing.T) {
	dir := t.TempDir()

	for _, declared := range []string{"../outside", "sub/../../outside"} {
		_, err := ResolveRoot(dir, declared)
		if err == nil {
			t.Errorf("ResolveRoot accepted an ESCAPING declared root %q, want rejection", declared)
			continue
		}
		var invalid *RootInvalidError
		if !errors.As(err, &invalid) {
			t.Errorf("declared %q: error %v (%T) is not a *RootInvalidError", declared, err, err)
			continue
		}
		if invalid.Reason == "" {
			t.Errorf("declared %q: RootInvalidError.Reason is empty", declared)
		}
	}
}

// TestResolveRoot_TypedErrorsRenderTheirOwnDiagnostic pins the reporting half of the
// two typed errors. SPEC-070's doctor renders these strings to an operator, so an
// error whose message omits the declared value or the reason is unusable there even
// though errors.As still classifies it.
func TestResolveRoot_TypedErrorsRenderTheirOwnDiagnostic(t *testing.T) {
	missing := &RootMissingError{Declared: ".backstop", Path: "/project/.backstop"}
	msg := missing.Error()
	if !strings.Contains(msg, ".backstop") || !strings.Contains(msg, "/project/.backstop") {
		t.Errorf("RootMissingError.Error() = %q, want it to name both the declared value and the path", msg)
	}

	invalid := &RootInvalidError{Declared: "../outside", Reason: "escapes the project root"}
	msg = invalid.Error()
	if !strings.Contains(msg, "../outside") || !strings.Contains(msg, "escapes the project root") {
		t.Errorf("RootInvalidError.Error() = %q, want it to name both the declared value and the reason", msg)
	}
}

// TestResolveRoot_RelativeProjectRootResolvesToAbsolutePath pins CLM-066: the
// absolute-path GUARANTEE. runGate passes the literal "." whenever DiscoverConfigPath
// fails (gate.go:77), and REQ-008's per-kind rule is a directory-string comparison
// that degenerates silently — to zero findings or to one per artifact — when one side
// is relative and the other absolute. Root.Dir must be absolute too, since that is
// what every consumer actually receives.
func TestResolveRoot_RelativeProjectRootResolvesToAbsolutePath(t *testing.T) {
	root, err := ResolveRoot(".", "")
	if err != nil {
		t.Fatalf("ResolveRoot(\".\", \"\") returned error %v, want nil", err)
	}
	if !filepath.IsAbs(root.Path) {
		t.Fatalf("root.Path = %q is not absolute; the guarantee is that no consumer receives a relative root", root.Path)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}
	if root.Path != filepath.Clean(cwd) {
		t.Errorf("root.Path = %q, want %q", root.Path, filepath.Clean(cwd))
	}
	if !filepath.IsAbs(root.Dir(KindSpec)) {
		t.Errorf("root.Dir(KindSpec) = %q is not absolute", root.Dir(KindSpec))
	}

	// A relative project root with a path segment resolves the same way.
	nested := filepath.Join(".", "testdata")
	nestedRoot, err := ResolveRoot(nested, "")
	if err != nil {
		t.Fatalf("ResolveRoot(%q, \"\") returned error %v, want nil", nested, err)
	}
	if !filepath.IsAbs(nestedRoot.Path) {
		t.Errorf("root.Path = %q is not absolute for a relative project root %q", nestedRoot.Path, nested)
	}
	wantNested := filepath.Clean(filepath.Join(cwd, "testdata"))
	if nestedRoot.Path != wantNested {
		t.Errorf("root.Path = %q, want %q", nestedRoot.Path, wantNested)
	}
	if !filepath.IsAbs(nestedRoot.Dir(KindBundle)) {
		t.Errorf("root.Dir(KindBundle) = %q is not absolute", nestedRoot.Dir(KindBundle))
	}
}
