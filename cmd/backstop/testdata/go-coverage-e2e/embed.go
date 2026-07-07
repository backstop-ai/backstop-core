package covge

// Embed is a root-package function fully covered by embed_test.go, reproducing the
// repo-root embed.go that collides on basename with cmd/x/embed.go.
func Embed() int {
	return 1
}
