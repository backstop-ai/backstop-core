# bclabs-portal

Client-facing portal for bclabs engagements. Renders live project status — requirement → bundle → spec → plan → implementation — from backstop lifecycle events, alongside compliance evidence (test coverage, gate outcomes, standards checks).

## Architecture boundaries (decided 2026-07-10)

- **backstop** owns event emission: a portal-agnostic webhook that POSTs artifact state on lifecycle transitions. Backstop never knows what a "client dashboard" is.
- **bclabs-portal** (this repo) owns the ingest endpoint, metrics store, read APIs, client auth, and dashboard UI. The read model (aggregation/keying across client/project/spec) lives here.
- **bclabs-web** (sibling repo) is the marketing site. It consumes this portal's public read API as just another client — showcasing a real open-source project built through backstop — and never touches portal internals or live client data.
- Work tracking backend: headless self-hosted PM tool (Plane or OpenProject, unmodified, swappable) rather than a fork. Compliance metrics come from backstop artifacts, not the PM tool.

## Status

Greenfield. This repo will be dogfooded through backstop itself — first bundles: the backstop webhook-emitter surface and the portal ingest/read-model.
