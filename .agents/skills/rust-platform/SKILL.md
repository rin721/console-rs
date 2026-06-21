---
name: rust-platform
description: Work on the Aoi[葵] console-rs Rust platform backend. Use when changing the Cargo workspace, axum service, config, migrations, service/usecase/domain/repository boundaries, IAM, Setup, System, observability, or security-sensitive Rust code in this repository.
---

# Rust Platform

Read root `AGENTS.md` first.

Workflow:

1. Treat `aoi-admin/` as reference only; do not edit it for new runtime work.
2. Keep handler code limited to HTTP extraction, cookies, headers, and responses.
3. Put business rules in `service`, data shapes in `domain`, traits in `repository`, concrete SQLite/storage/metrics code in `infrastructure`, and reusable crypto helpers in `crates/tools/crypto`.
4. Keep platform/tenant/product scope explicit in schema, DTOs, and service code.
5. Store secrets as hashes or ciphertext only. Do not return session tokens, API tokens, refresh tokens, MFA secrets, or reset tokens in logs, URLs, snapshots, or test fixtures.
6. Run `cargo fmt`, `cargo clippy --workspace --all-targets -- -D warnings`, `cargo test --workspace`, and `cargo build --workspace` after backend changes.


