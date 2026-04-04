package testdata

type Counter struct {
	count int
}

func NewCounter() *Counter {
	return &Counter{}
}
