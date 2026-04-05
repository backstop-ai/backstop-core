package testdata

import (
	"fmt"
	"os"
)

func ReadHandled(path string) ([]byte, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	return content, nil
}
