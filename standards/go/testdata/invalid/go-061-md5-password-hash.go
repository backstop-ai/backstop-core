package testdata

import "crypto/md5"

func WeakHash(input []byte) [16]byte {
	return md5.Sum(input)
}
