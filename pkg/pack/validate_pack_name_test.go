package pack

import (
	"strings"
	"testing"
)

// TestValidatePackName_ExportsTheExistingNameRule pins the exported name rule to the
// rule that already existed, rather than to "returns an error sometimes" (CLM-037).
//
// WHY THE EXPORT EXISTS. SPEC-056 REQ-003 makes the MANIFEST name a pack's install and
// runtime identity, which means pkg/pack/distribution has to decide whether a cloned
// tree's declared name is usable BEFORE it writes anything. Deciding that with a second
// copy of the rule is the failure mode the requirement names outright: two bodies that
// agree today drift tomorrow, and the drift surfaces as a pack that validates in one
// package and is rejected by the other.
//
// TWO ASSERTIONS PER ROW, AND BOTH ARE LOAD-BEARING.
//
// The first is the DISTINGUISHING MESSAGE. validateName produces three different
// failures with three different operator fixes — add a slash, fill in the empty part,
// remove the illegal character — and CLM-034/035/036 downstream depend on distribution
// being able to tell them apart when it builds an *IdentityError. Asserting merely
// non-nil would collapse all three into one unfalsifiable check, and a reimplementation
// that returned a single generic error for everything would still pass.
//
// The second is that ValidatePackName and validateName return the IDENTICAL error. That
// is what forecloses a SECOND COPY: the moment the call-through is replaced by a
// divergent reimplementation — even one that happens to agree on this table's verdicts
// but words a message differently — this fails. It is a stronger statement than
// scanning the source for one occurrence of the logic, because it constrains behavior
// rather than syntax.
func TestValidatePackName_ExportsTheExistingNameRule(t *testing.T) {
	tests := []struct {
		name string
		// wantMessage is the distinguishing fragment; empty means the name is valid.
		wantMessage string
		why         string
	}{
		{
			name:        "hermetic/valid-pack",
			wantMessage: "",
			why:         "one slash, both parts non-empty, every character inside [A-Za-z0-9-]",
		},
		{
			name:        "validpack",
			wantMessage: "exactly one slash",
			why:         "an unqualified name has no org part, so it cannot address a pack",
		},
		{
			name:        "a/b/c",
			wantMessage: "exactly one slash",
			why:         "two slashes is as unusable as none; the rule is exactly one, not at least one",
		},
		{
			name:        "hermetic/",
			wantMessage: "non-empty",
			why:         "the slash is present but the pack part is missing",
		},
		{
			name:        "/valid-pack",
			wantMessage: "non-empty",
			why:         "the slash is present but the org part is missing",
		},
		{
			name:        "hermetic/valid pack",
			wantMessage: "invalid characters",
			why:         "a space is outside namePartPattern and would break every path the name resolves to",
		},
		{
			name:        "hermetic/valid_pack",
			wantMessage: "invalid characters",
			why:         "an underscore is outside namePartPattern `^[A-Za-z0-9-]+$` — accepted by many registries, not by this one",
		},
		{
			name:        "",
			wantMessage: "exactly one slash",
			why:         "the empty string splits to a single empty part, so it fails on the slash count first",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ValidatePackName(tc.name)

			if tc.wantMessage == "" {
				if got != nil {
					t.Fatalf("ValidatePackName(%q) = %v, want nil (%s)", tc.name, got, tc.why)
				}
			} else {
				if got == nil {
					t.Fatalf("ValidatePackName(%q) = nil, want an error naming %q (%s)", tc.name, tc.wantMessage, tc.why)
				}
				if !strings.Contains(got.Error(), tc.wantMessage) {
					t.Errorf("ValidatePackName(%q) = %q, want a message containing %q (%s) — the three failure modes have three different operator fixes and distribution reports them as distinct identity errors",
						tc.name, got.Error(), tc.wantMessage, tc.why)
				}
			}

			// THE NO-SECOND-COPY ASSERTION. Identical results, message included.
			unexported := validateName(tc.name)
			switch {
			case got == nil && unexported == nil:
				// Agreed, both valid.
			case got == nil || unexported == nil:
				t.Errorf("ValidatePackName(%q) = %v but validateName(%q) = %v — they disagree on validity, so one is a second implementation of the other's rule",
					tc.name, got, tc.name, unexported)
			case got.Error() != unexported.Error():
				t.Errorf("ValidatePackName(%q) = %q but validateName(%q) = %q — the unexported name is no longer a call-through, which is exactly the duplicated rule REQ-003 forbids",
					tc.name, got.Error(), tc.name, unexported.Error())
			}
		})
	}
}
