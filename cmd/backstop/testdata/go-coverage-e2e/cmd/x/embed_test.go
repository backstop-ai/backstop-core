package x

import "testing"

func TestEmbed(t *testing.T) {
	if Embed() != 2 {
		t.Fatalf("Embed()=%d", Embed())
	}
}
