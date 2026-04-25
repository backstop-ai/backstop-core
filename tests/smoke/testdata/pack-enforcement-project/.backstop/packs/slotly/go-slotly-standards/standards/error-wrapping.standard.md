# Error Wrapping

## Standard

When returning errors from function calls to external packages (database,
HTTP clients, crypto, OAuth), wrap the error with `fmt.Errorf` using the
`%w` verb to preserve the error chain.

## Required Practice

```go
// Correct
if err != nil {
    return fmt.Errorf("failed to encrypt token: %w", err)
}

// Incorrect
if err != nil {
    return err
}
```

## Why

Bare error returns lose context about where the error originated. In a
system with multiple layers (handler -> oauth -> database -> crypto),
unwrapped errors produce stack traces that are difficult to diagnose.
The `%w` verb enables `errors.Is` and `errors.As` for callers.
