# Changelog

All notable changes to `go-service-kit` will be documented in this file.

The format is based on Keep a Changelog, adapted for this repository.

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

