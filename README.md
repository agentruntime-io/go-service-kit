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

Current release: `v0.1.0`

This is the first public cut of the kit. The package is ready for adoption, but the API is still expected to evolve while the first services migrate onto it.

## Package

Module path:

```go
github.com/agentruntime-io/go-service-kit
```

Install:

```powershell
go get github.com/agentruntime-io/go-service-kit@v0.1.0
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
    Format:  logging.FormatJSON,
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
logger.Error(
    "vault read failed while resolving MCP source spec values",
    logging.CorrelationFields(ctx)...,
    logging.Dependency("vault"),
    logging.Operation("source_spec_read"),
    logging.ErrorCode("vault_read_failed"),
)
```

## Included packages

- `logging`: shared `zap` logger construction, canonical fields, context correlation, request completion logs, dependency failure logs, and sanitization helpers

## Release files

- [CHANGELOG.md](C:/agentruntime/agentruntime/go-common/go-service-kit/CHANGELOG.md)
- [RELEASING.md](C:/agentruntime/agentruntime/go-common/go-service-kit/RELEASING.md)
- [RELEASE_NOTES_v0.1.0.md](C:/agentruntime/agentruntime/go-common/go-service-kit/RELEASE_NOTES_v0.1.0.md)

## License

This repository is released under the MIT License. See [LICENSE](C:/agentruntime/agentruntime/go-common/go-service-kit/LICENSE).
