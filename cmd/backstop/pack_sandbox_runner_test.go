package main

import (
	"errors"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/packval"
)

type recordingSandboxRunner struct {
	mode          packval.SandboxMode
	runResults    []packval.SandboxRunResult
	stdoutResults []packval.SandboxRunResult
	runFn         func(string, []string, string) (packval.SandboxRunResult, error)
	stdoutFn      func(string, []string, string, []byte) (packval.SandboxRunResult, error)
	err           error
}

func recordedSandboxResult(output []byte, applied bool) packval.SandboxRunResult {
	var result packval.SandboxRunResult
	result.Output = output
	result.NativeSandboxApplied = applied
	return result
}

func (r *recordingSandboxRunner) Mode() packval.SandboxMode { return r.mode }

func (r *recordingSandboxRunner) Run(cmd string, args []string, dir string) (packval.SandboxRunResult, error) {
	if r.runFn != nil {
		return r.runFn(cmd, args, dir)
	}
	if len(r.runResults) == 0 {
		return packval.SandboxRunResult{}, r.err
	}
	result := r.runResults[0]
	r.runResults = r.runResults[1:]
	return result, r.err
}

func (r *recordingSandboxRunner) RunStdout(cmd string, args []string, dir string, stdin []byte) (packval.SandboxRunResult, error) {
	if r.stdoutFn != nil {
		return r.stdoutFn(cmd, args, dir, stdin)
	}
	if len(r.stdoutResults) == 0 {
		return packval.SandboxRunResult{}, r.err
	}
	result := r.stdoutResults[0]
	r.stdoutResults = r.stdoutResults[1:]
	return result, r.err
}

func directConvertSandboxRunner(gotStdin *[]byte) *recordingSandboxRunner {
	return &recordingSandboxRunner{
		mode: packval.SandboxModeNative,
		stdoutFn: func(cmd string, _ []string, _ string, stdin []byte) (packval.SandboxRunResult, error) {
			if gotStdin != nil {
				*gotStdin = append([]byte(nil), stdin...)
			}
			out, err := runConvertScriptDirect(cmd, stdin)
			return recordedSandboxResult(out, true), err
		},
	}
}

func TestRecordingSandboxRunner_PreservesModeAndEvidence(t *testing.T) {
	runner := &recordingSandboxRunner{mode: packval.SandboxModeExternal, err: errors.New("sentinel")}
	if runner.Mode() != packval.SandboxModeExternal {
		t.Fatalf("mode = %q", runner.Mode())
	}
	result, err := runner.Run("ignored", nil, "ignored")
	if !errors.Is(err, runner.err) || result.NativeSandboxApplied {
		t.Fatalf("result = %#v, err = %v", result, err)
	}
}
