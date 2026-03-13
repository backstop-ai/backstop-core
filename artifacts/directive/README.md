# Directive

A formal instruction or constraint that backstop enforces. Directives are the atomic unit of a standards pack — each directive declares what it checks, how it checks it, and what happens on failure.

## Schema

- `schema.json` — JSON Schema for directive validation

## Versioning

Schemas are versioned in subdirectories (e.g., `v1/`). The root `schema.json` always points to the latest stable version.
