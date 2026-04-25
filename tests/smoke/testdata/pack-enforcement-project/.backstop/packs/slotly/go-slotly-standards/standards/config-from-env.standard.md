# Config From Environment

## Standard

All secret values — API keys, OAuth client secrets, signing secrets,
encryption keys, database passwords — must be loaded from environment
variables at startup. No secret value may be hardcoded in source code,
even as a "default" fallback.

## Required Practice

```go
// Correct
SlackClientSecret: os.Getenv("SLACK_CLIENT_SECRET")

// Incorrect
SlackClientSecret: "xoxb-1234567890-abcdef"
```

Non-secret configuration (ports, URLs, feature flags) may use hardcoded
defaults via a helper like `getEnv("PORT", "8080")`.

## Why

Hardcoded secrets are discoverable in version control history, build
artifacts, and crash dumps. Environment-variable-only secrets keep the
secret lifecycle outside the code lifecycle.
