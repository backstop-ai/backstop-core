//go:build !linux

package packval

import "errors"

// The non-Linux counterparts of the trampoline entry points.
//
// They exist so sandbox.go — which is platform-agnostic and switches on
// runtime.GOOS at run time — COMPILES everywhere. On darwin the linux branch is
// unreachable, so these are never called; the file's job is to keep the dispatch
// honest rather than to provide behaviour.
//
// MaybeRunSandboxHelper is the exception that matters: cmd/backstop/main.go calls it
// UNCONDITIONALLY as its first statement, so it must exist on every platform. A
// no-op here is correct — there is no trampoline to re-enter — but the function must
// not be deleted from this file, or the shipped darwin build stops compiling and
// someone "fixes" it by removing the call from main, which silently disarms the
// Linux sandbox.
//
// It returns an error to match the linux signature, and nil is the only value it can
// produce: "this process is not a sandbox helper" is the permanent truth on a
// platform with no helper mode.
func MaybeRunSandboxHelper() error { return nil }

func linuxSandboxedRun(_ string, _ []string, _ string) ([]byte, error) {
	return nil, errors.New("linux sandbox requested on a non-linux build")
}

func linuxSandboxedRunStdout(_ string, _ []string, _ string, _ []byte) ([]byte, error) {
	return nil, errors.New("linux sandbox requested on a non-linux build")
}
