# React Frontend Final Audit

Status: IN_PROGRESS  
Date: 2026-06-19  
Scope: active goal completion audit for the unified React public site, setup flow, and `/admin` SaaS frontend.

## Summary

The React migration is not complete yet. Current evidence proves the main implementation, route parity, i18n system, Markdown pipeline, Aoi React design system, Go root static hosting, deploy script syntax, and legacy Nuxt source cleanup. Completion is still blocked by missing local Docker tooling for image construction.

Current weighted phase progress is approximately 99.8%. The goal must remain active until Docker image verification passes locally or in CI.

## Requirement Audit

| Requirement | Current evidence | Status |
| --- | --- | --- |
| React 19, TypeScript, Vite, React Router Framework Mode, Tailwind v4, TanStack Query/Table, Zustand, React Hook Form, Zod, Radix UI, i18next, lucide-react, Markdown, Shiki, Vitest, RTL, Playwright, ESLint flat config, and Prettier are used by the new frontend. | `web/app/package.json`, `web/app/vite.config.ts`, `web/app/react-router.config.ts`, `web/app/eslint.config.js`, `web/app/prettier.config.js`. | Proven |
| Public site includes home, about, blog index/detail, terms, privacy, login, signup, password, and invitation routes. | `web/app/app/routes.ts`, public route files, Playwright smoke coverage. | Proven |
| First-install setup flow is independent from public and admin shells and uses backend setup schema/status/run contracts. | `web/app/app/routes/setup/*`, `web/app/app/lib/api/setup.ts`, setup Playwright coverage, live setup smoke. | Proven |
| `/admin` route remains and covers backend-exposed IAM, System, probes, media, versions, plugins, parameters, dictionaries, operation records, API tokens, sessions, audit, and server info without inventing unsupported backend production capabilities. | `docs/ai/react-admin-api-parity-audit.md`, `web/app/app/routes/admin/*`, live admin mutation/error/permission smoke. | Proven |
| Design-system and theme settings page exists but does not pretend backend persistence, audit, publish, or rollback exist before Go exposes them. | `/admin/design-system` route, local-only Playwright assertion that no theme/design API calls are made. | Proven |
| i18n is unified on i18next/react-i18next with `zh-CN` default and `en` support; visible copy is in locale resources. | `web/app/app/i18n/*`, `web/app/app/i18n/locales/{zh-CN,en}.json`, `pnpm --dir web/app lint:i18n`. | Proven |
| Backend locale bridge maps frontend `en` to backend `en-US` for `X-Locale`. | `web/app/app/i18n/locales.ts`, API client tests and public settings/request coverage. | Proven |
| Markdown blog content uses local `.md`, front matter, gray-matter, Zod validation, react-markdown, remark-gfm, rehype-sanitize, and Shiki build-time highlighting. | `web/app/scripts/generate-blog-posts.mjs`, `web/app/app/components/aoi/patterns/MarkdownProse.tsx`, blog content files, build output. | Proven |
| Aoi React component system owns reusable primitives/patterns/templates; Radix remains a primitive dependency and Tailwind remains styling infrastructure. | `web/app/app/components/aoi`, `web/app/design/rules.md`, `web/app/AGENTS.md`. | Proven |
| Go static hosting serves the unified React SPA from `/` while keeping `/api`, `/api/v1`, `/health`, `/ready`, and plugin protocol paths outside SPA fallback. | `internal/config/app_webui.go`, `pkg/web/web.go`, `internal/transport/http/router.go`, Go static-hosting tests, live Go-hosted browser smoke. | Proven |
| Docker, CI, packaging, deploy docs, and default configs use `web/app/build/client` rather than Nuxt `.output/public`. | `Dockerfile`, `.github/workflows/ci.yml`, `scripts/package.py`, `configs/*.yaml`, `deploy/config.production.example.yaml`, release/build docs, residual searches. | Mostly proven |
| `deploy.sh` uses current React root-hosting and build-arg semantics. | `deploy.sh` now allows `webui.mount_path=/`, defaults WebUI check to `/admin/server-info`, and passes `VITE_PUBLIC_API_BASE_URL`; deploy workflow no longer forwards `WEBUI_BUILD_BASE_URL`; ShellCheck 0.11.0 reports no issues. | Proven |
| Old Nuxt/Vue/Material Web production source tree and long-term rules are removed or replaced. | `web/admin` tracked source is deleted; root `AGENTS.md`, `web/app/AGENTS.md`, and `web/app/design/rules.md` describe React rules; residual search only finds historical AI records and explicit "do not recreate" rules. | Proven |

## Latest Local Verification

- `pnpm --dir web/app lint:i18n` passed.
- `pnpm --dir web/app lint` passed.
- `pnpm --dir web/app typecheck` passed.
- `pnpm --dir web/app test:unit` passed: 9 files, 36 tests.
- `pnpm --dir web/app build` passed and verified `web/app/build/client/index.html`.
- `pnpm --dir web/app test:e2e -- --grep "admin API tokens route"` passed on desktop and mobile projects.
- `go test ./internal/transport/http ./pkg/web ./internal/app/initapp ./internal/modules/system/service ./pkg/i18n ./internal/config -count=1 -mod=readonly` passed.
- `git diff --check` passed with only the existing CRLF warning for `configs/locales/ui/en-US.yaml`.
- Residual current-production search found no `NUXT_APP_BASE_URL`, `NUXT_PUBLIC_API_BASE_URL`, Nuxt build command, `.output/public`, or `web/admin` static path in current production config/docs/scripts after excluding ignored local config and historical AI records.
- Temporary ShellCheck 0.11.0 downloaded under `tmp/ai/shellcheck`; `tmp\ai\shellcheck\shellcheck.exe deploy.sh` and `tmp\ai\shellcheck\shellcheck.exe -S error deploy.sh` passed.

## Remaining Verification

1. Run Docker image verification when Docker is available. Local `docker --version` currently fails because `docker` is not installed or not on `PATH`; `where docker`, Docker service search, common Docker install path checks, and `winget list --name Docker` found no installed Docker runtime.
2. Re-run the final requirement audit after Docker image verification passes, then mark the thread goal complete.

## Docker Verification Options

- Local path: install or expose Docker/Podman-compatible tooling, then run `docker build -t go-scaffold:ci .` from the repository root.
- CI path: push the current worktree to a branch and open or update a pull request so GitHub Actions can run the updated `.github/workflows/ci.yml` Docker build gate. The remote `CI` workflow currently visible through `gh workflow view CI --yaml` is still the old `web/admin` version until this local worktree is pushed, so the existing remote workflow cannot prove the current React migration state by itself.

## Notes

- Ignored local `configs/config.local.yaml` still points at the old Nuxt output. It is not tracked and was not edited because project rules exclude local runtime overrides unless explicitly requested.
- Historical AI documents still mention `web/admin`, Nuxt, and `.output/public` as past-state evidence. Current production docs and rules now point at `web/app/build/client`.
