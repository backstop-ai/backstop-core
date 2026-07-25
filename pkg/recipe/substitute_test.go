package recipe

import (
	"strings"
	"testing"
)

func TestSubstitute_ResolvesDeclaredParam(t *testing.T) {
	params := map[string]string{
		"name":       "adopter",
		"config_dir": "config",
	}

	cases := []struct {
		name     string
		template string
		want     string
	}{
		{
			name:     "spaced placeholder",
			template: "owner: {{ name }}",
			want:     "owner: adopter",
		},
		{
			name:     "unspaced placeholder resolves identically",
			template: "owner: {{name}}",
			want:     "owner: adopter",
		},
		{
			name:     "extra-spaced placeholder resolves identically",
			template: "owner: {{  name  }}",
			want:     "owner: adopter",
		},
		{
			name:     "repeated placeholder resolves at every occurrence",
			template: "{{ name }}/{{ name }}/{{ name }}",
			want:     "adopter/adopter/adopter",
		},
		{
			name:     "multiple distinct placeholders",
			template: "{{ config_dir }}/{{ name }}.settings",
			want:     "config/adopter.settings",
		},
		{
			name:     "template with no placeholder passes through",
			template: "no placeholders here",
			want:     "no placeholders here",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Substitute(tc.template, params)
			if err != nil {
				t.Fatalf("Substitute(%q) errored: %v", tc.template, err)
			}
			if got != tc.want {
				t.Errorf("Substitute(%q) = %q, want %q", tc.template, got, tc.want)
			}
		})
	}
}

// TestSubstitute_NotTuringComplete_NoLogicEvaluated pins the code-in-data door
// shut: a logic/expression construct inside a placeholder is NEVER evaluated as
// code. The assertion is positive and falsifiable — the construct either fails
// loud as an undeclared param or is left literally untouched, and in NO case may
// the output contain an EVALUATED result. A text/template-backed implementation
// would evaluate these and fail here.
func TestSubstitute_NotTuringComplete_NoLogicEvaluated(t *testing.T) {
	// Positive control: the SAME params resolve a genuine declared placeholder,
	// so an implementation that passes by rejecting everything cannot pass.
	params := map[string]string{
		"x":     "1",
		"a":     "quiet",
		"Items": "one",
		"name":  "adopter",
	}
	if got, err := Substitute("owner: {{ name }}", params); err != nil || got != "owner: adopter" {
		t.Fatalf("positive control: Substitute(\"owner: {{ name }}\") = %q, err = %v; want %q, nil", got, err, "owner: adopter")
	}

	cases := []struct {
		name      string
		template  string
		forbidden []string
	}{
		{
			name:      "arithmetic expression is never computed",
			template:  "value: {{ 1 + 1 }}",
			forbidden: []string{"value: 2"},
		},
		{
			name:      "conditional is never branched",
			template:  "{{ if x }}TAKEN{{ end }}",
			forbidden: []string{"TAKENTAKEN"},
		},
		{
			name:      "range is never expanded",
			template:  "{{ range .Items }}ITEM{{ end }}",
			forbidden: []string{"ITEMITEM"},
		},
		{
			name:      "pipeline function is never applied",
			template:  "{{ a.b | upper }}",
			forbidden: []string{"QUIET", "quiet"},
		},
		{
			name:      "field traversal is never dereferenced",
			template:  "{{ .Items }}",
			forbidden: []string{"one"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Substitute(tc.template, params)

			if err != nil {
				// Fail-loud branch: nothing usable is emitted.
				if got != "" {
					t.Errorf("Substitute(%q) errored but still returned %q; a failed substitution must not emit a usable result", tc.template, got)
				}
				return
			}

			// Untouched branch: the construct is left literally in place.
			if got != tc.template {
				t.Errorf("Substitute(%q) = %q; a construct that is not an error must be left literally UNTOUCHED", tc.template, got)
			}
			for _, bad := range tc.forbidden {
				if strings.Contains(got, bad) {
					t.Errorf("Substitute(%q) = %q, which contains the EVALUATED result %q — substitution must not be Turing-complete", tc.template, got, bad)
				}
			}
		})
	}
}

func TestSubstitute_UndeclaredParamFailsLoud(t *testing.T) {
	params := map[string]string{"declared": "present"}
	template := "owner: {{ undeclared_param }} ({{ declared }})"

	got, err := Substitute(template, params)
	if err == nil {
		t.Fatalf("a placeholder naming an undeclared param must fail loud, got nil error and output %q", got)
	}
	if !strings.Contains(err.Error(), "undeclared_param") {
		t.Errorf("error must NAME the unresolvable placeholder 'undeclared_param', got: %v", err)
	}

	// Never silently blanked: neither the blanked string nor any partially
	// substituted string is returned as a usable result.
	if got != "" {
		t.Errorf("Substitute returned %q alongside the error; a failed substitution must not emit a usable result", got)
	}
	blanked := strings.ReplaceAll(template, "{{ undeclared_param }}", "")
	if got == blanked {
		t.Errorf("Substitute returned the silently-blanked string %q — the undeclared placeholder must fail loud instead", blanked)
	}
}
