# go-service-kit v0.1.2

Release date: 2026-04-26

## Summary

`v0.1.2` adds `prettytext`, a local human-readable rendering mode for Go backend logs.

The goal of this release is not to change the structured logging contract. The goal is to make local terminal logs easier to read while preserving the same canonical fields, request and trace correlation, and message text used by `json` and `text` modes.

## Included in this release

- new `prettytext` log format
- custom pretty text renderer for local backend logs
- inline `key=value` field rendering with stable field ordering
- `msg="..."` printed at the end of the line for human readability
- ANSI color when writing to an interactive terminal
- multiline rendering for SQL payloads and stacktraces

## Why this release exists

`v0.1.1` made the package migration-ready. During local validation, the default console output still read too much like a JSON blob for day-to-day operator use.

`v0.1.2` addresses that by adding a separate local rendering mode instead of changing the semantics of existing `json` or `text` modes.

## Intended use

- use `json` for machine-oriented logs and shipped environments
- keep `text` available as the plain structured text mode
- use `prettytext` for local developer and operator terminal sessions

## Notes

- `prettytext` is intended for local interactive use, not centralized production log pipelines
- this release does not change canonical field names or logging behavior; it changes local rendering only
