package pack

import "testing"

func TestParseCoordinate_FullFormat(t *testing.T) {
	c, err := ParseCoordinate("pack-name@1.0.0:item-name@2.0.0")
	if err != nil {
		t.Fatalf("ParseCoordinate returned error: %v", err)
	}
	if c.PackName != "pack-name" || c.PackVersion != "1.0.0" || c.ItemName != "item-name" || c.ItemVersion != "2.0.0" {
		t.Fatalf("unexpected coordinate parse: %+v", c)
	}
}

func TestParseCoordinate_NoItemVersion(t *testing.T) {
	c, err := ParseCoordinate("pack-name@1.0.0:item-name")
	if err != nil {
		t.Fatalf("ParseCoordinate returned error: %v", err)
	}
	if c.ItemVersion != "" {
		t.Fatalf("expected empty item version, got %q", c.ItemVersion)
	}
}

func TestParseCoordinate_Invalid(t *testing.T) {
	cases := []string{
		"",
		"pack-name@1.0.0",
		"pack-name:item-name",
		"pack-name@1.0.0:item-name@",
		"@1.0.0:item-name@1.0.0",
		"pack-name@1.0.0:item-name@2.0.0@3.0.0",
		"pack-name@1.0.0@2.0.0:item-name",
	}
	for _, c := range cases {
		if _, err := ParseCoordinate(c); err == nil {
			t.Fatalf("expected invalid coordinate to fail: %q", c)
		}
	}
}

func TestNamespaceRuleID_Correct(t *testing.T) {
	got := NamespacedRuleID("org/pack", "rule-id")
	if got != "org/pack/rule-id" {
		t.Fatalf("unexpected namespaced rule ID: %q", got)
	}
}

func TestValidateRuleID_ValidKebab(t *testing.T) {
	if err := ValidateRuleID("valid-rule-id"); err != nil {
		t.Fatalf("expected valid rule ID, got error: %v", err)
	}
}

func TestValidateRuleID_Uppercase(t *testing.T) {
	if err := ValidateRuleID("Invalid-Rule"); err == nil {
		t.Fatal("expected uppercase rule ID to fail")
	}
}

func TestValidateRuleID_Underscores(t *testing.T) {
	if err := ValidateRuleID("invalid_rule"); err == nil {
		t.Fatal("expected underscored rule ID to fail")
	}
}
