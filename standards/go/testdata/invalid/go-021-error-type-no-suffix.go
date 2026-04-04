package testdata

type ParseFailure struct{}

func (e ParseFailure) Error() string {
	return "parse failed"
}
