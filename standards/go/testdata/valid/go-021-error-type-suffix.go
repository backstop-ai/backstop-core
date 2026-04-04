package testdata

type ParseError struct{}

func (e ParseError) Error() string {
	return "parse error"
}
