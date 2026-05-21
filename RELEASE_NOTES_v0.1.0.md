# go-service-kit v0.1.0

Release date: 2026-04-26

## Summary

`v0.1.0` is the first public release of `go-service-kit`. This cut establishes the shared Go backend logging foundation for the AgentRuntime stack.

The initial release is intentionally narrow. It provides one focused package, `logging`, to standardize `zap` logger construction, field naming, request and trace correlation, request completion logging, dependency failure logging, and sanitization helpers.

## Included in this release

- shared `zap` logger construction with `text` and `json` output modes
- RFC3339 UTC timestamps with millisecond precision
- canonical snake_case field helpers
- context correlation helpers for request, trace, tenant, workflow, run, and resource identifiers
- request completion helper with shared field naming
- dependency failure helper with sanitized HTTP dependency metadata
- redaction and URL sanitization helpers
- package tests covering the initial logging contract

## Intended use

This release is intended to be adopted first by `control-service`, then by the other Go backend services as the shared logging contract is rolled out.

## Notes

- this is a `v0.x` release because the package surface is still expected to evolve during the first migrations
- consumers should pin to the release tag rather than a floating commit

