package pack

import (
	"testing"
)

// tempProjectDir returns a temporary directory to scaffold a pack into. ISSUE-032:
// the retired native-standards helpers (seedStandard / seedRecipeDir and the
// standards//recipes/ subdir creation) are gone with the .standard.md/.recipe.md
// scaffolder they served — engine packs scaffold into <root>/<slug>/.
func tempProjectDir(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}
