package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack/distribution"
)

// SPEC-055 REQ-012, the kind-classification half (CLM-084..087). These drive
// writeJSONError DIRECTLY against a buffer: classification is a pure function of the
// error value, so a process is the wrong instrument for it — the e2e half
// (json_error_e2e_test.go) proves the document reaches the real stdout, and this half
// proves what is IN it.
//
// The kind strings are written as LITERALS here rather than referenced from
// json_error.go's constants. A test that asserts a constant equals itself passes for
// any value the implementation happens to hold, including a renamed or emptied kind,
// and the kinds are the wire contract a consumer's switch is written against.

// decodeSingleJSONError unmarshals raw as exactly ONE JSON document and returns it.
// Decoding through a stream decoder and then demanding EOF is what makes "one document"
// falsifiable: a trailing second object, a log line, or human prose after the document
// all leave a token behind and fail here.
func decodeSingleJSONError(t *testing.T, raw string) map[string]any {
	t.Helper()

	dec := json.NewDecoder(bytes.NewBufferString(raw))
	var doc map[string]any
	if err := dec.Decode(&doc); err != nil {
		t.Fatalf("output is not a parseable JSON object: %v\noutput: %q", err, raw)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		t.Fatalf("output carries more than one JSON document; a consumer piping stdout into a parser sees garbage after the first\noutput: %q", raw)
	}
	return doc
}

// stringField reads a field the envelope must carry as a non-empty string.
func jsonErrorField(t *testing.T, doc map[string]any, field string) string {
	t.Helper()

	raw, present := doc[field]
	if !present {
		t.Fatalf("the JSON error document omits the %q field; keys present: %v", field, jsonDocKeys(doc))
	}
	value, ok := raw.(string)
	if !ok {
		t.Fatalf("field %q is %T, want string", field, raw)
	}
	return value
}

// keysOf lists a document's fields for a failure message.
func jsonDocKeys(doc map[string]any) []string {
	keys := make([]string, 0, len(doc))
	for k := range doc {
		keys = append(keys, k)
	}
	return keys
}

// writeJSONErrorTo runs writeJSONError against a buffer and returns what it wrote. The
// write error is checked here rather than at each call site: a document that failed to
// render is not a document a caller may then assert fields on.
func writeJSONErrorTo(t *testing.T, command string, err error) string {
	t.Helper()

	var buf bytes.Buffer
	if writeErr := writeJSONError(&buf, command, err); writeErr != nil {
		t.Fatalf("writeJSONError(%q, %v) returned %v", command, err, writeErr)
	}
	return buf.String()
}

// TestJSONError_CarriesCommandKindAndMessage — CLM-084. All three fields are asserted,
// so a document that renders only the message (the shape that is easiest to write and
// useless to a consumer routing on kind) cannot pass.
func TestJSONError_CarriesCommandKindAndMessage(t *testing.T) {
	const command = "pack add"
	const message = "reading demo-org/absent-pack: no pack.yml"

	doc := decodeSingleJSONError(t, writeJSONErrorTo(t, command, errors.New(message)))

	if got := jsonErrorField(t, doc, "command"); got != command {
		t.Errorf("command = %q, want %q", got, command)
	}
	if got := jsonErrorField(t, doc, "message"); got != message {
		t.Errorf("message = %q, want %q", got, message)
	}
	if got := jsonErrorField(t, doc, "kind"); got == "" {
		t.Error("kind is empty; a consumer's switch on kind falls through silently")
	}
}

// TestJSONError_ClassifiesGitAndValidationKinds — CLM-085. The wrapped sub-case is the
// load-bearing one: a bare type assertion classifies the unwrapped values correctly and
// silently misclassifies everything the pipelines return through fmt.Errorf("%w"), which
// is most of what a real failure looks like.
func TestJSONError_ClassifiesGitAndValidationKinds(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "git error",
			err:  &distribution.GitError{Message: "clone failed: tag v9.9.9 not found"},
			want: "git",
		},
		{
			name: "wrapped git error",
			err:  fmt.Errorf("cloning demo-org/demo-pack: %w", &distribution.GitError{Message: "tag v9.9.9 not found"}),
			want: "git",
		},
		{
			name: "validation error",
			err:  &distribution.ValidationError{Message: "pack check failed: phase1-structural"},
			want: "validation",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc := decodeSingleJSONError(t, writeJSONErrorTo(t, "pack add", tc.err))

			if got := jsonErrorField(t, doc, "kind"); got != tc.want {
				t.Errorf("kind = %q, want %q", got, tc.want)
			}
			if got := jsonErrorField(t, doc, "message"); got != tc.err.Error() {
				t.Errorf("message = %q, want the error's own text %q", got, tc.err.Error())
			}
		})
	}
}

// TestJSONError_ClassifiesDependencyAndCapabilityKinds — CLM-086. These two are the
// kinds a consumer must be able to tell apart from a genuine finding: a nil dependency
// is a mis-built binary and an unavailable capability is scheduled work, neither of
// which is the operator's project being wrong.
func TestJSONError_ClassifiesDependencyAndCapabilityKinds(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "missing dependency",
			err:  &distribution.MissingDependencyError{Command: "pack add", Dependency: "git cloner"},
			want: "dependency",
		},
		{
			name: "wrapped missing dependency",
			err: fmt.Errorf("assembling the add command: %w",
				&distribution.MissingDependencyError{Command: "pack add", Dependency: "validator"}),
			want: "dependency",
		},
		{
			name: "capability unavailable",
			err:  &distribution.CapabilityUnavailableError{Capability: "remediation generation", Reference: "BUNDLE-006 REQ-014"},
			want: "capability",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc := decodeSingleJSONError(t, writeJSONErrorTo(t, "pack upgrade", tc.err))

			if got := jsonErrorField(t, doc, "kind"); got != tc.want {
				t.Errorf("kind = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestJSONError_UnclassifiedFailureUsesDefaultKind — CLM-087. The assertion is
// non-emptiness plus distinctness from the four typed kinds: an EMPTY kind is the
// natural output of a classifier written as four ifs with no else, and it makes every
// unclassified failure invisible to a consumer switching on kind.
func TestJSONError_UnclassifiedFailureUsesDefaultKind(t *testing.T) {
	doc := decodeSingleJSONError(t, writeJSONErrorTo(t, "pack install", errors.New("backstop.lock not found")))

	kind := jsonErrorField(t, doc, "kind")
	if kind == "" {
		t.Fatal("an unclassified failure rendered an EMPTY kind; a consumer's switch falls through with no default to catch it")
	}
	for _, typed := range []string{"git", "validation", "dependency", "capability"} {
		if kind == typed {
			t.Errorf("a plain error was classified as %q; the typed kinds must mean the typed errors and nothing else", typed)
		}
	}
}

// refusingWriter fails every write. It stands in for the stdout a failing command
// cannot reach (closed pipe, full disk) — the case where "the JSON document explained
// the failure" is FALSE and acting on it anyway would make the run silent.
type refusingWriter struct{}

func (refusingWriter) Write([]byte) (int, error) { return 0, errors.New("stdout is unwritable") }

// TestPackLifecycleFailure_ExplainsOnlyWhenTheDocumentWasWritten covers the disposition
// the four pack commands share (SPEC-055 REQ-012). Explained is what suppresses
// reportError's stderr line, so it may be set ONLY when a document actually reached the
// consumer: --json off, or a render that failed, must both stay loud.
func TestPackLifecycleFailure_ExplainsOnlyWhenTheDocumentWasWritten(t *testing.T) {
	failure := errors.New("pack demo-org/absent-pack not found in backstop.yml")
	enabled, disabled := true, false

	t.Run("json mode explains with the document", func(t *testing.T) {
		var buf bytes.Buffer

		exitErr := packLifecycleFailure(&buf, &enabled, "pack update", failure)

		if !exitErr.Explained {
			t.Error("Explained is unset under --json; reportError will add a human line to stderr and stdout stops being one clean document")
		}
		if exitErr.Code != ExitViolations {
			t.Errorf("Code = %d, want ExitViolations (%d)", exitErr.Code, ExitViolations)
		}
		doc := decodeSingleJSONError(t, buf.String())
		if got := jsonErrorField(t, doc, "command"); got != "pack update" {
			t.Errorf("command = %q, want %q", got, "pack update")
		}
	})

	t.Run("without json the failure stays loud", func(t *testing.T) {
		var buf bytes.Buffer

		exitErr := packLifecycleFailure(&buf, &disabled, "pack update", failure)

		if exitErr.Explained {
			t.Error("Explained is set with --json off; the operator's only diagnostic would be suppressed")
		}
		if buf.Len() != 0 {
			t.Errorf("wrote %q to stdout with --json off, want the human path untouched", buf.String())
		}
	})

	t.Run("nil flag pointer stays loud", func(t *testing.T) {
		var buf bytes.Buffer

		exitErr := packLifecycleFailure(&buf, nil, "pack update", failure)

		if exitErr.Explained {
			t.Error("Explained is set for a command wired with no --json flag at all")
		}
	})

	t.Run("a failed render does not claim to have explained", func(t *testing.T) {
		exitErr := packLifecycleFailure(refusingWriter{}, &enabled, "pack update", failure)

		if exitErr.Explained {
			t.Error("Explained is set although the document never reached stdout; the run would exit non-zero having printed NOTHING on either stream")
		}
		if exitErr.Message != failure.Error() {
			t.Errorf("Message = %q, want the failure's own text %q", exitErr.Message, failure.Error())
		}
	})
}
