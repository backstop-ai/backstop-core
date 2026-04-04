package testdata

func Compute(flag bool) (value int, err error) {
	value = 10
	if flag {
		err = nil
		return
	}
	value = 20
	return
}
