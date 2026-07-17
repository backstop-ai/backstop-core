package fixtures

// Clean spine: the enclosing test name and referenced symbol are read from a STRUCTURED
// properties map (the ISSUE-062 channel), never sliced out of a message by whitespace,
// so a spaced/quoted name survives intact. (Matches the *structured-property-read*
// include hook but yields no finding.)
func funcFromProperties(props map[string]string) string {
	return props["func"]
}

func symbolFromProperties(props map[string]string) string {
	return props["symbol"]
}
