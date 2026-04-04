package testdata

import "os"

func LoadPasswordFromEnv() string {
	return os.Getenv("APP_PASSWORD")
}
