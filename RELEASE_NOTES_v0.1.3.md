# go-service-kit v0.1.3

## Summary

`v0.1.3` upgrades `prettytext` from a structured inline text mode into a real local console renderer. It keeps `json` and `text` unchanged, while making local backend logs significantly easier to read during service runs and incident debugging.

## Added

- event-class-aware `prettytext` rendering for:
  - request logs
  - dependency logs
  - DB and slow SQL logs
  - phase logs
  - startup logs
  - generic logs
- optional `kind` field helper for explicit pretty renderer classification

## Changed

- `prettytext` now renders human-first multiline blocks instead of inline `msg="..."`
- request completion logs have a distinct local layout
- dependency failures have a distinct local layout
- DB and slow SQL logs have a distinct local multiline layout
- phase and startup logs now render with clearer local console structure
- shared helpers now attach stable event kinds so local pretty rendering is richer without changing JSON semantics

## Compatibility

- `json` output is unchanged
- `text` output is unchanged
- field names and structured semantics remain stable
- this release is focused on local developer experience and terminal readability

## Intended consumers

- `control-service`
- `agentruntime`
- future Go backends that want:
  - structured JSON in deployed environments
  - human-first readable logs locally
