package covge

// Config carries no executable statements — a zero-statement file (the
// fieldcontract.go-shaped case) that a -coverprofile never emits a block for.
type Config struct {
	Name string
}

const Version = "1"
