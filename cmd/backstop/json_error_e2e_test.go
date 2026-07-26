package main

import (
	"sort"
	"strings"
	"testing"
)

// SPEC-055 REQ-012, the stdout-purity half (CLM-083, CLM-088, CLM-089). Every test here
// drives the BUILT BINARY through runBackstopStreams. Not one may use runBackstop:
// against a single merged CombinedOutput buffer "stdout holds exactly one JSON
// document" passes for a run that also wrote a human line to stderr, and "stderr is
// empty" cannot even be expressed — which is the exact vacuity REQ-014 exists to
// prevent (spec Review Question 7).
//
// Each failing run below reaches its error BEFORE any clone, so none of them needs the
// hermetic remote harness and none of them can touch the network: pack add is given a
// local ref naming no pack, pack install runs with no backstop.lock, and update/upgrade
// name a pack the consumer's backstop.yml does not declare.

// jsonErrorEnvelopeFields is the field set every command's envelope must carry. It is
// asserted as a SET rather than field-by-field so CLM-089's "same field set" cannot be
// satisfied by a document that merely happens to carry the three fields a test looks up
// while also carrying a fifth one for a single command.
func jsonErrorEnvelopeFields(t *testing.T, doc map[string]any) []string {
	t.Helper()

	fields := jsonDocKeys(doc)
	sort.Strings(fields)
	return fields
}

// requireJSONOnlyFailure runs the built binary under --json and returns the single
// document it wrote to the SEPARATED stdout, having first required the run to fail, the
// stdout to be exactly one parseable document, and the stderr to be entirely empty.
//
// All three conditions are checked together because each alone is passable by a broken
// implementation: a run that printed nothing at all has clean stderr, and a run that
// exits zero can emit whatever it likes.
func requireJSONOnlyFailure(t *testing.T, bin, dir string, args ...string) map[string]any {
	t.Helper()

	stdout, stderr, code := runBackstopStreams(t, bin, dir, args...)

	if code != ExitViolations {
		t.Fatalf("exit %d for %v, want ExitViolations (%d)\nstdout: %s\nstderr: %s", code, args, ExitViolations, stdout, stderr)
	}
	if stderr != "" {
		t.Errorf("stderr is NOT empty under --json; the human diagnostic was not suppressed and a consumer reading both streams sees noise\nstderr: %q", stderr)
	}
	return decodeSingleJSONError(t, stdout)
}

// TestJSONError_PackAdd_EmitsSingleParseableObject — CLM-083. The whole of stdout must
// unmarshal as ONE document: a trailing human line, a second object, or a log line all
// break the decode, which is precisely what a consumer piping stdout into a parser
// experiences.
func TestJSONError_PackAdd_EmitsSingleParseableObject(t *testing.T) {
	bin := buildBackstopBinary(t)
	proj := newConsumerProject(t)

	doc := requireJSONOnlyFailure(t, bin, proj, "pack", "add", "./absent-pack", "--json")

	if got := jsonErrorField(t, doc, "command"); got != "pack add" {
		t.Errorf("command = %q, want %q", got, "pack add")
	}
	if got := jsonErrorField(t, doc, "kind"); got == "" {
		t.Error("kind is empty in the document a real failing run produced")
	}
	message := jsonErrorField(t, doc, "message")
	if !strings.Contains(message, "absent-pack") {
		t.Errorf("message %q does not name the pack ref that failed; the envelope carried no usable diagnostic", message)
	}
}

// TestJSONError_SuppressesHumanDiagnostic — CLM-088. Emptiness, not absence of a
// particular string: asserting only that "Error:" is missing passes for a run that
// wrote some OTHER human line to stderr, and stdout purity is a property of the whole
// stream pair, not of one token.
func TestJSONError_SuppressesHumanDiagnostic(t *testing.T) {
	bin := buildBackstopBinary(t)
	proj := newConsumerProject(t)

	stdout, stderr, code := runBackstopStreams(t, bin, proj, "pack", "add", "./absent-pack", "--json")

	if code != ExitViolations {
		t.Fatalf("exit %d, want ExitViolations (%d)\nstdout: %s\nstderr: %s", code, ExitViolations, stdout, stderr)
	}
	if stderr != "" {
		t.Errorf("stderr = %q under --json, want it entirely EMPTY", stderr)
	}
	// The paired half: stderr being empty is only meaningful if the failure was
	// reported somewhere, and the document on stdout is where.
	if strings.TrimSpace(stdout) == "" {
		t.Fatal("both streams are empty: the failure was silenced rather than rendered as JSON")
	}
	decodeSingleJSONError(t, stdout)
}

// TestJSONError_InstallUpdateUpgradeShareEnvelope — CLM-089. The other three commands
// emit the SAME envelope, each with its OWN command path. Both halves matter: a shared
// shape is what makes the mode consumable, and a per-command path is what makes it
// useful — a single hardcoded path would satisfy the shape check while telling the
// consumer nothing about which command failed.
func TestJSONError_InstallUpdateUpgradeShareEnvelope(t *testing.T) {
	bin := buildBackstopBinary(t)

	addDoc := requireJSONOnlyFailure(t, bin, newConsumerProject(t), "pack", "add", "./absent-pack", "--json")
	wantFields := jsonErrorEnvelopeFields(t, addDoc)

	tests := []struct {
		name        string
		args        []string
		wantCommand string
	}{
		{
			name:        "install with no lockfile",
			args:        []string{"pack", "install", "--json"},
			wantCommand: "pack install",
		},
		{
			name:        "update of an undeclared pack",
			args:        []string{"pack", "update", "demo-org/absent-pack", "--json"},
			wantCommand: "pack update",
		},
		{
			name:        "upgrade of an undeclared pack",
			args:        []string{"pack", "upgrade", "demo-org/absent-pack@2.0.0", "--json"},
			wantCommand: "pack upgrade",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc := requireJSONOnlyFailure(t, bin, newConsumerProject(t), tc.args...)

			if got := jsonErrorEnvelopeFields(t, doc); !equalStringSlices(got, wantFields) {
				t.Errorf("envelope fields %v, want the same set pack add emits, %v", got, wantFields)
			}
			if got := jsonErrorField(t, doc, "command"); got != tc.wantCommand {
				t.Errorf("command = %q, want %q", got, tc.wantCommand)
			}
			if got := jsonErrorField(t, doc, "kind"); got == "" {
				t.Error("kind is empty in the document a real failing run produced")
			}
			if got := jsonErrorField(t, doc, "message"); strings.TrimSpace(got) == "" {
				t.Error("message is empty; the envelope carried no diagnostic at all")
			}
		})
	}
}

// equalStringSlices compares two already-sorted field lists.
func equalStringSlices(a, b []string) bool {
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
