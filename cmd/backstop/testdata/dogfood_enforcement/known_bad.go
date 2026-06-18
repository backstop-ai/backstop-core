package dogfoodenforcement

// known_bad.go is a self-contained fixture authored for the CLM-025
// enforcement-transfer test. It violates the backstop/go-standards GO-060
// rule (go.security.no-hardcoded-credentials): a credential-named variable
// assigned a string literal. It lives under testdata/ so the Go toolchain
// ignores it — it is fed to semgrep directly, never compiled as part of the
// module.

func connect() string {
	apiKey := "sk-live-abc123-not-a-real-secret"
	return apiKey
}
