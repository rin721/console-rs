# Agent Rules

This file is the long-term local rule entry for the `web/app` React frontend. The repository root `AGENTS.md` remains higher priority; this file adds React-specific constraints for the public site, first-install setup wizard, shared platform console, and future product-line entries.

## Product Scope

- `web/app` is the unified React frontend for the public site, first-install setup flow, `/admin` shared platform console, and future product-line entries.
- Public site routes include home, about, blog, terms, privacy, login, signup, password reset, and invitation acceptance.
- First-install setup routes live under `/setup/*` and cover language selection, file storage, database, cache, initial account, site/system information, connection tests, error recovery, and completion routing.
- `/admin/*` is the shared platform console for backend-exposed IAM, System, probe, media, version, plugin query, audit, and design-system/theme configuration capabilities. It manages common platform foundations, not product-line-specific business workflows.
- Future product-line entries may add their own public pages, business consoles, route namespaces, feature modules, and domain models, but must reuse the platform account, organization, permission, configuration, audit, API client, i18n, Aoi React design system, quality tooling, and build flow.
- The admin design-system and theme settings page manages only a runtime-configurable UI semantic token subset. It is a permission-controlled configuration surface, not a developer tool for arbitrary source token editing.
- Do not invent production capabilities, drivers, settings, permissions, or workflows that the Go backend has not exposed.
- The legacy Nuxt source tree has been removed from the current workspace. Do not recreate `web/admin`; use Git history and `docs/ai/react-frontend-migration-plan.md` only when historical behavior must be checked.

## Stack

- Use React 19, TypeScript, Vite, React Router Framework Mode, Tailwind CSS v4, TanStack Query, Zustand, React Hook Form, Zod, Radix UI, TanStack Table, i18next/react-i18next, lucide-react, react-markdown, remark-gfm, rehype-sanitize, gray-matter, Shiki, Vitest, React Testing Library, Playwright, ESLint flat config, and Prettier.
- Package management is pnpm only, pinned by package metadata to `pnpm@10.22.0`.
- React Router uses Framework SPA mode. `react-router.config.ts` must keep `ssr: false`; root build-time render must not read `window`, `document`, `localStorage`, or `sessionStorage`.
- Tailwind CSS v4 is integrated through `@tailwindcss/vite`; Tailwind utilities consume Aoi semantic tokens and do not replace the design system.
- Vite public base and React Router build output must stay compatible with Go static hosting.

## Directory Boundaries

- Route modules live in `app/routes`.
- Root layout, providers, and global error handling live in `app/root.tsx` and `app/providers`.
- Aoi React design system lives in `app/components/aoi` and is layered as tokens, primitives, patterns, and templates.
- Public website feature code lives in public route modules or `app/features/site`.
- First-install setup code lives in `app/features/setup` and `app/routes/setup`; setup must not be nested inside public or admin layouts.
- Reusable business compositions live in `app/features/<domain>`, not inside Aoi primitives.
- API endpoints, client, error normalization, and request configuration live in `app/lib/api`.
- Client-only state lives in `app/stores`; server data cache uses TanStack Query.
- i18n configuration and resources live in `app/i18n`.
- Local Markdown blog content lives in `content/blog/{locale}`.
- Generated output, dependency folders, Playwright reports, and runtime test output must not be committed.

## Aoi React Components

- Aoi layers are tokens, primitives, patterns, and templates. Lower layers must not import higher layers.
- Radix UI is accessibility primitive infrastructure only. Do not expose Radix or shadcn/ui copies as long-term business components.
- Tailwind utilities may only consume design tokens. Component variants must be represented through Aoi props, data attributes, or semantic classes.
- Buttons, links, inputs, selects, checkboxes, switches, dialogs, menus, tables, badges, state messages, layout containers, navigation, form groups, setup wizard, step navigation, setup report, recovery messages, and Markdown prose must be covered by Aoi components or patterns.
- Theme preview, contrast checks, token controls, draft status, publish confirmation, import/export, rollback notices, and reduced-motion controls must use Aoi components or patterns.
- `appearance` describes visual form; `intent` describes semantic meaning.
- Icons use `lucide-react`. Icon-only controls must provide accessible names.
- Forms must expose label, help text, error text, focus state, disabled state, loading state, and `aria-describedby` linkage.

## Visual And Responsive Rules

- Public pages use a soft, bright, modular product style: generous whitespace, restrained accent color, clear headings, lightweight media cards, short CTAs, and controlled motion.
- Setup pages use a focused, trustworthy wizard style: clear progress, low distraction, safe credential handling, visible tests, explicit blockers, recovery paths, and completion state.
- Platform console pages use denser management UI: stable sidebar or top bar, title area, filters, toolbars, tables, pagination, dialogs, drawers, and explicit state.
- Do not use decorative gradient orbs, bokeh blobs, heavy glass effects, marketing card piles, or stock-like imagery in admin or setup workflows.
- Touch targets must be at least 44x44px. Fixed mobile navigation or action bars must not cover content.
- Every interactive element needs `:focus-visible`, keyboard paths, and `prefers-reduced-motion` support.
- Visible UI changes must be checked at `1440x900` and `390x844`.

## API And State

- Backend endpoints must be centralized in `app/lib/api/endpoints.ts`; pages and components must not add scattered `/api/v1` strings.
- API client owns base URL resolution, `X-Locale`, configured product/client headers, CSRF double-submit headers, cookie credentials, 401 refresh retry, AbortController, error normalization, empty responses, and downloads.
- TanStack Query manages server data, invalidation, retries, and concurrency. Zustand stores client UI preferences, server session snapshots, and current context only.
- Do not retry non-idempotent mutations unless the backend provides an idempotency contract.
- Browser authentication must use backend-issued HttpOnly cookies plus `GET /api/v1/me/session` or auth response session snapshots. Do not store access tokens, refresh tokens, token pairs, CSRF secrets, credentials, or private API payloads in localStorage, sessionStorage, URLs, logs, screenshots, or test snapshots.
- Product code, client type, CSRF names, Cookie names, session policy, cache behavior, and auth runtime defaults must come from backend public settings, app config, route contract, or controlled registries. Do not hardcode deployable product/platform/auth/cache strategy inside pages, stores, components, or feature modules.
- Local UI preferences must not store credentials, tokens, private API payloads, or sensitive business data.
- Setup schema, available fields, drivers, options, testability, and step state must come from backend setup APIs. Do not hardcode unsupported drivers or settings.
- Setup may add UI-only local validation or confirmation fields, such as password confirmation, only when they are not backend capabilities. Such fields must never be sent in API payloads, persisted, logged, exposed in URLs, or described as schema fields.
- Setup drafts may only keep non-sensitive state in memory or in an explicitly documented lifecycle. Database, cache, storage credentials, and initial account passwords must not be written to localStorage, sessionStorage, URLs, logs, screenshots, or test snapshots.
- Theme settings may offer local preview, import/export, and in-memory draft interactions before backend persistence exists, but the UI must clearly label them as non-production. Do not expose save, publish, audit, or rollback as production capabilities until the Go backend provides matching persistence, permission, audit, and rollback contracts.
- Runtime theme configuration must validate contrast against WCAG AA, respect responsive layout and `prefers-reduced-motion`, and must never allow user-entered token values to inject raw CSS, HTML, script, URLs, or unsafe arbitrary Tailwind classes.

## i18n

- i18n uses i18next and react-i18next.
- Default locale is `zh-CN`; supported frontend locales are `zh-CN` and `en`.
- Fallback is current locale then `zh-CN`; components must not implement extra fallback logic.
- User language preference order is explicit user preference, backend public settings, browser language, then `zh-CN`; this order must be centralized.
- API requests must send `X-Locale` through the shared API client. Frontend `zh-CN` maps to backend `zh-CN`; frontend `en` maps to backend `en-US` until backend locale canonicalization is migrated.
- User-visible copy must not be hardcoded in pages, components, stores, config, form schemas, route metadata, or UI-rendering tests. It must live in locale resources.
- i18n keys are layered under `common`, `site`, `setup`, `auth`, `admin`, `forms`, `errors`, `empty`, `loading`, `seo`, `markdown`, and `a11y`.
- i18n covers the public site, admin, first-install setup, auth pages, validation, errors, empty states, loading states, table columns, operation buttons, step navigation, nav menus, Markdown metadata, SEO title/description, Open Graph copy, and accessibility labels.
- New pages or components must add matching `zh-CN` and `en` keys. Deleted surfaces must remove obsolete keys in the same phase.
- Stable technical names such as API Token, Redis, SMTP, HTTP methods, paths, and protocol names may remain English.

## Markdown Content

- Blog content uses local `.md` files in `content/blog/{locale}` during the first phase.
- Every article must include front matter: `title`, `description`, `date`, `updatedAt`, `slug`, `tags`, `locale`, `draft`, `cover`, and `author`.
- Front matter is parsed by gray-matter and validated with Zod.
- Markdown rendering uses react-markdown and remark-gfm.
- Arbitrary HTML is disabled by default. If raw HTML is ever enabled, rehype-sanitize with an explicit allowlist, security notes, and tests is required.
- Code highlighting uses Shiki and should be moved toward build-time processing.
- Markdown-derived titles, descriptions, tags, SEO copy, and Open Graph copy must be locale-specific.

## Verification

- Run `pnpm --dir web/app typecheck` after TypeScript, routes, components, stores, API, i18n, setup, Markdown, or config changes.
- Run `pnpm --dir web/app lint:i18n` after copy or locale-structure changes.
- Run `pnpm --dir web/app test` or `pnpm --dir web/app test:unit` after reusable logic, forms, API normalization, Markdown parsing, setup behavior, or Aoi primitives change.
- Run `pnpm --dir web/app lint` for source lint coverage.
- Run `pnpm --dir web/app test:e2e` or equivalent Playwright checks after visible UI, setup wizard, auth, route guard, or key admin flow changes.
- Run `pnpm --dir web/app build` after build, Vite, React Router, Tailwind, Markdown preprocessing, or Go static hosting path changes, and confirm `build/client/index.html` exists.
- After Go static hosting is switched, run backend tests proving `/api/v1`, `/health`, and `/ready` are not swallowed by SPA fallback.
- Run desktop and mobile Browser or Playwright checks for theme settings UI changes, including contrast validation, reduced-motion behavior, draft/publish disabled states, and import/export error handling.
