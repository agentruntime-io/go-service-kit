# Changelog

All notable changes to `go-service-kit` will be documented in this file.

The format is based on Keep a Changelog, adapted for this repository.

## [0.1.1] - 2026-04-26

### Added

- shared global logger registration via `SetDefault(...)` and package-level `Info/Warn/Error` helpers
- level-aware request completion logging
- phase logging with OpenTelemetry span events
- context field extractor registration for service-specific correlation fields
- `trace_sampled` support in correlation output
- shared GORM logger adapter for slow-query and DB error logging
- flexible attribute conversion helpers for mixed key/value and typed field call sites

### Changed

- dependency failure logging now supports caller-supplied human-readable message text and level selection by error/HTTP status
- logger config now supports custom output writers for tests and service-specific bootstrap code
- README examples now reflect the shared helper style used by `control-service`

## [0.1.0] - 2026-04-26

### Added

- initial `logging` package built on `zap`
- shared logger construction with `text` and `json` output modes
- canonical snake_case field helpers for request, trace, identity, and dependency metadata
- context correlation helpers for `request_id`, `trace_id`, `span_id`, and common resource identifiers
- standard request completion logging helper
- dependency failure logging helper with sanitized HTTP dependency metadata
- redaction and URL sanitization helpers
- package tests covering correlation fields, request completion logs, dependency messages, and URL sanitization
