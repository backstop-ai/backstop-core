package testdata

import (
	"strings"
	"testing"
)

// TestNoTargetCallExample has assertions but never calls the target package.
func TestNoTargetCallExample(t *testing.T) {
	s := strings.ToUpper("hello")
	if s != "HELLO" {
		t.Fatal("expected HELLO")
	}
}
