---
title: "Fix Contract Verifier"
number: DIR-007
created: "2026-04-19"
schema_version: directive/v1

directive:
  status: done
  source:
    - "SPEC-010"
---

## Description

The contract signature verifier (gate step 6) has 31 violations from two issues: trying to Go-parse non-Go files (YAML, markdown, shell scripts, JSON) and stale contracts in specs where the implementation diverged.

Fix: skip non-Go files gracefully, update stale spec contracts to match current implementation (ResolveID in scaffold.go, codeCheckCmd factory pattern, ValidArtifactTypes var signature).
