package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// defaultStandardsDirs returns the configured standards directories, falling
// back to ["standards/"] when none are configured.
func defaultStandardsDirs(configured []string) []string {
	if len(configured) == 0 {
		return []string{"standards/"}
	}
	return configured
}

// discoverStandards recursively walks each directory in dirs, returning a
// sorted slice of absolute paths to files matching *.standard.md.
// If a directory does not exist, it is skipped and an error is not returned
// for that directory — the caller is responsible for checking directory
// existence beforehand (see REQ-005 partial directory handling).
func discoverStandards(dirs []string) ([]string, error) {
	var paths []string

	for _, dir := range dirs {
		absDir, err := filepath.Abs(dir)
		if err != nil {
			return nil, err
		}

		err = filepath.WalkDir(absDir, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				return nil
			}
			if strings.HasSuffix(d.Name(), ".standard.md") {
				paths = append(paths, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	sort.Strings(paths)
	return paths, nil
}
