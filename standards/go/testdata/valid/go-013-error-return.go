package testdata

import "errors"

func ParseConfigSafe(path string) error {
	if path == "" {
		return errors.New("missing path")
	}
	return nil
}
