# go-service-kit v0.1.1

Release date: 2026-04-26

## Summary

`v0.1.1` is the first refinement release after the initial `v0.1.0` cut. It expands the shared logging package from a narrow baseline into a package that can support a full production service migration, starting with `control-service`.

This release keeps the package focused on logging and observability plumbing. It does not try to centralize service-specific message text or business decisions.

## Included in this release

- global shared logger registration with package-level logging helpers
- configurable text and json logger bootstrap with custom output writers
- correlation output that includes `trace_sampled`
- context field extractor registration for service-specific request and identity fields
- request completion helper with severity based on HTTP status
- dependency failure helper with caller-supplied readable messages and severity based on failure type
- phase log helper that also emits OpenTelemetry span events
- shared GORM logger adapter for slow-query and DB error logging
- improved test support for capturing logger output in consuming services

## Why this release exists

`v0.1.0` established the baseline logger shape. `v0.1.1` closes the gaps discovered during the `control-service` migration:

- package-level logger registration
- phase/span alignment
- request completion severity
- DB logger reuse
- service-specific context field bridging

## Intended use

This release is intended to be the first shared version consumed directly by `control-service`, replacing the local `slog` implementation path with `zap` through `go-service-kit/logging`.

## Notes

- this is still a `v0.x` release; the public surface is stronger now, but may still evolve before `v1.0.0`
- consumers should pin to `v0.1.1` rather than depending on a floating commit
