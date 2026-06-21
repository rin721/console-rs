---
name: frontend-quality
description: Plan or implement the future Aoi[葵] WebUI in console-rs. Use when working on React frontend migration, i18n parity, API client assumptions, design-system quality, setup/admin/public routes, or Playwright verification for the new Rust-backed UI.
---

# Frontend Quality

Read root `AGENTS.md` first.

Current boundary:

1. The old `aoi-admin/web/app` is reference material, not the new production frontend.
2. Do not expose frontend production capabilities until Rust APIs exist in the route registry.
3. Keep default locale `zh-CN`; maintain `zh-CN` and `en` parity for visible copy.
4. Centralize API endpoints and client behavior. Do not store tokens, credentials, private payloads, or CSRF secrets in localStorage, sessionStorage, URLs, logs, screenshots, or test snapshots.
5. For visible UI changes, verify desktop and mobile, then run typecheck, i18n lint, build, and Playwright.
6. When adding or changing System/WebUI capability surfaces, also run `powershell -NoProfile -ExecutionPolicy Bypass -File scripts/webui-capability-boundary-scan.ps1`; the scan must keep old plugin/external deployment/bucket policy/mock surfaces out of production UI and prove System calls, including operation-record summaries, are backed by Rust route registry entries.
