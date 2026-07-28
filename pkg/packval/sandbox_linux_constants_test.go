//go:build linux

package packval

import (
	"testing"

	"golang.org/x/sys/unix"
)

// TestSandboxCapability_ConstantsMatchKernelABI is the cross-check that makes the
// portable Landlock constants in sandbox_capability.go trustworthy.
//
// Those constants are declared by hand rather than imported, because
// LANDLOCK_ACCESS_FS_* live in golang.org/x/sys/unix's zerrors_linux.go and so do
// not exist on darwin — importing them would make the pure derivation impossible to
// compile, let alone test, on the machine where it is written.
//
// The cost of hand-declaring them is that a wrong bit would be SILENT AND TOTAL: the
// ruleset would install cleanly while handling or granting a right other than the one
// intended, and every portable test would still pass because they all read the same
// wrong constant. This test is the only thing standing between that and a shipped
// sandbox that confines the wrong operations. It runs wherever the real mechanism
// does.
func TestSandboxCapability_ConstantsMatchKernelABI(t *testing.T) {
	for name, pair := range map[string]struct{ ours, theirs uint64 }{
		"EXECUTE":     {AccessFSExecute, unix.LANDLOCK_ACCESS_FS_EXECUTE},
		"WRITE_FILE":  {AccessFSWriteFile, unix.LANDLOCK_ACCESS_FS_WRITE_FILE},
		"READ_FILE":   {AccessFSReadFile, unix.LANDLOCK_ACCESS_FS_READ_FILE},
		"READ_DIR":    {AccessFSReadDir, unix.LANDLOCK_ACCESS_FS_READ_DIR},
		"REMOVE_DIR":  {AccessFSRemoveDir, unix.LANDLOCK_ACCESS_FS_REMOVE_DIR},
		"REMOVE_FILE": {AccessFSRemoveFile, unix.LANDLOCK_ACCESS_FS_REMOVE_FILE},
		"MAKE_CHAR":   {AccessFSMakeChar, unix.LANDLOCK_ACCESS_FS_MAKE_CHAR},
		"MAKE_DIR":    {AccessFSMakeDir, unix.LANDLOCK_ACCESS_FS_MAKE_DIR},
		"MAKE_REG":    {AccessFSMakeReg, unix.LANDLOCK_ACCESS_FS_MAKE_REG},
		"MAKE_SOCK":   {AccessFSMakeSock, unix.LANDLOCK_ACCESS_FS_MAKE_SOCK},
		"MAKE_FIFO":   {AccessFSMakeFifo, unix.LANDLOCK_ACCESS_FS_MAKE_FIFO},
		"MAKE_BLOCK":  {AccessFSMakeBlock, unix.LANDLOCK_ACCESS_FS_MAKE_BLOCK},
		"MAKE_SYM":    {AccessFSMakeSym, unix.LANDLOCK_ACCESS_FS_MAKE_SYM},
		"REFER":       {AccessFSRefer, unix.LANDLOCK_ACCESS_FS_REFER},
		"TRUNCATE":    {AccessFSTruncate, unix.LANDLOCK_ACCESS_FS_TRUNCATE},
		"IOCTL_DEV":   {AccessFSIoctlDev, unix.LANDLOCK_ACCESS_FS_IOCTL_DEV},
	} {
		if pair.ours != pair.theirs {
			t.Errorf("LANDLOCK_ACCESS_FS_%s: sandbox_capability.go declares 0x%x, the kernel headers "+
				"say 0x%x — a wrong bit silently confines a different right than intended",
				name, pair.ours, pair.theirs)
		}
	}
}
