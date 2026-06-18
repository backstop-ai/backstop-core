package x

func forbiddenCall(n int) int { return n }

func useBad() int { return forbiddenCall(3) }
