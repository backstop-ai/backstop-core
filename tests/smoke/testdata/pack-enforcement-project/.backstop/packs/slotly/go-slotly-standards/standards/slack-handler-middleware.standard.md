# Slack Handler Middleware

## Standard

Every Go package that registers HTTP routes for Slack callbacks must include
signature verification — either as middleware applied to the router or as
inline verification at the top of each handler function.

## Required Practice

The package must contain at least one of:
- A call to `slack.NewSecretsVerifier` in a middleware function applied via
  router `.Use()`.
- A call to `slack.NewSecretsVerifier` in each handler that receives Slack
  payloads.

## Why

This is a presence check: it ensures no Slack-facing handler package ships
without verification code. The content-level correctness of the verification
is covered by slotly-002; this rule catches the case where verification is
entirely absent from a new package.
