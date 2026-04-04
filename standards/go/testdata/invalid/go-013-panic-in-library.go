package testdata

func ParseConfig(path string) string {
	if path == "" {
		panic("missing path")
	}
	return path
}
