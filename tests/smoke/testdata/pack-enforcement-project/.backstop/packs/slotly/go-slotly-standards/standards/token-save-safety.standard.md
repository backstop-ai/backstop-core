# Token Save Safety

## Standard

After loading a user from the database (via `GetUserByID`, `GetUserBySlackID`,
`GetUserByWorkspaceAndSlackID`, or any method that calls `decryptUserTokens`),
the in-memory user struct contains **decrypted** token fields. Calling
`db.Save(user)` writes the entire struct back — overwriting encrypted tokens
with their plaintext values.

## Required Practice

- Use `db.Model(&user).Update("field", value)` for single-field updates.
- Use `db.Model(&user).Updates(map[string]interface{}{...})` for multi-field.
- Only use `db.Save()` when creating a new user (`CreateUser`) or when the
  token encryption is handled explicitly by the calling function.

## Why

Token corruption is a critical security vulnerability. It silently degrades
encrypted storage to plaintext, and is only discovered when token operations
fail after a re-read.
