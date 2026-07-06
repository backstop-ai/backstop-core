package check

import "testing"

// TestCheckType_String_AllValues covers every arm of CheckType.String(),
// including the default fall-through, so the neutral pass-identity vocabulary
// (the surviving surface after the in-process check engine was deleted in
// ISSUE-018) stays fully exercised.
func TestCheckType_String_AllValues(t *testing.T) {
	cases := []struct {
		ct   CheckType
		want string
	}{
		{CheckTypeLint, "lint"},
		{CheckTypeBuild, "build"},
		{CheckTypeTest, "test"},
		{CheckTypeFindings, "findings"},
	}
	for _, c := range cases {
		if got := c.ct.String(); got != c.want {
			t.Errorf("CheckType(%d).String() = %q, want %q", int(c.ct), got, c.want)
		}
	}
	// default arm: an out-of-range value renders as unknown(N).
	if got := CheckType(99).String(); got != "unknown(99)" {
		t.Errorf("CheckType(99).String() = %q, want %q", got, "unknown(99)")
	}
}
