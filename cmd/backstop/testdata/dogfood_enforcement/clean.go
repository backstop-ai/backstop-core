package dogfoodenforcement

// clean.go is the negative-control fixture for the CLM-025 enforcement-transfer
// test. It loads its credential from the environment instead of a string
// literal, so the backstop/go-standards GO-060 rule
// (go.security.no-hardcoded-credentials) must NOT flag it. If the test flags
// this file, the semgrep config is mis-wired (flags everything) — which the
// negative control is here to catch.

import "os"

func connectClean() string {
	apiKey := os.Getenv("APP_API_KEY")
	return apiKey
}
