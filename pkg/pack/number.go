package pack

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var languagePattern = regexp.MustCompile(`^[a-z]+$`)

// ValidateLanguage checks that the language string matches ^[a-z]+$.
func ValidateLanguage(language string) error {
	if language == "" {
		return fmt.Errorf("language is required")
	}
	if !languagePattern.MatchString(language) {
		return fmt.Errorf("language must match ^[a-z]+$, got %q", language)
	}
	return nil
}

// ResolvePackNumber scans standards/<language>/ for existing standard files
// and returns the next available number. Gaps are not filled. If no existing
// standards exist for the language, the number starts at 1.
func ResolvePackNumber(language string, projectRoot string) (int, error) {
	langUpper := strings.ToUpper(language)
	dir := filepath.Join(projectRoot, "standards", language)

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 1, nil
		}
		return 0, fmt.Errorf("reading standards directory: %w", err)
	}

	// Pattern: STD-<LANG>-<NNN>-<slug>.standard.md
	pattern := regexp.MustCompile(fmt.Sprintf(`^STD-%s-(\d{3})-.*\.standard\.md$`, regexp.QuoteMeta(langUpper)))

	maxNum := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		matches := pattern.FindStringSubmatch(entry.Name())
		if matches == nil {
			continue
		}
		num, err := strconv.Atoi(matches[1])
		if err != nil {
			continue
		}
		if num > maxNum {
			maxNum = num
		}
	}

	return maxNum + 1, nil
}
