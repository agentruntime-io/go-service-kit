# go-service-kit v0.1.6

## Added

- Package **`chatpolicy`** — `MidTurnInboxEnabled(envEnabled, policy)` for agent inbox (I40) tenant/env gate (BFF + chat-service).

### Monorepo

```text
replace github.com/agentruntime-io/go-service-kit => ../packages/go-service-kit
```

Publish tag `v0.1.6` and remove `replace` in downstream services when ready.
