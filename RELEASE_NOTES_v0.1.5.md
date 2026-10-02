# go-service-kit v0.1.5

## Security

- OpenTelemetry Go (`go.opentelemetry.io/otel`, `go.opentelemetry.io/otel/trace`) **1.40.0 → 1.45.0** (CVE-2026-29181)

No API changes in `logging`.

> **Note:** Tag `v0.1.4` on GitHub predates this fix; use **`v0.1.5`** for Dependabot clearance.

### Upgrade

```bash
go get github.com/agentruntime-io/go-service-kit@v0.1.5
go mod tidy
```
