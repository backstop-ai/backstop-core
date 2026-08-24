//go:build linux

package packval

import (
	"errors"
	"reflect"
	"slices"
	"testing"
)

func TestExternalSandbox_UnavailableLandlockSkipsProbeAndApplication(t *testing.T) {
	var probeCalls, applicationCalls, acknowledgementCalls, childCalls int
	unavailable := errors.New("Landlock unavailable")
	probe := func() error {
		probeCalls++
		return unavailable
	}
	apply := func() { applicationCalls++ }
	acknowledge := func() { acknowledgementCalls++ }
	launchChild := func() SandboxRunResult {
		childCalls++
		return SandboxRunResult{Output: []byte("ran")}
	}

	nativeLeg := func(string, []string, string, []byte, bool) (SandboxRunResult, error) {
		if err := probe(); err != nil {
			return SandboxRunResult{}, err
		}
		apply()
		acknowledge()
		return launchChild(), nil
	}
	externalLeg := func(string, []string, string, []byte, bool) (SandboxRunResult, error) {
		return launchChild(), nil
	}

	external, err := newSandboxRunnerWithExecution(SandboxModeExternal, nativeLeg, externalLeg)
	if err != nil {
		t.Fatal(err)
	}
	result, err := external.Run("child", nil, t.TempDir())
	if err != nil || string(result.Output) != "ran" {
		t.Fatalf("external result=%#v err=%v", result, err)
	}
	if got := []int{probeCalls, applicationCalls, acknowledgementCalls, childCalls}; !slices.Equal(got, []int{0, 0, 0, 1}) {
		t.Fatalf("external counters probe/application/ack/child=%v, want [0 0 0 1]", got)
	}

	probeCalls, applicationCalls, acknowledgementCalls, childCalls = 0, 0, 0, 0
	native, err := newSandboxRunnerWithExecution(SandboxModeNative, nativeLeg, externalLeg)
	if err != nil {
		t.Fatal(err)
	}
	result, err = native.Run("child", nil, t.TempDir())
	if !errors.Is(err, unavailable) || result.NativeSandboxApplied {
		t.Fatalf("native unavailable result=%#v err=%v", result, err)
	}
	if got := []int{probeCalls, applicationCalls, acknowledgementCalls, childCalls}; !slices.Equal(got, []int{1, 0, 0, 0}) {
		t.Fatalf("native counters probe/application/ack/child=%v, want [1 0 0 0] with no fallback", got)
	}
}

func TestPackSandbox_NativeEvidenceInstrumentationPreservesProbeAndApplicationOrder(t *testing.T) {
	packDir := t.TempDir()
	request := sandboxHelperRequest{
		Capability:  ConvertValidatorCapability(packDir, 6),
		Command:     "child",
		Args:        []string{"one"},
		Dir:         packDir,
		Environment: []string{"KEEP=value"},
		AckFD:       sandboxAckFD,
	}
	wantPolicy := DeriveSandboxRestrictions(request.Capability)
	var events []string
	applicationCalls, acknowledgementCalls, childCalls := 0, 0, 0
	execObserved := errors.New("exec observed")
	err := applyRestrictionsAndExecWith(
		request,
		func(got SandboxRestrictionSpec) error {
			events = append(events, "landlock")
			applicationCalls++
			if !reflect.DeepEqual(got, wantPolicy) {
				t.Fatalf("Landlock policy changed through plumbing:\ngot  %#v\nwant %#v", got, wantPolicy)
			}
			return nil
		},
		func(got SandboxRestrictionSpec) error {
			events = append(events, "seccomp")
			if !reflect.DeepEqual(got, wantPolicy) {
				t.Fatalf("seccomp policy changed through plumbing:\ngot  %#v\nwant %#v", got, wantPolicy)
			}
			return nil
		},
		func(fd int) error {
			events = append(events, "ack")
			acknowledgementCalls++
			if fd != sandboxAckFD {
				t.Fatalf("ack fd=%d, want %d", fd, sandboxAckFD)
			}
			return nil
		},
		func(dir string) error {
			events = append(events, "chdir")
			if dir != packDir {
				t.Fatalf("chdir=%q, want %q", dir, packDir)
			}
			return nil
		},
		func(command string) (string, error) {
			events = append(events, "lookpath")
			return "/resolved/" + command, nil
		},
		func(path string, argv, environment []string) error {
			events = append(events, "exec")
			childCalls++
			if path != "/resolved/child" || !slices.Equal(argv, []string{"/resolved/child", "one"}) || !slices.Equal(environment, request.Environment) {
				t.Fatalf("child exec values changed: path=%q argv=%q env=%q", path, argv, environment)
			}
			return execObserved
		},
	)
	if !errors.Is(err, execObserved) {
		t.Fatalf("orchestration error=%v, want exec sentinel", err)
	}
	wantEvents := []string{"landlock", "seccomp", "ack", "chdir", "lookpath", "exec"}
	if !slices.Equal(events, wantEvents) {
		t.Fatalf("application events=%v, want %v", events, wantEvents)
	}
	if applicationCalls != 1 || acknowledgementCalls != 1 || childCalls != 1 {
		t.Fatalf("application/ack/child counters=%d/%d/%d, want 1/1/1", applicationCalls, acknowledgementCalls, childCalls)
	}
}

func TestApplyRestrictionsAndExecWith_FailureBranchesStopBeforeChild(t *testing.T) {
	stages := []string{"landlock", "seccomp", "ack", "chdir", "lookpath", "exec"}
	for _, failAt := range stages {
		t.Run(failAt, func(t *testing.T) {
			var events []string
			failure := errors.New(failAt + " failed")
			step := func(name string) error {
				events = append(events, name)
				if name == failAt {
					return failure
				}
				return nil
			}
			err := applyRestrictionsAndExecWith(
				sandboxHelperRequest{Capability: ConvertValidatorCapability(t.TempDir(), 6), Command: "child", Dir: "/pack"},
				func(SandboxRestrictionSpec) error { return step("landlock") },
				func(SandboxRestrictionSpec) error { return step("seccomp") },
				func(int) error { return step("ack") },
				func(string) error { return step("chdir") },
				func(string) (string, error) {
					if err := step("lookpath"); err != nil {
						return "", err
					}
					return "/resolved/child", nil
				},
				func(string, []string, []string) error { return step("exec") },
			)
			if !errors.Is(err, failure) {
				t.Fatalf("error=%v, want %v", err, failure)
			}
			if events[len(events)-1] != failAt {
				t.Fatalf("events continued after failure: %v", events)
			}
		})
	}
}
