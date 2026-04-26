# go-service-kit

Shared Go backend service utilities for the AgentRuntime stack.

The first package in this repo is `logging`, which standardizes:

- `zap` logger construction
- `text` and `json` output modes
- canonical field names
- request and trace correlation helpers
- request completion logging
- dependency failure logging
- redaction and URL sanitization helpers

## Status

Current release: `v0.1.2`

`v0.1.2` adds the first human-first local log renderer, `prettytext`, on top of the migration-ready logging surface from `v0.1.1`.

## Package

Module path:

```go
github.com/agentruntime-io/go-service-kit
```

Install:

```powershell
go get github.com/agentruntime-io/go-service-kit@v0.1.2
```

Import:

```go
import "github.com/agentruntime-io/go-service-kit/logging"
```

## Quick start

Create a shared logger:

```go
logger, err := logging.New(logging.Config{
    Service: "control-service",
    Format:  logging.FormatPrettyText,
    Level:   "info",
})
if err != nil {
    return err
}
```

Add request and trace correlation fields from context:

```go
logger.Info(
    "mcp config request was allowed",
    logging.CorrelationFields(ctx)...,
)
```

For local human-readable output, `prettytext` renders logs with:

- timestamp, level, and caller first
- stable `key=value` field ordering
- `msg="..."` at the end
- optional terminal colors
- multiline rendering for selected payloads such as SQL

Emit a standard request completion log:

```go
logging.LogRequestComplete(logger, ctx, logging.RequestComplete{
    Service:  "control",
    Method:   r.Method,
    Route:    "/internal/mcp/config",
    Status:   200,
    Duration: 42 * time.Millisecond,
})
```

Emit a dependency failure log:

```go
logging.LogDependencyFailure(logger, ctx, logging.DependencyFailure{
    Message:    "vault read failed while resolving MCP source spec values",
    Dependency: "vault",
    Operation:  "source_spec_read",
    ErrorCode:  "vault_read_failed",
    Err:        err,
})
```

Emit a phase log that also adds a span event:

```go
logging.LogPhase(logger, ctx, logging.PhaseLog{
    Message: "mcp config request is resolving allowed config keys",
    Phase:   "allowed_keys.resolve",
    Level:   logging.LevelInfo,
    Fields:  []any{"component", "mcp_config"},
})
```

## Included packages

- `logging`: shared `zap` logger construction, canonical fields, context correlation, request completion logs, dependency failure logs, phase logs, sanitization helpers, and a shared GORM logger adapter

## Release files

- [CHANGELOG.md](C:/agentruntime/agentruntime/go-common/go-service-kit/CHANGELOG.md)
- [RELEASING.md](C:/agentruntime/agentruntime/go-common/go-service-kit/RELEASING.md)
- [RELEASE_NOTES_v0.1.0.md](C:/agentruntime/agentruntime/go-common/go-service-kit/RELEASE_NOTES_v0.1.0.md)
- [RELEASE_NOTES_v0.1.1.md](C:/agentruntime/agentruntime/go-common/go-service-kit/RELEASE_NOTES_v0.1.1.md)
- [RELEASE_NOTES_v0.1.2.md](C:/agentruntime/agentruntime/go-common/go-service-kit/RELEASE_NOTES_v0.1.2.md)

## License

This repository is released under the MIT License. See [LICENSE](C:/agentruntime/agentruntime/go-common/go-service-kit/LICENSE).
