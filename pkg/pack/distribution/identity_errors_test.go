package distribution_test

import (
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack/distribution"
)

// The REQ-009 renderer suite: direct Error() tests over hand-built values.
//
// They are built by hand rather than driven through a command on purpose — that keeps
// them meaningful when the plumbing around them changes, and it makes each one a
// statement about the DIAGNOSTIC an operator reads rather than about the path that
// produced it.
//
// EVERY ASSERTION IS A SUBSTRING, NEVER THE WHOLE MESSAGE. A whole-message assertion
// turns every future wording improvement into a red, which trains people to update the
// expectation without reading it — and an expectation nobody reads stops being a check.

func TestVersionUnresolvedError_ErrorNamesReferenceAndRemedy(t *testing.T) {
	err := &distribution.VersionUnresolvedError{
		Reference: "acme/pack",
		Problem:   "no version supplied",
	}

	msg := err.Error()
	if !strings.Contains(msg, "acme/pack") {
		t.Errorf("message %q does not quote the reference the operator typed", msg)
	}
	// BOTH ways to supply a version. An operator told only "no version" does not know
	// which forms this command accepts.
	if !strings.Contains(msg, "@") {
		t.Errorf("message %q does not mention the @version suffix", msg)
	}
	if !strings.Contains(msg, "--version") {
		t.Errorf("message %q does not mention the --version flag", msg)
	}
}

func TestVersionMismatchError_ErrorNamesBothVersions(t *testing.T) {
	err := &distribution.VersionMismatchError{
		Coordinate:      "backstop-ai/backstop-harness-toolchain-pack",
		Tag:             "v0.1.1",
		ManifestVersion: "0.1.3",
		ExpectedVersion: "0.1.1",
	}

	msg := err.Error()
	// All four fields. The pair of versions side by side is what tells an operator the
	// fix is a retag of the repository rather than a reinstall.
	for _, want := range []string{
		"backstop-ai/backstop-harness-toolchain-pack",
		"v0.1.1",
		"0.1.3",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("message %q does not name %q", msg, want)
		}
	}
}

func TestIdentityError_ErrorNamesCoordinateAndField(t *testing.T) {
	err := &distribution.IdentityError{
		Coordinate: "acme/pack",
		Tag:        "v1.0.0",
		Field:      "name",
		Problem:    "must contain exactly one slash",
	}

	msg := err.Error()
	if !strings.Contains(msg, "acme/pack") {
		t.Errorf("message %q does not name the coordinate", msg)
	}
	// The FIELD is what points the pack author at a line of their pack.yml. Without it
	// the diagnostic describes a problem without locating it.
	if !strings.Contains(msg, "name") {
		t.Errorf("message %q does not name the offending pack.yml field", msg)
	}
	if !strings.Contains(msg, "must contain exactly one slash") {
		t.Errorf("message %q does not carry the underlying problem", msg)
	}
}
