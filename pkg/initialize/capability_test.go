package initialize

import (
	"sort"
	"strings"
	"testing"
)

// resolvedNames renders a resolved capability set as a sorted []string so an
// assertion reads as a set comparison rather than a map walk in nondeterministic
// order.
func resolvedNames(t *testing.T, set map[Capability]bool) []string {
	t.Helper()
	names := make([]string, 0, len(set))
	for capability, on := range set {
		if !on {
			t.Fatalf("resolved set carries %q mapped to false; a capability is present or absent, never present-and-off", capability)
		}
		names = append(names, string(capability))
	}
	sort.Strings(names)
	return names
}

// sameNames compares two sorted name slices.
func sameNames(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestInit_OnlyNarrowsToExactlyTheNamedCapabilities drives ResolveCapabilities
// directly (SPEC-069 CLM-011).
//
// `--only` is repeatable, narrows to EXACTLY the capabilities named and to no
// others, and can NEVER add, because its input is validated against the same
// seven-name vocabulary the default set is drawn from.
func TestInit_OnlyNarrowsToExactlyTheNamedCapabilities(t *testing.T) {
	t.Run("a single name narrows to exactly that one", func(t *testing.T) {
		set, err := ResolveCapabilities([]string{"git"}, nil)
		if err != nil {
			t.Fatalf("ResolveCapabilities(--only git) errored: %v", err)
		}
		got := resolvedNames(t, set)
		if !sameNames(got, []string{"git"}) {
			t.Fatalf("--only git resolved %v, want exactly [git]", got)
		}
	})

	t.Run("the flag is repeatable and the set is the union of the names", func(t *testing.T) {
		set, err := ResolveCapabilities([]string{"packs", "observe", "gitignore"}, nil)
		if err != nil {
			t.Fatalf("ResolveCapabilities(repeated --only) errored: %v", err)
		}
		got := resolvedNames(t, set)
		want := []string{"gitignore", "observe", "packs"}
		if !sameNames(got, want) {
			t.Fatalf("repeated --only resolved %v, want exactly %v", got, want)
		}
	})

	t.Run("naming every default capability reproduces the default set and adds nothing", func(t *testing.T) {
		all := make([]string, 0, len(DefaultCapabilities()))
		for _, capability := range DefaultCapabilities() {
			all = append(all, string(capability))
		}
		narrowed, err := ResolveCapabilities(all, nil)
		if err != nil {
			t.Fatalf("ResolveCapabilities(--only <every default>) errored: %v", err)
		}
		bare, err := ResolveCapabilities(nil, nil)
		if err != nil {
			t.Fatalf("ResolveCapabilities(bare) errored: %v", err)
		}
		if !sameNames(resolvedNames(t, narrowed), resolvedNames(t, bare)) {
			t.Fatalf("--only over every default resolved %v, want the bare default set %v",
				resolvedNames(t, narrowed), resolvedNames(t, bare))
		}
	})

	t.Run("--only can never reach outside the seven-name vocabulary", func(t *testing.T) {
		// The falsification half: if --only accepted an unvalidated name it would be
		// an ADD channel, which is the one thing the flag must never be.
		if _, err := ResolveCapabilities([]string{"git", "ci"}, nil); err == nil {
			t.Fatal("ResolveCapabilities(--only ci) succeeded; --only must never admit a name outside the seven")
		}
	})

	t.Run("--only combined with any --no- is a contradiction and errors", func(t *testing.T) {
		_, err := ResolveCapabilities([]string{"git"}, []string{"observe"})
		if err == nil {
			t.Fatal("combining --only and --no- succeeded; the two express contradictory intents about one set")
		}
	})

	t.Run("an unrecognized name errors listing all seven valid names", func(t *testing.T) {
		_, err := ResolveCapabilities(nil, []string{"lint"})
		if err == nil {
			t.Fatal("an unrecognized --no- name succeeded; it must be refused")
		}
		message := err.Error()
		for _, capability := range DefaultCapabilities() {
			if !strings.Contains(message, string(capability)) {
				t.Fatalf("the unrecognized-name error does not list %q; it must name all seven valid capabilities.\ngot: %s",
					capability, message)
			}
		}
	})
}

// TestInit_ScaffoldIsNotACapabilityName asserts `scaffold` is rejected as
// unrecognized exactly as any other eighth name would be (SPEC-069 CLM-132).
//
// The scaffold step is governed SOLELY by the presence of the `--scaffold` flag.
// There is no `--no-scaffold`, for the same reason there is no `--no-ci`: omission
// IS the opt-out, and adding `scaffold` to the capability set would give one outcome
// two report paths and two justifications.
func TestInit_ScaffoldIsNotACapabilityName(t *testing.T) {
	t.Run("scaffold is refused by --only", func(t *testing.T) {
		if _, err := ResolveCapabilities([]string{"scaffold"}, nil); err == nil {
			t.Fatal("--only scaffold was accepted; scaffold is not a capability name")
		}
	})

	t.Run("scaffold is refused by --no-", func(t *testing.T) {
		if _, err := ResolveCapabilities(nil, []string{"scaffold"}); err == nil {
			t.Fatal("--no-scaffold was accepted; there is no scaffold capability to subtract")
		}
	})

	t.Run("scaffold is refused for exactly the same reason any eighth name is", func(t *testing.T) {
		scaffoldErr := errorText(t, func() error {
			_, err := ResolveCapabilities([]string{"scaffold"}, nil)
			return err
		})
		eighthErr := errorText(t, func() error {
			_, err := ResolveCapabilities([]string{"telemetry"}, nil)
			return err
		})
		// Both messages quote the offending name and then list the same valid set, so
		// stripping the name leaves identical text. A scaffold-specific branch — a
		// special message, a tolerated alias — would break this.
		if strings.Replace(scaffoldErr, "scaffold", "<name>", 1) != strings.Replace(eighthErr, "telemetry", "<name>", 1) {
			t.Fatalf("scaffold is refused by a different code path than an ordinary unknown name.\nscaffold: %s\nother:    %s",
				scaffoldErr, eighthErr)
		}
	})

	t.Run("the vocabulary is exactly seven and holds neither scaffold nor ci", func(t *testing.T) {
		defaults := DefaultCapabilities()
		if len(defaults) != 7 {
			t.Fatalf("DefaultCapabilities() returned %d names (%v), want exactly seven", len(defaults), defaults)
		}
		want := []string{"baseline", "git", "gitignore", "observe", "packs", "sdlc", "toolchain"}
		got := make([]string, 0, len(defaults))
		for _, capability := range defaults {
			got = append(got, string(capability))
		}
		sort.Strings(got)
		if !sameNames(got, want) {
			t.Fatalf("the capability vocabulary is %v, want exactly %v", got, want)
		}
	})
}

// errorText runs fn and returns its error message, failing the test when fn
// unexpectedly succeeded.
func errorText(t *testing.T, fn func() error) string {
	t.Helper()
	err := fn()
	if err == nil {
		t.Fatal("expected an error and got none")
	}
	return err.Error()
}
