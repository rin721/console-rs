# Project Map

This file is a compact architecture map for AI-assisted work. Human-facing docs remain under `docs/`.

## Core Shape

| Area | Current responsibility |
| --- | --- |
| `cmd/aoi` | Thin process entrypoint and CLI command declarations. |
| `internal/app` | Composition root for config, logger, database, cache, storage, IAM, System, Plugins, HTTP/RPC, lifecycle, and reload. |
| `internal/config` | Explicit config loading, env overrides, validation, diagnostics, snapshots, and persistence. |
| `internal/plugin` | Project-side remote plugin module: assembles `pkg/plugin.Host`, injection providers, admin service/handler, and protocol handler. |
| `internal/transport/http` | HTTP route registration, IAM-protected APIs, plugin HTTP/WS protocol endpoints, WebUI static mount, and API catalog. |
| `internal/transport/rpc` | JSON-RPC system method registration only; plugin protocol is not registered here. |
| `internal/modules/iam` | User, org, role, permission, session, MFA, API token, invitation, password reset, and audit behavior. |
| `internal/modules/system` | Menus, API catalog, dictionaries, parameters, versions, media, and server status. |
| `pkg/plugin` | Reusable remote plugin core, protocol abstractions, injection hook, and HTTP/WS transport adapters. |
| `pkg` | Reusable infrastructure packages; must not import `internal`. |

## Plugin Boundaries

- Remote plugins are independent processes or services.
- The host discovers a plugin only after receiving a remote registration request.
- Host config contains only plugin host settings; plugin private config stays inside the remote plugin process.
- `pkg/plugin` must not import `internal` or `pkg/rpcserver`.
- `internal/plugin` is the only project-side plugin capability aggregation point.
- Formal code must not import `_examples/remote-plugins` or `plugins/...`.

## Verification Hints

```powershell
go test ./pkg/plugin/... ./internal/plugin/... ./internal/app/... ./internal/config/... -count=1 -mod=readonly
go test ./... -count=1 -mod=readonly
rg -n "github\\.com/rei0721/go-scaffold/(plugins|_examples)" cmd internal pkg types --glob "*.go"
```
