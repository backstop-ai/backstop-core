package pack

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// tempProjectDir creates a temporary directory simulating a project root
// with standards/ and recipes/ subdirectories. Returns the path.
// Registers t.Cleanup for removal.
func tempProjectDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "standards"), 0o755); err != nil {
		t.Fatalf("creating standards dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "recipes"), 0o755); err != nil {
		t.Fatalf("creating recipes dir: %v", err)
	}
	return dir
}

// seedStandard creates a minimal STD-<LANG>-<NNN>-<slug>.standard.md file
// in standards/<language>/ within the temp project dir.
func seedStandard(t *testing.T, root, language string, number int, slug string) {
	t.Helper()
	langUpper := strings.ToUpper(language)
	filename := fmt.Sprintf("STD-%s-%03d-%s.standard.md", langUpper, number, slug)
	dir := filepath.Join(root, "standards", language)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("creating standards/%s dir: %v", language, err)
	}
	filePath := filepath.Join(dir, filename)
	content := fmt.Sprintf("---\ntitle: %s\nnumber: STD-%s-%03d\n---\n", slug, langUpper, number)
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatalf("writing seed standard: %v", err)
	}
}

// seedRecipeDir creates recipes/<language>/<slug>/ directory within the
// temp project dir.
func seedRecipeDir(t *testing.T, root, language, slug string) {
	t.Helper()
	dir := filepath.Join(root, "recipes", language, slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("creating recipe dir: %v", err)
	}
}
