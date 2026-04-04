package testdata

import "os"

func IgnoredErr() int {
	_, _ = os.ReadFile("missing.txt")
	return 1
}
