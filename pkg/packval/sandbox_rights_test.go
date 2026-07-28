package packval

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// ⚠ A GREEN RUN OF THIS FILE IS NOT EVIDENCE THAT THE SANDBOX INSTALLS (CLM-041).
//
// Everything here runs on darwin, where no Landlock rule is ever added and no
// landlock_add_rule syscall is ever made. These tests prove the mask ARITHMETIC and
// the WIRING; they cannot prove the kernel accepts the result. The only proof of
// that is TASK-019's re-run on a real Linux host. Both failures in this lane so far
// were preceded by a fully green local suite.
//
// WHY THIS FILE EXISTS. CI run 30383453888 failed with:
//
//	install the sandbox restrictions: landlock_add_rule(/etc/ld.so.cache, 0xc):
//	invalid argument
//
// 0xc is AccessFSReadFile|AccessFSReadDir, the mask landlockReadRights() returns for
// EVERY readable path. /etc/ld.so.cache is a REGULAR FILE, and the kernel rejects a
// path_beneath rule carrying directory-only rights against a non-directory. One bad
// rule aborts the ENTIRE restriction install — there is no partial application — so
// a single mistyped path takes the whole sandbox down rather than degrading it.

// TestNarrowRuleToInodeType_DropsDirectoryOnlyRightsOnFile is the regression lock,
// written against the measured failure rather than against a rule.
//
// The mask is 0xc because that is the literal value the run-30383453888 diagnostic
// printed; the path it printed, /etc/ld.so.cache, is a regular file. READ_DIR must
// go, READ_FILE must stay — dropping both would deny the loader the very read the
// path exists to permit.
func TestNarrowRuleToInodeType_DropsDirectoryOnlyRightsOnFile(t *testing.T) {
	got := narrowRuleToInodeType(AccessFSReadFile|AccessFSReadDir, false)

	if got&AccessFSReadDir != 0 {
		t.Errorf("READ_DIR survived on a non-directory (mask 0x%x). landlock_add_rule rejects "+
			"directory-only rights against a regular file with EINVAL, and one bad rule aborts the "+
			"whole install — this is the exact 0xc/etc/ld.so.cache failure from run 30383453888", got)
	}
	if got&AccessFSReadFile == 0 {
		t.Errorf("READ_FILE was dropped on a regular file (mask 0x%x); the loader could no longer read "+
			"the very file this rule exists to permit", got)
	}
}

// TestNarrowRuleToInodeType_KeepsEverythingOnDirectory stops the previous test being
// satisfiable the lazy way.
//
// A function that simply dropped READ_DIR unconditionally would pass the regression
// lock and silently un-restrict directory reads everywhere — the confinement would
// still install, and it would confine less than it claims. That is a worse outcome
// than the crash, because nothing would report it.
func TestNarrowRuleToInodeType_KeepsEverythingOnDirectory(t *testing.T) {
	mask := AccessFSReadFile | AccessFSReadDir
	if got := narrowRuleToInodeType(mask, true); got != mask {
		t.Errorf("a directory's mask was altered: got 0x%x, want 0x%x. Directories are exactly where "+
			"READ_DIR belongs; narrowing here would un-restrict directory reads", got, mask)
	}
}

// TestNarrowRuleToInodeType_DropsTheWholeDirectoryOnlyClass is the forward-safety
// assertion, and the reason this is a class-wide fix rather than a one-bit patch.
//
// The write family is NOT granted today — the darwin-parity capability has an empty
// writable-paths list, so the write branch at sandbox_capability.go never runs. That
// is precisely why it needs a test: the day anything grants a writable path (a future
// capability, or BUNDLE-021 OQ-2 resolving toward scratch space), a READ_DIR-only fix
// would fail in exactly the same way, on a mask carrying ten directory-only bits
// instead of one.
func TestNarrowRuleToInodeType_DropsTheWholeDirectoryOnlyClass(t *testing.T) {
	directoryOnly := map[string]uint64{
		"REMOVE_DIR":  AccessFSRemoveDir,
		"REMOVE_FILE": AccessFSRemoveFile,
		"MAKE_CHAR":   AccessFSMakeChar,
		"MAKE_DIR":    AccessFSMakeDir,
		"MAKE_REG":    AccessFSMakeReg,
		"MAKE_SOCK":   AccessFSMakeSock,
		"MAKE_FIFO":   AccessFSMakeFifo,
		"MAKE_BLOCK":  AccessFSMakeBlock,
		"MAKE_SYM":    AccessFSMakeSym,
		"READ_DIR":    AccessFSReadDir,
	}

	full := AccessFSWriteFile
	for _, bit := range directoryOnly {
		full |= bit
	}

	got := narrowRuleToInodeType(full, false)

	for name, bit := range directoryOnly {
		if got&bit != 0 {
			t.Errorf("directory-only right %s survived on a non-directory (mask 0x%x); the kernel would "+
				"reject the rule and abort the entire install", name, got)
		}
	}
	if got&AccessFSWriteFile == 0 {
		t.Error("WRITE_FILE was dropped on a regular file; writing a file is file-applicable and must " +
			"survive, or a granted writable path would silently lose its grant")
	}
}

// TestNarrowRuleToInodeType_FileApplicableRightsSurvive guards the other direction.
//
// A narrowing that is too aggressive breaks the interpreter exactly as thoroughly as
// one that is too lax, and it fails in the same confusing place — as a broken convert
// script rather than as a sandbox decision. That is the ISSUE-029 failure mode.
func TestNarrowRuleToInodeType_FileApplicableRightsSurvive(t *testing.T) {
	fileApplicable := map[string]uint64{
		"EXECUTE":    AccessFSExecute,
		"WRITE_FILE": AccessFSWriteFile,
		"READ_FILE":  AccessFSReadFile,
		"TRUNCATE":   AccessFSTruncate,
		"IOCTL_DEV":  AccessFSIoctlDev,
	}

	var mask uint64
	for _, bit := range fileApplicable {
		mask |= bit
	}

	got := narrowRuleToInodeType(mask, false)

	for name, bit := range fileApplicable {
		if got&bit == 0 {
			t.Errorf("file-applicable right %s was dropped on a regular file (mask 0x%x); over-narrowing "+
				"breaks the interpreter just as thoroughly as under-narrowing", name, got)
		}
	}
}

// TestNarrowRuleToInodeType_OverTheRealMixedReadableSet drives the WHOLE declared
// readable list, not one hand-picked path.
//
// THE DEFECT WAS A MIXED-LIST DEFECT, SO THE FALSIFIER HAS TO BE A MIXED-LIST
// FALSIFIER. The list mixes kinds by design — /etc/ld.so.cache is a regular FILE and
// /etc/ld.so.conf.d is a DIRECTORY, adjacent lines in linuxSystemReadPaths — and a
// single-path test would have passed happily against whichever kind its author
// happened to pick. That is exactly how the bug reached a runner.
func TestNarrowRuleToInodeType_OverTheRealMixedReadableSet(t *testing.T) {
	// The real declared set, with each path's true kind on a Linux host. Only
	// /etc/ld.so.cache is a regular file; everything else is a directory.
	isDir := map[string]bool{
		"/usr/lib":          true,
		"/usr/lib64":        true,
		"/lib":              true,
		"/lib64":            true,
		"/usr/local/lib":    true,
		"/usr/bin":          true,
		"/bin":              true,
		"/usr/share":        true,
		"/etc/ld.so.cache":  false,
		"/etc/ld.so.conf.d": true,
	}

	capability := ConvertValidatorCapability(t.TempDir(), 7)
	readMask := landlockReadRights()

	sawFile, sawDir := false, false
	for _, path := range capability.ReadablePaths {
		kind, declared := isDir[path]
		if !declared {
			continue // packDir, which is a directory but whose name varies per run
		}
		got := narrowRuleToInodeType(readMask, kind)

		if kind {
			sawDir = true
			if got != readMask {
				t.Errorf("%s is a directory but its mask was narrowed to 0x%x (want 0x%x)", path, got, readMask)
			}
			continue
		}
		sawFile = true
		if got&AccessFSReadDir != 0 {
			t.Errorf("%s is a REGULAR FILE and kept READ_DIR (mask 0x%x) — this is the rule that aborted "+
				"the install in run 30383453888", path, got)
		}
		if got&AccessFSReadFile == 0 {
			t.Errorf("%s lost READ_FILE (mask 0x%x); it would be unreadable to the loader", path, got)
		}
	}

	// Without both kinds present the table above proves nothing about mixing.
	if !sawFile || !sawDir {
		t.Fatalf("the readable set no longer mixes kinds (file=%v dir=%v); this test is only meaningful "+
			"while it does", sawFile, sawDir)
	}
}

// TestConvertValidatorCapability_ReadableSetContainsRegularFiles pins the mixed-type
// property as INTENTIONAL rather than incidental.
//
// This is the test that would have predicted the defect. If someone later removes
// every regular-file path from the readable set, the narrowing stops being exercised
// by the default capability, and this test is what tells them so — rather than the
// next Linux runner.
func TestConvertValidatorCapability_ReadableSetContainsRegularFiles(t *testing.T) {
	knownRegularFiles := map[string]bool{"/etc/ld.so.cache": true}

	capability := ConvertValidatorCapability(t.TempDir(), 7)
	for _, path := range capability.ReadablePaths {
		if knownRegularFiles[path] {
			return
		}
	}

	t.Fatalf("the declared readable set contains no known regular-file path (looked for %v). The "+
		"file-type narrowing is then unexercised by the default capability, and the mixed-kind property "+
		"that made this defect real is gone — confirm that is intended before deleting this test",
		knownRegularFiles)
}

// TestSandboxLinuxApplySite_CallsNarrowRuleToInodeType is THE WIRING GUARD, and it is
// not optional.
//
// The apply site lives in //go:build linux code, so NO DARWIN TEST EVER EXECUTES THAT
// LINE. narrowRuleToInodeType could be perfectly correct and simply never called, and
// every other test in this file would still pass while the sandbox stayed broken —
// the same shape as the stderr fold, which was also correct and also unreachable.
//
// go/parser reads a source file regardless of its build constraints, so the call can
// be asserted structurally from darwin. This guard was proven control-vs-treatment in
// a detached worktree: against a tree with the call removed it FAILS.
func TestSandboxLinuxApplySite_CallsNarrowRuleToInodeType(t *testing.T) {
	// sandbox_linux_helper.go, not sandbox_linux.go: applyLandlock moved there with
	// the rest of the exec-side functions when the coverage-measurement boundary was
	// drawn. This assertion followed the function rather than the filename — and it
	// FAILED loudly at the move rather than passing against a file that no longer
	// declares it, which is the whole reason the presence check below exists.
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sandbox_linux_helper.go", nil, 0)
	if err != nil {
		t.Fatalf("parsing sandbox_linux_helper.go: %v", err)
	}

	const applyFn = "applyLandlock"
	var target *ast.FuncDecl
	for _, decl := range file.Decls {
		if fd, ok := decl.(*ast.FuncDecl); ok && fd.Name.Name == applyFn {
			target = fd
		}
	}
	if target == nil {
		t.Fatalf("sandbox_linux_helper.go does not declare %s; this test asserts a property OF that "+
			"function and is meaningless if it moved or was renamed", applyFn)
	}

	called := false
	fstatted := false
	ast.Inspect(target, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch fn := call.Fun.(type) {
		case *ast.Ident:
			if fn.Name == "narrowRuleToInodeType" {
				called = true
			}
		case *ast.SelectorExpr:
			if fn.Sel.Name == "Fstat" {
				fstatted = true
			}
		}
		return true
	})

	if !called {
		t.Errorf("%s never calls narrowRuleToInodeType. A correct-but-uncalled narrowing is a passing "+
			"test suite and a broken sandbox: every rule would still carry directory-only rights and "+
			"landlock_add_rule would still abort the install with EINVAL", applyFn)
	}
	if !fstatted {
		t.Errorf("%s never calls Fstat. The inode type must come from the descriptor the rule REGISTERS "+
			"— a separate path-based stat could describe a different inode than the one the rule binds, "+
			"and a filename heuristic could not describe it at all", applyFn)
	}
}
