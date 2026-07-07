package covge

import "testing"

func TestEmbed(t *testing.T) {
	if Embed() != 1 {
		t.Fatalf("Embed()=%d", Embed())
	}
}
