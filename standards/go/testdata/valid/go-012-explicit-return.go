package testdata

func ComputeExplicit(flag bool) (int, error) {
	if flag {
		return 10, nil
	}
	return 20, nil
}
