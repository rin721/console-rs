---
name: lifecycle-types
description: Keep Aoi[葵] console-rs shared application types in the app lifecycle layer. Use when moving, adding, or reviewing AppResult/AppError, AppContext/AppState, RequestContext, pagination, IDs, permission/audit context, configuration snapshots, runtime state, startup phase state, or other cross-module Rust types.
---

# Lifecycle Types

Read root `AGENTS.md` first.

Rules:

1. Put cross-module application types in the app lifecycle layer, currently `crates/core/app/src/app*`.
2. Do not create a broad root `types` module or move shared structs into unrelated domain modules.
3. Keep domain modules limited to their own domain models, domain enums, and domain errors.
4. Keep HTTP request/response DTOs in `handler` or `transport/http`; frontend display types stay in `web/app`.
5. If an infrastructure detail needs a DTO, keep it inside `infrastructure` and convert before crossing the repository/service boundary.
6. When moving shared types, update imports mechanically, run `rg "crate::error|crate::types|pub mod types"`, then run the Rust validation chain.

