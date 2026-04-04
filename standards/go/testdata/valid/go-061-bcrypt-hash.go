package testdata

import "golang.org/x/crypto/bcrypt"

func StrongHash(password string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
}
