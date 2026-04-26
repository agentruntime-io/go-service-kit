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

Current release: `v0.1.3`

`v0.1.3` refines `prettytext` into a real local console renderer with event-class layouts for request, dependency, DB, phase, startup, and generic logs.

## Package

Module path:

```go
github.com/agentruntime-io/go-service-kit
```

Install:

```powershell
go get github.com/agentruntime-io/go-service-kit@v0.1.3
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

- event-class-aware layouts for request, dependency, DB, phase, startup, and generic logs
- timestamp, level, event class, and caller in the header
- readable message-first body formatting instead of inline `msg="..."`
- stable field ordering
- optional terminal colors on interactive terminals
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

## License

This repository is released under the MIT License. See [LICENSE](C:/agentruntime/agentruntime/go-common/go-service-kit/LICENSE).
