# Slack Signature Verification

## Standard

Every HTTP handler that receives payloads from Slack (events, interactions,
slash commands) must verify the request signature **before** parsing or acting
on the payload. Verification uses `slack.NewSecretsVerifier` with the app's
signing secret.

## Required Practice

1. Read the raw request body.
2. Create a `SecretsVerifier` from the request headers and signing secret.
3. Write the body bytes to the verifier.
4. Call `verifier.Ensure()` and reject the request on failure.
5. Only then parse and process the payload.

## Why

Without signature verification, any party that knows the endpoint URL can
forge requests. This is Slack's primary authentication mechanism for incoming
webhooks and event subscriptions.
