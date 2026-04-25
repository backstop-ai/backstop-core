package config

import "os"

// LoadWithFallback reads from env but falls back to a hardcoded secret
// when the variable is empty. This defeats the purpose of env-only secrets
// because the fallback ships in the binary.
func LoadWithFallback() string {
	val := os.Getenv("SLACK_SIGNING_SECRET")
	if val == "" {
		// ruleid: slotly-004
		val = "sk-fallback-signing-secret-1234567890"
	}
	return val
}
