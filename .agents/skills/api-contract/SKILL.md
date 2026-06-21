---
name: api-contract
description: Maintain Aoi[葵] API contracts in console-rs. Use when adding, removing, or changing HTTP routes, permission metadata, OpenAPI output, API catalog synchronization, route registry entries, or frontend API assumptions.
---

# API Contract

Read root `AGENTS.md` first.

Rules:

1. Update `crates/core/app/src/transport/http/route_registry.rs` as the source of truth.
2. Route registration, API catalog, permission metadata, and OpenAPI must derive from the registry.
3. Do not add scattered `/api/v1` strings outside routing, tests, docs, or generated client work.
4. Use explicit `access`, `scope`, `permission`, `request_schema`, and `response_schema` metadata.
5. Public `/openapi.yaml` stays outside authenticated API catalog permissions.
6. After changes, run `cargo test --workspace` and inspect `cargo run -p app -- routes --config configs/console.example.yaml`.


